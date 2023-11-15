# frozen_string_literal: true

module ApplicationHelper
  def form_field(**kwargs)
    render partial: "shared/forms/form_field",
           locals: {
             hint: nil,
             margin: false
           }.merge(kwargs)
  end
end
