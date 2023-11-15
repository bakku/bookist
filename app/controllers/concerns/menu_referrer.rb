# frozen_string_literal: true

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
      @_referred_to_by_menu ||=
        if request.referer.blank?
          false
        else
          URI(request.referer).path == menu_path
        end
    end
    helper_method :referred_to_by_menu?
  end
end
