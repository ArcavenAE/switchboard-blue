// discovery_wire.go — router-mode-exclusive discovery multicast listener
// wiring for cmd/switchboard (S-BL.DISCOVERY-WIRE Task 3; AC-001).
//
// wireDiscoveryListener binds net.ListenMulticastUDP on a SVTN-derived group
// address, joins the group, and dispatches inbound datagrams into
// internal/discovery's router-side ingest path (SEC-DW-01..07). Only
// runRouter is meant to call it (mgmt_wire.go) — runAccess, runConsole, and
// runControl never join any multicast group and never receive
// advertisements directly from another node's socket (AC-001 postcondition
// 2, verified by inspection: neither imports internal/discovery).
//
// Task 6d wired wireDiscoveryListener into runRouter's daemon lifecycle:
// runRouter iterates routerKS.AllSVTNEntries() at startup and spawns one
// listener goroutine per admitted SVTN, passing an onRelay inline closure that
// chains RouterIngest.Ingest → relayRateCap.allow → relayDispatch.
//
// Dynamic SVTN group join/leave (Forward Obligation g): SVTNs admitted after
// runRouter startup via wireAdmissionSyncHandlers push do not automatically
// get a discovery listener goroutine. A follow-on story must add an
// admission-event hook (analogous to nodeConnHook) to call
// wireDiscoveryListener for newly admitted SVTNs. This requires new API
// surface in wireAdmissionSyncHandlers and is outside the current story's
// File-Change List.
//
// Purity classification (ARCH-09): effectful-boundary — network I/O
// (multicast socket bind/join/read).
package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/arcavenae/switchboard/internal/discovery"
)

// discoveryReadBufSize bounds the per-datagram read buffer. AC-011 PC-1
// explicitly forbids sizing this to the 65,507-byte UDP/IP theoretical
// maximum — it MUST be sized to a realistic worst-case legitimate
// advertisement instead. discovery.MaxDiscoveryDatagramSize is exactly that
// bound (F-DWIP3-002): it is exported specifically so this read buffer can
// share it rather than independently re-deriving a looser one.
// RouterIngest.Ingest still independently re-enforces the identical
// SEC-DW-02 bound (AC-011 PC-2) on whatever is read here — belt-and-braces,
// not redundant, since Ingest must reject an oversized datagram regardless
// of what any particular caller's read buffer happens to be sized to.
const discoveryReadBufSize = discovery.MaxDiscoveryDatagramSize

// wireDiscoveryListener starts the router-mode-exclusive discovery
// multicast listener for one SVTN's group address (discovery.MulticastAddrFor),
// joins the group, and dispatches inbound datagrams into ri's router-side
// ingest path until ctx is cancelled.
//
// AC-001 / BC-2.03.001 Postcondition 1 (delivery-mechanism note), Invariant 1
// (DI-004): the interface argument to net.ListenMulticastUDP is nil — the
// system default multicast interface is joined, not a single hardcoded
// named interface. Only the router ever calls this function (DI-004: no
// direct node-to-node communication; access nodes never join any multicast
// group).
//
// This function itself BLOCKS until ctx is cancelled (or a fatal socket
// error occurs) — it does not spawn an internal goroutine for the read loop
// and return early. The caller MUST call wg.Add(1) BEFORE dispatching `go
// wireDiscoveryListener(ctx, wg, ...)` (exactly as
// TestRunRouter_DiscoveryListener_JoinsGroup_RouterModeOnly does) — the
// canonical ARCH-01 §Goroutine WaitGroup Contract pattern used at every
// other wg-tracked call site in this package (access.go, mgmt_wire.go):
// Add happens synchronously in the parent before `go`, never inside the
// spawned goroutine itself, to avoid the race where a concurrent wg.Wait()
// could observe a zero WaitGroup counter and return before this goroutine
// has registered (F-DWIP3-001). wireDiscoveryListener itself only calls
// `defer wg.Done()`, covering both the early bind/join-error return path
// and the long-lived read loop. A second, short-lived goroutine closes the
// socket when ctx.Done() fires, to unblock the blocking ReadFromUDP call —
// the standard cancel-via-close idiom for net.Conn, which has no
// context-aware read variant.
//
// Malformed, unauthenticated, or rate-limited datagrams (RouterIngest.Ingest
// returning a non-nil error) are expected background noise on an open UDP
// multicast socket — SEC-DW-01..04 already rate-limit and count failures
// inside Ingest itself — so this loop does not log per-rejected-datagram
// and does not exit on an Ingest error; it only exits on a real socket
// error or ctx cancellation.
//
// onRelay, if non-nil, is invoked for every datagram where
// decision.Relay == true (HMAC-verified, replay-accepted). The callback
// contract is: the caller receives only relay-worthy decisions; nil is
// a valid no-op (discard relay dispatch — same behaviour as today's
// blank-identifier discard; fail-safe rather than fail-open, as nil
// suppresses relay amplification without affecting the HMAC/replay-gate
// invariants enforced by RouterIngest.Ingest; ruling Decision 2 nil semantics).
// See S-BL.DISCOVERY-WIRE-task6d-wiring-seam-ruling.md Decision 1/2.
func wireDiscoveryListener(
	ctx context.Context,
	wg *sync.WaitGroup,
	svtnID [16]byte,
	ri *discovery.RouterIngest,
	w io.Writer,
	onRelay func(discovery.RouterIngestDecision),
) error {
	defer wg.Done()

	groupAddr := discovery.MulticastAddrFor(svtnID)
	listenAddr := &net.UDPAddr{IP: groupAddr, Port: discovery.DiscoveryPort}

	conn, err := net.ListenMulticastUDP("udp4", nil, listenAddr)
	if err != nil {
		// Mirror the read-error path: write to w before returning so the
		// operator sees the failure (SOUL.md #4 — no silent failures).
		_, _ = fmt.Fprintf(w, "discovery: multicast join error (svtn=%x, group=%s:%d): %v\n", svtnID, groupAddr, discovery.DiscoveryPort, err)
		return fmt.Errorf("wireDiscoveryListener: join multicast group %s:%d: %w", groupAddr, discovery.DiscoveryPort, err)
	}
	defer func() { _ = conn.Close() }()

	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	buf := make([]byte, discoveryReadBufSize)
	for {
		n, _, readErr := conn.ReadFromUDP(buf)
		if readErr != nil {
			if ctx.Err() != nil {
				return nil
			}
			_, _ = fmt.Fprintf(w, "discovery: multicast read error (svtn=%x): %v\n", svtnID, readErr)
			return fmt.Errorf("wireDiscoveryListener: multicast read error: %w", readErr)
		}

		raw := make([]byte, n)
		copy(raw, buf[:n])

		decision, ingestErr := ri.Ingest(raw)
		if ingestErr != nil {
			continue
		}
		if decision.Relay && onRelay != nil {
			onRelay(decision)
		}
	}
}
