<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount, useId } from 'vue'
import { patchRice } from '@/api/artifacts'
import {
  riceScore,
  formatRice,
  validateRiceComponent,
  RICE_DEFAULTS,
  IMPACT_TIERS,
  type RiceComponents,
} from '@/lib/rice'
import type { ArtifactFrontmatter } from '@/types/api'

const props = defineProps<{
  project: string
  path: string
  type: string
  frontmatter: ArtifactFrontmatter
  readonly?: boolean
}>()

const emit = defineEmits<{
  changed: [components: RiceComponents]
  error: [message: string]
}>()

const uid = useId()

function extractComponents(fm: ArtifactFrontmatter): RiceComponents {
  return {
    rice_reach: fm.rice_reach ?? null,
    rice_impact: fm.rice_impact ?? null,
    rice_confidence: fm.rice_confidence ?? null,
    rice_effort: fm.rice_effort ?? null,
  }
}

// ── state ─────────────────────────────────────────────────────────────────────
const isOpen = ref(false)
const stored = ref<RiceComponents>(extractComponents(props.frontmatter))

const reachStr = ref('')
const impactStr = ref('')
const confidenceStr = ref('')
const effortStr = ref('')

const wrapRef = ref<HTMLElement | null>(null)
const triggerRef = ref<HTMLButtonElement | null>(null)

// ── watch prop for external WebSocket updates ─────────────────────────────────
watch(() => props.frontmatter, (fm) => {
  if (!isOpen.value) stored.value = extractComponents(fm)
}, { deep: true })

const closedLabel = computed(() => formatRice(riceScore(stored.value)))

// ── parsing + local live state ──────────────────────────────────────────────
function parseField(raw: string): number | null {
  const t = raw.trim()
  if (t === '') return null
  return Number(t)
}

const localComponents = computed<RiceComponents>(() => ({
  rice_reach: parseField(reachStr.value),
  rice_impact: parseField(impactStr.value),
  rice_confidence: parseField(confidenceStr.value),
  rice_effort: parseField(effortStr.value),
}))

function fieldError(field: keyof RiceComponents, raw: string, value: number | null): string | null {
  if (raw.trim() === '') return null
  return validateRiceComponent(field, value)
}

const reachError = computed(() => fieldError('rice_reach', reachStr.value, localComponents.value.rice_reach))
const impactError = computed(() => fieldError('rice_impact', impactStr.value, localComponents.value.rice_impact))
const confidenceError = computed(() => fieldError('rice_confidence', confidenceStr.value, localComponents.value.rice_confidence))
const effortError = computed(() => fieldError('rice_effort', effortStr.value, localComponents.value.rice_effort))

const hasErrors = computed(() =>
  !!(reachError.value || impactError.value || confidenceError.value || effortError.value))

const previewText = computed(() => formatRice(riceScore(localComponents.value)))

const canSave = computed(() => !hasErrors.value)

// ── open / close ──────────────────────────────────────────────────────────────
function openEditor() {
  if (isOpen.value || props.readonly) return
  isOpen.value = true
  const hasAny = stored.value.rice_reach != null || stored.value.rice_impact != null ||
    stored.value.rice_confidence != null || stored.value.rice_effort != null
  const seed = hasAny
    ? stored.value
    : {
        rice_reach: RICE_DEFAULTS.reach,
        rice_impact: RICE_DEFAULTS.impact,
        rice_confidence: RICE_DEFAULTS.confidence,
        rice_effort: RICE_DEFAULTS.effort,
      }
  reachStr.value = seed.rice_reach != null ? String(seed.rice_reach) : ''
  impactStr.value = seed.rice_impact != null ? String(seed.rice_impact) : ''
  confidenceStr.value = seed.rice_confidence != null ? String(seed.rice_confidence) : ''
  effortStr.value = seed.rice_effort != null ? String(seed.rice_effort) : ''
}

function closeEditor() {
  isOpen.value = false
}

function clearField(field: 'reach' | 'impact' | 'confidence' | 'effort') {
  if (field === 'reach') reachStr.value = ''
  if (field === 'impact') impactStr.value = ''
  if (field === 'confidence') confidenceStr.value = ''
  if (field === 'effort') effortStr.value = ''
}

// ── save ──────────────────────────────────────────────────────────────────────
function buildPayload(): RiceComponents {
  const payload: RiceComponents = {}
  const keys: (keyof RiceComponents)[] = ['rice_reach', 'rice_impact', 'rice_confidence', 'rice_effort']
  for (const k of keys) {
    if (localComponents.value[k] !== stored.value[k]) payload[k] = localComponents.value[k]
  }
  return payload
}

async function handleSave() {
  if (!canSave.value) return
  const payload = buildPayload()
  if (Object.keys(payload).length === 0) {
    closeEditor()
    return
  }

  const previous = stored.value
  const next = { ...stored.value, ...localComponents.value }
  stored.value = next   // optimistic update
  closeEditor()
  triggerRef.value?.focus()

  try {
    await patchRice(props.project, props.path, payload)
    emit('changed', next)
  } catch (e: unknown) {
    stored.value = previous   // revert on failure
    const msg = e instanceof Error ? e.message : 'RICE update failed'
    emit('error', msg)
  }
}

// ── outside click ─────────────────────────────────────────────────────────────
function onDocumentClick(e: MouseEvent) {
  if (!isOpen.value) return
  if (wrapRef.value?.contains(e.target as Node)) return
  closeEditor()
}

onMounted(() => document.addEventListener('click', onDocumentClick, true))
onBeforeUnmount(() => document.removeEventListener('click', onDocumentClick, true))
</script>

<template>
  <span ref="wrapRef" class="rice-editor-wrap">

    <!-- Read-only badge (no permission or foreign lock) -->
    <span v-if="readonly" class="rice-badge">{{ closedLabel }}</span>

    <!-- Interactive trigger -->
    <button
      v-else
      ref="triggerRef"
      type="button"
      class="rice-badge rice-badge--interactive"
      aria-haspopup="dialog"
      :aria-expanded="isOpen ? 'true' : 'false'"
      :aria-label="`Edit RICE score for this ${type}, currently ${closedLabel}`"
      @click="isOpen ? closeEditor() : openEditor()"
      @keydown.esc="closeEditor"
    >{{ closedLabel }}</button>

    <!-- Editor panel -->
    <div
      v-if="isOpen"
      class="rice-panel"
      role="dialog"
      aria-label="Edit RICE score"
      @keydown.esc="closeEditor"
      @click.stop
    >
      <div class="rice-field">
        <label :for="`rice-reach-${uid}`">Reach</label>
        <div class="rice-input-row">
          <input :id="`rice-reach-${uid}`" v-model="reachStr" type="number" step="any" min="0" />
          <button v-if="reachStr" type="button" class="rice-clear" aria-label="Clear Reach" @click="clearField('reach')">×</button>
        </div>
        <span v-if="reachError" class="rice-error">{{ reachError }}</span>
      </div>

      <div class="rice-field">
        <label :for="`rice-impact-${uid}`">Impact</label>
        <div class="rice-input-row">
          <select :id="`rice-impact-${uid}`" v-model="impactStr">
            <option value="">—</option>
            <option v-for="t in IMPACT_TIERS" :key="t" :value="String(t)">{{ t }}</option>
          </select>
          <button v-if="impactStr" type="button" class="rice-clear" aria-label="Clear Impact" @click="clearField('impact')">×</button>
        </div>
        <span v-if="impactError" class="rice-error">{{ impactError }}</span>
      </div>

      <div class="rice-field">
        <label :for="`rice-confidence-${uid}`">Confidence (%)</label>
        <div class="rice-input-row">
          <input :id="`rice-confidence-${uid}`" v-model="confidenceStr" type="number" step="any" min="0" max="100" />
          <button v-if="confidenceStr" type="button" class="rice-clear" aria-label="Clear Confidence" @click="clearField('confidence')">×</button>
        </div>
        <span v-if="confidenceError" class="rice-error">{{ confidenceError }}</span>
      </div>

      <div class="rice-field">
        <label :for="`rice-effort-${uid}`">Effort (person-months)</label>
        <div class="rice-input-row">
          <input :id="`rice-effort-${uid}`" v-model="effortStr" type="number" step="any" min="0" />
          <button v-if="effortStr" type="button" class="rice-clear" aria-label="Clear Effort" @click="clearField('effort')">×</button>
        </div>
        <span v-if="effortError" class="rice-error">{{ effortError }}</span>
      </div>

      <div class="rice-preview">
        Preview: <strong>{{ previewText }}</strong>
      </div>

      <div class="rice-actions">
        <button type="button" class="rice-btn-cancel" @click="closeEditor">Cancel</button>
        <button type="button" class="rice-btn-save" :disabled="!canSave" @click="handleSave">Save</button>
      </div>
    </div>

  </span>
</template>

<style scoped>
/* ── wrapper ─────────────────────────────────────────────────────────────────*/
.rice-editor-wrap {
  position: relative;
  display: inline-block;
}

/* ── badge ───────────────────────────────────────────────────────────────────*/
.rice-badge {
  display: inline-block;
  padding: 1px 8px;
  border-radius: 99px;
  border: 1px solid var(--color-border);
  font-size: 11px;
  font-weight: 500;
  white-space: nowrap;
  user-select: none;
  line-height: 1.6;
  font-variant-numeric: tabular-nums;
  color: var(--color-text);
  background: var(--color-surface-raised, rgba(0, 0, 0, 0.04));
}
.rice-badge--interactive {
  cursor: pointer;
  font-family: inherit;
  text-align: left;
}
.rice-badge--interactive:hover {
  opacity: 0.85;
}
.rice-badge--interactive:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
}

/* ── panel ───────────────────────────────────────────────────────────────────*/
.rice-panel {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  z-index: 100;
  width: 220px;
  background: var(--color-surface, #fff);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md, 6px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
  padding: 10px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.rice-field {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.rice-field label {
  font-size: 11px;
  font-weight: 600;
  color: var(--color-text-muted);
}
.rice-input-row {
  display: flex;
  align-items: center;
  gap: 4px;
}
.rice-input-row input,
.rice-input-row select {
  flex: 1;
  min-width: 0;
  padding: 3px 6px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm, 4px);
  background: var(--color-bg);
  color: var(--color-text);
  font-size: 12px;
}
.rice-clear {
  flex-shrink: 0;
  width: 18px;
  height: 18px;
  line-height: 1;
  border: none;
  background: none;
  color: var(--color-text-muted);
  cursor: pointer;
  font-size: 14px;
}
.rice-clear:hover { color: var(--color-text); }
.rice-error {
  font-size: 10px;
  color: var(--color-error, #dc2626);
}
.rice-preview {
  font-size: 12px;
  color: var(--color-text-muted);
  border-top: 1px solid var(--color-border);
  padding-top: 6px;
}
.rice-preview strong {
  color: var(--color-text);
  font-variant-numeric: tabular-nums;
}
.rice-actions {
  display: flex;
  justify-content: flex-end;
  gap: 6px;
}
.rice-btn-cancel,
.rice-btn-save {
  padding: 3px 10px;
  border-radius: var(--radius-sm, 4px);
  font-size: 12px;
  cursor: pointer;
}
.rice-btn-cancel {
  border: 1px solid var(--color-border);
  background: none;
  color: var(--color-text-muted);
}
.rice-btn-cancel:hover { background: var(--color-surface); color: var(--color-text); }
.rice-btn-save {
  border: 1px solid transparent;
  background: var(--color-accent);
  color: #fff;
}
.rice-btn-save:hover:not(:disabled) { opacity: 0.88; }
.rice-btn-save:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
