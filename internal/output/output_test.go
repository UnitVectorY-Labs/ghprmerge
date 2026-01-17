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
	writer := NewWriter(&buf, false)

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

	// Check for key elements in output
	checks := []string{
		"test-org",
		"dependabot/",
		"test-org/repo1",
		"main",
		"#1",
		"would merge",
	}

	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("Output missing expected string: %q\nOutput was:\n%s", check, output)
		}
	}
}

func TestWriterJSONOutput(t *testing.T) {
	var buf bytes.Buffer
	writer := NewWriter(&buf, true)

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
