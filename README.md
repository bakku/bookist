# Bookist

Bookist is a small web application from a book lover for other book lovers. It contains all the features which
I myself want in a book management software and which I use myself. Use it at [bookist.bakku.dev](https://bookist.bakku.dev).

Current features:

No features yet

## Tech Stack

- Backend: Ruby on Rails 7 with Ruby 3.2
- Database: PostgreSQL
- Frontend: Hotwire, Stimulus, esbuild, and tailwindcss

## Setup

It's possible to run `bookist` with Docker locally, but I like to develop on bare metal. I just spin up a necessary
PostgreSQL database in Docker, so that's what the following instructions will describe.

1. Clone the repository and make sure you have all local dependencies which are specified in `.tool-versions`
2. Run `docker-compose up db` and wait until PostgreSQL has finished starting up
3. Run `bin/setup`, it will install all required Ruby and Node dependencies and will setup your PostgreSQL database

## Development

The `bin/dev` script starts up everything you need; afterwards the web app will run on
[localhost:3000](http://localhost:3000).

## Test

Run `bundle exec rspec` to run all the tests including e2e tests.