// SPDX-License-Identifier: AGPL-3.0-or-later

// isArchitecturePath reports whether path falls under lifecycle/architecture/,
// the standing reference zone (catalog entries, promoted choices, ADRs, the
// architecture summary) that is exempt from lineage/index validation.
// Mirrors internal/artifact.IsArchitecturePath.
export function isArchitecturePath(path: string): boolean {
  return path.startsWith('lifecycle/architecture/')
}
