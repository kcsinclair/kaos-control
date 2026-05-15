import { test as base } from '@playwright/test'
import { spawnKaosControl, type KcTestInstance } from './harness/kaos-control.js'
import { bootstrapUser, loginPage, type UserCredentials } from './harness/auth.js'

export const ADMIN_CREDS: UserCredentials = {
  email: 'admin@kaos-e2e.local',
  password: 'TestPassword123!',
  name: 'Test Admin',
}

type KcFixtures = {
  kctest: KcTestInstance
  loggedInPage: import('@playwright/test').Page
}

export const test = base.extend<KcFixtures>({
  kctest: [
    async ({}, use) => {
      const instance = await spawnKaosControl()
      await bootstrapUser(instance.baseURL, ADMIN_CREDS)
      await use(instance)
      await instance.kill()
    },
    { scope: 'worker' },
  ],

  loggedInPage: async ({ kctest, page }, use) => {
    await loginPage(page, kctest.baseURL, ADMIN_CREDS)
    await use(page)
  },
})

export { expect } from '@playwright/test'
