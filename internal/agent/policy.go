// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"strings"
)

// PolicyConfig holds the permission parameters for a single agent run.
type PolicyConfig struct {
	// AllowedPaths are directory prefixes the agent may write to.
	// Derived from AgentConfig.AllowedPaths.
	AllowedPaths []string
	// LineagePaths, when non-empty, further restricts writes: the target path
	// must also have a prefix in this slice. Used to confine agents to the
	// lineage they are operating on.
	LineagePaths []string
	// BashAllowlist is the merged per-agent allowlist. When non-empty, bash
	// commands must match at least one entry (checked after BashDenylist).
	BashAllowlist []string
	// BashDenylist is the merged (default + per-agent) denylist. Bash commands
	// matching any entry are always denied.
	BashDenylist []string
	// ObserveOnly puts the policy engine in observe-only mode: the Decision
	// still reflects what would be decided, but callers are expected to always
	// allow the tool call.  See handlePermission for where this is enforced.
	ObserveOnly bool
}

// Decision is the result of a policy evaluation.
type Decision struct {
	Action string // "allow" or "deny"
	Reason string // human-readable explanation
	Rule   string // machine-readable rule that triggered the decision
}

// readOnlyTools is the set of tool names that are always permitted regardless
// of policy (FR13).
var readOnlyTools = map[string]bool{
	"Read":          true,
	"Glob":          true,
	"Grep":          true,
	"WebFetch":      true,
	"WebSearch":     true,
	"Agent":         true,
	"TodoWrite":     true,
	"NotebookEdit":  true,
	"LS":            true,
	"Bash":          false, // handled separately
	"Write":         false,
	"Edit":          false,
	"MultiEdit":     false,
	"NotebookWrite": false,
}

// fileMutatingTools is the set of tool names that write files.
var fileMutatingTools = map[string]bool{
	"Write":         true,
	"Edit":          true,
	"MultiEdit":     true,
	"NotebookWrite": true,
}

// Evaluate applies the policy and returns a Decision for the given tool call.
// toolInput is the decoded JSON payload Claude Code sends to the hook.
func Evaluate(cfg PolicyConfig, toolName string, toolInput map[string]any) Decision {
	// 1. Read-only tools → always allow (FR13).
	if allow, known := readOnlyTools[toolName]; known && allow {
		return Decision{Action: "allow", Reason: "read-only tool", Rule: "read_only"}
	}

	// 2. File-mutating tools → check AllowedPaths then LineagePaths (FR9, FR10).
	if fileMutatingTools[toolName] {
		filePath, _ := toolInput["file_path"].(string)
		if filePath == "" {
			// No file_path provided — allow conservatively (tool may handle it).
			return Decision{Action: "allow", Reason: "no file_path in input", Rule: "no_path"}
		}

		// Normalise to forward-slash, strip leading slash for prefix matching.
		filePath = strings.TrimLeft(filePath, "/")

		if len(cfg.AllowedPaths) > 0 {
			matched := false
			for _, p := range cfg.AllowedPaths {
				if pathHasPrefix(filePath, p) {
					matched = true
					break
				}
			}
			if !matched {
				return Decision{
					Action: "deny",
					Reason: "write target " + filePath + " is outside allowed_write_paths",
					Rule:   "allowed_paths",
				}
			}
		}

		if len(cfg.LineagePaths) > 0 {
			matched := false
			for _, p := range cfg.LineagePaths {
				if pathHasPrefix(filePath, p) {
					matched = true
					break
				}
			}
			if !matched {
				return Decision{
					Action: "deny",
					Reason: "write target " + filePath + " is outside lineage scope",
					Rule:   "lineage_scope",
				}
			}
		}

		return Decision{Action: "allow", Reason: "write target within allowed paths", Rule: "allowed_paths"}
	}

	// 3. Bash tool → denylist first, then allowlist (FR11).
	if toolName == "Bash" {
		command, _ := toolInput["command"].(string)

		// Check denylist (takes precedence over allowlist).
		for _, pattern := range cfg.BashDenylist {
			if matchGlob(pattern, command) {
				return Decision{
					Action: "deny",
					Reason: "command matches denylist pattern: " + pattern,
					Rule:   "bash_denylist",
				}
			}
		}

		// If allowlist is non-empty, command must match at least one entry.
		if len(cfg.BashAllowlist) > 0 {
			matched := false
			for _, pattern := range cfg.BashAllowlist {
				if matchGlob(pattern, command) {
					matched = true
					break
				}
			}
			if !matched {
				return Decision{
					Action: "deny",
					Reason: "command does not match any bash_allowlist pattern",
					Rule:   "bash_allowlist",
				}
			}
		}

		return Decision{Action: "allow", Reason: "bash command permitted", Rule: "bash_allowed"}
	}

	// 4. All other tools → allow.
	return Decision{Action: "allow", Reason: "tool not restricted by policy", Rule: "default_allow"}
}

// pathHasPrefix reports whether target starts with the given prefix.
// An empty prefix matches everything. Leading and trailing slashes are
// normalised before comparison so both "internal/" and "internal" work
// as prefixes for "internal/agent/foo.go".
// For non-directory prefixes (e.g. a lineage stem like "lifecycle/plans/foo"),
// target matches if it equals the prefix or starts with the prefix (any char).
func pathHasPrefix(target, prefix string) bool {
	if prefix == "" {
		return true
	}
	prefix = strings.TrimLeft(prefix, "/")
	target = strings.TrimLeft(target, "/")
	prefix = strings.TrimRight(prefix, "/")
	if prefix == "" {
		return true
	}
	if target == prefix {
		return true
	}
	// Prefer directory-boundary match first (prefix + "/"), then bare prefix match.
	return strings.HasPrefix(target, prefix+"/") || strings.HasPrefix(target, prefix)
}

// matchGlob reports whether s matches the glob pattern, where '*' matches any
// sequence of characters (including path separators and spaces). This differs
// from filepath.Match where '*' does not cross directory separators.
func matchGlob(pattern, s string) bool {
	for len(pattern) > 0 {
		if pattern[0] == '*' {
			pattern = pattern[1:]
			if len(pattern) == 0 {
				return true // trailing star matches everything
			}
			// Try matching the remainder of the pattern starting at every position.
			for i := 0; i <= len(s); i++ {
				if matchGlob(pattern, s[i:]) {
					return true
				}
			}
			return false
		}
		if len(s) == 0 || s[0] != pattern[0] {
			return false
		}
		pattern = pattern[1:]
		s = s[1:]
	}
	return len(s) == 0
}
