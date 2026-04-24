VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BINARY    := kaos-control
MODULE    := github.com/kaos-control/kaos-control
LDFLAGS   := -ldflags "-X main.version=$(VERSION) -X $(MODULE)/internal/http.Version=$(VERSION)"
BUILD_DIR := ./dist

.PHONY: all build build-web test test-unit test-integration lint clean run

all: build-web build

## build: compile the Go binary (embeds web/dist)
build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/kaos-control

## build-web: build the Vite frontend into web/dist
build-web:
	cd web && pnpm install && pnpm run build

## run: run the server in development mode (no TLS, default config)
run:
	LOG_LEVEL=debug go run $(LDFLAGS) ./cmd/kaos-control

## test-unit: run unit tests only
test-unit:
	go test ./... -count=1 -short

## test-integration: run integration tests (requires test fixtures)
test-integration:
	go test ./... -count=1 -tags=integration

## test: run all backend tests
test: test-unit test-integration

## lint: run go vet and staticcheck
lint:
	go vet ./...
	@which staticcheck > /dev/null 2>&1 && staticcheck ./... || echo "staticcheck not installed; skipping"

## clean: remove build artefacts
clean:
	rm -rf $(BUILD_DIR)
	rm -rf web/dist/assets web/dist/*.js web/dist/*.css 2>/dev/null; true

## tidy: tidy and verify module dependencies
tidy:
	go mod tidy
	go mod verify

## help: list available targets
help:
	@grep -E '^## ' Makefile | sed 's/## //'
