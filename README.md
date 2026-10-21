# Arcline Billing Portal

A billing and subscription management portal for Arcline IT customers. Customers
register, view their subscriptions and invoices, and manage their plans via
Stripe Checkout.

This is the Ruby on Rails rewrite of the original Go service (preserved in the
repository history).

## Features

- Customer registration and login (Rails 8 built-in authentication, bcrypt)
- Subscription management via Stripe (Checkout, cancel-at-period-end)
- Invoice history with PDF links
- Password reset via email
- Stripe webhook processing with signature verification
- Rate limiting on login and password reset

## Requirements

- Ruby 4.0+
- SQLite 3 (bundled via the `sqlite3` gem)
- A [Stripe](https://stripe.com) account
- SMTP credentials for email (password resets)

## Setup

1. Install dependencies and prepare the database:

   ```bash
   bin/setup
   ```

   Or manually:

   ```bash
   bundle install
   bin/rails db:prepare
   ```

2. Copy the example environment file and fill in your values:

   ```bash
   cp .env.example .env
   ```

3. Run the app (defaults to port 3000, or `PORT` if set):

   ```bash
   bin/rails server
   ```

   The SQLite database is created automatically on first run.

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `PORT` | `3000` | HTTP listen port (the original service used `8082`) |
| `DATABASE_PATH` | `storage/<env>.sqlite3` | SQLite database file path |
| `BASE_URL` | — | Public base URL (e.g. `https://client.arcline.it`) — used for mailer links and Stripe redirect URLs |
| `STRIPE_SECRET_KEY` | — | Stripe secret API key |
| `STRIPE_WEBHOOK_SECRET` | — | Stripe webhook signing secret |
| `SMTP_HOST` | — | SMTP server hostname |
| `SMTP_PORT` | `587` | SMTP server port |
| `SMTP_USER` | — | SMTP username |
| `SMTP_PASS` | — | SMTP password |
| `SMTP_FROM` | `support@arcline.it` | From address for outgoing email |

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

## Tests

```bash
bin/rails test
bin/rubocop
bin/brakeman
```

## Maintenance

Login sessions expire after 30 days. Purge stale rows on a schedule (e.g. cron):

```bash
bin/rails sessions:purge
```
