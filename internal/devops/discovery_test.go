// SPDX-License-Identifier: AGPL-3.0-or-later

package devops_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/devops"
)

func writeYAML(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("writeYAML: %v", err)
	}
}

func TestDiscover_ValidFile(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "deploy.yaml", `
name: Deploy to Production
type: deploy
steps:
  - name: Build
    description: Compile the binary
    command: make build
  - name: Push
    command: make push
`)
	pipelines, warnings := devops.Discover(dir)
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got: %v", warnings)
	}
	if len(pipelines) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(pipelines))
	}
	p := pipelines[0]
	if p.Slug != "deploy" {
		t.Errorf("slug = %q, want %q", p.Slug, "deploy")
	}
	if p.Name != "Deploy to Production" {
		t.Errorf("name = %q, want %q", p.Name, "Deploy to Production")
	}
	if p.Type != "deploy" {
		t.Errorf("type = %q, want %q", p.Type, "deploy")
	}
	if len(p.Steps) != 2 {
		t.Fatalf("steps len = %d, want 2", len(p.Steps))
	}
	if p.Steps[0].Name != "Build" {
		t.Errorf("step[0].Name = %q, want %q", p.Steps[0].Name, "Build")
	}
	if p.Steps[0].Description != "Compile the binary" {
		t.Errorf("step[0].Description = %q", p.Steps[0].Description)
	}
	if p.Steps[0].Command != "make build" {
		t.Errorf("step[0].Command = %q", p.Steps[0].Command)
	}
}

func TestDiscover_DefaultTimeout(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "test.yaml", `
name: Test Suite
type: test
steps:
  - name: Run Tests
    command: go test ./...
`)
	pipelines, _ := devops.Discover(dir)
	if len(pipelines) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(pipelines))
	}
	want := 60 * time.Second
	if pipelines[0].Steps[0].Timeout != want {
		t.Errorf("default timeout = %v, want %v", pipelines[0].Steps[0].Timeout, want)
	}
}

func TestDiscover_CustomTimeout(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "long.yaml", `
name: Long Build
type: build
steps:
  - name: Slow Step
    command: make slow
    timeout: 5m
`)
	pipelines, _ := devops.Discover(dir)
	if len(pipelines) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(pipelines))
	}
	want := 5 * time.Minute
	if pipelines[0].Steps[0].Timeout != want {
		t.Errorf("timeout = %v, want %v", pipelines[0].Steps[0].Timeout, want)
	}
}

func TestDiscover_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "bad.yaml", `{not: valid: yaml: [`)

	pipelines, warnings := devops.Discover(dir)
	if len(pipelines) != 0 {
		t.Errorf("expected 0 pipelines, got %d", len(pipelines))
	}
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(warnings))
	}
}

func TestDiscover_MissingName(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "noname.yaml", `
type: deploy
steps:
  - name: Step
    command: echo hi
`)
	pipelines, warnings := devops.Discover(dir)
	if len(pipelines) != 0 {
		t.Errorf("expected 0 pipelines, got %d", len(pipelines))
	}
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning for missing name, got %d", len(warnings))
	}
}

func TestDiscover_MissingType(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "notype.yaml", `
name: My Pipeline
steps:
  - name: Step
    command: echo hi
`)
	pipelines, warnings := devops.Discover(dir)
	if len(pipelines) != 0 {
		t.Errorf("expected 0 pipelines, got %d", len(pipelines))
	}
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning for missing type, got %d", len(warnings))
	}
}

func TestDiscover_MissingSteps(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "nosteps.yaml", `
name: Empty
type: build
steps: []
`)
	pipelines, warnings := devops.Discover(dir)
	if len(pipelines) != 0 {
		t.Errorf("expected 0 pipelines, got %d", len(pipelines))
	}
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning for missing steps, got %d", len(warnings))
	}
}

func TestDiscover_StepMissingCommand(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "nocommand.yaml", `
name: Bad Step
type: build
steps:
  - name: No Command
`)
	pipelines, warnings := devops.Discover(dir)
	if len(pipelines) != 0 {
		t.Errorf("expected 0 pipelines, got %d", len(pipelines))
	}
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning for missing command, got %d", len(warnings))
	}
}

func TestDiscover_InvalidStepTimeout(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "badtimeout.yaml", `
name: Bad Timeout
type: build
steps:
  - name: Step
    command: echo hi
    timeout: notaduration
`)
	pipelines, warnings := devops.Discover(dir)
	if len(pipelines) != 0 {
		t.Errorf("expected 0 pipelines, got %d", len(pipelines))
	}
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning for invalid timeout, got %d", len(warnings))
	}
}

func TestDiscover_NonexistentDir(t *testing.T) {
	pipelines, warnings := devops.Discover("/tmp/nonexistent-devops-dir-12345")
	if len(pipelines) != 0 {
		t.Errorf("expected 0 pipelines for nonexistent dir, got %d", len(pipelines))
	}
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings for nonexistent dir, got %d", len(warnings))
	}
}

func TestDiscover_MultipleFiles_MixedValidity(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "valid.yaml", `
name: Valid Pipeline
type: deploy
steps:
  - name: Deploy
    command: make deploy
`)
	writeYAML(t, dir, "invalid.yaml", `
name: Invalid
`)
	pipelines, warnings := devops.Discover(dir)
	if len(pipelines) != 1 {
		t.Errorf("expected 1 valid pipeline, got %d", len(pipelines))
	}
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(warnings))
	}
}

func TestDiscover_IgnoresNonYAMLFiles(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "pipeline.yaml", `
name: Good
type: deploy
steps:
  - name: Step
    command: echo ok
`)
	// Write a non-.yaml file that would fail parsing if included.
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# docs"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "script.sh"), []byte("#!/bin/sh"), 0o755); err != nil {
		t.Fatal(err)
	}

	pipelines, warnings := devops.Discover(dir)
	if len(pipelines) != 1 {
		t.Errorf("expected 1 pipeline, got %d", len(pipelines))
	}
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(warnings))
	}
}
