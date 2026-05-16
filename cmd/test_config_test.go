package main

import (
	"fmt"
	"os"

	"github.com/kaos-control/kaos-control/internal/config"
)

func main() {
	cfg, err := config.LoadProject("/Users/keith/Code/kaos-control/tests/e2e/fixtures")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Agents count: %d\n", len(cfg.Agents))
	for _, a := range cfg.Agents {
		fmt.Printf("  Agent: %s, Roles: %v, Driver: %s\n", a.Name, a.Roles, a.Driver)
	}
}
