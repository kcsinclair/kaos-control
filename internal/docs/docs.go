// SPDX-License-Identifier: AGPL-3.0-or-later

// Package docs provides lazy read/write access to <projectRoot>/docs/.
// Files here are not indexed in SQLite; they are enumerated on each List call.
package docs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kaos-control/kaos-control/internal/artifact"
	"github.com/kaos-control/kaos-control/internal/sandbox"
)

// ErrNotFound is returned when the requested doc file does not exist on disk.
var ErrNotFound = errors.New("doc not found")

// ErrPathTraversal is returned when the requested path escapes the docs/ root.
// It wraps sandbox.ErrPathTraversal so callers can also match via errors.Is.
var ErrPathTraversal = fmt.Errorf("docs: %w", sandbox.ErrPathTraversal)

// DocEntry describes one file discovered under <projectRoot>/docs/.
type DocEntry struct {
	// Path is slash-normalised, relative to <projectRoot>/docs/ (e.g. "arch.md", "sub/agents.md").
	Path string
	// Title is the resolved display title (see extractPreview for resolution order).
	Title string
	// Summary is a short preview excerpt (≤ 200 runes). Empty for non-markdown files.
	Summary string
	// IsMarkdown is true when the file extension is .md or .markdown (case-insensitive).
	IsMarkdown bool
	// SubDir is the immediate parent directory relative to docs/, or "" for root-level files.
	SubDir string
}

const (
	maxPreviewBytes = 64 * 1024
	maxSummaryRunes = 200
)

// List walks <projectRoot>/docs/ and returns one DocEntry per file.
// Returns (nil, nil) — not an error — when the docs/ directory does not exist.
// Entries are returned in stable walk order; the caller is responsible for sorting.
func List(projectRoot string) ([]DocEntry, error) {
	docsDir := filepath.Join(projectRoot, "docs")
	if _, err := os.Stat(docsDir); errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}

	var entries []DocEntry
	err := filepath.WalkDir(docsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries non-fatally
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		relPath, relErr := filepath.Rel(docsDir, path)
		if relErr != nil {
			return nil
		}
		relPath = filepath.ToSlash(relPath)

		subDir := filepath.ToSlash(filepath.Dir(relPath))
		if subDir == "." {
			subDir = ""
		}

		ext := strings.ToLower(filepath.Ext(path))
		isMarkdown := ext == ".md" || ext == ".markdown"

		var title, summary string
		if isMarkdown {
			t, s, _ := extractPreview(path)
			title = t
			summary = s
		} else {
			title = filepath.Base(path)
		}

		entries = append(entries, DocEntry{
			Path:       relPath,
			Title:      title,
			Summary:    summary,
			IsMarkdown: isMarkdown,
			SubDir:     subDir,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// extractPreview reads up to 64 KiB of absPath and derives a display title and
// short summary. It never returns an error for content issues; problems are
// surfaced as fallback values.
func extractPreview(absPath string) (title, summary string, err error) {
	f, openErr := os.Open(absPath)
	if openErr != nil {
		return filepath.Base(absPath), "", openErr
	}
	defer f.Close()

	buf := make([]byte, maxPreviewBytes)
	n, _ := f.Read(buf)
	raw := buf[:n]

	if !utf8.Valid(raw) {
		return filepath.Base(absPath), "(binary or non-text file — cannot preview)", nil
	}

	info, _ := f.Stat()
	var mtime time.Time
	if info != nil {
		mtime = info.ModTime()
	}

	// Use artifact.Parse to extract frontmatter fields and body text.
	// Pass just the base filename as relPath; stage/lineage fields are irrelevant here.
	a := artifact.Parse(raw, filepath.Base(absPath), mtime)

	// Title resolution: frontmatter title → first H1 in body → filename stem.
	switch {
	case a.FM.Title != "":
		title = a.FM.Title
	default:
		for _, line := range strings.Split(a.Body, "\n") {
			if strings.HasPrefix(line, "# ") {
				title = strings.TrimSpace(line[2:])
				break
			}
		}
	}
	if title == "" {
		ext := filepath.Ext(absPath)
		title = strings.TrimSuffix(filepath.Base(absPath), ext)
	}

	// Summary resolution: fm.Summary → fm.Description → first prose paragraph.
	switch {
	case a.FM.Summary != "":
		summary = a.FM.Summary
	case a.FM.Description != "":
		summary = a.FM.Description
	default:
		summary = extractFirstParagraph(a.Body)
	}

	// Truncate to maxSummaryRunes runes (multi-byte safe).
	runes := []rune(summary)
	if len(runes) > maxSummaryRunes {
		summary = string(runes[:maxSummaryRunes]) + "…"
	}

	return title, summary, nil
}

// extractFirstParagraph returns the first non-empty, non-heading, non-code-block
// paragraph from a markdown body. Returns "" when none is found.
func extractFirstParagraph(body string) string {
	lines := strings.Split(body, "\n")
	inCode := false
	var para []string
	collecting := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inCode = !inCode
			if collecting {
				break
			}
			continue
		}
		if inCode {
			continue
		}

		if collecting {
			if trimmed == "" {
				break
			}
			para = append(para, line)
		} else {
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				continue
			}
			collecting = true
			para = append(para, line)
		}
	}

	return strings.TrimSpace(strings.Join(para, "\n"))
}

// Read returns the raw bytes of relPath within <projectRoot>/docs/.
// Returns ErrNotFound when the file does not exist.
// Returns ErrPathTraversal when the path escapes the docs/ root.
func Read(projectRoot, relPath string) ([]byte, error) {
	docsRoot := filepath.Join(projectRoot, "docs")
	// If docs/ doesn't exist, the file can't either — report not-found rather
	// than letting sandbox.Resolve walk above the (absent) root and mistake it
	// for a traversal attempt. Mirrors the guard in List.
	if _, statErr := os.Stat(docsRoot); errors.Is(statErr, fs.ErrNotExist) {
		return nil, ErrNotFound
	}
	absPath, err := sandbox.Resolve(docsRoot, relPath)
	if err != nil {
		return nil, ErrPathTraversal
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return data, nil
}

// Write atomically replaces the contents of relPath within <projectRoot>/docs/.
// It preserves the existing file's permission mode.
// Returns ErrNotFound when the target file does not already exist (doc creation is out-of-band).
// Returns ErrPathTraversal when the path escapes the docs/ root.
func Write(projectRoot, relPath string, contents []byte) error {
	docsRoot := filepath.Join(projectRoot, "docs")
	absPath, err := sandbox.Resolve(docsRoot, relPath)
	if err != nil {
		return ErrPathTraversal
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return err
	}
	mode := info.Mode()

	dir := filepath.Dir(absPath)
	tmp, err := os.CreateTemp(dir, ".docs-tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	_, writeErr := tmp.Write(contents)
	_ = tmp.Close()
	if writeErr != nil {
		_ = os.Remove(tmpPath)
		return writeErr
	}

	if err := os.Chmod(tmpPath, mode); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	if err := os.Rename(tmpPath, absPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	return nil
}
