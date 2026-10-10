module Billing
  # Creates a matching Stripe customer for a locally-registered user and stores
  # the resulting customer ID. Failures are logged and swallowed so registration
  # can still succeed when Stripe is unreachable or unconfigured.
  class CreateCustomer
    def self.call(user)
      return if Stripe.api_key.blank?

      customer = Stripe::Customer.create(
        email: user.email_address,
        name: user.full_name
      )
      user.update_column(:stripe_customer_id, customer.id)
      customer.id
    rescue Stripe::StripeError => e
      Rails.logger.error("Stripe customer creation failed for user #{user.id}: #{e.message}")
      nil
    end
  end
end
