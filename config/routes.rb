# frozen_string_literal: true

Rails.application.routes.draw do
  root "users#me"

  # Reveal health status on /up that returns 200 if the app boots with no exceptions, otherwise 500.
  # Can be used by load balancers and uptime monitors to verify that the app is live.
  get "up" => "rails/health#show", as: :rails_health_check

  resources :users, only: %i[show new create] do
    get :me, on: :collection
  end

  get :menu, to: "menu#index"

  get :login, to: "sessions#new"
  post :login, to: "sessions#create"
  delete :logout, to: "sessions#destroy"

  resources :password_reset_request, only: %i[new create]
  resources :password_reset, only: %i[new create]
end
