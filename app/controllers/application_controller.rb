# frozen_string_literal: true

class ApplicationController < ActionController::Base
  include HttpAcceptLanguage::AutoLocale
  include UserAuthentication
  include MinimalLayout
  include MenuReferrer
end
