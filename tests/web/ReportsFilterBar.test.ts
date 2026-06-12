// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

import ReportsFilterBar from '../../web/src/components/reports/ReportsFilterBar.vue'
import type { AgentUsageFilter } from '../../web/src/types/api'

function defaultFilter(): AgentUsageFilter {
  return {
    bucket: 'day',
    agent: [],
    status: [],
    tz: 'UTC',
  }
}

beforeEach(() => {
  setActivePinia(createPinia())
})

describe('ReportsFilterBar', () => {
  it('preset Last 7d emits update with from/to approximately 7 days apart', async () => {
    const wrapper = mount(ReportsFilterBar, {
      props: { agents: [], filter: defaultFilter() },
    })
    await flushPromises()

    const buttons = wrapper.findAll('button.seg-btn')
    const last7dBtn = buttons.find((b) => b.text() === 'Last 7d')
    expect(last7dBtn).toBeDefined()
    await last7dBtn!.trigger('click')

    const events = wrapper.emitted('update')
    expect(events).toBeTruthy()
    expect(events!.length).toBeGreaterThan(0)

    const patch = events![events!.length - 1][0] as Partial<AgentUsageFilter>
    expect(patch.from).toBeTruthy()
    expect(patch.to).toBeTruthy()

    const diffMs = new Date(patch.to!).getTime() - new Date(patch.from!).getTime()
    const sevenDaysMs = 7 * 24 * 60 * 60 * 1000
    // Allow ±1 minute tolerance.
    expect(Math.abs(diffMs - sevenDaysMs)).toBeLessThan(60 * 1000)
  })

  it('switching to Custom reveals datetime-local inputs', async () => {
    const wrapper = mount(ReportsFilterBar, {
      props: { agents: [], filter: defaultFilter() },
    })
    await flushPromises()

    // Custom button should exist.
    const buttons = wrapper.findAll('button.seg-btn')
    const customBtn = buttons.find((b) => b.text() === 'Custom')
    expect(customBtn).toBeDefined()

    // Before clicking: no datetime-local inputs.
    expect(wrapper.findAll('input[type="datetime-local"]')).toHaveLength(0)

    // Set filter to 'custom' by providing non-matching from/to.
    // The detectPreset logic returns 'custom' when the diff doesn't match presets.
    // We'll pass a filter that already looks custom.
    await wrapper.setProps({
      filter: {
        ...defaultFilter(),
        from: '2025-01-01T00:00:00.000Z',
        to: '2025-06-15T12:00:00.000Z',
      },
    })
    await flushPromises()

    // Now datetime-local inputs should appear.
    const dateInputs = wrapper.findAll('input[type="datetime-local"]')
    expect(dateInputs.length).toBeGreaterThanOrEqual(2)
  })

  it('agent checkbox toggle emits update with agent array', async () => {
    const wrapper = mount(ReportsFilterBar, {
      props: {
        agents: ['qa', 'backend-developer'],
        filter: { ...defaultFilter(), agent: ['qa', 'backend-developer'] },
      },
    })
    await flushPromises()

    // Open the agent popover.
    const popoverTrigger = wrapper.find('button.popover-trigger')
    expect(popoverTrigger.exists()).toBe(true)
    await popoverTrigger.trigger('click')
    await flushPromises()

    // Find the qa checkbox and toggle it off.
    const checkboxes = wrapper.findAll('input[type="checkbox"]')
    const qaCheckbox = checkboxes.find((_, i) => {
      const label = wrapper.findAll('label.popover-item')[i]
      return label && label.text().includes('qa')
    })
    expect(qaCheckbox).toBeDefined()
    await qaCheckbox!.trigger('change')

    const events = wrapper.emitted('update')
    expect(events).toBeTruthy()
    const patch = events![events!.length - 1][0] as Partial<AgentUsageFilter>
    expect(patch.agent).toBeDefined()
    // qa should be removed from the agent list.
    expect(patch.agent).not.toContain('qa')
  })

  it('status chip toggle emits update with status array', async () => {
    const wrapper = mount(ReportsFilterBar, {
      props: { agents: [], filter: defaultFilter() },
    })
    await flushPromises()

    // Find the 'failed' status button.
    const buttons = wrapper.findAll('button.seg-btn')
    const failedBtn = buttons.find((b) => b.text() === 'failed')
    expect(failedBtn).toBeDefined()
    await failedBtn!.trigger('click')

    const events = wrapper.emitted('update')
    expect(events).toBeTruthy()
    const patch = events![events!.length - 1][0] as Partial<AgentUsageFilter>
    expect(patch.status).toContain('failed')
  })

  it('bucket segmented control emits update with bucket value', async () => {
    const wrapper = mount(ReportsFilterBar, {
      props: { agents: [], filter: defaultFilter() },
    })
    await flushPromises()

    // Find the 'Hour' bucket button.
    const buttons = wrapper.findAll('button.seg-btn')
    const hourBtn = buttons.find((b) => b.text() === 'Hour')
    expect(hourBtn).toBeDefined()
    await hourBtn!.trigger('click')

    const events = wrapper.emitted('update')
    expect(events).toBeTruthy()
    const patch = events![events!.length - 1][0] as Partial<AgentUsageFilter>
    expect(patch.bucket).toBe('hour')
  })

  it('controls are keyboard-navigable (no negative tabindex)', async () => {
    const wrapper = mount(ReportsFilterBar, {
      props: {
        agents: ['qa'],
        filter: defaultFilter(),
      },
    })
    await flushPromises()

    const buttons = wrapper.findAll('button')
    for (const btn of buttons) {
      const tabindex = btn.attributes('tabindex')
      if (tabindex !== undefined) {
        expect(parseInt(tabindex, 10)).toBeGreaterThanOrEqual(0)
      }
    }
  })
})
