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
	client           gh.Client
	config           *config.Config
	console          *output.Console
	scanDisplayLines int
}

// New creates a new Merger with the given client and configuration.
func New(client gh.Client, cfg *config.Config, console *output.Console) *Merger {
	return &Merger{
		client:  client,
		config:  cfg,
		console: console,
	}
}

// Run executes the merger logic and returns the result.
// Processing is strictly sequential: one repository at a time, one PR at a time.
func (m *Merger) Run(ctx context.Context) (*output.RunResult, error) {
	m.scanDisplayLines = 0
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

	// Discover repositories
	repos, err := m.discoverRepositories(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover repositories: %w", err)
	}

	// Print header and start progress
	if m.console != nil && !m.config.JSON {
		m.console.PrintHeader(m.config.Org, mode, m.config.SourceBranch)
		if m.config.RepoLimit > 0 {
			fmt.Fprintf(m.console.Writer(), "%s\n", m.console.Dim(fmt.Sprintf("Limit: %d repositories max", m.config.RepoLimit)))
		}
	}

	repoCount := 0
	showProgress := m.console != nil && !m.config.JSON && len(repos) > 0

	// Process each repository sequentially
	for i, repo := range repos {
		// Update progress bar
		if showProgress {
			m.console.ProgressBar(i+1, len(repos), "Scanning")
		}

		// Check repo limit
		var repoResult output.RepositoryResult
		if m.config.RepoLimit > 0 && repoCount >= m.config.RepoLimit {
			repoResult = output.RepositoryResult{
				Name:          repo.Name,
				FullName:      repo.FullName,
				DefaultBranch: repo.DefaultBranch,
				Skipped:       true,
				SkipReason:    "repo limit reached",
			}
			result.Repositories = append(result.Repositories, repoResult)
			result.Summary.ReposSkipped++
		} else {
			if m.config.Confirm {
				repoResult = m.processRepositoryScanOnly(ctx, repo)
			} else {
				repoResult = m.processRepository(ctx, repo)
			}
			result.Repositories = append(result.Repositories, repoResult)

			if repoResult.Skipped {
				result.Summary.ReposSkipped++
			} else {
				result.Summary.ReposProcessed++
				repoCount++
			}

			// Update summary with PR results
			for _, pr := range repoResult.PullRequests {
				result.Summary.CandidatesFound++
				m.updateSummary(&result.Summary, pr)
			}
		}

		if showProgress && m.shouldStreamScanResults() {
			m.scanDisplayLines += m.printRepoResultWithProgress(repoResult, i+1, len(repos), "Scanning")
		}
	}

	// Finish progress bar
	if showProgress {
		m.console.FinishProgress()
		if m.shouldStreamScanResults() {
			m.scanDisplayLines++
		}
	}

	// In the default human view, print matching repositories after the scan completes.
	// Confirm mode still needs this fallback when there is nothing to confirm so the
	// user can see which repos matched and why they were skipped.
	if m.console != nil && !m.config.JSON && !m.config.Verbose && (!m.config.Confirm || !hasPendingActions(result)) {
		fmt.Fprintln(m.console.Writer())
		for _, repo := range result.Repositories {
			if len(repo.PullRequests) > 0 {
				m.console.PrintRepoResult(repo)
			}
		}
	}

	result.Metadata.EndTime = time.Now()

	return result, nil
}

// RunWithActions executes actions on a previously scanned result (used with --confirm).
func (m *Merger) RunWithActions(ctx context.Context, scanResult *output.RunResult) (*output.RunResult, error) {
	// Reset summary counters that will be updated
	scanResult.Summary.MergedSuccess = 0
	scanResult.Summary.MergeFailed = 0
	scanResult.Summary.RebasedSuccess = 0
	scanResult.Summary.RebaseFailed = 0
	scanResult.Summary.WouldMerge = 0
	scanResult.Summary.WouldRebase = 0
	scanResult.Summary.Skipped = 0
	scanResult.Summary.SkippedByReason = make(map[string]int)

	// Count total actions for progress bar
	totalActions := 0
	for _, repo := range scanResult.Repositories {
		for _, pr := range repo.PullRequests {
			if pr.Action == output.ActionWouldRebase || pr.Action == output.ActionWouldMerge {
				totalActions++
			}
		}
	}

	actionNum := 0
	showProgress := m.console != nil && !m.config.JSON && totalActions > 0

	// Process each repository and execute pending actions
	for i := range scanResult.Repositories {
		repo := &scanResult.Repositories[i]
		if repo.Skipped {
			continue
		}

		owner := strings.Split(repo.FullName, "/")[0]

		for j := range repo.PullRequests {
			pr := &repo.PullRequests[j]

			// Execute actions based on what was planned
			switch pr.Action {
			case output.ActionWouldRebase:
				actionNum++
				if showProgress {
					m.console.ProgressBar(actionNum, totalActions, "Executing")
				}
				m.executeRebase(ctx, owner, repo.Name, pr)
			case output.ActionWouldMerge:
				actionNum++
				if showProgress {
					m.console.ProgressBar(actionNum, totalActions, "Executing")
				}
				m.executeMerge(ctx, owner, repo.Name, pr)
			}

			// Update summary
			m.updateSummary(&scanResult.Summary, *pr)
		}

		if showProgress && m.config.Verbose && hasCompletedActions(*repo) {
			m.printRepoResultWithProgress(*repo, actionNum, totalActions, "Executing")
		}
	}

	// Finish progress bar
	if showProgress {
		m.console.FinishProgress()
	}

	// Print action results
	if m.console != nil && !m.config.JSON {
		for _, repo := range scanResult.Repositories {
			if m.config.Verbose {
				continue
			}
			if hasCompletedActions(repo) {
				m.console.PrintRepoResult(repo)
			}
		}
	}

	scanResult.Metadata.EndTime = time.Now()
	return scanResult, nil
}

// ScanDisplayLines returns the number of scan-time terminal lines written for live verbose output.
func (m *Merger) ScanDisplayLines() int {
	return m.scanDisplayLines
}

// executeRebase executes a rebase action on a PR.
func (m *Merger) executeRebase(ctx context.Context, owner, repoName string, pr *output.PullRequestResult) {
	if gh.IsDependabotBranch(pr.HeadBranch) {
		if err := m.client.PostRebaseComment(ctx, owner, repoName, pr.Number); err != nil {
			pr.Action = output.ActionRebaseFailed
			pr.Reason = fmt.Sprintf("failed to post rebase comment: %v", err)
		} else {
			pr.Action = output.ActionRebased
			pr.Reason = "posted @dependabot rebase comment"
		}
	} else {
		if err := m.client.UpdateBranch(ctx, owner, repoName, pr.Number); err != nil {
			pr.Action = output.ActionRebaseFailed
			pr.Reason = fmt.Sprintf("failed to update branch: %v", err)
		} else {
			pr.Action = output.ActionRebased
			pr.Reason = "branch update requested via API"
		}
	}
}

// executeMerge executes a merge action on a PR.
func (m *Merger) executeMerge(ctx context.Context, owner, repoName string, pr *output.PullRequestResult) {
	if err := m.client.MergePullRequest(ctx, owner, repoName, pr.Number); err != nil {
		pr.Action = output.ActionMergeFailed
		pr.Reason = fmt.Sprintf("merge failed: %v", err)
	} else {
		pr.Action = output.ActionMerged
		pr.Reason = "successfully merged"
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
	// When in rebase-only mode, we don't require passing checks since rebasing
	// may resolve issues by incorporating upstream changes
	rebaseOnly := m.config.Rebase && !m.config.Merge
	checksState := "all checks passing"

	if checkStatus.NoChecks {
		checksState = "no checks configured"
	} else {
		if checkStatus.Pending && !rebaseOnly {
			result.Action = output.ActionSkipChecksPending
			result.Reason = checkStatus.Details
			result.SkipReason = output.ReasonChecksPending
			return result
		}

		if !checkStatus.AllPassing && !rebaseOnly {
			result.Action = output.ActionSkipChecksFailing
			result.Reason = checkStatus.Details
			result.SkipReason = output.ReasonChecksFailing
			return result
		}
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
		// If skip-rebase is enabled with merge, would merge despite being behind
		if m.config.SkipRebase && m.config.Merge {
			result.Action = output.ActionWouldMerge
			result.Reason = fmt.Sprintf("%s, would merge (branch is %d commits behind, rebase skipped)", checksState, branchStatus.BehindBy)
			return result
		}

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
		result.Reason = checksState + ", branch up to date"
	} else {
		result.Action = output.ActionReadyMerge
		result.Reason = checksState + ", branch up to date (use --merge to merge)"
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

func (m *Merger) shouldStreamScanResults() bool {
	return m.config.Verbose
}

func (m *Merger) printRepoResultWithProgress(repo output.RepositoryResult, current, total int, label string) int {
	if m.console == nil || m.config.JSON {
		return 0
	}

	m.console.ClearCurrentLine()
	lines := m.console.PrintRepoResult(repo)
	if total > 0 {
		m.console.ProgressBar(current, total, label)
	}
	return lines
}

func hasCompletedActions(repo output.RepositoryResult) bool {
	for _, pr := range repo.PullRequests {
		switch pr.Action {
		case output.ActionMerged, output.ActionMergeFailed, output.ActionRebased, output.ActionRebaseFailed:
			return true
		}
	}
	return false
}

func hasPendingActions(result *output.RunResult) bool {
	return result.Summary.WouldMerge > 0 || result.Summary.WouldRebase > 0
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
	// When in rebase-only mode, we don't require passing checks since rebasing
	// may resolve issues by incorporating upstream changes
	rebaseOnly := m.config.Rebase && !m.config.Merge
	checksState := "all checks passing"

	if checkStatus.NoChecks {
		checksState = "no checks configured"
	} else {
		if checkStatus.Pending && !rebaseOnly {
			result.Action = output.ActionSkipChecksPending
			result.Reason = checkStatus.Details
			result.SkipReason = output.ReasonChecksPending
			return result
		}

		if !checkStatus.AllPassing && !rebaseOnly {
			result.Action = output.ActionSkipChecksFailing
			result.Reason = checkStatus.Details
			result.SkipReason = output.ReasonChecksFailing
			return result
		}
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
		return m.handleOutdatedBranch(ctx, owner, repo, pr, branchStatus, checksState)
	}

	// All conditions met, ready to merge
	return m.handleMergeReady(ctx, owner, repo, pr, checksState)
}

// handleOutdatedBranch handles PRs where the branch is behind the default branch.
func (m *Merger) handleOutdatedBranch(ctx context.Context, owner string, repo gh.Repository, pr gh.PullRequest, branchStatus *gh.BranchStatus, checksState string) output.PullRequestResult {
	result := output.PullRequestResult{
		Number:     pr.Number,
		URL:        pr.URL,
		HeadBranch: pr.HeadBranch,
		Title:      pr.Title,
	}

	// If skip-rebase is enabled with merge, proceed to merge despite being behind
	if m.config.SkipRebase && m.config.Merge {
		// Perform merge (branch is behind but we're skipping the rebase requirement)
		if err := m.client.MergePullRequest(ctx, owner, repo.Name, pr.Number); err != nil {
			result.Action = output.ActionMergeFailed
			result.Reason = fmt.Sprintf("merge failed: %v", err)
			return result
		}
		result.Action = output.ActionMerged
		result.Reason = fmt.Sprintf("successfully merged (%s; branch was %d commits behind, rebase skipped)", checksState, branchStatus.BehindBy)
		return result
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
func (m *Merger) handleMergeReady(ctx context.Context, owner string, repo gh.Repository, pr gh.PullRequest, checksState string) output.PullRequestResult {
	result := output.PullRequestResult{
		Number:     pr.Number,
		URL:        pr.URL,
		HeadBranch: pr.HeadBranch,
		Title:      pr.Title,
	}

	// If merge is not enabled, just report
	if !m.config.Merge {
		result.Action = output.ActionReadyMerge
		result.Reason = checksState + ", branch up to date (use --merge to merge)"
		return result
	}

	// Perform merge
	if err := m.client.MergePullRequest(ctx, owner, repo.Name, pr.Number); err != nil {
		result.Action = output.ActionMergeFailed
		result.Reason = fmt.Sprintf("merge failed: %v", err)
		return result
	}

	result.Action = output.ActionMerged
	result.Reason = "successfully merged (" + checksState + ")"
	return result
}
