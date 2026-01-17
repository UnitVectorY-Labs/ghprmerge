package github

import (
	"testing"
)

func TestIsDependabotBranch(t *testing.T) {
	tests := []struct {
		name       string
		branchName string
		want       bool
	}{
		{
			name:       "dependabot branch",
			branchName: "dependabot/npm_and_yarn/lodash-4.17.21",
			want:       true,
		},
		{
			name:       "dependabot go branch",
			branchName: "dependabot/go_modules/golang.org/x/text-0.3.7",
			want:       true,
		},
		{
			name:       "regular feature branch",
			branchName: "feature/new-feature",
			want:       false,
		},
		{
			name:       "main branch",
			branchName: "main",
			want:       false,
		},
		{
			name:       "branch with dependabot in name but not prefix",
			branchName: "feature/dependabot-updates",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsDependabotBranch(tt.branchName); got != tt.want {
				t.Errorf("IsDependabotBranch(%q) = %v, want %v", tt.branchName, got, tt.want)
			}
		})
	}
}

func TestMatchesBranchPattern(t *testing.T) {
	tests := []struct {
		name       string
		branchName string
		pattern    string
		want       bool
	}{
		{
			name:       "exact match",
			branchName: "dependabot/npm_and_yarn/lodash",
			pattern:    "dependabot/npm_and_yarn/lodash",
			want:       true,
		},
		{
			name:       "prefix match",
			branchName: "dependabot/npm_and_yarn/lodash",
			pattern:    "dependabot/",
			want:       true,
		},
		{
			name:       "substring match",
			branchName: "dependabot/npm_and_yarn/lodash",
			pattern:    "npm",
			want:       true,
		},
		{
			name:       "no match",
			branchName: "feature/new-feature",
			pattern:    "dependabot/",
			want:       false,
		},
		{
			name:       "empty pattern matches everything",
			branchName: "any-branch",
			pattern:    "",
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MatchesBranchPattern(tt.branchName, tt.pattern); got != tt.want {
				t.Errorf("MatchesBranchPattern(%q, %q) = %v, want %v", tt.branchName, tt.pattern, got, tt.want)
			}
		})
	}
}
