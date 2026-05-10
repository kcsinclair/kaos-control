<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { computed } from 'vue'
import type { Release, ReleaseDetail } from '@/types/release'
import type { PeriodMode, FixedPeriod } from '@/stores/roadmapSettings'

type Granularity = 'week' | 'month' | 'quarter' | 'half-year' | 'year'

const props = withDefaults(defineProps<{
  releases: Release[]
  granularity: Granularity
  project: string
  releaseDetails?: Map<number, ReleaseDetail>
  periodMode?: PeriodMode
  fixedPeriod?: FixedPeriod
}>(), {
  periodMode: 'autoscale',
  fixedPeriod: 'month',
})

const emit = defineEmits<{
  clickRelease: [id: number]
  create: []
}>()

const TODAY = new Date()
TODAY.setHours(0, 0, 0, 0)

// ── Time axis helpers ───────────────────────────────────────────────────────

interface Column {
  label: string
  start: Date
  end: Date
}

function startOfWeek(d: Date): Date {
  const r = new Date(d)
  const day = r.getDay()
  r.setDate(r.getDate() - day)
  r.setHours(0, 0, 0, 0)
  return r
}

function addDays(d: Date, n: number): Date {
  const r = new Date(d)
  r.setDate(r.getDate() + n)
  return r
}

function startOfMonth(d: Date): Date {
  return new Date(d.getFullYear(), d.getMonth(), 1)
}

function startOfQuarter(d: Date): Date {
  const q = Math.floor(d.getMonth() / 3) * 3
  return new Date(d.getFullYear(), q, 1)
}

function startOfHalfYear(d: Date): Date {
  const h = d.getMonth() < 6 ? 0 : 6
  return new Date(d.getFullYear(), h, 1)
}

function startOfYear(d: Date): Date {
  return new Date(d.getFullYear(), 0, 1)
}

/** Snap a date to the start of its granularity period. */
function startOfGranularity(d: Date, gran: Granularity): Date {
  switch (gran) {
    case 'week':      return startOfWeek(d)
    case 'month':     return startOfMonth(d)
    case 'quarter':   return startOfQuarter(d)
    case 'half-year': return startOfHalfYear(d)
    case 'year':      return startOfYear(d)
  }
}

/** Return the last day of the granularity period containing d. */
function endOfGranularity(d: Date, gran: Granularity): Date {
  switch (gran) {
    case 'week':      return addDays(startOfWeek(d), 6)
    case 'month':     return new Date(d.getFullYear(), d.getMonth() + 1, 0)
    case 'quarter': {
      const q = Math.floor(d.getMonth() / 3) * 3
      return new Date(d.getFullYear(), q + 3, 0)
    }
    case 'half-year': {
      const h = d.getMonth() < 6 ? 0 : 6
      return new Date(d.getFullYear(), h + 6, 0)
    }
    case 'year':      return new Date(d.getFullYear(), 11, 31)
  }
}

/** Return the first day of the fixed calendar period containing d. */
function startOfPeriod(period: FixedPeriod, d: Date): Date {
  switch (period) {
    case 'month':     return startOfMonth(d)
    case 'quarter':   return startOfQuarter(d)
    case 'half-year': return startOfHalfYear(d)
    case 'year':      return startOfYear(d)
  }
}

/** Return the last day of the fixed calendar period containing d. */
function endOfPeriod(period: FixedPeriod, d: Date): Date {
  switch (period) {
    case 'month':     return new Date(d.getFullYear(), d.getMonth() + 1, 0)
    case 'quarter': {
      const q = Math.floor(d.getMonth() / 3) * 3
      return new Date(d.getFullYear(), q + 3, 0)
    }
    case 'half-year': {
      const h = d.getMonth() < 6 ? 0 : 6
      return new Date(d.getFullYear(), h + 6, 0)
    }
    case 'year':      return new Date(d.getFullYear(), 11, 31)
  }
}

/** Given a list of releases, compute a combined time range to display */
const timeRange = computed<{ start: Date; end: Date }>(() => {
  const gran = props.granularity

  if (props.periodMode === 'fixed') {
    // Fixed-period: anchor to the current calendar period containing today.
    return {
      start: startOfPeriod(props.fixedPeriod, TODAY),
      end:   endOfPeriod(props.fixedPeriod, TODAY),
    }
  }

  // Autoscale: cover exactly the release span snapped to granularity boundaries.
  const scheduled = props.releases.filter((r) => r.start_date && r.end_date)

  if (scheduled.length === 0) {
    // No releases: show a single column containing today.
    return {
      start: startOfGranularity(TODAY, gran),
      end:   endOfGranularity(TODAY, gran),
    }
  }

  const starts = scheduled.map((r) => new Date(r.start_date!))
  const ends   = scheduled.map((r) => new Date(r.end_date!))
  const minStart = starts.reduce((a, b) => (a < b ? a : b))
  const maxEnd   = ends.reduce((a, b) => (a > b ? a : b))

  return {
    start: startOfGranularity(minStart, gran),
    end:   endOfGranularity(maxEnd, gran),
  }
})

const columns = computed<Column[]>(() => {
  const { start, end } = timeRange.value
  const cols: Column[] = []
  let cur = new Date(start)

  while (cur < end) {
    let colStart: Date
    let colEnd: Date
    let label: string

    switch (props.granularity) {
      case 'week': {
        colStart = new Date(cur)
        colEnd = addDays(colStart, 6)
        label = `${colStart.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })}`
        cur = addDays(colStart, 7)
        break
      }
      case 'month': {
        colStart = new Date(cur.getFullYear(), cur.getMonth(), 1)
        colEnd = new Date(cur.getFullYear(), cur.getMonth() + 1, 0)
        label = colStart.toLocaleDateString(undefined, { month: 'short', year: 'numeric' })
        cur = new Date(cur.getFullYear(), cur.getMonth() + 1, 1)
        break
      }
      case 'quarter': {
        colStart = new Date(cur)
        const qEnd = new Date(cur.getFullYear(), cur.getMonth() + 3, 0)
        colEnd = qEnd
        const qNum = Math.floor(cur.getMonth() / 3) + 1
        label = `Q${qNum} ${cur.getFullYear()}`
        cur = new Date(cur.getFullYear(), cur.getMonth() + 3, 1)
        break
      }
      case 'half-year': {
        colStart = new Date(cur)
        const hEnd = new Date(cur.getFullYear(), cur.getMonth() + 6, 0)
        colEnd = hEnd
        const hNum = cur.getMonth() < 6 ? 1 : 2
        label = `H${hNum} ${cur.getFullYear()}`
        cur = new Date(cur.getFullYear(), cur.getMonth() + 6, 1)
        break
      }
      case 'year': {
        colStart = new Date(cur.getFullYear(), 0, 1)
        colEnd = new Date(cur.getFullYear(), 11, 31)
        label = String(cur.getFullYear())
        cur = new Date(cur.getFullYear() + 1, 0, 1)
        break
      }
      default:
        colStart = new Date(cur)
        colEnd = addDays(cur, 6)
        label = cur.toLocaleDateString()
        cur = addDays(cur, 7)
    }

    cols.push({ label, start: colStart, end: colEnd })
    if (cols.length > 200) break // safety cap
  }
  return cols
})

const totalMs = computed(() =>
  timeRange.value.end.getTime() - timeRange.value.start.getTime()
)

function pct(d: Date): number {
  const ms = d.getTime() - timeRange.value.start.getTime()
  return Math.max(0, Math.min(100, (ms / totalMs.value) * 100))
}

function colWidthPct(): number {
  return columns.value.length > 0 ? 100 / columns.value.length : 0
}

// ── Bar positioning ─────────────────────────────────────────────────────────

interface BarInfo {
  release: Release
  left: number        // %
  width: number       // %
  clippedLeft: boolean
  clippedRight: boolean
}

const scheduledBars = computed<BarInfo[]>(() => {
  const { start: rangeStart, end: rangeEnd } = timeRange.value
  return props.releases
    .filter((r) => r.start_date && r.end_date)
    .map((r) => {
      const s = new Date(r.start_date!)
      const e = new Date(r.end_date!)
      // Skip bars entirely outside the visible window.
      if (addDays(e, 1) <= rangeStart || s >= rangeEnd) return null
      const left = pct(s)
      const right = pct(addDays(e, 1))
      return {
        release: r,
        left,
        width: Math.max(right - left, 1),
        clippedLeft:  s < rangeStart,
        clippedRight: addDays(e, 1) > rangeEnd,
      }
    })
    .filter((b): b is BarInfo => b !== null)
    .sort((a, b) => a.left - b.left)
})

const unscheduled = computed(() =>
  props.releases.filter((r) => !r.start_date || !r.end_date)
)

const hasUnscheduled = computed(() => unscheduled.value.length > 0)

const unscheduledSorted = computed(() =>
  [...unscheduled.value].sort((a, b) => a.name.localeCompare(b.name))
)

const todayPct = computed(() => pct(TODAY))

function statusColor(status: string): string {
  switch (status) {
    case 'active':  return 'var(--color-accent, #3b82f6)'
    case 'shipped': return '#16a34a'
    default:        return '#94a3b8'
  }
}

function summaryBadge(release: Release): string {
  const detail = props.releaseDetails?.get(release.id)
  if (!detail) return ''
  const parts = []
  if (detail.idea_count > 0) parts.push(`${detail.idea_count} idea${detail.idea_count !== 1 ? 's' : ''}`)
  if (detail.defect_count > 0) parts.push(`${detail.defect_count} defect${detail.defect_count !== 1 ? 's' : ''}`)
  return parts.join(' · ')
}
</script>

<template>
  <div class="gantt-wrap">
    <!-- Empty state -->
    <div v-if="releases.length === 0" class="empty-state">
      <p class="empty-msg">No releases yet.</p>
      <button class="btn-primary" @click="emit('create')">Create your first release</button>
    </div>

    <template v-else>
      <!-- Time axis header -->
      <div class="gantt-header">
        <div class="header-date-area">
          <div
            v-for="col in columns"
            :key="col.label"
            class="col-header"
            :style="{ width: colWidthPct() + '%' }"
          >{{ col.label }}</div>
        </div>
        <div v-if="hasUnscheduled" class="col-header col-header--unscheduled">Unscheduled</div>
      </div>

      <!-- Chart body -->
      <div class="gantt-body">
        <!-- Scheduled rows -->
        <div
          v-for="bar in scheduledBars"
          :key="bar.release.id"
          class="gantt-row"
        >
          <div class="row-track">
            <!-- Column grid lines -->
            <div
              v-for="col in columns"
              :key="col.label"
              class="col-grid"
              :style="{ width: colWidthPct() + '%' }"
            />
            <!-- Today marker -->
            <div
              v-if="todayPct >= 0 && todayPct <= 100"
              class="today-marker"
              :style="{ left: todayPct + '%' }"
              title="Today"
            />
            <!-- Release bar -->
            <button
              class="release-bar"
              :class="{
                'release-bar--clipped-left':  bar.clippedLeft,
                'release-bar--clipped-right': bar.clippedRight,
              }"
              :style="{
                left: bar.left + '%',
                width: bar.width + '%',
                background: statusColor(bar.release.status),
              }"
              :title="bar.release.name"
              @click="emit('clickRelease', bar.release.id)"
            >
              <span v-if="bar.clippedLeft" class="clip-arrow clip-arrow--left" aria-hidden="true">&#8249;</span>
              <span class="bar-name">{{ bar.release.name }}</span>
              <span v-if="summaryBadge(bar.release)" class="bar-badge">{{ summaryBadge(bar.release) }}</span>
              <span v-if="bar.clippedRight" class="clip-arrow clip-arrow--right" aria-hidden="true">&#8250;</span>
            </button>
          </div>
          <!-- Placeholder cell to maintain grid alignment -->
          <div v-if="hasUnscheduled" class="unscheduled-cell"></div>
        </div>

        <!-- Unscheduled release rows — bar sits inside the sticky column -->
        <div
          v-for="r in unscheduledSorted"
          :key="r.id"
          class="gantt-row"
        >
          <div class="row-track">
            <!-- Column grid lines (keeps visual alignment with scheduled rows) -->
            <div
              v-for="col in columns"
              :key="col.label"
              class="col-grid"
              :style="{ width: colWidthPct() + '%' }"
            />
          </div>
          <div class="unscheduled-cell unscheduled-cell--bar">
            <button
              class="release-bar release-bar--unscheduled"
              :style="{ background: statusColor(r.status) }"
              :title="r.name"
              @click="emit('clickRelease', r.id)"
            >
              <span class="bar-name">{{ r.name }}</span>
              <span v-if="summaryBadge(r)" class="bar-badge">{{ summaryBadge(r) }}</span>
            </button>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.gantt-wrap {
  flex: 1;
  overflow: auto;
  padding: var(--space-3) var(--space-4);
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.empty-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--space-3);
}
.empty-msg {
  font-size: var(--text-base);
  color: var(--color-text-muted);
  margin: 0;
}
.btn-primary {
  padding: var(--space-2) var(--space-4);
  background: var(--color-accent);
  color: #fff;
  border: none;
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
}
.btn-primary:hover { opacity: 0.88; }

/* Header */
.gantt-header {
  display: flex;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-bottom: none;
  border-radius: var(--radius-sm) var(--radius-sm) 0 0;
  flex-shrink: 0;
}
.header-date-area {
  display: flex;
  flex: 1;
  overflow: hidden;
}
.col-header {
  flex-shrink: 0;
  padding: var(--space-1) var(--space-2);
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--color-text-muted);
  border-right: 1px solid var(--color-border);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  box-sizing: border-box;
}
.col-header:last-child { border-right: none; }

.col-header--unscheduled {
  flex: 0 0 120px;
  position: sticky;
  right: 0;
  border-left: 2px dashed var(--color-border);
  border-right: none;
  background: var(--color-surface);
  z-index: 10;
}

/* Body */
.gantt-body {
  display: flex;
  flex-direction: column;
  border: 1px solid var(--color-border);
  border-radius: 0 0 var(--radius-sm) var(--radius-sm);
  flex-shrink: 0;
}

.gantt-row {
  display: flex;
  align-items: stretch;
  border-bottom: 1px solid var(--color-border);
  min-height: 40px;
}
.gantt-row:last-child { border-bottom: none; }

.row-track {
  flex: 1;
  position: relative;
  display: flex;
  align-items: center;
  overflow: hidden;
}

/* Sticky placeholder cell in scheduled rows (maintains grid when unscheduled column is visible) */
.unscheduled-cell {
  flex: 0 0 120px;
  position: sticky;
  right: 0;
  background: var(--color-bg);
  border-left: 2px dashed var(--color-border);
  z-index: 4;
}

/* Unscheduled bar cell — contains the release bar button */
.unscheduled-cell--bar {
  display: flex;
  align-items: center;
  padding: var(--space-1);
}

/* Release bar variant for unscheduled releases — hatched overlay, same statusColor base */
.release-bar--unscheduled {
  position: relative;
  top: auto;
  left: auto;
  transform: none;
  width: 100%;
  background-image: repeating-linear-gradient(
    45deg,
    transparent,
    transparent 4px,
    rgba(255, 255, 255, 0.18) 4px,
    rgba(255, 255, 255, 0.18) 8px
  ) !important;
}

.col-grid {
  flex-shrink: 0;
  height: 100%;
  border-right: 1px dashed var(--color-border);
  box-sizing: border-box;
  pointer-events: none;
}
.col-grid:last-child { border-right: none; }

/* Today marker */
.today-marker {
  position: absolute;
  top: 0;
  bottom: 0;
  width: 2px;
  background: #ef4444;
  z-index: 10;
  pointer-events: none;
}
.today-marker::before {
  content: '';
  position: absolute;
  top: 2px;
  left: -3px;
  width: 0;
  height: 0;
  border-left: 4px solid transparent;
  border-right: 4px solid transparent;
  border-top: 6px solid #ef4444;
}

/* Release bar */
.release-bar {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  height: 28px;
  border: none;
  border-radius: var(--radius-sm);
  cursor: pointer;
  display: flex;
  align-items: center;
  padding: 0 var(--space-2);
  gap: var(--space-1);
  overflow: hidden;
  z-index: 5;
  transition: opacity 0.12s, box-shadow 0.12s;
  font-family: inherit;
}
.release-bar:hover {
  opacity: 0.85;
  box-shadow: 0 0 0 2px rgba(255,255,255,0.4);
}
.bar-name {
  font-size: 12px;
  font-weight: 600;
  color: #fff;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.bar-badge {
  font-size: 10px;
  color: rgba(255,255,255,0.85);
  white-space: nowrap;
  flex-shrink: 0;
}

/* Clip indicators shown when a bar extends beyond the fixed-period window */
.clip-arrow {
  font-size: 16px;
  font-weight: 700;
  color: rgba(255, 255, 255, 0.9);
  line-height: 1;
  flex-shrink: 0;
}
.clip-arrow--left  { margin-right: var(--space-1); }
.clip-arrow--right { margin-left: auto; }

/* Rounded corners are removed on the clipped edge */
.release-bar--clipped-left  { border-top-left-radius: 0; border-bottom-left-radius: 0; }
.release-bar--clipped-right { border-top-right-radius: 0; border-bottom-right-radius: 0; }
</style>
