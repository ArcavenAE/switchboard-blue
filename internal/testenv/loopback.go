// Package testenv — loopback.go extends NewLoopback/LoopbackEnv (testenv.go)
// from a same-goroutine DeliverFrame shortcut into a tick-driven,
// protocol-accurate loopback stack spanning internal/halfchannel +
// internal/arq + internal/multipath + internal/paths, so VP-042's benchmark
// measures the real round-trip path instead of an in-process echo shortcut.
//
// Implements S-BL.LOOPBACK-FULLSTACK's Design Constraints (Q2-Q8) and the
// placement note's Q4 Addendum (AC-001 DISCHARGED — ONE shared *arq.ARQ
// instance for the downstream direction; never split into
// arqServer/arqClient).
package testenv

import (
	"encoding/binary"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/arq"
	"github.com/arcavenae/switchboard/internal/halfchannel"
	"github.com/arcavenae/switchboard/internal/multipath"
	"github.com/arcavenae/switchboard/internal/paths"
	"github.com/arcavenae/switchboard/internal/session"
)

// loopbackDriver owns the tick-driven, protocol-accurate loopback stack
// backing LoopbackEnv's SendKeystroke/WaitForEcho/CreateSession API (Design
// Constraints Q2-Q7). It is unexported and owned by LoopbackEnv.
//
// Driver lifecycle pin (Design Constraints Q2 §H3): the constructor (invoked
// once, from NewLoopback, before CreateSession is ever called) builds pub/
// auth/access, both Multipath instances, and both HalfChannels fully
// initialized and immediately usable — but with the console UN-PROVISIONED
// (no Publish/RegisterKey/Attach has run). CreateSession is the only
// production/non-deterministic-test entry point that provisions the console
// and starts both ticker goroutines; createSessionNoTicker (F1 remediation)
// provisions the SAME console via the shared provisionConsole body but starts
// neither ticker, for deterministic tests that drive ticks manually.
type loopbackDriver struct {
	env *Env

	// pub/auth/access form the driver's own dedicated Publisher/SessionAuth/
	// AccessNode triple (Q2) — never env.defaultShard. newShard hardcodes
	// session.WithKeystrokeSink(session.NoOpSink{}) and AccessNode has no
	// SetSink, so the loopback driver builds its own triple with
	// session.WithKeystrokeSink(loopbackSink) from the start instead.
	pub    *session.Publisher
	auth   *session.SessionAuth
	access *session.AccessNode

	// upstreamHCMu serializes upstreamHC.Enqueue (test goroutine, via
	// SendKeystroke) and upstreamHC.Tick (upstream ticker goroutine) — H2:
	// HalfChannel is not safe for concurrent use (AC-015).
	upstreamHCMu sync.Mutex
	upstreamHC   *halfchannel.HalfChannel

	// downstreamHCMu serializes downstreamHC.Enqueue (upstream ticker
	// goroutine, via loopbackSink.SendInput) and downstreamHC.Tick
	// (downstream ticker goroutine) — H2 (AC-015).
	downstreamHCMu sync.Mutex
	downstreamHC   *halfchannel.HalfChannel

	// upstreamMP/downstreamMP are the two *multipath.Multipath instances —
	// one per direction (Q7). Each dispatches every payload over both
	// synthetic paths.RankedPaths from newLoopbackPaths (duplicate-and-race,
	// BC-2.02.001), with endpoint checksum dedup on Receive (BC-2.02.002).
	upstreamMP   *multipath.Multipath
	downstreamMP *multipath.Multipath

	// downstreamARQ is the SINGLE shared *arq.ARQ instance for the
	// downstream direction (AC-001 DISCHARGED, verdict REVISED — Design
	// Constraints Q4 as amended by the Q4 Addendum). EnqueueSend and OnAck
	// for a given ChanSeq MUST be called on this ONE instance, in that
	// order, within the same downstream-ticker tick. MUST NOT be split into
	// separate arqServer/arqClient instances (AC-014 is the regression
	// guard): OnAck's payload recovery reads only the calling instance's
	// own inFlight/reorderBuf, populated exclusively by that SAME
	// instance's prior EnqueueSend calls — a never-EnqueueSend'd second
	// instance returns (nil, nil) from OnAck on every call, silently, and
	// every WaitForEcho would time out.
	downstreamARQ *arq.ARQ

	// mu guards pending.
	mu sync.Mutex

	// pending correlates in-flight round trips by RoundTrip.id to their
	// completion channel (Q5, H1). The downstream ticker's completion path
	// unconditionally deletes the entry and sends the echo payload,
	// independent of whether WaitForEcho has been called (AC-009).
	pending map[uint64]chan []byte

	// rtSeq mints RoundTrip ids. rtSeq.Add(1) starts ids at 1 so id=0 (the
	// decodeRTID !ok sentinel) never collides with a real pending key.
	rtSeq atomic.Uint64

	// loopbackConsoleKey and sessionName are populated by CreateSession
	// ONLY — never by this constructor (H3 "Driver lifecycle pin"). Both
	// hold their zero value before CreateSession has run; AC-017's
	// pre-CreateSession SendKeystroke call depends on that zero-value state
	// to observe ErrConsoleNotFound via failLoud.
	loopbackConsoleKey session.ConsoleKey
	sessionName        string

	// downstreamDeliverNilCount/downstreamDeliverDupCount/
	// downstreamOnAckCount are test-visible instrumentation counters for
	// AC-005's downstream assertions (story text lines 882-887): per ticked
	// downstream data frame, deliverDownstream's dedup Receive call must
	// return nil exactly once and ErrDuplicate exactly once (one count
	// each), and downstreamARQ.OnAck must fire exactly once. atomic.Uint64
	// because they are read from a test goroutine while a ticker goroutine
	// (or a direct onDownstreamTick() call) may be running concurrently for
	// other in-flight round trips — pure counters, incrementing them changes
	// no control flow.
	downstreamDeliverNilCount atomic.Uint64
	downstreamDeliverDupCount atomic.Uint64
	downstreamOnAckCount      atomic.Uint64
}

// validateLoopbackConfig enforces NewLoopback's tick-interval bounds (Q6):
// cfg.TickIntervalUpstream/TickIntervalDownstream must fall within
// [halfchannel.MinTickInterval, halfchannel.MaxTickInterval] (5ms-50ms).
// b.Fatalf on an out-of-bounds value, matching the existing fail-loud
// convention (t.Fatalf on illegal construction throughout this package,
// e.g. NewWithRouters). VP-042's own downstreamInterval (50ms) sits exactly
// at MaxTickInterval — legal (AC-002).
func validateLoopbackConfig(b testing.TB, cfg LoopbackConfig) {
	b.Helper()
	if cfg.TickIntervalUpstream < halfchannel.MinTickInterval || cfg.TickIntervalUpstream > halfchannel.MaxTickInterval {
		b.Fatalf("testenv.NewLoopback: TickIntervalUpstream %v out of bounds [%v, %v]",
			cfg.TickIntervalUpstream, halfchannel.MinTickInterval, halfchannel.MaxTickInterval)
	}
	if cfg.TickIntervalDownstream < halfchannel.MinTickInterval || cfg.TickIntervalDownstream > halfchannel.MaxTickInterval {
		b.Fatalf("testenv.NewLoopback: TickIntervalDownstream %v out of bounds [%v, %v]",
			cfg.TickIntervalDownstream, halfchannel.MinTickInterval, halfchannel.MaxTickInterval)
	}
}

// newLoopbackDriver constructs a loopbackDriver fully initialized and
// immediately usable — pub/auth/access triple, both Multipath instances,
// both HalfChannels, and the single shared downstreamARQ instance — but
// with the console UN-PROVISIONED (Driver lifecycle pin, §H3). Only
// CreateSession provisions the console (Publish/RegisterKey/Attach) and
// starts the ticker goroutines.
//
// The driver needs its own dedicated Publisher/SessionAuth/AccessNode
// triple, not env.defaultShard (Q2): newShard hardcodes
// session.WithKeystrokeSink(session.NoOpSink{}), and session.AccessNode has
// no SetSink — the KeystrokeSink is fixed at construction. This duplication
// is isolated to the loopback path; it does not touch newShard or any other
// VP's shard (AC-007).
func newLoopbackDriver(env *Env, cfg LoopbackConfig) *loopbackDriver {
	ks := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(ks)
	auth := session.NewSessionAuth()

	d := &loopbackDriver{
		env:  env,
		pub:  pub,
		auth: auth,

		upstreamHC:   halfchannel.New(1, halfchannel.Upstream, cfg.TickIntervalUpstream),
		downstreamHC: halfchannel.New(2, halfchannel.Downstream, cfg.TickIntervalDownstream),

		upstreamMP:   multipath.NewMultipath(newLoopbackPaths(), multipath.DefaultDropCacheSize),
		downstreamMP: multipath.NewMultipath(newLoopbackPaths(), multipath.DefaultDropCacheSize),

		// AC-001 DISCHARGED (Q4 Addendum): a single shared *arq.ARQ instance
		// for the downstream direction — EnqueueSend and OnAck are called on
		// this SAME instance (AC-014).
		downstreamARQ: arq.New(arq.Config{TickInterval: cfg.TickIntervalDownstream}),

		pending: make(map[uint64]chan []byte),
	}
	d.access = session.NewAccessNode(pub, auth, session.WithKeystrokeSink(&loopbackSink{driver: d}))
	return d
}

// onUpstreamTick drives one upstream tick: HalfChannel.Tick() under
// upstreamHCMu, then — for a data frame — dispatch through upstreamMP.Send
// to deliverUpstream (Q3, AC-002, AC-004). Required directly-callable seam,
// binding per note §M2 §F-B-LENSB-01: the upstream ticker goroutine invokes
// this as its startLoopbackTicker tickBody, and AC-017's fault-injection
// test invokes it directly and synchronously with no ticker goroutine
// running.
func (d *loopbackDriver) onUpstreamTick() {
	d.upstreamHCMu.Lock()
	f := d.upstreamHC.Tick()
	d.upstreamHCMu.Unlock()

	if f.FrameType != halfchannel.FrameTypeData {
		// Empty ticks are produced (BC-2.01.002) but not wire-dispatched
		// (Non-Goals).
		return
	}

	// [v1.14 F-R18-LENSB-01] The per-path SendKeystroke error is checked
	// IN-PLACE inside deliverUpstream, not on Send's own return value here —
	// Send returns nil whenever at least one selected path's Sent is true,
	// and the dedup sibling always returns nil without calling
	// SendKeystroke, so a check here can never observe a failing delivering
	// path's error.
	_, _ = d.upstreamMP.Send(toMPFrame(f), d.deliverUpstream)
}

// onDownstreamTick drives one downstream tick: HalfChannel.Tick() under
// downstreamHCMu; for a data frame, captures chanSeq := f.ChanSeq (B1),
// calls downstreamARQ.EnqueueSend, dispatches through downstreamMP.Send to
// deliverDownstream, THEN — still in this tick body, using the same
// tick-body-local chanSeq, NOT gated by deliverDownstream's dedup outcome —
// calls downstreamARQ.OnAck exactly once, checks its error loudly via
// failLoud (M2, AC-016), and delivers each returned payload to its
// driver.pending waiter (Q4 as amended by the Q4 Addendum, AC-006).
// Required directly-callable seam, binding per note §M2 §B-3: the
// downstream ticker goroutine invokes this as its startLoopbackTicker
// tickBody, and AC-016's fault-injection test invokes it directly and
// synchronously with no ticker goroutine running.
func (d *loopbackDriver) onDownstreamTick() {
	d.downstreamHCMu.Lock()
	f := d.downstreamHC.Tick()
	d.downstreamHCMu.Unlock()

	if f.FrameType != halfchannel.FrameTypeData {
		// Empty ticks are produced (BC-2.01.002) but not wire-dispatched
		// (Non-Goals) — and never fed into downstreamARQ.
		return
	}

	// [B1] Capture chanSeq from the ChannelFrame BEFORE toMPFrame — multipath.Frame
	// has no ChanSeq field/method.
	chanSeq := f.ChanSeq
	d.downstreamARQ.EnqueueSend(chanSeq, f.Payload, time.Now().UTC())

	// deliverDownstream runs synchronously (once per selected path, up to 2)
	// within this goroutine before Send returns (Q3 rationale); chanSeq
	// remains valid, in scope, at the OnAck call site below regardless of
	// how deliverDownstream's dedup resolved.
	_, _ = d.downstreamMP.Send(toMPFrame(f), d.deliverDownstream)

	// [Q4 Addendum / AC-001] OnAck fires exactly once per tick, on the SAME
	// shared downstreamARQ instance that received EnqueueSend above — NOT
	// gated by, and not invoked from inside, deliverDownstream's dedup.
	delivered, err := d.downstreamARQ.OnAck(chanSeq, zeroSACK)
	// [F2/AC-005] Counted unconditionally, alongside the real call above —
	// the counter's cadence must match OnAck's own cadence exactly,
	// including the error path (AC-016's ErrAckOutOfWindow case still
	// called OnAck once).
	d.downstreamOnAckCount.Add(1)
	if err != nil {
		// [M2] err MUST be checked and fail loud — not swallowed.
		d.failLoud(fmt.Errorf("downstreamARQ.OnAck seq=%d: %w", chanSeq, err))
		return
	}

	for _, payload := range delivered {
		id, ok := decodeRTID(payload)
		if !ok {
			// Decode failure: payload too short — cannot be a real pending
			// key (rtSeq.Add(1) starts ids at 1, so id=0 never collides).
			continue
		}
		d.mu.Lock()
		ch := d.pending[id]
		delete(d.pending, id)
		d.mu.Unlock()
		if ch != nil {
			ch <- payload
		}
	}
}

// failLoud surfaces err as a loud test failure via t.Errorf (never
// t.Fatalf, so a ticker goroutine can return cleanly) — SOUL.md §4
// no-silent-failure (Design Constraints §M2). driver.errCh is deliberately
// NOT used (v1.5 F-B-4): a buffered-1 channel deadlocks when both ticker
// goroutines call failLoud concurrently.
func (d *loopbackDriver) failLoud(err error) {
	d.env.t.Helper()
	d.env.t.Errorf("loopbackDriver: %v", err)
}

// deliverUpstream is upstreamMP's multipath.SendFunc (Q3): endpoint
// checksum dedup via upstreamMP.Receive (ErrDuplicate discards the
// second-arriving copy), then — first-arriving path only —
// access.SendKeystroke(loopbackConsoleKey, sessionName, f.Payload). Its
// error is checked IN-PLACE here and surfaced via failLoud before still
// being returned (AC-017; v1.14 F-R18-LENSB-01) — NOT checked on
// upstreamMP.Send's return value in onUpstreamTick, which can never observe
// the masked per-path error (multipath.Send returns nil whenever at least
// one selected path's Sent is true, and the dedup sibling always returns
// nil without ever calling SendKeystroke).
func (d *loopbackDriver) deliverUpstream(_ uint64, f multipath.Frame) error {
	if err := d.upstreamMP.Receive(f); err != nil {
		// ErrDuplicate: second-arriving copy of a duplicate-and-raced frame
		// (BC-2.02.002) — discard without ever calling SendKeystroke.
		return nil
	}

	if err := d.access.SendKeystroke(d.loopbackConsoleKey, d.sessionName, f.Payload); err != nil {
		// [v1.14 F-R18-LENSB-01] Checked IN-PLACE here — see onUpstreamTick's
		// comment for why a post-Send check would be unsound.
		d.failLoud(err)
		return err
	}
	return nil
}

// deliverDownstream is downstreamMP's multipath.SendFunc (Q4): endpoint
// checksum dedup via downstreamMP.Receive only (ErrDuplicate discards the
// second-arriving copy) — it does NOT call downstreamARQ.OnAck.
// downstreamARQ.OnAck fires exactly once per downstream data tick, from
// onDownstreamTick's tick body, unconditionally — not gated by this dedup
// outcome (AC-005, AC-006).
func (d *loopbackDriver) deliverDownstream(_ uint64, f multipath.Frame) error {
	// Endpoint checksum dedup only (BC-2.02.002); ErrDuplicate discards the
	// second-arriving copy silently. downstreamARQ.OnAck fires exactly once
	// per downstream data tick from onDownstreamTick's own tick body,
	// unconditionally — not gated by this dedup outcome (AC-005, AC-006).
	//
	// [F2/AC-005] The nil/ErrDuplicate outcome is counted here — pure
	// instrumentation, does not change which branch runs or what is
	// returned.
	if err := d.downstreamMP.Receive(f); err != nil {
		d.downstreamDeliverDupCount.Add(1)
		return nil
	}
	d.downstreamDeliverNilCount.Add(1)
	return nil
}

// RoundTrip identifies one SendKeystroke → echo round trip in a loopback
// environment (Q5). Returned by LoopbackEnv.SendKeystroke; consumed exactly
// once by LoopbackEnv.WaitForEcho. Opaque outside this package.
type RoundTrip struct {
	id uint64
	// done is buffered 1; written by the downstream ticker goroutine (or
	// onDownstreamTick called directly) on delivery. Carries the full echo
	// payload, including the 8-byte RT-ID suffix — NOT a frame.OuterHeader
	// (H1).
	done chan []byte
}

// loopbackSink is the session.KeystrokeSink injected into the loopback
// driver's dedicated AccessNode via session.WithKeystrokeSink (Q2, Q4) — the
// echo generator. It does not need to understand the RT-ID correlation
// scheme; it just echoes bytes, like real tmux would.
type loopbackSink struct {
	driver *loopbackDriver
}

// SendInput implements session.KeystrokeSink. Echoes the full payload
// verbatim into downstreamHC.Enqueue (under downstreamHCMu — H2), including
// the embedded RT-ID suffix. Called while AccessNode holds sinkMu — must not
// call back into AccessNode under any lock; Enqueue only touches
// downstreamHC's own pending queue, so this is safe by construction, and it
// is also the correct modeling of BC-2.01.001: the echo is queued, not
// delivered synchronously.
func (s *loopbackSink) SendInput(payload []byte) error {
	s.driver.downstreamHCMu.Lock()
	defer s.driver.downstreamHCMu.Unlock()
	return s.driver.downstreamHC.Enqueue(payload)
}

// SendKeystroke drives a keystroke through the full loopback protocol stack
// (Q3) and returns a token identifying this specific round trip: mints
// RoundTrip{id: driver.rtSeq.Add(1)}, registers its completion channel under
// that id in driver.pending (guarded by driver.mu), encodes the payload via
// encodeRTID(key, id), and enqueues into upstreamHC (under upstreamHCMu) —
// pure and non-blocking; SendKeystroke does NOT block on delivery.
//
// SendKeystroke performs NO session-existence validation — this is
// deliberate and load-bearing for AC-017 (v1.8 F-LENSB-B-02): it
// unconditionally mints/encodes/enqueues regardless of whether sessionID
// has been provisioned via CreateSession. An implementer-added defensive
// session-existence guard would abort AC-017 before onUpstreamTick ever
// runs, defeating the failLoud path AC-017 exercises. No such guard is
// permitted.
func (lb *LoopbackEnv) SendKeystroke(t testing.TB, _ SessionID, key string) RoundTrip {
	t.Helper()
	d := lb.driver

	id := d.rtSeq.Add(1)
	done := make(chan []byte, 1)

	d.mu.Lock()
	d.pending[id] = done
	d.mu.Unlock()

	payload := encodeRTID(key, id)

	d.upstreamHCMu.Lock()
	err := d.upstreamHC.Enqueue(payload)
	d.upstreamHCMu.Unlock()
	if err != nil {
		t.Fatalf("LoopbackEnv.SendKeystroke: upstreamHC.Enqueue: %v", err)
	}

	return RoundTrip{id: id, done: done}
}

// WaitForEcho blocks until the echo tagged with rt arrives, or timeout
// elapses. Returns (payload, true) on delivery; (nil, false) on timeout —
// callers should t.Fatalf on timeout. Unlike Env.WaitForEcho, which returns
// as soon as ANY frame is buffered on the session (correct for the 10 other
// VPs' "did anything arrive" semantics), this reads only rt's own done
// channel — a concurrent or stale round trip's frame cannot satisfy it
// (Q5, AC-008). Does NOT read Env.CollectFrames' accumulating buffer.
func (lb *LoopbackEnv) WaitForEcho(t testing.TB, rt RoundTrip, timeout time.Duration) (payload []byte, ok bool) {
	t.Helper()
	select {
	case payload := <-rt.done:
		return payload, true
	case <-time.After(timeout):
		return nil, false
	}
}

// provisionConsole provisions the loopback driver's dedicated console (Q2,
// H3): sh.pub.Publish(sessionName) — BEFORE Attach, whose pub.Get gate
// requires it — then sh.auth.RegisterKey(sessionName, loopbackConsoleKey,
// session.RoleFull), then sh.access.Attach(loopbackConsoleKey, sessionName).
// sessionName and loopbackConsoleKey are stored on the driver at this point
// (never earlier — "Driver lifecycle pin"). It does NOT start either ticker
// goroutine and does NOT construct the driver, the multipath instances, or
// the half-channels; those already exist from NewLoopback. Extracted so
// CreateSession (starts both tickers) and createSessionNoTicker (starts
// neither, for deterministic tests — F1 remediation) share this single
// provisioning body instead of diverging.
func (lb *LoopbackEnv) provisionConsole(t testing.TB) SessionID {
	t.Helper()
	d := lb.driver

	sessionName := fmt.Sprintf("loopback-session-%d", d.env.sessionSeq.Add(1))
	d.sessionName = sessionName
	d.loopbackConsoleKey = d.env.newConsoleKey()

	// [v1.3 B-F1] Publish BEFORE Attach — Attach's pub.Get gate requires it.
	if err := d.pub.Publish(sessionName); err != nil {
		t.Fatalf("loopbackDriver: Publish session %q: %v", sessionName, err)
	}
	d.auth.RegisterKey(sessionName, d.loopbackConsoleKey, session.RoleFull)
	if _, _, err := d.access.Attach(d.loopbackConsoleKey, sessionName); err != nil {
		t.Fatalf("loopbackDriver: Attach loopback console: %v", err)
	}

	return SessionID{name: sessionName}
}

// CreateSession provisions the loopback driver's dedicated console (via
// provisionConsole) and additionally starts BOTH ticker goroutines, upstream
// and downstream — its sole start-site (M2, v1.10 F-LENSB-01, symmetric for
// both directions) — as a separate concern, unordered relative to the
// provisioning steps. This is the external contract every non-deterministic
// caller (AC-006/AC-014, the VP-042 benchmark) depends on; it is unchanged
// by the provisionConsole extraction.
func (lb *LoopbackEnv) CreateSession(t testing.TB) SessionID {
	t.Helper()
	sid := lb.provisionConsole(t)
	d := lb.driver

	// [M2, v1.10 F-LENSB-01] Sole start-site for both ticker goroutines,
	// symmetric — unordered relative to the provisioning steps above.
	startLoopbackTicker(d.env, d.upstreamHC.TickInterval(), d.onUpstreamTick)
	startLoopbackTicker(d.env, d.downstreamHC.TickInterval(), d.onDownstreamTick)

	return sid
}

// createSessionNoTicker provisions the loopback driver's console (via
// provisionConsole) WITHOUT starting either ticker goroutine (F1
// remediation). For deterministic tests that drive every tick manually via
// the onUpstreamTick()/onDownstreamTick()/downstreamHC.Tick() seams: a live
// ticker goroutine on the SAME HalfChannel would race those manual calls
// (HalfChannel has no internal lock — H2, AC-015) and could also consume or
// inflate the exact tick counts those tests assert on. Package-private:
// every non-deterministic caller (production code, the benchmark, and every
// other test) keeps using CreateSession, whose external contract is
// unchanged.
func (lb *LoopbackEnv) createSessionNoTicker(t testing.TB) SessionID {
	t.Helper()
	return lb.provisionConsole(t)
}

// startLoopbackTicker registers a ticker goroutine on env's EXISTING
// wg/closeCh (Q6) — no new WaitGroup or close channel, matching the
// AttachConsole/AttachProbe idiom. Tick-free (v1.5 F-B-1): it does NOT call
// hc.Tick() itself — tickBody owns the Tick() call under the appropriate
// per-direction mutex (H2), so every Tick() is mutex-guarded regardless of
// how the goroutine invokes the body. Wired: upstream ticker's tickBody is
// d.onUpstreamTick; downstream ticker's tickBody is d.onDownstreamTick.
func startLoopbackTicker(env *Env, interval time.Duration, tickBody func()) {
	env.wg.Add(1)
	go func() {
		defer env.wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-env.closeCh:
				return
			case <-ticker.C:
				tickBody()
			}
		}
	}()
}

// newLoopbackPaths constructs the two synthetic paths.RankedPath used by
// each direction's Multipath instance (Q7, AC-010) — four total across
// upstreamMP/downstreamMP. Each is backed by a fresh
// paths.NewPathTracker(1.0, 0.125): NewPathTracker sets active: true at
// construction, so a fresh tracker is immediately eligible for Rank without
// any probe history — no OnProbe calls needed.
func newLoopbackPaths() []paths.RankedPath {
	return []paths.RankedPath{
		{ID: 1, Tracker: paths.NewPathTracker(1.0, 0.125)},
		{ID: 2, Tracker: paths.NewPathTracker(1.0, 0.125)},
	}
}

// toMPFrame converts a halfchannel.ChannelFrame to a multipath.Frame:
// copies f.Payload into Frame.Payload. Does NOT carry f.ChanSeq into the
// returned struct — multipath.Frame has no ChanSeq field (B1 fix); the
// caller MUST capture chanSeq := f.ChanSeq BEFORE calling toMPFrame and use
// that captured value at the OnAck call site.
func toMPFrame(f halfchannel.ChannelFrame) multipath.Frame {
	return multipath.Frame{Payload: f.Payload}
}

// encodeRTID encodes a round-trip id as an 8-byte big-endian suffix
// appended to key. Returns the combined payload:
// append([]byte(key), id_bytes...). Package-private, pure (no I/O). Must be
// the inverse of decodeRTID: decodeRTID(encodeRTID(key, id)) == (id, true)
// for all key, id (§M4 obligation).
func encodeRTID(key string, id uint64) []byte {
	buf := make([]byte, len(key)+8)
	copy(buf, key)
	binary.BigEndian.PutUint64(buf[len(key):], id)
	return buf
}

// decodeRTID extracts the round-trip id from the last 8 bytes of payload.
// Returns (id, true) if len(payload) >= 8; (0, false) otherwise.
// Package-private, pure (no I/O). Because rtSeq.Add(1) starts ids at 1,
// id=0 (the !ok sentinel) never collides with a real driver.pending key —
// a decode failure is safely diagnosable, not a false hit.
func decodeRTID(payload []byte) (id uint64, ok bool) {
	if len(payload) < 8 {
		return 0, false
	}
	return binary.BigEndian.Uint64(payload[len(payload)-8:]), true
}

// zeroSACK is the all-zero SACK bitmap passed to downstreamARQ.OnAck in the
// no-loss harness (M4). Correct because Non-Goals excludes packet loss and
// reordering; a future loss-injection story must replace this with a real
// bitmap. Zero value, never written.
var zeroSACK [arq.SACKBitmapBytes]byte
