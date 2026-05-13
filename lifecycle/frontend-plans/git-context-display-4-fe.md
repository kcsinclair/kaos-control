---
title: "Git Context Display — Frontend Plan"
type: plan-frontend
status: draft
lineage: git-context-display
parent: lifecycle/requirements/git-context-display-2.md
---

# Git Context Display — Frontend Plan

This plan covers the Vue 3 components, Pinia store, and API integration needed to display git repository state in the sidebar. The implementation uses only existing dependencies: Vue 3, Pinia, Vue Router, `lucide-vue-next` (for the `GitBranch` icon), and the existing `useWebSocket` composable. The resolved-question in the requirement places the git info panel in the left sidebar, below the navigation menu items.

## Milestone 1: API Client Method

**Description:** Add a typed API client function to fetch git status from the [[git-context-display]] backend endpoint.

**Files to change:**
- `web/src/api/client.ts` (or a new `web/src/api/git.ts` if the project separates API modules) — add `fetchGitStatus(project: string): Promise<GitStatusResponse>`.
- `web/src/types/api.ts` — add `GitStatusResponse` type and register `"git.status"` in the `WsEventType` union.

**Details:**

```typescript
interface GitStatusResponse {
  available: boolean
  branch?: string
  dirty?: boolean
  head_sha?: string
  head_message?: string
  head_author?: string
  head_when?: string
}
```

The function calls `GET /api/p/{project}/git/status` and returns the typed response. Add `"git.status"` to the `WsEventType` union so the `useWebSocket` composable can subscribe to it.

**Acceptance criteria:**
- [ ] `fetchGitStatus` returns a correctly typed response for both git and non-git projects.
- [ ] `"git.status"` is a valid `WsEventType`.
- [ ] No new npm dependencies introduced (NFR2).

## Milestone 2: Git Status Pinia Store

**Description:** Create a lightweight Pinia store that holds the current git status and exposes actions to fetch and update it.

**Files to change:**
- `web/src/stores/gitStatus.ts` (new) — define `useGitStatusStore`.

**Details:**

```typescript
export const useGitStatusStore = defineStore('gitStatus', () => {
  const available = ref(false)
  const branch = ref('')
  const dirty = ref(false)
  const headSha = ref('')
  const headMessage = ref('')
  const headAuthor = ref('')
  const headWhen = ref('')

  async function fetch(project: string) { /* GET /api/p/{project}/git/status */ }
  function applyWsEvent(data: GitStatusResponse) { /* update refs from WS payload */ }

  return { available, branch, dirty, headSha, headMessage, headAuthor, headWhen, fetch, applyWsEvent }
})
```

The store is intentionally minimal — it holds the last-known state and provides two mutation paths: initial HTTP fetch and incremental WebSocket update.

**Acceptance criteria:**
- [ ] Store correctly populates state from the REST response.
- [ ] `applyWsEvent` updates all reactive refs from a WebSocket payload.
- [ ] Store resets to `available: false` when the project changes (prevents stale data from the previous project bleeding through).

## Milestone 3: GitStatusBar Component

**Description:** Create a `GitStatusBar.vue` component that renders branch name, dirty indicator, and last commit summary. Integrate it into `AppSidebar.vue` below the navigation list and above the version label (per the resolved question: "In the left menu bar under the application menu options include a panel for the git information").

**Files to change:**
- `web/src/components/layout/GitStatusBar.vue` (new) — the display component.
- `web/src/components/layout/AppSidebar.vue` — import and render `<GitStatusBar />` between the `<ul class="nav-list">` and `.sidebar-version` divs.

**Details:**

The component layout when `available` is `true` and the sidebar is expanded:

```
┌─────────────────────────┐
│ 🔀 main  ● modified     │  ← branch icon + name + dirty indicator
│ a1b2c3d  fix login bug  │  ← abbreviated SHA + first-line message
└─────────────────────────┘
```

When collapsed, show only the `GitBranch` icon (from `lucide-vue-next`) with a coloured dot overlay indicating dirty state — consistent with the sidebar's existing collapsed-badge pattern.

When `available` is `false`, render nothing (hidden, not a blank space) — FR4 requires graceful degradation.

**Implementation notes:**
- Use `GitBranch` icon from `lucide-vue-next` (already a project dependency) per resolved question 3.
- The dirty indicator should be a small coloured dot or the text "modified" / "clean" — keep it boolean per resolved question 2.
- The SHA should be rendered in monospace (`font-family: var(--font-mono)` or equivalent).
- Subscribe to `git.status` WebSocket events via `useWebSocket(project, 'git.status', handler)` for live updates.
- Call `gitStatusStore.fetch(project)` on mount and when the active project changes.

**Accessibility (NFR4):**
- The panel must have `role="status"` and `aria-label="Git repository status"` so screen readers announce it.
- The branch name and dirty state must be accessible: use `aria-label` on the dirty indicator (e.g. "Working tree has uncommitted changes" or "Working tree is clean").
- The component must be keyboard-navigable — no interactive elements are required (it's read-only display), but the container should be focusable with `tabindex="0"` so screen-reader users can navigate to it.

**Acceptance criteria:**
- [ ] `GitStatusBar` renders branch name with `GitBranch` icon when `available` is `true` (FR3).
- [ ] Dirty indicator shows boolean modified/clean state (FR3, resolved question 2).
- [ ] Abbreviated SHA and first-line commit message are displayed (FR3).
- [ ] Component is hidden when `available` is `false` (FR4).
- [ ] Component updates in real time when a `git.status` WebSocket event arrives (FR2, FR3).
- [ ] Component fetches fresh status on mount and when active project changes.
- [ ] Proper ARIA attributes for screen reader accessibility (NFR4).
- [ ] Respects sidebar collapsed/expanded state — shows icon-only view when collapsed.

## Milestone 4: Sidebar Integration and Styling

**Description:** Wire the `GitStatusBar` into the sidebar layout and ensure visual consistency with the existing design system (CSS custom properties, spacing, colour tokens).

**Files to change:**
- `web/src/components/layout/AppSidebar.vue` — add `<GitStatusBar />` in the template and import the store/composable wiring.
- `web/src/components/layout/GitStatusBar.vue` — scoped CSS using existing design tokens.

**Details:**

Place the component between the nav list and the version label:

```html
<ul class="nav-list"> ... </ul>
<!-- NEW: Git status panel -->
<GitStatusBar />
<div class="sidebar-version"> ... </div>
```

The panel should have:
- A top border (`border-top: 1px solid var(--color-border-dark)`) matching the sidebar footer.
- Padding consistent with `.sidebar-version` (`var(--space-2) var(--space-4)`).
- Text colours using `var(--color-sidebar-text)` and `var(--color-sidebar-text-muted)`.
- The dirty indicator dot should use `var(--color-warning)` when dirty and `var(--color-success)` when clean.
- In collapsed state, the panel shrinks to icon-only width matching `var(--sidebar-width-collapsed)`.

**Acceptance criteria:**
- [ ] Git panel is visually consistent with the rest of the sidebar (colours, spacing, fonts).
- [ ] Panel respects the collapsed/expanded sidebar state and animates with the same transition timing.
- [ ] No layout shift or overflow when branch names are long (use `text-overflow: ellipsis`).
- [ ] No new npm dependencies (NFR2).
