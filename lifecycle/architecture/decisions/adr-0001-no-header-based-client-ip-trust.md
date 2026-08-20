---
title: No header-based client IP trust (chi RealIP removed)
type: adr
status: draft
lineage: adr-no-header-based-client-ip-trust
created: "2026-08-20T14:52:00+10:00"
labels:
    - adr
    - security
related_to:
    - defects/go-toolchain-chi-govulncheck-advisories
---

# ADR-0001: No header-based client IP trust (chi RealIP removed)

## Context

`internal/http/server.go` registered `github.com/go-chi/chi/v5/middleware.RealIP`
on the root router. That middleware unconditionally overwrites
`http.Request.RemoteAddr` with the first value found in the
`True-Client-IP`, `X-Real-IP`, or `X-Forwarded-For` headers — headers any
client can set on a direct request.

`internal/http/auth.go:82` uses `r.RemoteAddr` to gate a trusted,
password-free authentication path: a request carrying an
`X-Kaos-Local-User: <email>` header is authenticated as that user
*without a password*, provided `isLoopback(r.RemoteAddr)` is true
(`internal/http/auth.go:335`). This exists so a local CLI/companion
process talking to `localhost` can authenticate without prompting.

kaos-control is distributed as a single Go binary that serves HTTP/TLS
directly (`internal/http/server.go` `ListenAndServe`/`ServeTLS`) — there
is no reverse proxy, load balancer, or other trusted hop in front of it
in any documented deployment topology. With `RealIP` installed, any
network-reachable client could send `X-Real-IP: 127.0.0.1` (or
`True-Client-IP`/`X-Forwarded-For`) together with
`X-Kaos-Local-User: <victim email>` and have `r.RemoteAddr` spoofed to a
loopback address, defeating `isLoopback` and impersonating any known
user with no credentials.

This was surfaced while fixing
[[go-toolchain-chi-govulncheck-advisories]] (GO-2026-5777, GO-2026-5775,
GO-2026-5774 — chi's `RealIP` IP-spoofing advisories). Bumping chi to
v5.3.1 silences `govulncheck` because upstream deprecated `RealIP` in
favour of new opt-in helpers, but `RealIP`'s own behaviour is
byte-for-byte unchanged — the advisory is closed by the *availability*
of a safer alternative, not by `RealIP` itself becoming safe. Simply
bumping the dependency would have left the auth-bypass exploitable while
reporting a clean lint.

## Decision

Remove `middleware.RealIP` from the router entirely
(`internal/http/server.go` `buildRouter`). `r.RemoteAddr` is left as Go's
`net/http` sets it from the raw TCP connection, which cannot be
influenced by request headers. `isLoopback` and any other
security-relevant client-IP check must keep reading `r.RemoteAddr`
un-mutated.

We deliberately do not adopt chi's `ClientIPFromXFFTrustedProxies`
(the suggested replacement for proxied deployments) because kaos-control
has no reverse-proxy hop to configure as trusted — there is nothing to
scope trust to.

## Consequences

- Closes the spoofing path: an external client can no longer forge
  `r.RemoteAddr` to pass `isLoopback`.
- If a future deployment topology puts kaos-control behind a reverse
  proxy or load balancer, `r.RemoteAddr` will observe the proxy's
  address rather than the real client's, breaking `isLoopback` for
  legitimate local callers and any future IP-based logging/rate
  limiting. That change must configure `ClientIPFromXFFTrustedProxies`
  scoped to the specific trusted proxy hop(s) — do not re-introduce
  `middleware.RealIP` or otherwise trust `X-Forwarded-For`/`X-Real-IP`
  from an unscoped set of peers. Revisit this ADR at that time.
- No code today derives a client IP for logging, rate limiting, or audit
  purposes, so this change has no other observable behaviour change.
