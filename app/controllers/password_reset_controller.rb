# frozen_string_literal: true

class PasswordResetController < ApplicationController
  skip_before_action :authenticate_user!, only: %i[new create]
  minimal_layout :new, :create

  def new
    @user = find_user(params[:token])
    authority = PasswordResetAuthority.new

    if @user.nil? || !authority.resettable?(@user)
      redirect_to login_path, flash: { error: t(".invalid_token") }
    end
  end

  def create
    user = find_user(password_reset_params[:password_reset_token])

    if user.present? && complete_reset!(user)
      redirect_to login_path, flash: { success: t(".reset_successful") }
    else
      redirect_to login_path, flash: { error: t(".reset_unsuccessful") }
    end
  end

  private

  def find_user(token)
    return nil if token.blank?

    User.find_by(password_reset_token: token)
  end

  def complete_reset!(user)
    authority = PasswordResetAuthority.new
    authority.complete_reset!(user, password_reset_params[:password], password_reset_params[:password_confirmation])
  end

  def password_reset_params
    params.require(:user).permit(:password_reset_token, :password, :password_confirmation)
  end
end
