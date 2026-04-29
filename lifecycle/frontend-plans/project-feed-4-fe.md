---
title: "Frontend Plan: Project Feed"
type: plan-frontend
status: done
lineage: project-feed
parent: lifecycle/requirements/project-feed-2.md
created: "2026-04-29"
---

# Frontend Plan: Project Feed

relates-to: [[project-feed]]

## Overview

Add a `/feed` route with a `ProjectFeedView.vue` component that renders a real-time, chronological activity stream. The view consumes the `GET /api/p/{project}/feed` endpoint from [[project-feed-3-be]] and listens for `feed.new` WebSocket events. It includes infinite scroll, event-type filter chips, and keyboard navigation.

---

## Milestone 1 — Feed API types and service

### Description

Define TypeScript types for feed events and create an API service module.

### Files to change

- `web/src/types/api.ts`
  - Add `FeedEvent` interface:
    ```ts
    export interface FeedEvent {
      id: number
      event_type: string
      timestamp: number
      actor: string
      artifact_path?: string
      run_id?: string
      summary: string
      payload_json?: string
    }
    ```
  - Add `FeedResponse` interface:
    ```ts
    export interface FeedResponse {
      events: FeedEvent[]
      next_cursor: number | null
    }
    ```
  - Add `'feed.new'` to the `WsEventType` union (line ~168).

- `web/src/api/feed.ts` (new file)
  - Export `fetchFeed(project: string, params?: { limit?: number; before?: number; types?: string }): Promise<FeedResponse>` using `api.get`.
  - Build the query string from params, omitting undefined values.

### Acceptance criteria

- `FeedEvent` and `FeedResponse` types are importable from `@/types/api`.
- `fetchFeed` calls `GET /api/p/{project}/feed` with correct query params.
- `WsEventType` includes `'feed.new'`.
- `pnpm exec vue-tsc --noEmit` passes.

---

## Milestone 2 — Feed Pinia store

### Description

Create a `useFeedStore` composable that manages feed state: event list, loading flag, cursor, active filters, and real-time prepend.

### Files to change

- `web/src/stores/feed.ts` (new file)
  - Use `defineStore('feed', () => { ... })` composition pattern matching existing stores.
  - State:
    - `events: ref<FeedEvent[]>([])` — the loaded event list, newest first.
    - `nextCursor: ref<number | null>(null)` — cursor for next page.
    - `loading: ref(false)` — true while fetching.
    - `activeTypes: ref<Set<string>>` — enabled event type filters (all enabled by default).
  - Actions:
    - `loadPage(project: string)` — fetch next page using `nextCursor`, append to `events`. If `activeTypes` is not the full set, pass `types` param.
    - `refresh(project: string)` — clear events, reset cursor, load first page.
    - `prepend(event: FeedEvent)` — insert at index 0 if it passes the active filter. Used by the WS listener.
    - `setFilter(type: string, enabled: boolean)` — toggle a type in `activeTypes`. Triggers a `refresh`.

### Acceptance criteria

- `useFeedStore()` is importable and provides reactive state.
- `loadPage` appends events and updates `nextCursor`.
- `prepend` adds events to the top of the list.
- Filter changes trigger a refresh.
- `pnpm exec vue-tsc --noEmit` passes.

---

## Milestone 3 — Feed route and navigation entry

### Description

Register the `/feed` route and add a sidebar nav item.

### Files to change

- `web/src/router/index.ts` — add a child route inside the `/p/:project` children array (after the `agents` route, line ~52):
  ```ts
  {
    path: 'feed',
    name: 'feed',
    component: () => import('@/views/project/ProjectFeedView.vue'),
  },
  ```

- `web/src/components/layout/AppSidebar.vue`
  - Import `Activity` icon from `lucide-vue-next` (line ~9 area). The `Activity` icon (a heartbeat-style line) is appropriate for an activity feed. Alternatively `Rss` could work — use whichever fits the existing icon style better.
  - Add entry to `navItems()` (line ~61 area):
    ```ts
    { label: 'Feed', to: `/p/${p}/feed`, icon: Activity },
    ```
    Place it after `'Agents'` and before `'Parse Errors'` since the feed is a primary navigation target.

### Acceptance criteria

- Navigating to `/p/{project}/feed` renders the `ProjectFeedView` component.
- The sidebar shows a "Feed" entry with an icon between Agents and Parse Errors.
- Clicking the nav entry navigates to the feed view.
- `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 4 — `ProjectFeedView.vue` component

### Description

Implement the main feed view with an event list, filter bar, and empty state.

### Files to change

- `web/src/views/project/ProjectFeedView.vue` (new file)
  - Script setup:
    - Use `useRoute()` to get `project` param.
    - Use `useFeedStore()` for state and actions.
    - Call `feedStore.refresh(project)` on mount.
    - Use `useWebSocket(project, 'feed.new', handler)` to prepend real-time events.
  - Template structure:
    ```html
    <div class="feed-view">
      <header class="feed-header">
        <h2>Activity Feed</h2>
        <FeedFilterBar :active-types="feedStore.activeTypes" @toggle="feedStore.setFilter" />
      </header>
      <ol class="feed-list" role="list" ref="feedListRef" @keydown="handleKeydown">
        <li v-for="event in feedStore.events" :key="event.id" class="feed-entry" ... >
          <FeedEntry :event="event" />
        </li>
      </ol>
      <div v-if="feedStore.events.length === 0 && !feedStore.loading" class="feed-empty">
        No activity yet
      </div>
      <div v-if="feedStore.loading" class="feed-loading">Loading…</div>
    </div>
    ```
  - Infinite scroll: use an `IntersectionObserver` on a sentinel element at the bottom of the list. When visible and `nextCursor` is not null, call `feedStore.loadPage(project)`.
  - Keyboard navigation (NFR-3): track `focusedIndex` ref. Arrow up/down moves focus between `<li>` elements. Enter navigates to the event's target.

### Acceptance criteria

- The feed renders a vertical, scrollable list of events newest-first.
- Filter chips toggle event categories.
- Scrolling near the bottom triggers the next page load.
- Arrow keys move focus between entries; Enter navigates.
- Empty state displays "No activity yet" when no events exist.
- `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 5 — `FeedEntry.vue` component

### Description

Implement the individual feed entry row that displays icon, timestamp, summary, actor, and handles click navigation.

### Files to change

- `web/src/components/feed/FeedEntry.vue` (new file)
  - Props: `event: FeedEvent`.
  - Template:
    ```html
    <router-link :to="navigationTarget" class="feed-entry-link">
      <span class="feed-icon"><component :is="iconForType(event.event_type)" :size="16" /></span>
      <time class="feed-time" :datetime="isoTimestamp">{{ relativeTime }}</time>
      <span class="feed-summary">{{ event.summary }}</span>
      <span class="feed-actor">{{ event.actor }}</span>
    </router-link>
    ```
  - Computed:
    - `iconForType(type)` — maps event types to lucide icons:
      - `status_transition` → `ArrowRightLeft`
      - `artifact_created` → `FilePlus`
      - `agent_started` → `Play`
      - `agent_finished` → `CheckCircle`
      - `agent_failed` → `XCircle`
      - `defect_raised` → `Bug`
      - `git_committed` → `GitCommit`
    - `relativeTime` — format `event.timestamp` (Unix seconds) as relative time ("3 min ago", "2 hours ago"). Use a simple formatter — no dependency needed for basic relative time.
    - `isoTimestamp` — ISO 8601 string for the `<time>` element's `datetime` attribute.
    - `navigationTarget` — route based on event data:
      - If `artifact_path` is set: `/p/{project}/artifacts/{artifact_path}`
      - If `run_id` is set (and no artifact_path): `/p/{project}/agents` (agent runs view, future enhancement could deep-link to run detail)
      - Fallback: current route (no navigation)
  - Styling: use scoped CSS with design tokens (`--color-surface`, `--text-sm`, `--space-*`). Add a brief highlight animation class (`feed-entry--new`) that fades a background colour over 1 second, applied when the event was prepended via WebSocket.

### Acceptance criteria

- Each entry displays an icon appropriate to its event type.
- Timestamp is shown as relative time (e.g. "5 min ago") with ISO datetime in the `<time>` element.
- Summary and actor are visible.
- Clicking navigates to the artifact detail or agents view.
- New real-time entries animate briefly on arrival.
- Semantic HTML is used (`<time>`, `<ol>`, `<li>`).
- `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 6 — `FeedFilterBar.vue` component

### Description

Implement the filter bar with toggle chips for each event category.

### Files to change

- `web/src/components/feed/FeedFilterBar.vue` (new file)
  - Props: `activeTypes: Set<string>`.
  - Emits: `toggle(type: string, enabled: boolean)`.
  - Template: a row of chip buttons, one per event category. Each chip shows an icon + label and is styled as active/inactive based on `activeTypes.has(type)`.
  - Event categories (matching FR-1):
    - `status_transition` — "Transitions"
    - `artifact_created` — "Created"
    - `agent_started` — "Agent Start"
    - `agent_finished` — "Agent Done"
    - `agent_failed` — "Agent Failed"
    - `defect_raised` — "Defects"
    - `git_committed` — "Commits"
  - Styling: chips use `--color-surface` background when inactive, `--color-primary` tint when active. Horizontal scroll on narrow viewports.

### Acceptance criteria

- All 7 event categories are represented as toggle chips.
- Clicking a chip emits `toggle` with the type and new enabled state.
- Active chips are visually distinct from inactive chips.
- The filter bar is usable on viewports ≥ 768 px (NFR-4).
- `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 7 — Responsive styling and accessibility

### Description

Polish the feed view for tablet+ viewports and ensure accessibility requirements are met.

### Files to change

- `web/src/views/project/ProjectFeedView.vue` — add responsive styles:
  - Full-width layout on viewports < 1024 px.
  - Max-width constraint and centered layout on wider viewports.
  - Appropriate padding using `--space-*` tokens.

- `web/src/components/feed/FeedEntry.vue` — ensure:
  - ARIA: `role="listitem"` on `<li>`, `aria-label` on icon-only elements.
  - `tabindex="0"` on each entry for keyboard focus.
  - Focus ring uses `outline` with sufficient contrast.

- `web/src/components/feed/FeedFilterBar.vue` — ensure:
  - `aria-pressed` attribute on chip buttons.
  - `role="toolbar"` on the container with `aria-label="Filter events"`.

### Acceptance criteria

- Feed is usable on viewports ≥ 768 px wide (NFR-4).
- All feed entries are keyboard-navigable with visible focus indicators.
- Screen readers announce event type, summary, and relative time.
- `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.
