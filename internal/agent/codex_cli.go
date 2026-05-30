// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// CodexCLIDriver spawns `codex exec` for a non-interactive agent run.
type CodexCLIDriver struct {
	// BinaryPath is the absolute path to the codex binary. Defaults to "codex" if empty.
	BinaryPath string
}

func (d *CodexCLIDriver) buildArgs(run Run) []string {
	return d.buildArgsWithOptions(run, false)
}

func (d *CodexCLIDriver) buildArgsWithOptions(run Run, supportsTimeout bool) []string {
	args := []string{"exec", "--json", "--dangerously-bypass-approvals-and-sandbox"}
	if run.ProjectRoot != "" {
		args = append(args, "--cd", run.ProjectRoot)
	}
	if run.Model != "" {
		args = append(args, "--model", run.Model)
	}
	if supportsTimeout {
		timeoutSeconds := 24 * 60 * 60
		if run.TimeoutMinutes > 0 {
			timeoutSeconds = run.TimeoutMinutes * 60
		}
		args = append(args, "--timeout", strconv.Itoa(timeoutSeconds))
	}
	args = append(args, run.PromptText)
	return args
}

func (d *CodexCLIDriver) Start(ctx context.Context, run Run) (Process, error) {
	binary := d.BinaryPath
	if binary == "" {
		binary = "codex"
	}

	supportsTimeout := d.BinaryPath == "" && codexExecSupportsTimeout(ctx, binary)
	args := d.buildArgsWithOptions(run, supportsTimeout)
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
		return nil, fmt.Errorf("starting codex: %w", err)
	}

	waitErr := make(chan error, 1)
	p := &cliProcess{cmd: cmd, progress: progressCh, stderr: rb, logFile: logFile, waitErr: waitErr}

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

	go func() {
		err := cmd.Wait()
		waitErr <- err
		close(waitErr)
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

func codexExecSupportsTimeout(ctx context.Context, binary string) bool {
	// 10 s rather than 2 s — the probe is one-shot at driver startup, not
	// perf-critical, and the smaller budget flaked in concurrent test runs
	// where fork+exec of a shell shim missed the deadline. The wider window
	// still bounds the worst case if the real codex binary genuinely hangs.
	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	out, err := exec.CommandContext(probeCtx, binary, "exec", "--help").CombinedOutput()
	if err != nil {
		return false
	}
	return bytes.Contains(out, []byte("--timeout"))
}
