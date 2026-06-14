// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestDocsGet_RejectsParentTraversal verifies that path traversal attempts via
// "../" do not expose files outside the docs/ root. Go's url.Parse resolves
// literal ".." segments, so the route typically won't match and chi returns 404.
// URL-encoded "%2e%2e" is treated as a literal directory name by the sandbox
// (not decoded to ".."), resulting in 404 for a non-existent path.
// Either response is acceptable; neither should be 200.
func TestDocsGet_RejectsParentTraversal(t *testing.T) {
	env := newTestEnv(t, nil)

	for _, tc := range []struct {
		name string
		path string
	}{
		{"literal-dotdot", "/api/p/testproject/docs/../README.md"},
		{"encoded-dotdot", "/api/p/testproject/docs/%2e%2e/README.md"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := env.doRequest("GET", tc.path, nil)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				t.Errorf("traversal attempt %q returned 200; expected non-200", tc.path)
			}
		})
	}
}

// TestDocsGet_RejectsAbsolutePath verifies that attempts to inject an absolute
// path via a double-slash prefix do not serve files outside docs/. Chi's path
// normalisation collapses "//" to "/" so the route resolves to a relative path
// that simply doesn't exist, returning 404.
func TestDocsGet_RejectsAbsolutePath(t *testing.T) {
	env := newTestEnv(t, nil)
	resp := env.doRequest("GET", "/api/p/testproject/docs//etc/passwd", nil)
	resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Errorf("absolute-path injection returned 200; expected non-200")
	}
}

// TestDocsGet_RejectsEscapingSymlink verifies that a symlink inside docs/ that
// points outside the project root is rejected with 400 path_traversal. This is
// the definitive sandbox check: EvalSymlinks resolves the real target, and
// sandbox.Resolve returns ErrPathTraversal when it escapes the docs root.
func TestDocsGet_RejectsEscapingSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink test skipped on Windows")
	}

	env := newTestEnv(t, nil)

	// Create a target file outside the project root.
	outsideDir := t.TempDir()
	outsideFile := filepath.Join(outsideDir, "outside.md")
	if err := os.WriteFile(outsideFile, []byte("# Outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create docs/ and a symlink inside it pointing at the outside file.
	docsDir := filepath.Join(env.projectRoot, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	symlinkPath := filepath.Join(docsDir, "escape.md")
	if err := os.Symlink(outsideFile, symlinkPath); err != nil {
		t.Fatal(err)
	}

	resp := env.doRequest("GET", "/api/p/testproject/docs/escape.md", nil)
	requireStatus(t, resp, http.StatusBadRequest)
	data := readJSON(t, resp)
	errObj, _ := data["error"].(map[string]any)
	if code, _ := errObj["code"].(string); code != "path_traversal" {
		t.Errorf("error code: expected %q, got %q", "path_traversal", code)
	}
}

// TestDocsPut_RejectsParentTraversal verifies that PUT requests with traversal
// paths do not write files outside docs/. Like the GET case, the encoded form
// is treated as a literal directory name (404), while the literal form is
// cleaned by url.Parse before the request is sent (non-matching route → 404).
func TestDocsPut_RejectsParentTraversal(t *testing.T) {
	env := newTestEnv(t, nil)
	seedDocs(t, env.projectRoot, map[string]string{"placeholder.md": "# Placeholder\n"})

	for _, tc := range []struct {
		name string
		path string
	}{
		{"literal-dotdot", "/api/p/testproject/docs/../etc/foo.md"},
		{"encoded-dotdot", "/api/p/testproject/docs/%2e%2e/etc/foo.md"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := env.doRequest("PUT", tc.path,
				map[string]any{"body": "# Evil\n"})
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				t.Errorf("traversal PUT %q returned 200; expected non-200", tc.path)
			}
		})
	}
}
