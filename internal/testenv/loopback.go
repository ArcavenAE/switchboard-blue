// Package testenv — loopback.go extends NewLoopback/LoopbackEnv (testenv.go)
// from a same-goroutine DeliverFrame shortcut into a tick-driven,
// protocol-accurate loopback stack spanning internal/halfchannel +
// internal/arq + internal/multipath + internal/paths, so VP-042's benchmark
// measures the real round-trip path instead of an in-process echo shortcut.
//
// STUB NOTICE (S-BL.LOOPBACK-FULLSTACK, Red Gate ①): every non-trivial body
// in this file is panic("TODO: ...") per BC-5.38.001. Shapes only — the
// implementer stage fills these in against the story's Design Constraints
// (Q2-Q8) and the placement note's Q4 Addendum (AC-001 DISCHARGED — ONE
// shared *arq.ARQ instance for the downstream direction; never split into
// arqServer/arqClient).
package testenv

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

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
// (no Publish/RegisterKey/Attach has run). Only CreateSession provisions the
// console and starts both ticker goroutines.
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
}

// onUpstreamTick drives one upstream tick: HalfChannel.Tick() under
// upstreamHCMu, then — for a data frame — dispatch through upstreamMP.Send
// to deliverUpstream (Q3, AC-002, AC-004). Required directly-callable seam,
// binding per note §M2 §F-B-LENSB-01: the upstream ticker goroutine invokes
// this as its startLoopbackTicker tickBody, and AC-017's fault-injection
// test invokes it directly and synchronously with no ticker goroutine
// running.
func (d *loopbackDriver) onUpstreamTick() {
	panic("TODO: loopbackDriver.onUpstreamTick — Red Gate stub, S-BL.LOOPBACK-FULLSTACK")
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
	panic("TODO: loopbackDriver.onDownstreamTick — Red Gate stub, S-BL.LOOPBACK-FULLSTACK")
}

// failLoud surfaces err as a loud test failure via t.Errorf (never
// t.Fatalf, so a ticker goroutine can return cleanly) — SOUL.md §4
// no-silent-failure (Design Constraints §M2). driver.errCh is deliberately
// NOT used (v1.5 F-B-4): a buffered-1 channel deadlocks when both ticker
// goroutines call failLoud concurrently.
func (d *loopbackDriver) failLoud(err error) {
	panic("TODO: loopbackDriver.failLoud — Red Gate stub, S-BL.LOOPBACK-FULLSTACK")
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
func (d *loopbackDriver) deliverUpstream(pathID uint64, f multipath.Frame) error {
	panic("TODO: loopbackDriver.deliverUpstream — Red Gate stub, S-BL.LOOPBACK-FULLSTACK")
}

// deliverDownstream is downstreamMP's multipath.SendFunc (Q4): endpoint
// checksum dedup via downstreamMP.Receive only (ErrDuplicate discards the
// second-arriving copy) — it does NOT call downstreamARQ.OnAck.
// downstreamARQ.OnAck fires exactly once per downstream data tick, from
// onDownstreamTick's tick body, unconditionally — not gated by this dedup
// outcome (AC-005, AC-006).
func (d *loopbackDriver) deliverDownstream(pathID uint64, f multipath.Frame) error {
	panic("TODO: loopbackDriver.deliverDownstream — Red Gate stub, S-BL.LOOPBACK-FULLSTACK")
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
	panic("TODO: loopbackSink.SendInput — Red Gate stub, S-BL.LOOPBACK-FULLSTACK")
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
func (lb *LoopbackEnv) SendKeystroke(t testing.TB, sessionID SessionID, key string) RoundTrip {
	panic("TODO: LoopbackEnv.SendKeystroke — Red Gate stub, S-BL.LOOPBACK-FULLSTACK")
}

// WaitForEcho blocks until the echo tagged with rt arrives, or timeout
// elapses. Returns (payload, true) on delivery; (nil, false) on timeout —
// callers should t.Fatalf on timeout. Unlike Env.WaitForEcho, which returns
// as soon as ANY frame is buffered on the session (correct for the 10 other
// VPs' "did anything arrive" semantics), this reads only rt's own done
// channel — a concurrent or stale round trip's frame cannot satisfy it
// (Q5, AC-008). Does NOT read Env.CollectFrames' accumulating buffer.
func (lb *LoopbackEnv) WaitForEcho(t testing.TB, rt RoundTrip, timeout time.Duration) (payload []byte, ok bool) {
	panic("TODO: LoopbackEnv.WaitForEcho — Red Gate stub, S-BL.LOOPBACK-FULLSTACK")
}

// CreateSession provisions the loopback driver's dedicated console (Q2,
// H3): sh.pub.Publish(sessionName) — BEFORE Attach, whose pub.Get gate
// requires it — then sh.auth.RegisterKey(sessionName, loopbackConsoleKey,
// session.RoleFull), then sh.access.Attach(loopbackConsoleKey, sessionName).
// sessionName and loopbackConsoleKey are stored on the driver at this point
// (never earlier — "Driver lifecycle pin"). CreateSession additionally
// starts BOTH ticker goroutines, upstream and downstream — its sole
// start-site (M2, v1.10 F-LENSB-01, symmetric for both directions) — as a
// separate concern, unordered relative to the provisioning steps above. It
// does NOT construct the driver, the multipath instances, or the
// half-channels; those already exist from NewLoopback.
func (lb *LoopbackEnv) CreateSession(t testing.TB) SessionID {
	panic("TODO: LoopbackEnv.CreateSession — Red Gate stub, S-BL.LOOPBACK-FULLSTACK")
}

// startLoopbackTicker registers a ticker goroutine on env's EXISTING
// wg/closeCh (Q6) — no new WaitGroup or close channel, matching the
// AttachConsole/AttachProbe idiom. Tick-free (v1.5 F-B-1): it does NOT call
// hc.Tick() itself — tickBody owns the Tick() call under the appropriate
// per-direction mutex (H2), so every Tick() is mutex-guarded regardless of
// how the goroutine invokes the body. Wired: upstream ticker's tickBody is
// d.onUpstreamTick; downstream ticker's tickBody is d.onDownstreamTick.
func startLoopbackTicker(env *Env, interval time.Duration, tickBody func()) {
	panic("TODO: startLoopbackTicker — Red Gate stub, S-BL.LOOPBACK-FULLSTACK")
}

// newLoopbackPaths constructs the two synthetic paths.RankedPath used by
// each direction's Multipath instance (Q7, AC-010) — four total across
// upstreamMP/downstreamMP. Each is backed by a fresh
// paths.NewPathTracker(1.0, 0.125): NewPathTracker sets active: true at
// construction, so a fresh tracker is immediately eligible for Rank without
// any probe history — no OnProbe calls needed.
func newLoopbackPaths() []paths.RankedPath {
	panic("TODO: newLoopbackPaths — Red Gate stub, S-BL.LOOPBACK-FULLSTACK")
}

// toMPFrame converts a halfchannel.ChannelFrame to a multipath.Frame:
// copies f.Payload into Frame.Payload. Does NOT carry f.ChanSeq into the
// returned struct — multipath.Frame has no ChanSeq field (B1 fix); the
// caller MUST capture chanSeq := f.ChanSeq BEFORE calling toMPFrame and use
// that captured value at the OnAck call site.
func toMPFrame(f halfchannel.ChannelFrame) multipath.Frame {
	panic("TODO: toMPFrame — Red Gate stub, S-BL.LOOPBACK-FULLSTACK")
}

// encodeRTID encodes a round-trip id as an 8-byte big-endian suffix
// appended to key. Returns the combined payload:
// append([]byte(key), id_bytes...). Package-private, pure (no I/O). Must be
// the inverse of decodeRTID: decodeRTID(encodeRTID(key, id)) == (id, true)
// for all key, id (§M4 obligation).
func encodeRTID(key string, id uint64) []byte {
	panic("TODO: encodeRTID — Red Gate stub, S-BL.LOOPBACK-FULLSTACK")
}

// decodeRTID extracts the round-trip id from the last 8 bytes of payload.
// Returns (id, true) if len(payload) >= 8; (0, false) otherwise.
// Package-private, pure (no I/O). Because rtSeq.Add(1) starts ids at 1,
// id=0 (the !ok sentinel) never collides with a real driver.pending key —
// a decode failure is safely diagnosable, not a false hit.
func decodeRTID(payload []byte) (id uint64, ok bool) {
	panic("TODO: decodeRTID — Red Gate stub, S-BL.LOOPBACK-FULLSTACK")
}

// zeroSACK is the all-zero SACK bitmap passed to downstreamARQ.OnAck in the
// no-loss harness (M4). Correct because Non-Goals excludes packet loss and
// reordering; a future loss-injection story must replace this with a real
// bitmap. Zero value, never written.
var zeroSACK [arq.SACKBitmapBytes]byte
