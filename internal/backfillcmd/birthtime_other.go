// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build !darwin

package backfillcmd

import (
	"os"
	"time"
)

// fileBirthTime falls back to the file's mtime on non-Darwin platforms.
// Linux can expose birth time via the statx(2) syscall on recent kernels
// + filesystems but `os.FileInfo` does not surface it portably; mtime is
// a serviceable approximation for the backfill case.
func fileBirthTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}
