package drain

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// TestDrain_DrainTimeoutFromConfig verifies BC-2.09.003 PC-7 zero-value
// semantics: a zero DrainTimeout resolves to DefaultTimeout (10s per
// ARCH-06). Also verifies the story's AC-005 wiring point.
func TestDrain_DrainTimeoutFromConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   time.Duration
		want time.Duration
	}{
		{"zero -> default 10s", 0, DefaultTimeout},
		{"positive is used verbatim", 15 * time.Second, 15 * time.Second},
		{"negative falls back to default (defensive)", -1 * time.Second, DefaultTimeout},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := New(tc.in)
			if got := d.Timeout(); got != tc.want {
				t.Fatalf("Timeout() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestDrain_NoObservers_CompletesImmediately verifies that a drain with no
// registered observers completes without blocking.
func TestDrain_NoObservers_CompletesImmediately(t *testing.T) {
	t.Parallel()
	d := New(1 * time.Second)
	d.Signal(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if err := d.Wait(ctx); err != nil {
		t.Fatalf("Wait: %v (want nil)", err)
	}
}

// TestDrain_ObserversACK_CompletesCleanly verifies BC-2.09.002 postcondition
// path where nodes migrate within the drain window.
func TestDrain_ObserversACK_CompletesCleanly(t *testing.T) {
	t.Parallel()
	d := New(1 * time.Second)

	var seen atomic.Int32
	d.RegisterObserver(func(ctx context.Context) {
		seen.Add(1)
	})
	d.RegisterObserver(func(ctx context.Context) {
		seen.Add(1)
	})

	d.Signal(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	if err := d.Wait(ctx); err != nil {
		t.Fatalf("Wait: %v (want nil)", err)
	}
	if got := seen.Load(); got != 2 {
		t.Fatalf("observers called %d times, want 2", got)
	}
}

// TestDrain_ObserverTimeout verifies BC-2.09.002 EC-003 — when the drain
// window elapses before an observer ACKs, Wait returns ErrTimeout so the
// caller proceeds with disconnect regardless.
func TestDrain_ObserverTimeout(t *testing.T) {
	t.Parallel()
	d := New(30 * time.Millisecond)

	release := make(chan struct{})
	d.RegisterObserver(func(ctx context.Context) {
		// Wait for either the drain ctx to cancel (proper unwind) or the
		// test to release, whichever comes first. This models a slow node
		// that cannot ACK within the window.
		select {
		case <-ctx.Done():
		case <-release:
		}
	})

	d.Signal(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	err := d.Wait(ctx)
	close(release)
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("Wait err = %v, want ErrTimeout", err)
	}
}

// TestDrain_SignalIsIdempotent verifies that a second Signal is a no-op —
// observers registered before the first Signal fire once, not twice.
func TestDrain_SignalIsIdempotent(t *testing.T) {
	t.Parallel()
	d := New(1 * time.Second)

	var calls atomic.Int32
	d.RegisterObserver(func(ctx context.Context) {
		calls.Add(1)
	})

	d.Signal(context.Background())
	d.Signal(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	if err := d.Wait(ctx); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("observer called %d times, want 1 (Signal must be idempotent)", got)
	}
}

// TestDrain_ObserverAfterSignal_Ignored verifies Drain single-use semantics:
// observers registered after Signal do NOT participate in that drain.
func TestDrain_ObserverAfterSignal_Ignored(t *testing.T) {
	t.Parallel()
	d := New(1 * time.Second)

	var calls atomic.Int32
	d.Signal(context.Background())
	d.RegisterObserver(func(ctx context.Context) {
		calls.Add(1)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if err := d.Wait(ctx); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("post-signal observer called %d times, want 0", got)
	}
}

// TestDrain_WaitRespectsCallerContext verifies Wait unblocks on caller-ctx
// cancel even before Signal has been called.
func TestDrain_WaitRespectsCallerContext(t *testing.T) {
	t.Parallel()
	d := New(1 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := d.Wait(ctx)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatalf("Wait returned nil; want ctx cancel error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Wait err = %v, want DeadlineExceeded", err)
	}
	if elapsed > 200*time.Millisecond {
		t.Fatalf("Wait blocked %v, want ~20ms", elapsed)
	}
}

// TestDrain_NilObserver_Ignored verifies defensive handling of a nil
// registration — must not panic on Signal.
func TestDrain_NilObserver_Ignored(t *testing.T) {
	t.Parallel()
	d := New(1 * time.Second)
	d.RegisterObserver(nil)
	d.Signal(context.Background())
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if err := d.Wait(ctx); err != nil {
		t.Fatalf("Wait: %v", err)
	}
}
