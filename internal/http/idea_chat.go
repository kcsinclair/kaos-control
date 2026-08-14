// SPDX-License-Identifier: AGPL-3.0-or-later

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
			if prompt, ok := a.PromptTemplates[templateKey]; ok {
				return ideachat.ModelConfig{
					Model:        model,
					SystemPrompt: prompt,
				}, nil
			}
			if prompt, ok := defaultTemplateFor(templateKey); ok {
				return ideachat.ModelConfig{
					Model:        model,
					SystemPrompt: prompt,
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

// defaultTemplateFor returns the built-in default system prompt for a
// generation template key, or ("", false) for an unknown key.
func defaultTemplateFor(templateKey string) (string, bool) {
	switch templateKey {
	case "idea-capture":
		return defaultIdeaCapturePrompt, true
	case "idea-generate":
		return defaultIdeaGeneratePrompt, true
	case "defect-generate":
		return defaultDefectGeneratePrompt, true
	case "doc-generate":
		return defaultDocGeneratePrompt, true
	default:
		return "", false
	}
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

// defaultIdeaGeneratePrompt is the fallback system prompt used for single-shot
// idea generation when no idea-capture agent (or no "idea-generate" template)
// is configured. Mirrors the idea-generate template shipped in
// internal/initcmd/templates/config.yaml.tmpl and lifecycle/config.yaml.
const defaultIdeaGeneratePrompt = `You are a single-shot idea-capture assistant for a software project lifecycle tool.
The user will send you a raw brain-dump describing a feature idea.
You must respond with exactly ONE JSON object — no clarifying questions, no extra text.

RULES:
1. Always produce action "propose". NEVER produce action "clarify".
2. Pick labels ONLY from the provided label vocabulary (if supplied).
   Do not invent new labels.
3. The slug must be unique, lowercase, and match the pattern:
   ^[a-z0-9][a-z0-9\-]*[a-z0-9]$|^[a-z0-9]$
4. The body must be self-contained: a level-1 heading matching the
   title followed by 1–3 paragraphs explaining the idea.
5. Set priority to "high" only if the input explicitly signals urgency
   (e.g. "urgent", "blocking", "critical", "ASAP"); otherwise use "normal".

ALWAYS respond with a single JSON object inside a ` + "```" + `json code block.
Never include any text outside the JSON code block.

` + "```" + `json
{"action":"propose","reply":"<short confirmation message>","slug":"<slug>","title":"<Idea Title>","labels":["<label>"],"body":"# <Idea Title>\n\n<paragraph 1>\n\n<paragraph 2>"}
` + "```" + `
`

// defaultDefectGeneratePrompt is the fallback system prompt used for
// single-shot defect generation when no idea-capture agent (or no
// "defect-generate" template) is configured. Mirrors the defect-generate
// template shipped in lifecycle/config.yaml (see Milestone 3 for adding it to
// the init template).
const defaultDefectGeneratePrompt = `You are a single-shot defect-capture assistant for a software project lifecycle tool.
The user will send you a raw brain-dump describing a bug or defect.
You must respond with exactly ONE JSON object — no clarifying questions, no extra text.

RULES:
1. Always produce action "propose". NEVER produce action "clarify".
2. Always include "defect" in the labels array.
3. Pick additional labels ONLY from the provided label vocabulary (if supplied).
   Do not invent new labels.
4. The slug must be unique, lowercase, and match the pattern:
   ^[a-z0-9][a-z0-9\-]*[a-z0-9]$|^[a-z0-9]$
5. The body must contain exactly these sections (use ## headings):
   ## Reproduction Steps
   ## Expected Behaviour
   ## Actual Behaviour
   Fill each section from the user's description. Infer reasonable
   content where the user has not been explicit.
6. Set priority to "high" only if the input explicitly signals urgency
   (e.g. "urgent", "blocking", "critical", "ASAP"); otherwise use "normal".

ALWAYS respond with a single JSON object inside a ` + "```" + `json code block.
Never include any text outside the JSON code block.

` + "```" + `json
{"action":"propose","reply":"<short confirmation message>","slug":"<slug>","title":"<Defect Title>","labels":["defect"],"body":"# <Defect Title>\n\n## Reproduction Steps\n\n<steps>\n\n## Expected Behaviour\n\n<expected>\n\n## Actual Behaviour\n\n<actual>"}
` + "```" + `
`

// defaultDocGeneratePrompt is the fallback system prompt used for single-shot
// documentation-brief generation when no docs-capture agent (or no
// "doc-generate" template) is configured. Mirrors the doc-generate template
// shipped in internal/initcmd/templates/config.yaml.tmpl and
// lifecycle/config.yaml.
const defaultDocGeneratePrompt = `You are a single-shot documentation-brief assistant for a software project lifecycle tool.
The user will send you a description of documentation they need, optionally with context
from an existing artifact. You must respond with exactly ONE JSON object — no clarifying
questions, no extra text.

RULES:
1. Always produce action "propose". NEVER produce action "clarify".
2. Pick labels ONLY from the provided label vocabulary (if supplied).
   Do not invent new labels.
3. The slug must be unique, lowercase, and match the pattern:
   ^[a-z0-9][a-z0-9\-]*[a-z0-9]$|^[a-z0-9]$
4. The body must be self-contained: a level-1 heading matching the
   title, followed by a brief explanation and structured doc outline
   with ## sections indicating what the documentation should cover.
5. Set priority to "high" only if the input explicitly signals urgency
   (e.g. "urgent", "blocking", "critical", "ASAP"); otherwise use "normal".
6. If source artifact context is provided, use it to inform the doc outline.

ALWAYS respond with a single JSON object inside a ` + "```" + `json code block.
Never include any text outside the JSON code block.

` + "```" + `json
{"action":"propose","reply":"<short confirmation message>","slug":"<slug>","title":"<Doc Title>","labels":["<label>"],"body":"# <Doc Title>\n\n<brief description>\n\n## Overview\n\n<outline section>\n\n## Details\n\n<outline section>"}
` + "```" + `
`
