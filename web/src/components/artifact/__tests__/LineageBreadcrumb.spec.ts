// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi } from 'vitest'
import { shallowMount } from '@vue/test-utils'

// Frontend plan: lifecycle/frontend-plans/idea-archiving-4-fe.md — Milestone 2
//
// Verifies the folder breadcrumb is built from rel_path: a flat artifact
// shows no folder segment beyond the stage, a nested one shows its folder(s).

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
}))

import LineageBreadcrumb from '../LineageBreadcrumb.vue'

describe('LineageBreadcrumb — folder segments from rel_path', () => {
  it('shows no folder segment for a flat artifact', () => {
    const wrapper = shallowMount(LineageBreadcrumb, {
      props: {
        project: 'proj',
        path: 'lifecycle/ideas/foo.md',
        relPath: 'foo.md',
        lineage: 'foo',
      },
    })
    const labels = wrapper.findAll('.crumb-intermediate, .crumb-current').map(n => n.text())
    expect(labels).toEqual(['lifecycle', 'ideas', 'foo.md'])
  })

  it('shows a folder segment for a nested artifact', () => {
    const wrapper = shallowMount(LineageBreadcrumb, {
      props: {
        project: 'proj',
        path: 'lifecycle/ideas/archive/foo.md',
        relPath: 'archive/foo.md',
        lineage: 'foo',
      },
    })
    const labels = wrapper.findAll('.crumb-intermediate, .crumb-current').map(n => n.text())
    expect(labels).toEqual(['lifecycle', 'ideas', 'archive', 'foo.md'])
  })

  it('shows deeply nested folder segments in order', () => {
    const wrapper = shallowMount(LineageBreadcrumb, {
      props: {
        project: 'proj',
        path: 'lifecycle/ideas/2026/q3/foo.md',
        relPath: '2026/q3/foo.md',
        lineage: 'foo',
      },
    })
    const labels = wrapper.findAll('.crumb-intermediate, .crumb-current').map(n => n.text())
    expect(labels).toEqual(['lifecycle', 'ideas', '2026', 'q3', 'foo.md'])
  })

  it('falls back to the full path when rel_path is empty (stale index row)', () => {
    const wrapper = shallowMount(LineageBreadcrumb, {
      props: {
        project: 'proj',
        path: 'lifecycle/ideas/foo.md',
        relPath: '',
        lineage: 'foo',
      },
    })
    const labels = wrapper.findAll('.crumb-intermediate, .crumb-current').map(n => n.text())
    expect(labels).toEqual(['lifecycle', 'ideas', 'foo.md'])
  })
})
