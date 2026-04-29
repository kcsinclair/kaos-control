package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

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
//             "preview": {...}|null, "artifact_path": string|null }
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
		writeJSON(w, http.StatusInternalServerError, apiError("config_error", err.Error()))
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

	// Record feed event.
	artifactPath := relPath
	summary := fmt.Sprintf("Created idea %q", fm.Title)
	_ = p.Idx.InsertEvent(&index.EventRow{
		EventType:    "artifact_created",
		Timestamp:    time.Now().Unix(),
		Actor:        actor,
		ArtifactPath: &artifactPath,
		Summary:      summary,
	})

	return relPath, nil
}

// resolveIdeaCaptureConfig finds the idea-capture agent configuration in the
// project config and returns a ModelConfig for the given templateKey.
// Known keys: "idea-capture" (conversational), "idea-generate", "defect-generate".
// When templateKey is "idea-capture" and no agent is configured, a built-in
// default prompt is returned so the conversational endpoint keeps working.
func resolveIdeaCaptureConfig(p *project.Project, templateKey string) (ideachat.ModelConfig, error) {
	for _, a := range p.Cfg.Agents {
		if a.Name == "idea-capture" {
			prompt, ok := a.PromptTemplates[templateKey]
			if !ok {
				return ideachat.ModelConfig{}, fmt.Errorf("idea-capture agent has no template %q", templateKey)
			}
			model := a.Model
			if model == "" {
				model = "claude-sonnet-4-6"
			}
			return ideachat.ModelConfig{
				Model:        model,
				SystemPrompt: prompt,
			}, nil
		}
	}
	// No agent configured – fall back to the built-in default for the conversational key only.
	if templateKey == "idea-capture" {
		return ideachat.ModelConfig{
			Model:        "claude-sonnet-4-6",
			SystemPrompt: defaultIdeaCapturePrompt,
		}, nil
	}
	return ideachat.ModelConfig{}, fmt.Errorf("idea-capture agent not configured")
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

// defaultIdeaCapturePrompt is the fallback system prompt used when no
// idea-capture agent is configured in lifecycle/config.yaml.
const defaultIdeaCapturePrompt = `You are an idea-capture assistant for a software project lifecycle tool.
Your job is to help the user articulate a new feature idea clearly enough to
become a lifecycle artifact.

RULES:
1. If the user's input is vague, ask ONE short clarifying question (max 3 questions total).
2. Once you have enough context, produce a proposal as structured JSON.
3. Pick labels ONLY from the provided label vocabulary.
4. The slug must match: ^[a-z0-9][a-z0-9\-]*[a-z0-9]$|^[a-z0-9]$

ALWAYS respond with a JSON object in a ` + "```" + `json code block:

For a clarifying question:
` + "```" + `json
{"action":"clarify","reply":"<your single clarifying question>","slug":"","title":"","labels":[],"body":""}
` + "```" + `

For a proposal:
` + "```" + `json
{"action":"propose","reply":"<short confirmation message>","slug":"<slug>","title":"<title>","labels":["<label>"],"body":"# <title>\n\n<1-3 paragraphs describing the idea>"}
` + "```" + `
`
