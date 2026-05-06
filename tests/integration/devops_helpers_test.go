//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// devopsCfgYAML extends the default project config to include the devops role.
// dev@test.local is granted the devops role so that role-gating tests can use
// the existing auth store users created by newTestEnvFull.
const devopsCfgYAML = `git:
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
  - devops

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
    roles: [backend-developer, frontend-developer, test-developer, devops]
  - email: qa@test.local
    roles: [qa]

required_plans:
  ticket: [plan-backend, plan-frontend, plan-test]
  epic: []
`

// Pipeline YAML fixtures used by devops integration tests.
const (
	pipelineQuickPass = `name: Quick Pass
type: build
steps:
  - name: Echo OK
    description: Verify the environment works
    command: echo ok
`

	pipelineQuickFail = `name: Quick Fail
type: build
steps:
  - name: Exit One
    description: Step that exits non-zero
    command: exit 1
`

	pipelineOrderedOutput = `name: Ordered Output
type: build
steps:
  - name: First
    command: echo FIRST
  - name: Second
    command: echo SECOND
  - name: Third
    command: echo THIRD
`

	pipelineSlowStep = `name: Slow Step
type: deploy
steps:
  - name: Sleep
    description: Long-running step for concurrency testing
    command: sleep 30
`

	pipelineSlowStep2 = `name: Slow Step Two
type: deploy
steps:
  - name: Sleep
    description: Second long-running step for concurrency testing
    command: sleep 30
`

	pipelineTimeoutStep = `name: Timeout Step
type: build
steps:
  - name: Will Timeout
    description: Step with very short timeout
    command: sleep 10
    timeout: 1s
`

	pipelineMultiStepWithFail = `name: Multi Step Fail
type: build
steps:
  - name: First OK
    command: echo first-ok
  - name: Then Fail
    command: exit 2
  - name: Never Runs
    command: echo never
`
)

// newDevopsTestEnv creates a test environment configured with the devops role.
// pipelines maps filename (e.g. "build.yaml") to YAML content to seed into lifecycle/devops/.
func newDevopsTestEnv(t *testing.T, pipelines map[string]string) *testEnv {
	t.Helper()
	env := newTestEnvWithCfgYAML(t, nil, devopsCfgYAML)
	writePipelineFiles(t, env, pipelines)
	return env
}

// writePipelineFiles creates lifecycle/devops/ in the project root and writes
// the given YAML files into it.
func writePipelineFiles(t *testing.T, env *testEnv, pipelines map[string]string) {
	t.Helper()
	devopsDir := filepath.Join(env.projectRoot, "lifecycle", "devops")
	if err := os.MkdirAll(devopsDir, 0o755); err != nil {
		t.Fatalf("creating devops dir: %v", err)
	}
	for name, content := range pipelines {
		path := filepath.Join(devopsDir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("writing pipeline file %s: %v", name, err)
		}
	}
}

// waitForRunComplete polls Runner.IsRunning(slug) until the run is done or the
// given timeout elapses.
func waitForRunComplete(t *testing.T, env *testEnv, slug string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !env.proj.DevopsRunner.IsRunning(slug) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("pipeline %q did not complete within %v", slug, timeout)
}

// devopsListURL returns the pipeline listing URL for the test project.
func devopsListURL(env *testEnv) string {
	return env.baseURL + "/api/p/testproject/devops/pipelines"
}

// devopsRunURL returns the run-trigger URL for the given pipeline slug.
func devopsRunURL(env *testEnv, slug string) string {
	return env.baseURL + "/api/p/testproject/devops/pipelines/" + slug + "/run"
}

// devopsCancelURL returns the cancel URL for the given pipeline slug.
func devopsCancelURL(env *testEnv, slug string) string {
	return env.baseURL + "/api/p/testproject/devops/pipelines/" + slug + "/cancel"
}

// devopsRunLogURL returns the run log URL for the given run ID.
func devopsRunLogURL(env *testEnv, runID string) string {
	return env.baseURL + "/api/p/testproject/devops/runs/" + runID
}
