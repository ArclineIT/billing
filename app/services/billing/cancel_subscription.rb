module Billing
  # Schedules a Stripe subscription to cancel at the end of the current billing
  # period rather than immediately.
  class CancelSubscription
    def self.call(stripe_subscription_id)
      Stripe::Subscription.update(stripe_subscription_id, cancel_at_period_end: true)
    end
  end
end
