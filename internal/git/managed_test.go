// SPDX-License-Identifier: AGPL-3.0-or-later

package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	gogit "github.com/go-git/go-git/v5"
)

func initTestRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	repo, err := gogit.PlainInit(root, false)
	if err != nil {
		t.Fatal(err)
	}
	cfg, _ := repo.Config()
	cfg.User.Name = "Test"
	cfg.User.Email = "test@test.local"
	if err := repo.SetConfig(cfg); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestTrackGenerated_NotARepo_NoOp(t *testing.T) {
	committed, cmds, err := TrackGenerated(t.TempDir(), []string{"AGENTS.md"}, "msg")
	if err != nil || committed || cmds != nil {
		t.Fatalf("expected a no-op for a non-repo, got committed=%v cmds=%v err=%v", committed, cmds, err)
	}
}

func TestTrackGenerated_UnmanagedRepo_ReturnsCommands(t *testing.T) {
	root := initTestRepo(t)
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	committed, cmds, err := TrackGenerated(root, []string{"AGENTS.md"}, "kaos-control: directives")
	if err != nil {
		t.Fatal(err)
	}
	if committed {
		t.Error("expected committed=false for a pre-existing (unmanaged) repo")
	}
	if len(cmds) != 2 {
		t.Fatalf("expected add + commit commands, got %v", cmds)
	}
	if !strings.Contains(cmds[0], " add") || !strings.Contains(cmds[0], "AGENTS.md") {
		t.Errorf("add command wrong: %q", cmds[0])
	}
	if !strings.Contains(cmds[1], "commit") {
		t.Errorf("second command is not a commit: %q", cmds[1])
	}
}

func TestTrackGenerated_ManagedRepo_AutoCommits(t *testing.T) {
	root := initTestRepo(t)
	if err := MarkManaged(root); err != nil {
		t.Fatal(err)
	}
	if !IsManaged(root) {
		t.Fatal("expected IsManaged=true after MarkManaged")
	}
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	committed, cmds, err := TrackGenerated(root, []string{"AGENTS.md"}, "kaos-control: directives")
	if err != nil {
		t.Fatal(err)
	}
	if !committed || cmds != nil {
		t.Fatalf("expected auto-commit for a managed repo, got committed=%v cmds=%v", committed, cmds)
	}

	// The generated file is now committed — the worktree is clean (the
	// .git/ managed marker lives outside the worktree and never shows).
	repo, err := gogit.PlainOpen(root)
	if err != nil {
		t.Fatal(err)
	}
	wt, _ := repo.Worktree()
	st, _ := wt.Status()
	if !st.IsClean() {
		t.Errorf("expected a clean worktree after auto-commit, got %v", st)
	}
}
