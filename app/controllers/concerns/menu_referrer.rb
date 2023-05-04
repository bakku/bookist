module MenuReferrer
  extend ActiveSupport::Concern

  included do
    # If the user was referred to the current page via the menu
    # then we want to show a small animation on the frontend for
    # the menu button.
    #
    # @return [Boolean] whether or not the user came from the menu to
    #   the current page
    def referred_to_by_menu?
      @_referred_to_by_menu ||= begin
        return false if request.referrer.blank?

        URI(request.referrer).path == menu_path
      end
    end
    helper_method :referred_to_by_menu?
  end
end
