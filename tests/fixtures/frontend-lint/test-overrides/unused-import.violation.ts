// Negative case: tests/web/eslint.config.js's ergonomics overrides only turn
// off @typescript-eslint/no-explicit-any — no-unused-vars keeps the same
// argsIgnorePattern/varsIgnorePattern('^_') as production, so a
// non-underscore-prefixed unused import is still a hard failure here.
import { useMock } from './any-mock.valid'

export function noop(): void {}
