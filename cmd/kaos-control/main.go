// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/kaos-control/kaos-control/cmd/kaos-control/authcmd"
	"github.com/kaos-control/kaos-control/cmd/kaos-control/hookcmd"
	"github.com/kaos-control/kaos-control/internal/auth"
	"github.com/kaos-control/kaos-control/internal/backfillcmd"
	"github.com/kaos-control/kaos-control/internal/config"
	khttp "github.com/kaos-control/kaos-control/internal/http"
	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/initcmd"
	"github.com/kaos-control/kaos-control/internal/project"
	"github.com/kaos-control/kaos-control/internal/queue"
	"github.com/kaos-control/kaos-control/web"
)

var version = "dev"

const usage = `Usage: kaos-control <command> [flags]

Commands:
  serve              Start the HTTP server (default)
  init               Initialise a new project directory
  auth               Manage users, passwords, and API tokens
  hook-helper        PreToolUse hook helper (called by Claude Code)
  backfill-created   Add created: frontmatter to legacy artefacts using
                     filesystem birth time

Flags:
  --version, -V      Print version, copyright, and licence
  --help, -h         Print this usage banner
  -config <path>     Path to app config.yaml (serve only)

Run 'kaos-control <command> --help' for command-specific usage.
`

// printVersion writes the three-line release header to stdout: project + URL,
// copyright, licence. Called from the --version / -V CLI flag.
func printVersion() {
	fmt.Printf("kaos-control %s      https://github.com/kcsinclair/kaos-control\n", version)
	fmt.Println("Copyright (c) 2026 Keith Sinclair <keith@sinclair.org.au>")
	fmt.Println("Licensed under the GNU AGPL v3.0 or later.")
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "init":
			if err := initcmd.Run(os.Args[2:]); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			return
		case "auth":
			os.Exit(authcmd.Run(os.Args[2:]))
		case "hook-helper":
			hookcmd.Run(os.Args[2:])
			return
		case "backfill-created":
			if err := backfillcmd.Run(os.Args[2:]); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			return
		case "serve":
			// Strip "serve" so the server's flag.Parse sees only its own flags.
			os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
		case "--help", "-help", "-h":
			fmt.Print(usage)
			os.Exit(0)
		case "--version", "-version", "-V":
			printVersion()
			os.Exit(0)
		default:
			arg := os.Args[1]
			if strings.HasPrefix(arg, "-") {
				// Allow -config / --config (with or without =value) to fall
				// through to flag.Parse() in run() — that's the implicit
				// `serve -config /path` form.
				if arg == "-config" || arg == "--config" ||
					strings.HasPrefix(arg, "-config=") ||
					strings.HasPrefix(arg, "--config=") {
					break
				}
				fmt.Fprintf(os.Stderr, "unknown flag %q\n\n%s", arg, usage)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "unknown subcommand %q\n\n%s", arg, usage)
			os.Exit(1)
		}
	}
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	setupLogging()

	var cfgPath string
	flag.StringVar(&cfgPath, "config", defaultConfigPath(), "path to app config.yaml")
	flag.Parse()

	khttp.Version = version

	appCfg, err := config.LoadApp(cfgPath)
	if err != nil {
		return fmt.Errorf("loading config from %s: %w", cfgPath, err)
	}

	// Load the project registry.
	entries, err := config.LoadProjectRegistry(appCfg.ProjectsDir)
	if err != nil {
		return fmt.Errorf("loading project registry from %s: %w", appCfg.ProjectsDir, err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Bind the TCP listener before opening projects so the port is reachable
	// from the moment startup begins. Index scans on large repos can take
	// time; previously the UI looked hung because nothing was listening yet.
	// httpSrv.Serve isn't called until after projects are opened, so requests
	// land in the OS accept queue and complete once Serve runs.
	ln, err := net.Listen("tcp", appCfg.Server.Listen)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", appCfg.Server.Listen, err)
	}
	slog.Info("listener bound; opening projects", "addr", ln.Addr().String())

	// Resolve the binary path once so hook-helper invocations are stable.
	selfBinary, err := os.Executable()
	if err != nil {
		slog.Warn("could not resolve binary path; claude-mediated agents will use os.Executable() at run time", "err", err)
		selfBinary = ""
	}

	// Build OpenOptions once; reused for runtime project registration.
	opts := project.OpenOptions{
		MaxConcurrentAgents:        appCfg.Limits.MaxConcurrentAgents,
		MaxConcurrentSchedulerJobs: appCfg.Limits.MaxConcurrentSchedulerJobs,
		SchedulerRunRetentionDays:  appCfg.Limits.SchedulerRunRetentionDays,
		OllamaInstances:            appCfg.OllamaInstances,
		AgentCfg:                   appCfg.Agent,
		// DevopsLogDir: store pipeline run logs at <appHome>/devops/<project>,
		// e.g. ~/.kaos-control/devops/<project>. This is the directory that
		// contains config.yaml, which is the app home directory.
		DevopsLogDir:   filepath.Dir(cfgPath),
		HookServerAddr: appCfg.Server.Listen,
		HookBinaryPath: selfBinary,
	}

	// Open each project (loads config + opens/scans SQLite index).
	// Each project gets its own derived context so that UnregisterProject can
	// cancel only that project's goroutines without affecting others.
	projects := make(map[string]*project.Project, len(entries))
	startupCancels := make(map[string]context.CancelFunc, len(entries))
	for _, e := range entries {
		slog.Info("opening project", "name", e.Name, "path", e.Path)
		p, err := project.Open(e, appCfg.DataDir, opts)
		if err != nil {
			// Log and skip — don't prevent other projects from loading.
			slog.Error("failed to open project", "name", e.Name, "err", err)
			continue
		}
		projects[e.Name] = p
		// #nosec G118 -- cancel is stored in startupCancels and handed to srv.TrackCancel below
		pCtx, cancel := context.WithCancel(ctx)
		startupCancels[e.Name] = cancel
		p.StartWatcher(pCtx)
		p.StartLockReaper(pCtx)
		p.StartSessionReaper(pCtx)
		p.StartScheduler(pCtx)
	}

	// Open the queue database (app-level, shared across all projects).
	queueStore, err := queue.Open(filepath.Join(appCfg.DataDir, "queue.db"))
	if err != nil {
		slog.Warn("failed to open queue database; agent work queue will be unavailable", "err", err)
		queueStore = nil
	}
	if queueStore != nil {
		if err := queueStore.RecoverOrphans(); err != nil {
			slog.Warn("queue: orphan recovery failed", "err", err)
		}
	}

	// Open the auth database (accounts + sessions, shared across projects).
	authStore, err := auth.Open(
		filepath.Join(appCfg.DataDir, "auth.db"),
		appCfg.Auth.SessionTTL,
	)
	if err != nil {
		slog.Warn("failed to open auth database; authentication will be unavailable", "err", err)
		authStore = nil
	}
	if authStore != nil {
		if count, cerr := authStore.UserCount(); cerr == nil && count == 0 {
			slog.Warn("No users found. Create the first admin user with: kaos-control auth create-user --email <email> --name <name> --admin")
		}
	}

	publicHost := appCfg.Server.PublicHost
	if publicHost == "" {
		publicHost = os.Getenv("KAOS_PUBLIC_HOST")
	}

	// Build the app-level hub for queue broadcast events.
	appHub := hub.New()

	// Build the server first so that the queue's project lookup can use
	// srv.GetProject — a mutex-protected accessor — rather than reading
	// the raw projects map directly. This eliminates data races between
	// concurrent project CRUD requests and queue dispatch.
	srv := khttp.New(khttp.ServerConfig{
		Listen:     appCfg.Server.Listen,
		Listener:   ln, // pre-bound; see net.Listen above
		TLSOn:      appCfg.Server.TLS.Enabled,
		TLSCert:    appCfg.Server.TLS.CertFile,
		TLSKey:     appCfg.Server.TLS.KeyFile,
		Frontend:   web.FS,
		Auth:       authStore,
		AppCfg:     appCfg,
		AppCfgPath: cfgPath,
		PublicHost: publicHost,
		AppHub:     appHub,
		// Queue wired below after creation.
		ProjectsDir:        appCfg.ProjectsDir,
		DataDir:            appCfg.DataDir,
		ProjectOpenOptions: opts,
	}, projects)

	// Register cancel functions for startup projects so that UnregisterProject
	// can cleanly stop their goroutines at runtime.
	for name, cancel := range startupCancels {
		srv.TrackCancel(name, cancel)
	}

	// Build the queue dispatcher after the server so the lookup closure can go
	// through the server's mutex-protected GetProject.
	var queueDispatcher *queue.Dispatcher
	if queueStore != nil {
		projectLookup := func(name string) (queue.ProjectAccess, bool) {
			p, ok := srv.GetProject(name)
			if !ok || p.Agents == nil {
				return queue.ProjectAccess{}, false
			}
			return queue.ProjectAccess{
				StartRun: func(runCtx context.Context, agentName, targetPath string) (string, error) {
					return p.Agents.StartRun(runCtx, agentName, targetPath, "", nil)
				},
				ArtifactStatus: func(relPath string) string {
					row, err := p.Idx.Get(relPath)
					if err != nil || row == nil {
						return ""
					}
					return row.Status
				},
				Hub: p.Hub,
			}, true
		}
		queueDispatcher = queue.New(queueStore, projectLookup, appHub, queue.Config{})
		queueDispatcher.Start(ctx)
		// Wire PauseQueue into startup projects and into the server so that
		// future RegisterProject calls also wire new projects automatically.
		for _, p := range projects {
			if p.Agents != nil {
				d := queueDispatcher
				p.Agents.PauseQueue = func(reason string) { d.Pause(reason) }
			}
		}
		srv.SetQueue(queueDispatcher)
	}

	return srv.ListenAndServe(ctx)
}

func setupLogging() {
	level := slog.LevelInfo
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		var l slog.Level
		if err := l.UnmarshalText([]byte(v)); err == nil {
			level = l
		}
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))
}

func defaultConfigPath() string {
	if base := os.Getenv("XDG_CONFIG_HOME"); base != "" {
		return filepath.Join(base, "kaos-control", "config.yaml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".kaos-control", "config.yaml")
}
