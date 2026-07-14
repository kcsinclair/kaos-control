// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/artifact"
)

// Milestone 1 — path-component parsing & rel_path derivation.
//
// internal/artifact's parsePathComponents is unexported, so it can't be
// tested directly from outside the package; the write scope for this test
// suite is restricted to tests/** (see lifecycle/tests/idea-archiving-5-test.md),
// so these cases drive the exported artifact.Parse entry point instead —
// parsePathComponents is a pure function of the same relPath argument, so
// this exercises identical logic end-to-end.

func rawArtifact(title, typ, status, lineage, parent string) []byte {
	fm := "---\ntitle: " + title + "\ntype: " + typ + "\nstatus: " + status + "\nlineage: " + lineage + "\n"
	if parent != "" {
		fm += "parent: " + parent + "\n"
	}
	fm += "---\n\nBody.\n"
	return []byte(fm)
}

func TestArtifactParse_FlatPath(t *testing.T) {
	a := artifact.Parse(rawArtifact("Login", "idea", "draft", "login", ""), "lifecycle/ideas/login.md", time.Time{})
	if a.Stage != "ideas" {
		t.Errorf("Stage: want %q, got %q", "ideas", a.Stage)
	}
	if a.RelPath != "login.md" {
		t.Errorf("RelPath: want %q, got %q", "login.md", a.RelPath)
	}
	if a.Slug != "login" || a.Index != 0 {
		t.Errorf("Slug/Index: want (login, 0), got (%s, %d)", a.Slug, a.Index)
	}
}

func TestArtifactParse_SingleNestedPath(t *testing.T) {
	flat := artifact.Parse(rawArtifact("Login", "idea", "draft", "login", ""), "lifecycle/ideas/login.md", time.Time{})
	nested := artifact.Parse(rawArtifact("Login", "idea", "draft", "login", ""), "lifecycle/ideas/done/login.md", time.Time{})

	if nested.Stage != "ideas" {
		t.Errorf("Stage: want %q, got %q", "ideas", nested.Stage)
	}
	if nested.RelPath != "done/login.md" {
		t.Errorf("RelPath: want %q, got %q", "done/login.md", nested.RelPath)
	}
	// Slug/Index must be identical to the flat case — folder placement is invisible
	// to lineage/index derivation (AC "unchanged by moving between folders").
	if nested.Slug != flat.Slug || nested.Index != flat.Index {
		t.Errorf("Slug/Index diverged by folder: flat=(%s,%d) nested=(%s,%d)", flat.Slug, flat.Index, nested.Slug, nested.Index)
	}
}

func TestArtifactParse_DeeplyNestedPath(t *testing.T) {
	a := artifact.Parse(rawArtifact("Release X", "release", "draft", "deep", ""), "lifecycle/ideas/2026/q3/release-x.md", time.Time{})
	if a.Stage != "ideas" {
		t.Errorf("Stage: want %q, got %q", "ideas", a.Stage)
	}
	if a.RelPath != "2026/q3/release-x.md" {
		t.Errorf("RelPath: want %q, got %q", "2026/q3/release-x.md", a.RelPath)
	}
}

// TestArtifactParse_CrossPlatformSeparators mirrors how every production
// caller constructs relPath: filepath.Join (OS-native separator) followed by
// filepath.ToSlash before it reaches artifact.Parse. On Windows that Join
// produces backslashes; ToSlash must still normalise to forward slashes.
func TestArtifactParse_CrossPlatformSeparators(t *testing.T) {
	native := filepath.Join("lifecycle", "ideas", "done", "login.md")
	relPath := filepath.ToSlash(native)

	a := artifact.Parse(rawArtifact("Login", "idea", "draft", "login", ""), relPath, time.Time{})
	if a.RelPath != "done/login.md" {
		t.Errorf("RelPath: want %q, got %q", "done/login.md", a.RelPath)
	}
	if a.Stage != "ideas" {
		t.Errorf("Stage: want %q, got %q", "ideas", a.Stage)
	}
}

// TestArtifactParse_FrontmatterInvariantAcrossFolders verifies type, status,
// lineage, and parent are byte-identical for the same content regardless of
// which folder the file lives in (req #5; AC "unchanged by moving between
// folders").
func TestArtifactParse_FrontmatterInvariantAcrossFolders(t *testing.T) {
	content := rawArtifact("Nested Req", "plan-backend", "in-development", "nested-req", "lifecycle/ideas/nested-req.md")

	flat := artifact.Parse(content, "lifecycle/backend-plans/nested-req-3-be.md", time.Time{})
	nested := artifact.Parse(content, "lifecycle/backend-plans/archive/nested-req-3-be.md", time.Time{})
	deep := artifact.Parse(content, "lifecycle/backend-plans/2026/q3/nested-req-3-be.md", time.Time{})

	for _, pair := range []struct {
		name string
		a    *artifact.Artifact
	}{{"nested", nested}, {"deep", deep}} {
		if pair.a.FM.Type != flat.FM.Type {
			t.Errorf("%s: Type diverged: flat=%q got=%q", pair.name, flat.FM.Type, pair.a.FM.Type)
		}
		if pair.a.FM.Status != flat.FM.Status {
			t.Errorf("%s: Status diverged: flat=%q got=%q", pair.name, flat.FM.Status, pair.a.FM.Status)
		}
		if pair.a.FM.Lineage != flat.FM.Lineage {
			t.Errorf("%s: Lineage diverged: flat=%q got=%q", pair.name, flat.FM.Lineage, pair.a.FM.Lineage)
		}
		if pair.a.FM.Parent != flat.FM.Parent {
			t.Errorf("%s: Parent diverged: flat=%q got=%q", pair.name, flat.FM.Parent, pair.a.FM.Parent)
		}
		// Slug/Index must also be folder-invariant (same stem, same suffix parsing).
		if pair.a.Slug != flat.Slug || pair.a.Index != flat.Index || pair.a.StageSuffix != flat.StageSuffix {
			t.Errorf("%s: Slug/Index/StageSuffix diverged from flat: flat=(%s,%d,%s) got=(%s,%d,%s)",
				pair.name, flat.Slug, flat.Index, flat.StageSuffix, pair.a.Slug, pair.a.Index, pair.a.StageSuffix)
		}
	}
}
