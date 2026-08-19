// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import RiceEditor from '../RiceEditor.vue'
import type { ArtifactFrontmatter } from '@/types/api'

const patchRiceMock = vi.fn(() => Promise.resolve({ artifact: {} }))
vi.mock('@/api/artifacts', () => ({
  patchRice: (...args: unknown[]) => patchRiceMock(...args),
}))

function mountEditor(fm: Partial<ArtifactFrontmatter> = {}) {
  return mount(RiceEditor, {
    props: {
      project: 'p',
      path: 'lifecycle/defects/x.md',
      type: 'defect',
      frontmatter: fm as ArtifactFrontmatter,
    },
    attachTo: document.body,
  })
}

describe('RiceEditor', () => {
  beforeEach(() => patchRiceMock.mockClear())

  it('saves the default values on a fresh artifact', async () => {
    const w = mountEditor({})
    await w.find('button.rice-badge--interactive').trigger('click')
    await w.find('button.rice-btn-save').trigger('click')
    expect(patchRiceMock).toHaveBeenCalledWith(
      'p',
      'lifecycle/defects/x.md',
      expect.objectContaining({ rice_reach: 100, rice_impact: 0.25, rice_confidence: 25, rice_effort: 1 }),
    )
  })

  it('saves an edited numeric value', async () => {
    const w = mountEditor({})
    await w.find('button.rice-badge--interactive').trigger('click')
    const numbers = w.findAll('input[type="number"]') // reach, confidence, effort (impact is a select)
    await numbers[0].setValue('200') // reach
    await w.find('button.rice-btn-save').trigger('click')
    expect(patchRiceMock).toHaveBeenCalledWith(
      'p',
      'lifecycle/defects/x.md',
      expect.objectContaining({ rice_reach: 200 }),
    )
  })

  it('saves an edited decimal effort on an already-scored artifact', async () => {
    const w = mountEditor({ rice_reach: 100, rice_impact: 0.25, rice_confidence: 25, rice_effort: 1 })
    await w.find('button.rice-badge--interactive').trigger('click')
    const numbers = w.findAll('input[type="number"]') // reach, confidence, effort
    await numbers[2].setValue('0.1') // effort
    await w.find('button.rice-btn-save').trigger('click')
    expect(patchRiceMock).toHaveBeenCalledWith(
      'p',
      'lifecycle/defects/x.md',
      expect.objectContaining({ rice_effort: 0.1 }),
    )
  })
})
