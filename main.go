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

	// Create console for terminal output (nil if JSON mode)
	var console *output.Console
	if !cfg.JSON {
		console = output.NewConsole(os.Stderr, cfg.NoColor, cfg.Verbose)
	}

	// Create merger with console
	m := merger.New(client, cfg, console)

	// Run merger
	ctx := context.Background()
	result, err := m.Run(ctx)
	if err != nil {
		return err
	}

	// If confirm mode is enabled and there are actions to take, prompt user
	if cfg.Confirm && hasActionsToPerform(result) {
		// Count pending action lines for clearing later
		pendingLines := countPendingLines(result)

		if !promptConfirmation(console, result) {
			if console != nil {
				fmt.Fprintln(os.Stderr, console.Dim("Operation cancelled by user."))
			} else {
				fmt.Fprintln(os.Stderr, "Operation cancelled by user.")
			}
			return nil
		}

		// Clear the pending actions and prompt from the terminal
		if console != nil {
			// +3 for the prompt line, blank line before prompt, and the user's input line
			console.ClearLines(pendingLines + 3)
		}

		// Re-run with actions enabled
		result, err = m.RunWithActions(ctx, result)
		if err != nil {
			return err
		}
	}

	// Output results (condensed summary for human mode, full JSON for JSON mode)
	writer := output.NewWriter(os.Stdout, cfg.JSON, cfg.Quiet, cfg.NoColor)
	return writer.WriteResult(result)
}

// hasActionsToPerform checks if the result contains actions that would be performed.
func hasActionsToPerform(result *output.RunResult) bool {
	return result.Summary.WouldMerge > 0 || result.Summary.WouldRebase > 0
}

// countPendingLines counts the number of terminal lines used for pending action display.
func countPendingLines(result *output.RunResult) int {
	lines := 0
	for _, repo := range result.Repositories {
		for _, pr := range repo.PullRequests {
			if pr.Action == output.ActionWouldMerge || pr.Action == output.ActionWouldRebase {
				lines += 2 // symbol line + action detail line
			}
		}
	}
	if lines > 0 {
		lines++ // "Pending actions:" header
	}
	return lines
}

// promptConfirmation displays pending actions and prompts for confirmation.
func promptConfirmation(console *output.Console, result *output.RunResult) bool {
	if console != nil {
		fmt.Fprintln(os.Stderr, console.Bold("Pending actions:"))
		for _, repo := range result.Repositories {
			for _, pr := range repo.PullRequests {
				if pr.Action == output.ActionWouldMerge || pr.Action == output.ActionWouldRebase {
					console.PrintPendingAction(repo, pr)
				}
			}
		}
	} else {
		fmt.Fprintln(os.Stderr, "Pending actions:")
		for _, repo := range result.Repositories {
			for _, pr := range repo.PullRequests {
				if pr.Action == output.ActionWouldMerge || pr.Action == output.ActionWouldRebase {
					fmt.Fprintf(os.Stderr, "  %s #%d %s ─ %s\n", repo.FullName, pr.Number, pr.Title, pr.Action)
				}
			}
		}
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprint(os.Stderr, "Do you want to proceed? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}
