---
title: "New Project Init: Existing or New Directory — Frontend Plan"
type: plan-frontend
status: in-development
lineage: new-project-init-directory-options
parent: lifecycle/requirements/new-project-init-directory-options-2.md
---

# New Project Init: Existing or New Directory — Frontend Plan

Adds the two-mode New Project experience to the SPA. The existing
`CreateProjectModal.vue` already has a single "Path" field + a "Check" button that calls
`checkDirectory`; this plan turns that into a mode-aware form with **Use existing
directory** and **Create new directory**, live per-mode validation, a resolved-path
preview (FR9), and correct handling of the distinct backend errors and notifications
defined in [[new-project-init-directory-options]] (backend plan). No graphical filesystem
picker — typed/pasted path only (resolved question).

Files in scope:
- `web/src/components/project/CreateProjectModal.vue` — the form.
- `web/src/api/projects.ts` — `createProject`, `checkDirectory`.
- `web/src/stores/project.ts` — `create`, `checkDirectory`.
- `web/src/types/api.ts` — `CreateProjectPayload`, `CheckDirectoryResult`,
  `ProjectSummary`.
- `web/src/stores/ui.ts` — `success`/`info` notifications.

## Milestone 1: Mode selector & adaptive form

**Description:** Add a `mode: 'existing' | 'new'` to the form state with a two-option
segmented control / radio group at the top of the modal body (FR1). Selecting a mode swaps
the shown fields: existing → a single absolute-path field; new → a parent-path field plus a
directory-name field (FR2, FR3). Clear stale validation state and `dirResult` on mode
switch. Project `name`, `description`, `owner` fields remain in both modes. No pre-fill of
the path/parent (resolved question).

**Files to change:**
- `web/src/components/project/CreateProjectModal.vue` — mode state, control, conditional
  field blocks, reset-on-switch.

**Acceptance criteria:**
- Exactly two mutually-exclusive modes are selectable; one is active at a time (FR1).
- Existing mode shows the path field only; new mode shows parent + directory-name fields
  only (FR2, FR3).
- Switching mode clears previous errors and any prior check result; no field is pre-filled.

## Milestone 2: Live per-mode validation & resolved-path preview

**Description:** Extend the "Check" affordance to call the mode-aware
`checkDirectory({mode, path | parent, name})`. Render the returned per-rule results:
existing → exists / is-directory / writable / already-initialised; new → parent-exists /
parent-writable / name-valid / target-exists. Always display the returned `resolvedPath`
(trimmed + `~`-expanded) so the user sees exactly what will be written (FR9). Client-side
guards mirror the backend: absolute path required in existing mode; directory name
non-empty and free of `/`, `\`, `..` in new mode (FR4) — surfaced inline before the request.

**Files to change:**
- `web/src/components/project/CreateProjectModal.vue` — check handler per mode, result
  rendering, resolved-path line.
- `web/src/api/projects.ts` / `web/src/stores/project.ts` — widen `checkDirectory`
  signature to pass mode/parent/name.
- `web/src/types/api.ts` — extend `CheckDirectoryResult` with the new-mode fields
  (`parentExists`, `parentWritable`, `nameValid`, `targetExists`, `resolvedPath`,
  `reason?`).

**Acceptance criteria:**
- Existing-mode check shows exists/writable/initialised as today, plus the resolved path.
- New-mode check shows parent-exists/parent-writable/name-valid/target-exists plus the
  resolved target path.
- An invalid directory name (`/`, `\`, `..`, empty) is flagged inline before any request
  (FR4).
- The resolved path shown matches the normalised path the backend will write (FR9).

## Milestone 3: Submit wiring, distinct errors & notifications

**Description:** Route submit through the mode-aware `create` with the full payload
(`{mode, name, description, owner, path?}` or `{mode, name, description, owner, parent,
dirName}`). Map each distinct backend error code to the right field/message (FR8):
`path_missing`, `not_a_directory`, `not_writable`, `target_exists`, `invalid_name`,
`already_initialised`, `parent_missing`, `parent_not_writable`, plus 409 name conflict.
On success emit `ui.success`; when the response reports `partialCompletion` (existing dir
had a partial `lifecycle/` tree that was completed), emit an **information notification**
per the resolved question. Show `resolvedPath` in the success message.

**Files to change:**
- `web/src/components/project/CreateProjectModal.vue` — `handleSubmit` per-mode payload,
  error-code → field mapping, success + info notifications.
- `web/src/api/projects.ts` / `web/src/stores/project.ts` — `create` accepts the extended
  payload and returns the extended summary.
- `web/src/types/api.ts` — extend `CreateProjectPayload` (mode/parent/dirName) and
  `ProjectSummary`/create result (`resolvedPath`, `alreadyInitialised`,
  `partialCompletion`).

**Acceptance criteria:**
- Each FR4/FR8 failure renders a specific, mode-appropriate message on the relevant field —
  no generic error for a known cause.
- Submitting an already-initialised target shows an "already initialised" message and does
  not report success.
- On success the modal closes/refreshes, the new project appears, and a completed-partial
  scaffold surfaces an info (not error) notification.
- The success path leaves the project ready to index without a separate manual "Init" step.

## Milestone 4: Accessibility, disabled/loading states & regression pass

**Description:** Ensure the mode control and both field sets are keyboard-navigable and
labelled; preserve existing disabled-while-submitting and spinner behaviour; keep Escape-to-
close. Verify the modal still works for the default existing-directory case identically to
today for users who ignore the new mode.

**Files to change:**
- `web/src/components/project/CreateProjectModal.vue` — aria labels on the mode control,
  focus handling, disabled states.

**Acceptance criteria:**
- Mode control is reachable and operable by keyboard with an accessible name.
- All inputs disable during submit; spinner shows; Escape closes the modal.
- Existing-directory flow is behaviourally unchanged from the current modal for that mode.
- `pnpm build` (via `make build-web`) succeeds with no type errors.
