module UserAuthenticateable
  extend ActiveSupport::Concern

  included do
    before_action :authenticate_user!

    def authenticate_user!
      if session[:user_id].blank?
        redirect_to login_path
      else
        current_user
      end
    end
    private :authenticate_user!

    # Memoized helper to retrieve the current user.
    #
    # @return [User] the user that is currently logged in.
    def current_user
      @current_user ||= User.find(session[:user_id])
    end
    helper_method :current_user
  end
end
