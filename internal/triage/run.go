// SPDX-License-Identifier: AGPL-3.0-or-later

package triage

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/kaos-control/kaos-control/internal/artifact"
	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/ideachat"
	"github.com/kaos-control/kaos-control/internal/index"
	"github.com/kaos-control/kaos-control/internal/sandbox"
)

// execute is the inner triage job started by Trigger.
// It reads the artifact, calls the LLM, rewrites the file, transitions the
// status to draft, and records the run. On any error it rolls the file back
// to its pre-call state.
func (m *Manager) execute(ctx context.Context, runID, relPath, lineage string, trigger TriggerSource) (retErr error) {
	startedAt := time.Now()

	// Record the run start.
	m.recordRunStart(relPath, runID, lineage, string(trigger), startedAt)

	// Deferred: record completion, emit log line. Named return (retErr) carries
	// the final error so the defer can inspect it after all return paths.
	defer func() {
		durationMs := time.Since(startedAt).Milliseconds()
		if retErr == nil {
			slog.Info("triage completed",
				"path", relPath,
				"lineage", lineage,
				"run_id", runID,
				"duration_ms", durationMs,
			)
			m.recordRunComplete(runID, "done", durationMs, "")
		} else {
			slog.Warn("triage failed",
				"path", relPath,
				"lineage", lineage,
				"run_id", runID,
				"reason", retErr.Error(),
			)
			m.recordRunComplete(runID, "failed", durationMs, retErr.Error())
		}
	}()

	// Resolve the absolute path and validate it stays inside lifecycle/ideas/.
	absPath, err := sandbox.Resolve(m.deps.ProjectRoot, relPath)
	if err != nil {
		return fmt.Errorf("sandbox resolve: %w", err)
	}
	ideasAbsDir := filepath.Join(m.deps.ProjectRoot, "lifecycle", "ideas")
	if !strings.HasPrefix(absPath, ideasAbsDir+string(filepath.Separator)) {
		return fmt.Errorf("path %q is outside lifecycle/ideas/", relPath)
	}

	// Read original bytes (for rollback).
	originalBytes, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("read artifact: %w", err)
	}

	// Parse to extract body and frontmatter.
	fi, _ := os.Stat(absPath)
	var mtime time.Time
	if fi != nil {
		mtime = fi.ModTime()
	}
	a := artifact.Parse(originalBytes, relPath, mtime)
	rawBody := a.Body

	// Resolve the idea-triage agent config to get model + system prompt.
	modelCfg, err := m.resolveAgentConfig()
	if err != nil {
		return fmt.Errorf("agent config: %w", err)
	}

	// Gather label vocabulary and slug list for the LLM.
	existingLabels, _ := m.deps.Idx.Labels()
	diskSlugs, _ := ideachat.CollectDiskSlugs(m.deps.ProjectRoot, "lifecycle/ideas")

	// Call the LLM (single-shot, no conversation).
	result, err := ideachat.Generate(ctx, ideachat.GenerateOptions{
		Input:          rawBody,
		ArtifactType:   "idea",
		ExistingLabels: existingLabels,
		ExistingSlugs:  diskSlugs,
		ModelCfg:       modelCfg,
	})
	if err != nil {
		return fmt.Errorf("ideachat.Generate: %w", err)
	}
	if result.Body == "" {
		return fmt.Errorf("LLM returned empty body")
	}

	// Rewrite the artifact body (preserve H1, wrap original in ## Raw Idea,
	// add ## Idea with the agent-generated content).
	newBody := rewriteBody(rawBody, result.Body)

	// Merge labels and ensure priority.
	mergedLabels := mergeAndFilterLabels(a.FM.Labels, result.Labels, existingLabels)
	newFM := a.FM
	newFM.Labels = mergedLabels
	if newFM.Priority == "" {
		newFM.Priority = "normal"
	}

	// Serialise to bytes.
	newBytes, err := marshalArtifact(newFM, newBody)
	if err != nil {
		return fmt.Errorf("marshal artifact: %w", err)
	}

	// Write to disk.
	if err := os.WriteFile(absPath, newBytes, 0o644); err != nil {
		return fmt.Errorf("write artifact: %w", err)
	}

	// Apply the raw → draft workflow transition, re-index, and commit.
	if err := m.applyDraftTransition(ctx, absPath, relPath, lineage, originalBytes); err != nil {
		// Roll back to pre-call state on transition failure.
		_ = os.WriteFile(absPath, originalBytes, 0o644)
		_ = m.deps.Idx.IndexFile(absPath)
		return err
	}

	return nil
}

// applyDraftTransition validates and applies the raw → draft status change.
// On success it re-indexes the artifact, commits it, and broadcasts the event.
func (m *Manager) applyDraftTransition(_ context.Context, absPath, relPath, lineage string, originalBytes []byte) error {
	if !m.deps.Workflow.CanTransition("raw", "draft", []string{"product-owner"}, "idea") {
		return fmt.Errorf("workflow_denied: raw → draft not permitted by engine")
	}

	// Re-read from disk (we just wrote it), then patch status field.
	current, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("re-read for transition: %w", err)
	}

	// Verify the on-disk status is still "raw" (guard against concurrent writes).
	a := artifact.Parse(current, relPath, time.Time{})
	if a.FM.Status != "raw" {
		// Already transitioned by another actor; treat as success.
		return nil
	}

	patched, ok := artifact.PatchFrontmatterField(current, "status", "draft")
	if !ok {
		// "status" field missing — file was mutated unexpectedly; roll back.
		_ = os.WriteFile(absPath, originalBytes, 0o644)
		return fmt.Errorf("status field not found in frontmatter")
	}

	if err := os.WriteFile(absPath, patched, 0o644); err != nil {
		_ = os.WriteFile(absPath, originalBytes, 0o644)
		return fmt.Errorf("write patched status: %w", err)
	}

	// Re-index so the UI sees the new status and body immediately.
	_ = m.deps.Idx.IndexFile(absPath)

	// Git commit (best-effort; non-fatal if git is unavailable).
	if m.deps.Git != nil {
		authorName, authorEmail := m.deps.Git.ResolveIdentity()
		msg := fmt.Sprintf("triage(%s): raw → draft", lineage)
		_, _ = m.deps.Git.AddAndCommit([]string{relPath}, msg, authorName, authorEmail)
	}

	// WebSocket broadcast.
	if m.deps.Hub != nil {
		m.deps.Hub.Broadcast(hub.Event{
			Type: "artifact.indexed",
			Payload: map[string]any{
				"path":   relPath,
				"action": "transitioned",
				"from":   "raw",
				"to":     "draft",
			},
		})
	}

	return nil
}

// resolveAgentConfig finds the idea-triage agent entry and returns its ModelConfig.
func (m *Manager) resolveAgentConfig() (ideachat.ModelConfig, error) {
	for _, a := range m.deps.Agents {
		if a.Name != m.opts.AgentName {
			continue
		}
		prompt, ok := a.PromptTemplates["idea-generate"]
		if !ok {
			return ideachat.ModelConfig{}, fmt.Errorf("agent %q has no idea-generate template", m.opts.AgentName)
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
	return ideachat.ModelConfig{}, fmt.Errorf("agent %q not found in project config", m.opts.AgentName)
}

// rewriteBody rewrites the artifact body to contain ## Raw Idea and ## Idea sections.
// On a re-run (both sections already present) only the ## Idea block is replaced.
func rewriteBody(originalBody, agentBody string) string {
	// Strip a leading H1 from the agent body to avoid duplication.
	stripped := stripLeadingH1(agentBody)

	// Re-run case: file already has both sections — replace ## Idea only.
	if strings.Contains(originalBody, "## Raw Idea") && strings.Contains(originalBody, "## Idea") {
		return replaceIdeaSection(originalBody, stripped)
	}

	// First-time triage: split H1 from the rest, wrap original in ## Raw Idea.
	h1, rest := splitH1(originalBody)
	var sb strings.Builder
	if h1 != "" {
		sb.WriteString(h1)
		sb.WriteString("\n\n")
	}
	sb.WriteString("## Raw Idea\n\n")
	sb.WriteString(strings.TrimSpace(rest))
	sb.WriteString("\n\n## Idea\n\n")
	sb.WriteString(strings.TrimSpace(stripped))
	return sb.String()
}

// splitH1 splits body into the leading "# Title" line and everything after it.
func splitH1(body string) (h1, rest string) {
	trimmed := strings.TrimSpace(body)
	nl := strings.IndexByte(trimmed, '\n')
	if nl < 0 {
		if strings.HasPrefix(trimmed, "# ") {
			return trimmed, ""
		}
		return "", trimmed
	}
	first := trimmed[:nl]
	if strings.HasPrefix(first, "# ") {
		return first, strings.TrimSpace(trimmed[nl+1:])
	}
	return "", trimmed
}

// stripLeadingH1 removes a leading "# Title" line from body.
func stripLeadingH1(body string) string {
	trimmed := strings.TrimSpace(body)
	nl := strings.IndexByte(trimmed, '\n')
	if nl < 0 {
		if strings.HasPrefix(trimmed, "# ") {
			return ""
		}
		return trimmed
	}
	if strings.HasPrefix(trimmed[:nl], "# ") {
		return strings.TrimSpace(trimmed[nl+1:])
	}
	return trimmed
}

// replaceIdeaSection replaces the ## Idea section with newContent.
// Everything from ## Idea to the next ## heading (or EOF) is replaced.
func replaceIdeaSection(body, newContent string) string {
	const marker = "## Idea"
	idx := strings.Index(body, "\n"+marker)
	if idx < 0 {
		if strings.HasPrefix(body, marker) {
			idx = -1 // header at start
		} else {
			return body // unexpected: marker not found
		}
	}

	var before string
	if idx >= 0 {
		before = body[:idx]
	}
	return before + "\n## Idea\n\n" + strings.TrimSpace(newContent)
}

// mergeAndFilterLabels merges existing + proposed labels, keeping only those
// in vocab. Existing labels come first; duplicates are removed.
func mergeAndFilterLabels(existing, proposed, vocab []string) []string {
	vocabSet := make(map[string]bool, len(vocab))
	for _, v := range vocab {
		vocabSet[v] = true
	}
	seen := make(map[string]bool, len(existing)+len(proposed))
	var out []string
	for _, l := range append(existing, proposed...) {
		if vocabSet[l] && !seen[l] {
			seen[l] = true
			out = append(out, l)
		}
	}
	return out
}

// marshalArtifact serialises frontmatter + body into a complete markdown file.
func marshalArtifact(fm artifact.Frontmatter, body string) ([]byte, error) {
	fmBytes, err := yaml.Marshal(fm)
	if err != nil {
		return nil, fmt.Errorf("marshal frontmatter: %w", err)
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
	return []byte(sb.String()), nil
}

// ----- run record helpers (Milestone 5) -----

func (m *Manager) recordRunStart(relPath, runID, lineage, trigger string, startedAt time.Time) {
	_ = m.deps.Idx.InsertAgentRun(&index.AgentRunRow{
		RunID:      runID,
		AgentName:  m.opts.AgentName,
		Role:       "product-owner",
		TargetPath: relPath,
		StartedAt:  startedAt,
		Status:     "running",
		StderrTail: "",
	})
}

func (m *Manager) recordRunComplete(runID, status string, durationMs int64, stderr string) {
	now := time.Now()
	_ = m.deps.Idx.UpdateAgentRun(&index.AgentRunRow{
		RunID:      runID,
		Status:     status,
		FinishedAt: &now,
		StderrTail: stderr,
	})
	_ = durationMs // used in log (M9)
}
