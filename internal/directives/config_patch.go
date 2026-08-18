// SPDX-License-Identifier: AGPL-3.0-or-later

package directives

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kaos-control/kaos-control/internal/architecture"
	"github.com/kaos-control/kaos-control/internal/config"
	"gopkg.in/yaml.v3"
)

// standardAgentRoles maps each of the six standard agents (by name) to the
// AgentConfig role it fulfils. Roles that also appear as a key in a stack
// profile's Roles map ("backend-developer", "frontend-developer",
// "test-developer") get their allowed_write_paths and build/lint/test
// commands tuned from it; "analyst" (used by both requirements-analyst and
// planning-analyst) and "qa" are not stack-specific, so only their
// architecture-awareness clause is (re)generated.
var standardAgentRoles = map[string]string{
	"requirements-analyst": "analyst",
	"planning-analyst":     "analyst",
	"backend-developer":    "backend-developer",
	"frontend-developer":   "frontend-developer",
	"test-developer":       "test-developer",
	"qa":                   "qa",
}

// constantLifecyclePaths lists the lifecycle/ paths each standard agent
// keeps write access to regardless of stack (its own stage's artifacts plus
// lifecycle/architecture/decisions for proposing ADRs), merged with any
// stack write_paths for that agent's role.
var constantLifecyclePaths = map[string][]string{
	"requirements-analyst": {"lifecycle/requirements", "lifecycle/ideas", "lifecycle/architecture/decisions"},
	"planning-analyst":     {"lifecycle/backend-plans", "lifecycle/frontend-plans", "lifecycle/test-plans", "lifecycle/requirements", "lifecycle/architecture/decisions"},
	"backend-developer":    {"lifecycle/backend-plans", "lifecycle/architecture/decisions"},
	"frontend-developer":   {"lifecycle/frontend-plans", "lifecycle/architecture/decisions"},
	"test-developer":       {"lifecycle/tests", "lifecycle/test-plans", "lifecycle/architecture/decisions"},
	"qa":                   {"lifecycle/defects", "lifecycle/architecture/decisions"},
}

// PatchAgentConfigResult reports what PatchAgentConfig changed.
type PatchAgentConfigResult struct {
	Changed bool
	// Disabled lists standard agent names disabled because their stack role
	// is marked required: false (OQ-4) — a static frontend-only stack
	// disables backend-developer, for example.
	Disabled []string
}

// PatchAgentConfig updates the six standard agents in
// projectRoot/lifecycle/config.yaml for the stack in m: each developer
// role's allowed_write_paths and build/lint/test commands come from the
// stack profile, every standard agent's prompt carries an
// architecture-awareness clause (FR-8), and a role marked required: false
// is disabled (an `enabled: false` field is added; nothing else in
// kaos-control consumes it yet — that's left to the agent-selection code
// that owns run dispatch).
//
// Edits are scoped to exactly those six agents, matched by name — roles,
// stages, users, kanban, dashboard, and any user-added agent are left
// untouched. The document is read and re-emitted as a yaml.Node tree (not
// round-tripped through the typed config.Project struct) specifically to
// preserve comments and any keys this package doesn't model. Nothing is
// written unless the result actually differs, and the patched content is
// validated via a temp project root and config.LoadProject before the real
// file is touched (FR-9): a patch that wouldn't parse never reaches disk.
func PatchAgentConfig(projectRoot string, m DirectiveModel) (PatchAgentConfigResult, error) {
	cfgPath := filepath.Join(projectRoot, "lifecycle", "config.yaml")
	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		return PatchAgentConfigResult{}, fmt.Errorf("reading %s: %w", cfgPath, err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return PatchAgentConfigResult{}, fmt.Errorf("parsing %s: %w", cfgPath, err)
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return PatchAgentConfigResult{}, fmt.Errorf("%s: not a YAML mapping document", cfgPath)
	}
	root := doc.Content[0]

	agentsNode := mappingValue(root, "agents")
	if agentsNode == nil || agentsNode.Kind != yaml.SequenceNode {
		// No agents sequence to patch — a valid state (a config that doesn't
		// declare the standard agents has nothing to tune to the stack). Skip
		// gracefully so directive-file generation still succeeds rather than
		// failing the whole scaffold.
		return PatchAgentConfigResult{}, nil
	}

	var disabled []string
	for _, agentNode := range agentsNode.Content {
		if agentNode.Kind != yaml.MappingNode {
			continue
		}
		nameNode := mappingValue(agentNode, "name")
		if nameNode == nil {
			continue
		}
		role, ok := standardAgentRoles[nameNode.Value]
		if !ok {
			continue // not one of the six standard agents — leave untouched
		}

		rp, hasStackRole := m.Stack.Roles[role]
		disabledForStack := hasStackRole && !rp.IsRequired()
		setDisabled(agentNode, disabledForStack)
		if disabledForStack {
			disabled = append(disabled, nameNode.Value)
		}

		writePaths := dedupStrings(append(append([]string{}, rp.WritePaths...), constantLifecyclePaths[nameNode.Value]...))
		setStringSequence(agentNode, "allowed_write_paths", writePaths)

		cmds := ""
		if hasStackRole {
			cmds = commandBlock(rp)
		}
		patchPromptTemplates(agentNode, architectureClause(role), cmds)
	}
	sort.Strings(disabled)

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return PatchAgentConfigResult{}, fmt.Errorf("encoding patched %s: %w", cfgPath, err)
	}
	if err := enc.Close(); err != nil {
		return PatchAgentConfigResult{}, fmt.Errorf("closing yaml encoder: %w", err)
	}
	newRaw := buf.Bytes()

	if bytes.Equal(raw, newRaw) {
		return PatchAgentConfigResult{Changed: false, Disabled: disabled}, nil
	}

	if err := validatePatchedConfig(newRaw); err != nil {
		return PatchAgentConfigResult{}, fmt.Errorf("patched %s would not parse as a project config, not writing: %w", cfgPath, err)
	}

	if err := writeAtomic(cfgPath, newRaw); err != nil {
		return PatchAgentConfigResult{}, fmt.Errorf("writing %s: %w", cfgPath, err)
	}

	return PatchAgentConfigResult{Changed: true, Disabled: disabled}, nil
}

// validatePatchedConfig confirms newRaw still parses as a valid project
// config via config.LoadProject (FR-9), by staging it into an isolated temp
// project root rather than the real one, so a parse failure never touches
// the real lifecycle/config.yaml.
func validatePatchedConfig(newRaw []byte) error {
	tmpRoot, err := os.MkdirTemp("", "kaos-directives-validate-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpRoot)

	tmpCfgPath := filepath.Join(tmpRoot, "lifecycle", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(tmpCfgPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(tmpCfgPath, newRaw, 0o644); err != nil {
		return err
	}

	_, err = config.LoadProject(tmpRoot)
	return err
}

// mappingValue returns the value node for key within mapping node, or nil
// if node isn't a mapping or the key isn't present.
func mappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

// removeMappingKey deletes key (and its value) from mapping node, if present.
func removeMappingKey(node *yaml.Node, key string) {
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			node.Content = append(node.Content[:i], node.Content[i+2:]...)
			return
		}
	}
}

// setDisabled adds or removes an `enabled: false` key on agentNode.
// Re-enabling (disabled=false) removes the key entirely rather than writing
// `enabled: true`, so an agent the user never touched stays untouched.
func setDisabled(agentNode *yaml.Node, disabled bool) {
	existing := mappingValue(agentNode, "enabled")
	if !disabled {
		if existing != nil {
			removeMappingKey(agentNode, "enabled")
		}
		return
	}
	if existing != nil {
		existing.Value = "false"
		existing.Tag = "!!bool"
		return
	}
	agentNode.Content = append(agentNode.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "enabled"},
		&yaml.Node{Kind: yaml.ScalarNode, Value: "false", Tag: "!!bool"},
	)
}

// setStringSequence sets key within mapping node to a block sequence of
// plain scalars built from values, creating the key if absent.
func setStringSequence(mapping *yaml.Node, key string, values []string) {
	val := mappingValue(mapping, key)
	if val == nil {
		val = &yaml.Node{}
		mapping.Content = append(mapping.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: key}, val)
	}
	val.Kind = yaml.SequenceNode
	val.Tag = "!!seq"
	val.Style = 0
	items := make([]*yaml.Node, len(values))
	for i, v := range values {
		items[i] = &yaml.Node{Kind: yaml.ScalarNode, Value: v, Tag: "!!str"}
	}
	val.Content = items
}

// dedupStrings drops empty and repeated entries from items, preserving the
// order of first occurrence.
func dedupStrings(items []string) []string {
	seen := make(map[string]bool, len(items))
	out := make([]string, 0, len(items))
	for _, it := range items {
		if it == "" || seen[it] {
			continue
		}
		seen[it] = true
		out = append(out, it)
	}
	return out
}

// patchPromptTemplates rewrites the managed clause+commands sub-block of
// every entry in agentNode's prompt_templates mapping, leaving the
// hand-written parts of each template untouched.
func patchPromptTemplates(agentNode *yaml.Node, clause, commands string) {
	pt := mappingValue(agentNode, "prompt_templates")
	if pt == nil || pt.Kind != yaml.MappingNode {
		return
	}
	fresh := renderManagedClause(clause, commands)
	for i := 1; i < len(pt.Content); i += 2 {
		valNode := pt.Content[i]
		merged, changed := mergeCommandBlock(valNode.Value, fresh)
		if changed {
			valNode.Value = merged
		}
	}
}

// renderManagedClause wraps the architecture-awareness clause (and, for
// stack-tuned roles, the build/lint/test command block) in the same
// managed-region markers RenderAgents uses (M2), so a prompt_templates
// value's generated sub-block is refreshed the same surgical way.
func renderManagedClause(clause, commands string) []byte {
	var b strings.Builder
	b.WriteString(genStart)
	b.WriteString("\n")
	b.WriteString(clause)
	if commands != "" {
		b.WriteString("\n\n")
		b.WriteString(commands)
	}
	b.WriteString("\n")
	b.WriteString(genEnd)
	return []byte(b.String())
}

// mergeCommandBlock splices freshBlock into promptValue. If promptValue
// already contains the marker pair (a prior generation), only that region
// is replaced. Otherwise (first run, or hand-written legacy prose) the
// block is appended, leaving all existing hand-written prose intact.
func mergeCommandBlock(promptValue string, freshBlock []byte) (string, bool) {
	existing := []byte(promptValue)
	if hasMarkers(existing) {
		merged, changed, err := mergeManaged(existing, freshBlock)
		if err == nil {
			return string(merged), changed
		}
	}
	trimmed := strings.TrimRight(promptValue, "\n")
	return trimmed + "\n\n" + string(freshBlock) + "\n", true
}

// architectureClause returns the role-flavoured architecture-awareness
// clause required by FR-8: analysts flag architecture-breaking
// requirements; qa judges failures against the recorded architecture;
// every other (developer) role conforms to standards/decisions and
// proposes an ADR rather than deviating silently.
func architectureClause(role string) string {
	switch role {
	case "analyst":
		return "Before writing, read lifecycle/architecture/ — the architecture " +
			"summary, the chosen architecture and tech stack, the ADRs in " +
			"decisions/, and the standards/ — and keep the output consistent " +
			"with them. Explicitly flag any architecture-breaking requirements " +
			"against lifecycle/architecture/architecture-summary.md. If a " +
			"requirement genuinely cannot be met within the recorded " +
			"architecture, stack, or standards, do not deviate silently: " +
			"propose a new ADR in lifecycle/architecture/decisions/ (type: adr) " +
			"capturing the decision, context, and consequences."
	case "qa":
		return "Before triaging failures, read lifecycle/architecture/ — the " +
			"architecture summary, the chosen architecture and tech stack, the " +
			"ADRs in decisions/, and the standards/ — and use them to judge " +
			"whether a failure reflects a genuine defect or a deliberate, " +
			"defensible deviation. If it's the latter, do not silently accept " +
			"it: propose a new ADR in lifecycle/architecture/decisions/ (type: " +
			"adr) capturing the decision, context, and consequences."
	default:
		return "Follow the architecture standards and ADRs in " +
			"lifecycle/architecture/ (standards/ and decisions/). If this work " +
			"forces a deviation from the recorded architecture or standards, do " +
			"not deviate silently: propose a new ADR in " +
			"lifecycle/architecture/decisions/ (type: adr) capturing the " +
			"decision, context, and consequences."
	}
}

// commandBlock renders the build/lint/test commands for a stack-tuned role,
// omitting any command the stack profile leaves empty. Returns "" if the
// role defines none (a disabled role typically has none).
func commandBlock(rp architecture.RoleProfile) string {
	var lines []string
	if rp.Build != "" {
		lines = append(lines, "  - "+rp.Build)
	}
	if rp.Lint != "" {
		lines = append(lines, "  - "+rp.Lint)
	}
	if rp.Test != "" {
		lines = append(lines, "  - "+rp.Test)
	}
	if len(lines) == 0 {
		return ""
	}
	return "After each milestone, run:\n" + strings.Join(lines, "\n") +
		"\nAll must pass before the milestone is considered complete."
}
