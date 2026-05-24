package merger

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/UnitVectorY-Labs/ghprmerge/internal/config"
	"github.com/UnitVectorY-Labs/ghprmerge/internal/github"
	"github.com/UnitVectorY-Labs/ghprmerge/internal/output"
)

func TestMergerAnalysisOnly(t *testing.T) {
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
		Org:            "testorg",
		SourceBranch:   "dependabot/",
		SourceBranches: []string{"dependabot/"},
		Rebase:         false,
		Merge:          false, // Analysis only mode
	}

	m := New(mock, cfg, nil)
	result, err := m.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify no mutations were made
	if len(mock.MergeCalls) > 0 {
		t.Errorf("Expected no merge calls in analysis mode, got %d", len(mock.MergeCalls))
	}

	// Verify result - should be ReadyToMerge since merge is not enabled
	if result.Summary.ReadyToMerge != 1 {
		t.Errorf("ReadyToMerge = %d, want 1", result.Summary.ReadyToMerge)
	}

	// Verify action is "ready to merge" (not "would merge" since merge is disabled)
	if len(result.Repositories) != 1 {
		t.Fatalf("Expected 1 repository, got %d", len(result.Repositories))
	}
	if len(result.Repositories[0].PullRequests) != 1 {
		t.Fatalf("Expected 1 PR, got %d", len(result.Repositories[0].PullRequests))
	}
	if result.Repositories[0].PullRequests[0].Action != output.ActionReadyMerge {
		t.Errorf("Action = %v, want %v", result.Repositories[0].PullRequests[0].Action, output.ActionReadyMerge)
	}

	// Verify mode description
	if result.Metadata.Mode != "analysis only (no mutations)" {
		t.Errorf("Mode = %v, want 'analysis only (no mutations)'", result.Metadata.Mode)
	}
}

func TestMergerMergeOnly(t *testing.T) {
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
		Org:            "testorg",
		SourceBranch:   "dependabot/",
		SourceBranches: []string{"dependabot/"},
		Rebase:         false,
		Merge:          true, // Merge mode
	}

	m := New(mock, cfg, nil)
	result, err := m.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify merge was called
	if len(mock.MergeCalls) != 1 {
		t.Errorf("Expected 1 merge call, got %d", len(mock.MergeCalls))
	}

	// Verify result
	if result.Summary.MergedSuccess != 1 {
		t.Errorf("MergedSuccess = %d, want 1", result.Summary.MergedSuccess)
	}

	if result.Repositories[0].PullRequests[0].Action != output.ActionMerged {
		t.Errorf("Action = %v, want %v", result.Repositories[0].PullRequests[0].Action, output.ActionMerged)
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
		Details:    "check 'CI' has conclusion 'failure'",
	}

	cfg := &config.Config{
		Org:            "testorg",
		SourceBranch:   "dependabot/",
		SourceBranches: []string{"dependabot/"},
		Merge:          true,
	}

	m := New(mock, cfg, nil)
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

	if result.Repositories[0].PullRequests[0].Action != output.ActionSkipChecksFailing {
		t.Errorf("Action = %v, want %v", result.Repositories[0].PullRequests[0].Action, output.ActionSkipChecksFailing)
	}
}

func TestMergerAllowsMergeWhenNoChecksExist(t *testing.T) {
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
		NoChecks:   true,
		Details:    "no checks found",
	}

	cfg := &config.Config{
		Org:            "testorg",
		SourceBranch:   "dependabot/",
		SourceBranches: []string{"dependabot/"},
		Merge:          true,
	}

	m := New(mock, cfg, nil)
	result, err := m.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(mock.MergeCalls) != 1 {
		t.Fatalf("Expected 1 merge call when no checks exist, got %d", len(mock.MergeCalls))
	}
	if result.Summary.MergedSuccess != 1 {
		t.Fatalf("MergedSuccess = %d, want 1", result.Summary.MergedSuccess)
	}
	if result.Repositories[0].PullRequests[0].Action != output.ActionMerged {
		t.Fatalf("Action = %v, want %v", result.Repositories[0].PullRequests[0].Action, output.ActionMerged)
	}
	if !strings.Contains(result.Repositories[0].PullRequests[0].Reason, "no checks configured") {
		t.Fatalf("expected merge reason to mention missing checks, got %q", result.Repositories[0].PullRequests[0].Reason)
	}
}

func TestMergerConfirmTreatsNoChecksAsPendingMerge(t *testing.T) {
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
		NoChecks:   true,
		Details:    "no checks found",
	}

	cfg := &config.Config{
		Org:            "testorg",
		SourceBranch:   "dependabot/",
		SourceBranches: []string{"dependabot/"},
		Merge:          true,
		Confirm:        true,
	}

	m := New(mock, cfg, nil)
	result, err := m.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.Summary.WouldMerge != 1 {
		t.Fatalf("WouldMerge = %d, want 1", result.Summary.WouldMerge)
	}
	if result.Repositories[0].PullRequests[0].Action != output.ActionWouldMerge {
		t.Fatalf("Action = %v, want %v", result.Repositories[0].PullRequests[0].Action, output.ActionWouldMerge)
	}
	if !strings.Contains(result.Repositories[0].PullRequests[0].Reason, "no checks configured") {
		t.Fatalf("expected pending merge reason to mention missing checks, got %q", result.Repositories[0].PullRequests[0].Reason)
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
		Org:            "testorg",
		SourceBranch:   "dependabot/",
		SourceBranches: []string{"dependabot/"},
		Rebase:         false, // Rebase disabled
		Merge:          true,
	}

	m := New(mock, cfg, nil)
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
	if result.Repositories[0].PullRequests[0].Action != output.ActionSkipBranchBehind {
		t.Errorf("Action = %v, want %v", result.Repositories[0].PullRequests[0].Action, output.ActionSkipBranchBehind)
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
		Org:            "testorg",
		SourceBranch:   "dependabot/",
		SourceBranches: []string{"dependabot/"},
		Merge:          true,
	}

	m := New(mock, cfg, nil)
	result, err := m.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify skipped with conflict reason
	if result.Summary.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", result.Summary.Skipped)
	}
	if result.Repositories[0].PullRequests[0].Action != output.ActionSkipConflict {
		t.Errorf("Action = %v, want %v", result.Repositories[0].PullRequests[0].Action, output.ActionSkipConflict)
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
		Org:            "testorg",
		SourceBranch:   "dependabot/",
		SourceBranches: []string{"dependabot/"},
	}

	m := New(mock, cfg, nil)
	result, err := m.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Should only have active-repo
	if result.Summary.ReposProcessed != 1 {
		t.Errorf("ReposProcessed = %d, want 1", result.Summary.ReposProcessed)
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
		Org:            "testorg",
		SourceBranch:   "dependabot/",
		SourceBranches: []string{"dependabot/"},
	}

	m := New(mock, cfg, nil)
	result, err := m.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Should only process non-draft PR
	if result.Summary.CandidatesFound != 1 {
		t.Errorf("CandidatesFound = %d, want 1", result.Summary.CandidatesFound)
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
		Org:            "testorg",
		SourceBranch:   "dependabot/",
		SourceBranches: []string{"dependabot/"},
		Repos:          []string{"repo1", "repo3"}, // Only these repos
	}

	m := New(mock, cfg, nil)
	result, err := m.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Should only have 2 repos
	if result.Summary.ReposProcessed != 2 {
		t.Errorf("ReposProcessed = %d, want 2", result.Summary.ReposProcessed)
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

func TestMergerRepoLimit(t *testing.T) {
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
		Org:            "testorg",
		SourceBranch:   "dependabot/",
		SourceBranches: []string{"dependabot/"},
		RepoLimit:      2, // Only process 2 repos
	}

	m := New(mock, cfg, nil)
	result, err := m.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Should have 2 processed and 1 skipped
	if result.Summary.ReposProcessed != 2 {
		t.Errorf("ReposProcessed = %d, want 2", result.Summary.ReposProcessed)
	}
	if result.Summary.ReposSkipped != 1 {
		t.Errorf("ReposSkipped = %d, want 1", result.Summary.ReposSkipped)
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
		Org:            "testorg",
		SourceBranch:   "dependabot/",
		SourceBranches: []string{"dependabot/"},
	}

	m := New(mock, cfg, nil)
	result, err := m.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Should only process dependabot PR
	if result.Summary.CandidatesFound != 1 {
		t.Errorf("CandidatesFound = %d, want 1", result.Summary.CandidatesFound)
	}
	if result.Repositories[0].PullRequests[0].Number != 1 {
		t.Errorf("Expected PR #1, got #%d", result.Repositories[0].PullRequests[0].Number)
	}
}

func TestMergerRebaseOnly(t *testing.T) {
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
		BehindBy:    3,
		HasConflict: false,
	}

	cfg := &config.Config{
		Org:            "testorg",
		SourceBranch:   "dependabot/",
		SourceBranches: []string{"dependabot/"},
		Rebase:         true,
		Merge:          false, // Rebase only, no merge
	}

	m := New(mock, cfg, nil)
	result, err := m.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify rebase was done but not merge
	if len(mock.PostRebaseCalls) != 1 {
		t.Errorf("Expected 1 rebase call, got %d", len(mock.PostRebaseCalls))
	}
	if len(mock.MergeCalls) != 0 {
		t.Errorf("Expected 0 merge calls in rebase-only mode, got %d", len(mock.MergeCalls))
	}

	// Verify result
	if result.Summary.RebasedSuccess != 1 {
		t.Errorf("RebasedSuccess = %d, want 1", result.Summary.RebasedSuccess)
	}
	if result.Repositories[0].PullRequests[0].Action != output.ActionRebased {
		t.Errorf("Action = %v, want %v", result.Repositories[0].PullRequests[0].Action, output.ActionRebased)
	}
}

func TestMergerConfirmDefaultDoesNotPrintRepoResultsDuringScan(t *testing.T) {
	mock := github.NewMockClient()
	mock.Repositories = []github.Repository{
		{
			Name:          "repo1",
			FullName:      "testorg/repo1",
			DefaultBranch: "main",
		},
		{
			Name:          "repo2",
			FullName:      "testorg/repo2",
			DefaultBranch: "main",
		},
	}
	mock.PullRequests["testorg/repo1"] = []github.PullRequest{
		{
			Number:     1,
			Title:      "Bump lodash",
			URL:        "https://github.com/testorg/repo1/pull/1",
			HeadBranch: "dependabot/npm/lodash",
			BaseBranch: "main",
			HeadSHA:    "abc123",
			RepoName:   "repo1",
		},
	}
	mock.CheckStatuses["testorg/repo1/abc123"] = &github.CheckStatus{
		AllPassing: true,
		Details:    "all checks passing",
	}

	var buf bytes.Buffer
	cfg := &config.Config{
		Org:            "testorg",
		SourceBranch:   "dependabot/",
		SourceBranches: []string{"dependabot/"},
		Merge:          true,
		Confirm:        true,
	}

	m := New(mock, cfg, output.NewConsole(&buf, true, false, false))
	if _, err := m.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	out := buf.String()
	if strings.Contains(out, "testorg/repo1") {
		t.Fatalf("expected confirm scan to avoid repo result output, got:\n%s", out)
	}
	if strings.Contains(out, "testorg/repo2") {
		t.Fatalf("expected confirm scan to avoid repo result output, got:\n%s", out)
	}
}

func TestMergerVerboseStreamsRepoOutcomesDuringScan(t *testing.T) {
	mock := github.NewMockClient()
	mock.Repositories = []github.Repository{
		{
			Name:          "repo1",
			FullName:      "testorg/repo1",
			DefaultBranch: "main",
		},
		{
			Name:          "repo2",
			FullName:      "testorg/repo2",
			DefaultBranch: "main",
		},
	}
	mock.PullRequests["testorg/repo2"] = []github.PullRequest{
		{
			Number:     1,
			Title:      "Bump lodash",
			URL:        "https://github.com/testorg/repo2/pull/1",
			HeadBranch: "dependabot/npm/lodash",
			BaseBranch: "main",
			HeadSHA:    "abc123",
			RepoName:   "repo2",
		},
	}
	mock.CheckStatuses["testorg/repo2/abc123"] = &github.CheckStatus{
		AllPassing: true,
		Details:    "all checks passing",
	}

	var buf bytes.Buffer
	cfg := &config.Config{
		Org:            "testorg",
		SourceBranch:   "dependabot/",
		SourceBranches: []string{"dependabot/"},
		Verbose:        true,
	}

	m := New(mock, cfg, output.NewConsole(&buf, true, true, false))
	if _, err := m.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "testorg/repo1 ─ no matching pull requests") {
		t.Fatalf("expected verbose scan output for repo without matching PRs, got:\n%s", out)
	}
	if !strings.Contains(out, "testorg/repo2 #1 Bump lodash") {
		t.Fatalf("expected verbose scan output for matching PR, got:\n%s", out)
	}
	if strings.Count(out, "testorg/repo2 #1 Bump lodash") != 1 {
		t.Fatalf("expected live verbose output without duplicate repo results, got:\n%s", out)
	}
}

func TestMergerRunWithActionsVerbosePrintsCompletedActions(t *testing.T) {
	mock := github.NewMockClient()
	mock.Repositories = []github.Repository{
		{
			Name:          "repo1",
			FullName:      "testorg/repo1",
			DefaultBranch: "main",
		},
		{
			Name:          "repo2",
			FullName:      "testorg/repo2",
			DefaultBranch: "main",
		},
	}
	mock.PullRequests["testorg/repo1"] = []github.PullRequest{
		{
			Number:     1,
			Title:      "Bump lodash",
			URL:        "https://github.com/testorg/repo1/pull/1",
			HeadBranch: "dependabot/npm/lodash",
			BaseBranch: "main",
			HeadSHA:    "abc123",
			RepoName:   "repo1",
		},
	}
	mock.CheckStatuses["testorg/repo1/abc123"] = &github.CheckStatus{
		AllPassing: true,
		Details:    "all checks passing",
	}

	var buf bytes.Buffer
	cfg := &config.Config{
		Org:            "testorg",
		SourceBranch:   "dependabot/",
		SourceBranches: []string{"dependabot/"},
		Merge:          true,
		Confirm:        true,
		Verbose:        true,
	}

	m := New(mock, cfg, output.NewConsole(&buf, true, true, false))
	scanResult, err := m.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	buf.Reset()
	result, err := m.RunWithActions(context.Background(), scanResult)
	if err != nil {
		t.Fatalf("RunWithActions() error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "merged") {
		t.Fatalf("expected execution output to show completed action, got:\n%s", out)
	}
	if strings.Contains(out, "would merge") {
		t.Fatalf("expected execution output to replace pending action text, got:\n%s", out)
	}
	if strings.Contains(out, "testorg/repo2") {
		t.Fatalf("expected verbose execution output to omit repos without completed actions, got:\n%s", out)
	}
	if result.Summary.MergedSuccess != 1 {
		t.Fatalf("MergedSuccess = %d, want 1", result.Summary.MergedSuccess)
	}
}

func TestMergerConfirmWithoutPendingActionsPrintsRepoResults(t *testing.T) {
	mock := github.NewMockClient()
	mock.Repositories = []github.Repository{
		{
			Name:          "repo1",
			FullName:      "testorg/repo1",
			DefaultBranch: "main",
		},
	}
	mock.PullRequests["testorg/repo1"] = []github.PullRequest{
		{
			Number:     1,
			Title:      "Bump lodash",
			URL:        "https://github.com/testorg/repo1/pull/1",
			HeadBranch: "dependabot/npm/lodash",
			BaseBranch: "main",
			HeadSHA:    "abc123",
			RepoName:   "repo1",
		},
	}
	mock.CheckStatuses["testorg/repo1/abc123"] = &github.CheckStatus{
		AllPassing: false,
		Details:    "check 'CI' has conclusion 'failure'",
	}

	var buf bytes.Buffer
	cfg := &config.Config{
		Org:            "testorg",
		SourceBranch:   "dependabot/",
		SourceBranches: []string{"dependabot/"},
		Merge:          true,
		Confirm:        true,
	}

	m := New(mock, cfg, output.NewConsole(&buf, true, false, false))
	result, err := m.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "testorg/repo1 #1 Bump lodash") {
		t.Fatalf("expected confirm mode without pending actions to print repo result, got:\n%s", out)
	}
	if !strings.Contains(out, "skip: checks failing") {
		t.Fatalf("expected confirm mode without pending actions to include skip reason, got:\n%s", out)
	}
	if result.Summary.Skipped != 1 {
		t.Fatalf("Skipped = %d, want 1", result.Summary.Skipped)
	}
}

func TestMergerSkipRebaseWithMerge(t *testing.T) {
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
		BehindBy:    3,
		HasConflict: false,
	}

	cfg := &config.Config{
		Org:            "testorg",
		SourceBranch:   "dependabot/",
		SourceBranches: []string{"dependabot/"},
		Rebase:         false,
		Merge:          true,
		SkipRebase:     true, // Skip rebase and merge anyway
	}

	m := New(mock, cfg, nil)
	result, err := m.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify merge was done (not rebase)
	if len(mock.MergeCalls) != 1 {
		t.Errorf("Expected 1 merge call, got %d", len(mock.MergeCalls))
	}
	if len(mock.PostRebaseCalls) != 0 {
		t.Errorf("Expected 0 rebase calls in skip-rebase mode, got %d", len(mock.PostRebaseCalls))
	}
	if len(mock.UpdateBranchCalls) != 0 {
		t.Errorf("Expected 0 update branch calls in skip-rebase mode, got %d", len(mock.UpdateBranchCalls))
	}

	// Verify result
	if result.Summary.MergedSuccess != 1 {
		t.Errorf("MergedSuccess = %d, want 1", result.Summary.MergedSuccess)
	}
	if result.Repositories[0].PullRequests[0].Action != output.ActionMerged {
		t.Errorf("Action = %v, want %v", result.Repositories[0].PullRequests[0].Action, output.ActionMerged)
	}

	// Verify reason mentions branch was behind and rebase was skipped
	reason := result.Repositories[0].PullRequests[0].Reason
	if !strings.Contains(reason, "behind") || !strings.Contains(reason, "skipped") {
		t.Errorf("Reason should mention branch was behind and rebase skipped, got: %s", reason)
	}
}

func TestMergerSkipRebaseWithConflict(t *testing.T) {
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
	// Branch has conflict - should still skip even with skip-rebase
	key := fmt.Sprintf("testorg/repo1/%c", rune(1))
	mock.BranchStatuses[key] = &github.BranchStatus{
		UpToDate:    false,
		BehindBy:    3,
		HasConflict: true,
	}

	cfg := &config.Config{
		Org:            "testorg",
		SourceBranch:   "dependabot/",
		SourceBranches: []string{"dependabot/"},
		Rebase:         false,
		Merge:          true,
		SkipRebase:     true,
	}

	m := New(mock, cfg, nil)
	result, err := m.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify no merge was attempted (conflicts should still block)
	if len(mock.MergeCalls) != 0 {
		t.Errorf("Expected 0 merge calls when there are conflicts, got %d", len(mock.MergeCalls))
	}

	// Verify skipped with conflict reason
	if result.Summary.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", result.Summary.Skipped)
	}
	if result.Repositories[0].PullRequests[0].Action != output.ActionSkipConflict {
		t.Errorf("Action = %v, want %v", result.Repositories[0].PullRequests[0].Action, output.ActionSkipConflict)
	}
}

func TestMergerSkipRebaseWithFailingChecks(t *testing.T) {
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
		Details:    "check 'CI' has conclusion 'failure'",
	}

	cfg := &config.Config{
		Org:            "testorg",
		SourceBranch:   "dependabot/",
		SourceBranches: []string{"dependabot/"},
		Rebase:         false,
		Merge:          true,
		SkipRebase:     true,
	}

	m := New(mock, cfg, nil)
	result, err := m.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify no merge was attempted (failing checks should still block)
	if len(mock.MergeCalls) != 0 {
		t.Errorf("Expected 0 merge calls when checks are failing, got %d", len(mock.MergeCalls))
	}

	// Verify skipped
	if result.Summary.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", result.Summary.Skipped)
	}
	if result.Repositories[0].PullRequests[0].Action != output.ActionSkipChecksFailing {
		t.Errorf("Action = %v, want %v", result.Repositories[0].PullRequests[0].Action, output.ActionSkipChecksFailing)
	}
}

func TestMergerRebaseWithFailingChecks(t *testing.T) {
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
		Details:    "check 'CI' has conclusion 'failure'",
	}
	// Branch is behind
	key := fmt.Sprintf("testorg/repo1/%c", rune(1))
	mock.BranchStatuses[key] = &github.BranchStatus{
		UpToDate:    false,
		BehindBy:    3,
		HasConflict: false,
	}

	cfg := &config.Config{
		Org:            "testorg",
		SourceBranch:   "dependabot/",
		SourceBranches: []string{"dependabot/"},
		Rebase:         true,
		Merge:          false, // Rebase only, no merge
	}

	m := New(mock, cfg, nil)
	result, err := m.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify rebase was done despite failing checks
	if len(mock.PostRebaseCalls) != 1 {
		t.Errorf("Expected 1 rebase call, got %d", len(mock.PostRebaseCalls))
	}
	if len(mock.MergeCalls) != 0 {
		t.Errorf("Expected 0 merge calls in rebase-only mode, got %d", len(mock.MergeCalls))
	}

	// Verify result
	if result.Summary.RebasedSuccess != 1 {
		t.Errorf("RebasedSuccess = %d, want 1", result.Summary.RebasedSuccess)
	}
	if result.Repositories[0].PullRequests[0].Action != output.ActionRebased {
		t.Errorf("Action = %v, want %v", result.Repositories[0].PullRequests[0].Action, output.ActionRebased)
	}
}

func TestMergerMergeModeStreamsActionResultsDuringScan(t *testing.T) {
	mock := github.NewMockClient()
	mock.Repositories = []github.Repository{
		{
			Name:          "repo1",
			FullName:      "testorg/repo1",
			DefaultBranch: "main",
		},
		{
			Name:          "repo2",
			FullName:      "testorg/repo2",
			DefaultBranch: "main",
		},
	}
	mock.PullRequests["testorg/repo1"] = []github.PullRequest{
		{
			Number:     1,
			Title:      "Bump lodash",
			URL:        "https://github.com/testorg/repo1/pull/1",
			HeadBranch: "dependabot/npm/lodash",
			BaseBranch: "main",
			HeadSHA:    "abc123",
			RepoName:   "repo1",
		},
	}
	mock.CheckStatuses["testorg/repo1/abc123"] = &github.CheckStatus{
		AllPassing: true,
		Details:    "all checks passing",
	}

	var buf bytes.Buffer
	cfg := &config.Config{
		Org:            "testorg",
		SourceBranch:   "dependabot/",
		SourceBranches: []string{"dependabot/"},
		Merge:          true,
	}

	m := New(mock, cfg, output.NewConsole(&buf, true, false, false))
	result, err := m.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	out := buf.String()
	// Action result should be streamed during the scan (not deferred)
	if !strings.Contains(out, "testorg/repo1 #1 Bump lodash") {
		t.Fatalf("expected merge action to be streamed during scan, got:\n%s", out)
	}
	if !strings.Contains(out, "merged") {
		t.Fatalf("expected merged status in streamed output, got:\n%s", out)
	}
	// Repo without matching PRs should NOT appear (non-verbose mode)
	if strings.Contains(out, "testorg/repo2") {
		t.Fatalf("expected repo without matching PRs to be omitted in non-verbose mode, got:\n%s", out)
	}
	// Should not be printed twice
	if strings.Count(out, "testorg/repo1 #1 Bump lodash") != 1 {
		t.Fatalf("expected action result to appear exactly once, got:\n%s", out)
	}

	if result.Summary.MergedSuccess != 1 {
		t.Fatalf("MergedSuccess = %d, want 1", result.Summary.MergedSuccess)
	}
}

func TestMergerRunWithActionsStreamsDuringExecution(t *testing.T) {
	mock := github.NewMockClient()
	mock.Repositories = []github.Repository{
		{
			Name:          "repo1",
			FullName:      "testorg/repo1",
			DefaultBranch: "main",
		},
	}
	mock.PullRequests["testorg/repo1"] = []github.PullRequest{
		{
			Number:     1,
			Title:      "Bump lodash",
			URL:        "https://github.com/testorg/repo1/pull/1",
			HeadBranch: "dependabot/npm/lodash",
			BaseBranch: "main",
			HeadSHA:    "abc123",
			RepoName:   "repo1",
		},
	}
	mock.CheckStatuses["testorg/repo1/abc123"] = &github.CheckStatus{
		AllPassing: true,
		Details:    "all checks passing",
	}

	var buf bytes.Buffer
	cfg := &config.Config{
		Org:            "testorg",
		SourceBranch:   "dependabot/",
		SourceBranches: []string{"dependabot/"},
		Merge:          true,
		Confirm:        true,
	}

	// Non-verbose confirm mode: scan first, then execute
	m := New(mock, cfg, output.NewConsole(&buf, true, false, false))
	scanResult, err := m.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	buf.Reset()
	result, err := m.RunWithActions(context.Background(), scanResult)
	if err != nil {
		t.Fatalf("RunWithActions() error = %v", err)
	}

	out := buf.String()
	// Action should be streamed during execution even without verbose
	if !strings.Contains(out, "merged") {
		t.Fatalf("expected execution output to show completed action, got:\n%s", out)
	}
	if strings.Contains(out, "would merge") {
		t.Fatalf("expected execution output to replace pending action text, got:\n%s", out)
	}
	// Should appear exactly once (no duplicate from post-loop printing)
	if strings.Count(out, "testorg/repo1 #1 Bump lodash") != 1 {
		t.Fatalf("expected action result to appear exactly once, got:\n%s", out)
	}

	if result.Summary.MergedSuccess != 1 {
		t.Fatalf("MergedSuccess = %d, want 1", result.Summary.MergedSuccess)
	}
}

func TestMergerRebaseWithPendingChecks(t *testing.T) {
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
		Pending:    true,
		Details:    "check 'CI' is pending",
	}
	// Branch is behind
	key := fmt.Sprintf("testorg/repo1/%c", rune(1))
	mock.BranchStatuses[key] = &github.BranchStatus{
		UpToDate:    false,
		BehindBy:    2,
		HasConflict: false,
	}

	cfg := &config.Config{
		Org:            "testorg",
		SourceBranch:   "dependabot/",
		SourceBranches: []string{"dependabot/"},
		Rebase:         true,
		Merge:          false, // Rebase only, no merge
	}

	m := New(mock, cfg, nil)
	result, err := m.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify rebase was done despite pending checks
	if len(mock.PostRebaseCalls) != 1 {
		t.Errorf("Expected 1 rebase call, got %d", len(mock.PostRebaseCalls))
	}
	if len(mock.MergeCalls) != 0 {
		t.Errorf("Expected 0 merge calls in rebase-only mode, got %d", len(mock.MergeCalls))
	}

	// Verify result
	if result.Summary.RebasedSuccess != 1 {
		t.Errorf("RebasedSuccess = %d, want 1", result.Summary.RebasedSuccess)
	}
	if result.Repositories[0].PullRequests[0].Action != output.ActionRebased {
		t.Errorf("Action = %v, want %v", result.Repositories[0].PullRequests[0].Action, output.ActionRebased)
	}
}

func TestMergerMultipleSourceBranches(t *testing.T) {
	mock := github.NewMockClient()
	mock.Repositories = []github.Repository{
		{
			Name:          "repo-a",
			FullName:      "testorg/repo-a",
			DefaultBranch: "main",
			Archived:      false,
		},
		{
			Name:          "repo-b",
			FullName:      "testorg/repo-b",
			DefaultBranch: "main",
			Archived:      false,
		},
		{
			Name:          "repo-c",
			FullName:      "testorg/repo-c",
			DefaultBranch: "main",
			Archived:      false,
		},
	}
	// repo-a has PRs matching both patterns
	mock.PullRequests["testorg/repo-a"] = []github.PullRequest{
		{
			Number:     1,
			Title:      "Bump lodash",
			URL:        "https://github.com/testorg/repo-a/pull/1",
			HeadBranch: "dependabot/npm/lodash",
			BaseBranch: "main",
			State:      "open",
			Draft:      false,
			HeadSHA:    "sha-a1",
			RepoName:   "repo-a",
		},
		{
			Number:     2,
			Title:      "Repver update",
			URL:        "https://github.com/testorg/repo-a/pull/2",
			HeadBranch: "repver/update-1.0",
			BaseBranch: "main",
			State:      "open",
			Draft:      false,
			HeadSHA:    "sha-a2",
			RepoName:   "repo-a",
		},
	}
	// repo-b has PR matching first pattern only
	mock.PullRequests["testorg/repo-b"] = []github.PullRequest{
		{
			Number:     1,
			Title:      "Bump lodash",
			URL:        "https://github.com/testorg/repo-b/pull/1",
			HeadBranch: "dependabot/npm/lodash",
			BaseBranch: "main",
			State:      "open",
			Draft:      false,
			HeadSHA:    "sha-b1",
			RepoName:   "repo-b",
		},
	}
	// repo-c has PR matching second pattern only
	mock.PullRequests["testorg/repo-c"] = []github.PullRequest{
		{
			Number:     1,
			Title:      "Repver update",
			URL:        "https://github.com/testorg/repo-c/pull/1",
			HeadBranch: "repver/update-1.0",
			BaseBranch: "main",
			State:      "open",
			Draft:      false,
			HeadSHA:    "sha-c1",
			RepoName:   "repo-c",
		},
	}

	// Set up check statuses for all PRs
	mock.CheckStatuses["testorg/repo-a/sha-a1"] = &github.CheckStatus{AllPassing: true, Details: "all checks passing"}
	mock.CheckStatuses["testorg/repo-a/sha-a2"] = &github.CheckStatus{AllPassing: true, Details: "all checks passing"}
	mock.CheckStatuses["testorg/repo-b/sha-b1"] = &github.CheckStatus{AllPassing: true, Details: "all checks passing"}
	mock.CheckStatuses["testorg/repo-c/sha-c1"] = &github.CheckStatus{AllPassing: true, Details: "all checks passing"}

	cfg := &config.Config{
		Org:            "testorg",
		SourceBranch:   "dependabot/",
		SourceBranches: []string{"dependabot/", "repver/"},
		Merge:          true,
	}

	m := New(mock, cfg, nil)
	result, err := m.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// repo-a should only have the dependabot PR (first matching pattern wins;
	// "repver/" is silently skipped because "dependabot/" matched first)
	var repoA, repoB, repoC *output.RepositoryResult
	for i := range result.Repositories {
		switch result.Repositories[i].Name {
		case "repo-a":
			repoA = &result.Repositories[i]
		case "repo-b":
			repoB = &result.Repositories[i]
		case "repo-c":
			repoC = &result.Repositories[i]
		}
	}

	if repoA == nil {
		t.Fatal("repo-a not found in results")
	}
	if len(repoA.PullRequests) != 1 {
		t.Fatalf("repo-a: expected 1 PR (first pattern only), got %d", len(repoA.PullRequests))
	}
	if repoA.PullRequests[0].HeadBranch != "dependabot/npm/lodash" {
		t.Errorf("repo-a: expected PR from dependabot/ pattern, got head branch %q", repoA.PullRequests[0].HeadBranch)
	}

	// repo-b should have the dependabot PR
	if repoB == nil {
		t.Fatal("repo-b not found in results")
	}
	if len(repoB.PullRequests) != 1 {
		t.Fatalf("repo-b: expected 1 PR, got %d", len(repoB.PullRequests))
	}
	if repoB.PullRequests[0].HeadBranch != "dependabot/npm/lodash" {
		t.Errorf("repo-b: expected dependabot PR, got head branch %q", repoB.PullRequests[0].HeadBranch)
	}

	// repo-c should have the repver PR
	if repoC == nil {
		t.Fatal("repo-c not found in results")
	}
	if len(repoC.PullRequests) != 1 {
		t.Fatalf("repo-c: expected 1 PR, got %d", len(repoC.PullRequests))
	}
	if repoC.PullRequests[0].HeadBranch != "repver/update-1.0" {
		t.Errorf("repo-c: expected repver PR, got head branch %q", repoC.PullRequests[0].HeadBranch)
	}

	// Total candidates: 3 (repo-a gets 1, repo-b gets 1, repo-c gets 1)
	// repo-a's repver PR is filtered out by discoverPullRequests
	if result.Summary.CandidatesFound != 3 {
		t.Errorf("CandidatesFound = %d, want 3", result.Summary.CandidatesFound)
	}

	// All 3 should be merged
	if result.Summary.MergedSuccess != 3 {
		t.Errorf("MergedSuccess = %d, want 3", result.Summary.MergedSuccess)
	}
}

func TestMergerMultipleSourceBranchesConcurrentSkip(t *testing.T) {
	mock := github.NewMockClient()
	mock.Repositories = []github.Repository{
		{
			Name:          "repo-a",
			FullName:      "testorg/repo-a",
			DefaultBranch: "main",
			Archived:      false,
		},
	}
	// repo-a has PRs matching two different patterns
	mock.PullRequests["testorg/repo-a"] = []github.PullRequest{
		{
			Number:     1,
			Title:      "Pattern A update",
			URL:        "https://github.com/testorg/repo-a/pull/1",
			HeadBranch: "pattern-a/update-1",
			BaseBranch: "main",
			State:      "open",
			Draft:      false,
			HeadSHA:    "sha-pa1",
			RepoName:   "repo-a",
		},
		{
			Number:     2,
			Title:      "Pattern B update",
			URL:        "https://github.com/testorg/repo-a/pull/2",
			HeadBranch: "pattern-b/update-1",
			BaseBranch: "main",
			State:      "open",
			Draft:      false,
			HeadSHA:    "sha-pb1",
			RepoName:   "repo-a",
		},
	}

	mock.CheckStatuses["testorg/repo-a/sha-pa1"] = &github.CheckStatus{AllPassing: true, Details: "all checks passing"}
	mock.CheckStatuses["testorg/repo-a/sha-pb1"] = &github.CheckStatus{AllPassing: true, Details: "all checks passing"}

	cfg := &config.Config{
		Org:            "testorg",
		SourceBranch:   "pattern-a/",
		SourceBranches: []string{"pattern-a/", "pattern-b/"},
		Merge:          true,
	}

	m := New(mock, cfg, nil)
	result, err := m.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Only the PR matching "pattern-a/" should be discovered; "pattern-b/" is
	// silently skipped because a different pattern already matched this repo.
	if len(result.Repositories) != 1 {
		t.Fatalf("Expected 1 repository, got %d", len(result.Repositories))
	}

	repo := result.Repositories[0]
	if len(repo.PullRequests) != 1 {
		t.Fatalf("Expected 1 PR (pattern-a/ only), got %d", len(repo.PullRequests))
	}
	if repo.PullRequests[0].HeadBranch != "pattern-a/update-1" {
		t.Errorf("Expected PR from pattern-a/, got head branch %q", repo.PullRequests[0].HeadBranch)
	}
	if repo.PullRequests[0].Number != 1 {
		t.Errorf("Expected PR #1, got #%d", repo.PullRequests[0].Number)
	}

	// The pattern-b/ PR should not appear in results at all
	if result.Summary.CandidatesFound != 1 {
		t.Errorf("CandidatesFound = %d, want 1 (pattern-b/ PR should be filtered out)", result.Summary.CandidatesFound)
	}

	if result.Summary.MergedSuccess != 1 {
		t.Errorf("MergedSuccess = %d, want 1", result.Summary.MergedSuccess)
	}
}
