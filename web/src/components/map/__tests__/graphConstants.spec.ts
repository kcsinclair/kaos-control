// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useGraphTheme } from '../graphConstants'
import { useThemeStore } from '@/stores/theme'

// Frontend plan: lifecycle/frontend-plans/architectural-artefacts-4-fe.md — Milestone 1
//
// architecture/tech-stack/adr must render with a defined (non-fallback) colour in both
// the dark and light node palettes.

const NEW_TYPES = ['architecture', 'tech-stack', 'adr']

describe('graphConstants — architectural artefact node colours', () => {
  it('defines a colour for each new type in the dark palette', () => {
    setActivePinia(createPinia())
    const themeStore = useThemeStore()
    themeStore.setTheme('dark')
    const { palette } = useGraphTheme()

    for (const type of NEW_TYPES) {
      expect(palette.value.nodeColors[type]).toBeDefined()
      expect(palette.value.nodeColors[type]).toMatch(/^#[0-9a-f]{6}$/i)
    }
  })

  it('defines a colour for each new type in the light palette', () => {
    setActivePinia(createPinia())
    const themeStore = useThemeStore()
    themeStore.setTheme('light')
    const { palette } = useGraphTheme()

    for (const type of NEW_TYPES) {
      expect(palette.value.nodeColors[type]).toBeDefined()
      expect(palette.value.nodeColors[type]).toMatch(/^#[0-9a-f]{6}$/i)
    }
  })

  it('gives each new type a colour distinct from the existing plan/idea/doc families', () => {
    setActivePinia(createPinia())
    const themeStore = useThemeStore()
    themeStore.setTheme('dark')
    const { palette } = useGraphTheme()
    const { nodeColors } = palette.value

    for (const type of NEW_TYPES) {
      expect(nodeColors[type]).not.toBe(nodeColors.idea)
      expect(nodeColors[type]).not.toBe(nodeColors['plan-backend'])
      expect(nodeColors[type]).not.toBe(nodeColors['plan-frontend'])
      expect(nodeColors[type]).not.toBe(nodeColors['plan-test'])
      expect(nodeColors[type]).not.toBe(nodeColors.doc)
    }
  })
})
