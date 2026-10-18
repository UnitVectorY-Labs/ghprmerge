package merger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/UnitVectorY-Labs/ghprmerge/internal/config"
	gh "github.com/UnitVectorY-Labs/ghprmerge/internal/github"
	"github.com/UnitVectorY-Labs/ghprmerge/internal/output"
)

func TestRebaseFreshnessAndChecks(t *testing.T) {
	for name, checks := range map[string]*gh.CheckStatus{
		"failing": {AllPassing: false}, "pending": {Pending: true},
		"passing": {AllPassing: true}, "absent": {NoChecks: true},
	} {
		for _, confirm := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/confirm=%t", name, confirm), func(t *testing.T) {
				mock := gh.NewMockClient()
				mock.Repositories = []gh.Repository{{Name: "repo", FullName: "org/repo", DefaultBranch: "main"}}
				mock.PullRequests["org/repo"] = []gh.PullRequest{
					{Number: 1, HeadBranch: "dependabot/a", BaseBranch: "main", HeadSHA: "sha"},
					{Number: 2, HeadBranch: "dependabot/b", BaseBranch: "main", HeadSHA: "sha"},
				}
				mock.CheckStatuses["org/repo/sha"] = checks
				mock.BranchStatuses["org/repo/"+string(rune(2))] = &gh.BranchStatus{BehindBy: 3}
				cfg := &config.Config{Org: "org", SourceBranches: []string{"dependabot/b", "dependabot/a"}, Rebase: true, Confirm: confirm, Workers: 2}
				m := New(mock, cfg, nil)
				result, err := m.Run(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				if confirm {
					if len(mock.PostRebaseCalls) != 0 || result.Summary.WouldRebase != 1 {
						t.Fatalf("scan mutated or missed action: %+v", result)
					}
					result, err = m.RunWithActions(context.Background(), result)
					if err != nil {
						t.Fatal(err)
					}
				}
				if len(mock.PostRebaseCalls) != 1 || len(mock.MergeCalls) != 0 {
					t.Fatalf("unexpected actions: %+v", mock)
				}
				if result.Summary.AlreadyUpToDate != 1 || result.Summary.ReadyToMerge != 0 || result.Summary.RebasedSuccess != 1 {
					t.Fatalf("bad summary: %+v", result.Summary)
				}
				fresh := result.Repositories[0].PullRequests[0]
				if fresh.Action != output.ActionNoRebaseNeeded || !strings.Contains(fresh.Reason, describeChecks(checks)) {
					t.Fatalf("incorrect freshness: %+v", fresh)
				}
				var buf bytes.Buffer
				if err := output.NewWriter(&buf, true, true).WriteResult(result); err != nil {
					t.Fatal(err)
				}
				var decoded output.RunResult
				if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
					t.Fatal(err)
				}
				if decoded.Summary.AlreadyUpToDate != 1 || decoded.Repositories[0].PullRequests[1].Action != output.ActionRebased {
					t.Fatalf("bad JSON: %s", buf.String())
				}
				buf.Reset()
				output.NewConsole(&buf, true, false, false).PrintSummary(result.Summary)
				if strings.Contains(buf.String(), "ready to merge") || !strings.Contains(buf.String(), "1 already up to date") || !strings.Contains(buf.String(), "rebase requests submitted") {
					t.Fatal(buf.String())
				}
			})
		}
	}
}

func TestReportIndependentStatus(t *testing.T) {
	for _, checks := range []*gh.CheckStatus{{AllPassing: false}, {Pending: true}, {AllPassing: true}, {NoChecks: true}} {
		mock := gh.NewMockClient()
		mock.CheckStatuses["org/repo/sha"] = checks
		mock.BranchStatuses["org/repo/"+string(rune(1))] = &gh.BranchStatus{BehindBy: 6}
		m := New(mock, &config.Config{Org: "org"}, nil)
		pr := m.buildReportPR(context.Background(), "repo", gh.PullRequest{Number: 1, HeadSHA: "sha"}, true, "standard")
		if pr.Checks != describeChecks(checks) || pr.BranchStatus != "needs-rebase" || pr.BehindBy != 6 {
			t.Fatalf("lost status: %+v", pr)
		}
		result := &output.ReportResult{Groups: []output.ReportGroup{{SourceBranch: "deps", Count: 1, PullRequests: []output.ReportPullRequest{pr}}}}
		var buf bytes.Buffer
		if err := output.NewWriter(&buf, false, true).WriteReportResult(result, "standard"); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(buf.String(), "6 commits behind") || !strings.Contains(buf.String(), pr.Checks) {
			t.Fatal(buf.String())
		}
		buf.Reset()
		if err := output.NewWriter(&buf, true, true).WriteReportResult(result, "standard"); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(buf.String(), `"behindBy": 6`) || !strings.Contains(buf.String(), `"branchStatus": "needs-rebase"`) {
			t.Fatal(buf.String())
		}
	}
}

func TestPatternSelectionUsesFlagOrderAfterAuthorFilter(t *testing.T) {
	mock := gh.NewMockClient()
	mock.Repositories = []gh.Repository{{Name: "repo", FullName: "org/repo", DefaultBranch: "main"}}
	mock.PullRequests["org/repo"] = []gh.PullRequest{
		{Number: 1, HeadBranch: "a", BaseBranch: "main", Author: "other"},
		{Number: 2, HeadBranch: "b", BaseBranch: "main", Author: "bot"},
		{Number: 3, HeadBranch: "c", BaseBranch: "main", Author: "bot"},
	}
	for _, confirm := range []bool{false, true} {
		m := New(mock, &config.Config{Org: "org", SourceBranches: []string{"a", "c", "b"}, Author: "bot", Merge: true, Confirm: confirm}, nil)
		result, err := m.Run(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		prs := result.Repositories[0].PullRequests
		if len(prs) != 2 || prs[0].Action != output.ActionSkipOtherPattern || prs[1].Action == output.ActionSkipOtherPattern {
			t.Fatalf("incorrect selection: %+v", prs)
		}
	}
}

func TestConfirmedRebaseDisplaysNonActionRepositories(t *testing.T) {
	for _, noProgress := range []bool{false, true} {
		t.Run(fmt.Sprintf("noProgress=%t", noProgress), func(t *testing.T) {
			mock := gh.NewMockClient()
			mock.Repositories = []gh.Repository{
				{Name: "fresh", FullName: "org/fresh", DefaultBranch: "main"},
				{Name: "behind", FullName: "org/behind", DefaultBranch: "main"},
			}
			for _, repo := range mock.Repositories {
				mock.PullRequests[repo.FullName] = []gh.PullRequest{{Number: 1, HeadBranch: "dependabot/a", BaseBranch: "main", HeadSHA: "sha"}}
			}
			mock.CheckStatuses["org/fresh/sha"] = &gh.CheckStatus{AllPassing: false}
			mock.BranchStatuses["org/behind/"+string(rune(1))] = &gh.BranchStatus{BehindBy: 2}
			var buf bytes.Buffer
			m := New(mock, &config.Config{Org: "org", SourceBranches: []string{"dependabot/"}, Rebase: true, Confirm: true, NoProgress: noProgress}, output.NewConsole(&buf, true, false, false))
			result, err := m.Run(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			buf.Reset()
			if _, err = m.RunWithActions(context.Background(), result); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(buf.String(), "no rebase needed") || !strings.Contains(buf.String(), "checks failing") {
				t.Fatal(buf.String())
			}
		})
	}
}
