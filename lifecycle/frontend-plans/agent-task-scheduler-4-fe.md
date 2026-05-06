---
title: "Agent & Task Scheduler — Frontend Plan"
type: plan-frontend
status: done
lineage: agent-task-scheduler
parent: lifecycle/requirements/agent-task-scheduler-2.md
created: "2026-05-06T00:00:00+10:00"
assignees:
    - role: frontend-developer
      who: agent
---

# Agent & Task Scheduler — Frontend Plan

Implements the Scheduler UI: a new top-level navigation section with a jobs list view, a job detail/edit view, a create-job modal, and real-time status updates via WebSocket. Follows existing patterns from `AgentsRunsView` and `ArtifactListView`.

Cross-references: [[agent-task-scheduler-3-be]] (backend API and WS events), [[agent-task-scheduler-5-test]] (integration tests).

---

## Milestone 1 — API client and types

### Description
Add TypeScript types for scheduler domain objects and API client functions that call the backend endpoints defined in [[agent-task-scheduler-3-be]].

### Files to change
- `web/src/types/api.ts` — add `SchedulerJob`, `SchedulerRun`, `ScheduleSpec`, `Precondition`, `RunStatus` types. Add `scheduler.job.started` and `scheduler.job.completed` to `WsEventType`.
- `web/src/api/scheduler.ts` (new) — API functions: `listJobs`, `getJob`, `createJob`, `updateJob`, `deleteJob`, `triggerJob`, `pauseJob`, `resumeJob`, `listRuns`, `getRunLog`.

### Type definitions

```typescript
interface ScheduleSpec {
  type: 'cron' | 'interval' | 'once';
  expression: string;  // cron expr, duration string, or ISO datetime
}

interface Precondition {
  type: 'after_job' | 'file_exists' | 'http_ok' | 'shell';
  value: string;
}

interface SchedulerJob {
  name: string;
  target_type: 'agent' | 'shell';
  target: string;
  args?: Record<string, string>;
  schedule: ScheduleSpec;
  preconditions?: Precondition[];
  enabled: boolean;
  priority: number;
  timeout_sec: number;
  next_run_at?: string;
  last_run_status?: RunStatus;
  last_run_at?: string;
  created_at: string;
  updated_at: string;
}

type RunStatus = 'running' | 'success' | 'failure' | 'timeout' | 'skipped';

interface SchedulerRun {
  id: number;
  job_name: string;
  start_time: string;
  end_time?: string;
  status: RunStatus;
  log_path?: string;
}
```

### Acceptance criteria
- [ ] All API functions compile and correctly type their request/response payloads.
- [ ] Error responses are surfaced as `ApiError` consistent with existing client pattern.

---

## Milestone 2 — Pinia store

### Description
Create a `scheduler` Pinia store (composition API style) that manages job list state, individual job detail, and reacts to WebSocket events.

### Files to change
- `web/src/stores/scheduler.ts` (new) — `useSchedulerStore()` with refs for `jobs`, `selectedJob`, `runs`, loading flags. Actions: `fetchJobs`, `fetchJob`, `createJob`, `updateJob`, `deleteJob`, `triggerJob`, `pauseJob`, `resumeJob`, `fetchRuns`, `fetchRunLog`. WS handler: `onWsEvent(event)`.
- `web/src/views/WorkspaceView.vue` — add scheduler WS event fan-out to the store (same pattern as agents/locks stores).

### Acceptance criteria
- [ ] `fetchJobs` populates the `jobs` ref; views reactively update.
- [ ] `scheduler.job.started` WS event updates the matching job's `last_run_status` to `running` in real time.
- [ ] `scheduler.job.completed` WS event updates `last_run_status` and `last_run_at` and triggers a toast via `uiStore`.
- [ ] Failed job completions show an error-styled toast with the job name.

---

## Milestone 3 — Navigation and routing

### Description
Add the "Scheduler" entry to the sidebar and configure routes for the list and detail views.

### Files to change
- `web/src/components/layout/AppSidebar.vue` — add a nav item `{ label: 'Scheduler', to: '/p/${project}/scheduler', icon: CalendarClock }` (from `lucide-vue-next`). Position it after "Agents" in the nav items array.
- `web/src/router/index.ts` — add child routes under the project workspace:
  - `scheduler` → `SchedulerListView` (lazy loaded)
  - `scheduler/:name` → `SchedulerDetailView` (lazy loaded)

### Acceptance criteria
- [ ] "Scheduler" appears in the sidebar for all authenticated users.
- [ ] Clicking it navigates to the scheduler list view.
- [ ] Clicking a job name navigates to the detail view.
- [ ] Browser back/forward works correctly between list and detail.

---

## Milestone 4 — Jobs list view

### Description
Build the main scheduler list view showing all jobs with their status, schedule, next run time, and action buttons.

### Files to change
- `web/src/views/SchedulerListView.vue` (new)

### Layout
Follow the established list-view pattern (`ArtifactListView`, `AgentsRunsView`):
- **Header bar**: title "Scheduler" + "New Job" button (opens create modal).
- **Filter bar**: dropdowns for status filter (all / enabled / paused) and target type (all / agent / shell).
- **Table** with columns: Name (link to detail), Target, Schedule, Priority, Last Run (badge with status colour), Next Run, Enabled (toggle), Actions (trigger / pause-resume / delete).
- **Pagination** using `usePagination` composable and `<TablePagination>`.
- **Sorting** via `useSortableTable` composable on Name, Priority, Next Run, Last Run columns using `<SortHeader>`.

### Behaviour
- On mount: `schedulerStore.fetchJobs()`.
- Trigger button: calls `schedulerStore.triggerJob(name)`, shows success toast.
- Pause/Resume toggle: calls `pauseJob`/`resumeJob`, updates inline.
- Delete: confirmation modal, then `deleteJob`, removes row.
- Real-time: WS events update run status badges without page refresh.

### Acceptance criteria
- [ ] All jobs are listed with correct data in each column.
- [ ] Clicking "New Job" opens the create modal (Milestone 6).
- [ ] Trigger, pause/resume, and delete actions work and reflect immediately in the UI.
- [ ] Status badges use consistent `data-status` colouring: `success` → green, `failure` → red, `timeout` → amber, `running` → blue, `skipped` → grey.
- [ ] Table is sortable and paginated.
- [ ] Empty state shows a message and a "Create your first job" CTA.

---

## Milestone 5 — Job detail view

### Description
Build the detail view for a single job, showing its configuration, upcoming schedule preview, and run history.

### Files to change
- `web/src/views/SchedulerDetailView.vue` (new)

### Layout
- **Header**: job name, status badge, action buttons (trigger, pause/resume, edit, delete).
- **Config section**: read-only display of target, schedule, preconditions, priority, timeout. "Edit" button switches to edit mode (inline form, same as create form fields).
- **Upcoming runs**: a short list (next 5 computed fire times) based on the schedule expression, rendered client-side if possible or fetched from backend.
- **Run history table**: paginated table of recent runs — columns: Run ID, Started, Duration, Status (badge), Log (view button).
- **Log viewer**: clicking "View Log" on a run opens a modal or expandable section with the log content fetched via `getRunLog`, rendered in a `<pre>` block with monospace font.

### Acceptance criteria
- [ ] Detail view loads job config and run history on mount.
- [ ] Edit mode allows changing schedule, target, preconditions, priority, timeout, and saving via `updateJob`.
- [ ] Run history is paginated and sorted by most recent first.
- [ ] Log viewer displays the full log output with preserved formatting.
- [ ] Real-time WS updates refresh run history when a new run completes.

---

## Milestone 6 — Create/edit job modal and form

### Description
Build the form component used by both the create modal (from list view) and the inline edit mode (in detail view).

### Files to change
- `web/src/components/scheduler/JobForm.vue` (new) — reusable form component.

### Form fields
| Field | Input type | Validation |
|-------|-----------|------------|
| Name | text input | Required, alphanumeric + hyphens, 1–64 chars. Disabled in edit mode. |
| Target type | radio: Agent / Shell | Required |
| Target | text input (shell path) or select (agent role from project config) | Required; switches based on target type |
| Schedule type | radio: Cron / Interval / Once | Required |
| Schedule expression | text input | Required; placeholder changes per type (e.g. `0 2 * * *`, `30m`, ISO datetime) |
| Priority | number input or range slider | 1–10, default 5 |
| Timeout | text input (duration) | e.g. `30m`, `1h` |
| Preconditions | repeatable group: type select + value input, with add/remove buttons | Optional |
| Args | key-value pair editor (add/remove rows) | Optional |

### Acceptance criteria
- [ ] Form validates all fields client-side before submission.
- [ ] Target field switches between a text input and a dropdown when target type changes.
- [ ] Preconditions can be added, removed, and reordered.
- [ ] Args can be added and removed as key-value pairs.
- [ ] In create mode: submits to `createJob`, closes modal, refreshes list.
- [ ] In edit mode: submits to `updateJob`, exits edit mode, refreshes detail.
- [ ] Server-side validation errors (e.g. sandbox violation, duplicate name) are displayed inline.

---

## Milestone 7 — Styles and polish

### Description
Apply consistent styling using existing design tokens and ensure the scheduler views match the visual language of the rest of the app.

### Files to change
- `web/src/views/SchedulerListView.vue` — scoped styles.
- `web/src/views/SchedulerDetailView.vue` — scoped styles.
- `web/src/components/scheduler/JobForm.vue` — scoped styles.

### Acceptance criteria
- [ ] All scheduler views use CSS custom properties from `styles/tokens.css`.
- [ ] Status badges match the existing `[data-status]` colour scheme.
- [ ] Tables, forms, modals, and buttons are visually consistent with other views.
- [ ] Responsive: views remain usable at 1024px viewport width.
- [ ] No horizontal scrollbar at standard viewport widths.
- [ ] Loading states show appropriate indicators (consistent with existing patterns).
