class SessionsController < ApplicationController
  skip_before_action :authenticate_user!

  def new
  end

  def create
    user = User.find_by(username: login_params[:username])

    if user && user.authenticate(login_params[:password])
      session[:user_id] = user.id

      redirect_to me_users_path
    else
      render :new, status: :unauthorized
    end
  end

  private

    def login_params
      params.permit(:username, :password)
    end
end
