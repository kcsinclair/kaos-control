// Positive case for the tests/web ergonomics override: `any` is routinely
// needed to type mock objects and spy return values, and is turned off by
// tests/web/eslint.config.js's `tests/ergonomics-overrides` block (unlike
// the production web/eslint.config.js, where it stays on).
function makeMockApiClient(): unknown {
  const mock: any = {
    fetch: (): any => ({ status: 200, body: {} }),
  }
  return mock
}

export function useMock(): unknown {
  return makeMockApiClient()
}
