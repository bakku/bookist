class UsersController < ApplicationController
  skip_before_action :authenticate_user!, only: %i[new create]
  minimal_layout :new, :create

  def new
    @user = User.new
  end

  def create
    @user = User.new(user_params)

    if @user.save
      redirect_to login_path, flash: { success: t(".account_creation_success") }
    else
      flash.now[:error] = t(".creation_failed")
      render :new, status: :unprocessable_entity
    end
  end

  def me
  end

  private

    def user_params
      params.require(:user).permit(:username, :email, :password, :password_confirmation)
    end
end
