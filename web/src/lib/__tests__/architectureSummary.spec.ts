// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect } from 'vitest'
import { extractSummarySection, summaryWithoutSection, resolveSummaryLinks } from '../architectureSummary'

const BODY = `# Architecture Summary

## Architecture-breaking requirements

| Requirement | Signal | How it's satisfied |
| --- | --- | --- |
| Must scale horizontally | scale | Modular monolith with stateless services |

## Selection Q&A

- **Q:** What's your team size?
  **A:** 3-5 engineers

## Links

- Architecture: [modular-monolith.md](modular-monolith.md)
- ADR: [adr-0001-datastore.md](decisions/adr-0001-datastore.md)
`

describe('extractSummarySection', () => {
  it('extracts a top-level section including its heading', () => {
    const section = extractSummarySection(BODY, 'Selection Q&A')
    expect(section).toContain('## Selection Q&A')
    expect(section).toContain('3-5 engineers')
    expect(section).not.toContain('Architecture-breaking requirements')
  })

  it('is case-insensitive on the heading text', () => {
    expect(extractSummarySection(BODY, 'selection q&a')).toContain('## Selection Q&A')
  })

  it('returns null when the heading is absent', () => {
    expect(extractSummarySection(BODY, 'Nonexistent Section')).toBeNull()
  })
})

describe('summaryWithoutSection', () => {
  it('removes only the named section, leaving the rest intact', () => {
    const rest = summaryWithoutSection(BODY, 'Architecture-breaking requirements')
    expect(rest).not.toContain('Must scale horizontally')
    expect(rest).toContain('## Selection Q&A')
    expect(rest).toContain('## Links')
  })

  it('returns the body unchanged when the heading is absent', () => {
    expect(summaryWithoutSection(BODY, 'Nonexistent')).toBe(BODY)
  })
})

describe('resolveSummaryLinks', () => {
  it('rewrites relative .md links to absolute artifact-editor paths', () => {
    const out = resolveSummaryLinks('[adr-0001-x.md](decisions/adr-0001-x.md)', 'myproj')
    expect(out).toBe('[adr-0001-x.md](/p/myproj/artifacts/lifecycle/architecture/decisions/adr-0001-x.md)')
  })

  it('leaves absolute and external links untouched', () => {
    expect(resolveSummaryLinks('[x](https://example.com/a.md)', 'p')).toBe('[x](https://example.com/a.md)')
    expect(resolveSummaryLinks('[x](/already/absolute.md)', 'p')).toBe('[x](/already/absolute.md)')
  })

  it('leaves non-.md links untouched', () => {
    expect(resolveSummaryLinks('[x](some-anchor)', 'p')).toBe('[x](some-anchor)')
  })
})
