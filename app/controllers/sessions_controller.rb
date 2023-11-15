# frozen_string_literal: true

class SessionsController < ApplicationController
  skip_before_action :authenticate_user!, only: %i[new create]
  minimal_layout :new, :create

  def new
  end

  def create
    user = User.find_by(username: login_params[:username])

    if user.present? && user.authenticate(login_params[:password])
      session[:user_id] = user.id

      redirect_to me_users_path
    else
      flash.now[:error] = t(".authentication_failed")
      render :new, status: :unauthorized
    end
  end

  def destroy
    session.clear
    redirect_to login_path, flash: { success: t(".logout_success") }
  end

  private

  def login_params
    params.permit(:username, :password)
  end
end
