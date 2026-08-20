// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// filesFromRunLog derives the set of files an agent created or modified from
// its own stream-json run log — the file-mutating tool_use events (Write,
// Edit, MultiEdit, NotebookWrite). This is git-independent: it works in
// projects without a git repo and reflects exactly what the agent did, rather
// than relying on `git status`. Paths are returned project-root-relative
// (forward-slash), de-duplicated and sorted; absolute paths that escape the
// project root are dropped.
func filesFromRunLog(logPath, projectRoot string) []string {
	f, err := os.Open(logPath)
	if err != nil {
		return nil
	}
	defer f.Close() //nolint:errcheck

	seen := map[string]bool{}
	var out []string

	sc := bufio.NewScanner(f)
	// Stream-json lines can be large (full tool inputs); grow the buffer.
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)

	for sc.Scan() {
		var ev struct {
			Type    string `json:"type"`
			Message struct {
				Content []struct {
					Type  string         `json:"type"`
					Name  string         `json:"name"`
					Input map[string]any `json:"input"`
				} `json:"content"`
			} `json:"message"`
		}
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			continue
		}
		if ev.Type != "assistant" {
			continue
		}
		for _, c := range ev.Message.Content {
			if c.Type != "tool_use" || !fileMutatingTools[c.Name] {
				continue
			}
			p, _ := c.Input["file_path"].(string)
			if p == "" {
				p, _ = c.Input["notebook_path"].(string) // NotebookWrite
			}
			rel := projectRelPath(p, projectRoot)
			if rel == "" || seen[rel] {
				continue
			}
			seen[rel] = true
			out = append(out, rel)
		}
	}

	sort.Strings(out)
	return out
}

// projectRelPath converts a tool_use file path (absolute, as Claude Code emits,
// or already relative) to a clean project-root-relative forward-slash path.
// Returns "" for an empty path or one that escapes the project root.
func projectRelPath(p, projectRoot string) string {
	if p == "" {
		return ""
	}
	if filepath.IsAbs(p) {
		if projectRoot == "" {
			return ""
		}
		rel, err := filepath.Rel(projectRoot, filepath.Clean(p))
		if err != nil || strings.HasPrefix(rel, "..") {
			return ""
		}
		return filepath.ToSlash(rel)
	}
	return strings.TrimLeft(filepath.ToSlash(p), "/")
}

// unionStrings merges b into a, de-duplicating and returning a sorted slice.
func unionStrings(a, b []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range append(append([]string{}, a...), b...) {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
