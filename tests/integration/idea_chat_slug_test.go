// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Milestone 4 – Slug Generation and Collision Tests
//
// All tests here require ANTHROPIC_API_KEY. They validate that proposed slugs
// are well-formed, content-derived, and collision-safe.

// TestIdeaChatSlugIsValid verifies that the lineage field in preview.frontmatter
// matches the slug regex ^[a-z0-9][a-z0-9\-]*[a-z0-9]$|^[a-z0-9]$.
func TestIdeaChatSlugIsValid(t *testing.T) {
	skipIfNoAPIKey(t)
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	_, data := convergeToProposal(t, env, uniqueIdeaMessage("slug-valid"))

	preview, _ := data["preview"].(map[string]any)
	if preview == nil {
		t.Fatal("preview is nil")
	}
	fm, _ := preview["frontmatter"].(map[string]any)
	lineage, _ := fm["lineage"].(string)

	if lineage == "" {
		t.Fatal("preview.frontmatter.lineage is empty")
	}
	if !slugPattern.MatchString(lineage) {
		t.Errorf("lineage %q does not match slug pattern ^[a-z0-9][a-z0-9\\-]*[a-z0-9]$|^[a-z0-9]$", lineage)
	}
}

// TestIdeaChatSlugDerivedFromContent verifies that for a message about
// "dark mode toggle for settings" the generated slug contains at least one
// of the key terms: dark-mode, settings, toggle, dark, mode.
func TestIdeaChatSlugDerivedFromContent(t *testing.T) {
	skipIfNoAPIKey(t)
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	_, data := convergeToProposal(t, env,
		`I want a dark mode toggle in the settings panel of the application. ` +
		`Users should be able to switch between light and dark themes and the ` +
		`preference should persist across sessions. The toggle should appear in ` +
		`the user profile settings area with a clear label.`)

	preview, _ := data["preview"].(map[string]any)
	if preview == nil {
		t.Fatal("preview is nil")
	}
	fm, _ := preview["frontmatter"].(map[string]any)
	lineage, _ := fm["lineage"].(string)
	if lineage == "" {
		t.Fatal("lineage is empty")
	}

	keyTerms := []string{"dark", "mode", "toggle", "settings", "theme"}
	var matched bool
	for _, term := range keyTerms {
		if strings.Contains(lineage, term) {
			matched = true
			break
		}
	}
	if !matched {
		t.Errorf("slug %q does not contain any key term from %v", lineage, keyTerms)
	}
}

// TestIdeaChatSlugCollisionResolution verifies that when a file with slug
// "dark-mode" already exists, the generated slug for a similar idea differs
// from "dark-mode" (collision detection avoids overwriting the existing file).
func TestIdeaChatSlugCollisionResolution(t *testing.T) {
	skipIfNoAPIKey(t)

	// Pre-create lifecycle/ideas/dark-mode.md so its lineage is in the index.
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/dark-mode.md",
			content: makeArtifact("Dark Mode", "idea", "draft", "dark-mode", "", "An existing dark mode idea."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	_, data := convergeToProposal(t, env,
		`Add a dark mode to the application interface. Users want to switch ` +
		`between a light and dark colour scheme from the settings page, with ` +
		`the choice persisted in their profile so it is restored on next login.`)

	preview, _ := data["preview"].(map[string]any)
	if preview == nil {
		t.Fatal("preview is nil")
	}
	fm, _ := preview["frontmatter"].(map[string]any)
	lineage, _ := fm["lineage"].(string)
	if lineage == "" {
		t.Fatal("lineage is empty")
	}

	if lineage == "dark-mode" {
		t.Errorf("slug collision not resolved: resulting slug is still 'dark-mode', which already exists")
	}

	// The file at lifecycle/ideas/dark-mode.md must still exist (not overwritten).
	existingPath := filepath.Join(env.projectRoot, "lifecycle", "ideas", "dark-mode.md")
	if _, err := os.Stat(existingPath); os.IsNotExist(err) {
		t.Error("pre-existing lifecycle/ideas/dark-mode.md was deleted during collision resolution")
	}
}

// TestIdeaChatSlugLength verifies that the generated slug consists of 2–5
// hyphen-separated word segments (ignoring any trailing numeric disambiguator).
func TestIdeaChatSlugLength(t *testing.T) {
	skipIfNoAPIKey(t)
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	_, data := convergeToProposal(t, env, uniqueIdeaMessage("slug-length"))

	preview, _ := data["preview"].(map[string]any)
	if preview == nil {
		t.Fatal("preview is nil")
	}
	fm, _ := preview["frontmatter"].(map[string]any)
	lineage, _ := fm["lineage"].(string)
	if lineage == "" {
		t.Fatal("lineage is empty")
	}

	segments := strings.Split(lineage, "-")

	// Strip a trailing numeric segment that may have been added for collision
	// resolution (e.g. "my-idea-2" → keep 2 semantic segments "my" and "idea").
	if len(segments) > 1 {
		last := segments[len(segments)-1]
		if isNumericStr(last) {
			segments = segments[:len(segments)-1]
		}
	}

	if len(segments) < 2 || len(segments) > 5 {
		t.Errorf("slug %q has %d word segment(s) (want 2–5)", lineage, len(segments))
	}
}

// isNumericStr returns true if s consists entirely of ASCII digits.
func isNumericStr(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
