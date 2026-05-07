VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BINARY    := kaos-control
MODULE    := github.com/kaos-control/kaos-control
LDFLAGS   := -ldflags "-X main.version=$(VERSION) -X $(MODULE)/internal/http.Version=$(VERSION)"
BUILD_DIR := ./dist

PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64
RELEASE_LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X $(MODULE)/internal/http.Version=$(VERSION)"

# Resolve Go's bin dir so `go install`-ed tools (staticcheck, etc.) are
# discoverable regardless of the calling shell's PATH.
GOBIN := $(shell go env GOPATH)/bin
export PATH := $(GOBIN):$(PATH)

.PHONY: all build build-web release test test-unit test-integration lint clean run

all: build-web build

## build: compile the Go binary (embeds web/dist)
build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/kaos-control

## release: build static binaries for every platform in PLATFORMS into ./dist/
release:
	@mkdir -p $(BUILD_DIR)
	@for p in $(PLATFORMS); do \
	  os=$${p%/*}; arch=$${p#*/}; ext=$$([ "$$os" = "windows" ] && echo .exe); \
	  echo "→ $$os/$$arch"; \
	  CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch \
	    go build -trimpath $(RELEASE_LDFLAGS) \
	    -o $(BUILD_DIR)/$(BINARY)-$$os-$$arch$$ext ./cmd/kaos-control || exit 1; \
	done

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
	@if [ -x "$(GOBIN)/staticcheck" ]; then \
	  "$(GOBIN)/staticcheck" ./...; \
	elif command -v staticcheck >/dev/null 2>&1; then \
	  staticcheck ./...; \
	else \
	  echo "staticcheck not installed; install with: go install honnef.co/go/tools/cmd/staticcheck@latest"; \
	fi

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
