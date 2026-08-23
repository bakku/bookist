.DEFAULT_GOAL := build

CSS_INPUT := internal/web/assets/app.css
CSS_OUTPUT := internal/web/static/app.css
BINARY := ./bin/bookist

.PHONY: css css-watch build run test clean

css:
	mise exec -- tailwindcss -i $(CSS_INPUT) -o $(CSS_OUTPUT) --minify

css-watch:
	mise exec -- tailwindcss -i $(CSS_INPUT) -o $(CSS_OUTPUT) --watch

build: css
	mkdir -p ./bin
	go build -o $(BINARY) ./cmd/bookist

run: css
	go run ./cmd/bookist serve

test: css
	go test ./...

clean:
	rm -rf ./bin
