// Positive case: `count` is declared with `const` since it is never
// reassigned.
export function fixed(): number {
  const count = 5
  return count
}
