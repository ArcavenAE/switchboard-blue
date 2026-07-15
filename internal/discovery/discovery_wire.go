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

// RouterIngest is the router-mode-exclusive hop-1 ingest path
// (SEC-DW-01..07). Once implemented (Task 2 Green step) it is stateful
// across calls to Ingest: it holds the lastSeen replay-discard map
// (AC-008/AC-009/AC-010) that must persist between datagrams for the same
// (SVTNID, NodeAddr) pair, guarded by a mutex for concurrent socket-read-loop
// callers — neither is declared on this stub, since a Red Gate skeleton
// carries no behavioral state (BC-5.38.001).
//
// All exported methods are safe for concurrent use.
type RouterIngest struct {
	cfg RouterIngestConfig
}

// NewRouterIngest constructs a RouterIngest from cfg. cfg.Router must not be
// nil.
func NewRouterIngest(cfg RouterIngestConfig) *RouterIngest {
	return &RouterIngest{cfg: cfg}
}

// Ingest authenticates and processes one raw inbound multicast datagram.
//
// Fixed-offset key-selector extraction (SEC-DW-01, AC-005): SVTNID and
// NodeAddr are read via direct byte-slice indexing to select the
// verification key — decodeBody's variable-length, attacker-controlled
// session-entry walk is never invoked before HMAC verification succeeds.
// The HMAC computation itself covers the complete raw body bytes.
//
// HMAC-first fail-closed verification (AC-006): a lookup-miss and an
// HMAC-tag mismatch both resolve to the identical rejection outcome
// (RouterIngestDecision{Accept: false}), with no distinguishing return
// value, log line, or other externally observable signal.
//
// Replay/forward-acceptance gate (AC-008/AC-009/AC-010, SEC-DW-07): a
// cold-start datagram (no prior lastSeen entry) is always accepted; a
// non-increasing Sequence for an existing entry is discarded
// (Accept: true, Relay: false, no registry/lastSeen mutation); a
// strictly-increasing Sequence advances lastSeen and is accepted+relayed.
//
// Bounded read buffer (SEC-DW-02, AC-011), aggregate rate cap + visibility
// counter (SEC-DW-03, AC-012), and rate-limited failure logging (SEC-DW-04,
// AC-013) are enforced within Ingest's Green-step implementation; their
// exact composition with the caller's socket-read loop is Task 2's Green
// step, not fixed by this stub.
//
// STUB — S-BL.DISCOVERY-WIRE (Red Gate, BC-5.38.001). Not yet implemented;
// body panics unconditionally so no test can accidentally pass before
// Task 2's Green step.
func (ri *RouterIngest) Ingest(raw []byte) (RouterIngestDecision, error) {
	panic("not implemented: S-BL.DISCOVERY-WIRE RouterIngest.Ingest")
}
