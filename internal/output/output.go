// Package output handles formatting and displaying results.
package output

import (
	"encoding/json"
	"io"
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
	ReposProcessed  int            `json:"repos_processed"`
	ReposSkipped    int            `json:"repos_skipped"`
	CandidatesFound int            `json:"candidates_found"`
	MergedSuccess   int            `json:"merged_success"`
	MergeFailed     int            `json:"merge_failed"`
	RebasedSuccess  int            `json:"rebased_success"`
	RebaseFailed    int            `json:"rebase_failed"`
	WouldMerge      int            `json:"would_merge,omitempty"`
	WouldRebase     int            `json:"would_rebase,omitempty"`
	ReadyToMerge    int            `json:"ready_to_merge,omitempty"`
	Skipped         int            `json:"skipped"`
	SkippedByReason map[string]int `json:"skipped_by_reason,omitempty"`
}

// Writer handles output formatting.
type Writer struct {
	out      io.Writer
	jsonMode bool
	noColor  bool
}

// NewWriter creates a new Writer.
func NewWriter(out io.Writer, jsonMode bool, noColor bool) *Writer {
	return &Writer{
		out:      out,
		jsonMode: jsonMode,
		noColor:  noColor,
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

// writeHuman writes the result in human-readable format as a condensed summary.
func (w *Writer) writeHuman(result *RunResult) error {
	c := NewConsole(w.out, w.noColor, false)
	c.PrintSummary(result.Summary)
	return nil
}
