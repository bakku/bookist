module MinimalLayout
  extend ActiveSupport::Concern

  included do
    # Helper method that indicates whether or not to show the navbar.
    #
    # @return [Boolean] true if no navbar should be shown, otherwise false.
    def minimal_layout?
      @minimal_layout.present?
    end
    helper_method :minimal_layout?

    def set_minimal_layout
      @minimal_layout = true
    end
    private :set_minimal_layout
  end

  class_methods do
    # Use this in a controller to disable the navbar for specific controller actions.
    #
    # @example
    #   class SomeController < ApplicationController
    #     minimal_layout :index
    #
    #     def index
    #       # no navbar
    #     end
    #
    #     def new
    #       # navbar
    #     end
    #   end
    #
    # @param actions [Array<Symbol>] the actions for which no navbar should be shown
    def minimal_layout(*actions)
      before_action :set_minimal_layout, only: actions
    end
  end
end
