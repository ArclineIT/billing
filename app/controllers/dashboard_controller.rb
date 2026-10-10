class DashboardController < ApplicationController
  def show
    @user = Current.user
    @subscription = @user.subscriptions.current.first
    @invoices = @user.invoices.recent.limit(5)
    flash.now[:notice] = status_message if status_message
  end

  private

  def status_message
    if params[:checkout] == "success"
      "Your subscription is being activated. It may take a moment to appear."
    elsif params[:checkout] == "cancelled"
      "Checkout was cancelled. No charge was made."
    elsif params[:cancelled] == "1"
      "Your subscription has been cancelled and will not renew. " \
        "You retain access until the end of the current billing period."
    elsif params[:error] == "cancel_failed"
      "Could not cancel subscription. Please contact support."
    elsif params[:error] == "already_cancelling"
      "Your subscription is already scheduled for cancellation."
    elsif params[:error] == "no_subscription"
      "No active subscription found."
    elsif params[:error] == "already_subscribed"
      "You already have an active subscription."
    elsif params[:error] == "checkout_failed"
      "Could not start checkout. Please try again or contact support."
    end
  end
end
