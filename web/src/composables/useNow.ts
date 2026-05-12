// SPDX-License-Identifier: AGPL-3.0-or-later

import { ref, onMounted, onUnmounted } from 'vue'
import type { Ref } from 'vue'

// useNow returns a Ref<Date> that ticks once per second.
// The interval is started on component mount and cleared on unmount so it
// never leaks across components — multiple concurrent callers each get their
// own independent tick, but the cost is trivial (one 1-second interval each).
export function useNow(): Ref<Date> {
  const now = ref(new Date())
  let timer: ReturnType<typeof setInterval> | null = null

  onMounted(() => {
    timer = setInterval(() => {
      now.value = new Date()
    }, 1_000)
  })

  onUnmounted(() => {
    if (timer !== null) {
      clearInterval(timer)
      timer = null
    }
  })

  return now
}
