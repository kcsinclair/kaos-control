---
title: 'make lint fails govulncheck: Go 1.25.12 stdlib and chi v5.2.5 have known CVEs'
type: defect
status: in-development
lineage: go-toolchain-chi-govulncheck-advisories
created: "2026-08-20T12:20:00+10:00"
labels:
    - defect
release: KC-Release5
assignees:
    - role: backend-developer
      who: agent
---

# make lint fails govulncheck: Go 1.25.12 stdlib and chi v5.2.5 have known CVEs

## Reproduction Steps

1. `make lint`
2. Observe `go vet` and `staticcheck` pass with no output.
3. `govulncheck ./...` reports 8 vulnerabilities and exits non-zero, failing the `lint` target.

## Expected Behaviour

`make lint` exits 0 with no unresolved vulnerability advisories against code paths actually reachable from this project's call graph.

## Actual Behaviour

`govulncheck ./...` finds 8 reachable vulnerabilities:

- Go stdlib (fixed in go1.25.13, currently on go1.25.12):
  - GO-2026-6218 — quadratic complexity in `net/url` `resolvePath` (reachable via `internal/agent/gemini.go:202` → `http.Client.Do` → `url.URL.Parse`)
  - GO-2026-6090 — `crypto/tls` unbounded post-handshake messages (reachable via `internal/http/server.go:430`, `internal/devops/logger.go:449`, `internal/agent/gemini.go:166,202`)
  - GO-2026-6089 — missing `ReadHeaderTimeout` on unencrypted HTTP/2 check in `net/http` (reachable via `internal/http/server.go:428,430`)
  - GO-2026-5972 — missing max recursion depth in `encoding/asn1` (reachable via `internal/http/server.go:428` → `ServeTLS` → `asn1.Unmarshal`)
  - GO-2026-5026 — ASCII-only punycode label rejection missing in `golang.org/x/net/idna` (reachable via `internal/agent/gemini.go:202` → `http.Client.Do`)
- `github.com/go-chi/chi/v5@v5.2.5` (fixed in v5.3.0), all reachable via `internal/http/server.go:147` → `chi.Mux.Post` → `middleware.RealIP`:
  - GO-2026-5777, GO-2026-5775, GO-2026-5774 — `middleware.RealIP` IP-spoofing via unvalidated `X-Forwarded-For`/`X-Real-IP` headers

## Logs / Output

```
make[1]: *** [lint] Error 3
```

Full `govulncheck ./...` output captured in this run's lint log (8 vulnerabilities from 1 module + Go standard library; 3 additional advisories in required-but-not-called modules, not blocking).

## Fix guidance

- Bump the Go toolchain to `go1.25.13` (`go.mod`'s `go`/`toolchain` directive) to clear the 5 stdlib CVEs.
- Bump `github.com/go-chi/chi/v5` to `>=v5.3.0` to clear the 3 `RealIP` CVEs — verify whether the project's `RealIP` usage trusts `X-Forwarded-For` from an untrusted edge (if kaos-control sits behind no reverse proxy, note the mitigation instead of only bumping).
- Re-run `make lint` and confirm a clean `govulncheck` exit.
