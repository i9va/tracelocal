BINARY := tracelocal
CMD    := ./cmd/tracelocal

.PHONY: build run test lint tidy

build:
	go build -o $(BINARY) $(CMD)

run:
	go run $(CMD)

test:
	go test ./...

lint:
	golangci-lint run ./...

tidy:
	go mod tidy
