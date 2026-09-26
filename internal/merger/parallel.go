package merger

import (
	"context"
	"sync"
)

// parallel runs independent repository jobs. Only the caller invokes collect,
// serializing console output and summary updates. Outstanding jobs reserve limit
// slots so even mutations cannot overshoot the successful-repository limit.
func parallel[T any](ctx context.Context, count, workers, limit int, work func(int) T, collect func(int, T, bool) bool) error {
	if workers < 1 {
		workers = 1
	}
	if workers > count {
		workers = count
	}
	type completion struct {
		index int
		value T
	}
	jobs := make(chan int)
	done := make(chan completion, workers)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				done <- completion{i, work(i)}
			}
		}()
	}
	defer func() { close(jobs); wg.Wait() }()
	next, active, successful := 0, 0, 0
	for next < count || active > 0 {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if limit > 0 && successful >= limit && active == 0 {
			var zero T
			for ; next < count; next++ {
				collect(next, zero, true)
			}
			break
		}
		var send chan int
		if next < count && active < workers && (limit <= 0 || successful+active < limit) {
			send = jobs
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case send <- next:
			next++
			active++
		case c := <-done:
			active--
			if collect(c.index, c.value, false) {
				successful++
			}
		}
	}
	return nil
}
