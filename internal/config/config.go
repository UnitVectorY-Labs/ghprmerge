// Package config handles configuration and command line argument parsing for ghprmerge.
package config

import (
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
	DryRun       bool
	Rebase       bool
	Repos        []string
	Limit        int
	JSON         bool
	Token        string
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
	return nil
}

// ErrHelp is returned when -help or -h is passed.
var ErrHelp = flag.ErrHelp

// ParseFlags parses command-line flags and environment variables.
func ParseFlags(args []string) (*Config, error) {
	fs := flag.NewFlagSet("ghprmerge", flag.ContinueOnError)

	var repos StringSliceFlag

	org := fs.String("org", os.Getenv("GITHUB_ORG"), "GitHub organization to scan")
	sourceBranch := fs.String("source-branch", "", "Branch name pattern to match pull request head branches")
	dryRun := fs.Bool("dry-run", true, "When enabled, no mutations are performed")
	rebase := fs.Bool("rebase", false, "When enabled, out-of-date pull request branches are updated before merging")
	limit := fs.Int("limit", 0, "Maximum number of pull requests to merge in a single run (0 = unlimited)")
	jsonOutput := fs.Bool("json", false, "Output structured JSON instead of human-readable text")

	fs.Var(&repos, "repo", "Limit execution to specific repositories (may be repeated)")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	// Resolve authentication token
	token := resolveToken()

	return &Config{
		Org:          *org,
		SourceBranch: *sourceBranch,
		DryRun:       *dryRun,
		Rebase:       *rebase,
		Repos:        repos,
		Limit:        *limit,
		JSON:         *jsonOutput,
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
