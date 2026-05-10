// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kaos-control/kaos-control/internal/artifact"
	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/index"
	"github.com/kaos-control/kaos-control/internal/sandbox"
	"gopkg.in/yaml.v3"
)

// stageSuffix maps a lifecycle stage directory name to the optional filename suffix.
var stageSuffix = map[string]string{
	"backend-plans":  "be",
	"frontend-plans": "fe",
	"test-plans":     "test",
}

// handleCreateArtifact handles POST /api/p/:project/artifacts
func (s *Server) handleCreateArtifact(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	var req struct {
		Stage       string               `json:"stage"`
		Slug        string               `json:"slug"`
		Frontmatter artifact.Frontmatter `json:"frontmatter"`
		Body        string               `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}

	if req.Slug == "" || req.Stage == "" {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "stage and slug are required"))
		return
	}
	if !isValidSlug(req.Slug) {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "slug must be lowercase alphanumeric with hyphens"))
		return
	}

	// Resolve stage directory.
	stageDir := p.Cfg.StageDir(req.Stage)
	if stageDir == "" {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "unknown stage: "+req.Stage))
		return
	}

	// Determine next index for this lineage.
	lineage := req.Frontmatter.Lineage
	if lineage == "" {
		lineage = req.Slug
		req.Frontmatter.Lineage = lineage
	}
	nextIdx, err := p.Idx.NextIndexForLineage(lineage)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}

	// Build filename.
	filename := buildFilename(req.Slug, nextIdx, stageDir)
	relPath := "lifecycle/" + stageDir + "/" + filename

	// Sandbox check.
	absPath, err := sandbox.Resolve(p.Entry.Path, relPath)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid_path", err.Error()))
		return
	}

	// Check file doesn't already exist.
	if _, err := os.Stat(absPath); err == nil {
		writeJSON(w, http.StatusConflict, apiError("conflict", "artifact already exists at "+relPath))
		return
	}

	// Stamp created time; always set by the server and never overridden by the caller.
	req.Frontmatter.Created = time.Now().Format(time.RFC3339)

	// Marshal frontmatter to YAML.
	content, err := buildMarkdown(req.Frontmatter, req.Body)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("marshal_error", err.Error()))
		return
	}

	// Write the file.
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}

	// Update index.
	if err := p.Idx.IndexFile(absPath); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("index_error", err.Error()))
		return
	}

	// Git: ensure lineage branch exists, stage and commit.
	if p.Git != nil {
		branch := p.BranchForLineage(lineage, req.Slug)
		if err := p.Git.EnsureBranch(branch); err != nil {
			// Non-fatal: log and continue.
			_ = err
		}
		authorName, authorEmail := p.Git.ResolveIdentity()
		msg := fmt.Sprintf("create(%s): %s", req.Stage, relPath)
		if _, err := p.Git.AddAndCommit([]string{relPath}, msg, authorName, authorEmail); err != nil {
			// Non-fatal for now: index is updated but no commit.
			_ = err
		}
	}

	// Broadcast event.
	p.Hub.Broadcast(hub.Event{
		Type:    "artifact.indexed",
		Payload: map[string]string{"path": relPath, "action": "created"},
	})

	// Record feed event and broadcast feed.new.
	{
		actor := ""
		if u := userFromCtx(r.Context()); u != nil {
			actor = u.Email
		}
		artifactPath := relPath
		summary := fmt.Sprintf("Created %s %q", req.Frontmatter.Type, req.Frontmatter.Title)
		feedEvent := &index.EventRow{
			EventType:    "artifact_created",
			Timestamp:    time.Now().Unix(),
			Actor:        actor,
			ArtifactPath: &artifactPath,
			Summary:      summary,
		}
		if err := p.Idx.InsertEvent(feedEvent); err == nil {
			p.Hub.Broadcast(hub.Event{Type: "feed.new", Payload: feedEvent})
		}
	}

	row, _ := p.Idx.Get(relPath)
	writeJSON(w, http.StatusCreated, map[string]any{"artifact": row, "path": relPath})
}

// handleUpdateArtifact handles PUT /api/p/:project/artifacts/*path
func (s *Server) handleUpdateArtifact(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	relPath := chi.URLParam(r, "*")

	var req struct {
		Frontmatter artifact.Frontmatter `json:"frontmatter"`
		Body        string               `json:"body"`
		ExpectedSHA string               `json:"expected_sha"` // hex sha256 of current file
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}

	// Validate assignee roles against the project's configured role list.
	if len(req.Frontmatter.Assignees) > 0 {
		validRoles := make(map[string]bool, len(p.Cfg.Roles))
		for _, r := range p.Cfg.Roles {
			validRoles[r] = true
		}
		var invalid []string
		for _, a := range req.Frontmatter.Assignees {
			if a.Who == "" {
				writeJSON(w, http.StatusBadRequest, apiError("bad_request", "assignee who must not be empty"))
				return
			}
			if !validRoles[a.Role] {
				invalid = append(invalid, a.Role)
			}
		}
		if len(invalid) > 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": map[string]any{
					"code":          "invalid_role",
					"message":       "assignees contain unknown role(s): " + strings.Join(invalid, ", "),
					"invalid_roles": invalid,
				},
			})
			return
		}
	}

	absPath, err := sandbox.Resolve(p.Entry.Path, relPath)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid_path", err.Error()))
		return
	}

	// Always read the current file: needed for SHA check and to preserve created.
	current, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusNotFound, apiError("not_found", "artifact not found"))
		} else {
			writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		}
		return
	}

	// Optimistic concurrency: check SHA matches current file.
	if req.ExpectedSHA != "" {
		sum := sha256.Sum256(current)
		if hex.EncodeToString(sum[:]) != req.ExpectedSHA {
			writeJSON(w, http.StatusConflict, apiError("conflict", "artifact has been modified since last read"))
			return
		}
	}

	// Preserve the existing created value — it is immutable once set.
	// Parse the on-disk frontmatter to extract it; ignore any created value
	// the caller may have sent.
	{
		info, statErr := os.Stat(absPath)
		if statErr == nil {
			existing := artifact.Parse(current, relPath, info.ModTime())
			req.Frontmatter.Created = existing.FM.Created
		}
	}

	content, err := buildMarkdown(req.Frontmatter, req.Body)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("marshal_error", err.Error()))
		return
	}

	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}

	if err := p.Idx.IndexFile(absPath); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("index_error", err.Error()))
		return
	}

	if p.Git != nil {
		authorName, authorEmail := p.Git.ResolveIdentity()
		msg := fmt.Sprintf("update: %s", relPath)
		_, _ = p.Git.AddAndCommit([]string{relPath}, msg, authorName, authorEmail)
	}

	p.Hub.Broadcast(hub.Event{
		Type:    "artifact.indexed",
		Payload: map[string]string{"path": relPath, "action": "updated"},
	})

	row, _ := p.Idx.Get(relPath)
	writeJSON(w, http.StatusOK, map[string]any{"artifact": row})
}

// handleDeleteArtifact handles DELETE /api/p/:project/artifacts/*path
func (s *Server) handleDeleteArtifact(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	relPath := chi.URLParam(r, "*")

	absPath, err := sandbox.Resolve(p.Entry.Path, relPath)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid_path", err.Error()))
		return
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "artifact not found"))
		return
	}

	if err := os.Remove(absPath); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}

	if err := p.Idx.DeletePath(relPath); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("index_error", err.Error()))
		return
	}

	if p.Git != nil {
		authorName, authorEmail := p.Git.ResolveIdentity()
		msg := fmt.Sprintf("delete: %s", relPath)
		_, _ = p.Git.AddAndCommit([]string{relPath}, msg, authorName, authorEmail)
	}

	p.Hub.Broadcast(hub.Event{
		Type:    "artifact.indexed",
		Payload: map[string]string{"path": relPath, "action": "deleted"},
	})

	w.WriteHeader(http.StatusNoContent)
}

// handleRenameArtifact handles POST /api/p/:project/artifacts/*path/rename
func (s *Server) handleRenameArtifact(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	// Strip /rename suffix from the wildcard.
	rawParam := chi.URLParam(r, "*")
	oldRelPath := strings.TrimSuffix(rawParam, "/rename")

	var req struct {
		NewSlug string `json:"new_slug"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}
	if req.NewSlug == "" || !isValidSlug(req.NewSlug) {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "new_slug must be lowercase alphanumeric with hyphens"))
		return
	}

	oldAbsPath, err := sandbox.Resolve(p.Entry.Path, oldRelPath)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid_path", err.Error()))
		return
	}
	if _, err := os.Stat(oldAbsPath); os.IsNotExist(err) {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "artifact not found"))
		return
	}

	// Derive new path: same directory, same index/suffix, new slug.
	oldBase := filepath.Base(oldAbsPath)
	oldStem := strings.TrimSuffix(oldBase, ".md")
	_, idx, sfx := artifact.ParseFilename(oldStem)
	newFilename := buildFilenameFromParts(req.NewSlug, idx, sfx)
	newRelPath := filepath.ToSlash(filepath.Join(filepath.Dir(oldRelPath), newFilename))
	newAbsPath := filepath.Join(p.Entry.Path, newRelPath)

	if _, err := os.Stat(newAbsPath); err == nil {
		writeJSON(w, http.StatusConflict, apiError("conflict", "target path already exists: "+newRelPath))
		return
	}

	// Find all files that link to the old path.
	inbound, err := p.Idx.InboundLinks(oldRelPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}

	// Compute old slug for link rewriting.
	oldSlug, _, _ := artifact.ParseFilename(oldStem)

	// Rewrite inbound links in source files.
	changed := []string{}
	for _, srcRelPath := range inbound {
		srcAbs := filepath.Join(p.Entry.Path, srcRelPath)
		raw, err := os.ReadFile(srcAbs)
		if err != nil {
			continue
		}
		rewritten := rewriteLinks(raw, oldRelPath, newRelPath, oldSlug, req.NewSlug)
		if !bytes.Equal(raw, rewritten) {
			if err := os.WriteFile(srcAbs, rewritten, 0o644); err == nil {
				changed = append(changed, srcRelPath)
				_ = p.Idx.IndexFile(srcAbs)
			}
		}
	}

	// Rename the artifact file itself.
	if err := os.Rename(oldAbsPath, newAbsPath); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}
	_ = p.Idx.DeletePath(oldRelPath)
	_ = p.Idx.IndexFile(newAbsPath)
	changed = append(changed, oldRelPath, newRelPath)

	// Single atomic commit covering all changed files.
	if p.Git != nil {
		authorName, authorEmail := p.Git.ResolveIdentity()
		msg := fmt.Sprintf("rename: %s → %s", oldSlug, req.NewSlug)
		_, _ = p.Git.AddAndCommit(changed, msg, authorName, authorEmail)
	}

	p.Hub.Broadcast(hub.Event{
		Type:    "artifact.indexed",
		Payload: map[string]any{"old_path": oldRelPath, "new_path": newRelPath, "action": "renamed"},
	})

	row, _ := p.Idx.Get(newRelPath)
	writeJSON(w, http.StatusOK, map[string]any{"artifact": row, "path": newRelPath})
}

// handlePatchPriority handles PATCH /api/p/:project/artifacts/*path/priority
// It updates only the priority field in the artifact's YAML frontmatter.
func (s *Server) handlePatchPriority(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	// Priority is a meaningful workflow signal; require authentication.
	user := userFromCtx(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, apiError("unauthorized", "authentication required"))
		return
	}

	rawParam := chi.URLParam(r, "*")
	relPath := strings.TrimSuffix(rawParam, "/priority")

	var req struct {
		Priority string `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}

	absPath, err := sandbox.Resolve(p.Entry.Path, relPath)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid_path", err.Error()))
		return
	}

	raw, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusNotFound, apiError("not_found", "artifact not found"))
		} else {
			writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		}
		return
	}

	info, err := os.Stat(absPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}

	a := artifact.Parse(raw, relPath, info.ModTime())

	// Lineage lock check: if another user holds the lock, reject with 423 Locked.
	if p.Locks != nil && a.FM.Lineage != "" {
		if lockRow, lerr := p.Locks.Get(a.FM.Lineage); lerr == nil && lockRow != nil && lockRow.Holder != user.Email {
			writeJSON(w, http.StatusLocked, map[string]any{
				"error": map[string]any{
					"code":    "locked",
					"message": "artifact lineage is locked by another user",
				},
				"lock": lockRow,
			})
			return
		}
	}

	a.FM.Priority = req.Priority

	content, err := buildMarkdown(a.FM, a.Body)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("marshal_error", err.Error()))
		return
	}

	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}

	if err := p.Idx.IndexFile(absPath); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("index_error", err.Error()))
		return
	}

	p.Hub.Broadcast(hub.Event{
		Type:    "artifact.indexed",
		Payload: map[string]string{"path": relPath, "action": "updated"},
	})

	row, _ := p.Idx.Get(relPath)
	writeJSON(w, http.StatusOK, map[string]any{"artifact": row})
}

// ----- helpers -----

var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9\-]*[a-z0-9]$|^[a-z0-9]$`)

func isValidSlug(s string) bool {
	return slugRe.MatchString(s)
}

// buildFilename constructs the filename for a new artifact.
func buildFilename(slug string, idx int, stageDir string) string {
	sfx := stageSuffix[stageDir]
	return buildFilenameFromParts(slug, idx, sfx)
}

// buildFilenameFromParts reconstructs a filename from its components.
func buildFilenameFromParts(slug string, idx int, sfx string) string {
	if idx == 0 {
		return slug + ".md"
	}
	if sfx != "" {
		return fmt.Sprintf("%s-%d-%s.md", slug, idx, sfx)
	}
	return fmt.Sprintf("%s-%d.md", slug, idx)
}

// buildMarkdown serialises frontmatter + body into a complete markdown file.
func buildMarkdown(fm artifact.Frontmatter, body string) (string, error) {
	fmBytes, err := yaml.Marshal(fm)
	if err != nil {
		return "", fmt.Errorf("marshalling frontmatter: %w", err)
	}
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.Write(fmBytes)
	sb.WriteString("---\n")
	if body != "" {
		sb.WriteString("\n")
		sb.WriteString(strings.TrimSpace(body))
		sb.WriteString("\n")
	}
	return sb.String(), nil
}

// rewriteLinks replaces references to oldRelPath/oldSlug with newRelPath/newSlug
// throughout the raw file content (frontmatter + body).
func rewriteLinks(raw []byte, oldRelPath, newRelPath, oldSlug, newSlug string) []byte {
	s := string(raw)

	// Replace exact full relative paths (with and without .md extension).
	oldNoExt := strings.TrimSuffix(oldRelPath, ".md")
	newNoExt := strings.TrimSuffix(newRelPath, ".md")
	s = strings.ReplaceAll(s, oldRelPath, newRelPath)
	s = strings.ReplaceAll(s, oldNoExt, newNoExt)

	// Replace wiki-link style slug references: [[old-slug ...]] → [[new-slug ...]]
	oldBase := filepath.Base(oldNoExt)
	newBase := filepath.Base(newNoExt)
	if oldBase != newBase {
		s = strings.ReplaceAll(s, "[["+oldBase, "[["+newBase)
		s = strings.ReplaceAll(s, oldBase+"|", newBase+"|")
		s = strings.ReplaceAll(s, oldBase+"]]", newBase+"]]")
	}

	_ = oldSlug // slug is derivable from the path bases above
	_ = newSlug
	return []byte(s)
}
