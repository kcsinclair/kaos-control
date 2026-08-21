// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { project: 'testproject' } }),
  RouterLink: { props: ['to'], template: '<a><slot /></a>' },
}))

const listArtifacts = vi.fn()
vi.mock('@/api/artifacts', () => ({
  listArtifacts: (...args: unknown[]) => listArtifacts(...args),
}))

import FeaturesView from '../FeaturesView.vue'

function feature(title: string, fn: string, summary = '') {
  return {
    path: `lifecycle/features/${title.toLowerCase().replace(/\s+/g, '-')}.md`,
    slug: title.toLowerCase().replace(/\s+/g, '-'),
    type: 'feature',
    status: 'approved',
    title,
    frontmatter: { title, type: 'feature', status: 'approved', function: fn, summary },
  }
}

describe('FeaturesView', () => {
  beforeEach(() => listArtifacts.mockReset())

  it('groups feature names by function, with named groups before Other', async () => {
    listArtifacts.mockResolvedValue({
      items: [
        feature('Architecture Wizard', 'Architecture'),
        feature('Architecture Overview', 'Architecture'),
        feature('Agent Directives', 'Agents'),
        feature('Loose Feature', ''), // no function → "Other"
      ],
      total: 4,
    })
    const wrapper = mount(FeaturesView)
    await flushPromises()

    const groupTitles = wrapper.findAll('.group-title').map((n) => n.text())
    // "Agents" and "Architecture" sort alphabetically; "Other" sinks last.
    expect(groupTitles[0]).toContain('Agents')
    expect(groupTitles[1]).toContain('Architecture')
    expect(groupTitles[groupTitles.length - 1]).toContain('Other')
    // Architecture group has both its features.
    expect(wrapper.text()).toContain('Architecture Wizard')
    expect(wrapper.text()).toContain('Architecture Overview')
  })

  it('requests only feature-type artifacts', async () => {
    listArtifacts.mockResolvedValue({ items: [], total: 0 })
    mount(FeaturesView)
    await flushPromises()
    expect(listArtifacts).toHaveBeenCalledWith('testproject', { type: 'feature', limit: 500 })
  })

  it('filters by search text', async () => {
    listArtifacts.mockResolvedValue({
      items: [
        feature('Architecture Wizard', 'Architecture', 'guided Q&A'),
        feature('Agent Directives', 'Agents', 'AGENTS.md canonical'),
      ],
      total: 2,
    })
    const wrapper = mount(FeaturesView)
    await flushPromises()

    await wrapper.find('.search-input').setValue('directives')
    await flushPromises()
    expect(wrapper.text()).toContain('Agent Directives')
    expect(wrapper.text()).not.toContain('Architecture Wizard')
  })
})
