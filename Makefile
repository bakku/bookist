.DEFAULT_GOAL := build

BINARY := ./bin/bookist

.PHONY: build run test clean

build:
	mkdir -p ./bin
	go build -o $(BINARY) ./cmd/bookist

run:
	go run ./cmd/bookist serve

test:
	go test ./...

clean:
	rm -rf ./bin
