// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// GeminiCliDriver spawns `agy --dangerously-skip-permissions --prompt "<prompt>"`
// to execute agents using the Antigravity CLI and automatically skips permission prompts.
type GeminiCliDriver struct {
	// BinaryPath is the absolute path to the agy binary. Defaults to "agy" if empty.
	BinaryPath string
}

func (d *GeminiCliDriver) buildArgs(run Run) []string {
	args := []string{
		"--dangerously-skip-permissions",
	}
	// agy ignores cmd.Dir for its workspace context and defaults to
	// ~/.gemini/antigravity-cli/scratch, so the agent spends the whole run
	// hunting for the project ("I will list the parent directory…") and
	// eventually hits --print-timeout. --add-dir tells agy which directory
	// to include in the workspace; without it agy's log records
	// workspaceDirs=[] and the model has no project context to work with.
	if run.ProjectRoot != "" {
		args = append(args, "--add-dir", run.ProjectRoot)
	}
	args = append(args, "--prompt", run.PromptText)
	return args
}

func (d *GeminiCliDriver) Start(ctx context.Context, run Run) (Process, error) {
	binary := d.BinaryPath
	if binary == "" {
		binary = "agy"
	}

	args := d.buildArgs(run)
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = run.ProjectRoot

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	rb := newRingBuf(4 * 1024)
	progressCh := make(chan ProgressEvent, 64)

	// Open the per-run log file if configured.
	var logFile *os.File
	if run.LogPath != "" {
		if err := os.MkdirAll(filepath.Dir(run.LogPath), 0o755); err != nil {
			slog.Warn("agent: creating log dir failed", "path", run.LogPath, "err", err)
		} else {
			f, err := os.OpenFile(run.LogPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				slog.Warn("agent: opening log file failed", "path", run.LogPath, "err", err)
			} else {
				logFile = f
				fmt.Fprintf(logFile, "# kaos-control agent run %s\n# agent=%s role=%s model=%s\n# args=%v\n# started=%s\n\n",
					run.RunID, run.AgentName, run.Role, run.Model, args, time.Now().Format(time.RFC3339))
			}
		}
	}

	if err := cmd.Start(); err != nil {
		if logFile != nil {
			_ = logFile.Close()
		}
		return nil, fmt.Errorf("starting agy: %w", err)
	}

	waitErr := make(chan error, 1)
	p := &claudeProcess{cmd: cmd, progress: progressCh, stderr: rb, logFile: logFile, waitErr: waitErr}

	// Stream starts with opening started event to let UI know it initiated.
	select {
	case progressCh <- ProgressEvent{
		Raw: "started",
		Event: map[string]any{
			"type": "started",
		},
	}:
	default:
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// Pipe stdout: tee to log file, parse json or wrap raw text, send as progress events.
	go func() {
		defer wg.Done()
		sc := bufio.NewScanner(stdout)
		sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
		for sc.Scan() {
			line := sc.Text()
			if logFile != nil {
				_, _ = io.WriteString(logFile, line+"\n")
			}
			ev := ProgressEvent{Raw: line}
			var parsed map[string]any
			if err := json.Unmarshal([]byte(line), &parsed); err == nil {
				ev.Event = parsed
			} else {
				// Wrap raw text line as an output progress event.
				ev.Event = map[string]any{
					"type": "output",
					"text": line + "\n",
				}
			}
			select {
			case progressCh <- ev:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Pipe stderr: tee to log file and ring buffer.
	go func() {
		defer wg.Done()
		buf := make([]byte, 1024)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				rb.Write(buf[:n])
				if logFile != nil {
					_, _ = logFile.Write(buf[:n])
				}
			}
			if err != nil {
				break
			}
		}
	}()

	// Process-exit watcher: the agy CLI (Antigravity) detaches a grandchild
	// that inherits stdout/stderr and keeps writing its FDs even after agy
	// itself exits, so the pipe goroutines above would block on Read forever
	// waiting for an EOF that never comes. Run cmd.Wait() asynchronously —
	// it reaps the agy process and closes the parent ends of the pipes,
	// which unblocks the readers. The result is stashed in waitErr so
	// claudeProcess.Wait() (called by supervise after the drain loop) can
	// return it without double-calling cmd.Wait.
	go func() {
		err := cmd.Wait()
		waitErr <- err
		close(waitErr)
	}()

	// Wait and clean up goroutine.
	go func() {
		wg.Wait()
		close(progressCh)
		if logFile != nil {
			fmt.Fprintf(logFile, "\n# finished=%s\n", time.Now().Format(time.RFC3339))
			_ = logFile.Close()
		}
	}()

	return p, nil
}
