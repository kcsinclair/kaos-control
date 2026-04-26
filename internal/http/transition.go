package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/kaos-control/kaos-control/internal/artifact"
	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/index"
	"github.com/kaos-control/kaos-control/internal/project"
	"github.com/kaos-control/kaos-control/internal/sandbox"
	"github.com/kaos-control/kaos-control/internal/workflow"
)

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
	if !p.Workflow.CanTransition(row.Status, req.To, userRoles) {
		allowed := p.Workflow.AllowedTargets(row.Status, userRoles)
		writeJSON(w, http.StatusForbidden, map[string]any{
			"error": map[string]any{
				"code":    "forbidden",
				"message": fmt.Sprintf("role(s) %v cannot transition %q → %q", userRoles, row.Status, req.To),
			},
			"allowed_targets": allowed,
		})
		return
	}

	// Required-plans gate: requirement leaving 'planning' must have all required plan types approved.
	if row.Status == "planning" && req.To == "in-development" {
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

	// Patch status in the file (frontmatter only; body is untouched).
	absPath, err := sandbox.Resolve(p.Entry.Path, relPath)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid_path", err.Error()))
		return
	}
	raw, err := os.ReadFile(absPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}
	patched, ok := patchFrontmatterField(raw, "status", req.To)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, apiError("patch_error", "status field not found in frontmatter"))
		return
	}
	if err := os.WriteFile(absPath, patched, 0o644); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}
	_ = p.Idx.IndexFile(absPath)

	changedPaths := []string{relPath}

	// Rejection: write a child artifact with the reviewer's feedback.
	var rejectionPath string
	if req.To == "rejected" && req.Comment != "" {
		rejectionPath, err = writeRejectionArtifact(p, row, relPath, req.Comment)
		if err != nil {
			// Non-fatal: transition is committed even if the child artifact fails.
			_ = err
		} else {
			changedPaths = append(changedPaths, rejectionPath)
		}
	}

	// Git commit covering the status change (and rejection child if created).
	if p.Git != nil {
		authorName, authorEmail := p.Git.ResolveIdentity()
		msg := fmt.Sprintf("transition(%s): %s → %s", row.FM.Lineage, row.Status, req.To)
		if req.Comment != "" {
			msg += "\n\n" + req.Comment
		}
		_, _ = p.Git.AddAndCommit(changedPaths, msg, authorName, authorEmail)
	}

	p.Hub.Broadcast(hub.Event{
		Type: "artifact.indexed",
		Payload: map[string]any{
			"path": relPath, "action": "transitioned",
			"from": row.Status, "to": req.To,
		},
	})

	result, _ := p.Idx.Get(relPath)
	resp := map[string]any{"artifact": result}
	if rejectionPath != "" {
		resp["rejection_artifact"] = rejectionPath
	}
	writeJSON(w, http.StatusOK, resp)
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

// patchFrontmatterField replaces the value of key within the YAML frontmatter.
// Edits only between the opening and closing --- fences; body is unchanged.
// Returns (patched, true) on success or (raw, false) if the key is not found.
func patchFrontmatterField(raw []byte, key, value string) ([]byte, bool) {
	s := string(raw)
	if !strings.HasPrefix(s, "---") {
		return raw, false
	}
	closeIdx := strings.Index(s[3:], "\n---")
	if closeIdx < 0 {
		return raw, false
	}
	fmEnd := 3 + closeIdx // index of '\n' before the closing fence
	fmSection := s[3:fmEnd]

	lineRe := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(key) + `:\s*.*$`)
	replaced := lineRe.ReplaceAllLiteralString(fmSection, key+": "+value)
	if replaced == fmSection {
		return raw, false
	}
	return []byte("---" + replaced + s[fmEnd:]), true
}
