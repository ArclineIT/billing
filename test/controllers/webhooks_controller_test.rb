require "test_helper"

class WebhooksControllerTest < ActionDispatch::IntegrationTest
  setup { ENV["STRIPE_WEBHOOK_SECRET"] = "whsec_test" }
  teardown { ENV.delete("STRIPE_WEBHOOK_SECRET") }

  test "rejects a payload with an invalid signature" do
    post webhook_path, params: "{}", headers: { "HTTP_STRIPE_SIGNATURE" => "t=1,v1=bad" }

    assert_response :bad_request
  end

  test "returns service unavailable when no webhook secret is configured" do
    ENV.delete("STRIPE_WEBHOOK_SECRET")

    post webhook_path, params: "{}", headers: { "HTTP_STRIPE_SIGNATURE" => "t=1,v1=bad" }

    assert_response :service_unavailable
  end

  test "marks a subscription cancelled on customer.subscription.deleted" do
    event = Stripe::Event.construct_from(
      type: "customer.subscription.deleted",
      data: { object: { id: subscriptions(:one).stripe_subscription_id } }
    )

    stub_method(Stripe::Webhook, :construct_event, event) do
      post webhook_path, params: "{}", headers: { "HTTP_STRIPE_SIGNATURE" => "verified" }
    end

    assert_response :ok
    assert_equal "cancelled", subscriptions(:one).reload.status
  end

  test "records a paid invoice on invoice.paid" do
    user = users(:one)
    user.update_column(:stripe_customer_id, "cus_test")

    event = Stripe::Event.construct_from(
      type: "invoice.paid",
      data: { object: {
        id: "in_webhook_1",
        customer: "cus_test",
        amount_paid: 4200,
        currency: "usd",
        invoice_pdf: "https://example.com/in_webhook_1.pdf",
        period_start: Time.utc(2026, 10, 1).to_i,
        period_end: Time.utc(2026, 11, 1).to_i,
        subscription: subscriptions(:one).stripe_subscription_id
      } }
    )

    assert_difference "Invoice.count", 1 do
      stub_method(Stripe::Webhook, :construct_event, event) do
        post webhook_path, params: "{}", headers: { "HTTP_STRIPE_SIGNATURE" => "verified" }
      end
    end

    assert_response :ok
    invoice = Invoice.find_by(stripe_invoice_id: "in_webhook_1")
    assert_equal "paid", invoice.status
    assert_equal 4200, invoice.amount_cents
    assert_equal "$42.00", invoice.amount_display
  end
end
