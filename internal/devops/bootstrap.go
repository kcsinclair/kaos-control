// SPDX-License-Identifier: AGPL-3.0-or-later

package devops

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/kaos-control/kaos-control/internal/architecture"
)

// bootstrapKinds are the CI pipelines derived from a stack profile: each picks
// the matching per-role command (build/lint/test) from a RoleProfile.
var bootstrapKinds = []struct {
	name    string
	typ     string
	timeout string
	pick    func(architecture.RoleProfile) string
}{
	{name: "Build", typ: "build", timeout: "600s", pick: func(r architecture.RoleProfile) string { return r.Build }},
	{name: "Lint", typ: "lint", timeout: "120s", pick: func(r architecture.RoleProfile) string { return r.Lint }},
	{name: "Test", typ: "test", timeout: "600s", pick: func(r architecture.RoleProfile) string { return r.Test }},
}

// BootstrapPipelines writes build/lint/test pipeline YAML files under
// lifecycle/devops/ derived from a promoted stack's per-role commands. One
// step per required role that defines the command, identical commands
// de-duplicated (e.g. a stack whose test-developer and backend-developer both
// run `go test ./...` yields a single test step). Skip-if-exists: a pipeline
// file that already exists is never overwritten. Returns the project-root
// relative paths of the files it created.
func BootstrapPipelines(projectRoot string, profile architecture.StackProfile) ([]string, error) {
	devDir := filepath.Join(projectRoot, "lifecycle", "devops")
	if err := os.MkdirAll(devDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating lifecycle/devops: %w", err)
	}

	// Deterministic role order so generation is reproducible.
	roles := make([]string, 0, len(profile.Roles))
	for name := range profile.Roles {
		roles = append(roles, name)
	}
	sort.Strings(roles)

	var created []string
	for _, kind := range bootstrapKinds {
		var steps []stepYAML
		seen := map[string]bool{}
		for _, role := range roles {
			rp := profile.Roles[role]
			if !rp.IsRequired() {
				continue
			}
			cmd := kind.pick(rp)
			if cmd == "" || seen[cmd] {
				continue
			}
			seen[cmd] = true
			steps = append(steps, stepYAML{
				Name:        role + " " + kind.typ,
				Description: fmt.Sprintf("%s step for the %s role, from the stack profile", kind.typ, role),
				Command:     cmd,
				Timeout:     kind.timeout,
			})
		}
		if len(steps) == 0 {
			continue
		}

		fileName := kind.typ + ".yaml"
		abs := filepath.Join(devDir, fileName)
		if _, err := os.Stat(abs); err == nil {
			continue // skip-if-exists — never clobber a user's pipeline
		}

		body, err := yaml.Marshal(pipelineYAML{Name: kind.name, Type: kind.typ, Steps: steps})
		if err != nil {
			return created, fmt.Errorf("marshalling %s pipeline: %w", kind.typ, err)
		}
		header := "# Bootstrapped by kaos-control from the promoted tech-stack's stack_profile.\n" +
			"# Edit freely — re-running scaffolding will not overwrite this file.\n\n"
		if err := os.WriteFile(abs, append([]byte(header), body...), 0o644); err != nil {
			return created, fmt.Errorf("writing %s: %w", fileName, err)
		}
		created = append(created, path.Join("lifecycle", "devops", fileName))
	}
	return created, nil
}
