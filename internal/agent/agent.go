// Package agent implements the agent runner: driver interface, claude-code-cli
// driver, run lifecycle management, and scope enforcement.
package agent

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kaos-control/kaos-control/internal/artifact"
	"github.com/kaos-control/kaos-control/internal/auth"
	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/index"
	kgit "github.com/kaos-control/kaos-control/internal/git"
	"github.com/kaos-control/kaos-control/internal/lock"
)

// ErrBusy is returned when the global semaphore is full.
var ErrBusy = errors.New("max_concurrent_agents limit reached")

// ErrNotFound is returned when an agent name is unknown.
var ErrNotFound = errors.New("agent not found")

// Run holds everything a driver needs to start one agent execution.
type Run struct {
	RunID        string
	AgentName    string
	Role         string
	Model        string // empty → CLI default
	PromptText   string
	ProjectRoot  string
	AllowedPaths []string
	GitIdentity  config.GitIdentity
	LogPath      string // absolute path; if empty, no log file is written
	// Status lifecycle fields (copied from AgentConfig).
	TargetPath    string // project-relative path to the target artifact
	ActiveStatus  string // status to set on target when run starts (empty = no change)
	DoneOnSuccess bool   // if true, set target status to "done" on successful completion
	TimeoutMinutes int   // 0 = driver default
	// Ollama-specific fields (only used when Driver == "ollama").
	OllamaInstanceName string // resolved from AgentConfig.OllamaInstanceName
	OllamaEndpoint     string // "chat" or "generate"
}

// Process is a handle to a running agent.
type Process interface {
	// Wait blocks until the agent exits and returns any error.
	Wait() error
	// Kill sends SIGTERM to the running process.
	Kill() error
	// Progress returns a channel of structured output events (closed on exit).
	Progress() <-chan ProgressEvent
	// StderrTail returns the last 4 KB of stderr output.
	StderrTail() string
}

// Driver is the pluggable execution interface.
type Driver interface {
	Start(ctx context.Context, run Run) (Process, error)
}

// ----- ring buffer -----

type ringBuf struct {
	mu   sync.Mutex
	data []byte
	cap  int
}

func newRingBuf(capacity int) *ringBuf { return &ringBuf{cap: capacity} }

func (rb *ringBuf) Write(p []byte) (int, error) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.data = append(rb.data, p...)
	if len(rb.data) > rb.cap {
		rb.data = rb.data[len(rb.data)-rb.cap:]
	}
	return len(p), nil
}

func (rb *ringBuf) String() string {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	return string(rb.data)
}

// ----- claude-code-cli driver -----

// ClaudeCodeDriver spawns `claude --dangerously-skip-permissions -p "<prompt>"`.
type ClaudeCodeDriver struct{}

type claudeProcess struct {
	cmd      *exec.Cmd
	progress chan ProgressEvent
	stderr   *ringBuf
	logFile  *os.File // nil if no log path was configured
}

// ProgressEvent is one structured update from the agent. raw is the original
// stdout line; event is the parsed JSON payload (nil if the line wasn't valid
// JSON — e.g. when stream-json is disabled or the line is partial).
type ProgressEvent struct {
	Raw   string         `json:"raw"`
	Event map[string]any `json:"event,omitempty"`
}

func (d *ClaudeCodeDriver) Start(ctx context.Context, run Run) (Process, error) {
	args := []string{
		"--dangerously-skip-permissions",
		"-p", run.PromptText,
		"--output-format", "stream-json",
		"--verbose",
	}
	if run.Model != "" {
		args = append(args, "--model", run.Model)
	}

	cmd := exec.CommandContext(ctx, "claude", args...)
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
		return nil, fmt.Errorf("starting claude: %w", err)
	}

	p := &claudeProcess{cmd: cmd, progress: progressCh, stderr: rb, logFile: logFile}

	// Pipe stdout: tee to log file, parse each line as JSON, send progress events.
	go func() {
		sc := bufio.NewScanner(stdout)
		// stream-json events can be larger than the default 64 KiB buffer.
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
			}
			select {
			case progressCh <- ev:
			default:
			}
		}
	}()

	// Pipe stderr: tee to log file and ring buffer.
	go func() {
		defer close(progressCh)
		defer func() {
			if logFile != nil {
				fmt.Fprintf(logFile, "\n# finished=%s\n", time.Now().Format(time.RFC3339))
				_ = logFile.Close()
			}
		}()
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

	return p, nil
}

func (p *claudeProcess) Wait() error                     { return p.cmd.Wait() }
func (p *claudeProcess) Progress() <-chan ProgressEvent  { return p.progress }
func (p *claudeProcess) StderrTail() string              { return p.stderr.String() }

func (p *claudeProcess) Kill() error {
	if p.cmd.Process != nil {
		return p.cmd.Process.Kill()
	}
	return nil
}

// ----- active run record -----

type activeRun struct {
	proc   Process
	cancel context.CancelFunc
}

// ----- Manager -----

// Manager runs agents with lineage locking and a global concurrency semaphore.
type Manager struct {
	agents  []config.AgentConfig
	drivers map[string]Driver
	sem     chan struct{}

	mu     sync.Mutex
	active map[string]*activeRun

	idx     *index.Index
	git     *kgit.Repo
	hub     *hub.Hub
	locks   *lock.Manager
	root    string
	logsDir string // per-run log files go in <logsDir>/<run_id>.log
}

// New creates an agent Manager. maxConcurrent caps parallel runs across the project.
// logsDir is where per-run .log files are written; empty disables log files.
// ollamaInstances is the app-level list of registered Ollama servers.
func New(
	agents []config.AgentConfig,
	maxConcurrent int,
	idx *index.Index,
	git *kgit.Repo,
	h *hub.Hub,
	locks *lock.Manager,
	root string,
	logsDir string,
	ollamaInstances []config.OllamaInstance,
) *Manager {
	if maxConcurrent <= 0 {
		maxConcurrent = 4
	}
	m := &Manager{
		agents: agents,
		drivers: map[string]Driver{
			"claude-code-cli": &ClaudeCodeDriver{},
			"ollama":          &OllamaDriver{Instances: ollamaInstances},
		},
		sem:     make(chan struct{}, maxConcurrent),
		active:  make(map[string]*activeRun),
		idx:     idx,
		git:     git,
		hub:     h,
		locks:   locks,
		root:    root,
		logsDir: logsDir,
	}
	// Crash recovery: any run still marked running from a prior process is now failed.
	if err := idx.RecoverRunningRuns(); err != nil {
		slog.Warn("agent manager: error recovering running runs", "err", err)
	}
	return m
}

// LogPath returns the absolute path to the per-run log file, or "" when logging
// is disabled (no logsDir was configured).
func (m *Manager) LogPath(runID string) string {
	if m.logsDir == "" {
		return ""
	}
	return filepath.Join(m.logsDir, runID+".log")
}

// Agents returns the configured agent list (read-only).
func (m *Manager) Agents() []config.AgentConfig { return m.agents }

// GetAgent returns the named agent config.
func (m *Manager) GetAgent(name string) (*config.AgentConfig, bool) {
	for i := range m.agents {
		if m.agents[i].Name == name {
			return &m.agents[i], true
		}
	}
	return nil, false
}

// StartRun initiates an agent run and returns the run_id.
// agentName must match a configured agent. targetPath is the artifact to operate on.
// role selects the prompt template; if empty, the agent's first role is used.
func (m *Manager) StartRun(ctx context.Context, agentName, targetPath, role string, user *auth.User) (string, error) {
	ag, ok := m.GetAgent(agentName)
	if !ok {
		return "", ErrNotFound
	}

	// Pick role and prompt template.
	if role == "" && len(ag.Roles) > 0 {
		role = ag.Roles[0]
	}
	promptTpl, ok := ag.PromptTemplates[role]
	if !ok {
		return "", fmt.Errorf("agent %q has no prompt template for role %q", agentName, role)
	}
	prompt := strings.NewReplacer("{target_path}", targetPath).Replace(promptTpl)

	// Determine lineage and previous status from the target artifact (if present in index).
	lineage := targetPath
	prevStatus := ""
	if row, err := m.idx.Get(targetPath); err == nil && row != nil {
		lineage = row.FM.Lineage
		prevStatus = row.Status
	}

	// Check lineage lock.
	if existing, err := m.locks.Get(lineage); err != nil {
		return "", fmt.Errorf("checking lock: %w", err)
	} else if existing != nil {
		return "", fmt.Errorf("lineage %q is locked by %s (%s): %w", lineage, existing.Holder, existing.Kind, lock.ErrLocked)
	}

	// Fail fast: look up driver before acquiring any resources.
	drv, drvOK := m.drivers[ag.Driver]
	if !drvOK {
		return "", fmt.Errorf("unknown driver %q for agent %q", ag.Driver, agentName)
	}

	// Try semaphore (non-blocking).
	select {
	case m.sem <- struct{}{}:
	default:
		return "", ErrBusy
	}

	runID := randomHex(8)
	identity := ag.GitIdentity
	if identity.Name == "" {
		identity.Name = agentName
		identity.Email = agentName + "@kaos-control.local"
	}

	run := Run{
		RunID:              runID,
		AgentName:          agentName,
		Role:               role,
		Model:              ag.Model,
		PromptText:         prompt,
		ProjectRoot:        m.root,
		AllowedPaths:       ag.AllowedPaths,
		GitIdentity:        identity,
		LogPath:            m.LogPath(runID),
		TargetPath:         targetPath,
		ActiveStatus:       ag.ActiveStatus,
		DoneOnSuccess:      ag.DoneOnSuccess,
		TimeoutMinutes:     ag.TimeoutMinutes,
		OllamaInstanceName: ag.OllamaInstanceName,
		OllamaEndpoint:     ag.OllamaEndpoint,
	}

	// Acquire lineage lock.
	if _, err := m.locks.Acquire(lineage, runID, "agent"); err != nil {
		<-m.sem
		return "", fmt.Errorf("acquiring lock: %w", err)
	}

	// Insert run record.
	now := time.Now()
	runRow := &index.AgentRunRow{
		RunID:      runID,
		AgentName:  agentName,
		Role:       role,
		TargetPath: targetPath,
		StartedAt:  now,
		Status:     "running",
	}
	if err := m.idx.InsertAgentRun(runRow); err != nil {
		_ = m.locks.Release(lineage)
		<-m.sem
		return "", fmt.Errorf("inserting run record: %w", err)
	}

	// If configured, mark the target artifact as active before launching.
	if ag.ActiveStatus != "" && targetPath != "" {
		if err := m.setArtifactStatus(targetPath, ag.ActiveStatus); err != nil {
			slog.Warn("agent: setting active status", "target", targetPath, "status", ag.ActiveStatus, "err", err)
		} else if m.git != nil {
			authorName, authorEmail := m.git.ResolveIdentity()
			msg := fmt.Sprintf("status(%s): %s → %s [run:%s]", lineage, prevStatus, ag.ActiveStatus, runID)
			_, _ = m.git.AddAndCommit([]string{targetPath}, msg, authorName, authorEmail)
		}
	}

	// Start the driver process. timeout_minutes=0 disables the timeout.
	var runCtx context.Context
	var cancel context.CancelFunc
	if ag.TimeoutMinutes > 0 {
		runCtx, cancel = context.WithTimeout(context.Background(), time.Duration(ag.TimeoutMinutes)*time.Minute)
	} else {
		runCtx, cancel = context.WithCancel(context.Background())
	}
	proc, err := drv.Start(runCtx, run)
	if err != nil {
		cancel()
		_ = m.locks.Release(lineage)
		<-m.sem
		finishedAt := time.Now()
		runRow.Status = "failed"
		runRow.FinishedAt = &finishedAt
		_ = m.idx.UpdateAgentRun(runRow)
		return "", fmt.Errorf("starting process: %w", err)
	}

	m.mu.Lock()
	m.active[runID] = &activeRun{proc: proc, cancel: cancel}
	m.mu.Unlock()

	m.hub.Broadcast(hub.Event{
		Type:    "agent.started",
		Payload: map[string]any{"run_id": runID, "agent": agentName, "lineage": lineage, "target_path": targetPath},
	})

	// Record feed event and broadcast feed.new.
	{
		runIDCopy := runID
		summary := fmt.Sprintf("Agent %s started on %s", agentName, targetPath)
		feedEvent := &index.EventRow{
			EventType: "agent_started",
			Timestamp: time.Now().Unix(),
			Actor:     agentName,
			RunID:     &runIDCopy,
			Summary:   summary,
		}
		if err := m.idx.InsertEvent(feedEvent); err == nil {
			m.hub.Broadcast(hub.Event{Type: "feed.new", Payload: feedEvent})
		}
	}

	// Supervisor goroutine.
	go m.supervise(runCtx, cancel, run, runRow, lineage)

	return runID, nil
}

func (m *Manager) supervise(ctx context.Context, cancel context.CancelFunc, run Run, row *index.AgentRunRow, lineage string) {
	defer cancel()
	proc := m.getProc(run.RunID)
	if proc == nil {
		return
	}

	// Forward progress events as structured payloads.
	go func() {
		for ev := range proc.Progress() {
			payload := map[string]any{
				"run_id": run.RunID,
				"line":   ev.Raw, // backward-compat: existing UI reads `line`
				"raw":    ev.Raw,
			}
			if ev.Event != nil {
				payload["event"] = ev.Event
			}
			m.hub.Broadcast(hub.Event{Type: "agent.progress", Payload: payload})
		}
	}()

	waitErr := proc.Wait()
	finishedAt := time.Now()

	m.mu.Lock()
	_, wasActive := m.active[run.RunID]
	delete(m.active, run.RunID)
	m.mu.Unlock()

	// Release semaphore.
	<-m.sem

	// Determine exit status. Timeouts are distinct from user-initiated kills.
	exitCode := 0
	status := "done"
	if waitErr != nil {
		status = "failed"
		if wasActive {
			switch {
			case errors.Is(ctx.Err(), context.DeadlineExceeded):
				status = "killed-timeout"
			case errors.Is(ctx.Err(), context.Canceled):
				status = "killed"
			}
		}
	}

	// If configured and successful, mark the target artifact done before committing
	// so the status change is bundled into the agent's commit automatically.
	if status == "done" && run.DoneOnSuccess && run.TargetPath != "" {
		if err := m.setArtifactStatus(run.TargetPath, "done"); err != nil {
			slog.Warn("agent: setting done status", "target", run.TargetPath, "err", err)
		}
	}

	// Commit any files produced within allowed paths (even on failure = partial commit).
	var produced []string
	if m.git != nil {
		files, err := m.git.ModifiedFiles(run.AllowedPaths)
		if err != nil {
			slog.Warn("agent: git status failed", "run_id", run.RunID, "err", err)
		} else if len(files) > 0 {
			produced = files
			commitMsg := fmt.Sprintf("agent(%s): run %s [%s]", run.AgentName, run.RunID, status)
			if _, commitErr := m.git.AddAndCommit(files, commitMsg, run.GitIdentity.Name, run.GitIdentity.Email); commitErr != nil {
				slog.Warn("agent: commit failed", "run_id", run.RunID, "err", commitErr)
			} else {
				// Only re-index lifecycle markdown — code files committed by
				// developer agents are not artifacts.
				for _, f := range files {
					if !strings.HasPrefix(f, "lifecycle/") || !strings.HasSuffix(f, ".md") {
						continue
					}
					_ = m.idx.IndexFile(m.root + "/" + f)
				}
				m.hub.Broadcast(hub.Event{
					Type:    "git.committed",
					Payload: map[string]any{"run_id": run.RunID, "files": files},
				})

				// Record feed event and broadcast feed.new.
				runIDCopy := run.RunID
				summary := fmt.Sprintf("Agent %s committed %d file(s)", run.AgentName, len(files))
				feedEvent := &index.EventRow{
					EventType: "git_committed",
					Timestamp: time.Now().Unix(),
					Actor:     run.AgentName,
					RunID:     &runIDCopy,
					Summary:   summary,
				}
				if err := m.idx.InsertEvent(feedEvent); err == nil {
					m.hub.Broadcast(hub.Event{Type: "feed.new", Payload: feedEvent})
				}
			}
		}
	}

	// Release lock.
	_ = m.locks.Release(lineage)

	// Update run record.
	row.Status = status
	row.FinishedAt = &finishedAt
	row.ExitCode = &exitCode
	if waitErr != nil {
		code := -1
		row.ExitCode = &code
	}
	row.StderrTail = proc.StderrTail()
	row.ArtifactsProduced = produced
	_ = m.idx.UpdateAgentRun(row)

	eventType := "agent.finished"
	feedEventType := "agent_finished"
	if status == "failed" || status == "killed" || status == "killed-timeout" {
		eventType = "agent.failed"
		feedEventType = "agent_failed"
	}
	m.hub.Broadcast(hub.Event{
		Type: eventType,
		Payload: map[string]any{
			"run_id":      run.RunID,
			"agent":       run.AgentName,
			"lineage":     lineage,
			"status":      status,
			"artifacts":   produced,
			"target_path": row.TargetPath,
		},
	})

	// Record feed event and broadcast feed.new.
	{
		runIDCopy := run.RunID
		summary := fmt.Sprintf("Agent %s finished with status %s; produced %d artifact(s)", run.AgentName, status, len(produced))
		feedEvent := &index.EventRow{
			EventType: feedEventType,
			Timestamp: time.Now().Unix(),
			Actor:     run.AgentName,
			RunID:     &runIDCopy,
			Summary:   summary,
		}
		if err := m.idx.InsertEvent(feedEvent); err == nil {
			m.hub.Broadcast(hub.Event{Type: "feed.new", Payload: feedEvent})
		}
	}
}

// Kill sends SIGTERM to the running agent with the given run_id.
func (m *Manager) Kill(runID string) error {
	m.mu.Lock()
	ar, ok := m.active[runID]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("run %q is not active", runID)
	}
	ar.cancel() // cancels the process context → SIGKILL via CommandContext
	return ar.proc.Kill()
}

// GetRun returns the run record from the index.
func (m *Manager) GetRun(runID string) (*index.AgentRunRow, error) {
	return m.idx.GetAgentRun(runID)
}

// ListRuns returns run records from the index.
func (m *Manager) ListRuns(status string, limit int) ([]*index.AgentRunRow, error) {
	return m.idx.ListAgentRuns(status, limit)
}

// ListRunsByTargetPath returns run records for a given target path, newest first.
func (m *Manager) ListRunsByTargetPath(targetPath string) ([]*index.AgentRunRow, error) {
	return m.idx.ListAgentRunsByTargetPath(targetPath)
}

func (m *Manager) getProc(runID string) Process {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ar, ok := m.active[runID]; ok {
		return ar.proc
	}
	return nil
}

// setArtifactStatus patches the status field of an artifact on disk and re-indexes it.
func (m *Manager) setArtifactStatus(relPath, status string) error {
	absPath := filepath.Join(m.root, relPath)
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", relPath, err)
	}
	patched, ok := artifact.PatchFrontmatterField(raw, "status", status)
	if !ok {
		return fmt.Errorf("status field not found in frontmatter of %s", relPath)
	}
	if err := os.WriteFile(absPath, patched, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", relPath, err)
	}
	_ = m.idx.IndexFile(absPath)
	return nil
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
