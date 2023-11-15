# frozen_string_literal: true

# rubocop:disable Rails/Output
puts "== Seeding the database with fixtures =="

system("FIXTURES_PATH=spec/fixtures bin/rails db:fixtures:load")
# rubocop:enable Rails/Output
