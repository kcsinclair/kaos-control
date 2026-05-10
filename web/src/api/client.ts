// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ApiErrorBody } from '@/types/api'

export class ApiError extends Error {
  constructor(
    public readonly code: string,
    message: string,
    public readonly status: number,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

function getCsrfToken(): string {
  const match = document.cookie.match(/(?:^|;\s*)kc_csrf=([^;]+)/)
  return match ? decodeURIComponent(match[1]) : ''
}

// buildUrl returns an absolute URL for the given API path. Browser fetch
// resolves relative URLs against window.location, but Node's undici (used by
// vitest) requires an absolute URL — so build one explicitly. In production,
// origin is always the page origin; in tests, fall back to a placeholder so
// URL parsing succeeds even when happy-dom's window.location is 'about:blank'.
function buildUrl(path: string): string {
  const origin =
    typeof window !== 'undefined' &&
    window.location?.origin &&
    window.location.origin !== 'null'
      ? window.location.origin
      : 'http://localhost'
  return `${origin}/api${path}`
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = {}

  if (body !== undefined) {
    headers['Content-Type'] = 'application/json'
  }
  // CSRF double-submit: send the token only on mutating methods and only when
  // the cookie is present (unauthenticated state has no cookie — the 401
  // interceptor below handles that case instead of failing on a CSRF error).
  if (method !== 'GET' && method !== 'HEAD' && method !== 'OPTIONS') {
    const csrf = getCsrfToken()
    if (csrf) headers['X-CSRF-Token'] = csrf
  }

  const res = await fetch(buildUrl(path), {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
    credentials: 'same-origin',
  })

  if (res.status === 204) return undefined as T

  const data = await res.json().catch(() => null)

  if (!res.ok) {
    if (res.status === 401) {
      // Dynamic imports avoid the circular dependency:
      //   router → stores/auth → api/auth → api/client → router
      const [{ default: router }, { useAuthStore }] = await Promise.all([
        import('@/router'),
        import('@/stores/auth'),
      ])
      if (router.currentRoute.value.path !== '/login') {
        useAuthStore().clearSession()
        const currentPath = router.currentRoute.value.fullPath
        await router.push({ path: '/login', query: { redirect: currentPath, expired: '1' } })
      }
    }
    const err: ApiErrorBody = data?.error ?? { code: 'unknown', message: res.statusText }
    throw new ApiError(err.code, err.message, res.status)
  }

  return data as T
}

async function getText(path: string): Promise<string> {
  const res = await fetch(buildUrl(path), {
    method: 'GET',
    credentials: 'same-origin',
  })

  if (!res.ok) {
    const data = await res.json().catch(() => null)
    const err: ApiErrorBody = data?.error ?? { code: 'unknown', message: res.statusText }
    throw new ApiError(err.code, err.message, res.status)
  }

  return res.text()
}

export const api = {
  get: <T>(path: string) => request<T>('GET', path),
  getText: (path: string) => getText(path),
  post: <T>(path: string, body?: unknown) => request<T>('POST', path, body),
  put: <T>(path: string, body?: unknown) => request<T>('PUT', path, body),
  patch: <T>(path: string, body?: unknown) => request<T>('PATCH', path, body),
  delete: <T>(path: string) => request<T>('DELETE', path),
}
