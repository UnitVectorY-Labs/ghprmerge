// Package config handles configuration and command line argument parsing for ghprmerge.
package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
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

type CommandDescription struct {
	Name        Command
	Description string
}

var commandDescriptions = []CommandDescription{
	{
		Name:        CommandMerge,
		Description: "merge ready pull requests after safety checks pass",
	},
	{
		Name:        CommandRebase,
		Description: "update pull request branches that are behind the default branch",
	},
	{
		Name:        CommandReport,
		Description: "scan open pull requests and group them by source branch",
	},
}

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
	NoProgress         bool
	Token              string
	Report             bool
	SourceBranchPrefix []string
	MinGroupSize       int
	Verbosity          string
	Command            Command
	Author             string
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
		if c.Command == CommandNone {
			return errors.New(formatSubcommandGuidanceError("choose a subcommand or provide --source-branch for analysis-only mode"))
		}
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

var semverRe = regexp.MustCompile(`^\d+\.\d+\.\d+`)

func buildVersionOutput(version string) string {
	normalized := version
	if semverRe.MatchString(normalized) && !strings.HasPrefix(normalized, "v") {
		normalized = "v" + normalized
	}
	return fmt.Sprintf("%s (%s, %s/%s)", normalized, runtime.Version(), runtime.GOOS, runtime.GOARCH)
}

// ErrHelp is returned when -help or -h is passed.
var ErrHelp = flag.ErrHelp

// ErrVersion is returned when --version is passed.
var ErrVersion = errors.New("version requested")

// ParseFlags parses command-line flags and environment variables.
// Supports subcommands: merge, rebase, report
// Usage: ghprmerge <command> [flags]
func ParseFlags(args []string, version string) (*Config, error) {
	if len(args) > 0 {
		// Check for --version or --help before subcommand parsing
		for _, arg := range args {
			if arg == "--version" || arg == "-version" {
				fmt.Printf("ghprmerge version %s\n", buildVersionOutput(version))
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

	org := os.Getenv("GITHUB_ORG")
	repoLimit := 0
	jsonOutput := false
	verbose := false
	noColor := false
	noProgress := false
	author := os.Getenv("GHPRMERGE_AUTHOR")

	// Root-only flags are parsed before a subcommand. All operational flags are
	// registered on the selected subcommand so they can be placed after it.
	globalFS := flag.NewFlagSet(commandName(), flag.ContinueOnError)
	showVersion := globalFS.Bool("version", false, "Show version information and exit")

	globalFS.Usage = func() {
		printGlobalUsage(globalFS.Output(), globalFS)
	}

	if err := globalFS.Parse(globalArgs); err != nil {
		return nil, err
	}

	// Handle --version flag
	if *showVersion {
		fmt.Printf("ghprmerge version %s\n", buildVersionOutput(version))
		return nil, ErrVersion
	}

	if command == CommandNone {
		remainingArgs := globalFS.Args()
		if len(remainingArgs) > 0 {
			return nil, errors.New(formatSubcommandGuidanceError(fmt.Sprintf("unknown subcommand %q", remainingArgs[0])))
		}
	}

	// Parse subcommand-specific flags
	var sourceBranches StringSliceFlag
	var skipRebase bool
	var confirm bool
	var sourceBranchPrefixStr string
	var minGroupSize int
	var verbosity string
	var repos StringSliceFlag

	if command != CommandNone {
		subFS := flag.NewFlagSet(string(command), flag.ContinueOnError)
		subFS.Usage = func() {
			printSubcommandUsage(subFS.Output(), command, subFS)
		}
		subFS.StringVar(&org, "org", org, "GitHub organization to scan")
		subFS.Var(&repos, "repo", "Exact repository name in the organization to scan (may be repeated)")
		subFS.IntVar(&repoLimit, "repo-limit", repoLimit, "Maximum number of repositories to process (0 = unlimited)")
		subFS.BoolVar(&jsonOutput, "json", jsonOutput, "Output structured JSON instead of human-readable text")
		subFS.BoolVar(&noColor, "no-color", noColor, "Disable colored output")
		subFS.BoolVar(&noProgress, "no-progress", noProgress, "Suppress progress bar output (useful for scripting, CI, and non-TTY environments)")
		subFS.StringVar(&author, "author", author, "Filter pull requests by author login (e.g. app/dependabot or a GitHub username)")

		switch command {
		case CommandMerge:
			subFS.BoolVar(&verbose, "verbose", verbose, "Show all repositories including those with no matching pull requests")
			subFS.Var(&sourceBranches, "source-branch", "Branch name pattern to match pull request head branches (repeatable)")
			subFS.BoolVar(&skipRebase, "skip-rebase", false, "Skip rebase check and merge PRs that are behind")
			subFS.BoolVar(&confirm, "confirm", false, "Scan all repos first, then prompt for confirmation")
		case CommandRebase:
			subFS.BoolVar(&verbose, "verbose", verbose, "Show all repositories including those with no matching pull requests")
			subFS.Var(&sourceBranches, "source-branch", "Branch name pattern to match pull request head branches (repeatable)")
			subFS.BoolVar(&confirm, "confirm", false, "Scan all repos first, then prompt for confirmation")
		case CommandReport:
			subFS.String("source-branch-prefix", "", "Comma-separated list of branch prefixes to include in report")
			defaultMinGroupSize := 2
			if v := os.Getenv("GHPRMERGE_MIN_GROUP_SIZE"); v != "" {
				n, err := fmt.Sscan(v, &defaultMinGroupSize)
				if err != nil || n != 1 || defaultMinGroupSize < 1 {
					return nil, fmt.Errorf("invalid GHPRMERGE_MIN_GROUP_SIZE value %q: must be a positive integer (1 or greater)", v)
				}
			}
			subFS.Int("min-group-size", defaultMinGroupSize, "Minimum number of PRs in a group to include in report")
			subFS.String("verbosity", "", "Report output verbosity: brief, standard, or verbose")
		}

		if err := subFS.Parse(subArgs); err != nil {
			return nil, err
		}

		// Extract report-specific parsed values
		if command == CommandReport {
			if f := subFS.Lookup("source-branch-prefix"); f != nil {
				sourceBranchPrefixStr = f.Value.String()
			}
			// min-group-size is always registered for CommandReport; flag.Int ensures
			// the value is a valid integer. Range validation (>= 1) is enforced by Config.Validate().
			if f := subFS.Lookup("min-group-size"); f != nil {
				fmt.Sscan(f.Value.String(), &minGroupSize) //nolint:errcheck // flag.Int ensures valid int
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
		for p := range strings.SplitSeq(sourceBranchPrefixStr, ",") {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				prefixes = append(prefixes, trimmed)
			}
		}
	}

	// Resolve authentication token
	token := resolveToken()

	// Set sourceBranch for backward compatibility
	var sourceBranch string
	if len(sourceBranches) > 0 {
		sourceBranch = sourceBranches[0]
	}

	return &Config{
		Org:                org,
		SourceBranches:     sourceBranches,
		SourceBranch:       sourceBranch,
		Rebase:             command == CommandRebase,
		Merge:              command == CommandMerge,
		SkipRebase:         skipRebase,
		Repos:              repos,
		RepoLimit:          repoLimit,
		JSON:               jsonOutput,
		Confirm:            confirm,
		Verbose:            verbose,
		NoColor:            noColor,
		NoProgress:         noProgress,
		Token:              token,
		Report:             command == CommandReport,
		SourceBranchPrefix: prefixes,
		MinGroupSize:       minGroupSize,
		Verbosity:          verbosity,
		Command:            command,
		Author:             author,
	}, nil
}

func commandName() string {
	if len(os.Args) == 0 || os.Args[0] == "" {
		return "ghprmerge"
	}
	return filepath.Base(os.Args[0])
}

func subcommandSummary() string {
	var b strings.Builder
	maxNameWidth := 0
	for _, cmd := range commandDescriptions {
		if len(string(cmd.Name)) > maxNameWidth {
			maxNameWidth = len(string(cmd.Name))
		}
	}
	for _, cmd := range commandDescriptions {
		fmt.Fprintf(&b, "  %-*s %s\n", maxNameWidth, cmd.Name, cmd.Description)
	}
	return strings.TrimRight(b.String(), "\n")
}

func printGlobalUsage(w io.Writer, globalFS *flag.FlagSet) {
	fmt.Fprintf(w, "Usage:\n  %s <command> [flags]\n\n", commandName())
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, subcommandSummary())
	fmt.Fprintf(w, "\nUse '%s <command> --help' for each command's behavior and flags.\n", commandName())
	printSharedCommandFlags(w)
	printGlobalFlags(w)
	printEnvironmentVariables(w)
}

func printSubcommandUsage(w io.Writer, command Command, subFS *flag.FlagSet) {
	fmt.Fprintf(w, "Usage:\n  %s %s --org <organization> [flags]\n\n", commandName(), command)
	fmt.Fprintf(w, "%s\n", commandHelpDescription(command))
	printSharedCommandFlags(w)
	printGlobalFlags(w)
	printCommandFlags(w, command)
	printEnvironmentVariables(w)
	fmt.Fprintf(w, "\nRun '%s --help' to compare commands.\n", commandName())
}

func commandHelpDescription(command Command) string {
	switch command {
	case CommandMerge:
		return "merge scans matching pull requests and merges only those that are ready: not drafts, targeting the default branch, conflict-free, up to date (unless --skip-rebase is set), and passing checks."
	case CommandRebase:
		return "rebase scans matching pull requests and updates branches that are behind their repositories' default branch. It does not merge pull requests."
	case CommandReport:
		return "report is read-only: it scans open pull requests, groups them by source branch, and reports the matching groups. It never merges or rebases."
	default:
		return ""
	}
}

func printSharedCommandFlags(w io.Writer) {
	fmt.Fprintln(w, "\nRequired setup:")
	fmt.Fprintln(w, "  --org <organization>       GitHub organization to scan. Required unless GITHUB_ORG is set.")
	fmt.Fprintln(w, "\nRepository filter:")
	fmt.Fprintln(w, "  --repo <repository>        Exact repository name within the organization to scan; may be repeated.")
}

func printGlobalFlags(w io.Writer) {
	fmt.Fprintln(w, "\nFiltering and execution flags:")
	fmt.Fprintln(w, "  --author <login>           Only include pull requests opened by this GitHub login.")
	fmt.Fprintln(w, "  --repo-limit <n>           Process at most n repositories (0 means unlimited).")
	fmt.Fprintln(w, "\nOutput flags:")
	fmt.Fprintln(w, "  --json                     Emit structured JSON instead of human-readable output.")
	fmt.Fprintln(w, "  --no-color                 Disable ANSI color output.")
	fmt.Fprintln(w, "  --no-progress              Suppress progress-bar output for CI or scripts.")
	fmt.Fprintln(w, "  --version                  Print version information and exit.")
}

func printCommandFlags(w io.Writer, command Command) {
	switch command {
	case CommandMerge:
		fmt.Fprintln(w, "\nMerge flags:")
		fmt.Fprintln(w, "  --source-branch <pattern>  Pull request head-branch prefix to match; required and may be repeated.")
		fmt.Fprintln(w, "  --skip-rebase              Allow merge attempts when a branch is behind its default branch.")
		fmt.Fprintln(w, "  --confirm                  Scan first, then prompt before merging candidates.")
		fmt.Fprintln(w, "  --verbose                  Show repositories with no matching pull requests as they are scanned.")
	case CommandRebase:
		fmt.Fprintln(w, "\nRebase flags:")
		fmt.Fprintln(w, "  --source-branch <pattern>  Pull request head-branch prefix to match; required and may be repeated.")
		fmt.Fprintln(w, "  --confirm                  Scan first, then prompt before rebasing candidates.")
		fmt.Fprintln(w, "  --verbose                  Show repositories with no matching pull requests as they are scanned.")
	case CommandReport:
		fmt.Fprintln(w, "\nReport flags:")
		fmt.Fprintln(w, "  --source-branch-prefix <prefixes>  Comma-separated head-branch prefixes to include.")
		fmt.Fprintln(w, "  --min-group-size <n>               Include only groups with at least n pull requests (default 2).")
		fmt.Fprintln(w, "  --verbosity <level>                 Text detail: brief, standard, or verbose.")
	}
}

func printEnvironmentVariables(w io.Writer) {
	fmt.Fprintln(w, "\nEnvironment variables:")
	fmt.Fprintln(w, "  GITHUB_TOKEN               GitHub token. If unset, ghprmerge uses 'gh auth token'.")
	fmt.Fprintln(w, "  GITHUB_ORG                 Default organization for --org.")
	fmt.Fprintln(w, "  GHPRMERGE_AUTHOR           Default GitHub login for --author.")
	fmt.Fprintln(w, "  GHPRMERGE_MIN_GROUP_SIZE   Default --min-group-size value for report.")
}

func formatSubcommandGuidanceError(summary string) string {
	return fmt.Sprintf(
		`%s

Choose a subcommand:
%s

Use '%s --help' for full usage`,
		summary,
		subcommandSummary(),
		commandName(),
	)
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
