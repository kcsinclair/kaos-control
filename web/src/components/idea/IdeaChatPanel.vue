<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onBeforeUnmount } from 'vue'
import { X, SendHorizontal } from 'lucide-vue-next'
import MarkdownIt from 'markdown-it'
import { useIdeaChatStore } from '@/stores/ideaChat'
import { useUiStore } from '@/stores/ui'

const props = defineProps<{
  project: string
}>()

const emit = defineEmits<{
  close: []
  created: [path: string]
}>()

const store = useIdeaChatStore()
const ui = useUiStore()
const md = new MarkdownIt({ html: false, linkify: true, typographer: true })

// --- refs ---
const inputText = ref('')
const messagesEl = ref<HTMLElement | null>(null)
const textareaEl = ref<HTMLTextAreaElement | null>(null)
const panelEl = ref<HTMLElement | null>(null)
const confirmDiscard = ref(false)
const closeButtonEl = ref<HTMLButtonElement | null>(null)

// --- computed ---
const renderedPreviewBody = computed(() => {
  if (!store.preview?.body) return ''
  return md.render(store.preview.body)
})

const previewMeta = computed(() => {
  const fm = store.preview?.frontmatter ?? {}
  const lineage = (fm.lineage as string) ?? ''
  const title = (fm.title as string) ?? ''
  const labels = Array.isArray(fm.labels) ? (fm.labels as string[]) : []
  const slug = lineage ? `${lineage}.md` : ''
  return { title, slug, labels, lineage }
})

// --- helpers ---
function scrollToBottom() {
  nextTick(() => {
    if (messagesEl.value) {
      messagesEl.value.scrollTop = messagesEl.value.scrollHeight
    }
  })
}

function autoGrow(el: HTMLTextAreaElement) {
  el.style.height = 'auto'
  const lineHeight = parseInt(getComputedStyle(el).lineHeight) || 20
  const maxH = lineHeight * 4
  el.style.height = Math.min(el.scrollHeight, maxH) + 'px'
}

// --- actions ---
async function send() {
  const text = inputText.value.trim()
  if (!text || store.loading) return
  inputText.value = ''
  if (textareaEl.value) {
    textareaEl.value.style.height = 'auto'
  }
  await store.sendMessage(props.project, text)
  scrollToBottom()
  nextTick(() => textareaEl.value?.focus())
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    send()
  }
}

async function onAccept() {
  await store.acceptProposal(props.project)
  if (store.status === 'created' && store.createdPath) {
    ui.success('Idea created!')
    emit('created', store.createdPath)
  }
}

async function onEdit() {
  await store.sendMessage(props.project, "I'd like to make some changes")
  scrollToBottom()
  nextTick(() => textareaEl.value?.focus())
}

async function onDiscard() {
  await store.rejectProposal(props.project)
  ui.info('Conversation discarded.')
  emit('close')
}

function tryClose() {
  if (store.status !== 'idle') {
    confirmDiscard.value = true
  } else {
    emit('close')
  }
}

function cancelDiscard() {
  confirmDiscard.value = false
  nextTick(() => textareaEl.value?.focus())
}

function confirmAndClose() {
  store.reset()
  emit('close')
}

// --- focus trap ---
function getFocusables(): HTMLElement[] {
  if (!panelEl.value) return []
  return Array.from(
    panelEl.value.querySelectorAll<HTMLElement>(
      'button:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])',
    ),
  ).filter((el) => !el.hasAttribute('disabled'))
}

function onPanelKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    tryClose()
    return
  }
  if (e.key === 'Tab') {
    const focusables = getFocusables()
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
}

// --- lifecycle ---
watch(
  () => store.messages.length,
  () => scrollToBottom(),
)

onMounted(() => {
  nextTick(() => textareaEl.value?.focus())
})

onBeforeUnmount(() => {
  // return focus to trigger element — handled by parent via ref
})
</script>

<template>
  <div
    class="icp-overlay"
    aria-modal="true"
    role="dialog"
    aria-labelledby="icp-title"
    @click.self="tryClose"
    @keydown="onPanelKeydown"
    tabindex="-1"
  >
    <div class="icp-panel" ref="panelEl">
      <!-- Header -->
      <div class="icp-header">
        <span class="icp-title" id="icp-title">New Idea</span>
        <button class="icp-close" @click="tryClose" aria-label="Close" ref="closeButtonEl">
          <X :size="18" />
        </button>
      </div>

      <!-- Discard confirmation -->
      <div v-if="confirmDiscard" class="icp-confirm-bar">
        <span class="icp-confirm-text">Discard this conversation?</span>
        <button class="btn-danger-sm" @click="confirmAndClose">Discard</button>
        <button class="btn-ghost-sm" @click="cancelDiscard">Keep going</button>
      </div>

      <!-- Message area -->
      <div class="icp-messages" ref="messagesEl" aria-live="polite" aria-relevant="additions">
        <div v-if="store.messages.length === 0" class="icp-empty-hint">
          Describe your idea and I'll help you capture it as an artifact.
        </div>
        <div
          v-for="(msg, idx) in store.messages"
          :key="idx"
          class="icp-message"
          :class="msg.role === 'user' ? 'icp-message--user' : 'icp-message--assistant'"
        >
          {{ msg.content }}
        </div>
        <div v-if="store.loading" class="icp-loading">
          <span class="icp-dot" /><span class="icp-dot" /><span class="icp-dot" />
        </div>
      </div>

      <!-- Proposal state: preview + actions -->
      <template v-if="store.status === 'proposed' && store.preview">
        <div class="icp-preview-card">
          <!-- Metadata summary -->
          <div class="icp-preview-meta">
            <div class="icp-meta-row" v-if="previewMeta.title">
              <span class="icp-meta-label">Title</span>
              <span class="icp-meta-value">{{ previewMeta.title }}</span>
            </div>
            <div class="icp-meta-row" v-if="previewMeta.slug">
              <span class="icp-meta-label">File</span>
              <code class="icp-meta-value icp-meta-slug">{{ previewMeta.slug }}</code>
            </div>
            <div class="icp-meta-row" v-if="previewMeta.lineage">
              <span class="icp-meta-label">Lineage</span>
              <span class="icp-meta-value">{{ previewMeta.lineage }}</span>
            </div>
            <div class="icp-meta-row" v-if="previewMeta.labels.length">
              <span class="icp-meta-label">Labels</span>
              <span class="icp-meta-chips">
                <span v-for="lbl in previewMeta.labels" :key="lbl" class="icp-chip">{{ lbl }}</span>
              </span>
            </div>
          </div>
          <!-- Body preview -->
          <div class="icp-preview-divider" />
          <div class="icp-preview-body md-preview" v-html="renderedPreviewBody" />
        </div>
        <div class="icp-proposal-actions">
          <button class="btn-primary" :disabled="store.loading" @click="onAccept">Accept</button>
          <button class="btn-ghost" :disabled="store.loading" @click="onEdit">Edit</button>
          <button class="btn-ghost" :disabled="store.loading" @click="onDiscard">Discard</button>
        </div>
      </template>

      <!-- Input area (active conversation) -->
      <template v-else-if="store.status !== 'created'">
        <div class="icp-input-row">
          <textarea
            ref="textareaEl"
            class="icp-textarea"
            v-model="inputText"
            placeholder="Type your idea…"
            rows="1"
            :disabled="store.loading"
            @keydown="onKeydown"
            @input="autoGrow($event.target as HTMLTextAreaElement)"
            aria-label="Message input"
          />
          <button
            class="icp-send"
            :disabled="store.loading || !inputText.trim()"
            @click="send"
            aria-label="Send message"
          >
            <SendHorizontal :size="18" />
          </button>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.icp-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 300;
}

.icp-panel {
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  width: 100%;
  max-width: 560px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* Header */
.icp-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4) var(--space-6);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}

.icp-title {
  font-size: var(--text-lg);
  font-weight: 600;
  color: var(--color-text);
}

.icp-close {
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

.icp-close:hover {
  background: var(--color-surface);
  color: var(--color-text);
}

.icp-close:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
}

/* Discard confirm bar */
.icp-confirm-bar {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-6);
  background: var(--color-surface);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}

.icp-confirm-text {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  flex: 1;
}

.btn-danger-sm {
  padding: var(--space-1) var(--space-3);
  background: var(--color-error);
  color: #fff;
  border: none;
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  cursor: pointer;
}

.btn-danger-sm:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
}

.btn-ghost-sm {
  padding: var(--space-1) var(--space-3);
  background: none;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  cursor: pointer;
}

.btn-ghost-sm:hover {
  background: var(--color-surface);
}

.btn-ghost-sm:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
}

/* Messages */
.icp-messages {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding: var(--space-4) var(--space-6);
  min-height: 0;
}

.icp-empty-hint {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  text-align: center;
  margin-top: var(--space-8);
}

.icp-message {
  max-width: 88%;
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-word;
}

.icp-message--user {
  align-self: flex-end;
  background: var(--color-accent);
  color: #fff;
  border-bottom-right-radius: var(--radius-sm);
}

.icp-message--assistant {
  align-self: flex-start;
  background: var(--color-surface);
  color: var(--color-text);
  border: 1px solid var(--color-border);
  border-bottom-left-radius: var(--radius-sm);
}

/* Loading dots */
.icp-loading {
  align-self: flex-start;
  display: flex;
  gap: 4px;
  padding: var(--space-2) var(--space-3);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  border-bottom-left-radius: var(--radius-sm);
}

.icp-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--color-text-muted);
  animation: icp-bounce 1.2s infinite ease-in-out;
}

.icp-dot:nth-child(1) { animation-delay: 0s; }
.icp-dot:nth-child(2) { animation-delay: 0.2s; }
.icp-dot:nth-child(3) { animation-delay: 0.4s; }

@keyframes icp-bounce {
  0%, 80%, 100% { transform: translateY(0); opacity: 0.4; }
  40% { transform: translateY(-5px); opacity: 1; }
}

/* Preview card */
.icp-preview-card {
  margin: 0 var(--space-6);
  border: 1px solid var(--color-accent);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  overflow-y: auto;
  max-height: 240px;
  flex-shrink: 0;
}

/* Metadata summary */
.icp-preview-meta {
  padding: var(--space-3) var(--space-4);
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.icp-meta-row {
  display: flex;
  align-items: flex-start;
  gap: var(--space-2);
  font-size: var(--text-xs);
  line-height: 1.5;
}

.icp-meta-label {
  flex-shrink: 0;
  width: 52px;
  color: var(--color-text-muted);
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  font-size: 10px;
  padding-top: 1px;
}

.icp-meta-value {
  color: var(--color-text);
  font-size: var(--text-xs);
}

.icp-meta-slug {
  font-family: monospace;
  font-size: 11px;
  background: var(--color-border);
  padding: 1px 4px;
  border-radius: 3px;
}

.icp-meta-chips {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-1);
}

.icp-chip {
  display: inline-block;
  padding: 1px 6px;
  border-radius: 99px;
  font-size: 10px;
  font-weight: 500;
  background: var(--color-border);
  color: var(--color-text-muted);
  border: 1px solid var(--color-border);
}

.icp-preview-divider {
  height: 1px;
  background: var(--color-border);
  margin: 0;
}

.icp-preview-body {
  padding: var(--space-4);
  font-size: var(--text-sm);
  line-height: 1.6;
  color: var(--color-text);
  max-width: none;
}

/* Proposal actions */
.icp-proposal-actions {
  display: flex;
  gap: var(--space-2);
  padding: var(--space-3) var(--space-6) var(--space-4);
  flex-shrink: 0;
}

/* Input row */
.icp-input-row {
  display: flex;
  align-items: flex-end;
  gap: var(--space-2);
  padding: var(--space-3) var(--space-6) var(--space-4);
  border-top: 1px solid var(--color-border);
  flex-shrink: 0;
}

.icp-textarea {
  flex: 1;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-bg);
  color: var(--color-text);
  font-size: var(--text-sm);
  font-family: inherit;
  line-height: 1.5;
  resize: none;
  overflow-y: hidden;
}

.icp-textarea:focus {
  outline: none;
  border-color: var(--color-accent);
}

.icp-textarea:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
}

.icp-textarea:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.icp-send {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  background: var(--color-accent);
  color: #fff;
  border: none;
  border-radius: var(--radius-md);
  cursor: pointer;
}

.icp-send:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.icp-send:hover:not(:disabled) {
  opacity: 0.88;
}

.icp-send:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
}

/* Shared buttons */
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
}

.btn-ghost:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.btn-ghost:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
}

/* Inline markdown preview resets for scoped context */
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
</style>
