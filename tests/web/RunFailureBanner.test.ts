// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 2 — Unit tests for the RunFailureBanner component
 *
 * Covers:
 *   - Renders the correct heading for known precheck reason codes.
 *   - Includes the observed mode in the body text when present.
 *   - Renders each remediation step.
 *   - Renders a fallback heading for unknown reason codes.
 *
 * Component: web/src/components/agent/RunFailureBanner.vue
 * Props: failureReason (string), observedMode? (string|null), remediation? (string[]|null)
 */

import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import RunFailureBanner from '../../web/src/components/agent/RunFailureBanner.vue'

describe('RunFailureBanner', () => {
  describe('heading derivation', () => {
    it('shows the permission_mode_default heading', () => {
      const wrapper = mount(RunFailureBanner, {
        props: { failureReason: 'permission_mode_default' },
      })
      expect(wrapper.text()).toContain('Claude Code is in default permission mode')
    })

    it('shows the precheck_timeout heading', () => {
      const wrapper = mount(RunFailureBanner, {
        props: { failureReason: 'precheck_timeout' },
      })
      expect(wrapper.text()).toContain('Claude Code did not start within the expected time')
    })

    it('shows a fallback heading for an unknown reason code', () => {
      const wrapper = mount(RunFailureBanner, {
        props: { failureReason: 'some_unknown_reason' },
      })
      expect(wrapper.text()).toContain('Run failed: some_unknown_reason')
    })
  })

  describe('body text includes observed mode', () => {
    it('includes the observed mode when present', () => {
      const wrapper = mount(RunFailureBanner, {
        props: {
          failureReason: 'permission_mode_default',
          observedMode: 'default',
        },
      })
      expect(wrapper.text()).toContain('default')
      expect(wrapper.text()).toContain('bypassPermissions')
    })

    it('renders without observed mode when not provided', () => {
      const wrapper = mount(RunFailureBanner, {
        props: { failureReason: 'permission_mode_default' },
      })
      // Should still render body text mentioning bypassPermissions
      expect(wrapper.text()).toContain('bypassPermissions')
    })
  })

  describe('remediation steps', () => {
    it('renders each remediation step', () => {
      const remediation = [
        'Open a terminal and run `claude config set permission-mode bypassPermissions`',
        'Restart the agent process',
        'Check the logs for further errors',
      ]
      const wrapper = mount(RunFailureBanner, {
        props: {
          failureReason: 'permission_mode_default',
          remediation,
        },
      })
      const items = wrapper.findAll('.failure-banner__step')
      expect(items).toHaveLength(3)
      expect(items[0].text()).toContain('claude config set permission-mode bypassPermissions')
      expect(items[1].text()).toContain('Restart the agent process')
      expect(items[2].text()).toContain('Check the logs for further errors')
    })

    it('renders backtick-delimited parts as code elements', () => {
      const wrapper = mount(RunFailureBanner, {
        props: {
          failureReason: 'permission_mode_default',
          remediation: ['Run `claude --version` to check'],
        },
      })
      const code = wrapper.find('.failure-banner__step code')
      expect(code.exists()).toBe(true)
      expect(code.text()).toBe('claude --version')
    })

    it('renders nothing when remediation is null', () => {
      const wrapper = mount(RunFailureBanner, {
        props: {
          failureReason: 'permission_mode_default',
          remediation: null,
        },
      })
      expect(wrapper.find('.failure-banner__steps').exists()).toBe(false)
    })

    it('renders nothing when remediation is empty', () => {
      const wrapper = mount(RunFailureBanner, {
        props: {
          failureReason: 'permission_mode_default',
          remediation: [],
        },
      })
      expect(wrapper.find('.failure-banner__steps').exists()).toBe(false)
    })
  })

  describe('disclosure', () => {
    it('renders the What does this mean? disclosure', () => {
      const wrapper = mount(RunFailureBanner, {
        props: { failureReason: 'permission_mode_default' },
      })
      expect(wrapper.find('details').exists()).toBe(true)
      expect(wrapper.find('summary').text()).toBe('What does this mean?')
    })
  })

  describe('accessibility', () => {
    it('has role=alert on the root element', () => {
      const wrapper = mount(RunFailureBanner, {
        props: { failureReason: 'permission_mode_default' },
      })
      expect(wrapper.attributes('role')).toBe('alert')
    })
  })
})
