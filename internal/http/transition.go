package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kaos-control/kaos-control/internal/artifact"
	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/index"
	"github.com/kaos-control/kaos-control/internal/project"
	"github.com/kaos-control/kaos-control/internal/sandbox"
	"github.com/kaos-control/kaos-control/internal/workflow"
)

// handleAllowedTargets handles GET /api/p/:project/artifacts/*path/allowed-targets
func (s *Server) handleAllowedTargets(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	user := userFromCtx(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, apiError("unauthorized", "authentication required"))
		return
	}

	rawParam := chi.URLParam(r, "*")
	relPath := strings.TrimSuffix(rawParam, "/allowed-targets")

	row, err := p.Idx.Get(relPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	if row == nil {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "artifact not found"))
		return
	}

	userRoles := p.Cfg.RolesFor(user.Email)
	targets := p.Workflow.AllowedTargets(row.Status, userRoles, row.Type)
	writeJSON(w, http.StatusOK, map[string]any{"targets": targets})
}

// handleTransitionArtifact handles POST /api/p/:project/artifacts/*path/transition
func (s *Server) handleTransitionArtifact(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	user := userFromCtx(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, apiError("unauthorized", "authentication required"))
		return
	}

	rawParam := chi.URLParam(r, "*")
	relPath := strings.TrimSuffix(rawParam, "/transition")

	var req struct {
		To      string `json:"to"`
		Comment string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}
	if req.To == "" {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "field 'to' is required"))
		return
	}

	row, err := p.Idx.Get(relPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	if row == nil {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "artifact not found"))
		return
	}

	userRoles := p.Cfg.RolesFor(user.Email)
	if !p.Workflow.CanTransition(row.Status, req.To, userRoles, row.Type) {
		allowed := p.Workflow.AllowedTargets(row.Status, userRoles, row.Type)
		writeJSON(w, http.StatusForbidden, map[string]any{
			"error": map[string]any{
				"code":    "forbidden",
				"message": fmt.Sprintf("role(s) %v cannot transition %q → %q", userRoles, row.Status, req.To),
			},
			"allowed_targets": allowed,
		})
		return
	}

	// Required-plans gate: requirement leaving 'planning' must have all required plan
	// types approved. Product-owner bypasses the gate for maintenance / recovery.
	if !workflow.HasProductOwner(userRoles) && row.Status == "planning" && req.To == "in-development" {
		required := p.Cfg.RequiredPlans[row.Type]
		if ok, missing, err := workflow.GateReady(p.Idx, row.FM.Lineage, required); err != nil {
			writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
			return
		} else if !ok {
			writeJSON(w, http.StatusConflict, map[string]any{
				"error":   map[string]any{"code": "gate_not_ready", "message": "required plans are not yet approved"},
				"missing": missing,
			})
			return
		}
	}

	if err := applyTransition(p, row, relPath, req.To, user.Email, req.Comment); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("transition_error", err.Error()))
		return
	}

	// Rejection: write a child artifact with the reviewer's feedback.
	var rejectionPath string
	if req.To == "rejected" && req.Comment != "" {
		var rerr error
		rejectionPath, rerr = writeRejectionArtifact(p, row, relPath, req.Comment)
		if rerr != nil {
			// Non-fatal: transition is committed even if the child artifact fails.
			_ = rerr
		}
	}

	result, _ := p.Idx.Get(relPath)
	resp := map[string]any{"artifact": result}
	if rejectionPath != "" {
		resp["rejection_artifact"] = rejectionPath
	}
	writeJSON(w, http.StatusOK, resp)
}

// applyTransition writes the new status to disk, re-indexes, commits to git,
// broadcasts the artifact.indexed WebSocket event, and records a feed entry.
// It is the shared core used by both handleTransitionArtifact and the batch
// advance endpoint in status_check.go.
//
// row must reflect the artifact's state *before* the transition.
// actor is the user email (for feed events); comment is optional.
func applyTransition(p *project.Project, row *index.ArtifactRow, relPath, toStatus, actor, comment string) error {
	absPath, err := sandbox.Resolve(p.Entry.Path, relPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}
	patched, ok := patchFrontmatterField(raw, "status", toStatus)
	if !ok {
		return fmt.Errorf("status field not found in frontmatter of %s", relPath)
	}
	if err := os.WriteFile(absPath, patched, 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	_ = p.Idx.IndexFile(absPath)

	// Git commit.
	if p.Git != nil {
		authorName, authorEmail := p.Git.ResolveIdentity()
		msg := fmt.Sprintf("transition(%s): %s → %s", row.FM.Lineage, row.Status, toStatus)
		if comment != "" {
			msg += "\n\n" + comment
		}
		_, _ = p.Git.AddAndCommit([]string{relPath}, msg, authorName, authorEmail)
	}

	// WebSocket broadcast.
	p.Hub.Broadcast(hub.Event{
		Type: "artifact.indexed",
		Payload: map[string]any{
			"path": relPath, "action": "transitioned",
			"from": row.Status, "to": toStatus,
		},
	})

	// Feed event.
	artifactPath := relPath
	summary := fmt.Sprintf("%q transitioned from %s → %s", row.FM.Title, row.Status, toStatus)
	feedEvent := &index.EventRow{
		EventType:    "status_transition",
		Timestamp:    time.Now().Unix(),
		Actor:        actor,
		ArtifactPath: &artifactPath,
		Summary:      summary,
	}
	if err := p.Idx.InsertEvent(feedEvent); err == nil {
		p.Hub.Broadcast(hub.Event{Type: "feed.new", Payload: feedEvent})
	}

	return nil
}

// writeRejectionArtifact creates a <slug>-<N>-rejection.md child in the same
// stage directory as the original artifact, containing the reviewer's comment.
func writeRejectionArtifact(p *project.Project, row *index.ArtifactRow, relPath, comment string) (string, error) {
	nextIdx, err := p.Idx.NextIndexForLineage(row.FM.Lineage)
	if err != nil {
		return "", fmt.Errorf("next index: %w", err)
	}
	stageDir := filepath.Dir(relPath)
	filename := fmt.Sprintf("%s-%d-rejection.md", row.Slug, nextIdx)
	newRelPath := filepath.ToSlash(filepath.Join(stageDir, filename))
	newAbsPath := filepath.Join(p.Entry.Path, newRelPath)

	fm := artifact.Frontmatter{
		Title:   "Rejection: " + row.FM.Title,
		Type:    row.Type,
		Status:  "rejected",
		Lineage: row.FM.Lineage,
		Parent:  relPath,
	}
	content, err := buildMarkdown(fm, comment)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(newAbsPath, []byte(content), 0o644); err != nil {
		return "", err
	}
	_ = p.Idx.IndexFile(newAbsPath)
	return newRelPath, nil
}

func patchFrontmatterField(raw []byte, key, value string) ([]byte, bool) {
	return artifact.PatchFrontmatterField(raw, key, value)
}
