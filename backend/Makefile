.PHONY: deps fmt test build run cli

deps:
	go mod tidy

fmt:
	gofmt -w $(shell find . -name '*.go' -not -path './vendor/*')

test:
	go test ./...

build:
	go build ./...

run:
	go run ./cmd/server

cli:
	go run ./cmd/vpnctl --help
