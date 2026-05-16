---
title: Queue Work button missing on approved doc artifact; tech-writer agent not routed
type: defect
status: in-development
lineage: tech-writer-agent
parent: lifecycle/tests/tech-writer-agent-6-test.md
labels:
    - defect
release: KC-Release2
assignees:
    - role: frontend-developer
      who: agent
---

# Queue Work button missing on approved doc artifact; tech-writer agent not routed

## Reproduction Steps

**TC1 / TC2 — Queue Work button:**

1. Navigate to an `approved` doc artifact detail page (e.g., `lifecycle/docs/smoke-doc-approved.md`).
2. Wait for the page to load (note: this is blocked by `tech-writer-agent-10-defect.md`).
3. Look for `.btn-queue` or `button:has-text("Queue Work")`.
4. If found, click it and intercept the resulting `POST /api/queue` request.

## Expected Behaviour

- A "Queue Work" button is visible on `approved` doc artifacts.
- Clicking it sends `POST /api/queue` with `{"agent": "tech-writer", "artifact_path": "lifecycle/docs/smoke-doc-approved.md", ...}`.

## Actual Behaviour

- TC1: No `.btn-queue` or "Queue Work" button found (timeout 10 s). The page also shows "project not found" (see `tech-writer-agent-10-defect.md`), blocking the entire artifact view.
- TC2: Queue Work button not visible, so click cannot proceed.

```
Error: expect(locator).toBeVisible() failed
Locator: locator('.status-badge, [data-status]').first()
Timeout: 10000ms — element(s) not found
```

**TC3 — Agents endpoint ready_count:**

The `GET /api/p/testproject/agents` endpoint does not include a `tech-writer` agent entry, or returns `ready_count = 0` even though an `approved` doc fixture exists.

```
flows/08-doc-queue.spec.ts:74:24 — TypeError at res.json(): 
  agents list does not contain {name: "tech-writer"} or ready_count < 1
```

This is likely a backend gap: the agents/ready-count endpoint does not account for `doc` type artifacts as input for the `tech-writer` agent.

**Failing tests:** `Flow 08 TC1`, `Flow 08 TC2`, `Flow 08 TC3` (`tests/e2e/flows/08-doc-queue.spec.ts`).

---

*TC3 (agents endpoint) may require a backend fix; TC1/TC2 (Queue Work button) require a frontend fix. Both are filed here for triage.*
