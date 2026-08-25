// Positive case: strict equality is used for the value comparison, and the
// `== null` idiom (permitted by the "smart" eqeqeq option) checks for both
// null and undefined in one expression.
export function isOne(value: number): boolean {
  return value === 1
}

export function isNullish(value: unknown): boolean {
  return value == null
}
