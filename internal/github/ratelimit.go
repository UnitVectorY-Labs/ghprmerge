package github

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	api "github.com/google/go-github/v60/github"
)

// rateLimitedClient shares GitHub cooldowns across all repository workers.
// Only explicit rate-limit rejections are retried; ambiguous mutation failures
// (including network errors) are returned without replaying the action.
type rateLimitedClient struct {
	Client
	mu    sync.Mutex
	until time.Time
}

// NewRateLimitedClient adds bounded retries and a shared cooldown. Wrap the
// client once, before starting workers.
func NewRateLimitedClient(client Client) Client {
	c := &rateLimitedClient{Client: client}
	if real, ok := client.(*RealClient); ok && real.httpClient != nil {
		real.httpClient.Transport = &cooldownTransport{base: real.httpClient.Transport, gate: c}
	}
	return c
}

// Gate each HTTP request as well as each logical operation: pagination and
// multi-request status checks must observe cooldowns discovered by other workers.
type cooldownTransport struct {
	base http.RoundTripper
	gate *rateLimitedClient
}

func (t *cooldownTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := t.gate.wait(req.Context()); err != nil {
		return nil, err
	}
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}

func (c *rateLimitedClient) wait(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		c.mu.Lock()
		delay := time.Until(c.until)
		c.mu.Unlock()
		if delay <= 0 {
			return nil
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func retry[T any](ctx context.Context, c *rateLimitedClient, call func() (T, error)) (T, error) {
	for attempt := 0; ; attempt++ {
		if err := c.wait(ctx); err != nil {
			var zero T
			return zero, err
		}
		value, err := call()
		var primary *api.RateLimitError
		var secondary *api.AbuseRateLimitError
		until := time.Now().Add(time.Minute * time.Duration(1<<attempt))
		switch {
		case errors.As(err, &primary):
			if primary.Rate.Reset.Time.After(time.Now()) {
				until = primary.Rate.Reset.Time.Add(time.Second)
			}
		case errors.As(err, &secondary):
			if secondary.RetryAfter != nil {
				until = time.Now().Add(*secondary.RetryAfter)
			}
		default:
			return value, err
		}
		c.mu.Lock()
		if until.After(c.until) {
			c.until = until
		}
		c.mu.Unlock()
		if attempt >= 3 {
			return value, err
		}
	}
}

func (c *rateLimitedClient) ListRepositories(ctx context.Context, org string) ([]Repository, error) {
	return retry(ctx, c, func() ([]Repository, error) { return c.Client.ListRepositories(ctx, org) })
}

func (c *rateLimitedClient) ListPullRequests(ctx context.Context, owner, repo, defaultBranch string) ([]PullRequest, error) {
	return retry(ctx, c, func() ([]PullRequest, error) { return c.Client.ListPullRequests(ctx, owner, repo, defaultBranch) })
}

func (c *rateLimitedClient) GetPullRequest(ctx context.Context, owner, repo string, number int) (*PullRequest, error) {
	return retry(ctx, c, func() (*PullRequest, error) { return c.Client.GetPullRequest(ctx, owner, repo, number) })
}

func (c *rateLimitedClient) GetCheckStatus(ctx context.Context, owner, repo, ref string) (*CheckStatus, error) {
	return retry(ctx, c, func() (*CheckStatus, error) { return c.Client.GetCheckStatus(ctx, owner, repo, ref) })
}

func (c *rateLimitedClient) GetBranchStatus(ctx context.Context, owner, repo string, prNumber int) (*BranchStatus, error) {
	return retry(ctx, c, func() (*BranchStatus, error) { return c.Client.GetBranchStatus(ctx, owner, repo, prNumber) })
}

func (c *rateLimitedClient) UpdateBranch(ctx context.Context, owner, repo string, prNumber int) error {
	_, err := retry(ctx, c, func() (struct{}, error) { return struct{}{}, c.Client.UpdateBranch(ctx, owner, repo, prNumber) })
	return err
}

func (c *rateLimitedClient) PostRebaseComment(ctx context.Context, owner, repo string, prNumber int) error {
	_, err := retry(ctx, c, func() (struct{}, error) { return struct{}{}, c.Client.PostRebaseComment(ctx, owner, repo, prNumber) })
	return err
}

func (c *rateLimitedClient) MergePullRequest(ctx context.Context, owner, repo string, prNumber int) error {
	_, err := retry(ctx, c, func() (struct{}, error) { return struct{}{}, c.Client.MergePullRequest(ctx, owner, repo, prNumber) })
	return err
}

func (c *rateLimitedClient) ClosePullRequest(ctx context.Context, owner, repo string, prNumber int) error {
	_, err := retry(ctx, c, func() (struct{}, error) { return struct{}{}, c.Client.ClosePullRequest(ctx, owner, repo, prNumber) })
	return err
}

func (c *rateLimitedClient) DeleteBranch(ctx context.Context, owner, repo, branch string) error {
	_, err := retry(ctx, c, func() (struct{}, error) { return struct{}{}, c.Client.DeleteBranch(ctx, owner, repo, branch) })
	return err
}
