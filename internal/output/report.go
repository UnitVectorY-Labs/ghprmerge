// Package output handles formatting and displaying results.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// ReportPullRequest represents a single PR in a report group.
type ReportPullRequest struct {
	Repository string `json:"repository"`
	Number     int    `json:"number"`
	Status     string `json:"status"`
	Title      string `json:"title,omitempty"`
	URL        string `json:"url,omitempty"`
}

// ReportGroup represents a group of PRs sharing the same source branch.
type ReportGroup struct {
	SourceBranch string              `json:"sourceBranch"`
	Count        int                 `json:"count"`
	PullRequests []ReportPullRequest `json:"pullRequests"`
}

// ReportResult represents the complete report output.
type ReportResult struct {
	Groups []ReportGroup `json:"groups"`
}

// WriteReportResult writes the report result in JSON or human-readable format.
func (w *Writer) WriteReportResult(result *ReportResult, verbosity string) error {
	if w.jsonMode {
		return w.writeReportJSON(result)
	}
	return w.writeReportHuman(result, verbosity)
}

// writeReportJSON writes the report as JSON.
func (w *Writer) writeReportJSON(result *ReportResult) error {
	encoder := json.NewEncoder(w.out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// writeReportHuman writes the report in human-readable format.
func (w *Writer) writeReportHuman(result *ReportResult, verbosity string) error {
	if len(result.Groups) == 0 {
		fmt.Fprintln(w.out, "No grouped source branches found.")
		return nil
	}

	c := NewConsole(w.out, w.noColor, false)
	c.PrintReport(result, verbosity)
	return nil
}

// PrintReport prints the report output to the console.
func (c *Console) PrintReport(result *ReportResult, verbosity string) {
	for i, group := range result.Groups {
		if i > 0 {
			fmt.Fprintln(c.w)
		}
		c.printReportGroup(group, verbosity)
	}
}

// PrintReportHeader prints the report header.
func (c *Console) PrintReportHeader(org string, repoCount int) {
	fmt.Fprintf(c.w, "%s %s\n", c.Bold(c.Cyan("ghprmerge")), c.Dim("─ "+org))
	fmt.Fprintf(c.w, "%s\n", c.Dim("Mode: report"))
}

// printReportGroup prints a single report group.
func (c *Console) printReportGroup(group ReportGroup, verbosity string) {
	// Branch name and count
	fmt.Fprintf(c.w, "%s %s\n", c.Bold(group.SourceBranch), c.Dim(fmt.Sprintf("(%d PRs)", group.Count)))

	if verbosity == "brief" {
		return
	}

	// Standard and verbose: show each PR
	for _, pr := range group.PullRequests {
		symbol := c.reportStatusSymbol(pr.Status)
		colored := c.reportStatusColor(symbol, pr.Status)

		line := fmt.Sprintf("  %s %s #%d", colored, pr.Repository, pr.Number)
		if verbosity == "verbose" && pr.Title != "" {
			line += " " + truncateString(pr.Title, 50)
		}
		line += " " + c.reportStatusColor(pr.Status, pr.Status)
		fmt.Fprintln(c.w, line)
	}
}

// reportStatusSymbol returns a symbol for the report status.
func (c *Console) reportStatusSymbol(status string) string {
	switch status {
	case "passing", "ready to merge", "no checks configured":
		return "✓"
	case "needs-rebase":
		return "↻"
	case "conflict", "checks failing", "checks pending", "draft":
		return "⊘"
	default:
		return "•"
	}
}

// reportStatusColor colorizes the text based on report status.
func (c *Console) reportStatusColor(text string, status string) string {
	switch status {
	case "passing", "ready to merge", "no checks configured":
		return c.Green(text)
	case "needs-rebase":
		return c.Yellow(text)
	case "conflict", "checks failing":
		return c.Red(text)
	case "checks pending", "draft":
		return c.Dim(text)
	default:
		return c.Dim(text)
	}
}

// PrintReportSummary prints the report summary line.
func (c *Console) PrintReportSummary(totalGroups, totalPRs int) {
	fmt.Fprintf(c.w, "%s\n", c.Dim("────────────────────────────────────────────────────"))
	parts := []string{
		fmt.Sprintf("%d branch groups", totalGroups),
		fmt.Sprintf("%d PRs", totalPRs),
	}
	fmt.Fprintf(c.w, "%s\n", strings.Join(parts, " │ "))
}

// FormatReportEmptyJSON returns an empty report JSON result with an empty groups array.
func FormatReportEmptyJSON(w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(&ReportResult{Groups: []ReportGroup{}})
}
