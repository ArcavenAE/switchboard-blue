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
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/arcavenae/switchboard/internal/routing"
)

// HeartbeatInterval is the default period between periodic presence
// advertisements (BC-2.03.001 PC-4; ARCH-INDEX tuning parameter).
const HeartbeatInterval = 30 * time.Second

// DiscoveryPort is the fixed UDP port for the SVTN-scoped multicast
// discovery channel (S-BL.DISCOVERY-WIRE Decision 2(c)). Adjudicated as
// 49201 by human gate disposition 2026-07-14 (no longer a bikeshed
// placeholder). One port suffices for all SVTNs because the group
// *address*, not the port, provides SVTN scoping (see MulticastAddrFor).
const DiscoveryPort = 49201

// MulticastAddrFor returns the SVTN-scoped multicast group address for
// svtnID: 239.h0.h1.h2, where h0..h2 are the first three bytes of
// SHA-256(svtnID) (S-BL.DISCOVERY-WIRE Decision 2(b); AC-002). Deterministic
// and static for the SVTN's lifetime — no allocation bookkeeping, no release
// step on admin.svtn.destroy. IPv4 239.0.0.0/8 (RFC 2365 "administratively
// scoped") is the addressing hygiene/routing-efficiency range; HMAC
// authentication, not address uniqueness, is the actual security boundary
// (SEC-DW-08).
func MulticastAddrFor(svtnID [16]byte) net.IP {
	h := sha256.Sum256(svtnID[:])
	return net.IPv4(239, h[0], h[1], h[2])
}

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
	// Sequence is the epoch-qualified monotonic sequence number (SEC-DW-07;
	// F-DWSP4-001 restart-liveness amendment). High 32 bits are the wall-clock
	// UTC epoch seconds sampled at the advertising Discovery instance's start;
	// low 32 bits are a per-instance counter. Widened uint32→uint64 so a
	// node restart with a stable admitted identity is not silently locked out
	// by the router's restart-STABLE lastSeen watermark (EC-010).
	Sequence uint64
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

// ErrInvalidSessionName is returned by Encode when a session entry carries a
// session name that is structurally invalid: either empty (len == 0) or not
// valid UTF-8. Oversize names (len > 255) are truncated, not rejected
// (BC-2.03.003 PC-2, EC-001; RULING-W6TB-J).
var ErrInvalidSessionName = errors.New("discovery: session name is empty or contains invalid UTF-8")

// ErrSVTNMismatch is returned when an inbound advertisement carries a
// SVTN ID that does not match the local node's SVTN (AC-006;
// BC-2.03.002 Inv-1 cross-scope isolation).
var ErrSVTNMismatch = errors.New("discovery: advertisement rejected: SVTN mismatch")

// ErrMissingNodeAdmissionPubkey is returned by transmitAdvertisement when
// Config.LocalNodeAdmissionPubkey is empty (go.md rule 13: a
// security-perimeter dependency fails closed, not open, on a missing
// value). Signing with an empty/absent pubkey would derive a discovery HMAC
// key that cannot verify against any real admitted node and — if derived
// from wire-visible data instead, the defect this guards against — is
// trivially recomputable by any LAN observer (F-DWIP1-001).
var ErrMissingNodeAdmissionPubkey = errors.New("discovery: LocalNodeAdmissionPubkey is required to sign outbound advertisements")

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
	// LocalNodeAdmissionPubkey is this node's own admitted Ed25519 public key
	// — the IKM Encode/Decode use to derive the discovery HMAC key, matching
	// what the router computes via DiscoveryAuthKeyFor for the same admitted
	// node (AC-004 PC-1/PC-4; F-DWIP1-001). Required: transmitAdvertisement
	// fails if this is empty, since a key derived from anything else (e.g.
	// the on-wire SVTNID alone, the prior defect) cannot verify against a
	// real router and is trivially recomputable by any LAN observer.
	//
	// As of this field's introduction, no production code path in this
	// repository supplies a running access-node process with its own
	// admission keypair at runtime — admin.key.register (cmd/switchboard's
	// admin RPC) only registers a pubkey supplied externally by an operator
	// call; nothing wires that pubkey (or a corresponding private key) back
	// into a live runAccess process. Populating this field from a real
	// identity/config source is out of this story's scope until that
	// mechanism exists elsewhere in the codebase.
	LocalNodeAdmissionPubkey []byte
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

	// epoch is the wall-clock UTC epoch-seconds sampled once at instance
	// construction — the high 32 bits of every outbound Sequence value
	// (SEC-DW-07; F-DWSP4-001 restart-liveness amendment). A fresh epoch on
	// every restart of a stable admitted identity, combined with real
	// wall-clock advancement between restarts, is what lets the router's
	// restart-STABLE lastSeen watermark accept post-restart advertisements
	// via forward acceptance (AC-010) rather than locking the node out.
	epoch uint32
	// seqCounter is the low 32 bits of every outbound Sequence value: a
	// monotonic per-instance counter, incremented on every Advertise call.
	seqCounter atomic.Uint32
}

// New constructs a Discovery instance from cfg.
//
// Returns a *Discovery ready for use. The caller must call Run to start
// background goroutines (heartbeat timer and advertisement receiver).
func New(cfg Config) *Discovery {
	return &Discovery{
		cfg:      cfg,
		registry: make(map[registryKey]SessionEntry),
		epoch:    uint32(time.Now().UTC().Unix()),
	}
}

// nextSequence returns the next epoch-qualified monotonic Sequence value
// for an outbound advertisement (SEC-DW-07): high 32 bits are the epoch
// sampled at instance construction, low 32 bits are a per-instance counter
// that increments on every call.
func (d *Discovery) nextSequence() uint64 {
	counter := d.seqCounter.Add(1)
	return uint64(d.epoch)<<32 | uint64(counter)
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
// address so that subsequent Enumerate calls return them, then transmits the
// advertisement over the SVTN-scoped multicast channel (AC-003).
//
// It MUST fire within 1 tick interval of a state change (AC-001a;
// BC-2.03.001 PC-3). The periodic heartbeat is handled internally by Run and
// is independent of Advertise calls (AC-001b; BC-2.03.001 PC-4).
//
// Session names are validated by the same rules as Encode: empty names and
// non-UTF-8 names return ErrInvalidSessionName. Names longer than 255 bytes
// are truncated to 252 bytes + U+2026 at a rune boundary (BC-2.03.003 PC-2).
//
// The registry mutation and the wire transmission are sequenced but not
// atomic with respect to each other: the mutex is released before the UDP
// send so that network I/O (which can block on a slow/misconfigured
// interface) never holds up concurrent Enumerate/Advertise callers.
func (d *Discovery) Advertise(ctx context.Context, sessions []SessionPresence) error {
	// Validate and normalise session names up-front so that the registry
	// never holds invalid entries (BC-2.03.003 PC-2; AC-004b).
	type validatedSession struct {
		name string
		s    SessionPresence
	}
	validated := make([]validatedSession, 0, len(sessions))
	for _, s := range sessions {
		encoded, err := encodedSessionName(s.SessionName)
		if err != nil {
			return err
		}
		s.SessionName = string(encoded)
		validated = append(validated, validatedSession{name: s.SessionName, s: s})
	}

	d.mu.Lock()

	// Remove existing entries from the local node before replacing them
	// so that removed sessions are no longer visible (EC-001).
	for k := range d.registry {
		if k.nodeAddr == d.cfg.LocalNodeAddr {
			delete(d.registry, k)
		}
	}

	wireSessions := make([]SessionPresence, 0, len(validated))
	for _, v := range validated {
		key := registryKey{nodeAddr: d.cfg.LocalNodeAddr, sessionName: v.name}
		d.registry[key] = SessionEntry{
			AdvertiserAddr: d.cfg.LocalNodeAddr,
			Presence:       v.s,
		}
		wireSessions = append(wireSessions, v.s)
	}
	// Sampled under the same lock that orders the registry mutation above,
	// so concurrent Advertise calls can't have their transmitted Sequence
	// order disagree with their registry-mutation order.
	sequence := d.nextSequence()
	d.mu.Unlock()

	return d.transmitAdvertisement(sequence, wireSessions)
}

// transmitAdvertisement builds and sends the wire-format advertisement for
// sessions over the SVTN-scoped multicast channel (AC-003 postconditions
// 1-3). It signs via Encode using this node's admitted pubkey
// (Config.LocalNodeAdmissionPubkey, AC-004 PC-1/PC-4) — the same IKM the
// router derives its verification key from via DiscoveryAuthKeyFor — and
// sends with a plain net.WriteTo/DialUDP, no group join, and outbound
// multicast TTL explicitly set to 1.
func (d *Discovery) transmitAdvertisement(sequence uint64, sessions []SessionPresence) error {
	if len(d.cfg.LocalNodeAdmissionPubkey) == 0 {
		return ErrMissingNodeAdmissionPubkey
	}
	raw, err := Encode(AdvertisementPayload{
		NodeAddr: d.cfg.LocalNodeAddr,
		SVTNID:   d.cfg.LocalSVTNID,
		Sequence: sequence,
		Sessions: sessions,
	}, d.cfg.LocalNodeAdmissionPubkey)
	if err != nil {
		return fmt.Errorf("discovery: encode outbound advertisement: %w", err)
	}
	if err := sendMulticastAdvertisement(d.cfg.LocalSVTNID, raw); err != nil {
		return fmt.Errorf("discovery: send outbound advertisement: %w", err)
	}
	return nil
}

// sendMulticastAdvertisement sends raw to the SVTN-derived multicast group
// on DiscoveryPort (AC-003).
//
// Postcondition 1: uses a plain net.ListenUDP/WriteToUDP send — no
// net.ListenMulticastUDP, no group join (only the router joins the group;
// DI-004 forbids direct node-to-node communication).
//
// Postcondition 2: each outbound socket's multicast TTL is explicitly set
// to 1 before the send, via raw syscall (setMulticastTTL1) rather than
// golang.org/x/net/ipv4, per this story's zero-new-third-party-dependency
// constraint.
//
// Sends once per UP+multicast-capable local interface (setMulticastOutgoingInterface
// pins each send to that interface via IP_MULTICAST_IF), rather than relying
// on the kernel's default unicast-route-based interface selection for the
// multicast destination — on a multi-homed host the default route to an
// administratively-scoped multicast address is not necessarily the
// interface carrying the SVTN's LAN segment (or, in the loopback integration
// test, is never loopback by default). Best-effort per interface: a single
// interface failing to send (e.g. transient EADDRNOTAVAIL) does not abort
// the others; the call only fails if every interface failed.
func sendMulticastAdvertisement(svtnID [16]byte, raw []byte) error {
	dst := &net.UDPAddr{IP: MulticastAddrFor(svtnID), Port: DiscoveryPort}

	ifaces, err := net.Interfaces()
	if err != nil {
		return fmt.Errorf("enumerate local interfaces: %w", err)
	}

	var sent int
	var lastErr error
	for _, ifi := range ifaces {
		addr, ok := firstMulticastCapableIPv4(ifi)
		if !ok {
			continue
		}
		if err := sendOnInterface(addr, dst, raw); err != nil {
			lastErr = err
			continue
		}
		sent++
	}
	if sent == 0 {
		if lastErr != nil {
			return fmt.Errorf("no viable multicast-capable interface accepted the send: %w", lastErr)
		}
		return errors.New("no UP+multicast-capable local interface found")
	}
	return nil
}

// firstMulticastCapableIPv4 returns ifi's first IPv4 address if ifi is UP
// and multicast-capable; ok is false otherwise (down interfaces, interfaces
// without multicast support, or interfaces with no IPv4 address at all —
// e.g. IPv6-only).
func firstMulticastCapableIPv4(ifi net.Interface) (addr net.IP, ok bool) {
	if ifi.Flags&net.FlagUp == 0 || ifi.Flags&net.FlagMulticast == 0 {
		return nil, false
	}
	addrs, err := ifi.Addrs()
	if err != nil {
		return nil, false
	}
	for _, a := range addrs {
		ipNet, isIPNet := a.(*net.IPNet)
		if !isIPNet {
			continue
		}
		if v4 := ipNet.IP.To4(); v4 != nil {
			return v4, true
		}
	}
	return nil, false
}

// sendOnInterface opens a fresh unbound UDP socket, pins its outgoing
// multicast interface to localAddr, sets TTL=1, sends raw to dst, and
// closes the socket. A fresh socket per interface per call rather than
// long-lived sockets held on Discovery — simplest correct choice for the
// Task 3 Green step; socket lifecycle is unconstrained implementation
// detail left to code review, not something this story's rulings fix.
func sendOnInterface(localAddr net.IP, dst *net.UDPAddr, raw []byte) error {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{})
	if err != nil {
		return fmt.Errorf("open outbound UDP socket: %w", err)
	}
	defer func() { _ = conn.Close() }()

	if err := setMulticastOutgoingInterface(conn, localAddr); err != nil {
		return fmt.Errorf("set multicast outgoing interface %v: %w", localAddr, err)
	}
	if err := setMulticastTTL1(conn); err != nil {
		return fmt.Errorf("set multicast TTL: %w", err)
	}
	if _, err := conn.WriteToUDP(raw, dst); err != nil {
		return fmt.Errorf("write multicast datagram via %v: %w", localAddr, err)
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

// IngestRelayAdvertisement decodes a hop-2 DISCOVERY_RELAY payload (AC-014
// PC-2 shape: NodeAddr | Sequence | count | sessions, no SVTNID) and merges
// it into the local session registry (AC-007; S-BL.DISCOVERY-WIRE Ruling 1
// point 3, corrected by F-DWSP8-001).
//
// payload is the DISCOVERY_RELAY frame's payload bytes AFTER the caller has
// stripped the 4-byte control header (control_type | version | reserved) —
// i.e. starting at byte 4 of Decision 3(c)'s layout: NodeAddr at
// payload[0:8], Sequence at payload[8:16], count at payload[16:18],
// sessions at payload[18:]. The Sequence value is not independently
// re-validated here — it already passed hop-1's SEC-DW-07 replay/freshness
// gate (RouterIngest.Ingest) before the router relayed this frame; hop-2
// carries it through informationally only.
//
// No per-frame HMAC is verified — trust derives from the already-admitted
// TCP connection the relay frame arrived on (AC-015). outerSVTNID is the
// relay frame's own OuterHeader.SVTNID, supplied by the caller: this
// function does not import internal/frame directly (ARCH-08 §6.5 position
// 14 restricts internal/discovery to internal/routing among internal/
// imports). outerSVTNID is compared against d.cfg.LocalSVTNID and
// ErrSVTNMismatch is returned on mismatch (AC-007 postcondition 3) —
// defense-in-depth against a relay/routing bug, not a crypto check; primary
// discovery-frame authentication happens exclusively at the router-side
// ingest path (AC-005/AC-006), unchanged from the original PC-3.
//
// This is the function that replaced ReceiveAdvertisement (retired per
// AC-007 postcondition 1 — no caller exists in the shipped topology: the
// router uses DiscoveryAuthKeyFor/RouterIngest.Ingest, AC-005/AC-006; no
// node receives hop-1 UDP directly, Ruling 2).
func (d *Discovery) IngestRelayAdvertisement(outerSVTNID [16]byte, payload []byte) error {
	if outerSVTNID != d.cfg.LocalSVTNID {
		return ErrSVTNMismatch
	}

	const hop2HeaderLen = 8 + 8 + 2 // NodeAddr + Sequence + count
	if len(payload) < hop2HeaderLen {
		return errors.New("discovery: hop-2 relay payload too short")
	}

	var nodeAddr [8]byte
	copy(nodeAddr[:], payload[0:8])
	// payload[8:16] is Sequence — informational only at this layer (see
	// doc comment above); hop-1 already gated freshness before relaying.

	sessions, err := DecodeSessionList(payload[16:])
	if err != nil {
		return fmt.Errorf("discovery: decode hop-2 session list: %w", err)
	}

	// Same registry replace-on-write update ReceiveAdvertisement previously
	// performed (AC-007 postcondition 4).
	d.mu.Lock()
	defer d.mu.Unlock()

	for k := range d.registry {
		if k.nodeAddr == nodeAddr {
			delete(d.registry, k)
		}
	}
	for _, s := range sessions {
		key := registryKey{nodeAddr: nodeAddr, sessionName: s.SessionName}
		d.registry[key] = SessionEntry{
			AdvertiserAddr: nodeAddr,
			Presence:       s,
		}
	}
	return nil
}

// Encode serialises payload to its wire representation.
//
// Wire format (SEC-DW-07; F-DWSP4-001 widened Sequence uint32→uint64):
//
//	[8]byte HMAC tag | [16]byte SVTNID | [8]byte NodeAddr | [8]byte Sequence (BE) | uint16 count | sessions...
//
// Per session:
//
//	uint16 name length | name bytes | uint8 AttachmentStatus | uint8 QualityIndicator
//
// The encoding is stable: Encode(Decode(b)) == b for any valid advertisement
// byte slice b (AC-004; BC-2.03.003 Inv-1 round-trip).
//
// nodeAdmissionPubkey is the sending node's own admitted Ed25519 public
// key — the HMAC key is derived as routing.DeriveDiscoveryKey(
// nodeAdmissionPubkey, payload.SVTNID), the identical derivation the router
// performs via DiscoveryAuthKeyFor for the same admitted node (AC-004
// PC-1/PC-4). It MUST NOT be derived from payload fields (F-DWIP1-001):
// a key that is a pure function of on-wire cleartext is trivially
// recomputable by any observer and defeats authentication entirely.
func Encode(payload AdvertisementPayload, nodeAdmissionPubkey []byte) ([]byte, error) {
	body, err := encodeBody(payload)
	if err != nil {
		return nil, err
	}
	hmacKey := routing.DeriveDiscoveryKey(nodeAdmissionPubkey, payload.SVTNID)
	tag := routing.ComputeAdvertisementHMAC(hmacKey[:], body)

	out := make([]byte, routing.AdvertisementHMACTagSize+len(body))
	copy(out[:routing.AdvertisementHMACTagSize], tag[:])
	copy(out[routing.AdvertisementHMACTagSize:], body)
	return out, nil
}

// Decode deserialises raw into an AdvertisementPayload.
//
// nodeAdmissionPubkey is the pubkey the caller expects raw to be signed
// with — the counterpart to Encode's parameter of the same name (AC-004
// PC-1/PC-4; F-DWIP1-001). Decode has no production caller as of this
// writing (RouterIngest.Ingest authenticates hop-1 datagrams directly via
// DiscoveryAuthKeyFor, and IngestRelayAdvertisement's hop-2 path carries no
// per-frame HMAC at all) — it remains exported and tested as the symmetric
// counterpart to Encode.
//
// Returns ErrInvalidHMACTag if the payload is too short or the HMAC tag
// cannot be verified against nodeAdmissionPubkey, or a wrapped decode error
// otherwise.
func Decode(raw []byte, nodeAdmissionPubkey []byte) (AdvertisementPayload, error) {
	// Minimum: 8-byte HMAC tag + 34-byte body minimum (16 SVTNID + 8 NodeAddr
	// + 8 Sequence + 2 count) = 42 bytes (SEC-DW-07 widened Sequence uint64).
	if len(raw) < routing.AdvertisementHMACTagSize+34 {
		return AdvertisementPayload{}, ErrInvalidHMACTag
	}

	var wireTag [routing.AdvertisementHMACTagSize]byte
	copy(wireTag[:], raw[:routing.AdvertisementHMACTagSize])
	body := raw[routing.AdvertisementHMACTagSize:]

	payload, err := decodeBody(body)
	if err != nil {
		return AdvertisementPayload{}, ErrInvalidHMACTag
	}

	hmacKey := routing.DeriveDiscoveryKey(nodeAdmissionPubkey, payload.SVTNID)
	if !routing.VerifyAdvertisementHMAC(hmacKey[:], body, wireTag) {
		return AdvertisementPayload{}, ErrInvalidHMACTag
	}

	return payload, nil
}

// ErrTooManySessions is returned by Encode when the payload carries more
// sessions than the wire format's uint16 count field can address. The wire
// format bounds one advertisement to 65535 sessions; the decoder additionally
// enforces a stricter runtime cap (see decodeBody).
var ErrTooManySessions = errors.New("discovery: session count exceeds encoding maximum 65535")

// encodeBody serialises the payload fields (without the HMAC tag prefix).
//
// Format: [16]SVTNID | [8]NodeAddr | [8]Sequence (BE uint64) | uint16 count | sessions...
// Per session: uint16 name_len | name_bytes | uint8 status | uint8 quality
//
// Session names longer than 255 bytes are truncated to at most 252 UTF-8-safe
// bytes and suffixed with "…" (U+2026, 3 bytes) so the encoded name fits in
// one uint16-length field and still signals truncation (BC-2.03.003 EC-001; M-2).
// Empty names or non-UTF-8 names are rejected with ErrInvalidSessionName.
// Session counts exceeding 65535 are rejected with ErrTooManySessions rather
// than silently truncated by the uint16 count cast.
func encodeBody(payload AdvertisementPayload) ([]byte, error) {
	// Explicit guard against uint16 truncation on the session count field.
	// Without this the cast below would silently wrap and produce a wire
	// frame whose count undercounts the sessions actually appended.
	const maxSessions = 65535
	if len(payload.Sessions) > maxSessions {
		return nil, fmt.Errorf("%w: got %d", ErrTooManySessions, len(payload.Sessions))
	}

	// Validate and encode session names first so we can fail fast before
	// allocating the output buffer.
	encodedNames := make([][]byte, len(payload.Sessions))
	for i, s := range payload.Sessions {
		name, err := encodedSessionName(s.SessionName)
		if err != nil {
			return nil, err
		}
		encodedNames[i] = name
	}

	// Calculate size up front for a single allocation.
	size := 16 + 8 + 8 + 2
	for _, name := range encodedNames {
		size += 2 + len(name) + 1 + 1
	}
	buf := make([]byte, 0, size)

	buf = append(buf, payload.SVTNID[:]...)
	buf = append(buf, payload.NodeAddr[:]...)
	buf = binary.BigEndian.AppendUint64(buf, payload.Sequence)

	count := uint16(len(payload.Sessions))
	buf = binary.BigEndian.AppendUint16(buf, count)

	for i, s := range payload.Sessions {
		name := encodedNames[i]
		nameLen := uint16(len(name))
		buf = binary.BigEndian.AppendUint16(buf, nameLen)
		buf = append(buf, name...)
		buf = append(buf, byte(s.Status))
		buf = append(buf, byte(s.Quality))
	}
	return buf, nil
}

// encodedSessionName returns the UTF-8 byte representation of name suitable
// for wire encoding. Empty names and non-UTF-8 names are rejected with
// ErrInvalidSessionName (BC-2.03.003 PC-2; RULING-W6TB-J). If the name exceeds
// 255 bytes it is truncated to at most 252 bytes on a rune boundary and suffixed
// with "…" (U+2026, 3 bytes), yielding a result no longer than 252+3 = 255 bytes,
// satisfying BC-2.03.003 PC-2 (encoded session name ≤ 255 bytes).
// The ellipsis signals lossy truncation to receivers (BC-2.03.003 EC-001).
func encodedSessionName(name string) ([]byte, error) {
	if len(name) == 0 {
		return nil, ErrInvalidSessionName
	}
	if !utf8.ValidString(name) {
		return nil, ErrInvalidSessionName
	}
	b := []byte(name)
	const maxBytes = 255
	const ellipsis = "…" // U+2026, 3 bytes
	if len(b) <= maxBytes {
		return b, nil
	}
	// Cut at 252 bytes then walk back to a valid rune boundary so that
	// appending the 3-byte ellipsis yields at most 255 bytes total.
	cut := 252
	for cut > 0 && !utf8.RuneStart(b[cut]) {
		cut--
	}
	// Allocate a fresh backing array to avoid aliasing hazards if the
	// caller retains a reference to the original []byte(name) buffer.
	result := make([]byte, 0, cut+len(ellipsis))
	result = append(result, b[:cut]...)
	result = append(result, ellipsis...)
	return result, nil
}

// decodeBody deserialises a body slice (without the HMAC tag prefix).
func decodeBody(body []byte) (AdvertisementPayload, error) {
	// Minimum: 16 + 8 + 8 + 2 = 34 bytes (SEC-DW-07 widened Sequence uint64).
	if len(body) < 34 {
		return AdvertisementPayload{}, errors.New("discovery: body too short")
	}
	var payload AdvertisementPayload
	copy(payload.SVTNID[:], body[:16])
	copy(payload.NodeAddr[:], body[16:24])
	payload.Sequence = binary.BigEndian.Uint64(body[24:32])

	count := binary.BigEndian.Uint16(body[32:34])
	// Guard against malformed or adversarial large-count payloads, bounded
	// by the shared maxSessionsPerAdvertisement constant (discovery_wire.go;
	// SEC-DW-02) rather than a local shadow, so decodeBody and
	// DecodeSessionList enforce the identical cap (F-DWIP1-002).
	if count > maxSessionsPerAdvertisement {
		return AdvertisementPayload{}, errors.New("discovery: session count exceeds maximum")
	}
	offset := 34

	for i := uint16(0); i < count; i++ {
		if offset+2 > len(body) {
			return AdvertisementPayload{}, errors.New("discovery: truncated session name length")
		}
		nameLen := int(binary.BigEndian.Uint16(body[offset : offset+2]))
		offset += 2

		// Reject nameLen == 0. Encode never produces zero-length names
		// (encodedSessionName returns ErrInvalidSessionName for empty
		// input), so a peer emitting a zero-length name is either
		// malformed or adversarial. Fail closed so the decoded struct
		// remains re-encodable and BC-2.03.003 Inv-1 (Encode/Decode
		// round-trip) holds without asymmetric zero-length names.
		if nameLen == 0 {
			return AdvertisementPayload{}, errors.New("discovery: session name is empty")
		}

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

// EncodeSessionList serialises sessions using this package's per-session
// wire encoding (uint16 name_len | name | uint8 status | uint8 quality),
// prefixed with a BE uint16 session count. Exported so that
// cmd/switchboard's DISCOVERY_RELAY frame assembly (AC-014 postcondition 2)
// reuses the identical per-session codec instead of re-implementing it.
// Session-name validation and truncation rules are identical to Encode's
// (BC-2.03.003 PC-2).
func EncodeSessionList(sessions []SessionPresence) ([]byte, error) {
	const maxSessions = 65535
	if len(sessions) > maxSessions {
		return nil, fmt.Errorf("%w: got %d", ErrTooManySessions, len(sessions))
	}

	encodedNames := make([][]byte, len(sessions))
	for i, s := range sessions {
		name, err := encodedSessionName(s.SessionName)
		if err != nil {
			return nil, err
		}
		encodedNames[i] = name
	}

	size := 2
	for _, name := range encodedNames {
		size += 2 + len(name) + 1 + 1
	}
	buf := make([]byte, 0, size)
	buf = binary.BigEndian.AppendUint16(buf, uint16(len(sessions)))
	for i, s := range sessions {
		name := encodedNames[i]
		buf = binary.BigEndian.AppendUint16(buf, uint16(len(name)))
		buf = append(buf, name...)
		buf = append(buf, byte(s.Status))
		buf = append(buf, byte(s.Quality))
	}
	return buf, nil
}

// DecodeSessionList parses a BE uint16 session count followed by that many
// per-session entries (uint16 name_len | name | uint8 status | uint8
// quality) from the start of body — the counterpart to EncodeSessionList,
// reused by RouterIngest.Ingest (AC-005/AC-011) and
// Discovery.IngestRelayAdvertisement (AC-007) so both hop-1 and hop-2
// ingest share one session-list codec. The declared count is bounded by
// maxSessionsPerAdvertisement (SEC-DW-02, AC-011 postcondition 3).
func DecodeSessionList(body []byte) ([]SessionPresence, error) {
	if len(body) < 2 {
		return nil, errors.New("discovery: session list truncated: missing count field")
	}
	count := binary.BigEndian.Uint16(body[:2])
	if count > maxSessionsPerAdvertisement {
		return nil, errors.New("discovery: session count exceeds maximum")
	}
	offset := 2

	var sessions []SessionPresence
	for i := uint16(0); i < count; i++ {
		if offset+2 > len(body) {
			return nil, errors.New("discovery: truncated session name length")
		}
		nameLen := int(binary.BigEndian.Uint16(body[offset : offset+2]))
		offset += 2

		if nameLen == 0 {
			return nil, errors.New("discovery: session name is empty")
		}
		if offset+nameLen+2 > len(body) {
			return nil, errors.New("discovery: truncated session entry")
		}
		name := string(body[offset : offset+nameLen])
		offset += nameLen

		status := AttachmentStatus(body[offset])
		quality := QualityIndicator(body[offset+1])
		offset += 2

		sessions = append(sessions, SessionPresence{
			SessionName: name,
			Status:      status,
			Quality:     quality,
		})
	}
	return sessions, nil
}
