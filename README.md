# Arcline Billing Portal

A lightweight billing and subscription management portal for Arcline IT customers. Customers can register, view their subscriptions and invoices, and manage their plans via Stripe Checkout.

## Features

- Customer registration and login with bcrypt password hashing
- Subscription management via Stripe (checkout, cancellation)
- Invoice history with PDF links
- Password reset via email
- Stripe webhook processing
- Rate limiting and security headers

## Requirements

- Go 1.22+
- A [Stripe](https://stripe.com) account
- SMTP credentials for email (password resets)

## Setup

1. Copy the example environment file and fill in your values:

   ```bash
   cp .env.example .env
   ```

2. Build the binary:

   ```bash
   go build -o billing .
   ```

3. Run:

   ```bash
   ./billing
   ```

   The server listens on `0.0.0.0:8082` by default. The SQLite database and schema are created automatically on first run.

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8082` | HTTP listen port |
| `DATABASE_PATH` | `./arcline-billing.db` | SQLite database file path |
| `BASE_URL` | — | Public base URL (e.g. `https://client.arclineit.com`) — used for redirect URLs and password reset links |
| `SESSION_SECURE` | `true` | Set `Secure` flag on session cookies (disable for local HTTP dev) |
| `STRIPE_SECRET_KEY` | — | Stripe secret API key |
| `STRIPE_WEBHOOK_SECRET` | — | Stripe webhook signing secret |
| `SMTP_HOST` | — | SMTP server hostname |
| `SMTP_PORT` | — | SMTP server port |
| `SMTP_USER` | — | SMTP username |
| `SMTP_PASS` | — | SMTP password |
| `SMTP_FROM` | — | From address for outgoing email |

### Stripe Price IDs

Configure the price IDs for each plan tier:

| Variable | Plan |
|---|---|
| `STRIPE_PRICE_SHARED_STARTER` | Shared Hosting — Starter |
| `STRIPE_PRICE_SHARED_PRO` | Shared Hosting — Pro |
| `STRIPE_PRICE_SHARED_BUSINESS` | Shared Hosting — Business |
| `STRIPE_PRICE_WP_STARTER` | WordPress Hosting — Starter |
| `STRIPE_PRICE_WP_PRO` | WordPress Hosting — Pro |
| `STRIPE_PRICE_WP_BUSINESS` | WordPress Hosting — Business |
| `STRIPE_PRICE_VPS_1` | VPS — Tier 1 |
| `STRIPE_PRICE_VPS_2` | VPS — Tier 2 |
| `STRIPE_PRICE_VPS_3` | VPS — Tier 3 |
| `STRIPE_PRICE_VPS_4` | VPS — Tier 4 |

## Stripe Webhooks

Point your Stripe webhook to `POST /webhook`. The following events are handled:

- `checkout.session.completed` — creates a subscription record
- `invoice.paid` — records a paid invoice
- `invoice.payment_failed` — updates subscription status
- `customer.subscription.deleted` — marks subscription as cancelled