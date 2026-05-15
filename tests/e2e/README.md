# kaos-control E2E Smoke Tests

Five Playwright flows that verify end-to-end contract integrity between the
Vue SPA and the Go backend.

## How to run

```sh
make test-e2e                         # build binary + run all flows
pnpm --dir tests/e2e test:ui          # interactive Playwright UI
pnpm --dir tests/e2e test:debug       # step-through debugger (PWDEBUG=1)
```

## How to add a flow

1. Create `flows/NN-name.spec.ts` (NN = next integer after 05).
2. Import from `../fixtures.js` and use the `test` and `expect` exports.
3. Use the `loggedInPage` fixture for flows that require auth.
4. Keep flows independent — each test spawns its own server instance.

Example skeleton:

```typescript
import { test, expect } from '../fixtures.js'

test.describe('Flow NN — My new flow', () => {
  test('does something useful', async ({ kctest, loggedInPage: page }) => {
    await page.goto(`${kctest.baseURL}/p/testproject/dashboard`)
    // ... assertions ...
  })
})
```

## How to debug a failing flow

1. Run `pnpm --dir tests/e2e test:ui` to open the Playwright test runner UI.
2. Open `playwright-report/index.html` for the HTML report with traces.
3. Inspect a trace: `pnpm exec playwright show-trace <trace.zip>`.
4. On failure, the harness prints the server's stdout/stderr to the test output.

## Test data fixtures

Fixture files live in `tests/e2e/fixtures/lifecycle/`. The standard seed
contains 10 ideas, 3 requirements, and 1 defect (14 tracked artifacts total).
Each test gets a fresh copy in a temp directory — mutations do not persist.
