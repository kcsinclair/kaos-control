// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Tests for auto-create-projects-dir: verifies that config.LoadApp creates
// the projects/ and data/ directories automatically on first run.
//
// Covers test plan: lifecycle/test-plans/auto-create-projects-dir-4-test.md

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/kaos-control/kaos-control/internal/config"
)

// TestLoadApp_CreatesMissingProjectsDir verifies that LoadApp creates
// projects/ and data/ directories (with 0o700 permissions) when they do
// not yet exist. Covers Milestone 1.
func TestLoadApp_CreatesMissingProjectsDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission checks not applicable on Windows")
	}

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	// Neither projects/ nor data/ should exist yet.
	projectsDir := filepath.Join(tmpDir, "projects")
	dataDir := filepath.Join(tmpDir, "data")
	for _, d := range []string{projectsDir, dataDir} {
		if _, err := os.Stat(d); !os.IsNotExist(err) {
			t.Fatalf("pre-condition failed: %s already exists", d)
		}
	}

	cfg, err := config.LoadApp(cfgPath)
	if err != nil {
		t.Fatalf("LoadApp returned unexpected error: %v", err)
	}
	_ = cfg

	for _, dir := range []struct {
		name string
		path string
	}{
		{"projects", projectsDir},
		{"data", dataDir},
	} {
		info, err := os.Stat(dir.path)
		if err != nil {
			t.Errorf("%s dir not created: %v", dir.name, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s path exists but is not a directory", dir.name)
		}
		// Check permissions: expect 0o700.
		got := info.Mode().Perm()
		if got != 0o700 {
			t.Errorf("%s dir permissions = %04o, want %04o", dir.name, got, 0o700)
		}
	}
}

// TestLoadApp_IdempotentWhenDirsExist verifies that LoadApp succeeds and
// leaves pre-existing directories untouched when projects/ and data/ already
// exist. Covers Milestone 2.
func TestLoadApp_IdempotentWhenDirsExist(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	projectsDir := filepath.Join(tmpDir, "projects")
	dataDir := filepath.Join(tmpDir, "data")

	// Pre-create the directories with 0o755.
	for _, d := range []string{projectsDir, dataDir} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatalf("failed to pre-create %s: %v", d, err)
		}
	}

	cfg, err := config.LoadApp(cfgPath)
	if err != nil {
		t.Fatalf("LoadApp returned unexpected error: %v", err)
	}
	_ = cfg

	for _, dir := range []struct {
		name string
		path string
	}{
		{"projects", projectsDir},
		{"data", dataDir},
	} {
		info, err := os.Stat(dir.path)
		if err != nil {
			t.Errorf("%s dir no longer exists after LoadApp: %v", dir.name, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s path is not a directory after LoadApp", dir.name)
		}
		if runtime.GOOS != "windows" {
			// MkdirAll must not change permissions of existing directories.
			got := info.Mode().Perm()
			if got != 0o755 {
				t.Errorf("%s dir permissions changed: got %04o, want %04o", dir.name, got, 0o755)
			}
		}
	}
}

// TestLoadApp_ErrorOnUnwritableParent verifies that LoadApp returns a
// meaningful error when os.MkdirAll cannot create projects/ or data/ because
// the target parent directory is read-only. Covers Milestone 3.
//
// Strategy: write a config file that points projects_dir at a path inside a
// read-only directory. LoadApp reads the config successfully but then cannot
// create the projects directory.
func TestLoadApp_ErrorOnUnwritableParent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("read-only directory enforcement is unreliable on Windows")
	}
	// Skip when running as root because root ignores permission bits.
	if os.Getuid() == 0 {
		t.Skip("running as root — permission restrictions are not enforced")
	}

	tmpDir := t.TempDir()

	// Create a directory that will hold the config file.
	cfgDir := filepath.Join(tmpDir, "config-home")
	if err := os.Mkdir(cfgDir, 0o755); err != nil {
		t.Fatalf("creating cfgDir: %v", err)
	}

	// Create a separate directory that will become read-only; projects/ and
	// data/ will be requested inside it.
	readonlyParent := filepath.Join(tmpDir, "readonly-target")
	if err := os.Mkdir(readonlyParent, 0o755); err != nil {
		t.Fatalf("creating readonlyParent: %v", err)
	}

	// Write a minimal config file that directs LoadApp to create projects/ and
	// data/ inside the soon-to-be-read-only directory.
	cfgContent := "server:\n  listen: \":8042\"\nauth:\n  method: local\n  session_ttl: 720h\nprojects_dir: " +
		filepath.Join(readonlyParent, "projects") + "\ndata_dir: " + filepath.Join(readonlyParent, "data") + "\n"
	cfgPath := filepath.Join(cfgDir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("writing config file: %v", err)
	}

	// Now make the target parent read-only so MkdirAll fails.
	if err := os.Chmod(readonlyParent, 0o444); err != nil {
		t.Fatalf("chmod readonlyParent: %v", err)
	}
	// Restore write permission so t.TempDir cleanup can remove it.
	t.Cleanup(func() { _ = os.Chmod(readonlyParent, 0o755) })

	_, err := config.LoadApp(cfgPath)
	if err == nil {
		t.Fatal("LoadApp succeeded with read-only target directory, expected an error")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "creating projects dir") && !strings.Contains(errStr, "creating data dir") {
		t.Errorf("error message %q does not mention 'creating projects dir' or 'creating data dir'", errStr)
	}
}
