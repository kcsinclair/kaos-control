// SPDX-License-Identifier: AGPL-3.0-or-later

import { computed, onBeforeUnmount, onMounted, ref } from 'vue'

// Keep these in sync with --bp-mobile / --bp-tablet in web/src/styles/tokens.css.
const MOBILE_MAX = 640
const TABLET_MAX = 1024

/**
 * Reactive viewport-size flags, driven by `window.matchMedia` so subscribers
 * only re-render at the breakpoint transitions, not on every resize tick.
 * SSR-safe (returns desktop defaults when window is undefined).
 *
 * Usage:
 *   const { isMobile, isTablet, isDesktop } = useViewport()
 *   ...
 *   <Drawer v-if="isMobile" /> <Sidebar v-else />
 */
export function useViewport() {
  const isMobile = ref(false)
  const isTablet = ref(false)

  let mobileMq: MediaQueryList | null = null
  let tabletMq: MediaQueryList | null = null

  const onMobile = (e: MediaQueryListEvent | MediaQueryList) => {
    isMobile.value = e.matches
  }
  const onTablet = (e: MediaQueryListEvent | MediaQueryList) => {
    isTablet.value = e.matches
  }

  onMounted(() => {
    if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return
    mobileMq = window.matchMedia(`(max-width: ${MOBILE_MAX}px)`)
    tabletMq = window.matchMedia(`(max-width: ${TABLET_MAX}px)`)
    onMobile(mobileMq)
    onTablet(tabletMq)
    mobileMq.addEventListener('change', onMobile)
    tabletMq.addEventListener('change', onTablet)
  })

  onBeforeUnmount(() => {
    mobileMq?.removeEventListener('change', onMobile)
    tabletMq?.removeEventListener('change', onTablet)
  })

  // isDesktop = explicitly NOT tablet-or-narrower, so it stays false on tablets.
  const isDesktop = computed(() => !isTablet.value)
  // isTabletOnly excludes mobile so callers can target "between phone and desktop".
  const isTabletOnly = computed(() => isTablet.value && !isMobile.value)

  return { isMobile, isTablet, isTabletOnly, isDesktop }
}
