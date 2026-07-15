// discovery_wire.go implements the router-side hop-1 ingest path for SVTN
// discovery advertisements (S-BL.DISCOVERY-WIRE Task 2; SEC-DW-01..07).
//
// RouterIngest.Ingest authenticates an inbound raw multicast datagram via
// the admitted-node HMAC key surface (Ruling 1's DiscoveryAuthKeyFor),
// applies the replay/forward-acceptance Sequence gate (SEC-DW-07), and
// returns an accept/relay decision to its caller — it performs no relay I/O
// itself (Design Constraint: Router-Mode Discovery Wiring). The caller
// (cmd/switchboard's multicast listener, Task 3) owns the actual
// net.ListenMulticastUDP socket and the hop-2 relay-dispatch closure
// (Task 6, GATED).
//
// Purity classification (ARCH-09): boundary — RouterIngest holds the
// per-(SVTN,NodeAddr) lastSeen replay-discard map as of Task 2's Green step
// (mutable under mutex); the accept/reject decision itself is a pure
// function of (raw bytes, current lastSeen state, admitted-key lookup
// result).

package discovery

import (
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/arcavenae/switchboard/internal/routing"
)

// RouterIngestConfig holds the parameters used to construct a RouterIngest.
type RouterIngestConfig struct {
	// Router is used for admitted-node HMAC key lookup via
	// DiscoveryAuthKeyFor (Ruling 1; AC-005/AC-006). internal/discovery
	// imports ONLY internal/routing among internal/ packages (ARCH-08 §6.5
	// position 14) — FailureCounter visibility (SEC-DW-03/SEC-DW-04) and
	// any HMAC-failure recording are invoked through this Router, never
	// through a direct internal/admission import.
	Router *routing.Router
	// Logger receives rate-limited, threshold-crossing HMAC-failure log
	// lines (SEC-DW-04, AC-013) — never per-packet (that's BC-2.05.008's
	// TCP policy, explicitly not adopted here). Added during Red Gate step
	// 2 (test-writing): routing.Router.logger is unexported, so Ingest has
	// no other seam to log through without either a new Router accessor
	// method or a direct internal/admission import (forbidden by ARCH-08
	// §6.5 position 14). Optional; nil means log emissions are silently
	// discarded, matching routing.Router's own nopLogger default.
	Logger routing.Logger
}

// RouterIngestDecision is the accept/relay decision RouterIngest.Ingest
// returns to its caller. internal/discovery performs no relay I/O itself
// (Design Constraint: Router-Mode Discovery Wiring) — the caller uses this
// decision to decide whether to invoke the hop-2 relay-dispatch closure
// (Task 6, GATED — depends_on S-BL.NODE-IDENTIFY-WIRE) and to construct the
// DISCOVERY_RELAY frame (AC-014, Task 5).
type RouterIngestDecision struct {
	// Accept records whether the datagram passed HMAC verification
	// (AC-006). A rejected datagram (Accept == false) never mutates the
	// registry or lastSeen state and is never relayed.
	Accept bool
	// Relay records whether an accepted datagram also passed the
	// Sequence replay-discard gate (AC-008/AC-009/AC-010) and should be
	// forwarded via hop-2. An accepted-but-stale datagram (AC-009) has
	// Accept == true, Relay == false.
	Relay bool
	// SVTNID is the advertisement's declared SVTN (AC-005 postcondition 1).
	SVTNID [16]byte
	// NodeAddr is the ORIGINATING access node's 8-byte address.
	NodeAddr [8]byte
	// Sequence is the accepted datagram's uint64 BE, epoch-qualified
	// Sequence value (F-DWSP4-001, SEC-DW-07) — the same value AC-014's
	// relay-frame assembly (Task 5) re-serializes.
	Sequence uint64
	// Sessions is the decoded per-session list (populated only when
	// Accept is true).
	Sessions []SessionPresence
}

// discoveryFailureRecorder mirrors *admission.FailureCounter's
// RecordHMACFailure method surface without internal/discovery importing
// internal/admission directly (ARCH-08 §6.5 position 14). Satisfied
// structurally by the value routing.NewDiscoveryFailureCounter returns.
type discoveryFailureRecorder interface {
	RecordHMACFailure(srcAddr string)
}

// nopDiscoveryLogger discards every log line. Used when RouterIngestConfig.Logger
// is nil, mirroring routing.Router's own nopLogger default.
type nopDiscoveryLogger struct{}

func (nopDiscoveryLogger) Log(string) {}

// discoveryFailureThreshold and discoveryFailureWindow are the FailureCounter
// constants reused for discovery's ingest path (BC-2.05.005 PC-3; AC-012/
// AC-013), matching the existing TCP-path threshold/window exactly
// (cmd/switchboard's hmacFailureThreshold/hmacFailureWindow).
const (
	discoveryFailureThreshold = 5
	discoveryFailureWindow    = 60 * time.Second
)

// keySelectorMinRaw is the minimum raw datagram length before any key
// lookup is attempted: 8-byte HMAC tag + 16-byte SVTNID + 8-byte NodeAddr
// (SEC-DW-01, AC-005 postcondition 4).
const keySelectorMinRaw = routing.AdvertisementHMACTagSize + 16 + 8

// hop1BodyMinLen is the minimum post-tag body length for a syntactically
// complete hop-1 frame once HMAC has passed: 16-byte SVTNID + 8-byte
// NodeAddr + 8-byte Sequence + 2-byte session count (AC-005's note: the
// full valid-frame minimum is 42 raw bytes = 8 tag + 34 body).
const hop1BodyMinLen = 16 + 8 + 8 + 2

// maxDiscoveryDatagramSize bounds the router's ingest path against an
// oversized raw datagram (SEC-DW-02, AC-011) — sized to a realistic
// worst-case legitimate advertisement (comfortably above
// maxSessionsPerAdvertisement sessions with reasonable name lengths), not
// the 65,507-byte UDP/IP theoretical maximum.
const maxDiscoveryDatagramSize = 32768

// maxSessionsPerAdvertisement bounds the declared per-advertisement session
// count (SEC-DW-02, AC-011 postcondition 3). Re-derived from realistic
// tmux-sessions-per-access-node scale — low hundreds, not the prior
// TCP/length-prefixed-framing-era 1024.
const maxSessionsPerAdvertisement = 256

// aggregateRateBurst and aggregateRateFillPerSec size the router's
// aggregate (not per-source) ingest token bucket (SEC-DW-03(a), AC-012
// postconditions 1 and 3) — generous headroom for a legitimate SVTN's
// initial admission burst, while still bounding sustained flood volume
// regardless of how many distinct declared NodeAddr values an attacker
// rotates through.
const (
	aggregateRateBurst      = 500
	aggregateRateFillPerSec = 100.0
)

// lastSeenKey is the composite map key for RouterIngest's per-(SVTN,
// NodeAddr) replay-discard watermark (SEC-DW-07).
type lastSeenKey struct {
	svtnID   [16]byte
	nodeAddr [8]byte
}

// tokenBucket is a minimal thread-safe token bucket used for the ingest
// path's aggregate rate cap (SEC-DW-03(a)).
type tokenBucket struct {
	mu         sync.Mutex
	tokens     float64
	capacity   float64
	fillPerSec float64
	last       time.Time
	now        func() time.Time
}

func newTokenBucket(capacity, fillPerSec float64) *tokenBucket {
	return &tokenBucket{
		tokens:     capacity,
		capacity:   capacity,
		fillPerSec: fillPerSec,
		last:       time.Now(),
		now:        time.Now,
	}
}

// allow reports whether one token is available, consuming it if so.
func (b *tokenBucket) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := b.now()
	elapsed := now.Sub(b.last).Seconds()
	b.last = now
	if elapsed > 0 {
		b.tokens += elapsed * b.fillPerSec
		if b.tokens > b.capacity {
			b.tokens = b.capacity
		}
	}
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// RouterIngest is the router-mode-exclusive hop-1 ingest path
// (SEC-DW-01..07). It is stateful across calls to Ingest: it holds the
// lastSeen replay-discard map (AC-008/AC-009/AC-010) that persists between
// datagrams for the same (SVTNID, NodeAddr) pair, guarded by a mutex for
// concurrent socket-read-loop callers, plus an aggregate rate-limiter
// (SEC-DW-03(a)) and a reused FailureCounter for HMAC-rejection visibility
// (SEC-DW-03(b)/SEC-DW-04).
//
// All exported methods are safe for concurrent use.
type RouterIngest struct {
	cfg RouterIngestConfig

	mu       sync.Mutex
	lastSeen map[lastSeenKey]uint64

	rateLimiter    *tokenBucket
	failureCounter discoveryFailureRecorder
}

// NewRouterIngest constructs a RouterIngest from cfg. cfg.Router must not be
// nil — the router is the sole source of admitted-node key material this
// ingest path authenticates against (go.md rule 13: a security-perimeter
// constructor fails closed, not open, on a missing dependency).
func NewRouterIngest(cfg RouterIngestConfig) *RouterIngest {
	if cfg.Router == nil {
		panic("discovery: NewRouterIngest: cfg.Router must not be nil")
	}
	logger := cfg.Logger
	if logger == nil {
		logger = nopDiscoveryLogger{}
	}
	return &RouterIngest{
		cfg:            cfg,
		lastSeen:       make(map[lastSeenKey]uint64),
		rateLimiter:    newTokenBucket(aggregateRateBurst, aggregateRateFillPerSec),
		failureCounter: routing.NewDiscoveryFailureCounter(discoveryFailureThreshold, discoveryFailureWindow, logger),
	}
}

// Ingest authenticates and processes one raw inbound multicast datagram.
//
// Fixed-offset key-selector extraction (SEC-DW-01, AC-005): SVTNID and
// NodeAddr are read via direct byte-slice indexing to select the
// verification key — DecodeSessionList's variable-length,
// attacker-controlled session-entry walk is never invoked before HMAC
// verification succeeds. The HMAC computation itself covers the complete
// raw body bytes.
//
// HMAC-first fail-closed verification (AC-006): a lookup-miss and an
// HMAC-tag mismatch both resolve to the identical rejection outcome
// (RouterIngestDecision{Accept: false}, ErrInvalidHMACTag), with no
// distinguishing return value, log line, or other externally observable
// signal.
//
// Replay/forward-acceptance gate (AC-008/AC-009/AC-010, SEC-DW-07): a
// cold-start datagram (no prior lastSeen entry) is always accepted; a
// non-increasing Sequence for an existing entry is discarded
// (Accept: true, Relay: false, no lastSeen mutation); a strictly-increasing
// Sequence advances lastSeen and is accepted+relayed.
//
// Bounded read buffer (SEC-DW-02, AC-011), aggregate rate cap + visibility
// counter (SEC-DW-03, AC-012), and rate-limited failure logging (SEC-DW-04,
// AC-013) are enforced here; the socket-read loop that owns the actual
// network I/O is the caller's responsibility (Task 3).
func (ri *RouterIngest) Ingest(raw []byte) (RouterIngestDecision, error) {
	// SEC-DW-03(a): aggregate (not per-source) rate cap — the cheapest,
	// earliest defense, evaluated before any parsing so a flood cannot be
	// used to drive unbounded HMAC-computation cost either.
	if !ri.rateLimiter.allow() {
		return RouterIngestDecision{}, ErrInvalidHMACTag
	}

	// SEC-DW-01 / AC-005 postcondition 4: a raw datagram shorter than the
	// 8-byte tag + 24-byte SVTNID/NodeAddr key selector is rejected before
	// any key lookup is attempted.
	if len(raw) < keySelectorMinRaw {
		return RouterIngestDecision{}, ErrInvalidHMACTag
	}

	// SEC-DW-02 / AC-011: bounded read buffer — an oversized datagram is
	// rejected without partial-parse, before any further processing.
	if len(raw) > maxDiscoveryDatagramSize {
		return RouterIngestDecision{}, ErrInvalidHMACTag
	}

	body := raw[routing.AdvertisementHMACTagSize:]
	var wireTag [routing.AdvertisementHMACTagSize]byte
	copy(wireTag[:], raw[:routing.AdvertisementHMACTagSize])

	// SEC-DW-01 postconditions 1/2: fixed-offset extraction of the
	// key-selector fields via direct byte-slice indexing — never a call to
	// DecodeSessionList (which walks the variable-length,
	// attacker-controlled session-entry list) before HMAC succeeds.
	var svtnID [16]byte
	var nodeAddr [8]byte
	copy(svtnID[:], body[0:16])
	copy(nodeAddr[:], body[16:24])

	// AC-006 / SEC-DW-05 (MUST clause): a lookup-miss and an HMAC-tag
	// mismatch resolve to the identical rejection outcome
	// (RouterIngestDecision{}, ErrInvalidHMACTag) with no distinguishing
	// return value, log line, or other externally observable signal —
	// there is no wire-visible accept/reject differential, satisfying
	// SEC-DW-05's MUST clause (advertisements are one-way fire-and-forget
	// UDP with no ack, so no response-content oracle exists by
	// construction either).
	//
	// `ok &&` short-circuits: on a lookup-miss (ok == false) Go does NOT
	// evaluate VerifyAdvertisementHMAC, so processing time is NOT
	// symmetric between a lookup-miss and a tag-mismatch — a lookup-miss
	// returns measurably faster. This is an accepted SEC-DW-05 residual:
	// the story adopts the outcome-unification MUST clause but explicitly
	// leaves processing-time symmetry as optional hardening
	// (dummy-HMAC-on-lookup-miss), not required and not implemented here.
	key, ok := ri.cfg.Router.DiscoveryAuthKeyFor(svtnID, nodeAddr)
	verified := ok && routing.VerifyAdvertisementHMAC(key[:], body, wireTag)
	if !verified {
		// AC-012 postcondition 2 / AC-013: FailureCounter is invoked for
		// operator visibility only — its own threshold-crossing emission is
		// the sole logging trigger (never per-packet); it never gates this
		// or any future Ingest call on the declared, attacker-controlled
		// NodeAddr.
		ri.failureCounter.RecordHMACFailure(fmt.Sprintf("%x", nodeAddr))
		return RouterIngestDecision{}, ErrInvalidHMACTag
	}

	// HMAC verified — safe to parse the remainder of the body now
	// (SEC-DW-01 postcondition 2). A syntactically-too-short-but-somehow-
	// authenticated body (unreachable from any real sender, but not
	// provably impossible) fails closed here rather than slicing out of
	// bounds below.
	if len(body) < hop1BodyMinLen {
		return RouterIngestDecision{}, ErrInvalidHMACTag
	}
	sequence := binary.BigEndian.Uint64(body[24:32])
	sessions, err := DecodeSessionList(body[32:])
	if err != nil {
		return RouterIngestDecision{}, ErrInvalidHMACTag
	}

	decision := RouterIngestDecision{
		Accept:   true,
		SVTNID:   svtnID,
		NodeAddr: nodeAddr,
		Sequence: sequence,
		Sessions: sessions,
	}

	// SEC-DW-07 / AC-008/AC-009/AC-010: replay/forward-acceptance gate.
	ri.mu.Lock()
	defer ri.mu.Unlock()
	k := lastSeenKey{svtnID: svtnID, nodeAddr: nodeAddr}
	last, seen := ri.lastSeen[k]
	if !seen || sequence > last {
		ri.lastSeen[k] = sequence
		decision.Relay = true
	}
	return decision, nil
}
