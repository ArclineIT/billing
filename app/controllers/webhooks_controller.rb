class WebhooksController < ApplicationController
  allow_unauthenticated_access
  skip_before_action :verify_authenticity_token
  skip_before_action :restrict_to_modern_browsers

  def create
    secret = ENV["STRIPE_WEBHOOK_SECRET"]
    if secret.blank?
      Rails.logger.error("Stripe webhook received but STRIPE_WEBHOOK_SECRET is not configured")
      return head :service_unavailable
    end

    event = Stripe::Webhook.construct_event(
      request.body.read,
      request.env["HTTP_STRIPE_SIGNATURE"],
      secret
    )

    Billing::WebhookProcessor.new(event).call
    head :ok
  rescue JSON::ParserError, Stripe::SignatureVerificationError => e
    Rails.logger.warn("Stripe webhook rejected: #{e.message}")
    head :bad_request
  end
end
