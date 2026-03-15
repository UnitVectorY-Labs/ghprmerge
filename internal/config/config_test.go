package config

import (
	"os"
	"testing"
)

func TestParseFlags(t *testing.T) {
	// Save original env
	origToken := os.Getenv("GITHUB_TOKEN")
	origOrg := os.Getenv("GITHUB_ORG")
	defer func() {
		os.Setenv("GITHUB_TOKEN", origToken)
		os.Setenv("GITHUB_ORG", origOrg)
	}()

	tests := []struct {
		name           string
		args           []string
		envToken       string
		envOrg         string
		wantOrg        string
		wantBranch     string
		wantRebase     bool
		wantMerge      bool
		wantSkipRebase bool
		wantRepoLimit  int
		wantJSON       bool
		wantVerbose    bool
		wantNoColor    bool
		wantRepos      []string
		wantErr        bool
	}{
		{
			name:          "rebase with json and limit",
			args:          []string{"--org", "myorg", "--source-branch", "dependabot/", "--rebase", "--repo-limit", "10", "--json"},
			envToken:      "test-token",
			wantOrg:       "myorg",
			wantBranch:    "dependabot/",
			wantRebase:    true,
			wantMerge:     false,
			wantRepoLimit: 10,
			wantJSON:      true,
		},
		{
			name:          "defaults applied",
			args:          []string{"--org", "myorg", "--source-branch", "dependabot/"},
			envToken:      "test-token",
			wantOrg:       "myorg",
			wantBranch:    "dependabot/",
			wantRebase:    false,
			wantMerge:     false,
			wantRepoLimit: 0,
			wantJSON:      false,
		},
		{
			name:       "org from env",
			args:       []string{"--source-branch", "dependabot/"},
			envToken:   "test-token",
			envOrg:     "envorg",
			wantOrg:    "envorg",
			wantBranch: "dependabot/",
		},
		{
			name:       "multiple repos",
			args:       []string{"--org", "myorg", "--source-branch", "test", "--repo", "repo1", "--repo", "repo2"},
			envToken:   "test-token",
			wantOrg:    "myorg",
			wantBranch: "test",
			wantRepos:  []string{"repo1", "repo2"},
		},
		{
			name:       "rebase only",
			args:       []string{"--org", "myorg", "--source-branch", "dependabot/", "--rebase"},
			envToken:   "test-token",
			wantOrg:    "myorg",
			wantBranch: "dependabot/",
			wantRebase: true,
			wantMerge:  false,
		},
		{
			name:       "merge only",
			args:       []string{"--org", "myorg", "--source-branch", "dependabot/", "--merge"},
			envToken:   "test-token",
			wantOrg:    "myorg",
			wantBranch: "dependabot/",
			wantRebase: false,
			wantMerge:  true,
		},
		{
			name:           "skip-rebase with merge",
			args:           []string{"--org", "myorg", "--source-branch", "dependabot/", "--skip-rebase", "--merge"},
			envToken:       "test-token",
			wantOrg:        "myorg",
			wantBranch:     "dependabot/",
			wantMerge:      true,
			wantSkipRebase: true,
		},
		{
			name:        "verbose mode",
			args:        []string{"--org", "myorg", "--source-branch", "dependabot/", "--verbose"},
			envToken:    "test-token",
			wantOrg:     "myorg",
			wantBranch:  "dependabot/",
			wantVerbose: true,
		},
		{
			name:        "no-color mode",
			args:        []string{"--org", "myorg", "--source-branch", "dependabot/", "--no-color"},
			envToken:    "test-token",
			wantOrg:     "myorg",
			wantBranch:  "dependabot/",
			wantNoColor: true,
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
			if len(tt.wantRepos) > 0 {
				if len(cfg.Repos) != len(tt.wantRepos) {
					t.Errorf("Repos = %v, want %v", cfg.Repos, tt.wantRepos)
				}
			}
		})
	}
}

func TestParseFlagsRejectsQuiet(t *testing.T) {
	os.Setenv("GITHUB_TOKEN", "test-token")
	defer os.Unsetenv("GITHUB_TOKEN")

	if _, err := ParseFlags([]string{"--org", "myorg", "--source-branch", "dependabot/", "--quiet"}, "test"); err == nil {
		t.Fatal("ParseFlags() expected error for removed --quiet flag")
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
			name: "valid config",
			config: Config{
				Org:          "myorg",
				SourceBranch: "dependabot/",
				Token:        "test-token",
			},
			wantErr: false,
		},
		{
			name: "valid config with rebase",
			config: Config{
				Org:          "myorg",
				SourceBranch: "dependabot/",
				Token:        "test-token",
				Rebase:       true,
			},
			wantErr: false,
		},
		{
			name: "valid config with merge",
			config: Config{
				Org:          "myorg",
				SourceBranch: "dependabot/",
				Token:        "test-token",
				Merge:        true,
			},
			wantErr: false,
		},
		{
			name: "missing org",
			config: Config{
				SourceBranch: "dependabot/",
				Token:        "test-token",
			},
			wantErr: true,
			errMsg:  "--org is required",
		},
		{
			name: "missing source branch",
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
				Org:          "myorg",
				SourceBranch: "dependabot/",
			},
			wantErr: true,
			errMsg:  "no GitHub token found",
		},
		{
			name: "rebase and merge mutually exclusive",
			config: Config{
				Org:          "myorg",
				SourceBranch: "dependabot/",
				Token:        "test-token",
				Rebase:       true,
				Merge:        true,
			},
			wantErr: true,
			errMsg:  "mutually exclusive",
		},
		{
			name: "valid config with skip-rebase and merge",
			config: Config{
				Org:          "myorg",
				SourceBranch: "dependabot/",
				Token:        "test-token",
				SkipRebase:   true,
				Merge:        true,
			},
			wantErr: false,
		},
		{
			name: "skip-rebase and rebase mutually exclusive",
			config: Config{
				Org:          "myorg",
				SourceBranch: "dependabot/",
				Token:        "test-token",
				SkipRebase:   true,
				Rebase:       true,
			},
			wantErr: true,
			errMsg:  "--skip-rebase and --rebase are mutually exclusive",
		},
		{
			name: "skip-rebase requires merge",
			config: Config{
				Org:          "myorg",
				SourceBranch: "dependabot/",
				Token:        "test-token",
				SkipRebase:   true,
				Merge:        false,
			},
			wantErr: true,
			errMsg:  "--skip-rebase requires --merge",
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
			name:   "default is analysis only",
			config: Config{},
			want:   true,
		},
		{
			name:   "rebase only is not analysis only",
			config: Config{Rebase: true},
			want:   false,
		},
		{
			name:   "merge only is not analysis only",
			config: Config{Merge: true},
			want:   false,
		},
		// Note: --rebase and --merge together is invalid and rejected by Validate()
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.IsAnalysisOnly(); got != tt.want {
				t.Errorf("IsAnalysisOnly() = %v, want %v", got, tt.want)
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
			},
			wantErr: false,
		},
		{
			name: "report with source-branch is invalid",
			config: Config{
				Org:          "myorg",
				Token:        "test-token",
				Report:       true,
				SourceBranch: "dependabot/",
				MinGroupSize: 2,
			},
			wantErr: true,
			errMsg:  "--source-branch cannot be used with --report",
		},
		{
			name: "report with rebase is invalid",
			config: Config{
				Org:          "myorg",
				Token:        "test-token",
				Report:       true,
				Rebase:       true,
				MinGroupSize: 2,
			},
			wantErr: true,
			errMsg:  "--rebase cannot be used with --report",
		},
		{
			name: "report with merge is invalid",
			config: Config{
				Org:          "myorg",
				Token:        "test-token",
				Report:       true,
				Merge:        true,
				MinGroupSize: 2,
			},
			wantErr: true,
			errMsg:  "--merge cannot be used with --report",
		},
		{
			name: "report with skip-rebase is invalid",
			config: Config{
				Org:          "myorg",
				Token:        "test-token",
				Report:       true,
				SkipRebase:   true,
				MinGroupSize: 2,
			},
			wantErr: true,
			errMsg:  "--skip-rebase cannot be used with --report",
		},
		{
			name: "report with confirm is invalid",
			config: Config{
				Org:          "myorg",
				Token:        "test-token",
				Report:       true,
				Confirm:      true,
				MinGroupSize: 2,
			},
			wantErr: true,
			errMsg:  "--confirm cannot be used with --report",
		},
		{
			name: "report with invalid verbosity",
			config: Config{
				Org:          "myorg",
				Token:        "test-token",
				Report:       true,
				MinGroupSize: 2,
				Verbosity:    "invalid",
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
			},
			wantErr: false,
		},
		{
			name: "non-report with source-branch-prefix is invalid",
			config: Config{
				Org:                "myorg",
				Token:              "test-token",
				SourceBranch:       "dependabot/",
				SourceBranchPrefix: []string{"dependabot/"},
			},
			wantErr: true,
			errMsg:  "--source-branch-prefix can only be used with --report",
		},
		{
			name: "non-report with verbosity is invalid",
			config: Config{
				Org:          "myorg",
				Token:        "test-token",
				SourceBranch: "dependabot/",
				Verbosity:    "brief",
			},
			wantErr: true,
			errMsg:  "--verbosity can only be used with --report",
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

func TestParseFlagsReport(t *testing.T) {
	origToken := os.Getenv("GITHUB_TOKEN")
	origOrg := os.Getenv("GITHUB_ORG")
	defer func() {
		os.Setenv("GITHUB_TOKEN", origToken)
		os.Setenv("GITHUB_ORG", origOrg)
	}()

	os.Setenv("GITHUB_TOKEN", "test-token")
	os.Setenv("GITHUB_ORG", "")

	// Test basic report flag parsing
	cfg, err := ParseFlags([]string{"--org", "myorg", "--report"}, "test")
	if err != nil {
		t.Fatalf("ParseFlags() error = %v", err)
	}
	if !cfg.Report {
		t.Error("expected Report = true")
	}
	if cfg.MinGroupSize != 2 {
		t.Errorf("expected default MinGroupSize = 2, got %d", cfg.MinGroupSize)
	}

	// Test report with source-branch-prefix
	cfg, err = ParseFlags([]string{"--org", "myorg", "--report", "--source-branch-prefix", "dependabot/,repver/"}, "test")
	if err != nil {
		t.Fatalf("ParseFlags() error = %v", err)
	}
	if len(cfg.SourceBranchPrefix) != 2 {
		t.Errorf("expected 2 prefixes, got %d", len(cfg.SourceBranchPrefix))
	}
	if cfg.SourceBranchPrefix[0] != "dependabot/" {
		t.Errorf("expected first prefix = dependabot/, got %s", cfg.SourceBranchPrefix[0])
	}
	if cfg.SourceBranchPrefix[1] != "repver/" {
		t.Errorf("expected second prefix = repver/, got %s", cfg.SourceBranchPrefix[1])
	}

	// Test report with min-group-size
	cfg, err = ParseFlags([]string{"--org", "myorg", "--report", "--min-group-size", "5"}, "test")
	if err != nil {
		t.Fatalf("ParseFlags() error = %v", err)
	}
	if cfg.MinGroupSize != 5 {
		t.Errorf("expected MinGroupSize = 5, got %d", cfg.MinGroupSize)
	}

	// Test report with verbosity
	cfg, err = ParseFlags([]string{"--org", "myorg", "--report", "--verbosity", "brief"}, "test")
	if err != nil {
		t.Fatalf("ParseFlags() error = %v", err)
	}
	if cfg.Verbosity != "brief" {
		t.Errorf("expected Verbosity = brief, got %s", cfg.Verbosity)
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
