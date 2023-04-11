class UsersController < ApplicationController
  skip_before_action :authenticate_user!, only: [:new, :create]

  def new
  end

  def create
    @user = User.new(user_params)

    if @user.save
      redirect_to login_path, flash: { success: t(".account_creation_success") }
    else
      render :new, status: :unprocessable_entity
    end
  end

  def me
    
  end

  private

    def user_params
      params.permit(:username, :email, :password, :password_confirmation)
    end
end
