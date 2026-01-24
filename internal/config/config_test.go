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
		name          string
		args          []string
		envToken      string
		envOrg        string
		wantOrg       string
		wantBranch    string
		wantRebase    bool
		wantMerge     bool
		wantRepoLimit int
		wantJSON      bool
		wantQuiet     bool
		wantRepos     []string
		wantErr       bool
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
			wantQuiet:     false,
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
			name:       "quiet mode",
			args:       []string{"--org", "myorg", "--source-branch", "dependabot/", "--quiet"},
			envToken:   "test-token",
			wantOrg:    "myorg",
			wantBranch: "dependabot/",
			wantQuiet:  true,
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
			if cfg.RepoLimit != tt.wantRepoLimit {
				t.Errorf("RepoLimit = %v, want %v", cfg.RepoLimit, tt.wantRepoLimit)
			}
			if cfg.JSON != tt.wantJSON {
				t.Errorf("JSON = %v, want %v", cfg.JSON, tt.wantJSON)
			}
			if cfg.Quiet != tt.wantQuiet {
				t.Errorf("Quiet = %v, want %v", cfg.Quiet, tt.wantQuiet)
			}
			if len(tt.wantRepos) > 0 {
				if len(cfg.Repos) != len(tt.wantRepos) {
					t.Errorf("Repos = %v, want %v", cfg.Repos, tt.wantRepos)
				}
			}
		})
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
