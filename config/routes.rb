Rails.application.routes.draw do
  root "users#me"

  resources :users, only: [:show, :new, :create] do
    get :me, on: :collection
  end

  get :menu, to: "menu#index"

  get :login, to: "sessions#new"
  post :login, to: "sessions#create"
  delete :logout, to: "sessions#destroy"

  resources :password_reset_request, only: %i[new create]
  resources :password_reset, only: %i[new create]
end
