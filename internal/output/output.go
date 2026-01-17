// Package output handles formatting and displaying results.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// Action represents the action taken or planned for a pull request.
type Action string

const (
	ActionWouldMerge      Action = "would merge"
	ActionMerged          Action = "merged"
	ActionWouldRebase     Action = "would rebase then merge"
	ActionRebased         Action = "rebased"
	ActionSkippedChecks   Action = "skipped checks failing"
	ActionSkippedOutdated Action = "skipped branch out of date"
	ActionSkippedConflict Action = "skipped merge conflict"
	ActionSkippedDraft    Action = "skipped draft"
	ActionSkippedMerge    Action = "skipped merge failed"
	ActionSkippedRebase   Action = "skipped rebase failed"
	ActionSkippedLimit    Action = "skipped limit reached"
)

// PullRequestResult represents the result for a single pull request.
type PullRequestResult struct {
	Number     int    `json:"number"`
	URL        string `json:"url"`
	HeadBranch string `json:"head_branch"`
	Title      string `json:"title"`
	Action     Action `json:"action"`
	Reason     string `json:"reason,omitempty"`
}

// RepositoryResult represents the results for a single repository.
type RepositoryResult struct {
	Name          string              `json:"name"`
	FullName      string              `json:"full_name"`
	DefaultBranch string              `json:"default_branch"`
	PullRequests  []PullRequestResult `json:"pull_requests"`
}

// RunResult represents the complete result of a run.
type RunResult struct {
	Metadata     RunMetadata        `json:"metadata"`
	Repositories []RepositoryResult `json:"repositories"`
	Summary      RunSummary         `json:"summary"`
}

// RunMetadata contains metadata about the run.
type RunMetadata struct {
	Org          string    `json:"org"`
	SourceBranch string    `json:"source_branch"`
	DryRun       bool      `json:"dry_run"`
	Rebase       bool      `json:"rebase"`
	Limit        int       `json:"limit,omitempty"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
}

// RunSummary contains summary statistics for the run.
type RunSummary struct {
	TotalRepositories int `json:"total_repositories"`
	TotalPullRequests int `json:"total_pull_requests"`
	Merged            int `json:"merged"`
	WouldMerge        int `json:"would_merge"`
	Rebased           int `json:"rebased"`
	WouldRebase       int `json:"would_rebase"`
	Skipped           int `json:"skipped"`
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
	fmt.Fprintf(w.out, "ghprmerge - %s\n", result.Metadata.Org)
	fmt.Fprintf(w.out, "Source branch pattern: %s\n", result.Metadata.SourceBranch)
	if result.Metadata.DryRun {
		fmt.Fprintf(w.out, "Mode: dry-run (no changes will be made)\n")
	} else {
		fmt.Fprintf(w.out, "Mode: live\n")
	}
	fmt.Fprintf(w.out, "Rebase: %v\n", result.Metadata.Rebase)
	if result.Metadata.Limit > 0 {
		fmt.Fprintf(w.out, "Limit: %d\n", result.Metadata.Limit)
	}
	fmt.Fprintln(w.out, "")

	// Print each repository
	for _, repo := range result.Repositories {
		fmt.Fprintf(w.out, "Repository: %s (default: %s)\n", repo.FullName, repo.DefaultBranch)

		if len(repo.PullRequests) == 0 {
			fmt.Fprintf(w.out, "  No matching pull requests\n")
		} else {
			for _, pr := range repo.PullRequests {
				fmt.Fprintf(w.out, "  PR #%d: %s\n", pr.Number, pr.Title)
				fmt.Fprintf(w.out, "    Branch: %s\n", pr.HeadBranch)
				fmt.Fprintf(w.out, "    URL: %s\n", pr.URL)
				fmt.Fprintf(w.out, "    Action: %s\n", pr.Action)
				if pr.Reason != "" {
					fmt.Fprintf(w.out, "    Reason: %s\n", pr.Reason)
				}
			}
		}
		fmt.Fprintln(w.out, "")
	}

	// Print summary
	fmt.Fprintln(w.out, "Summary:")
	fmt.Fprintf(w.out, "  Repositories scanned: %d\n", result.Summary.TotalRepositories)
	fmt.Fprintf(w.out, "  Pull requests found: %d\n", result.Summary.TotalPullRequests)
	if result.Metadata.DryRun {
		fmt.Fprintf(w.out, "  Would merge: %d\n", result.Summary.WouldMerge)
		fmt.Fprintf(w.out, "  Would rebase: %d\n", result.Summary.WouldRebase)
	} else {
		fmt.Fprintf(w.out, "  Merged: %d\n", result.Summary.Merged)
		fmt.Fprintf(w.out, "  Rebased: %d\n", result.Summary.Rebased)
	}
	fmt.Fprintf(w.out, "  Skipped: %d\n", result.Summary.Skipped)

	return nil
}
