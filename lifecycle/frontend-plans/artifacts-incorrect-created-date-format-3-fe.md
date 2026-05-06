---
title: "Fix Incorrect Created Date Format — Frontend Plan"
type: plan-frontend
status: done
lineage: artifacts-incorrect-created-date-format
parent: lifecycle/defects/artifacts-incorrect-created-date-format.md
---

# Frontend Plan: Fix Incorrect Created Date Format

The frontend receives the `created` field from the API as a string. While the backend fix will normalise storage, the frontend should handle both plain-date and RFC3339 formats gracefully during the transition period, and display dates consistently.

## Milestone 1: Robust date parsing for the `created` field

**Description:** Audit all locations where the `created` field from artifact data is parsed or displayed. Ensure the parsing logic accepts both `"2026-04-27"` (plain date) and `"2026-04-27T00:00:00+10:00"` (RFC3339) without errors, displaying a consistent human-readable format in either case.

**Files to change:**
- `web/src/` — search for references to `created`, `createdAt`, or `Created` in components and stores. Likely locations:
  - Artifact detail/metadata components that render the created date.
  - Any utility or formatter function used for date display.
- If no centralised date formatter exists, add a small helper (e.g. `web/src/utils/date.ts`) that normalises both formats to a `Date` object.

**Acceptance criteria:**
- An artifact with `created: "2026-04-27"` renders a valid, readable date in the UI (not "Invalid Date" or epoch).
- An artifact with `created: "2026-04-27T00:00:00+10:00"` continues to render correctly.
- An artifact with a missing `created` field shows a sensible fallback (e.g. the git-derived date, or "Unknown").
- No console errors or warnings when parsing either format.

## Milestone 2: Consistent date display format

**Description:** Ensure all created-date displays across the SPA use the same locale-aware format (e.g. `27 Apr 2026` or the user's browser locale). If different components currently format dates differently, unify them through the shared helper from Milestone 1.

**Files to change:**
- Any component identified in Milestone 1 that formats dates inline rather than through a shared function.

**Acceptance criteria:**
- The created date is displayed in the same format on the artifact list, artifact detail view, and any other surface where it appears.
- The format respects the user's browser locale settings.

## Cross-links

- [[artifacts-incorrect-created-date-format]] — the originating defect
- The [[artifacts-incorrect-created-date-format-2-be|backend plan]] normalises dates at the storage layer; this plan handles the display layer.
- The [[artifacts-incorrect-created-date-format-4-test|test plan]] covers UI date rendering verification.
