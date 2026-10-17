# Variables
BINARY_NAME := wtr
# The same program is also built under its explicit name, meant for shell rc
# files (the invoked name is the one shown in messages and in the picker).
BINARY_ALIAS := what-to-run
GO := go
CMD := ./cmd/wtr
KEY ?= ctrl-s
# Where `make install` puts the binaries (override: PREFIX=/usr/local/bin).
PREFIX ?= $(HOME)/.local/bin

.PHONY: all build run test fmt lint tidy clean bind install help

all: fmt test build

## build: Build bin/wtr and bin/what-to-run
build:
	@echo "Building $(BINARY_NAME) and $(BINARY_ALIAS)..."
	$(GO) build -o bin/$(BINARY_NAME) $(CMD)
	$(GO) build -o bin/$(BINARY_ALIAS) $(CMD)

## bind: Build and print the zsh integration for eval (eval "$(make bind)", KEY=combo)
bind:
	@$(MAKE) --no-print-directory build 1>&2
	@bin/$(BINARY_NAME) init zsh --key $(KEY)

## install: Install both names into PREFIX (default ~/.local/bin)
install: build
	mkdir -p $(PREFIX)
	install -m755 bin/$(BINARY_NAME) $(PREFIX)/$(BINARY_NAME)
	install -m755 bin/$(BINARY_ALIAS) $(PREFIX)/$(BINARY_ALIAS)
	@echo "Installed $(BINARY_NAME) and $(BINARY_ALIAS) into $(PREFIX)"
	@case ":$$PATH:" in *:$(PREFIX):*) ;; \
	  *) echo "Note: $(PREFIX) is not in your PATH" ;; esac

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
