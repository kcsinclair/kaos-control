<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { basicSetup, EditorView } from 'codemirror'
import { keymap } from '@codemirror/view'
import { markdown } from '@codemirror/lang-markdown'
import { oneDark } from '@codemirror/theme-one-dark'
import { defaultKeymap } from '@codemirror/commands'

const props = defineProps<{ modelValue: string }>()
const emit = defineEmits<{
  'update:modelValue': [value: string]
  save: []
}>()

const container = ref<HTMLElement>()
let view: EditorView | null = null
let ignoreNextUpdate = false

onMounted(() => {
  if (!container.value) return
  view = new EditorView({
    doc: props.modelValue,
    extensions: [
      basicSetup,
      markdown(),
      oneDark,
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
  <div ref="container" class="md-editor" />
</template>

<style scoped>
.md-editor {
  height: 100%;
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
