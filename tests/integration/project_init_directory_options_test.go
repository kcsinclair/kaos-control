// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Integration tests for the mode-aware New Project onboarding flow
// ("Use existing directory" / "Create new directory") covering
// lifecycle/test-plans/new-project-init-directory-options-5-test.md
// Milestones 2-6. Milestone 1 (NormalizePath/ValidateDirName/ResolveNewTarget
// unit tests) lives in internal/config/config_test.go.
//
// Reuses the crudTestEnv harness (newCRUDTestEnv, doRequest, errCode,
// requireCRUDStatus, makeTempProjectDir) defined in projects_crud_test.go.
//
// Endpoints under test:
//   POST /api/projects                 (mode-aware create/onboard)
//   POST /api/projects/check-directory (mode-aware pre-submit validation)

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"syscall"
	"testing"

	"github.com/kaos-control/kaos-control/internal/initcmd"
)

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

// referenceScaffoldPaths runs the CLI `kaos-control init` (initcmd.Run) into a
// throwaway temp dir and returns the sorted set of relative file paths it
// produces, so onboarded projects can be compared against it (NFR3).
func referenceScaffoldPaths(t *testing.T) []string {
	t.Helper()
	ref := t.TempDir()
	if err := initcmd.Run([]string{ref}); err != nil {
		t.Fatalf("reference initcmd.Run failed: %v", err)
	}
	return sortedRelFiles(t, ref)
}

// sortedRelFiles walks root and returns the sorted, slash-separated relative
// paths of every regular file under it (directories themselves are omitted;
// their presence is implied by the files inside them).
func sortedRelFiles(t *testing.T, root string) []string {
	t.Helper()
	var paths []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root || d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}
	sort.Strings(paths)
	return paths
}

// requireSubset fails the test if any element of want is missing from got.
func requireSubset(t *testing.T, got []string, want []string, context string) {
	t.Helper()
	set := make(map[string]bool, len(got))
	for _, p := range got {
		set[p] = true
	}
	for _, w := range want {
		if !set[w] {
			t.Errorf("%s: expected scaffold file %q not found; got %v", context, w, got)
		}
	}
}

// createProjectBody builds a request body for POST /api/projects.
func createProjectBody(name, mode string, extra map[string]any) map[string]any {
	body := map[string]any{"name": name, "mode": mode}
	for k, v := range extra {
		body[k] = v
	}
	return body
}

// skipIfRootOrWindows skips permission-based tests where they can't be
// meaningfully enforced (matches the pattern already used in
// projects_crud_test.go's TestCheckDirectory_ExistsNotWritable).
func skipIfRootOrWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("permission enforcement test not applicable on Windows")
	}
	if os.Getuid() == 0 {
		t.Skip("running as root — permission restrictions not enforced")
	}
}

// ---------------------------------------------------------------------------
// Milestone 2 — Existing mode: happy path & non-destructive scaffold (FR5)
// ---------------------------------------------------------------------------

// TestOnboard_ExistingMode_NonDestructive verifies that scaffolding into a
// pre-populated existing directory leaves every pre-existing file
// byte-for-byte unchanged, only adds missing scaffold files, registers the
// project at the resolved path, and matches a CLI-init project at rest.
func TestOnboard_ExistingMode_NonDestructive(t *testing.T) {
	env := newCRUDTestEnv(t, nil)
	dir := makeTempProjectDir(t)

	// Pre-populate sentinel content.
	claudeMd := filepath.Join(dir, "CLAUDE.md")
	claudeMdContent := []byte("# My Custom CLAUDE.md\nDo not touch this.\n")
	if err := os.WriteFile(claudeMd, claudeMdContent, 0o644); err != nil {
		t.Fatal(err)
	}
	ideasDir := filepath.Join(dir, "lifecycle", "ideas")
	if err := os.MkdirAll(ideasDir, 0o755); err != nil {
		t.Fatal(err)
	}
	keepMd := filepath.Join(ideasDir, "keep.md")
	keepMdContent := []byte("---\ntitle: Keep me\ntype: idea\nstatus: draft\nlineage: keep\n---\n\nBody.\n")
	if err := os.WriteFile(keepMd, keepMdContent, 0o644); err != nil {
		t.Fatal(err)
	}
	dataTxt := filepath.Join(dir, "data.txt")
	dataTxtContent := []byte("unrelated user data\n")
	if err := os.WriteFile(dataTxt, dataTxtContent, 0o644); err != nil {
		t.Fatal(err)
	}

	resp := env.doRequest("POST", "/api/projects", createProjectBody("existing-nondestructive", "existing", map[string]any{
		"path": dir,
	}))
	requireCRUDStatus(t, resp, 201)
	body := readCRUDJSON(t, resp)

	// Every pre-existing file is byte-for-byte identical after onboarding.
	for path, want := range map[string][]byte{
		claudeMd: claudeMdContent,
		keepMd:   keepMdContent,
		dataTxt:  dataTxtContent,
	} {
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s after onboarding: %v", path, err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s was modified by onboarding; want %q, got %q", path, want, got)
		}
	}

	// Only missing scaffold files/dirs are created — CLAUDE.md must NOT be
	// in the created list since it already existed with custom content.
	created, _ := body["created"].([]any)
	for _, c := range created {
		if c == "CLAUDE.md" {
			t.Error("created list includes CLAUDE.md, which already existed and must be skipped")
		}
	}
	createdSet := make(map[string]bool, len(created))
	for _, c := range created {
		if s, ok := c.(string); ok {
			createdSet[s] = true
		}
	}
	if !createdSet["lifecycle/config.yaml"] {
		t.Errorf("created list missing lifecycle/config.yaml, got %v", created)
	}

	// Project registered at the resolved path and lifecycle/config.yaml exists.
	if resolved, _ := body["resolvedPath"].(string); resolved != dir {
		t.Errorf("resolvedPath = %q, want %q", resolved, dir)
	}
	if _, err := os.Stat(filepath.Join(dir, "lifecycle", "config.yaml")); err != nil {
		t.Fatalf("lifecycle/config.yaml not created: %v", err)
	}

	// Matches a CLI-init project at rest: every reference scaffold file is present.
	requireSubset(t, sortedRelFiles(t, dir), referenceScaffoldPaths(t), "existing mode onboarding")
}

// ---------------------------------------------------------------------------
// Milestone 3 — Existing mode: validation failures (FR4/FR8)
// ---------------------------------------------------------------------------

// TestOnboard_ExistingMode_PathMissing verifies a non-existent path is
// rejected with a path-missing message and writes nothing.
func TestOnboard_ExistingMode_PathMissing(t *testing.T) {
	env := newCRUDTestEnv(t, nil)
	missing := filepath.Join(t.TempDir(), "does-not-exist")

	resp := env.doRequest("POST", "/api/projects", createProjectBody("existing-missing", "existing", map[string]any{
		"path": missing,
	}))
	requireCRUDStatus(t, resp, 400)
	body := readCRUDJSON(t, resp)
	if code := errCode(body); code != "path_missing" {
		t.Errorf("error code = %q, want %q", code, "path_missing")
	}
	if _, err := os.Stat(missing); err == nil {
		t.Error("path was created despite validation failure")
	}
}

// TestOnboard_ExistingMode_NotADirectory verifies a path that is a file (not
// a directory) is rejected with a not-a-directory message.
func TestOnboard_ExistingMode_NotADirectory(t *testing.T) {
	env := newCRUDTestEnv(t, nil)
	dir := makeTempProjectDir(t)
	file := filepath.Join(dir, "im-a-file")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	resp := env.doRequest("POST", "/api/projects", createProjectBody("existing-notdir", "existing", map[string]any{
		"path": file,
	}))
	requireCRUDStatus(t, resp, 400)
	body := readCRUDJSON(t, resp)
	if code := errCode(body); code != "not_a_directory" {
		t.Errorf("error code = %q, want %q", code, "not_a_directory")
	}
}

// TestOnboard_ExistingMode_NotWritable verifies a non-writable directory is
// rejected with a not-writable message and writes nothing.
func TestOnboard_ExistingMode_NotWritable(t *testing.T) {
	skipIfRootOrWindows(t)

	env := newCRUDTestEnv(t, nil)
	dir := makeTempProjectDir(t)
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	resp := env.doRequest("POST", "/api/projects", createProjectBody("existing-notwritable", "existing", map[string]any{
		"path": dir,
	}))
	requireCRUDStatus(t, resp, 400)
	body := readCRUDJSON(t, resp)
	if code := errCode(body); code != "not_writable" {
		t.Errorf("error code = %q, want %q", code, "not_writable")
	}
	if _, err := os.Stat(filepath.Join(dir, "lifecycle")); err == nil {
		t.Error("lifecycle/ was created inside a non-writable directory")
	}
}

// TestOnboard_ExistingMode_AlreadyInitialised verifies that a target already
// containing an initialised project returns alreadyInitialised=true and the
// directory is left unmodified (NFR2).
func TestOnboard_ExistingMode_AlreadyInitialised(t *testing.T) {
	env := newCRUDTestEnv(t, nil)
	dir := makeTempProjectDir(t)
	lcDir := filepath.Join(dir, "lifecycle")
	if err := os.MkdirAll(lcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgContent := []byte("stages: []\n")
	cfgPath := filepath.Join(lcDir, "config.yaml")
	if err := os.WriteFile(cfgPath, cfgContent, 0o644); err != nil {
		t.Fatal(err)
	}

	resp := env.doRequest("POST", "/api/projects", createProjectBody("existing-already-init", "existing", map[string]any{
		"path": dir,
	}))
	requireCRUDStatus(t, resp, 200)
	body := readCRUDJSON(t, resp)
	if already, _ := body["alreadyInitialised"].(bool); !already {
		t.Error("alreadyInitialised = false, want true")
	}

	got, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("reading config.yaml after rejected re-scaffold: %v", err)
	}
	if !bytes.Equal(got, cfgContent) {
		t.Error("lifecycle/config.yaml was modified despite already-initialised rejection")
	}
}

// ---------------------------------------------------------------------------
// Milestone 4 — New mode: happy path & creation semantics (FR6)
// ---------------------------------------------------------------------------

// TestOnboard_NewMode_CreatesCleanProject verifies that new mode creates
// exactly the target directory, scaffolds a clean project into it, registers
// it at the resolved path, and matches a CLI-init project at rest.
func TestOnboard_NewMode_CreatesCleanProject(t *testing.T) {
	env := newCRUDTestEnv(t, nil)
	parent := makeTempProjectDir(t)

	before, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}

	resp := env.doRequest("POST", "/api/projects", createProjectBody("new-mode-clean", "new", map[string]any{
		"parent":  parent,
		"dirName": "brand-new-dir",
	}))
	requireCRUDStatus(t, resp, 201)
	body := readCRUDJSON(t, resp)

	target := filepath.Join(parent, "brand-new-dir")
	if resolved, _ := body["resolvedPath"].(string); resolved != target {
		t.Errorf("resolvedPath = %q, want %q", resolved, target)
	}

	// Only the target itself was created under parent (one new entry).
	after, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before)+1 {
		t.Errorf("parent now has %d entries, want %d (before %d)", len(after), len(before)+1, len(before))
	}

	// A clean project is scaffolded and registered at the resolved path (FR7).
	if _, err := os.Stat(filepath.Join(target, "lifecycle", "config.yaml")); err != nil {
		t.Fatalf("lifecycle/config.yaml not created in new target: %v", err)
	}

	getResp := env.doRequest("GET", "/api/projects/new-mode-clean", nil)
	requireCRUDStatus(t, getResp, 200)
	getBody := readCRUDJSON(t, getResp)
	if getBody["path"] != target {
		t.Errorf("registered path = %v, want %q", getBody["path"], target)
	}

	// Matches a CLI-init project at rest exactly (target started empty).
	got := sortedRelFiles(t, target)
	want := referenceScaffoldPaths(t)
	if len(got) != len(want) {
		t.Errorf("new-mode scaffold has %d files, reference has %d\ngot:  %v\nwant: %v", len(got), len(want), got, want)
	}
	requireSubset(t, got, want, "new mode onboarding")
}

// ---------------------------------------------------------------------------
// Milestone 5 — New mode: validation failures & invalid names (FR4/FR8)
// ---------------------------------------------------------------------------

// TestOnboard_NewMode_TargetExists verifies that an already-existing target
// is rejected with a target-exists message and its content is untouched.
func TestOnboard_NewMode_TargetExists(t *testing.T) {
	env := newCRUDTestEnv(t, nil)
	parent := makeTempProjectDir(t)
	target := filepath.Join(parent, "already-here")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(target, "sentinel.txt")
	sentinelContent := []byte("do not touch\n")
	if err := os.WriteFile(sentinel, sentinelContent, 0o644); err != nil {
		t.Fatal(err)
	}

	resp := env.doRequest("POST", "/api/projects", createProjectBody("new-target-exists", "new", map[string]any{
		"parent":  parent,
		"dirName": "already-here",
	}))
	requireCRUDStatus(t, resp, 400)
	body := readCRUDJSON(t, resp)
	if code := errCode(body); code != "target_exists" {
		t.Errorf("error code = %q, want %q", code, "target_exists")
	}

	got, err := os.ReadFile(sentinel)
	if err != nil {
		t.Fatalf("reading sentinel after rejected create: %v", err)
	}
	if !bytes.Equal(got, sentinelContent) {
		t.Error("existing target content was modified despite target_exists rejection")
	}
}

// TestOnboard_NewMode_ParentMissing verifies a non-existent parent is
// rejected with a parent-missing message.
func TestOnboard_NewMode_ParentMissing(t *testing.T) {
	env := newCRUDTestEnv(t, nil)
	missingParent := filepath.Join(t.TempDir(), "no-such-parent")

	resp := env.doRequest("POST", "/api/projects", createProjectBody("new-parent-missing", "new", map[string]any{
		"parent":  missingParent,
		"dirName": "newdir",
	}))
	requireCRUDStatus(t, resp, 400)
	body := readCRUDJSON(t, resp)
	if code := errCode(body); code != "parent_missing" {
		t.Errorf("error code = %q, want %q", code, "parent_missing")
	}
}

// TestOnboard_NewMode_ParentNotWritable verifies a non-writable parent is
// rejected and nothing is written.
func TestOnboard_NewMode_ParentNotWritable(t *testing.T) {
	skipIfRootOrWindows(t)

	env := newCRUDTestEnv(t, nil)
	parent := makeTempProjectDir(t)
	if err := os.Chmod(parent, 0o500); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { os.Chmod(parent, 0o755) })

	resp := env.doRequest("POST", "/api/projects", createProjectBody("new-parent-notwritable", "new", map[string]any{
		"parent":  parent,
		"dirName": "newdir",
	}))
	requireCRUDStatus(t, resp, 400)
	body := readCRUDJSON(t, resp)
	if code := errCode(body); code != "parent_not_writable" {
		t.Errorf("error code = %q, want %q", code, "parent_not_writable")
	}
	if _, err := os.Stat(filepath.Join(parent, "newdir")); err == nil {
		t.Error("target was created under a non-writable parent")
	}
}

// TestOnboard_NewMode_InvalidDirName verifies that names containing '/',
// '\', '..', or the empty name are rejected before any filesystem write.
func TestOnboard_NewMode_InvalidDirName(t *testing.T) {
	env := newCRUDTestEnv(t, nil)

	cases := []struct {
		desc    string
		dirName string
	}{
		{"empty", ""},
		{"forward slash", "a/b"},
		{"backslash", `a\b`},
		{"traversal", ".."},
	}

	for i, tc := range cases {
		parent := makeTempProjectDir(t)
		before, err := os.ReadDir(parent)
		if err != nil {
			t.Fatal(err)
		}

		resp := env.doRequest("POST", "/api/projects", createProjectBody(
			"new-invalid-name", "new", map[string]any{
				"parent":  parent,
				"dirName": tc.dirName,
			}))
		if resp.StatusCode != 400 {
			b := readCRUDJSON(t, resp)
			t.Errorf("case %d (%s): status = %d, want 400 (body: %v)", i, tc.desc, resp.StatusCode, b)
			continue
		}
		body := readCRUDJSON(t, resp)
		if code := errCode(body); code != "invalid_name" {
			t.Errorf("case %d (%s): error code = %q, want %q", i, tc.desc, code, "invalid_name")
		}

		after, err := os.ReadDir(parent)
		if err != nil {
			t.Fatal(err)
		}
		if len(after) != len(before) {
			t.Errorf("case %d (%s): parent directory was written to before any filesystem write should occur", i, tc.desc)
		}
	}
}

// ---------------------------------------------------------------------------
// Milestone 6 — Cleanup, path safety & normalisation (FR8, NFR1, FR9)
// ---------------------------------------------------------------------------

// TestOnboard_NewMode_ScaffoldFailureRollsBackCreatedDir induces a scaffold
// failure after the target directory has been created (by dropping the
// process umask so os.Mkdir produces a target with no write permission,
// deterministically failing the first nested MkdirAll inside
// initcmd.ScaffoldProject) and verifies the tool removes the directory it
// created (FR8).
func TestOnboard_NewMode_ScaffoldFailureRollsBackCreatedDir(t *testing.T) {
	skipIfRootOrWindows(t)

	env := newCRUDTestEnv(t, nil)
	parent := makeTempProjectDir(t)
	target := filepath.Join(parent, "will-fail")

	// 0o755 masked by umask 0o277 yields a target directory with mode 0o500
	// (r-x for the owner, nothing else): the process can create the empty
	// directory itself but cannot write files/subdirectories inside it, so
	// initcmd.ScaffoldProject's first nested MkdirAll fails deterministically
	// while the (still-empty) target remains fully removable by its parent.
	old := syscall.Umask(0o277)
	resp := env.doRequest("POST", "/api/projects", createProjectBody("new-scaffold-fail", "new", map[string]any{
		"parent":  parent,
		"dirName": "will-fail",
	}))
	syscall.Umask(old)

	requireCRUDStatus(t, resp, 500)
	body := readCRUDJSON(t, resp)
	if code := errCode(body); code != "scaffold_failed" {
		t.Errorf("error code = %q, want %q", code, "scaffold_failed")
	}

	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Errorf("target directory %s was not removed after scaffold failure (stat err: %v)", target, err)
	}
}

// TestOnboard_ExistingMode_ScaffoldFailureNeverRemovesDir induces the same
// class of scaffold failure in existing mode (a plain file blocking the
// lifecycle/ directory MkdirAll) and verifies the pre-existing directory is
// never removed, unlike new mode's rollback.
func TestOnboard_ExistingMode_ScaffoldFailureNeverRemovesDir(t *testing.T) {
	env := newCRUDTestEnv(t, nil)
	dir := makeTempProjectDir(t)

	// A plain file named "lifecycle" blocks os.MkdirAll(dir/lifecycle/ideas, ...).
	blocker := filepath.Join(dir, "lifecycle")
	if err := os.WriteFile(blocker, []byte("i am a file, not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}

	resp := env.doRequest("POST", "/api/projects", createProjectBody("existing-scaffold-fail", "existing", map[string]any{
		"path": dir,
	}))
	requireCRUDStatus(t, resp, 500)
	body := readCRUDJSON(t, resp)
	if code := errCode(body); code != "scaffold_failed" {
		t.Errorf("error code = %q, want %q", code, "scaffold_failed")
	}

	if _, err := os.Stat(dir); err != nil {
		t.Errorf("pre-existing directory %s was removed after scaffold failure: %v", dir, err)
	}
	if _, err := os.Stat(blocker); err != nil {
		t.Errorf("pre-existing blocker file %s was removed after scaffold failure: %v", blocker, err)
	}
}

// TestOnboard_NewMode_TraversalNameCannotEscapeParent verifies a crafted
// traversal directory name is rejected and never joined onto the filesystem
// outside the chosen parent.
func TestOnboard_NewMode_TraversalNameCannotEscapeParent(t *testing.T) {
	env := newCRUDTestEnv(t, nil)
	parent := makeTempProjectDir(t)
	grandparent := filepath.Dir(parent)

	beforeGP, err := os.ReadDir(grandparent)
	if err != nil {
		t.Fatal(err)
	}

	resp := env.doRequest("POST", "/api/projects", createProjectBody("new-traversal", "new", map[string]any{
		"parent":  parent,
		"dirName": "../escape",
	}))
	requireCRUDStatus(t, resp, 400)
	body := readCRUDJSON(t, resp)
	if code := errCode(body); code != "invalid_name" {
		t.Errorf("error code = %q, want %q", code, "invalid_name")
	}

	afterGP, err := os.ReadDir(grandparent)
	if err != nil {
		t.Fatal(err)
	}
	if len(afterGP) != len(beforeGP) {
		t.Error("a traversal directory name caused a write outside the chosen parent")
	}
	if _, err := os.Stat(filepath.Join(grandparent, "escape")); err == nil {
		t.Error("traversal name 'escaped' the parent and created a sibling directory")
	}
}

// TestOnboard_ExistingMode_TargetInsideConfigDirRejected verifies a resolved
// target inside the kaos-control config directory is rejected (NFR1).
func TestOnboard_ExistingMode_TargetInsideConfigDirRejected(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	cfgDir := filepath.Join(xdg, "kaos-control")
	inside := filepath.Join(cfgDir, "sub")
	if err := os.MkdirAll(inside, 0o755); err != nil {
		t.Fatal(err)
	}

	env := newCRUDTestEnv(t, nil)
	resp := env.doRequest("POST", "/api/projects", createProjectBody("existing-in-config-dir", "existing", map[string]any{
		"path": inside,
	}))
	requireCRUDStatus(t, resp, 400)
	body := readCRUDJSON(t, resp)
	if code := errCode(body); code != "invalid_path" {
		t.Errorf("error code = %q, want %q", code, "invalid_path")
	}
}

// TestOnboard_NewMode_TargetInsideConfigDirRejected verifies the same NFR1
// guard for new mode, where the parent itself resolves inside the config dir.
func TestOnboard_NewMode_TargetInsideConfigDirRejected(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	cfgDir := filepath.Join(xdg, "kaos-control")
	parentInside := filepath.Join(cfgDir, "projects")
	if err := os.MkdirAll(parentInside, 0o755); err != nil {
		t.Fatal(err)
	}

	env := newCRUDTestEnv(t, nil)
	resp := env.doRequest("POST", "/api/projects", createProjectBody("new-in-config-dir", "new", map[string]any{
		"parent":  parentInside,
		"dirName": "newproj",
	}))
	requireCRUDStatus(t, resp, 400)
	body := readCRUDJSON(t, resp)
	if code := errCode(body); code != "invalid_path" {
		t.Errorf("error code = %q, want %q", code, "invalid_path")
	}
	if _, err := os.Stat(filepath.Join(parentInside, "newproj")); err == nil {
		t.Error("target was created inside the kaos-control config directory")
	}
}

// TestCheckDirectory_ExistingMode_NormalisesWhitespaceAndTilde verifies FR9:
// a whitespace-padded and "~"-prefixed input resolves to the expected
// absolute path in the check-directory preview.
func TestCheckDirectory_ExistingMode_NormalisesWhitespaceAndTilde(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	sub := filepath.Join(home, "sub-project")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	env := newCRUDTestEnv(t, nil)
	resp := env.doRequest("POST", "/api/projects/check-directory", map[string]any{
		"mode": "existing",
		"path": "  ~/sub-project  ",
	})
	requireCRUDStatus(t, resp, 200)
	body := readCRUDJSON(t, resp)

	if resolved, _ := body["resolvedPath"].(string); resolved != sub {
		t.Errorf("resolvedPath = %q, want %q", resolved, sub)
	}
	if exists, _ := body["exists"].(bool); !exists {
		t.Error("exists = false, want true for the normalised path")
	}
}

// TestOnboard_NewMode_ResolvedPathMatchesWrittenPath verifies FR9's stronger
// claim end-to-end: for a whitespace-padded, "~"-prefixed parent, the
// resolvedPath reported by the create endpoint is exactly where the project
// was actually written on disk.
func TestOnboard_NewMode_ResolvedPathMatchesWrittenPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	parentDir := filepath.Join(home, "proj-parent")
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	resolvedParent, err := filepath.EvalSymlinks(parentDir)
	if err != nil {
		resolvedParent = parentDir
	}

	env := newCRUDTestEnv(t, nil)
	resp := env.doRequest("POST", "/api/projects", createProjectBody("new-fr9-path", "new", map[string]any{
		"parent":  "  ~/proj-parent  ",
		"dirName": "newdir",
	}))
	requireCRUDStatus(t, resp, 201)
	body := readCRUDJSON(t, resp)

	wantTarget := filepath.Join(resolvedParent, "newdir")
	resolved, _ := body["resolvedPath"].(string)
	if resolved != wantTarget {
		t.Errorf("resolvedPath = %q, want %q", resolved, wantTarget)
	}
	if _, err := os.Stat(filepath.Join(resolved, "lifecycle", "config.yaml")); err != nil {
		t.Errorf("project was not actually written at the reported resolvedPath %q: %v", resolved, err)
	}
}
