// Package github provides interfaces and implementations for interacting with GitHub API.
package github

import (
	"context"
)

// MockClient is a mock implementation of the Client interface for testing.
type MockClient struct {
	Repositories    []Repository
	PullRequests    map[string][]PullRequest // key: "owner/repo"
	CheckStatuses   map[string]*CheckStatus  // key: "owner/repo/ref"
	BranchStatuses  map[string]*BranchStatus // key: "owner/repo/prNumber"
	UpdateBranchErr map[string]error         // key: "owner/repo/prNumber"
	PostRebaseErr   map[string]error         // key: "owner/repo/prNumber"
	MergeErr        map[string]error         // key: "owner/repo/prNumber"
	ListReposErr    error
	ListPRsErr      map[string]error // key: "owner/repo"
	GetPRErr        map[string]error // key: "owner/repo/prNumber"

	// Track calls for verification
	UpdateBranchCalls []string
	PostRebaseCalls   []string
	MergeCalls        []string
}

// NewMockClient creates a new MockClient with initialized maps.
func NewMockClient() *MockClient {
	return &MockClient{
		Repositories:      []Repository{},
		PullRequests:      make(map[string][]PullRequest),
		CheckStatuses:     make(map[string]*CheckStatus),
		BranchStatuses:    make(map[string]*BranchStatus),
		UpdateBranchErr:   make(map[string]error),
		PostRebaseErr:     make(map[string]error),
		MergeErr:          make(map[string]error),
		ListPRsErr:        make(map[string]error),
		GetPRErr:          make(map[string]error),
		UpdateBranchCalls: []string{},
		PostRebaseCalls:   []string{},
		MergeCalls:        []string{},
	}
}

// ListRepositories returns mock repositories, excluding archived ones.
func (m *MockClient) ListRepositories(ctx context.Context, org string) ([]Repository, error) {
	if m.ListReposErr != nil {
		return nil, m.ListReposErr
	}
	var repos []Repository
	for _, repo := range m.Repositories {
		if !repo.Archived {
			repos = append(repos, repo)
		}
	}
	return repos, nil
}

// ListPullRequests returns mock pull requests.
func (m *MockClient) ListPullRequests(ctx context.Context, owner, repo, defaultBranch string) ([]PullRequest, error) {
	key := owner + "/" + repo
	if err, ok := m.ListPRsErr[key]; ok && err != nil {
		return nil, err
	}
	return m.PullRequests[key], nil
}

// GetPullRequest returns a mock pull request.
func (m *MockClient) GetPullRequest(ctx context.Context, owner, repo string, number int) (*PullRequest, error) {
	key := owner + "/" + repo
	if err, ok := m.GetPRErr[key]; ok && err != nil {
		return nil, err
	}
	prs := m.PullRequests[key]
	for _, pr := range prs {
		if pr.Number == number {
			return &pr, nil
		}
	}
	return nil, nil
}

// GetCheckStatus returns mock check status.
func (m *MockClient) GetCheckStatus(ctx context.Context, owner, repo, ref string) (*CheckStatus, error) {
	key := owner + "/" + repo + "/" + ref
	if status, ok := m.CheckStatuses[key]; ok {
		return status, nil
	}
	return &CheckStatus{AllPassing: true, Details: "all checks passing"}, nil
}

// GetBranchStatus returns mock branch status.
func (m *MockClient) GetBranchStatus(ctx context.Context, owner, repo string, prNumber int) (*BranchStatus, error) {
	key := owner + "/" + repo + "/" + string(rune(prNumber))
	if status, ok := m.BranchStatuses[key]; ok {
		return status, nil
	}
	return &BranchStatus{UpToDate: true, BehindBy: 0, HasConflict: false}, nil
}

// UpdateBranch mocks updating a branch.
func (m *MockClient) UpdateBranch(ctx context.Context, owner, repo string, prNumber int) error {
	key := owner + "/" + repo + "/" + string(rune(prNumber))
	m.UpdateBranchCalls = append(m.UpdateBranchCalls, key)
	if err, ok := m.UpdateBranchErr[key]; ok {
		return err
	}
	return nil
}

// PostRebaseComment mocks posting a rebase comment.
func (m *MockClient) PostRebaseComment(ctx context.Context, owner, repo string, prNumber int) error {
	key := owner + "/" + repo + "/" + string(rune(prNumber))
	m.PostRebaseCalls = append(m.PostRebaseCalls, key)
	if err, ok := m.PostRebaseErr[key]; ok {
		return err
	}
	return nil
}

// MergePullRequest mocks merging a pull request.
func (m *MockClient) MergePullRequest(ctx context.Context, owner, repo string, prNumber int) error {
	key := owner + "/" + repo + "/" + string(rune(prNumber))
	m.MergeCalls = append(m.MergeCalls, key)
	if err, ok := m.MergeErr[key]; ok {
		return err
	}
	return nil
}
