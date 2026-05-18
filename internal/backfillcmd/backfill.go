// SPDX-License-Identifier: AGPL-3.0-or-later

// Package backfillcmd implements the `kaos-control backfill-created`
// subcommand. It walks every markdown artifact under lifecycle/ and adds a
// `created:` frontmatter field where one is missing, using the file's
// filesystem birth time (or mtime as a fallback) so downstream views like
// the artifact list's Created column have a meaningful value without
// walking git history.
package backfillcmd

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Run is the entrypoint for `kaos-control backfill-created <path>`. If <path>
// is omitted the current working directory is used.
func Run(args []string) error {
	fs := flag.NewFlagSet("backfill-created", flag.ContinueOnError)
	dryRun := fs.Bool("dry-run", false, "show what would be changed without writing")
	verbose := fs.Bool("v", false, "log every file decision")
	if err := fs.Parse(args); err != nil {
		return err
	}

	root := "."
	if fs.NArg() > 0 {
		root = fs.Arg(0)
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolving root %q: %w", root, err)
	}
	lcDir := filepath.Join(absRoot, "lifecycle")
	if _, err := os.Stat(lcDir); err != nil {
		return fmt.Errorf("expected %s to be a lifecycle project root: %w", absRoot, err)
	}

	var (
		seen, alreadyHad, updated, failed int
	)
	walkErr := filepath.WalkDir(lcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		seen++
		had, wrote, ferr := processFile(path, *dryRun)
		switch {
		case ferr != nil:
			failed++
			fmt.Fprintf(os.Stderr, "  fail     %s: %v\n", relTo(absRoot, path), ferr)
		case had:
			alreadyHad++
			if *verbose {
				fmt.Printf("  skip     %s (already had created:)\n", relTo(absRoot, path))
			}
		case wrote:
			updated++
			fmt.Printf("  updated  %s\n", relTo(absRoot, path))
		default:
			if *verbose {
				fmt.Printf("  no-op    %s (no frontmatter block?)\n", relTo(absRoot, path))
			}
		}
		return nil
	})
	if walkErr != nil {
		return fmt.Errorf("walking %s: %w", lcDir, walkErr)
	}

	verb := "would update"
	if !*dryRun {
		verb = "updated"
	}
	fmt.Printf("\nScanned %d files: %d already had created:, %s %d, %d failed.\n",
		seen, alreadyHad, verb, updated, failed)
	if failed > 0 {
		return fmt.Errorf("%d file(s) failed", failed)
	}
	return nil
}

// fmDelim matches the opening or closing line of a YAML frontmatter block.
var fmDelim = regexp.MustCompile(`^---\s*$`)

// createdLine matches an existing `created:` field at the start of a line.
var createdLine = regexp.MustCompile(`(?m)^created:\s`)

// processFile is the per-file worker. It returns:
//   - hadCreated=true if the file already has a `created:` field (no change).
//   - wrote=true if the file's frontmatter was rewritten (or would be in dry-run).
//   - err if anything failed.
func processFile(path string, dryRun bool) (hadCreated, wrote bool, err error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false, false, err
	}

	// Find the frontmatter block. We only ever modify files that start with
	// a `---` delimiter followed by a closing `---`.
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	var startLine, endLine = -1, -1
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if fmDelim.MatchString(line) {
			if startLine == -1 {
				startLine = lineNum
				continue
			}
			endLine = lineNum
			break
		}
	}
	if startLine == -1 || endLine == -1 {
		return false, false, nil
	}

	if createdLine.Match(raw) {
		return true, false, nil
	}

	birth := fileBirthTime(path)
	createdValue := fmt.Sprintf(`created: "%s"`, birth.Format(time.RFC3339))

	// Insert the line immediately after the opening `---`.
	lines := bytes.SplitAfter(raw, []byte("\n"))
	insertAt := startLine
	// The line just after the opening delimiter is index [startLine] in the
	// 0-indexed slice if we treat startLine as 1-indexed.
	out := make([]byte, 0, len(raw)+len(createdValue)+1)
	for i, l := range lines {
		out = append(out, l...)
		if i+1 == insertAt {
			out = append(out, []byte(createdValue+"\n")...)
		}
	}

	if dryRun {
		return false, true, nil
	}

	// Atomic write: temp file in the same dir, fsync, rename.
	dir, base := filepath.Split(path)
	tmp, err := os.CreateTemp(dir, "."+base+".bf-")
	if err != nil {
		return false, false, err
	}
	tmpPath := tmp.Name()
	if _, werr := tmp.Write(out); werr != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return false, false, werr
	}
	if cerr := tmp.Close(); cerr != nil {
		_ = os.Remove(tmpPath)
		return false, false, cerr
	}
	// Preserve original mode.
	if info, ierr := os.Stat(path); ierr == nil {
		_ = os.Chmod(tmpPath, info.Mode())
	}
	if rerr := os.Rename(tmpPath, path); rerr != nil {
		_ = os.Remove(tmpPath)
		return false, false, rerr
	}
	return false, true, nil
}

func relTo(root, path string) string {
	if r, err := filepath.Rel(root, path); err == nil {
		return r
	}
	return path
}
