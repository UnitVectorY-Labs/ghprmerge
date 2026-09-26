package merger

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestParallelBoundAndLimit(t *testing.T) {
	var active, peak, calls atomic.Int32
	collected := 0
	err := parallel(context.Background(), 12, 3, 5, func(i int) bool {
		n := active.Add(1)
		for old := peak.Load(); n > old && !peak.CompareAndSwap(old, n); old = peak.Load() {
		}
		calls.Add(1)
		time.Sleep(10 * time.Millisecond)
		active.Add(-1)
		return i != 0 // Failed repositories release their reserved limit slot.
	}, func(i int, ok, limited bool) bool { collected++; return ok })
	if err != nil || calls.Load() != 6 || peak.Load() != 3 || collected != 12 {
		t.Fatalf("err=%v calls=%d peak=%d collected=%d", err, calls.Load(), peak.Load(), collected)
	}
}

func TestParallelCancellationJoinsWorkers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var active atomic.Int32
	err := parallel(ctx, 20, 4, 0, func(i int) bool {
		active.Add(1)
		defer active.Add(-1)
		cancel()
		return true
	}, func(int, bool, bool) bool { return true })
	if err != context.Canceled || active.Load() != 0 {
		t.Fatalf("err=%v active=%d", err, active.Load())
	}
}
