// SPDX-License-Identifier: AGPL-3.0-or-later

// Small text-level helpers for rendering architecture-summary.md sections
// as-is (OQ-1 — no structured heading parsing: we split on the summary
// generator's stable "## Heading" lines and hand the raw markdown straight
// to MarkdownPreview, we never parse Q&A/requirements into typed records).
// Mirrors the section headings internal/architecture/summary.go writes.

/**
 * Returns the raw markdown of the top-level ("## Heading") section matching
 * `heading` (case-insensitive), including its own heading line, up to the
 * next top-level heading or end of body. Returns null if the heading isn't
 * present so callers can fall back to rendering the whole body (NFR-5).
 */
export function extractSummarySection(body: string, heading: string): string | null {
  const lines = body.split('\n')
  const target = heading.trim().toLowerCase()
  const startIdx = lines.findIndex((l) => /^##\s+/.test(l) && l.replace(/^##\s+/, '').trim().toLowerCase() === target)
  if (startIdx === -1) return null

  let endIdx = lines.length
  for (let i = startIdx + 1; i < lines.length; i++) {
    if (/^##\s+/.test(lines[i])) {
      endIdx = i
      break
    }
  }
  return lines.slice(startIdx, endIdx).join('\n').trim()
}

/** Returns `body` with the named top-level section removed, as-is otherwise. */
export function summaryWithoutSection(body: string, heading: string): string {
  const section = extractSummarySection(body, heading)
  if (!section) return body
  return body.replace(section, '').replace(/\n{3,}/g, '\n\n').trim()
}

// summary.go's summaryLink() renders links relative to architecture-summary.md's
// own location (lifecycle/architecture/), e.g. "[adr-0001-x.md](decisions/adr-0001-x.md)".
// Rewritten here to an absolute in-app artifact-editor path so click-through
// works from the overview route rather than resolving against the browser's
// current location (FR-4/FR-5).
const RELATIVE_MD_LINK_RE = /\]\(((?!https?:\/\/|\/|#)[^)\s]+\.md)\)/g

export function resolveSummaryLinks(markdown: string, project: string): string {
  return markdown.replace(RELATIVE_MD_LINK_RE, (_match, target: string) => {
    return `](/p/${encodeURIComponent(project)}/artifacts/lifecycle/architecture/${target})`
  })
}
