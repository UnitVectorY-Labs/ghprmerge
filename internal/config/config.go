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

// Config holds all configuration for ghprmerge.
type Config struct {
	Org          string
	SourceBranch string
	Rebase       bool
	Merge        bool
	SkipRebase   bool
	Repos        []string
	RepoLimit    int
	JSON         bool
	Confirm      bool
	Quiet        bool
	Token        string
}

// IsAnalysisOnly returns true if neither --rebase nor --merge is set.
func (c *Config) IsAnalysisOnly() bool {
	return !c.Rebase && !c.Merge
}

// Validate checks that all required configuration is present.
func (c *Config) Validate() error {
	if c.Org == "" {
		return fmt.Errorf("--org is required (or set GITHUB_ORG environment variable)")
	}
	if c.SourceBranch == "" {
		return fmt.Errorf("--source-branch is required")
	}
	if c.Token == "" {
		return fmt.Errorf("no GitHub token found: set GITHUB_TOKEN environment variable or authenticate with 'gh auth login'")
	}
	// --rebase and --merge are mutually exclusive
	if c.Rebase && c.Merge {
		return fmt.Errorf("--rebase and --merge are mutually exclusive; use --rebase first to update branches, then --merge after checks pass")
	}
	// --skip-rebase and --rebase are mutually exclusive
	if c.SkipRebase && c.Rebase {
		return fmt.Errorf("--skip-rebase and --rebase are mutually exclusive; --skip-rebase skips rebasing entirely, while --rebase updates branches")
	}
	// --skip-rebase requires --merge
	if c.SkipRebase && !c.Merge {
		return fmt.Errorf("--skip-rebase requires --merge; it allows merging PRs without requiring the branch to be up-to-date")
	}
	return nil
}

// ErrHelp is returned when -help or -h is passed.
var ErrHelp = flag.ErrHelp

// ErrVersion is returned when --version is passed.
var ErrVersion = errors.New("version requested")

// ParseFlags parses command-line flags and environment variables.
func ParseFlags(args []string, version string) (*Config, error) {
	fs := flag.NewFlagSet("ghprmerge", flag.ContinueOnError)

	var repos StringSliceFlag

	org := fs.String("org", os.Getenv("GITHUB_ORG"), "GitHub organization to scan")
	sourceBranch := fs.String("source-branch", "", "Branch name pattern to match pull request head branches")
	rebase := fs.Bool("rebase", false, "Update out-of-date branches (mutually exclusive with --merge and --skip-rebase)")
	merge := fs.Bool("merge", false, "Merge pull requests that are in a valid state (mutually exclusive with --rebase)")
	skipRebase := fs.Bool("skip-rebase", false, "Skip rebase check and merge PRs that are behind (requires --merge, mutually exclusive with --rebase)")
	repoLimit := fs.Int("repo-limit", 0, "Maximum number of repositories to process (0 = unlimited)")
	jsonOutput := fs.Bool("json", false, "Output structured JSON instead of human-readable text")
	confirm := fs.Bool("confirm", false, "Scan all repos first, then prompt for confirmation before taking actions")
	quiet := fs.Bool("quiet", false, "Reduce output by suppressing repos with no matching pull requests")
	showVersion := fs.Bool("version", false, "Show version information and exit")

	fs.Var(&repos, "repo", "Limit execution to specific repositories (may be repeated)")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	// Handle --version flag
	if *showVersion {
		fmt.Printf("ghprmerge version %s\n", version)
		return nil, ErrVersion
	}

	// Resolve authentication token
	token := resolveToken()

	return &Config{
		Org:          *org,
		SourceBranch: *sourceBranch,
		Rebase:       *rebase,
		Merge:        *merge,
		SkipRebase:   *skipRebase,
		Repos:        repos,
		RepoLimit:    *repoLimit,
		JSON:         *jsonOutput,
		Confirm:      *confirm,
		Quiet:        *quiet,
		Token:        token,
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
