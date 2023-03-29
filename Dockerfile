FROM ruby:3.1.2-alpine

WORKDIR /app

ARG BUILD_ENV=development
ENV RAILS_ENV ${BUILD_ENV}

ARG BUILD_MASTER_KEY
ENV RAILS_MASTER_KEY ${BUILD_MASTER_KEY}

RUN apk add --no-cache --update build-base \
			linux-headers \
			git \
			postgresql-dev \
			nodejs \
			npm \
            tzdata \
            shared-mime-info \
            gcompat


COPY Gemfile Gemfile.lock package.json yarn.lock ./

RUN echo "gem: --no-rdoc --no-ri" >> /root/.gemrc && \
    bundle install && \
    npm install -g yarn && \
    yarn

COPY . .

RUN bundle exec rails assets:precompile

CMD ["bundle", "exec", "rails", "server", "-p", "3000", "-b", "0.0.0.0"]