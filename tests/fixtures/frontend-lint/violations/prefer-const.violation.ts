// Violates prefer-const: `count` is declared with `let` but is never
// reassigned after initialization.
export function unreassigned(): number {
  let count = 5
  return count
}
