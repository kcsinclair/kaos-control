// SPDX-License-Identifier: AGPL-3.0-or-later

package architecture

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// adrFileRe matches an ADR filename and captures its zero-padded number.
var adrFileRe = regexp.MustCompile(`^adr-(\d{4})-.*\.md$`)

// slugNonAlnumRe collapses runs of non-alphanumeric characters for slugify.
var slugNonAlnumRe = regexp.MustCompile(`[^a-z0-9]+`)

// NextADRNumber returns the next monotonic ADR number for the project: one
// greater than the highest numeric prefix found among files in
// lifecycle/architecture/decisions/ (including status: superseded/rejected
// ones — numbers are never reused, FR-14), or 1 when the directory is
// empty or absent. Numbering derives from files present on disk: deleting
// the highest-numbered file lowers the next allocation.
func NextADRNumber(projectRoot string) (int, error) {
	dir := filepath.Join(projectRoot, filepath.FromSlash(architectureDir), "decisions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 1, nil
		}
		return 0, err
	}

	max := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := adrFileRe.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		n, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		if n > max {
			max = n
		}
	}
	return max + 1, nil
}

// ADRRequest is the input to CreateADR. Status defaults to "draft" when empty.
type ADRRequest struct {
	Slug   string
	Title  string
	Status string
	Body   string
}

// CreateADR allocates the next ADR number, writes
// lifecycle/architecture/decisions/adr-<NNNN>-<slug>.md, and returns its
// repo-relative path. Allocate-then-write is not atomic across processes, so
// on a collision (a file already claimed the allocated number by the time we
// write) the number is re-allocated once before giving up.
func CreateADR(projectRoot string, req ADRRequest) (string, error) {
	status := req.Status
	if status == "" {
		status = "draft"
	}

	decisionsDir := filepath.Join(projectRoot, filepath.FromSlash(architectureDir), "decisions")
	if err := os.MkdirAll(decisionsDir, 0o755); err != nil {
		return "", err
	}

	for attempt := 0; attempt < 2; attempt++ {
		n, err := NextADRNumber(projectRoot)
		if err != nil {
			return "", err
		}
		filename := fmt.Sprintf("adr-%04d-%s.md", n, req.Slug)
		absPath := filepath.Join(decisionsDir, filename)

		if _, statErr := os.Stat(absPath); statErr == nil {
			// Another writer claimed this number since we scanned — retry once.
			continue
		} else if !os.IsNotExist(statErr) {
			return "", statErr
		}

		content, cerr := buildADRContent(req.Title, status, req.Body)
		if cerr != nil {
			return "", cerr
		}
		if werr := writeAtomic(absPath, content); werr != nil {
			return "", werr
		}
		return path.Join(architectureDir, "decisions", filename), nil
	}
	return "", fmt.Errorf("could not allocate a free ADR number after retry")
}

// WriteADR0001 deterministically authors (or re-authors) the wizard's
// "Adopt <arch> with <stack>" ADR, including the Q&A trail and rejected
// alternatives. It is idempotent (FR-12, NFR-3): if an adr-0001-*.md already
// exists, that same file is overwritten rather than allocating adr-0002.
// Unlike agent-proposed ADRs (CreateADR, which default to status: draft,
// OQ-4), ADR-0001 records a decision the wizard has already made — a fait
// accompli — so it is authored as status: approved.
func WriteADR0001(projectRoot, arch, stack, qaTrail string, rejected []string) (string, error) {
	decisionsDir := filepath.Join(projectRoot, filepath.FromSlash(architectureDir), "decisions")
	if err := os.MkdirAll(decisionsDir, 0o755); err != nil {
		return "", err
	}

	entries, err := os.ReadDir(decisionsDir)
	if err != nil {
		return "", err
	}
	var filename string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "adr-0001-") && strings.HasSuffix(e.Name(), ".md") {
			filename = e.Name()
			break
		}
	}
	if filename == "" {
		filename = fmt.Sprintf("adr-0001-adopt-%s.md", slugify(arch))
	}

	title := fmt.Sprintf("Adopt %s with %s", arch, stack)
	var body strings.Builder
	body.WriteString(strings.TrimSpace(qaTrail))
	if len(rejected) > 0 {
		body.WriteString("\n\n## Rejected alternatives\n\n")
		for _, r := range rejected {
			body.WriteString("- " + r + "\n")
		}
	}

	content, cerr := buildADRContent(title, "approved", body.String())
	if cerr != nil {
		return "", cerr
	}

	absPath := filepath.Join(decisionsDir, filename)
	if werr := writeAtomic(absPath, content); werr != nil {
		return "", werr
	}
	return path.Join(architectureDir, "decisions", filename), nil
}

// adrFrontmatter is the minimal frontmatter shape written for ADR files. The
// date field is intentionally omitted — this primitive is deterministic and
// carries no clock; callers/UI stamp a date if wanted.
type adrFrontmatter struct {
	Title  string `yaml:"title"`
	Type   string `yaml:"type"`
	Status string `yaml:"status"`
}

func buildADRContent(title, status, body string) ([]byte, error) {
	fmBytes, err := yaml.Marshal(adrFrontmatter{Title: title, Type: "adr", Status: status})
	if err != nil {
		return nil, fmt.Errorf("marshalling ADR frontmatter: %w", err)
	}
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.Write(fmBytes)
	sb.WriteString("---\n")
	if body = strings.TrimSpace(body); body != "" {
		sb.WriteString("\n")
		sb.WriteString(body)
		sb.WriteString("\n")
	}
	return []byte(sb.String()), nil
}

// slugify derives a kebab-case slug from an arbitrary string.
func slugify(s string) string {
	s = strings.ToLower(s)
	s = slugNonAlnumRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "adr"
	}
	return s
}
