// Package merger implements the core logic for discovering, evaluating, and merging pull requests.
package merger

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/UnitVectorY-Labs/ghprmerge/internal/config"
	gh "github.com/UnitVectorY-Labs/ghprmerge/internal/github"
	"github.com/UnitVectorY-Labs/ghprmerge/internal/output"
)

// Merger handles the discovery, evaluation, and merging of pull requests.
type Merger struct {
	client gh.Client
	config *config.Config
}

// New creates a new Merger with the given client and configuration.
func New(client gh.Client, cfg *config.Config) *Merger {
	return &Merger{
		client: client,
		config: cfg,
	}
}

// Run executes the merger logic and returns the result.
func (m *Merger) Run(ctx context.Context) (*output.RunResult, error) {
	startTime := time.Now()

	result := &output.RunResult{
		Metadata: output.RunMetadata{
			Org:          m.config.Org,
			SourceBranch: m.config.SourceBranch,
			DryRun:       m.config.DryRun,
			Rebase:       m.config.Rebase,
			Limit:        m.config.Limit,
			StartTime:    startTime,
		},
		Repositories: []output.RepositoryResult{},
		Summary:      output.RunSummary{},
	}

	// Discover repositories
	repos, err := m.discoverRepositories(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover repositories: %w", err)
	}

	mergeCount := 0
	limitReached := false

	// Process each repository
	for _, repo := range repos {
		if limitReached {
			break
		}

		repoResult := output.RepositoryResult{
			Name:          repo.Name,
			FullName:      repo.FullName,
			DefaultBranch: repo.DefaultBranch,
			PullRequests:  []output.PullRequestResult{},
		}

		// Discover pull requests
		prs, err := m.discoverPullRequests(ctx, repo)
		if err != nil {
			// Log error but continue with other repos
			continue
		}

		result.Summary.TotalPullRequests += len(prs)

		// Process each pull request
		for _, pr := range prs {
			if m.config.Limit > 0 && mergeCount >= m.config.Limit {
				limitReached = true
				repoResult.PullRequests = append(repoResult.PullRequests, output.PullRequestResult{
					Number:     pr.Number,
					URL:        pr.URL,
					HeadBranch: pr.HeadBranch,
					Title:      pr.Title,
					Action:     output.ActionSkippedLimit,
					Reason:     "merge limit reached",
				})
				result.Summary.Skipped++
				continue
			}

			prResult := m.processPullRequest(ctx, repo, pr)
			repoResult.PullRequests = append(repoResult.PullRequests, prResult)

			// Update summary
			switch prResult.Action {
			case output.ActionMerged:
				result.Summary.Merged++
				mergeCount++
			case output.ActionWouldMerge:
				result.Summary.WouldMerge++
				mergeCount++
			case output.ActionRebased:
				result.Summary.Rebased++
			case output.ActionWouldRebase:
				result.Summary.WouldRebase++
			default:
				if strings.HasPrefix(string(prResult.Action), "skipped") {
					result.Summary.Skipped++
				}
			}
		}

		result.Repositories = append(result.Repositories, repoResult)
		result.Summary.TotalRepositories++
	}

	result.Metadata.EndTime = time.Now()

	return result, nil
}

// discoverRepositories discovers all eligible repositories in the organization.
func (m *Merger) discoverRepositories(ctx context.Context) ([]gh.Repository, error) {
	allRepos, err := m.client.ListRepositories(ctx, m.config.Org)
	if err != nil {
		return nil, err
	}

	// Filter repositories
	var repos []gh.Repository
	for _, repo := range allRepos {
		// Skip archived repositories
		if repo.Archived {
			continue
		}

		// If specific repos are specified, filter by them
		if len(m.config.Repos) > 0 {
			found := false
			for _, r := range m.config.Repos {
				if r == repo.Name {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		repos = append(repos, repo)
	}

	return repos, nil
}

// discoverPullRequests discovers all candidate pull requests for a repository.
func (m *Merger) discoverPullRequests(ctx context.Context, repo gh.Repository) ([]gh.PullRequest, error) {
	owner := strings.Split(repo.FullName, "/")[0]

	allPRs, err := m.client.ListPullRequests(ctx, owner, repo.Name, repo.DefaultBranch)
	if err != nil {
		return nil, err
	}

	// Filter pull requests
	var prs []gh.PullRequest
	for _, pr := range allPRs {
		// Skip drafts
		if pr.Draft {
			continue
		}

		// Match source branch pattern
		if !gh.MatchesBranchPattern(pr.HeadBranch, m.config.SourceBranch) {
			continue
		}

		// Ensure targeting default branch
		if pr.BaseBranch != repo.DefaultBranch {
			continue
		}

		prs = append(prs, pr)
	}

	return prs, nil
}

// processPullRequest processes a single pull request and returns the result.
func (m *Merger) processPullRequest(ctx context.Context, repo gh.Repository, pr gh.PullRequest) output.PullRequestResult {
	owner := strings.Split(repo.FullName, "/")[0]

	result := output.PullRequestResult{
		Number:     pr.Number,
		URL:        pr.URL,
		HeadBranch: pr.HeadBranch,
		Title:      pr.Title,
	}

	// Check if checks are passing
	checkStatus, err := m.client.GetCheckStatus(ctx, owner, repo.Name, pr.HeadSHA)
	if err != nil {
		result.Action = output.ActionSkippedChecks
		result.Reason = fmt.Sprintf("failed to get check status: %v", err)
		return result
	}

	if !checkStatus.AllPassing {
		result.Action = output.ActionSkippedChecks
		result.Reason = checkStatus.Details
		return result
	}

	// Check branch status
	branchStatus, err := m.client.GetBranchStatus(ctx, owner, repo.Name, pr.Number)
	if err != nil {
		result.Action = output.ActionSkippedOutdated
		result.Reason = fmt.Sprintf("failed to get branch status: %v", err)
		return result
	}

	// Check for merge conflicts
	if branchStatus.HasConflict {
		result.Action = output.ActionSkippedConflict
		result.Reason = "pull request has merge conflicts"
		return result
	}

	// Check if branch is up to date
	if !branchStatus.UpToDate {
		if !m.config.Rebase {
			result.Action = output.ActionSkippedOutdated
			result.Reason = fmt.Sprintf("branch is %d commits behind base", branchStatus.BehindBy)
			return result
		}

		// Handle branch update
		if m.config.DryRun {
			result.Action = output.ActionWouldRebase
			result.Reason = fmt.Sprintf("would update branch (%d commits behind)", branchStatus.BehindBy)
			return result
		}

		// Perform actual update
		if gh.IsDependabotBranch(pr.HeadBranch) {
			// Post rebase comment for Dependabot
			if err := m.client.PostRebaseComment(ctx, owner, repo.Name, pr.Number); err != nil {
				result.Action = output.ActionSkippedRebase
				result.Reason = fmt.Sprintf("failed to post rebase comment: %v", err)
				return result
			}
			result.Action = output.ActionRebased
			result.Reason = "posted @dependabot rebase comment"
			return result
		} else {
			// Update branch via API
			if err := m.client.UpdateBranch(ctx, owner, repo.Name, pr.Number); err != nil {
				result.Action = output.ActionSkippedRebase
				result.Reason = fmt.Sprintf("failed to update branch: %v", err)
				return result
			}
			result.Action = output.ActionRebased
			result.Reason = "branch update requested"
			return result
		}
	}

	// All conditions met, ready to merge
	if m.config.DryRun {
		result.Action = output.ActionWouldMerge
		result.Reason = "all checks passing, branch up to date"
		return result
	}

	// Perform merge
	if err := m.client.MergePullRequest(ctx, owner, repo.Name, pr.Number); err != nil {
		result.Action = output.ActionSkippedMerge
		result.Reason = fmt.Sprintf("merge failed: %v", err)
		return result
	}

	result.Action = output.ActionMerged
	result.Reason = "successfully merged"
	return result
}
