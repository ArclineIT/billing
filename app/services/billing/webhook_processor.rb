module Billing
  # Translates verified Stripe webhook events into local subscription and
  # invoice records. Unknown event types are ignored.
  class WebhookProcessor
    def initialize(event)
      @event = event
    end

    def call
      case @event.type
      when "checkout.session.completed"
        handle_checkout_completed
      when "invoice.paid"
        handle_invoice_paid
      when "invoice.payment_failed"
        handle_invoice_payment_failed
      when "customer.subscription.deleted"
        handle_subscription_deleted
      else
        Rails.logger.info("Ignoring unhandled Stripe event: #{@event.type}")
      end
    end

    private

    def object
      @event.data.object
    end

    def handle_checkout_completed
      session = object
      stripe_subscription_id = session.subscription
      return if stripe_subscription_id.blank?

      user = user_for_checkout(session)
      return if user.nil?

      price_id = fetch_price_id(stripe_subscription_id)

      subscription = user.subscriptions.find_or_initialize_by(
        stripe_subscription_id: stripe_subscription_id
      )
      subscription.assign_attributes(
        stripe_price_id: price_id,
        plan_name: Billing::PlanCatalog.plan_name_for(price_id).to_s,
        status: :active
      )
      subscription.save!
    end

    def handle_invoice_paid
      invoice = object
      user = User.find_by(stripe_customer_id: invoice.customer)
      return if user.nil?

      record = user.invoices.find_or_initialize_by(stripe_invoice_id: invoice.id)
      record.assign_attributes(
        amount_cents: invoice.amount_paid.to_i,
        currency: invoice.currency.presence || "usd",
        status: :paid,
        invoice_pdf_url: invoice.invoice_pdf.to_s,
        period_start: timestamp(invoice.period_start),
        period_end: timestamp(invoice.period_end)
      )
      record.save!

      return if invoice.subscription.blank?

      user.subscriptions
          .find_by(stripe_subscription_id: invoice.subscription)
          &.update(current_period_end: timestamp(invoice.period_end))
    end

    def handle_invoice_payment_failed
      invoice = object
      return if invoice.subscription.blank?

      Subscription.find_by(stripe_subscription_id: invoice.subscription)&.update(status: :past_due)
    end

    def handle_subscription_deleted
      subscription = object
      Subscription.find_by(stripe_subscription_id: subscription.id)&.update(status: :cancelled)
    end

    def user_for_checkout(session)
      user_id = session.metadata&.[]("user_id")
      User.find_by(id: user_id) || User.find_by(stripe_customer_id: session.customer)
    end

    def fetch_price_id(stripe_subscription_id)
      subscription = Stripe::Subscription.retrieve(stripe_subscription_id)
      subscription.items.data.first&.price&.id.to_s
    rescue Stripe::StripeError => e
      Rails.logger.warn("Could not retrieve Stripe subscription #{stripe_subscription_id}: #{e.message}")
      ""
    end

    def timestamp(value)
      return nil if value.blank?

      Time.zone.at(value)
    end
  end
end
