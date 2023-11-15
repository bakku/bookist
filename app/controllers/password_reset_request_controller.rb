# frozen_string_literal: true

class PasswordResetRequestController < ApplicationController
  skip_before_action :authenticate_user!, only: %i[new create]
  minimal_layout :new, :create

  def new
    @password_reset_request = PasswordResetRequest.new
  end

  def create
    @password_reset_request = PasswordResetRequest.new(password_reset_request_params)

    if @password_reset_request.valid?
      PreparePasswordResetJob.perform_async(@password_reset_request.email, I18n.locale.to_s)
      redirect_to login_path, flash: { success: t(".reset_password_success") }
    else
      flash.now[:error] = t(".reset_password_failure")
      render :new, status: :unprocessable_entity
    end
  end

  private

  def password_reset_request_params
    params.require(:password_reset_request).permit(:email)
  end
end
