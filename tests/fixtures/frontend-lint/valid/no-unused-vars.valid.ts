// Positive case: `_ignored` is prefixed with `_`, so the
// argsIgnorePattern/varsIgnorePattern override permits it to stay unused.
export function greet(name: string, _ignored: string): string {
  return `hello ${name}`
}
