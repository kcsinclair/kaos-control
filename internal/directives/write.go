// SPDX-License-Identifier: AGPL-3.0-or-later

package directives

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

// FileWrite reports the outcome of writing (or attempting to write) one
// generated file.
type FileWrite struct {
	Path    string `json:"path"`
	Created bool   `json:"created"`
	Changed bool   `json:"changed"`
	Skipped bool   `json:"skipped"`
	// Diff is set when an existing file has no managed-region markers (they
	// were removed, or the file predates markers entirely — the migration
	// case) and the whole-file replacement was withheld pending Force, so a
	// human can confirm before it's applied.
	Diff string `json:"diff,omitempty"`
}

// hasMarkers reports whether b contains both managed-region markers, in order.
func hasMarkers(b []byte) bool {
	start := bytes.Index(b, []byte(genStart))
	if start < 0 {
		return false
	}
	return bytes.Contains(b[start:], []byte(genEnd))
}

// mergeManaged merges freshBody — a full marker-wrapped body, as produced by
// RenderAgents — into existing. If existing already contains the marker
// pair, only the region between them is replaced, preserving any prose
// written above or below (OQ-6). Otherwise the whole file is treated as
// generated and replaced outright. Returns the merged bytes and whether
// they differ from existing.
func mergeManaged(existing, freshBody []byte) ([]byte, bool, error) {
	if !hasMarkers(existing) {
		return freshBody, !bytes.Equal(existing, freshBody), nil
	}

	startIdx := bytes.Index(existing, []byte(genStart))
	afterStart := startIdx + len(genStart)
	endIdx := bytes.Index(existing[afterStart:], []byte(genEnd))
	if endIdx < 0 {
		return freshBody, !bytes.Equal(existing, freshBody), nil
	}
	endIdx += afterStart

	freshStart := bytes.Index(freshBody, []byte(genStart))
	if freshStart < 0 {
		return nil, false, fmt.Errorf("mergeManaged: freshBody has no start marker")
	}
	freshAfterStart := freshStart + len(genStart)
	freshEndIdx := bytes.Index(freshBody[freshAfterStart:], []byte(genEnd))
	if freshEndIdx < 0 {
		return nil, false, fmt.Errorf("mergeManaged: freshBody has no end marker")
	}
	freshEndIdx += freshAfterStart
	freshInner := freshBody[freshAfterStart:freshEndIdx]

	var merged bytes.Buffer
	merged.Write(existing[:afterStart])
	merged.Write(freshInner)
	merged.Write(existing[endIdx:])

	return merged.Bytes(), !bytes.Equal(existing, merged.Bytes()), nil
}

// writeFile writes fresh to the absolute path absPath, merging with any
// existing content via mergeManaged. A missing file is created outright, no
// diff prompt (FR-11). An existing file whose managed-region markers are
// intact is refreshed surgically, no prompt needed — that's the point of
// the markers. An existing file with no markers (tampered or legacy) is
// only replaced when force is true; otherwise a Diff is returned and
// nothing is written. FileWrite.Path echoes absPath — callers relativize it
// for display.
func writeFile(absPath string, fresh []byte, force bool) (FileWrite, error) {
	existing, err := os.ReadFile(absPath)
	switch {
	case os.IsNotExist(err):
		if err := writeAtomic(absPath, fresh); err != nil {
			return FileWrite{}, err
		}
		return FileWrite{Path: absPath, Created: true, Changed: true}, nil
	case err != nil:
		return FileWrite{}, err
	}

	merged, changed, err := mergeManaged(existing, fresh)
	if err != nil {
		return FileWrite{}, fmt.Errorf("merging %s: %w", absPath, err)
	}
	if !changed {
		return FileWrite{Path: absPath, Skipped: true}, nil
	}

	if !hasMarkers(existing) && !force {
		diff := fmt.Sprintf("--- %s (on disk, no managed markers found)\n%s\n+++ %s (generated)\n%s\n",
			absPath, existing, absPath, merged)
		return FileWrite{Path: absPath, Diff: diff}, nil
	}

	if err := writeAtomic(absPath, merged); err != nil {
		return FileWrite{}, err
	}
	return FileWrite{Path: absPath, Changed: true}, nil
}

// writeAtomic writes content to absPath via a temp file + rename in the same
// directory, mirroring internal/architecture's write pattern.
func writeAtomic(absPath string, content []byte) error {
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".directives-tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	if _, werr := tmp.Write(content); werr != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return werr
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, absPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}
