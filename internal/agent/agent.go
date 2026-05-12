// SPDX-License-Identifier: AGPL-3.0-or-later

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
	"syscall"
	"time"

	"github.com/kaos-control/kaos-control/internal/artifact"
	"github.com/kaos-control/kaos-control/internal/auth"
	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/index"
	kgit "github.com/kaos-control/kaos-control/internal/git"
	"github.com/kaos-control/kaos-control/internal/lock"
)

// processKiller abstracts graceful process termination for testability.
// The production implementation sends SIGTERM and, after a 2-second grace
// period, SIGKILL. Tests can inject a no-op or recording implementation.
type processKiller interface {
	Kill(proc Process)
}

// defaultProcessKiller is the production processKiller: SIGTERM → 2 s → SIGKILL.
type defaultProcessKiller struct{}

func (defaultProcessKiller) Kill(proc Process) {
	cp, ok := proc.(*claudeProcess)
	if !ok || cp.cmd.Process == nil {
		_ = proc.Kill()
		return
	}
	// Best-effort SIGTERM; ignore error (process may have already exited).
	_ = cp.cmd.Process.Signal(syscall.SIGTERM)
	go func() {
		time.Sleep(2 * time.Second)
		_ = cp.cmd.Process.Kill() // SIGKILL if still alive
	}()
}

// WorkflowEngine is the subset of workflow.Engine used by the agent Manager.
// The interface avoids an import cycle: agent → workflow → (none), but
// workflow → index → (none) while index → workflow would be circular.
type WorkflowEngine interface {
	CanTransition(from, to string, userRoles []string, artifactType string) bool
}

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
	// RelatedTestPath is set when the target artifact is a test artifact.
	// It is passed to the agent prompt via the {related_test} placeholder so
	// the agent can reference the test in defect frontmatter (related_to field).
	RelatedTestPath string
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

// ClaudeCodeDriver spawns `claude --permission-mode bypassPermissions
// --dangerously-skip-permissions -p "<prompt>"` (dual-flag invocation so that
// both current and legacy Claude Code binaries enable bypass mode).
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

// buildArgs constructs the CLI argument slice for a claude invocation.
// It is exported for use by tests that want to inspect the argument list
// without starting a real subprocess.
func (d *ClaudeCodeDriver) buildArgs(run Run) []string {
	args := []string{
		"--permission-mode", "bypassPermissions",
		"--dangerously-skip-permissions", // legacy alias; older binaries ignore --permission-mode
		"-p", run.PromptText,
		"--output-format", "stream-json",
		"--verbose",
	}
	if run.Model != "" {
		args = append(args, "--model", run.Model)
	}
	return args
}

func (d *ClaudeCodeDriver) Start(ctx context.Context, run Run) (Process, error) {
	args := d.buildArgs(run)

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
	wf      WorkflowEngine // may be nil; used for type-aware transition validation
	root    string
	logsDir string // per-run log files go in <logsDir>/<run_id>.log

	// Precheck configuration (from AppAgentConfig).
	initEventTimeout   time.Duration // how long to wait for the system/init event
	requireBypassPerms bool          // whether to reject non-bypassPermissions runs
	killer             processKiller // injectable for tests
}

// New creates an agent Manager. maxConcurrent caps parallel runs across the project.
// logsDir is where per-run .log files are written; empty disables log files.
// ollamaInstances is the app-level list of registered Ollama servers.
// wf is the optional workflow engine used for type-aware transition validation.
// agentCfg supplies the precheck timeout and bypass-permissions requirement.
func New(
	agents []config.AgentConfig,
	maxConcurrent int,
	idx *index.Index,
	git *kgit.Repo,
	h *hub.Hub,
	locks *lock.Manager,
	wf WorkflowEngine,
	root string,
	logsDir string,
	ollamaInstances []config.OllamaInstance,
	agentCfg config.AppAgentConfig,
) *Manager {
	if maxConcurrent <= 0 {
		maxConcurrent = 4
	}
	timeout := time.Duration(agentCfg.InitEventTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	requireBypass := true
	if agentCfg.RequireBypassPermissions != nil {
		requireBypass = *agentCfg.RequireBypassPermissions
	}
	m := &Manager{
		agents: agents,
		drivers: map[string]Driver{
			"claude-code-cli": &ClaudeCodeDriver{},
			"ollama":          &OllamaDriver{Instances: ollamaInstances},
		},
		sem:                make(chan struct{}, maxConcurrent),
		active:             make(map[string]*activeRun),
		idx:                idx,
		git:                git,
		hub:                h,
		locks:              locks,
		wf:                 wf,
		root:               root,
		logsDir:            logsDir,
		initEventTimeout:   timeout,
		requireBypassPerms: requireBypass,
		killer:             defaultProcessKiller{},
	}
	// Crash recovery: any run still marked running from a prior process is now failed.
	if err := idx.RecoverRunningRuns(); err != nil {
		slog.Warn("agent manager: error recovering running runs", "err", err)
	}
	// Crash recovery: reset orphaned test artifacts left in-qa from a prior crash.
	if err := m.recoverOrphanedTests(); err != nil {
		slog.Warn("agent manager: error recovering orphaned test artifacts", "err", err)
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
	prompt := strings.NewReplacer(
		"{target_path}", targetPath,
		"{related_test}", targetPath, // populated for test artifacts; harmless for others
	).Replace(promptTpl)

	// Determine lineage and previous status from the target artifact (if present in index).
	lineage := targetPath
	prevStatus := ""
	var targetArtifactType string
	if row, err := m.idx.Get(targetPath); err == nil && row != nil {
		lineage = row.FM.Lineage
		prevStatus = row.Status
		targetArtifactType = row.Type

		// Milestone 4: concurrent run guard for test artifacts.
		// If the test is already in-qa, reject immediately before acquiring any resources.
		if row.Type == "test" && row.Status == "in-qa" {
			return "", fmt.Errorf("test artifact is already in-qa; another QA run may be active")
		}
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

	// For test artifacts, set RelatedTestPath so the agent prompt can reference it.
	relatedTestPath := ""
	if targetArtifactType == "test" {
		relatedTestPath = targetPath
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
		RelatedTestPath:    relatedTestPath,
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
		// For test artifacts transitioning to in-qa, validate via the workflow
		// engine rather than bypassing it with the raw setArtifactStatus call.
		if targetArtifactType == "test" && ag.ActiveStatus == "in-qa" {
			if m.wf != nil && !m.wf.CanTransition(prevStatus, "in-qa", ag.Roles, "test") {
				_ = m.locks.Release(lineage)
				<-m.sem
				finishedAt := time.Now()
				runRow.Status = "failed"
				runRow.FinishedAt = &finishedAt
				_ = m.idx.UpdateAgentRun(runRow)
				return "", fmt.Errorf("workflow: transition %q → in-qa not permitted for test artifact %q (current status: %q)", prevStatus, targetPath, prevStatus)
			}
		}

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

	// runPrecheck forwards events to the hub AND watches for the system/init event.
	// It blocks until the precheck is resolved (init seen, timeout, or channel closed).
	precheckResult, observedMode := runPrecheck(
		proc.Progress(),
		m.initEventTimeout,
		m.requireBypassPerms,
		run.RunID,
		func(e hub.Event) { m.hub.Broadcast(e) },
		func() { m.killer.Kill(proc) },
	)

	if precheckResult == precheckFailedMode || precheckResult == precheckFailedTimeout {
		m.killAndFail(run, row, proc, lineage, precheckResult, observedMode)
		return
	}

	// Precheck passed (or was not applicable — process exited before init).
	// Wait for the process to fully exit.
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
	if status == "done" && run.TargetPath != "" {
		// For test artifacts, successful completion transitions back to approved (system role).
		// For all other artifacts, use the DoneOnSuccess flag as before.
		targetRow, _ := m.idx.Get(run.TargetPath)
		isTestArtifact := targetRow != nil && targetRow.Type == "test"

		if isTestArtifact {
			// Post-run: test artifact successfully completed QA → reset to approved.
			if err := m.setArtifactStatus(run.TargetPath, "approved"); err != nil {
				slog.Warn("agent: resetting test artifact to approved after successful QA run",
					"target", run.TargetPath, "err", err)
			} else {
				slog.Info("agent: test artifact reset to approved after successful QA run",
					"target", run.TargetPath, "run_id", run.RunID)
			}
		} else if run.DoneOnSuccess {
			if err := m.setArtifactStatus(run.TargetPath, "done"); err != nil {
				slog.Warn("agent: setting done status", "target", run.TargetPath, "err", err)
			}
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

// killAndFail handles the full precheck failure path:
//  1. Waits for the process to exit (the kill was already initiated by the
//     processKiller inside runPrecheck).
//  2. Releases the semaphore and lineage lock.
//  3. Appends a JSON precheck_failure line to the on-disk run log.
//  4. Updates the run record to status=failed.
//  5. Broadcasts agent.failed with precheck details and a remediation list.
//  6. Records a feed event.
func (m *Manager) killAndFail(
	run Run,
	row *index.AgentRunRow,
	proc Process,
	lineage string,
	state precheckState,
	observedMode string,
) {
	// Wait for the process to exit (kill was initiated by runPrecheck).
	_ = proc.Wait()

	m.mu.Lock()
	delete(m.active, run.RunID)
	m.mu.Unlock()
	<-m.sem

	var reason string
	var remediation []string
	switch state {
	case precheckFailedMode:
		reason = "permission_mode_default"
		if observedMode == "acceptEdits" {
			reason = "permission_mode_accept_edits"
		}
		remediation = modeRemediation
	case precheckFailedTimeout:
		reason = "precheck_timeout"
		remediation = timeoutRemediation
	default:
		reason = "precheck_unknown"
		remediation = modeRemediation
	}

	// Append precheck failure line to the on-disk run log (Milestone 4).
	if run.LogPath != "" {
		line := precheckFailureLogLine(run.RunID, reason, observedMode, remediation)
		if f, err := os.OpenFile(run.LogPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644); err == nil {
			_, _ = f.Write(line)
			_ = f.Close()
		}
	}

	// Update run record.
	finishedAt := time.Now()
	code := -1
	row.Status = "failed"
	row.FinishedAt = &finishedAt
	row.ExitCode = &code
	row.StderrTail = fmt.Sprintf("precheck failed: %s (observed_permission_mode=%q)", reason, observedMode)
	_ = m.idx.UpdateAgentRun(row)

	// Release lineage lock.
	_ = m.locks.Release(lineage)

	// Broadcast agent.failed with structured precheck payload.
	m.hub.Broadcast(hub.Event{
		Type: "agent.failed",
		Payload: map[string]any{
			"run_id":                   run.RunID,
			"agent":                    run.AgentName,
			"lineage":                  lineage,
			"status":                   "failed",
			"reason":                   reason,
			"observed_permission_mode": observedMode,
			"remediation":              remediation,
			"target_path":              row.TargetPath,
		},
	})

	// Record feed event.
	runIDCopy := run.RunID
	summary := fmt.Sprintf("Agent %s precheck failed: %s", run.AgentName, reason)
	feedEvent := &index.EventRow{
		EventType: "agent_failed",
		Timestamp: time.Now().Unix(),
		Actor:     run.AgentName,
		RunID:     &runIDCopy,
		Summary:   summary,
	}
	if err := m.idx.InsertEvent(feedEvent); err == nil {
		m.hub.Broadcast(hub.Event{Type: "feed.new", Payload: feedEvent})
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

// recoverOrphanedTests is called at startup to reset test artifacts that were
// left in in-qa status by a previous crash (i.e. there is no active agent run
// for them). Each recovered artifact is patched back to approved and committed.
func (m *Manager) recoverOrphanedTests() error {
	rows, _, err := m.idx.List(index.Filter{Type: "test", Status: "in-qa", Unlimited: true})
	if err != nil {
		return fmt.Errorf("listing in-qa test artifacts: %w", err)
	}
	if len(rows) == 0 {
		return nil
	}

	// Fetch running agent runs to distinguish orphans from legitimately active ones.
	activeRuns, err := m.idx.ListAgentRuns("running", 500)
	if err != nil {
		return fmt.Errorf("listing active agent runs: %w", err)
	}
	activeTargets := make(map[string]bool, len(activeRuns))
	for _, r := range activeRuns {
		activeTargets[r.TargetPath] = true
	}

	for _, row := range rows {
		if activeTargets[row.Path] {
			// Legitimate in-qa with an active run — leave it alone.
			continue
		}

		slog.Warn("agent: orphaned test artifact found in in-qa; resetting to approved",
			"path", row.Path, "lineage", row.Lineage)

		if err := m.setArtifactStatus(row.Path, "approved"); err != nil {
			slog.Warn("agent: failed to reset orphaned test artifact", "path", row.Path, "err", err)
			continue
		}

		if m.git != nil {
			authorName, authorEmail := m.git.ResolveIdentity()
			msg := fmt.Sprintf("recover(%s): in-qa → approved [orphan recovery]", row.FM.Lineage)
			_, _ = m.git.AddAndCommit([]string{row.Path}, msg, authorName, authorEmail)
		}
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
