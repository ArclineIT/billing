# Configure the Stripe client and expose the price IDs for each plan.
#
# The price IDs are read from the environment at runtime (see Billing::PlanCatalog)
# so tests and local development can supply their own values via .env.
Stripe.api_key = ENV["STRIPE_SECRET_KEY"].presence
