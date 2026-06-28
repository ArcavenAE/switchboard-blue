// Package main — access_goroutine_join_test.go
//
// TestRunAccessWithConnectorNoGoroutineLeak (AC-008 / BC-2.04.007 PC-2 postcon-6)
//
// Verifies the goroutine-join contract on the PRODUCTION runAccessWithConnector:
// all goroutines it spawns MUST have exited before it returns to its caller.
//
// Finding I-1 (architect-ruled "join-required"):
//
//	startSweepTicker and startFramesDroppedTicker each spawn a goroutine that
//	exits on ctx.Done(), but neither is added to the wg that wg.Wait() joins on
//	shutdown. wg.Wait() can therefore return — and runAccessWithConnector can
//	return — while those goroutines are still executing, violating BC-2.04.007
//	v1.3 PC-2 postcondition 6, ARCH-01 v1.7 §Goroutine WaitGroup Contract, and
//	ARCH-08 v2.2 obligations 3 & 6.
//
// Discriminating strategy — channel handshake using blockingRelayConnector:
//
//	blockingRelayConnector.RelayDropped() is called by the frames-dropped ticker
//	goroutine on each tick. The test injects a 1ms tick interval (by setting the
//	package-level framesDroppedInterval variable before calling
//	runAccessWithConnector — this requires framesDroppedInterval to be a var, not
//	a const, which is a required testability change in the fix).
//
//	Once the ticker fires and the goroutine parks inside RelayDropped(), the test:
//	  1. Cancels ctx (PC-2 clean-shutdown, identical to TestRunAccessWithConnectorPC2).
//	  2. Asserts `done` does NOT close within 150ms while the goroutine is parked:
//	       select {
//	       case <-done:   → t.Fatal (function returned without joining ticker) → RED
//	       case <-time.After(150ms): → expected on FIXED code (blocked in wg.Wait)
//	       }
//	  3. Closes `release`, unblocking RelayDropped() → goroutine exits →
//	     wg.Wait() can complete → function returns → `done` closes → PASS.
//
//	BUGGY code (no wg tracking): wg.Wait() returns immediately, function returns,
//	`done` closes while goroutine is still in RelayDropped() → first select fires
//	→ t.Fatal → RED.
//
//	FIXED code (tickers in wg): wg.Wait() blocks on the ticker goroutine. `done`
//	stays open for 150ms → PASS (first select), then closes after release → PASS.
//
//	This is deterministic, has no silent-pass path, and goes red→green with ONLY
//	the production changes (make framesDroppedInterval a var + add wg tracking).
//
// Sweep ticker note:
//
//	startSweepTicker's goroutine is covered by the same wg.Wait(). It is not
//	directly exercised here because an.Sweep() on a concrete *session.AccessNode
//	has no injectable blocking seam. The fix must add wg tracking to both tickers;
//	this test exercises the frames-dropped ticker as the controllable lever.
package main

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/halfchannel"
)

// blockingRelayConnector is a connectorIface test double whose RelayDropped()
// participates in a deterministic channel handshake with the test goroutine.
//
// First call: closes `entered` (goroutine is parked here), blocks on `release`.
// Subsequent calls: return immediately (avoids deadlock on fixed code path).
//
// All other connectorIface methods delegate to the embedded fakeConnector.
type blockingRelayConnector struct {
	fakeConnector
	entered chan struct{} // closed by first RelayDropped() call
	release chan struct{} // closed by test to unblock RelayDropped()
	once    sync.Once     // ensures first-call logic fires exactly once
}

// RelayDropped implements connectorIface. First call parks inside this method
// until the test releases it; subsequent calls return immediately.
func (b *blockingRelayConnector) RelayDropped() uint64 {
	b.once.Do(func() {
		close(b.entered) // signal: ticker goroutine is now parked here
		<-b.release      // block until test closes release
	})
	return b.relayDropped //nolint:staticcheck // embedded-field selector intentional
}

// TestRunAccessWithConnectorNoGoroutineLeak — AC-008 / BC-2.04.007 PC-2 postcon-6
//
// Drives the PRODUCTION runAccessWithConnector with a blockingRelayConnector
// that parks the frames-dropped ticker goroutine inside RelayDropped(). The
// channel handshake then discriminates buggy from fixed behaviour.
//
// Pre-condition: framesDroppedInterval must be a var in access.go so the test
// can inject a 1ms tick interval. As long as it remains a const, this test fails
// at compile time — a valid (if noisy) Red Gate that enforces the required
// testability refactoring as part of the fix.
func TestRunAccessWithConnectorNoGoroutineLeak(t *testing.T) {
	// NOT t.Parallel(): modifies framesDroppedInterval (package-level var).

	// Inject a 1ms tick interval so the frames-dropped ticker fires immediately.
	// Requires framesDroppedInterval to be a var in access.go (part of the fix).
	// On current code (const), this line is a compile error — Red Gate enforced
	// at build time.
	origInterval := framesDroppedInterval
	framesDroppedInterval = time.Millisecond
	t.Cleanup(func() { framesDroppedInterval = origInterval })

	an, router := newMinimalAccessComponents(t)

	entered := make(chan struct{})
	release := make(chan struct{})

	brc := &blockingRelayConnector{
		fakeConnector: fakeConnector{
			errCh:    make(chan error),
			framesCh: make(chan halfchannel.ChannelFrame),
		},
		entered: entered,
		release: release,
	}
	t.Cleanup(func() { _ = brc.Close() })

	var stderr bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// Launch the PRODUCTION function. `done` is closed when it returns.
	done := make(chan struct{})
	go func() {
		_ = runAccessWithConnector(ctx, &stderr, brc, an, router)
		close(done)
	}()

	// Wait for the frames-dropped ticker to fire and park inside RelayDropped().
	// With a 1ms interval this should happen within a few milliseconds.
	// A 500ms timeout guards against unexpected delays; on fixed code the goroutine
	// parks almost immediately.
	select {
	case <-entered:
		// goroutine is now parked inside RelayDropped() — proceed to discriminator
	case <-time.After(500 * time.Millisecond):
		t.Fatal("frames-dropped ticker goroutine did not call RelayDropped() within 500ms — " +
			"expected ticker to fire within ~1ms of starting with a 1ms interval " +
			"(Finding I-1; BC-2.04.007 PC-2 postcon-6)")
	}

	// Trigger PC-2 clean shutdown. Same mechanism as TestRunAccessWithConnectorPC2:
	// cancel the context passed to runAccessWithConnector.
	cancel()

	// THE DISCRIMINATOR:
	//
	// BUGGY code (ticker NOT in wg): wg.Wait() returns after errDrain + bridge
	// exit, without waiting for the ticker goroutine. runAccessWithConnector
	// returns. `done` closes WHILE the goroutine is still parked in RelayDropped().
	// → first select hits <-done → t.Fatal → RED.
	//
	// FIXED code (ticker in wg): wg.Wait() blocks until the ticker goroutine
	// exits. The goroutine is parked in RelayDropped() and cannot exit until
	// `release` is closed. `done` stays open for the full 150ms window.
	// → first select times out → continue → PASS (first assertion).
	select {
	case <-done:
		close(release) // unblock goroutine to avoid test-process leak
		t.Fatal("runAccessWithConnector returned while the frames-dropped ticker goroutine " +
			"was still parked in RelayDropped() — wg.Wait() did not join it. " +
			"Fix: add wg.Add(1) before startSweepTicker AND startFramesDroppedTicker " +
			"in runAccessWithConnector, with defer wg.Done() inside each goroutine. " +
			"(Finding I-1; BC-2.04.007 PC-2 postcon-6; ARCH-01 v1.7 §Goroutine " +
			"WaitGroup Contract; ARCH-08 v2.2 obligations 3 & 6)")
	case <-time.After(150 * time.Millisecond):
		// expected on FIXED code: function blocked in wg.Wait() for the ticker
	}

	// Release the parked goroutine. On fixed code: RelayDropped() returns,
	// goroutine exits, wg.Wait() returns, function returns, `done` closes.
	close(release)

	select {
	case <-done:
		// success: function returned only after the ticker goroutine was joined
	case <-time.After(2 * time.Second):
		t.Fatal("runAccessWithConnector did not return within 2s after releasing the ticker goroutine")
	}
}
