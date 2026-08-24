---
title: Wizard Skip / Selective Scaffolding — Integration Tests
type: test
status: approved
lineage: wizard-skip-scaffolding
parent: lifecycle/test-plans/wizard-skip-scaffolding-5-test.md
created: "2026-08-22T15:10:00+10:00"
---

# Wizard Skip / Selective Scaffolding — Integration Tests

Covers Milestone 3 of [[wizard-skip-scaffolding]]'s test plan: the HTTP seam
for `GET`/`POST .../architecture/wizard/scaffold`, exercised end-to-end
through the integration harness. Milestones 1, 2, and 4 (Go unit tests under
`internal/`, and Vitest tests under `web/src/`) are backend-developer and
frontend-developer scope and are not covered here.

**File:** `tests/integration/architecture_wizard_scaffold_test.go`

Run with: `go test -tags integration ./tests/integration/... -run TestWizardScaffold`

| Test function | Scenario |
|---|---|
| `TestWizardScaffold_DirectivesScaffolder_GeneratesDirectiveFiles` | Adapted: the choice body now carries `"selected":true` alongside `use_defaults`, proving the explicit-selection contract still generates `AGENTS.md`/`CLAUDE.md` + `lifecycle/devops/{build,lint,test}.yaml` and reports `git_commands` for an unmanaged repo (FR-9) |
| `TestWizardScaffoldGet_ReportsPresence` | `GET .../wizard/scaffold` reports `steps[0].present == false` before any run, and `== true` after a commit + a `selected:true` run (FR-4) |
| `TestWizardScaffoldGet_AuthenticatedNonProductOwner_Allowed` | A logged-in `dev@test.local` (backend/frontend/test-developer roles, no product-owner) gets `200` from `GET .../wizard/scaffold` (NFR-3) |
| `TestWizardScaffoldPost_NonProductOwner_Forbidden` | The same non-product-owner user gets `403` from `POST .../wizard/scaffold`, and no `AGENTS.md`/`CLAUDE.md` are written (NFR-3) |
| `TestWizardScaffoldPost_SelectedFalse_WritesNothing` | `POST` with the step's only choice `selected:false` applies nothing, and writes neither the directive files nor the devops pipeline files — the single-step scaffolder's stand-in for "zero selection" (FR-9, FR-10, FR-11) |
| `TestWizardScaffoldPost_NoNewArchitectureTreeArtefacts` | After commit, a `selected:true` scaffold run leaves the `lifecycle/architecture/` file count unchanged — the run writes to the project root and `lifecycle/devops/`, not the architecture tree, so promotion remains the only thing that grows it (NFR-5) |
| `TestWizardScaffoldGet_NoScaffolderRegistered_ReturnsUnavailable` | Unchanged (NFR-4): with no `Scaffolder` registered, `GET` reports `available:false` with the "not yet available" message |
| `TestWizardScaffoldPost_NoScaffolderRegistered_GracefulNoWrites` | Unchanged (NFR-4): with no `Scaffolder` registered, `POST` reports the same graceful response and writes nothing under the project |
| `TestWizardCommit_WithoutScaffolding_YieldsCompleteProject` | Unchanged: a wizard committed without ever touching the scaffold endpoints still produces the complete, valid outcome |

All 9 test functions in the file pass under `go test -tags integration ./tests/...`.
`TestOnboard_ExistingMode_AlreadyInitialised` (a different, pre-existing
project-init test in `project_init_directory_options_test.go`) fails on this
branch independent of this change — confirmed failing identically on the
commit before this work landed.
