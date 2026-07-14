// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, afterEach } from 'vitest'
import { getArtifact } from '../artifacts'
import type { ArtifactRow } from '@/types/api'

function mockFetch(body: unknown): ReturnType<typeof vi.fn> {
  const mock = vi.fn().mockResolvedValue(
    new Response(JSON.stringify(body), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    }),
  )
  vi.stubGlobal('fetch', mock)
  return mock
}

function makeRow(path: string, relPath: string): ArtifactRow {
  return {
    path,
    rel_path: relPath,
    slug: 'foo',
    lineage: 'foo',
    index: 0,
    stage: 'ideas',
    type: 'idea',
    status: 'draft',
    title: 'Foo',
    frontmatter: { title: 'Foo', type: 'idea', status: 'draft', lineage: 'foo' },
    mtime: '2026-01-01T00:00:00Z',
    created: '2026-01-01T00:00:00Z',
    agent_run_count: 0,
  }
}

describe('getArtifact — rel_path passthrough', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('deserialises a flat artifact with rel_path equal to its bare filename', async () => {
    mockFetch({
      artifact: makeRow('lifecycle/ideas/foo.md', 'foo.md'),
      body: '',
      body_html: '',
    })
    const data = await getArtifact('proj', 'lifecycle/ideas/foo.md')
    expect(data.artifact.rel_path).toBe('foo.md')
  })

  it('deserialises a nested artifact with rel_path equal to the sub-path', async () => {
    mockFetch({
      artifact: makeRow('lifecycle/ideas/sub/dir/foo.md', 'sub/dir/foo.md'),
      body: '',
      body_html: '',
    })
    const data = await getArtifact('proj', 'lifecycle/ideas/sub/dir/foo.md')
    expect(data.artifact.rel_path).toBe('sub/dir/foo.md')
  })
})
