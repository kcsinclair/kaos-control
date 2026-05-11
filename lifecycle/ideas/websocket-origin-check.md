---
title: Validate WebSocket Origin Header
type: idea
status: done
lineage: websocket-origin-check
created: "2026-05-11T08:15:00+10:00"
priority: high
labels:
    - security
    - backend
    - release-blocker
release: KC-Release0
---

# Validate WebSocket Origin Header

## Problem

`internal/http/ws.go:22-25` accepts WebSocket connections from any origin:

```go
conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
    // Accept all origins for now; M4 auth will tighten this.
    InsecureSkipVerify: true,
})
```

`InsecureSkipVerify: true` disables the `coder/websocket` library's built-in Origin-header check. The Same-Origin Policy does NOT apply to WebSocket upgrades — a browser will happily allow a script on `evil.example.com` to open a WebSocket to `kaos-control.local` and the browser will attach the kaos-control session cookie automatically.

## Attack scenario

1. A user is logged into kaos-control in tab A.
2. They click a link, opening `evil.example.com` in tab B (or load a compromised dependency, ad iframe, etc).
3. The malicious page runs `new WebSocket("wss://kaos-control.local/api/p/kaos-control/ws")` — the browser attaches `kc_session`.
4. The new global auth middleware now passes the request through (cookie is valid). The WS handler accepts the upgrade.
5. Every `artifact.indexed`, `file.changed`, `agent.started`, `lock.acquired`, etc. event is streamed to the attacker, including artifact paths, agent target paths, file paths, and lock holders. This is project-content leakage.

This is **Cross-Site WebSocket Hijacking (CSWSH)**. It is exploitable today, with no fix required on the malicious side — only social engineering to get the user to open a page.

## Impact

- **Confidentiality**: full event stream leak (file paths, status changes, lineage).
- **Single-user deployments are NOT immune.** Any user who clicks a link while logged in is vulnerable. This is not gated by "do you have multiple users?"
- **Defence in depth gap**: the SPA's CSRF cookie protects mutations, but WebSocket data is read-only and not protected by CSRF.

## Desired outcome

The WebSocket handshake rejects any request whose `Origin` header does not match the server's expected host. Same-origin browser tabs continue to work transparently.

## Related

- `coder/websocket` library exposes `OriginPatterns []string` on `AcceptOptions` precisely for this use case.
- The earlier `M4 auth will tighten this` comment in the code indicates the author already flagged this for closing.
