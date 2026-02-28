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
		Org:          "testorg",
		SourceBranch: "dependabot/",
		Rebase:       false,
		Merge:        false, // Analysis only mode
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
		Org:          "testorg",
		SourceBranch: "dependabot/",
		Rebase:       false,
		Merge:        true, // Merge mode
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
		Org:          "testorg",
		SourceBranch: "dependabot/",
		Merge:        true,
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
		Org:          "testorg",
		SourceBranch: "dependabot/",
		Merge:        true,
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
		Org:          "testorg",
		SourceBranch: "dependabot/",
		Merge:        true,
		Confirm:      true,
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
		Org:          "testorg",
		SourceBranch: "dependabot/",
		Rebase:       false, // Rebase disabled
		Merge:        true,
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
		Org:          "testorg",
		SourceBranch: "dependabot/",
		Merge:        true,
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
		Org:          "testorg",
		SourceBranch: "dependabot/",
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
		Org:          "testorg",
		SourceBranch: "dependabot/",
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
		Org:          "testorg",
		SourceBranch: "dependabot/",
		Repos:        []string{"repo1", "repo3"}, // Only these repos
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
		Org:          "testorg",
		SourceBranch: "dependabot/",
		RepoLimit:    2, // Only process 2 repos
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
		Org:          "testorg",
		SourceBranch: "dependabot/",
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
		Org:          "testorg",
		SourceBranch: "dependabot/",
		Rebase:       true,
		Merge:        false, // Rebase only, no merge
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
		Org:          "testorg",
		SourceBranch: "dependabot/",
		Merge:        true,
		Confirm:      true,
	}

	m := New(mock, cfg, output.NewConsole(&buf, true, false))
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
		Org:          "testorg",
		SourceBranch: "dependabot/",
		Verbose:      true,
	}

	m := New(mock, cfg, output.NewConsole(&buf, true, true))
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
		Org:          "testorg",
		SourceBranch: "dependabot/",
		Merge:        true,
		Confirm:      true,
		Verbose:      true,
	}

	m := New(mock, cfg, output.NewConsole(&buf, true, true))
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
		Org:          "testorg",
		SourceBranch: "dependabot/",
		Merge:        true,
		Confirm:      true,
	}

	m := New(mock, cfg, output.NewConsole(&buf, true, false))
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
		Org:          "testorg",
		SourceBranch: "dependabot/",
		Rebase:       false,
		Merge:        true,
		SkipRebase:   true, // Skip rebase and merge anyway
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
		Org:          "testorg",
		SourceBranch: "dependabot/",
		Rebase:       false,
		Merge:        true,
		SkipRebase:   true,
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
		Org:          "testorg",
		SourceBranch: "dependabot/",
		Rebase:       false,
		Merge:        true,
		SkipRebase:   true,
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
		Org:          "testorg",
		SourceBranch: "dependabot/",
		Rebase:       true,
		Merge:        false, // Rebase only, no merge
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
		Org:          "testorg",
		SourceBranch: "dependabot/",
		Rebase:       true,
		Merge:        false, // Rebase only, no merge
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
