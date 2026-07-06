//go:build integration

// Package config_test VP-038 e2e test.
//
// TestE2E_EtoPE_GraduationByConfigChange discharges VP-038 (E→PE graduation
// by config change only) using the testenv rig.
//
// Proof status: PARTIAL — the testenv RouterHandle provides in-process
// mode inspection and in-place Restart().  Full verification that the
// binary mode changes without restart requires the production PE connector
// wire (S-7.04-FU-PE-CONNECTOR + S-7.04-FU-SIGHUP-RELOAD).  The harness
// provided here confirms the mode enum transition and SVTN-ID preservation.
//
// Traces to: VP-038, BC-2.09.001
package config_test

import (
	"context"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/testenv"
)

// TestE2E_EtoPE_GraduationByConfigChange verifies that adding upstream_routers
// to a router config and restarting promotes it to PE mode without changing
// the SVTN ID (no binary replacement, no SVTN re-initialization).
//
// Traces to: VP-038, BC-2.09.001
func TestE2E_EtoPE_GraduationByConfigChange(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	// Start a PE (upstream) router that the E router will connect to.
	env := testenv.New(t, ctx)
	t.Cleanup(env.Close)

	peAddr := env.PERouterAddr(t)

	// Start the router in E mode (no upstream_routers).
	eRouter := env.StartRouter(t, testenv.RouterConfig{
		UpstreamRouters: nil,
	})
	if eRouter.Mode() != testenv.ModeE {
		t.Fatalf("expected E mode at startup, got %v", eRouter.Mode())
	}

	svtnIDBefore := eRouter.SVTNID()

	// Restart the same router handle with upstream_routers populated.
	eRouter.Restart(t, testenv.RouterConfig{
		UpstreamRouters: []string{peAddr},
	})

	// Allow startup.
	time.Sleep(500 * time.Millisecond)

	if eRouter.Mode() != testenv.ModePE {
		t.Errorf("expected PE mode after restart with upstream_routers, got %v", eRouter.Mode())
	}

	// SVTN must be unchanged.
	if eRouter.SVTNID() != svtnIDBefore {
		t.Errorf("SVTN ID changed after graduation: before=%v after=%v",
			svtnIDBefore, eRouter.SVTNID())
	}
}
