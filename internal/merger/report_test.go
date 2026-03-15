package merger

import (
	"context"
	"testing"

	"github.com/UnitVectorY-Labs/ghprmerge/internal/config"
	gh "github.com/UnitVectorY-Labs/ghprmerge/internal/github"
)

func TestRunReportGroupsByExactBranch(t *testing.T) {
	mock := gh.NewMockClient()
	mock.Repositories = []gh.Repository{
		{Name: "repo-a", FullName: "myorg/repo-a", DefaultBranch: "main"},
		{Name: "repo-b", FullName: "myorg/repo-b", DefaultBranch: "main"},
		{Name: "repo-c", FullName: "myorg/repo-c", DefaultBranch: "main"},
	}
	mock.PullRequests["myorg/repo-a"] = []gh.PullRequest{
		{Number: 1, Title: "Bump foo", HeadBranch: "dependabot/go_modules/foo-1.2.3", BaseBranch: "main", HeadSHA: "sha1", RepoFullName: "myorg/repo-a"},
		{Number: 2, Title: "Bump bar", HeadBranch: "dependabot/go_modules/bar-4.5.6", BaseBranch: "main", HeadSHA: "sha2", RepoFullName: "myorg/repo-a"},
	}
	mock.PullRequests["myorg/repo-b"] = []gh.PullRequest{
		{Number: 3, Title: "Bump foo", HeadBranch: "dependabot/go_modules/foo-1.2.3", BaseBranch: "main", HeadSHA: "sha3", RepoFullName: "myorg/repo-b"},
	}
	mock.PullRequests["myorg/repo-c"] = []gh.PullRequest{
		{Number: 4, Title: "Bump foo", HeadBranch: "dependabot/go_modules/foo-1.2.3", BaseBranch: "main", HeadSHA: "sha4", RepoFullName: "myorg/repo-c"},
		{Number: 5, Title: "Bump bar", HeadBranch: "dependabot/go_modules/bar-4.5.6", BaseBranch: "main", HeadSHA: "sha5", RepoFullName: "myorg/repo-c"},
	}

	cfg := &config.Config{
		Org:          "myorg",
		Report:       true,
		MinGroupSize: 2,
		JSON:         true, // suppress console output
	}

	m := New(mock, cfg, nil)
	result, err := m.RunReport(context.Background())
	if err != nil {
		t.Fatalf("RunReport() error = %v", err)
	}

	// Should have 2 groups: foo (3 PRs) and bar (2 PRs)
	if len(result.Groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(result.Groups))
	}

	// First group should be foo (3 PRs) - higher count
	if result.Groups[0].SourceBranch != "dependabot/go_modules/foo-1.2.3" {
		t.Errorf("expected first group branch = dependabot/go_modules/foo-1.2.3, got %s", result.Groups[0].SourceBranch)
	}
	if result.Groups[0].Count != 3 {
		t.Errorf("expected first group count = 3, got %d", result.Groups[0].Count)
	}

	// Second group should be bar (2 PRs)
	if result.Groups[1].SourceBranch != "dependabot/go_modules/bar-4.5.6" {
		t.Errorf("expected second group branch = dependabot/go_modules/bar-4.5.6, got %s", result.Groups[1].SourceBranch)
	}
	if result.Groups[1].Count != 2 {
		t.Errorf("expected second group count = 2, got %d", result.Groups[1].Count)
	}
}

func TestRunReportMinGroupSize(t *testing.T) {
	mock := gh.NewMockClient()
	mock.Repositories = []gh.Repository{
		{Name: "repo-a", FullName: "myorg/repo-a", DefaultBranch: "main"},
		{Name: "repo-b", FullName: "myorg/repo-b", DefaultBranch: "main"},
	}
	mock.PullRequests["myorg/repo-a"] = []gh.PullRequest{
		{Number: 1, Title: "Bump foo", HeadBranch: "dependabot/foo-1.0", BaseBranch: "main", HeadSHA: "sha1", RepoFullName: "myorg/repo-a"},
		{Number: 2, Title: "Unique branch", HeadBranch: "feature/unique", BaseBranch: "main", HeadSHA: "sha2", RepoFullName: "myorg/repo-a"},
	}
	mock.PullRequests["myorg/repo-b"] = []gh.PullRequest{
		{Number: 3, Title: "Bump foo", HeadBranch: "dependabot/foo-1.0", BaseBranch: "main", HeadSHA: "sha3", RepoFullName: "myorg/repo-b"},
	}

	cfg := &config.Config{
		Org:          "myorg",
		Report:       true,
		MinGroupSize: 2,
		JSON:         true,
	}

	m := New(mock, cfg, nil)
	result, err := m.RunReport(context.Background())
	if err != nil {
		t.Fatalf("RunReport() error = %v", err)
	}

	// Only the foo group (2 PRs) should be included, not unique (1 PR)
	if len(result.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(result.Groups))
	}
	if result.Groups[0].SourceBranch != "dependabot/foo-1.0" {
		t.Errorf("expected group branch = dependabot/foo-1.0, got %s", result.Groups[0].SourceBranch)
	}
}

func TestRunReportMinGroupSizeThree(t *testing.T) {
	mock := gh.NewMockClient()
	mock.Repositories = []gh.Repository{
		{Name: "repo-a", FullName: "myorg/repo-a", DefaultBranch: "main"},
		{Name: "repo-b", FullName: "myorg/repo-b", DefaultBranch: "main"},
	}
	mock.PullRequests["myorg/repo-a"] = []gh.PullRequest{
		{Number: 1, HeadBranch: "branch-a", BaseBranch: "main", HeadSHA: "sha1", RepoFullName: "myorg/repo-a"},
	}
	mock.PullRequests["myorg/repo-b"] = []gh.PullRequest{
		{Number: 2, HeadBranch: "branch-a", BaseBranch: "main", HeadSHA: "sha2", RepoFullName: "myorg/repo-b"},
	}

	cfg := &config.Config{
		Org:          "myorg",
		Report:       true,
		MinGroupSize: 3,
		JSON:         true,
	}

	m := New(mock, cfg, nil)
	result, err := m.RunReport(context.Background())
	if err != nil {
		t.Fatalf("RunReport() error = %v", err)
	}

	// Group has 2 PRs but min is 3, so no groups
	if len(result.Groups) != 0 {
		t.Fatalf("expected 0 groups, got %d", len(result.Groups))
	}
}

func TestRunReportSourceBranchPrefix(t *testing.T) {
	mock := gh.NewMockClient()
	mock.Repositories = []gh.Repository{
		{Name: "repo-a", FullName: "myorg/repo-a", DefaultBranch: "main"},
		{Name: "repo-b", FullName: "myorg/repo-b", DefaultBranch: "main"},
	}
	mock.PullRequests["myorg/repo-a"] = []gh.PullRequest{
		{Number: 1, HeadBranch: "dependabot/foo-1.0", BaseBranch: "main", HeadSHA: "sha1", RepoFullName: "myorg/repo-a"},
		{Number: 2, HeadBranch: "feature/bar", BaseBranch: "main", HeadSHA: "sha2", RepoFullName: "myorg/repo-a"},
	}
	mock.PullRequests["myorg/repo-b"] = []gh.PullRequest{
		{Number: 3, HeadBranch: "dependabot/foo-1.0", BaseBranch: "main", HeadSHA: "sha3", RepoFullName: "myorg/repo-b"},
		{Number: 4, HeadBranch: "feature/bar", BaseBranch: "main", HeadSHA: "sha4", RepoFullName: "myorg/repo-b"},
	}

	cfg := &config.Config{
		Org:                "myorg",
		Report:             true,
		MinGroupSize:       2,
		SourceBranchPrefix: []string{"dependabot/"},
		JSON:               true,
	}

	m := New(mock, cfg, nil)
	result, err := m.RunReport(context.Background())
	if err != nil {
		t.Fatalf("RunReport() error = %v", err)
	}

	// Only dependabot/ PRs should be included
	if len(result.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(result.Groups))
	}
	if result.Groups[0].SourceBranch != "dependabot/foo-1.0" {
		t.Errorf("expected group branch = dependabot/foo-1.0, got %s", result.Groups[0].SourceBranch)
	}
}

func TestRunReportMultiplePrefixes(t *testing.T) {
	mock := gh.NewMockClient()
	mock.Repositories = []gh.Repository{
		{Name: "repo-a", FullName: "myorg/repo-a", DefaultBranch: "main"},
		{Name: "repo-b", FullName: "myorg/repo-b", DefaultBranch: "main"},
	}
	mock.PullRequests["myorg/repo-a"] = []gh.PullRequest{
		{Number: 1, HeadBranch: "dependabot/foo-1.0", BaseBranch: "main", HeadSHA: "sha1", RepoFullName: "myorg/repo-a"},
		{Number: 2, HeadBranch: "repver/bar-2.0", BaseBranch: "main", HeadSHA: "sha2", RepoFullName: "myorg/repo-a"},
		{Number: 3, HeadBranch: "feature/baz", BaseBranch: "main", HeadSHA: "sha3", RepoFullName: "myorg/repo-a"},
	}
	mock.PullRequests["myorg/repo-b"] = []gh.PullRequest{
		{Number: 4, HeadBranch: "dependabot/foo-1.0", BaseBranch: "main", HeadSHA: "sha4", RepoFullName: "myorg/repo-b"},
		{Number: 5, HeadBranch: "repver/bar-2.0", BaseBranch: "main", HeadSHA: "sha5", RepoFullName: "myorg/repo-b"},
		{Number: 6, HeadBranch: "feature/baz", BaseBranch: "main", HeadSHA: "sha6", RepoFullName: "myorg/repo-b"},
	}

	cfg := &config.Config{
		Org:                "myorg",
		Report:             true,
		MinGroupSize:       2,
		SourceBranchPrefix: []string{"dependabot/", "repver/"},
		JSON:               true,
	}

	m := New(mock, cfg, nil)
	result, err := m.RunReport(context.Background())
	if err != nil {
		t.Fatalf("RunReport() error = %v", err)
	}

	// Should have 2 groups: dependabot/foo-1.0 and repver/bar-2.0, but not feature/baz
	if len(result.Groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(result.Groups))
	}
}

func TestRunReportSortOrder(t *testing.T) {
	mock := gh.NewMockClient()
	mock.Repositories = []gh.Repository{
		{Name: "repo-a", FullName: "myorg/repo-a", DefaultBranch: "main"},
		{Name: "repo-b", FullName: "myorg/repo-b", DefaultBranch: "main"},
		{Name: "repo-c", FullName: "myorg/repo-c", DefaultBranch: "main"},
	}
	// branch-b appears in all 3 repos, branch-a in 2, branch-c in 2
	mock.PullRequests["myorg/repo-a"] = []gh.PullRequest{
		{Number: 1, HeadBranch: "branch-a", BaseBranch: "main", HeadSHA: "sha1", RepoFullName: "myorg/repo-a"},
		{Number: 2, HeadBranch: "branch-b", BaseBranch: "main", HeadSHA: "sha2", RepoFullName: "myorg/repo-a"},
		{Number: 3, HeadBranch: "branch-c", BaseBranch: "main", HeadSHA: "sha3", RepoFullName: "myorg/repo-a"},
	}
	mock.PullRequests["myorg/repo-b"] = []gh.PullRequest{
		{Number: 4, HeadBranch: "branch-a", BaseBranch: "main", HeadSHA: "sha4", RepoFullName: "myorg/repo-b"},
		{Number: 5, HeadBranch: "branch-b", BaseBranch: "main", HeadSHA: "sha5", RepoFullName: "myorg/repo-b"},
		{Number: 6, HeadBranch: "branch-c", BaseBranch: "main", HeadSHA: "sha6", RepoFullName: "myorg/repo-b"},
	}
	mock.PullRequests["myorg/repo-c"] = []gh.PullRequest{
		{Number: 7, HeadBranch: "branch-b", BaseBranch: "main", HeadSHA: "sha7", RepoFullName: "myorg/repo-c"},
	}

	cfg := &config.Config{
		Org:          "myorg",
		Report:       true,
		MinGroupSize: 2,
		JSON:         true,
	}

	m := New(mock, cfg, nil)
	result, err := m.RunReport(context.Background())
	if err != nil {
		t.Fatalf("RunReport() error = %v", err)
	}

	// Should have 3 groups: branch-b (3), branch-a (2), branch-c (2)
	// Ties broken by branch name ascending
	if len(result.Groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(result.Groups))
	}
	if result.Groups[0].SourceBranch != "branch-b" || result.Groups[0].Count != 3 {
		t.Errorf("expected first group: branch-b (3), got %s (%d)", result.Groups[0].SourceBranch, result.Groups[0].Count)
	}
	if result.Groups[1].SourceBranch != "branch-a" || result.Groups[1].Count != 2 {
		t.Errorf("expected second group: branch-a (2), got %s (%d)", result.Groups[1].SourceBranch, result.Groups[1].Count)
	}
	if result.Groups[2].SourceBranch != "branch-c" || result.Groups[2].Count != 2 {
		t.Errorf("expected third group: branch-c (2), got %s (%d)", result.Groups[2].SourceBranch, result.Groups[2].Count)
	}
}

func TestRunReportSkipsDraftPRs(t *testing.T) {
	mock := gh.NewMockClient()
	mock.Repositories = []gh.Repository{
		{Name: "repo-a", FullName: "myorg/repo-a", DefaultBranch: "main"},
		{Name: "repo-b", FullName: "myorg/repo-b", DefaultBranch: "main"},
	}
	mock.PullRequests["myorg/repo-a"] = []gh.PullRequest{
		{Number: 1, HeadBranch: "branch-a", BaseBranch: "main", HeadSHA: "sha1", RepoFullName: "myorg/repo-a", Draft: true},
	}
	mock.PullRequests["myorg/repo-b"] = []gh.PullRequest{
		{Number: 2, HeadBranch: "branch-a", BaseBranch: "main", HeadSHA: "sha2", RepoFullName: "myorg/repo-b"},
	}

	cfg := &config.Config{
		Org:          "myorg",
		Report:       true,
		MinGroupSize: 2,
		JSON:         true,
	}

	m := New(mock, cfg, nil)
	result, err := m.RunReport(context.Background())
	if err != nil {
		t.Fatalf("RunReport() error = %v", err)
	}

	// Draft PR should be skipped, leaving only 1 PR for branch-a (below min)
	if len(result.Groups) != 0 {
		t.Fatalf("expected 0 groups (draft filtered out), got %d", len(result.Groups))
	}
}

func TestRunReportEmptyResult(t *testing.T) {
	mock := gh.NewMockClient()
	mock.Repositories = []gh.Repository{
		{Name: "repo-a", FullName: "myorg/repo-a", DefaultBranch: "main"},
	}
	mock.PullRequests["myorg/repo-a"] = []gh.PullRequest{}

	cfg := &config.Config{
		Org:          "myorg",
		Report:       true,
		MinGroupSize: 2,
		JSON:         true,
	}

	m := New(mock, cfg, nil)
	result, err := m.RunReport(context.Background())
	if err != nil {
		t.Fatalf("RunReport() error = %v", err)
	}

	if len(result.Groups) != 0 {
		t.Fatalf("expected 0 groups, got %d", len(result.Groups))
	}
}

func TestRunReportRepoLimit(t *testing.T) {
	mock := gh.NewMockClient()
	mock.Repositories = []gh.Repository{
		{Name: "repo-a", FullName: "myorg/repo-a", DefaultBranch: "main"},
		{Name: "repo-b", FullName: "myorg/repo-b", DefaultBranch: "main"},
		{Name: "repo-c", FullName: "myorg/repo-c", DefaultBranch: "main"},
	}
	mock.PullRequests["myorg/repo-a"] = []gh.PullRequest{
		{Number: 1, HeadBranch: "branch-x", BaseBranch: "main", HeadSHA: "sha1", RepoFullName: "myorg/repo-a"},
	}
	mock.PullRequests["myorg/repo-b"] = []gh.PullRequest{
		{Number: 2, HeadBranch: "branch-x", BaseBranch: "main", HeadSHA: "sha2", RepoFullName: "myorg/repo-b"},
	}
	mock.PullRequests["myorg/repo-c"] = []gh.PullRequest{
		{Number: 3, HeadBranch: "branch-x", BaseBranch: "main", HeadSHA: "sha3", RepoFullName: "myorg/repo-c"},
	}

	cfg := &config.Config{
		Org:          "myorg",
		Report:       true,
		MinGroupSize: 2,
		RepoLimit:    2,
		JSON:         true,
	}

	m := New(mock, cfg, nil)
	result, err := m.RunReport(context.Background())
	if err != nil {
		t.Fatalf("RunReport() error = %v", err)
	}

	// With repo limit of 2, only 2 repos should be scanned
	// branch-x appears in those 2 repos = group of 2
	if len(result.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(result.Groups))
	}
	if result.Groups[0].Count != 2 {
		t.Errorf("expected group count = 2 (repo limit), got %d", result.Groups[0].Count)
	}
}

func TestRunReportRepoFilter(t *testing.T) {
	mock := gh.NewMockClient()
	mock.Repositories = []gh.Repository{
		{Name: "repo-a", FullName: "myorg/repo-a", DefaultBranch: "main"},
		{Name: "repo-b", FullName: "myorg/repo-b", DefaultBranch: "main"},
		{Name: "repo-c", FullName: "myorg/repo-c", DefaultBranch: "main"},
	}
	mock.PullRequests["myorg/repo-a"] = []gh.PullRequest{
		{Number: 1, HeadBranch: "branch-x", BaseBranch: "main", HeadSHA: "sha1", RepoFullName: "myorg/repo-a"},
	}
	mock.PullRequests["myorg/repo-b"] = []gh.PullRequest{
		{Number: 2, HeadBranch: "branch-x", BaseBranch: "main", HeadSHA: "sha2", RepoFullName: "myorg/repo-b"},
	}
	mock.PullRequests["myorg/repo-c"] = []gh.PullRequest{
		{Number: 3, HeadBranch: "branch-x", BaseBranch: "main", HeadSHA: "sha3", RepoFullName: "myorg/repo-c"},
	}

	cfg := &config.Config{
		Org:          "myorg",
		Report:       true,
		MinGroupSize: 2,
		Repos:        []string{"repo-a", "repo-b"},
		JSON:         true,
	}

	m := New(mock, cfg, nil)
	result, err := m.RunReport(context.Background())
	if err != nil {
		t.Fatalf("RunReport() error = %v", err)
	}

	// With --repo filter, only repo-a and repo-b are scanned
	if len(result.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(result.Groups))
	}
	if result.Groups[0].Count != 2 {
		t.Errorf("expected group count = 2 (filtered repos), got %d", result.Groups[0].Count)
	}
}

func TestRunReportStatusEvaluation(t *testing.T) {
	mock := gh.NewMockClient()
	mock.Repositories = []gh.Repository{
		{Name: "repo-a", FullName: "myorg/repo-a", DefaultBranch: "main"},
		{Name: "repo-b", FullName: "myorg/repo-b", DefaultBranch: "main"},
	}
	mock.PullRequests["myorg/repo-a"] = []gh.PullRequest{
		{Number: 1, HeadBranch: "branch-x", BaseBranch: "main", HeadSHA: "sha-passing", RepoFullName: "myorg/repo-a"},
	}
	mock.PullRequests["myorg/repo-b"] = []gh.PullRequest{
		{Number: 2, HeadBranch: "branch-x", BaseBranch: "main", HeadSHA: "sha-failing", RepoFullName: "myorg/repo-b"},
	}

	// Set check statuses
	mock.CheckStatuses["myorg/repo-a/sha-passing"] = &gh.CheckStatus{AllPassing: true, Details: "all passing"}
	mock.CheckStatuses["myorg/repo-b/sha-failing"] = &gh.CheckStatus{AllPassing: false, Pending: false, Details: "tests failed"}

	cfg := &config.Config{
		Org:          "myorg",
		Report:       true,
		MinGroupSize: 2,
		JSON:         true,
	}

	m := New(mock, cfg, nil)
	result, err := m.RunReport(context.Background())
	if err != nil {
		t.Fatalf("RunReport() error = %v", err)
	}

	if len(result.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(result.Groups))
	}

	// Check status is evaluated
	for _, pr := range result.Groups[0].PullRequests {
		if pr.Repository == "repo-a" && pr.Status != "passing" {
			t.Errorf("expected repo-a PR status = passing, got %s", pr.Status)
		}
		if pr.Repository == "repo-b" && pr.Status != "checks failing" {
			t.Errorf("expected repo-b PR status = checks failing, got %s", pr.Status)
		}
	}
}

func TestRunReportNonDefaultBranchFiltered(t *testing.T) {
	mock := gh.NewMockClient()
	mock.Repositories = []gh.Repository{
		{Name: "repo-a", FullName: "myorg/repo-a", DefaultBranch: "main"},
		{Name: "repo-b", FullName: "myorg/repo-b", DefaultBranch: "main"},
	}
	mock.PullRequests["myorg/repo-a"] = []gh.PullRequest{
		{Number: 1, HeadBranch: "branch-x", BaseBranch: "develop", HeadSHA: "sha1", RepoFullName: "myorg/repo-a"},
	}
	mock.PullRequests["myorg/repo-b"] = []gh.PullRequest{
		{Number: 2, HeadBranch: "branch-x", BaseBranch: "main", HeadSHA: "sha2", RepoFullName: "myorg/repo-b"},
	}

	cfg := &config.Config{
		Org:          "myorg",
		Report:       true,
		MinGroupSize: 2,
		JSON:         true,
	}

	m := New(mock, cfg, nil)
	result, err := m.RunReport(context.Background())
	if err != nil {
		t.Fatalf("RunReport() error = %v", err)
	}

	// PR targeting develop should be filtered out, leaving only 1 PR
	if len(result.Groups) != 0 {
		t.Fatalf("expected 0 groups (non-default branch filtered), got %d", len(result.Groups))
	}
}
