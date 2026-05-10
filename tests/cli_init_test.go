// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

// Package cli_test contains integration tests that exercise the
// kaos-control CLI (specifically the `init` subcommand) by invoking
// the compiled binary in a subprocess.
package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// binPath is the path to the compiled kaos-control binary, built once per
// test run inside TestMain.
var binPath string

// TestMain builds the kaos-control binary into a temp directory before any
// test runs and removes it afterward. The working directory for `go test
// ./tests/` is tests/, so "../" is the repo root.
func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "kaos-control-cli-bin-*")
	if err != nil {
		panic("MkdirTemp: " + err.Error())
	}
	defer os.RemoveAll(tmp)

	name := "kaos-control"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binPath = filepath.Join(tmp, name)

	build := exec.Command("go", "build", "-o", binPath, "./cmd/kaos-control")
	build.Dir = ".."
	if out, err := build.CombinedOutput(); err != nil {
		panic("go build failed:\n" + string(out))
	}

	os.Exit(m.Run())
}

// runInit executes "kaos-control init <args...>" and returns stdout, stderr,
// and the process exit code. A non-zero exit code is not a test failure on
// its own; callers assert it explicitly.
func runInit(t *testing.T, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command(binPath, append([]string{"init"}, args...)...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		if ex, ok := err.(*exec.ExitError); ok {
			code = ex.ExitCode()
		} else {
			t.Fatalf("exec: %v", err)
		}
	}
	return outBuf.String(), errBuf.String(), code
}

// lifecycleDirs lists the eleven directories that `init` must create
// (forward-slash paths; use filepath.FromSlash for OS-native paths).
var lifecycleDirs = []string{
	"lifecycle/ideas",
	"lifecycle/requirements",
	"lifecycle/backend-plans",
	"lifecycle/frontend-plans",
	"lifecycle/test-plans",
	"lifecycle/tests",
	"lifecycle/prototypes",
	"lifecycle/releases",
	"lifecycle/sprints",
	"lifecycle/defects",
	"tests",
}

// seedFiles lists the four seed files that `init` writes
// (forward-slash paths as they appear in stdout output).
var seedFiles = []string{
	"lifecycle/config.yaml",
	"CLAUDE.md",
	".claude/settings.json",
	".gitignore",
}

// claudeMdSections are the five required top-level sections in CLAUDE.md.
var claudeMdSections = []string{
	"## Repository Layout",
	"## Lineage Filename Convention",
	"## Frontmatter Requirements",
	"## Commit Conventions",
	"## Agent Roles",
}

// ─── Milestone 4: Full Init Flow (Empty Directory) ───────────────────────────

// TestInit_FullFlow_EmptyDir verifies that running init on an empty temp dir
// produces the complete lifecycle scaffold with correct exit code and output.
func TestInit_FullFlow_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	stdout, _, code := runInit(t, dir)

	// Exit code.
	if code != 0 {
		t.Fatalf("want exit 0, got %d\nstdout:\n%s", code, stdout)
	}

	// FR-7: success banner must name the absolute path.
	wantBanner := "Initialized kaos-control project at " + dir
	if !strings.Contains(stdout, wantBanner) {
		t.Errorf("stdout missing init banner\nwant substring: %q\ngot:\n%s", wantBanner, stdout)
	}

	// All 11 directories must exist with .gitkeep and appear as "created".
	for _, d := range lifecycleDirs {
		gk := filepath.Join(dir, filepath.FromSlash(d), ".gitkeep")
		if _, err := os.Stat(gk); err != nil {
			t.Errorf("missing .gitkeep in %s: %v", d, err)
		}
		wantLine := "created  " + filepath.Join(filepath.FromSlash(d), ".gitkeep")
		if !strings.Contains(stdout, wantLine) {
			t.Errorf("stdout does not list %q as created\nstdout:\n%s", wantLine, stdout)
		}
	}

	// All 4 seed files must exist and appear as "created".
	for _, f := range seedFiles {
		absF := filepath.Join(dir, filepath.FromSlash(f))
		if _, err := os.Stat(absF); err != nil {
			t.Errorf("missing seed file %s: %v", f, err)
		}
		wantLine := "created  " + f
		if !strings.Contains(stdout, wantLine) {
			t.Errorf("stdout does not list %q as created\nstdout:\n%s", wantLine, stdout)
		}
	}

	// lifecycle/config.yaml must parse as valid YAML.
	cfgData, err := os.ReadFile(filepath.Join(dir, "lifecycle", "config.yaml"))
	if err != nil {
		t.Fatalf("reading lifecycle/config.yaml: %v", err)
	}
	var yamlOut any
	if err := yaml.Unmarshal(cfgData, &yamlOut); err != nil {
		t.Errorf("lifecycle/config.yaml is not valid YAML: %v", err)
	}

	// .claude/settings.json must parse as valid JSON.
	settingsData, err := os.ReadFile(filepath.Join(dir, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("reading .claude/settings.json: %v", err)
	}
	var jsonOut any
	if err := json.Unmarshal(settingsData, &jsonOut); err != nil {
		t.Errorf(".claude/settings.json is not valid JSON: %v", err)
	}

	// CLAUDE.md must contain all five required sections.
	claudeMdData, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("reading CLAUDE.md: %v", err)
	}
	claudeMd := string(claudeMdData)
	for _, section := range claudeMdSections {
		if !strings.Contains(claudeMd, section) {
			t.Errorf("CLAUDE.md missing section %q", section)
		}
	}

	// .gitignore must contain the SQLite index pattern.
	gitignoreData, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}
	if !strings.Contains(string(gitignoreData), "lifecycle/.kaos-control.db") {
		t.Error(".gitignore missing 'lifecycle/.kaos-control.db'")
	}
}

// ─── Milestone 5: Idempotency (Double Run) ───────────────────────────────────

// TestInit_Idempotency verifies that a second run without --force produces no
// writes: all items are reported as skipped and file contents are unchanged.
func TestInit_Idempotency(t *testing.T) {
	dir := t.TempDir()

	// First run — establishes the scaffold.
	if _, _, code := runInit(t, dir); code != 0 {
		t.Fatalf("first run: want exit 0, got %d", code)
	}

	// Snapshot seed file contents after first run.
	snapshot := make(map[string][]byte, len(seedFiles))
	for _, f := range seedFiles {
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(f)))
		if err != nil {
			t.Fatalf("snapshot read %s: %v", f, err)
		}
		snapshot[f] = data
	}

	// Second run — no force flags.
	stdout, stderr, code := runInit(t, dir)
	if code != 0 {
		t.Fatalf("second run: want exit 0, got %d", code)
	}

	// Stdout must contain no "created" items.
	if strings.Contains(stdout, "  created  ") {
		t.Errorf("second run stdout still contains 'created'\nstdout:\n%s", stdout)
	}

	// All 11 directories must appear as "skipped" in stdout.
	for _, d := range lifecycleDirs {
		wantLine := "skipped  " + filepath.Join(filepath.FromSlash(d), ".gitkeep") + " (already exists)"
		if !strings.Contains(stdout, wantLine) {
			t.Errorf("second run: directory %s not shown as skipped\nstdout:\n%s", d, stdout)
		}
	}

	// All 4 seed files must appear as "skipped" in stdout.
	for _, f := range seedFiles {
		wantLine := "skipped  " + f + " (already exists)"
		if !strings.Contains(stdout, wantLine) {
			t.Errorf("second run: seed file %s not shown as skipped in stdout\nstdout:\n%s", f, stdout)
		}
	}

	// Stderr must include the --force hint for each seed file.
	for _, f := range seedFiles {
		wantMsg := "skipped: " + f + " (already exists; use --force to overwrite)"
		if !strings.Contains(stderr, wantMsg) {
			t.Errorf("second run: stderr missing skip message for %s\nstderr:\n%s", f, stderr)
		}
	}

	// Seed file contents must be byte-identical after the second run.
	for _, f := range seedFiles {
		after, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(f)))
		if err != nil {
			t.Fatalf("post-run read %s: %v", f, err)
		}
		if !bytes.Equal(snapshot[f], after) {
			t.Errorf("%s was modified by the second run (contents changed)", f)
		}
	}
}

// ─── Milestone 6: Force Flags ─────────────────────────────────────────────────

// TestInit_ForceFlags verifies granular and blanket force flags.
func TestInit_ForceFlags(t *testing.T) {
	// --force-config overwrites only lifecycle/config.yaml.
	t.Run("force-config", func(t *testing.T) {
		dir := t.TempDir()
		if _, _, code := runInit(t, dir); code != 0 {
			t.Fatalf("first run: want exit 0, got %d", code)
		}

		// Snapshot CLAUDE.md to verify it remains unchanged.
		origClaudeMd, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
		if err != nil {
			t.Fatal(err)
		}

		// Corrupt lifecycle/config.yaml so we can detect if it is overwritten.
		if err := os.WriteFile(
			filepath.Join(dir, "lifecycle", "config.yaml"),
			[]byte("INVALID: yaml: [broken"),
			0o644,
		); err != nil {
			t.Fatal(err)
		}

		if _, _, code := runInit(t, "--force-config", dir); code != 0 {
			t.Fatalf("force-config run: want exit 0, got %d", code)
		}

		// config.yaml must now be valid YAML (was overwritten).
		cfgData, err := os.ReadFile(filepath.Join(dir, "lifecycle", "config.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		var out any
		if err := yaml.Unmarshal(cfgData, &out); err != nil {
			t.Errorf("lifecycle/config.yaml still invalid after --force-config: %v", err)
		}

		// CLAUDE.md must be byte-identical (was NOT overwritten).
		afterClaudeMd, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(origClaudeMd, afterClaudeMd) {
			t.Error("CLAUDE.md was modified by --force-config (should be skipped)")
		}
	})

	// --force overwrites all four seed files; directories and non-seed files are untouched.
	t.Run("force-all", func(t *testing.T) {
		dir := t.TempDir()
		if _, _, code := runInit(t, dir); code != 0 {
			t.Fatalf("first run: want exit 0, got %d", code)
		}

		// Overwrite all seed files with a recognisable marker.
		for _, f := range seedFiles {
			if err := os.WriteFile(
				filepath.Join(dir, filepath.FromSlash(f)),
				[]byte("MARKER"),
				0o644,
			); err != nil {
				t.Fatalf("planting marker in %s: %v", f, err)
			}
		}

		// Plant a non-seed file that must never be touched.
		nonSeed := filepath.Join(dir, "custom.txt")
		if err := os.WriteFile(nonSeed, []byte("user file"), 0o644); err != nil {
			t.Fatal(err)
		}

		stdout, _, code := runInit(t, "--force", dir)
		if code != 0 {
			t.Fatalf("--force run: want exit 0, got %d", code)
		}

		// All seed files must have been overwritten (no longer "MARKER").
		for _, f := range seedFiles {
			data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(f)))
			if err != nil {
				t.Fatalf("reading %s: %v", f, err)
			}
			if string(data) == "MARKER" {
				t.Errorf("%s was NOT overwritten by --force", f)
			}
			if !strings.Contains(stdout, "created  "+f) {
				t.Errorf("stdout does not list %s as created after --force\nstdout:\n%s", f, stdout)
			}
		}

		// All 11 directories must still be intact.
		for _, d := range lifecycleDirs {
			gk := filepath.Join(dir, filepath.FromSlash(d), ".gitkeep")
			if _, err := os.Stat(gk); err != nil {
				t.Errorf("directory %s .gitkeep missing after --force: %v", d, err)
			}
		}

		// Non-seed file must be completely untouched.
		custom, err := os.ReadFile(nonSeed)
		if err != nil {
			t.Fatal(err)
		}
		if string(custom) != "user file" {
			t.Errorf("non-seed file was modified by --force (got %q)", string(custom))
		}
	})
}

// ─── Milestone 7: Non-Existent Target Path ────────────────────────────────────

// TestInit_NonExistentPath verifies that init creates intermediate directories
// when the target path does not yet exist.
func TestInit_NonExistentPath(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "nested", "project")

	_, _, code := runInit(t, target)
	if code != 0 {
		t.Fatalf("want exit 0, got %d", code)
	}

	// Target directory must now exist.
	if _, err := os.Stat(target); err != nil {
		t.Errorf("target directory was not created: %v", err)
	}

	// Full scaffold must be present inside the created directory.
	for _, d := range lifecycleDirs {
		gk := filepath.Join(target, filepath.FromSlash(d), ".gitkeep")
		if _, err := os.Stat(gk); err != nil {
			t.Errorf("scaffold missing %s/.gitkeep in created directory: %v", d, err)
		}
	}
}

// ─── Milestone 8: Existing Repo With Code ─────────────────────────────────────

// TestInit_ExistingCodeRepo verifies that running init in a directory that
// already contains go.mod, main.go, and README.md leaves those files unchanged.
func TestInit_ExistingCodeRepo(t *testing.T) {
	dir := t.TempDir()

	// Pre-populate the directory with existing project files.
	existing := map[string][]byte{
		"go.mod":    []byte("module example.com/myproject\n\ngo 1.22\n"),
		"main.go":   []byte("package main\n\nfunc main() {}\n"),
		"README.md": []byte("# My Project\n"),
	}
	for name, content := range existing {
		if err := os.WriteFile(filepath.Join(dir, name), content, 0o644); err != nil {
			t.Fatalf("seeding %s: %v", name, err)
		}
	}

	_, _, code := runInit(t, dir)
	if code != 0 {
		t.Fatalf("want exit 0, got %d", code)
	}

	// Pre-existing files must be byte-identical.
	for name, want := range existing {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		if !bytes.Equal(want, got) {
			t.Errorf("%s was modified by init (content changed)", name)
		}
	}

	// Lifecycle scaffold must exist alongside the existing files.
	gk := filepath.Join(dir, "lifecycle", "ideas", ".gitkeep")
	if _, err := os.Stat(gk); err != nil {
		t.Errorf("lifecycle scaffold not created: %v", err)
	}
}

// ─── Milestone 9: Error Cases ─────────────────────────────────────────────────

// TestInit_ErrorCases verifies correct exit codes for known failure modes.
func TestInit_ErrorCases(t *testing.T) {
	// Unwritable target: init must exit 1 with a meaningful error message.
	t.Run("unwritable-path", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("running as root: filesystem permission restrictions are not enforced")
		}

		// Create a read-only parent directory. Cleanup restores permissions
		// before removal so that t.TempDir-style cleanup succeeds.
		parent, err := os.MkdirTemp("", "kaos-control-ro-*")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_ = os.Chmod(parent, 0o755)
			_ = os.RemoveAll(parent)
		})
		if err := os.Chmod(parent, 0o444); err != nil {
			t.Fatal(err)
		}

		target := filepath.Join(parent, "new-project")
		_, stderr, code := runInit(t, target)

		if code != 1 {
			t.Errorf("want exit 1 for unwritable path, got %d", code)
		}
		if stderr == "" {
			t.Error("expected error output in stderr for unwritable path, got nothing")
		}
	})

	// Unknown flag: init must exit 1 and emit usage/error text.
	t.Run("unknown-flag", func(t *testing.T) {
		_, stderr, code := runInit(t, "--this-flag-does-not-exist-xyz")

		if code != 1 {
			t.Errorf("want exit 1 for unknown flag, got %d", code)
		}
		if stderr == "" {
			t.Errorf("expected usage/error output in stderr for unknown flag, got nothing")
		}
	})
}
