// Package routing_test — W3-R2-M2 regression witness: concurrent
// RegisterForwardingEntry vs RouteFrame under the race detector.
//
// # Adjudication (W3-R2-M2)
//
// Wave-3 restart pass-r2 M-2 observed that RouteFrame's src-authkey snapshot
// (steps 1–2 in RouteFrame) is taken under one RLock, and SVTNRoute's
// dst-entry lookup is taken under a separate RLock later. A concurrent
// RegisterForwardingEntry between the two lookups can therefore change the
// forwarding table between the "verify who sent this" moment and the "look
// up where to deliver it" moment. The pass rated this MEDIUM (not HIGH)
// because it does NOT admit forged frames — HMAC verification still runs
// against a real registered key.
//
// Verdict: BENIGN-BY-DESIGN.
//
//   - The forwarding table is documented last-write-wins (ADR-003; see
//     RegisterForwardingEntry doc comment). LWW means concurrent updates
//     may take effect between any two lookups; that is the accepted
//     contract, not a defect.
//   - RouteFrame's step-1 lookup authenticates the SENDER at the moment
//     of snapshot; SVTNRoute's dst lookup selects the DELIVERY TARGET at
//     a later moment. These are two independent decisions against a
//     mutable table. A concurrent registration between them changes the
//     delivery target but never retroactively unauthenticates the sender.
//   - FrameAuthKey is [hmac.KeySize]byte (a value type, not a pointer);
//     step 1 copies it into a local variable before the RUnlock, so the
//     HMAC computation cannot observe a torn key.
//   - Verify-then-lookup is preserved by statement order in RouteFrame
//     (ADR-009 v1.6 §"Ordering specification"); it is not preserved by
//     one continuous lock, and the ADR explicitly declines to hold the
//     RLock across the CPU-bound HMAC verification.
//
// The tests below are the DURABLE WITNESS of this contract. They provoke
// the exact interleaving W3-R2-M2 described and assert:
//
//  1. Race-detector clean — the [hmac.KeySize]byte copy pattern in
//     RouteFrame keeps step 1 atomic wrt registration.
//  2. Every outcome is one of {nil, ErrHMACVerificationFailed,
//     ErrNoForwardingEntry, ErrNotAdmitted}. No panic, no unexpected
//     error, no goroutine leak.
//  3. Successful routing implies HMAC verification succeeded against
//     SOME registered key at snapshot time — no forgery admission.
//  4. When the concurrent registrar alternates authKey K1 → K2 → K1,
//     RouteFrame calls carrying a K1-tagged frame return one of two
//     defensible outcomes (nil OR ErrHMACVerificationFailed) — never
//     a torn state, never a panic.
//
// Traces to: Wave-3 adversary pass-r2 M-2; ADR-003 (LWW); ADR-009 v1.6
// §"Ordering specification"; BC-2.05.008 PC-3.
//
// Run with `-race` to exercise the race detector; standard test invocation
// suffices for behavioral witness.
package routing_test

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/hmac"
	"github.com/arcavenae/switchboard/internal/routing"
)

// TestRouteFrame_ConcurrentRegisterForwardingEntry_LWWSnapshot is the
// W3-R2-M2 regression witness. It provokes the interleaving W3-R2-M2
// described — concurrent RegisterForwardingEntry writes vs RouteFrame
// reads — and asserts the accepted LWW semantics hold: race-detector
// clean, no torn state, and every outcome is a defensible member of the
// return-value set.
//
// The concurrent registrar flips the src forwarding entry's authKey
// between the frame's ORIGINAL key (K1) and a WRONG key (K2). Frames
// carry a valid tag computed with K1. Under the LWW contract:
//
//   - When the snapshotted key equals K1, HMAC verification succeeds and
//     RouteFrame returns nil (or ErrNoForwardingEntry if the dst entry is
//     concurrently rewritten — also acceptable LWW).
//   - When the snapshotted key equals K2, HMAC verification fails and
//     RouteFrame returns ErrHMACVerificationFailed (the sender must
//     re-sign with the new key; expected under key rotation).
//
// Both outcomes are defensible. No other outcomes are legal. Anything
// else — panic, torn read, race detector firing — is a failure of the
// contract that would require the code fix W3-R2-M2 speculated about.
//
// Traces to: Wave-3 adversary pass-r2 M-2 (adjudicated BENIGN-BY-DESIGN);
// ADR-003 LWW; ADR-009 v1.6.
func TestRouteFrame_ConcurrentRegisterForwardingEntry_LWWSnapshot(t *testing.T) {
	t.Parallel()

	var svtnID [16]byte
	copy(svtnID[:], "w3r2m2-svtn-0001")

	// Admitted router + real key K1 that produces valid tags.
	r, srcAddr, authK1 := admittedRouterSetup(t, svtnID, 0xA1, 0xB1)

	// Wrong key K2 — same shape, different content.
	var authK2 [hmac.KeySize]byte
	copy(authK2[:], "w3r2m2-WRONG-key-32-bytes-01234")

	// Destination.
	var dstAddr [8]byte
	copy(dstAddr[:], "w3r2m2ds")
	var dstKey [hmac.KeySize]byte
	copy(dstKey[:], "w3r2m2-dst-key-32-bytes-0000000")
	r.RegisterForwardingEntry(svtnID, dstAddr, dstKey)

	// Pre-compute the wire frame ONCE — HMAC tag over K1.
	payload := []byte("w3r2m2-payload-value")
	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    svtnID,
		SrcAddr:   srcAddr,
		DstAddr:   dstAddr,
	}
	hdr.HMACTag = computeValidTag(hdr, payload, authK1)

	// Kick off the interleaving. One registrar goroutine flips authKey
	// between K1 and K2 in a tight loop; multiple router goroutines
	// call RouteFrame with the K1-signed frame concurrently.
	const (
		routers    = 8
		iterations = 2000
	)

	var (
		routerWG    sync.WaitGroup
		registrarWG sync.WaitGroup
		stop        atomic.Bool
		gotOK       atomic.Uint64
		gotHMAC     atomic.Uint64
		gotNoFwd    atomic.Uint64
		gotNotAdmit atomic.Uint64
		gotOther    atomic.Uint64
	)

	// Registrar: alternate authKey between K1 and K2 until stop.
	registrarWG.Add(1)
	go func() {
		defer registrarWG.Done()
		useK1 := true
		for !stop.Load() {
			if useK1 {
				r.RegisterForwardingEntry(svtnID, srcAddr, authK1)
			} else {
				r.RegisterForwardingEntry(svtnID, srcAddr, authK2)
			}
			useK1 = !useK1
		}
	}()

	// Routers: call RouteFrame with the K1-signed frame `iterations` times each.
	for range routers {
		routerWG.Add(1)
		go func() {
			defer routerWG.Done()
			for range iterations {
				err := routing.RouteFrame(hdr, payload, r)
				switch {
				case err == nil:
					gotOK.Add(1)
				case errors.Is(err, routing.ErrHMACVerificationFailed):
					gotHMAC.Add(1)
				case errors.Is(err, routing.ErrNoForwardingEntry):
					// Also acceptable: dst-side registrar contention could
					// (in a broader test) reroute delivery. We don't provoke
					// dst rewrites here, but the outcome set is documented
					// so this branch stays honest.
					gotNoFwd.Add(1)
				case errors.Is(err, admission.ErrNotAdmitted):
					gotNotAdmit.Add(1)
				default:
					gotOther.Add(1)
					t.Errorf("W3-R2-M2 witness: unexpected error from RouteFrame under concurrent "+
						"RegisterForwardingEntry: %v (want one of {nil, ErrHMACVerificationFailed, "+
						"ErrNoForwardingEntry, ErrNotAdmitted})", err)
				}
			}
		}()
	}

	// Wait for routers, then stop the registrar and wait for it too.
	routerWG.Wait()
	stop.Store(true)
	registrarWG.Wait()

	// Contract invariants.
	if gotOther.Load() > 0 {
		t.Fatalf("W3-R2-M2 witness: %d unexpected errors — contract violated", gotOther.Load())
	}
	total := gotOK.Load() + gotHMAC.Load() + gotNoFwd.Load() + gotNotAdmit.Load()
	if total != uint64(routers*iterations) {
		t.Fatalf("W3-R2-M2 witness: accounting error — expected %d outcomes, got %d",
			routers*iterations, total)
	}
	// Under this interleaving we expect BOTH nil and ErrHMACVerificationFailed
	// to be observed at least once — the whole point of the witness is to
	// prove both branches of the LWW race fire. If we only ever see one
	// outcome, the concurrent registrar isn't actually racing.
	if gotOK.Load() == 0 {
		t.Errorf("W3-R2-M2 witness: expected at least one successful route under LWW race; "+
			"got 0 (of %d total). Interleaving may not be provoking the race — "+
			"check goroutine count / iterations.", total)
	}
	if gotHMAC.Load() == 0 {
		t.Errorf("W3-R2-M2 witness: expected at least one ErrHMACVerificationFailed under LWW race; "+
			"got 0 (of %d total). Registrar may not be racing routers — "+
			"check goroutine count / iterations.", total)
	}
}

// TestRouteFrame_ConcurrentRegisterForwardingEntry_NoForgery is a second
// W3-R2-M2 witness with a stronger invariant: even under aggressive
// concurrent registration, RouteFrame NEVER admits a frame signed with a
// key that was never registered.
//
// The registrar cycles authKey between K1 and K2 (both valid, both
// registered at various points in time). The test sender uses key K3
// which is NEVER registered. All calls MUST return
// ErrHMACVerificationFailed — the LWW race cannot produce a "false OK"
// because HMAC verification runs against whichever key IS in the table
// at snapshot time, and K3 matches neither.
//
// This is the load-bearing security witness: LWW may re-order which of
// {K1, K2} is current, but it can never make an un-registered K3 look
// valid.
//
// Traces to: Wave-3 adversary pass-r2 M-2 §"Does NOT admit forged frames";
// BC-2.05.008 PC-2; ADR-009 v1.6.
func TestRouteFrame_ConcurrentRegisterForwardingEntry_NoForgery(t *testing.T) {
	t.Parallel()

	var svtnID [16]byte
	copy(svtnID[:], "w3r2m2-svtn-0002")

	r, srcAddr, authK1 := admittedRouterSetup(t, svtnID, 0xA2, 0xB2)

	var authK2 [hmac.KeySize]byte
	copy(authK2[:], "w3r2m2-K2-key-32-bytes-00000000")

	// K3 is the FORGERY key — never registered on the router.
	var authK3 [hmac.KeySize]byte
	copy(authK3[:], "w3r2m2-K3-FORGERY-key-000000000")

	var dstAddr [8]byte
	copy(dstAddr[:], "w3r2m2fs")
	r.RegisterForwardingEntry(svtnID, dstAddr, [hmac.KeySize]byte{0xDE})

	// Sign the frame with K3 — the forgery key.
	payload := []byte("forgery-payload")
	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    svtnID,
		SrcAddr:   srcAddr,
		DstAddr:   dstAddr,
	}
	hdr.HMACTag = computeValidTag(hdr, payload, authK3)

	const (
		routers    = 8
		iterations = 2000
	)

	var (
		routerWG    sync.WaitGroup
		registrarWG sync.WaitGroup
		stop        atomic.Bool
		forgeries   atomic.Uint64
	)

	// Registrar flips the src entry between K1 and K2 — both real,
	// neither matches K3.
	registrarWG.Add(1)
	go func() {
		defer registrarWG.Done()
		useK1 := true
		for !stop.Load() {
			if useK1 {
				r.RegisterForwardingEntry(svtnID, srcAddr, authK1)
			} else {
				r.RegisterForwardingEntry(svtnID, srcAddr, authK2)
			}
			useK1 = !useK1
		}
	}()

	// Routers submit K3-signed frames.
	for range routers {
		routerWG.Add(1)
		go func() {
			defer routerWG.Done()
			for range iterations {
				err := routing.RouteFrame(hdr, payload, r)
				// A K3-signed frame MUST NEVER be accepted, regardless of
				// which of {K1, K2} is current in the table. Any nil return
				// is a forgery admission — the core contract W3-R2-M2 needs
				// to preserve.
				if err == nil {
					forgeries.Add(1)
					t.Errorf("W3-R2-M2 no-forgery witness: RouteFrame returned nil for K3-signed "+
						"frame under concurrent K1↔K2 registration. LWW must NEVER admit a frame "+
						"signed with a key that was never registered. err=%v", err)
					return
				}
				// PATH-A (no entry — momentarily unlikely but possible if
				// the registrar-side lock hands off oddly) and PATH-B (tag
				// mismatch) both produce ErrHMACVerificationFailed. That is
				// the only legal outcome; anything else (ErrNotAdmitted,
				// unknown error) also violates the contract.
				if !errors.Is(err, routing.ErrHMACVerificationFailed) {
					t.Errorf("W3-R2-M2 no-forgery witness: K3-signed frame under LWW race must "+
						"return ErrHMACVerificationFailed; got %v", err)
					return
				}
			}
		}()
	}

	routerWG.Wait()
	stop.Store(true)
	registrarWG.Wait()

	if forgeries.Load() > 0 {
		t.Fatalf("W3-R2-M2 no-forgery witness: %d frames signed with an un-registered key were "+
			"admitted — LWW race would need code-level remediation (single-lock verify+forward, "+
			"or entry-copy through SVTNRoute)", forgeries.Load())
	}
}
