// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

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

// ToolCall represents an OpenAI function tool call.
type ToolCall struct {
	Index    *int             `json:"index,omitempty"`
	ID       string           `json:"id"`
	Type     string           `json:"type"` // "function"
	Function FunctionCallInfo `json:"function"`
}

// FunctionCallInfo contains the function name and JSON arguments string.
type FunctionCallInfo struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
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
// BashTool is the shell tool for the openai-compatible driver. It is
// deliberately NOT part of DefaultOpenAITools: v1 excluded shell because small
// local models are the least trustworthy with it. The driver advertises this
// tool only when the agent has a non-empty bash_allowlist, so shell access is
// opt-in per agent and the allowlist doubles as the grant.
func BashTool() OpenAITool {
	return OpenAITool{
		Type: "function",
		Function: OpenAIFunctionDesc{
			Name: "bash",
			Description: "Run a shell command in the project root. Only commands " +
				"permitted by this agent's configured allowlist will execute; " +
				"anything else is refused and reported back to you.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]any{
						"type":        "string",
						"description": "The shell command to run, e.g. 'make test-unit'.",
					},
				},
				"required": []string{"command"},
			},
		},
	}
}

// bashTimeout bounds a single bash tool call so one hung command cannot consume
// the whole run.
const bashTimeout = 5 * time.Minute

// bashOutputLimit caps how much command output is fed back to the model.
const bashOutputLimit = 16 * 1024

type ToolExecutor struct {
	ProjectRoot  string
	AllowedPaths []string
	// Policy governs bash calls. Nil means bash is not available: every call is
	// refused. Reuses the same PolicyConfig/Evaluate path as claude-mediated so
	// there is exactly one shell policy implementation.
	Policy *PolicyConfig
	// OnDenial is called for each refused bash call so the run record can carry
	// denied_tool_calls, matching the mediated driver.
	OnDenial func(d Decision, toolName string, toolInput map[string]any)
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

	case "bash":
		var args struct {
			Command string `json:"command"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return fmt.Sprintf("invalid arguments: %v", err), nil
		}
		return e.bash(ctx, args.Command)

	default:
		return fmt.Sprintf("unknown tool: %s", name), nil
	}
}

// bash evaluates a shell command against the agent's policy and, if permitted,
// runs it in the project root. A refusal is returned to the model as tool
// output (not an error) so the agent can adapt instead of the run dying.
func (e *ToolExecutor) bash(ctx context.Context, command string) (string, error) {
	if strings.TrimSpace(command) == "" {
		return "permission denied: empty command", nil
	}
	if e.Policy == nil {
		return "permission denied: bash is not enabled for this agent " +
			"(set bash_allowlist in the agent config to enable it)", nil
	}

	// Policy name is "Bash" — the same key claude-mediated uses — so both
	// drivers share one denylist/allowlist implementation.
	toolInput := map[string]any{"command": command}
	decision := Evaluate(*e.Policy, "Bash", toolInput)
	if decision.Action == "deny" {
		if e.OnDenial != nil {
			e.OnDenial(decision, "bash", toolInput)
		}
		return fmt.Sprintf("permission denied: %s (rule: %s)", decision.Reason, decision.Rule), nil
	}

	cmdCtx, cancel := context.WithTimeout(ctx, bashTimeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "sh", "-c", command)
	cmd.Dir = e.ProjectRoot
	out, err := cmd.CombinedOutput()

	text := string(out)
	if len(text) > bashOutputLimit {
		text = text[:bashOutputLimit] + "\n... [output truncated]"
	}
	if cmdCtx.Err() == context.DeadlineExceeded {
		return fmt.Sprintf("command timed out after %s\n%s", bashTimeout, text), nil
	}
	if err != nil {
		// A non-zero exit is information for the model (a failing test suite is
		// the normal case for a QA agent), not a run-ending error.
		return fmt.Sprintf("exit error: %v\n%s", err, text), nil
	}
	if strings.TrimSpace(text) == "" {
		return "(command produced no output)", nil
	}
	return text, nil
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
