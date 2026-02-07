// Package github provides interfaces and implementations for interacting with GitHub API.
package github

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"
)

// Repository represents a GitHub repository.
type Repository struct {
	Name          string
	FullName      string
	DefaultBranch string
	Archived      bool
}

// PullRequest represents a GitHub pull request.
type PullRequest struct {
	Number       int
	Title        string
	URL          string
	HeadBranch   string
	BaseBranch   string
	State        string
	Draft        bool
	Mergeable    *bool
	HeadSHA      string
	RepoName     string
	RepoFullName string
}

// CheckStatus represents the overall status of checks on a commit.
type CheckStatus struct {
	AllPassing bool
	Pending    bool
	NoChecks   bool
	Details    string
}

// BranchStatus represents the status of a branch relative to its base.
type BranchStatus struct {
	UpToDate    bool
	BehindBy    int
	HasConflict bool
}

// ActionResult represents the result of an action on a pull request.
type ActionResult struct {
	Action  string
	Success bool
	Error   error
}

// Client defines the interface for GitHub API operations.
type Client interface {
	// ListRepositories lists all repositories in an organization.
	ListRepositories(ctx context.Context, org string) ([]Repository, error)

	// ListPullRequests lists open pull requests for a repository targeting its default branch.
	ListPullRequests(ctx context.Context, owner, repo, defaultBranch string) ([]PullRequest, error)

	// GetPullRequest gets detailed information about a pull request.
	GetPullRequest(ctx context.Context, owner, repo string, number int) (*PullRequest, error)

	// GetCheckStatus gets the check status for a commit.
	GetCheckStatus(ctx context.Context, owner, repo, ref string) (*CheckStatus, error)

	// GetBranchStatus gets the status of a PR branch relative to its base.
	GetBranchStatus(ctx context.Context, owner, repo string, prNumber int) (*BranchStatus, error)

	// UpdateBranch updates a pull request branch with the base branch.
	UpdateBranch(ctx context.Context, owner, repo string, prNumber int) error

	// PostRebaseComment posts a rebase comment on a pull request for Dependabot.
	PostRebaseComment(ctx context.Context, owner, repo string, prNumber int) error

	// MergePullRequest merges a pull request.
	MergePullRequest(ctx context.Context, owner, repo string, prNumber int) error
}

// RealClient implements the Client interface using the real GitHub API.
type RealClient struct {
	client *github.Client
}

// NewRealClient creates a new RealClient with the given token.
func NewRealClient(token string) *RealClient {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return &RealClient{
		client: github.NewClient(tc),
	}
}

// ListRepositories lists all repositories in an organization.
func (c *RealClient) ListRepositories(ctx context.Context, org string) ([]Repository, error) {
	var allRepos []Repository
	opts := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		repos, resp, err := c.client.Repositories.ListByOrg(ctx, org, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list repositories: %w", err)
		}

		for _, repo := range repos {
			allRepos = append(allRepos, Repository{
				Name:          repo.GetName(),
				FullName:      repo.GetFullName(),
				DefaultBranch: repo.GetDefaultBranch(),
				Archived:      repo.GetArchived(),
			})
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allRepos, nil
}

// ListPullRequests lists open pull requests for a repository targeting its default branch.
func (c *RealClient) ListPullRequests(ctx context.Context, owner, repo, defaultBranch string) ([]PullRequest, error) {
	var allPRs []PullRequest
	opts := &github.PullRequestListOptions{
		State:       "open",
		Base:        defaultBranch,
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		prs, resp, err := c.client.PullRequests.List(ctx, owner, repo, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list pull requests: %w", err)
		}

		for _, pr := range prs {
			allPRs = append(allPRs, PullRequest{
				Number:       pr.GetNumber(),
				Title:        pr.GetTitle(),
				URL:          pr.GetHTMLURL(),
				HeadBranch:   pr.GetHead().GetRef(),
				BaseBranch:   pr.GetBase().GetRef(),
				State:        pr.GetState(),
				Draft:        pr.GetDraft(),
				Mergeable:    pr.Mergeable,
				HeadSHA:      pr.GetHead().GetSHA(),
				RepoName:     repo,
				RepoFullName: fmt.Sprintf("%s/%s", owner, repo),
			})
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allPRs, nil
}

// GetPullRequest gets detailed information about a pull request.
func (c *RealClient) GetPullRequest(ctx context.Context, owner, repo string, number int) (*PullRequest, error) {
	pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request: %w", err)
	}

	return &PullRequest{
		Number:       pr.GetNumber(),
		Title:        pr.GetTitle(),
		URL:          pr.GetHTMLURL(),
		HeadBranch:   pr.GetHead().GetRef(),
		BaseBranch:   pr.GetBase().GetRef(),
		State:        pr.GetState(),
		Draft:        pr.GetDraft(),
		Mergeable:    pr.Mergeable,
		HeadSHA:      pr.GetHead().GetSHA(),
		RepoName:     repo,
		RepoFullName: fmt.Sprintf("%s/%s", owner, repo),
	}, nil
}

// GetCheckStatus gets the check status for a commit.
func (c *RealClient) GetCheckStatus(ctx context.Context, owner, repo, ref string) (*CheckStatus, error) {
	// Get check runs
	checkRuns, _, err := c.client.Checks.ListCheckRunsForRef(ctx, owner, repo, ref, &github.ListCheckRunsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get check runs: %w", err)
	}

	// Get commit statuses
	combinedStatus, _, err := c.client.Repositories.GetCombinedStatus(ctx, owner, repo, ref, &github.ListOptions{PerPage: 100})
	if err != nil {
		return nil, fmt.Errorf("failed to get commit status: %w", err)
	}

	// Check if there are no checks at all
	if len(checkRuns.CheckRuns) == 0 && len(combinedStatus.Statuses) == 0 {
		return &CheckStatus{
			AllPassing: false,
			NoChecks:   true,
			Details:    "no checks found",
		}, nil
	}

	// Check all check runs
	for _, check := range checkRuns.CheckRuns {
		status := check.GetStatus()
		conclusion := check.GetConclusion()

		// Check if still in progress
		if status == "queued" || status == "in_progress" {
			return &CheckStatus{
				AllPassing: false,
				Pending:    true,
				Details:    fmt.Sprintf("check '%s' is %s", check.GetName(), status),
			}, nil
		}

		// Only "success" is considered passing
		if conclusion != "success" {
			return &CheckStatus{
				AllPassing: false,
				Details:    fmt.Sprintf("check '%s' has conclusion '%s'", check.GetName(), conclusion),
			}, nil
		}
	}

	// Check all commit statuses
	for _, status := range combinedStatus.Statuses {
		state := status.GetState()
		if state == "pending" {
			return &CheckStatus{
				AllPassing: false,
				Pending:    true,
				Details:    fmt.Sprintf("status '%s' is pending", status.GetContext()),
			}, nil
		}
		if state != "success" {
			return &CheckStatus{
				AllPassing: false,
				Details:    fmt.Sprintf("status '%s' has state '%s'", status.GetContext(), state),
			}, nil
		}
	}

	return &CheckStatus{
		AllPassing: true,
		Details:    "all checks passing",
	}, nil
}

// GetBranchStatus gets the status of a PR branch relative to its base.
func (c *RealClient) GetBranchStatus(ctx context.Context, owner, repo string, prNumber int) (*BranchStatus, error) {
	// Get fresh PR data to check mergeability
	pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request: %w", err)
	}

	// Wait for mergeability to be calculated if needed
	for i := 0; i < 5 && pr.Mergeable == nil; i++ {
		time.Sleep(time.Second)
		pr, _, err = c.client.PullRequests.Get(ctx, owner, repo, prNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to get pull request: %w", err)
		}
	}

	if pr.Mergeable == nil {
		return nil, fmt.Errorf("could not determine mergeability")
	}

	// Check if the PR is mergeable (no conflicts)
	hasConflict := !pr.GetMergeable()

	// Compare branches to determine if PR is up to date
	base := pr.GetBase().GetRef()
	head := pr.GetHead().GetRef()
	headOwner := pr.GetHead().GetRepo().GetOwner().GetLogin()

	comparison, _, err := c.client.Repositories.CompareCommits(ctx, owner, repo, base, fmt.Sprintf("%s:%s", headOwner, head), &github.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to compare commits: %w", err)
	}

	// PR is up to date if base is not ahead
	upToDate := comparison.GetBehindBy() == 0

	return &BranchStatus{
		UpToDate:    upToDate,
		BehindBy:    comparison.GetBehindBy(),
		HasConflict: hasConflict,
	}, nil
}

// UpdateBranch updates a pull request branch with the base branch.
// When GitHub returns HTTP 202 Accepted, it means the branch update has been
// successfully scheduled as a background job. This is treated as success since
// the rebase was triggered, even though it completes asynchronously.
func (c *RealClient) UpdateBranch(ctx context.Context, owner, repo string, prNumber int) error {
	_, _, err := c.client.PullRequests.UpdateBranch(ctx, owner, repo, prNumber, nil)
	if err != nil {
		// Check if this is a rate limit or specific error
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			// HTTP 202 means the update was accepted and scheduled - this is success
			// GitHub will process the update asynchronously via a background job
			if ghErr.Response != nil && ghErr.Response.StatusCode == http.StatusAccepted {
				return nil
			}
			if ghErr.Response != nil && ghErr.Response.StatusCode == http.StatusUnprocessableEntity {
				return fmt.Errorf("branch update not supported or failed: %w", err)
			}
		}
		return fmt.Errorf("failed to update branch: %w", err)
	}
	return nil
}

// PostRebaseComment posts a rebase comment on a pull request for Dependabot.
func (c *RealClient) PostRebaseComment(ctx context.Context, owner, repo string, prNumber int) error {
	comment := &github.IssueComment{
		Body: github.String("@dependabot rebase"),
	}
	_, _, err := c.client.Issues.CreateComment(ctx, owner, repo, prNumber, comment)
	if err != nil {
		return fmt.Errorf("failed to post rebase comment: %w", err)
	}
	return nil
}

// MergePullRequest merges a pull request.
func (c *RealClient) MergePullRequest(ctx context.Context, owner, repo string, prNumber int) error {
	_, _, err := c.client.PullRequests.Merge(ctx, owner, repo, prNumber, "", &github.PullRequestOptions{})
	if err != nil {
		return fmt.Errorf("failed to merge pull request: %w", err)
	}
	return nil
}

// IsDependabotBranch checks if a branch name indicates Dependabot ownership.
func IsDependabotBranch(branchName string) bool {
	return strings.HasPrefix(branchName, "dependabot/")
}

// MatchesBranchPattern checks if a branch name matches the given pattern using substring matching.
func MatchesBranchPattern(branchName, pattern string) bool {
	return strings.Contains(branchName, pattern)
}
