// node_identify_wire.go — NODE_IDENTIFY (control_type=0x04) three-message
// handshake codec and driver for cmd/switchboard (S-BL.NODE-IDENTIFY-WIRE).
//
// Provides:
//   - Payload-size constants for the three handshake messages (rulings §§4–6).
//   - Pure codec functions: encodeNodeIdentify, encodeChallenge,
//     encodeChallengeResponse (pure-core, no I/O).
//   - Pure decode functions: decodeNodeIdentify, decodeChallengeResponse
//     (pure-core, all size guards enforced).
//   - nodeIdentifyHandshake driver: effectful-shell, TCP I/O, all failure
//     paths fail-closed (BC-2.01.009 PC-1 through PC-8; rulings §7, §13).
//
// Purity classification (ARCH-09): codec functions are pure-core; handshake
// driver is effectful-shell (TCP reads/writes, conn.SetDeadline).
//
// Architecture note: cmd/switchboard already imports internal/netingress via
// mgmt_wire.go; this file adding the same import does not gain a new
// package-level dep (ARCH-08 compliance, "cmd/switchboard MUST NOT gain
// internal/netingress import" refers to a new package-level edge, not a
// per-file re-import within the existing package).
//
// Traces to BC-2.01.009; rulings S-BL.NODE-IDENTIFY-WIRE-rulings.md §§2–9,13.
package main

import (
	"crypto/ed25519"
	"errors"
	"net"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/netingress"
	"github.com/arcavenae/switchboard/internal/routing"
)

// nodeIdentifyHandshakeTimeout is the deadline for the complete NODE_IDENTIFY
// three-message exchange. Set via conn.SetDeadline before the first io.ReadFull;
// cleared on success. Matches admission_sync_client.go:154 handshakeTimeout
// precedent (rulings §13).
//
// Traces to BC-2.01.009 Precondition 4; E-ADM-022.
const nodeIdentifyHandshakeTimeout = 10 * time.Second

// nodeIdentifyPayloadSize is the exact fixed wire size of the NodeIdentify
// payload in bytes (rulings §4):
//
//	control_type(1) + version(1) + msg_kind(1) + reserved(1) + node_pubkey(32) = 36
//
// BC-2.01.009 Invariant 5: exact payload lengths are enforced; any deviation
// closes the connection immediately.
const nodeIdentifyPayloadSize = 36

// challengePayloadSize is the exact fixed wire size of the Challenge payload
// in bytes (rulings §5):
//
//	control_type(1) + version(1) + msg_kind(1) + reserved(1) + nonce(32) + router_sig(64) = 100
const challengePayloadSize = 100

// challengeResponsePayloadSize is the exact fixed wire size of the
// ChallengeResponse payload in bytes (rulings §6):
//
//	control_type(1) + version(1) + msg_kind(1) + reserved(1) + nonce_sig(64) = 68
const challengeResponsePayloadSize = 68

// encodeNodeIdentify assembles the 80-byte NodeIdentify frame (44-byte outer
// header + 36-byte payload, msg_kind=0x01) for a connecting node. Pure-core:
// no I/O; deterministic serialization.
//
// The returned frame carries a zero HMACTag (pre-admission trust boundary is
// the challenge-response itself, not a per-frame HMAC — rulings §3).
//
// Traces to BC-2.01.009 Postcondition 1; rulings §4.
func encodeNodeIdentify(svtnID [16]byte, pubkey ed25519.PublicKey) []byte {
	// todo: unimplemented
	return nil
}

// encodeChallenge assembles the 144-byte Challenge frame (44-byte outer
// header + 100-byte payload, msg_kind=0x02) from a router-generated
// admission.Challenge. Pure-core.
//
// Traces to BC-2.01.009 Postcondition 2; rulings §5.
func encodeChallenge(svtnID [16]byte, challenge admission.Challenge) []byte {
	// todo: unimplemented
	return nil
}

// encodeChallengeResponse assembles the 112-byte ChallengeResponse frame
// (44-byte outer header + 68-byte payload, msg_kind=0x03) from a node's
// admission.ChallengeResponse. Pure-core.
//
// Used by tests to construct ChallengeResponse frames simulating a connecting
// node; the production node-side caller is loadOrGenerateAdmissionKeypair
// (access.go) but that path is out of scope here.
//
// Traces to BC-2.01.009 Postcondition 3; rulings §6.
func encodeChallengeResponse(svtnID [16]byte, resp admission.ChallengeResponse) []byte {
	// todo: unimplemented
	return nil
}

// decodeNodeIdentify decodes the 36-byte NodeIdentify payload received from a
// connecting node. Returns the 32-byte Ed25519 public key. Returns a non-nil
// error on any size or field-value violation (fail-closed per BC-2.01.009
// Invariant 5). Pure-core.
//
// Decoder preconditions (all must hold; violation → error → connection close):
//   - len(payload) == nodeIdentifyPayloadSize (36)
//   - payload[0] == 0x04 (control_type NODE_IDENTIFY)
//   - payload[1] == 0x01 (version frame.VersionByte)
//   - payload[2] == 0x01 (msg_kind NodeIdentify)
//   - payload[3] == 0x00 (reserved byte; non-zero is a hard decoder error)
//
// Traces to BC-2.01.009 Invariant 5; rulings §4.
func decodeNodeIdentify(payload []byte) (ed25519.PublicKey, error) {
	return nil, errors.New("unimplemented: decodeNodeIdentify")
}

// decodeChallengeResponse decodes the 68-byte ChallengeResponse payload
// received from a connecting node. Returns an admission.ChallengeResponse.
// Returns a non-nil error on any size or field-value violation. Pure-core.
//
// Decoder preconditions (all must hold; violation → error → connection close):
//   - len(payload) == challengeResponsePayloadSize (68)
//   - payload[0] == 0x04 (control_type NODE_IDENTIFY)
//   - payload[1] == 0x01 (version frame.VersionByte)
//   - payload[2] == 0x03 (msg_kind ChallengeResponse)
//   - payload[3] == 0x00 (reserved byte; non-zero is a hard decoder error)
//
// Traces to BC-2.01.009 Invariant 5; rulings §6.
func decodeChallengeResponse(payload []byte) (admission.ChallengeResponse, error) {
	return admission.ChallengeResponse{}, errors.New("unimplemented: decodeChallengeResponse")
}

// nodeIdentifyHandshake executes the complete three-message NODE_IDENTIFY
// handshake (NodeIdentify → Challenge → ChallengeResponse) directly on conn
// before netingress.ServeConn starts reading. On success it records the
// (svtnID, nodeAddr) → h.IfaceID binding via r.BindInterface and clears the
// conn deadline.
//
// Called from the onAccept closure in runRouter, BEFORE sendMap.Store, satisfying
// the F-P2L1-001 register-before-serve invariant (identical to the
// wireAdmissionSyncHandlers/serveMgmtServer ordering in admission-sync-wire).
//
// Failure posture: any error at any step closes conn immediately and returns
// a non-nil error. The caller (onAccept) MUST return a no-op cleanup func and
// NOT call sendMap.Store on failure. After this function returns, the
// connection is either fully bound or closed — no "unbound but open" state.
//
// Signature note (discrepancy with story Task 18): the story pins
// `h netingress.ConnHandle` but the existing netingress type is
// `netingress.NodeHandle`. This stub uses the corrected type name.
//
// Traces to BC-2.01.009 PC-1 through PC-8; rulings §7, §13.
func nodeIdentifyHandshake(
	conn net.Conn,
	r *routing.Router,
	routerPrivKey ed25519.PrivateKey,
	ks *admission.AdmittedKeySet,
	h netingress.NodeHandle,
) (svtnID [16]byte, nodeAddr [8]byte, err error) {
	return [16]byte{}, [8]byte{}, errors.New("unimplemented: nodeIdentifyHandshake")
}
