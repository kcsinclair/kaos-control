---
title: Architecture Summary
type: doc
status: approved
created: "2026-08-21T10:46:34+10:00"
---

# Architecture Summary

## Architecture-breaking requirements

None surfaced by the questionnaire.

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
