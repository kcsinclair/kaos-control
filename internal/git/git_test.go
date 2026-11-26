// SPDX-License-Identifier: AGPL-3.0-or-later

package git

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func newTestRepo(t *testing.T) (*Repo, *gogit.Repository, string) {
	t.Helper()
	dir := t.TempDir()
	gr, err := gogit.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("PlainInit: %v", err)
	}
	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return repo, gr, dir
}

func commitAt(t *testing.T, gr *gogit.Repository, dir, name string, when time.Time) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(name), 0o644); err != nil {
		t.Fatal(err)
	}
	wt, err := gr.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wt.Add(name); err != nil {
		t.Fatal(err)
	}
	sig := &object.Signature{Name: "Test", Email: "test@example.com", When: when}
	if _, err := wt.Commit("commit "+name, &gogit.CommitOptions{Author: sig}); err != nil {
		t.Fatal(err)
	}
}

// TestCommitsSince_NoCommitsAfterCutoff verifies a clean restart case
// (FR-7.2): when nothing has been committed since a job started, the
// interrupted run produced no partial commit.
func TestCommitsSince_NoCommitsAfterCutoff(t *testing.T) {
	repo, gr, dir := newTestRepo(t)
	base := time.Now().Add(-time.Hour)
	commitAt(t, gr, dir, "a.txt", base)

	cutoff := time.Now() // after the only commit
	commits, err := repo.CommitsSince(cutoff)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 0 {
		t.Fatalf("expected no commits since cutoff, got %d", len(commits))
	}
}

// TestCommitsSince_FindsCommitAfterCutoff verifies the partial-commit
// suspicion case (FR-7.1/7.3): a commit authored after a job's start time
// is evidence the run reached its commit step before failing.
func TestCommitsSince_FindsCommitAfterCutoff(t *testing.T) {
	repo, gr, dir := newTestRepo(t)
	cutoff := time.Now()
	commitAt(t, gr, dir, "a.txt", cutoff.Add(time.Minute))

	commits, err := repo.CommitsSince(cutoff)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 1 {
		t.Fatalf("expected 1 commit since cutoff, got %d", len(commits))
	}
	if commits[0].Message != "commit a.txt" {
		t.Errorf("unexpected commit message %q", commits[0].Message)
	}
}
