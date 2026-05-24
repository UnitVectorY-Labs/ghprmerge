package output

import (
	"fmt"
	"io"
	"strings"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
)

// Unicode block characters for smooth progress bar rendering (1/8 to 7/8 fill).
var partialBlocks = [8]string{" ", "▏", "▎", "▍", "▌", "▋", "▊", "▉"}

const fullBlock = "█"

// minBarWidth is the minimum number of character cells for the progress bar.
const minBarWidth = 10

// defaultTermWidth is the assumed terminal width when detection is not available.
const defaultTermWidth = 80

// Console handles colored, formatted terminal output with progress bar support.
type Console struct {
	w          io.Writer
	noColor    bool
	verbose    bool
	noProgress bool
}

// NewConsole creates a new Console for terminal output.
func NewConsole(w io.Writer, noColor, verbose, noProgress bool) *Console {
	return &Console{
		w:          w,
		noColor:    noColor,
		verbose:    verbose,
		noProgress: noProgress,
	}
}

// IsVerbose returns whether verbose mode is enabled.
func (c *Console) IsVerbose() bool {
	return c.verbose
}

// Writer returns the underlying io.Writer.
func (c *Console) Writer() io.Writer {
	return c.w
}

// color wraps a string with ANSI color codes if color is enabled.
func (c *Console) color(code, s string) string {
	if c.noColor {
		return s
	}
	return code + s + colorReset
}

// Green returns a green colored string.
func (c *Console) Green(s string) string { return c.color(colorGreen, s) }

// Yellow returns a yellow colored string.
func (c *Console) Yellow(s string) string { return c.color(colorYellow, s) }

// Red returns a red colored string.
func (c *Console) Red(s string) string { return c.color(colorRed, s) }

// Cyan returns a cyan colored string.
func (c *Console) Cyan(s string) string { return c.color(colorCyan, s) }

// Bold returns a bold string.
func (c *Console) Bold(s string) string { return c.color(colorBold, s) }

// Dim returns a dim string.
func (c *Console) Dim(s string) string { return c.color(colorDim, s) }

// ProgressBar renders a progress bar on the current line using carriage return.
// It is a no-op when noProgress is set.
func (c *Console) ProgressBar(current, total int, label string) {
	if c.noProgress {
		return
	}
	if total == 0 {
		return
	}

	if current < 0 {
		current = 0
	}
	if current > total {
		current = total
	}

	pct := 100 * current / total

	// Build the counter string: current is right-padded with spaces to match total width.
	tw := digitCount(total)
	counterStr := fmt.Sprintf("%*d/%d", tw, current, total)

	// Calculate bar width based on terminal width.
	// Layout: "\r" + "  " (2) + label + "  " (2) + counter + " " (1) + "[" (1) + bar + "]" (1) + " " (1) + pct (4)
	counterWidth := tw + 1 + digitCount(total)                          // current digits + "/" + total digits
	fixedWidth := 2 + len(label) + 2 + counterWidth + 1 + 1 + 1 + 1 + 4 // spaces, label, counter, brackets, pct
	barWidth := defaultTermWidth - fixedWidth
	if barWidth < minBarWidth {
		barWidth = minBarWidth
	}

	// Compute filled portion using eighths for smooth partial steps.
	filledEighths := (current * barWidth * 8) / total
	fullBlocks := filledEighths / 8
	partialEighths := filledEighths % 8
	emptyBlocks := barWidth - fullBlocks
	if partialEighths > 0 {
		emptyBlocks--
	}
	if emptyBlocks < 0 {
		emptyBlocks = 0
	}

	// Build progress bar content.
	var bar strings.Builder
	bar.WriteString(strings.Repeat(fullBlock, fullBlocks))
	if partialEighths > 0 {
		bar.WriteString(partialBlocks[partialEighths])
	}
	bar.WriteString(strings.Repeat(" ", emptyBlocks))

	// Percentage color: green at 100%, yellow at 60%+, cyan otherwise.
	percentColor := colorCyan
	if pct >= 100 {
		percentColor = colorGreen
	} else if pct >= 60 {
		percentColor = colorYellow
	}

	// Assemble the line.
	var b strings.Builder
	b.WriteString("\r  ")
	b.WriteString(c.color(colorCyan, label))
	b.WriteString("  ")
	b.WriteString(c.color(colorGray, counterStr))
	b.WriteString(" ")
	b.WriteString(c.color(colorGray, "["))
	b.WriteString(c.color(colorGreen, bar.String()))
	b.WriteString(c.color(colorGray, "]"))
	b.WriteString(" ")
	b.WriteString(c.color(percentColor, fmt.Sprintf("%3d%%", pct)))

	fmt.Fprint(c.w, b.String())
}

// digitCount returns the number of decimal digits in a non-negative integer.
func digitCount(n int) int {
	if n <= 0 {
		return 1
	}
	count := 0
	for n > 0 {
		count++
		n /= 10
	}
	return count
}

// FinishProgress completes the progress bar by adding a newline.
// It is a no-op when noProgress is set.
func (c *Console) FinishProgress() {
	if c.noProgress {
		return
	}
	fmt.Fprintln(c.w)
}

// ClearCurrentLine clears the current terminal line.
// It is a no-op when noProgress is set.
func (c *Console) ClearCurrentLine() {
	if c.noProgress {
		return
	}
	fmt.Fprint(c.w, "\r\033[2K")
}

// ClearLines clears n lines above the current position using ANSI escape codes.
func (c *Console) ClearLines(n int) {
	for i := 0; i < n; i++ {
		fmt.Fprint(c.w, "\033[A\033[2K")
	}
}

// PrintHeader prints the application header.
func (c *Console) PrintHeader(org, mode, branch string) {
	fmt.Fprintf(c.w, "%s %s\n", c.Bold(c.Cyan("ghprmerge")), c.Dim("─ "+org))
	fmt.Fprintf(c.w, "%s\n", c.Dim(fmt.Sprintf("Mode: %s │ Branch: %s", mode, branch)))
}

// PrintRepoResult prints the result for a single repository's pull requests.
// It returns the number of terminal lines written.
func (c *Console) PrintRepoResult(repo RepositoryResult) int {
	if repo.Skipped {
		fmt.Fprintf(c.w, "  %s %s\n", c.Dim("⊘"), c.Dim(repo.FullName+" ─ "+repo.SkipReason))
		return 1
	}

	if len(repo.PullRequests) == 0 {
		fmt.Fprintf(c.w, "  %s %s\n", c.Dim("─"), c.Dim(repo.FullName+" ─ no matching pull requests"))
		return 1
	}
	lines := 0
	for _, pr := range repo.PullRequests {
		symbol := c.getActionSymbol(pr.Action)
		colored := c.colorAction(symbol, pr.Action)
		title := truncateString(pr.Title, 50)

		fmt.Fprintf(c.w, "  %s %s #%d %s\n", colored, c.Bold(repo.FullName), pr.Number, title)
		actionStr := string(pr.Action)
		if pr.Reason != "" {
			actionStr += " ─ " + pr.Reason
		}
		fmt.Fprintf(c.w, "    %s\n", c.colorActionText(actionStr, pr.Action))
		lines += 2
	}
	return lines
}

// PrintPendingAction prints a single pending action line for confirmation mode.
// It returns the number of terminal lines written.
func (c *Console) PrintPendingAction(repo RepositoryResult, pr PullRequestResult) int {
	symbol := c.getActionSymbol(pr.Action)
	colored := c.colorAction(symbol, pr.Action)
	title := truncateString(pr.Title, 50)
	fmt.Fprintf(c.w, "  %s %s #%d %s\n", colored, repo.FullName, pr.Number, title)
	fmt.Fprintf(c.w, "    %s\n", c.colorActionText(string(pr.Action), pr.Action))
	return 2
}

// PrintSummary prints a condensed summary line.
func (c *Console) PrintSummary(summary RunSummary) {
	fmt.Fprintf(c.w, "%s\n", c.Dim("────────────────────────────────────────────────────"))

	parts := []string{
		fmt.Sprintf("%d repos scanned", summary.ReposProcessed),
		fmt.Sprintf("%d PRs found", summary.CandidatesFound),
	}

	if summary.MergedSuccess > 0 {
		parts = append(parts, c.Green(fmt.Sprintf("%d merged", summary.MergedSuccess)))
	}
	if summary.RebasedSuccess > 0 {
		parts = append(parts, c.Yellow(fmt.Sprintf("%d rebased", summary.RebasedSuccess)))
	}
	if summary.WouldMerge > 0 {
		parts = append(parts, c.Green(fmt.Sprintf("%d would merge", summary.WouldMerge)))
	}
	if summary.WouldRebase > 0 {
		parts = append(parts, c.Yellow(fmt.Sprintf("%d would rebase", summary.WouldRebase)))
	}
	if summary.ReadyToMerge > 0 {
		parts = append(parts, c.Green(fmt.Sprintf("%d ready to merge", summary.ReadyToMerge)))
	}
	if summary.MergeFailed > 0 {
		parts = append(parts, c.Red(fmt.Sprintf("%d merge failed", summary.MergeFailed)))
	}
	if summary.RebaseFailed > 0 {
		parts = append(parts, c.Red(fmt.Sprintf("%d rebase failed", summary.RebaseFailed)))
	}
	if summary.Skipped > 0 {
		parts = append(parts, c.Dim(fmt.Sprintf("%d skipped", summary.Skipped)))
	}

	fmt.Fprintf(c.w, "%s\n", strings.Join(parts, " │ "))
}

// getActionSymbol returns a Unicode symbol for the action type.
func (c *Console) getActionSymbol(action Action) string {
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

// colorAction returns the symbol colored based on the action type.
func (c *Console) colorAction(symbol string, action Action) string {
	switch action {
	case ActionMerged, ActionWouldMerge, ActionReadyMerge:
		return c.Green(symbol)
	case ActionRebased, ActionWouldRebase:
		return c.Yellow(symbol)
	case ActionMergeFailed, ActionRebaseFailed:
		return c.Red(symbol)
	default:
		if strings.HasPrefix(string(action), "skip:") {
			return c.Dim(symbol)
		}
		return symbol
	}
}

// colorActionText returns the action text colored based on the action type.
func (c *Console) colorActionText(text string, action Action) string {
	switch action {
	case ActionMerged, ActionWouldMerge, ActionReadyMerge:
		return c.Green(text)
	case ActionRebased, ActionWouldRebase:
		return c.Yellow(text)
	case ActionMergeFailed, ActionRebaseFailed:
		return c.Red(text)
	default:
		return c.Dim(text)
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
