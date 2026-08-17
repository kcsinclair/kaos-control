// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import PathChoiceStep from '../PathChoiceStep.vue'

// Frontend plan: lifecycle/frontend-plans/onboarding-architecture-selection-4-fe.md — Milestone 3
//
// The Browse vs Guided fork (FR-4). Also reused in a compact "inline" form as
// the persistent "show me everything anyway" control threaded through Guided.

describe('PathChoiceStep', () => {
  it('renders both path cards and emits choose with the picked path', async () => {
    const wrapper = mount(PathChoiceStep)

    const buttons = wrapper.findAll('button.path-card')
    expect(buttons).toHaveLength(2)

    await buttons[0].trigger('click')
    expect(wrapper.emitted('choose')?.[0]).toEqual(['guided'])

    await buttons[1].trigger('click')
    expect(wrapper.emitted('choose')?.[1]).toEqual(['browse'])
  })

  it('renders a single "show me everything anyway" control in inline variant', async () => {
    const wrapper = mount(PathChoiceStep, { props: { variant: 'inline' } })

    expect(wrapper.find('button.path-card').exists()).toBe(false)
    const btn = wrapper.find('button.show-everything-btn')
    expect(btn.exists()).toBe(true)

    await btn.trigger('click')
    expect(wrapper.emitted('choose')?.[0]).toEqual(['browse'])
  })
})
