Rails.application.routes.draw do
  root "users#me"

  resources :users, only: [:show, :new, :create] do
    get :me, on: :collection
  end

  get :login, to: "sessions#new"
  post :login, to: "sessions#create"
  delete :logout, to: "sessions#destroy"
end
