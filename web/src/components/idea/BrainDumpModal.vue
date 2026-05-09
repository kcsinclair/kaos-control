<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, computed, nextTick, onMounted, onBeforeUnmount, watch } from 'vue'
import { X } from 'lucide-vue-next'
import MarkdownIt from 'markdown-it'
import { useBrainDumpStore } from '@/stores/brainDump'

const props = withDefaults(
  defineProps<{
    project: string
    artifactType?: 'idea' | 'defect'
  }>(),
  { artifactType: 'idea' },
)

const emit = defineEmits<{
  close: []
  created: [path: string]
}>()

const store = useBrainDumpStore()
const md = new MarkdownIt({ html: false, linkify: true, typographer: true })

// Sync artifactType prop into store on open
store.artifactType = props.artifactType

const textareaEl = ref<HTMLTextAreaElement | null>(null)
const editTextareaEl = ref<HTMLTextAreaElement | null>(null)
const panelEl = ref<HTMLElement | null>(null)

// ── Derived labels ─────────────────────────────────────────────────────────

const headerLabel = computed(() =>
  props.artifactType === 'defect' ? 'New Defect' : 'New Idea',
)

const placeholderText = computed(() =>
  props.artifactType === 'defect'
    ? 'Describe the defect — what happened, what you expected...'
    : 'Describe your idea — paste, ramble, brain dump...',
)

const renderedBody = computed(() => {
  const body = store.proposal?.body ?? ''
  return md.render(body)
})

// ── Keyboard handling ──────────────────────────────────────────────────────

function onTextareaKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
    e.preventDefault()
    if (store.canSubmit) {
      store.generate(props.project)
    }
  }
}

function tryClose() {
  if (store.input.trim().length > 0 || store.phase !== 'input') {
    if (!window.confirm('Discard this draft?')) return
  }
  store.reset()
  emit('close')
}

function onPanelKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    tryClose()
    return
  }
  if (e.key === 'Tab') {
    trapFocus(e)
  }
}

function trapFocus(e: KeyboardEvent) {
  if (!panelEl.value) return
  const focusables = Array.from(
    panelEl.value.querySelectorAll<HTMLElement>(
      'button:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])',
    ),
  ).filter((el) => !el.hasAttribute('disabled'))
  if (focusables.length === 0) return
  const first = focusables[0]
  const last = focusables[focusables.length - 1]
  if (e.shiftKey) {
    if (document.activeElement === first) {
      e.preventDefault()
      last.focus()
    }
  } else {
    if (document.activeElement === last) {
      e.preventDefault()
      first.focus()
    }
  }
}

// ── Actions ────────────────────────────────────────────────────────────────

async function onGenerate() {
  if (!store.canSubmit) return
  await store.generate(props.project)
  // focus panel after transition so keyboard users can Tab to actions
  nextTick(() => panelEl.value?.focus())
}

async function onAccept() {
  const path = await store.acceptProposal(props.project)
  if (path) {
    store.reset()
    emit('created', path)
  }
}

function onEdit() {
  store.startEdit()
  nextTick(() => editTextareaEl.value?.focus())
}

function onApplyEdit() {
  store.applyEdit()
}

function onDiscard() {
  store.reset()
  emit('close')
}

// ── Lifecycle ──────────────────────────────────────────────────────────────

onMounted(() => {
  nextTick(() => textareaEl.value?.focus())
})

// Prevent body scroll while modal is open
onMounted(() => { document.body.style.overflow = 'hidden' })
onBeforeUnmount(() => { document.body.style.overflow = '' })

// When entering editing phase, focus the edit textarea
watch(
  () => store.phase,
  (p) => {
    if (p === 'editing') {
      nextTick(() => editTextareaEl.value?.focus())
    }
  },
)
</script>

<template>
  <Teleport to="body">
    <div
      class="bdm-overlay"
      aria-modal="true"
      role="dialog"
      aria-labelledby="bdm-title"
      @click.self="tryClose"
      @keydown="onPanelKeydown"
      tabindex="-1"
    >
      <div class="bdm-panel" ref="panelEl" tabindex="-1">
        <!-- Header -->
        <div class="bdm-header">
          <span class="bdm-title" id="bdm-title">{{ headerLabel }}</span>
          <button class="bdm-close" @click="tryClose" aria-label="Close">
            <X :size="18" />
          </button>
        </div>

        <!-- ── Input phase ───────────────────────────────────────────────── -->
        <template v-if="store.phase === 'input' || store.phase === 'generating'">
          <div class="bdm-body">
            <textarea
              ref="textareaEl"
              class="bdm-textarea"
              v-model="store.input"
              :placeholder="placeholderText"
              :rows="6"
              :disabled="store.phase === 'generating'"
              @keydown="onTextareaKeydown"
              aria-label="Brain dump input"
            />
            <!-- Error message -->
            <div v-if="store.error" class="bdm-error" role="alert">
              {{ store.error }}
            </div>
          </div>
          <div class="bdm-footer">
            <button
              class="btn-primary"
              :disabled="!store.canSubmit || store.phase === 'generating'"
              @click="onGenerate"
            >
              <template v-if="store.phase === 'generating'">
                <span class="bdm-dot" /><span class="bdm-dot" /><span class="bdm-dot" />
              </template>
              <template v-else>Generate</template>
            </button>
          </div>
        </template>

        <!-- ── Preview phase ─────────────────────────────────────────────── -->
        <template v-else-if="store.phase === 'preview' && store.proposal">
          <div class="bdm-body bdm-preview-body">
            <!-- Metadata -->
            <div class="bdm-meta">
              <h3 class="bdm-meta-title">{{ store.proposal.title }}</h3>
              <code class="bdm-meta-slug">{{ store.proposal.slug }}.md</code>
              <div class="bdm-chips" v-if="store.proposal.labels.length">
                <span
                  v-for="lbl in store.proposal.labels"
                  :key="lbl"
                  class="bdm-chip"
                >{{ lbl }}</span>
              </div>
            </div>
            <div class="bdm-divider" />
            <!-- Rendered markdown -->
            <div class="bdm-md-preview md-preview" v-html="renderedBody" />
            <!-- Error from accept -->
            <div v-if="store.error" class="bdm-error" role="alert">
              {{ store.error }}
            </div>
          </div>
          <div class="bdm-footer">
            <button class="btn-primary" @click="onAccept">Accept</button>
            <button class="btn-ghost" @click="onEdit">Edit</button>
            <button class="btn-ghost" @click="onDiscard">Discard</button>
          </div>
        </template>

        <!-- ── Editing phase ──────────────────────────────────────────────── -->
        <template v-else-if="store.phase === 'editing'">
          <div class="bdm-body">
            <textarea
              ref="editTextareaEl"
              class="bdm-textarea bdm-textarea--edit"
              v-model="store.editedBody"
              rows="12"
              aria-label="Edit generated body"
            />
            <div v-if="store.error" class="bdm-error" role="alert">
              {{ store.error }}
            </div>
          </div>
          <div class="bdm-footer">
            <button class="btn-primary" @click="onApplyEdit">Done editing</button>
            <button class="btn-ghost" @click="store.discard(); emit('close')">Cancel</button>
          </div>
        </template>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.bdm-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 300;
}

.bdm-panel {
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  width: 100%;
  max-width: 640px;
  max-height: 88vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  margin: var(--space-4);
  outline: none;
}

/* Header */
.bdm-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4) var(--space-6);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}

.bdm-title {
  font-size: var(--text-lg);
  font-weight: 600;
  color: var(--color-text);
}

.bdm-close {
  background: none;
  border: none;
  cursor: pointer;
  color: var(--color-text-muted);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--space-1);
  border-radius: var(--radius-sm);
}

.bdm-close:hover {
  background: var(--color-surface);
  color: var(--color-text);
}

.bdm-close:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
}

/* Body */
.bdm-body {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-4) var(--space-6);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  min-height: 0;
}

.bdm-preview-body {
  padding-top: var(--space-4);
}

/* Textarea */
.bdm-textarea {
  width: 100%;
  min-height: calc(6 * 1.5em + 2 * var(--space-2));
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-bg);
  color: var(--color-text);
  font-size: var(--text-sm);
  font-family: inherit;
  line-height: 1.5;
  resize: vertical;
  box-sizing: border-box;
}

.bdm-textarea:focus {
  outline: none;
  border-color: var(--color-accent);
}

.bdm-textarea:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
}

.bdm-textarea:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.bdm-textarea--edit {
  min-height: calc(12 * 1.5em + 2 * var(--space-2));
}

/* Error */
.bdm-error {
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  color: var(--color-error, #dc2626);
  background: color-mix(in srgb, var(--color-error, #dc2626) 10%, transparent);
  border: 1px solid color-mix(in srgb, var(--color-error, #dc2626) 30%, transparent);
}

/* Preview metadata */
.bdm-meta {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.bdm-meta-title {
  font-size: var(--text-base);
  font-weight: 600;
  color: var(--color-text);
  margin: 0;
}

.bdm-meta-slug {
  font-family: monospace;
  font-size: 12px;
  color: var(--color-text-muted);
  background: var(--color-border);
  padding: 1px 5px;
  border-radius: 3px;
  align-self: flex-start;
}

.bdm-chips {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-1);
  margin-top: var(--space-1);
}

.bdm-chip {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 99px;
  font-size: 11px;
  font-weight: 500;
  background: var(--color-border);
  color: var(--color-text-muted);
}

.bdm-divider {
  height: 1px;
  background: var(--color-border);
  flex-shrink: 0;
}

/* Markdown preview */
.bdm-md-preview {
  font-size: var(--text-sm);
  line-height: 1.6;
  color: var(--color-text);
}

/* Footer */
.bdm-footer {
  display: flex;
  gap: var(--space-2);
  padding: var(--space-3) var(--space-6) var(--space-4);
  border-top: 1px solid var(--color-border);
  flex-shrink: 0;
}

/* Buttons */
.btn-primary {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: var(--space-2) var(--space-4);
  background: var(--color-accent);
  color: #fff;
  border: none;
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-primary:hover:not(:disabled) {
  opacity: 0.88;
}

.btn-primary:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
}

.btn-ghost {
  padding: var(--space-2) var(--space-3);
  background: none;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  cursor: pointer;
}

.btn-ghost:hover:not(:disabled) {
  background: var(--color-surface);
  color: var(--color-text);
}

.btn-ghost:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.btn-ghost:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
}

/* Generating dots */
.bdm-dot {
  display: inline-block;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
  animation: bdm-bounce 1.2s infinite ease-in-out;
}

.bdm-dot:nth-child(1) { animation-delay: 0s; }
.bdm-dot:nth-child(2) { animation-delay: 0.2s; }
.bdm-dot:nth-child(3) { animation-delay: 0.4s; }

@keyframes bdm-bounce {
  0%, 80%, 100% { transform: translateY(0); opacity: 0.4; }
  40% { transform: translateY(-4px); opacity: 1; }
}

/* Markdown styles */
.md-preview :deep(h1),
.md-preview :deep(h2),
.md-preview :deep(h3),
.md-preview :deep(h4) {
  font-weight: 600;
  margin: 0.75em 0 0.25em;
  line-height: 1.3;
  color: var(--color-text);
}

.md-preview :deep(h1) { font-size: var(--text-xl); }
.md-preview :deep(h2) { font-size: var(--text-lg); }
.md-preview :deep(h3) { font-size: var(--text-base); }

.md-preview :deep(p) { margin: 0.5em 0; }

.md-preview :deep(ul),
.md-preview :deep(ol) { padding-left: 1.25em; margin: 0.5em 0; }

.md-preview :deep(li) { margin: 0.15em 0; }

.md-preview :deep(code) {
  font-family: monospace;
  font-size: 0.875em;
  background: var(--color-border);
  padding: 1px 4px;
  border-radius: 3px;
}

.md-preview :deep(pre) {
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  padding: var(--space-3);
  overflow-x: auto;
  font-size: var(--text-xs);
}

.md-preview :deep(a) { color: var(--color-accent); text-decoration: none; }
.md-preview :deep(a:hover) { text-decoration: underline; }

.md-preview :deep(blockquote) {
  border-left: 3px solid var(--color-border);
  margin: 0.5em 0;
  padding-left: var(--space-3);
  color: var(--color-text-muted);
}

.md-preview :deep(hr) {
  border: none;
  border-top: 1px solid var(--color-border);
  margin: 0.75em 0;
}

/* Responsive */
@media (max-width: 768px) {
  .bdm-panel {
    margin: var(--space-2);
    max-height: 95vh;
  }
}
</style>
