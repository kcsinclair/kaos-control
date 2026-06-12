<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { AgentUsageFilter } from '@/types/api'

const props = defineProps<{
  agents: string[]
  filter: AgentUsageFilter
}>()

const emit = defineEmits<{
  update: [patch: Partial<AgentUsageFilter>]
}>()

type Preset = 'last24h' | 'last7d' | 'last30d' | 'last90d' | 'custom'

const STATUS_OPTIONS = ['done', 'failed', 'killed', 'killed-timeout'] as const
const BUCKET_OPTIONS = [
  { value: 'hour' as const, label: 'Hour' },
  { value: 'day' as const, label: 'Day' },
  { value: 'week' as const, label: 'Week' },
]

const agentPopoverOpen = ref(false)

function toRFC3339(d: Date): string {
  return d.toISOString()
}

function detectPreset(filter: AgentUsageFilter): Preset {
  if (!filter.from && !filter.to) return 'last30d'
  const from = filter.from ? new Date(filter.from).getTime() : 0
  const to = filter.to ? new Date(filter.to).getTime() : Date.now()
  const diffMs = to - from
  const h = 1000 * 60 * 60
  if (Math.abs(diffMs - 24 * h) < 60000) return 'last24h'
  if (Math.abs(diffMs - 7 * 24 * h) < 60000) return 'last7d'
  if (Math.abs(diffMs - 30 * 24 * h) < 60000) return 'last30d'
  if (Math.abs(diffMs - 90 * 24 * h) < 60000) return 'last90d'
  return 'custom'
}

const activePreset = computed(() => detectPreset(props.filter))

function applyPreset(preset: Preset) {
  if (preset === 'custom') return
  const now = new Date()
  const h = 1000 * 60 * 60
  const offsets: Record<Exclude<Preset, 'custom'>, number> = {
    last24h: 24 * h,
    last7d: 7 * 24 * h,
    last30d: 30 * 24 * h,
    last90d: 90 * 24 * h,
  }
  const from = new Date(now.getTime() - offsets[preset as Exclude<Preset, 'custom'>])
  emit('update', { from: toRFC3339(from), to: toRFC3339(now) })
}

function toDatetimeLocal(iso: string | undefined): string {
  if (!iso) return ''
  const d = new Date(iso)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function fromDatetimeLocal(value: string): string {
  return new Date(value).toISOString()
}

function onCustomFrom(e: Event) {
  const v = (e.target as HTMLInputElement).value
  if (v) emit('update', { from: fromDatetimeLocal(v) })
}

function onCustomTo(e: Event) {
  const v = (e.target as HTMLInputElement).value
  if (v) emit('update', { to: fromDatetimeLocal(v) })
}

function toggleAgent(name: string) {
  const current = props.filter.agent ?? []
  const next = current.includes(name)
    ? current.filter((a) => a !== name)
    : [...current, name]
  emit('update', { agent: next })
}

function toggleStatus(s: string) {
  const current = props.filter.status ?? []
  const next = current.includes(s)
    ? current.filter((x) => x !== s)
    : [...current, s]
  emit('update', { status: next })
}

function agentLabel(filter: AgentUsageFilter): string {
  const sel = filter.agent ?? []
  if (sel.length === 0) return 'All agents'
  if (sel.length === 1) return sel[0]
  return `${sel.length} agents`
}

const PRESETS: { value: Preset; label: string }[] = [
  { value: 'last24h', label: 'Last 24h' },
  { value: 'last7d',  label: 'Last 7d' },
  { value: 'last30d', label: 'Last 30d' },
  { value: 'last90d', label: 'Last 90d' },
  { value: 'custom',  label: 'Custom' },
]
</script>

<template>
  <div class="filter-bar" role="group" aria-label="Report filters">
    <!-- Date range presets -->
    <div class="filter-group" role="group" aria-label="Date range">
      <button
        v-for="p in PRESETS"
        :key="p.value"
        class="seg-btn"
        :class="{ 'seg-btn--active': activePreset === p.value }"
        :aria-pressed="activePreset === p.value"
        @click="applyPreset(p.value)"
      >{{ p.label }}</button>
    </div>

    <!-- Custom date range -->
    <template v-if="activePreset === 'custom'">
      <div class="filter-group filter-group--date">
        <label class="filter-label" for="fb-from">From</label>
        <input
          id="fb-from"
          type="datetime-local"
          class="date-input"
          :value="toDatetimeLocal(filter.from)"
          @change="onCustomFrom"
        />
        <label class="filter-label" for="fb-to">To</label>
        <input
          id="fb-to"
          type="datetime-local"
          class="date-input"
          :value="toDatetimeLocal(filter.to)"
          @change="onCustomTo"
        />
      </div>
    </template>

    <!-- Agent multi-select -->
    <div class="filter-group filter-group--popover" v-if="agents.length > 0">
      <button
        class="seg-btn popover-trigger"
        :aria-expanded="agentPopoverOpen"
        aria-haspopup="listbox"
        @click="agentPopoverOpen = !agentPopoverOpen"
      >{{ agentLabel(filter) }}</button>
      <div v-if="agentPopoverOpen" class="popover" role="listbox" aria-multiselectable="true" aria-label="Select agents">
        <label
          v-for="a in agents"
          :key="a"
          class="popover-item"
          :role="'option'"
          :aria-selected="(filter.agent ?? []).includes(a)"
        >
          <input
            type="checkbox"
            :checked="(filter.agent ?? []).includes(a)"
            @change="toggleAgent(a)"
          />
          {{ a }}
        </label>
        <button class="popover-close" @click="agentPopoverOpen = false">Done</button>
      </div>
    </div>

    <!-- Status multi-select -->
    <div class="filter-group" role="group" aria-label="Status filter">
      <span class="filter-label">Status</span>
      <button
        v-for="s in STATUS_OPTIONS"
        :key="s"
        class="seg-btn"
        :class="{ 'seg-btn--active': (filter.status ?? []).includes(s) }"
        :aria-pressed="(filter.status ?? []).includes(s)"
        @click="toggleStatus(s)"
      >{{ s }}</button>
    </div>

    <!-- Bucket -->
    <div class="filter-group" role="group" aria-label="Bucket size">
      <span class="filter-label">Bucket</span>
      <button
        v-for="b in BUCKET_OPTIONS"
        :key="b.value"
        class="seg-btn"
        :class="{ 'seg-btn--active': filter.bucket === b.value }"
        :aria-pressed="filter.bucket === b.value"
        @click="emit('update', { bucket: b.value })"
      >{{ b.label }}</button>
    </div>
  </div>
</template>

<style scoped>
.filter-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  overflow-x: auto;
}
.filter-group {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
}
.filter-group--date {
  gap: var(--space-2);
}
.filter-label {
  font-size: var(--text-xs);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--color-text-muted);
  margin-right: var(--space-1);
  white-space: nowrap;
}
.seg-btn {
  background: none;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  cursor: pointer;
  font-size: var(--text-xs);
  padding: var(--space-1) var(--space-2);
  color: var(--color-text-muted);
  transition: background 0.12s, color 0.12s, border-color 0.12s;
  white-space: nowrap;
}
.seg-btn:hover, .seg-btn:focus-visible {
  background: var(--color-sidebar-hover);
  color: var(--color-text);
  outline: 2px solid var(--color-primary, #6366f1);
  outline-offset: 1px;
}
.seg-btn--active {
  background: var(--color-primary, #6366f1);
  border-color: var(--color-primary, #6366f1);
  color: #fff;
  font-weight: 600;
}
.date-input {
  font-size: var(--text-xs);
  padding: var(--space-1) var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  color: var(--color-text);
  cursor: pointer;
}
.filter-group--popover {
  position: relative;
}
.popover {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  z-index: 50;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: var(--space-2);
  min-width: 180px;
  box-shadow: var(--shadow-lg);
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.popover-item {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--text-sm);
  color: var(--color-text);
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-md);
  cursor: pointer;
  user-select: none;
}
.popover-item:hover {
  background: var(--color-sidebar-hover);
}
.popover-close {
  margin-top: var(--space-1);
  background: none;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  cursor: pointer;
  font-size: var(--text-xs);
  padding: var(--space-1) var(--space-2);
  color: var(--color-text-muted);
  align-self: flex-end;
}
.popover-close:hover {
  background: var(--color-sidebar-hover);
  color: var(--color-text);
}
</style>
