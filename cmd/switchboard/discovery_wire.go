// discovery_wire.go — router-mode-exclusive discovery multicast listener
// wiring for cmd/switchboard (S-BL.DISCOVERY-WIRE Task 3; AC-001).
//
// wireDiscoveryListener binds net.ListenMulticastUDP on a SVTN-derived group
// address, joins the group, and dispatches inbound datagrams into
// internal/discovery's router-side ingest path (SEC-DW-01..07). Only
// runRouter calls it (mgmt_wire.go) — runAccess, runConsole, and runControl
// never join any multicast group and never receive advertisements directly
// from another node's socket (AC-001 postcondition 2). Wiring this function
// into runRouter's register-before-serve sequence (mirroring
// wireMetricsHandlers/wireRouterControlHandlers) is Task 3's Green-step
// action, not performed by this stub.
//
// Multi-SVTN group-membership dynamics — which SVTN group address(es) a
// running router process joins, and how that set changes as SVTNs are
// admitted or reloaded — are Task 3 Green-step design; this stub fixes the
// single-SVTN-per-call shape so AC-001's join/no-join test can compile
// against it.
//
// Purity classification (ARCH-09): effectful-boundary — network I/O
// (multicast socket bind/join/read).
package main

import (
	"context"
	"io"
	"sync"

	"github.com/arcavenae/switchboard/internal/discovery"
)

// wireDiscoveryListener starts the router-mode-exclusive discovery
// multicast listener for one SVTN's group address (discovery.MulticastAddrFor),
// joins the group, and dispatches inbound datagrams into ri's router-side
// ingest path until ctx is cancelled.
//
// AC-001 / BC-2.03.001 Postcondition 1 (delivery-mechanism note), Invariant 1
// (DI-004).
//
// STUB — S-BL.DISCOVERY-WIRE (Red Gate, BC-5.38.001). Not yet implemented;
// body panics unconditionally so no test can accidentally pass before
// Task 3's Green step. No call site yet: wiring into runRouter is Task 3's
// Green-step action (mgmt_wire.go) — calling this stub eagerly at router
// startup during Red Gate would panic on every existing router-mode test.
//
//nolint:unused // see doc comment above: wiring deferred to Task 3 Green step
func wireDiscoveryListener(ctx context.Context, wg *sync.WaitGroup, svtnID [16]byte, ri *discovery.RouterIngest, w io.Writer) error {
	panic("not implemented: S-BL.DISCOVERY-WIRE wireDiscoveryListener")
}
