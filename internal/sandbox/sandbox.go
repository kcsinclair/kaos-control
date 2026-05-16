// SPDX-License-Identifier: AGPL-3.0-or-later

// Package sandbox validates user-supplied paths to prevent traversal outside the project root.
package sandbox

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

var ErrPathTraversal = errors.New("path traversal detected")
var ErrAbsolutePath = errors.New("absolute paths not allowed")

// Resolve validates and resolves a user-supplied relative path within projectRoot.
// It returns the cleaned absolute path or an error if the path would escape the root.
// The target file need not exist yet (e.g. when creating a new artifact).
func Resolve(projectRoot, userPath string) (string, error) {
	if filepath.IsAbs(userPath) {
		return "", ErrAbsolutePath
	}

	clean := filepath.Clean(userPath)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", ErrPathTraversal
	}

	// Resolve the root itself so symlinks in the project path don't cause
	// false-positive traversal detections (e.g. on macOS where /var→/private/var).
	resolvedRoot, err := filepath.EvalSymlinks(projectRoot)
	if err != nil {
		resolvedRoot = filepath.Clean(projectRoot)
	}

	abs := filepath.Join(resolvedRoot, clean)

	// If the target exists, resolve symlinks and verify it stays inside the root.
	resolved, err := filepath.EvalSymlinks(abs)
	if err == nil {
		if !hasPrefix(resolved, resolvedRoot) {
			return "", ErrPathTraversal
		}
		return resolved, nil
	}

	// Target doesn't exist yet — walk up to the nearest existing ancestor and
	// verify it stays inside the root. This handles the case where the stage
	// directory itself (e.g. lifecycle/docs) hasn't been created yet.
	ancestor := filepath.Dir(abs)
	for {
		resolvedAncestor, aerr := filepath.EvalSymlinks(ancestor)
		if aerr == nil {
			if !hasPrefix(resolvedAncestor, resolvedRoot) {
				return "", ErrPathTraversal
			}
			return abs, nil
		}
		next := filepath.Dir(ancestor)
		if next == ancestor {
			// Reached the filesystem root without finding an existing directory.
			return "", fmt.Errorf("resolving parent directory: %w", aerr)
		}
		ancestor = next
	}
}

// hasPrefix returns true if path is equal to root or is rooted within root.
func hasPrefix(path, root string) bool {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	return path == root || strings.HasPrefix(path, root+string(filepath.Separator))
}
