// Package discovery implements SVTN-scoped multicast session presence
// advertisement and enumeration (BC-2.03.001, BC-2.03.002, BC-2.03.003).
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
	"errors"
	"time"

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
}

// Discovery is the SVTN-scoped multicast session presence subsystem.
//
// It satisfies two roles:
//  1. Advertiser — sends SessionPresence payloads to the SVTN multicast
//     channel on state change and on periodic heartbeat.
//  2. Enumerator — aggregates inbound advertisements into a local session
//     registry and returns SVTN-scoped results on demand.
type Discovery struct {
	cfg Config
}

// New constructs a Discovery instance from cfg.
//
// Returns a *Discovery ready for use. The caller must call Run to start
// background goroutines (heartbeat timer and advertisement receiver).
func New(cfg Config) *Discovery {
	return &Discovery{cfg: cfg}
}

// Run starts the heartbeat timer and advertisement receive loop.
//
// It blocks until ctx is cancelled. The caller is responsible for
// cancelling ctx to stop background goroutines cleanly.
func (d *Discovery) Run(ctx context.Context) error {
	panic("not implemented") //nolint:gocritic // stub: todo!() equivalent per BC-5.38.001
}

// Advertise enqueues an advertisement for the supplied session presence list.
//
// It MUST fire within 1 tick interval of a state change (AC-001a;
// BC-2.03.001 PC-3). The periodic heartbeat is handled internally by Run and
// is independent of Advertise calls (AC-001b; BC-2.03.001 PC-4).
//
// The advertisement payload is HMAC-authenticated via the Router's HMAC
// surface before transmission (AC-005; BC-2.03.001 PC-5).
func (d *Discovery) Advertise(ctx context.Context, sessions []SessionPresence) error {
	panic("not implemented") //nolint:gocritic // stub: todo!() equivalent per BC-5.38.001
}

// Enumerate returns the set of sessions currently known on the local SVTN.
//
// Sessions from a different SVTN MUST NOT appear in the result (AC-006;
// BC-2.03.002 Inv-1). The result MUST aggregate sessions from all known
// advertisers — a valid production result includes entries from at least
// 2 distinct node addresses (AC-002; BC-2.03.002 PC-3).
func (d *Discovery) Enumerate(ctx context.Context) ([]SessionEntry, error) {
	panic("not implemented") //nolint:gocritic // stub: todo!() equivalent per BC-5.38.001
}

// ReceiveAdvertisement processes a raw advertisement payload received from
// the multicast channel.
//
// It authenticates the HMAC tag via the Router's HMAC surface (AC-005) and
// rejects payloads from a foreign SVTN (AC-006). Valid payloads are merged
// into the local session registry.
func (d *Discovery) ReceiveAdvertisement(ctx context.Context, raw []byte) error {
	panic("not implemented") //nolint:gocritic // stub: todo!() equivalent per BC-5.38.001
}

// Encode serialises payload to its wire representation.
//
// The encoding is stable: Encode(Decode(b)) == b for any valid advertisement
// byte slice b (AC-004; BC-2.03.003 Inv-1 round-trip).
func Encode(payload AdvertisementPayload) ([]byte, error) {
	panic("not implemented") //nolint:gocritic // stub: todo!() equivalent per BC-5.38.001
}

// Decode deserialises raw into an AdvertisementPayload.
//
// Returns ErrInvalidHMACTag if the payload is malformed in a way that
// indicates an authentication failure, or a wrapped decode error otherwise.
func Decode(raw []byte) (AdvertisementPayload, error) {
	panic("not implemented") //nolint:gocritic // stub: todo!() equivalent per BC-5.38.001
}
