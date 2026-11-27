# CodeQL `go/path-injection` — analysis

**Date:** 2026-08-31
**Status:** working notes, not committed to git
**Question asked:** *Can an unauthenticated user inject data?*

---

## Short answer

**No.** An unauthenticated user cannot reach any of the flagged endpoints. An
earlier framing of this as "the authorization question, now with evidence"
overstated the concern; the corrected reading is below.

---

## Why the alerts cannot be reached anonymously

Every CodeQL-flagged route sits inside this group in
[internal/http/server.go:152](internal/http/server.go#L152):

```go
r.Use(s.sessionMiddleware)
r.Use(s.csrfMiddleware)
r.Use(s.requireAuth)
```

An unauthenticated request is rejected with `401` before any handler runs.
There are exactly three ways to authenticate, and none is open:

| Path | Requirement |
|---|---|
| Session cookie | valid login |
| `Authorization: Bearer <token>` | valid token |
| `X-Kaos-Local-User: <email>` | TCP peer must be **loopback** *and* the email must match an existing user |

The third is deliberately hardened. `middleware.RealIP` is explicitly **not**
installed, so `RemoteAddr` cannot be spoofed through `X-Forwarded-For`,
`X-Real-IP` or `True-Client-IP`. The reasoning is recorded in
[adr-0001-no-header-based-client-ip-trust](lifecycle/architecture/decisions/adr-0001-no-header-based-client-ip-trust.md),
and the comment at `server.go:141` states the dependency directly:

> No middleware.RealIP: kaos-control has no reverse proxy in front of it, so
> r.RemoteAddr (the real TCP peer) must not be overwritten from
> client-supplied X-Forwarded-For/X-Real-IP/True-Client-IP headers — the
> loopback-only local-identity bypass in auth.go depends on it being
> trustworthy.

**The one condition that would open it:** putting a same-host reverse proxy in
front of kaos-control without stripping `X-Kaos-Local-User`. Every request would
then arrive from `127.0.0.1`, and any remote caller able to set that header
would be authenticated as any named user. The ADR names this as the standing
assumption — worth re-checking if the deployment ever changes.

---

## Why 116 alerts look alarming anyway

CodeQL's `remote` threat model treats **any HTTP request body as an untrusted
remote source. It does not model authentication or authorization at all.** A
request carrying a logged-in admin's session cookie and an anonymous internet
request are indistinguishable to the analysis.

That is why "This path depends on a user-provided value" appears 116 times for
what is, in practice, *a value provided by an authenticated operator*.

---

## Verified measurements

CodeQL 2.26.4, local database of this repo. The local baseline reproduces GitHub
code scanning exactly.

| | alerts |
|---|---|
| baseline (no model pack) | **116** |
| with barrier models | **79** |
| removed | 37 |

Cleared where `sandbox.Resolve` is genuinely applied:

| File | Before | After |
|---|---|---|
| `internal/http/write.go` | 23 | 2 |
| `internal/release/disksync.go` | 5 | 0 |
| `internal/docs/docs.go` | 4 | 0 |

Reproduction commands are in [.github/codeql/README.md](.github/codeql/README.md).

### The remaining 79, by taint source

| Source | Alerts | Assessment |
|---|---|---|
| `handleCreateProject` (`internal/http/projects.go:566`) | 48 | Working as intended — see below |
| `internal/http/agents.go:171` (`target_path`) | 9 | **Unreviewed** — the one worth tracing |
| `handleCheckDirectory` (`internal/http/projects.go:346`) | 5 | Same class as `handleCreateProject` |
| `internal/http/architecture.go:38` | 4 | Sanitised deeper in the call chain than the model reaches |
| others | 13 | Mostly second-order from the project root |

---

## The 48 project-registration alerts are not a finding

Pointing kaos-control at a directory on disk **is the feature**. An authenticated
operator supplies an absolute path; the server then reads and writes beneath it.
`config.ValidatePathFormat` requires only that the path be absolute and outside
`~/.kaos-control`, and `handleCreateProject` has no role check beyond
authentication.

That is a **privilege** question among already-trusted users, not an attack
surface — and a mild one. The application does apply role checks where it
considers them warranted:

- `insufficient role to enqueue this agent` (`queue.go:71`)
- `only the enqueuer or product-owner may cancel` (`queue.go:155`)
- `product-owner role required` (`auth.go:301`)

So the absence of one on project registration may well be deliberate. If every
authenticated user is a trusted operator — which for a development tool they are
— there is nothing to fix here.

**If** a role is ever wanted that can use projects but not register them at
arbitrary filesystem paths, `handleCreateProject` is where the check would go.

---

## Revised plan

1. **Trace `agents.go:171`.** Still worth doing — the only remaining case where a
   user-supplied value reaches a file operation with no visible sanitiser *and*
   no clear design rationale. `target_path` is passed to `Manager.StartRun`,
   where it lands in a prompt template; it reaches a file operation somewhere
   downstream. Not yet followed to its sink, so no conclusion either way.
2. ~~Decide on project registration~~ — **dropped.** Working as intended.
3. **Push, disable default setup, merge.** Expected steady state is **79, not 0**.
   The README should say plainly that most of the remainder are
   authenticated-operator flows, rather than leaving it reading as unresolved
   risk.

---

## Correction log

- Initially described all 116 alerts as false positives. They are not: the
  barrier models clear 37, and the rest are real dataflow.
- Initially framed the 48 project-registration alerts as "the authorization
  question, now with evidence". Overstated — CodeQL does not model
  authorization, and these are authenticated-operator flows.
