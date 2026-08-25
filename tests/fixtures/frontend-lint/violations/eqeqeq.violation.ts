// Violates eqeqeq (rule set to "smart"): loose equality against a non-null
// literal is not the `== null` idiom the "smart" option exempts.
export function isOne(value: number): boolean {
  return value == 1
}
