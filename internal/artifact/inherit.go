// SPDX-License-Identifier: AGPL-3.0-or-later

package artifact

// ApplyInheritedFields copies Priority and Release from parent into child when
// the child's value is empty. A non-empty child value always wins. Empty parent
// values are left as-is — no fabricated defaults are introduced.
func ApplyInheritedFields(child *Frontmatter, parent Frontmatter) {
	if child.Priority == "" && parent.Priority != "" {
		child.Priority = parent.Priority
	}
	if child.Release == "" && parent.Release != "" {
		child.Release = parent.Release
	}
}
