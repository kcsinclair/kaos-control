// SPDX-License-Identifier: AGPL-3.0-or-later

package catalogfs_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kaos-control/kaos-control/internal/architecture/catalogfs"
)

// projectLocalFrontmatter are frontmatter keys that legitimately differ between
// the shipped/embedded catalog and this repo's own copies: they are per-project
// lifecycle state, not catalog content. The embedded catalog ships a neutral
// default (status: draft) for fresh projects to seed from, while this repo may
// approve its own copies for use — that divergence must not read as drift.
// created: is likewise project-local (stamped per project, never on the
// shipped catalog). Everything else — the architecture description, labels,
// related_to, summary — must stay byte-identical.
var projectLocalFrontmatter = []string{"status:", "created:"}

// canonicalForDrift strips the project-local frontmatter lines so drift is
// judged on catalog content alone. Body is left untouched.
func canonicalForDrift(raw []byte) string {
	s := string(raw)
	if !strings.HasPrefix(s, "---") {
		return s
	}
	closeIdx := strings.Index(s[3:], "\n---")
	if closeIdx < 0 {
		return s
	}
	fmEnd := 3 + closeIdx
	var keep []string
	for _, line := range strings.Split(s[3:fmEnd], "\n") {
		trimmed := strings.TrimSpace(line)
		skip := false
		for _, k := range projectLocalFrontmatter {
			if strings.HasPrefix(trimmed, k) {
				skip = true
				break
			}
		}
		if !skip {
			keep = append(keep, line)
		}
	}
	return "---" + strings.Join(keep, "\n") + s[fmEnd:]
}

// repoLifecycleArchitectureDir locates this repo's own lifecycle/architecture/
// tree from the catalogfs package's test working directory
// (internal/architecture/catalogfs), so drift can be detected without a
// build step.
func repoLifecycleArchitectureDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(wd, "..", "..", "..", "lifecycle", "architecture")
}

// TestFS_MatchesRepoCatalog fails if the checked-in embedded copy under
// internal/architecture/catalogfs/ ever drifts from this repo's own
// lifecycle/architecture/ catalog (the source both are meant to mirror).
func TestFS_MatchesRepoCatalog(t *testing.T) {
	liveDir := repoLifecycleArchitectureDir(t)

	seen := map[string]bool{}
	if err := fs.WalkDir(catalogfs.FS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		seen[p] = true

		embedded, err := fs.ReadFile(catalogfs.FS, p)
		if err != nil {
			return err
		}
		live, err := os.ReadFile(filepath.Join(liveDir, filepath.FromSlash(p)))
		if err != nil {
			t.Errorf("%s: embedded copy has no live counterpart at lifecycle/architecture/%s: %v", p, p, err)
			return nil
		}
		if canonicalForDrift(embedded) != canonicalForDrift(live) {
			t.Errorf("%s: embedded copy has drifted from lifecycle/architecture/%s (content differs beyond project-local status/created)", p, p)
		}
		return nil
	}); err != nil {
		t.Fatalf("walking embedded catalog: %v", err)
	}

	// Catch the reverse: a live catalog file that was never copied in.
	for _, sub := range []string{"architectures", "tech-stacks"} {
		entries, err := os.ReadDir(filepath.Join(liveDir, sub))
		if err != nil {
			t.Fatalf("reading lifecycle/architecture/%s: %v", sub, err)
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			rel := sub + "/" + e.Name()
			if !seen[rel] {
				t.Errorf("lifecycle/architecture/%s has no embedded counterpart in internal/architecture/catalogfs/%s", rel, rel)
			}
		}
	}
	if !seen["README.md"] {
		t.Error("embedded catalog is missing README.md")
	}
}
