// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build darwin

package backfillcmd

import (
	"os"
	"syscall"
	"time"
)

// fileBirthTime returns the filesystem birth time of the file at path
// (HFS+/APFS `st_birthtime`). On any error or zero result, falls back to
// the file's mtime.
func fileBirthTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		// Birthtimespec is documented on Darwin. A zero value means the
		// filesystem didn't record one, so fall through to mtime.
		bt := st.Birthtimespec
		if bt.Sec != 0 {
			//nolint:unconvert // Sec/Nsec are int64 on 64-bit darwin, int32 elsewhere
			return time.Unix(int64(bt.Sec), int64(bt.Nsec))
		}
	}
	return info.ModTime()
}
