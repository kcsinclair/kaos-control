// SPDX-License-Identifier: AGPL-3.0-or-later

import { onMounted, onUnmounted } from 'vue'
import type { Ref } from 'vue'

/** Component instance type returned by a TextFilter ref (via defineExpose). */
interface TextFilterInstance {
  focus: () => void
}

/**
 * Registers a document-level keydown listener for `/`.
 * When `/` is pressed and no input/textarea/contenteditable element is focused,
 * calls focus() on the provided TextFilter ref.
 * Cleans up on unmount.
 */
export function useTextFilterShortcut(filterRef: Ref<TextFilterInstance | null>) {
  function onKeydown(e: KeyboardEvent) {
    if (e.key !== '/') return
    // Do not steal focus from other text inputs or content-editable regions
    const active = document.activeElement as HTMLElement | null
    if (!active) return
    const tag = active.tagName
    if (tag === 'INPUT' || tag === 'TEXTAREA' || active.isContentEditable) return
    e.preventDefault()
    filterRef.value?.focus()
  }

  onMounted(() => document.addEventListener('keydown', onKeydown))
  onUnmounted(() => document.removeEventListener('keydown', onKeydown))
}
