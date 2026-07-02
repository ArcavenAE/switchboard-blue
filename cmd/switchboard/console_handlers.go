// console_handlers.go — daemon-side console RPC handler builder for cmd/switchboard.
//
// BuildConsoleHandlers returns the []mgmt.Handler slice for all console RPCs:
//
//	console.attach  (BC-2.08.001 PC-1; AC-001)
//	console.detach  (BC-2.08.001 PC-2; AC-002)
//	console.switch  (BC-2.08.001 PC-3; AC-003)
//
// Only the console-mode daemon calls BuildConsoleHandlers (ADR-004 role-exclusion;
// ARCH-04 disambiguation table; AC-004). Access, control, and router daemons pass
// nil handlers.
//
// Tier-2 admission (L1-C4): each handler checks that the authenticated caller
// holds RoleControl or RoleConsole via the mgmt-plane context (CallerPubkey +
// AdmittedKeySet lookup). Callers with any other role receive E-ADM-006.
//
// Purity classification (ARCH-09): boundary — depends on session.ConsoleServer
// (boundary) and mgmt.Handler (struct). No data-plane imports permitted
// (ADR-004 + ARCH-12 data-plane/management-plane separation).
//
// Forbidden imports: internal/routing, internal/multipath, internal/arq,
// internal/replay, internal/paths, internal/halfchannel, cmd/sbctl.
package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/mgmt"
	"github.com/arcavenae/switchboard/internal/session"
)

// BuildConsoleHandlers returns a []mgmt.Handler for the three console RPCs.
// cs must not be nil; ks must not be nil (both are required for handler dispatch).
//
// The admission check uses ks to resolve the caller's role from the public key
// injected into the context by the mgmt.Server handshake. Callers without
// RoleControl or RoleConsole receive E-ADM-006 (L1-C4; BC-2.08.001 Inv-1).
func BuildConsoleHandlers(cs *session.ConsoleServer, ks *admission.AdmittedKeySet) []mgmt.Handler {
	if cs == nil {
		panic("BuildConsoleHandlers: ConsoleServer must not be nil")
	}
	if ks == nil {
		panic("BuildConsoleHandlers: AdmittedKeySet must not be nil")
	}
	return []mgmt.Handler{
		{Command: "console.attach", Fn: makeConsoleAttachHandler(cs, ks)},
		{Command: "console.detach", Fn: makeConsoleDetachHandler(cs, ks)},
		{Command: "console.switch", Fn: makeConsoleSwitchHandler(cs, ks)},
	}
}

// verifyConsoleCallerRole checks that the authenticated caller has RoleControl
// or RoleConsole. Returns E-ADM-006 for any other role (L1-C4; BC-2.08.001 Inv-1).
//
// The caller public key is resolved from the mgmt-plane context via
// mgmt.CallerPubkey. The key's role is then resolved from the AdmittedKeySet.
// If the context has no caller pubkey (e.g., a unit test calling the handler
// outside a live mgmt.Server), the check fails closed (E-ADM-006).
//
// The zero svtnID ([16]byte{}) is the console-daemon's global partition — console
// keys are not SVTN-scoped (ARCH-04 §Console Key Scope; ADR-006).
func verifyConsoleCallerRole(ctx context.Context, ks *admission.AdmittedKeySet, cmd string) error {
	callerPub, ok := mgmt.CallerPubkey(ctx)
	if !ok {
		return fmt.Errorf("E-ADM-006: authorization denied for %s: no authenticated caller in context", cmd)
	}

	var zeroSVTN [16]byte
	entry, found := ks.LookupByPubkey(zeroSVTN, callerPub)
	if !found {
		fp := keyFingerprintAdmin(callerPub)
		return fmt.Errorf("E-ADM-006: authorization denied for %s: key %s not registered in console admission set", cmd, fp)
	}
	if entry.Role != admission.RoleControl && entry.Role != admission.RoleConsole {
		fp := keyFingerprintAdmin(callerPub)
		return fmt.Errorf("E-ADM-006: authorization denied for %s: key %s has role %s (requires control or console)", cmd, fp, entry.Role.String())
	}
	return nil
}

// makeConsoleAttachHandler returns the console.attach handler function.
// Traces to BC-2.08.001 PC-1; AC-001; L1-C1; L1-C4.
func makeConsoleAttachHandler(cs *session.ConsoleServer, ks *admission.AdmittedKeySet) func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		// L1-C4: Tier-2 admission check before any state mutation.
		if err := verifyConsoleCallerRole(ctx, ks, "console.attach"); err != nil {
			return nil, err
		}

		var req session.ConsoleAttachRequest
		if err := json.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("E-CFG-001: invalid request args: %w", err)
		}
		if req.SessionName == "" {
			return nil, fmt.Errorf("E-CFG-001: missing required field: session_name")
		}

		return cs.HandleConsoleAttach(ctx, req)
	}
}

// makeConsoleDetachHandler returns the console.detach handler function.
// Traces to BC-2.08.001 PC-2; AC-002; L1-C1; L1-C4.
func makeConsoleDetachHandler(cs *session.ConsoleServer, ks *admission.AdmittedKeySet) func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		// L1-C4: Tier-2 admission check.
		if err := verifyConsoleCallerRole(ctx, ks, "console.detach"); err != nil {
			return nil, err
		}

		// console.detach has no required args — pass the empty request struct.
		return cs.HandleConsoleDetach(ctx, session.ConsoleDetachRequest{})
	}
}

// makeConsoleSwitchHandler returns the console.switch handler function.
// Traces to BC-2.08.001 PC-3; AC-003; L1-C1; L1-C4.
func makeConsoleSwitchHandler(cs *session.ConsoleServer, ks *admission.AdmittedKeySet) func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		// L1-C4: Tier-2 admission check.
		if err := verifyConsoleCallerRole(ctx, ks, "console.switch"); err != nil {
			return nil, err
		}

		var req session.ConsoleSwitchRequest
		if err := json.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("E-CFG-001: invalid request args: %w", err)
		}
		if req.SessionName == "" {
			return nil, fmt.Errorf("E-CFG-001: missing required field: session_name")
		}

		return cs.HandleConsoleSwitch(ctx, req)
	}
}
