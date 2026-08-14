// SPDX-License-Identifier: AGPL-3.0-or-later

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { generateIdea } from '@/api/ideaChat'
import { api } from '@/api/client'
import { ApiError } from '@/api/client'
import type { IdeaGenerateResponse } from '@/types/api'

function slugify(text: string): string {
  const slug = text
    .toLowerCase()
    .replace(/[^a-z0-9\s]/g, ' ')
    .trim()
    .replace(/\s+/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '')
    .slice(0, 60)
    .replace(/-+$/, '')
  return slug && /^[a-z0-9]/.test(slug) ? slug : `doc-${Date.now()}`
}

function deriveTitle(text: string): string {
  const first = text.split(/[\n.!?]/)[0].trim()
  return (first || text).slice(0, 120)
}

// Error codes that mean "generation is unavailable because of project
// config", as opposed to a transient/unknown failure — see
// lifecycle/frontend-plans/defect-generate-missing-template-3-fe.md Milestone 1.
const CONFIG_ERROR_CODES = new Set(['template_unavailable', 'config_error'])

export type BrainDumpErrorKind = 'config' | 'generic'

const GENERATION_LABEL: Record<'idea' | 'defect' | 'doc', string> = {
  idea: 'Idea',
  defect: 'Defect',
  doc: 'Doc',
}

const GENERATION_TEMPLATE_KEY: Record<'idea' | 'defect' | 'doc', string> = {
  idea: 'idea-generate',
  defect: 'defect-generate',
  doc: 'doc-generate',
}

export type BrainDumpPhase = 'input' | 'generating' | 'preview' | 'editing'

export const useBrainDumpStore = defineStore('brainDump', () => {
  const input = ref('')
  const artifactType = ref<'idea' | 'defect' | 'doc'>('idea')
  const phase = ref<BrainDumpPhase>('input')
  const error = ref<string | null>(null)
  const errorKind = ref<BrainDumpErrorKind | null>(null)
  const proposal = ref<IdeaGenerateResponse | null>(null)
  const editedBody = ref<string | null>(null)

  // Maps a caught error to a display message and a kind, so the UI can branch
  // on a stable classification (config vs. generic) rather than message text.
  function applyError(e: unknown): void {
    if (e instanceof ApiError && CONFIG_ERROR_CODES.has(e.code)) {
      const type = artifactType.value
      error.value = `${GENERATION_LABEL[type]} generation isn't configured for this project. Ask an admin to add a \`${GENERATION_TEMPLATE_KEY[type]}\` template to the idea-capture agent, or create the ${type} manually.`
      errorKind.value = 'config'
    } else if (e instanceof ApiError) {
      error.value = e.message
      errorKind.value = 'generic'
    } else {
      error.value = 'Something went wrong — please try again.'
      errorKind.value = 'generic'
    }
  }

  const canSubmit = computed(
    () => input.value.trim().length > 0 && phase.value === 'input',
  )

  async function generate(
    project: string,
    opts?: { sourceLineage?: string; sourcePath?: string },
  ): Promise<void> {
    error.value = null
    errorKind.value = null
    phase.value = 'generating'
    try {
      const res = await generateIdea(
        project,
        input.value,
        artifactType.value,
        opts?.sourceLineage,
        opts?.sourcePath,
      )
      proposal.value = res
      phase.value = 'preview'
    } catch (e: unknown) {
      applyError(e)
      phase.value = 'input'
    }
  }

  async function acceptProposal(project: string): Promise<string | null> {
    if (!proposal.value) return null
    const p = proposal.value
    // For doc type always use 'docs' stage; otherwise derive from target_dir
    const stage =
      artifactType.value === 'doc'
        ? 'docs'
        : p.target_dir.replace(/^lifecycle\//, '')
    try {
      const res = await api.post<{ artifact: { path: string } }>(
        `/p/${encodeURIComponent(project)}/artifacts`,
        {
          stage,
          slug: p.slug,
          frontmatter: p.frontmatter,
          body: p.body,
        },
      )
      return res.artifact.path
    } catch (e: unknown) {
      if (e instanceof ApiError) {
        error.value = e.message
      } else {
        error.value = 'Something went wrong — please try again.'
      }
      return null
    }
  }

  function startEdit(): void {
    if (!proposal.value) return
    editedBody.value = proposal.value.body
    phase.value = 'editing'
  }

  function applyEdit(): boolean {
    if (!editedBody.value?.trim()) {
      error.value = 'Body cannot be empty.'
      return false
    }
    if (proposal.value) {
      proposal.value = { ...proposal.value, body: editedBody.value }
    }
    editedBody.value = null
    error.value = null
    phase.value = 'preview'
    return true
  }

  async function createDoc(
    project: string,
    opts?: { sourceLineage?: string; sourcePath?: string },
  ): Promise<string | null> {
    const raw = input.value.trim()
    if (!raw) return null
    error.value = null
    errorKind.value = null
    phase.value = 'generating'
    try {
      const slug = opts?.sourceLineage ?? slugify(raw)
      const title = deriveTitle(raw)
      const frontmatter: Record<string, unknown> = {
        title,
        type: 'doc',
        status: 'raw',
        lineage: slug,
      }
      if (opts?.sourcePath) frontmatter.parent = opts.sourcePath
      const res = await api.post<{ artifact: { path: string } }>(
        `/p/${encodeURIComponent(project)}/artifacts`,
        { stage: 'docs', slug, frontmatter, body: raw },
      )
      return res.artifact.path
    } catch (e: unknown) {
      applyError(e)
      phase.value = 'input'
      return null
    }
  }

  // Manual escape hatch for when defect generation is unavailable (config
  // error) — writes a minimal defect artifact directly from the raw input,
  // bypassing the generation step entirely.
  async function createDefectManually(project: string): Promise<string | null> {
    const raw = input.value.trim()
    if (!raw) return null
    error.value = null
    errorKind.value = null
    phase.value = 'generating'
    try {
      const slug = slugify(raw)
      const title = deriveTitle(raw)
      const frontmatter: Record<string, unknown> = {
        title,
        type: 'defect',
        status: 'raw',
        lineage: slug,
        labels: ['defect'],
      }
      const res = await api.post<{ artifact: { path: string } }>(
        `/p/${encodeURIComponent(project)}/artifacts`,
        { stage: 'defects', slug, frontmatter, body: raw },
      )
      return res.artifact.path
    } catch (e: unknown) {
      applyError(e)
      phase.value = 'input'
      return null
    }
  }

  function discard(): void {
    input.value = ''
    artifactType.value = 'idea'
    phase.value = 'input'
    error.value = null
    errorKind.value = null
    proposal.value = null
    editedBody.value = null
  }

  function reset(): void {
    discard()
  }

  return {
    input,
    artifactType,
    phase,
    error,
    errorKind,
    proposal,
    editedBody,
    canSubmit,
    generate,
    acceptProposal,
    createDoc,
    createDefectManually,
    startEdit,
    applyEdit,
    discard,
    reset,
  }
})
