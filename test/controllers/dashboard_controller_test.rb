require "test_helper"

class DashboardControllerTest < ActionDispatch::IntegrationTest
  test "requires authentication" do
    get dashboard_path
    assert_redirected_to new_session_path
  end

  test "renders the current user's subscription and invoices" do
    user = users(:one)
    sign_in_as user

    get dashboard_path

    assert_response :success
    assert_select ".dash-header__name", text: user.first_name
    assert_select ".dash-header__email", text: user.email_address
    assert_select ".status-badge", text: "active"
    assert_select ".inv-table__row", count: 1
  end

  test "shows an empty state without a subscription" do
    user = users(:two)
    sign_in_as user

    get dashboard_path

    assert_response :success
    assert_select ".dash-empty", text: "No active subscription."
  end

  test "translates the checkout query params into a flash message" do
    sign_in_as users(:one)

    get dashboard_path(checkout: "success")

    assert_select ".flash", text: /being activated/
  end
end
