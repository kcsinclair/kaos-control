// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import type { OverviewItem } from '@/types/api'
import ArchiveStrip from '../ArchiveStrip.vue'

// Frontend plan: lifecycle/frontend-plans/architecture-overview-view-4-fe.md
// — Milestone F5 (OQ-5): the archive strip is collapsed on first open and
// shows at most 10 items once expanded.

function makeItems(n: number): OverviewItem[] {
  return Array.from({ length: n }, (_, i) => ({
    path: `lifecycle/architecture/archive/item-${i}.md`,
    title: `Item ${i}`,
    status: 'approved',
    type: 'architecture',
    catalog_role: 'archive' as const,
  }))
}

const RouterLinkStub = { template: '<a><slot /></a>' }

function mountStrip(archive: OverviewItem[]) {
  return mount(ArchiveStrip, {
    props: { project: 'proj', archive },
    global: { stubs: { RouterLink: RouterLinkStub } },
  })
}

describe('ArchiveStrip', () => {
  it('is collapsed by default — no items rendered until expanded', () => {
    const wrapper = mountStrip(makeItems(3))
    expect(wrapper.find('[aria-expanded="false"]').exists()).toBe(true)
    expect(wrapper.findAll('[role="list"] li').length).toBe(0)
  })

  it('expands to show items, capped at 10, with a "+N more" note beyond that', async () => {
    const wrapper = mountStrip(makeItems(14))
    await wrapper.find('.archive-toggle').trigger('click')

    expect(wrapper.find('[aria-expanded="true"]').exists()).toBe(true)
    expect(wrapper.findAll('[role="list"] li').length).toBe(10)
    expect(wrapper.text()).toContain('and 4 more')
  })

  it('shows an absent state when there is nothing archived', async () => {
    const wrapper = mountStrip([])
    await wrapper.find('.archive-toggle').trigger('click')
    expect(wrapper.text()).toContain('Nothing archived yet.')
  })
})
