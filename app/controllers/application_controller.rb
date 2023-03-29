class ApplicationController < ActionController::Base
  include HttpAcceptLanguage::AutoLocale

  before_action :authenticate_user!

  def authenticate_user!
    if session[:user_id].blank?
      redirect_to login_path
    else
      current_user
    end
  end

  def current_user
    @current_user ||= User.find(session[:user_id])
  end
  helper_method :current_user
end
