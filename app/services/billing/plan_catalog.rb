module Billing
  # Maps the human-readable plan keys used throughout the app to the Stripe
  # price IDs configured via environment variables.
  module PlanCatalog
    ENV_KEYS = {
      "shared_starter"  => "STRIPE_PRICE_SHARED_STARTER",
      "shared_pro"      => "STRIPE_PRICE_SHARED_PRO",
      "shared_business" => "STRIPE_PRICE_SHARED_BUSINESS",
      "wp_starter"      => "STRIPE_PRICE_WP_STARTER",
      "wp_pro"          => "STRIPE_PRICE_WP_PRO",
      "wp_business"     => "STRIPE_PRICE_WP_BUSINESS",
      "vps_1"           => "STRIPE_PRICE_VPS_1",
      "vps_2"           => "STRIPE_PRICE_VPS_2",
      "vps_3"           => "STRIPE_PRICE_VPS_3",
      "vps_4"           => "STRIPE_PRICE_VPS_4"
    }.freeze

    module_function

    # plan key => Stripe price ID, omitting any tiers that aren't configured.
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
