<script setup lang="ts">
import { computed, ref } from 'vue'
import type { ArtifactDetail, GraphEdge } from '@/types/api'
import { formatShortDate, formatFullDateTime } from '@/composables/useFormatDate'
import ArtifactRunHistory from './ArtifactRunHistory.vue'
import RunDetailModal from '@/components/agent/RunDetailModal.vue'
import StatusDropdown from './StatusDropdown.vue'

const props = defineProps<{
  artifact: ArtifactDetail
  project?: string
  targetPath?: string
  edges?: GraphEdge[]
}>()

const emit = defineEmits<{
  transitioned: [newStatus: string]
  error: [message: string]
}>()

const inbound = computed(() =>
  (props.edges ?? []).filter((e) => e.target === props.artifact.path)
)
const outbound = computed(() =>
  (props.edges ?? []).filter((e) => e.source === props.artifact.path)
)

const selectedRunId = ref<string | null>(null)

function fmt(v: string | undefined): string {
  if (!v) return '—'
  return v
}

</script>

<template>
  <aside class="fm-panel">
    <h3 class="fm-title">Details</h3>
    <dl class="fm-list">
      <div class="fm-row">
        <dt>Type</dt>
        <dd>{{ fmt(artifact.type) }}</dd>
      </div>
      <div class="fm-row">
        <dt>Status</dt>
        <dd>
          <StatusDropdown
            v-if="project && targetPath"
            :project="project"
            :path="targetPath"
            :status="artifact.status"
            @transitioned="emit('transitioned', $event)"
            @error="emit('error', $event)"
          />
          <span v-else class="badge" :data-status="artifact.status">{{ fmt(artifact.status) }}</span>
        </dd>
      </div>
      <div class="fm-row">
        <dt>Stage</dt>
        <dd>{{ fmt(artifact.stage) }}</dd>
      </div>
      <div class="fm-row">
        <dt>Lineage</dt>
        <dd>{{ fmt(artifact.lineage) }}</dd>
      </div>
      <div class="fm-row" v-if="artifact.frontmatter.parent">
        <dt>Parent</dt>
        <dd class="mono">{{ artifact.frontmatter.parent }}</dd>
      </div>
      <div class="fm-row" v-if="artifact.frontmatter.labels?.length">
        <dt>Labels</dt>
        <dd>
          <span v-for="l in artifact.frontmatter.labels" :key="l" class="tag">{{ l }}</span>
        </dd>
      </div>
      <div class="fm-row" v-if="artifact.frontmatter.assignees?.length">
        <dt>Assignees</dt>
        <dd>
          <div v-for="a in artifact.frontmatter.assignees" :key="a.who" class="assignee">
            <span class="assignee-role">{{ a.role }}</span>
            <span>{{ a.who }}</span>
          </div>
        </dd>
      </div>
      <div class="fm-row" v-if="artifact.frontmatter.depends_on?.length">
        <dt>Depends on</dt>
        <dd class="mono-list">
          <div v-for="d in artifact.frontmatter.depends_on" :key="d">{{ d }}</div>
        </dd>
      </div>
      <div class="fm-row" v-if="artifact.frontmatter.blocks?.length">
        <dt>Blocks</dt>
        <dd class="mono-list">
          <div v-for="b in artifact.frontmatter.blocks" :key="b">{{ b }}</div>
        </dd>
      </div>
      <div class="fm-row" v-if="artifact.frontmatter.release">
        <dt>Release</dt>
        <dd>{{ artifact.frontmatter.release }}</dd>
      </div>
      <div class="fm-row" v-if="artifact.frontmatter.sprint">
        <dt>Sprint</dt>
        <dd>{{ artifact.frontmatter.sprint }}</dd>
      </div>
      <div class="fm-row" v-if="artifact.created">
        <dt>Created</dt>
        <dd>
          <span class="date-tip" :title="formatFullDateTime(artifact.created)">
            {{ formatShortDate(artifact.created) }}
          </span>
        </dd>
      </div>
      <div class="fm-row">
        <dt>Modified</dt>
        <dd>
          <span class="date-tip" :title="formatFullDateTime(artifact.mtime)">
            {{ formatShortDate(artifact.mtime) }}
          </span>
        </dd>
      </div>
    </dl>

    <ArtifactRunHistory
      v-if="project && targetPath"
      :project="project"
      :target-path="targetPath"
      @select-run="selectedRunId = $event"
    />

    <RunDetailModal
      v-if="selectedRunId && project"
      :project="project"
      :run-id="selectedRunId"
      @close="selectedRunId = null"
    />

    <div v-if="outbound.length || inbound.length" class="rel-section">
      <h3 class="fm-title">Relationships</h3>
      <div v-if="outbound.length" class="rel-group">
        <div class="rel-group-label">Outbound</div>
        <div v-for="e in outbound" :key="e.target + e.kind" class="rel-item">
          <span class="rel-kind">{{ e.kind }}</span>
          <span class="rel-path">{{ e.target }}</span>
        </div>
      </div>
      <div v-if="inbound.length" class="rel-group">
        <div class="rel-group-label">Inbound</div>
        <div v-for="e in inbound" :key="e.source + e.kind" class="rel-item">
          <span class="rel-kind">{{ e.kind }}</span>
          <span class="rel-path">{{ e.source }}</span>
        </div>
      </div>
    </div>
  </aside>
</template>

<style scoped>
.fm-panel {
  width: 220px;
  flex-shrink: 0;
  background: var(--color-surface);
  border-left: 1px solid var(--color-border);
  padding: var(--space-4);
  overflow-y: auto;
}
.fm-title {
  font-size: var(--text-sm);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
  margin: 0 0 var(--space-3);
}
.fm-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  margin: 0;
}
.fm-row {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.fm-row dt {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--color-text-muted);
}
.fm-row dd {
  margin: 0;
  font-size: var(--text-sm);
  color: var(--color-text);
}
/* Fallback badge for when project/path props are absent (StatusDropdown not rendered) */
.badge {
  display: inline-block;
  padding: 1px 8px;
  border-radius: 99px;
  font-size: 11px;
  font-weight: 500;
  background: var(--color-border);
  color: var(--color-text);
}
.badge[data-status="draft"]          { background: #f3f4f6; color: #374151; }
.badge[data-status="clarifying"]     { background: #ede9fe; color: #5b21b6; }
.badge[data-status="planning"]       { background: #fef3c7; color: #92400e; }
.badge[data-status="in-development"] { background: #dbeafe; color: #1e40af; }
.badge[data-status="in-qa"]          { background: #ede9fe; color: #6d28d9; }
.badge[data-status="approved"]       { background: #d1fae5; color: #065f46; }
.badge[data-status="done"]           { background: #bbf7d0; color: #14532d; }
.badge[data-status="blocked"]        { background: #fee2e2; color: #991b1b; }
.badge[data-status="rejected"]       { background: #fef2f2; color: #b91c1c; }
.badge[data-status="abandoned"]      { background: #f3f4f6; color: #6b7280; }
.badge[data-status="in-progress"]    { background: #fef3c7; color: #92400e; }
@media (prefers-color-scheme: dark) {
  .badge[data-status="draft"]          { background: #374151; color: #d1d5db; }
  .badge[data-status="clarifying"]     { background: #3b2f6e; color: #c4b5fd; }
  .badge[data-status="planning"]       { background: #422006; color: #fcd34d; }
  .badge[data-status="in-development"] { background: #1e3a5f; color: #93c5fd; }
  .badge[data-status="in-qa"]          { background: #2e1065; color: #c4b5fd; }
  .badge[data-status="approved"]       { background: #064e3b; color: #6ee7b7; }
  .badge[data-status="done"]           { background: #052e16; color: #4ade80; }
  .badge[data-status="blocked"]        { background: #7f1d1d; color: #fca5a5; }
  .badge[data-status="rejected"]       { background: #7f1d1d; color: #fca5a5; }
  .badge[data-status="abandoned"]      { background: #1f2937; color: #9ca3af; }
  .badge[data-status="in-progress"]    { background: #422006; color: #fcd34d; }
}
.tag {
  display: inline-block;
  background: var(--color-border);
  border-radius: 4px;
  padding: 1px 6px;
  font-size: 11px;
  margin-right: 4px;
  margin-bottom: 2px;
}
.mono { font-family: monospace; font-size: 12px; }
.mono-list { font-family: monospace; font-size: 12px; display: flex; flex-direction: column; gap: 2px; }
.assignee { display: flex; gap: var(--space-2); align-items: baseline; }
.assignee-role { font-size: 11px; color: var(--color-text-muted); }
.date-tip { cursor: default; }
.rel-section {
  margin-top: var(--space-4);
  border-top: 1px solid var(--color-border);
  padding-top: var(--space-4);
}
.rel-group {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin-bottom: var(--space-3);
}
.rel-group-label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--color-text-muted);
  margin-bottom: 2px;
}
.rel-item {
  display: flex;
  flex-direction: column;
  gap: 1px;
  padding: var(--space-1) 0;
}
.rel-kind {
  font-size: 10px;
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.rel-path {
  font-family: monospace;
  font-size: 11px;
  color: var(--color-text);
  word-break: break-all;
}
</style>
