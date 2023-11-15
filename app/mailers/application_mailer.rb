# frozen_string_literal: true

class ApplicationMailer < ActionMailer::Base
  default from: "bookist@bakku.dev"
  layout "mailer"
end
