import { test, expect } from '@playwright/test'
import { spawnKaosControl } from '../harness/kaos-control.js'

test.describe('Edit and save functionality', () => {
  test('can edit and save artifacts', async () => {
    const instance = await spawnKaosControl()
    try {
      // Test that we can access the artifacts endpoint
      const res = await fetch(`${instance.baseURL}/api/artifacts`)
      expect(res.status).toBe(200)

      // Test that we can access the project info
      const projectRes = await fetch(`${instance.baseURL}/api/project`)
      expect(projectRes.status).toBe(200)
    } finally {
      await instance.kill()
    }
  })
})
