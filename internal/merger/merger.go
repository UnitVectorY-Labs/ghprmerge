// Package merger implements the core logic for discovering, evaluating, and merging pull requests.
package merger

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/UnitVectorY-Labs/ghprmerge/internal/config"
	gh "github.com/UnitVectorY-Labs/ghprmerge/internal/github"
	"github.com/UnitVectorY-Labs/ghprmerge/internal/output"
)

// Merger handles the discovery, evaluation, and merging of pull requests.
type Merger struct {
	client    gh.Client
	config    *config.Config
	logOutput io.Writer
}

// New creates a new Merger with the given client and configuration.
func New(client gh.Client, cfg *config.Config, logOutput io.Writer) *Merger {
	return &Merger{
		client:    client,
		config:    cfg,
		logOutput: logOutput,
	}
}

// log writes a progress message to the log output.
func (m *Merger) log(format string, args ...interface{}) {
	if m.logOutput != nil && !m.config.JSON {
		fmt.Fprintf(m.logOutput, format+"\n", args...)
	}
}

// Run executes the merger logic and returns the result.
// Processing is strictly sequential: one repository at a time, one PR at a time.
func (m *Merger) Run(ctx context.Context) (*output.RunResult, error) {
	startTime := time.Now()

	// Determine mode description
	mode := m.getModeDescription()

	// Build repo limit description
	repoLimitDesc := ""
	if m.config.RepoLimit > 0 {
		repoLimitDesc = fmt.Sprintf("%d repositories max", m.config.RepoLimit)
	}

	result := &output.RunResult{
		Metadata: output.RunMetadata{
			Org:           m.config.Org,
			SourceBranch:  m.config.SourceBranch,
			Mode:          mode,
			Rebase:        m.config.Rebase,
			Merge:         m.config.Merge,
			RepoLimit:     m.config.RepoLimit,
			RepoLimitDesc: repoLimitDesc,
			StartTime:     startTime,
		},
		Repositories: []output.RepositoryResult{},
		Summary: output.RunSummary{
			SkippedByReason: make(map[string]int),
		},
	}

	// Log startup info (minimal)
	m.log("ghprmerge - %s", m.config.Org)
	m.log("Mode: %s | Branch pattern: %s", mode, m.config.SourceBranch)
	if m.config.RepoLimit > 0 {
		m.log("Limit: %d repositories max", m.config.RepoLimit)
	}
	m.log("")

	// Discover repositories
	repos, err := m.discoverRepositories(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover repositories: %w", err)
	}
	m.log("Found %d repositories", len(repos))

	repoCount := 0

	// Process each repository sequentially
	for i, repo := range repos {
		// Check repo limit
		if m.config.RepoLimit > 0 && repoCount >= m.config.RepoLimit {
			result.Repositories = append(result.Repositories, output.RepositoryResult{
				Name:          repo.Name,
				FullName:      repo.FullName,
				DefaultBranch: repo.DefaultBranch,
				Skipped:       true,
				SkipReason:    "repo limit reached",
			})
			result.Summary.ReposSkipped++
			continue
		}

		// In quiet mode, don't log repos with no matching PRs
		// We need to process first and then decide whether to log
		// In confirm mode, only scan without actions
		var repoResult output.RepositoryResult
		if m.config.Confirm {
			repoResult = m.processRepositoryScanOnly(ctx, repo)
		} else {
			repoResult = m.processRepository(ctx, repo)
		}
		result.Repositories = append(result.Repositories, repoResult)

		// In quiet mode, skip logging repos with no matching PRs and no skip reason
		shouldLogRepo := !m.config.Quiet || repoResult.Skipped || len(repoResult.PullRequests) > 0
		if shouldLogRepo {
			m.log("[%d/%d] %s", i+1, len(repos), repo.FullName)
		}

		if repoResult.Skipped {
			result.Summary.ReposSkipped++
			m.log("      ⊘ Repository skipped: %s", repoResult.SkipReason)
		} else {
			result.Summary.ReposProcessed++
			repoCount++

			if len(repoResult.PullRequests) == 0 {
				if !m.config.Quiet {
					m.log("      No matching pull requests")
				}
			} else {
				// Log each PR with its complete status on one line
				for _, pr := range repoResult.PullRequests {
					symbol := m.getActionSymbol(pr.Action)
					m.log("      %s #%-4d %-50s", symbol, pr.Number, truncateString(pr.Title, 50))
					m.log("               Branch: %s", pr.HeadBranch)
					m.log("               Status: %s", pr.Action)
					if pr.Reason != "" {
						m.log("               Detail: %s", pr.Reason)
					}
				}
			}
		}

		// Update summary with PR results
		for _, pr := range repoResult.PullRequests {
			result.Summary.CandidatesFound++
			m.updateSummary(&result.Summary, pr)
		}
	}

	result.Metadata.EndTime = time.Now()
	m.log("")

	return result, nil
}

// getActionSymbol returns a symbol for the action type.
func (m *Merger) getActionSymbol(action output.Action) string {
	switch action {
	case output.ActionMerged, output.ActionWouldMerge, output.ActionReadyMerge:
		return "✓"
	case output.ActionRebased, output.ActionWouldRebase:
		return "↻"
	case output.ActionMergeFailed, output.ActionRebaseFailed:
		return "✗"
	default:
		if strings.HasPrefix(string(action), "skip:") {
			return "⊘"
		}
		return "•"
	}
}

// truncateString truncates a string to maxLen and adds "..." if needed.
func truncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// RunWithActions executes actions on a previously scanned result (used with --confirm).
func (m *Merger) RunWithActions(ctx context.Context, scanResult *output.RunResult) (*output.RunResult, error) {
	m.log("Executing actions...")

	// Reset summary counters that will be updated
	scanResult.Summary.MergedSuccess = 0
	scanResult.Summary.MergeFailed = 0
	scanResult.Summary.RebasedSuccess = 0
	scanResult.Summary.RebaseFailed = 0
	scanResult.Summary.WouldMerge = 0
	scanResult.Summary.WouldRebase = 0
	scanResult.Summary.Skipped = 0
	scanResult.Summary.SkippedByReason = make(map[string]int)

	// Process each repository and execute pending actions
	for i := range scanResult.Repositories {
		repo := &scanResult.Repositories[i]
		if repo.Skipped {
			continue
		}

		owner := strings.Split(repo.FullName, "/")[0]
		m.log("Processing actions for: %s", repo.FullName)

		for j := range repo.PullRequests {
			pr := &repo.PullRequests[j]

			// Execute actions based on what was planned
			switch pr.Action {
			case output.ActionWouldRebase:
				m.executeRebase(ctx, owner, repo.Name, pr)
			case output.ActionWouldMerge:
				m.executeMerge(ctx, owner, repo.Name, pr)
			}

			// Update summary
			m.updateSummary(&scanResult.Summary, *pr)
		}
	}

	scanResult.Metadata.EndTime = time.Now()
	return scanResult, nil
}

// executeRebase executes a rebase action on a PR.
func (m *Merger) executeRebase(ctx context.Context, owner, repoName string, pr *output.PullRequestResult) {
	if gh.IsDependabotBranch(pr.HeadBranch) {
		if err := m.client.PostRebaseComment(ctx, owner, repoName, pr.Number); err != nil {
			pr.Action = output.ActionRebaseFailed
			pr.Reason = fmt.Sprintf("failed to post rebase comment: %v", err)
			m.log("  ✗ PR #%d: rebase failed - %v", pr.Number, err)
		} else {
			pr.Action = output.ActionRebased
			pr.Reason = "posted @dependabot rebase comment"
			m.log("  ↻ PR #%d: rebased (posted @dependabot rebase comment)", pr.Number)
		}
	} else {
		if err := m.client.UpdateBranch(ctx, owner, repoName, pr.Number); err != nil {
			pr.Action = output.ActionRebaseFailed
			pr.Reason = fmt.Sprintf("failed to update branch: %v", err)
			m.log("  ✗ PR #%d: rebase failed - %v", pr.Number, err)
		} else {
			pr.Action = output.ActionRebased
			pr.Reason = "branch update requested via API"
			m.log("  ↻ PR #%d: rebased (branch updated via API)", pr.Number)
		}
	}
}

// executeMerge executes a merge action on a PR.
func (m *Merger) executeMerge(ctx context.Context, owner, repoName string, pr *output.PullRequestResult) {
	if err := m.client.MergePullRequest(ctx, owner, repoName, pr.Number); err != nil {
		pr.Action = output.ActionMergeFailed
		pr.Reason = fmt.Sprintf("merge failed: %v", err)
		m.log("  ✗ PR #%d: merge failed - %v", pr.Number, err)
	} else {
		pr.Action = output.ActionMerged
		pr.Reason = "successfully merged"
		m.log("  ✓ PR #%d: merged", pr.Number)
	}
}

// processRepositoryScanOnly processes a repository without taking actions (for --confirm mode).
func (m *Merger) processRepositoryScanOnly(ctx context.Context, repo gh.Repository) output.RepositoryResult {
	owner := strings.Split(repo.FullName, "/")[0]

	repoResult := output.RepositoryResult{
		Name:          repo.Name,
		FullName:      repo.FullName,
		DefaultBranch: repo.DefaultBranch,
		PullRequests:  []output.PullRequestResult{},
	}

	// Discover pull requests
	prs, err := m.discoverPullRequests(ctx, repo)
	if err != nil {
		repoResult.Skipped = true
		repoResult.SkipReason = fmt.Sprintf("API error: %v", err)
		return repoResult
	}

	// Process each pull request sequentially (scan only)
	for _, pr := range prs {
		prResult := m.evaluatePullRequest(ctx, owner, repo, pr)
		repoResult.PullRequests = append(repoResult.PullRequests, prResult)
	}

	return repoResult
}

// evaluatePullRequest evaluates a PR and returns what action would be taken (no side effects).
func (m *Merger) evaluatePullRequest(ctx context.Context, owner string, repo gh.Repository, pr gh.PullRequest) output.PullRequestResult {
	result := output.PullRequestResult{
		Number:     pr.Number,
		URL:        pr.URL,
		HeadBranch: pr.HeadBranch,
		Title:      pr.Title,
	}

	// Get check status
	checkStatus, err := m.client.GetCheckStatus(ctx, owner, repo.Name, pr.HeadSHA)
	if err != nil {
		result.Action = output.ActionSkipAPIError
		result.Reason = fmt.Sprintf("failed to get check status: %v", err)
		result.SkipReason = output.ReasonAPIError
		return result
	}

	// Handle check status
	if checkStatus.NoChecks {
		result.Action = output.ActionSkipNoChecks
		result.Reason = "no checks found for this pull request"
		result.SkipReason = output.ReasonNoChecks
		return result
	}

	if checkStatus.Pending {
		result.Action = output.ActionSkipChecksPending
		result.Reason = checkStatus.Details
		result.SkipReason = output.ReasonChecksPending
		return result
	}

	if !checkStatus.AllPassing {
		result.Action = output.ActionSkipChecksFailing
		result.Reason = checkStatus.Details
		result.SkipReason = output.ReasonChecksFailing
		return result
	}

	// Check branch status
	branchStatus, err := m.client.GetBranchStatus(ctx, owner, repo.Name, pr.Number)
	if err != nil {
		result.Action = output.ActionSkipAPIError
		result.Reason = fmt.Sprintf("failed to get branch status: %v", err)
		result.SkipReason = output.ReasonAPIError
		return result
	}

	// Check for merge conflicts
	if branchStatus.HasConflict {
		result.Action = output.ActionSkipConflict
		result.Reason = "pull request has merge conflicts"
		result.SkipReason = output.ReasonConflict
		return result
	}

	// Check if branch is up to date
	if !branchStatus.UpToDate {
		// If rebase is not enabled, skip
		if !m.config.Rebase {
			result.Action = output.ActionSkipBranchBehind
			result.Reason = fmt.Sprintf("branch is %d commits behind base (use --rebase to update)", branchStatus.BehindBy)
			result.SkipReason = output.ReasonBranchBehind
			return result
		}

		// Report what would be done
		if gh.IsDependabotBranch(pr.HeadBranch) {
			result.Action = output.ActionWouldRebase
			result.Reason = fmt.Sprintf("would post @dependabot rebase comment (%d commits behind)", branchStatus.BehindBy)
		} else {
			result.Action = output.ActionWouldRebase
			result.Reason = fmt.Sprintf("would update branch via API (%d commits behind)", branchStatus.BehindBy)
		}
		return result
	}

	// All conditions met, ready to merge
	if m.config.Merge {
		result.Action = output.ActionWouldMerge
		result.Reason = "all checks passing, branch up to date"
	} else {
		result.Action = output.ActionReadyMerge
		result.Reason = "all checks passing, branch up to date (use --merge to merge)"
	}

	return result
}

// getModeDescription returns a human-readable description of the current mode.
func (m *Merger) getModeDescription() string {
	if m.config.Rebase {
		return "rebase mode"
	}
	if m.config.Merge {
		return "merge mode"
	}
	return "analysis only (no mutations)"
}

// updateSummary updates the run summary based on a PR result.
func (m *Merger) updateSummary(summary *output.RunSummary, pr output.PullRequestResult) {
	switch pr.Action {
	case output.ActionMerged:
		summary.MergedSuccess++
	case output.ActionMergeFailed:
		summary.MergeFailed++
	case output.ActionRebased:
		summary.RebasedSuccess++
	case output.ActionRebaseFailed:
		summary.RebaseFailed++
	case output.ActionWouldMerge:
		summary.WouldMerge++
	case output.ActionWouldRebase:
		summary.WouldRebase++
	case output.ActionReadyMerge:
		summary.ReadyToMerge++
	default:
		if strings.HasPrefix(string(pr.Action), "skip:") {
			summary.Skipped++
			if pr.SkipReason != "" {
				summary.SkippedByReason[string(pr.SkipReason)]++
			}
		}
	}
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

// processRepository processes a single repository and returns the result.
func (m *Merger) processRepository(ctx context.Context, repo gh.Repository) output.RepositoryResult {
	owner := strings.Split(repo.FullName, "/")[0]

	repoResult := output.RepositoryResult{
		Name:          repo.Name,
		FullName:      repo.FullName,
		DefaultBranch: repo.DefaultBranch,
		PullRequests:  []output.PullRequestResult{},
	}

	// Discover pull requests
	prs, err := m.discoverPullRequests(ctx, repo)
	if err != nil {
		repoResult.Skipped = true
		repoResult.SkipReason = fmt.Sprintf("API error: %v", err)
		return repoResult
	}

	// Process each pull request sequentially
	for _, pr := range prs {
		prResult := m.processPullRequest(ctx, owner, repo, pr)
		repoResult.PullRequests = append(repoResult.PullRequests, prResult)
	}

	return repoResult
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
func (m *Merger) processPullRequest(ctx context.Context, owner string, repo gh.Repository, pr gh.PullRequest) output.PullRequestResult {
	result := output.PullRequestResult{
		Number:     pr.Number,
		URL:        pr.URL,
		HeadBranch: pr.HeadBranch,
		Title:      pr.Title,
	}

	// Get check status
	checkStatus, err := m.client.GetCheckStatus(ctx, owner, repo.Name, pr.HeadSHA)
	if err != nil {
		result.Action = output.ActionSkipAPIError
		result.Reason = fmt.Sprintf("failed to get check status: %v", err)
		result.SkipReason = output.ReasonAPIError
		return result
	}

	// Handle check status
	if checkStatus.NoChecks {
		result.Action = output.ActionSkipNoChecks
		result.Reason = "no checks found for this pull request"
		result.SkipReason = output.ReasonNoChecks
		return result
	}

	if checkStatus.Pending {
		result.Action = output.ActionSkipChecksPending
		result.Reason = checkStatus.Details
		result.SkipReason = output.ReasonChecksPending
		return result
	}

	if !checkStatus.AllPassing {
		result.Action = output.ActionSkipChecksFailing
		result.Reason = checkStatus.Details
		result.SkipReason = output.ReasonChecksFailing
		return result
	}

	// Check branch status
	branchStatus, err := m.client.GetBranchStatus(ctx, owner, repo.Name, pr.Number)
	if err != nil {
		result.Action = output.ActionSkipAPIError
		result.Reason = fmt.Sprintf("failed to get branch status: %v", err)
		result.SkipReason = output.ReasonAPIError
		return result
	}

	// Check for merge conflicts
	if branchStatus.HasConflict {
		result.Action = output.ActionSkipConflict
		result.Reason = "pull request has merge conflicts"
		result.SkipReason = output.ReasonConflict
		return result
	}

	// Check if branch is up to date
	if !branchStatus.UpToDate {
		return m.handleOutdatedBranch(ctx, owner, repo, pr, branchStatus)
	}

	// All conditions met, ready to merge
	return m.handleMergeReady(ctx, owner, repo, pr)
}

// handleOutdatedBranch handles PRs where the branch is behind the default branch.
func (m *Merger) handleOutdatedBranch(ctx context.Context, owner string, repo gh.Repository, pr gh.PullRequest, branchStatus *gh.BranchStatus) output.PullRequestResult {
	result := output.PullRequestResult{
		Number:     pr.Number,
		URL:        pr.URL,
		HeadBranch: pr.HeadBranch,
		Title:      pr.Title,
	}

	// If rebase is not enabled, skip
	if !m.config.Rebase {
		result.Action = output.ActionSkipBranchBehind
		result.Reason = fmt.Sprintf("branch is %d commits behind base (use --rebase to update)", branchStatus.BehindBy)
		result.SkipReason = output.ReasonBranchBehind
		return result
	}

	// Perform actual rebase/update
	if gh.IsDependabotBranch(pr.HeadBranch) {
		if err := m.client.PostRebaseComment(ctx, owner, repo.Name, pr.Number); err != nil {
			result.Action = output.ActionRebaseFailed
			result.Reason = fmt.Sprintf("failed to post rebase comment: %v", err)
			return result
		}
		result.Action = output.ActionRebased
		result.Reason = fmt.Sprintf("posted @dependabot rebase comment (%d commits behind)", branchStatus.BehindBy)
	} else {
		if err := m.client.UpdateBranch(ctx, owner, repo.Name, pr.Number); err != nil {
			result.Action = output.ActionRebaseFailed
			result.Reason = fmt.Sprintf("failed to update branch: %v", err)
			return result
		}
		result.Action = output.ActionRebased
		result.Reason = fmt.Sprintf("branch update requested via API (%d commits behind)", branchStatus.BehindBy)
	}

	return result
}

// handleMergeReady handles PRs that are ready to merge.
func (m *Merger) handleMergeReady(ctx context.Context, owner string, repo gh.Repository, pr gh.PullRequest) output.PullRequestResult {
	result := output.PullRequestResult{
		Number:     pr.Number,
		URL:        pr.URL,
		HeadBranch: pr.HeadBranch,
		Title:      pr.Title,
	}

	// If merge is not enabled, just report
	if !m.config.Merge {
		result.Action = output.ActionReadyMerge
		result.Reason = "all checks passing, branch up to date (use --merge to merge)"
		return result
	}

	// Perform merge
	if err := m.client.MergePullRequest(ctx, owner, repo.Name, pr.Number); err != nil {
		result.Action = output.ActionMergeFailed
		result.Reason = fmt.Sprintf("merge failed: %v", err)
		return result
	}

	result.Action = output.ActionMerged
	result.Reason = "successfully merged"
	return result
}
