// SPDX-License-Identifier: AGPL-3.0-or-later

package release

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// File is the on-disk representation of a release markdown file.
type File struct {
	Title     string
	Slug      string
	Status    string
	StartDate *time.Time
	EndDate   *time.Time
	UpdatedAt time.Time
	Body      string
}

var (
	slugStripRe    = regexp.MustCompile(`[^a-z0-9-]`)
	slugCollapseRe = regexp.MustCompile(`-{2,}`)
)

// Slugify converts name to a URL-safe slug (lowercase, spaces→"-",
// strip [^a-z0-9-], collapse runs of "-", trim leading/trailing "-").
// Returns an empty string when no usable characters remain; callers
// are responsible for the "release-<id>" fallback in that case.
func Slugify(name string) string {
	s := strings.ToLower(name)
	s = strings.ReplaceAll(s, " ", "-")
	s = slugStripRe.ReplaceAllString(s, "")
	s = slugCollapseRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// Parse parses a release markdown file from raw bytes.
// path is used only to derive the Slug from the filename stem.
func Parse(path string, raw []byte) (*File, error) {
	s := string(raw)
	if !strings.HasPrefix(s, "---") {
		return nil, errors.New("missing frontmatter: file must start with ---")
	}
	rest := s[3:]
	closeIdx := strings.Index(rest, "\n---")
	if closeIdx < 0 {
		return nil, errors.New("unclosed frontmatter: missing closing ---")
	}
	fmYAML := rest[:closeIdx]
	body := strings.TrimSpace(rest[closeIdx+4:])

	var fm struct {
		Title     string `yaml:"title"`
		Type      string `yaml:"type"`
		Status    string `yaml:"status"`
		StartDate string `yaml:"start_date"`
		EndDate   string `yaml:"end_date"`
		UpdatedAt string `yaml:"updated_at"`
	}
	if err := yaml.Unmarshal([]byte(fmYAML), &fm); err != nil {
		return nil, fmt.Errorf("parsing frontmatter: %w", err)
	}

	var errs []error
	if fm.Title == "" {
		errs = append(errs, errors.New("missing required field: title"))
	}
	if fm.Type != "release" {
		errs = append(errs, fmt.Errorf("expected type %q, got %q", "release", fm.Type))
	}
	if fm.Status == "" {
		errs = append(errs, errors.New("missing required field: status"))
	} else if !ValidStatuses[fm.Status] {
		errs = append(errs, fmt.Errorf("invalid status %q; must be one of: planned, active, shipped, unscheduled", fm.Status))
	}
	if len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}
		return nil, errors.New(strings.Join(msgs, "; "))
	}

	f := &File{
		Title:  fm.Title,
		Status: fm.Status,
		Body:   body,
		Slug:   strings.TrimSuffix(filepath.Base(path), ".md"),
	}

	if fm.StartDate != "" {
		t, err := time.Parse("2006-01-02", fm.StartDate)
		if err != nil {
			return nil, fmt.Errorf("invalid start_date %q: %w", fm.StartDate, err)
		}
		f.StartDate = &t
	}
	if fm.EndDate != "" {
		t, err := time.Parse("2006-01-02", fm.EndDate)
		if err != nil {
			return nil, fmt.Errorf("invalid end_date %q: %w", fm.EndDate, err)
		}
		f.EndDate = &t
	}
	if f.StartDate != nil && f.EndDate != nil && f.EndDate.Before(*f.StartDate) {
		return nil, errors.New("end_date must be on or after start_date")
	}
	if fm.UpdatedAt != "" {
		t, err := time.Parse(time.RFC3339, fm.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("invalid updated_at %q: %w", fm.UpdatedAt, err)
		}
		f.UpdatedAt = t
	}

	return f, nil
}

// releaseFM is used for deterministic YAML key ordering in Marshal.
// Fields must be declared in the desired output order.
type releaseFM struct {
	Title     string `yaml:"title"`
	Type      string `yaml:"type"`
	Status    string `yaml:"status"`
	StartDate string `yaml:"start_date,omitempty"`
	EndDate   string `yaml:"end_date,omitempty"`
	UpdatedAt string `yaml:"updated_at"`
}

// Marshal serialises f to a markdown file with YAML frontmatter.
// Keys are written in deterministic order: title, type, status,
// start_date, end_date, updated_at.
func (f *File) Marshal() ([]byte, error) {
	fm := releaseFM{
		Title:     f.Title,
		Type:      "release",
		Status:    f.Status,
		UpdatedAt: f.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if f.StartDate != nil {
		fm.StartDate = f.StartDate.Format("2006-01-02")
	}
	if f.EndDate != nil {
		fm.EndDate = f.EndDate.Format("2006-01-02")
	}

	yamlBytes, err := yaml.Marshal(fm)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(yamlBytes)
	buf.WriteString("---\n")
	if f.Body != "" {
		buf.WriteString(f.Body)
		if !strings.HasSuffix(f.Body, "\n") {
			buf.WriteByte('\n')
		}
	}
	return buf.Bytes(), nil
}
