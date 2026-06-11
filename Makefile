VERSION   ?= $(shell cat VERSION)
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

.PHONY: all build build-web release package test test-unit test-integration lint clean run

all: build-web build

## build: compile the Go binary (embeds web/dist)
build:
	go build -p 1 $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/kaos-control

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

## package: build release binaries (via `release`) and bundle each into a
##          versioned zip alongside README, LICENSE, CONTRIBUTING.md; also
##          writes ./dist/SHA256SUMS.
package: release
	./scripts/package-release.sh

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

## test-e2e: run Playwright e2e smoke tests (builds binary first)
test-e2e: build
	cd tests/e2e && pnpm install && pnpm test

## test-all: run all test suites
test-all: test-unit test-integration test-e2e
	cd tests/web && pnpm test

## lint: run go vet, staticcheck, govulncheck, gosec, and gitleaks
##       Go security tooling requires GOBIN on PATH (handled by the
##       `export PATH` near the top of this file).
##
##       gosec exclusions (justified in security-plan.md):
##         G104  unhandled errors — pervasive in defer/_ = patterns
##         G124  cookie flags — sessionCookie HAS all flags; csrf cookie
##               must be HttpOnly:false for the double-submit pattern;
##               Secure tracks the TLSOn flag, which is correct
##         G201/G202  SQL string formatting/concat — IN(?,?,?) builders
##         G204  subprocess with variable — agent runner, scheduler shell
##               jobs, devops pipelines, ollama/ideachat HTTP — every
##               site is intentional and reviewed in security-plan.md §2.1/§2.2
##         G301/G302/G306  file/dir perms 0644/0755 — standard for shared content
##         G304/G703  file inclusion/path traversal via variable —
##                    every flagged path goes through internal/sandbox/
##         G705  XSS via taint — only flagged site is NDJSON output
##               (Content-Type: application/x-ndjson), not HTML
lint:
	go vet ./...
	@if [ -x "$(GOBIN)/staticcheck" ]; then \
	  "$(GOBIN)/staticcheck" ./...; \
	elif command -v staticcheck >/dev/null 2>&1; then \
	  staticcheck ./...; \
	else \
	  echo "staticcheck not installed; install with: go install honnef.co/go/tools/cmd/staticcheck@latest"; \
	fi
	@if [ -x "$(GOBIN)/govulncheck" ]; then \
	  "$(GOBIN)/govulncheck" ./...; \
	elif command -v govulncheck >/dev/null 2>&1; then \
	  govulncheck ./...; \
	else \
	  echo "govulncheck not installed; install with: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
	fi
	@if [ -x "$(GOBIN)/gosec" ]; then \
	  "$(GOBIN)/gosec" -quiet \
	    -exclude=G104,G124,G201,G202,G204,G301,G302,G304,G306,G703,G705 \
	    -exclude-dir=tests/web/node_modules \
	    -exclude-dir=node_modules \
	    -exclude-dir=web/node_modules \
	    ./...; \
	elif command -v gosec >/dev/null 2>&1; then \
	  gosec -quiet -exclude=G104,G124,G201,G202,G204,G301,G302,G304,G306,G703,G705 ./...; \
	else \
	  echo "gosec not installed; install with: go install github.com/securego/gosec/v2/cmd/gosec@latest"; \
	fi
	@if command -v gitleaks >/dev/null 2>&1; then \
	  gitleaks detect --no-banner --no-color --redact; \
	else \
	  echo "gitleaks not installed; install with: brew install gitleaks"; \
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
