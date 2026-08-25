// Positive case: the promise is explicitly discarded with `void`, which
// no-floating-promises accepts as an intentional fire-and-forget.
async function loadData(): Promise<void> {}

export function run(): void {
  void loadData()
}
