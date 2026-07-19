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
	"fmt"
	"net"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/frame"
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

// nodeIdentifyControlType is the control_type discriminator for the
// NODE_IDENTIFY handshake sub-protocol (BC-2.01.008 registry; rulings §2).
const nodeIdentifyControlType = 0x04

// msg_kind constants for the three NODE_IDENTIFY handshake messages (rulings §2).
const (
	msgKindNodeIdentify      = 0x01
	msgKindChallenge         = 0x02
	msgKindChallengeResponse = 0x03
)

// encodeNodeIdentify assembles the 80-byte NodeIdentify frame (44-byte outer
// header + 36-byte payload, msg_kind=0x01) for a connecting node. Pure-core:
// no I/O; deterministic serialization.
//
// The returned frame carries a zero HMACTag (pre-admission trust boundary is
// the challenge-response itself, not a per-frame HMAC — rulings §3).
//
// Traces to BC-2.01.009 Postcondition 1; rulings §4.
func encodeNodeIdentify(svtnID [16]byte, pubkey ed25519.PublicKey) []byte {
	payload := make([]byte, nodeIdentifyPayloadSize)
	payload[0] = nodeIdentifyControlType
	payload[1] = frame.VersionByte
	payload[2] = msgKindNodeIdentify
	payload[3] = 0x00 // reserved
	copy(payload[4:36], pubkey)

	hdr := frame.EncodeOuterHeader(frame.OuterHeader{
		Version:    frame.VersionByte,
		FrameType:  frame.FrameTypeCtl,
		SVTNID:     svtnID,
		PayloadLen: uint16(nodeIdentifyPayloadSize),
	})
	raw := make([]byte, 0, frame.OuterHeaderSize+nodeIdentifyPayloadSize)
	raw = append(raw, hdr[:]...)
	raw = append(raw, payload...)
	return raw
}

// encodeChallenge assembles the 144-byte Challenge frame (44-byte outer
// header + 100-byte payload, msg_kind=0x02) from a router-generated
// admission.Challenge. Pure-core.
//
// Traces to BC-2.01.009 Postcondition 2; rulings §5.
func encodeChallenge(svtnID [16]byte, challenge admission.Challenge) []byte {
	payload := make([]byte, challengePayloadSize)
	payload[0] = nodeIdentifyControlType
	payload[1] = frame.VersionByte
	payload[2] = msgKindChallenge
	payload[3] = 0x00 // reserved
	copy(payload[4:36], challenge.Nonce[:])
	copy(payload[36:100], challenge.RouterSig)

	hdr := frame.EncodeOuterHeader(frame.OuterHeader{
		Version:    frame.VersionByte,
		FrameType:  frame.FrameTypeCtl,
		SVTNID:     svtnID,
		PayloadLen: uint16(challengePayloadSize),
	})
	raw := make([]byte, 0, frame.OuterHeaderSize+challengePayloadSize)
	raw = append(raw, hdr[:]...)
	raw = append(raw, payload...)
	return raw
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
	payload := make([]byte, challengeResponsePayloadSize)
	payload[0] = nodeIdentifyControlType
	payload[1] = frame.VersionByte
	payload[2] = msgKindChallengeResponse
	payload[3] = 0x00 // reserved
	copy(payload[4:68], resp.NonceSig)

	hdr := frame.EncodeOuterHeader(frame.OuterHeader{
		Version:    frame.VersionByte,
		FrameType:  frame.FrameTypeCtl,
		SVTNID:     svtnID,
		PayloadLen: uint16(challengeResponsePayloadSize),
	})
	raw := make([]byte, 0, frame.OuterHeaderSize+challengeResponsePayloadSize)
	raw = append(raw, hdr[:]...)
	raw = append(raw, payload...)
	return raw
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
	if len(payload) != nodeIdentifyPayloadSize {
		return nil, fmt.Errorf("node_identify: NodeIdentify payload size %d != %d", len(payload), nodeIdentifyPayloadSize)
	}
	if payload[0] != nodeIdentifyControlType {
		return nil, fmt.Errorf("node_identify: NodeIdentify control_type %#x != 0x04", payload[0])
	}
	if payload[1] != frame.VersionByte {
		return nil, fmt.Errorf("node_identify: NodeIdentify version %#x != 0x01", payload[1])
	}
	if payload[2] != msgKindNodeIdentify {
		return nil, fmt.Errorf("node_identify: NodeIdentify msg_kind %#x != 0x01", payload[2])
	}
	if payload[3] != 0x00 {
		return nil, fmt.Errorf("node_identify: NodeIdentify reserved byte %#x != 0x00", payload[3])
	}
	pubkey := make(ed25519.PublicKey, ed25519.PublicKeySize)
	copy(pubkey, payload[4:36])
	return pubkey, nil
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
	if len(payload) != challengeResponsePayloadSize {
		return admission.ChallengeResponse{}, fmt.Errorf("node_identify: ChallengeResponse payload size %d != %d", len(payload), challengeResponsePayloadSize)
	}
	if payload[0] != nodeIdentifyControlType {
		return admission.ChallengeResponse{}, fmt.Errorf("node_identify: ChallengeResponse control_type %#x != 0x04", payload[0])
	}
	if payload[1] != frame.VersionByte {
		return admission.ChallengeResponse{}, fmt.Errorf("node_identify: ChallengeResponse version %#x != 0x01", payload[1])
	}
	if payload[2] != msgKindChallengeResponse {
		return admission.ChallengeResponse{}, fmt.Errorf("node_identify: ChallengeResponse msg_kind %#x != 0x03", payload[2])
	}
	if payload[3] != 0x00 {
		return admission.ChallengeResponse{}, fmt.Errorf("node_identify: ChallengeResponse reserved byte %#x != 0x00", payload[3])
	}
	resp := admission.ChallengeResponse{
		NonceSig: make([]byte, ed25519.SignatureSize),
	}
	copy(resp.NonceSig, payload[4:68])
	return resp, nil
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
	// Set 10s deadline for the entire three-message exchange (rulings §13;
	// E-ADM-022). Cleared on success below.
	if err = conn.SetDeadline(time.Now().Add(nodeIdentifyHandshakeTimeout)); err != nil {
		_ = conn.Close()
		return [16]byte{}, [8]byte{}, fmt.Errorf("node_identify: SetDeadline: %w", err)
	}

	// Message 1: read NodeIdentify (80 bytes = 44 header + 36 payload).
	hdr, payload, err := frame.ReadOuterFrame(conn)
	if err != nil {
		_ = conn.Close()
		return [16]byte{}, [8]byte{}, fmt.Errorf("node_identify: reading NodeIdentify: %w", err)
	}

	// Validate outer header payload size against expected NodeIdentify size.
	if hdr.PayloadLen != nodeIdentifyPayloadSize {
		_ = conn.Close()
		return [16]byte{}, [8]byte{}, fmt.Errorf("node_identify: malformed NodeIdentify frame: payload_len=%d want %d", hdr.PayloadLen, nodeIdentifyPayloadSize)
	}

	// Decode NodeIdentify payload (also validates field values).
	pubkey, err := decodeNodeIdentify(payload)
	if err != nil {
		_ = conn.Close()
		return [16]byte{}, [8]byte{}, err
	}

	// Validate SVTNID non-zero (rulings §9; zero SVTN ID is explicitly rejected).
	svtnID = hdr.SVTNID
	var zeroSVTN [16]byte
	if svtnID == zeroSVTN {
		_ = conn.Close()
		return [16]byte{}, [8]byte{}, fmt.Errorf("node_identify: zero SVTN ID rejected")
	}

	// Derive node address from (svtnID, pubkey).
	nodeAddr = frame.DeriveNodeAddress(svtnID, []byte(pubkey))

	// Generate challenge (router signs the nonce with its private key).
	challenge, err := admission.GenerateChallenge(routerPrivKey)
	if err != nil {
		_ = conn.Close()
		return [16]byte{}, [8]byte{}, fmt.Errorf("node_identify: GenerateChallenge: %w", err)
	}

	// Message 2: send Challenge (144 bytes = 44 header + 100 payload).
	challengeFrame := encodeChallenge(svtnID, challenge)
	if _, err = conn.Write(challengeFrame); err != nil {
		_ = conn.Close()
		return [16]byte{}, [8]byte{}, fmt.Errorf("node_identify: sending Challenge: %w", err)
	}

	// Message 3: read ChallengeResponse (112 bytes = 44 header + 68 payload).
	crHdr, crPayload, err := frame.ReadOuterFrame(conn)
	if err != nil {
		_ = conn.Close()
		return [16]byte{}, [8]byte{}, fmt.Errorf("node_identify: reading ChallengeResponse: %w", err)
	}

	// Validate outer header payload size against expected ChallengeResponse size.
	if crHdr.PayloadLen != challengeResponsePayloadSize {
		_ = conn.Close()
		return [16]byte{}, [8]byte{}, fmt.Errorf("node_identify: malformed ChallengeResponse: payload_len=%d want %d", crHdr.PayloadLen, challengeResponsePayloadSize)
	}

	// Decode ChallengeResponse payload.
	resp, err := decodeChallengeResponse(crPayload)
	if err != nil {
		_ = conn.Close()
		return [16]byte{}, [8]byte{}, err
	}

	// Admit the node: verify signature, check nonce, check revocation/expiry.
	if err = admission.AdmitNode(challenge, resp, pubkey, svtnID, ks); err != nil {
		_ = conn.Close()
		return [16]byte{}, [8]byte{}, err
	}

	// Handshake succeeded: record binding and clear deadline.
	r.BindInterface(svtnID, nodeAddr, h.IfaceID)
	if err = conn.SetDeadline(time.Time{}); err != nil {
		// Clearing the deadline failed: treat as a connection error.
		_ = conn.Close()
		r.UnbindInterface(svtnID, nodeAddr, h.IfaceID)
		return [16]byte{}, [8]byte{}, fmt.Errorf("node_identify: clearing deadline: %w", err)
	}

	return svtnID, nodeAddr, nil
}
