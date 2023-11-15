# frozen_string_literal: true

# Rails 7.1 introduced the following issue: https://github.com/hotwired/turbo-rails/issues/512
# We need to keep this workaround until it is fixed.
Rails.autoloaders.once.do_not_eager_load("#{Turbo::Engine.root}/app/channels")
