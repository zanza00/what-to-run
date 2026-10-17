# Variables
BINARY_NAME := what-to-run
GO := go
CMD := ./cmd/what-to-run

.PHONY: all build run test fmt lint tidy clean bind help

all: fmt test build

## build: Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	$(GO) build -o bin/$(BINARY_NAME) $(CMD)

## bind: Build and print the source command for the calling shell (eval "$(make bind)")
bind:
	@$(MAKE) --no-print-directory build 1>&2
	@echo "source $(CURDIR)/scripts/ctrl-s.zsh"

## run: Run the application without building a binary
run:
	$(GO) run $(CMD)

## test: Run tests with the race detector and coverage
test:
	$(GO) test -race -coverprofile=coverage.out ./...

## fmt: Format and vet the code
fmt:
	$(GO) fmt ./...
	$(GO) vet ./...

## lint: Run golangci-lint (requires golangci-lint installed)
lint:
	golangci-lint run ./...

## tidy: Tidy go.mod and go.sum
tidy:
	$(GO) mod tidy

## clean: Remove build artifacts
clean:
	rm -rf bin/ coverage.out coverage.html

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
