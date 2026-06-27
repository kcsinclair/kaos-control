import { test, expect } from '@playwright/test'
import { spawnKaosControl } from '../harness/kaos-control.js'

test.describe('Harness smoke', () => {
  test('spawn, GET /api/health, kill', async () => {
    const instance = await spawnKaosControl()
    try {
      const res = await fetch(`${instance.baseURL}/api/health`)
      expect(res.status).toBe(200)
    } finally {
      await instance.kill()
    }
  })
})

test.describe('Basic API endpoints', () => {
  test('can access root endpoint', async () => {
    const instance = await spawnKaosControl()
    try {
      const res = await fetch(`${instance.baseURL}/`)
      expect(res.status).toBe(200)
    } finally {
      await instance.kill()
    }
  })
})