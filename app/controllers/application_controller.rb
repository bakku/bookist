class ApplicationController < ActionController::Base
  include HttpAcceptLanguage::AutoLocale
  include UserAuthenticateable
  include MinimalLayoutable
end
