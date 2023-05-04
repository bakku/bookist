module NavbarHelper
  # Returns a hash of attributes which are supposed to be used
  # in the HTML data attribute of the hamburger menu button in
  # the navbar when the menu is NOT opened.
  #
  # @param show_animation [Boolean]
  # @return Hash<Symbol, String | Boolean>
  def navbar_menu_data_attributes(show_animation: false)
    { testid: "navbar-menu-open", turbo_temporary: true }.tap do |attributes|
      if show_animation
        attributes[:controller] = "menu-button"
        attributes[:menu_button_animation_type_value] = "close"
      end
    end
  end
end
