package config

import (
	"errors"
	"os"
	"testing"
)

func TestParseFlags(t *testing.T) {
	origToken := os.Getenv("GITHUB_TOKEN")
	origOrg := os.Getenv("GITHUB_ORG")
	defer func() {
		os.Setenv("GITHUB_TOKEN", origToken)
		os.Setenv("GITHUB_ORG", origOrg)
	}()

	tests := []struct {
		name             string
		args             []string
		envToken         string
		envOrg           string
		wantOrg          string
		wantBranch       string
		wantBranches     []string
		wantRebase       bool
		wantMerge        bool
		wantSkipRebase   bool
		wantRepoLimit    int
		wantJSON         bool
		wantVerbose      bool
		wantNoColor      bool
		wantConfirm      bool
		wantRepos        []string
		wantReport       bool
		wantCommand      Command
		wantErr          bool
	}{
		{
			name:          "rebase subcommand with json and limit",
			args:          []string{"--org", "myorg", "--repo-limit", "10", "--json", "rebase", "--source-branch", "dependabot/"},
			envToken:      "test-token",
			wantOrg:       "myorg",
			wantBranch:    "dependabot/",
			wantBranches:  []string{"dependabot/"},
			wantRebase:    true,
			wantMerge:     false,
			wantRepoLimit: 10,
			wantJSON:      true,
			wantCommand:   CommandRebase,
		},
		{
			name:        "no subcommand defaults",
			args:        []string{"--org", "myorg"},
			envToken:    "test-token",
			wantOrg:     "myorg",
			wantRebase:  false,
			wantMerge:   false,
			wantReport:  false,
			wantCommand: CommandNone,
		},
		{
			name:       "org from env",
			args:       []string{"merge", "--source-branch", "dependabot/"},
			envToken:   "test-token",
			envOrg:     "envorg",
			wantOrg:    "envorg",
			wantBranch: "dependabot/",
			wantMerge:  true,
			wantCommand: CommandMerge,
		},
		{
			name:       "multiple global repos",
			args:       []string{"--org", "myorg", "--repo", "repo1", "--repo", "repo2", "merge", "--source-branch", "test"},
			envToken:   "test-token",
			wantOrg:    "myorg",
			wantBranch: "test",
			wantRepos:  []string{"repo1", "repo2"},
			wantMerge:  true,
			wantCommand: CommandMerge,
		},
		{
			name:       "repos from both global and subcommand",
			args:       []string{"--org", "myorg", "--repo", "repo1", "merge", "--source-branch", "test", "--repo", "repo2"},
			envToken:   "test-token",
			wantOrg:    "myorg",
			wantBranch: "test",
			wantRepos:  []string{"repo1", "repo2"},
			wantMerge:  true,
			wantCommand: CommandMerge,
		},
		{
			name:        "rebase subcommand",
			args:        []string{"--org", "myorg", "rebase", "--source-branch", "dependabot/"},
			envToken:    "test-token",
			wantOrg:     "myorg",
			wantBranch:  "dependabot/",
			wantRebase:  true,
			wantMerge:   false,
			wantCommand: CommandRebase,
		},
		{
			name:        "merge subcommand",
			args:        []string{"--org", "myorg", "merge", "--source-branch", "dependabot/"},
			envToken:    "test-token",
			wantOrg:     "myorg",
			wantBranch:  "dependabot/",
			wantRebase:  false,
			wantMerge:   true,
			wantCommand: CommandMerge,
		},
		{
			name:           "merge subcommand with skip-rebase",
			args:           []string{"--org", "myorg", "merge", "--source-branch", "dependabot/", "--skip-rebase"},
			envToken:       "test-token",
			wantOrg:        "myorg",
			wantBranch:     "dependabot/",
			wantMerge:      true,
			wantSkipRebase: true,
			wantCommand:    CommandMerge,
		},
		{
			name:        "verbose global flag",
			args:        []string{"--org", "myorg", "--verbose", "merge", "--source-branch", "dependabot/"},
			envToken:    "test-token",
			wantOrg:     "myorg",
			wantBranch:  "dependabot/",
			wantVerbose: true,
			wantMerge:   true,
			wantCommand: CommandMerge,
		},
		{
			name:        "no-color global flag",
			args:        []string{"--org", "myorg", "--no-color", "rebase", "--source-branch", "dependabot/"},
			envToken:    "test-token",
			wantOrg:     "myorg",
			wantBranch:  "dependabot/",
			wantNoColor: true,
			wantRebase:  true,
			wantCommand: CommandRebase,
		},
		{
			name:         "multiple source-branch flags",
			args:         []string{"--org", "myorg", "merge", "--source-branch", "dep/", "--source-branch", "repver/"},
			envToken:     "test-token",
			wantOrg:      "myorg",
			wantBranch:   "dep/",
			wantBranches: []string{"dep/", "repver/"},
			wantMerge:    true,
			wantCommand:  CommandMerge,
		},
		{
			name:        "confirm flag under merge",
			args:        []string{"--org", "myorg", "merge", "--source-branch", "dependabot/", "--confirm"},
			envToken:    "test-token",
			wantOrg:     "myorg",
			wantBranch:  "dependabot/",
			wantMerge:   true,
			wantConfirm: true,
			wantCommand: CommandMerge,
		},
		{
			name:        "confirm flag under rebase",
			args:        []string{"--org", "myorg", "rebase", "--source-branch", "dependabot/", "--confirm"},
			envToken:    "test-token",
			wantOrg:     "myorg",
			wantBranch:  "dependabot/",
			wantRebase:  true,
			wantConfirm: true,
			wantCommand: CommandRebase,
		},
		{
			name:        "report subcommand",
			args:        []string{"--org", "myorg", "report"},
			envToken:    "test-token",
			wantOrg:     "myorg",
			wantReport:  true,
			wantCommand: CommandReport,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("GITHUB_TOKEN", tt.envToken)
			os.Setenv("GITHUB_ORG", tt.envOrg)

			cfg, err := ParseFlags(tt.args, "test")
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseFlags() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			if cfg.Org != tt.wantOrg {
				t.Errorf("Org = %v, want %v", cfg.Org, tt.wantOrg)
			}
			if cfg.SourceBranch != tt.wantBranch {
				t.Errorf("SourceBranch = %v, want %v", cfg.SourceBranch, tt.wantBranch)
			}
			if cfg.Rebase != tt.wantRebase {
				t.Errorf("Rebase = %v, want %v", cfg.Rebase, tt.wantRebase)
			}
			if cfg.Merge != tt.wantMerge {
				t.Errorf("Merge = %v, want %v", cfg.Merge, tt.wantMerge)
			}
			if cfg.SkipRebase != tt.wantSkipRebase {
				t.Errorf("SkipRebase = %v, want %v", cfg.SkipRebase, tt.wantSkipRebase)
			}
			if cfg.RepoLimit != tt.wantRepoLimit {
				t.Errorf("RepoLimit = %v, want %v", cfg.RepoLimit, tt.wantRepoLimit)
			}
			if cfg.JSON != tt.wantJSON {
				t.Errorf("JSON = %v, want %v", cfg.JSON, tt.wantJSON)
			}
			if cfg.Verbose != tt.wantVerbose {
				t.Errorf("Verbose = %v, want %v", cfg.Verbose, tt.wantVerbose)
			}
			if cfg.NoColor != tt.wantNoColor {
				t.Errorf("NoColor = %v, want %v", cfg.NoColor, tt.wantNoColor)
			}
			if cfg.Confirm != tt.wantConfirm {
				t.Errorf("Confirm = %v, want %v", cfg.Confirm, tt.wantConfirm)
			}
			if cfg.Report != tt.wantReport {
				t.Errorf("Report = %v, want %v", cfg.Report, tt.wantReport)
			}
			if cfg.Command != tt.wantCommand {
				t.Errorf("Command = %v, want %v", cfg.Command, tt.wantCommand)
			}
			if len(tt.wantRepos) > 0 {
				if len(cfg.Repos) != len(tt.wantRepos) {
					t.Errorf("Repos = %v, want %v", cfg.Repos, tt.wantRepos)
				} else {
					for i, r := range tt.wantRepos {
						if cfg.Repos[i] != r {
							t.Errorf("Repos[%d] = %v, want %v", i, cfg.Repos[i], r)
						}
					}
				}
			}
			if len(tt.wantBranches) > 0 {
				if len(cfg.SourceBranches) != len(tt.wantBranches) {
					t.Errorf("SourceBranches = %v, want %v", cfg.SourceBranches, tt.wantBranches)
				} else {
					for i, b := range tt.wantBranches {
						if cfg.SourceBranches[i] != b {
							t.Errorf("SourceBranches[%d] = %v, want %v", i, cfg.SourceBranches[i], b)
						}
					}
				}
			}
		})
	}
}

func TestParseFlagsVersion(t *testing.T) {
	origToken := os.Getenv("GITHUB_TOKEN")
	defer os.Setenv("GITHUB_TOKEN", origToken)
	os.Setenv("GITHUB_TOKEN", "test-token")

	_, err := ParseFlags([]string{"--version"}, "1.0.0")
	if !errors.Is(err, ErrVersion) {
		t.Fatalf("ParseFlags(--version) error = %v, want ErrVersion", err)
	}

	// Version flag anywhere in args
	_, err = ParseFlags([]string{"--org", "myorg", "--version"}, "1.0.0")
	if !errors.Is(err, ErrVersion) {
		t.Fatalf("ParseFlags(--org myorg --version) error = %v, want ErrVersion", err)
	}
}

func TestParseFlagsRejectsUnknownGlobalFlag(t *testing.T) {
	origToken := os.Getenv("GITHUB_TOKEN")
	defer os.Setenv("GITHUB_TOKEN", origToken)
	os.Setenv("GITHUB_TOKEN", "test-token")

	if _, err := ParseFlags([]string{"--org", "myorg", "--quiet", "merge", "--source-branch", "dependabot/"}, "test"); err == nil {
		t.Fatal("ParseFlags() expected error for unknown --quiet flag")
	}
}

func TestParseFlagsRejectsUnknownSubcommandFlag(t *testing.T) {
	origToken := os.Getenv("GITHUB_TOKEN")
	defer os.Setenv("GITHUB_TOKEN", origToken)
	os.Setenv("GITHUB_TOKEN", "test-token")

	// --skip-rebase is not valid under rebase subcommand
	if _, err := ParseFlags([]string{"--org", "myorg", "rebase", "--source-branch", "dep/", "--skip-rebase"}, "test"); err == nil {
		t.Fatal("ParseFlags() expected error for --skip-rebase under rebase subcommand")
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid analysis-only config",
			config: Config{
				Org:            "myorg",
				SourceBranches: []string{"dependabot/"},
				SourceBranch:   "dependabot/",
				Token:          "test-token",
			},
			wantErr: false,
		},
		{
			name: "valid config with rebase",
			config: Config{
				Org:            "myorg",
				SourceBranches: []string{"dependabot/"},
				SourceBranch:   "dependabot/",
				Token:          "test-token",
				Rebase:         true,
				Command:        CommandRebase,
			},
			wantErr: false,
		},
		{
			name: "valid config with merge",
			config: Config{
				Org:            "myorg",
				SourceBranches: []string{"dependabot/"},
				SourceBranch:   "dependabot/",
				Token:          "test-token",
				Merge:          true,
				Command:        CommandMerge,
			},
			wantErr: false,
		},
		{
			name: "valid config with merge and skip-rebase",
			config: Config{
				Org:            "myorg",
				SourceBranches: []string{"dependabot/"},
				SourceBranch:   "dependabot/",
				Token:          "test-token",
				Merge:          true,
				SkipRebase:     true,
				Command:        CommandMerge,
			},
			wantErr: false,
		},
		{
			name: "valid config with multiple source branches",
			config: Config{
				Org:            "myorg",
				SourceBranches: []string{"dep/", "repver/"},
				SourceBranch:   "dep/",
				Token:          "test-token",
				Merge:          true,
				Command:        CommandMerge,
			},
			wantErr: false,
		},
		{
			name: "missing org",
			config: Config{
				SourceBranches: []string{"dependabot/"},
				SourceBranch:   "dependabot/",
				Token:          "test-token",
			},
			wantErr: true,
			errMsg:  "--org is required",
		},
		{
			name: "missing source branch in non-report mode",
			config: Config{
				Org:   "myorg",
				Token: "test-token",
			},
			wantErr: true,
			errMsg:  "--source-branch is required",
		},
		{
			name: "missing token",
			config: Config{
				Org:            "myorg",
				SourceBranches: []string{"dependabot/"},
				SourceBranch:   "dependabot/",
			},
			wantErr: true,
			errMsg:  "no GitHub token found",
		},
		{
			name: "skip-rebase requires merge command",
			config: Config{
				Org:            "myorg",
				SourceBranches: []string{"dependabot/"},
				SourceBranch:   "dependabot/",
				Token:          "test-token",
				SkipRebase:     true,
				Merge:          false,
			},
			wantErr: true,
			errMsg:  "--skip-rebase requires the merge command",
		},
		{
			name: "skip-rebase cannot be used with rebase command",
			config: Config{
				Org:            "myorg",
				SourceBranches: []string{"dependabot/"},
				SourceBranch:   "dependabot/",
				Token:          "test-token",
				SkipRebase:     true,
				Rebase:         true,
				Merge:          true,
				Command:        CommandRebase,
			},
			wantErr: true,
			errMsg:  "--skip-rebase cannot be used with the rebase command",
		},
		{
			name: "source-branch-prefix only valid with report",
			config: Config{
				Org:                "myorg",
				SourceBranches:     []string{"dependabot/"},
				SourceBranch:       "dependabot/",
				Token:              "test-token",
				SourceBranchPrefix: []string{"dependabot/"},
			},
			wantErr: true,
			errMsg:  "--source-branch-prefix can only be used with the report command",
		},
		{
			name: "verbosity only valid with report",
			config: Config{
				Org:            "myorg",
				SourceBranches: []string{"dependabot/"},
				SourceBranch:   "dependabot/",
				Token:          "test-token",
				Verbosity:      "brief",
			},
			wantErr: true,
			errMsg:  "--verbosity can only be used with the report command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want to contain %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestConfigValidateReportMode(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid report config",
			config: Config{
				Org:          "myorg",
				Token:        "test-token",
				Report:       true,
				MinGroupSize: 2,
				Command:      CommandReport,
			},
			wantErr: false,
		},
		{
			name: "report with source-branches is invalid",
			config: Config{
				Org:            "myorg",
				Token:          "test-token",
				Report:         true,
				SourceBranches: []string{"dependabot/"},
				MinGroupSize:   2,
				Command:        CommandReport,
			},
			wantErr: true,
			errMsg:  "--source-branch cannot be used with the report command",
		},
		{
			name: "report with skip-rebase is invalid",
			config: Config{
				Org:          "myorg",
				Token:        "test-token",
				Report:       true,
				SkipRebase:   true,
				MinGroupSize: 2,
				Command:      CommandReport,
			},
			wantErr: true,
			errMsg:  "--skip-rebase cannot be used with the report command",
		},
		{
			name: "report with confirm is invalid",
			config: Config{
				Org:          "myorg",
				Token:        "test-token",
				Report:       true,
				Confirm:      true,
				MinGroupSize: 2,
				Command:      CommandReport,
			},
			wantErr: true,
			errMsg:  "--confirm cannot be used with the report command",
		},
		{
			name: "report with invalid verbosity",
			config: Config{
				Org:          "myorg",
				Token:        "test-token",
				Report:       true,
				MinGroupSize: 2,
				Verbosity:    "invalid",
				Command:      CommandReport,
			},
			wantErr: true,
			errMsg:  "--verbosity must be one of",
		},
		{
			name: "report with valid verbosity brief",
			config: Config{
				Org:          "myorg",
				Token:        "test-token",
				Report:       true,
				MinGroupSize: 2,
				Verbosity:    "brief",
				Command:      CommandReport,
			},
			wantErr: false,
		},
		{
			name: "report with valid verbosity standard",
			config: Config{
				Org:          "myorg",
				Token:        "test-token",
				Report:       true,
				MinGroupSize: 2,
				Verbosity:    "standard",
				Command:      CommandReport,
			},
			wantErr: false,
		},
		{
			name: "report with valid verbosity verbose",
			config: Config{
				Org:          "myorg",
				Token:        "test-token",
				Report:       true,
				MinGroupSize: 2,
				Verbosity:    "verbose",
				Command:      CommandReport,
			},
			wantErr: false,
		},
		{
			name: "report with min-group-size zero",
			config: Config{
				Org:          "myorg",
				Token:        "test-token",
				Report:       true,
				MinGroupSize: 0,
				Command:      CommandReport,
			},
			wantErr: true,
			errMsg:  "--min-group-size must be at least 1",
		},
		{
			name: "report with source-branch-prefix",
			config: Config{
				Org:                "myorg",
				Token:              "test-token",
				Report:             true,
				MinGroupSize:       2,
				SourceBranchPrefix: []string{"dependabot/"},
				Command:            CommandReport,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want to contain %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestIsAnalysisOnly(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		want   bool
	}{
		{
			name:   "no subcommand is analysis only",
			config: Config{},
			want:   true,
		},
		{
			name:   "report is analysis only",
			config: Config{Report: true, Command: CommandReport},
			want:   true,
		},
		{
			name:   "rebase is not analysis only",
			config: Config{Rebase: true, Command: CommandRebase},
			want:   false,
		},
		{
			name:   "merge is not analysis only",
			config: Config{Merge: true, Command: CommandMerge},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.IsAnalysisOnly(); got != tt.want {
				t.Errorf("IsAnalysisOnly() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseFlagsReport(t *testing.T) {
	origToken := os.Getenv("GITHUB_TOKEN")
	origOrg := os.Getenv("GITHUB_ORG")
	defer func() {
		os.Setenv("GITHUB_TOKEN", origToken)
		os.Setenv("GITHUB_ORG", origOrg)
	}()

	os.Setenv("GITHUB_TOKEN", "test-token")
	os.Setenv("GITHUB_ORG", "")

	// Test basic report subcommand
	cfg, err := ParseFlags([]string{"--org", "myorg", "report"}, "test")
	if err != nil {
		t.Fatalf("ParseFlags() error = %v", err)
	}
	if !cfg.Report {
		t.Error("expected Report = true")
	}
	if cfg.Command != CommandReport {
		t.Errorf("expected Command = report, got %v", cfg.Command)
	}
	if cfg.MinGroupSize != 2 {
		t.Errorf("expected default MinGroupSize = 2, got %d", cfg.MinGroupSize)
	}

	// Test report with source-branch-prefix
	cfg, err = ParseFlags([]string{"--org", "myorg", "report", "--source-branch-prefix", "dependabot/,repver/"}, "test")
	if err != nil {
		t.Fatalf("ParseFlags() error = %v", err)
	}
	if len(cfg.SourceBranchPrefix) != 2 {
		t.Fatalf("expected 2 prefixes, got %d: %v", len(cfg.SourceBranchPrefix), cfg.SourceBranchPrefix)
	}
	if cfg.SourceBranchPrefix[0] != "dependabot/" {
		t.Errorf("expected first prefix = dependabot/, got %s", cfg.SourceBranchPrefix[0])
	}
	if cfg.SourceBranchPrefix[1] != "repver/" {
		t.Errorf("expected second prefix = repver/, got %s", cfg.SourceBranchPrefix[1])
	}

	// Test report with min-group-size
	cfg, err = ParseFlags([]string{"--org", "myorg", "report", "--min-group-size", "5"}, "test")
	if err != nil {
		t.Fatalf("ParseFlags() error = %v", err)
	}
	if cfg.MinGroupSize != 5 {
		t.Errorf("expected MinGroupSize = 5, got %d", cfg.MinGroupSize)
	}

	// Test report with verbosity
	cfg, err = ParseFlags([]string{"--org", "myorg", "report", "--verbosity", "brief"}, "test")
	if err != nil {
		t.Fatalf("ParseFlags() error = %v", err)
	}
	if cfg.Verbosity != "brief" {
		t.Errorf("expected Verbosity = brief, got %s", cfg.Verbosity)
	}

	// Test report with repo under subcommand
	cfg, err = ParseFlags([]string{"--org", "myorg", "report", "--repo", "myrepo"}, "test")
	if err != nil {
		t.Fatalf("ParseFlags() error = %v", err)
	}
	if len(cfg.Repos) != 1 || cfg.Repos[0] != "myrepo" {
		t.Errorf("expected Repos = [myrepo], got %v", cfg.Repos)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
