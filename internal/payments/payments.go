package payments

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	stripelib "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"
	stripecustomer "github.com/stripe/stripe-go/v81/customer"
	"github.com/stripe/stripe-go/v81/subscription"
)

// Config holds Stripe credentials loaded from environment variables.
type Config struct {
	SecretKey     string
	WebhookSecret string
	PriceIDs      map[string]string // plan key → Stripe price ID
}

// ConfigFromEnv reads Stripe settings from environment variables.
func ConfigFromEnv() Config {
	return Config{
		SecretKey:     os.Getenv("STRIPE_SECRET_KEY"),
		WebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
		PriceIDs: map[string]string{
			"shared_starter":  os.Getenv("STRIPE_PRICE_SHARED_STARTER"),
			"shared_pro":      os.Getenv("STRIPE_PRICE_SHARED_PRO"),
			"shared_business": os.Getenv("STRIPE_PRICE_SHARED_BUSINESS"),
			"wp_starter":      os.Getenv("STRIPE_PRICE_WP_STARTER"),
			"wp_pro":          os.Getenv("STRIPE_PRICE_WP_PRO"),
			"wp_business":     os.Getenv("STRIPE_PRICE_WP_BUSINESS"),
			"vps_1":           os.Getenv("STRIPE_PRICE_VPS_1"),
			"vps_2":           os.Getenv("STRIPE_PRICE_VPS_2"),
			"vps_3":           os.Getenv("STRIPE_PRICE_VPS_3"),
			"vps_4":           os.Getenv("STRIPE_PRICE_VPS_4"),
		},
	}
}

// Ready reports whether the required Stripe credentials are set.
func (c Config) Ready() bool {
	return c.SecretKey != "" && c.WebhookSecret != ""
}

// PlanName returns the human-readable plan key for a Stripe price ID,
// or an empty string if the price ID isn't in the configured map.
func (c Config) PlanName(priceID string) string {
	for name, id := range c.PriceIDs {
		if id == priceID {
			return name
		}
	}
	return ""
}

// CreateCustomer creates a Stripe customer and returns the Stripe customer ID.
func CreateCustomer(email, firstName, lastName string) (string, error) {
	params := &stripelib.CustomerParams{
		Email: stripelib.String(email),
		Name:  stripelib.String(firstName + " " + lastName),
	}
	c, err := stripecustomer.New(params)
	if err != nil {
		return "", fmt.Errorf("stripe create customer: %w", err)
	}
	return c.ID, nil
}

// CreateCheckoutSession creates a Stripe Checkout session and returns the hosted URL.
func CreateCheckoutSession(
	cfg Config,
	customerID int64,
	stripeCustomerID,
	priceID,
	successURL,
	cancelURL string,
) (string, error) {
	params := &stripelib.CheckoutSessionParams{
		Customer: stripelib.String(stripeCustomerID),
		Mode:     stripelib.String(string(stripelib.CheckoutSessionModeSubscription)),
		LineItems: []*stripelib.CheckoutSessionLineItemParams{
			{
				Price:    stripelib.String(priceID),
				Quantity: stripelib.Int64(1),
			},
		},
		SuccessURL: stripelib.String(successURL),
		CancelURL:  stripelib.String(cancelURL),
		Metadata: map[string]string{
			"customer_id": fmt.Sprintf("%d", customerID),
		},
	}

	s, err := session.New(params)
	if err != nil {
		return "", fmt.Errorf("stripe checkout session: %w", err)
	}
	return s.URL, nil
}

// CancelSubscription schedules a Stripe subscription to cancel at the end of
// the current billing period rather than immediately.
func CancelSubscription(stripeSubID string) error {
	params := &stripelib.SubscriptionParams{
		CancelAtPeriodEnd: stripelib.Bool(true),
	}
	_, err := subscription.Update(stripeSubID, params)
	if err != nil {
		return fmt.Errorf("stripe cancel subscription: %w", err)
	}
	return nil
}

// HandleCheckoutCompleted processes a checkout.session.completed webhook event.
func HandleCheckoutCompleted(db *sql.DB, cfg Config, raw json.RawMessage) error {
	var cs stripelib.CheckoutSession
	if err := json.Unmarshal(raw, &cs); err != nil {
		return fmt.Errorf("unmarshal checkout session: %w", err)
	}

	if cs.Subscription == nil {
		return nil
	}

	customerIDStr, ok := cs.Metadata["customer_id"]
	if !ok {
		return fmt.Errorf("missing customer_id in metadata")
	}

	var customerID int64
	fmt.Sscanf(customerIDStr, "%d", &customerID)

	// Fetch the full subscription to get the price ID and plan name.
	priceID := ""
	planName := ""
	sub, err := subscription.Get(cs.Subscription.ID, nil)
	if err != nil {
		slog.Warn("checkout completed: fetch subscription from stripe", "err", err)
	} else if len(sub.Items.Data) > 0 {
		priceID = sub.Items.Data[0].Price.ID
		planName = cfg.PlanName(priceID)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = db.Exec(
		`INSERT INTO subscriptions
			(customer_id, stripe_subscription_id, stripe_price_id, plan_name, status, current_period_end, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'active', '', ?, ?)
		ON CONFLICT(stripe_subscription_id) DO UPDATE SET
			status = 'active',
			stripe_price_id = excluded.stripe_price_id,
			plan_name = excluded.plan_name,
			updated_at = excluded.updated_at`,
		customerID, cs.Subscription.ID, priceID, planName, now, now,
	)
	return err
}

// HandleInvoicePaid processes an invoice.paid webhook event.
func HandleInvoicePaid(db *sql.DB, raw json.RawMessage) error {
	var inv stripelib.Invoice
	if err := json.Unmarshal(raw, &inv); err != nil {
		return fmt.Errorf("unmarshal invoice: %w", err)
	}

	var customerID int64
	if inv.Customer != nil {
		err := db.QueryRow(
			`SELECT id FROM customers WHERE stripe_customer_id = ?`, inv.Customer.ID,
		).Scan(&customerID)
		if err != nil {
			return fmt.Errorf("find customer for stripe id %s: %w", inv.Customer.ID, err)
		}
	}

	pdfURL := ""
	if inv.InvoicePDF != "" {
		pdfURL = inv.InvoicePDF
	}

	periodStart := time.Unix(inv.PeriodStart, 0).UTC().Format(time.RFC3339)
	periodEnd := time.Unix(inv.PeriodEnd, 0).UTC().Format(time.RFC3339)
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := db.Exec(
		`INSERT INTO invoices
			(customer_id, stripe_invoice_id, amount_cents, currency, status, invoice_pdf_url, period_start, period_end, created_at)
		VALUES (?, ?, ?, ?, 'paid', ?, ?, ?, ?)
		ON CONFLICT(stripe_invoice_id) DO UPDATE SET
			status = 'paid',
			invoice_pdf_url = excluded.invoice_pdf_url`,
		customerID, inv.ID, inv.AmountPaid, string(inv.Currency), pdfURL, periodStart, periodEnd, now,
	)
	if err != nil {
		return fmt.Errorf("upsert invoice: %w", err)
	}

	if inv.Subscription != nil {
		_, _ = db.Exec(
			`UPDATE subscriptions SET current_period_end = ?, updated_at = ? WHERE stripe_subscription_id = ?`,
			periodEnd, now, inv.Subscription.ID,
		)
	}

	return nil
}

// HandleInvoicePaymentFailed processes an invoice.payment_failed webhook event.
func HandleInvoicePaymentFailed(db *sql.DB, raw json.RawMessage) error {
	var inv stripelib.Invoice
	if err := json.Unmarshal(raw, &inv); err != nil {
		return fmt.Errorf("unmarshal invoice: %w", err)
	}

	if inv.Subscription == nil {
		return nil
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(
		`UPDATE subscriptions SET status = 'past_due', updated_at = ? WHERE stripe_subscription_id = ?`,
		now, inv.Subscription.ID,
	)
	return err
}

// HandleSubscriptionDeleted processes a customer.subscription.deleted webhook event.
func HandleSubscriptionDeleted(db *sql.DB, raw json.RawMessage) error {
	var sub stripelib.Subscription
	if err := json.Unmarshal(raw, &sub); err != nil {
		return fmt.Errorf("unmarshal subscription: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(
		`UPDATE subscriptions SET status = 'cancelled', updated_at = ? WHERE stripe_subscription_id = ?`,
		now, sub.ID,
	)
	return err
}
