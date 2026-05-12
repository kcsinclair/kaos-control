<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { computed, ref } from 'vue'

const props = defineProps<{
  failureReason: string
  observedMode?: string | null
  remediation?: string[] | null
}>()

const disclosureOpen = ref(false)

const heading = computed(() => {
  if (props.failureReason === 'permission_mode_default') {
    return 'Claude Code is in default permission mode'
  }
  if (props.failureReason === 'precheck_timeout') {
    return 'Claude Code did not start within the expected time'
  }
  return `Run failed: ${props.failureReason}`
})

const bodyText = computed(() => {
  if (props.failureReason === 'permission_mode_default') {
    const observed = props.observedMode ? `\`${props.observedMode}\`` : 'an unexpected mode'
    return `kaos-control needs Claude Code to run in \`bypassPermissions\` mode, but the agent run reported ${observed}.`
  }
  if (props.failureReason === 'precheck_timeout') {
    return 'The agent process did not emit a startup event within the allowed window. Check that Claude Code is installed and the agent configuration is correct.'
  }
  return null
})

/**
 * Split a remediation string into segments so that backtick-delimited spans
 * can be rendered as <code> elements.
 */
function parseRemediationSegments(text: string): Array<{ code: boolean; text: string }> {
  const parts = text.split('`')
  return parts.map((part, i) => ({ code: i % 2 === 1, text: part }))
}
</script>

<template>
  <div class="failure-banner" role="alert">
    <div class="failure-banner__header">
      <span class="failure-banner__icon" aria-hidden="true">✕</span>
      <strong class="failure-banner__heading">{{ heading }}</strong>
    </div>

    <p v-if="bodyText" class="failure-banner__body">
      <template v-for="(seg, i) in parseRemediationSegments(bodyText)" :key="i">
        <code v-if="seg.code" class="failure-banner__inline-code">{{ seg.text }}</code>
        <template v-else>{{ seg.text }}</template>
      </template>
    </p>

    <ol v-if="remediation && remediation.length" class="failure-banner__steps">
      <li v-for="(step, idx) in remediation" :key="idx" class="failure-banner__step">
        <span class="failure-banner__step-num">{{ idx + 1 }}</span>
        <span class="failure-banner__step-text">
          <template v-for="(seg, i) in parseRemediationSegments(step)" :key="i">
            <code v-if="seg.code" class="failure-banner__inline-code">{{ seg.text }}</code>
            <template v-else>{{ seg.text }}</template>
          </template>
        </span>
      </li>
    </ol>

    <details class="failure-banner__disclosure" @toggle="disclosureOpen = ($event.target as HTMLDetailsElement).open">
      <summary class="failure-banner__summary">What does this mean?</summary>
      <p class="failure-banner__disclosure-body">
        kaos-control launches Claude Code agents with the
        <code class="failure-banner__inline-code">--permission-mode bypassPermissions</code>
        flag so that agents can read and write files without interactive prompts.
        When the flag is not respected — typically because a global Claude Code setting
        overrides it — the agent reports <code class="failure-banner__inline-code">default</code>
        mode at startup. kaos-control detects this via the
        <code class="failure-banner__inline-code">system/init</code> event and stops the run
        immediately rather than letting it proceed without the expected permissions scope.
        Follow the remediation steps above to configure Claude Code for automated use.
      </p>
    </details>
  </div>
</template>

<style scoped>
.failure-banner {
  border: 1px solid var(--color-error);
  background: var(--badge-blocked-bg);
  border-radius: var(--radius-md);
  padding: var(--space-4);
  margin-bottom: var(--space-4);
  color: var(--badge-blocked-text);
}

.failure-banner__header {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-bottom: var(--space-2);
}

.failure-banner__icon {
  font-size: var(--text-sm);
  font-weight: 700;
  color: var(--color-error);
  flex-shrink: 0;
}

.failure-banner__heading {
  font-size: var(--text-sm);
  font-weight: 600;
}

.failure-banner__body {
  font-size: var(--text-sm);
  margin: 0 0 var(--space-3) 0;
  line-height: 1.5;
}

.failure-banner__steps {
  margin: 0 0 var(--space-3) 0;
  padding-left: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.failure-banner__step {
  display: flex;
  align-items: baseline;
  gap: var(--space-2);
  font-size: var(--text-sm);
}

.failure-banner__step-num {
  flex-shrink: 0;
  width: 18px;
  height: 18px;
  background: var(--color-error);
  color: #fff;
  border-radius: 50%;
  font-size: 10px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  line-height: 1;
}

.failure-banner__step-text {
  line-height: 1.5;
}

.failure-banner__inline-code {
  font-family: monospace;
  font-size: 0.85em;
  background: rgba(0, 0, 0, 0.08);
  padding: 1px 4px;
  border-radius: var(--radius-sm);
}

.failure-banner__disclosure {
  margin-top: var(--space-2);
}

.failure-banner__summary {
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
  color: var(--color-error);
  user-select: none;
}

.failure-banner__summary:hover {
  text-decoration: underline;
}

.failure-banner__disclosure-body {
  font-size: var(--text-sm);
  margin: var(--space-2) 0 0 0;
  line-height: 1.5;
  color: var(--badge-blocked-text);
}
</style>
