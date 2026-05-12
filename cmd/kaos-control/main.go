// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/kaos-control/kaos-control/cmd/kaos-control/authcmd"
	"github.com/kaos-control/kaos-control/internal/auth"
	"github.com/kaos-control/kaos-control/internal/config"
	khttp "github.com/kaos-control/kaos-control/internal/http"
	"github.com/kaos-control/kaos-control/internal/initcmd"
	"github.com/kaos-control/kaos-control/internal/project"
	"github.com/kaos-control/kaos-control/web"
)

var version = "dev"

const usage = `Usage: kaos-control <command> [flags]

Commands:
  serve    Start the HTTP server (default)
  init     Initialise a new project directory
  auth     Manage users, passwords, and API tokens

Run 'kaos-control <command> --help' for command-specific usage.
`

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
		case "serve":
			// Strip "serve" so the server's flag.Parse sees only its own flags.
			os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
		case "--help", "-help", "-h":
			fmt.Print(usage)
			os.Exit(0)
		default:
			if !strings.HasPrefix(os.Args[1], "-") {
				fmt.Fprintf(os.Stderr, "unknown subcommand %q\n\n%s", os.Args[1], usage)
				os.Exit(1)
			}
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

	// Open each project (loads config + opens/scans SQLite index).
	projects := make(map[string]*project.Project, len(entries))
	opts := project.OpenOptions{
		MaxConcurrentAgents:        appCfg.Limits.MaxConcurrentAgents,
		MaxConcurrentSchedulerJobs: appCfg.Limits.MaxConcurrentSchedulerJobs,
		SchedulerRunRetentionDays:  appCfg.Limits.SchedulerRunRetentionDays,
		OllamaInstances:            appCfg.OllamaInstances,
		AgentCfg:                   appCfg.Agent,
		// DevopsLogDir: store pipeline run logs at <appHome>/devops/<project>,
		// e.g. ~/.kaos-control/devops/<project>. This is the directory that
		// contains config.yaml, which is the app home directory.
		DevopsLogDir: filepath.Dir(cfgPath),
	}
	for _, e := range entries {
		slog.Info("opening project", "name", e.Name, "path", e.Path)
		p, err := project.Open(e, appCfg.DataDir, opts)
		if err != nil {
			// Log and skip — don't prevent other projects from loading.
			slog.Error("failed to open project", "name", e.Name, "err", err)
			continue
		}
		projects[e.Name] = p
	}

	// Start file watchers, lock reapers, session reapers, and the scheduler
	// (non-blocking; each runs until ctx is cancelled or the project is closed).
	for _, p := range projects {
		p.StartWatcher(ctx)
		p.StartLockReaper(ctx)
		p.StartSessionReaper(ctx)
		p.StartScheduler(ctx)
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

	srv := khttp.New(khttp.ServerConfig{
		Listen:     appCfg.Server.Listen,
		TLSOn:      appCfg.Server.TLS.Enabled,
		TLSCert:    appCfg.Server.TLS.CertFile,
		TLSKey:     appCfg.Server.TLS.KeyFile,
		Frontend:   web.FS,
		Auth:       authStore,
		AppCfg:     appCfg,
		AppCfgPath: cfgPath,
		PublicHost: publicHost,
	}, projects)

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
