---
title: Architecture Summary
type: doc
status: approved
created: "2026-08-21T10:46:34+10:00"
---

# Architecture Summary

## Architecture-breaking requirements

The questionnaire surfaced none, but the following standing constraints are
genuinely architecture-breaking — a change that violates one invalidates the
chosen architecture or stack, and must be raised as a new ADR:

- **Single self-contained binary.** Distribution is one binary with no external
  database or services. This forbids cgo/native dependencies and mandates
  pure-Go, embeddable choices — pure-Go SQLite
  ([[adr-0003-pure-go-sqlite-index]]) and the embedded SPA
  ([[adr-0004-embedded-spa-single-binary]]). Introducing a required external
  datastore or a cgo dependency would break this.
- **Local filesystem is the source of truth.** Markdown artifacts under
  `lifecycle/` are authoritative; the index is a rebuildable cache
  ([[index-is-a-cache]]). External edits (e.g. Obsidian sync) must reconcile via
  the watcher. A design that made the database authoritative would break this.
- **Agents execute arbitrary tools.** Orchestrated agent runs mutate files and
  run commands, so tool calls must be mediated and scope-enforced
  ([[adr-0006-mediated-agent-driver-permission-model]],
  [[filesystem-sandboxing]]). Removing mediation would break the trust model.
- **Direct-served, no trusted proxy hop.** The binary serves HTTP/TLS directly;
  no reverse proxy is assumed, so client identity must not be derived from
  spoofable headers ([[adr-0001-no-header-based-client-ip-trust]]).

## Selection Q&A

- **Q:** Does this need to work fully offline, with no network connection required?
  **A:** No
- **Q:** Will multiple people view or edit shared data at the same time?
  **A:** Yes
- **Q:** Does it need realtime updates or streaming data?
  **A:** Yes
- **Q:** Do you expect high scale (many users or requests) from the start?
  **A:** No
- **Q:** Is this primarily a phone or tablet app?
  **A:** No
- **Q:** Is AI/ML central to what the product does?
  **A:** Yes
- **Q:** How much operational complexity can your team take on?
  **A:** Medium
- **Q:** Is minimising cost at launch a priority?
  **A:** Yes
- **Q:** What's your team's strongest language?
  **A:** Go

## Links

- Architecture: [modular-monolith.md](modular-monolith.md)
- Tech stack: [go-vue.md](go-vue.md)
- ADR: [adr-0002-readopt-modular-monolith.md](decisions/adr-0002-readopt-modular-monolith.md)
