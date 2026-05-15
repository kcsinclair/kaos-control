export interface UserCredentials {
  email: string
  password: string
  name?: string
}

/** Bootstrap the first admin user via the unauthenticated POST /api/admin/users endpoint. */
export async function bootstrapUser(
  baseURL: string,
  creds: UserCredentials,
): Promise<{ id: string }> {
  const res = await fetch(`${baseURL}/api/admin/users`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      email: creds.email,
      password: creds.password,
      name: creds.name ?? 'Test Admin',
      admin: true,
    }),
  })
  if (!res.ok) {
    const body = await res.text()
    throw new Error(`bootstrapUser failed (${res.status}): ${body}`)
  }
  const data = (await res.json()) as { id?: string; user?: { id: string } }
  return { id: (data.id ?? data.user?.id ?? 'unknown') }
}

import type { Page } from '@playwright/test'

/** Drive the SPA login form to acquire session cookies in the browser context. */
export async function loginPage(
  page: Page,
  baseURL: string,
  creds: UserCredentials,
): Promise<void> {
  await page.goto(`${baseURL}/login`)
  await page.fill('#email', creds.email)
  await page.fill('#password', creds.password)
  await page.click('button[type="submit"]')
  // Wait for redirect away from /login (successful auth redirects to /projects or the redirect query param)
  await page.waitForURL((url) => !url.pathname.startsWith('/login'), { timeout: 10_000 })
}
