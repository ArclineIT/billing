package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	stripelib "github.com/stripe/stripe-go/v81"
	"golang.org/x/time/rate"

	"arclineit.com/billing/internal/auth"
	"arclineit.com/billing/internal/db"
	"arclineit.com/billing/internal/mail"
	"arclineit.com/billing/internal/payments"
	"arclineit.com/billing/internal/web"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	if err := godotenv.Load(); err != nil {
		slog.Info(".env not found, using system environment")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:" + port
	}

	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "./arcline-billing.db"
	}

	database, err := db.Open(dbPath)
	if err != nil {
		slog.Error("failed to open database", "err", err)
		os.Exit(1)
	}
	defer database.Close()

	smtpCfg := mail.ConfigFromEnv()
	if !smtpCfg.Ready() {
		slog.Warn("SMTP not configured: password reset emails will not be sent. Set SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS, SMTP_FROM.")
	}

	stripeCfg := payments.ConfigFromEnv()
	if !stripeCfg.Ready() {
		slog.Warn("Stripe not configured: payments will not work. Set STRIPE_SECRET_KEY and STRIPE_WEBHOOK_SECRET.")
	} else {
		stripelib.Key = stripeCfg.SecretKey
	}

	h, err := web.New(database, stripeCfg, smtpCfg, baseURL)
	if err != nil {
		slog.Error("failed to init handlers", "err", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()

	// Static assets.
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Public routes.
	mux.HandleFunc("GET /{$}", h.IndexHandler)
	mux.HandleFunc("GET /login", h.LoginGET)
	mux.HandleFunc("POST /login", h.LoginPOST)
	mux.HandleFunc("GET /register", h.RegisterGET)
	mux.HandleFunc("POST /register", h.RegisterPOST)
	mux.HandleFunc("POST /logout", h.LogoutPOST)
	mux.HandleFunc("GET /reset", h.ResetGET)
	mux.HandleFunc("POST /reset", h.ResetPOST)
	mux.HandleFunc("GET /reset/{token}", h.ResetConfirmGET)
	mux.HandleFunc("POST /reset/{token}", h.ResetConfirmPOST)

	// Stripe webhook — no auth (Stripe signs the payload instead).
	mux.HandleFunc("POST /webhook", h.WebhookPOST)

	// Authenticated routes.
	authed := auth.Middleware(database)
	mux.Handle("GET /dashboard", authed(http.HandlerFunc(h.DashboardGET)))
	mux.Handle("GET /checkout", authed(http.HandlerFunc(h.CheckoutGET)))
	mux.Handle("POST /cancel", authed(http.HandlerFunc(h.CancelPOST)))

	// Rate limiter: 60 req/s sustained, burst 120.
	rl := web.NewRateLimiter(rate.Limit(60), 120)
	handler := web.BuildMiddleware(mux, rl, 10*time.Second)

	addr := "0.0.0.0:" + port
	slog.Info("arcline-billing listening", "addr", "http://"+addr)

	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	// Background job: purge expired sessions and password reset tokens every 24 hours.
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := db.PurgeExpired(database); err != nil {
					slog.Error("purge expired rows", "err", err)
				} else {
					slog.Info("purged expired sessions and reset tokens")
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	<-ctx.Done()
	stop()
	slog.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "err", err)
		os.Exit(1)
	}
	slog.Info("shutdown complete")
}
