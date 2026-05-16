// SPDX-License-Identifier: AGPL-3.0-or-later

package testrunner

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kaos-control/kaos-control/internal/index"
)

// ArtifactMapper maps test failures to lifecycle/tests/*.md artifacts using a
// three-tier lookup: slug/filename match → label match → lineage match.
type ArtifactMapper struct {
	idx *index.Index
}

// NewArtifactMapper creates an ArtifactMapper backed by the given index.
func NewArtifactMapper(idx *index.Index) *ArtifactMapper {
	return &ArtifactMapper{idx: idx}
}

// MapFailure attempts to find the lifecycle test artifact that corresponds to f.
// Returns nil when no match is found; the caller should use the "tests-orphaned"
// lineage in that case.
func (m *ArtifactMapper) MapFailure(f TestFailure) (*index.ArtifactRow, error) {
	// Derive a slug from the test file name.
	slug := fileToSlug(f.File)

	// Tier 1: slug derived from test filename matches artifact lineage.
	if slug != "" {
		rows, _, err := m.idx.List(index.Filter{
			Stage:     "tests",
			Lineage:   slug,
			Unlimited: true,
		})
		if err != nil {
			return nil, err
		}
		if len(rows) > 0 {
			return mostRecentTestArtifact(rows), nil
		}
	}

	// Tier 2: label match — look for test artifacts whose labels include the
	// Go package name or the parent directory of the test file.
	label := packageToLabel(f.Package)
	if label != "" {
		rows, _, err := m.idx.List(index.Filter{
			Stage:     "tests",
			Label:     label,
			Unlimited: true,
		})
		if err != nil {
			return nil, err
		}
		if len(rows) > 0 {
			return mostRecentTestArtifact(rows), nil
		}
	}

	// Tier 3: lineage match — derive a slug from the full package/file path.
	lineageSlug := pathToSlug(f.Package)
	if lineageSlug != "" && lineageSlug != slug {
		rows, _, err := m.idx.List(index.Filter{
			Stage:     "tests",
			Lineage:   lineageSlug,
			Unlimited: true,
		})
		if err != nil {
			return nil, err
		}
		if len(rows) > 0 {
			return mostRecentTestArtifact(rows), nil
		}
	}

	return nil, nil
}

// DetectOrphans returns test artifact paths that have no failures in results,
// indicating potential coverage gaps (tests exist as artifacts but didn't
// produce any failures this run — note: this is an informational signal, not
// an error, since passing tests don't appear in failures).
//
// The returned slice contains test artifact paths that have no corresponding
// failure in any suite result.
func DetectOrphans(results []SuiteResult, testArtifacts []*index.ArtifactRow) []string {
	// Build a set of lineages/slugs covered by this run's failures.
	covered := make(map[string]bool)
	for _, sr := range results {
		for _, f := range sr.Failures {
			slug := fileToSlug(f.File)
			if slug != "" {
				covered[slug] = true
			}
			label := packageToLabel(f.Package)
			if label != "" {
				covered[label] = true
			}
		}
	}

	var orphans []string
	for _, a := range testArtifacts {
		if covered[a.Lineage] || covered[a.Slug] {
			continue
		}
		// Check labels.
		found := false
		for _, lbl := range a.FM.Labels {
			if covered[lbl] {
				found = true
				break
			}
		}
		if !found {
			orphans = append(orphans, a.Path)
		}
	}
	return orphans
}

// ----- slug derivation helpers -----

// nonAlnumRe matches any non-alphanumeric character (used to normalise slugs).
var nonAlnumRe = regexp.MustCompile(`[^a-z0-9]+`)

// fileToSlug converts a test filename into a lineage slug.
// "artifact_store_test.go" → "artifact-store"
// "login.spec.ts" → "login"
// "bar.spec.ts" → "bar"
func fileToSlug(file string) string {
	base := filepath.Base(file)
	// Strip known test suffixes.
	for _, sfx := range []string{"_test.go", ".spec.ts", ".test.ts", ".spec.js", ".test.js"} {
		if strings.HasSuffix(base, sfx) {
			base = base[:len(base)-len(sfx)]
			break
		}
	}
	// Strip .go extension for non-test Go files.
	base = strings.TrimSuffix(base, ".go")
	// Normalise to slug format.
	base = strings.ToLower(base)
	base = strings.ReplaceAll(base, "_", "-")
	base = nonAlnumRe.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")
	return base
}

// packageToLabel returns a label-friendly string derived from the package path
// (last component of a Go package path, or the base directory of a file path).
func packageToLabel(pkg string) string {
	if pkg == "" {
		return ""
	}
	// For a Go package path like "github.com/foo/bar/internal/index", return "index".
	// For a file path like "/project/tests/web/login.spec.ts", return the parent dir.
	pkg = filepath.ToSlash(pkg)
	parts := strings.Split(pkg, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		p := strings.TrimSpace(parts[i])
		if p != "" && !strings.HasSuffix(p, ".go") && !strings.HasSuffix(p, ".ts") && !strings.HasSuffix(p, ".js") {
			return strings.ToLower(p)
		}
	}
	return ""
}

// pathToSlug converts a package or file path into a slug for lineage matching.
// "github.com/foo/bar/internal/index" → "internal-index"
func pathToSlug(pkg string) string {
	if pkg == "" {
		return ""
	}
	pkg = filepath.ToSlash(pkg)
	// Strip module prefix (everything up to and including the module root).
	// Heuristic: take the last two path components as the slug.
	parts := strings.Split(pkg, "/")
	// Remove empty parts and file extensions.
	var clean []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// Remove file extension if this is a filename.
		if ext := filepath.Ext(p); ext != "" {
			p = p[:len(p)-len(ext)]
			// Strip test suffix.
			p = strings.TrimSuffix(p, "_test")
		}
		clean = append(clean, p)
	}
	if len(clean) == 0 {
		return ""
	}
	// Take the last two components.
	start := len(clean) - 2
	if start < 0 {
		start = 0
	}
	slug := strings.Join(clean[start:], "-")
	slug = strings.ToLower(slug)
	slug = nonAlnumRe.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	return slug
}

// mostRecentTestArtifact returns the artifact with the highest index (most recent).
func mostRecentTestArtifact(rows []*index.ArtifactRow) *index.ArtifactRow {
	if len(rows) == 0 {
		return nil
	}
	best := rows[0]
	for _, r := range rows[1:] {
		if r.Index > best.Index {
			best = r
		}
	}
	return best
}
