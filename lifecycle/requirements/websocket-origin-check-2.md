---
title: WebSocket Origin Validation
type: requirement
status: in-development
lineage: websocket-origin-check
created: "2026-05-11T08:20:00+10:00"
priority: high
parent: lifecycle/ideas/websocket-origin-check.md
labels:
    - security
    - backend
    - release-blocker
release: KC-Release0
---

# WebSocket Origin Validation

Parent: [[websocket-origin-check]].

## Goal

The WebSocket upgrade at `GET /api/p/:project/ws` accepts connections only from origins that match the server's expected host. Cross-origin connection attempts are rejected at the handshake layer (HTTP 403, no upgrade).

## Functional requirements

1. **Replace `InsecureSkipVerify: true`** in `internal/http/ws.go` with an `OriginPatterns: []string{...}` allowlist passed to `websocket.Accept`.

2. **The allowlist is derived at server start from server config:**
   - The `Listen` address (e.g. `0.0.0.0:8042`, `127.0.0.1:8042`) — extract host portion, default to `localhost` if the listen address is `0.0.0.0` or `:`.
   - Allow `localhost` and `127.0.0.1` regardless, to support local development and the embedded SPA loading over `http://localhost:8042/`.
   - Allow the public hostname if `TLSOn` is true — typically inferred from the certificate's SAN list, but for KC-Release0 a `PublicHost string` field on `ServerConfig` (optional, comma-separated host list) is sufficient.

3. **Same-host requests over both ports MAY differ**. The `OriginPatterns` field matches the *hostname only* (the library strips port). Documented behaviour; acceptable for our use case.

4. **Failure mode.** When a rejected origin attempts to connect, `coder/websocket` returns HTTP 403 with the body `request Origin "<origin>" is not allowed` and the connection never upgrades. No need for custom error handling.

5. **Browser same-origin still works without configuration.** A user navigating to `http://localhost:8042/` and opening the kanban board MUST continue to receive WS events — no regression.

## Non-functional requirements

- No additional dependencies. The `coder/websocket` library already supports `OriginPatterns`.
- No frontend changes. The SPA's WS URL is already `ws(s)://<location.host>/api/p/...`, which the browser populates the `Origin` header from automatically.
- The change must not break the auto-login integration test setup (the test server uses `127.0.0.1:<random-port>`).

## Acceptance criteria

- **AC-1 — Same origin allowed.** A WebSocket connection with `Origin: http://localhost:8042` to `ws://localhost:8042/api/p/.../ws` upgrades successfully.
- **AC-2 — Cross-origin rejected.** A WebSocket connection with `Origin: http://evil.example.com` is rejected with HTTP 403 before the upgrade. No `101 Switching Protocols` response is sent.
- **AC-3 — Missing Origin header.** Server-to-server WS clients (no browser, no Origin header) are allowed — many CLI clients and proxies omit the header. This matches `coder/websocket` default `OriginPatterns` semantics: missing Origin = no enforcement.
- **AC-4 — Configured public hostname allowed.** If `PublicHost` is set in `ServerConfig`, requests with that origin host are accepted.
- **AC-5 — Auth still required.** A successfully upgraded WebSocket from an allowed origin still receives `close 4401` if the session is invalid (existing behaviour from earlier WS-auth work).
- **AC-6 — Integration test.** A new test in `tests/integration/websocket_origin_test.go` covers AC-1, AC-2, AC-3, AC-4, AC-5.

## Out of scope

- Per-project origin allowlists (a future workspace-sharing feature).
- Strict origin checking for the HTTP API as well (HTTP API mutations are already covered by the CSRF double-submit pattern).
- Auto-extraction of allowed hosts from the TLS certificate's SAN list. Manual `PublicHost` config is sufficient for KC-Release0.

## No questions

None — the implementation is well-defined by the library's `OriginPatterns` semantics.
