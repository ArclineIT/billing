module Billing
  # Maps the human-readable plan keys used throughout the app to the Stripe
  # price IDs configured via environment variables.
  module PlanCatalog
    ENV_KEYS = {
      "managed_security" => "STRIPE_PRICE_MANAGED_SECURITY"
    }.freeze

    module_function

    # plan key => Stripe price ID, omitting any plans that aren't configured.
    def prices
      ENV_KEYS.each_with_object({}) do |(plan, env_key), result|
        value = ENV[env_key]
        result[plan] = value if value.present?
      end
    end

    def valid_price?(price_id)
      price_id.present? && prices.value?(price_id)
    end

    def plan_name_for(price_id)
      prices.key(price_id)
    end
  end
end
