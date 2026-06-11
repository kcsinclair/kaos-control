// SPDX-License-Identifier: AGPL-3.0-or-later

package release

import (
	"strings"
	"testing"
	"time"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"spaces to hyphens", "Q1 2026", "q1-2026"},
		{"already hyphenated", "Q1-2026", "q1-2026"},
		{"emoji only", "🚀🚀", ""},
		{"mixed emoji", "🚀 Release", "release"},
		{"uppercase", "ALPHA", "alpha"},
		{"special chars", "v1.0 (beta)", "v10-beta"},
		{"leading trailing hyphens", " - foo - ", "foo"},
		{"multiple spaces", "foo  bar", "foo-bar"},
		{"numbers", "2026 Q2", "2026-q2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Slugify(tt.in)
			if got != tt.want {
				t.Errorf("Slugify(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParse_Errors(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr string
	}{
		{
			name:    "no frontmatter",
			raw:     "# Hello",
			wantErr: "missing frontmatter",
		},
		{
			name:    "unclosed frontmatter",
			raw:     "---\ntitle: foo\n",
			wantErr: "unclosed frontmatter",
		},
		{
			name:    "missing title",
			raw:     "---\ntype: release\nstatus: planned\nupdated_at: 2026-01-01T00:00:00Z\n---\n",
			wantErr: "missing required field: title",
		},
		{
			name:    "wrong type",
			raw:     "---\ntitle: Q1\ntype: idea\nstatus: planned\nupdated_at: 2026-01-01T00:00:00Z\n---\n",
			wantErr: `expected type "release"`,
		},
		{
			name:    "unknown status",
			raw:     "---\ntitle: Q1\ntype: release\nstatus: foo\nupdated_at: 2026-01-01T00:00:00Z\n---\n",
			wantErr: `invalid status "foo"`,
		},
		{
			name:    "malformed start_date",
			raw:     "---\ntitle: Q1\ntype: release\nstatus: planned\nstart_date: not-a-date\nupdated_at: 2026-01-01T00:00:00Z\n---\n",
			wantErr: "invalid start_date",
		},
		{
			name:    "end before start",
			raw:     "---\ntitle: Q1\ntype: release\nstatus: planned\nstart_date: 2026-06-01\nend_date: 2026-01-01\nupdated_at: 2026-01-01T00:00:00Z\n---\n",
			wantErr: "end_date must be on or after start_date",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse("q1-2026.md", []byte(tt.raw))
			if err == nil {
				t.Fatalf("Parse: expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Parse error = %q, want it to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestParse_ValidStatuses(t *testing.T) {
	for _, st := range []string{"planned", "active", "shipped", "unscheduled"} {
		raw := "---\ntitle: Q1\ntype: release\nstatus: " + st + "\nupdated_at: 2026-01-01T00:00:00Z\n---\n"
		f, err := Parse("q1.md", []byte(raw))
		if err != nil {
			t.Errorf("Parse with status=%q: unexpected error: %v", st, err)
			continue
		}
		if f.Status != st {
			t.Errorf("Parse with status=%q: got status=%q", st, f.Status)
		}
	}
}

func TestParse_RejectsUnknownStatus(t *testing.T) {
	raw := "---\ntitle: Q1\ntype: release\nstatus: foo\nupdated_at: 2026-01-01T00:00:00Z\n---\n"
	_, err := Parse("q1.md", []byte(raw))
	if err == nil {
		t.Error("expected error for unknown status 'foo'")
	}
}

func TestMarshalRoundTrip(t *testing.T) {
	start := mustDate(t, "2026-01-01")
	end := mustDate(t, "2026-03-31")
	updated := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	orig := &File{
		Title:     "Q1 2026",
		Slug:      "q1-2026",
		Status:    "planned",
		StartDate: &start,
		EndDate:   &end,
		UpdatedAt: updated,
		Body:      "This is the release body.",
	}

	data, err := orig.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	parsed, err := Parse("q1-2026.md", data)
	if err != nil {
		t.Fatalf("Parse after Marshal: %v", err)
	}

	if parsed.Title != orig.Title {
		t.Errorf("Title: got %q, want %q", parsed.Title, orig.Title)
	}
	if parsed.Status != orig.Status {
		t.Errorf("Status: got %q, want %q", parsed.Status, orig.Status)
	}
	if parsed.Slug != orig.Slug {
		t.Errorf("Slug: got %q, want %q", parsed.Slug, orig.Slug)
	}
	if parsed.StartDate == nil || !parsed.StartDate.Equal(start) {
		t.Errorf("StartDate: got %v, want %v", parsed.StartDate, start)
	}
	if parsed.EndDate == nil || !parsed.EndDate.Equal(end) {
		t.Errorf("EndDate: got %v, want %v", parsed.EndDate, end)
	}
	// UpdatedAt may lose sub-second precision through RFC3339 serialisation.
	if !parsed.UpdatedAt.Equal(orig.UpdatedAt.Truncate(time.Second)) {
		t.Errorf("UpdatedAt: got %v, want %v", parsed.UpdatedAt, orig.UpdatedAt.Truncate(time.Second))
	}
	if parsed.Body != orig.Body {
		t.Errorf("Body: got %q, want %q", parsed.Body, orig.Body)
	}
}

func TestMarshalKeyOrder(t *testing.T) {
	f := &File{
		Title:     "Test",
		Status:    "planned",
		UpdatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	data, err := f.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	s := string(data)
	titleIdx := strings.Index(s, "title:")
	typeIdx := strings.Index(s, "type:")
	statusIdx := strings.Index(s, "status:")
	updatedIdx := strings.Index(s, "updated_at:")
	if titleIdx > typeIdx || typeIdx > statusIdx || statusIdx > updatedIdx {
		t.Errorf("key order wrong in output:\n%s", s)
	}
}

func mustDate(t *testing.T, s string) time.Time {
	t.Helper()
	tt, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("mustDate(%q): %v", s, err)
	}
	return tt
}
