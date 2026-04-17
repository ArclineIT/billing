package web

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/stripe/stripe-go/v81/webhook"

	"arclineit.com/billing/internal/auth"
	"arclineit.com/billing/internal/mail"
	"arclineit.com/billing/internal/payments"
)

//go:embed templates
var templateFS embed.FS

// Handler holds all HTTP handler dependencies.
type Handler struct {
	DB      *sql.DB
	Stripe  payments.Config
	SMTP    mail.Config
	BaseURL string
	ts      *templateSet
}

// New creates a Handler and pre-parses all HTML templates.
func New(db *sql.DB, stripe payments.Config, smtp mail.Config, baseURL string) (*Handler, error) {
	ts, err := loadTemplates()
	if err != nil {
		return nil, err
	}
	return &Handler{DB: db, Stripe: stripe, SMTP: smtp, BaseURL: baseURL, ts: ts}, nil
}

// ---- template set ----

type templateSet struct {
	tmpl *template.Template
}

var templateFuncs = template.FuncMap{
	"slice": func(start, end int, s string) string {
		runes := []rune(s)
		if start < 0 {
			start = 0
		}
		if end > len(runes) {
			end = len(runes)
		}
		if start > end {
			return ""
		}
		return string(runes[start:end])
	},
	"fmtDate": func(s string) string {
		if s == "" {
			return "—"
		}
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return s
		}
		return t.Format("Jan 2, 2006")
	},
}

func loadTemplates() (*templateSet, error) {
	tmpl, err := template.New("").Funcs(templateFuncs).ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, err
	}
	return &templateSet{tmpl: tmpl}, nil
}

func (ts *templateSet) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ts.tmpl.ExecuteTemplate(w, name, data); err != nil {
		slog.Error("template render", "name", name, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

// ---- data models ----

type customer struct {
	ID               int64
	Email            string
	FirstName        string
	LastName         string
	StripeCustomerID string
	CreatedAt        string
}

type subscriptionRow struct {
	ID                   int64
	StripeSubscriptionID string
	StripePriceID        string
	PlanName             string
	Status               string
	CurrentPeriodEnd     string
}

type invoiceRow struct {
	ID              int64
	StripeInvoiceID string
	AmountCents     int64
	Currency        string
	Status          string
	InvoicePDFURL   string
	PeriodStart     string
	PeriodEnd       string
	CreatedAt       string
	AmountDisplay   string
}

type dashboardData struct {
	Customer     customer
	Subscription *subscriptionRow
	Invoices     []invoiceRow
	Flash        string
	CSRFToken    string
}

// ---- DB helpers ----

func loadCustomer(db *sql.DB, id int64) (customer, error) {
	var c customer
	err := db.QueryRow(
		`SELECT id, email, first_name, last_name, stripe_customer_id, created_at FROM customers WHERE id = ?`,
		id,
	).Scan(&c.ID, &c.Email, &c.FirstName, &c.LastName, &c.StripeCustomerID, &c.CreatedAt)
	return c, err
}

func loadSubscription(db *sql.DB, customerID int64) (*subscriptionRow, error) {
	var s subscriptionRow
	err := db.QueryRow(
		`SELECT id, stripe_subscription_id, stripe_price_id, plan_name, status, current_period_end
		FROM subscriptions WHERE customer_id = ? ORDER BY created_at DESC LIMIT 1`,
		customerID,
	).Scan(&s.ID, &s.StripeSubscriptionID, &s.StripePriceID, &s.PlanName, &s.Status, &s.CurrentPeriodEnd)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &s, err
}

func loadRecentInvoices(db *sql.DB, customerID int64) ([]invoiceRow, error) {
	rows, err := db.Query(
		`SELECT id, stripe_invoice_id, amount_cents, currency, status, invoice_pdf_url,
			period_start, period_end, created_at
		FROM invoices WHERE customer_id = ? ORDER BY created_at DESC LIMIT 5`,
		customerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []invoiceRow
	for rows.Next() {
		var inv invoiceRow
		if err := rows.Scan(
			&inv.ID, &inv.StripeInvoiceID, &inv.AmountCents, &inv.Currency,
			&inv.Status, &inv.InvoicePDFURL, &inv.PeriodStart, &inv.PeriodEnd, &inv.CreatedAt,
		); err != nil {
			return nil, err
		}
		dollars := inv.AmountCents / 100
		cents := inv.AmountCents % 100
		inv.AmountDisplay = formatCurrency(dollars, cents, strings.ToUpper(inv.Currency))
		result = append(result, inv)
	}
	return result, rows.Err()
}

func validEmail(s string) bool {
	at := strings.Index(s, "@")
	return at > 0 && at < len(s)-1 && strings.Contains(s[at+1:], ".")
}

func formatCurrency(dollars, cents int64, currency string) string {
	if currency == "USD" || currency == "" {
		return fmt.Sprintf("$%d.%02d", dollars, cents)
	}
	return fmt.Sprintf("%d.%02d %s", dollars, cents, currency)
}

// ---- session cookie helpers ----

func sessionSecure() bool {
	return os.Getenv("SESSION_SECURE") != "false"
}

func setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		MaxAge:   int(auth.SessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   sessionSecure(),
		SameSite: http.SameSiteStrictMode,
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   sessionSecure(),
		SameSite: http.SameSiteStrictMode,
	})
}

// ---- route handlers ----

func (h *Handler) IndexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	cookie, err := r.Cookie("session")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	_, err = auth.GetSession(h.DB, cookie.Value)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (h *Handler) LoginGET(w http.ResponseWriter, r *http.Request) {
	token := ensureCSRFToken(w, r, sessionSecure())
	h.ts.render(w, "login.html", map[string]any{
		"Error":     r.URL.Query().Get("error"),
		"reset":     r.URL.Query().Get("reset"),
		"CSRFToken": token,
	})
}

func (h *Handler) LoginPOST(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !validateCSRF(r) {
		http.Error(w, "invalid request", http.StatusForbidden)
		return
	}

	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")

	if email == "" || password == "" {
		http.Redirect(w, r, "/login?error=missing_fields", http.StatusSeeOther)
		return
	}

	var id int64
	var hash string
	err := h.DB.QueryRow(
		`SELECT id, password_hash FROM customers WHERE email = ?`, email,
	).Scan(&id, &hash)

	if errors.Is(err, sql.ErrNoRows) || !auth.CheckPassword(hash, password) {
		http.Redirect(w, r, "/login?error=invalid_credentials", http.StatusSeeOther)
		return
	}
	if err != nil {
		slog.Error("login: db query", "err", err)
		http.Redirect(w, r, "/login?error=server_error", http.StatusSeeOther)
		return
	}

	token, err := auth.CreateSession(h.DB, id)
	if err != nil {
		slog.Error("login: create session", "err", err)
		http.Redirect(w, r, "/login?error=server_error", http.StatusSeeOther)
		return
	}

	setSessionCookie(w, token)
	slog.Info("customer logged in", "customer_id", id, "email", email)
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (h *Handler) RegisterGET(w http.ResponseWriter, r *http.Request) {
	token := ensureCSRFToken(w, r, sessionSecure())
	h.ts.render(w, "register.html", map[string]any{
		"Error":     r.URL.Query().Get("error"),
		"CSRFToken": token,
	})
}

func (h *Handler) RegisterPOST(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !validateCSRF(r) {
		http.Error(w, "invalid request", http.StatusForbidden)
		return
	}

	firstName := strings.TrimSpace(r.FormValue("first_name"))
	lastName := strings.TrimSpace(r.FormValue("last_name"))
	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")
	confirm := r.FormValue("confirm_password")

	if firstName == "" || lastName == "" || email == "" || password == "" {
		http.Redirect(w, r, "/register?error=missing_fields", http.StatusSeeOther)
		return
	}
	if !validEmail(email) {
		http.Redirect(w, r, "/register?error=invalid_email", http.StatusSeeOther)
		return
	}
	if password != confirm {
		http.Redirect(w, r, "/register?error=password_mismatch", http.StatusSeeOther)
		return
	}
	if len(password) < 8 {
		http.Redirect(w, r, "/register?error=password_too_short", http.StatusSeeOther)
		return
	}

	var existing int64
	_ = h.DB.QueryRow(`SELECT id FROM customers WHERE email = ?`, email).Scan(&existing)
	if existing != 0 {
		http.Redirect(w, r, "/register?error=email_taken", http.StatusSeeOther)
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		slog.Error("register: hash password", "err", err)
		http.Redirect(w, r, "/register?error=server_error", http.StatusSeeOther)
		return
	}

	stripeCustomerID := ""
	if h.Stripe.Ready() {
		stripeCustomerID, err = payments.CreateCustomer(email, firstName, lastName)
		if err != nil {
			slog.Error("register: create stripe customer", "err", err)
		}
	}

	res, err := h.DB.Exec(
		`INSERT INTO customers (email, password_hash, first_name, last_name, stripe_customer_id) VALUES (?, ?, ?, ?, ?)`,
		email, hash, firstName, lastName, stripeCustomerID,
	)
	if err != nil {
		slog.Error("register: insert customer", "err", err)
		http.Redirect(w, r, "/register?error=server_error", http.StatusSeeOther)
		return
	}

	customerID, _ := res.LastInsertId()

	token, err := auth.CreateSession(h.DB, customerID)
	if err != nil {
		slog.Error("register: create session", "err", err)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	setSessionCookie(w, token)
	slog.Info("customer registered", "customer_id", customerID, "email", email)
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (h *Handler) DashboardGET(w http.ResponseWriter, r *http.Request) {
	customerID := auth.CustomerIDFromContext(r.Context())

	c, err := loadCustomer(h.DB, customerID)
	if err != nil {
		slog.Error("dashboard: load customer", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	sub, err := loadSubscription(h.DB, customerID)
	if err != nil {
		slog.Error("dashboard: load subscription", "err", err)
	}

	invoices, err := loadRecentInvoices(h.DB, customerID)
	if err != nil {
		slog.Error("dashboard: load invoices", "err", err)
	}

	flash := ""
	switch r.URL.Query().Get("checkout") {
	case "success":
		flash = "Your subscription is being activated. It may take a moment to appear."
	case "cancelled":
		flash = "Checkout was cancelled. No charge was made."
	}
	switch r.URL.Query().Get("cancelled") {
	case "1":
		flash = "Your subscription has been cancelled and will not renew. You retain access until the end of the current billing period."
	}
	switch r.URL.Query().Get("error") {
	case "cancel_failed":
		flash = "Could not cancel subscription. Please contact support."
	case "already_cancelling":
		flash = "Your subscription is already scheduled for cancellation."
	case "no_subscription":
		flash = "No active subscription found."
	case "already_subscribed":
		flash = "You already have an active subscription."
	}

	csrfToken := ensureCSRFToken(w, r, sessionSecure())

	h.ts.render(w, "dashboard.html", dashboardData{
		Customer:     c,
		Subscription: sub,
		Invoices:     invoices,
		Flash:        flash,
		CSRFToken:    csrfToken,
	})
}

func (h *Handler) LogoutPOST(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !validateCSRF(r) {
		http.Error(w, "invalid request", http.StatusForbidden)
		return
	}
	cookie, err := r.Cookie("session")
	if err == nil {
		_ = auth.DeleteSession(h.DB, cookie.Value)
	}
	clearSessionCookie(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *Handler) ResetGET(w http.ResponseWriter, r *http.Request) {
	token := ensureCSRFToken(w, r, sessionSecure())
	h.ts.render(w, "reset-request.html", map[string]any{
		"Sent":      r.URL.Query().Get("sent"),
		"Error":     r.URL.Query().Get("error"),
		"CSRFToken": token,
	})
}

func (h *Handler) ResetPOST(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !validateCSRF(r) {
		http.Error(w, "invalid request", http.StatusForbidden)
		return
	}

	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	if email == "" {
		http.Redirect(w, r, "/reset?error=missing_email", http.StatusSeeOther)
		return
	}

	var customerID int64
	err := h.DB.QueryRow(`SELECT id FROM customers WHERE email = ?`, email).Scan(&customerID)

	if err == nil {
		token, tokenErr := auth.CreateResetToken(h.DB, customerID)
		if tokenErr != nil {
			slog.Error("reset: create token", "err", tokenErr)
		} else if h.SMTP.Ready() {
			resetURL := h.BaseURL + "/reset/" + token
			if mailErr := mail.SendPasswordReset(h.SMTP, email, resetURL); mailErr != nil {
				slog.Error("reset: send email", "err", mailErr)
			}
		} else {
			slog.Warn("reset: SMTP not configured, reset token not sent", "email", email)
		}
	}

	http.Redirect(w, r, "/reset?sent=1", http.StatusSeeOther)
}

func (h *Handler) ResetConfirmGET(w http.ResponseWriter, r *http.Request) {
	pathToken := r.PathValue("token")
	csrfToken := ensureCSRFToken(w, r, sessionSecure())
	h.ts.render(w, "reset-confirm.html", map[string]any{
		"Token":     pathToken,
		"Error":     r.URL.Query().Get("error"),
		"CSRFToken": csrfToken,
	})
}

func (h *Handler) ResetConfirmPOST(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !validateCSRF(r) {
		http.Error(w, "invalid request", http.StatusForbidden)
		return
	}

	password := r.FormValue("password")
	confirm := r.FormValue("confirm_password")

	if password == "" {
		http.Redirect(w, r, "/reset/"+token+"?error=missing_password", http.StatusSeeOther)
		return
	}
	if password != confirm {
		http.Redirect(w, r, "/reset/"+token+"?error=password_mismatch", http.StatusSeeOther)
		return
	}
	if len(password) < 8 {
		http.Redirect(w, r, "/reset/"+token+"?error=password_too_short", http.StatusSeeOther)
		return
	}

	customerID, err := auth.ValidateResetToken(h.DB, token)
	if err != nil {
		slog.Warn("reset confirm: invalid token", "err", err)
		http.Redirect(w, r, "/reset/"+token+"?error=invalid_token", http.StatusSeeOther)
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		slog.Error("reset confirm: hash password", "err", err)
		http.Redirect(w, r, "/reset/"+token+"?error=server_error", http.StatusSeeOther)
		return
	}

	_, err = h.DB.Exec(
		`UPDATE customers SET password_hash = ? WHERE id = ?`, hash, customerID,
	)
	if err != nil {
		slog.Error("reset confirm: update password", "err", err)
		http.Redirect(w, r, "/reset/"+token+"?error=server_error", http.StatusSeeOther)
		return
	}

	if err := auth.ConsumeResetToken(h.DB, token); err != nil {
		slog.Error("reset confirm: consume token", "err", err)
	}

	_, _ = h.DB.Exec(`DELETE FROM sessions WHERE customer_id = ?`, customerID)

	slog.Info("password reset completed", "customer_id", customerID)
	http.Redirect(w, r, "/login?reset=1", http.StatusSeeOther)
}

func (h *Handler) CheckoutGET(w http.ResponseWriter, r *http.Request) {
	customerID := auth.CustomerIDFromContext(r.Context())
	priceID := r.URL.Query().Get("plan")
	if priceID == "" {
		http.Error(w, "missing plan parameter", http.StatusBadRequest)
		return
	}

	// Reject price IDs not in the configured set.
	validPrice := false
	for _, id := range h.Stripe.PriceIDs {
		if id == priceID {
			validPrice = true
			break
		}
	}
	if !validPrice {
		http.Error(w, "invalid plan", http.StatusBadRequest)
		return
	}

	// Block customers who already have an active or cancelling subscription.
	var existingCount int
	_ = h.DB.QueryRow(
		`SELECT COUNT(*) FROM subscriptions WHERE customer_id = ? AND status IN ('active', 'cancelling')`,
		customerID,
	).Scan(&existingCount)
	if existingCount > 0 {
		http.Redirect(w, r, "/dashboard?error=already_subscribed", http.StatusSeeOther)
		return
	}

	var stripeCustomerID string
	err := h.DB.QueryRow(
		`SELECT stripe_customer_id FROM customers WHERE id = ?`, customerID,
	).Scan(&stripeCustomerID)
	if err != nil {
		slog.Error("checkout: load customer", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	successURL := h.BaseURL + "/dashboard?checkout=success"
	cancelURL := h.BaseURL + "/dashboard?checkout=cancelled"

	url, err := payments.CreateCheckoutSession(h.Stripe, customerID, stripeCustomerID, priceID, successURL, cancelURL)
	if err != nil {
		slog.Error("checkout: create session", "err", err)
		http.Error(w, "could not create checkout session", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, url, http.StatusSeeOther)
}

func (h *Handler) WebhookPOST(w http.ResponseWriter, r *http.Request) {
	const maxBodyBytes = 65536
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}

	sig := r.Header.Get("Stripe-Signature")
	event, err := webhook.ConstructEvent(payload, sig, h.Stripe.WebhookSecret)
	if err != nil {
		slog.Warn("webhook: signature verification failed", "err", err)
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}

	slog.Info("webhook received", "type", event.Type)

	switch event.Type {
	case "checkout.session.completed":
		if err := payments.HandleCheckoutCompleted(h.DB, h.Stripe, event.Data.Raw); err != nil {
			slog.Error("webhook: checkout.session.completed", "err", err)
		}
	case "invoice.paid":
		if err := payments.HandleInvoicePaid(h.DB, event.Data.Raw); err != nil {
			slog.Error("webhook: invoice.paid", "err", err)
		}
	case "invoice.payment_failed":
		if err := payments.HandleInvoicePaymentFailed(h.DB, event.Data.Raw); err != nil {
			slog.Error("webhook: invoice.payment_failed", "err", err)
		}
	case "customer.subscription.deleted":
		if err := payments.HandleSubscriptionDeleted(h.DB, event.Data.Raw); err != nil {
			slog.Error("webhook: customer.subscription.deleted", "err", err)
		}
	default:
		slog.Info("webhook: unhandled event type", "type", event.Type)
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) CancelPOST(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !validateCSRF(r) {
		http.Error(w, "invalid request", http.StatusForbidden)
		return
	}

	customerID := auth.CustomerIDFromContext(r.Context())

	var stripeSubID, subStatus string
	err := h.DB.QueryRow(
		`SELECT stripe_subscription_id, status FROM subscriptions
		WHERE customer_id = ? AND status IN ('active', 'cancelling')
		ORDER BY created_at DESC LIMIT 1`,
		customerID,
	).Scan(&stripeSubID, &subStatus)
	if err != nil {
		slog.Error("cancel: find subscription", "err", err)
		http.Redirect(w, r, "/dashboard?error=no_subscription", http.StatusSeeOther)
		return
	}
	if subStatus == "cancelling" {
		http.Redirect(w, r, "/dashboard?error=already_cancelling", http.StatusSeeOther)
		return
	}

	if err := payments.CancelSubscription(stripeSubID); err != nil {
		slog.Error("cancel: stripe cancel", "err", err)
		http.Redirect(w, r, "/dashboard?error=cancel_failed", http.StatusSeeOther)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = h.DB.Exec(
		`UPDATE subscriptions SET status = 'cancelling', updated_at = ? WHERE stripe_subscription_id = ?`,
		now, stripeSubID,
	)

	slog.Info("subscription scheduled for cancellation", "customer_id", customerID, "stripe_sub_id", stripeSubID)
	http.Redirect(w, r, "/dashboard?cancelled=1", http.StatusSeeOther)
}
