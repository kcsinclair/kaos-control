// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 5 — RunFailureBanner local-model failure taxonomy tests
 * (local-model-operability: lifecycle/test-plans/local-model-operability-5-test.md)
 *
 * Covers all 9 structured failure_reason codes introduced by
 * local-model-operability (FR-3/FR-4), backed by
 * web/src/lib/failureReasons.ts:
 *   tools_unsupported, model_not_found, model_unloaded, endpoint_unreachable,
 *   context_window_exceeded, turn_token_ceiling, max_iterations_reached,
 *   auth_error, timeout
 *
 * Assertions per code:
 *   - Heading and explanatory body text render.
 *   - Numbered remediation steps render, with backtick spans as <code>.
 *   - Provider/model-related codes show links to /settings/providers and
 *     /agents (Agent Config), when denialCount is unset.
 *   - role="alert" is present on the root element.
 *   - No masked secret value ("***") leaks unexpectedly, and error_details
 *     containing a masked "***" value never appears verbatim as a real secret
 *     — the banner only ever surfaces what the backend already masked.
 *
 * The existing tests/web/RunFailureBanner.test.ts covers the two legacy
 * precheck-only codes (permission_mode_default, precheck_timeout) and generic
 * remediation/disclosure/accessibility behaviour; this file is additive.
 */

import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { createRouter, createMemoryHistory } from 'vue-router'
import RunFailureBanner from '../../web/src/components/agent/RunFailureBanner.vue'
import { FAILURE_REASON_INFO } from '../../web/src/lib/failureReasons'
import type { FailureReason } from '../../web/src/types/api'

const ALL_REASONS = Object.keys(FAILURE_REASON_INFO) as FailureReason[]

async function mountBanner(props: Record<string, unknown>) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/p/:project/:sub*', component: { template: '<div/>' } },
      { path: '/:pathMatch(.*)*', component: { template: '<div/>' } },
    ],
  })
  await router.push('/p/my-project/agents')
  await router.isReady()

  return mount(RunFailureBanner, {
    props,
    global: { plugins: [router] },
  })
}

describe('RunFailureBanner — local-model failure taxonomy', () => {
  it('covers all 9 structured failure reason codes with defined copy', () => {
    expect(ALL_REASONS.sort()).toEqual(
      [
        'tools_unsupported',
        'model_not_found',
        'model_unloaded',
        'endpoint_unreachable',
        'context_window_exceeded',
        'turn_token_ceiling',
        'max_iterations_reached',
        'auth_error',
        'timeout',
      ].sort(),
    )
  })

  describe.each(ALL_REASONS)('%s', (reason) => {
    it('renders the expected heading and body text', async () => {
      const wrapper = await mountBanner({ failureReason: reason })
      const info = FAILURE_REASON_INFO[reason]!
      expect(wrapper.text()).toContain(info.heading)
      expect(wrapper.text()).toContain(info.explanation(null, undefined))
    })

    it('renders role="alert" on the root element', async () => {
      const wrapper = await mountBanner({ failureReason: reason })
      expect(wrapper.attributes('role')).toBe('alert')
    })

    it('renders backend-computed remediation as numbered steps with inline code', async () => {
      const remediation = ['Do the first `thing`', 'Then do the second thing']
      const wrapper = await mountBanner({ failureReason: reason, remediation })
      const items = wrapper.findAll('.failure-banner__step')
      expect(items).toHaveLength(2)
      expect(items[0].find('code').exists()).toBe(true)
      expect(items[0].find('code').text()).toBe('thing')
      expect(items[1].text()).toContain('Then do the second thing')
    })

    it('falls back to the static remediation list when the backend sends none', async () => {
      const wrapper = await mountBanner({ failureReason: reason })
      const info = FAILURE_REASON_INFO[reason]!
      const items = wrapper.findAll('.failure-banner__step')
      expect(items).toHaveLength(info.remediation.length)
    })

    it('shows Provider Settings and Agent Config links', async () => {
      const wrapper = await mountBanner({ failureReason: reason })
      const links = wrapper.findAll('.failure-banner__link')
      const hrefs = links.map((l) => l.attributes('href'))
      expect(hrefs).toContain('/p/my-project/settings/providers')
      expect(hrefs).toContain('/p/my-project/agents')
    })

    it('does not show settings links when denialCount is set', async () => {
      const wrapper = await mountBanner({ failureReason: reason, denialCount: 2 })
      expect(wrapper.find('.failure-banner__links').exists()).toBe(false)
    })
  })

  describe('secret masking passthrough', () => {
    it('never renders a raw secret value, only the masked marker already sent by the backend', async () => {
      const wrapper = await mountBanner({
        failureReason: 'auth_error',
        errorDetails: { authorization: '***', api_key: '***' },
      })
      const text = wrapper.text()
      expect(text).not.toMatch(/sk-[a-zA-Z0-9]{10,}/)
      expect(text).not.toMatch(/Bearer [a-zA-Z0-9._-]{10,}/)
    })

    it('interpolates the model name from error_details without leaking other fields', async () => {
      const wrapper = await mountBanner({
        failureReason: 'model_not_found',
        errorDetails: { model: 'gemma-4-26b', api_key: '***' },
        providerName: 'local-llama',
      })
      expect(wrapper.text()).toContain('gemma-4-26b')
      expect(wrapper.text()).toContain('local-llama')
      expect(wrapper.text()).not.toContain('***')
    })
  })

  describe('context/token ceiling codes share copy', () => {
    it('context_window_exceeded and turn_token_ceiling render the same heading', async () => {
      const a = await mountBanner({ failureReason: 'context_window_exceeded' })
      const b = await mountBanner({ failureReason: 'turn_token_ceiling' })
      expect(a.text()).toContain(FAILURE_REASON_INFO.context_window_exceeded!.heading)
      expect(b.text()).toContain(FAILURE_REASON_INFO.turn_token_ceiling!.heading)
      expect(FAILURE_REASON_INFO.context_window_exceeded!.heading).toBe(
        FAILURE_REASON_INFO.turn_token_ceiling!.heading,
      )
    })
  })
})
