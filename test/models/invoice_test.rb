require "test_helper"

class InvoiceTest < ActiveSupport::TestCase
  test "formats a USD amount" do
    assert_equal "$19.00", invoices(:one).amount_display
  end

  test "formats a non-USD amount with its currency code" do
    invoice = invoices(:one)
    invoice.currency = "eur"
    invoice.amount_cents = 1234

    assert_equal "12.34 EUR", invoice.amount_display
  end
end
