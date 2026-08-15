// SPDX-License-Identifier: AGPL-3.0-or-later

package directives

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// NeedsMigration reports whether projectRoot is on the legacy
// single-CLAUDE.md layout that Migrate can upgrade: a root CLAUDE.md
// exists, no AGENTS.md exists yet, and CLAUDE.md is not already a bare
// `@AGENTS.md` pointer (i.e. migration hasn't already happened).
func NeedsMigration(projectRoot string) (bool, error) {
	claudeRaw, err := os.ReadFile(filepath.Join(projectRoot, claudeFile))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("reading %s: %w", claudeFile, err)
	}

	if _, err := os.Stat(filepath.Join(projectRoot, agentsFile)); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("statting %s: %w", agentsFile, err)
	}

	return !isBarePointer(claudeRaw), nil
}

// isBarePointer reports whether raw is exactly the `@AGENTS.md` pointer
// body RenderPointer produces (ignoring surrounding whitespace).
func isBarePointer(raw []byte) bool {
	return strings.TrimSpace(string(raw)) == "@"+agentsFile
}

// MigrateOptions configures Migrate.
type MigrateOptions struct {
	// Force allows overwriting an AGENTS.md that already exists and differs
	// from the migrated legacy content, instead of withholding it behind a
	// Diff.
	Force bool
}

// Migrate upgrades a project from the legacy single-CLAUDE.md layout to the
// AGENTS.md-primary set (FR-16): the existing CLAUDE.md body is renamed to
// AGENTS.md, wrapped in the managed-region markers so a later
// stack-tuned Generate can refresh it surgically; CLAUDE.md itself becomes
// an `@AGENTS.md` pointer; GEMINI.md is added when a gemini driver is
// configured (same driver-gating as Generate). It is idempotent: called
// again on an already-migrated project (or one with no CLAUDE.md at all) it
// is a no-op, returning an empty GenerateResult. If AGENTS.md already
// exists and differs from the migrated content, nothing is written and the
// difference is reported via FileWrite.Diff unless opts.Force is set.
func Migrate(projectRoot string, opts MigrateOptions) (GenerateResult, error) {
	claudePath := filepath.Join(projectRoot, claudeFile)
	agentsPath := filepath.Join(projectRoot, agentsFile)

	legacyRaw, err := os.ReadFile(claudePath)
	if err != nil {
		if os.IsNotExist(err) {
			return GenerateResult{}, nil // nothing to migrate
		}
		return GenerateResult{}, fmt.Errorf("reading %s: %w", claudeFile, err)
	}
	if isBarePointer(legacyRaw) {
		return GenerateResult{}, nil // already migrated
	}

	wrapped := legacyRaw
	if !hasMarkers(legacyRaw) {
		wrapped = wrapLegacyBody(legacyRaw)
	}

	var result GenerateResult

	fw, err := writeFile(agentsPath, wrapped, opts.Force)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("writing %s: %w", agentsFile, err)
	}
	result.Files = append(result.Files, relativizeFileWrite(fw, agentsFile))
	if fw.Diff != "" {
		// AGENTS.md already exists with content that doesn't match the
		// migrated legacy body, and Force wasn't given — stop here without
		// touching CLAUDE.md or GEMINI.md (FR-16).
		return result, nil
	}

	// The legacy content now lives in AGENTS.md — CLAUDE.md becomes a
	// pointer unconditionally; there's nothing left in it worth diffing.
	pointer := RenderPointer(agentsFile)
	fw, err = writeFile(claudePath, pointer, true)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("writing %s: %w", claudeFile, err)
	}
	result.Files = append(result.Files, relativizeFileWrite(fw, claudeFile))

	drivers, err := configuredDrivers(projectRoot)
	if err != nil {
		return GenerateResult{}, err
	}
	if hasGeminiDriver(drivers) {
		fw, err = writeFile(filepath.Join(projectRoot, geminiFile), pointer, opts.Force)
		if err != nil {
			return GenerateResult{}, fmt.Errorf("writing %s: %w", geminiFile, err)
		}
		result.Files = append(result.Files, relativizeFileWrite(fw, geminiFile))
	} else {
		result.Skipped = append(result.Skipped, geminiFile)
	}

	return result, nil
}

// wrapLegacyBody wraps raw in the managed-region marker pair so it becomes
// eligible for a later surgical refresh (see writeFile).
func wrapLegacyBody(raw []byte) []byte {
	body := bytes.TrimRight(raw, "\n")
	var out bytes.Buffer
	out.WriteString(genStart)
	out.WriteString("\n")
	out.Write(body)
	out.WriteString("\n")
	out.WriteString(genEnd)
	out.WriteString("\n")
	return out.Bytes()
}
