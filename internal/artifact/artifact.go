// Package artifact parses kaos-control lifecycle markdown files.
package artifact

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"go.abhg.dev/goldmark/frontmatter"
)

// KnownTypes is the allowed vocabulary for the type field.
var KnownTypes = map[string]bool{
	"idea": true, "requirement": true,
	"plan-backend": true, "plan-frontend": true, "plan-test": true,
	"test": true, "prototype": true, "defect": true,
}

// KnownStatuses is the allowed vocabulary for the status field.
var KnownStatuses = map[string]bool{
	"draft": true, "clarifying": true, "planning": true,
	"in-development": true, "in-qa": true, "approved": true,
	"rejected": true, "abandoned": true, "done": true,
	"blocked": true,
}

// Artifact is a fully parsed lifecycle markdown file.
type Artifact struct {
	Path        string      // relative to project root, e.g. "lifecycle/ideas/login.md"
	Slug        string      // derived from filename stem before any -N suffix
	Index       int         // 0 = originating file; >=2 for descendants
	StageSuffix string      // e.g. "be", "fe" from filename
	Stage       string      // lifecycle stage dir name, e.g. "ideas"
	FM          Frontmatter
	Body        string      // raw markdown body (after frontmatter)
	Links       []Link
	Mtime       time.Time
	SHA256      [32]byte
	Raw         []byte      // full file content; not stored in index
	ParseErrs   []string    // non-fatal validation messages
}

// Frontmatter holds the structured YAML header fields.
type Frontmatter struct {
	Title     string     `yaml:"title"               json:"title"`
	Type      string     `yaml:"type"                json:"type"`
	Status    string     `yaml:"status"              json:"status"`
	Lineage   string     `yaml:"lineage"             json:"lineage"`
	Priority  string     `yaml:"priority,omitempty"  json:"priority,omitempty"`
	Parent    string     `yaml:"parent,omitempty"    json:"parent,omitempty"`
	Labels    []string   `yaml:"labels,omitempty"    json:"labels,omitempty"`
	DependsOn []string   `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	Blocks    []string   `yaml:"blocks,omitempty"    json:"blocks,omitempty"`
	Related   []string   `yaml:"related_to,omitempty" json:"related_to,omitempty"`
	Members   []string   `yaml:"members,omitempty"   json:"members,omitempty"`
	Release   string     `yaml:"release,omitempty"   json:"release,omitempty"`
	Sprint    string     `yaml:"sprint,omitempty"    json:"sprint,omitempty"`
	Assignees []Assignee `yaml:"assignees,omitempty" json:"assignees,omitempty"`
}

// Assignee is a role/who binding from the assignees field.
type Assignee struct {
	Role string `yaml:"role" json:"role"`
	Who  string `yaml:"who"  json:"who"`
}

// Link is a directed relationship extracted from the artifact.
type Link struct {
	From   string // relative path from project root
	To     string // relative path from project root
	Kind   string // parent | depends_on | blocks | related_to | members | wiki
	Source string // e.g. "frontmatter:parent" or "body:wiki"
}

var md = goldmark.New(
	goldmark.WithExtensions(&frontmatter.Extender{}),
	goldmark.WithRendererOptions(html.WithUnsafe()),
)

// Parse parses the raw bytes of a lifecycle markdown file.
// relPath is relative to the project root (e.g. "lifecycle/ideas/login.md").
func Parse(raw []byte, relPath string, mtime time.Time) *Artifact {
	sum := sha256.Sum256(raw)
	stage, slug, idx, sfx := parsePathComponents(relPath)

	a := &Artifact{
		Path:        relPath,
		Slug:        slug,
		Index:       idx,
		StageSuffix: sfx,
		Stage:       stage,
		Mtime:       mtime,
		SHA256:      sum,
		Raw:         raw,
	}

	ctx := parser.NewContext()
	var bodyBuf bytes.Buffer
	if err := md.Convert(raw, &bodyBuf, parser.WithContext(ctx)); err != nil {
		a.ParseErrs = append(a.ParseErrs, fmt.Sprintf("goldmark parse error: %v", err))
		return a
	}

	// Extract frontmatter.
	d := frontmatter.Get(ctx)
	if d != nil {
		if err := d.Decode(&a.FM); err != nil {
			a.ParseErrs = append(a.ParseErrs, fmt.Sprintf("frontmatter decode error: %v", err))
		}
	}

	// Derive body from raw by stripping the frontmatter block.
	a.Body = stripFrontmatter(raw)

	// Validate required fields; record errors but still index.
	if a.FM.Title == "" {
		a.ParseErrs = append(a.ParseErrs, "missing required field: title")
		// Best-effort: use first heading or filename.
		a.FM.Title = derivedTitle(a.Body, slug)
	}
	if a.FM.Type == "" {
		a.ParseErrs = append(a.ParseErrs, "missing required field: type")
		a.FM.Type = stageToType(stage)
	}
	if a.FM.Status == "" {
		a.ParseErrs = append(a.ParseErrs, "missing required field: status")
		a.FM.Status = "draft"
	}
	if a.FM.Lineage == "" {
		a.ParseErrs = append(a.ParseErrs, "missing required field: lineage")
		a.FM.Lineage = slug
	}
	if !KnownTypes[a.FM.Type] {
		a.ParseErrs = append(a.ParseErrs, fmt.Sprintf("unknown type %q", a.FM.Type))
	}
	if !KnownStatuses[a.FM.Status] {
		a.ParseErrs = append(a.ParseErrs, fmt.Sprintf("unknown status %q", a.FM.Status))
	}

	a.Links = extractLinks(a.FM, a.Body, relPath)
	return a
}

// PatchFrontmatterField replaces the value of key within the YAML frontmatter.
// Only the region between the opening and closing --- fences is modified; the
// document body is left untouched. Returns (patched, true) on success or
// (raw, false) if the key is not present or the file has no frontmatter fence.
func PatchFrontmatterField(raw []byte, key, value string) ([]byte, bool) {
	s := string(raw)
	if !strings.HasPrefix(s, "---") {
		return raw, false
	}
	closeIdx := strings.Index(s[3:], "\n---")
	if closeIdx < 0 {
		return raw, false
	}
	fmEnd := 3 + closeIdx
	fmSection := s[3:fmEnd]
	lineRe := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(key) + `:\s*.*$`)
	replaced := lineRe.ReplaceAllLiteralString(fmSection, key+": "+value)
	if replaced == fmSection {
		return raw, false
	}
	return []byte("---" + replaced + s[fmEnd:]), true
}

// RenderHTML renders markdown source to HTML using goldmark.
func RenderHTML(src string) string {
	var buf bytes.Buffer
	if err := goldmark.New(
		goldmark.WithRendererOptions(html.WithUnsafe()),
	).Convert([]byte(src), &buf); err != nil {
		return "<p>render error</p>"
	}
	return buf.String()
}

// ----- filename / path helpers -----

// indexSuffixRe matches an optional -N(-suffix) at the end of a filename stem.
// Group 1 = slug, group 2 = index digits, group 3 = optional alpha suffix.
var indexSuffixRe = regexp.MustCompile(`^(.+)-(\d+)(?:-([a-zA-Z]+))?$`)

// parsePathComponents extracts stage, slug, index, and stage-suffix from a
// project-relative path like "lifecycle/backend-plans/login-3-be.md".
func parsePathComponents(relPath string) (stage, slug string, idx int, sfx string) {
	// relPath is like "lifecycle/<stage>/<filename>.md"
	parts := strings.SplitN(relPath, "/", 3)
	if len(parts) >= 2 {
		stage = parts[len(parts)-2] // parent directory name
	}
	stem := strings.TrimSuffix(filepath.Base(relPath), ".md")
	slug, idx, sfx = ParseFilename(stem)
	return
}

// ParseFilename extracts (slug, index, stageSuffix) from a filename stem.
// "login" → ("login", 0, "")
// "login-3-be" → ("login", 3, "be")
func ParseFilename(stem string) (slug string, idx int, sfx string) {
	m := indexSuffixRe.FindStringSubmatch(stem)
	if m == nil {
		return stem, 0, ""
	}
	n, _ := strconv.Atoi(m[2])
	return m[1], n, m[3]
}

// ----- link extraction -----

var wikiLinkRe = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

func extractLinks(fm Frontmatter, body, fromPath string) []Link {
	var links []Link

	addFM := func(kind string, targets []string) {
		for _, t := range targets {
			links = append(links, Link{
				From:   fromPath,
				To:     normaliseLinkTarget(t, fromPath),
				Kind:   kind,
				Source: "frontmatter:" + kind,
			})
		}
	}

	if fm.Parent != "" {
		links = append(links, Link{
			From:   fromPath,
			To:     normaliseLinkTarget(fm.Parent, fromPath),
			Kind:   "parent",
			Source: "frontmatter:parent",
		})
	}
	addFM("depends_on", fm.DependsOn)
	addFM("blocks", fm.Blocks)
	addFM("related_to", fm.Related)
	addFM("members", fm.Members)

	// Wiki-style body links.
	for _, m := range wikiLinkRe.FindAllStringSubmatch(body, -1) {
		content := m[1]
		target := content
		if i := strings.Index(content, "|"); i >= 0 {
			target = strings.TrimSpace(content[:i])
		}
		links = append(links, Link{
			From:   fromPath,
			To:     normaliseLinkTarget(target, fromPath),
			Kind:   "wiki",
			Source: "body:wiki",
		})
	}
	return links
}

// normaliseLinkTarget resolves a link target to a project-relative path.
// Targets are relative to lifecycle/ root; we add the "lifecycle/" prefix
// and ensure the .md extension is present.
func normaliseLinkTarget(target, fromPath string) string {
	target = strings.TrimSpace(target)
	// If already has lifecycle/ prefix, keep it.
	if strings.HasPrefix(target, "lifecycle/") {
		return ensureMD(target)
	}
	// If the target already looks like an absolute path in the project, keep it.
	if strings.Contains(target, "/") {
		return ensureMD("lifecycle/" + target)
	}
	// Single-component: treat as relative to the same stage dir.
	stage := filepath.Dir(fromPath)
	return ensureMD(filepath.Join(stage, target))
}

func ensureMD(p string) string {
	if !strings.HasSuffix(p, ".md") {
		return p + ".md"
	}
	return p
}

// ----- body / title helpers -----

var fmFenceRe = regexp.MustCompile(`(?s)^---\n.*?\n---\n?`)

// stripFrontmatter returns the markdown body after the YAML frontmatter block.
func stripFrontmatter(raw []byte) string {
	s := string(raw)
	if strings.HasPrefix(s, "---") {
		if loc := fmFenceRe.FindStringIndex(s); loc != nil {
			return strings.TrimSpace(s[loc[1]:])
		}
	}
	return strings.TrimSpace(s)
}

var h1Re = regexp.MustCompile(`(?m)^#\s+(.+)$`)

// derivedTitle extracts the first H1 heading from the body, or falls back to slug.
func derivedTitle(body, slug string) string {
	if m := h1Re.FindStringSubmatch(body); m != nil {
		return strings.TrimSpace(m[1])
	}
	return slug
}

// stageToType maps a lifecycle stage directory name to a default artifact type.
func stageToType(stage string) string {
	switch stage {
	case "ideas":
		return "idea"
	case "requirements":
		return "requirement"
	case "backend-plans":
		return "plan-backend"
	case "frontend-plans":
		return "plan-frontend"
	case "test-plans":
		return "plan-test"
	case "tests":
		return "test"
	case "prototypes":
		return "prototype"
	case "defects":
		return "defect"
	default:
		return "requirement"
	}
}
