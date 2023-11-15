# frozen_string_literal: true

source "https://rubygems.org"
git_source(:github) { |repo| "https://github.com/#{repo}.git" }

ruby "3.2.2"

gem "rails", "7.1.2"

# Backend
gem "bcrypt", "~> 3.1.7"
gem "bootsnap", require: false
gem "http_accept_language"
gem "pg"
gem "puma", "~> 5.0"
gem "sidekiq"
gem "tzinfo-data", platforms: %i[windows jruby]

# Frontend
gem "cssbundling-rails"
gem "jsbundling-rails"
gem "sprockets-rails"
gem "stimulus-rails"
gem "turbo-rails"

group :development, :test do
  gem "debug", platforms: %i[mri windows]
  gem "rspec-rails", "~> 6.0.0"
end

group :development do
  gem "rubocop", require: false
  gem "rubocop-capybara", require: false
  gem "rubocop-rails", require: false
  gem "rubocop-rspec", require: false

  gem "web-console"
end

group :test do
  gem "capybara"
  gem "selenium-webdriver"
end
