// SPDX-License-Identifier: AGPL-3.0-or-later

package docs

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kaos-control/kaos-control/internal/sandbox"
)

// makeProjectRoot creates a temp dir representing a project root with a docs/ subdir.
func makeProjectRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	return root
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdirall %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestListBasic(t *testing.T) {
	root := makeProjectRoot(t)

	// a.md with frontmatter title
	writeFile(t, filepath.Join(root, "docs", "a.md"), "---\ntitle: Alpha Doc\ntype: doc\nstatus: draft\nlineage: a\n---\nSome content.\n")

	// sub/b.md with no frontmatter, H1 first line
	writeFile(t, filepath.Join(root, "docs", "sub", "b.md"), "# Beta Doc\n\nFirst paragraph.\n")

	entries, err := List(root)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	byPath := make(map[string]DocEntry)
	for _, e := range entries {
		byPath[e.Path] = e
	}

	a, ok := byPath["a.md"]
	if !ok {
		t.Fatal("entry a.md not found")
	}
	if a.Title != "Alpha Doc" {
		t.Errorf("a.md title: got %q, want %q", a.Title, "Alpha Doc")
	}
	if a.SubDir != "" {
		t.Errorf("a.md SubDir: got %q, want %q", a.SubDir, "")
	}
	if !a.IsMarkdown {
		t.Error("a.md IsMarkdown should be true")
	}

	b, ok := byPath["sub/b.md"]
	if !ok {
		t.Fatal("entry sub/b.md not found")
	}
	if b.Title != "Beta Doc" {
		t.Errorf("sub/b.md title: got %q, want %q", b.Title, "Beta Doc")
	}
	if b.SubDir != "sub" {
		t.Errorf("sub/b.md SubDir: got %q, want %q", b.SubDir, "sub")
	}
	if !b.IsMarkdown {
		t.Error("sub/b.md IsMarkdown should be true")
	}
}

func TestListNoDocsDir(t *testing.T) {
	root := t.TempDir()
	// No docs/ created.

	entries, err := List(root)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if entries != nil {
		t.Fatalf("expected nil entries, got: %v", entries)
	}
}

func TestExtractPreviewTruncation(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "*.md")
	if err != nil {
		t.Fatal(err)
	}
	// Write a 500-char paragraph (all ASCII 'a').
	para := strings.Repeat("a", 500)
	if _, err := f.WriteString(para); err != nil {
		t.Fatal(err)
	}
	f.Close()

	_, summary, err := extractPreview(f.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	runes := []rune(summary)
	// Should be exactly 200 runes + the ellipsis character.
	if len(runes) != maxSummaryRunes+1 {
		t.Errorf("summary rune count: got %d, want %d", len(runes), maxSummaryRunes+1)
	}
	if !strings.HasSuffix(summary, "…") {
		t.Error("summary should end with …")
	}
}

func TestExtractPreviewBinaryFallback(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "*.bin")
	if err != nil {
		t.Fatal(err)
	}
	// Write invalid UTF-8 bytes.
	f.Write([]byte{0xFF, 0xFE, 0x00, 0x01})
	f.Close()

	title, summary, err := extractPreview(f.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if title != filepath.Base(f.Name()) {
		t.Errorf("title: got %q, want %q", title, filepath.Base(f.Name()))
	}
	if summary != "(binary or non-text file — cannot preview)" {
		t.Errorf("summary: got %q", summary)
	}
}

func TestExtractPreviewHeadingAndParagraphs(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "*.md")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("# Title\n\nFirst para\n\nSecond para\n")
	f.Close()

	title, summary, err := extractPreview(f.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if title != "Title" {
		t.Errorf("title: got %q, want %q", title, "Title")
	}
	if summary != "First para" {
		t.Errorf("summary: got %q, want %q", summary, "First para")
	}
}

func TestReadWritePathTraversal(t *testing.T) {
	root := makeProjectRoot(t)

	// Test relative path with ".."
	_, err := Read(root, "../README.md")
	if !errors.Is(err, ErrPathTraversal) {
		t.Errorf("Read ../README.md: want ErrPathTraversal, got %v", err)
	}
	if !errors.Is(err, sandbox.ErrPathTraversal) {
		t.Errorf("Read ../README.md: want errors.Is sandbox.ErrPathTraversal, got %v", err)
	}

	err = Write(root, "../README.md", []byte("x"))
	if !errors.Is(err, ErrPathTraversal) {
		t.Errorf("Write ../README.md: want ErrPathTraversal, got %v", err)
	}

	// Test absolute path
	_, err = Read(root, "/etc/passwd")
	if !errors.Is(err, ErrPathTraversal) {
		t.Errorf("Read /etc/passwd: want ErrPathTraversal, got %v", err)
	}

	err = Write(root, "/etc/passwd", []byte("x"))
	if !errors.Is(err, ErrPathTraversal) {
		t.Errorf("Write /etc/passwd: want ErrPathTraversal, got %v", err)
	}
}

func TestReadWriteSymlinkEscape(t *testing.T) {
	root := makeProjectRoot(t)

	// Create a file outside docs/ that a symlink will target.
	outside := filepath.Join(root, "outside.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a symlink inside docs/ pointing to the outside file.
	link := filepath.Join(root, "docs", "escape.md")
	if err := os.Symlink("../outside.txt", link); err != nil {
		t.Skip("symlink creation not supported:", err)
	}

	_, err := Read(root, "escape.md")
	if !errors.Is(err, ErrPathTraversal) {
		t.Errorf("Read symlink escape: want ErrPathTraversal, got %v", err)
	}

	err = Write(root, "escape.md", []byte("x"))
	if !errors.Is(err, ErrPathTraversal) {
		t.Errorf("Write symlink escape: want ErrPathTraversal, got %v", err)
	}
}

func TestWriteNotFound(t *testing.T) {
	root := makeProjectRoot(t)

	err := Write(root, "nonexistent.md", []byte("hello"))
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Write nonexistent: want ErrNotFound, got %v", err)
	}
}
