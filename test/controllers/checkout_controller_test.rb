require "test_helper"

class CheckoutControllerTest < ActionDispatch::IntegrationTest
  PRICE_ID = "price_test_starter".freeze

  setup do
    ENV["STRIPE_PRICE_SHARED_STARTER"] = PRICE_ID
  end

  teardown do
    ENV.delete("STRIPE_PRICE_SHARED_STARTER")
  end

  test "redirects an unknown plan back to the dashboard" do
    sign_in_as users(:two)

    get checkout_path, params: { plan: "price_unknown" }

    assert_redirected_to dashboard_path(error: "checkout_failed")
  end

  test "blocks a user who already has a billable subscription" do
    sign_in_as users(:one)

    get checkout_path, params: { plan: PRICE_ID }

    assert_redirected_to dashboard_path(error: "already_subscribed")
  end

  test "redirects to the hosted Stripe checkout for a valid plan" do
    user = users(:two)
    user.update_column(:stripe_customer_id, "cus_test")
    sign_in_as user

    session = Struct.new(:url).new("https://checkout.stripe.com/c/pay/test")
    stub_method(Stripe::Checkout::Session, :create, session) do
      get checkout_path, params: { plan: PRICE_ID }
    end

    assert_redirected_to "https://checkout.stripe.com/c/pay/test"
  end
end
