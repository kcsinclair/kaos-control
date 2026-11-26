<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, computed, nextTick, onMounted, onBeforeUnmount } from 'vue'
import { X } from 'lucide-vue-next'
import { useOpenQuestions } from '@/composables/useOpenQuestions'
import MarkdownPreview from './MarkdownPreview.vue'
import type { ArtifactFrontmatter } from '@/types/api'

const props = defineProps<{
  project: string
  path: string
  frontmatter: ArtifactFrontmatter
}>()

const emit = defineEmits<{
  close: []
  finished: []
}>()

const { questions, answers, loading, saving, error, load, save, finish } =
  useOpenQuestions(props.project, props.path)

const currentIndex = ref(0)
const panelEl = ref<HTMLElement | null>(null)
const textareaEl = ref<HTMLTextAreaElement | null>(null)

const currentQuestion = computed(() => questions.value[currentIndex.value] ?? null)
const isFirst = computed(() => currentIndex.value === 0)
const isLast = computed(() => currentIndex.value === questions.value.length - 1)
const allAnswered = computed(() =>
  questions.value.every((q) => (answers[q.index] ?? '').trim().length > 0),
)

function focusTextarea() {
  void nextTick(() => textareaEl.value?.focus())
}

async function trySave(): Promise<boolean> {
  try {
    await save(props.frontmatter)
    return true
  } catch {
    return false // error is already surfaced via the composable's `error` ref
  }
}

async function goNext() {
  if (isLast.value) return
  if (!(await trySave())) return
  currentIndex.value++
  focusTextarea()
}

async function goBack() {
  if (isFirst.value) return
  if (!(await trySave())) return
  currentIndex.value--
  focusTextarea()
}

async function handleFinish() {
  if (!allAnswered.value) return
  try {
    await finish(props.frontmatter)
    emit('finished')
  } catch {
    // error is already surfaced via the composable's `error` ref
  }
}

async function handleClose() {
  await trySave()
  emit('close')
}

function onTextareaKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
    e.preventDefault()
    if (isLast.value) {
      if (allAnswered.value) void handleFinish()
    } else {
      void goNext()
    }
  }
}

function trapFocus(e: KeyboardEvent) {
  if (!panelEl.value) return
  const focusables = Array.from(
    panelEl.value.querySelectorAll<HTMLElement>(
      'button:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])',
    ),
  )
  if (focusables.length === 0) return
  const first = focusables[0]
  const last = focusables[focusables.length - 1]
  if (e.shiftKey) {
    if (document.activeElement === first) {
      e.preventDefault()
      last.focus()
    }
  } else if (document.activeElement === last) {
    e.preventDefault()
    first.focus()
  }
}

function onPanelKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    void handleClose()
    return
  }
  if (e.key === 'Tab') trapFocus(e)
}

onMounted(async () => {
  document.body.style.overflow = 'hidden'
  await load()
  focusTextarea()
})
onBeforeUnmount(() => { document.body.style.overflow = '' })
</script>

<template>
  <Teleport to="body">
    <div
      class="rqm-overlay"
      role="dialog"
      aria-modal="true"
      aria-labelledby="rqm-title"
      tabindex="-1"
      @keydown="onPanelKeydown"
    >
      <div class="rqm-panel" ref="panelEl" tabindex="-1">
        <div class="rqm-header">
          <span class="rqm-title" id="rqm-title">Resolve Open Questions</span>
          <span v-if="questions.length" class="rqm-progress">{{ currentIndex + 1 }} of {{ questions.length }}</span>
          <button class="rqm-close" aria-label="Close" @click="handleClose">
            <X :size="18" />
          </button>
        </div>

        <div v-if="loading" class="rqm-state-msg">Loading…</div>
        <div v-else-if="!questions.length" class="rqm-state-msg">Nothing to resolve.</div>

        <template v-else-if="currentQuestion">
          <div class="rqm-body">
            <div class="rqm-question">
              <MarkdownPreview :source="currentQuestion.text" :project="project" />
            </div>
            <label class="rqm-answer-label" :for="`rqm-answer-${currentQuestion.index}`">Your answer</label>
            <textarea
              :id="`rqm-answer-${currentQuestion.index}`"
              ref="textareaEl"
              class="rqm-textarea"
              v-model="answers[currentQuestion.index]"
              rows="6"
              placeholder="Type your answer…"
              @keydown="onTextareaKeydown"
            />
          </div>

          <div v-if="error" class="rqm-error">{{ error }}</div>

          <div class="rqm-actions">
            <button class="btn-ghost" :disabled="isFirst || saving" @click="goBack">Back</button>
            <button
              v-if="!isLast"
              class="btn-primary"
              :disabled="saving"
              @click="goNext"
            >{{ saving ? 'Saving…' : 'Next' }}</button>
            <button
              v-else
              class="btn-primary"
              :disabled="!allAnswered || saving"
              :title="!allAnswered ? 'Answer every question to finish' : undefined"
              @click="handleFinish"
            >{{ saving ? 'Saving…' : 'Finish' }}</button>
          </div>
        </template>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.rqm-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 300;
}
.rqm-panel {
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  padding: var(--space-6);
  width: 520px;
  max-width: calc(100vw - var(--space-6));
  max-height: calc(100vh - var(--space-6));
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}
.rqm-header {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}
.rqm-title {
  font-size: var(--text-lg);
  font-weight: 600;
  color: var(--color-text);
}
.rqm-progress {
  margin-left: auto;
  font-size: var(--text-sm);
  color: var(--color-text-muted);
}
.rqm-close {
  background: none;
  border: none;
  padding: var(--space-1);
  cursor: pointer;
  color: var(--color-text-muted);
  display: flex;
  align-items: center;
}
.rqm-close:hover { color: var(--color-text); }
.rqm-close:focus-visible { outline: 2px solid var(--color-accent); outline-offset: 2px; }
.rqm-state-msg {
  padding: var(--space-6) 0;
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}
.rqm-body {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}
.rqm-question {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: var(--space-3) var(--space-4);
}
.rqm-answer-label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--color-text-muted);
}
.rqm-textarea {
  padding: var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  color: var(--color-text);
  font-size: var(--text-sm);
  font-family: inherit;
  width: 100%;
  box-sizing: border-box;
  resize: vertical;
}
.rqm-textarea:focus {
  outline: none;
  border-color: var(--color-accent);
}
.rqm-error {
  font-size: var(--text-sm);
  color: #dc2626;
  background: #fee2e2;
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm);
}
.rqm-actions {
  display: flex;
  gap: var(--space-2);
  justify-content: flex-end;
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
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-primary:hover:not(:disabled) { opacity: 0.88; }
.btn-primary:focus-visible { outline: 2px solid var(--color-accent); outline-offset: 2px; }
.btn-ghost {
  padding: var(--space-2) var(--space-3);
  background: none;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  cursor: pointer;
}
.btn-ghost:hover:not(:disabled) { background: var(--color-surface); }
.btn-ghost:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-ghost:focus-visible { outline: 2px solid var(--color-accent); outline-offset: 2px; }
</style>
