class CheckoutController < ApplicationController
  def show
    price_id = params[:plan]

    unless Billing::PlanCatalog.valid_price?(price_id)
      redirect_to dashboard_path(error: "checkout_failed") and return
    end

    if Current.user.subscriptions.billable.exists?
      redirect_to dashboard_path(error: "already_subscribed") and return
    end

    url = Billing::CreateCheckoutSession.call(
      user: Current.user,
      price_id: price_id,
      success_url: dashboard_url(checkout: "success"),
      cancel_url: dashboard_url(checkout: "cancelled")
    )

    redirect_to url, allow_other_host: true
  rescue Stripe::StripeError => e
    Rails.logger.error("Checkout session failed for user #{Current.user.id}: #{e.message}")
    redirect_to dashboard_path(error: "checkout_failed")
  end
end
