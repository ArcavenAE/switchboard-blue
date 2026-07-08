// Package upstreamdial owns the outbound TCP dial loop for PE-mode routers
// (S-7.04-FU-PE-CONNECTOR, BC-2.09.001 PC-2/PC-3).
//
// A Connector is constructed at runRouter startup with the initial upstream
// address list and a keepalive interval.  It dials each configured upstream
// router, bootstraps the session with an outerassembler.Assemble wire frame,
// and maintains connections using a keepalive ticker.  Address set updates
// arrive via a buffered chan []string (channel-passed snapshot per Q3 ruling).
//
// DAG position: 19 (effectful — network I/O).
// Allowed internal imports: {halfchannel, outerassembler}.
// Forbidden: drain, routing, testenv, and any package at positions 20–23.
package upstreamdial

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/arcavenae/switchboard/internal/halfchannel"
	"github.com/arcavenae/switchboard/internal/outerassembler"
)

// Backoff parameters for per-address reconnect (Q5 ruling, AC-002).
// Base=500ms, Cap=30s, Jitter ±25% per-attempt.
const (
	// BackoffBase is the initial reconnect delay after a failed dial.
	BackoffBase = 500 * time.Millisecond
	// BackoffCap is the maximum reconnect delay after repeated failures.
	BackoffCap = 30 * time.Second
	// BackoffJitterFraction is the ±fraction applied to each backoff interval.
	// Value 0.25 means ±25% uniform jitter as specified in Q5.
	BackoffJitterFraction = 0.25
)

// ConnMode is the operating mode of a Connector.
// ModeE means no upstreams are connected; ModePE means ≥1 upstream is connected.
//
// This type lives here (not in testenv) because testenv is a test-helper
// composition root at DAG position 23 and MUST NOT be imported by production
// code.  testenv maps from ConnMode to testenv.RouterMode in its wiring glue.
type ConnMode int32

const (
	// ModeE is edge mode: no upstream router connections.
	ModeE ConnMode = 0
	// ModePE is provider-edge mode: ≥1 upstream router connection established.
	ModePE ConnMode = 1
)

// Handle is the control surface runRouter and testenv hold on a live Connector.
// All methods are goroutine-safe.
type Handle interface {
	// ReloadAddrs reconciles the running upstream set to the new address list
	// using set-equal semantics (a reorder MUST NOT trigger redial or teardown).
	// Non-blocking: the snapshot is enqueued on the internal update channel.
	// Called from runRouter's SIGHUP select case.
	ReloadAddrs(addrs []string)

	// Mode returns ModeE when no upstreams are connected, ModePE when ≥1.
	// Safe to call from any goroutine — backed by atomic.Int32 per Q3.
	Mode() ConnMode

	// Stop cancels all dial goroutines and blocks until they exit.
	Stop()
}

// Connector owns the outbound dial loop, per-address reconnect backoff timers,
// keepalive ticker, and connected-count atomic that drives Mode().
//
// Construction: use New.  The Connector implements Handle.
type Connector struct {
	// w is the io.Writer for operator-visible log lines (EC-001 log contract).
	// Nil-safe — writes are skipped when nil.
	w io.Writer

	// addrsCh is a buffered channel (cap 1) over which runRouter sends address
	// list snapshots.  The reconcile goroutine reads from this channel.
	// Sends use the fully non-blocking fast-path/drain/resend pattern in
	// ReloadAddrs (F-P5-001 — see placement note Q3 v1.4 erratum; the old
	// drop-oldest pattern with a blocking inner receive deadlocks under a
	// reader-drain race).
	addrsCh chan []string

	// env carries the per-dial envelope fields (this router's identity).
	env outerassembler.Envelope

	// keepaliveInterval is the base cadence for keepalive health probes and
	// the first reconnect delay after a live connection drops (AC-003, Q7).
	keepaliveInterval time.Duration

	// connectedCount is an atomic counter: 0 = ModeE, ≥1 = ModePE.
	// Only the reconcile/reconnect goroutine(s) increment/decrement it.
	// Mode() reads it with atomic.Load from any goroutine (go.md rule 12).
	connectedCount atomic.Int32

	// stopCh is closed by Stop() to cancel all internal goroutines.
	stopCh chan struct{}

	// doneCh is closed when all internal goroutines have exited (Stop blocks on this).
	doneCh chan struct{}

	// stopOnce ensures Stop() is idempotent: only the first call closes stopCh.
	// Subsequent callers still wait on doneCh.
	stopOnce sync.Once

	// initialAddrs is the address list from New, consumed by Start.
	initialAddrs []string
}

// New constructs a Connector with the given parameters and returns a *Connector
// (also satisfying Handle).  The Connector does NOT start dialing until Start is
// called.  Callers must call Stop() when the router shuts down.
//
// Parameters:
//   - w: operator log writer (nil-safe; EC-001 "upstream router <addr> unreachable").
//   - env: outer-header envelope for session bootstrap frames (Q6).
//   - keepaliveInterval: keepalive ticker cadence; drives per-connection health
//     probing and reconnect scheduling (Q7, AC-003).
//   - initialAddrs: address list from upstreamRoutersFor(cfg) at startup.
func New(w io.Writer, env outerassembler.Envelope, keepaliveInterval time.Duration, initialAddrs []string) *Connector {
	addrsCopy := make([]string, len(initialAddrs))
	copy(addrsCopy, initialAddrs)
	return &Connector{
		w:                 w,
		env:               env,
		keepaliveInterval: keepaliveInterval,
		addrsCh:           make(chan []string, 1),
		stopCh:            make(chan struct{}),
		doneCh:            make(chan struct{}),
		initialAddrs:      addrsCopy,
	}
}

// Start launches the Connector's reconcile/reconnect goroutines with the
// initial address list provided to New.  Must be called exactly once after New.
func (c *Connector) Start() {
	go c.reconcileLoop(c.initialAddrs)
}

// ReloadAddrs enqueues a new address list snapshot for the reconciler.
// Set-equal semantics: same addresses in different order MUST NOT trigger
// teardown or redial (Q1, AC-001).  Non-blocking under all interleavings
// (F-P5-001): both the drain and the resend are non-blocking selects.
//
// Race that motivated the fix (F-P5-001): the old default branch did a
// BLOCKING <-c.addrsCh.  When the reconcile goroutine drained the slot
// between the failed fast-send and the blocking drain, the channel was
// empty and the drain blocked forever — wedging runRouter's select loop
// (ctx.Done() and all future SIGHUPs unreachable).
//
// The second select's default (drop our snap if another value snuck in
// between the drain and the resend) is unreachable with the single
// production caller (runRouter select loop), but is cheap insurance.
// State: a subsequent reload always wins eventually because the last
// successful send is the newest snapshot.
func (c *Connector) ReloadAddrs(addrs []string) {
	snap := make([]string, len(addrs))
	copy(snap, addrs)
	// Fast path: channel empty — send without contention.
	select {
	case c.addrsCh <- snap:
		return
	default:
	}
	// Slow path: channel full — drain (non-blocking) then resend (non-blocking).
	select {
	case <-c.addrsCh:
	default:
	}
	select {
	case c.addrsCh <- snap:
	default:
	}
}

// Mode returns ModeE when connected count is 0, ModePE when ≥1.
// Goroutine-safe (atomic load per Q3; go.md rule 12 permits scalar atomic
// reads without a mutex).
func (c *Connector) Mode() ConnMode {
	if c.connectedCount.Load() > 0 {
		return ModePE
	}
	return ModeE
}

// Stop is idempotent and safe for concurrent callers.  The first call closes
// stopCh, which cancels all internal goroutines; every caller (including
// concurrent ones) then blocks on doneCh until shutdown completes.
// Subsequent calls after the first are no-ops other than the doneCh wait,
// which returns immediately once the goroutines have exited.
func (c *Connector) Stop() {
	c.stopOnce.Do(func() {
		close(c.stopCh)
	})
	<-c.doneCh
}

// addrCancel holds a cancel function for a per-address dial goroutine.
type addrCancel struct {
	cancel func()
	done   chan struct{}
}

// reconcileLoop is the main goroutine that maintains the set of per-address
// dial goroutines. It starts with initialAddrs and processes reload snapshots.
func (c *Connector) reconcileLoop(initialAddrs []string) {
	defer close(c.doneCh)

	// running tracks active per-address goroutines: addr → addrCancel.
	running := make(map[string]*addrCancel)

	// Perform initial reconcile.
	c.reconcile(running, initialAddrs)

	for {
		select {
		case <-c.stopCh:
			// Cancel all running per-address goroutines and wait for them.
			for addr, ac := range running {
				ac.cancel()
				<-ac.done
				delete(running, addr)
			}
			return

		case newAddrs := <-c.addrsCh:
			c.reconcile(running, newAddrs)
		}
	}
}

// reconcile computes the set-diff between running goroutines and the new
// address list, starts goroutines for added addresses, and cancels goroutines
// for removed addresses (Q1 set-equal semantics, AC-001).
func (c *Connector) reconcile(running map[string]*addrCancel, newAddrs []string) {
	newSet := make(map[string]struct{}, len(newAddrs))
	for _, a := range newAddrs {
		newSet[a] = struct{}{}
	}

	// Remove addresses no longer in the set.
	for addr, ac := range running {
		if _, ok := newSet[addr]; !ok {
			ac.cancel()
			<-ac.done
			delete(running, addr)
		}
	}

	// Add addresses not yet running.
	for addr := range newSet {
		if _, ok := running[addr]; !ok {
			dialCtx, cancel := makeAddrContext(c.stopCh)
			ac := &addrCancel{
				cancel: cancel,
				done:   make(chan struct{}),
			}
			running[addr] = ac
			go c.dialLoop(dialCtx, addr, ac.done)
		}
	}
}

// makeAddrContext returns a context and cancel function for one per-address
// dial goroutine.  The context is cancelled when either cancel() is called
// or the connector-level stopCh is closed — ensuring net.DialContext returns
// promptly when Stop() fires (rather than blocking for the TCP timeout).
func makeAddrContext(stopCh <-chan struct{}) (ctx context.Context, cancel context.CancelFunc) {
	ctx, cancel = context.WithCancel(context.Background())
	// Bridge: if the connector stops, cancel the per-address context.
	go func() {
		select {
		case <-stopCh:
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, cancel
}

// dialLoop runs in a per-address goroutine.  It dials, bootstraps, maintains
// the connection with keepalive probes, and reconnects with exponential backoff
// on failure.  It exits when ctx is cancelled (Q3).
func (c *Connector) dialLoop(ctx context.Context, addr string, done chan<- struct{}) {
	defer close(done)

	backoff := operativeBase(c.keepaliveInterval) // first retry: keepaliveInterval floored at BackoffBase per Q7/F-P2-002
	keepaliveTick := time.NewTicker(c.keepaliveInterval)
	defer keepaliveTick.Stop()

	dialer := &net.Dialer{}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn, err := dialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			// Context cancelled — stop without logging EC-001.
			if ctx.Err() != nil {
				return
			}
			// EC-001: log "upstream router <addr> unreachable" (verbatim BC contract).
			c.logf("upstream router %s unreachable\n", addr)

			// Wait for backoff duration or stop signal.
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			backoff = nextBackoff(backoff)
			continue
		}

		// Connection established step 1: net.Dial succeeded.
		// Step 2: outerassembler.Assemble bootstrap frame (Q6, delivered with
		// placeholder frame type). FrameTypeData is a placeholder: the distinct
		// PE-CONNECT bootstrap frame type (Q6's frame.FrameTypePEConnect) is
		// deferred to S-BL.PE-RECEIVE-LOOP, the consumer that must distinguish
		// bootstrap frames from session data. Deferral is symmetric with the
		// zero-valued Envelope deferral documented in mgmt_wire.go.
		cf := halfchannel.ChannelFrame{
			FrameType: halfchannel.FrameTypeData,
		}
		var sackBitmap [outerassembler.SACKBitmapSize]byte
		wire, aErr := outerassembler.Assemble(cf, sackBitmap, c.env)
		if aErr != nil {
			_ = conn.Close()
			c.logf("upstream router %s unreachable\n", addr)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			backoff = nextBackoff(backoff)
			continue
		}

		// Step 3: Write bootstrap frame.
		n, wErr := conn.Write(wire)
		if wErr != nil || n != len(wire) {
			_ = conn.Close()
			c.logf("upstream router %s unreachable\n", addr)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			backoff = nextBackoff(backoff)
			continue
		}

		// All three steps succeeded: connection established.
		// Increment connected count — Mode() becomes ModePE when ≥1.
		c.connectedCount.Add(1)
		backoff = operativeBase(c.keepaliveInterval) // reset to operative base on success per Q5/Q7/F-P2-002

		// Maintain connection: send keepalive probes and detect dead connections.
		c.maintainConn(addr, conn, ctx.Done(), keepaliveTick.C)

		// Connection dropped — decrement count.
		c.connectedCount.Add(-1)
		_ = conn.Close()

		// Log EC-004 if all upstreams are now unreachable (count → 0).
		// Guard: skip emission when the drop was caused by context cancellation
		// (graceful Stop) — EC-004's trigger is upstream-LOSS, not self-initiated
		// teardown.  Symmetric with the EC-001 ctx.Err() early-return guard in
		// the dial-failure branch above (F-P4-001).
		if c.connectedCount.Load() == 0 && ctx.Err() == nil {
			c.logf("mode=E (no upstream_routers configured)\n")
		}

		// Loop to reconnect.
	}
}

// maintainConn keeps a connected upstream alive using keepalive probes.
// It returns when the connection dies or stopAddr is closed.
func (c *Connector) maintainConn(addr string, conn net.Conn, stopAddr <-chan struct{}, tick <-chan time.Time) {
	for {
		select {
		case <-stopAddr:
			return

		case <-tick:
			// Send a keepalive probe (AC-003 PC-2: ticker drives health probing).
			cf := halfchannel.ChannelFrame{
				FrameType: halfchannel.FrameTypeEmptyTick,
			}
			var sackBitmap [outerassembler.SACKBitmapSize]byte
			wire, aErr := outerassembler.Assemble(cf, sackBitmap, c.env)
			if aErr != nil {
				// Probe assembly failed — treat as dead connection.
				c.logf("upstream router %s unreachable\n", addr)
				return
			}
			if err := conn.SetWriteDeadline(time.Now().UTC().Add(c.keepaliveInterval)); err != nil {
				return
			}
			n, wErr := conn.Write(wire)
			if wErr != nil || n != len(wire) {
				// Write failed — connection is dead.
				c.logf("upstream router %s unreachable\n", addr)
				return
			}
			// Reset deadline.
			_ = conn.SetWriteDeadline(time.Time{})
		}
	}
}

// operativeBase returns the effective first-retry delay and backoff-reset value
// for a given keepaliveInterval (Q7, AC-003, F-P2-002 architect ruling).
//
// Semantics: the operative base IS keepaliveInterval, floored at BackoffBase so
// that an operator-configured interval below 500 ms still produces a sane
// reconnect schedule.  BackoffBase is a floor constant only — not the default
// reconnect delay.
func operativeBase(keepaliveInterval time.Duration) time.Duration {
	if keepaliveInterval < BackoffBase {
		return BackoffBase
	}
	return keepaliveInterval
}

// nextBackoff computes the next exponential backoff value with ±25% jitter,
// capped at BackoffCap (Q5, AC-002).
func nextBackoff(current time.Duration) time.Duration {
	doubled := current * 2
	if doubled > BackoffCap {
		doubled = BackoffCap
	}
	// Apply ±25% uniform jitter.
	jitter := float64(doubled) * BackoffJitterFraction * (2*rand.Float64() - 1) //nolint:gosec // non-crypto jitter
	result := time.Duration(float64(doubled) + jitter)
	if result < BackoffBase {
		result = BackoffBase
	}
	if result > BackoffCap {
		result = BackoffCap
	}
	return result
}

// logf writes a formatted message to c.w (nil-safe).
func (c *Connector) logf(format string, args ...any) {
	if c.w == nil {
		return
	}
	_, _ = fmt.Fprintf(c.w, format, args...)
}
