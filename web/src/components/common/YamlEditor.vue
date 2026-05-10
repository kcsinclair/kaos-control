<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { onMounted, onUnmounted, watch } from 'vue'
import { basicSetup, EditorView } from 'codemirror'
import { yaml } from '@codemirror/lang-yaml'
import { oneDark } from '@codemirror/theme-one-dark'
import { EditorState } from '@codemirror/state'
import { ref } from 'vue'

const props = defineProps<{
  modelValue: string
  readonly?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
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
      yaml(),
      oneDark,
      EditorView.lineWrapping,
      EditorState.readOnly.of(props.readonly ?? false),
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
  <div ref="container" class="yaml-editor" />
</template>

<style scoped>
.yaml-editor {
  height: 100%;
  overflow: hidden;
  font-size: 13px;
}
.yaml-editor :deep(.cm-editor) {
  height: 100%;
}
.yaml-editor :deep(.cm-scroller) {
  overflow: auto;
  font-family: 'JetBrains Mono', 'Fira Code', ui-monospace, monospace;
  line-height: 1.6;
}
</style>
