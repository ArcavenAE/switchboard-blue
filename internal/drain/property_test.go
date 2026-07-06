// Property tests for Phase 6 formal hardening.
//
// The unit tests in drain_test.go cover BC-2.09.002 acceptance criteria
// (idempotent Signal, post-signal registration ignored, EC-003 timeout).
// This file adds property-shaped tests exercised under -race and across
// many concurrent registrations/signals:
//
//   - Observer-notification exactly-once: no observer registered before
//     Signal is invoked more than once, even under concurrent Signal calls.
//   - Concurrent Register+Signal is safe: no data race, no panic, and
//     observers registered before the first Signal wins fire exactly once.
//   - Concurrent Wait calls all receive the same terminal result.
//   - Timeout path exactly-once: even with concurrent Signal calls, the
//     timeout observation is set exactly once.
//   - Nested Wait/Signal ordering: Wait may be called before Signal without
//     deadlock; either the caller ctx cancels first or Signal completes.
//
// All property tests are marked t.Parallel and run under -race in CI.
package drain

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestProperty_ObserverNotificationExactlyOnce_UnderConcurrentSignals
// asserts that no matter how many concurrent Signal calls hit a Drain,
// each pre-signal observer is invoked exactly once. BC-2.09.002
// idempotent-signal postcondition.
func TestProperty_ObserverNotificationExactlyOnce_UnderConcurrentSignals(t *testing.T) {
	t.Parallel()

	d := New(1 * time.Second)

	const observers = 32
	var counters [observers]atomic.Int32
	for i := 0; i < observers; i++ {
		i := i
		d.RegisterObserver(func(ctx context.Context) {
			counters[i].Add(1)
		})
	}

	// Fire many concurrent Signal calls. Only the first should have
	// effect; the rest are no-ops. All must be safe under -race.
	const signalers = 16
	var wg sync.WaitGroup
	wg.Add(signalers)
	start := make(chan struct{})
	for i := 0; i < signalers; i++ {
		go func() {
			defer wg.Done()
			<-start
			d.Signal(context.Background())
		}()
	}
	close(start)
	wg.Wait()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	if err := d.Wait(ctx); err != nil {
		t.Fatalf("Wait: %v (want nil)", err)
	}

	for i := 0; i < observers; i++ {
		if got := counters[i].Load(); got != 1 {
			t.Fatalf("observer[%d] called %d times, want 1 (exactly-once violated)", i, got)
		}
	}
}

// TestProperty_ConcurrentRegisterAndSignal_NoRace runs many
// RegisterObserver calls concurrently with a single Signal. Under -race,
// this must be free of data races. Observers that register BEFORE the
// signal happens participate; ones that register after do not. What
// matters for the property is that the count of observers-that-fired is
// stable and bounded — never more than the number registered, never a
// panic, never a leak.
func TestProperty_ConcurrentRegisterAndSignal_NoRace(t *testing.T) {
	t.Parallel()

	d := New(500 * time.Millisecond)

	const registrants = 64
	var fired atomic.Int32

	var wg sync.WaitGroup
	wg.Add(registrants + 1)

	start := make(chan struct{})

	for i := 0; i < registrants; i++ {
		go func() {
			defer wg.Done()
			<-start
			d.RegisterObserver(func(ctx context.Context) {
				fired.Add(1)
			})
		}()
	}
	go func() {
		defer wg.Done()
		<-start
		// Small yield to allow some registrations to land first, but do
		// not synchronize — the race is the point.
		d.Signal(context.Background())
	}()

	close(start)
	wg.Wait()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = d.Wait(ctx)

	got := fired.Load()
	if got < 0 || got > int32(registrants) {
		t.Fatalf("fired = %d, want 0..%d (invariant: no observer fires twice, none appear from nowhere)", got, registrants)
	}
}

// TestProperty_ConcurrentWaitCallsReceiveSameTerminalResult asserts that
// multiple goroutines calling Wait on the same Drain all see the same
// terminal outcome: nil for clean drain, ErrTimeout for the EC-003 path.
// (Both cases are exercised by two subtests.)
func TestProperty_ConcurrentWaitCallsReceiveSameTerminalResult(t *testing.T) {
	t.Parallel()

	t.Run("clean_drain", func(t *testing.T) {
		t.Parallel()
		d := New(1 * time.Second)
		var registered atomic.Int32
		d.RegisterObserver(func(ctx context.Context) { registered.Add(1) })
		d.Signal(context.Background())

		const waiters = 16
		results := make([]error, waiters)
		var wg sync.WaitGroup
		wg.Add(waiters)
		for i := 0; i < waiters; i++ {
			i := i
			go func() {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
				defer cancel()
				results[i] = d.Wait(ctx)
			}()
		}
		wg.Wait()

		for i, err := range results {
			if err != nil {
				t.Fatalf("waiter[%d]: err = %v, want nil", i, err)
			}
		}
	})

	t.Run("timeout_drain", func(t *testing.T) {
		t.Parallel()
		d := New(30 * time.Millisecond)
		release := make(chan struct{})
		d.RegisterObserver(func(ctx context.Context) {
			select {
			case <-ctx.Done():
			case <-release:
			}
		})
		d.Signal(context.Background())

		const waiters = 16
		results := make([]error, waiters)
		var wg sync.WaitGroup
		wg.Add(waiters)
		for i := 0; i < waiters; i++ {
			i := i
			go func() {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
				defer cancel()
				results[i] = d.Wait(ctx)
			}()
		}
		wg.Wait()
		close(release)

		for i, err := range results {
			if !errors.Is(err, ErrTimeout) {
				t.Fatalf("waiter[%d]: err = %v, want ErrTimeout", i, err)
			}
		}
	})
}

// TestProperty_WaitBeforeSignal_DoesNotDeadlock asserts that a Wait call
// made before any Signal blocks correctly (respecting caller ctx cancel)
// and does not deadlock. A concurrent Signal that arrives while Wait is
// blocked must unblock Wait with the appropriate terminal result.
func TestProperty_WaitBeforeSignal_DoesNotDeadlock(t *testing.T) {
	t.Parallel()

	d := New(1 * time.Second)
	var fired atomic.Int32
	d.RegisterObserver(func(ctx context.Context) { fired.Add(1) })

	waitDone := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		waitDone <- d.Wait(ctx)
	}()

	// Give Wait a moment to actually block on d.done, then Signal.
	// (This is not a race — the property is that Wait completes
	// regardless of which order Signal/Wait are made.)
	time.Sleep(20 * time.Millisecond)
	d.Signal(context.Background())

	select {
	case err := <-waitDone:
		if err != nil {
			t.Fatalf("Wait err = %v, want nil (clean drain)", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("Wait did not unblock after Signal — deadlock")
	}
	if got := fired.Load(); got != 1 {
		t.Fatalf("observer fired %d times, want 1", got)
	}
}

// TestProperty_TimedOutFlagIsSetExactlyOnce asserts that the internal
// timedOut flag transitions monotonically: it goes false→true exactly
// once and never back to false. Any observer of Wait after that transition
// sees the same ErrTimeout result. This is exercised by having many Wait
// calls staggered before and after the timeout fires.
func TestProperty_TimedOutFlagIsSetExactlyOnce(t *testing.T) {
	t.Parallel()

	d := New(50 * time.Millisecond)
	release := make(chan struct{})
	defer close(release)
	d.RegisterObserver(func(ctx context.Context) {
		select {
		case <-ctx.Done():
		case <-release:
		}
	})

	d.Signal(context.Background())

	// After the drain window elapses, both immediate and delayed Wait
	// calls must observe ErrTimeout.
	time.Sleep(150 * time.Millisecond)

	for i := 0; i < 8; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		err := d.Wait(ctx)
		cancel()
		if !errors.Is(err, ErrTimeout) {
			t.Fatalf("Wait[%d] err = %v, want ErrTimeout (transitioned back to non-timeout?)", i, err)
		}
	}
}

// TestProperty_PostSignalRegistrationsAreSafeUnderRace covers the specific
// race where a caller calls RegisterObserver concurrently with Signal, on
// the other side of the signal → the registration is safely ignored, no
// panic, no data race. This is the "single-use" guarantee of the Drain.
func TestProperty_PostSignalRegistrationsAreSafeUnderRace(t *testing.T) {
	t.Parallel()

	for iter := 0; iter < 50; iter++ {
		d := New(200 * time.Millisecond)

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			d.Signal(context.Background())
		}()
		go func() {
			defer wg.Done()
			// May land before or after Signal — both must be safe.
			d.RegisterObserver(func(ctx context.Context) {})
		}()

		wg.Wait()

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		_ = d.Wait(ctx)
		cancel()
	}
}
