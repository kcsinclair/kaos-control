// Package project holds per-project runtime state.
package project

import (
	"fmt"
	"path/filepath"

	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/internal/index"
)

// Project is the runtime services container for one registered project.
type Project struct {
	Entry *config.ProjectEntry
	Cfg   *config.Project
	Idx   *index.Index
}

// Open loads the project config, opens the SQLite index, and scans the lifecycle tree.
// dbDir is the app-level data directory; per-project DBs live at dbDir/<name>/index.db.
func Open(entry *config.ProjectEntry, dbDir string) (*Project, error) {
	cfg, err := config.LoadProject(entry.Path)
	if err != nil {
		return nil, fmt.Errorf("project %q: loading config: %w", entry.Name, err)
	}

	dbPath := filepath.Join(dbDir, entry.Name, "index.db")
	idx, err := index.Open(dbPath, entry.Path, cfg.Stages)
	if err != nil {
		return nil, fmt.Errorf("project %q: opening index: %w", entry.Name, err)
	}

	return &Project{Entry: entry, Cfg: cfg, Idx: idx}, nil
}

// Close releases resources held by the project.
func (p *Project) Close() error {
	return p.Idx.Close()
}

// LifecycleDir returns the absolute path to the lifecycle/ directory.
func (p *Project) LifecycleDir() string {
	return filepath.Join(p.Entry.Path, "lifecycle")
}
