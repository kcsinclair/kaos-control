// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// ShellStubDriver is a test-only agent driver that runs a configurable shell
// command. It is intentionally minimal: stdout lines are forwarded as progress
// events; the process is treated as succeeded when it exits 0.
//
// Configure via Run.ShellCommand. If ShellCommand is empty, the driver emits
// one synthetic result event and exits immediately.
type ShellStubDriver struct{}

type stubProcess struct {
	cmd       *exec.Cmd
	progress  chan ProgressEvent
	stderrBuf *ringBuf
	logFile   *os.File // nil if no log path was configured
}

func (d *ShellStubDriver) Start(ctx context.Context, run Run) (Process, error) {
	command := run.ShellCommand
	if command == "" {
		command = `printf '{"type":"result","subtype":"success","is_error":false}\n'`
	}
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = run.ProjectRoot

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("shell-stub stdout pipe: %w", err)
	}
	rb := newRingBuf(4 * 1024)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("shell-stub stderr pipe: %w", err)
	}

	progressCh := make(chan ProgressEvent, 64)

	// Open the per-run log file if configured, mirroring startCommandProcess.
	var logFile *os.File
	if run.LogPath != "" {
		if err := os.MkdirAll(filepath.Dir(run.LogPath), 0o755); err != nil {
			slog.Warn("shell-stub agent: creating log dir failed", "path", run.LogPath, "err", err)
		} else {
			f, err := os.OpenFile(run.LogPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				slog.Warn("shell-stub agent: opening log file failed", "path", run.LogPath, "err", err)
			} else {
				logFile = f
				writeRunLogHeader(logFile, run, nil)
			}
		}
	}

	if err := cmd.Start(); err != nil {
		if logFile != nil {
			_ = logFile.Close()
		}
		return nil, fmt.Errorf("shell-stub start: %w", err)
	}

	p := &stubProcess{cmd: cmd, progress: progressCh, stderrBuf: rb, logFile: logFile}

	// Both reader goroutines below only ever write to progressCh (the stdout
	// scanner) or read from stderr (never touching progressCh). Neither may
	// close progressCh until the other has finished, or a send on the
	// already-closed channel panics. A WaitGroup lets a third goroutine close
	// it exactly once, after both readers are done — mirroring the pattern in
	// startCommandProcess (agent.go).
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			line := sc.Text()
			ev := ProgressEvent{Raw: line}
			var parsed map[string]any
			if json.Unmarshal([]byte(line), &parsed) == nil {
				ev.Event = parsed
			}
			select {
			case progressCh <- ev:
			default:
			}
		}
	}()

	go func() {
		defer wg.Done()
		buf := make([]byte, 1024)
		for {
			n, readErr := stderr.Read(buf)
			if n > 0 {
				rb.Write(buf[:n])
			}
			if readErr != nil {
				break
			}
		}
	}()

	go func() {
		wg.Wait()
		if logFile != nil {
			fmt.Fprintf(logFile, "\n# finished=%s\n", time.Now().Format(time.RFC3339))
			_ = logFile.Close()
		}
		close(progressCh)
	}()

	return p, nil
}

func (p *stubProcess) Wait() error                    { return p.cmd.Wait() }
func (p *stubProcess) Progress() <-chan ProgressEvent { return p.progress }
func (p *stubProcess) StderrTail() string             { return p.stderrBuf.String() }

func (p *stubProcess) Kill() error {
	if p.cmd.Process != nil {
		return p.cmd.Process.Kill()
	}
	return nil
}
