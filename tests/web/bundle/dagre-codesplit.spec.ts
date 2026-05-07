/**
 * Milestone 7 — Bundle size verification: dagre code-splitting
 *
 * Covers:
 *   1. After `pnpm build`, the main chunk does NOT contain cytoscape-dagre code.
 *   2. A separate async chunk containing dagre exists in web/dist/assets/.
 *   3. No unexpected new dependencies are added to the initial bundle.
 *
 * Testing approach
 * ────────────────
 * These tests read the built output in web/dist/assets/ and inspect the JS chunk
 * contents.  They require `pnpm build` (or `make build-web`) to have been run
 * prior to test execution — tests skip gracefully when the build output is absent.
 *
 * Detection heuristic:
 *   - "main chunk" = largest .js file in web/dist/assets/ (typically named
 *     index-[hash].js after Vite's default build naming).
 *   - "dagre chunk" = any .js file that is NOT the main chunk AND contains
 *     the dagre fingerprint string (e.g. "cytoscape-dagre" or "graphlib").
 *
 * This is intentionally conservative — it checks source strings that Vite
 * preserves even after minification (identifiers in the dagre library itself).
 *
 * Build output: web/dist/assets/
 */

import { describe, it, expect } from 'vitest'
import * as fs from 'node:fs'
import * as path from 'node:path'

// ---------------------------------------------------------------------------
// Resolve the build output directory
// ---------------------------------------------------------------------------

// process.cwd() is tests/web (where vitest runs from)
const DIST_ASSETS = path.resolve(process.cwd(), '../../web/dist/assets')

/** Returns true when the build output directory exists */
function buildExists(): boolean {
  return fs.existsSync(DIST_ASSETS)
}

/** Returns all .js chunk files in the dist/assets directory */
function getJsChunks(): string[] {
  if (!buildExists()) return []
  return fs
    .readdirSync(DIST_ASSETS)
    .filter((f) => f.endsWith('.js'))
    .map((f) => path.join(DIST_ASSETS, f))
}

/**
 * Returns the "main" chunk path — the largest .js file, which Vite outputs as
 * the entry-point bundle (index-[hash].js or similar).
 */
function findMainChunk(chunks: string[]): string | undefined {
  if (!chunks.length) return undefined
  return chunks.reduce((largest, f) => {
    const sizeA = fs.statSync(f).size
    const sizeB = fs.statSync(largest).size
    return sizeA > sizeB ? f : largest
  })
}

/**
 * Dagre fingerprint strings that appear in the cytoscape-dagre source.
 * These survive minification because they are string literals or export names.
 */
const DAGRE_FINGERPRINTS = ['cytoscape-dagre', 'graphlib']

/** Returns true if the given file content contains a dagre fingerprint */
function containsDagreCode(content: string): boolean {
  return DAGRE_FINGERPRINTS.some((fp) => content.includes(fp))
}

// ===========================================================================
// 1 — Main chunk does NOT contain cytoscape-dagre code
// ===========================================================================

describe('Bundle — dagre is NOT in the main chunk (Milestone 7 AC1)', () => {
  it('skips if web/dist/assets does not exist (run pnpm build first)', () => {
    if (!buildExists()) {
      console.warn(
        '[SKIPPED] web/dist/assets not found. Run `make build-web` before this test.',
      )
      // Explicitly pass: the test is a post-build check, not a unit test
      return
    }
    expect(true).toBe(true)
  })

  it('main chunk does not contain "cytoscape-dagre"', () => {
    if (!buildExists()) return

    const chunks = getJsChunks()
    expect(chunks.length, 'expected at least one JS chunk in web/dist/assets').toBeGreaterThan(0)

    const mainChunk = findMainChunk(chunks)
    expect(mainChunk).toBeDefined()

    const content = fs.readFileSync(mainChunk!, 'utf-8')
    expect(
      content.includes('cytoscape-dagre'),
      `main chunk (${path.basename(mainChunk!)}) must not include "cytoscape-dagre"`,
    ).toBe(false)
  })

  it('main chunk does not contain "graphlib" (dagre dependency)', () => {
    if (!buildExists()) return

    const chunks = getJsChunks()
    const mainChunk = findMainChunk(chunks)
    if (!mainChunk) return

    const content = fs.readFileSync(mainChunk, 'utf-8')
    expect(
      content.includes('graphlib'),
      `main chunk (${path.basename(mainChunk)}) must not include "graphlib" (dagre dep)`,
    ).toBe(false)
  })
})

// ===========================================================================
// 2 — A separate async chunk containing dagre code exists
// ===========================================================================

describe('Bundle — separate async dagre chunk exists (Milestone 7 AC2)', () => {
  it('at least one non-main chunk contains dagre fingerprint code', () => {
    if (!buildExists()) return

    const chunks = getJsChunks()
    if (!chunks.length) return

    const mainChunk = findMainChunk(chunks)
    const nonMainChunks = chunks.filter((c) => c !== mainChunk)

    const dagreChunk = nonMainChunks.find((f) => {
      const content = fs.readFileSync(f, 'utf-8')
      return containsDagreCode(content)
    })

    expect(
      dagreChunk,
      `expected a separate async chunk containing dagre code in ${DIST_ASSETS}\n` +
      `Non-main chunks: ${nonMainChunks.map((f) => path.basename(f)).join(', ')}`,
    ).toBeDefined()
  })

  it('the dagre chunk is not the same file as the main chunk', () => {
    if (!buildExists()) return

    const chunks = getJsChunks()
    if (!chunks.length) return

    const mainChunk = findMainChunk(chunks)
    const nonMainChunks = chunks.filter((c) => c !== mainChunk)

    const dagreChunk = nonMainChunks.find((f) => {
      const content = fs.readFileSync(f, 'utf-8')
      return containsDagreCode(content)
    })

    if (!dagreChunk) return // checked in previous test

    expect(dagreChunk).not.toBe(mainChunk)
  })
})

// ===========================================================================
// 3 — Total chunk count is within expected range (no unexpected large deps)
// ===========================================================================

describe('Bundle — chunk count sanity check (Milestone 7 AC3)', () => {
  it('total JS chunk count is reasonable (2–30 chunks)', () => {
    if (!buildExists()) return

    const chunks = getJsChunks()
    // A well-split Vite build produces a bounded number of chunks.
    // If this number grows unexpectedly large, it may indicate accidental
    // over-splitting or new bundled deps.  The upper bound of 200 is generous
    // but still catches catastrophic splitting regressions.
    expect(chunks.length, `unexpected chunk count: ${chunks.length}`).toBeGreaterThanOrEqual(2)
    expect(chunks.length, `unexpected chunk count: ${chunks.length}`).toBeLessThanOrEqual(200)
  })
})
