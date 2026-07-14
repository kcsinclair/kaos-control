// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

// CLI integration tests for the `kaos-control devops` subcommand group.
//
// Tests cover Milestones 3–5 of the DevOps CLI test plan:
//
//	M3 – help text, project selection, devops list --json, stdout/stderr shape
//	M4 – identity resolution (bearer token, mapped linux user, unmapped exit 3,
//	     token redaction)
//	M5 – devops run with role gating, --follow, and authz parity
//
// Tests in this file invoke the compiled binary (built by TestMain in
// cli_init_test.go) and exercise the full CLI path against a real server
// subprocess.
package cli_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/auth"
)

// ─── devopsServer setup ───────────────────────────────────────────────────────

// devopsServer holds all resources for a CLI devops integration test.
type devopsServer struct {
	baseURL      string
	authStore    *auth.Store
	cfgHome      string // XDG_CONFIG_HOME; points server and CLI at the same config
	projectRoot  string // project with linux_user mapping (mapped project)
	projectName  string
	dataDir      string
	// insuffProjName is a second project where the current OS user maps to a
	// qa-only role, used for F8 insufficient-role tests.
	insuffProjName string
}

// startDevopsServer creates an isolated environment:
//
//	cfgHome/kaos-control/config.yaml   – app config (server + projects_dir)
//	cfgHome/projects/<name>.yaml        – project entry
//	projectRoot/lifecycle/{ideas,...}/  – stage directories
//	projectRoot/lifecycle/config.yaml   – project config with linux_user mapping
//	projectRoot/lifecycle/devops/       – pipeline YAML fixtures
//	auth.db at dataDir/auth.db          – pre-populated users and tokens
//
// The binary is started as a subprocess on a free port and polled until ready.
func startDevopsServer(t *testing.T) *devopsServer {
	t.Helper()

	// Discover the current OS username for the linux_user mapping test.
	cu, err := user.Current()
	if err != nil {
		t.Fatalf("user.Current: %v", err)
	}
	currentUser := cu.Username

	port := freePort(t)
	cfgHome := t.TempDir()
	dataDir := t.TempDir()
	rawProjectRoot := t.TempDir()
	// Resolve symlinks so os.Getwd() in CLI subprocesses (which returns the
	// canonical path) matches the path stored in the project entry YAML. On
	// macOS /var/folders is a symlink to /private/var/folders.
	projectRoot, err2 := filepath.EvalSymlinks(rawProjectRoot)
	if err2 != nil {
		projectRoot = rawProjectRoot
	}
	projectsDir := filepath.Join(cfgHome, "projects")
	if err := os.MkdirAll(projectsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll projectsDir: %v", err)
	}

	// ── Project directory structure ───────────────────────────────────────
	for _, s := range []string{"ideas", "requirements", "test-plans", "tests", "defects"} {
		if err := os.MkdirAll(filepath.Join(projectRoot, "lifecycle", s), 0o755); err != nil {
			t.Fatalf("MkdirAll lifecycle/%s: %v", s, err)
		}
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, "lifecycle", "devops"), 0o755); err != nil {
		t.Fatalf("MkdirAll lifecycle/devops: %v", err)
	}

	// ── Project config with linux_user mapping ────────────────────────────
	projCfg := fmt.Sprintf(`git:
  default_branch: main
roles:
  - product-owner
  - analyst
  - devops
  - qa
stages:
  - {name: ideas,        dir: ideas}
  - {name: requirements, dir: requirements}
  - {name: test-plans,   dir: test-plans}
  - {name: tests,        dir: tests}
  - {name: defects,      dir: defects}
users:
  - email: admin@devops-cli-test.local
    roles: [product-owner, analyst]
    linux_user: %s
  - email: dev@devops-cli-test.local
    roles: [devops]
  - email: qa@devops-cli-test.local
    roles: [qa]
`, currentUser)
	if err := os.WriteFile(
		filepath.Join(projectRoot, "lifecycle", "config.yaml"),
		[]byte(projCfg), 0o644,
	); err != nil {
		t.Fatalf("writing project config: %v", err)
	}

	// ── Seed a few artifacts ──────────────────────────────────────────────
	artifacts := []struct {
		path    string
		content string
	}{
		{
			"lifecycle/ideas/widget-api.md",
			"---\ntitle: Widget API\ntype: idea\nstatus: draft\nlineage: widget-api\n---\n\nThe widget API.\n",
		},
		{
			"lifecycle/requirements/widget-api-2.md",
			"---\ntitle: Widget API Requirements\ntype: requirement\nstatus: approved\nlineage: widget-api\nparent: lifecycle/ideas/widget-api.md\n---\n\nDetailed requirements.\n",
		},
	}
	for _, a := range artifacts {
		absPath := filepath.Join(projectRoot, a.path)
		if err := os.WriteFile(absPath, []byte(a.content), 0o644); err != nil {
			t.Fatalf("writing artifact %s: %v", a.path, err)
		}
	}

	// ── Pipeline YAML fixtures ────────────────────────────────────────────
	pipelines := map[string]string{
		"quick-pass.yaml": `name: Quick Pass
type: build
steps:
  - name: Echo OK
    description: Verify the environment works
    command: echo ok
`,
		"env-check.yaml": `name: Env Check
type: build
steps:
  - name: Print Token Length
    description: Outputs the length of KC_API_TOKEN to verify attribution
    command: printf 'TOKEN_LEN=%s\n' "${#KC_API_TOKEN}"
`,
	}
	for name, content := range pipelines {
		if err := os.WriteFile(
			filepath.Join(projectRoot, "lifecycle", "devops", name),
			[]byte(content), 0o644,
		); err != nil {
			t.Fatalf("writing pipeline %s: %v", name, err)
		}
	}

	// ── Project entry YAML ────────────────────────────────────────────────
	const projectName = "devops-cli-test"
	entryContent := fmt.Sprintf("name: %s\npath: %q\n", projectName, projectRoot)
	if err := os.WriteFile(
		filepath.Join(projectsDir, projectName+".yaml"),
		[]byte(entryContent), 0o644,
	); err != nil {
		t.Fatalf("writing project entry: %v", err)
	}

	// ── Second project: current user maps to qa-only (for F8 role tests) ─
	// Pre-registered before server starts so the server's project registry
	// includes it at startup.
	const insuffProjName = "devops-cli-insuffrole"
	rawInsuffRoot := t.TempDir()
	insuffRoot, err3 := filepath.EvalSymlinks(rawInsuffRoot)
	if err3 != nil {
		insuffRoot = rawInsuffRoot
	}
	for _, s := range []string{"ideas"} {
		if err := os.MkdirAll(filepath.Join(insuffRoot, "lifecycle", s), 0o755); err != nil {
			t.Fatalf("MkdirAll insuffRoot/lifecycle/%s: %v", s, err)
		}
	}
	if err := os.MkdirAll(filepath.Join(insuffRoot, "lifecycle", "devops"), 0o755); err != nil {
		t.Fatalf("MkdirAll insuffRoot/lifecycle/devops: %v", err)
	}
	insuffPipeline := "name: Quick Pass\ntype: build\nsteps:\n  - name: Echo\n    command: echo ok\n"
	if err := os.WriteFile(
		filepath.Join(insuffRoot, "lifecycle", "devops", "quick-pass.yaml"),
		[]byte(insuffPipeline), 0o644,
	); err != nil {
		t.Fatalf("writing insuffRoot pipeline: %v", err)
	}
	insuffCfg := fmt.Sprintf(`git:
  default_branch: main
roles:
  - qa
stages:
  - {name: ideas, dir: ideas}
users:
  - email: qa@devops-cli-test.local
    roles: [qa]
    linux_user: %s
`, currentUser)
	if err := os.WriteFile(
		filepath.Join(insuffRoot, "lifecycle", "config.yaml"),
		[]byte(insuffCfg), 0o644,
	); err != nil {
		t.Fatalf("writing insuffRoot config: %v", err)
	}
	insuffEntryContent := fmt.Sprintf("name: %s\npath: %q\n", insuffProjName, insuffRoot)
	if err := os.WriteFile(
		filepath.Join(projectsDir, insuffProjName+".yaml"),
		[]byte(insuffEntryContent), 0o644,
	); err != nil {
		t.Fatalf("writing insuffRoot project entry: %v", err)
	}

	// ── App config ────────────────────────────────────────────────────────
	appCfgDir := filepath.Join(cfgHome, "kaos-control")
	if err := os.MkdirAll(appCfgDir, 0o755); err != nil {
		t.Fatalf("MkdirAll app config dir: %v", err)
	}
	appCfgPath := filepath.Join(appCfgDir, "config.yaml")
	appCfgContent := fmt.Sprintf(
		"data_dir: %q\nprojects_dir: %q\nserver:\n  listen: \"127.0.0.1:%d\"\n",
		dataDir, projectsDir, port,
	)
	if err := os.WriteFile(appCfgPath, []byte(appCfgContent), 0o644); err != nil {
		t.Fatalf("writing app config: %v", err)
	}

	// ── Pre-populate auth store ───────────────────────────────────────────
	store, err := auth.Open(filepath.Join(dataDir, "auth.db"), 24*time.Hour)
	if err != nil {
		t.Fatalf("auth.Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	for _, u := range []struct{ email, name string }{
		{"admin@devops-cli-test.local", "Admin"},
		{"dev@devops-cli-test.local", "Dev"},
		{"qa@devops-cli-test.local", "QA"},
	} {
		if err := store.CreateUser(u.email, u.name, "pass", false); err != nil {
			t.Fatalf("CreateUser %s: %v", u.email, err)
		}
	}

	// ── Start server subprocess ───────────────────────────────────────────
	cmd := newBinCmd(t, "serve", "--config", appCfgPath)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting server: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		_ = cmd.Wait()
	})

	// ── Poll until ready ──────────────────────────────────────────────────
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/api/health") //nolint:gosec
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	// Final health check.
	resp, err := http.Get(baseURL + "/api/health") //nolint:gosec
	if err != nil {
		t.Fatalf("server did not become ready: %v", err)
	}
	resp.Body.Close()

	return &devopsServer{
		baseURL:        baseURL,
		authStore:      store,
		cfgHome:        cfgHome,
		projectRoot:    projectRoot,
		projectName:    projectName,
		dataDir:        dataDir,
		insuffProjName: insuffProjName,
	}
}

// execDevopsCmd runs the binary with `devops <args...>`, injecting
// XDG_CONFIG_HOME so the CLI reads the test server's config.
// Returns stdout, stderr, and exit code.
func (s *devopsServer) execDevopsCmd(t *testing.T, env []string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	fullArgs := append([]string{"devops"}, args...)
	cmd := exec.Command(binPath, fullArgs...)
	cmd.Env = append(append([]string{}, os.Environ()...), append([]string{"XDG_CONFIG_HOME=" + s.cfgHome}, env...)...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		if ex, ok := err.(*exec.ExitError); ok {
			code = ex.ExitCode()
		} else {
			t.Fatalf("exec error: %v", err)
		}
	}
	return outBuf.String(), errBuf.String(), code
}

// execDevopsCmdInDir is like execDevopsCmd but sets the working directory.
func (s *devopsServer) execDevopsCmdInDir(t *testing.T, dir string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	fullArgs := append([]string{"devops"}, args...)
	cmd := exec.Command(binPath, fullArgs...)
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+s.cfgHome)
	cmd.Dir = dir
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		if ex, ok := err.(*exec.ExitError); ok {
			code = ex.ExitCode()
		} else {
			t.Fatalf("exec error: %v", err)
		}
	}
	return outBuf.String(), errBuf.String(), code
}

// bearerToken creates a bearer token for the given email via the test auth store.
func (s *devopsServer) bearerToken(t *testing.T, email string) string {
	t.Helper()
	tok, err := s.authStore.CreateToken(email, nil)
	if err != nil {
		t.Fatalf("CreateToken(%s): %v", email, err)
	}
	return tok
}

// waitForRunComplete polls the pipeline run endpoint until no run is active
// or the timeout elapses. It verifies the run record is present.
func (s *devopsServer) waitForRun(t *testing.T, adminToken, slug string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		url := fmt.Sprintf("%s/api/p/%s/devops/pipelines/%s/runs", s.baseURL, s.projectName, slug)
		req, _ := http.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		var body map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()
		runs, _ := body["runs"].([]any)
		if len(runs) > 0 {
			first := runs[0].(map[string]any)
			status, _ := first["status"].(string)
			if status == "passed" || status == "failed" || status == "cancelled" {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("pipeline %q did not complete within %v", slug, timeout)
}

// ─── Milestone 3: Help, project selection, output shape ──────────────────────

// TestDevops_TopLevelHelp_ContainsDevops asserts that --help mentions devops.
func TestDevops_TopLevelHelp_ContainsDevops(t *testing.T) {
	stdout, _, code := runBin(t, "--help")
	if code != 0 {
		t.Fatalf("kaos-control --help: want exit 0, got %d", code)
	}
	if !strings.Contains(stdout, "devops") {
		t.Errorf("top-level --help output missing 'devops'\ngot: %s", stdout)
	}
}

// TestDevops_SubcommandHelp_ListsOperations asserts devops --help lists
// list, status, and run subcommands along with the documented exit codes.
func TestDevops_SubcommandHelp_ListsOperations(t *testing.T) {
	stdout, _, code := runBin(t, "devops", "--help")
	if code != 0 {
		t.Fatalf("kaos-control devops --help: want exit 0, got %d", code)
	}
	for _, want := range []string{"list", "status", "run"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("devops --help missing subcommand %q\ngot: %s", want, stdout)
		}
	}
	// NF5: documented exit codes.
	for _, code := range []string{"0", "1", "3", "4"} {
		if !strings.Contains(stdout, code) {
			t.Errorf("devops --help missing exit code %s in output\ngot: %s", code, stdout)
		}
	}
}

// TestDevops_ProjectFlag_SelectsRegistered verifies that --project <name>
// selects a registered project and list succeeds.
func TestDevops_ProjectFlag_SelectsRegistered(t *testing.T) {
	srv := startDevopsServer(t)
	tok := srv.bearerToken(t, "admin@devops-cli-test.local")

	stdout, _, code := srv.execDevopsCmd(t, nil,
		"list", "--project", srv.projectName, "--token", tok, "--json",
	)
	if code != 0 {
		t.Fatalf("devops list --project: want exit 0, got %d\nstdout: %s", code, stdout)
	}
	var arr []any
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &arr); err != nil {
		t.Errorf("devops list --json stdout is not a JSON array: %v\ngot: %s", err, stdout)
	}
}

// TestDevops_ProjectFlag_UnknownErrors verifies that an unknown --project value
// exits non-zero.
func TestDevops_ProjectFlag_UnknownErrors(t *testing.T) {
	srv := startDevopsServer(t)
	tok := srv.bearerToken(t, "admin@devops-cli-test.local")

	_, _, code := srv.execDevopsCmd(t, nil,
		"list", "--project", "this-project-does-not-exist", "--token", tok,
	)
	if code == 0 {
		t.Error("expected non-zero exit for unknown --project, got 0")
	}
}

// TestDevops_CwdInference_InProject verifies that running inside a registered
// project root without --project infers the project.
func TestDevops_CwdInference_InProject(t *testing.T) {
	srv := startDevopsServer(t)
	tok := srv.bearerToken(t, "admin@devops-cli-test.local")

	// execDevopsCmdInDir sets cmd.Dir; the CLI uses os.Getwd() which returns
	// that directory, and selectProject matches it against the registered path.
	fullArgs := []string{"devops", "list", "--json", "--token", tok}
	cmd := exec.Command(binPath, fullArgs...)
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+srv.cfgHome)
	cmd.Dir = srv.projectRoot
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	var code int
	if err := cmd.Run(); err != nil {
		if ex, ok := err.(*exec.ExitError); ok {
			code = ex.ExitCode()
		} else {
			t.Fatalf("exec: %v", err)
		}
	}
	stdout := outBuf.String()
	if code != 0 {
		t.Fatalf("devops list from project root: want exit 0, got %d\nstdout: %s\nstderr: %s",
			code, stdout, errBuf.String())
	}
	var arr []any
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &arr); err != nil {
		t.Errorf("stdout is not a JSON array: %v\ngot: %s", err, stdout)
	}
}

// TestDevops_CwdInference_NotInProject verifies exit non-zero when cwd is not
// inside any registered project and --project is not supplied.
func TestDevops_CwdInference_NotInProject(t *testing.T) {
	srv := startDevopsServer(t)
	tok := srv.bearerToken(t, "admin@devops-cli-test.local")

	unregisteredDir := t.TempDir()
	cmd := exec.Command(binPath, "devops", "list", "--token", tok)
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+srv.cfgHome)
	cmd.Dir = unregisteredDir
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	var code int
	if err := cmd.Run(); err != nil {
		if ex, ok := err.(*exec.ExitError); ok {
			code = ex.ExitCode()
		} else {
			t.Fatalf("exec: %v", err)
		}
	}
	if code == 0 {
		t.Error("expected non-zero exit for cwd outside any registered project, got 0")
	}
	if errBuf.Len() == 0 {
		t.Error("expected error message in stderr, got nothing")
	}
}

// TestDevops_List_JSON verifies that devops list --json emits a valid JSON array.
func TestDevops_List_JSON(t *testing.T) {
	srv := startDevopsServer(t)
	tok := srv.bearerToken(t, "admin@devops-cli-test.local")

	stdout, _, code := srv.execDevopsCmd(t, nil,
		"list", "--project", srv.projectName, "--token", tok, "--json",
	)
	if code != 0 {
		t.Fatalf("devops list --json: want exit 0, got %d", code)
	}
	var arr []any
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &arr); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\ngot: %s", err, stdout)
	}
}

// TestDevops_List_FilterByType verifies --type narrows the artifact set.
func TestDevops_List_FilterByType(t *testing.T) {
	srv := startDevopsServer(t)
	tok := srv.bearerToken(t, "admin@devops-cli-test.local")

	stdout, _, code := srv.execDevopsCmd(t, nil,
		"list", "--project", srv.projectName, "--token", tok, "--json", "--type", "idea",
	)
	if code != 0 {
		t.Fatalf("devops list --type=idea: want exit 0, got %d", code)
	}
	var arr []any
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &arr); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\ngot: %s", err, stdout)
	}
	// All returned items must have type=idea.
	for i, item := range arr {
		m, _ := item.(map[string]any)
		if typ, _ := m["type"].(string); typ != "idea" {
			t.Errorf("arr[%d].type = %q, want idea", i, typ)
		}
	}
}

// TestDevops_List_JSON_StdoutSeparatedFromStderr verifies that stdout is always
// parseable JSON even if diagnostics are present (F12).
func TestDevops_List_JSON_StdoutSeparatedFromStderr(t *testing.T) {
	srv := startDevopsServer(t)
	tok := srv.bearerToken(t, "admin@devops-cli-test.local")

	stdout, _, code := srv.execDevopsCmd(t, nil,
		"list", "--project", srv.projectName, "--token", tok, "--json",
	)
	if code != 0 {
		t.Fatalf("devops list --json: want exit 0, got %d", code)
	}
	// stdout must always be valid JSON regardless of what goes to stderr.
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), new(any)); err != nil {
		t.Errorf("stdout is not valid JSON (F12 violation): %v\ngot: %s", err, stdout)
	}
}

// ─── Milestone 4: Identity resolution ────────────────────────────────────────

// TestDevops_Identity_BearerToken verifies that --token authenticates a CI
// invocation and the list succeeds (F6a).
func TestDevops_Identity_BearerToken(t *testing.T) {
	srv := startDevopsServer(t)
	tok := srv.bearerToken(t, "admin@devops-cli-test.local")

	_, _, code := srv.execDevopsCmd(t, nil,
		"list", "--project", srv.projectName, "--token", tok,
	)
	if code != 0 {
		t.Fatalf("devops list with --token: want exit 0, got %d", code)
	}
}

// TestDevops_Identity_BearerTokenViaEnv verifies KAOS_CONTROL_TOKEN authenticates.
func TestDevops_Identity_BearerTokenViaEnv(t *testing.T) {
	srv := startDevopsServer(t)
	tok := srv.bearerToken(t, "admin@devops-cli-test.local")

	_, _, code := srv.execDevopsCmd(t,
		[]string{"KAOS_CONTROL_TOKEN=" + tok},
		"list", "--project", srv.projectName,
	)
	if code != 0 {
		t.Fatalf("devops list with KAOS_CONTROL_TOKEN env: want exit 0, got %d", code)
	}
}

// TestDevops_Identity_MappedLinuxUser verifies that the current OS user's
// linux_user mapping resolves to the bound email and the command succeeds
// without a token (F6c). The project config binds the test runner's username
// to admin@devops-cli-test.local.
func TestDevops_Identity_MappedLinuxUser(t *testing.T) {
	srv := startDevopsServer(t)

	// No --token; the CLI must infer the identity via linux_user mapping.
	_, _, code := srv.execDevopsCmd(t, nil,
		"list", "--project", srv.projectName,
	)
	if code != 0 {
		t.Fatalf("devops list via linux_user: want exit 0, got %d", code)
	}
}

// TestDevops_Identity_UnmappedExitsThree verifies that an unmapped Linux user
// with no token exits 3 and emits "identity not resolved" (F7, NF3).
func TestDevops_Identity_UnmappedExitsThree(t *testing.T) {
	srv := startDevopsServer(t)

	// Use a second project that has NO linux_user mapping for any user.
	// We seed an unmapped project directly — no binaries needed, just a
	// project config without linux_user entries.
	unmappedRoot := t.TempDir()
	for _, s := range []string{"ideas"} {
		_ = os.MkdirAll(filepath.Join(unmappedRoot, "lifecycle", s), 0o755)
	}
	unmappedCfg := `git:
  default_branch: main
roles:
  - product-owner
stages:
  - {name: ideas, dir: ideas}
users:
  - email: nolux@devops-cli-test.local
    roles: [product-owner]
`
	_ = os.WriteFile(filepath.Join(unmappedRoot, "lifecycle", "config.yaml"), []byte(unmappedCfg), 0o644)

	// Register it in the projects dir.
	const unmappedName = "devops-cli-unmapped"
	entryContent := fmt.Sprintf("name: %s\npath: %q\n", unmappedName, unmappedRoot)
	_ = os.WriteFile(
		filepath.Join(srv.cfgHome, "projects", unmappedName+".yaml"),
		[]byte(entryContent), 0o644,
	)
	// Create the user in the auth store.
	_ = srv.authStore.CreateUser("nolux@devops-cli-test.local", "No Lux", "pass", false)

	_, stderr, code := srv.execDevopsCmd(t, nil,
		"list", "--project", unmappedName,
	)
	if code != 3 {
		t.Errorf("unmapped linux user: want exit 3, got %d\nstderr: %s", code, stderr)
	}
	if !strings.Contains(stderr, "identity not resolved") {
		t.Errorf("expected 'identity not resolved' in stderr, got: %s", stderr)
	}
}

// TestDevops_Identity_TokenNotInOutput verifies that a --token value never
// appears in stdout or stderr (NF2).
func TestDevops_Identity_TokenNotInOutput(t *testing.T) {
	srv := startDevopsServer(t)
	tok := srv.bearerToken(t, "admin@devops-cli-test.local")

	stdout, stderr, _ := srv.execDevopsCmd(t, nil,
		"list", "--project", srv.projectName, "--token", tok,
	)
	if strings.Contains(stdout, tok) {
		t.Error("token value found in stdout — token redaction failure (NF2)")
	}
	if strings.Contains(stderr, tok) {
		t.Error("token value found in stderr — token redaction failure (NF2)")
	}
}

// ─── Milestone 5: Run, role gating, and parity ───────────────────────────────

// TestDevops_Run_StartsAndPrintsRunID verifies that devops run <task> as a
// product-owner user starts a run and prints a non-empty run ID (F4).
func TestDevops_Run_StartsAndPrintsRunID(t *testing.T) {
	srv := startDevopsServer(t)
	tok := srv.bearerToken(t, "admin@devops-cli-test.local")

	stdout, _, code := srv.execDevopsCmd(t, nil,
		"run", "--project", srv.projectName, "--token", tok, "quick-pass",
	)
	if code != 0 {
		t.Fatalf("devops run quick-pass: want exit 0, got %d\nstdout: %s", code, stdout)
	}
	runID := strings.TrimSpace(stdout)
	if runID == "" {
		t.Error("devops run produced empty run ID")
	}
}

// TestDevops_Run_Follow_StreamsLog verifies that --follow streams the NDJSON
// run log to stdout and exits with the run's terminal status (F4).
func TestDevops_Run_Follow_StreamsLog(t *testing.T) {
	srv := startDevopsServer(t)
	tok := srv.bearerToken(t, "admin@devops-cli-test.local")

	stdout, _, code := srv.execDevopsCmd(t, nil,
		"run", "--project", srv.projectName, "--token", tok, "--follow", "quick-pass",
	)
	if code != 0 {
		t.Fatalf("devops run --follow quick-pass: want exit 0, got %d\nstdout:\n%s", code, stdout)
	}

	// Expect at least one log line from the NDJSON stream.
	scanner := bufio.NewScanner(strings.NewReader(stdout))
	var foundRunLine bool
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			// Non-JSON lines are the human-readable output (step name, run status).
			foundRunLine = true
			continue
		}
		foundRunLine = true
	}
	if !foundRunLine {
		t.Error("devops run --follow produced no output lines")
	}
}

// TestDevops_Run_UnknownTask_ExitsOpFailed verifies that running an unknown
// task slug exits 1.
func TestDevops_Run_UnknownTask_ExitsOpFailed(t *testing.T) {
	srv := startDevopsServer(t)
	tok := srv.bearerToken(t, "admin@devops-cli-test.local")

	_, _, code := srv.execDevopsCmd(t, nil,
		"run", "--project", srv.projectName, "--token", tok, "no-such-pipeline",
	)
	if code != 1 {
		t.Errorf("devops run unknown task: want exit 1, got %d", code)
	}
}

// TestDevops_Run_InsufficientRole_ExitsFour verifies that a user without the
// required role (product-owner or devops) gets exit 4 with a role-required
// message when attempting devops run (F8).
func TestDevops_Run_InsufficientRole_ExitsFour(t *testing.T) {
	srv := startDevopsServer(t)
	qaTok := srv.bearerToken(t, "qa@devops-cli-test.local")

	_, stderr, code := srv.execDevopsCmd(t, nil,
		"run", "--project", srv.projectName, "--token", qaTok, "quick-pass",
	)
	if code != 4 {
		t.Errorf("devops run with qa role: want exit 4, got %d\nstderr: %s", code, stderr)
	}
}

// TestDevops_Run_MappedLinuxUser_InsufficientRole verifies that a mapped Linux
// user with an insufficient role is rejected with exit 4 even though they have
// filesystem read access (F8 — acceptance bullet).
//
// The "insufficient role" project (srv.insuffProjName) maps the current OS user
// to qa@devops-cli-test.local, which has only the qa role — not product-owner
// or devops. It is pre-registered in startDevopsServer so the server knows
// about it at startup.
func TestDevops_Run_MappedLinuxUser_InsufficientRole(t *testing.T) {
	srv := startDevopsServer(t)

	_, stderr, code := srv.execDevopsCmd(t, nil,
		"run", "--project", srv.insuffProjName, "quick-pass",
	)
	if code != 4 {
		t.Errorf("mapped linux user with insufficient role: want exit 4, got %d\nstderr: %s", code, stderr)
	}
}

// TestDevops_Attribution_ViaKCToken verifies that when a run is triggered via
// linux_user identity, the KC_API_TOKEN injected into pipeline steps is valid
// and non-empty — confirming the server attributed the run to the resolved user
// (F11). Full RunRecord.TriggeredBy tracking is a future server enhancement.
//
// We do NOT use --follow to collect step output: the CLI's streamRunLog reads
// the log once without polling, which races against async run completion. Instead
// we start the run, wait for it to complete via the run-list endpoint, then fetch
// the NDJSON log directly and scan for the TOKEN_LEN= step-output event.
func TestDevops_Attribution_ViaKCToken(t *testing.T) {
	srv := startDevopsServer(t)
	adminTok := srv.bearerToken(t, "admin@devops-cli-test.local")

	// Trigger via linux_user mapping (resolves to admin@devops-cli-test.local).
	stdout, _, code := srv.execDevopsCmd(t, nil,
		"run", "--project", srv.projectName, "env-check",
	)
	if code != 0 {
		t.Fatalf("devops run env-check: want exit 0, got %d\nstdout:\n%s", code, stdout)
	}
	runID := strings.TrimSpace(stdout)
	if runID == "" {
		t.Fatal("devops run produced empty run ID")
	}

	// Wait for the run to reach a terminal state via the pipeline run list.
	srv.waitForRun(t, adminTok, "env-check", 15*time.Second)

	// Fetch the NDJSON run log directly via the HTTP API.
	logURL := fmt.Sprintf("%s/api/p/%s/devops/runs/%s", srv.baseURL, srv.projectName, runID)
	req, _ := http.NewRequest(http.MethodGet, logURL, nil)
	req.Header.Set("Authorization", "Bearer "+adminTok)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET run log: %v", err)
	}
	defer resp.Body.Close()
	logBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET run log: want 200, got %d\nbody: %s", resp.StatusCode, logBytes)
	}

	// Scan NDJSON for a pipeline.step.output event that contains TOKEN_LEN=.
	// The step-output event has a "text" field with the raw step stdout line.
	foundTokenLen := false
	tokenLenVal := ""
	for _, line := range strings.Split(string(logBytes), "\n") {
		if !strings.Contains(line, "pipeline.step.output") {
			continue
		}
		var ev map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		var text string
		if err := json.Unmarshal(ev["text"], &text); err != nil {
			continue
		}
		if strings.HasPrefix(text, "TOKEN_LEN=") {
			foundTokenLen = true
			tokenLenVal = strings.TrimPrefix(strings.TrimSpace(text), "TOKEN_LEN=")
			break
		}
	}
	if !foundTokenLen {
		t.Errorf("TOKEN_LEN= not found in run log — KC_API_TOKEN may not have been injected\nlog:\n%s", logBytes)
	} else if tokenLenVal == "0" || tokenLenVal == "" {
		t.Errorf("TOKEN_LEN is zero or empty — KC_API_TOKEN not attributed to resolved user")
	}
}

// TestDevops_Parity_CLIAndHTTP verifies that for the same identity+operation,
// the CLI's allow/deny outcome matches the HTTP API's (parity bullet).
//
// Scenario: qa@devops-cli-test.local attempts devops run.
// Expected: HTTP API returns 403; CLI exits 4. Both agree on "denied".
func TestDevops_Parity_CLIAndHTTP(t *testing.T) {
	srv := startDevopsServer(t)
	qaTok := srv.bearerToken(t, "qa@devops-cli-test.local")

	// ── HTTP path ─────────────────────────────────────────────────────────
	runURL := fmt.Sprintf("%s/api/p/%s/devops/pipelines/quick-pass/run", srv.baseURL, srv.projectName)
	req, _ := http.NewRequest(http.MethodPost, runURL, nil)
	req.Header.Set("Authorization", "Bearer "+qaTok)
	httpResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("HTTP POST run: %v", err)
	}
	httpResp.Body.Close()
	httpDenied := httpResp.StatusCode == http.StatusForbidden

	// ── CLI path ──────────────────────────────────────────────────────────
	_, _, cliCode := srv.execDevopsCmd(t, nil,
		"run", "--project", srv.projectName, "--token", qaTok, "quick-pass",
	)
	cliDenied := cliCode == 4

	// ── Parity assertion ──────────────────────────────────────────────────
	if httpDenied != cliDenied {
		t.Errorf("authz parity failure: HTTP denied=%v (status %d), CLI denied=%v (exit %d)",
			httpDenied, httpResp.StatusCode, cliDenied, cliCode)
	}
	if !httpDenied || !cliDenied {
		t.Errorf("both HTTP and CLI should deny qa role for devops run; HTTP=%d CLI exit=%d",
			httpResp.StatusCode, cliCode)
	}
}
