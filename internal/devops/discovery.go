// SPDX-License-Identifier: AGPL-3.0-or-later

package devops

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const defaultStepTimeout = 60 * time.Second

// pipelineYAML is the raw structure parsed from a pipeline YAML file.
type pipelineYAML struct {
	Name  string     `yaml:"name"`
	Type  string     `yaml:"type"`
	Steps []stepYAML `yaml:"steps"`
}

type stepYAML struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Command     string `yaml:"command"`
	Timeout     string `yaml:"timeout,omitempty"`
}

// Discover reads all *.yaml files in dir, parses them as pipeline definitions,
// and returns valid pipelines plus a list of per-file parse warnings. Malformed
// files are excluded from the result but described in the error list.
// If dir does not exist, both slices are nil.
func Discover(dir string) ([]Pipeline, []error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, []error{fmt.Errorf("devops: reading dir %s: %w", dir, err)}
	}

	var pipelines []Pipeline
	var warnings []error

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		p, err := parsePipelineFile(path)
		if err != nil {
			w := fmt.Errorf("devops: skipping %s: %w", entry.Name(), err)
			warnings = append(warnings, w)
			slog.Warn("devops: pipeline parse warning", "file", entry.Name(), "err", err)
			continue
		}
		pipelines = append(pipelines, *p)
	}

	return pipelines, warnings
}

// ValidateDefinition parses and validates a pipeline definition from raw YAML
// bytes. It returns the parsed Pipeline (without a slug, since the slug is
// derived from the filename) or an error describing the first validation
// failure. The slug field of the returned Pipeline is always empty; callers
// must set it from the target filename.
func ValidateDefinition(data []byte) (*Pipeline, error) {
	var raw pipelineYAML
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}

	if raw.Name == "" {
		return nil, fmt.Errorf("missing required field: name")
	}
	if raw.Type == "" {
		return nil, fmt.Errorf("missing required field: type")
	}
	if len(raw.Steps) == 0 {
		return nil, fmt.Errorf("missing required field: steps (must have at least one step)")
	}

	steps := make([]Step, 0, len(raw.Steps))
	for i, s := range raw.Steps {
		if s.Name == "" {
			return nil, fmt.Errorf("step[%d] missing required field: name", i)
		}
		if s.Command == "" {
			return nil, fmt.Errorf("step[%d] %q missing required field: command", i, s.Name)
		}
		timeout := defaultStepTimeout
		if s.Timeout != "" {
			d, err := time.ParseDuration(s.Timeout)
			if err != nil {
				return nil, fmt.Errorf("step[%d] %q invalid timeout %q: %w", i, s.Name, s.Timeout, err)
			}
			timeout = d
		}
		steps = append(steps, Step{
			Name:        s.Name,
			Description: s.Description,
			Command:     s.Command,
			Timeout:     timeout,
		})
	}

	return &Pipeline{
		Name:  raw.Name,
		Type:  raw.Type,
		Steps: steps,
	}, nil
}

func parsePipelineFile(path string) (*Pipeline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	p, err := ValidateDefinition(data)
	if err != nil {
		return nil, err
	}

	p.Slug = strings.TrimSuffix(filepath.Base(path), ".yaml")
	return p, nil
}
