// SPDX-License-Identifier: AGPL-3.0-or-later

package initcmd

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kaos-control/kaos-control/internal/auth"
	"github.com/kaos-control/kaos-control/internal/config"
	"golang.org/x/term"
)

// TemplateData is passed to all seed-file templates.
type TemplateData struct {
	ProjectName string
	Language    string
	// OwnerEmail, when non-empty, is written into lifecycle/config.yaml as the
	// initial owner user with the standard owner role set.
	OwnerEmail string
}

// ForceFlags controls which existing seed files may be overwritten.
type ForceFlags struct {
	Config    bool
	ClaudeMd  bool
	Settings  bool
	Gitignore bool
}

// Result records the outcome of creating or skipping one file or directory.
type Result struct {
	Path    string // relative path from the project root
	Created bool   // true = created/written, false = skipped
}

// Run is the entrypoint for the `kaos-control init` subcommand.
func Run(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)

	var (
		force              bool
		forceConfig        bool
		forceClaudeMd      bool
		forceSettings      bool
		forceGitignore     bool
		projectName        string
		language           string
		ownerEmail         string
		ownerName          string
		ownerPasswordStdin bool
	)

	fs.BoolVar(&force, "force", false, "overwrite all existing seed files")
	fs.BoolVar(&forceConfig, "force-config", false, "overwrite lifecycle/config.yaml if it exists")
	fs.BoolVar(&forceClaudeMd, "force-claude-md", false, "overwrite CLAUDE.md if it exists")
	fs.BoolVar(&forceSettings, "force-settings", false, "overwrite .claude/settings.json if it exists")
	fs.BoolVar(&forceGitignore, "force-gitignore", false, "overwrite .gitignore if it exists")
	fs.StringVar(&projectName, "project-name", "", "project name interpolated into CLAUDE.md (defaults to directory name)")
	fs.StringVar(&language, "language", "", "primary language hint for CLAUDE.md")
	fs.StringVar(&ownerEmail, "owner-email", "", "email of the initial owner user (creates account if absent)")
	fs.StringVar(&ownerName, "owner-name", "", "display name for the owner user (defaults to email)")
	fs.BoolVar(&ownerPasswordStdin, "owner-password-stdin", false, "read owner password from stdin instead of prompting")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	// Positional argument: path (defaults to current directory).
	targetPath := "."
	if fs.NArg() > 0 {
		targetPath = fs.Arg(0)
	}

	// Resolve to an absolute path and create it if absent.
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("resolving path %q: %w", targetPath, err)
	}
	if err := os.MkdirAll(absPath, 0o755); err != nil {
		return fmt.Errorf("creating directory %q: %w", absPath, err)
	}

	// Default project name to the directory basename.
	if projectName == "" {
		projectName = filepath.Base(absPath)
	}

	// --force implies all granular force flags.
	if force {
		forceConfig = true
		forceClaudeMd = true
		forceSettings = true
		forceGitignore = true
	}

	ff := ForceFlags{
		Config:    forceConfig,
		ClaudeMd:  forceClaudeMd,
		Settings:  forceSettings,
		Gitignore: forceGitignore,
	}

	data := TemplateData{
		ProjectName: projectName,
		Language:    language,
	}

	// If an owner email was supplied, create the user in the auth database and
	// populate OwnerEmail so the config template emits the users: section.
	if ownerEmail != "" {
		if err := createOwnerUser(ownerEmail, ownerName, ownerPasswordStdin); err != nil {
			return fmt.Errorf("creating owner user: %w", err)
		}
		data.OwnerEmail = ownerEmail
	}

	// Scaffold lifecycle directories.
	dirResults, err := scaffoldDirs(absPath)
	if err != nil {
		return fmt.Errorf("scaffolding directories: %w", err)
	}

	// Write seed files.
	fileResults, err := writeSeedFiles(absPath, data, ff)
	if err != nil {
		return fmt.Errorf("writing seed files: %w", err)
	}

	// Print summary (FR-7).
	fmt.Printf("Initialized kaos-control project at %s\n", absPath)
	for _, r := range dirResults {
		if r.Created {
			fmt.Printf("  created  %s\n", r.Path)
		} else {
			fmt.Printf("  skipped  %s (already exists)\n", r.Path)
		}
	}
	for _, r := range fileResults {
		if r.Created {
			fmt.Printf("  created  %s\n", r.Path)
		} else {
			fmt.Printf("  skipped  %s (already exists)\n", r.Path)
		}
	}
	if ownerEmail != "" {
		fmt.Printf("  owner    %s assigned roles: product-owner, analyst, reviewer, approver, devops\n", ownerEmail)
	}

	return nil
}

// createOwnerUser opens the auth database (using the default app config path),
// creates the user if they do not already exist, and marks them as admin.
// If the user already exists, the function is a no-op (idempotent).
func createOwnerUser(email, displayName string, passwordStdin bool) error {
	if displayName == "" {
		displayName = email
	}

	cfgPath := defaultAppConfigPath()
	appCfg, err := config.LoadApp(cfgPath)
	if err != nil {
		return fmt.Errorf("loading app config from %s: %w", cfgPath, err)
	}

	dbPath := filepath.Join(appCfg.DataDir, "auth.db")
	store, err := auth.Open(dbPath, appCfg.Auth.SessionTTL)
	if err != nil {
		return fmt.Errorf("opening auth db at %s: %w", dbPath, err)
	}
	defer store.Close()

	// Idempotent: if the user already exists, skip creation.
	existing, err := store.GetUser(email)
	if err != nil {
		return fmt.Errorf("looking up user %q: %w", email, err)
	}
	if existing != nil {
		fmt.Fprintf(os.Stderr, "note: user %q already exists; skipping creation\n", email)
		return nil
	}

	password, err := readOwnerPassword(passwordStdin)
	if err != nil {
		return fmt.Errorf("reading password: %w", err)
	}
	if password == "" {
		return fmt.Errorf("password must not be empty")
	}

	if err := store.CreateUser(email, displayName, password, true); err != nil {
		return fmt.Errorf("creating user %q: %w", email, err)
	}
	fmt.Printf("  user     %q created (admin=true)\n", email)
	return nil
}

// readOwnerPassword reads a password from stdin or prompts interactively.
func readOwnerPassword(fromStdin bool) (string, error) {
	if fromStdin {
		buf := make([]byte, 4096)
		n, err := os.Stdin.Read(buf)
		if err != nil && n == 0 {
			return "", err
		}
		return strings.TrimRight(string(buf[:n]), "\r\n"), nil
	}

	fmt.Fprint(os.Stderr, "Owner password: ")
	pw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return string(pw), nil
}

// defaultAppConfigPath returns the same default config path used by the serve
// and auth subcommands.
func defaultAppConfigPath() string {
	if base := os.Getenv("XDG_CONFIG_HOME"); base != "" {
		return filepath.Join(base, "kaos-control", "config.yaml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".kaos-control", "config.yaml")
}
