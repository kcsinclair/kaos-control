// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seedDocs writes files under <projectRoot>/docs/. Keys are slash-separated
// paths relative to docs/ (e.g. "sub/gamma.md"); values are file contents.
func seedDocs(t *testing.T, projectRoot string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		abs := filepath.Join(projectRoot, "docs", filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDocsList_EmptyWhenNoDocsDir(t *testing.T) {
	env := newTestEnv(t, nil)
	// No docs/ directory seeded.
	resp := env.doRequest("GET", "/api/p/testproject/docs", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	if present, _ := data["docs_dir_present"].(bool); present {
		t.Error("expected docs_dir_present: false when docs/ is absent")
	}
	docs, _ := data["docs"].([]any)
	if len(docs) != 0 {
		t.Errorf("expected empty docs list, got %d entries", len(docs))
	}
}

func TestDocsList_ReturnsSortedEntries(t *testing.T) {
	env := newTestEnv(t, nil)
	seedDocs(t, env.projectRoot, map[string]string{
		"zeta.md":     "---\ntitle: Aardvark\n---\n\nContent.\n",
		"alpha.md":    "# Beta\n\nContent.\n",
		"sub/gamma.md": "# Gamma\n\nContent.\n",
	})

	resp := env.doRequest("GET", "/api/p/testproject/docs", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	if present, _ := data["docs_dir_present"].(bool); !present {
		t.Fatal("expected docs_dir_present: true")
	}
	docs, _ := data["docs"].([]any)
	if len(docs) != 3 {
		t.Fatalf("expected 3 doc entries, got %d", len(docs))
	}

	// Server sorts by title ascending (case-insensitive): Aardvark, Beta, Gamma.
	titles := make([]string, 3)
	for i, d := range docs {
		m, _ := d.(map[string]any)
		titles[i], _ = m["title"].(string)
	}
	if titles[0] != "Aardvark" || titles[1] != "Beta" || titles[2] != "Gamma" {
		t.Errorf("unexpected sort order: %v", titles)
	}

	// sub_dir is empty for root files, "sub" for sub/gamma.md.
	rootEntry, _ := docs[0].(map[string]any)
	if sd, _ := rootEntry["sub_dir"].(string); sd != "" {
		t.Errorf("sub_dir for root file: expected empty, got %q", sd)
	}
	subEntry, _ := docs[2].(map[string]any)
	if sd, _ := subEntry["sub_dir"].(string); sd != "sub" {
		t.Errorf("sub_dir for sub/gamma.md: expected %q, got %q", "sub", sd)
	}
}

func TestDocsList_TitleFallbackChain(t *testing.T) {
	env := newTestEnv(t, nil)
	seedDocs(t, env.projectRoot, map[string]string{
		// Frontmatter title wins over H1.
		"fm-title.md": "---\ntitle: FM Title\n---\n# H1 Title\n\nBody.\n",
		// H1 wins over filename stem when frontmatter title is absent.
		"h1-title.md": "# H1 Title\n\nBody.\n",
		// Filename stem used when neither frontmatter title nor H1 is present.
		"no-title.md": "Just some text without a heading.\n",
	})

	resp := env.doRequest("GET", "/api/p/testproject/docs", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)
	docs, _ := data["docs"].([]any)

	titleFor := func(path string) string {
		for _, d := range docs {
			m, _ := d.(map[string]any)
			if m["path"] == path {
				s, _ := m["title"].(string)
				return s
			}
		}
		return ""
	}

	if got := titleFor("fm-title.md"); got != "FM Title" {
		t.Errorf("frontmatter title: expected %q, got %q", "FM Title", got)
	}
	if got := titleFor("h1-title.md"); got != "H1 Title" {
		t.Errorf("H1 title: expected %q, got %q", "H1 Title", got)
	}
	if got := titleFor("no-title.md"); got != "no-title" {
		t.Errorf("filename-stem fallback: expected %q, got %q", "no-title", got)
	}
}

func TestDocsList_SummaryTruncation(t *testing.T) {
	env := newTestEnv(t, nil)

	longPara := strings.Repeat("x", 250)

	seedDocs(t, env.projectRoot, map[string]string{
		// Body paragraph longer than 200 runes → truncated with ellipsis.
		"long-body.md": "# Long\n\n" + longPara + "\n",
		// Frontmatter summary: overrides body extraction.
		"fm-summary.md": "---\nsummary: FM Summary\n---\n\n" + longPara + "\n",
		// Frontmatter description: used when summary is absent.
		"fm-desc.md": "---\ndescription: FM Description\n---\n\nShort body.\n",
	})

	resp := env.doRequest("GET", "/api/p/testproject/docs", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)
	docs, _ := data["docs"].([]any)

	summaryFor := func(path string) string {
		for _, d := range docs {
			m, _ := d.(map[string]any)
			if m["path"] == path {
				s, _ := m["summary"].(string)
				return s
			}
		}
		return ""
	}

	longSummary := summaryFor("long-body.md")
	if !strings.HasSuffix(longSummary, "…") {
		t.Errorf("long body: expected summary ending with ellipsis, got %q", longSummary)
	}
	// The truncated summary is at most 200 runes + the 1-rune ellipsis.
	if r := []rune(longSummary); len(r) > 201 {
		t.Errorf("long body: summary exceeds 201 runes (%d runes)", len(r))
	}

	if got := summaryFor("fm-summary.md"); got != "FM Summary" {
		t.Errorf("fm summary override: expected %q, got %q", "FM Summary", got)
	}
	if got := summaryFor("fm-desc.md"); got != "FM Description" {
		t.Errorf("fm description fallback: expected %q, got %q", "FM Description", got)
	}
}

func TestDocsList_NonMarkdownEntry(t *testing.T) {
	env := newTestEnv(t, nil)
	// Write a minimal PNG header so the file is clearly non-markdown.
	pngHeader := string([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a})
	seedDocs(t, env.projectRoot, map[string]string{
		"diagram.png": pngHeader,
	})

	resp := env.doRequest("GET", "/api/p/testproject/docs", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)
	docs, _ := data["docs"].([]any)
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc entry, got %d", len(docs))
	}

	entry, _ := docs[0].(map[string]any)
	if md, _ := entry["is_markdown"].(bool); md {
		t.Error("expected is_markdown: false for .png file")
	}
	if title, _ := entry["title"].(string); title != "diagram.png" {
		t.Errorf("title: expected %q, got %q", "diagram.png", title)
	}
	if summary, _ := entry["summary"].(string); summary != "" {
		t.Errorf("summary: expected empty for non-markdown, got %q", summary)
	}
}

func TestDocsList_NonUtf8Fallback(t *testing.T) {
	env := newTestEnv(t, nil)
	// Write a .md file with non-UTF-8 byte sequence.
	seedDocs(t, env.projectRoot, map[string]string{
		"binary.md": string([]byte{0xff, 0xfe, 0x00, 0x01}),
	})

	resp := env.doRequest("GET", "/api/p/testproject/docs", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)
	docs, _ := data["docs"].([]any)
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc entry, got %d", len(docs))
	}

	entry, _ := docs[0].(map[string]any)
	wantSummary := "(binary or non-text file — cannot preview)"
	if got, _ := entry["summary"].(string); got != wantSummary {
		t.Errorf("binary summary: expected %q, got %q", wantSummary, got)
	}
	// Title falls back to the filename (including extension) for non-UTF-8 files.
	if title, _ := entry["title"].(string); title != "binary.md" {
		t.Errorf("binary title: expected %q, got %q", "binary.md", title)
	}
}

func TestDocsGet_HappyPath(t *testing.T) {
	env := newTestEnv(t, nil)
	content := "# Alpha\n\nHello, world.\n"
	seedDocs(t, env.projectRoot, map[string]string{"alpha.md": content})

	resp := env.doRequest("GET", "/api/p/testproject/docs/alpha.md", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	if body, _ := data["body"].(string); body != content {
		t.Errorf("body mismatch: got %q, want %q", body, content)
	}
	if sha, _ := data["file_sha"].(string); sha == "" {
		t.Error("expected non-empty file_sha")
	}
	if md, _ := data["is_markdown"].(bool); !md {
		t.Error("expected is_markdown: true for .md file")
	}
}

func TestDocsGet_Subdirectory(t *testing.T) {
	env := newTestEnv(t, nil)
	content := "# Gamma\n\nSub-directory content.\n"
	seedDocs(t, env.projectRoot, map[string]string{"sub/gamma.md": content})

	resp := env.doRequest("GET", "/api/p/testproject/docs/sub/gamma.md", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	if body, _ := data["body"].(string); body != content {
		t.Errorf("body mismatch: got %q, want %q", body, content)
	}
}

func TestDocsGet_NotFound(t *testing.T) {
	env := newTestEnv(t, nil)
	resp := env.doRequest("GET", "/api/p/testproject/docs/missing.md", nil)
	requireStatus(t, resp, http.StatusNotFound)
	data := readJSON(t, resp)
	errObj, _ := data["error"].(map[string]any)
	if code, _ := errObj["code"].(string); code != "not_found" {
		t.Errorf("error code: expected %q, got %q", "not_found", code)
	}
}
