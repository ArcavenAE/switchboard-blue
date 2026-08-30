// Package testenv — loopback_test.go is the Red Gate ② failing-test suite for
// S-BL.LOOPBACK-FULLSTACK (BC-2.01.001, BC-2.01.002, BC-2.01.003, BC-2.02.001,
// BC-2.02.002, BC-2.02.003, BC-2.02.005; VP-042).
//
// package testenv (white-box, not testenv_test): every AC below exercises
// unexported loopbackDriver internals (driver.pending, driver.downstreamHC,
// driver.downstreamARQ, the onUpstreamTick()/onDownstreamTick() seams,
// encodeRTID/decodeRTID, newLoopbackPaths) that are not reachable from the
// black-box testenv_test package. This mirrors the story's own binding
// requirement for AC-016/AC-017's fault-injection tests.
//
// GREEN STATE: loopback.go's loopbackDriver is fully implemented — every
// body below is real (no panic("TODO: ...") stubs remain, Red Gate ①/②
// discharged per BC-5.38.001). Every test in this file passes against the
// current loopback.go, individually and as part of a single combined
// `go test ./...` run; the Red Gate log documents the (now historical)
// per-test verification matrix that was required while the stubs were in
// place.
package testenv

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/halfchannel"
	"github.com/arcavenae/switchboard/internal/multipath"
	"github.com/arcavenae/switchboard/internal/session"
)

// --- shared fault-injection / diagnostic-capture stubs ---------------------

// recordingTB is the AC-016/AC-017 fault-injection stub specified verbatim by
// the story's Design Constraints ("Recording testing.TB requirement for
// AC-016/AC-017 fault-injection tests"). It EMBEDS the real enclosing t
// (constructed as &recordingTB{TB: t}, never the zero value) so that
// Helper/Cleanup/Fatalf — which NewLoopback/newEnv call unconditionally —
// promote through to a live TB instead of nil-panicking. Only Errorf is
// overridden, so driver.failLoud's t.Errorf is captured into errorfCalls
// rather than marking the enclosing (passing) fault-injection test FAILED.
//
// This shape is BINDING per the story (S-BL.LOOPBACK-FULLSTACK Design
// Constraints, Q6/§M2) — it must not be altered to serve any single test.
type recordingTB struct {
	testing.TB

	mu          sync.Mutex
	errorfCalls []string
}

func (r *recordingTB) Errorf(format string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.errorfCalls = append(r.errorfCalls, fmt.Sprintf(format, args...))
}

func (r *recordingTB) calls() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.errorfCalls))
	copy(out, r.errorfCalls)
	return out
}

// diagnosticRecordingTB captures Logf calls for AC-011's non-blocking
// t.Cleanup pending-map diagnostic. This is a SEPARATE stub from recordingTB
// (whose Errorf-only-override shape is a binding contract for AC-016/AC-017
// and must not be altered here). Logf is the established non-fatal-diagnostic
// idiom already used elsewhere in this package (see Console.Detach's
// `t.Logf("testenv: Console.Detach: %v (non-fatal)", err)` in testenv.go).
type diagnosticRecordingTB struct {
	testing.TB

	mu       sync.Mutex
	logCalls []string
}

func (d *diagnosticRecordingTB) Logf(format string, args ...any) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.logCalls = append(d.logCalls, fmt.Sprintf(format, args...))
}

func (d *diagnosticRecordingTB) calls() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]string, len(d.logCalls))
	copy(out, d.logCalls)
	return out
}

// fatalTB is the detection mechanism for TestNewLoopback_RejectsOutOfBoundsTickInterval
// (AC-002(a)). It EMBEDS the real enclosing t (constructed as &fatalTB{TB:
// t}) so Helper/Cleanup/Logf/etc. promote through to a live TB — but its own
// Fatal-family methods (Fatal/Fatalf/FailNow) never delegate to the embedded
// real TB's Fatal-family methods: they record failed=true locally and call
// runtime.Goexit() themselves, exactly mirroring what testing.T.Fatalf does
// internally, without ever touching the real *testing.T's failure state.
// Error-family methods (Error/Errorf/Fail) record failed=true without
// aborting, matching testing.TB's own non-fatal semantics.
//
// This sidesteps the documented restriction that FailNow/Fatal/Fatalf must
// be called only from "the goroutine running the Test function": since
// fatalTB's Fatal-family methods never call through to the real TB's
// Fatal-family methods, the real t.Fatal is never invoked from the
// background goroutine NewLoopback runs on below.
//
// Detecting rejection via `!t.Run(...)` (the prior shape of this test)
// marked the SUBTEST — and therefore the parent test and the package's
// process exit code — FAILED whenever NewLoopback correctly called
// b.Fatalf on an out-of-bounds config: a correct implementation could never
// make that test report success. fatalTB replaces only that detection
// mechanism; the three cases and their assertion semantics are unchanged.
type fatalTB struct {
	testing.TB

	mu     sync.Mutex
	failed bool
}

func (f *fatalTB) Fatal(args ...any) {
	f.setFailed()
	runtime.Goexit()
}

func (f *fatalTB) Fatalf(format string, args ...any) {
	f.setFailed()
	runtime.Goexit()
}

func (f *fatalTB) FailNow() {
	f.setFailed()
	runtime.Goexit()
}

func (f *fatalTB) Error(args ...any) {
	f.setFailed()
}

func (f *fatalTB) Errorf(format string, args ...any) {
	f.setFailed()
}

func (f *fatalTB) Fail() {
	f.setFailed()
}

func (f *fatalTB) setFailed() {
	f.mu.Lock()
	f.failed = true
	f.mu.Unlock()
}

func (f *fatalTB) didFail() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.failed
}

// --- AC-002 (traces to BC-2.01.001, BC-2.01.003; Q6) ------------------------

// TestNewLoopback_RejectsOutOfBoundsTickInterval — AC-002(a): NewLoopback
// must validate cfg.TickIntervalUpstream/TickIntervalDownstream against
// halfchannel.MinTickInterval/MaxTickInterval and b.Fatalf on an
// out-of-bounds value. Table-driven per the AC: below MinTickInterval, above
// MaxTickInterval, and exactly at MaxTickInterval (VP-042's own
// downstreamInterval, 50ms — legal boundary case).
//
// Safe to execute directly (does not panic): b.Fatalf uses runtime.Goexit,
// which the fatalTB-driven goroutine below catches cleanly — unlike the
// panic("TODO") stubs elsewhere in this file, this test fails via a
// genuine, non-crashing assertion today (NewLoopback's current unmodified
// body performs no interval validation at all).
//
// Detection mechanism: each case runs NewLoopback(ctx, ft, cfg) on its own
// goroutine against a fresh fatalTB. A rejection (b.Fatalf) is captured as
// ft.didFail() == true via runtime.Goexit on that goroutine, never as a
// failure of the real enclosing t — so a correct implementation reports
// this test as PASSING, not failing-by-design (see fatalTB doc above).
func TestNewLoopback_RejectsOutOfBoundsTickInterval(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name       string
		upstream   time.Duration
		downstream time.Duration
		wantReject bool
	}{
		{
			name:       "upstream below MinTickInterval",
			upstream:   1 * time.Millisecond,
			downstream: 50 * time.Millisecond,
			wantReject: true,
		},
		{
			name:       "downstream above MaxTickInterval",
			upstream:   10 * time.Millisecond,
			downstream: 51 * time.Millisecond,
			wantReject: true,
		},
		{
			name:       "downstream exactly at MaxTickInterval is legal (VP-042 boundary)",
			upstream:   10 * time.Millisecond,
			downstream: 50 * time.Millisecond,
			wantReject: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		ft := &fatalTB{TB: t}
		done := make(chan bool, 1)
		go func() {
			defer func() { done <- ft.didFail() }()
			NewLoopback(ctx, ft, LoopbackConfig{
				TickIntervalUpstream:   tt.upstream,
				TickIntervalDownstream: tt.downstream,
			})
		}()
		rejected := <-done
		if rejected != tt.wantReject {
			t.Errorf("AC-002(a): reject=%v (want %v) for upstream=%v downstream=%v — NewLoopback must b.Fatalf on out-of-bounds intervals against halfchannel.MinTickInterval/MaxTickInterval",
				rejected, tt.wantReject, tt.upstream, tt.downstream)
		}
	}
}

// TestLoopbackDriver_TicksFireOnSchedule — AC-002(b): both ticker goroutines
// fire HalfChannel.Tick() on their configured schedule independent of
// Enqueue timing — a keystroke enqueued between ticks waits for the next
// tick and never triggers an out-of-band delivery.
func TestLoopbackDriver_TicksFireOnSchedule(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	const (
		upstreamInterval   = 20 * time.Millisecond
		downstreamInterval = 20 * time.Millisecond
	)
	lb := NewLoopback(ctx, t, LoopbackConfig{
		TickIntervalUpstream:   upstreamInterval,
		TickIntervalDownstream: downstreamInterval,
	})
	t.Cleanup(lb.Env.Close)

	sid := lb.CreateSession(t)

	start := time.Now()
	rt := lb.SendKeystroke(t, sid, "x")
	payload, ok := lb.WaitForEcho(t, rt, 500*time.Millisecond)
	elapsed := time.Since(start)

	if !ok {
		t.Fatalf("AC-002(b): WaitForEcho timed out; tick-driven delivery never fired")
	}
	if elapsed < upstreamInterval {
		t.Errorf("AC-002(b): round trip completed in %v, faster than the configured upstream tick interval %v — Enqueue must not trigger an out-of-band delivery; the driver must wait for the next scheduled tick",
			elapsed, upstreamInterval)
	}
	id, ok2 := decodeRTID(payload)
	if !ok2 || id != rt.id {
		t.Errorf("AC-002(b): delivered payload did not decode to the sent round trip id (ok2=%v id=%d want=%d)", ok2, id, rt.id)
	}
}

// --- AC-003 (traces to BC-2.01.002; Non-Goals) ------------------------------

// TestLoopbackDriver_EmptyTicksNotDispatched — AC-003: the upstream ticker
// calls Tick() every interval regardless of whether data is enqueued (empty
// ticks are produced, satisfying BC-2.01.002), but an empty-tick
// ChannelFrame is never passed to multipath.Send — only data frames are
// wire-dispatched. Verified via the direct-callable onUpstreamTick() seam:
// upstreamHC.Seq() is the tick-count proxy (every call increments it), and a
// subsequent data-bearing tick must still complete a full round trip,
// proving the empty ticks above did not corrupt the driver's tick cadence.
//
// Uses createSessionNoTicker (F1 remediation): this test drives every tick
// manually via the onUpstreamTick()/onDownstreamTick() seams and reads
// upstreamHC.Seq() directly. CreateSession's auto-started upstream ticker
// goroutine would race those manual Tick() calls on the SAME un-locked
// HalfChannel (H2, AC-015) and could itself fire extra ticks, inflating
// upstreamHC.Seq() past the exact emptyTicks count this test asserts on.
func TestLoopbackDriver_EmptyTicksNotDispatched(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	lb := NewLoopback(ctx, t, LoopbackConfig{
		TickIntervalUpstream:   10 * time.Millisecond,
		TickIntervalDownstream: 50 * time.Millisecond,
	})
	t.Cleanup(lb.Env.Close)

	sid := lb.createSessionNoTicker(t)

	const emptyTicks = 5
	for i := 0; i < emptyTicks; i++ {
		lb.driver.onUpstreamTick()
	}
	if got := lb.driver.upstreamHC.Seq(); got != emptyTicks {
		t.Errorf("AC-003: after %d empty onUpstreamTick() calls, upstreamHC.Seq() = %d, want %d — Tick() must fire on every call regardless of data availability",
			emptyTicks, got, emptyTicks)
	}

	rt := lb.SendKeystroke(t, sid, "x")
	lb.driver.onUpstreamTick()
	lb.driver.onDownstreamTick()
	if _, ok := lb.WaitForEcho(t, rt, 200*time.Millisecond); !ok {
		t.Errorf("AC-003: a data-bearing tick issued after empty ticks did not complete a round trip")
	}
}

// --- AC-004 (traces to BC-2.02.001, BC-2.02.003; Q3, Q7) --------------------

// TestLoopbackDriver_DuplicateAndRaceDispatch — AC-004: a single Enqueue'd
// payload, once ticked, is dispatched by multipath.Send to BOTH synthetic
// paths.RankedPaths from newLoopbackPaths() (duplicate-and-race,
// BC-2.02.001) — the SendFunc fires exactly twice, once per selected path.
// multipath.Send also calls paths.Rank() on every dispatch (BC-2.02.003
// harness-scope aspect), incidental to this assertion — see Anchors
// Consumed.
func TestLoopbackDriver_DuplicateAndRaceDispatch(t *testing.T) {
	t.Parallel()

	rp := newLoopbackPaths()
	if len(rp) != 2 {
		t.Fatalf("AC-004: newLoopbackPaths() returned %d paths, want 2 (duplicate-and-race dispatch requires exactly two synthetic paths per direction)", len(rp))
	}

	mp := multipath.NewMultipath(rp, multipath.DefaultDropCacheSize)

	var mu sync.Mutex
	var calls int
	fn := func(pathID uint64, f multipath.Frame) error {
		mu.Lock()
		calls++
		mu.Unlock()
		return nil
	}

	f := multipath.Frame{Payload: encodeRTID("x", 1)}
	if _, err := mp.Send(f, fn); err != nil {
		t.Fatalf("AC-004: Multipath.Send: %v", err)
	}
	if calls != 2 {
		t.Errorf("AC-004: SendFunc fired %d times, want exactly 2 (once per synthetic path from newLoopbackPaths())", calls)
	}
}

// --- AC-005 (traces to BC-2.02.002; Q3, Q4) ---------------------------------

// TestLoopbackDriver_EndpointDedupDiscardsSecondArrival — AC-005: the
// second-arriving copy of a duplicate-and-raced frame is discarded by
// multipath.Receive's endpoint checksum dedup, both upstream and downstream
// (story text lines 882-887).
//
// Upstream leg: this gates accessNode.SendKeystroke directly — of the two
// deliverUpstream calls per tick, exactly one payload reaches downstreamHC
// via loopbackSink.SendInput, never two. Verified by draining downstreamHC
// directly after one upstream tick and counting data frames.
//
// Downstream leg (F2 remediation): downstreamARQ.OnAck's cadence is
// tick-driven, NOT gated by deliverDownstream's dedup outcome (see AC-006)
// — so this leg asserts the dedup separately, via the
// downstreamDeliverNilCount/downstreamDeliverDupCount/downstreamOnAckCount
// test-instrumentation counters: exactly one nil Receive, one ErrDuplicate
// Receive (deliverDownstream fires twice, once per synthetic path), and
// exactly one OnAck call, per ticked downstream data frame.
//
// Uses createSessionNoTicker (F1 remediation): this test drives every tick
// manually via the onUpstreamTick()/onDownstreamTick()/downstreamHC.Tick()
// seams. CreateSession's auto-started ticker goroutines would race those
// manual calls on the SAME un-locked HalfChannels (H2, AC-015) — and the
// downstream ticker could itself consume the queued data frame before this
// test's own drain loop observes it, driving dataFrames to 0.
func TestLoopbackDriver_EndpointDedupDiscardsSecondArrival(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	lb := NewLoopback(ctx, t, LoopbackConfig{
		TickIntervalUpstream:   10 * time.Millisecond,
		TickIntervalDownstream: 50 * time.Millisecond,
	})
	t.Cleanup(lb.Env.Close)

	sid := lb.createSessionNoTicker(t)

	// --- upstream leg: dedup gates accessNode.SendKeystroke ---
	_ = lb.SendKeystroke(t, sid, "x")
	lb.driver.onUpstreamTick()

	dataFrames := 0
	for i := 0; i < 4; i++ {
		cf := lb.driver.downstreamHC.Tick()
		if cf.FrameType == halfchannel.FrameTypeData {
			dataFrames++
		}
	}
	if dataFrames != 1 {
		t.Errorf("AC-005: downstreamHC received %d data frames after one upstream tick, want exactly 1 — endpoint dedup must discard the second-arriving duplicate before SendKeystroke ever runs a second time",
			dataFrames)
	}

	// --- downstream leg: dedup does NOT gate OnAck (see AC-006) ---
	rt2 := lb.SendKeystroke(t, sid, "y")
	lb.driver.onUpstreamTick()
	lb.driver.onDownstreamTick()

	if n := lb.driver.downstreamDeliverNilCount.Load(); n != 1 {
		t.Errorf("AC-005: deliverDownstream's Receive returned nil %d time(s) for the ticked data frame, want exactly 1 (the first-arriving path)", n)
	}
	if n := lb.driver.downstreamDeliverDupCount.Load(); n != 1 {
		t.Errorf("AC-005: deliverDownstream's Receive returned ErrDuplicate %d time(s) for the ticked data frame, want exactly 1 (the second-arriving path)", n)
	}
	if n := lb.driver.downstreamOnAckCount.Load(); n != 1 {
		t.Errorf("AC-005: downstreamARQ.OnAck fired %d time(s) for the ticked data frame, want exactly 1 — its cadence is tick-driven, not gated by deliverDownstream's dedup outcome (see AC-006)", n)
	}

	if _, ok := lb.WaitForEcho(t, rt2, 200*time.Millisecond); !ok {
		t.Errorf("AC-005: downstream leg's round trip did not complete despite exactly one OnAck delivery")
	}
}

// --- AC-006 (traces to BC-2.02.005; Q4 as REVISED by the Q4 Addendum) ------

// TestLoopbackDriver_DownstreamARQWiring — AC-006: every downstream tick
// that produces a data frame calls driver.downstreamARQ.EnqueueSend, then —
// in the SAME tick body, after Send returns — the SAME instance's OnAck
// exactly once. Verified end-to-end by a completed round trip (proving
// EnqueueSend/OnAck ran on the same *arq.ARQ instance — see AC-014) and by
// confirming no DegradationEvent (TLPKTDROP) fired for this no-loss tick.
func TestLoopbackDriver_DownstreamARQWiring(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	lb := NewLoopback(ctx, t, LoopbackConfig{
		TickIntervalUpstream:   10 * time.Millisecond,
		TickIntervalDownstream: 50 * time.Millisecond,
	})
	t.Cleanup(lb.Env.Close)

	sid := lb.CreateSession(t)
	rt := lb.SendKeystroke(t, sid, "x")
	lb.driver.onUpstreamTick()
	lb.driver.onDownstreamTick()

	payload, ok := lb.WaitForEcho(t, rt, 200*time.Millisecond)
	if !ok {
		t.Fatalf("AC-006: WaitForEcho timed out — EnqueueSend/OnAck on the same *arq.ARQ instance must deliver the payload within one downstream tick")
	}
	id, ok2 := decodeRTID(payload)
	if !ok2 || id != rt.id {
		t.Errorf("AC-006: delivered payload decode mismatch (ok2=%v id=%d want=%d)", ok2, id, rt.id)
	}

	select {
	case ev := <-lb.driver.downstreamARQ.DegradationEvents:
		t.Errorf("AC-006: unexpected DegradationEvent (TLPKTDROP) fired: %+v — GapsToRetransmit/TLPKTDROP must never be called in this no-loss harness", ev)
	default:
		// expected: no degradation event queued
	}
}

// --- AC-007 (traces to Q2 — dedicated shard) --------------------------------

// TestLoopbackDriver_DedicatedShard_NoDefaultShardMutation — AC-007: the
// loopback driver constructs its own dedicated Publisher/SessionAuth/
// AccessNode triple; env.defaultShard must never be mutated by constructing
// or exercising a LoopbackEnv. Verified via env.defaultShard.publisher's own
// session count staying unchanged across LoopbackEnv.CreateSession.
func TestLoopbackDriver_DedicatedShard_NoDefaultShardMutation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	lb := NewLoopback(ctx, t, LoopbackConfig{
		TickIntervalUpstream:   10 * time.Millisecond,
		TickIntervalDownstream: 50 * time.Millisecond,
	})
	t.Cleanup(lb.Env.Close)

	before := len(lb.Env.defaultShard.publisher.ListSessions())
	_ = lb.CreateSession(t)
	after := len(lb.Env.defaultShard.publisher.ListSessions())

	if after != before {
		t.Errorf("AC-007: env.defaultShard.publisher gained %d session(s) after LoopbackEnv.CreateSession — the loopback driver must publish into its OWN dedicated Publisher, never env.defaultShard",
			after-before)
	}
}

// TestSessionAccessNode_NoSetSinkMethod — AC-007 companion: a
// compile-time/reflection guard confirming session.AccessNode gained no new
// sink-mutation method. This invariant already holds on unmodified
// production code — it locks the prohibition ("no SetSink escape hatch is
// added to production session.AccessNode") so a future implementer cannot
// silently violate it while wiring the loopback driver's own dedicated
// AccessNode. Expected to PASS today by design: it is a negative-space
// regression guard on already-shipped code, not a test of new
// S-BL.LOOPBACK-FULLSTACK functionality.
func TestSessionAccessNode_NoSetSinkMethod(t *testing.T) {
	t.Parallel()
	typ := reflect.TypeOf((*session.AccessNode)(nil))
	for i := 0; i < typ.NumMethod(); i++ {
		name := typ.Method(i).Name
		if name == "SetSink" || name == "SetKeystrokeSink" {
			t.Errorf("AC-007: session.AccessNode gained a sink-mutation method %q — KeystrokeSink must remain fixed at construction via WithKeystrokeSink; no SetSink escape hatch is permitted",
				name)
		}
	}
}

// --- AC-008 (traces to Q5 — RoundTrip token API) ----------------------------

// TestLoopbackEnv_WaitForEcho_DoesNotConsumeOtherRoundTrips — AC-008:
// WaitForEcho consumes exactly one token, reading only that token's own
// completion channel — a concurrent round trip's frame must not satisfy a
// WaitForEcho call for a different token.
func TestLoopbackEnv_WaitForEcho_DoesNotConsumeOtherRoundTrips(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	lb := NewLoopback(ctx, t, LoopbackConfig{
		TickIntervalUpstream:   10 * time.Millisecond,
		TickIntervalDownstream: 50 * time.Millisecond,
	})
	t.Cleanup(lb.Env.Close)

	sid := lb.CreateSession(t)
	rt1 := lb.SendKeystroke(t, sid, "first")
	rt2 := lb.SendKeystroke(t, sid, "second")

	// Wait on the SECOND token first — it must not return early because of
	// the first round trip's frame arriving.
	payload2, ok2 := lb.WaitForEcho(t, rt2, 500*time.Millisecond)
	if !ok2 {
		t.Fatalf("AC-008: WaitForEcho(rt2) timed out")
	}
	id2, decOK2 := decodeRTID(payload2)
	if !decOK2 || id2 != rt2.id {
		t.Errorf("AC-008: WaitForEcho(rt2) returned payload id %d (ok=%v), want %d — a concurrent round trip's frame must not satisfy a different token's wait",
			id2, decOK2, rt2.id)
	}

	payload1, ok1 := lb.WaitForEcho(t, rt1, 500*time.Millisecond)
	if !ok1 {
		t.Fatalf("AC-008: WaitForEcho(rt1) timed out after rt2 was consumed")
	}
	id1, decOK1 := decodeRTID(payload1)
	if !decOK1 || id1 != rt1.id {
		t.Errorf("AC-008: WaitForEcho(rt1) returned payload id %d (ok=%v), want %d", id1, decOK1, rt1.id)
	}
}

// TestLoopbackEnv_WaitForEcho_IgnoresStaleCollectFramesBuffer — AC-008:
// WaitForEcho never reads Env.CollectFrames' accumulating buffer —
// pre-populating it (on an unrelated session) with a frame must not satisfy
// a round trip's own completion channel.
func TestLoopbackEnv_WaitForEcho_IgnoresStaleCollectFramesBuffer(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	lb := NewLoopback(ctx, t, LoopbackConfig{
		TickIntervalUpstream:   10 * time.Millisecond,
		TickIntervalDownstream: 50 * time.Millisecond,
	})
	t.Cleanup(lb.Env.Close)

	// Populate an unrelated frame on a completely separate Env-level
	// session — WaitForEcho must never consult this generic accumulating
	// buffer.
	unrelatedSID := lb.Env.CreateSession(t)
	_ = lb.Env.AttachConsole(t, unrelatedSID)
	lb.Env.SendKeystroke(t, unrelatedSID, "unrelated")

	sid := lb.CreateSession(t)
	rt := lb.SendKeystroke(t, sid, "x")
	payload, ok := lb.WaitForEcho(t, rt, 500*time.Millisecond)
	if !ok {
		t.Fatalf("AC-008: WaitForEcho timed out despite a pre-populated (unrelated) CollectFrames buffer on a different session")
	}
	id, decOK := decodeRTID(payload)
	if !decOK || id != rt.id {
		t.Errorf("AC-008: delivered payload decode mismatch (ok=%v id=%d want=%d) — WaitForEcho must read only rt's own completion channel, never Env.CollectFrames",
			decOK, id, rt.id)
	}
}

// --- AC-009 (traces to Risk 3 — RoundTrip.done buffering/no-leak/no-block) -

// TestLoopbackEnv_WaitForEcho_TimeoutThenLateArrival_NoLeak — AC-009:
// RoundTrip.done is buffered 1; the downstream ticker's completion path
// unconditionally drains driver.pending and sends the echo payload whether
// or not WaitForEcho has been called. A WaitForEcho timeout followed by a
// late arrival must not block the ticker goroutine, and driver.pending must
// no longer hold the entry once delivery lands.
func TestLoopbackEnv_WaitForEcho_TimeoutThenLateArrival_NoLeak(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	const downstreamInterval = 50 * time.Millisecond
	lb := NewLoopback(ctx, t, LoopbackConfig{
		TickIntervalUpstream:   10 * time.Millisecond,
		TickIntervalDownstream: downstreamInterval,
	})
	t.Cleanup(lb.Env.Close)

	sid := lb.CreateSession(t)
	rt := lb.SendKeystroke(t, sid, "x")

	// Timeout much shorter than one full round-trip cadence: the ticker has
	// not yet delivered when WaitForEcho gives up.
	if _, ok := lb.WaitForEcho(t, rt, 1*time.Millisecond); ok {
		t.Fatalf("AC-009: WaitForEcho unexpectedly succeeded within 1ms — cannot exercise the timeout-then-late-arrival path")
	}

	// Allow the real ticker cadence to deliver the echo well after the
	// above timeout, then confirm the send into done did not block and the
	// pending entry was drained.
	deadline := time.Now().Add(10 * downstreamInterval)
	for time.Now().Before(deadline) {
		lb.driver.mu.Lock()
		_, stillPending := lb.driver.pending[rt.id]
		lb.driver.mu.Unlock()
		if !stillPending {
			return // drained — AC-009 satisfied
		}
		time.Sleep(5 * time.Millisecond)
	}

	lb.driver.mu.Lock()
	_, stillPending := lb.driver.pending[rt.id]
	lb.driver.mu.Unlock()
	if stillPending {
		t.Errorf("AC-009: driver.pending[%d] still present after the late echo should have arrived — the downstream ticker's completion path must unconditionally delete the entry at delivery, independent of whether WaitForEcho consumed it",
			rt.id)
	}
}

// --- AC-010 (traces to Risk 2 — PathTracker.IsActive() initial-state) ------

// TestNewLoopbackPaths_TrackersActiveWithoutProbe — AC-010: every
// paths.RankedPath returned by newLoopbackPaths() must report
// IsActive() == true immediately at construction, with zero OnProbe calls —
// insurance against a future internal/paths change silently breaking
// loopback path activation.
func TestNewLoopbackPaths_TrackersActiveWithoutProbe(t *testing.T) {
	t.Parallel()
	rp := newLoopbackPaths()
	if len(rp) != 2 {
		t.Fatalf("AC-010: newLoopbackPaths() returned %d paths, want 2", len(rp))
	}
	for i, p := range rp {
		if p.Tracker == nil {
			t.Fatalf("AC-010: newLoopbackPaths()[%d].Tracker is nil", i)
		}
		if !p.Tracker.IsActive() {
			t.Errorf("AC-010: newLoopbackPaths()[%d].Tracker.IsActive() = false immediately after construction, want true (paths.NewPathTracker defaults active=true; no OnProbe call needed)",
				i)
		}
	}
}

// --- AC-011 / DECISION (traces to Risk 4 — pending-map diagnostic) --------

// TestLoopbackEnv_Cleanup_DiagnosticOnPendingLeak — AC-011: LoopbackEnv
// construction registers a t.Cleanup diagnostic that, at teardown,
// logs/reports any entries still in driver.pending. Deterministic via an
// injected decode-mismatch scenario: a synthetic mismatched-id entry is
// injected directly into driver.pending, and teardown must report it —
// non-fatal, does NOT use t.Fatalf. Captured via a diagnosticRecordingTB
// (Logf override) constructed inside a subtest so its t.Cleanup fires
// before this test inspects the result (t.Run blocks until the subtest,
// including its registered cleanups, has completed).
func TestLoopbackEnv_Cleanup_DiagnosticOnPendingLeak(t *testing.T) {
	t.Parallel()
	stub := &diagnosticRecordingTB{}
	const syntheticID = uint64(1 << 40) // deliberately never delivered

	t.Run("subtest", func(st *testing.T) {
		stub.TB = st
		ctx := context.Background()
		lb := NewLoopback(ctx, stub, LoopbackConfig{
			TickIntervalUpstream:   10 * time.Millisecond,
			TickIntervalDownstream: 50 * time.Millisecond,
		})
		sid := lb.CreateSession(stub)
		_ = lb.SendKeystroke(stub, sid, "x")

		lb.driver.mu.Lock()
		lb.driver.pending[syntheticID] = make(chan []byte, 1)
		lb.driver.mu.Unlock()
	})

	if calls := stub.calls(); len(calls) == 0 {
		t.Errorf("AC-011: no diagnostic reported for an undrained driver.pending entry at teardown (synthetic id %d)", syntheticID)
	}
}

// TestLoopbackEnv_Cleanup_SilentWhenDrained — AC-011 companion: a normal
// round trip + WaitForEcho leaves driver.pending empty at teardown, with no
// diagnostic firing.
func TestLoopbackEnv_Cleanup_SilentWhenDrained(t *testing.T) {
	t.Parallel()
	stub := &diagnosticRecordingTB{}

	t.Run("subtest", func(st *testing.T) {
		stub.TB = st
		ctx := context.Background()
		lb := NewLoopback(ctx, stub, LoopbackConfig{
			TickIntervalUpstream:   10 * time.Millisecond,
			TickIntervalDownstream: 50 * time.Millisecond,
		})
		sid := lb.CreateSession(stub)
		rt := lb.SendKeystroke(stub, sid, "x")
		if _, ok := lb.WaitForEcho(stub, rt, 500*time.Millisecond); !ok {
			st.Fatalf("AC-011 setup: WaitForEcho timed out")
		}
	})

	if calls := stub.calls(); len(calls) != 0 {
		t.Errorf("AC-011: diagnostic unexpectedly fired for a fully-drained driver.pending map: %v", calls)
	}
}

// --- AC-012 (traces to Q6 — goroutine lifecycle) ----------------------------

// TestLoopbackEnv_TickerGoroutines_JoinOnClose — AC-012: both ticker
// goroutines register on the existing Env.wg/Env.closeCh — no new
// WaitGroup or close channel. Env.Close() must join both goroutines
// deterministically within a bounded timeout, matching the existing
// AttachConsole/AttachProbe leak-check convention
// (TestClose_NoGoroutineLeak in testenv_test.go).
func TestLoopbackEnv_TickerGoroutines_JoinOnClose(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	lb := NewLoopback(ctx, t, LoopbackConfig{
		TickIntervalUpstream:   10 * time.Millisecond,
		TickIntervalDownstream: 50 * time.Millisecond,
	})
	sid := lb.CreateSession(t)
	_ = lb.SendKeystroke(t, sid, "x")

	done := make(chan struct{})
	go func() {
		lb.Env.Close()
		close(done)
	}()
	select {
	case <-done:
		// OK — both ticker goroutines joined.
	case <-time.After(1 * time.Second):
		t.Error("AC-012: Env.Close() did not complete within 1s — ticker goroutine leak suspected")
	}
}

// --- AC-014 (regression guard; traces to AC-001 Q4 Addendum) --------------

// TestLoopbackEnv_RoundTripCompletes_SingleSharedARQInstance — AC-014
// (MANDATORY behavioral regression guard): a full SendKeystroke ->
// WaitForEcho round trip through a REAL LoopbackEnv must COMPLETE — ok ==
// true AND the delivered payload decodes to the sent round trip's own id.
// This is the load-bearing assertion: a bare "did it return" check would
// not catch the ruled-out arqServer/arqClient two-instance shape, whose
// failure mode is a SILENT (nil, nil) forever from OnAck, manifesting only
// as a timeout.
func TestLoopbackEnv_RoundTripCompletes_SingleSharedARQInstance(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	lb := NewLoopback(ctx, t, LoopbackConfig{
		TickIntervalUpstream:   10 * time.Millisecond,
		TickIntervalDownstream: 50 * time.Millisecond,
	})
	t.Cleanup(lb.Env.Close)

	sid := lb.CreateSession(t)
	rt := lb.SendKeystroke(t, sid, "x")

	payload, ok := lb.WaitForEcho(t, rt, 2*time.Second)
	if !ok {
		t.Fatalf("AC-014: WaitForEcho timed out — a full round trip did not complete. This is the exact silent-failure symptom of a two-instance arqServer/arqClient split (AC-001 Addendum): OnAck on a never-EnqueueSend'd instance returns (nil, nil) forever")
	}
	id, ok2 := decodeRTID(payload)
	if !ok2 || id != rt.id {
		t.Fatalf("AC-014: delivered payload did not decode to the sent round trip id (ok2=%v id=%d want=%d) — non-empty, correctly-correlated delivery is the load-bearing assertion, not merely that WaitForEcho returned",
			ok2, id, rt.id)
	}
}

// --- AC-015 (traces to BC-2.01.001 / H2 — halfchannel synchronization) ----

// TestLoopbackDriver_NoRaceUnderConcurrentSendEcho — AC-015: the loopback
// driver runs clean under go test -race. The per-direction mutexes
// (upstreamHCMu, downstreamHCMu) must serialize concurrent Enqueue (test
// goroutines) against Tick (ticker goroutines) on each HalfChannel. This
// test is the per-story hook into the authoritative `just test-race` CI
// gate for this specific race class.
func TestLoopbackDriver_NoRaceUnderConcurrentSendEcho(t *testing.T) {
	ctx := context.Background()
	lb := NewLoopback(ctx, t, LoopbackConfig{
		TickIntervalUpstream:   5 * time.Millisecond,
		TickIntervalDownstream: 5 * time.Millisecond,
	})
	t.Cleanup(lb.Env.Close)

	sid := lb.CreateSession(t)

	const concurrency = 8
	var wg sync.WaitGroup
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func(n int) {
			defer wg.Done()
			rt := lb.SendKeystroke(t, sid, fmt.Sprintf("k%d", n))
			if _, ok := lb.WaitForEcho(t, rt, 2*time.Second); !ok {
				t.Errorf("AC-015: concurrent round trip %d timed out", n)
			}
		}(i)
	}
	wg.Wait()
}

// --- AC-016 (traces to BC-2.02.005 / §M2 note — OnAck error loud) ---------

// TestLoopbackDriver_OnAckError_SurfacesLoud — AC-016: if
// downstreamARQ.OnAck returns an error, the harness MUST surface it as a
// loud failure (driver.failLoud → t.Errorf) — never silently convert it to
// a WaitForEcho timeout. Uses the onDownstreamTick() direct seam (BINDING
// per note §M2 §B-3), single-goroutine throughout — no ticker goroutine is
// ever started.
func TestLoopbackDriver_OnAckError_SurfacesLoud(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	stub := &recordingTB{TB: t}
	lb := NewLoopback(ctx, stub, LoopbackConfig{
		TickIntervalUpstream:   10 * time.Millisecond,
		TickIntervalDownstream: 50 * time.Millisecond,
	})

	// Step 2: 65 synchronous empty onDownstreamTick() calls — each
	// increments downstreamHC.seq without calling EnqueueSend, so
	// downstreamARQ.nextExpected stays 0.
	for i := 0; i < 65; i++ {
		lb.driver.onDownstreamTick()
	}

	// Step 3: enqueue one data payload directly into downstreamHC.
	lb.driver.downstreamHCMu.Lock()
	if err := lb.driver.downstreamHC.Enqueue(encodeRTID("x", 1)); err != nil {
		lb.driver.downstreamHCMu.Unlock()
		t.Fatalf("AC-016 setup: downstreamHC.Enqueue: %v", err)
	}
	lb.driver.downstreamHCMu.Unlock()

	// Step 4: one more synchronous tick — fires EnqueueSend(chanSeq>64)
	// then OnAck(chanSeq, zeroSACK) with nextExpected=0 -> ErrAckOutOfWindow.
	lb.driver.onDownstreamTick()

	// Step 5: assert exactly one failLoud (Errorf) call was captured.
	if calls := stub.calls(); len(calls) != 1 {
		t.Errorf("AC-016: driver.failLoud fired %d times, want exactly 1 (OnAck's ErrAckOutOfWindow must be surfaced loud, not swallowed): %v",
			len(calls), calls)
	}
}

// --- AC-017 (traces to BC-2.01.001 — upstream-error loud failure) ---------

// TestLoopbackDriver_UpstreamDeliveryError_SurfacesLoud — AC-017: checks
// accessNode.SendKeystroke's error IN-PLACE inside deliverUpstream and
// surfaces it via driver.failLoud, symmetric with AC-016. Direct-seam
// 4-step method: never call CreateSession (so no console is ever attached
// and — per [v1.10 F-LENSB-01] — no upstream ticker goroutine ever exists),
// call SendKeystroke before CreateSession, then drive onUpstreamTick()
// synchronously.
func TestLoopbackDriver_UpstreamDeliveryError_SurfacesLoud(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	stub := &recordingTB{TB: t}
	lb := NewLoopback(ctx, stub, LoopbackConfig{
		TickIntervalUpstream:   10 * time.Millisecond,
		TickIntervalDownstream: 50 * time.Millisecond,
	})

	// Step 2: SendKeystroke before CreateSession — no console registered,
	// so accessNode.SendKeystroke will return ErrConsoleNotFound once the
	// tick runs. Per Q3/[v1.8 F-LENSB-B-02], SendKeystroke performs no
	// session-existence validation, so this call succeeds at the
	// mint/encode/enqueue level regardless of sessionID.
	sid := SessionID{}
	_ = lb.SendKeystroke(stub, sid, "x")

	// Step 3: drive the upstream tick body directly and synchronously.
	lb.driver.onUpstreamTick()

	// Step 4: assert exactly one failLoud (Errorf) call was captured.
	if calls := stub.calls(); len(calls) != 1 {
		t.Errorf("AC-017: driver.failLoud fired %d times, want exactly 1 (accessNode.SendKeystroke's ErrConsoleNotFound must be checked and surfaced loud, in-place inside deliverUpstream): %v",
			len(calls), calls)
	}
}
