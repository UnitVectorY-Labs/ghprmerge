package output

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestConsoleColorEnabled(t *testing.T) {
	var buf bytes.Buffer
	c := NewConsole(&buf, false, false, false) // noColor=false (colors enabled)

	green := c.Green("success")
	if !strings.Contains(green, "\033[32m") {
		t.Errorf("Expected ANSI green code in output, got: %q", green)
	}
	if !strings.Contains(green, "success") {
		t.Errorf("Expected 'success' in output, got: %q", green)
	}
	if !strings.Contains(green, "\033[0m") {
		t.Errorf("Expected ANSI reset code in output, got: %q", green)
	}
}

func TestConsoleColorDisabled(t *testing.T) {
	var buf bytes.Buffer
	c := NewConsole(&buf, true, false, false) // color disabled

	green := c.Green("success")
	if green != "success" {
		t.Errorf("Expected plain 'success' with color disabled, got: %q", green)
	}

	red := c.Red("error")
	if red != "error" {
		t.Errorf("Expected plain 'error' with color disabled, got: %q", red)
	}
}

func TestConsoleProgressBar(t *testing.T) {
	var buf bytes.Buffer
	c := NewConsole(&buf, true, false, false) // noColor for predictable output

	c.ProgressBar(5, 10, "Testing")
	output := buf.String()

	if !strings.Contains(output, "5/10") {
		t.Errorf("Expected '5/10' in progress bar, got: %q", output)
	}
	if !strings.Contains(output, "50%") {
		t.Errorf("Expected '50%%' in progress bar, got: %q", output)
	}
	if !strings.Contains(output, "█") {
		t.Errorf("Expected filled blocks in progress bar, got: %q", output)
	}
	if !strings.HasPrefix(output, "\r") {
		t.Errorf("Expected carriage return at start, got: %q", output)
	}
	if !strings.Contains(output, "[") || !strings.Contains(output, "]") {
		t.Errorf("Expected brackets in progress bar, got: %q", output)
	}
	if !strings.Contains(output, "Testing") {
		t.Errorf("Expected label in progress bar, got: %q", output)
	}
}

func TestConsoleProgressBarZeroTotal(t *testing.T) {
	var buf bytes.Buffer
	c := NewConsole(&buf, true, false, false)

	c.ProgressBar(0, 0, "Testing")
	if buf.Len() > 0 {
		t.Errorf("Expected no output for zero total, got: %q", buf.String())
	}
}

func TestConsolePrintHeader(t *testing.T) {
	var buf bytes.Buffer
	c := NewConsole(&buf, true, false, false)

	c.PrintHeader("myorg", "merge mode", "dependabot/")
	output := buf.String()

	if !strings.Contains(output, "ghprmerge") {
		t.Errorf("Expected 'ghprmerge' in header, got: %q", output)
	}
	if !strings.Contains(output, "myorg") {
		t.Errorf("Expected 'myorg' in header, got: %q", output)
	}
	if !strings.Contains(output, "merge mode") {
		t.Errorf("Expected 'merge mode' in header, got: %q", output)
	}
	if !strings.Contains(output, "dependabot/") {
		t.Errorf("Expected 'dependabot/' in header, got: %q", output)
	}
}

func TestConsolePrintRepoResult(t *testing.T) {
	var buf bytes.Buffer
	c := NewConsole(&buf, true, false, false) // noColor for predictable output

	repo := RepositoryResult{
		Name:          "repo1",
		FullName:      "testorg/repo1",
		DefaultBranch: "main",
		PullRequests: []PullRequestResult{
			{
				Number:     42,
				HeadBranch: "dependabot/npm/lodash",
				Title:      "Bump lodash",
				Action:     ActionMerged,
				Reason:     "successfully merged",
			},
		},
	}

	c.PrintRepoResult(repo)
	output := buf.String()

	if !strings.Contains(output, "testorg/repo1") {
		t.Errorf("Expected repo full name in output, got: %q", output)
	}
	if !strings.Contains(output, "#42") {
		t.Errorf("Expected PR number in output, got: %q", output)
	}
	if !strings.Contains(output, "Bump lodash") {
		t.Errorf("Expected PR title in output, got: %q", output)
	}
	if !strings.Contains(output, "merged") {
		t.Errorf("Expected action in output, got: %q", output)
	}
	if !strings.Contains(output, "✓") {
		t.Errorf("Expected checkmark symbol in output, got: %q", output)
	}
}

func TestConsolePrintSummary(t *testing.T) {
	var buf bytes.Buffer
	c := NewConsole(&buf, true, false, false)

	summary := RunSummary{
		ReposProcessed:  100,
		CandidatesFound: 5,
		MergedSuccess:   2,
		RebasedSuccess:  1,
		Skipped:         2,
	}

	c.PrintSummary(summary)
	output := buf.String()

	if !strings.Contains(output, "100 repos scanned") {
		t.Errorf("Expected '100 repos scanned' in summary, got: %q", output)
	}
	if !strings.Contains(output, "5 PRs found") {
		t.Errorf("Expected '5 PRs found' in summary, got: %q", output)
	}
	if !strings.Contains(output, "2 merged") {
		t.Errorf("Expected '2 merged' in summary, got: %q", output)
	}
	if !strings.Contains(output, "1 rebased") {
		t.Errorf("Expected '1 rebased' in summary, got: %q", output)
	}
	if !strings.Contains(output, "2 skipped") {
		t.Errorf("Expected '2 skipped' in summary, got: %q", output)
	}
}

func TestConsoleIsVerbose(t *testing.T) {
	var buf bytes.Buffer

	c1 := NewConsole(&buf, false, true, false)
	if !c1.IsVerbose() {
		t.Error("Expected IsVerbose() to return true")
	}

	c2 := NewConsole(&buf, false, false, false)
	if c2.IsVerbose() {
		t.Error("Expected IsVerbose() to return false")
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a longer string", 10, "this is..."},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc"},
		{"", 5, ""},
		{"test", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := truncateString(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestConsoleColorActions(t *testing.T) {
	var buf bytes.Buffer
	c := NewConsole(&buf, true, false, false) // noColor for predictable output

	tests := []struct {
		action Action
		symbol string
	}{
		{ActionMerged, "✓"},
		{ActionWouldMerge, "✓"},
		{ActionReadyMerge, "✓"},
		{ActionRebased, "↻"},
		{ActionWouldRebase, "↻"},
		{ActionMergeFailed, "✗"},
		{ActionRebaseFailed, "✗"},
		{ActionSkipConflict, "⊘"},
		{ActionSkipChecksFailing, "⊘"},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			got := c.getActionSymbol(tt.action)
			if got != tt.symbol {
				t.Errorf("getActionSymbol(%v) = %q, want %q", tt.action, got, tt.symbol)
			}
		})
	}
}

func TestDigitCount(t *testing.T) {
	tests := []struct {
		input int
		want  int
	}{
		{0, 1},
		{1, 1},
		{9, 1},
		{10, 2},
		{99, 2},
		{100, 3},
		{999, 3},
		{1000, 4},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.input), func(t *testing.T) {
			got := digitCount(tt.input)
			if got != tt.want {
				t.Errorf("digitCount(%d) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestProgressBarPaddedCounter(t *testing.T) {
	var buf bytes.Buffer
	c := NewConsole(&buf, true, false, false)

	c.ProgressBar(1, 100, "Scanning")
	output := buf.String()

	// Counter should be padded: "  1/100"
	if !strings.Contains(output, "  1/100") {
		t.Errorf("Expected padded counter '  1/100' in progress bar, got: %q", output)
	}
}

func TestProgressBarSmoothBlocks(t *testing.T) {
	var buf bytes.Buffer
	c := NewConsole(&buf, true, false, false)

	// At 33%, there should be partial blocks in the output
	c.ProgressBar(1, 3, "Testing")
	output := buf.String()

	// Should contain at least one of the partial block characters
	hasPartial := false
	for _, ch := range "▏▎▍▌▋▊▉" {
		if strings.ContainsRune(output, ch) {
			hasPartial = true
			break
		}
	}
	if !hasPartial && !strings.Contains(output, "█") {
		t.Errorf("Expected filled or partial blocks in progress bar at 33%%, got: %q", output)
	}
}

func TestProgressBarComplete(t *testing.T) {
	var buf bytes.Buffer
	c := NewConsole(&buf, true, false, false)

	c.ProgressBar(10, 10, "Done")
	output := buf.String()

	if !strings.Contains(output, "100%") {
		t.Errorf("Expected '100%%' in completed progress bar, got: %q", output)
	}
	if !strings.Contains(output, "10/10") {
		t.Errorf("Expected '10/10' in completed progress bar, got: %q", output)
	}
}

func TestProgressBarColoredPercentage(t *testing.T) {
	var buf bytes.Buffer
	c := NewConsole(&buf, false, false, false) // noColor=false (colors enabled)

	// Test 100% - should use green for percentage
	c.ProgressBar(10, 10, "Testing")
	output := buf.String()
	if !strings.Contains(output, colorGreen+"100%") {
		t.Errorf("Expected green percentage at 100%%, got: %q", output)
	}

	// Test 70% - should use yellow for percentage
	buf.Reset()
	c.ProgressBar(7, 10, "Testing")
	output = buf.String()
	if !strings.Contains(output, colorYellow+" 70%") {
		t.Errorf("Expected yellow percentage at 70%%, got: %q", output)
	}

	// Test 30% - should use cyan for percentage
	buf.Reset()
	c.ProgressBar(3, 10, "Testing")
	output = buf.String()
	if !strings.Contains(output, colorCyan+" 30%") {
		t.Errorf("Expected cyan percentage at 30%%, got: %q", output)
	}
}

func TestProgressBarGrayBrackets(t *testing.T) {
	var buf bytes.Buffer
	c := NewConsole(&buf, false, false, false) // noColor=false (colors enabled)

	c.ProgressBar(5, 10, "Testing")
	output := buf.String()

	if !strings.Contains(output, colorGray+"[") {
		t.Errorf("Expected gray opening bracket, got: %q", output)
	}
	if !strings.Contains(output, colorGray+"]") {
		t.Errorf("Expected gray closing bracket, got: %q", output)
	}
}

func TestProgressBarGreenFill(t *testing.T) {
	var buf bytes.Buffer
	c := NewConsole(&buf, false, false, false) // noColor=false (colors enabled)

	c.ProgressBar(5, 10, "Testing")
	output := buf.String()

	if !strings.Contains(output, colorGreen) {
		t.Errorf("Expected green color in progress bar fill, got: %q", output)
	}
}

func TestProgressBarEdgeCases(t *testing.T) {
	var buf bytes.Buffer
	c := NewConsole(&buf, true, false, false)

	// current > total should clamp
	c.ProgressBar(15, 10, "Testing")
	output := buf.String()
	if !strings.Contains(output, "100%") {
		t.Errorf("Expected clamped to 100%% when current > total, got: %q", output)
	}

	// current < 0 should clamp
	buf.Reset()
	c.ProgressBar(-5, 10, "Testing")
	output = buf.String()
	if !strings.Contains(output, "0%") {
		t.Errorf("Expected clamped to 0%% when current < 0, got: %q", output)
	}
}

func TestProgressBarConsistentWidth(t *testing.T) {
	var buf bytes.Buffer
	c := NewConsole(&buf, true, false, false)

	// Render at different progress values and verify consistent visible width
	var widths []int
	for i := 1; i <= 10; i++ {
		buf.Reset()
		c.ProgressBar(i, 10, "Scanning")
		// Strip the leading \r
		output := strings.TrimPrefix(buf.String(), "\r")
		widths = append(widths, utf8.RuneCountInString(output))
	}

	for i := 1; i < len(widths); i++ {
		if widths[i] != widths[0] {
			t.Errorf("Progress bar width inconsistent: step 1 = %d runes, step %d = %d runes", widths[0], i+1, widths[i])
		}
	}
}
