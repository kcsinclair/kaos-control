<script setup lang="ts">
import { computed } from 'vue'
import type { Release, ReleaseDetail } from '@/types/release'

type Granularity = 'week' | 'month' | 'quarter' | 'half-year' | 'year'

const props = defineProps<{
  releases: Release[]
  granularity: Granularity
  project: string
  releaseDetails?: Map<number, ReleaseDetail>
}>()

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

/** Given a list of releases, compute a combined time range to display */
const timeRange = computed<{ start: Date; end: Date }>(() => {
  const scheduled = props.releases.filter((r) => r.start_date && r.end_date)

  let earliest = new Date(TODAY)
  let latest = addDays(TODAY, 90)

  if (scheduled.length > 0) {
    const starts = scheduled.map((r) => new Date(r.start_date!))
    const ends = scheduled.map((r) => new Date(r.end_date!))
    const minStart = starts.reduce((a, b) => (a < b ? a : b))
    const maxEnd = ends.reduce((a, b) => (a > b ? a : b))

    // Pad by one column unit on each side
    earliest = minStart < TODAY ? minStart : TODAY
    latest = maxEnd > latest ? maxEnd : latest
  }

  // Snap to column boundaries
  let start: Date
  let end: Date

  switch (props.granularity) {
    case 'week':
      start = startOfWeek(addDays(earliest, -7))
      end = addDays(startOfWeek(addDays(latest, 7)), 6)
      break
    case 'month':
      start = startOfMonth(new Date(earliest.getFullYear(), earliest.getMonth() - 1, 1))
      end = new Date(latest.getFullYear(), latest.getMonth() + 2, 0)
      break
    case 'quarter':
      start = startOfQuarter(new Date(earliest.getFullYear(), earliest.getMonth() - 3, 1))
      end = new Date(startOfQuarter(new Date(latest.getFullYear(), latest.getMonth() + 3, 1)))
      end = new Date(end.getFullYear(), end.getMonth() + 3, 0)
      break
    case 'half-year':
      start = startOfHalfYear(new Date(earliest.getFullYear(), earliest.getMonth() - 6, 1))
      end = new Date(startOfHalfYear(new Date(latest.getFullYear(), latest.getMonth() + 6, 1)))
      end = new Date(end.getFullYear(), end.getMonth() + 6, 0)
      break
    case 'year':
      start = startOfYear(new Date(earliest.getFullYear() - 1, 0, 1))
      end = new Date(latest.getFullYear() + 1, 11, 31)
      break
    default:
      start = earliest
      end = latest
  }

  return { start, end }
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
  left: number   // %
  width: number  // %
}

const scheduledBars = computed<BarInfo[]>(() => {
  return props.releases
    .filter((r) => r.start_date && r.end_date)
    .map((r) => {
      const s = new Date(r.start_date!)
      const e = new Date(r.end_date!)
      const left = pct(s)
      const right = pct(addDays(e, 1))
      return { release: r, left, width: Math.max(right - left, 1) }
    })
    .sort((a, b) => a.left - b.left)
})

const unscheduled = computed(() =>
  props.releases.filter((r) => !r.start_date || !r.end_date)
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
        <div
          v-for="col in columns"
          :key="col.label"
          class="col-header"
          :style="{ width: colWidthPct() + '%' }"
        >{{ col.label }}</div>
      </div>

      <!-- Chart body -->
      <div class="gantt-body">
        <!-- Scheduled rows -->
        <div
          v-for="bar in scheduledBars"
          :key="bar.release.id"
          class="gantt-row"
        >
          <div class="row-label">{{ bar.release.name }}</div>
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
              :style="{
                left: bar.left + '%',
                width: bar.width + '%',
                background: statusColor(bar.release.status),
              }"
              :title="bar.release.name"
              @click="emit('clickRelease', bar.release.id)"
            >
              <span class="bar-name">{{ bar.release.name }}</span>
              <span v-if="summaryBadge(bar.release)" class="bar-badge">{{ summaryBadge(bar.release) }}</span>
            </button>
          </div>
        </div>

        <!-- Unscheduled section -->
        <template v-if="unscheduled.length > 0">
          <div class="unscheduled-heading">Unscheduled</div>
          <div class="unscheduled-cards">
            <button
              v-for="r in unscheduled"
              :key="r.id"
              class="unscheduled-card"
              :style="{ borderLeftColor: statusColor(r.status) }"
              @click="emit('clickRelease', r.id)"
            >
              <span class="card-name">{{ r.name }}</span>
              <span class="card-status" :class="`status--${r.status}`">{{ r.status }}</span>
            </button>
          </div>
        </template>
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
  overflow: hidden;
  flex-shrink: 0;
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

/* Body */
.gantt-body {
  display: flex;
  flex-direction: column;
  border: 1px solid var(--color-border);
  border-radius: 0 0 var(--radius-sm) var(--radius-sm);
  overflow: hidden;
  flex-shrink: 0;
}

.gantt-row {
  display: flex;
  align-items: stretch;
  border-bottom: 1px solid var(--color-border);
  min-height: 40px;
}
.gantt-row:last-child { border-bottom: none; }

.row-label {
  width: 140px;
  flex-shrink: 0;
  padding: var(--space-2) var(--space-3);
  font-size: var(--text-sm);
  font-weight: 500;
  color: var(--color-text);
  border-right: 1px solid var(--color-border);
  display: flex;
  align-items: center;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.row-track {
  flex: 1;
  position: relative;
  display: flex;
  align-items: center;
  overflow: hidden;
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

/* Unscheduled section */
.unscheduled-heading {
  padding: var(--space-2) var(--space-3);
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
  background: var(--color-surface);
  border-top: 1px solid var(--color-border);
}
.unscheduled-cards {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  padding: var(--space-3);
  background: var(--color-surface);
}
.unscheduled-card {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: var(--space-2) var(--space-3);
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-left-width: 3px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-family: inherit;
  text-align: left;
  transition: border-color 0.12s, background 0.12s;
  min-width: 140px;
}
.unscheduled-card:hover { background: var(--color-surface); }
.card-name {
  font-size: var(--text-sm);
  font-weight: 500;
  color: var(--color-text);
}
.card-status {
  font-size: 10px;
  font-weight: 500;
  color: var(--color-text-muted);
}
.status--active  { color: var(--color-accent); }
.status--shipped { color: #16a34a; }
</style>
