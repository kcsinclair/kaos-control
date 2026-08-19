// SPDX-License-Identifier: AGPL-3.0-or-later

package architecture_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/kaos-control/kaos-control/internal/architecture"
)

// linkTargetRe extracts the target of a markdown link, e.g. "foo.md" from
// "[foo](foo.md)".
var linkTargetRe = regexp.MustCompile(`\]\(([^)]*)\)`)

func summaryInputFixture() architecture.SummaryInput {
	return architecture.SummaryInput{
		Architecture: "lifecycle/architecture/modular-monolith.md",
		TechStack:    "lifecycle/architecture/go-vue.md",
		BreakingRequirements: []architecture.BreakingReq{
			{Label: "offline-capable", Requirement: "Must work with no network connection", Mapping: "Not required — edge/mobile architectures were excluded"},
		},
		QA: []architecture.QAPair{
			{Question: "Does this need to work fully offline?", Answer: "No"},
		},
		ADRPaths:      []string{"lifecycle/architecture/decisions/adr-0001-adopt-modular-monolith.md"},
		StandardPaths: []string{"lifecycle/architecture/standards/secrets-handling.md"},
	}
}

// writeSummaryFixtureFiles creates placeholder files at every path referenced
// by summaryInputFixture(), so the test can assert the summary's links
// actually resolve to files on disk.
func writeSummaryFixtureFiles(t *testing.T, root string, in architecture.SummaryInput) {
	t.Helper()
	paths := append([]string{in.Architecture, in.TechStack}, in.ADRPaths...)
	paths = append(paths, in.StandardPaths...)
	for _, p := range paths {
		mustWrite(t, filepath.Join(root, filepath.FromSlash(p)), "---\ntitle: x\ntype: doc\nstatus: draft\n---\n\nBody.\n")
	}
}

func TestWriteSummary_WritesExpectedContent(t *testing.T) {
	root := t.TempDir()
	in := summaryInputFixture()
	writeSummaryFixtureFiles(t, root, in)

	relPath, err := architecture.WriteSummary(root, in)
	if err != nil {
		t.Fatalf("WriteSummary: %v", err)
	}
	if relPath != "lifecycle/architecture/architecture-summary.md" {
		t.Errorf("relPath = %q, want lifecycle/architecture/architecture-summary.md", relPath)
	}

	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relPath)))
	if err != nil {
		t.Fatalf("reading written summary: %v", err)
	}
	content := string(raw)

	if !strings.Contains(content, "type: doc") {
		t.Error("summary frontmatter missing type: doc")
	}
	if !strings.Contains(content, "## Architecture-breaking requirements") ||
		!strings.Contains(content, "offline-capable") ||
		!strings.Contains(content, "Must work with no network connection") {
		t.Error("summary missing the breaking-requirement mapping table")
	}
	if !strings.Contains(content, "## Selection Q&A") ||
		!strings.Contains(content, "Does this need to work fully offline?") {
		t.Error("summary missing the Q&A trail")
	}

	// Every link target must resolve to a real file relative to the
	// summary's own directory (lifecycle/architecture/).
	summaryDir := filepath.Dir(filepath.Join(root, filepath.FromSlash(relPath)))
	for _, m := range linkTargetRe.FindAllStringSubmatch(content, -1) {
		target := m[1]
		if target == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(summaryDir, filepath.FromSlash(target))); err != nil {
			t.Errorf("link target %q does not resolve to a file: %v", target, err)
		}
	}
}

func TestWriteSummary_IsIdempotent(t *testing.T) {
	root := t.TempDir()
	in := summaryInputFixture()
	writeSummaryFixtureFiles(t, root, in)

	relPath1, err := architecture.WriteSummary(root, in)
	if err != nil {
		t.Fatalf("first WriteSummary: %v", err)
	}
	first, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relPath1)))
	if err != nil {
		t.Fatalf("reading first write: %v", err)
	}

	relPath2, err := architecture.WriteSummary(root, in)
	if err != nil {
		t.Fatalf("second WriteSummary: %v", err)
	}
	second, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relPath2)))
	if err != nil {
		t.Fatalf("reading second write: %v", err)
	}

	if string(first) != string(second) {
		t.Errorf("WriteSummary is not idempotent:\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}
	if !strings.Contains(string(first), "created:") {
		t.Errorf("expected a created: field in the summary frontmatter, got:\n%s", first)
	}

	entries, err := os.ReadDir(filepath.Join(root, "lifecycle", "architecture"))
	if err != nil {
		t.Fatalf("reading architecture dir: %v", err)
	}
	count := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "architecture-summary") {
			count++
		}
	}
	if count != 1 {
		t.Errorf("found %d architecture-summary* files, want exactly 1", count)
	}
}
