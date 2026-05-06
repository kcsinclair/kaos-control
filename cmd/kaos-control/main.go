package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/kaos-control/kaos-control/internal/auth"
	"github.com/kaos-control/kaos-control/internal/config"
	khttp "github.com/kaos-control/kaos-control/internal/http"
	"github.com/kaos-control/kaos-control/internal/project"
	"github.com/kaos-control/kaos-control/web"
)

var version = "dev"

func main() {
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

	srv := khttp.New(khttp.ServerConfig{
		Listen:   appCfg.Server.Listen,
		TLSOn:    appCfg.Server.TLS.Enabled,
		TLSCert:  appCfg.Server.TLS.CertFile,
		TLSKey:   appCfg.Server.TLS.KeyFile,
		Frontend: web.FS,
		Auth:     authStore,
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
