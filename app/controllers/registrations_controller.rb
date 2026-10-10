class RegistrationsController < ApplicationController
  allow_unauthenticated_access only: %i[ new create ]

  def new
    @user = User.new
  end

  def create
    @user = User.new(registration_params)

    if @user.save
      Billing::CreateCustomer.call(@user)
      start_new_session_for @user
      redirect_to dashboard_path, notice: "Welcome to Arcline IT."
    else
      render :new, status: :unprocessable_entity
    end
  end

  private

  def registration_params
    params.require(:user).permit(
      :first_name, :last_name, :email_address, :password, :password_confirmation
    )
  end
end
