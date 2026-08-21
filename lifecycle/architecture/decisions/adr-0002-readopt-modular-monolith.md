---
title: Re-adopt Modular Monolith with Go + Vue (High-Performance Lean Stack)
type: adr
status: approved
created: "2026-08-21T10:46:34+10:00"
---

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


## Rejected alternatives

- Serverless / Functions-as-a-Service
- Cloud-Native Microservices
