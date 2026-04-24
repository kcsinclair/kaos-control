// Package web holds the embedded Vue SPA built by Vite.
// The dist/ directory is populated by `make build-web` and embedded here
// so that `go build` produces a single self-contained binary.
package web

import "embed"

//go:embed all:dist
var FS embed.FS
