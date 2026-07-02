package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"strings"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/mgmt"
	"github.com/arcavenae/switchboard/internal/session"
)

// zeroSVTN is the global console-admission SVTN partition key (not SVTN-scoped;
// console keys are registered under the zero svtnID per ARCH-04 §Console Key Scope).
var zeroSVTN [16]byte

// newConsoleE2EStack constructs the full console daemon infrastructure stack for
// an integration test: AdmittedKeySet, Publisher (with pre-seeded sessions), and
// ConsoleServer. Returns the assembled components so the caller can register keys
// and seed sessions as needed.
func newConsoleE2EStack(t *testing.T, sessionNames ...string) (*admission.AdmittedKeySet, *session.ConsoleServer) {
	t.Helper()
	ks := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(ks)
	for _, name := range sessionNames {
		if err := pub.Publish(name); err != nil {
			t.Fatalf("newConsoleE2EStack: Publish %q: %v", name, err)
		}
	}
	consoleState := session.NewConsoleState()
	consoleSrv := session.NewConsoleServer(pub, consoleState)
	return ks, consoleSrv
}

// startConsoleE2EServer starts a real mgmt.Server with BuildConsoleHandlers
// registered. callerPub is added to the OperatorKeySet so that the mgmt.Server
// handshake admits the caller (non-bootstrap mode; mgmt.Server admits any key in
// the OperatorKeySet). The daemon keypair is ephemeral and generated per call.
//
// Returns the e2eServer and the caller's public key for use with sendAdminRPCAsKey.
func startConsoleE2EServer(
	t *testing.T,
	ks *admission.AdmittedKeySet,
	consoleSrv *session.ConsoleServer,
	callerPubs ...ed25519.PublicKey,
) *e2eServer {
	t.Helper()

	_, daemonPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("startConsoleE2EServer: generate daemon keypair: %v", err)
	}

	// Build OperatorKeySet with all caller public keys so mgmt.Server admits them.
	ops := mgmt.NewOperatorKeySet(callerPubs)

	es := startE2EServerWithOps(t, BuildConsoleHandlers(consoleSrv, ks), daemonPriv, ops)

	// Allow the server to start accepting connections.
	time.Sleep(10 * time.Millisecond)

	return es
}

// TestConsoleRemote_E2E_VP050 exercises the full attach→switch→detach cycle
// through a real mgmt.Server using the console.attach, console.detach, and
// console.switch commands.
//
// VP-050 — console remotely controllable via mgmt-plane socket.
// BC-2.08.001 PC-1/PC-2/PC-3.
//
// Non-tautological assertion strategy (L2-T2/T3/T4):
//   - Each RPC is sent over a real mgmt.Server with TLS-like Ed25519 handshake
//   - Admission is enforced: the caller key is registered in AdmittedKeySet as RoleConsole
//   - Server-side state is verified via subsequent RPC (attach → detach echoes name;
//     switch → subsequent detach echoes new name — proving L1-C3 state tracking)
func TestConsoleRemote_E2E_VP050(t *testing.T) {
	ks, consoleSrv := newConsoleE2EStack(t, "agent-01", "agent-02")

	// Generate caller keypair and register with RoleConsole in the admission set.
	callerPub, callerPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("VP-050: generate caller keypair: %v", err)
	}
	// Register in AdmittedKeySet so verifyConsoleCallerRole passes (L1-C4).
	ks.RegisterKey(zeroSVTN, callerPub, admission.RoleConsole)

	// Start server with callerPub in OperatorKeySet so mgmt handshake admits it.
	es := startConsoleE2EServer(t, ks, consoleSrv, callerPub)

	// AC-001: console.attach — attach to agent-01.
	// BC-2.08.001 PC-1.
	attachResp := sendAdminRPCAsKey(t, es.socketPath, callerPub, callerPriv,
		"console.attach", map[string]any{"session_name": "agent-01"})
	if errObj, _ := attachResp["error"].(map[string]any); errObj != nil {
		t.Fatalf("VP-050 AC-001 — console.attach: unexpected error: %v", errObj)
	}
	attachData, _ := attachResp["data"].(map[string]any)
	if attachData == nil {
		t.Fatalf("VP-050 AC-001 — console.attach: data is nil; full response: %v", attachResp)
	}
	if got, _ := attachData["session_name"].(string); got != "agent-01" {
		t.Errorf("VP-050 AC-001 — console.attach result.session_name: got %q; want %q", got, "agent-01")
	}

	// AC-002: console.detach — detach; assert server-side state had tracked agent-01.
	// BC-2.08.001 PC-2.
	// Non-tautological: mgmt.Server dispatched to handler; state was set by prior RPC.
	detachResp := sendAdminRPCAsKey(t, es.socketPath, callerPub, callerPriv,
		"console.detach", nil)
	if errObj, _ := detachResp["error"].(map[string]any); errObj != nil {
		t.Fatalf("VP-050 AC-002 — console.detach: unexpected error: %v", errObj)
	}
	detachResult, _ := detachResp["data"].(map[string]any)
	if detachResult == nil {
		t.Fatalf("VP-050 AC-002 — console.detach: data is nil; full response: %v", detachResp)
	}
	if got, _ := detachResult["session_name"].(string); got != "agent-01" {
		t.Errorf("VP-050 AC-002 — console.detach result.session_name: got %q; want %q (must echo the attached session)", got, "agent-01")
	}

	// AC-003: console.switch — re-attach to agent-01 then switch to agent-02.
	// BC-2.08.001 PC-3.
	// Non-tautological: state changed by attach RPC; switch verifies atomic transition.
	attachResp2 := sendAdminRPCAsKey(t, es.socketPath, callerPub, callerPriv,
		"console.attach", map[string]any{"session_name": "agent-01"})
	if errObj, _ := attachResp2["error"].(map[string]any); errObj != nil {
		t.Fatalf("VP-050 AC-003 setup — console.attach: unexpected error: %v", errObj)
	}

	switchResp := sendAdminRPCAsKey(t, es.socketPath, callerPub, callerPriv,
		"console.switch", map[string]any{"session_name": "agent-02"})
	if errObj, _ := switchResp["error"].(map[string]any); errObj != nil {
		t.Fatalf("VP-050 AC-003 — console.switch: unexpected error: %v", errObj)
	}
	switchResult, _ := switchResp["data"].(map[string]any)
	if switchResult == nil {
		t.Fatalf("VP-050 AC-003 — console.switch: data is nil; full response: %v", switchResp)
	}
	if got, _ := switchResult["session_name"].(string); got != "agent-02" {
		t.Errorf("VP-050 AC-003 — console.switch result.session_name: got %q; want %q", got, "agent-02")
	}

	// L1-C3 assertion: after switch, server-side state tracks agent-02 (not "" or agent-01).
	// Detach must echo agent-02, proving the state was SET to agent-02, not cleared.
	// Non-tautological: server-side state is verified via a real RPC, not by reading struct.
	detachResp2 := sendAdminRPCAsKey(t, es.socketPath, callerPub, callerPriv,
		"console.detach", nil)
	if errObj, _ := detachResp2["error"].(map[string]any); errObj != nil {
		t.Fatalf("VP-050 L1-C3 — post-switch detach: unexpected error: %v", errObj)
	}
	detachResult2, _ := detachResp2["data"].(map[string]any)
	if detachResult2 == nil {
		t.Fatalf("VP-050 L1-C3 — post-switch detach: data is nil; full response: %v", detachResp2)
	}
	if got, _ := detachResult2["session_name"].(string); got != "agent-02" {
		t.Errorf("VP-050 L1-C3 — post-switch detach.session_name: got %q; want %q (L1-C3: state must track new session after switch)", got, "agent-02")
	}
}

// TestConsoleRemote_E2E_AttachUnknown verifies that console.attach with an
// unknown session name returns E-SES-001 through the mgmt-plane (non-tautological:
// full RPC round-trip through mgmt.Server).
//
// BC-2.08.001 PC-1 EC-001.
func TestConsoleRemote_E2E_AttachUnknown(t *testing.T) {
	ks, consoleSrv := newConsoleE2EStack(t, "agent-01")

	callerPub, callerPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate caller keypair: %v", err)
	}
	ks.RegisterKey(zeroSVTN, callerPub, admission.RoleConsole)

	es := startConsoleE2EServer(t, ks, consoleSrv, callerPub)

	resp := sendAdminRPCAsKey(t, es.socketPath, callerPub, callerPriv,
		"console.attach", map[string]any{"session_name": "does-not-exist"})
	errObj, hasErr := resp["error"].(map[string]any)
	if !hasErr {
		t.Fatalf("BC-2.08.001 PC-1 EC-001 — expected error for unknown session; got success: %v", resp)
	}
	code, _ := errObj["code"].(string)
	if code != "E-SES-001" {
		t.Errorf("BC-2.08.001 PC-1 EC-001 — error code: got %q; want %q", code, "E-SES-001")
	}
}

// TestConsoleRemote_E2E_DetachNotAttached verifies that console.detach when no
// session is attached returns E-SES-004 through the mgmt-plane (non-tautological).
//
// BC-2.08.001 PC-2 EC-002.
func TestConsoleRemote_E2E_DetachNotAttached(t *testing.T) {
	ks, consoleSrv := newConsoleE2EStack(t)

	callerPub, callerPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate caller keypair: %v", err)
	}
	ks.RegisterKey(zeroSVTN, callerPub, admission.RoleConsole)

	es := startConsoleE2EServer(t, ks, consoleSrv, callerPub)

	resp := sendAdminRPCAsKey(t, es.socketPath, callerPub, callerPriv,
		"console.detach", nil)
	errObj, hasErr := resp["error"].(map[string]any)
	if !hasErr {
		t.Fatalf("BC-2.08.001 PC-2 EC-002 — expected error when not attached; got success: %v", resp)
	}
	code, _ := errObj["code"].(string)
	if code != "E-SES-004" {
		t.Errorf("BC-2.08.001 PC-2 EC-002 — error code: got %q; want %q", code, "E-SES-004")
	}
}

// TestConsoleRemote_E2E_AdmissionDenied verifies that console.attach with a
// key that does not have RoleControl or RoleConsole returns an error carrying
// E-ADM-006 wrapped in E-RPC-011 through the mgmt-plane (non-tautological; L1-C4).
//
// Wire envelope assertion (F-P2L2-003): sendAdminRPCAsKey lifts the inner
// domain code when the outer code is E-RPC-011 (see admin_handlers_e2e_test.go
// lines ~305-315). After lifting, errObj["code"] == "E-ADM-006" and
// errObj["message"] still contains "E-ADM-006:" — proving E-RPC-011 was the
// actual wire code (the lift was performed). Both codes are asserted below.
//
// BC-2.08.001 Inv-1; L1-C4; RULING-W6TB-C.
func TestConsoleRemote_E2E_AdmissionDenied(t *testing.T) {
	ks, consoleSrv := newConsoleE2EStack(t, "agent-01")

	// Register caller with RoleAccess (insufficient for console commands).
	accessPub, accessPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate access keypair: %v", err)
	}
	// Register in AdmittedKeySet with RoleAccess — verifyConsoleCallerRole must deny.
	ks.RegisterKey(zeroSVTN, accessPub, admission.RoleAccess)

	// Add callerPub to OperatorKeySet so mgmt.Server handshake admits it.
	es := startConsoleE2EServer(t, ks, consoleSrv, accessPub)

	resp := sendAdminRPCAsKey(t, es.socketPath, accessPub, accessPriv,
		"console.attach", map[string]any{"session_name": "agent-01"})
	errObj, hasErr := resp["error"].(map[string]any)
	if !hasErr {
		t.Fatalf("L1-C4 — expected E-ADM-006 for RoleAccess caller; got success: %v", resp)
	}

	// F-P2L2-003: assert both E-RPC-011 (wire envelope) and E-ADM-006 (domain code).
	// sendAdminRPCAsKey lifts the inner code into errObj["code"] when the wire code
	// was E-RPC-011; the message field retains "E-ADM-006: ..." proving the domain
	// error was embedded in the E-RPC-011 message (BC-2.08.001 Inv-1; RULING-W6TB-C).
	code, _ := errObj["code"].(string)
	if code != "E-ADM-006" {
		t.Errorf("L1-C4 F-P2L2-003 — lifted error code: got %q; want %q (E-RPC-011 must wrap E-ADM-006)", code, "E-ADM-006")
	}
	msg, _ := errObj["message"].(string)
	if !strings.Contains(msg, "E-ADM-006") {
		t.Errorf("L1-C4 F-P2L2-003 — E-RPC-011 message must embed E-ADM-006; got: %q", msg)
	}
}

// TestConsoleRemote_E2E_AdmissionDenied_SwitchAndDetach verifies that
// console.switch and console.detach with a key holding RoleAccess both return
// E-ADM-006 wrapped in E-RPC-011 (F-P2L2-004; BC-2.08.001 Inv-1; L1-C4).
//
// These complement TestConsoleRemote_E2E_AdmissionDenied (which covers attach)
// to ensure all three console commands enforce Tier-2 admission consistently.
func TestConsoleRemote_E2E_AdmissionDenied_SwitchAndDetach(t *testing.T) {
	tests := []struct {
		name    string
		command string
		args    map[string]any
	}{
		{
			name:    "switch_auth_denied",
			command: "console.switch",
			args:    map[string]any{"session_name": "agent-01"},
		},
		{
			name:    "detach_auth_denied",
			command: "console.detach",
			args:    nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ks, consoleSrv := newConsoleE2EStack(t, "agent-01")

			accessPub, accessPriv, err := ed25519.GenerateKey(rand.Reader)
			if err != nil {
				t.Fatalf("generate access keypair: %v", err)
			}
			// RoleAccess is insufficient — verifyConsoleCallerRole must deny (L1-C4).
			ks.RegisterKey(zeroSVTN, accessPub, admission.RoleAccess)

			es := startConsoleE2EServer(t, ks, consoleSrv, accessPub)

			resp := sendAdminRPCAsKey(t, es.socketPath, accessPub, accessPriv,
				tc.command, tc.args)
			errObj, hasErr := resp["error"].(map[string]any)
			if !hasErr {
				t.Fatalf("F-P2L2-004 %s — expected E-ADM-006 for RoleAccess caller; got success: %v", tc.command, resp)
			}

			// Both E-RPC-011 (wire envelope, lifted by sendAdminRPCAsKey) and
			// E-ADM-006 (domain code embedded in message) must be present.
			code, _ := errObj["code"].(string)
			if code != "E-ADM-006" {
				t.Errorf("F-P2L2-004 %s — lifted error code: got %q; want E-ADM-006", tc.command, code)
			}
			msg, _ := errObj["message"].(string)
			if !strings.Contains(msg, "E-ADM-006") {
				t.Errorf("F-P2L2-004 %s — E-RPC-011 message must embed E-ADM-006; got: %q", tc.command, msg)
			}
		})
	}
}

// TestConsoleRemote_E2E_ControlRoleAllowed verifies that a caller with RoleControl
// can call console commands (L1-C4: RoleControl OR RoleConsole allowed).
//
// BC-2.08.001 Inv-1; L1-C4.
func TestConsoleRemote_E2E_ControlRoleAllowed(t *testing.T) {
	ks, consoleSrv := newConsoleE2EStack(t, "agent-01")

	callerPub, callerPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate caller keypair: %v", err)
	}
	// Register caller with RoleControl — must be allowed for console operations.
	ks.RegisterKey(zeroSVTN, callerPub, admission.RoleControl)

	es := startConsoleE2EServer(t, ks, consoleSrv, callerPub)

	resp := sendAdminRPCAsKey(t, es.socketPath, callerPub, callerPriv,
		"console.attach", map[string]any{"session_name": "agent-01"})
	if errObj, _ := resp["error"].(map[string]any); errObj != nil {
		t.Errorf("L1-C4 — RoleControl caller should be allowed; got error: %v", errObj)
	}
}
