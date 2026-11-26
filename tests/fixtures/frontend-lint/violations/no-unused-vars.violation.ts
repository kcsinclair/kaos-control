// Violates @typescript-eslint/no-unused-vars: `leftover` is assigned but
// never read, and is not prefixed with `_` so the override does not exempt it.
export function greet(name: string): string {
  const leftover = 'never read'
  return `hello ${name}`
}
