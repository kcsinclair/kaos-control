// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"strings"
	"testing"
)

// Milestone 2 – Conversation Flow Tests
//
// All tests in this file require ANTHROPIC_API_KEY because they drive live LLM
// conversation turns. They validate the multi-turn flow: clarification,
// proposal, and preview structure.

// TestIdeaChatVagueInputClarification verifies that a very short/vague message
// (e.g. "something cool") returns status "conversing" with a non-empty reply
// that looks like a question (ends with "?").
func TestIdeaChatVagueInputClarification(t *testing.T) {
	skipIfNoAPIKey(t)
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := converseAPI(env, "", "something cool")
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	status, _ := data["status"].(string)
	// The model may return "conversing" (clarifying question) or "proposed"
	// (if it interprets the message confidently). Either is valid; but when
	// "conversing", the reply should be a question.
	reply, _ := data["reply"].(string)
	if reply == "" {
		t.Error("expected non-empty reply")
	}

	if status == "conversing" {
		if !strings.HasSuffix(strings.TrimSpace(reply), "?") {
			t.Errorf("expected reply to be a question ending with '?', got: %q", reply)
		}
	}

	sessionID, _ := data["session_id"].(string)
	if sessionID == "" {
		t.Error("expected non-empty session_id")
	}
}

// TestIdeaChatDetailedInputProposal verifies that a sufficiently detailed
// message (50+ words describing a feature) returns status "proposed" with a
// non-null preview containing frontmatter and body.
func TestIdeaChatDetailedInputProposal(t *testing.T) {
	skipIfNoAPIKey(t)
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	detailedMessage := `I want to add a real-time notification system to the application. ` +
		`Users should receive browser notifications when an artifact they are tracking ` +
		`changes status. The notifications should include the artifact title, its new status, ` +
		`and a link to navigate directly to it. The system should use WebSocket to push ` +
		`events without polling. Users can opt out per artifact or globally in their settings.`

	sessionID, data := convergeToProposal(t, env, detailedMessage)
	if sessionID == "" {
		t.Fatal("missing session_id after reaching proposal")
	}
	_ = sessionID

	status, _ := data["status"].(string)
	if status != "proposed" {
		t.Errorf("expected status 'proposed', got %q", status)
	}

	preview, _ := data["preview"].(map[string]any)
	if preview == nil {
		t.Fatal("expected non-null preview when status is 'proposed'")
	}
	if _, ok := preview["frontmatter"]; !ok {
		t.Error("preview missing 'frontmatter' field")
	}
	if _, ok := preview["body"]; !ok {
		t.Error("preview missing 'body' field")
	}
}

// TestIdeaChatMaxClarifications verifies that after 3 rounds of clarification
// the agent stops asking and produces a proposal by the 4th response.
func TestIdeaChatMaxClarifications(t *testing.T) {
	skipIfNoAPIKey(t)
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// Deliberately vague messages to consume the 3-clarification budget.
	vagueMessages := []string{"something", "not sure", "anything works"}

	var sessionID string
	for i, msg := range vagueMessages {
		httpResp := converseAPI(env, sessionID, msg)
		requireStatus(t, httpResp, 200)
		data := readJSON(t, httpResp)

		sid, _ := data["session_id"].(string)
		if sid != "" {
			sessionID = sid
		}
		if sessionID == "" {
			t.Fatalf("turn %d: missing session_id", i+1)
		}

		status, _ := data["status"].(string)
		if status == "proposed" {
			// Model proposed early – remaining vague turns are moot; test passes.
			t.Logf("model proposed at turn %d (before exhausting clarifications)", i+1)
			return
		}
	}

	// 4th message – must now produce a proposal (forcing instruction kicks in).
	finalResp := converseAPI(env, sessionID, "just make something up")
	requireStatus(t, finalResp, 200)
	data := readJSON(t, finalResp)

	status, _ := data["status"].(string)
	if status != "proposed" {
		t.Errorf("after 4 vague messages expected status 'proposed', got %q", status)
	}
}

// TestIdeaChatProposalFrontmatterRequiredFields verifies that when status is
// "proposed", preview.frontmatter contains title (non-empty string),
// type: "idea", status: "draft", lineage (slug pattern), and labels (array).
func TestIdeaChatProposalFrontmatterRequiredFields(t *testing.T) {
	skipIfNoAPIKey(t)
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	_, data := convergeToProposal(t, env,
		`A feature to export project artifacts as a downloadable PDF report, ` +
		`including frontmatter metadata and the body content, ` +
		`so stakeholders can review the project state offline without needing access to the tool.`)

	preview, _ := data["preview"].(map[string]any)
	if preview == nil {
		t.Fatal("preview is nil – cannot check frontmatter fields")
	}
	fm, _ := preview["frontmatter"].(map[string]any)
	if fm == nil {
		t.Fatal("preview.frontmatter is nil")
	}

	// title: non-empty string
	title, _ := fm["title"].(string)
	if title == "" {
		t.Error("preview.frontmatter.title must be a non-empty string")
	}

	// type: "idea"
	typ, _ := fm["type"].(string)
	if typ != "idea" {
		t.Errorf("preview.frontmatter.type: want 'idea', got %q", typ)
	}

	// status: "draft"
	fmStatus, _ := fm["status"].(string)
	if fmStatus != "draft" {
		t.Errorf("preview.frontmatter.status: want 'draft', got %q", fmStatus)
	}

	// lineage: non-empty and matches slug pattern
	lineage, _ := fm["lineage"].(string)
	if lineage == "" {
		t.Error("preview.frontmatter.lineage must be a non-empty string")
	}
	if !slugPattern.MatchString(lineage) {
		t.Errorf("preview.frontmatter.lineage %q does not match slug pattern", lineage)
	}

	// labels: array (may be empty)
	rawLabels, exists := fm["labels"]
	if !exists {
		t.Error("preview.frontmatter.labels field is absent")
	}
	if rawLabels != nil {
		if _, ok := rawLabels.([]any); !ok {
			t.Errorf("preview.frontmatter.labels must be an array, got %T", rawLabels)
		}
	}
}

// TestIdeaChatProposalBodyValid verifies that preview.body starts with a
// level-1 heading ("# ") and contains at least one paragraph of text.
func TestIdeaChatProposalBodyValid(t *testing.T) {
	skipIfNoAPIKey(t)
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	_, data := convergeToProposal(t, env,
		`Add a keyboard shortcut system so users can navigate between artifacts, ` +
		`trigger transitions, and open the editor without using the mouse. ` +
		`Shortcuts should be discoverable via a help overlay and configurable per user.`)

	preview, _ := data["preview"].(map[string]any)
	if preview == nil {
		t.Fatal("preview is nil – cannot check body")
	}
	body, _ := preview["body"].(string)
	if body == "" {
		t.Fatal("preview.body is empty")
	}

	if !strings.HasPrefix(body, "# ") {
		t.Errorf("preview.body must start with '# ' (level-1 heading), got: %q", body[:min(len(body), 80)])
	}

	// At least one paragraph: body should have content beyond the heading line.
	lines := strings.Split(body, "\n")
	var hasContent bool
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "#") {
			hasContent = true
			break
		}
	}
	if !hasContent {
		t.Error("preview.body must contain at least one paragraph after the heading")
	}
}

// min returns the smaller of a and b. Replaces math.Min for int.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
