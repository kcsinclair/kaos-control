<script setup lang="ts">
import { ref } from 'vue'
import type { ArtifactDetail } from '@/types/api'
import { formatShortDate, formatFullDateTime } from '@/composables/useFormatDate'
import ArtifactRunHistory from './ArtifactRunHistory.vue'
import RunDetailModal from '@/components/agent/RunDetailModal.vue'

defineProps<{
  artifact: ArtifactDetail
  project?: string
  targetPath?: string
}>()

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
        <dd><span class="badge" :data-status="artifact.status">{{ fmt(artifact.status) }}</span></dd>
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
.badge {
  display: inline-block;
  padding: 1px 8px;
  border-radius: 99px;
  font-size: 11px;
  font-weight: 500;
  background: var(--color-border);
}
.badge[data-status="done"] { background: #d1fae5; color: #065f46; }
.badge[data-status="approved"] { background: #dbeafe; color: #1e40af; }
.badge[data-status="in-progress"] { background: #fef3c7; color: #92400e; }
.badge[data-status="blocked"] { background: #fee2e2; color: #991b1b; }
.badge[data-status="clarifying"] { background: var(--badge-clarifying-bg); color: var(--badge-clarifying-text); }
.badge[data-status="planning"]   { background: var(--badge-planning-bg);   color: var(--badge-planning-text); }
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
</style>
