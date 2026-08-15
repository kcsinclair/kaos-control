// SPDX-License-Identifier: AGPL-3.0-or-later

package index

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/artifact"
)

func riceF64(v float64) *float64 { return &v }

// makeRiceTestArtifact builds a minimal idea Artifact, optionally with all
// four RICE components set.
func makeRiceTestArtifact(path, slug string, reach, impact, confidence, effort *float64) *artifact.Artifact {
	return &artifact.Artifact{
		Path:  path,
		Slug:  slug,
		Stage: "ideas",
		Index: 0,
		Mtime: time.Now(),
		FM: artifact.Frontmatter{
			Title:          slug,
			Type:           "idea",
			Status:         "draft",
			Lineage:        slug,
			RiceReach:      reach,
			RiceImpact:     impact,
			RiceConfidence: confidence,
			RiceEffort:     effort,
		},
	}
}

// TestUpsert_RiceScore_ScoredAndUnscored verifies Upsert stores the derived
// rice_score for a fully-scored artifact and leaves it NULL for one with an
// incomplete or absent RICE set.
func TestUpsert_RiceScore_ScoredAndUnscored(t *testing.T) {
	idx := openTestIndex(t)

	scored := makeRiceTestArtifact("lifecycle/ideas/scored.md", "scored",
		riceF64(100), riceF64(2), riceF64(50), riceF64(4))
	unscored := makeRiceTestArtifact("lifecycle/ideas/unscored.md", "unscored",
		riceF64(100), nil, riceF64(50), riceF64(4))
	absent := makeRiceTestArtifact("lifecycle/ideas/absent.md", "absent", nil, nil, nil, nil)

	for _, a := range []*artifact.Artifact{scored, unscored, absent} {
		if err := idx.Upsert(a); err != nil {
			t.Fatalf("Upsert(%s): %v", a.Path, err)
		}
	}

	row, err := idx.Get("lifecycle/ideas/scored.md")
	if err != nil {
		t.Fatalf("Get(scored): %v", err)
	}
	if row.RiceScore == nil {
		t.Fatal("scored: expected non-nil RiceScore")
	}
	wantScore, _ := artifact.RiceScore(scored.FM)
	if *row.RiceScore != wantScore {
		t.Errorf("scored: RiceScore = %v, want %v", *row.RiceScore, wantScore)
	}

	for _, path := range []string{"lifecycle/ideas/unscored.md", "lifecycle/ideas/absent.md"} {
		row, err := idx.Get(path)
		if err != nil {
			t.Fatalf("Get(%s): %v", path, err)
		}
		if row.RiceScore != nil {
			t.Errorf("%s: expected nil RiceScore, got %v", path, *row.RiceScore)
		}
	}
}

// TestRiceScore_BackwardCompat_JSONOmitted verifies that an artifact with no
// RICE fields round-trips through JSON with rice_score absent (omitempty),
// not present as null.
func TestRiceScore_BackwardCompat_JSONOmitted(t *testing.T) {
	idx := openTestIndex(t)

	a := makeRiceTestArtifact("lifecycle/ideas/no-rice.md", "no-rice", nil, nil, nil, nil)
	if err := idx.Upsert(a); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	row, err := idx.Get("lifecycle/ideas/no-rice.md")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if row.RiceScore != nil {
		t.Fatalf("expected nil RiceScore, got %v", *row.RiceScore)
	}

	out, err := json.Marshal(row)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if strings.Contains(string(out), "rice_score") {
		t.Errorf("expected rice_score key absent from JSON, got:\n%s", out)
	}
}

// TestList_SortByRice_NullsLastBothDirections verifies sort=rice:asc and
// sort=rice:desc both order scored rows numerically and group all NULL
// (unscored) rows together after the scored ones, in both directions.
func TestList_SortByRice_NullsLastBothDirections(t *testing.T) {
	idx := openTestIndex(t)

	artifacts := []*artifact.Artifact{
		makeRiceTestArtifact("lifecycle/ideas/low.md", "low", riceF64(1), riceF64(1), riceF64(50), riceF64(10)),     // score 0.05
		makeRiceTestArtifact("lifecycle/ideas/high.md", "high", riceF64(100), riceF64(2), riceF64(100), riceF64(2)), // score 100
		makeRiceTestArtifact("lifecycle/ideas/mid.md", "mid", riceF64(10), riceF64(1), riceF64(100), riceF64(2)),    // score 5
		makeRiceTestArtifact("lifecycle/ideas/null-a.md", "null-a", nil, nil, nil, nil),
		makeRiceTestArtifact("lifecycle/ideas/null-b.md", "null-b", riceF64(1), nil, nil, nil),
	}
	for _, a := range artifacts {
		if err := idx.Upsert(a); err != nil {
			t.Fatalf("Upsert(%s): %v", a.Path, err)
		}
	}

	t.Run("asc", func(t *testing.T) {
		rows, _, err := idx.List(Filter{Sort: "rice:asc", Unlimited: true})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		wantOrder := []string{"lifecycle/ideas/low.md", "lifecycle/ideas/mid.md", "lifecycle/ideas/high.md"}
		assertRicePrefixOrder(t, rows, wantOrder)
		assertNullsLast(t, rows, 3)
	})

	t.Run("desc", func(t *testing.T) {
		rows, _, err := idx.List(Filter{Sort: "rice:desc", Unlimited: true})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		wantOrder := []string{"lifecycle/ideas/high.md", "lifecycle/ideas/mid.md", "lifecycle/ideas/low.md"}
		assertRicePrefixOrder(t, rows, wantOrder)
		assertNullsLast(t, rows, 3)
	})
}

func assertRicePrefixOrder(t *testing.T, rows []*ArtifactRow, wantOrder []string) {
	t.Helper()
	if len(rows) < len(wantOrder) {
		t.Fatalf("got %d rows, want at least %d", len(rows), len(wantOrder))
	}
	for i, wantPath := range wantOrder {
		if rows[i].Path != wantPath {
			t.Errorf("position %d: want %s, got %s", i, wantPath, rows[i].Path)
		}
	}
}

// assertNullsLast verifies that from index scoredCount onward, every row has
// a nil RiceScore, and no nil RiceScore appears before that index.
func assertNullsLast(t *testing.T, rows []*ArtifactRow, scoredCount int) {
	t.Helper()
	for i, r := range rows {
		if i < scoredCount && r.RiceScore == nil {
			t.Errorf("position %d (%s): expected scored row, got nil RiceScore", i, r.Path)
		}
		if i >= scoredCount && r.RiceScore != nil {
			t.Errorf("position %d (%s): expected NULL-grouped row, got RiceScore=%v", i, r.Path, *r.RiceScore)
		}
	}
}

// TestRiceScore_BackfillAfterSchemaRebuild verifies that a schemaVersion-
// triggered rebuild (dropAndRecreate) followed by re-indexing recomputes
// rice_score for every artifact — the mtime/SHA guards that normally skip
// unchanged files are bypassed because the rebuilt table starts empty.
func TestRiceScore_BackfillAfterSchemaRebuild(t *testing.T) {
	idx := openTestIndex(t)

	a := makeRiceTestArtifact("lifecycle/ideas/rebuild.md", "rebuild",
		riceF64(100), riceF64(2), riceF64(50), riceF64(4))
	if err := idx.Upsert(a); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	before, err := idx.Get(a.Path)
	if err != nil || before.RiceScore == nil {
		t.Fatalf("Get before rebuild: row=%+v err=%v", before, err)
	}

	// Simulate the Open()-time schemaVersion-mismatch path.
	if err := idx.dropAndRecreate(); err != nil {
		t.Fatalf("dropAndRecreate: %v", err)
	}
	if row, err := idx.Get(a.Path); err != nil || row != nil {
		t.Fatalf("expected empty table immediately after rebuild, got row=%+v err=%v", row, err)
	}

	// Re-index (what Scan does on the next startup walk).
	if err := idx.Upsert(a); err != nil {
		t.Fatalf("Upsert after rebuild: %v", err)
	}
	after, err := idx.Get(a.Path)
	if err != nil {
		t.Fatalf("Get after rebuild: %v", err)
	}
	if after.RiceScore == nil {
		t.Fatal("expected rice_score to be back-filled after rebuild")
	}
	if *after.RiceScore != *before.RiceScore {
		t.Errorf("rice_score after rebuild = %v, want %v", *after.RiceScore, *before.RiceScore)
	}
}
