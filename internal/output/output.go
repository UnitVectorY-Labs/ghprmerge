// Package output handles formatting and displaying results.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// Action represents the action taken or planned for a pull request.
type Action string

const (
	// Analysis mode actions (what would happen)
	ActionWouldMerge  Action = "would merge"
	ActionWouldRebase Action = "would rebase"
	ActionReadyMerge  Action = "ready to merge" // Ready but merge not enabled

	// Execution mode actions (what happened)
	ActionMerged       Action = "merged"
	ActionMergeFailed  Action = "merge failed"
	ActionRebased      Action = "rebased"
	ActionRebaseFailed Action = "rebase failed"

	// Skip reasons
	ActionSkipNotTargetingDefault Action = "skip: not targeting default branch"
	ActionSkipBranchNoMatch       Action = "skip: branch does not match source pattern"
	ActionSkipDraft               Action = "skip: draft PR"
	ActionSkipConflict            Action = "skip: merge conflict"
	ActionSkipChecksFailing       Action = "skip: checks failing"
	ActionSkipChecksPending       Action = "skip: checks pending"
	ActionSkipNoChecks            Action = "skip: no checks found"
	ActionSkipBranchBehind        Action = "skip: branch behind default"
	ActionSkipAwaitingChecks      Action = "skip: branch updated, awaiting checks"
	ActionSkipPermissions         Action = "skip: insufficient permissions"
	ActionSkipAPIError            Action = "skip: API error"
	ActionSkipRepoLimit           Action = "skip: repo limit reached"
)

// SkipReason represents a categorized skip reason for summary grouping.
type SkipReason string

const (
	ReasonNotTargetingDefault SkipReason = "not targeting default branch"
	ReasonBranchNoMatch       SkipReason = "branch does not match source pattern"
	ReasonDraft               SkipReason = "draft PR"
	ReasonConflict            SkipReason = "merge conflict"
	ReasonChecksFailing       SkipReason = "checks failing"
	ReasonChecksPending       SkipReason = "checks pending"
	ReasonNoChecks            SkipReason = "no checks found"
	ReasonBranchBehind        SkipReason = "branch behind default"
	ReasonAwaitingChecks      SkipReason = "branch updated, awaiting checks"
	ReasonPermissions         SkipReason = "insufficient permissions"
	ReasonAPIError            SkipReason = "API error"
	ReasonRepoLimit           SkipReason = "repo limit reached"
)

// PullRequestResult represents the result for a single pull request.
type PullRequestResult struct {
	Number     int        `json:"number"`
	URL        string     `json:"url"`
	HeadBranch string     `json:"head_branch"`
	Title      string     `json:"title"`
	Action     Action     `json:"action"`
	Reason     string     `json:"reason,omitempty"`
	SkipReason SkipReason `json:"skip_reason,omitempty"`
}

// RepositoryResult represents the results for a single repository.
type RepositoryResult struct {
	Name          string              `json:"name"`
	FullName      string              `json:"full_name"`
	DefaultBranch string              `json:"default_branch"`
	PullRequests  []PullRequestResult `json:"pull_requests"`
	Skipped       bool                `json:"skipped,omitempty"`
	SkipReason    string              `json:"skip_reason,omitempty"`
}

// RunResult represents the complete result of a run.
type RunResult struct {
	Metadata     RunMetadata        `json:"metadata"`
	Repositories []RepositoryResult `json:"repositories"`
	Summary      RunSummary         `json:"summary"`
}

// RunMetadata contains metadata about the run.
type RunMetadata struct {
	Org           string    `json:"org"`
	SourceBranch  string    `json:"source_branch"`
	Mode          string    `json:"mode"`
	Rebase        bool      `json:"rebase"`
	Merge         bool      `json:"merge"`
	RepoLimit     int       `json:"repo_limit,omitempty"`
	RepoLimitDesc string    `json:"repo_limit_desc,omitempty"`
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
}

// RunSummary contains summary statistics for the run.
type RunSummary struct {
	ReposProcessed    int            `json:"repos_processed"`
	ReposSkipped      int            `json:"repos_skipped"`
	CandidatesFound   int            `json:"candidates_found"`
	MergedSuccess     int            `json:"merged_success"`
	MergeFailed       int            `json:"merge_failed"`
	RebasedSuccess    int            `json:"rebased_success"`
	RebaseFailed      int            `json:"rebase_failed"`
	WouldMerge        int            `json:"would_merge,omitempty"`
	WouldRebase       int            `json:"would_rebase,omitempty"`
	ReadyToMerge      int            `json:"ready_to_merge,omitempty"`
	Skipped           int            `json:"skipped"`
	SkippedByReason   map[string]int `json:"skipped_by_reason,omitempty"`
}

// Writer handles output formatting.
type Writer struct {
	out      io.Writer
	jsonMode bool
}

// NewWriter creates a new Writer.
func NewWriter(out io.Writer, jsonMode bool) *Writer {
	return &Writer{
		out:      out,
		jsonMode: jsonMode,
	}
}

// WriteResult writes the complete run result.
func (w *Writer) WriteResult(result *RunResult) error {
	if w.jsonMode {
		return w.writeJSON(result)
	}
	return w.writeHuman(result)
}

// writeJSON writes the result as JSON.
func (w *Writer) writeJSON(result *RunResult) error {
	encoder := json.NewEncoder(w.out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// writeHuman writes the result in human-readable format.
func (w *Writer) writeHuman(result *RunResult) error {
	// Print header
	fmt.Fprintf(w.out, "╔══════════════════════════════════════════════════════════════════════════════╗\n")
	fmt.Fprintf(w.out, "║  ghprmerge - %s\n", result.Metadata.Org)
	fmt.Fprintf(w.out, "╚══════════════════════════════════════════════════════════════════════════════╝\n")
	fmt.Fprintf(w.out, "\n")
	fmt.Fprintf(w.out, "  Source branch: %s\n", result.Metadata.SourceBranch)
	fmt.Fprintf(w.out, "  Mode:          %s\n", result.Metadata.Mode)
	if result.Metadata.RepoLimitDesc != "" {
		fmt.Fprintf(w.out, "  Limit:         %s\n", result.Metadata.RepoLimitDesc)
	}
	fmt.Fprintf(w.out, "\n")

	// Print each repository
	for _, repo := range result.Repositories {
		fmt.Fprintf(w.out, "┌─ %s (default: %s)\n", repo.FullName, repo.DefaultBranch)

		if repo.Skipped {
			fmt.Fprintf(w.out, "│  ⊘ Repository skipped: %s\n", repo.SkipReason)
			fmt.Fprintf(w.out, "└────────────────────────────────────────────────────────────────────────────────\n\n")
			continue
		}

		if len(repo.PullRequests) == 0 {
			fmt.Fprintf(w.out, "│  No matching pull requests\n")
		} else {
			for _, pr := range repo.PullRequests {
				symbol := w.getActionSymbol(pr.Action)
				fmt.Fprintf(w.out, "│  %s PR #%-5d  %-40s\n", symbol, pr.Number, truncateString(pr.Title, 40))
				fmt.Fprintf(w.out, "│              Branch: %s\n", pr.HeadBranch)
				fmt.Fprintf(w.out, "│              Action: %s\n", pr.Action)
				if pr.Reason != "" {
					fmt.Fprintf(w.out, "│              Detail: %s\n", pr.Reason)
				}
			}
		}
		fmt.Fprintf(w.out, "└────────────────────────────────────────────────────────────────────────────────\n\n")
	}

	// Print summary
	fmt.Fprintf(w.out, "═══════════════════════════════════════════════════════════════════════════════\n")
	fmt.Fprintf(w.out, "                                    SUMMARY\n")
	fmt.Fprintf(w.out, "═══════════════════════════════════════════════════════════════════════════════\n")
	fmt.Fprintf(w.out, "  Repositories processed:  %d\n", result.Summary.ReposProcessed)
	fmt.Fprintf(w.out, "  Repositories skipped:    %d\n", result.Summary.ReposSkipped)
	fmt.Fprintf(w.out, "  Candidates found:        %d\n", result.Summary.CandidatesFound)
	fmt.Fprintf(w.out, "\n")

	// Show analysis mode results
	if result.Summary.WouldMerge > 0 || result.Summary.WouldRebase > 0 {
		fmt.Fprintf(w.out, "  Would merge:             %d\n", result.Summary.WouldMerge)
		fmt.Fprintf(w.out, "  Would rebase:            %d\n", result.Summary.WouldRebase)
	}

	// Show ready to merge (when merge not enabled)
	if result.Summary.ReadyToMerge > 0 {
		fmt.Fprintf(w.out, "  Ready to merge:          %d\n", result.Summary.ReadyToMerge)
	}

	// Show execution mode results
	if result.Summary.MergedSuccess > 0 || result.Summary.MergeFailed > 0 {
		fmt.Fprintf(w.out, "  Merged successfully:     %d\n", result.Summary.MergedSuccess)
		fmt.Fprintf(w.out, "  Merge failed:            %d\n", result.Summary.MergeFailed)
	}
	if result.Summary.RebasedSuccess > 0 || result.Summary.RebaseFailed > 0 {
		fmt.Fprintf(w.out, "  Rebased successfully:    %d\n", result.Summary.RebasedSuccess)
		fmt.Fprintf(w.out, "  Rebase failed:           %d\n", result.Summary.RebaseFailed)
	}

	fmt.Fprintf(w.out, "  Skipped:                 %d\n", result.Summary.Skipped)

	// Show skipped by reason
	if len(result.Summary.SkippedByReason) > 0 {
		fmt.Fprintf(w.out, "\n  Skipped by reason:\n")
		for reason, count := range result.Summary.SkippedByReason {
			fmt.Fprintf(w.out, "    %-30s %d\n", reason, count)
		}
	}

	fmt.Fprintf(w.out, "═══════════════════════════════════════════════════════════════════════════════\n")

	return nil
}

// getActionSymbol returns a symbol for the action type.
func (w *Writer) getActionSymbol(action Action) string {
	switch action {
	case ActionMerged, ActionWouldMerge, ActionReadyMerge:
		return "✓"
	case ActionRebased, ActionWouldRebase:
		return "↻"
	case ActionMergeFailed, ActionRebaseFailed:
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
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
