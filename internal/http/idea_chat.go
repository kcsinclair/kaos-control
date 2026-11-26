// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/internal/config/defaults"
	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/ideachat"
	"github.com/kaos-control/kaos-control/internal/index"
	"github.com/kaos-control/kaos-control/internal/project"
	"github.com/kaos-control/kaos-control/internal/sandbox"
)

// handleIdeaConverse handles POST /api/p/:project/ideas/converse.
//
// Request:  { "session_id": string|null, "message": string }
// Response: { "session_id": string, "reply": string, "status": string,
//
//	"preview": {...}|null, "artifact_path": string|null }
//
// Special message values:
//   - "__accept__"  – accept the current proposal and write the artifact
//   - "__reject__"  – discard the session
func (s *Server) handleIdeaConverse(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	user := userFromCtx(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, apiError("unauthorized", "authentication required"))
		return
	}

	var req struct {
		SessionID string `json:"session_id"`
		Message   string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}
	if req.Message == "" {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "message is required"))
		return
	}

	store := p.IdeaChatStore

	// Resolve or create session.
	var sess *ideachat.Session
	if req.SessionID == "" {
		sess = store.Create(p.Entry.Name, user.Email)
	} else {
		var ok bool
		sess, ok = store.Get(req.SessionID)
		if !ok {
			writeJSON(w, http.StatusNotFound, apiError("session_not_found", "session not found or expired"))
			return
		}
	}
	store.Touch(sess.ID)

	// Handle special control messages.
	switch req.Message {
	case "__reject__":
		store.Delete(sess.ID)
		writeJSON(w, http.StatusOK, map[string]any{
			"session_id":    nil,
			"reply":         "Idea discarded.",
			"status":        ideachat.StatusConversing,
			"preview":       nil,
			"artifact_path": nil,
		})
		return

	case "__accept__":
		if sess.Status != ideachat.StatusProposed {
			writeJSON(w, http.StatusConflict, apiError("no_proposal", "no proposal to accept in this session"))
			return
		}
		actor := ""
		if u := userFromCtx(r.Context()); u != nil {
			actor = u.Email
		}
		relPath, err := writeIdeaArtifact(p, sess, actor)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, apiError("write_error", err.Error()))
			return
		}
		sess.Status = ideachat.StatusCreated
		store.Delete(sess.ID)
		writeJSON(w, http.StatusOK, map[string]any{
			"session_id":    sess.ID,
			"reply":         "Idea captured! Your idea has been saved.",
			"status":        ideachat.StatusCreated,
			"preview":       nil,
			"artifact_path": relPath,
		})
		return
	}

	// Regular conversation turn – look up the idea-capture agent config.
	modelCfg, err := resolveIdeaCaptureConfig(p, "idea-capture")
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, templateUnavailableError("idea-capture", err))
		return
	}

	// Gather project vocabulary.
	existingLabels, _ := p.Idx.Labels()
	existingSlugs, _ := collectSlugs(p)

	// Delegate to the conversation engine.
	resp, err := ideachat.Converse(r.Context(), sess, req.Message, existingLabels, existingSlugs, modelCfg)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("llm_error", err.Error()))
		return
	}

	// Build the HTTP response.
	var preview map[string]any
	if resp.Status == ideachat.StatusProposed && resp.ProposedFM != nil {
		preview = map[string]any{
			"frontmatter": resp.ProposedFM,
			"body":        resp.ProposedBody,
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"session_id":    sess.ID,
		"reply":         resp.Reply,
		"status":        resp.Status,
		"preview":       preview,
		"artifact_path": nil,
	})
}

// writeIdeaArtifact writes the proposed idea artifact to disk, updates the
// index, and broadcasts the artifact.indexed event.
// It returns the project-relative path of the written file.
func writeIdeaArtifact(p *project.Project, sess *ideachat.Session, actor string) (string, error) {
	slug := sess.ProposedSlug
	if slug == "" {
		return "", fmt.Errorf("session has no proposed slug")
	}

	relPath := "lifecycle/ideas/" + slug + ".md"

	absPath, err := sandbox.Resolve(p.Entry.Path, relPath)
	if err != nil {
		return "", fmt.Errorf("sandbox resolve: %w", err)
	}

	// Race-guard: refuse to overwrite an existing file.
	if _, err := os.Stat(absPath); err == nil {
		return "", fmt.Errorf("artifact already exists: %s", relPath)
	}

	fm := sess.ProposedFM
	fm.Type = "idea"
	fm.Status = "draft"
	fm.Lineage = slug
	fm.Created = time.Now().Format(time.RFC3339)

	content, err := buildMarkdown(fm, sess.ProposedBody)
	if err != nil {
		return "", fmt.Errorf("building markdown: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return "", fmt.Errorf("creating directory: %w", err)
	}
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("writing file: %w", err)
	}

	if err := p.Idx.IndexFile(absPath); err != nil {
		return "", fmt.Errorf("indexing file: %w", err)
	}

	p.Hub.Broadcast(hub.Event{
		Type:    "artifact.indexed",
		Payload: map[string]string{"path": relPath, "action": "created"},
	})

	// Record feed event and broadcast feed.new.
	artifactPath := relPath
	summary := fmt.Sprintf("Created idea %q", fm.Title)
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

	return relPath, nil
}

// resolveIdeaCaptureConfig finds the inline agent configuration in the project
// config that owns the given templateKey and returns a ModelConfig for it.
// Known keys: "idea-capture" (conversational), "idea-generate", "defect-generate",
// "doc-generate".
// "doc-generate" is owned by the docs-capture agent; all other keys are looked
// up in the idea-capture agent.
// When the resolved agent (or no agent at all) lacks the requested template
// key, a built-in default prompt is returned via defaultTemplateFor so the
// generation endpoints never hard-error on a missing template. An error is
// returned only for a templateKey with no built-in default.
func resolveIdeaCaptureConfig(p *project.Project, templateKey string) (ideachat.ModelConfig, error) {
	// Determine which agent owns this template key.
	agentName := "idea-capture"
	if templateKey == "doc-generate" {
		agentName = "docs-capture"
	}

	for _, a := range p.Config().Agents {
		if a.Name == agentName {
			model := a.Model
			if model == "" {
				model = "claude-sonnet-4-6"
			}
			provider := resolveAgentProvider(p, a.Provider)
			if prompt, ok := a.PromptTemplates[templateKey]; ok {
				return ideachat.ModelConfig{
					Model:        model,
					SystemPrompt: prompt,
					Provider:     provider,
				}, nil
			}
			if prompt, ok := defaultTemplateFor(templateKey); ok {
				return ideachat.ModelConfig{
					Model:        model,
					SystemPrompt: prompt,
					Provider:     provider,
				}, nil
			}
			return ideachat.ModelConfig{}, fmt.Errorf("%s agent has no template %q", agentName, templateKey)
		}
	}
	// No agent configured – fall back to the built-in default for the key.
	if prompt, ok := defaultTemplateFor(templateKey); ok {
		return ideachat.ModelConfig{
			Model:        "claude-sonnet-4-6",
			SystemPrompt: prompt,
		}, nil
	}
	return ideachat.ModelConfig{}, fmt.Errorf("%s agent not configured", agentName)
}

// resolveAgentProvider looks up providerName in the project's app-level
// provider snapshot and returns a pointer to it, or nil when providerName is
// empty or unregistered (CLI default). config.ValidateAgentProviders rejects
// an unregistered provider at project load, so an unregistered name here
// should not normally occur — resolving to nil (CLI default) rather than
// erroring keeps this a pure lookup.
func resolveAgentProvider(p *project.Project, providerName string) *config.ProviderConfig {
	if providerName == "" {
		return nil
	}
	for i := range p.Providers {
		if p.Providers[i].Name == providerName {
			prov := p.Providers[i]
			return &prov
		}
	}
	return nil
}

// templateUnavailableError builds the actionable "template_unavailable" API
// error body returned when resolveIdeaCaptureConfig cannot resolve a system
// prompt for templateKey. The underlying resolver error may name internal
// agent/template details, so it is logged rather than sent to the client —
// callers must never forward err.Error() in the HTTP response.
func templateUnavailableError(templateKey string, err error) map[string]any {
	slog.Warn("idea generation: no prompt template available", "template_key", templateKey, "err", err)
	return apiError("template_unavailable", fmt.Sprintf(
		"No prompt template is configured for %q generation. Ask a project admin to add it under lifecycle/config.yaml (agents[].prompt_templates), or check GET /config/health for repair details.",
		templateKey,
	))
}

// defaultTemplateFor returns the built-in default system prompt for a
// generation template key, or ("", false) for an unknown key. It is the sole
// consumer, on the HTTP side, of the canonical templates in
// internal/config/defaults — internal/config.Project.ValidateAndRepair reads
// from the same source so the two never drift.
func defaultTemplateFor(templateKey string) (string, bool) {
	prompt, ok := defaults.DefaultGenerationTemplates()[templateKey]
	return prompt, ok
}

// collectSlugs returns all lineage slugs currently in the project index.
func collectSlugs(p *project.Project) ([]string, error) {
	summaries, err := p.Idx.Lineages()
	if err != nil {
		return nil, err
	}
	slugs := make([]string, 0, len(summaries))
	for _, s := range summaries {
		slugs = append(slugs, s.Lineage)
	}
	return slugs, nil
}
