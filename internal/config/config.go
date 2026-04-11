// Package config handles configuration and command line argument parsing for ghprmerge.
package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// StringSliceFlag is a custom flag type that collects multiple string values.
type StringSliceFlag []string

func (s *StringSliceFlag) String() string {
	return strings.Join(*s, ", ")
}

func (s *StringSliceFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

// Command represents the subcommand to execute.
type Command string

const (
	CommandNone   Command = ""
	CommandMerge  Command = "merge"
	CommandRebase Command = "rebase"
	CommandReport Command = "report"
)

// Config holds all configuration for ghprmerge.
type Config struct {
	Org                string
	SourceBranches     []string
	SourceBranch       string // First source branch (for backward compat in merger)
	Rebase             bool
	Merge              bool
	SkipRebase         bool
	Repos              []string
	RepoLimit          int
	JSON               bool
	Confirm            bool
	Verbose            bool
	NoColor            bool
	Token              string
	Report             bool
	SourceBranchPrefix []string
	MinGroupSize       int
	Verbosity          string
	Command            Command
}

// IsAnalysisOnly returns true if neither rebase nor merge subcommand is used.
func (c *Config) IsAnalysisOnly() bool {
	return !c.Rebase && !c.Merge
}

// Validate checks that all required configuration is present.
func (c *Config) Validate() error {
	if c.Org == "" {
		return fmt.Errorf("--org is required (or set GITHUB_ORG environment variable)")
	}
	if c.Token == "" {
		return fmt.Errorf("no GitHub token found: set GITHUB_TOKEN environment variable or authenticate with 'gh auth login'")
	}

	// Report mode validation
	if c.Report {
		if len(c.SourceBranches) > 0 {
			return fmt.Errorf("--source-branch cannot be used with the report command; report mode aggregates all matching branches")
		}
		if c.SkipRebase {
			return fmt.Errorf("--skip-rebase cannot be used with the report command; report mode is read-only")
		}
		if c.Confirm {
			return fmt.Errorf("--confirm cannot be used with the report command; report mode does not perform actions")
		}
		if c.MinGroupSize < 1 {
			return fmt.Errorf("--min-group-size must be at least 1")
		}
		if c.Verbosity != "" && c.Verbosity != "brief" && c.Verbosity != "standard" && c.Verbosity != "verbose" {
			return fmt.Errorf("--verbosity must be one of: brief, standard, verbose")
		}
		return nil
	}

	// Non-report mode validation
	if len(c.SourceBranches) == 0 {
		return fmt.Errorf("--source-branch is required")
	}
	if len(c.SourceBranchPrefix) > 0 {
		return fmt.Errorf("--source-branch-prefix can only be used with the report command")
	}
	if c.Verbosity != "" {
		return fmt.Errorf("--verbosity can only be used with the report command")
	}
	// --skip-rebase requires merge subcommand
	if c.SkipRebase && !c.Merge {
		return fmt.Errorf("--skip-rebase requires the merge command; it allows merging PRs without requiring the branch to be up-to-date")
	}
	// --skip-rebase is invalid in rebase mode
	if c.SkipRebase && c.Rebase {
		return fmt.Errorf("--skip-rebase cannot be used with the rebase command")
	}
	return nil
}

// ErrHelp is returned when -help or -h is passed.
var ErrHelp = flag.ErrHelp

// ErrVersion is returned when --version is passed.
var ErrVersion = errors.New("version requested")

// ParseFlags parses command-line flags and environment variables.
// Supports subcommands: merge, rebase, report
// Usage: ghprmerge [global-flags] <command> [command-flags]
func ParseFlags(args []string, version string) (*Config, error) {
	if len(args) > 0 {
		// Check for --version or --help before subcommand parsing
		for _, arg := range args {
			if arg == "--version" || arg == "-version" {
				fmt.Printf("ghprmerge version %s\n", version)
				return nil, ErrVersion
			}
		}
	}

	// Detect subcommand
	var command Command
	var subArgs []string
	globalArgs := make([]string, 0, len(args))

	// Find the subcommand position
	subCmdIdx := -1
	for i, arg := range args {
		switch arg {
		case "merge", "rebase", "report":
			command = Command(arg)
			subCmdIdx = i
		}
		if subCmdIdx >= 0 {
			break
		}
	}

	if subCmdIdx >= 0 {
		globalArgs = args[:subCmdIdx]
		subArgs = args[subCmdIdx+1:]
	} else {
		globalArgs = args
	}

	// Parse global flags
	globalFS := flag.NewFlagSet("ghprmerge", flag.ContinueOnError)
	org := globalFS.String("org", os.Getenv("GITHUB_ORG"), "GitHub organization to scan")
	repoLimit := globalFS.Int("repo-limit", 0, "Maximum number of repositories to process (0 = unlimited)")
	jsonOutput := globalFS.Bool("json", false, "Output structured JSON instead of human-readable text")
	verbose := globalFS.Bool("verbose", false, "Show all repositories including those with no matching pull requests")
	noColor := globalFS.Bool("no-color", false, "Disable colored output")
	showVersion := globalFS.Bool("version", false, "Show version information and exit")

	var globalRepos StringSliceFlag
	globalFS.Var(&globalRepos, "repo", "Limit execution to specific repositories (may be repeated)")

	if err := globalFS.Parse(globalArgs); err != nil {
		return nil, err
	}

	// Handle --version flag
	if *showVersion {
		fmt.Printf("ghprmerge version %s\n", version)
		return nil, ErrVersion
	}

	// Parse subcommand-specific flags
	var sourceBranches StringSliceFlag
	var skipRebase bool
	var confirm bool
	var sourceBranchPrefixStr string
	var minGroupSize int
	var verbosity string

	// Additional repos from subcommand flags
	var subRepos StringSliceFlag

	if command != CommandNone {
		subFS := flag.NewFlagSet(string(command), flag.ContinueOnError)

		switch command {
		case CommandMerge:
			subFS.Var(&sourceBranches, "source-branch", "Branch name pattern to match pull request head branches (repeatable)")
			subFS.BoolVar(&skipRebase, "skip-rebase", false, "Skip rebase check and merge PRs that are behind")
			subFS.BoolVar(&confirm, "confirm", false, "Scan all repos first, then prompt for confirmation")
			subFS.Var(&subRepos, "repo", "Limit execution to specific repositories (may be repeated)")
		case CommandRebase:
			subFS.Var(&sourceBranches, "source-branch", "Branch name pattern to match pull request head branches (repeatable)")
			subFS.BoolVar(&confirm, "confirm", false, "Scan all repos first, then prompt for confirmation")
			subFS.Var(&subRepos, "repo", "Limit execution to specific repositories (may be repeated)")
		case CommandReport:
			subFS.String("source-branch-prefix", "", "Comma-separated list of branch prefixes to include in report")
			subFS.Int("min-group-size", 2, "Minimum number of PRs in a group to include in report")
			subFS.String("verbosity", "", "Report output verbosity: brief, standard, or verbose")
			subFS.Var(&subRepos, "repo", "Limit execution to specific repositories (may be repeated)")
		}

		if err := subFS.Parse(subArgs); err != nil {
			return nil, err
		}

		// Extract report-specific parsed values
		if command == CommandReport {
			if f := subFS.Lookup("source-branch-prefix"); f != nil {
				sourceBranchPrefixStr = f.Value.String()
			}
			if f := subFS.Lookup("min-group-size"); f != nil {
				fmt.Sscanf(f.Value.String(), "%d", &minGroupSize)
			} else {
				minGroupSize = 2
			}
			if f := subFS.Lookup("verbosity"); f != nil {
				verbosity = f.Value.String()
			}
		}
	}

	// Parse source-branch-prefix into a slice
	var prefixes []string
	if sourceBranchPrefixStr != "" {
		for _, p := range strings.Split(sourceBranchPrefixStr, ",") {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				prefixes = append(prefixes, trimmed)
			}
		}
	}

	// Merge repos from global and subcommand
	allRepos := append([]string(nil), globalRepos...)
	allRepos = append(allRepos, subRepos...)

	// Resolve authentication token
	token := resolveToken()

	// Set sourceBranch for backward compatibility
	var sourceBranch string
	if len(sourceBranches) > 0 {
		sourceBranch = sourceBranches[0]
	}

	return &Config{
		Org:                *org,
		SourceBranches:     sourceBranches,
		SourceBranch:       sourceBranch,
		Rebase:             command == CommandRebase,
		Merge:              command == CommandMerge,
		SkipRebase:         skipRebase,
		Repos:              allRepos,
		RepoLimit:          *repoLimit,
		JSON:               *jsonOutput,
		Confirm:            confirm,
		Verbose:            *verbose,
		NoColor:            *noColor,
		Token:              token,
		Report:             command == CommandReport,
		SourceBranchPrefix: prefixes,
		MinGroupSize:       minGroupSize,
		Verbosity:          verbosity,
		Command:            command,
	}, nil
}

// resolveToken resolves the GitHub authentication token.
// Order of precedence:
// 1. GITHUB_TOKEN environment variable
// 2. GitHub CLI authentication via 'gh auth token'
func resolveToken() string {
	// Check GITHUB_TOKEN environment variable first
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token
	}

	// Fall back to GitHub CLI
	cmd := exec.Command("gh", "auth", "token")
	output, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(output))
	}

	return ""
}
