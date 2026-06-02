// SPDX-License-Identifier: AGPL-3.0-or-later

// Package agent implements the agent runner: driver interface, CLI drivers,
// run lifecycle management, and scope enforcement.
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
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/kaos-control/kaos-control/internal/artifact"
	"github.com/kaos-control/kaos-control/internal/auth"
	"github.com/kaos-control/kaos-control/internal/config"
	kgit "github.com/kaos-control/kaos-control/internal/git"
	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/index"
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
	cp, ok := proc.(*cliProcess)
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
	Driver       string // "claude-code-cli", "codex-cli", "ollama", … — controls supervisor behaviour (e.g. precheck applicability)
	Model        string // empty → CLI default
	PromptText   string
	ProjectRoot  string
	AllowedPaths []string
	GitIdentity  config.GitIdentity
	LogPath      string // absolute path; if empty, no log file is written
	// Status lifecycle fields (copied from AgentConfig).
	TargetPath     string // project-relative path to the target artifact
	ActiveStatus   string // status to set on target when run starts (empty = no change)
	DoneOnSuccess  bool   // if true, set target status to "done" on successful completion
	TimeoutMinutes int    // 0 = driver default
	// RelatedTestPath is set when the target artifact is a test artifact.
	// It is passed to the agent prompt via the {related_test} placeholder so
	// the agent can reference the test in defect frontmatter (related_to field).
	RelatedTestPath string
	// Ollama-specific fields (only used when Driver == "ollama").
	OllamaInstanceName string // resolved from AgentConfig.OllamaInstanceName
	OllamaEndpoint     string // "chat" or "generate"
	ShellCommand       string // shell-stub driver: command to run (empty = default stub behavior)
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

type cliProcess struct {
	cmd      *exec.Cmd
	progress chan ProgressEvent
	stderr   *ringBuf
	logFile  *os.File // nil if no log path was configured

	// waitErr, when non-nil, makes Wait() return the value read from this
	// channel instead of calling cmd.Wait() directly. Drivers whose binary
	// detaches grandchildren that hold stdout/stderr open (agy via the
	// gemini-cli driver, codex via codex-cli) must call cmd.Wait()
	// asynchronously to close the pipes and stash the result here so
	// Wait() can be safely called later by the supervisor without
	// double-calling cmd.Wait.
	waitErr <-chan error
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
	return startCommandProcess(ctx, cmd, run, args, "claude")
}

// startCommandProcess is the shared subprocess launcher used by CLI drivers.
// It takes an already-configured exec.Cmd (with Dir and Env set by the caller)
// plus the arg list (only used for the log-file header) and returns a process
// with stdout/stderr piped, progress events streaming, and the log file open
// (FR3).
func startCommandProcess(_ context.Context, cmd *exec.Cmd, run Run, args []string, commandName string) (Process, error) {
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
		return nil, fmt.Errorf("starting %s: %w", commandName, err)
	}

	waitErr := make(chan error, 1)
	p := &cliProcess{cmd: cmd, progress: progressCh, stderr: rb, logFile: logFile, waitErr: waitErr}

	var wg sync.WaitGroup
	wg.Add(2)

	// Pipe stdout: tee to log file, parse each line as JSON, send progress events.
	go func() {
		defer wg.Done()
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
			progressCh <- ev
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

	// Watcher goroutine: cmd.Wait() reaps the process and closes the parent
	// pipe FDs, which unblocks the readers above even if a detached
	// grandchild is still holding the write ends. Stash the result so
	// proc.Wait() can return it without double-calling cmd.Wait.
	go func() {
		err := cmd.Wait()
		waitErr <- err
		close(waitErr)
	}()

	// Cleanup goroutine: wait for the readers to drain, write the run-log
	// footer, then close the progress channel so the supervisor exits.
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

func (p *cliProcess) Wait() error {
	if p.waitErr != nil {
		return <-p.waitErr
	}
	return p.cmd.Wait()
}
func (p *cliProcess) Progress() <-chan ProgressEvent { return p.progress }
func (p *cliProcess) StderrTail() string             { return p.stderr.String() }

func (p *cliProcess) Kill() error {
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

// DenialRecord stores details about a single denied tool call.
type DenialRecord struct {
	ToolName string `json:"tool_name"`
	Path     string `json:"path,omitempty"`
	Command  string `json:"command,omitempty"`
	Reason   string `json:"reason"`
	Rule     string `json:"rule"`
}

// Manager runs agents with lineage locking and a global concurrency semaphore.
type Manager struct {
	agents  []config.AgentConfig
	drivers map[string]Driver
	sem     chan struct{}

	mu     sync.Mutex
	active map[string]*activeRun

	// Per-run permission state (claude-mediated driver).
	runSecrets  map[string]string        // runID → per-run secret
	runPolicies map[string]*PolicyConfig // runID → policy config
	deniedCalls map[string][]DenialRecord

	// PauseQueue is an optional callback invoked when a run completes with
	// denied tool calls (FR16). Typically set to queueDispatcher.Pause().
	// nil means queue pausing is not configured.
	PauseQueue func(reason string)

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
// serverAddr is the listen address used by the HTTP server (for hook-helper).
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
		agents:             agents,
		sem:                make(chan struct{}, maxConcurrent),
		active:             make(map[string]*activeRun),
		runSecrets:         make(map[string]string),
		runPolicies:        make(map[string]*PolicyConfig),
		deniedCalls:        make(map[string][]DenialRecord),
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

	// Build hook driver with a StoreSecret callback into this Manager.
	hookDriver := &ClaudeHooksDriver{
		StoreSecret: m.StoreRunSecret,
	}
	m.drivers = map[string]Driver{
		"claude-code-cli": &ClaudeCodeDriver{},
		"claude-mediated": hookDriver,
		"codex-cli":       &CodexCLIDriver{},
		"ollama":          &OllamaDriver{Instances: ollamaInstances},
		"gemini":          &GeminiDriver{},
		"gemini-cli":      &GeminiCliDriver{},
		"shell-stub":      &ShellStubDriver{},
	}
	// Crash recovery: any run still marked running from a prior process is now failed.
	if err := idx.RecoverRunningRuns(); err != nil {
		slog.Warn("agent manager: error recovering running runs", "err", err)
	}
	// Crash recovery: reset orphaned test artifacts left in-qa from a prior crash.
	if err := m.recoverOrphanedTests(); err != nil {
		slog.Warn("agent manager: error recovering orphaned test artifacts", "err", err)
	}

	// Non-blocking version check: warn if Claude Code is below the minimum
	// version required for hooks API support (NFR5).
	go checkClaudeVersion()

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
		Driver:             ag.Driver,
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
		ShellCommand:       ag.ShellCommand,
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

	// For claude-mediated runs, build and store the PolicyConfig before
	// starting the driver so the permission endpoint is ready immediately.
	if ag.Driver == "claude-mediated" {
		bashDenylist := mergeDenylist(ag.BashDenylist)
		policy := &PolicyConfig{
			ProjectRoot:   m.root,
			AllowedPaths:  ag.AllowedPaths,
			BashAllowlist: ag.BashAllowlist,
			BashDenylist:  bashDenylist,
			ObserveOnly:   ag.ObserveOnly,
		}
		m.StoreRunPolicy(runID, policy)
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

	// Event-forwarding closure shared by the precheck (Claude) and the plain
	// drain loop (Ollama and any other non-Claude driver). Also detects
	// rate-limit payloads (M4) and re-broadcasts them as queue.rate_limit so
	// the queue dispatcher can pause and re-enqueue the job.
	broadcast := func(e hub.Event) {
		m.hub.Broadcast(e)
		if e.Type == "agent.progress" {
			if payload, ok := e.Payload.(map[string]any); ok {
				if rawText, isRL := extractRateLimitText(payload); isRL {
					m.hub.Broadcast(hub.Event{
						Type: "queue.rate_limit",
						Payload: map[string]any{
							"run_id":   run.RunID,
							"raw_text": rawText,
						},
					})
				}
			}
		}
	}

	// Branch on driver type for the init-event precheck.
	switch run.Driver {
	case "claude-code-cli":
		// Full bypass-permissions check: fail if mode is not bypassPermissions.
		precheckResult, observedMode := runPrecheck(
			proc.Progress(),
			m.initEventTimeout,
			m.requireBypassPerms,
			run.RunID,
			broadcast,
			func() { m.killer.Kill(proc) },
		)
		if precheckResult == precheckFailedMode || precheckResult == precheckFailedTimeout {
			m.killAndFail(run, row, proc, lineage, precheckResult, observedMode)
			return
		}

	case "claude-mediated":
		// Mediated precheck: fail if mode IS bypassPermissions (hooks not active).
		precheckResult, observedMode := runMediatedPrecheck(
			proc.Progress(),
			m.initEventTimeout,
			run.RunID,
			broadcast,
			func() { m.killer.Kill(proc) },
		)
		if precheckResult == precheckFailedMode || precheckResult == precheckFailedTimeout {
			m.killAndFail(run, row, proc, lineage, precheckResult, observedMode)
			return
		}

	default:
		// Non-Claude drivers: just drain and forward events.
		for ev := range proc.Progress() {
			payload := map[string]any{
				"run_id": run.RunID,
				"line":   ev.Raw,
				"raw":    ev.Raw,
			}
			if ev.Event != nil {
				payload["event"] = ev.Event
			}
			broadcast(hub.Event{Type: "agent.progress", Payload: payload})
		}
	}

	// Precheck passed (or was not applicable — process exited before init).
	// Wait for the process to fully exit.
	waitErr := proc.Wait()
	finishedAt := time.Now()

	m.mu.Lock()
	_, wasActive := m.active[run.RunID]
	delete(m.active, run.RunID)
	m.mu.Unlock()

	// Release per-run secret and policy state (denial records handled later).
	m.cleanupRunState(run.RunID)

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

	// Check for denied tool calls from claude-mediated runs (FR15/FR16).
	denials := m.DeniedCalls(run.RunID)
	hasDenials := len(denials) > 0

	// Commit any files produced within allowed paths — unless the run had
	// denied tool calls, in which case we skip auto-commit (FR15).
	var produced []string
	if m.git != nil && !hasDenials {
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

	// Persist denial records in the run row (FR21).
	if hasDenials {
		denialMaps := make([]map[string]any, len(denials))
		for i, d := range denials {
			denialMaps[i] = map[string]any{
				"tool_name": d.ToolName,
				"path":      d.Path,
				"command":   d.Command,
				"reason":    d.Reason,
				"rule":      d.Rule,
			}
		}
		row.DeniedToolCalls = denialMaps
	}
	m.clearDeniedCalls(run.RunID)

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

	// Pause queue if there were denials (FR16).
	if hasDenials && m.PauseQueue != nil {
		m.PauseQueue("denied_tool_calls: run " + run.RunID)
	}

	eventType := "agent.finished"
	feedEventType := "agent_finished"
	if status == "failed" || status == "killed" || status == "killed-timeout" {
		eventType = "agent.failed"
		feedEventType = "agent_failed"
	}

	// Attempt to parse the run result from the log and include it in the
	// broadcast payload. Parsing errors are non-fatal — the broadcast proceeds
	// with result: null (expected for Ollama runs or logs without a result line).
	var runResult any
	if run.LogPath != "" {
		if logData, readErr := os.ReadFile(run.LogPath); readErr == nil {
			if parsed, parseErr := ParseResultLine(string(logData)); parseErr == nil {
				runResult = parsed
			}
		}
	}

	m.hub.Broadcast(hub.Event{
		Type: eventType,
		Payload: map[string]any{
			"run_id":            run.RunID,
			"agent":             run.AgentName,
			"lineage":           lineage,
			"status":            status,
			"artifacts":         produced,
			"target_path":       row.TargetPath,
			"result":            runResult,
			"denied_tool_calls": row.DeniedToolCalls,
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
		if run.Driver == "claude-mediated" && observedMode == "bypassPermissions" {
			reason = "precheck_mediated_bypass"
			remediation = mediatedBypassRemediation
		} else {
			reason = "permission_mode_default"
			if observedMode == "acceptEdits" {
				reason = "permission_mode_accept_edits"
			}
			remediation = modeRemediation
		}
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

// StoreRunSecret records the per-run secret generated by the claude-mediated
// driver so the permission endpoint can validate incoming requests.
func (m *Manager) StoreRunSecret(runID, secret string) {
	m.mu.Lock()
	m.runSecrets[runID] = secret
	m.mu.Unlock()
}

// StoreRunPolicy records the PolicyConfig for an active claude-mediated run.
func (m *Manager) StoreRunPolicy(runID string, cfg *PolicyConfig) {
	m.mu.Lock()
	m.runPolicies[runID] = cfg
	m.mu.Unlock()
}

// ValidateRunSecret reports whether secret matches the stored per-run secret
// for runID. Returns false if the run is unknown.
func (m *Manager) ValidateRunSecret(runID, secret string) bool {
	m.mu.Lock()
	stored, ok := m.runSecrets[runID]
	m.mu.Unlock()
	return ok && stored != "" && stored == secret
}

// PolicyForRun returns the PolicyConfig for the given run, or an error if the
// run is unknown or has no policy (e.g. non-mediated driver).
func (m *Manager) PolicyForRun(runID string) (*PolicyConfig, error) {
	m.mu.Lock()
	p, ok := m.runPolicies[runID]
	m.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("no policy for run %q", runID)
	}
	return p, nil
}

// RecordDenial appends a denial record for the given run.
func (m *Manager) RecordDenial(runID string, d Decision, toolName string, toolInput map[string]any) {
	rec := DenialRecord{
		ToolName: toolName,
		Reason:   d.Reason,
		Rule:     d.Rule,
	}
	if p, _ := toolInput["file_path"].(string); p != "" {
		rec.Path = p
	}
	if c, _ := toolInput["command"].(string); c != "" {
		rec.Command = c
	}
	m.mu.Lock()
	m.deniedCalls[runID] = append(m.deniedCalls[runID], rec)
	m.mu.Unlock()
}

// DeniedCalls returns a copy of all denial records for the given run.
func (m *Manager) DeniedCalls(runID string) []DenialRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	src := m.deniedCalls[runID]
	if len(src) == 0 {
		return nil
	}
	out := make([]DenialRecord, len(src))
	copy(out, src)
	return out
}

// cleanupRunState removes all per-run secret/policy/denial state for runID.
// Called by supervise() after the run completes.
func (m *Manager) cleanupRunState(runID string) {
	m.mu.Lock()
	delete(m.runSecrets, runID)
	delete(m.runPolicies, runID)
	// deniedCalls is retained until after supervise reads it; caller is
	// responsible for deleting it once the data has been persisted.
	m.mu.Unlock()
}

// clearDeniedCalls removes denial state after it has been consumed.
func (m *Manager) clearDeniedCalls(runID string) {
	m.mu.Lock()
	delete(m.deniedCalls, runID)
	m.mu.Unlock()
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

// mergeDenylist returns the union of DefaultBashDenylist and perAgent entries
// (duplicates are harmless — each pattern is checked independently).
func mergeDenylist(perAgent []string) []string {
	merged := make([]string, len(DefaultBashDenylist), len(DefaultBashDenylist)+len(perAgent))
	copy(merged, DefaultBashDenylist)
	return append(merged, perAgent...)
}

// MinClaudeVersion is the minimum Claude Code version required for the hooks
// API (PreToolUse) used by the claude-mediated driver (NFR5).
const MinClaudeVersion = "1.9.0"

// checkClaudeVersion runs `claude --version`, parses the output, and logs a
// warning if the version is below MinClaudeVersion. It is called in a
// goroutine so it never blocks startup.
func checkClaudeVersion() {
	out, err := exec.Command("claude", "--version").Output()
	if err != nil {
		slog.Warn("agent: could not determine Claude Code version; hooks may be unavailable",
			"err", err, "min_version", MinClaudeVersion)
		return
	}
	verStr := strings.TrimSpace(string(out))
	if compareVersions(verStr, MinClaudeVersion) < 0 {
		slog.Warn("agent: Claude Code version may not support hooks API",
			"detected", verStr, "min_required", MinClaudeVersion,
			"hint", "upgrade Claude Code to enable the claude-mediated driver")
	} else {
		slog.Debug("agent: Claude Code version check passed", "version", verStr, "min_required", MinClaudeVersion)
	}
}

// compareVersions compares two "MAJOR.MINOR.PATCH" version strings.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
// Non-numeric version strings (e.g. "claude 1.9.0") are parsed leniently —
// we scan for the first numeric segment.
func compareVersions(a, b string) int {
	return cmpSemver(parseVersion(a), parseVersion(b))
}

func parseVersion(s string) [3]int {
	// Find first digit in the string to handle "claude/1.9.0" or "Claude Code 1.9.0".
	start := strings.IndexAny(s, "0123456789")
	if start < 0 {
		return [3]int{}
	}
	s = s[start:]
	// Trim anything after the first space or non-semver character.
	if idx := strings.IndexAny(s, " \t\n"); idx >= 0 {
		s = s[:idx]
	}
	var v [3]int
	parts := strings.SplitN(s, ".", 3)
	for i, p := range parts {
		if i >= 3 {
			break
		}
		n := 0
		for _, c := range p {
			if c < '0' || c > '9' {
				break
			}
			n = n*10 + int(c-'0')
		}
		// #nosec G602 -- guarded by `if i >= 3 { break }` above; v is [3]int
		v[i] = n
	}
	return v
}

func cmpSemver(a, b [3]int) int {
	for i := 0; i < 3; i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

// ConfigureHookDriver sets the ServerAddr and BinaryPath on the
// ClaudeHooksDriver registered under the "claude-mediated" key. Must be called
// before any claude-mediated runs are started. Typically invoked after New()
// once the HTTP server's listen address is known.
func (m *Manager) ConfigureHookDriver(serverAddr, binaryPath string) {
	if drv, ok := m.drivers["claude-mediated"]; ok {
		if hd, ok := drv.(*ClaudeHooksDriver); ok {
			hd.ServerAddr = serverAddr
			hd.BinaryPath = binaryPath
		}
	}
}

// extractRateLimitText inspects a decoded agent.progress event payload and
// returns (rawText, true) when the underlying stream event signals a
// rate-limit / quota-exhausted error. It handles three Claude Code stream-json
// shapes:
//
//  1. Top-level "error" == "rate_limit" with message.content[0].text
//     {"error":"rate_limit","message":{"content":[{"type":"text","text":"..."}]}}
//
//  2. Nested error object with type containing "rate_limit"
//     {"type":"error","error":{"type":"rate_limit_error","message":"..."}}
//
//  3. Terminal "type":"result" event with is_error:true and a usage-exhausted
//     message in the "result" field. The model exited normally but the API
//     replied with a quota message:
//     {"type":"result","is_error":true,
//     "result":"You're out of extra usage · resets 11:10pm (Australia/Brisbane)"}
//
// The raw text is used by the queue dispatcher to parse the reset time; an
// empty raw_text causes the dispatcher to fall back to the configured
// fallback_pause_minutes.
func extractRateLimitText(payload map[string]any) (rawText string, ok bool) {
	ev, _ := payload["event"].(map[string]any)
	if ev == nil {
		return "", false
	}

	// Format 1: top-level error field is the string "rate_limit".
	if errStr, _ := ev["error"].(string); errStr == "rate_limit" {
		rawText = extractMessageText(ev)
		return rawText, true
	}

	// Format 2: nested error object whose type contains "rate_limit".
	if errObj, ok2 := ev["error"].(map[string]any); ok2 {
		errType, _ := errObj["type"].(string)
		if strings.Contains(errType, "rate_limit") {
			// Prefer nested message text; fall back to the error message string.
			rawText = extractMessageText(ev)
			if rawText == "" {
				rawText, _ = errObj["message"].(string)
			}
			return rawText, true
		}
	}

	// Format 3: terminal "result" event with is_error:true that carries a
	// quota-exhausted message in the result field. The run completed without
	// emitting a rate_limit stream event; the quota verdict is only visible
	// in the wrap-up result line. Match the result text against a small set
	// of phrases that indicate a usage/quota condition (not every is_error
	// result is a rate limit — e.g. a generic API error shouldn't be
	// re-enqueued).
	if evType, _ := ev["type"].(string); evType == "result" {
		if isErr, _ := ev["is_error"].(bool); isErr {
			result, _ := ev["result"].(string)
			if result != "" && looksLikeQuotaExhausted(result) {
				return result, true
			}
		}
	}

	return "", false
}

// looksLikeQuotaExhausted returns true when text contains a phrase that
// indicates a rate-limit, quota, or transient-overload condition that should
// trigger a queue pause-and-retry. Matched case-insensitively.
//
// Conservative on purpose: only matches phrases that signal "retry the
// same job later". Generic API errors (4xx that aren't 429, 5xx that isn't
// 529, malformed responses) are NOT matched, so they fail through as hard
// failures.
//
// Patterns covered:
//   - Quota / usage exhausted: "out of usage", "usage … resets", "message limit",
//     "exceeded … quota|limit"
//   - Rate limited: "rate limit", "rate-limit" — and HTTP 429
//   - Anthropic 529 "Overloaded" — transient server-overload, retry-after.
//     Includes both the structured "overloaded_error" type and the bare word.
var quotaExhaustedRE = regexp.MustCompile(`(?i)(out of (extra )?usage|usage[\s\S]*resets?|rate.?limit|message limit|exceeded[\s\S]{0,40}(quota|limit)|overloaded|\b(429|529)\b)`)

func looksLikeQuotaExhausted(text string) bool {
	return quotaExhaustedRE.MatchString(text)
}

// extractMessageText attempts to read a human-readable string from
// ev["message"]["content"][0]["text"], as used in the Claude stream-json format.
func extractMessageText(ev map[string]any) string {
	msg, _ := ev["message"].(map[string]any)
	if msg == nil {
		return ""
	}
	content, _ := msg["content"].([]any)
	if len(content) == 0 {
		return ""
	}
	first, _ := content[0].(map[string]any)
	text, _ := first["text"].(string)
	return text
}
