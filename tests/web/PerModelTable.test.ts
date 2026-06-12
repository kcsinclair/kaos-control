// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

import PerModelTable from '../../web/src/components/reports/PerModelTable.vue'
import type { AgentUsageGroupSummary } from '../../web/src/types/api'

type ModelRow = AgentUsageGroupSummary & { model: string }

function makeRow(model: string, cost: number, runCount = 5): ModelRow {
  return {
    model,
    run_count: runCount,
    success_count: runCount,
    failure_count: 0,
    metrics_unavailable_count: 0,
    total_cost_usd: cost,
    total_input_cost_usd: cost * 0.6,
    total_output_cost_usd: cost * 0.4,
    total_duration_ms: 5000,
    total_input_tokens: 500,
    total_cache_creation_tokens: 10,
    total_cache_read_tokens: 50,
    total_output_tokens: 250,
    mean_duration_ms: 1000,
    median_duration_ms: 900,
    p95_duration_ms: 1800,
    mean_cost_usd: cost / runCount,
    mean_output_tokens_per_second: 50.0,
    mean_ttft_ms: 120.0,
    p95_ttft_ms: 300.0,
    cache_hit_ratio: 0.4,
  }
}

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
})

describe('PerModelTable', () => {
  it('renders one row per model', async () => {
    const rows = [
      makeRow('claude-opus-4', 1.50),
      makeRow('claude-sonnet-4', 0.80),
      makeRow('claude-haiku-4', 0.20),
    ]
    const wrapper = mount(PerModelTable, { props: { rows } })
    await flushPromises()

    const trs = wrapper.findAll('tbody tr')
    expect(trs).toHaveLength(3)
  })

  it('default sort is total cost desc — highest cost first', async () => {
    const rows = [
      makeRow('cheap-model', 0.10),
      makeRow('expensive-model', 5.00),
      makeRow('mid-model', 1.00),
    ]
    const wrapper = mount(PerModelTable, { props: { rows } })
    await flushPromises()

    const trs = wrapper.findAll('tbody tr')
    // First row should be 'expensive-model' (highest total_cost_usd).
    expect(trs[0].text()).toContain('expensive-model')
  })

  it('column header click toggles sort direction', async () => {
    const rows = [makeRow('a', 1.0), makeRow('b', 2.0)]
    const wrapper = mount(PerModelTable, { props: { rows } })
    await flushPromises()

    // Find the "Total cost" header.
    const ths = wrapper.findAll('th.sortable-th')
    const costTh = ths.find((th) => th.text().includes('Total cost'))
    expect(costTh).toBeDefined()

    // Initially sorted desc by total cost.
    expect(costTh!.attributes('aria-sort')).toBe('descending')

    // First click on the same column → toggles to ascending.
    await costTh!.trigger('click')
    expect(costTh!.attributes('aria-sort')).toBe('ascending')

    // Second click → back to descending.
    await costTh!.trigger('click')
    expect(costTh!.attributes('aria-sort')).toBe('descending')
  })

  it('Export CSV produces correct header and data rows', async () => {
    // Mock URL.createObjectURL and capture the blob.
    const createObjectURL = vi.fn().mockReturnValue('blob:mock-url')
    const revokeObjectURL = vi.fn()
    Object.defineProperty(globalThis, 'URL', {
      writable: true,
      value: { createObjectURL, revokeObjectURL },
    })

    const capturedBlobs: Blob[] = []
    const origBlob = globalThis.Blob
    vi.stubGlobal('Blob', function (content: any[], opts: any) {
      const b = new origBlob(content, opts)
      capturedBlobs.push(b)
      return b
    })

    // Mock document.createElement to intercept the link click.
    const clickSpy = vi.fn()
    const origCreateElement = document.createElement.bind(document)
    vi.spyOn(document, 'createElement').mockImplementation((tag: string) => {
      if (tag === 'a') {
        const el = origCreateElement(tag)
        el.click = clickSpy
        return el
      }
      return origCreateElement(tag)
    })

    const rows = [makeRow('model-a', 1.0, 3), makeRow('model-b', 2.0, 5)]
    const wrapper = mount(PerModelTable, { props: { rows } })
    await flushPromises()

    const exportBtn = wrapper.find('button.btn-secondary')
    expect(exportBtn.exists()).toBe(true)
    await exportBtn.trigger('click')

    // createObjectURL should have been called with a Blob.
    expect(createObjectURL).toHaveBeenCalledOnce()
    expect(capturedBlobs.length).toBeGreaterThan(0)

    // Read the CSV text.
    const csvText = await capturedBlobs[capturedBlobs.length - 1].text()
    const lines = csvText.trim().split('\n')
    // Header row should contain "Model" and "Total cost".
    expect(lines[0]).toContain('Model')
    expect(lines[0]).toContain('Total cost')
    // Data rows should include both model names.
    const dataContent = lines.slice(1).join('\n')
    expect(dataContent).toContain('model-a')
    expect(dataContent).toContain('model-b')

    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })
})
