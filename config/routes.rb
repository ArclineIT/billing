Rails.application.routes.draw do
  root "dashboard#show"

  # Authentication
  get    "login",  to: "sessions#new",     as: :new_session
  post   "login",  to: "sessions#create",  as: :session
  delete "logout", to: "sessions#destroy", as: :logout

  # Registration
  get  "register", to: "registrations#new",    as: :new_registration
  post "register", to: "registrations#create", as: :registration

  # Password reset
  get   "reset",        to: "passwords#new",    as: :new_password
  post  "reset",        to: "passwords#create", as: :passwords
  get   "reset/:token", to: "passwords#edit",   as: :edit_password
  patch "reset/:token", to: "passwords#update", as: :password
  put   "reset/:token", to: "passwords#update"

  # Billing
  get  "dashboard", to: "dashboard#show",       as: :dashboard
  get  "checkout",  to: "checkout#show",        as: :checkout
  post "cancel",    to: "subscriptions#cancel", as: :cancel_subscription
  post "webhook",   to: "webhooks#create",      as: :webhook

  # Health check for load balancers and uptime monitors.
  get "up" => "rails/health#show", as: :rails_health_check
end
