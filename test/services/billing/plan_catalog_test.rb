require "test_helper"

class Billing::PlanCatalogTest < ActiveSupport::TestCase
  test "resolves a plan name from a configured price" do
    ENV["STRIPE_PRICE_VPS_1"] = "price_vps_1"

    assert_equal "vps_1", Billing::PlanCatalog.plan_name_for("price_vps_1")
    assert Billing::PlanCatalog.valid_price?("price_vps_1")
  ensure
    ENV.delete("STRIPE_PRICE_VPS_1")
  end

  test "rejects a price that is not configured" do
    assert_not Billing::PlanCatalog.valid_price?("price_unknown")
    assert_not Billing::PlanCatalog.valid_price?(nil)
  end
end
