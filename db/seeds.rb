puts "== Seeding the database with fixtures =="
system("FIXTURES_PATH=spec/fixtures bin/rails db:fixtures:load")