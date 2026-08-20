// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"context"
	"sync"
	"testing"
)

// TestShellStubDriver_ConcurrentStart_NoPanic is a regression test for the
// "send on closed channel" panic: progressCh must only be closed after both
// the stdout-scan and stderr-drain goroutines have returned, regardless of
// which one finishes first. Run under `go test -race` to also catch data
// races on progressCh/stderrBuf.
func TestShellStubDriver_ConcurrentStart_NoPanic(t *testing.T) {
	d := &ShellStubDriver{}
	root := t.TempDir()

	// Emit output on both stdout and stderr with no ordering guarantee
	// between the two streams, so the stdout-scan and stderr-drain
	// goroutines race to finish first.
	const cmd = `for i in 1 2 3 4 5; do echo "{\"type\":\"progress\",\"i\":$i}"; echo "err $i" >&2; done`

	const concurrency = 50
	var wg sync.WaitGroup
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()

			proc, err := d.Start(context.Background(), Run{
				ProjectRoot:  root,
				ShellCommand: cmd,
			})
			if err != nil {
				t.Errorf("Start: %v", err)
				return
			}

			for range proc.Progress() {
				// Drain until the channel is closed.
			}

			if err := proc.Wait(); err != nil {
				t.Errorf("Wait: %v", err)
			}
		}()
	}
	wg.Wait()
}
