require "test_helper"

class SubscriptionTest < ActiveSupport::TestCase
  test "defaults to active" do
    assert Subscription.new.active?
  end

  test "billable scope includes active subscriptions" do
    assert_includes Subscription.billable, subscriptions(:one)
  end

  test "billable scope excludes cancelled subscriptions" do
    subscriptions(:one).update!(status: :cancelled)

    assert_not_includes Subscription.billable, subscriptions(:one)
  end
end
