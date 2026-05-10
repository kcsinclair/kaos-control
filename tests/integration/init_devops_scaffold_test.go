// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Milestone 1 — Init Scaffold: devops/ directory creation
//
// Tests for kaos-control init covering:
//   - devops/ directory and .gitkeep are created
//   - devops/sample.yaml is created and parseable
//   - Running init twice is idempotent (no error, sample.yaml unchanged)
//   - Pre-existing files in devops/ are preserved

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/kaos-control/kaos-control/internal/devops"
	"github.com/kaos-control/kaos-control/internal/initcmd"
)

// TestInit_CreatesDevopsDir verifies that kaos-control init creates a devops/
// directory containing a .gitkeep file.
func TestInit_CreatesDevopsDir(t *testing.T) {
	dir := t.TempDir()

	if err := initcmd.Run([]string{dir}); err != nil {
		t.Fatalf("initcmd.Run failed: %v", err)
	}

	devopsDirPath := filepath.Join(dir, "devops")
	info, err := os.Stat(devopsDirPath)
	if err != nil {
		t.Fatalf("devops/ directory not found after init: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("devops/ exists but is not a directory")
	}

	gitkeep := filepath.Join(devopsDirPath, ".gitkeep")
	if _, err := os.Stat(gitkeep); err != nil {
		t.Fatalf("devops/.gitkeep not found after init: %v", err)
	}
}

// TestInit_CreatesSamplePipeline verifies that kaos-control init creates
// devops/sample.yaml and that it parses as a valid pipeline with the expected
// name ("Sample Pipeline"), type ("build"), and exactly one step.
func TestInit_CreatesSamplePipeline(t *testing.T) {
	dir := t.TempDir()

	if err := initcmd.Run([]string{dir}); err != nil {
		t.Fatalf("initcmd.Run failed: %v", err)
	}

	samplePath := filepath.Join(dir, "devops", "sample.yaml")
	if _, err := os.Stat(samplePath); err != nil {
		t.Fatalf("devops/sample.yaml not found after init: %v", err)
	}

	// Use devops.Discover to parse the devops/ directory.
	pipelines, warnings := devops.Discover(filepath.Join(dir, "devops"))
	for _, w := range warnings {
		t.Errorf("unexpected discovery warning: %v", w)
	}
	if len(pipelines) == 0 {
		t.Fatal("devops.Discover returned no pipelines; expected at least the sample")
	}

	var found *devops.Pipeline
	for i := range pipelines {
		if pipelines[i].Slug == "sample" {
			found = &pipelines[i]
			break
		}
	}
	if found == nil {
		slugs := make([]string, len(pipelines))
		for i, p := range pipelines {
			slugs[i] = p.Slug
		}
		t.Fatalf("sample pipeline not found; discovered slugs: %v", slugs)
	}
	if found.Name != "Sample Pipeline" {
		t.Errorf("sample pipeline name = %q, want %q", found.Name, "Sample Pipeline")
	}
	if found.Type != "build" {
		t.Errorf("sample pipeline type = %q, want %q", found.Type, "build")
	}
	if len(found.Steps) != 1 {
		t.Errorf("sample pipeline step count = %d, want 1", len(found.Steps))
	}
}

// TestInit_Idempotent_DevopsDir verifies that running kaos-control init twice
// on the same directory produces no error and leaves devops/sample.yaml with
// identical content (i.e. the second run does not modify the file).
func TestInit_Idempotent_DevopsDir(t *testing.T) {
	dir := t.TempDir()

	if err := initcmd.Run([]string{dir}); err != nil {
		t.Fatalf("first initcmd.Run failed: %v", err)
	}

	samplePath := filepath.Join(dir, "devops", "sample.yaml")
	contentBefore, err := os.ReadFile(samplePath)
	if err != nil {
		t.Fatalf("reading devops/sample.yaml before second init: %v", err)
	}

	if err := initcmd.Run([]string{dir}); err != nil {
		t.Fatalf("second initcmd.Run failed: %v", err)
	}

	contentAfter, err := os.ReadFile(samplePath)
	if err != nil {
		t.Fatalf("reading devops/sample.yaml after second init: %v", err)
	}

	if !bytes.Equal(contentBefore, contentAfter) {
		t.Error("devops/sample.yaml was modified by the second init run; init must be idempotent")
	}
}

// TestInit_PreservesExistingDevops verifies that kaos-control init does not
// modify files already present in devops/ (e.g. a user-authored pipeline).
func TestInit_PreservesExistingDevops(t *testing.T) {
	dir := t.TempDir()

	// Create devops/ and write a custom pipeline before init.
	devopsDirPath := filepath.Join(dir, "devops")
	if err := os.MkdirAll(devopsDirPath, 0o755); err != nil {
		t.Fatalf("pre-creating devops/ dir: %v", err)
	}
	customContent := []byte("name: Custom\ntype: deploy\nsteps:\n  - name: Deploy\n    command: make deploy\n")
	customPath := filepath.Join(devopsDirPath, "custom.yaml")
	if err := os.WriteFile(customPath, customContent, 0o644); err != nil {
		t.Fatalf("writing custom.yaml: %v", err)
	}

	if err := initcmd.Run([]string{dir}); err != nil {
		t.Fatalf("initcmd.Run failed: %v", err)
	}

	// custom.yaml must be byte-for-byte identical after init.
	after, err := os.ReadFile(customPath)
	if err != nil {
		t.Fatalf("reading custom.yaml after init: %v", err)
	}
	if !bytes.Equal(customContent, after) {
		t.Error("custom.yaml was modified by init; pre-existing pipeline files must be preserved")
	}
}
