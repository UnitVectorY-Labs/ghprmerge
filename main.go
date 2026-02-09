package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/UnitVectorY-Labs/ghprmerge/internal/config"
	"github.com/UnitVectorY-Labs/ghprmerge/internal/github"
	"github.com/UnitVectorY-Labs/ghprmerge/internal/merger"
	"github.com/UnitVectorY-Labs/ghprmerge/internal/output"
)

// Version is set by the build system to the release version
var Version = "dev"

func main() {
	// Set the build version from the build info if not set by the build system
	if Version == "dev" || Version == "" {
		if bi, ok := debug.ReadBuildInfo(); ok {
			if bi.Main.Version != "" && bi.Main.Version != "(devel)" {
				Version = bi.Main.Version
			}
		}
	}

	if err := run(); err != nil {
		// Don't print error for help request
		if errors.Is(err, config.ErrHelp) {
			os.Exit(0)
		}
		// Don't print error for version request
		if errors.Is(err, config.ErrVersion) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Parse configuration
	cfg, err := config.ParseFlags(os.Args[1:], Version)
	if err != nil {
		return err
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return err
	}

	// Create GitHub client
	client := github.NewRealClient(cfg.Token)

	// Create merger with progress logging
	m := merger.New(client, cfg, os.Stderr)

	// Run merger
	ctx := context.Background()
	result, err := m.Run(ctx)
	if err != nil {
		return err
	}

	// If confirm mode is enabled and there are actions to take, prompt user
	if cfg.Confirm && hasActionsToPerform(result) {
		if !promptConfirmation(result) {
			fmt.Fprintln(os.Stderr, "Operation cancelled by user.")
			return nil
		}
		// Re-run with actions enabled
		result, err = m.RunWithActions(ctx, result)
		if err != nil {
			return err
		}
	}

	// Output results
	writer := output.NewWriter(os.Stdout, cfg.JSON, cfg.Quiet)
	return writer.WriteResult(result)
}

// hasActionsToPerform checks if the result contains actions that would be performed.
func hasActionsToPerform(result *output.RunResult) bool {
	return result.Summary.WouldMerge > 0 || result.Summary.WouldRebase > 0
}

// promptConfirmation displays a summary of planned actions and prompts for confirmation.
func promptConfirmation(result *output.RunResult) bool {
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "═══════════════════════════════════════════════════════════════════════════════")
	fmt.Fprintln(os.Stderr, "                           CONFIRMATION REQUIRED")
	fmt.Fprintln(os.Stderr, "═══════════════════════════════════════════════════════════════════════════════")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintf(os.Stderr, "  Actions to be performed:\n")
	if result.Summary.WouldMerge > 0 {
		fmt.Fprintf(os.Stderr, "    • Merge %d pull request(s)\n", result.Summary.WouldMerge)
	}
	if result.Summary.WouldRebase > 0 {
		fmt.Fprintf(os.Stderr, "    • Rebase/update %d pull request(s)\n", result.Summary.WouldRebase)
	}
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprint(os.Stderr, "  Do you want to proceed? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}
