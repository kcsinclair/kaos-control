// SPDX-License-Identifier: AGPL-3.0-or-later

import { api } from './client'

export interface DocEntry {
  path: string
  title: string
  summary: string
  is_markdown: boolean
  sub_dir: string
}

export interface DocListResponse {
  docs: DocEntry[]
  docs_dir_present: boolean
}

export interface DocReadResponse {
  path: string
  body?: string
  body_base64?: string
  mime?: string
  file_sha: string
  is_markdown: boolean
}

function encodePath(relPath: string): string {
  return relPath.split('/').map(encodeURIComponent).join('/')
}

export function listDocs(project: string): Promise<DocListResponse> {
  return api.get<DocListResponse>(`/p/${encodeURIComponent(project)}/docs`)
}

export function getDoc(project: string, relPath: string): Promise<DocReadResponse> {
  return api.get<DocReadResponse>(`/p/${encodeURIComponent(project)}/docs/${encodePath(relPath)}`)
}

export function putDoc(
  project: string,
  relPath: string,
  body: string,
  expectedSha: string,
): Promise<{ file_sha: string }> {
  return api.put<{ file_sha: string }>(
    `/p/${encodeURIComponent(project)}/docs/${encodePath(relPath)}`,
    { body, expected_sha: expectedSha },
  )
}
