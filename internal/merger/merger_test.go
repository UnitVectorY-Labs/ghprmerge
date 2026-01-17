package merger

import (
	"context"
	"fmt"
	"testing"

	"github.com/UnitVectorY-Labs/ghprmerge/internal/config"
	"github.com/UnitVectorY-Labs/ghprmerge/internal/github"
	"github.com/UnitVectorY-Labs/ghprmerge/internal/output"
)

func TestMergerDryRun(t *testing.T) {
	mock := github.NewMockClient()
	mock.Repositories = []github.Repository{
		{
			Name:          "repo1",
			FullName:      "testorg/repo1",
			DefaultBranch: "main",
			Archived:      false,
		},
	}
	mock.PullRequests["testorg/repo1"] = []github.PullRequest{
		{
			Number:     1,
			Title:      "Bump lodash",
			URL:        "https://github.com/testorg/repo1/pull/1",
			HeadBranch: "dependabot/npm/lodash",
			BaseBranch: "main",
			State:      "open",
			Draft:      false,
			HeadSHA:    "abc123",
			RepoName:   "repo1",
		},
	}
	mock.CheckStatuses["testorg/repo1/abc123"] = &github.CheckStatus{
		AllPassing: true,
		Details:    "all checks passing",
	}

	cfg := &config.Config{
		Org:          "testorg",
		SourceBranch: "dependabot/",
		DryRun:       true,
		Rebase:       false,
	}

	m := New(mock, cfg)
	result, err := m.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify no mutations were made
	if len(mock.MergeCalls) > 0 {
		t.Errorf("Expected no merge calls in dry run, got %d", len(mock.MergeCalls))
	}

	// Verify result
	if result.Summary.WouldMerge != 1 {
		t.Errorf("WouldMerge = %d, want 1", result.Summary.WouldMerge)
	}

	// Verify action is "would merge"
	if len(result.Repositories) != 1 {
		t.Fatalf("Expected 1 repository, got %d", len(result.Repositories))
	}
	if len(result.Repositories[0].PullRequests) != 1 {
		t.Fatalf("Expected 1 PR, got %d", len(result.Repositories[0].PullRequests))
	}
	if result.Repositories[0].PullRequests[0].Action != output.ActionWouldMerge {
		t.Errorf("Action = %v, want %v", result.Repositories[0].PullRequests[0].Action, output.ActionWouldMerge)
	}
}

func TestMergerSkipsFailingChecks(t *testing.T) {
	mock := github.NewMockClient()
	mock.Repositories = []github.Repository{
		{
			Name:          "repo1",
			FullName:      "testorg/repo1",
			DefaultBranch: "main",
			Archived:      false,
		},
	}
	mock.PullRequests["testorg/repo1"] = []github.PullRequest{
		{
			Number:     1,
			Title:      "Bump lodash",
			URL:        "https://github.com/testorg/repo1/pull/1",
			HeadBranch: "dependabot/npm/lodash",
			BaseBranch: "main",
			State:      "open",
			Draft:      false,
			HeadSHA:    "abc123",
			RepoName:   "repo1",
		},
	}
	mock.CheckStatuses["testorg/repo1/abc123"] = &github.CheckStatus{
		AllPassing: false,
		Details:    "CI failed",
	}

	cfg := &config.Config{
		Org:          "testorg",
		SourceBranch: "dependabot/",
		DryRun:       false,
		Rebase:       false,
	}

	m := New(mock, cfg)
	result, err := m.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify no merge was attempted
	if len(mock.MergeCalls) > 0 {
		t.Errorf("Expected no merge calls when checks failing, got %d", len(mock.MergeCalls))
	}

	// Verify skipped
	if result.Summary.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", result.Summary.Skipped)
	}

	if result.Repositories[0].PullRequests[0].Action != output.ActionSkippedChecks {
		t.Errorf("Action = %v, want %v", result.Repositories[0].PullRequests[0].Action, output.ActionSkippedChecks)
	}
}

func TestMergerSkipsOutdatedBranch(t *testing.T) {
	mock := github.NewMockClient()
	mock.Repositories = []github.Repository{
		{
			Name:          "repo1",
			FullName:      "testorg/repo1",
			DefaultBranch: "main",
			Archived:      false,
		},
	}
	mock.PullRequests["testorg/repo1"] = []github.PullRequest{
		{
			Number:     1,
			Title:      "Bump lodash",
			URL:        "https://github.com/testorg/repo1/pull/1",
			HeadBranch: "dependabot/npm/lodash",
			BaseBranch: "main",
			State:      "open",
			Draft:      false,
			HeadSHA:    "abc123",
			RepoName:   "repo1",
		},
	}
	mock.CheckStatuses["testorg/repo1/abc123"] = &github.CheckStatus{
		AllPassing: true,
		Details:    "all checks passing",
	}
	// Branch is behind
	key := fmt.Sprintf("testorg/repo1/%c", rune(1))
	mock.BranchStatuses[key] = &github.BranchStatus{
		UpToDate:    false,
		BehindBy:    5,
		HasConflict: false,
	}

	cfg := &config.Config{
		Org:          "testorg",
		SourceBranch: "dependabot/",
		DryRun:       false,
		Rebase:       false, // Rebase disabled
	}

	m := New(mock, cfg)
	result, err := m.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify no merge was attempted
	if len(mock.MergeCalls) > 0 {
		t.Errorf("Expected no merge calls when branch outdated, got %d", len(mock.MergeCalls))
	}

	// Verify skipped with correct reason
	if result.Summary.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", result.Summary.Skipped)
	}
	if result.Repositories[0].PullRequests[0].Action != output.ActionSkippedOutdated {
		t.Errorf("Action = %v, want %v", result.Repositories[0].PullRequests[0].Action, output.ActionSkippedOutdated)
	}
}

func TestMergerSkipsConflicts(t *testing.T) {
	mock := github.NewMockClient()
	mock.Repositories = []github.Repository{
		{
			Name:          "repo1",
			FullName:      "testorg/repo1",
			DefaultBranch: "main",
			Archived:      false,
		},
	}
	mock.PullRequests["testorg/repo1"] = []github.PullRequest{
		{
			Number:     1,
			Title:      "Bump lodash",
			URL:        "https://github.com/testorg/repo1/pull/1",
			HeadBranch: "dependabot/npm/lodash",
			BaseBranch: "main",
			State:      "open",
			Draft:      false,
			HeadSHA:    "abc123",
			RepoName:   "repo1",
		},
	}
	mock.CheckStatuses["testorg/repo1/abc123"] = &github.CheckStatus{
		AllPassing: true,
		Details:    "all checks passing",
	}
	// Has merge conflict
	key := fmt.Sprintf("testorg/repo1/%c", rune(1))
	mock.BranchStatuses[key] = &github.BranchStatus{
		UpToDate:    true,
		BehindBy:    0,
		HasConflict: true,
	}

	cfg := &config.Config{
		Org:          "testorg",
		SourceBranch: "dependabot/",
		DryRun:       false,
		Rebase:       false,
	}

	m := New(mock, cfg)
	result, err := m.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify skipped with conflict reason
	if result.Summary.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", result.Summary.Skipped)
	}
	if result.Repositories[0].PullRequests[0].Action != output.ActionSkippedConflict {
		t.Errorf("Action = %v, want %v", result.Repositories[0].PullRequests[0].Action, output.ActionSkippedConflict)
	}
}

func TestMergerSkipsArchivedRepos(t *testing.T) {
	mock := github.NewMockClient()
	mock.Repositories = []github.Repository{
		{
			Name:          "archived-repo",
			FullName:      "testorg/archived-repo",
			DefaultBranch: "main",
			Archived:      true,
		},
		{
			Name:          "active-repo",
			FullName:      "testorg/active-repo",
			DefaultBranch: "main",
			Archived:      false,
		},
	}
	mock.PullRequests["testorg/active-repo"] = []github.PullRequest{
		{
			Number:     1,
			Title:      "Bump lodash",
			URL:        "https://github.com/testorg/active-repo/pull/1",
			HeadBranch: "dependabot/npm/lodash",
			BaseBranch: "main",
			State:      "open",
			Draft:      false,
			HeadSHA:    "abc123",
			RepoName:   "active-repo",
		},
	}
	mock.CheckStatuses["testorg/active-repo/abc123"] = &github.CheckStatus{
		AllPassing: true,
		Details:    "all checks passing",
	}

	cfg := &config.Config{
		Org:          "testorg",
		SourceBranch: "dependabot/",
		DryRun:       true,
		Rebase:       false,
	}

	m := New(mock, cfg)
	result, err := m.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Should only have active-repo
	if result.Summary.TotalRepositories != 1 {
		t.Errorf("TotalRepositories = %d, want 1", result.Summary.TotalRepositories)
	}
	if len(result.Repositories) != 1 {
		t.Fatalf("Expected 1 repository, got %d", len(result.Repositories))
	}
	if result.Repositories[0].Name != "active-repo" {
		t.Errorf("Repository name = %v, want active-repo", result.Repositories[0].Name)
	}
}

func TestMergerSkipsDrafts(t *testing.T) {
	mock := github.NewMockClient()
	mock.Repositories = []github.Repository{
		{
			Name:          "repo1",
			FullName:      "testorg/repo1",
			DefaultBranch: "main",
			Archived:      false,
		},
	}
	mock.PullRequests["testorg/repo1"] = []github.PullRequest{
		{
			Number:     1,
			Title:      "Draft PR",
			URL:        "https://github.com/testorg/repo1/pull/1",
			HeadBranch: "dependabot/npm/lodash",
			BaseBranch: "main",
			State:      "open",
			Draft:      true, // This is a draft
			HeadSHA:    "abc123",
			RepoName:   "repo1",
		},
		{
			Number:     2,
			Title:      "Ready PR",
			URL:        "https://github.com/testorg/repo1/pull/2",
			HeadBranch: "dependabot/npm/express",
			BaseBranch: "main",
			State:      "open",
			Draft:      false,
			HeadSHA:    "def456",
			RepoName:   "repo1",
		},
	}
	mock.CheckStatuses["testorg/repo1/def456"] = &github.CheckStatus{
		AllPassing: true,
		Details:    "all checks passing",
	}

	cfg := &config.Config{
		Org:          "testorg",
		SourceBranch: "dependabot/",
		DryRun:       true,
		Rebase:       false,
	}

	m := New(mock, cfg)
	result, err := m.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Should only process non-draft PR
	if result.Summary.TotalPullRequests != 1 {
		t.Errorf("TotalPullRequests = %d, want 1", result.Summary.TotalPullRequests)
	}
	if result.Repositories[0].PullRequests[0].Number != 2 {
		t.Errorf("Expected PR #2, got #%d", result.Repositories[0].PullRequests[0].Number)
	}
}

func TestMergerRepoFilter(t *testing.T) {
	mock := github.NewMockClient()
	mock.Repositories = []github.Repository{
		{
			Name:          "repo1",
			FullName:      "testorg/repo1",
			DefaultBranch: "main",
			Archived:      false,
		},
		{
			Name:          "repo2",
			FullName:      "testorg/repo2",
			DefaultBranch: "main",
			Archived:      false,
		},
		{
			Name:          "repo3",
			FullName:      "testorg/repo3",
			DefaultBranch: "main",
			Archived:      false,
		},
	}

	cfg := &config.Config{
		Org:          "testorg",
		SourceBranch: "dependabot/",
		DryRun:       true,
		Rebase:       false,
		Repos:        []string{"repo1", "repo3"}, // Only these repos
	}

	m := New(mock, cfg)
	result, err := m.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Should only have 2 repos
	if result.Summary.TotalRepositories != 2 {
		t.Errorf("TotalRepositories = %d, want 2", result.Summary.TotalRepositories)
	}

	// Verify correct repos
	repoNames := make(map[string]bool)
	for _, repo := range result.Repositories {
		repoNames[repo.Name] = true
	}
	if !repoNames["repo1"] {
		t.Error("Expected repo1 to be included")
	}
	if repoNames["repo2"] {
		t.Error("Expected repo2 to be excluded")
	}
	if !repoNames["repo3"] {
		t.Error("Expected repo3 to be included")
	}
}

func TestMergerLimit(t *testing.T) {
	mock := github.NewMockClient()
	mock.Repositories = []github.Repository{
		{
			Name:          "repo1",
			FullName:      "testorg/repo1",
			DefaultBranch: "main",
			Archived:      false,
		},
	}
	mock.PullRequests["testorg/repo1"] = []github.PullRequest{
		{
			Number:     1,
			Title:      "PR 1",
			URL:        "https://github.com/testorg/repo1/pull/1",
			HeadBranch: "dependabot/npm/pkg1",
			BaseBranch: "main",
			State:      "open",
			Draft:      false,
			HeadSHA:    "sha1",
			RepoName:   "repo1",
		},
		{
			Number:     2,
			Title:      "PR 2",
			URL:        "https://github.com/testorg/repo1/pull/2",
			HeadBranch: "dependabot/npm/pkg2",
			BaseBranch: "main",
			State:      "open",
			Draft:      false,
			HeadSHA:    "sha2",
			RepoName:   "repo1",
		},
		{
			Number:     3,
			Title:      "PR 3",
			URL:        "https://github.com/testorg/repo1/pull/3",
			HeadBranch: "dependabot/npm/pkg3",
			BaseBranch: "main",
			State:      "open",
			Draft:      false,
			HeadSHA:    "sha3",
			RepoName:   "repo1",
		},
	}
	mock.CheckStatuses["testorg/repo1/sha1"] = &github.CheckStatus{AllPassing: true}
	mock.CheckStatuses["testorg/repo1/sha2"] = &github.CheckStatus{AllPassing: true}
	mock.CheckStatuses["testorg/repo1/sha3"] = &github.CheckStatus{AllPassing: true}

	cfg := &config.Config{
		Org:          "testorg",
		SourceBranch: "dependabot/",
		DryRun:       true,
		Rebase:       false,
		Limit:        2, // Only merge 2
	}

	m := New(mock, cfg)
	result, err := m.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Should have 2 would-merge and 1 skipped due to limit
	if result.Summary.WouldMerge != 2 {
		t.Errorf("WouldMerge = %d, want 2", result.Summary.WouldMerge)
	}
	if result.Summary.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", result.Summary.Skipped)
	}

	// Verify third PR was skipped due to limit
	prs := result.Repositories[0].PullRequests
	if prs[2].Action != output.ActionSkippedLimit {
		t.Errorf("Third PR action = %v, want %v", prs[2].Action, output.ActionSkippedLimit)
	}
}

func TestMergerSourceBranchFiltering(t *testing.T) {
	mock := github.NewMockClient()
	mock.Repositories = []github.Repository{
		{
			Name:          "repo1",
			FullName:      "testorg/repo1",
			DefaultBranch: "main",
			Archived:      false,
		},
	}
	mock.PullRequests["testorg/repo1"] = []github.PullRequest{
		{
			Number:     1,
			Title:      "Dependabot PR",
			URL:        "https://github.com/testorg/repo1/pull/1",
			HeadBranch: "dependabot/npm/lodash",
			BaseBranch: "main",
			State:      "open",
			Draft:      false,
			HeadSHA:    "sha1",
			RepoName:   "repo1",
		},
		{
			Number:     2,
			Title:      "Feature PR",
			URL:        "https://github.com/testorg/repo1/pull/2",
			HeadBranch: "feature/new-feature",
			BaseBranch: "main",
			State:      "open",
			Draft:      false,
			HeadSHA:    "sha2",
			RepoName:   "repo1",
		},
	}
	mock.CheckStatuses["testorg/repo1/sha1"] = &github.CheckStatus{AllPassing: true}

	cfg := &config.Config{
		Org:          "testorg",
		SourceBranch: "dependabot/",
		DryRun:       true,
		Rebase:       false,
	}

	m := New(mock, cfg)
	result, err := m.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Should only process dependabot PR
	if result.Summary.TotalPullRequests != 1 {
		t.Errorf("TotalPullRequests = %d, want 1", result.Summary.TotalPullRequests)
	}
	if result.Repositories[0].PullRequests[0].Number != 1 {
		t.Errorf("Expected PR #1, got #%d", result.Repositories[0].PullRequests[0].Number)
	}
}
