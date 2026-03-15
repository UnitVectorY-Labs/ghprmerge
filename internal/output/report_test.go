package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestWriteReportResultJSON(t *testing.T) {
	result := &ReportResult{
		Groups: []ReportGroup{
			{
				SourceBranch: "dependabot/go_modules/foo-1.2.3",
				Count:        3,
				PullRequests: []ReportPullRequest{
					{Repository: "repo-a", Number: 123, Status: "passing", Title: "Bump foo", URL: "https://github.com/org/repo-a/pull/123"},
					{Repository: "repo-b", Number: 456, Status: "needs-rebase", Title: "Bump foo", URL: "https://github.com/org/repo-b/pull/456"},
					{Repository: "repo-c", Number: 789, Status: "checks failing", Title: "Bump foo", URL: "https://github.com/org/repo-c/pull/789"},
				},
			},
		},
	}

	var buf bytes.Buffer
	w := NewWriter(&buf, true, false)
	if err := w.WriteReportResult(result, "standard"); err != nil {
		t.Fatalf("WriteReportResult() error = %v", err)
	}

	// Verify it's valid JSON
	var parsed ReportResult
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	if len(parsed.Groups) != 1 {
		t.Errorf("expected 1 group, got %d", len(parsed.Groups))
	}
	if parsed.Groups[0].SourceBranch != "dependabot/go_modules/foo-1.2.3" {
		t.Errorf("expected sourceBranch = dependabot/go_modules/foo-1.2.3, got %s", parsed.Groups[0].SourceBranch)
	}
	if parsed.Groups[0].Count != 3 {
		t.Errorf("expected count = 3, got %d", parsed.Groups[0].Count)
	}
	if len(parsed.Groups[0].PullRequests) != 3 {
		t.Errorf("expected 3 PRs, got %d", len(parsed.Groups[0].PullRequests))
	}
}

func TestWriteReportResultEmptyJSON(t *testing.T) {
	result := &ReportResult{
		Groups: []ReportGroup{},
	}

	var buf bytes.Buffer
	w := NewWriter(&buf, true, false)
	if err := w.WriteReportResult(result, "standard"); err != nil {
		t.Fatalf("WriteReportResult() error = %v", err)
	}

	var parsed ReportResult
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	if len(parsed.Groups) != 0 {
		t.Errorf("expected 0 groups, got %d", len(parsed.Groups))
	}
}

func TestWriteReportResultHumanEmpty(t *testing.T) {
	result := &ReportResult{
		Groups: []ReportGroup{},
	}

	var buf bytes.Buffer
	w := NewWriter(&buf, false, true)
	if err := w.WriteReportResult(result, "standard"); err != nil {
		t.Fatalf("WriteReportResult() error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "No grouped source branches found.") {
		t.Errorf("expected empty message, got: %s", out)
	}
}

func TestWriteReportResultHumanBrief(t *testing.T) {
	result := &ReportResult{
		Groups: []ReportGroup{
			{
				SourceBranch: "dependabot/foo-1.0",
				Count:        3,
				PullRequests: []ReportPullRequest{
					{Repository: "repo-a", Number: 1, Status: "passing"},
					{Repository: "repo-b", Number: 2, Status: "passing"},
					{Repository: "repo-c", Number: 3, Status: "passing"},
				},
			},
		},
	}

	var buf bytes.Buffer
	w := NewWriter(&buf, false, true) // noColor=true
	if err := w.WriteReportResult(result, "brief"); err != nil {
		t.Fatalf("WriteReportResult() error = %v", err)
	}

	out := buf.String()
	// Brief mode: only branch name and count
	if !strings.Contains(out, "dependabot/foo-1.0") {
		t.Errorf("expected branch name in output, got: %s", out)
	}
	if !strings.Contains(out, "(3 PRs)") {
		t.Errorf("expected PR count in output, got: %s", out)
	}
	// Brief mode should NOT show individual PRs
	if strings.Contains(out, "repo-a") {
		t.Errorf("brief mode should not show individual PRs, got: %s", out)
	}
}

func TestWriteReportResultHumanStandard(t *testing.T) {
	result := &ReportResult{
		Groups: []ReportGroup{
			{
				SourceBranch: "dependabot/foo-1.0",
				Count:        2,
				PullRequests: []ReportPullRequest{
					{Repository: "repo-a", Number: 1, Status: "passing"},
					{Repository: "repo-b", Number: 2, Status: "needs-rebase"},
				},
			},
		},
	}

	var buf bytes.Buffer
	w := NewWriter(&buf, false, true) // noColor=true
	if err := w.WriteReportResult(result, "standard"); err != nil {
		t.Fatalf("WriteReportResult() error = %v", err)
	}

	out := buf.String()
	// Standard mode: branch, count, repo, PR number, status
	if !strings.Contains(out, "dependabot/foo-1.0") {
		t.Errorf("expected branch name in output, got: %s", out)
	}
	if !strings.Contains(out, "repo-a") {
		t.Errorf("expected repo name in output, got: %s", out)
	}
	if !strings.Contains(out, "#1") {
		t.Errorf("expected PR number in output, got: %s", out)
	}
	if !strings.Contains(out, "passing") {
		t.Errorf("expected status in output, got: %s", out)
	}
	if !strings.Contains(out, "needs-rebase") {
		t.Errorf("expected needs-rebase status in output, got: %s", out)
	}
}

func TestWriteReportResultHumanVerbose(t *testing.T) {
	result := &ReportResult{
		Groups: []ReportGroup{
			{
				SourceBranch: "dependabot/foo-1.0",
				Count:        2,
				PullRequests: []ReportPullRequest{
					{Repository: "repo-a", Number: 1, Status: "passing", Title: "Bump foo to 1.0"},
					{Repository: "repo-b", Number: 2, Status: "needs-rebase", Title: "Bump foo to 1.0"},
				},
			},
		},
	}

	var buf bytes.Buffer
	w := NewWriter(&buf, false, true) // noColor=true
	if err := w.WriteReportResult(result, "verbose"); err != nil {
		t.Fatalf("WriteReportResult() error = %v", err)
	}

	out := buf.String()
	// Verbose mode: includes title
	if !strings.Contains(out, "Bump foo to 1.0") {
		t.Errorf("verbose mode should show PR title, got: %s", out)
	}
}

func TestFormatReportEmptyJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := FormatReportEmptyJSON(&buf); err != nil {
		t.Fatalf("FormatReportEmptyJSON() error = %v", err)
	}

	var parsed ReportResult
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if parsed.Groups == nil || len(parsed.Groups) != 0 {
		t.Errorf("expected empty groups array")
	}
}
