<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { computed } from 'vue'
import type { GraphNode, GraphEdge } from '@/types/api'
import { useGraphTheme } from './graphConstants'
import { edgeStyle, scaleBucket, signalGlyphs, SCALE_BUCKET_LABELS, SCALE_COLOURS } from './archMapStyle'
import type { ScaleBucket, SignalGlyph } from './archMapStyle'

const props = defineProps<{
  nodes: GraphNode[]
  edges: GraphEdge[]
}>()

const { palette, isDark } = useGraphTheme()

function humanizeKind(kind: string): string {
  return kind.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase())
}

// Relationship kinds present in the current map only (FR-6).
const relationshipKinds = computed(() => {
  const seen = new Map<string, { kind: string; label: string; lineStyle: string; arrow: boolean }>()
  for (const e of props.edges) {
    if (seen.has(e.kind)) continue
    const style = edgeStyle(e.kind)
    seen.set(e.kind, {
      kind: e.kind,
      label: style.label || humanizeKind(e.kind),
      lineStyle: style.lineStyle,
      arrow: style.arrow,
    })
  }
  return [...seen.values()]
})

// Scale buckets present in the current node set only (FR-6).
const scaleBucketsPresent = computed(() => {
  const present = new Set<ScaleBucket>()
  for (const n of props.nodes) present.add(scaleBucket(n.labels))
  return (['low', 'medium', 'high', 'neutral'] as ScaleBucket[])
    .filter((b) => present.has(b))
    .map((bucket) => ({ bucket, label: SCALE_BUCKET_LABELS[bucket], color: SCALE_COLOURS[bucket] }))
})

// Decision-signal glyphs present in the current node set only (FR-6).
const glyphsPresent = computed(() => {
  const seen = new Map<string, SignalGlyph>()
  for (const n of props.nodes) {
    for (const g of signalGlyphs(n.labels)) seen.set(g.key, g)
  }
  return [...seen.values()]
})

const legendStyle = computed(() => ({
  background: isDark.value ? 'rgba(15, 23, 42, 0.85)' : 'rgba(255, 255, 255, 0.92)',
  color: palette.value.labelColor,
}))

const titleStyle = computed(() => ({
  color: isDark.value ? 'rgba(241, 245, 249, 0.5)' : 'rgba(15, 23, 42, 0.45)',
}))

const itemLabelStyle = computed(() => ({
  color: isDark.value ? 'rgba(241, 245, 249, 0.8)' : 'rgba(15, 23, 42, 0.75)',
}))
</script>

<template>
  <div class="arch-legend" :style="legendStyle">
    <div v-if="relationshipKinds.length" class="legend-section">
      <div class="legend-title" :style="titleStyle">Relationships</div>
      <div v-for="item in relationshipKinds" :key="item.kind" class="legend-item">
        <span class="legend-line" :class="{ dashed: item.lineStyle === 'dashed' }" />
        <span class="legend-label" :style="itemLabelStyle">{{ item.label }}{{ item.arrow ? ' →' : '' }}</span>
      </div>
    </div>
    <div v-if="scaleBucketsPresent.length" class="legend-section">
      <div class="legend-title" :style="titleStyle">Scale</div>
      <div v-for="item in scaleBucketsPresent" :key="item.bucket" class="legend-item">
        <span class="legend-dot" :style="{ background: item.color }" />
        <span class="legend-label" :style="itemLabelStyle">{{ item.label }}</span>
      </div>
    </div>
    <div v-if="glyphsPresent.length" class="legend-section">
      <div class="legend-title" :style="titleStyle">Signals</div>
      <div v-for="item in glyphsPresent" :key="item.key" class="legend-item">
        <span class="legend-glyph">{{ item.icon }}</span>
        <span class="legend-label" :style="itemLabelStyle">{{ item.label }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.arch-legend {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  border: 1px solid rgba(148, 163, 184, 0.15);
  border-radius: var(--radius-md);
  padding: var(--space-3);
  backdrop-filter: blur(4px);
}
.legend-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}
.legend-title {
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  margin-bottom: var(--space-1);
}
.legend-item {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}
.legend-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
}
.legend-line {
  width: 16px;
  height: 2px;
  border-radius: 1px;
  background: currentColor;
  color: #94a3b8;
  flex-shrink: 0;
}
.legend-line.dashed {
  background: repeating-linear-gradient(to right, currentColor 0 4px, transparent 4px 7px);
}
.legend-glyph {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 24px;
  height: 14px;
  padding: 0 3px;
  border-radius: 3px;
  border: 1px solid #94a3b8;
  font-size: 8px;
  font-weight: 700;
  flex-shrink: 0;
}
.legend-label {
  font-size: 11px;
}
</style>
