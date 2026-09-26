package github

import (
	"context"
	"errors"
	"testing"
	"time"

	api "github.com/google/go-github/v60/github"
)

func TestRateLimitRetryBound(t *testing.T) {
	c := &rateLimitedClient{}
	zero := time.Duration(0)
	calls := 0
	rejection := &api.AbuseRateLimitError{RetryAfter: &zero}
	_, err := retry(context.Background(), c, func() (int, error) { calls++; return 0, rejection })
	if err != rejection || calls != 4 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

func TestRateLimitSharedCooldownAndCancellation(t *testing.T) {
	c := &rateLimitedClient{}
	ctx, cancel := context.WithCancel(context.Background())
	_, err := retry(ctx, c, func() (int, error) {
		cancel()
		return 0, &api.RateLimitError{Rate: api.Rate{Reset: api.Timestamp{Time: time.Now().Add(time.Hour)}}}
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	second, stop := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer stop()
	calls := 0
	_, err = retry(second, c, func() (int, error) { calls++; return 0, nil })
	if calls != 0 || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

func TestRetryDoesNotReplayOtherErrors(t *testing.T) {
	for _, failure := range []error{errors.New("connection lost"), &api.ErrorResponse{Message: "forbidden"}} {
		calls := 0
		_, err := retry(context.Background(), &rateLimitedClient{}, func() (int, error) { calls++; return 0, failure })
		if calls != 1 || err != failure {
			t.Fatalf("calls=%d err=%v", calls, err)
		}
	}
}

func TestRateLimitRetrySucceeds(t *testing.T) {
	zero := time.Duration(0)
	calls := 0
	value, err := retry(context.Background(), &rateLimitedClient{}, func() (int, error) {
		calls++
		if calls == 1 {
			return 0, &api.AbuseRateLimitError{RetryAfter: &zero}
		}
		return 42, nil
	})
	if err != nil || value != 42 || calls != 2 {
		t.Fatalf("value=%d calls=%d err=%v", value, calls, err)
	}
}
