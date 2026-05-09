<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { basicSetup, EditorView } from 'codemirror'
import { keymap } from '@codemirror/view'
import { markdown } from '@codemirror/lang-markdown'
import { oneDark } from '@codemirror/theme-one-dark'
import { defaultKeymap } from '@codemirror/commands'
import { Compartment } from '@codemirror/state'

const props = defineProps<{ modelValue: string }>()
const emit = defineEmits<{
  'update:modelValue': [value: string]
  save: []
}>()

const container = ref<HTMLElement>()
let view: EditorView | null = null
let ignoreNextUpdate = false

const wrapCompartment = new Compartment()
const wrapLines = ref(localStorage.getItem('kaos-editor-wrap') !== 'false')

onMounted(() => {
  if (!container.value) return
  view = new EditorView({
    doc: props.modelValue,
    extensions: [
      basicSetup,
      markdown(),
      oneDark,
      wrapCompartment.of(wrapLines.value ? EditorView.lineWrapping : []),
      keymap.of([
        ...defaultKeymap,
        { key: 'Mod-s', run: () => { emit('save'); return true } },
      ]),
      EditorView.updateListener.of((upd) => {
        if (upd.docChanged && !ignoreNextUpdate) {
          emit('update:modelValue', upd.state.doc.toString())
        }
      }),
    ],
    parent: container.value,
  })
})

onUnmounted(() => {
  view?.destroy()
  view = null
})

function toggleWrap() {
  wrapLines.value = !wrapLines.value
  localStorage.setItem('kaos-editor-wrap', String(wrapLines.value))
  view?.dispatch({
    effects: wrapCompartment.reconfigure(wrapLines.value ? EditorView.lineWrapping : []),
  })
}

// Reload content without triggering update:modelValue (e.g. after external change reload)
watch(
  () => props.modelValue,
  (val) => {
    if (!view) return
    const cur = view.state.doc.toString()
    if (cur === val) return
    ignoreNextUpdate = true
    view.dispatch({ changes: { from: 0, to: cur.length, insert: val } })
    ignoreNextUpdate = false
  },
)
</script>

<template>
  <div class="md-editor-wrap">
    <div class="md-toolbar">
      <button
        class="toolbar-btn"
        :class="{ 'toolbar-btn--active': wrapLines }"
        :title="wrapLines ? 'Disable line wrap' : 'Enable line wrap'"
        @click="toggleWrap"
      >
        Wrap
      </button>
    </div>
    <div ref="container" class="md-editor" />
  </div>
</template>

<style scoped>
.md-editor-wrap {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}
.md-toolbar {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: 4px var(--space-2);
  background: var(--color-surface);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}
.toolbar-btn {
  padding: 2px 8px;
  background: none;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: 11px;
  color: var(--color-text-muted);
  cursor: pointer;
  transition: background 0.1s, color 0.1s, border-color 0.1s;
}
.toolbar-btn:hover {
  background: var(--color-border);
  color: var(--color-text);
}
.toolbar-btn--active {
  background: var(--color-accent);
  border-color: var(--color-accent);
  color: #fff;
}
.md-editor {
  flex: 1;
  overflow: hidden;
  font-size: 14px;
}
.md-editor :deep(.cm-editor) {
  height: 100%;
}
.md-editor :deep(.cm-scroller) {
  overflow: auto;
  font-family: 'JetBrains Mono', 'Fira Code', ui-monospace, monospace;
  line-height: 1.6;
}
</style>
