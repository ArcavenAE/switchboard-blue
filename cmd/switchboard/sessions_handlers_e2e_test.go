package main

// End-to-end test for the sessions.status RPC handler
// (S-BL.CONSOLE-OBS; BC-2.06.001 v1.7 PC-5 console-half; BC-2.06.002 v1.4 PC-3;
// DRIFT-001b + DRIFT-002).
//
// Exercises the full path from a real mgmt.Server-authenticated caller
// through the daemon-side BuildSessionsHandlers wiring into
// Publisher.HandleSessionsStatus, verifying:
//   - AC-004 (RULING-W6TB-C): new sessions.status RPC surface exists and returns
//     per-session {name, published_at, quality, miss_count} tuples for the
//     live session set.
//   - AC-005 (RULING-W6TB-C): the miss_count value surfaced in the response
//     traces back through the Publisher's per-session QualityIndicator
//     wrapper to the LIFETIME miss counter exposed via
//     QualityIndicator.MissCount() (DRIFT-002 closure).
//   - L1-C4 (Tier-2 admission): callers whose keys are NOT registered with
//     RoleControl or RoleConsole receive E-ADM-006, even when they pass the
//     mgmt.Server handshake.

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/mgmt"
	"github.com/arcavenae/switchboard/internal/session"
)

// newSessionsE2EStack constructs an AdmittedKeySet + sessionQualitySource
// pair with the requested session names pre-published on an internal
// Publisher that fires the source's OnPublished hook. Only ks + src are
// returned because the tests drive observations exclusively through src per
// the ARCH-08 §6.6-preserving hook wiring; Publisher is a private seeding
// mechanism, not part of the sessions.status handler surface.
func newSessionsE2EStack(t *testing.T, sessionNames ...string) (
	*admission.AdmittedKeySet, *sessionQualitySource,
) {
	t.Helper()
	ks := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(ks)
	src := newSessionQualitySourceFromPublisher(pub)
	for _, name := range sessionNames {
		if err := pub.Publish(name); err != nil {
			t.Fatalf("newSessionsE2EStack: Publish %q: %v", name, err)
		}
	}
	return ks, src
}

// startSessionsE2EServer starts a real mgmt.Server with BuildSessionsHandlers
// registered. callerPubs is added to the OperatorKeySet so mgmt.Server admits
// the caller during the Ed25519 challenge-response handshake.
//
// BuildSessionsHandlers reads through src (the boundary registry), not
// through the Publisher — Publisher only feeds the source via hook callbacks
// installed in newSessionQualitySourceFromPublisher. This matches the
// runConsole wiring in mgmt_wire.go.
func startSessionsE2EServer(
	t *testing.T,
	ks *admission.AdmittedKeySet,
	src *sessionQualitySource,
	callerPubs ...ed25519.PublicKey,
) *e2eServer {
	t.Helper()

	_, daemonPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("startSessionsE2EServer: generate daemon keypair: %v", err)
	}
	ops := mgmt.NewOperatorKeySet(callerPubs)
	es := startE2EServerWithOps(t, BuildSessionsHandlers(src, ks), daemonPriv, ops)
	time.Sleep(10 * time.Millisecond)
	return es
}

// TestSessionsStatus_E2E_AllSessions_QualityAndMissCount is the primary
// evidence for AC-004 (console session-list surface exists) and AC-005
// (miss_count observable from the mgmt-plane) end-to-end.
//
// Non-tautological assertion strategy:
//   - Two sessions are published; only ONE receives observations.
//   - The observed session takes 3 misses on a green baseline, forcing a
//     downgrade to yellow (BC-2.06.002 PC-2) and setting MissCount=3.
//   - The unobserved session must remain quality "pending" with miss_count 0.
//   - The RPC response must show BOTH sessions in name-sorted order, with
//     each carrying its correct quality and miss_count.
//
// Traces to BC-2.06.001 v1.7 PC-5 console-half; BC-2.06.002 v1.4 PC-3;
// DRIFT-001b + DRIFT-002; RULING-W6TB-C AC-004 + AC-005; L1-C4.
func TestSessionsStatus_E2E_AllSessions_QualityAndMissCount(t *testing.T) {
	// Two sessions: agent-01 (will get observations) + agent-02 (stays pending).
	ks, src := newSessionsE2EStack(t, "agent-01", "agent-02")

	// Drive observations on agent-01 through the boundary source:
	//   1) One green measurement (moves out of pending, MissCount stays 0)
	//   2) Three consecutive misses (green ⇒ yellow; MissCount ⇒ 3)
	if err := src.OnSessionMeasurement("agent-01", 50, 1); err != nil {
		t.Fatalf("seed: OnSessionMeasurement: %v", err)
	}
	for i := 0; i < 3; i++ {
		if err := src.OnSessionMissingFrame("agent-01"); err != nil {
			t.Fatalf("seed: OnSessionMissingFrame %d: %v", i, err)
		}
	}

	callerPub, callerPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate caller keypair: %v", err)
	}
	// L1-C4: register with RoleConsole so verifyConsoleCallerRole admits us.
	ks.RegisterKey(zeroSVTN, callerPub, admission.RoleConsole)

	es := startSessionsE2EServer(t, ks, src, callerPub)

	// Empty args ⇒ "all sessions" query.
	resp := sendAdminRPCAsKey(t, es.socketPath, callerPub, callerPriv,
		"sessions.status", nil)

	if errObj, _ := resp["error"].(map[string]any); errObj != nil {
		t.Fatalf("sessions.status: unexpected error: %v", errObj)
	}
	data, _ := resp["data"].(map[string]any)
	if data == nil {
		t.Fatalf("sessions.status: data is nil; full response: %v", resp)
	}
	sessionsRaw, _ := data["sessions"].([]any)
	if len(sessionsRaw) != 2 {
		t.Fatalf("sessions.status: sessions len = %d; want 2; data = %v",
			len(sessionsRaw), data)
	}

	// The Publisher orders by name, so agent-01 comes first.
	first, _ := sessionsRaw[0].(map[string]any)
	second, _ := sessionsRaw[1].(map[string]any)
	if first == nil || second == nil {
		t.Fatalf("sessions.status: entries not maps: %v", sessionsRaw)
	}

	// AC-004: entries carry per-session Name + Quality + MissCount fields.
	if got, _ := first["name"].(string); got != "agent-01" {
		t.Errorf("sessions[0].name = %q; want %q", got, "agent-01")
	}
	if got, _ := first["quality"].(string); got != "yellow" {
		t.Errorf("sessions[0].quality = %q; want %q "+
			"(3 misses on green ⇒ yellow per BC-2.06.002 PC-2)", got, "yellow")
	}
	// JSON numbers decode as float64 via map[string]any; compare accordingly.
	if got, _ := first["miss_count"].(float64); got != 3 {
		t.Errorf("sessions[0].miss_count = %v; want 3 "+
			"(AC-005: lifetime MissCount surfaced via mgmt-plane; DRIFT-002)", got)
	}
	if got, _ := first["published_at"].(string); got == "" {
		t.Errorf("sessions[0].published_at is empty; want RFC 3339 UTC timestamp")
	}

	if got, _ := second["name"].(string); got != "agent-02" {
		t.Errorf("sessions[1].name = %q; want %q", got, "agent-02")
	}
	if got, _ := second["quality"].(string); got != "pending" {
		t.Errorf("sessions[1].quality = %q; want %q "+
			"(no observations yet ⇒ pending)", got, "pending")
	}
	if got, _ := second["miss_count"].(float64); got != 0 {
		t.Errorf("sessions[1].miss_count = %v; want 0", got)
	}
}

// TestSessionsStatus_E2E_SingleSession_ByName exercises the "by name" query
// shape returning exactly one entry for the requested session.
func TestSessionsStatus_E2E_SingleSession_ByName(t *testing.T) {
	ks, src := newSessionsE2EStack(t, "agent-01", "agent-02")

	callerPub, callerPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate caller keypair: %v", err)
	}
	ks.RegisterKey(zeroSVTN, callerPub, admission.RoleConsole)

	es := startSessionsE2EServer(t, ks, src, callerPub)

	resp := sendAdminRPCAsKey(t, es.socketPath, callerPub, callerPriv,
		"sessions.status", map[string]any{"session_name": "agent-02"})

	if errObj, _ := resp["error"].(map[string]any); errObj != nil {
		t.Fatalf("sessions.status by-name: unexpected error: %v", errObj)
	}
	data, _ := resp["data"].(map[string]any)
	sessionsRaw, _ := data["sessions"].([]any)
	if len(sessionsRaw) != 1 {
		t.Fatalf("sessions.status by-name: sessions len = %d; want 1", len(sessionsRaw))
	}
	only, _ := sessionsRaw[0].(map[string]any)
	if got, _ := only["name"].(string); got != "agent-02" {
		t.Errorf("sessions[0].name = %q; want %q", got, "agent-02")
	}
}

// TestSessionsStatus_E2E_UnknownSession_ESES001 verifies that a by-name query
// for an unknown session returns an E-SES-001 error envelope on the wire —
// not a silent empty list.
func TestSessionsStatus_E2E_UnknownSession_ESES001(t *testing.T) {
	ks, src := newSessionsE2EStack(t, "agent-01")

	callerPub, callerPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate caller keypair: %v", err)
	}
	ks.RegisterKey(zeroSVTN, callerPub, admission.RoleConsole)

	es := startSessionsE2EServer(t, ks, src, callerPub)

	resp := sendAdminRPCAsKey(t, es.socketPath, callerPub, callerPriv,
		"sessions.status", map[string]any{"session_name": "does-not-exist"})

	errObj, _ := resp["error"].(map[string]any)
	if errObj == nil {
		t.Fatalf("sessions.status by-name(unknown): expected error envelope; got %v", resp)
	}
	msg, _ := errObj["message"].(string)
	if !containsString(msg, "E-SES-001") {
		t.Errorf("error message = %q; want to include E-SES-001", msg)
	}
	if !containsString(msg, "does-not-exist") {
		t.Errorf("error message = %q; want to include session name", msg)
	}
}

// TestSessionsStatus_E2E_AdmissionDenied_E_ADM_006 verifies that a caller
// with a key in the OperatorKeySet (Layer 1 passes handshake) but NOT in the
// AdmittedKeySet with RoleControl/RoleConsole (Layer 2 denies) receives
// E-ADM-006. This is the L1-C4 fail-closed contract.
//
// Non-tautological: two separate key sets (Layer 1 admits any handshake-valid
// caller; Layer 2 admits only RoleControl/RoleConsole). The handler MUST reach
// Layer 2 check before reading the session state.
func TestSessionsStatus_E2E_AdmissionDenied_E_ADM_006(t *testing.T) {
	ks, src := newSessionsE2EStack(t, "agent-01")

	callerPub, callerPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate caller keypair: %v", err)
	}
	// Deliberately DO NOT call ks.RegisterKey — Layer 2 must fail.
	// The mgmt.Server handshake will still succeed because callerPub is in
	// the OperatorKeySet (Layer 1) passed to startSessionsE2EServer.

	es := startSessionsE2EServer(t, ks, src, callerPub)

	resp := sendAdminRPCAsKey(t, es.socketPath, callerPub, callerPriv,
		"sessions.status", nil)

	errObj, _ := resp["error"].(map[string]any)
	if errObj == nil {
		t.Fatalf("sessions.status without RoleConsole: expected error; got %v", resp)
	}
	msg, _ := errObj["message"].(string)
	if !containsString(msg, "E-ADM-006") {
		t.Errorf("error message = %q; want to include E-ADM-006 "+
			"(Tier-2 admission must deny keys absent from AdmittedKeySet)", msg)
	}
}

// containsString is a tiny local helper because importing "strings" for a
// single Contains call in a test file feels heavy — matches the style used
// in sessions_status_test.go.
func containsString(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
