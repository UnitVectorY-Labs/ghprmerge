package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestWriterHumanOutput(t *testing.T) {
	var buf bytes.Buffer
	writer := NewWriter(&buf, false, false, true) // noColor=true for predictable output

	result := &RunResult{
		Metadata: RunMetadata{
			Org:          "test-org",
			SourceBranch: "dependabot/",
			Mode:         "analysis only (no mutations)",
			Rebase:       false,
			Merge:        false,
			StartTime:    time.Now(),
			EndTime:      time.Now(),
		},
		Repositories: []RepositoryResult{
			{
				Name:          "repo1",
				FullName:      "test-org/repo1",
				DefaultBranch: "main",
				PullRequests: []PullRequestResult{
					{
						Number:     1,
						URL:        "https://github.com/test-org/repo1/pull/1",
						HeadBranch: "dependabot/npm/lodash",
						Title:      "Bump lodash to 4.17.21",
						Action:     ActionWouldMerge,
						Reason:     "all checks passing",
					},
				},
			},
		},
		Summary: RunSummary{
			ReposProcessed:  1,
			CandidatesFound: 1,
			WouldMerge:      1,
		},
	}

	err := writer.WriteResult(result)
	if err != nil {
		t.Fatalf("WriteResult() error = %v", err)
	}

	output := buf.String()

	// Check for key summary elements
	checks := []string{
		"1 repos scanned",
		"1 PRs found",
		"1 would merge",
	}

	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("Output missing expected string: %q\nOutput was:\n%s", check, output)
		}
	}
}

func TestWriterJSONOutput(t *testing.T) {
	var buf bytes.Buffer
	writer := NewWriter(&buf, true, false, true)

	result := &RunResult{
		Metadata: RunMetadata{
			Org:          "test-org",
			SourceBranch: "dependabot/",
			Mode:         "analysis only",
			Rebase:       false,
			Merge:        false,
			StartTime:    time.Now(),
			EndTime:      time.Now(),
		},
		Repositories: []RepositoryResult{
			{
				Name:          "repo1",
				FullName:      "test-org/repo1",
				DefaultBranch: "main",
				PullRequests: []PullRequestResult{
					{
						Number:     1,
						URL:        "https://github.com/test-org/repo1/pull/1",
						HeadBranch: "dependabot/npm/lodash",
						Title:      "Bump lodash to 4.17.21",
						Action:     ActionWouldMerge,
						Reason:     "all checks passing",
					},
				},
			},
		},
		Summary: RunSummary{
			ReposProcessed:  1,
			CandidatesFound: 1,
			WouldMerge:      1,
		},
	}

	err := writer.WriteResult(result)
	if err != nil {
		t.Fatalf("WriteResult() error = %v", err)
	}

	// Verify it's valid JSON
	var parsed RunResult
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify key fields
	if parsed.Metadata.Org != "test-org" {
		t.Errorf("Org = %v, want test-org", parsed.Metadata.Org)
	}
	if len(parsed.Repositories) != 1 {
		t.Errorf("Repositories count = %v, want 1", len(parsed.Repositories))
	}
	if len(parsed.Repositories[0].PullRequests) != 1 {
		t.Errorf("PullRequests count = %v, want 1", len(parsed.Repositories[0].PullRequests))
	}
	if parsed.Repositories[0].PullRequests[0].Action != ActionWouldMerge {
		t.Errorf("Action = %v, want %v", parsed.Repositories[0].PullRequests[0].Action, ActionWouldMerge)
	}
}

func TestActionConstants(t *testing.T) {
	// Verify action constants are what we expect
	tests := []struct {
		action Action
		want   string
	}{
		{ActionWouldMerge, "would merge"},
		{ActionMerged, "merged"},
		{ActionWouldRebase, "would rebase"},
		{ActionSkipChecksFailing, "skip: checks failing"},
		{ActionSkipBranchBehind, "skip: branch behind default"},
		{ActionSkipConflict, "skip: merge conflict"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if string(tt.action) != tt.want {
				t.Errorf("Action = %v, want %v", tt.action, tt.want)
			}
		})
	}
}

func TestWriterSummaryShowsCorrectCounts(t *testing.T) {
	var buf bytes.Buffer
	writer := NewWriter(&buf, false, false, true)

	result := &RunResult{
		Metadata: RunMetadata{
			Org:          "test-org",
			SourceBranch: "dependabot/",
			Mode:         "merge mode",
		},
		Repositories: []RepositoryResult{
			{
				Name:          "repo1",
				FullName:      "test-org/repo1",
				DefaultBranch: "main",
				PullRequests: []PullRequestResult{
					{Number: 1, Action: ActionMerged},
				},
			},
			{
				Name:          "repo2",
				FullName:      "test-org/repo2",
				DefaultBranch: "main",
				PullRequests:  []PullRequestResult{},
			},
		},
		Summary: RunSummary{
			ReposProcessed:  2,
			CandidatesFound: 1,
			MergedSuccess:   1,
			Skipped:         0,
		},
	}

	err := writer.WriteResult(result)
	if err != nil {
		t.Fatalf("WriteResult() error = %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "2 repos scanned") {
		t.Errorf("Expected '2 repos scanned' in output:\n%s", output)
	}
	if !strings.Contains(output, "1 PRs found") {
		t.Errorf("Expected '1 PRs found' in output:\n%s", output)
	}
	if !strings.Contains(output, "1 merged") {
		t.Errorf("Expected '1 merged' in output:\n%s", output)
	}
}

func TestWriterSummaryHidesZeroCounts(t *testing.T) {
	var buf bytes.Buffer
	writer := NewWriter(&buf, false, false, true)

	result := &RunResult{
		Metadata: RunMetadata{
			Org:          "test-org",
			SourceBranch: "dependabot/",
			Mode:         "analysis only",
		},
		Summary: RunSummary{
			ReposProcessed:  5,
			CandidatesFound: 0,
		},
	}

	err := writer.WriteResult(result)
	if err != nil {
		t.Fatalf("WriteResult() error = %v", err)
	}

	output := buf.String()

	// Should show repos scanned and PRs found
	if !strings.Contains(output, "5 repos scanned") {
		t.Errorf("Expected '5 repos scanned' in output:\n%s", output)
	}
	if !strings.Contains(output, "0 PRs found") {
		t.Errorf("Expected '0 PRs found' in output:\n%s", output)
	}

	// Should NOT show merge/rebase counts when zero
	if strings.Contains(output, "merged") {
		t.Errorf("Expected no merge count when zero:\n%s", output)
	}
	if strings.Contains(output, "rebased") {
		t.Errorf("Expected no rebase count when zero:\n%s", output)
	}
}
