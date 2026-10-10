module Billing
  # Starts a Stripe Checkout session for a subscription and returns the hosted
  # checkout URL.
  class CreateCheckoutSession
    def self.call(user:, price_id:, success_url:, cancel_url:)
      session = Stripe::Checkout::Session.create(
        customer: user.stripe_customer_id,
        mode: "subscription",
        line_items: [ { price: price_id, quantity: 1 } ],
        success_url: success_url,
        cancel_url: cancel_url,
        metadata: { user_id: user.id }
      )

      session.url
    end
  end
end
