require "test_helper"

class RegistrationsControllerTest < ActionDispatch::IntegrationTest
  test "new" do
    get new_registration_path
    assert_response :success
    assert_select "form input[name='user[email_address]']"
  end

  test "creates a user and signs them in" do
    assert_difference "User.count", 1 do
      post registration_path, params: { user: {
        first_name: "Ada",
        last_name: "Lovelace",
        email_address: "ada@example.com",
        password: "password123",
        password_confirmation: "password123"
      } }
    end

    assert_redirected_to dashboard_path
    assert cookies[:session_id]
    assert_equal "ada@example.com", User.order(:id).last.email_address
  end

  test "rejects a password shorter than eight characters" do
    assert_no_difference "User.count" do
      post registration_path, params: { user: {
        first_name: "Ada",
        last_name: "Lovelace",
        email_address: "ada2@example.com",
        password: "short",
        password_confirmation: "short"
      } }
    end

    assert_response :unprocessable_entity
  end

  test "rejects a duplicate email address" do
    assert_no_difference "User.count" do
      post registration_path, params: { user: {
        first_name: "Test",
        last_name: "One",
        email_address: users(:one).email_address,
        password: "password123",
        password_confirmation: "password123"
      } }
    end

    assert_response :unprocessable_entity
  end
end
