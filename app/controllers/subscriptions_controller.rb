class SubscriptionsController < ApplicationController
  def cancel
    subscription = Current.user.subscriptions.billable.current.first

    if subscription.nil?
      redirect_to dashboard_path(error: "no_subscription") and return
    end

    if subscription.cancelling?
      redirect_to dashboard_path(error: "already_cancelling") and return
    end

    Billing::CancelSubscription.call(subscription.stripe_subscription_id)
    subscription.update(status: :cancelling)

    redirect_to dashboard_path(cancelled: 1)
  rescue Stripe::StripeError => e
    Rails.logger.error("Cancel subscription failed for user #{Current.user.id}: #{e.message}")
    redirect_to dashboard_path(error: "cancel_failed")
  end
end
