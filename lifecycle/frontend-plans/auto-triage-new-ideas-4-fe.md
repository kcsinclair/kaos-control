---
title: "Auto-Triage Raw Ideas — Frontend Plan"
type: plan-frontend
status: done
lineage: auto-triage-new-ideas
parent: lifecycle/requirements/auto-triage-new-ideas-2.md
assignees:
    - role: frontend-developer
      who: agent
---

# Auto-Triage Raw Ideas — Frontend Plan

Adds a "Triage now" action button to the artifact detail view for `raw` ideas, wired to `POST /api/p/:project/ideas/{slug}/triage`. No new view, no new route — reuses the existing `ArtifactEditorView`, agent-run-history panel, and role-gated button patterns already present in the codebase.

Cross-references: [[auto-triage-new-ideas-3-be]] (endpoint + run records + WS events the UI consumes), [[auto-triage-new-ideas-5-test]] (integration tests).

---

## Milestone 1 — API client function

### Description

Add a single typed wrapper around `POST /api/p/:project/ideas/{slug}/triage`. Follows the existing pattern in `web/src/api/` (one function per endpoint, returning a parsed JSON object or throwing `ApiError`).

### Files to change

- `web/src/api/ideas.ts` (extend if it exists; otherwise create):
  - `export interface TriageResponse { run_id: string }`.
  - `export interface TriageError { error: string; reason?: string }`.
  - `export async function triageIdea(project: string, slug: string): Promise<TriageResponse>`. On non-2xx, parse the body as `TriageError` and throw `new ApiError(status, body.error, body.reason)` consistent with the existing `ApiError` shape (verify by reading `web/src/api/client.ts` or equivalent).

### Acceptance criteria

- [ ] `pnpm exec vue-tsc --noEmit` passes.
- [ ] `pnpm build` produces a bundle that includes the new function.
- [ ] A unit test (Vitest, alongside existing `web/src/api/*.test.ts` if any exist) calls `triageIdea` against a mocked fetch and verifies: success → `{ run_id }`; 409 → throws `ApiError` with `reason` set; 401 → throws `ApiError` with status 401.

---

## Milestone 2 — "Triage now" button component

### Description

A small button component that renders only for `raw` ideas, gated on the current user's roles. Lives next to the existing `QueueWorkButton` in `ArtifactEditorView.vue` to keep the action cluster consistent.

### Files to change

- `web/src/components/artifact/TriageNowButton.vue` (new):
  - Props: `artifact: Artifact`, `project: string`.
  - Computed `visible`:
    - `artifact.type === 'idea'` AND `artifact.status === 'raw'` AND the current user (from the auth store) has at least one of `product-owner`, `analyst`, `reviewer` roles.
  - Renders nothing when `!visible` (so non-raw ideas don't show the button, and ineligible users see no UI affordance).
  - On click: call `triageIdea(project, artifact.lineage)`; set local `loading = true`; on success, surface a toast / inline notice "Triage started" and emit a `triage-started` event with the returned `run_id`; on `ApiError`, show the error reason inline (e.g. `"Cannot triage: wrong_status"` for the 409 case).
  - Uses the existing button styling — match `QueueWorkButton.vue` for visual consistency (same button classes, same `lucide-vue-next` icon, e.g. `Sparkles` or `Wand2`).
- `web/src/views/project/ArtifactEditorView.vue`:
  - Import `TriageNowButton` and render it adjacent to `<QueueWorkButton>` at line ~301. Pass `artifact` and `project`.

### Acceptance criteria

- [ ] The button renders for a `type: idea` / `status: raw` artifact viewed by a user whose roles include any of `product-owner`, `analyst`, or `reviewer`.
- [ ] The button does NOT render for `status: draft` ideas (covers the FR-2 negative case in the UI).
- [ ] The button does NOT render for users without any of the three permitted roles.
- [ ] The button does NOT render for `type: defect` or any non-`idea` type.
- [ ] Clicking the button while `loading` is `true` is a no-op (no duplicate request).
- [ ] On 409, the rendered error message names the server-provided `reason` (e.g. `wrong_status`, `locked`).
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` succeed.

---

## Milestone 3 — Surface the run in the existing agent-run-history panel

### Description

The agent run produced by triage is recorded in the existing `agent_runs` table by [[auto-triage-new-ideas-3-be]] with `agent_name = "idea-triage"` and `target_path = <artifact path>`. The existing `ArtifactRunHistory.vue` panel already lists runs keyed by `target_path`, so no new component is needed — but verify it refreshes when triage runs arrive.

### Files to change

- `web/src/components/artifact/ArtifactRunHistory.vue`:
  - Confirm the component subscribes to the WS event types the backend broadcasts on run start/complete. If it only listens to a hard-coded list (e.g. `agent.run.started`), extend that list to include whatever event the triage manager emits, or move to a generic `agent.run.*` subscription if the underlying hub already supports prefix matching. (Backend side already emits the standard agent-run events per [[auto-triage-new-ideas-3-be]] Milestone 5; the UI just needs to receive them.)
  - On receiving a run event whose `target_path` matches the currently displayed artifact, re-fetch the run list. No new code path is acceptable if the component already does this — in that case this milestone is purely a verification step plus any small fix.
- `web/src/components/artifact/TriageNowButton.vue`:
  - After a successful POST, optimistically prepend a placeholder row (`agent_name: 'idea-triage'`, `status: 'running'`, `run_id` from the response) to the run-history list via the Pinia store, or simply emit an event that the parent uses to trigger a re-fetch. Prefer the lighter "emit + parent re-fetches" path to avoid coupling the button to store internals.

### Acceptance criteria

- [ ] After clicking "Triage now", the agent-run-history panel for the artifact shows the new run with `agent_name: idea-triage` within ~1 s without a manual page refresh.
- [ ] When the run completes (success or failure), the row's `status` updates in place (verified by watching the WS event stream in DevTools and observing the row transition `running → success` or `running → failed`).
- [ ] On failure, the row's stderr is viewable through the existing run-detail modal (`RunDetailModal.vue`) — no UI changes needed if it already renders `stderr` from the run record.

---

## Milestone 4 — Status indicator for `raw` ideas in the artifact list

### Description

Operators should be able to tell at a glance which ideas are still `raw` and therefore candidates for (or stuck pending) triage. The existing `ArtifactListView.vue` already renders status; this milestone is a small visual tweak to make `raw` stand out, plus a list-level filter chip.

### Files to change

- `web/src/components/artifact/StatusDropdown.vue` (or wherever the status pill is rendered):
  - Confirm `raw` has a distinct colour token in the existing status colour map. If not, add one (e.g. a neutral grey) so the visual treatment is consistent with `draft`/`clarifying`/etc. No new vocabulary — `raw` is already in `KnownStatuses` per `internal/artifact/artifact.go`.
- `web/src/views/project/ArtifactListView.vue`:
  - Verify the status filter offers `raw` as an option (it should, because the dropdown is populated from indexed values). If `raw` is currently hidden behind a "show all" toggle, no change needed.

### Acceptance criteria

- [ ] The status pill for a `raw` idea renders with a recognisable colour distinct from `draft`.
- [ ] The artifact list's status filter includes `raw` and filtering by `raw` returns only ideas in that state.
- [ ] No changes are required to type-script types — `raw` is already a valid status string.

---

## Milestone 5 — End-to-end smoke pass

### Description

Final verification pass exercising the full flow in the running dev server. No file changes — this milestone documents the manual smoke run the agent must complete before marking the plan done.

### Files to change

- None (manual verification + screenshots).

### Smoke steps

1. `make run`.
2. Open the project in the browser; sign in as `keith@sinclair.org.au` (has all three permitted roles).
3. Use the brain-dump modal to create a `raw` idea via the existing `BrainDumpModal.vue` flow (or `cat > lifecycle/ideas/triage-smoke.md` with `status: raw`).
4. Within ~5 s, the watcher-triggered triage completes; the artifact's detail view shows `status: draft`, `## Raw Idea` / `## Idea` sections, and an `idea-triage` run in the history panel.
5. Manually set `status: raw` back via the status dropdown. The "Triage now" button reappears.
6. Click "Triage now". Confirm a 202 in the network tab, an immediate run row, and a `success` status when the run completes.
7. Sign out and sign back in as a synthetic `backend-developer`-only user (or temporarily edit the in-memory user roles fixture). Verify the button is hidden.

### Acceptance criteria

- [ ] All 7 smoke steps execute cleanly with no console errors.
- [ ] Screenshots of (a) the button visible on a `raw` idea, (b) the button hidden for an ineligible user, (c) the run appearing in history are attached to the implementation commit message.
- [ ] No regressions observed in adjacent flows (artifact list, queue, kanban) verified by clicking through them once.
