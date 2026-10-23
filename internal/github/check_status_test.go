package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-github/v60/github"
)

func TestGetCheckStatusConclusions(t *testing.T) {
	tests := []struct {
		name         string
		conclusions  []string
		status       string
		commitState  string
		wantPassing  bool
		wantPending  bool
		wantNoChecks bool
	}{
		{name: "success", conclusions: []string{"success"}, wantPassing: true},
		{name: "skipped", conclusions: []string{"skipped"}, wantPassing: true},
		{name: "neutral", conclusions: []string{"neutral"}, wantPassing: true},
		{name: "mixed passing", conclusions: []string{"success", "skipped", "neutral"}, wantPassing: true},
		{name: "failure after skipped", conclusions: []string{"skipped", "failure"}},
		{name: "cancelled", conclusions: []string{"cancelled"}},
		{name: "timed out", conclusions: []string{"timed_out"}},
		{name: "action required", conclusions: []string{"action_required"}},
		{name: "unknown conclusion", conclusions: []string{"unknown"}},
		{name: "empty conclusion", conclusions: []string{""}},
		{name: "queued after neutral", conclusions: []string{"neutral", ""}, status: "queued", wantPending: true},
		{name: "in progress after skipped", conclusions: []string{"skipped", ""}, status: "in_progress", wantPending: true},
		{name: "successful commit status", conclusions: []string{"skipped"}, commitState: "success", wantPassing: true},
		{name: "failed commit status", conclusions: []string{"neutral"}, commitState: "failure"},
		{name: "pending commit status", conclusions: []string{"skipped"}, commitState: "pending", wantPending: true},
		{name: "skipped commit status is not passing", commitState: "skipped"},
		{name: "neutral commit status is not passing", commitState: "neutral"},
		{name: "no checks", wantNoChecks: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/repos/owner/repo/commits/sha/check-runs", func(w http.ResponseWriter, r *http.Request) {
				runs := make([]map[string]string, 0, len(tt.conclusions))
				for i, conclusion := range tt.conclusions {
					status := "completed"
					if i == len(tt.conclusions)-1 && tt.status != "" {
						status = tt.status
					}
					runs = append(runs, map[string]string{"name": "check", "status": status, "conclusion": conclusion})
				}
				json.NewEncoder(w).Encode(map[string]any{"total_count": len(runs), "check_runs": runs})
			})
			mux.HandleFunc("/repos/owner/repo/commits/sha/status", func(w http.ResponseWriter, r *http.Request) {
				statuses := []map[string]string{}
				if tt.commitState != "" {
					statuses = append(statuses, map[string]string{"context": "legacy", "state": tt.commitState})
				}
				json.NewEncoder(w).Encode(map[string]any{"statuses": statuses})
			})
			server := httptest.NewServer(mux)
			defer server.Close()
			api := github.NewClient(server.Client())
			baseURL, err := url.Parse(server.URL + "/")
			if err != nil {
				t.Fatal(err)
			}
			api.BaseURL = baseURL
			client := &RealClient{client: api}
			got, err := client.GetCheckStatus(context.Background(), "owner", "repo", "sha")
			if err != nil {
				t.Fatal(err)
			}
			if got.AllPassing != tt.wantPassing || got.Pending != tt.wantPending || got.NoChecks != tt.wantNoChecks {
				t.Fatalf("GetCheckStatus() = %+v; want AllPassing=%v Pending=%v NoChecks=%v", got, tt.wantPassing, tt.wantPending, tt.wantNoChecks)
			}
		})
	}
}
