---
title: "Backend Plan — WebSocket Origin Validation"
type: plan-backend
status: done
lineage: websocket-origin-check
parent: lifecycle/requirements/websocket-origin-check-2.md
created: "2026-05-11T08:25:00+10:00"
priority: high
labels:
    - security
    - backend
    - release-blocker
release: KC-Release0
---

# Backend Plan — WebSocket Origin Validation

Closes [[websocket-origin-check-2]]. Small, surgical change in two files.

## Milestone 1 — Add `PublicHost` to `ServerConfig`

### Description

Extend `kaoshttp.ServerConfig` (in `internal/http/server.go`) with an optional `PublicHost` field. The field accepts a comma-separated list of hostnames (e.g. `"kaos.internal,kaos-control.example.com"`). Empty string means "no extra hosts beyond `localhost`/`127.0.0.1`".

### Files to change

- **Edit** `internal/http/server.go`:
  - Add to `ServerConfig`:
    ```go
    // PublicHost is a comma-separated list of additional hostnames the
    // server is reachable at; used to populate WebSocket OriginPatterns.
    // Local listen addresses (localhost, 127.0.0.1) are always allowed.
    PublicHost string
    ```
  - Add a helper method on `Server` (or a package-level function):
    ```go
    func (s *Server) allowedWSOrigins() []string {
        out := []string{"localhost", "127.0.0.1"}
        if h, _, err := net.SplitHostPort(s.cfg.Listen); err == nil && h != "" && h != "0.0.0.0" && h != "::" {
            out = append(out, h)
        }
        for _, h := range strings.Split(s.cfg.PublicHost, ",") {
            if h = strings.TrimSpace(h); h != "" {
                out = append(out, h)
            }
        }
        return out
    }
    ```

- **Edit** `cmd/kaos-control/main.go`:
  - Read `PublicHost` from `~/.kaos-control/config.yaml` (if a field exists in the app config) or from a `KAOS_PUBLIC_HOST` environment variable. Pass through to `ServerConfig.PublicHost`. The exact wiring depends on the existing app-config loader — match its pattern.

### Acceptance criteria

- `go build ./...` clean.
- The default (no `PublicHost` configured) yields at minimum `["localhost", "127.0.0.1"]` plus the listen host if non-wildcard.
- Unit test in `internal/http/ws_origin_test.go` covering: empty `PublicHost`, wildcard listen, single public host, comma-separated public hosts.

---

## Milestone 2 — Apply `OriginPatterns` in the WebSocket handler

### Description

Replace `InsecureSkipVerify: true` in `internal/http/ws.go` with the computed pattern list.

### Files to change

- **Edit** `internal/http/ws.go` (lines 22-25):
  ```go
  conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
      OriginPatterns: s.allowedWSOrigins(),
  })
  ```
  Remove the `// Accept all origins for now; M4 auth will tighten this.` comment — this *is* the M4 tightening.

### Acceptance criteria

- `go build ./...` clean.
- `coder/websocket` v1.8.14 (current pin) supports `OriginPatterns`; no version bump required.

---

## Milestone 3 — Integration test

### Description

Add `tests/integration/websocket_origin_test.go` covering the requirement's AC-1 through AC-5.

### Files to change

- **New** `tests/integration/websocket_origin_test.go`:
  ```go
  //go:build integration

  package integration

  // TestWebSocketOrigin_SameOriginAllowed asserts a 101 upgrade with a
  // matching Origin header (AC-1).
  func TestWebSocketOrigin_SameOriginAllowed(t *testing.T) { ... }

  // TestWebSocketOrigin_CrossOriginRejected asserts 403 (no upgrade) when
  // the Origin header points to an unrelated host (AC-2).
  func TestWebSocketOrigin_CrossOriginRejected(t *testing.T) { ... }

  // TestWebSocketOrigin_MissingOriginAllowed asserts that a request with
  // no Origin header (e.g. a CLI client) still upgrades (AC-3).
  func TestWebSocketOrigin_MissingOriginAllowed(t *testing.T) { ... }

  // TestWebSocketOrigin_PublicHostAllowed seeds ServerConfig.PublicHost
  // with "example.test" and asserts the upgrade succeeds for that origin
  // (AC-4). Requires a small testEnv extension to inject PublicHost.
  func TestWebSocketOrigin_PublicHostAllowed(t *testing.T) { ... }

  // TestWebSocketOrigin_AuthStillEnforced asserts that an allowed-origin
  // connection without a valid session still receives close code 4401
  // (AC-5).
  func TestWebSocketOrigin_AuthStillEnforced(t *testing.T) { ... }
  ```

### Test approach

Use the existing `helpers_test.go` patterns. For each test:

1. Spin up `newTestEnv(t, nil)` (auto-logs in).
2. Construct a manual `http.NewRequest("GET", env.baseURL+"/api/p/testproject/ws", nil)`.
3. Set the headers required for a WebSocket handshake: `Upgrade: websocket`, `Connection: Upgrade`, `Sec-WebSocket-Key: <base64-16-bytes>`, `Sec-WebSocket-Version: 13`, and explicitly set `Origin: ...` to the value under test.
4. Attach the session cookies from `env.cookies`.
5. Send via `http.DefaultClient.Do(req)`.
6. Assert the response status code (101 for allowed, 403 for rejected).
7. For AC-5: read the WebSocket close frame and assert close code 4401.

Helper to extend `helpers_test.go`: a `newTestEnvWithServerCfg(t, seeds, cfgFn func(*kaoshttp.ServerConfig))` variant so the public-host test can set `PublicHost: "example.test"` on the server.

### Acceptance criteria

- All five test cases pass.
- `make lint` clean.
- Existing WebSocket-dependent tests (`tests/integration/agent_ws_test.go` etc.) still pass — same-origin from `127.0.0.1:<port>` is in the default allowlist.

---

## Verification (end-to-end)

1. `make build && make run` against the kaos-control project.
2. Open `http://localhost:8042/p/kaos-control/artifacts/board` — events stream as before.
3. From DevTools console on a *different* origin (e.g. `https://example.com`), run:
   ```js
   new WebSocket("ws://localhost:8042/api/p/kaos-control/ws")
   ```
   Expect the connection to close immediately. Network tab shows a `403 Forbidden`.

## Risk notes

- **Browser dev-tools tab opened directly on a `file://` URL** — the Origin header is `null`. `coder/websocket` treats `null` as "no origin" (allowed). Acceptable: a `file://` page is fundamentally cross-origin to a `http://` server, but the user-visible behaviour is benign in practice and matches library defaults. Not worth special-casing.
- **Proxies that strip the Origin header** — allowed by AC-3. CLI clients (curl, websocat, our integration tests' raw socket code) intentionally fall into this bucket.
- **Reverse proxy in front of kaos-control** — the proxy must forward the original `Origin` header. If a proxy rewrites the Origin to its own host, document the requirement to add the proxy's public host to `PublicHost`. This is the same operational expectation any origin-checking server has.
