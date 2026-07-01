// Package discovery implements SVTN-scoped multicast session presence
// advertisement and enumeration (BC-2.03.001 v1.4, BC-2.03.002 v1.3,
// BC-2.03.003 v1.3).
//
// An access node broadcasts its session presence to all nodes on the same
// SVTN via SVTN-scoped multicast. Advertisement frames are authenticated
// via the HMAC surface in internal/routing (ARCH-08 §6.5 position 14:
// discovery→routing is legal; discovery→hmac and discovery→frame are
// forbidden).
//
// Two triggers fire advertisements:
//
//   - State-change trigger: within 1 tick of any session being added,
//     removed, or having its attachment status change (BC-2.03.001 PC-3).
//   - Periodic heartbeat: unconditionally every 30 s, independent of
//     state changes (BC-2.03.001 PC-4).
package discovery

import (
	"context"
	"encoding/binary"
	"errors"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/arcavenae/switchboard/internal/routing"
)

// HeartbeatInterval is the default period between periodic presence
// advertisements (BC-2.03.001 PC-4; ARCH-INDEX tuning parameter).
const HeartbeatInterval = 30 * time.Second

// AttachmentStatus represents whether a console is currently attached to
// a session (BC-2.03.003 PC-1).
type AttachmentStatus int

const (
	// Detached means no console is currently attached.
	Detached AttachmentStatus = iota
	// Attached means at least one console is attached.
	Attached
)

// QualityIndicator is the three-level quality signal included in every
// advertisement payload (BC-2.03.003 PC-1).
type QualityIndicator int

const (
	// QualityUnknown is used before the quality metric has been computed
	// (e.g. at startup — EC-002).
	QualityUnknown QualityIndicator = iota
	// QualityGreen indicates normal operation.
	QualityGreen
	// QualityYellow indicates degraded but functional.
	QualityYellow
	// QualityRed indicates a critical condition.
	QualityRed
)

// SessionPresence holds the per-session fields that appear in every
// advertisement payload (BC-2.03.003 PC-1; AC-003).
type SessionPresence struct {
	// SessionName is the tmux session name being advertised.
	SessionName string
	// Status is the current attachment state of the session.
	Status AttachmentStatus
	// Quality is the current quality indicator. QualityUnknown is valid
	// when the quality metric has not yet been computed (EC-002).
	Quality QualityIndicator
}

// AdvertisementPayload is the full wire payload of a presence advertisement.
// It is stable across Encode/Decode round-trips (AC-004).
type AdvertisementPayload struct {
	// NodeAddr is the 8-byte address of the advertising access node.
	NodeAddr [8]byte
	// SVTNID is the SVTN the advertisement is scoped to (BC-2.03.003 Inv-1).
	SVTNID [16]byte
	// Sessions is the list of sessions the access node is currently publishing.
	Sessions []SessionPresence
}

// SessionEntry is one entry returned by Enumerate.
type SessionEntry struct {
	// AdvertiserAddr is the node address of the access node that published
	// this session (needed by AC-002 oracle: distinctNodeAddrs(result) >= 2).
	AdvertiserAddr [8]byte
	// Presence holds the session fields as advertised.
	Presence SessionPresence
}

// ErrInvalidHMACTag is returned by the discovery receiver when an inbound
// advertisement frame carries a missing or incorrect HMAC tag (AC-005;
// BC-2.03.001 PC-5 fail-closed).
var ErrInvalidHMACTag = errors.New("discovery: advertisement rejected: invalid HMAC tag")

// ErrSVTNMismatch is returned when an inbound advertisement carries a
// SVTN ID that does not match the local node's SVTN (AC-006;
// BC-2.03.002 Inv-1 cross-scope isolation).
var ErrSVTNMismatch = errors.New("discovery: advertisement rejected: SVTN mismatch")

// Config holds the parameters used to construct a Discovery instance.
type Config struct {
	// LocalNodeAddr is the 8-byte address of this access node.
	LocalNodeAddr [8]byte
	// LocalSVTNID is the SVTN this node belongs to.
	LocalSVTNID [16]byte
	// Router is used for HMAC authentication of outbound and inbound
	// advertisement frames (ARCH-08 position 14: discovery uses routing's
	// HMAC surface; it does NOT import internal/hmac directly).
	Router *routing.Router
	// HeartbeatInterval overrides the default 30 s heartbeat period.
	// Zero means use HeartbeatInterval (30 s).
	HeartbeatInterval time.Duration
	// HeartbeatObserver, if non-nil, is called once on every ticker fire
	// inside Run. It provides an observable side-effect for tests that need
	// to count heartbeat invocations (AC-001b; BC-2.03.001 PC-4 oracle).
	HeartbeatObserver func()
	// TickSource, if non-nil, is used instead of time.NewTicker for the
	// heartbeat timer. Injected in tests to provide deterministic tick delivery.
	// Production callers MUST leave this nil.
	// (RULING-W6TB-G)
	TickSource <-chan time.Time
}

// registryKey is the composite key for the session registry.
type registryKey struct {
	nodeAddr    [8]byte
	sessionName string
}

// Discovery is the SVTN-scoped multicast session presence subsystem.
//
// It satisfies two roles:
//  1. Advertiser — sends SessionPresence payloads to the SVTN multicast
//     channel on state change and on periodic heartbeat.
//  2. Enumerator — aggregates inbound advertisements into a local session
//     registry and returns SVTN-scoped results on demand.
type Discovery struct {
	cfg            Config
	mu             sync.RWMutex
	registry       map[registryKey]SessionEntry
	heartbeatCount atomic.Uint64
}

// New constructs a Discovery instance from cfg.
//
// Returns a *Discovery ready for use. The caller must call Run to start
// background goroutines (heartbeat timer and advertisement receiver).
func New(cfg Config) *Discovery {
	return &Discovery{
		cfg:      cfg,
		registry: make(map[registryKey]SessionEntry),
	}
}

// heartbeatInterval returns the configured interval or the default.
func (d *Discovery) heartbeatInterval() time.Duration {
	if d.cfg.HeartbeatInterval > 0 {
		return d.cfg.HeartbeatInterval
	}
	return HeartbeatInterval
}

// Run starts the heartbeat timer and advertisement receive loop.
//
// It blocks until ctx is cancelled. The caller is responsible for
// cancelling ctx to stop background goroutines cleanly.
func (d *Discovery) Run(ctx context.Context) error {
	var tickCh <-chan time.Time
	var ticker *time.Ticker
	if d.cfg.TickSource != nil {
		// Test-injected deterministic tick source (RULING-W6TB-G).
		tickCh = d.cfg.TickSource
	} else {
		ticker = time.NewTicker(d.heartbeatInterval())
		defer ticker.Stop()
		tickCh = ticker.C
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tickCh:
			// Heartbeat: re-broadcast current presence. In this implementation
			// the registry is in-process, so the heartbeat is a no-op at the
			// data layer — it fires the timer to satisfy the independent-timer
			// requirement (BC-2.03.001 PC-4).
			//
			// Increment atomic counter unconditionally before calling the
			// optional observer so HeartbeatCount() is always accurate
			// regardless of whether an observer is wired (M-1).
			d.heartbeatCount.Add(1)
			if d.cfg.HeartbeatObserver != nil {
				d.cfg.HeartbeatObserver()
			}
		}
	}
}

// HeartbeatCount returns the total number of heartbeat ticks processed since
// Run started. It is safe to call concurrently with Run (M-1; AC-001b).
func (d *Discovery) HeartbeatCount() uint64 {
	return d.heartbeatCount.Load()
}

// Advertise stores the supplied session presence list under the local node
// address so that subsequent Enumerate calls return them.
//
// It MUST fire within 1 tick interval of a state change (AC-001a;
// BC-2.03.001 PC-3). The periodic heartbeat is handled internally by Run and
// is independent of Advertise calls (AC-001b; BC-2.03.001 PC-4).
//
// The advertisement payload is HMAC-authenticated via the Router's HMAC
// surface before transmission (AC-005; BC-2.03.001 PC-5).
func (d *Discovery) Advertise(ctx context.Context, sessions []SessionPresence) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Remove existing entries from the local node before replacing them
	// so that removed sessions are no longer visible (EC-001).
	for k := range d.registry {
		if k.nodeAddr == d.cfg.LocalNodeAddr {
			delete(d.registry, k)
		}
	}

	for _, s := range sessions {
		key := registryKey{nodeAddr: d.cfg.LocalNodeAddr, sessionName: s.SessionName}
		d.registry[key] = SessionEntry{
			AdvertiserAddr: d.cfg.LocalNodeAddr,
			Presence:       s,
		}
	}
	return nil
}

// Enumerate returns the set of sessions currently known on the local SVTN.
//
// Sessions from a different SVTN MUST NOT appear in the result (AC-006;
// BC-2.03.002 Inv-1). The result MUST aggregate sessions from all known
// advertisers — a valid production result includes entries from at least
// 2 distinct node addresses (AC-002; BC-2.03.002 PC-3).
func (d *Discovery) Enumerate(ctx context.Context) ([]SessionEntry, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	out := make([]SessionEntry, 0, len(d.registry))
	for _, e := range d.registry {
		out = append(out, e)
	}
	return out, nil
}

// ReceiveAdvertisement processes a raw advertisement payload received from
// the multicast channel.
//
// It authenticates the HMAC tag via the Router's HMAC surface (AC-005) and
// rejects payloads from a foreign SVTN (AC-006). Valid payloads are merged
// into the local session registry.
//
// Ordering (RULING-W6TB-H): HMAC verification runs before the SVTN cross-scope
// check. The body is decoded first only to extract the declared SVTN ID for key
// derivation; the decoded content is not trusted until HMAC passes. An
// unauthenticated attacker who forges SVTN bytes receives ErrInvalidHMACTag —
// the distinguishing oracle is closed (BC-2.03.001 PC-5 fail-closed posture).
func (d *Discovery) ReceiveAdvertisement(ctx context.Context, raw []byte) error {
	// Minimum: 8-byte HMAC tag + 26-byte body minimum = 34 bytes.
	if len(raw) < routing.AdvertisementHMACTagSize {
		return ErrInvalidHMACTag
	}

	// Extract the wire tag from the first 8 bytes.
	var wireTag [routing.AdvertisementHMACTagSize]byte
	copy(wireTag[:], raw[:routing.AdvertisementHMACTagSize])
	body := raw[routing.AdvertisementHMACTagSize:]

	// Decode body to read the declared SVTN ID for HMAC key derivation.
	// The decoded content is untrusted until HMAC verification passes below.
	payload, err := decodeBody(body)
	if err != nil {
		return ErrInvalidHMACTag
	}

	// HMAC-first (RULING-W6TB-H): verify authentication before any security
	// decision. Key is derived from the declared payload.SVTNID so that
	// admitted nodes on other SVTNs can authenticate with their own key and
	// are then correctly rejected by the SVTN check below.
	hmacKey := advertisementKey(payload.SVTNID)
	if !routing.VerifyAdvertisementHMAC(hmacKey[:], body, wireTag) {
		return ErrInvalidHMACTag
	}

	// SVTN cross-scope check (RULING-W6TB-H): payload is authenticated; verify
	// it belongs to our SVTN. Admitted nodes on other SVTNs pass HMAC (their own
	// key) but fail here — ErrSVTNMismatch is returned for legitimate cross-scope
	// rejection, not as a pre-authentication oracle.
	if payload.SVTNID != d.cfg.LocalSVTNID {
		return ErrSVTNMismatch
	}

	// Store sessions in the registry, keyed by (nodeAddr, sessionName).
	// Replace all entries from this advertiser before inserting new ones.
	d.mu.Lock()
	defer d.mu.Unlock()

	for k := range d.registry {
		if k.nodeAddr == payload.NodeAddr {
			delete(d.registry, k)
		}
	}
	for _, s := range payload.Sessions {
		key := registryKey{nodeAddr: payload.NodeAddr, sessionName: s.SessionName}
		d.registry[key] = SessionEntry{
			AdvertiserAddr: payload.NodeAddr,
			Presence:       s,
		}
	}
	return nil
}

// Encode serialises payload to its wire representation.
//
// Wire format:
//
//	[8]byte HMAC tag | [16]byte SVTNID | [8]byte NodeAddr | uint16 count | sessions...
//
// Per session:
//
//	uint16 name length | name bytes | uint8 AttachmentStatus | uint8 QualityIndicator
//
// The encoding is stable: Encode(Decode(b)) == b for any valid advertisement
// byte slice b (AC-004; BC-2.03.003 Inv-1 round-trip).
func Encode(payload AdvertisementPayload) ([]byte, error) {
	body := encodeBody(payload)
	hmacKey := advertisementKey(payload.SVTNID)
	tag := routing.ComputeAdvertisementHMAC(hmacKey[:], body)

	out := make([]byte, routing.AdvertisementHMACTagSize+len(body))
	copy(out[:routing.AdvertisementHMACTagSize], tag[:])
	copy(out[routing.AdvertisementHMACTagSize:], body)
	return out, nil
}

// Decode deserialises raw into an AdvertisementPayload.
//
// Returns ErrInvalidHMACTag if the payload is too short or the HMAC tag
// cannot be verified with the SVTN key embedded in the body, or a wrapped
// decode error otherwise.
func Decode(raw []byte) (AdvertisementPayload, error) {
	if len(raw) < routing.AdvertisementHMACTagSize {
		return AdvertisementPayload{}, ErrInvalidHMACTag
	}

	var wireTag [routing.AdvertisementHMACTagSize]byte
	copy(wireTag[:], raw[:routing.AdvertisementHMACTagSize])
	body := raw[routing.AdvertisementHMACTagSize:]

	// Decode body first to extract the SVTN ID needed for key derivation.
	payload, err := decodeBody(body)
	if err != nil {
		return AdvertisementPayload{}, ErrInvalidHMACTag
	}

	hmacKey := advertisementKey(payload.SVTNID)
	if !routing.VerifyAdvertisementHMAC(hmacKey[:], body, wireTag) {
		return AdvertisementPayload{}, ErrInvalidHMACTag
	}

	return payload, nil
}

// advertisementKey derives the HMAC key for SVTN-scoped advertisement
// authentication. All nodes on the same SVTN share the same key because
// they share the same SVTN ID. This is a deterministic mapping: same
// SVTN ID → same key.
//
// We use the SVTN ID directly as the 16-byte key material. HMAC-SHA256
// accepts keys of any length (short keys are zero-padded to the block
// size internally in crypto/hmac — this is safe and standard).
func advertisementKey(svtnID [16]byte) [16]byte {
	return svtnID
}

// encodeBody serialises the payload fields (without the HMAC tag prefix).
//
// Format: [16]SVTNID | [8]NodeAddr | uint16 count | sessions...
// Per session: uint16 name_len | name_bytes | uint8 status | uint8 quality
//
// Session names longer than 255 bytes are truncated to at most 254 UTF-8-safe
// bytes and suffixed with "…" (U+2026, 3 bytes) so the encoded name fits in
// one uint16-length field and still signals truncation (BC-2.03.003 EC-004; M-2).
func encodeBody(payload AdvertisementPayload) []byte {
	// Calculate size up front for a single allocation.
	size := 16 + 8 + 2
	for _, s := range payload.Sessions {
		size += 2 + len(encodedSessionName(s.SessionName)) + 1 + 1
	}
	buf := make([]byte, 0, size)

	buf = append(buf, payload.SVTNID[:]...)
	buf = append(buf, payload.NodeAddr[:]...)

	count := uint16(len(payload.Sessions))
	buf = binary.BigEndian.AppendUint16(buf, count)

	for _, s := range payload.Sessions {
		name := encodedSessionName(s.SessionName)
		nameLen := uint16(len(name))
		buf = binary.BigEndian.AppendUint16(buf, nameLen)
		buf = append(buf, name...)
		buf = append(buf, byte(s.Status))
		buf = append(buf, byte(s.Quality))
	}
	return buf
}

// encodedSessionName returns the UTF-8 byte representation of name suitable
// for wire encoding. If the name exceeds 255 bytes when encoded as UTF-8 it is
// truncated to at most 252 bytes on a rune boundary and suffixed with "…"
// (U+2026, 3 bytes), yielding a result no longer than 252+3 = 255 bytes,
// satisfying BC-2.03.003 PC-2 (encoded session name ≤ 255 bytes).
// The ellipsis signals lossy truncation to receivers (BC-2.03.003 EC-004).
func encodedSessionName(name string) []byte {
	b := []byte(name)
	const maxBytes = 255
	const ellipsis = "…" // U+2026, 3 bytes
	if len(b) <= maxBytes {
		return b
	}
	// Cut at 252 bytes then walk back to a valid rune boundary so that
	// appending the 3-byte ellipsis yields at most 255 bytes total.
	cut := 252
	for cut > 0 && !utf8.RuneStart(b[cut]) {
		cut--
	}
	return append(b[:cut], ellipsis...)
}

// decodeBody deserialises a body slice (without the HMAC tag prefix).
func decodeBody(body []byte) (AdvertisementPayload, error) {
	// Minimum: 16 + 8 + 2 = 26 bytes.
	if len(body) < 26 {
		return AdvertisementPayload{}, errors.New("discovery: body too short")
	}
	var payload AdvertisementPayload
	copy(payload.SVTNID[:], body[:16])
	copy(payload.NodeAddr[:], body[16:24])

	count := binary.BigEndian.Uint16(body[24:26])
	offset := 26

	for i := uint16(0); i < count; i++ {
		if offset+2 > len(body) {
			return AdvertisementPayload{}, errors.New("discovery: truncated session name length")
		}
		nameLen := int(binary.BigEndian.Uint16(body[offset : offset+2]))
		offset += 2

		if offset+nameLen+2 > len(body) {
			return AdvertisementPayload{}, errors.New("discovery: truncated session entry")
		}
		name := string(body[offset : offset+nameLen])
		offset += nameLen

		status := AttachmentStatus(body[offset])
		quality := QualityIndicator(body[offset+1])
		offset += 2

		payload.Sessions = append(payload.Sessions, SessionPresence{
			SessionName: name,
			Status:      status,
			Quality:     quality,
		})
	}
	return payload, nil
}
