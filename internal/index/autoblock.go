package index

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/kaos-control/kaos-control/internal/artifact"
	"github.com/kaos-control/kaos-control/internal/hub"
)

// applyOpenQuestionTransition inspects a freshly indexed artifact and
// automatically transitions its status when the ## Open Questions section
// is present or absent:
//
//   - Has open questions AND status != "blocked"  → transition to "blocked"
//   - No open questions  AND status == "blocked"  → transition to "draft"
//   - Otherwise                                   → no-op
//
// Precondition: idx.hub and idx.wf must both be non-nil (callers must check).
func (idx *Index) applyOpenQuestionTransition(a *artifact.Artifact, absPath string) error {
	hasOQ := artifact.HasOpenQuestions(a.Body)

	switch {
	case hasOQ && a.FM.Status != "blocked":
		return idx.autoBlock(a, absPath)
	case !hasOQ && a.FM.Status == "blocked":
		return idx.autoUnblock(a, absPath)
	}
	return nil // idempotent no-op
}

func (idx *Index) autoBlock(a *artifact.Artifact, absPath string) error {
	if !idx.wf.CanTransition(a.FM.Status, "blocked", []string{"system"}, a.FM.Type) {
		slog.Warn("auto-transition: workflow rejected",
			"path", a.Path,
			"old_status", a.FM.Status,
			"new_status", "blocked",
			"reason", "transition_not_permitted",
		)
		return nil
	}

	oldStatus := a.FM.Status
	a.FM.Status = "blocked"

	// Ensure product-owner assignee is present.
	hasAssignee := false
	for _, as := range a.FM.Assignees {
		if as.Role == "product-owner" && as.Who == "agent" {
			hasAssignee = true
			break
		}
	}
	if !hasAssignee {
		a.FM.Assignees = append(a.FM.Assignees, artifact.Assignee{
			Role: "product-owner",
			Who:  "agent",
		})
	}

	if err := idx.writeAndReindex(a, absPath); err != nil {
		return err
	}

	artifactPath := a.Path
	payloadJSON := `{"reason":"open_questions_detected"}`
	event := &EventRow{
		EventType:    "status_changed",
		Timestamp:    time.Now().Unix(),
		Actor:        "system",
		ArtifactPath: &artifactPath,
		Summary:      fmt.Sprintf("Status changed from %s to blocked (open questions detected)", oldStatus),
		PayloadJSON:  &payloadJSON,
	}
	_ = idx.InsertEvent(event)

	idx.hub.Broadcast(hub.Event{
		Type: "artifact.indexed",
		Payload: map[string]any{
			"path":           a.Path,
			"action":         "transitioned",
			"from":           oldStatus,
			"to":             "blocked",
			"blocked_reason": "open_questions_detected",
		},
	})
	idx.hub.Broadcast(hub.Event{Type: "feed.new", Payload: event})

	slog.Info("auto-transition: open questions detected",
		"path", a.Path,
		"old_status", oldStatus,
		"new_status", "blocked",
		"reason", "open_questions_detected",
	)
	return nil
}

func (idx *Index) autoUnblock(a *artifact.Artifact, absPath string) error {
	if !idx.wf.CanTransition("blocked", "draft", []string{"system"}, a.FM.Type) {
		slog.Warn("auto-transition: workflow rejected",
			"path", a.Path,
			"old_status", "blocked",
			"new_status", "draft",
			"reason", "transition_not_permitted",
		)
		return nil
	}

	a.FM.Status = "draft"

	if err := idx.writeAndReindex(a, absPath); err != nil {
		return err
	}

	artifactPath := a.Path
	payloadJSON := `{"reason":"open_questions_resolved"}`
	event := &EventRow{
		EventType:    "status_changed",
		Timestamp:    time.Now().Unix(),
		Actor:        "system",
		ArtifactPath: &artifactPath,
		Summary:      "Status changed from blocked to draft (open questions resolved)",
		PayloadJSON:  &payloadJSON,
	}
	_ = idx.InsertEvent(event)

	idx.hub.Broadcast(hub.Event{
		Type: "artifact.indexed",
		Payload: map[string]any{
			"path":   a.Path,
			"action": "transitioned",
			"from":   "blocked",
			"to":     "draft",
		},
	})
	idx.hub.Broadcast(hub.Event{Type: "feed.new", Payload: event})

	slog.Info("auto-transition: open questions resolved",
		"path", a.Path,
		"old_status", "blocked",
		"new_status", "draft",
		"reason", "open_questions_resolved",
	)
	return nil
}

// writeAndReindex serialises the (mutated) artifact frontmatter + body to disk
// using an atomic write-then-rename, then re-parses and upserts the result.
func (idx *Index) writeAndReindex(a *artifact.Artifact, absPath string) error {
	content, err := marshalArtifact(a)
	if err != nil {
		return fmt.Errorf("autoblock: marshal: %w", err)
	}
	if err := atomicWrite(absPath, content); err != nil {
		return fmt.Errorf("autoblock: write: %w", err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("autoblock: stat after write: %w", err)
	}
	updated := artifact.Parse(content, a.Path, info.ModTime())
	if err := idx.Upsert(updated); err != nil {
		return fmt.Errorf("autoblock: upsert: %w", err)
	}
	return nil
}

// marshalArtifact serialises frontmatter + body into a complete markdown file,
// matching the format produced by internal/http.buildMarkdown.
func marshalArtifact(a *artifact.Artifact) ([]byte, error) {
	fmBytes, err := yaml.Marshal(a.FM)
	if err != nil {
		return nil, err
	}
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.Write(fmBytes)
	sb.WriteString("---\n")
	if a.Body != "" {
		sb.WriteString("\n")
		sb.WriteString(strings.TrimSpace(a.Body))
		sb.WriteString("\n")
	}
	return []byte(sb.String()), nil
}

// atomicWrite writes content to absPath via a temporary file then renames it,
// preventing partial-write corruption.
func atomicWrite(absPath string, content []byte) error {
	tmp := absPath + ".tmp"
	if err := os.WriteFile(tmp, content, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, absPath); err != nil {
		os.Remove(tmp) //nolint:errcheck
		return err
	}
	return nil
}
