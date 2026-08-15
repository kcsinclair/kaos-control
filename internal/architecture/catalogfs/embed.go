// SPDX-License-Identifier: AGPL-3.0-or-later

// Package catalogfs embeds the shipped architecture catalog — the README
// plus every architectures/*.md and tech-stacks/*.md entry — so it can be
// seeded into new and existing projects without a runtime dependency on
// this repository's own lifecycle/architecture/ tree.
//
// The embedded files are a checked-in copy of lifecycle/architecture/: Go's
// //go:embed directive cannot reach outside its own package directory, so
// the source of truth at lifecycle/architecture/ cannot be embedded
// directly from internal/. embed_test.go guards against the two drifting
// apart.
package catalogfs

import "embed"

//go:embed README.md architectures tech-stacks
var FS embed.FS
