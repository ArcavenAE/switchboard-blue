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
// Multi-SVTN group-membership dynamics — which SVTN group address(es) a
// running router process should join at startup, and how that set changes
// as SVTNs are admitted or reloaded — is EXPLICITLY NOT resolved by this
// function or this story. internal/admission.AdmittedKeySet exposes no
// SVTN-enumeration method and no admission-event hook (the nodeConnHook/
// drainObserverFiredHook pattern this codebase uses elsewhere for other
// lifecycle events has no SVTN-admission equivalent), and config.Config
// carries no static SVTN identifier — so runRouter has no data source from
// which to derive "which SVTNs to join at startup" without inventing new
// production API surface in internal/admission or internal/routing, which
// this story's File-Change List does not authorize (it lists only
// cmd/switchboard/mgmt_wire.go as touched for this concern). Wiring
// wireDiscoveryListener into runRouter's daemon lifecycle is therefore left
// to a follow-on story once an SVTN-admission-event source exists;
// wireDiscoveryListener itself is fully implemented and independently
// tested (TestRunRouter_DiscoveryListener_JoinsGroup_RouterModeOnly), ready
// for that story to call.
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

// discoveryReadBufSize bounds the per-datagram read buffer. Sized to the
// maximum possible IPv4 UDP payload (65536 bytes, rounded up from the
// 65507-byte theoretical max) rather than internal/discovery's own
// maxDiscoveryDatagramSize (unexported) — RouterIngest.Ingest independently
// re-enforces the tighter SEC-DW-02 bound on whatever is read here, so an
// oversized local buffer is harmless.
const discoveryReadBufSize = 65536

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
// and return early. The caller is expected to invoke it via `go
// wireDiscoveryListener(ctx, wg, ...)` (exactly as
// TestRunRouter_DiscoveryListener_JoinsGroup_RouterModeOnly does), so that
// this call's own goroutine IS the WaitGroup-tracked unit of work (ARCH-01
// §Goroutine WaitGroup Contract) — wg.Add(1)/defer wg.Done() bracket the
// entire function, covering both the early bind/join-error return path and
// the long-lived read loop. A second, short-lived goroutine closes the
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
func wireDiscoveryListener(ctx context.Context, wg *sync.WaitGroup, svtnID [16]byte, ri *discovery.RouterIngest, w io.Writer) error {
	wg.Add(1)
	defer wg.Done()

	groupAddr := discovery.MulticastAddrFor(svtnID)
	listenAddr := &net.UDPAddr{IP: groupAddr, Port: discovery.DiscoveryPort}

	conn, err := net.ListenMulticastUDP("udp4", nil, listenAddr)
	if err != nil {
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

		if _, ingestErr := ri.Ingest(raw); ingestErr != nil {
			continue
		}
	}
}
