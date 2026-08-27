// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenAITools_DefaultTools(t *testing.T) {
	tools := DefaultOpenAITools()
	if len(tools) != 4 {
		t.Fatalf("expected 4 default tools, got %d", len(tools))
	}
	expected := map[string]bool{
		"read_file":  false,
		"write_file": false,
		"list_dir":   false,
		"grep":       false,
	}
	for _, tool := range tools {
		if tool.Type != "function" {
			t.Errorf("tool type: got %q, want function", tool.Type)
		}
		if _, ok := expected[tool.Function.Name]; !ok {
			t.Errorf("unexpected tool: %q", tool.Function.Name)
		}
		expected[tool.Function.Name] = true
	}
	for name, found := range expected {
		if !found {
			t.Errorf("expected tool %q not found", name)
		}
	}
}

func TestOpenAITools_ReadFile(t *testing.T) {
	root := t.TempDir()
	testFile := filepath.Join(root, "lifecycle", "ideas", "idea-1.md")
	if err := os.MkdirAll(filepath.Dir(testFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testFile, []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}

	exec := &ToolExecutor{ProjectRoot: root}
	ctx := context.Background()

	t.Run("read existing file", func(t *testing.T) {
		res, err := exec.Execute(ctx, "read_file", `{"path": "lifecycle/ideas/idea-1.md"}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res != "hello world" {
			t.Errorf("read result: got %q, want %q", res, "hello world")
		}
	})

	t.Run("read non-existent file", func(t *testing.T) {
		res, err := exec.Execute(ctx, "read_file", `{"path": "lifecycle/ideas/missing.md"}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(res, "file not found") {
			t.Errorf("expected file not found, got: %q", res)
		}
	})

	t.Run("path traversal rejected", func(t *testing.T) {
		res, err := exec.Execute(ctx, "read_file", `{"path": "../secret.txt"}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(res, "permission denied") {
			t.Errorf("expected permission denied for traversal, got: %q", res)
		}
	})
}

func TestOpenAITools_WriteFile(t *testing.T) {
	root := t.TempDir()
	exec := &ToolExecutor{
		ProjectRoot:  root,
		AllowedPaths: []string{"lifecycle/requirements", "lifecycle/ideas"},
	}
	ctx := context.Background()

	t.Run("write inside allowed_write_paths", func(t *testing.T) {
		res, err := exec.Execute(ctx, "write_file", `{"path": "lifecycle/requirements/req-1.md", "content": "# Requirements"}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(res, "file written successfully") {
			t.Errorf("write result: got %q", res)
		}
		data, err := os.ReadFile(filepath.Join(root, "lifecycle", "requirements", "req-1.md"))
		if err != nil {
			t.Fatalf("file not written to disk: %v", err)
		}
		if string(data) != "# Requirements" {
			t.Errorf("content: got %q, want %q", string(data), "# Requirements")
		}
	})

	t.Run("write outside allowed_write_paths refused without disk mutation", func(t *testing.T) {
		res, err := exec.Execute(ctx, "write_file", `{"path": "cmd/main.go", "content": "package main"}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(res, "outside allowed_write_paths") {
			t.Errorf("expected allowed_write_paths refusal, got: %q", res)
		}
		if _, err := os.Stat(filepath.Join(root, "cmd", "main.go")); !os.IsNotExist(err) {
			t.Error("file should not have been created on disk")
		}
	})

	t.Run("path traversal rejected without disk mutation", func(t *testing.T) {
		res, err := exec.Execute(ctx, "write_file", `{"path": "../../escape.txt", "content": "escape"}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(res, "permission denied") {
			t.Errorf("expected permission denied for traversal, got: %q", res)
		}
	})
}

func TestOpenAITools_ListDir(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "dir1", "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "dir1", "file.txt"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	exec := &ToolExecutor{ProjectRoot: root}
	ctx := context.Background()

	t.Run("list directory", func(t *testing.T) {
		res, err := exec.Execute(ctx, "list_dir", `{"path": "dir1"}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(res, "subdir/") || !strings.Contains(res, "file.txt") {
			t.Errorf("list_dir output unexpected: %q", res)
		}
	})

	t.Run("list empty directory", func(t *testing.T) {
		res, err := exec.Execute(ctx, "list_dir", `{"path": "dir1/subdir"}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res != "(empty directory)" {
			t.Errorf("list_dir empty: got %q", res)
		}
	})

	t.Run("list traversal rejected", func(t *testing.T) {
		res, err := exec.Execute(ctx, "list_dir", `{"path": "../"}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(res, "permission denied") {
			t.Errorf("expected permission denied for traversal, got: %q", res)
		}
	})
}

func TestOpenAITools_Grep(t *testing.T) {
	root := t.TempDir()
	f1 := filepath.Join(root, "doc1.txt")
	f2 := filepath.Join(root, "sub", "doc2.txt")
	if err := os.MkdirAll(filepath.Dir(f2), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f1, []byte("line 1\nFIND_ME target\nline 3"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f2, []byte("another line\nFIND_ME here too\nend"), 0o644); err != nil {
		t.Fatal(err)
	}

	exec := &ToolExecutor{ProjectRoot: root}
	ctx := context.Background()

	t.Run("grep matches across files", func(t *testing.T) {
		res, err := exec.Execute(ctx, "grep", `{"pattern": "FIND_ME", "path": "."}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(res, "doc1.txt:2:FIND_ME target") || !strings.Contains(res, "sub/doc2.txt:2:FIND_ME here too") {
			t.Errorf("grep result unexpected: %q", res)
		}
	})

	t.Run("grep no matches", func(t *testing.T) {
		res, err := exec.Execute(ctx, "grep", `{"pattern": "NOT_FOUND"}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res != "(no matches found)" {
			t.Errorf("grep result: got %q", res)
		}
	})

	t.Run("grep invalid regex", func(t *testing.T) {
		res, err := exec.Execute(ctx, "grep", `{"pattern": "[invalid"}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(res, "invalid regex") {
			t.Errorf("expected invalid regex error, got: %q", res)
		}
	})

	t.Run("grep traversal rejected", func(t *testing.T) {
		res, err := exec.Execute(ctx, "grep", `{"pattern": "foo", "path": "../"}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(res, "permission denied") {
			t.Errorf("expected permission denied for traversal, got: %q", res)
		}
	})
}

func TestOpenAITools_UnknownTool(t *testing.T) {
	// This used "bash" as its example of an unrecognised tool, which was true
	// until bash became a real (opt-in, policy-gated) tool. The behaviour under
	// test is unchanged — an unrecognised name is reported, not executed — so
	// only the example moves. Bash-specific refusal is covered directly by
	// TestBash_RefusedWhenPolicyNil in openai_bash_test.go.
	exec := &ToolExecutor{ProjectRoot: t.TempDir()}
	res, err := exec.Execute(context.Background(), "not_a_real_tool", `{"foo": "bar"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(res, "unknown tool") {
		t.Errorf("expected unknown tool, got: %q", res)
	}
}
