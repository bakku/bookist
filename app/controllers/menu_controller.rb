class MenuController < ApplicationController
  minimal_layout :index

  def index
    @title = params[:title] || t("users.me.dashboard")
    @return_to = params[:return_to] || me_users_path
  end
end
