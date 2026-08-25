// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// AgentProviderPatch describes one agent's provider/model mutation applied by
// PatchAgentProviders. Provider and Model always overwrite the agent's
// current values. PrimaryProvider/PrimaryModel are pointers so the patch can
// distinguish "leave untouched" (nil) from "set" (*v != "") from "delete the
// key" (*v == "").
type AgentProviderPatch struct {
	AgentName       string
	Provider        string
	Model           string
	PrimaryProvider *string
	PrimaryModel    *string
}

// PatchAgentProviders atomically rewrites the provider/model (and, where
// requested, primary_provider/primary_model) fields of the named agents in
// projectRoot/lifecycle/config.yaml, preserving comments, formatting, and
// every other section of the document. The document is read and re-emitted
// as a yaml.Node tree (not round-tripped through the typed Project struct)
// so only the targeted agent entries are touched.
//
// The patched document is validated via LoadProject against a temporary
// project root before the real file is written (fail closed — an invalid
// patch never reaches disk), then written with a temp-file + rename swap.
func PatchAgentProviders(projectRoot string, patches []AgentProviderPatch) error {
	if len(patches) == 0 {
		return nil
	}

	cfgPath := filepath.Join(projectRoot, "lifecycle", "config.yaml")
	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", cfgPath, err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("parsing %s: %w", cfgPath, err)
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return fmt.Errorf("%s: not a YAML mapping document", cfgPath)
	}
	root := doc.Content[0]

	agentsNode := patchMappingValue(root, "agents")
	if agentsNode == nil || agentsNode.Kind != yaml.SequenceNode {
		return fmt.Errorf("%s: no agents sequence found", cfgPath)
	}

	byName := make(map[string]*yaml.Node, len(agentsNode.Content))
	for _, agentNode := range agentsNode.Content {
		if agentNode.Kind != yaml.MappingNode {
			continue
		}
		if nameNode := patchMappingValue(agentNode, "name"); nameNode != nil {
			byName[nameNode.Value] = agentNode
		}
	}

	for _, p := range patches {
		agentNode, ok := byName[p.AgentName]
		if !ok {
			return fmt.Errorf("agent %q not found in %s", p.AgentName, cfgPath)
		}
		patchScalar(agentNode, "provider", p.Provider)
		patchScalar(agentNode, "model", p.Model)
		if p.PrimaryProvider != nil {
			if *p.PrimaryProvider == "" {
				removeMappingKeyLocal(agentNode, "primary_provider")
			} else {
				patchScalar(agentNode, "primary_provider", *p.PrimaryProvider)
			}
		}
		if p.PrimaryModel != nil {
			if *p.PrimaryModel == "" {
				removeMappingKeyLocal(agentNode, "primary_model")
			} else {
				patchScalar(agentNode, "primary_model", *p.PrimaryModel)
			}
		}
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return fmt.Errorf("encoding patched %s: %w", cfgPath, err)
	}
	if err := enc.Close(); err != nil {
		return fmt.Errorf("closing yaml encoder: %w", err)
	}
	newRaw := buf.Bytes()

	if err := validatePatchedProjectConfig(newRaw); err != nil {
		return fmt.Errorf("patched %s would not parse as a project config, not writing: %w", cfgPath, err)
	}

	return writeAtomicConfig(cfgPath, newRaw)
}

// validatePatchedProjectConfig confirms newRaw still parses as a valid
// project config via LoadProject, staged into an isolated temp project root
// so a parse failure never touches the real lifecycle/config.yaml.
func validatePatchedProjectConfig(newRaw []byte) error {
	tmpRoot, err := os.MkdirTemp("", "kaos-provider-patch-validate-*")
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

	_, err = LoadProject(tmpRoot)
	return err
}

// writeAtomicConfig writes content to absPath via a temp file in the same
// directory followed by os.Rename, so a crash mid-write never leaves a
// corrupt lifecycle/config.yaml.
func writeAtomicConfig(absPath string, content []byte) error {
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".config-patch-tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, werr := tmp.Write(content); werr != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return werr
	}
	if cerr := tmp.Close(); cerr != nil {
		_ = os.Remove(tmpPath)
		return cerr
	}
	if rerr := os.Rename(tmpPath, absPath); rerr != nil {
		_ = os.Remove(tmpPath)
		return rerr
	}
	return nil
}

// patchMappingValue returns the value node for key within mapping node, or
// nil if node isn't a mapping or the key isn't present.
func patchMappingValue(node *yaml.Node, key string) *yaml.Node {
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

// removeMappingKeyLocal deletes key (and its value) from mapping node, if present.
func removeMappingKeyLocal(node *yaml.Node, key string) {
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			node.Content = append(node.Content[:i], node.Content[i+2:]...)
			return
		}
	}
}

// patchScalar sets key within mapping node to a plain string scalar,
// creating the key (appended at the end) if it isn't already present.
func patchScalar(mapping *yaml.Node, key, value string) {
	val := patchMappingValue(mapping, key)
	if val == nil {
		mapping.Content = append(mapping.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: key},
			&yaml.Node{Kind: yaml.ScalarNode, Value: value, Tag: "!!str"},
		)
		return
	}
	val.Kind = yaml.ScalarNode
	val.Tag = "!!str"
	val.Value = value
}
