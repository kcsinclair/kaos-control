// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Shared fixtures for the directives_*_test.go files. Test plan:
// lifecycle/test-plans/agent-directives-generation-5-test.md — verifies the
// acceptance criteria of [[agent-directives-generation]] end-to-end through
// the HTTP API (and, in tests/cli_directives_test.go, the CLI binary).

import (
	"testing"

	"github.com/kaos-control/kaos-control/internal/architecture/catalogfs"
)

// directivesCfgYAML extends the pattern used elsewhere in this package
// (e.g. agentLifecycleCfgYAML) with the six standard agents named by
// internal/directives' standardAgentRoles, plus one hand-added custom
// agent, so PatchAgentConfig has real targets to tune and a non-standard
// agent to prove untouched (FR-9). qa runs on driver: gemini-cli so
// GEMINI.md generation can be exercised by default; Milestone 5's
// selectivity tests derive a "no gemini driver" variant from this constant
// via strings.Replace.
const directivesCfgYAML = `git:
  default_branch: main
  branch_template: "ticket/{slug}"

roles:
  - product-owner
  - analyst
  - backend-developer
  - frontend-developer
  - test-developer
  - qa
  - reviewer
  - approver

stages:
  - {name: ideas, dir: ideas}
  - {name: requirements, dir: requirements}
  - {name: backend-plans, dir: backend-plans}
  - {name: frontend-plans, dir: frontend-plans}
  - {name: test-plans, dir: test-plans}
  - {name: tests, dir: tests}
  - {name: prototypes, dir: prototypes}
  - {name: releases, dir: releases}
  - {name: sprints, dir: sprints}
  - {name: defects, dir: defects}

users:
  - email: admin@test.local
    roles: [product-owner, analyst, reviewer, approver]
  - email: dev@test.local
    roles: [backend-developer, frontend-developer, test-developer]
  - email: qa@test.local
    roles: [qa]

required_plans:
  ticket: [plan-backend, plan-frontend, plan-test]
  epic: []

agents:
  - name: requirements-analyst
    role: [analyst]
    driver: claude-code-cli
    allowed_write_paths:
      - lifecycle/requirements
      - lifecycle/ideas
    prompt_templates:
      analyst: |
        You are an analyst. Read the idea artifact at {target_path}.
  - name: planning-analyst
    role: [analyst]
    driver: claude-code-cli
    allowed_write_paths:
      - lifecycle/backend-plans
      - lifecycle/frontend-plans
      - lifecycle/test-plans
    prompt_templates:
      analyst: |
        You are an analyst. Read the requirement at {target_path}.
  - name: backend-developer
    role: [backend-developer]
    driver: claude-code-cli
    allowed_write_paths:
      - internal
      - cmd
    prompt_templates:
      backend-developer: |
        You are a backend developer. Implement the backend plan at {target_path}.
  - name: frontend-developer
    role: [frontend-developer]
    driver: claude-code-cli
    allowed_write_paths:
      - web/src
    prompt_templates:
      frontend-developer: |
        You are a frontend developer. Implement the frontend plan at {target_path}.
  - name: test-developer
    role: [test-developer]
    driver: claude-code-cli
    allowed_write_paths:
      - tests
    prompt_templates:
      test-developer: |
        You are a test developer. Implement the test plan at {target_path}.
  - name: qa
    role: [qa]
    driver: gemini-cli
    allowed_write_paths:
      - lifecycle/defects
    prompt_templates:
      qa: |
        You are a QA agent. Given the artifact at {target_path}, run tests.
  - name: my-custom-agent
    role: [product-owner]
    driver: inline
    allowed_write_paths:
      - lifecycle/ideas
    prompt_templates:
      idea-capture: |
        A hand-written custom agent prompt that must never be touched.
`

// catalogStackContent reads a shipped tech-stack catalog file's raw
// markdown (including its embedded stack_profile: block) from the
// compiled-in catalogfs.
func catalogStackContent(t *testing.T, name string) string {
	t.Helper()
	raw, err := catalogfs.FS.ReadFile("tech-stacks/" + name)
	if err != nil {
		t.Fatalf("reading embedded catalog file %q: %v", name, err)
	}
	return string(raw)
}

// promotedStackSeed returns a seedArtifact that plants name's raw catalog
// content at the lifecycle/architecture/ root — the "promoted" location
// architecture.LoadPromotedStackProfile scans for.
func promotedStackSeed(t *testing.T, name string) seedArtifact {
	t.Helper()
	return seedArtifact{
		relPath: "lifecycle/architecture/" + name,
		content: catalogStackContent(t, name),
	}
}

// refreshDirectives POSTs .../directives/refresh for testproject and
// returns the decoded GenerateResult.
func refreshDirectives(t *testing.T, env *testEnv, force bool) map[string]any {
	t.Helper()
	resp := env.doRequest("POST", "/api/p/testproject/directives/refresh", map[string]any{"force": force})
	requireStatus(t, resp, 200)
	return readJSON(t, resp)
}

// fileWriteByPath locates the FileWrite entry for path within a decoded
// GenerateResult's "files" array, failing the test if absent.
func fileWriteByPath(t *testing.T, data map[string]any, path string) map[string]any {
	t.Helper()
	files, _ := data["files"].([]any)
	for _, f := range files {
		fw, _ := f.(map[string]any)
		if fw["path"] == path {
			return fw
		}
	}
	t.Fatalf("no FileWrite for %q in %v", path, data["files"])
	return nil
}
