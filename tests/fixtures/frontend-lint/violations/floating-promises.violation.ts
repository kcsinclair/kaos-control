// Violates @typescript-eslint/no-floating-promises: the returned promise is
// neither awaited, chained with .catch/.then, nor discarded with `void`.
async function loadData(): Promise<void> {}

export function run(): void {
  loadData()
}
