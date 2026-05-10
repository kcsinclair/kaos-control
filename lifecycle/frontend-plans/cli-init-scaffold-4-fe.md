---
title: "CLI Init Scaffold — Frontend Plan"
type: plan-frontend
status: approved
lineage: cli-init-scaffold
parent: lifecycle/requirements/cli-init-scaffold-2.md
---

# CLI Init Scaffold — Frontend Plan

The `kaos-control init` command is a CLI-only feature — it runs before the server starts and produces files on disk. There is no direct UI for invoking it. However, the frontend should handle the case where a registered project has **not** been initialised (missing `lifecycle/config.yaml`) gracefully, rather than showing cryptic errors. This plan covers that guard and a minor informational display.

## Milestone 1: Project Load Error Handling — "Not Initialised" State

**Description:** When the backend fails to load a project because `lifecycle/config.yaml` is missing, the API already returns an error. The frontend should detect this specific failure mode and show a helpful message directing the user to run `kaos-control init` rather than displaying a generic error.

**Files to change:**
- `web/src/stores/project.ts` — In the project-loading action, detect a response indicating a missing project config (e.g., HTTP 404 or a specific error code/message from the backend). Set a reactive `initRequired: boolean` flag on the store.
- `web/src/views/project/ProjectShell.vue` (or equivalent layout wrapper) — When `initRequired` is true, render an `InitRequiredBanner` instead of the normal project content.
- `web/src/components/project/InitRequiredBanner.vue` (new) — A centered card component displaying:
  - An icon (e.g., `FolderPlus` from lucide-vue-next).
  - Heading: "Project not initialised".
  - Body: "Run `kaos-control init <path>` to scaffold the lifecycle directory structure."
  - A copyable code block with the command, pre-filled with the project path.

**Acceptance criteria:**
- [ ] When a project's `lifecycle/config.yaml` is missing, the banner renders instead of the normal views.
- [ ] The banner includes the project path in the suggested command.
- [ ] Normal project views render as before when the project is properly initialised.
- [ ] No console errors or unhandled promise rejections on the "not initialised" path.

## Milestone 2: CLI Output Reference in Documentation View

**Description:** If the project has a `CLAUDE.md` at its root (created by `kaos-control init`), the existing artifact/file viewer should be able to display it. Verify that the current markdown rendering pipeline handles `CLAUDE.md` files that are outside the `lifecycle/` directory. No changes expected if the viewer already supports arbitrary project-root files; document findings.

**Files to change:**
- Likely no changes needed — verify only. If the viewer is restricted to `lifecycle/**/*.md`, file a follow-up defect.

**Acceptance criteria:**
- [ ] `CLAUDE.md` at the project root is viewable through the file browser if one exists in the UI.
- [ ] If the viewer cannot render files outside `lifecycle/`, a defect is raised rather than a workaround added here.

## Cross-references

- [[cli-init-scaffold-3-be]] — Backend plan: the init command produces the files this plan's guard checks for. The "not initialised" state is specifically the absence of `lifecycle/config.yaml` which the backend's `config.LoadProject()` requires.
- [[cli-init-scaffold-5-test]] — Test plan: includes an E2E test for the "not initialised" banner.
