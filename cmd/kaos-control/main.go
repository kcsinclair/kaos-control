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
	for _, e := range entries {
		slog.Info("opening project", "name", e.Name, "path", e.Path)
		p, err := project.Open(e, appCfg.DataDir)
		if err != nil {
			// Log and skip — don't prevent other projects from loading.
			slog.Error("failed to open project", "name", e.Name, "err", err)
			continue
		}
		projects[e.Name] = p
	}

	// Start file watchers (non-blocking; each runs until ctx is cancelled).
	for _, p := range projects {
		p.StartWatcher(ctx)
	}

	srv := khttp.New(khttp.ServerConfig{
		Listen:   appCfg.Server.Listen,
		TLSOn:    appCfg.Server.TLS.Enabled,
		TLSCert:  appCfg.Server.TLS.CertFile,
		TLSKey:   appCfg.Server.TLS.KeyFile,
		Frontend: web.FS,
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
