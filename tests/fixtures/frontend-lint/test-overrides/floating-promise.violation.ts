// Negative case: the ergonomics overrides relax `no-explicit-any` for test
// files, but must NOT relax @typescript-eslint/no-floating-promises — an
// unawaited promise is still a real bug in a test file.
async function loadFixture(): Promise<void> {}

export function run(): void {
  loadFixture()
}
