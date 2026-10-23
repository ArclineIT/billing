require "test_helper"

class Billing::PlanCatalogTest < ActiveSupport::TestCase
  test "resolves a plan name from a configured price" do
    ENV["STRIPE_PRICE_MANAGED_SECURITY"] = "price_managed_security"

    assert_equal "managed_security", Billing::PlanCatalog.plan_name_for("price_managed_security")
    assert Billing::PlanCatalog.valid_price?("price_managed_security")
  ensure
    ENV.delete("STRIPE_PRICE_MANAGED_SECURITY")
  end

  test "rejects a price that is not configured" do
    assert_not Billing::PlanCatalog.valid_price?("price_unknown")
    assert_not Billing::PlanCatalog.valid_price?(nil)
  end
end
