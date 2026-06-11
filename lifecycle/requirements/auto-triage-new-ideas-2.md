---
title: Auto-Triage Raw Ideas Into Drafts
type: requirement
status: done
lineage: auto-triage-new-ideas
parent: lifecycle/ideas/auto-triage-new-ideas.md
labels:
    - agent
    - agents
    - workflow
    - artifacts
    - process
release: KC-Release3
assignees:
    - role: analyst
      who: agent
    - role: product-owner
      who: agent
---

# Auto-Triage Raw Ideas Into Drafts

## Problem

Ideas captured through quick-capture flows (and any other path that produces `status: raw` artifacts under `lifecycle/ideas/`) currently sit in the `raw` state until a human or analyst manually intervenes. The lifecycle is intended to flow `raw → draft → clarifying → approved → …`, but nothing automates the first hop. As a result:

- `raw` artifacts accumulate as a backlog of unstructured brain-dumps that downstream agents and the agent launcher cannot act on (the launcher only offers `approved` predecessors, and analyst/clarifying flows expect at minimum `draft`).
- The original brain-dump text is the only content present, with no structured representation suitable for downstream `requirements-analyst` consumption.
- Operators must remember to manually edit each new idea, restructure the body, and transition the status. There is no provenance trail showing the unedited original alongside the cleaned-up version.

The lifecycle state machine already permits `raw → draft` for the `system`, `analyst`, and `product-owner` roles ([internal/workflow/workflow.go](../../internal/workflow/workflow.go) line 29), and an `idea-capture` agent with an `idea-generate` prompt that produces structured ideas already exists in [lifecycle/config.yaml](../config.yaml). What is missing is an automated triage step that wires these pieces together and runs whenever a `raw` idea appears on disk or via the API.

## Goals / Non-goals

### Goals

- Automatically detect new or modified artifacts under `lifecycle/ideas/` whose `status` is `raw` and enrich them into a `draft` idea without operator intervention.
- Preserve the original brain-dump verbatim under a `## Raw Idea` heading; place the LLM-produced structured idea under a sibling `## Idea` heading. Original text must never be silently overwritten.
- Reuse the existing structured-idea generation logic (the `idea-generate` prompt template on the `idea-capture` agent in [lifecycle/config.yaml](../config.yaml)) rather than introducing a parallel implementation.
- Transition the artifact from `status: raw` to `status: draft` via the workflow state machine, recording the transition as a `system` actor so existing role-based audit logging captures it.
- Make triage triggerable both:
  - automatically by the file watcher when a `raw` idea appears, and
  - on demand via an authenticated REST endpoint and a UI action (so an operator can re-run triage on a single artifact).
- Configure the agent (prompt, write scope, model, on-failure behaviour) in `lifecycle/config.yaml`, consistent with how the other agents are wired.
- Survive transient failures: a triage failure must leave the artifact in `raw` (not partially modified) and be visible in agent run history.

### Non-goals

- Triaging artifact types other than `idea` (e.g. there is no auto-triage for `raw` defects in this requirement, even though `raw` is a valid status in the vocabulary).
- Producing requirements, plans, or any downstream artifact from the triaged idea. Triage stops at `draft`; promoting `draft → clarifying → approved` and onward remains the operator's / analyst's responsibility.
- Changing the workflow state machine, the `KnownStatuses` vocabulary, or the `idea-capture` agent's existing prompts/JSON schema.
- Introducing distributed scheduling, queues, or multi-instance coordination. Single-process, single-node only.
- Adding new label vocabularies or modifying how labels are picked. The existing `idea-generate` rules (only labels from the project vocabulary) apply unchanged.

## Detailed Requirements

### Functional

1. **FR-1 — Trigger sources.** The triage subsystem MUST run an idea through the agent when ANY of the following occur:
   - The fsnotify watcher reports a created or modified file under `lifecycle/ideas/` and the reindexed artifact's frontmatter has `status: raw` and `type: idea`.
   - On server startup, after the initial full scan, every `lifecycle/ideas/*.md` artifact with `status: raw` and `type: idea` is enqueued for triage (to recover from artifacts that appeared while the server was offline).
   - An authenticated request to a new endpoint `POST /api/ideas/{slug}/triage` is received (see FR-8).

2. **FR-2 — Eligibility filter.** Only artifacts that satisfy ALL of the following are triaged:
   - File path matches `lifecycle/ideas/*.md`.
   - Frontmatter `type` is `idea`.
   - Frontmatter `status` is exactly `raw`.
   - The artifact is not already being processed by an in-flight triage run for the same path (deduplication — see FR-7).

3. **FR-3 — Body transformation.** On successful triage, the artifact's body MUST be rewritten as follows, preserving the leading H1 title heading if present:
   - The existing body content (everything below the H1, or the entire body if no H1 exists) is moved under a new `## Raw Idea` heading, verbatim and unchanged.
   - The structured idea produced by the agent (the `body` field of the `idea-generate` JSON, stripped of its own H1) is appended under a new `## Idea` heading.
   - The H1 title at the top of the file is preserved. If the agent's proposal includes a different title, the frontmatter `title` MAY be updated but the body H1 MUST match the (possibly updated) frontmatter title.
   - Re-running triage on an artifact that already has both `## Raw Idea` and `## Idea` sections (e.g. via the on-demand endpoint) MUST replace the `## Idea` section in place and leave `## Raw Idea` untouched.

4. **FR-4 — Frontmatter mutation.** On successful triage, the following frontmatter fields MUST be updated:
   - `status`: set to `draft`.
   - `labels`: merge the agent's proposed labels with any existing labels, de-duplicated, preserving order (existing first, new appended). Labels not present in the project vocabulary MUST be discarded.
   - `priority`: set from the agent's proposal only if no `priority` is already present in frontmatter; existing values are preserved.
   - All other frontmatter fields (`lineage`, `parent`, `release`, `created`, `assignees`, custom keys) MUST be left unchanged.

5. **FR-5 — Status transition path.** The status change MUST go through the same workflow transition path used by any other actor (i.e. the `system` role transitioning `raw → draft`), so that any existing transition hooks, audit logs, and WebSocket events fire as they would for a manual edit.

6. **FR-6 — Agent configuration.** The triage agent MUST be defined in `lifecycle/config.yaml` under the existing `agents:` list. Required fields:
   - `name`: a stable identifier (e.g. `idea-triage`).
   - `role`: includes `analyst` (or `product-owner` — see Open Questions) so the existing transition role check succeeds.
   - `driver`: reuses an existing driver (`inline` preferred so it consumes the existing `idea-generate` prompt logic and JSON contract).
   - `allowed_write_paths`: `lifecycle/ideas` only.
   - `active_status`: `draft` (the status the agent sets on the artifact it produces, consistent with the convention used by `requirements-analyst` and others).
   - `source_types`: `[idea]`.
   - `prompt_templates`: either references the existing `idea-generate` template verbatim or supplies an equivalent template that obeys the same JSON contract.

7. **FR-7 — Concurrency and deduplication.** The triage worker MUST:
   - Process at most one triage run per artifact path at a time. A second trigger for the same path while a run is in flight is coalesced (no second run is started; the in-flight result satisfies the request).
   - Run at most N concurrent triage jobs across all artifacts (N configurable, default 2) to avoid hammering upstream LLM APIs.
   - Use the existing lineage lock manager ([internal/lock](../../internal/lock)) to acquire a write lock on the lineage before mutating the file.

8. **FR-8 — REST API.** Add `POST /api/ideas/{slug}/triage`:
   - Authenticated callers with the `product-owner`, `analyst`, or `reviewer` role MAY call it. Unauthenticated requests return 401; authenticated callers without those roles return 403.
   - Looks up the artifact by lineage slug under `lifecycle/ideas/`. Returns 404 if not found.
   - If the artifact is not eligible (FR-2), returns 409 with a body that names the reason.
   - On success returns 202 with a JSON body containing the agent run ID; the actual triage completes asynchronously.
   - Once the run completes (success or failure), the standard `artifact.indexed` and agent run WebSocket events broadcast as usual.

9. **FR-9 — UI surface.** The artifact detail view for `raw` ideas MUST show a "Triage now" action button visible to users with the roles permitted by FR-8. Clicking it calls the endpoint and surfaces the resulting run in the existing agent run history panel. No new UI route is required; reuse existing components.

10. **FR-10 — Failure handling.** If the agent returns an error, returns malformed JSON (action other than `propose`), or its proposed body is empty:
    - The artifact MUST remain in `status: raw` with its body unchanged.
    - The run MUST be recorded as failed in agent run history with stderr/stdout captured per the existing agent run conventions.
    - A structured log line at warn level is emitted with the artifact path and the failure reason.
    - The triage subsystem MUST NOT enter a retry loop on its own. Retry is only via the explicit trigger sources in FR-1 (e.g. a subsequent file modification, a restart, or an operator hitting the endpoint).

### Non-functional

11. **NFR-1 — Idempotency.** Triggering triage on an already-triaged artifact (status `draft`) MUST be a no-op at the eligibility layer (FR-2) — the run is never started and no body or frontmatter mutation occurs.

12. **NFR-2 — Observability.** Emit structured logs at info level for triage start/complete and warn/error for failures. Each log line MUST include the artifact relative path, the lineage slug, and (on completion) the duration in milliseconds.

13. **NFR-3 — Performance.** A successful triage run MUST add no more than a single SQLite write and a single file write per artifact. The fsnotify-triggered path MUST not introduce any additional polling loops; it reuses the existing debounce window (150 ms) before evaluating eligibility.

14. **NFR-4 — Security.** The triage agent's write scope MUST be enforced by the existing `allowed_write_paths` policy ([internal/agent/policy.go](../../internal/agent/policy.go)). Any attempt to write outside `lifecycle/ideas/` MUST be denied and the run failed.

15. **NFR-5 — Compatibility.** No changes to `KnownStatuses`, no changes to the lineage filename convention, no changes to other agents' prompts or write paths.

## Acceptance Criteria

- [ ] Creating a new file at `lifecycle/ideas/foo.md` with frontmatter `type: idea` and `status: raw` causes triage to run within ~1 second of the file system notification.
- [ ] After successful triage, the file's frontmatter shows `status: draft`, the body contains `## Raw Idea` with the original text verbatim, and `## Idea` with the agent-generated structured content.
- [ ] Re-running triage on the same artifact via `POST /api/ideas/{slug}/triage` (after manually resetting status to `raw`) replaces the `## Idea` section and leaves `## Raw Idea` byte-for-byte identical to the previous run's `## Raw Idea` block.
- [ ] An artifact whose `status` is already `draft` (or any non-`raw` value) does not trigger triage when modified; no agent run is recorded.
- [ ] An artifact whose `type` is not `idea` (e.g. a misplaced `defect` under `lifecycle/ideas/` during development) is not triaged; no agent run is recorded.
- [ ] On server startup with a `raw` idea already on disk, the artifact is triaged within ~5 seconds of the indexer completing its initial scan.
- [ ] When two file modifications for the same artifact arrive within the debounce window while a triage run is in flight, only one agent run is started (verified via agent run history count).
- [ ] When `POST /api/ideas/{slug}/triage` is called with a slug that does not resolve to a file, the response is HTTP 404; when called against a `draft` artifact, the response is HTTP 409 with a reason naming the current status; when called unauthenticated, the response is HTTP 401.
- [ ] The triage agent is defined in `lifecycle/config.yaml` and is loaded successfully on startup (verified via `GET /api/agents`).
- [ ] When the agent returns invalid JSON, the artifact remains at `status: raw` with its original body, an agent run record exists with status `failed`, and a warn-level log line names the artifact path and the parse failure.
- [ ] Writes attempted by the triage agent outside `lifecycle/ideas/` are denied by policy; the run fails and no file outside the scope is modified.
- [ ] The artifact detail view for a `raw` idea shows a "Triage now" button visible to authenticated `product-owner`/`analyst`/`reviewer` users and hidden for other roles.
- [ ] Related: [[auto-triage-new-ideas]], [[agent-task-scheduler]] (concurrency/queue patterns), [[analyst-agent-sees-draft-ideas]] (downstream consumer that depends on `approved`, not `draft` — triage does not change that).

## Resolved Questions

- Which role should the triage agent run as for the workflow `raw → draft` transition: `system` (matches the spirit of an automated step and is already permitted by the state machine), `analyst`, or `product-owner`? `system` is recommended; confirm before implementation.

> product-owner

- Should the `priority` proposed by the agent be applied when the existing frontmatter `priority` matches the project default (e.g. `medium`), or only when `priority` is absent entirely? The current FR-4 wording chooses "only when absent" for safety; flag if a smarter merge is wanted.

> Include a priority and set to normal as the default

- Should the structured-idea generation be done with the existing `inline` driver and `idea-generate` template (lowest-friction, no new dependency surface), or via the `claude-mediated` driver to gain bash-allowlist policy and run-log streaming consistent with `requirements-analyst`? The former is recommended; confirm before implementation.

> Yes, reuse existing inline driver.

- Is the default concurrency cap of 2 simultaneous triage runs (NFR-3 / FR-7) appropriate, or should it default to 1 to be conservative with LLM spend?

> 2 for concurrency works.
