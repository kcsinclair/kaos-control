// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kaos-control/kaos-control/internal/sandbox"
)

// OpenAITool defines a tool exposed to an OpenAI-compatible endpoint.
type OpenAITool struct {
	Type     string             `json:"type"`
	Function OpenAIFunctionDesc `json:"function"`
}

// OpenAIFunctionDesc describes a single callable function.
type OpenAIFunctionDesc struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// DefaultOpenAITools returns the standard v1 OpenAI tool schemas.
func DefaultOpenAITools() []OpenAITool {
	return []OpenAITool{
		{
			Type: "function",
			Function: OpenAIFunctionDesc{
				Name:        "read_file",
				Description: "Read the full contents of a file within the project.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "The relative path to the file to read.",
						},
					},
					"required": []string{"path"},
				},
			},
		},
		{
			Type: "function",
			Function: OpenAIFunctionDesc{
				Name:        "write_file",
				Description: "Write full content to a file within the project. The target path must be inside allowed write paths.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "The relative path to the file to write.",
						},
						"content": map[string]any{
							"type":        "string",
							"description": "The full content to write to the file.",
						},
					},
					"required": []string{"path", "content"},
				},
			},
		},
		{
			Type: "function",
			Function: OpenAIFunctionDesc{
				Name:        "list_dir",
				Description: "List files and subdirectories within a directory in the project.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "The relative path to the directory (e.g. '.' or 'lifecycle').",
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: OpenAIFunctionDesc{
				Name:        "grep",
				Description: "Search for a regex pattern in files within a project directory or file.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"pattern": map[string]any{
							"type":        "string",
							"description": "The regex pattern to search for.",
						},
						"path": map[string]any{
							"type":        "string",
							"description": "The relative path to search within (defaults to '.').",
						},
					},
					"required": []string{"pattern"},
				},
			},
		},
	}
}

// ToolExecutor executes sandboxed tools locally against a project root.
type ToolExecutor struct {
	ProjectRoot  string
	AllowedPaths []string
}

// Execute parses argsJSON and invokes the named tool against the sandbox.
func (e *ToolExecutor) Execute(ctx context.Context, name string, argsJSON string) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	switch name {
	case "read_file":
		var args struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return fmt.Sprintf("invalid arguments: %v", err), nil
		}
		return e.readFile(args.Path)

	case "write_file":
		var args struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return fmt.Sprintf("invalid arguments: %v", err), nil
		}
		return e.writeFile(args.Path, args.Content)

	case "list_dir":
		var args struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil && len(strings.TrimSpace(argsJSON)) > 0 {
			return fmt.Sprintf("invalid arguments: %v", err), nil
		}
		return e.listDir(args.Path)

	case "grep":
		var args struct {
			Pattern string `json:"pattern"`
			Path    string `json:"path"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return fmt.Sprintf("invalid arguments: %v", err), nil
		}
		return e.grep(args.Pattern, args.Path)

	default:
		return fmt.Sprintf("unknown tool: %s", name), nil
	}
}

func (e *ToolExecutor) readFile(relPath string) (string, error) {
	cleanPath := strings.TrimPrefix(relPath, "/")
	if cleanPath == "" {
		return "path is required", nil
	}
	resolved, err := sandbox.Resolve(e.ProjectRoot, cleanPath)
	if err != nil {
		return fmt.Sprintf("permission denied: %v", err), nil
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Sprintf("file not found: %s", cleanPath), nil
		}
		return fmt.Sprintf("read error: %v", err), nil
	}
	return string(data), nil
}

func (e *ToolExecutor) writeFile(relPath string, content string) (string, error) {
	cleanPath := strings.TrimPrefix(relPath, "/")
	if cleanPath == "" {
		return "path is required", nil
	}
	resolved, err := sandbox.Resolve(e.ProjectRoot, cleanPath)
	if err != nil {
		return fmt.Sprintf("permission denied: %v", err), nil
	}

	if len(e.AllowedPaths) > 0 {
		matched := false
		for _, p := range e.AllowedPaths {
			if pathHasPrefix(cleanPath, p) {
				matched = true
				break
			}
		}
		if !matched {
			return "permission denied: path is outside allowed_write_paths", nil
		}
	}

	dir := filepath.Dir(resolved)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Sprintf("failed to create parent directory: %v", err), nil
	}
	if err := os.WriteFile(resolved, []byte(content), 0o644); err != nil {
		return fmt.Sprintf("failed to write file: %v", err), nil
	}
	return fmt.Sprintf("file written successfully: %s", cleanPath), nil
}

func (e *ToolExecutor) listDir(relPath string) (string, error) {
	cleanPath := strings.TrimPrefix(relPath, "/")
	if cleanPath == "" {
		cleanPath = "."
	}
	resolved, err := sandbox.Resolve(e.ProjectRoot, cleanPath)
	if err != nil {
		return fmt.Sprintf("permission denied: %v", err), nil
	}
	entries, err := os.ReadDir(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Sprintf("directory not found: %s", cleanPath), nil
		}
		return fmt.Sprintf("list directory error: %v", err), nil
	}
	if len(entries) == 0 {
		return "(empty directory)", nil
	}
	var b strings.Builder
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		b.WriteString(name + "\n")
	}
	return strings.TrimRight(b.String(), "\n"), nil
}

func (e *ToolExecutor) grep(pattern string, relPath string) (string, error) {
	if pattern == "" {
		return "pattern is required", nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Sprintf("invalid regex: %v", err), nil
	}
	cleanPath := strings.TrimPrefix(relPath, "/")
	if cleanPath == "" {
		cleanPath = "."
	}
	resolved, err := sandbox.Resolve(e.ProjectRoot, cleanPath)
	if err != nil {
		return fmt.Sprintf("permission denied: %v", err), nil
	}

	info, err := os.Stat(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Sprintf("path not found: %s", cleanPath), nil
		}
		return fmt.Sprintf("grep error: %v", err), nil
	}

	var results []string
	const maxMatches = 500

	searchFile := func(path string, displayPath string) error {
		f, err := os.Open(path)
		if err != nil {
			return nil // ignore unreadable files
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		lineNum := 1
		for scanner.Scan() {
			line := scanner.Text()
			if re.MatchString(line) {
				results = append(results, fmt.Sprintf("%s:%d:%s", displayPath, lineNum, line))
				if len(results) >= maxMatches {
					return fmt.Errorf("limit reached")
				}
			}
			lineNum++
		}
		return nil
	}

	if !info.IsDir() {
		_ = searchFile(resolved, cleanPath)
	} else {
		resolvedRoot, _ := filepath.EvalSymlinks(e.ProjectRoot)
		if resolvedRoot == "" {
			resolvedRoot = filepath.Clean(e.ProjectRoot)
		}
		_ = filepath.Walk(resolved, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				if info.Name() == ".git" {
					return filepath.SkipDir
				}
				return nil
			}
			rel, err := filepath.Rel(resolvedRoot, path)
			if err != nil {
				rel = path
			}
			if err := searchFile(path, rel); err != nil {
				return err
			}
			return nil
		})
	}

	if len(results) == 0 {
		return "(no matches found)", nil
	}
	return strings.Join(results, "\n"), nil
}
