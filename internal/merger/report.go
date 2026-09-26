package merger

import (
	"context"
	"fmt"
	"sort"
	"strings"

	gh "github.com/UnitVectorY-Labs/ghprmerge/internal/github"
	"github.com/UnitVectorY-Labs/ghprmerge/internal/output"
)

// RunReport executes the report mode: discovers open PRs across repositories,
// groups them by exact source branch name, filters and sorts the results.
func (m *Merger) RunReport(ctx context.Context) (*output.ReportResult, error) {
	// Discover repositories (reuses existing repo scan logic)
	repos, err := m.discoverRepositories(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover repositories: %w", err)
	}

	// Print header and start progress
	if m.console != nil && !m.config.JSON {
		m.console.PrintReportHeader(m.config.Org, len(repos))
		if m.config.RepoLimit > 0 {
			fmt.Fprintf(m.console.Writer(), "%s\n", m.console.Dim(fmt.Sprintf("Limit: %d repositories max", m.config.RepoLimit)))
		}
	}

	showProgress := m.console != nil && !m.config.JSON && !m.config.NoProgress && len(repos) > 0

	// Collect all open PRs from all repositories
	type prEntry struct {
		repoName   string
		pr         gh.PullRequest
		checkState string
	}

	var allPRs []prEntry

	type scan struct {
		entries []prEntry
		ok      bool
	}
	scans := make([]scan, len(repos))
	completed := 0
	err = parallel(ctx, len(repos), m.config.Workers, m.config.RepoLimit, func(i int) scan {
		repo := repos[i]
		owner := strings.Split(repo.FullName, "/")[0]
		prs, err := m.client.ListPullRequests(ctx, owner, repo.Name, repo.DefaultBranch)
		if err != nil {
			return scan{}
		}
		var entries []prEntry
		for _, pr := range prs {
			// Skip draft PRs
			if pr.Draft {
				continue
			}

			// Ensure targeting default branch
			if pr.BaseBranch != repo.DefaultBranch {
				continue
			}

			// Apply source branch prefix filter
			if len(m.config.SourceBranchPrefix) > 0 {
				matched := false
				for _, prefix := range m.config.SourceBranchPrefix {
					if strings.HasPrefix(pr.HeadBranch, prefix) {
						matched = true
						break
					}
				}
				if !matched {
					continue
				}
			}

			// Apply author filter if specified
			if m.config.Author != "" && pr.Author != m.config.Author {
				continue
			}

			entries = append(entries, prEntry{
				repoName: repo.Name,
				pr:       pr,
			})
		}
		return scan{entries, true}
	}, func(i int, value scan, limited bool) bool {
		scans[i] = value
		completed++
		if showProgress {
			m.console.ProgressBar(completed, len(repos), "Scanning")
		}
		return value.ok
	})
	if err != nil {
		return nil, err
	}
	for _, scan := range scans {
		allPRs = append(allPRs, scan.entries...)
	}

	// Finish progress bar
	if showProgress {
		m.console.FinishProgress()
	}

	// Group PRs by exact source branch name
	type groupEntry struct {
		branch string
		prs    []prEntry
	}
	groupMap := make(map[string]*groupEntry)
	for _, entry := range allPRs {
		branch := entry.pr.HeadBranch
		if g, ok := groupMap[branch]; ok {
			g.prs = append(g.prs, entry)
		} else {
			groupMap[branch] = &groupEntry{
				branch: branch,
				prs:    []prEntry{entry},
			}
		}
	}

	// Filter by min-group-size
	var groups []groupEntry
	for _, g := range groupMap {
		if len(g.prs) >= m.config.MinGroupSize {
			groups = append(groups, *g)
		}
	}

	// Sort: descending count, then ascending branch name for ties
	sort.Slice(groups, func(i, j int) bool {
		if len(groups[i].prs) != len(groups[j].prs) {
			return len(groups[i].prs) > len(groups[j].prs)
		}
		return groups[i].branch < groups[j].branch
	})

	// Determine verbosity
	verbosity := m.config.Verbosity
	if verbosity == "" {
		verbosity = "standard"
	}

	// Evaluate status for each PR in report mode
	// Only evaluate if we need status (standard or verbose modes, or JSON)
	needsStatus := verbosity != "brief" || m.config.JSON

	// Build report result
	result := &output.ReportResult{
		Groups: make([]output.ReportGroup, 0, len(groups)),
	}

	// Group evaluation jobs by repository to keep its PRs sequential, even
	// when they belong to different source-branch groups.
	type evaluation struct {
		group, pr int
		entry     prEntry
	}
	var jobs [][]evaluation
	repoJobs := make(map[string]int)
	totalPRs := 0
	for gi, g := range groups {
		result.Groups = append(result.Groups, output.ReportGroup{SourceBranch: g.branch, Count: len(g.prs), PullRequests: make([]output.ReportPullRequest, len(g.prs))})
		for pi, entry := range g.prs {
			index, exists := repoJobs[entry.repoName]
			if !exists {
				index = len(jobs)
				repoJobs[entry.repoName] = index
				jobs = append(jobs, nil)
			}
			jobs[index] = append(jobs[index], evaluation{gi, pi, entry})
			totalPRs++
		}
	}
	evaluated := 0
	err = parallel(ctx, len(jobs), m.config.Workers, 0, func(i int) int {
		for _, job := range jobs[i] {
			result.Groups[job.group].PullRequests[job.pr] = m.buildReportPR(ctx, job.entry.repoName, job.entry.pr, needsStatus, verbosity)
		}
		return len(jobs[i])
	}, func(i, count int, _ bool) bool {
		evaluated += count
		if showProgress && needsStatus {
			m.console.ProgressBar(evaluated, totalPRs, "Evaluating")
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	if showProgress && needsStatus && totalPRs > 0 {
		m.console.FinishProgress()
	}

	return result, nil
}

// buildReportPR builds a ReportPullRequest from a PR entry.
func (m *Merger) buildReportPR(ctx context.Context, repoName string, pr gh.PullRequest, needsStatus bool, verbosity string) output.ReportPullRequest {
	rpr := output.ReportPullRequest{
		Repository: repoName,
		Number:     pr.Number,
		Title:      pr.Title,
		URL:        pr.URL,
	}

	if !needsStatus {
		return rpr
	}

	// Evaluate status using the same logic as ghprmerge's evaluatePullRequest
	owner := strings.Split(pr.RepoFullName, "/")[0]
	if owner == "" {
		// Fallback: extract owner from the org config
		owner = m.config.Org
	}

	rpr.Status = m.evaluateReportStatus(ctx, owner, repoName, pr)
	return rpr
}

// evaluateReportStatus evaluates the status of a PR for report mode.
// It reuses the same assessment logic as ghprmerge's normal evaluation.
func (m *Merger) evaluateReportStatus(ctx context.Context, owner, repoName string, pr gh.PullRequest) string {
	// Get check status
	checkStatus, err := m.client.GetCheckStatus(ctx, owner, repoName, pr.HeadSHA)
	if err != nil {
		return "error"
	}

	if checkStatus.NoChecks {
		// No checks configured - check branch status
		return m.evaluateReportBranchStatus(ctx, owner, repoName, pr, "no checks configured")
	}

	if checkStatus.Pending {
		return "checks pending"
	}

	if !checkStatus.AllPassing {
		return "checks failing"
	}

	return m.evaluateReportBranchStatus(ctx, owner, repoName, pr, "passing")
}

// evaluateReportBranchStatus evaluates branch status for report mode.
func (m *Merger) evaluateReportBranchStatus(ctx context.Context, owner, repoName string, pr gh.PullRequest, checksState string) string {
	branchStatus, err := m.client.GetBranchStatus(ctx, owner, repoName, pr.Number)
	if err != nil {
		return "error"
	}

	if branchStatus.HasConflict {
		return "conflict"
	}

	if !branchStatus.UpToDate {
		return "needs-rebase"
	}

	return checksState
}
