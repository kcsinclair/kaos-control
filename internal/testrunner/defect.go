// SPDX-License-Identifier: AGPL-3.0-or-later

package testrunner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kaos-control/kaos-control/internal/index"
)

// DefectFiler creates and updates defect artifacts in lifecycle/defects/.
type DefectFiler struct {
	idx        *index.Index
	projectDir string
}

// NewDefectFiler creates a DefectFiler that writes to projectDir/lifecycle/defects/.
func NewDefectFiler(idx *index.Index, projectDir string) *DefectFiler {
	return &DefectFiler{idx: idx, projectDir: projectDir}
}

// FileDefect creates a new defect artifact for the given cluster of failures.
// matched is the test artifact that owns these failures (nil for orphaned failures).
// Returns the project-relative path of the created defect artifact.
func (df *DefectFiler) FileDefect(failures []TestFailure, matched *index.ArtifactRow) (string, error) {
	if len(failures) == 0 {
		return "", fmt.Errorf("FileDefect: no failures provided")
	}

	primary := failures[0]

	// Determine lineage.
	var lineage, parentPath string
	if matched != nil {
		lineage = matched.Lineage
		parentPath = matched.Path
	} else {
		lineage = "tests-orphaned"
		if err := df.ensureOrphanedIdeaArtifact(); err != nil {
			return "", fmt.Errorf("ensuring tests-orphaned idea: %w", err)
		}
	}

	// Get the next monotonic index for this lineage.
	nextIdx, err := df.idx.NextIndexForLineage(lineage)
	if err != nil {
		return "", fmt.Errorf("NextIndexForLineage(%q): %w", lineage, err)
	}
	if nextIdx == 0 {
		nextIdx = 2 // first non-originating artifact
	}

	filename := fmt.Sprintf("%s-%d-defect.md", lineage, nextIdx)
	relPath := "lifecycle/defects/" + filename
	absPath := filepath.Join(df.projectDir, filepath.FromSlash(relPath))

	// Build deduplication labels.
	labels := []string{"defect", "auto-filed"}
	labels = append(labels, autoTestLabel(primary))
	if primary.File != "" && primary.Line > 0 {
		labels = append(labels, autoLocLabel(primary))
	}

	// Build assignees.
	role := df.routeRole(primary, matched)

	// Build content.
	content := df.buildContent(failures, primary, lineage, parentPath, nextIdx, role, labels)

	// Write file atomically.
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}
	if err := atomicWriteFile(absPath, []byte(content)); err != nil {
		return "", fmt.Errorf("writing defect: %w", err)
	}

	// Re-index.
	if err := df.idx.IndexFile(absPath); err != nil {
		return "", fmt.Errorf("indexing defect: %w", err)
	}

	return relPath, nil
}

// AppendWitness adds a witness entry to an existing defect's body, recording
// that the same failure was seen again in a subsequent run.
func (df *DefectFiler) AppendWitness(defectRelPath string, f TestFailure) error {
	absPath := filepath.Join(df.projectDir, filepath.FromSlash(defectRelPath))
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("reading defect: %w", err)
	}

	witness := fmt.Sprintf("\n## Witness: %s\n\n- Suite: %s\n- Test: %s\n- File: %s:%d\n- Error: %s\n- Seen: %s\n",
		f.TestName,
		f.Suite,
		f.TestName,
		f.File, f.Line,
		f.ErrorMsg,
		time.Now().UTC().Format(time.RFC3339),
	)

	updated := strings.TrimRight(string(raw), "\n") + "\n" + witness
	if err := atomicWriteFile(absPath, []byte(updated)); err != nil {
		return fmt.Errorf("writing witness: %w", err)
	}
	return df.idx.IndexFile(absPath)
}

// ----- internal helpers -----

// routeRole implements F7 routing: label-based first, then path-based.
func (df *DefectFiler) routeRole(f TestFailure, matched *index.ArtifactRow) string {
	if matched != nil {
		for _, lbl := range matched.FM.Labels {
			switch lbl {
			case "backend", "backend-developer":
				return "backend-developer"
			case "frontend", "frontend-developer":
				return "frontend-developer"
			case "test", "test-developer":
				return "test-developer"
			}
		}
	}
	return pathBasedRole(f.Package)
}

// pathBasedRole derives a role from the failure's file/package path.
func pathBasedRole(pkg string) string {
	pkg = filepath.ToSlash(pkg)
	switch {
	case strings.Contains(pkg, "tests/web") || strings.Contains(pkg, "tests/e2e"):
		return "frontend-developer"
	default:
		return "backend-developer"
	}
}

// buildTitle derives a concise defect title from the primary failure.
func buildTitle(f TestFailure) string {
	name := f.TestName
	if name == "" {
		name = f.Package
	}
	msg := f.ErrorMsg
	if msg == "" && f.Output != "" {
		// Take first non-empty output line.
		for _, l := range strings.Split(f.Output, "\n") {
			l = strings.TrimSpace(l)
			if l != "" {
				msg = l
				break
			}
		}
	}
	if msg == "" {
		return truncate(name, 80)
	}
	title := name + ": " + msg
	return truncate(title, 80)
}

// buildReproduction generates the exact command to re-run just one test.
func buildReproduction(f TestFailure) string {
	switch f.Suite {
	case "go":
		testArg := f.TestName
		if idx := strings.Index(testArg, "/"); idx >= 0 {
			testArg = testArg[:idx] // base test name only for -run flag
		}
		return fmt.Sprintf("go test -run %s -count=1 %s", testArg, f.Package)
	case "vitest":
		return fmt.Sprintf("pnpm exec vitest run %q", f.Package)
	case "playwright":
		return fmt.Sprintf("pnpm exec playwright test --grep %q", f.TestName)
	default:
		return ""
	}
}

// buildContent assembles the full defect markdown content.
func (df *DefectFiler) buildContent(
	failures []TestFailure,
	primary TestFailure,
	lineage, parentPath string,
	nextIdx int,
	role string,
	labels []string,
) string {
	title := buildTitle(primary)

	// Format labels as YAML list.
	var labelLines []string
	for _, l := range labels {
		labelLines = append(labelLines, "    - "+l)
	}
	labelsYAML := strings.Join(labelLines, "\n")

	// Parent field.
	parentField := ""
	if parentPath != "" {
		parentField = "\nparent: " + parentPath
	}

	repro := buildReproduction(primary)

	// Witness list for grouped failures.
	var witnesses strings.Builder
	if len(failures) > 1 {
		witnesses.WriteString("\n## Witnesses\n\n")
		for _, f := range failures {
			witnesses.WriteString(fmt.Sprintf("- `%s` at `%s:%d`\n", f.TestName, f.File, f.Line))
		}
	}

	// Location info.
	location := ""
	if primary.File != "" {
		location = fmt.Sprintf("`%s`", primary.File)
		if primary.Line > 0 {
			location += fmt.Sprintf(" line %d", primary.Line)
		}
	}

	return fmt.Sprintf(`---
title: '%s'
type: defect
status: draft
lineage: %s%s
labels:
%s
assignees:
    - role: %s
      who: agent
---

# %s

**Suite**: %s | **Test**: `+"`%s`"+` | **Location**: %s

## Failure

`+"```"+`
%s
`+"```"+`

## Reproduction

`+"```"+`sh
%s
`+"```"+`
%s
## Actual Behaviour

%s

## Expected Behaviour

<!-- Describe expected behaviour here -->
`,
		escapeSingleQuote(title),
		lineage,
		parentField,
		labelsYAML,
		role,
		title,
		primary.Suite,
		primary.TestName,
		location,
		truncate(primary.Output, 2000),
		repro,
		witnesses.String(),
		truncate(primary.ErrorMsg, 500),
	)
}

// ensureOrphanedIdeaArtifact creates lifecycle/ideas/tests-orphaned.md if it
// does not yet exist.
func (df *DefectFiler) ensureOrphanedIdeaArtifact() error {
	relPath := "lifecycle/ideas/tests-orphaned.md"
	absPath := filepath.Join(df.projectDir, filepath.FromSlash(relPath))

	if _, err := os.Stat(absPath); err == nil {
		return nil // already exists
	}

	content := `---
title: 'Tests-Orphaned: test failures with no matching lifecycle/tests artifact'
type: idea
status: draft
lineage: tests-orphaned
---

# Tests-Orphaned

Auto-created by the test-runner agent to group test failures that have no
corresponding ` + "`lifecycle/tests/*.md`" + ` artifact. Review these failures
and create the appropriate test artifacts to give them proper lineage.
`
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return err
	}
	return atomicWriteFile(absPath, []byte(content))
}

// atomicWriteFile writes content to absPath via a temp file then renames.
func atomicWriteFile(absPath string, content []byte) error {
	tmp := absPath + ".tmp"
	if err := os.WriteFile(tmp, content, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, absPath); err != nil {
		os.Remove(tmp) //nolint:errcheck
		return err
	}
	return nil
}

// truncate returns s truncated to at most maxChars UTF-8 characters.
func truncate(s string, maxChars int) string {
	if utf8.RuneCountInString(s) <= maxChars {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxChars])
}

// escapeSingleQuote escapes single quotes in a YAML single-quoted string.
func escapeSingleQuote(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
