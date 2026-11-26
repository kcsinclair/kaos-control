// Violates @typescript-eslint/no-misused-promises: an async function is
// passed where a synchronous void-returning callback is expected.
async function loadData(): Promise<void> {}

export function registerHandler(onClick: () => void): void {
  onClick()
}

registerHandler(async () => {
  await loadData()
})
