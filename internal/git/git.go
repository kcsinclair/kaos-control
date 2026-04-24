// Package git wraps go-git for kaos-control artifact commits and branch management.
package git

import (
	"errors"
	"fmt"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// ErrNotARepo is returned when the path is not a git repository.
var ErrNotARepo = errors.New("not a git repository")

// Repo wraps a go-git repository for kaos-control operations.
type Repo struct {
	r    *gogit.Repository
	root string
}

// CommitInfo is a summary of one git commit.
type CommitInfo struct {
	SHA     string    `json:"sha"`
	Message string    `json:"message"`
	Author  string    `json:"author"`
	When    time.Time `json:"when"`
}

// Open opens an existing git repository at path.
func Open(path string) (*Repo, error) {
	r, err := gogit.PlainOpen(path)
	if err != nil {
		if errors.Is(err, gogit.ErrRepositoryNotExists) {
			return nil, ErrNotARepo
		}
		return nil, fmt.Errorf("opening git repo at %s: %w", path, err)
	}
	return &Repo{r: r, root: path}, nil
}

// IsRepo returns true if path is a git repository.
func IsRepo(path string) bool {
	_, err := gogit.PlainOpen(path)
	return err == nil
}

// CurrentBranch returns the short name of the currently checked-out branch.
func (repo *Repo) CurrentBranch() (string, error) {
	ref, err := repo.r.Head()
	if err != nil {
		return "", fmt.Errorf("reading HEAD: %w", err)
	}
	return ref.Name().Short(), nil
}

// BranchExists returns true if the named branch ref exists.
func (repo *Repo) BranchExists(name string) bool {
	_, err := repo.r.Reference(plumbing.NewBranchReferenceName(name), false)
	return err == nil
}

// EnsureBranch creates a branch pointing at HEAD if it does not already exist.
// It does not switch to the branch (worktree is unaffected).
func (repo *Repo) EnsureBranch(name string) error {
	if repo.BranchExists(name) {
		return nil
	}
	head, err := repo.r.Head()
	if err != nil {
		return fmt.Errorf("reading HEAD to create branch: %w", err)
	}
	ref := plumbing.NewHashReference(plumbing.NewBranchReferenceName(name), head.Hash())
	if err := repo.r.Storer.SetReference(ref); err != nil {
		return fmt.Errorf("creating branch %q: %w", name, err)
	}
	return nil
}

// AddAndCommit stages the given project-relative paths and creates a commit.
// A Co-Authored-By trailer is appended to every commit message.
func (repo *Repo) AddAndCommit(relPaths []string, msg, authorName, authorEmail string) (string, error) {
	wt, err := repo.r.Worktree()
	if err != nil {
		return "", fmt.Errorf("getting worktree: %w", err)
	}

	for _, p := range relPaths {
		if _, err := wt.Add(p); err != nil {
			return "", fmt.Errorf("staging %s: %w", p, err)
		}
	}

	fullMsg := msg + "\n\nCo-Authored-By: kaos-control <noreply@kaos-control.local>"
	hash, err := wt.Commit(fullMsg, &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  authorName,
			Email: authorEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		return "", fmt.Errorf("committing: %w", err)
	}
	return hash.String(), nil
}

// Log returns up to limit commits that touched relPath (relative to repo root).
func (repo *Repo) Log(relPath string, limit int) ([]*CommitInfo, error) {
	iter, err := repo.r.Log(&gogit.LogOptions{
		FileName: &relPath,
		Order:    gogit.LogOrderCommitterTime,
	})
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}
	defer iter.Close()

	var out []*CommitInfo
	for i := 0; i < limit; i++ {
		c, err := iter.Next()
		if err != nil {
			break
		}
		// Trim to first line for the summary.
		summary := strings.SplitN(c.Message, "\n", 2)[0]
		out = append(out, &CommitInfo{
			SHA:     c.Hash.String(),
			Message: summary,
			Author:  c.Author.Name,
			When:    c.Author.When,
		})
	}
	return out, nil
}

// BranchNameFor evaluates a branch template with {slug} and {lineage} placeholders.
func BranchNameFor(template, slug, lineage string) string {
	r := strings.NewReplacer(
		"{slug}", slug,
		"{lineage}", lineage,
		"{type}", lineage,
		"{index}", "",
	)
	return strings.TrimRight(r.Replace(template), "-/")
}

// ModifiedFiles returns project-relative paths of files that are new or modified
// in the working tree but not yet committed. If allowedPaths is non-empty, only
// files whose path starts with one of those prefixes are included.
func (repo *Repo) ModifiedFiles(allowedPaths []string) ([]string, error) {
	wt, err := repo.r.Worktree()
	if err != nil {
		return nil, fmt.Errorf("getting worktree: %w", err)
	}
	status, err := wt.Status()
	if err != nil {
		return nil, fmt.Errorf("git status: %w", err)
	}
	var out []string
	for path, s := range status {
		if s.Worktree != gogit.Untracked && s.Worktree != gogit.Modified && s.Worktree != gogit.Added {
			continue
		}
		if len(allowedPaths) == 0 {
			out = append(out, path)
			continue
		}
		for _, ap := range allowedPaths {
			if strings.HasPrefix(path, ap) {
				out = append(out, path)
				break
			}
		}
	}
	return out, nil
}

// ResolveIdentity returns the author name/email to use for commits.
// Falls back to defaults when the git config has no user identity set.
func (repo *Repo) ResolveIdentity() (name, email string) {
	cfg, err := repo.r.Config()
	if err == nil && cfg.User.Name != "" {
		return cfg.User.Name, cfg.User.Email
	}
	return "kaos-control", "noreply@kaos-control.local"
}
