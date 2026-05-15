// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
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
}

func (d *ShellStubDriver) Start(ctx context.Context, run Run) (Process, error) {
	command := run.ShellCommand
	if command == "" {
		command = fmt.Sprintf(`printf '{"type":"result","subtype":"success","is_error":false}\n'`)
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

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("shell-stub start: %w", err)
	}

	p := &stubProcess{cmd: cmd, progress: progressCh, stderrBuf: rb}

	go func() {
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
		defer close(progressCh)
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
