// Package session_test — Tier-2 per-session authorization and read-only
// enforcement tests (S-3.03; BC-2.04.005; BC-2.05.003).
//
// Red Gate: all tests in this file must FAIL before implementation of
// internal/session/auth.go (BC-5.38.001). The stubs panic("not implemented").
//
// Test-to-AC traceability:
//   - AC-001 → TestSessionAuth_Authorize_RegisteredKey_Succeeds
//   - AC-002 → TestSessionAuth_Authorize_UnregisteredKey_ErrAdmSix
//   - AC-003 → TestSessionAuth_RouterHasNoTier2State
//   - AC-004 → TestSessionAuth_Authorize_PerSession_NoSpillover
//   - AC-005 → TestReadOnlyConsole_UpstreamRejected_DownstreamContinues
//   - AC-006 → TestReadOnlyConsole_EmptyTickAccepted
//   - Edge cases: EC-002, EC-003, EC-004
package session_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/session"
)

// newAuthPublisher returns a Publisher with the named sessions pre-published.
// Helper used by AC-005/AC-006 integration tests that need a live session.
// Named "newAuthPublisher" (not "newTestPublisher") to avoid collision with
// the homonymous helper in session_test.go which has a different signature.
func newAuthPublisher(t *testing.T, sessionNames ...string) *session.Publisher {
	t.Helper()
	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)
	for _, name := range sessionNames {
		if err := pub.Publish(name); err != nil {
			t.Fatalf("newAuthPublisher: Publish(%q): %v", name, err)
		}
	}
	return pub
}

// mustAttachConsole calls AccessNode.Attach and fatals on error. Named
// "mustAttachConsole" to avoid collision if session_test.go ever declares
// a similar helper. t.Helper() ensures failures point to the caller.
func mustAttachConsole(t *testing.T, an *session.AccessNode, key session.ConsoleKey, sessionName string) <-chan frame.OuterHeader { //nolint:unparam // sessionName is a genuine parameter; current callers happen to use "agent-01" but the helper is reused across tests
	t.Helper()
	ds, _, err := an.Attach(key, sessionName)
	if err != nil {
		t.Fatalf("mustAttachConsole(%q, %q): %v", key, sessionName, err)
	}
	return ds
}

// drainN receives exactly n frames from ch, fataling if the channel is closed
// before n frames arrive. Returns the received frames.
func drainN(t *testing.T, ch <-chan frame.OuterHeader, n int) []frame.OuterHeader { //nolint:unparam // n is semantically variable; current callers all pass 1 but the helper is reused across tests
	t.Helper()
	out := make([]frame.OuterHeader, 0, n)
	for range n {
		hdr, ok := <-ch
		if !ok {
			t.Fatalf("drainN: downstream channel closed after %d/%d frames", len(out), n)
		}
		out = append(out, hdr)
	}
	return out
}

// ---------------------------------------------------------------------------
// AC-001: TestSessionAuth_Authorize_RegisteredKey_Succeeds
// BC-2.05.003 PC-1 — registered key → nil error and correct role returned.
// Exercises RoleFull and RoleReadOnly variants.
// ---------------------------------------------------------------------------

// TestSessionAuth_Authorize_RegisteredKey_Succeeds verifies that
// SessionAuth.Authorize returns (role, nil) when the console's key is
// registered in the named session's authorization list.
//
// Exercises VP-012 (SessionAuth rejects unauthorized console key — the
// positive complement: authorized key is not rejected).
// BC-2.05.003 PC-1.
func TestSessionAuth_Authorize_RegisteredKey_Succeeds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		key         session.ConsoleKey
		sessionName string
		role        session.Role
	}{
		{
			name:        "full-access key",
			key:         session.ConsoleKey("console-full-001"),
			sessionName: "agent-01",
			role:        session.RoleFull,
		},
		{
			name:        "read-only key",
			key:         session.ConsoleKey("console-ro-001"),
			sessionName: "agent-01",
			role:        session.RoleReadOnly,
		},
		{
			name:        "full-access on different session",
			key:         session.ConsoleKey("console-full-002"),
			sessionName: "agent-02",
			role:        session.RoleFull,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sa := session.NewSessionAuth()
			sa.RegisterKey(tc.sessionName, tc.key, tc.role)

			got, err := sa.Authorize(tc.key, tc.sessionName)
			if err != nil {
				t.Fatalf("Authorize: unexpected error: %v", err)
			}
			if got != tc.role {
				t.Errorf("Authorize: got role %v, want %v", got, tc.role)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// AC-002: TestSessionAuth_Authorize_UnregisteredKey_ErrAdmSix
// BC-2.05.003 PC-2 — unregistered key → ErrSessionAuthDenied (E-ADM-006).
// ---------------------------------------------------------------------------

// TestSessionAuth_Authorize_UnregisteredKey_ErrAdmSix verifies that
// SessionAuth.Authorize returns ErrSessionAuthDenied when the console's key
// is NOT in the session's authorization list (E-ADM-006).
//
// BC-2.05.003 PC-2; VP-012.
func TestSessionAuth_Authorize_UnregisteredKey_ErrAdmSix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		registered  []session.ConsoleKey // keys registered in the session
		queried     session.ConsoleKey   // key being authorized
		sessionName string
	}{
		{
			name:        "completely empty auth list",
			registered:  nil,
			queried:     session.ConsoleKey("console-unknown"),
			sessionName: "agent-01",
		},
		{
			name:        "different key registered",
			registered:  []session.ConsoleKey{"console-other"},
			queried:     session.ConsoleKey("console-unknown"),
			sessionName: "agent-01",
		},
		{
			name:        "key registered on different session only",
			registered:  []session.ConsoleKey{"console-x"},
			queried:     session.ConsoleKey("console-x"),
			sessionName: "agent-02", // registered on agent-01, queried on agent-02
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sa := session.NewSessionAuth()
			// Register any keys on agent-01 (not agent-02 for the spillover case).
			for _, k := range tc.registered {
				sa.RegisterKey("agent-01", k, session.RoleFull)
			}

			_, err := sa.Authorize(tc.queried, tc.sessionName)
			if err == nil {
				t.Fatal("Authorize: expected ErrSessionAuthDenied, got nil")
			}
			if !errors.Is(err, session.ErrSessionAuthDenied) {
				t.Errorf("Authorize: got %v, want errors.Is(err, ErrSessionAuthDenied)", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// AC-003: TestSessionAuth_RouterHasNoTier2State
// BC-2.05.003 PC-3 + invariant 1; VP-012 code-audit.
//
// This test is a static code-audit assertion: it scans every .go source file
// under internal/routing/ and asserts that none import or reference the
// per-session authorization symbols (SessionAuth, ErrSessionAuthDenied,
// ErrUpstreamReadOnly, authEntry). The test fails at Red Gate only when those
// symbols are actually wired into routing — which they must never be.
//
// The test also asserts that internal/routing/ contains no import path
// referencing "internal/session", enforcing the layer boundary (ARCH-08 §6.6).
//
// NOTE: This test will PASS at Red Gate (routing.go has no session imports),
// and must continue to pass after implementation. It is a regression guard,
// not a Red Gate failure. The story's AC-003 is a "code-audit assertion" test
// by design; including it here ensures it is always exercised.
// ---------------------------------------------------------------------------

// TestSessionAuth_RouterHasNoTier2State verifies that internal/routing/ has
// no data structures or imports related to per-session Tier-2 authorization
// (DI-010; BC-2.05.003 invariant 1; VP-012).
func TestSessionAuth_RouterHasNoTier2State(t *testing.T) {
	t.Parallel()

	// Find the module root by walking up from this test file's directory.
	// We use os.Getwd() which gives the package directory when `go test` is run.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}

	// Walk up to find go.mod (module root).
	moduleRoot := findModuleRoot(t, wd)
	routingDir := filepath.Join(moduleRoot, "internal", "routing")

	// Symbols that must never appear in routing source.
	forbidden := []string{
		"SessionAuth",
		"ErrSessionAuthDenied",
		"ErrUpstreamReadOnly",
		"authEntry",
		`"github.com/arcavenae/switchboard/internal/session"`,
	}

	entries, err := os.ReadDir(routingDir)
	if err != nil {
		t.Fatalf("ReadDir(%q): %v", routingDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Only inspect non-test .go source files (test files might legitimately
		// import session for integration testing).
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		path := filepath.Join(routingDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%q): %v", path, err)
		}
		content := string(data)

		for _, sym := range forbidden {
			if strings.Contains(content, sym) {
				t.Errorf("AC-003 VIOLATION: %q contains forbidden Tier-2 symbol %q; "+
					"per-session auth must live in internal/session only (DI-010; BC-2.05.003 invariant 1)",
					path, sym)
			}
		}
	}
}

// findModuleRoot walks up from dir until it finds a go.mod file.
func findModuleRoot(t *testing.T, dir string) string {
	t.Helper()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("findModuleRoot: could not locate go.mod")
		}
		dir = parent
	}
}

// ---------------------------------------------------------------------------
// AC-004: TestSessionAuth_Authorize_PerSession_NoSpillover
// BC-2.05.003 PC-4 — authorization is per-session, no cross-session spillover.
// ---------------------------------------------------------------------------

// TestSessionAuth_Authorize_PerSession_NoSpillover verifies that a console
// authorized for session-A is NOT automatically authorized for session-B.
//
// BC-2.05.003 PC-4; EC-003 (cross-session request rejected).
func TestSessionAuth_Authorize_PerSession_NoSpillover(t *testing.T) {
	t.Parallel()

	const (
		sessionA = "agent-01"
		sessionB = "agent-02"
	)
	key := session.ConsoleKey("console-alpha")

	sa := session.NewSessionAuth()
	// Register key on session-A only.
	sa.RegisterKey(sessionA, key, session.RoleFull)

	// Must succeed for session-A.
	role, err := sa.Authorize(key, sessionA)
	if err != nil {
		t.Fatalf("Authorize(%q, %q): unexpected error: %v", key, sessionA, err)
	}
	if role != session.RoleFull {
		t.Errorf("Authorize(%q, %q): got role %v, want RoleFull", key, sessionA, role)
	}

	// Must fail for session-B — authorization MUST NOT spill over.
	_, err = sa.Authorize(key, sessionB)
	if err == nil {
		t.Fatalf("Authorize(%q, %q): expected ErrSessionAuthDenied but got nil; "+
			"authorization must not spill across sessions (BC-2.05.003 PC-4)", key, sessionB)
	}
	if !errors.Is(err, session.ErrSessionAuthDenied) {
		t.Errorf("Authorize(%q, %q): got %v, want ErrSessionAuthDenied", key, sessionB, err)
	}
}

// ---------------------------------------------------------------------------
// EC-002: TestSessionAuth_EmptyAuthList_AllRejected
// BC-2.05.003 EC-002 — empty auth list rejects all attach requests.
// ---------------------------------------------------------------------------

// TestSessionAuth_EmptyAuthList_AllRejected verifies that when no keys are
// registered for a session, every authorization request is rejected with
// ErrSessionAuthDenied (BC-2.05.003 EC-002).
func TestSessionAuth_EmptyAuthList_AllRejected(t *testing.T) {
	t.Parallel()

	sa := session.NewSessionAuth()
	// No RegisterKey calls — auth list is empty.

	keys := []session.ConsoleKey{"console-a", "console-b", "console-c"}
	for _, k := range keys {
		t.Run(string(k), func(t *testing.T) {
			t.Parallel()
			_, err := sa.Authorize(k, "agent-01")
			if err == nil {
				t.Fatalf("Authorize(%q): expected ErrSessionAuthDenied for empty auth list, got nil", k)
			}
			if !errors.Is(err, session.ErrSessionAuthDenied) {
				t.Errorf("Authorize(%q): got %v, want ErrSessionAuthDenied", k, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// EC-003: TestSessionAuth_CrossSession_Rejected
// BC-2.05.003 EC-003 — cross-session authorization request is rejected.
// (Overlaps with AC-004; this variant uses Allow, exercising the full path.)
// ---------------------------------------------------------------------------

// TestSessionAuth_CrossSession_Rejected verifies that Allow rejects a
// payload-bearing keystroke when the key is registered on agent-01 but the
// frame is sent for agent-02 (BC-2.05.003 EC-003; AC-004 via Allow path).
func TestSessionAuth_CrossSession_Rejected(t *testing.T) {
	t.Parallel()

	const (
		sessionA = "agent-01"
		sessionB = "agent-02"
	)
	key := session.ConsoleKey("console-crosssess")

	sa := session.NewSessionAuth()
	sa.RegisterKey(sessionA, key, session.RoleFull)

	// Allow on session-A should succeed for a full-access key.
	if err := sa.Allow(key, sessionA, []byte("some keystroke")); err != nil {
		t.Fatalf("Allow(%q, %q, payload): unexpected error %v", key, sessionA, err)
	}

	// Allow on session-B must be rejected — key not registered there.
	err := sa.Allow(key, sessionB, []byte("some keystroke"))
	if err == nil {
		t.Fatalf("Allow(%q, %q, payload): expected error, got nil; "+
			"cross-session access must be rejected (BC-2.05.003 EC-003)", key, sessionB)
	}
	if !errors.Is(err, session.ErrSessionAuthDenied) {
		t.Errorf("Allow(%q, %q, payload): got %v, want ErrSessionAuthDenied", key, sessionB, err)
	}
}

// ---------------------------------------------------------------------------
// AC-005: TestReadOnlyConsole_UpstreamRejected_DownstreamContinues
// BC-2.04.005 PC-3 + PC-4; VP-013; VP-035.
//
// MUTATION-RESISTANCE CONTRACT:
// This test will fail if the read-only enforcement check in Allow is absent
// or inverted because:
//   1. It asserts errors.Is(sendErr, ErrUpstreamReadOnly) — if the check
//      were removed, sendErr would be nil and the test fails at the assertion.
//   2. It then delivers downstream frames and asserts the subscription is
//      still alive — proving the rejection did NOT terminate the channel.
//
// The combination of (1) explicit error check and (2) downstream liveness
// proof is the mutation-resistant structure the S-3.02 adversary required.
// ---------------------------------------------------------------------------

// TestReadOnlyConsole_UpstreamRejected_DownstreamContinues verifies that a
// read-only console's upstream keystroke is rejected with ErrUpstreamReadOnly
// (E-ADM-007), and that the rejection does NOT terminate the console's
// downstream subscription (BC-2.04.005 PC-3 + PC-4).
//
// The test wires SessionAuth as the Authorizer in a live AccessNode and uses
// the real AccessNode.SendKeystroke + AccessNode.DeliverFrame path.
func TestReadOnlyConsole_UpstreamRejected_DownstreamContinues(t *testing.T) {
	t.Parallel()

	const (
		sessionName = "agent-01"
		roKey       = session.ConsoleKey("console-readonly-001")
	)

	// Build SessionAuth and register the read-only console.
	sa := session.NewSessionAuth()
	sa.RegisterKey(sessionName, roKey, session.RoleReadOnly)

	// Build a live AccessNode with SessionAuth wired as the Authorizer.
	pub := newAuthPublisher(t, sessionName)
	an := session.NewAccessNode(pub, sa, session.WithKeystrokeSink(session.NoOpSink{}))

	// Attach the read-only console.
	ds := mustAttachConsole(t, an, roKey, sessionName)

	// Attempt to send a keystroke (payload-bearing upstream frame).
	payload := []byte("a") // canonical test vector: BC-2.04.005 §Test Vectors row 2
	sendErr := an.SendKeystroke(roKey, sessionName, payload)

	// ASSERTION 1 (mutation-resistance): the error must be ErrUpstreamReadOnly.
	// If the enforcement check is absent, sendErr is nil and this fatals.
	if sendErr == nil {
		t.Fatal("SendKeystroke from read-only console: expected ErrUpstreamReadOnly, got nil; " +
			"read-only enforcement is not wired (BC-2.04.005 PC-3)")
	}
	if !errors.Is(sendErr, session.ErrUpstreamReadOnly) {
		t.Errorf("SendKeystroke: got %v, want errors.Is(err, ErrUpstreamReadOnly)", sendErr)
	}

	// ASSERTION 2 (downstream liveness): the rejection must NOT close the
	// downstream channel. Deliver a frame and assert it is received.
	hdr := frame.OuterHeader{} // zero-value frame (type byte 0x00; valid for liveness)
	an.DeliverFrame(hdr)

	frames := drainN(t, ds, 1)
	if len(frames) != 1 {
		t.Fatalf("downstream: expected 1 frame after upstream rejection, got %d; "+
			"rejection must not terminate downstream subscription (BC-2.04.005 PC-4)", len(frames))
	}

	// Detach cleanly to avoid channel leak.
	if err := an.Detach(roKey, sessionName); err != nil {
		t.Errorf("Detach: %v", err)
	}
}

// ---------------------------------------------------------------------------
// AC-006: TestReadOnlyConsole_EmptyTickAccepted
// BC-2.04.005 PC-3 + EC-004 — empty-tick (zero-length payload) accepted.
//
// MUTATION-RESISTANCE CONTRACT:
// This test is mutation-resistant because it:
//   1. Sends a payload-bearing frame first and asserts it IS rejected
//      (ErrUpstreamReadOnly). This ensures the enforcement check is active.
//   2. Then sends an empty-tick frame and asserts it is accepted (nil error).
//
// If the enforcement check were removed entirely, both calls would return nil
// and assertion (1) would fail. If the check were inverted (reject empty,
// accept non-empty), assertion (1) would succeed but (2) would fail.
// The two-assertion structure prevents vacuous passing.
// ---------------------------------------------------------------------------

// TestReadOnlyConsole_EmptyTickAccepted verifies that an empty-tick frame
// (zero-length payload, a liveness probe) from a read-only console is accepted
// by the access node, while a payload-bearing frame from the same console is
// correctly rejected (BC-2.04.005 EC-004; AC-006).
func TestReadOnlyConsole_EmptyTickAccepted(t *testing.T) {
	t.Parallel()

	const (
		sessionName = "agent-01"
		roKey       = session.ConsoleKey("console-readonly-002")
	)

	sa := session.NewSessionAuth()
	sa.RegisterKey(sessionName, roKey, session.RoleReadOnly)

	pub := newAuthPublisher(t, sessionName)
	an := session.NewAccessNode(pub, sa, session.WithKeystrokeSink(session.NoOpSink{}))

	_ = mustAttachConsole(t, an, roKey, sessionName)

	// STEP 1: Send a non-empty payload first. This MUST be rejected.
	// Failing to reject it would mean the enforcement check is absent,
	// which would make the empty-tick acceptance in STEP 2 meaningless.
	payloadErr := an.SendKeystroke(roKey, sessionName, []byte("keystroke"))
	if payloadErr == nil {
		t.Fatal("SendKeystroke (payload-bearing): expected ErrUpstreamReadOnly, got nil; " +
			"enforcement check appears absent — empty-tick acceptance is not meaningful without it")
	}
	if !errors.Is(payloadErr, session.ErrUpstreamReadOnly) {
		t.Errorf("SendKeystroke (payload-bearing): got %v, want ErrUpstreamReadOnly", payloadErr)
	}

	// STEP 2: Send an empty-tick frame (nil/zero-length payload). MUST succeed.
	emptyErr := an.SendKeystroke(roKey, sessionName, []byte{})
	if emptyErr != nil {
		t.Errorf("SendKeystroke (empty-tick): expected nil, got %v; "+
			"empty-tick frames must be accepted from read-only consoles (BC-2.04.005 EC-004)", emptyErr)
	}

	// Also test with a nil payload (equivalent to empty-tick).
	nilErr := an.SendKeystroke(roKey, sessionName, nil)
	if nilErr != nil {
		t.Errorf("SendKeystroke (nil payload): expected nil, got %v; "+
			"nil payload is an empty-tick and must be accepted (BC-2.04.005 EC-004)", nilErr)
	}

	if err := an.Detach(roKey, sessionName); err != nil {
		t.Errorf("Detach: %v", err)
	}
}

// ---------------------------------------------------------------------------
// EC-004: TestReadOnlyConsole_FullAndReadOnly_BothAttached
// BC-2.04.005 EC-001 — full-access and read-only consoles both attached.
// Full-access keystrokes forwarded; read-only keystrokes rejected; both
// receive the same downstream output.
// ---------------------------------------------------------------------------

// TestReadOnlyConsole_FullAndReadOnly_BothAttached verifies the mixed-console
// scenario: a full-access console and a read-only console are both attached to
// the same session. Full-access keystrokes are forwarded; read-only keystrokes
// are rejected with ErrUpstreamReadOnly; both consoles receive identical
// downstream frames (BC-2.04.005 EC-001).
func TestReadOnlyConsole_FullAndReadOnly_BothAttached(t *testing.T) {
	t.Parallel()

	const sessionName = "agent-01"
	const fullKey = session.ConsoleKey("console-full-ec4")
	const roKey = session.ConsoleKey("console-ro-ec4")

	// Track what the sink receives to confirm full-access keystrokes forwarded.
	var sinkReceived [][]byte
	sink := &captureSink{received: &sinkReceived}

	sa := session.NewSessionAuth()
	sa.RegisterKey(sessionName, fullKey, session.RoleFull)
	sa.RegisterKey(sessionName, roKey, session.RoleReadOnly)

	pub := newAuthPublisher(t, sessionName)
	an := session.NewAccessNode(pub, sa, session.WithKeystrokeSink(sink))

	// Attach both consoles.
	dsFull := mustAttachConsole(t, an, fullKey, sessionName)
	dsRO := mustAttachConsole(t, an, roKey, sessionName)

	// Full-access console sends a keystroke — must be forwarded.
	fullPayload := []byte("hello")
	if err := an.SendKeystroke(fullKey, sessionName, fullPayload); err != nil {
		t.Errorf("SendKeystroke (full-access): unexpected error: %v", err)
	}

	// Verify the sink received the full-access keystroke.
	if len(sinkReceived) != 1 {
		t.Errorf("sink: got %d calls, want 1 (full-access keystroke must be forwarded)", len(sinkReceived))
	} else if !bytes.Equal(sinkReceived[0], fullPayload) {
		t.Errorf("sink: got payload %q, want %q", sinkReceived[0], fullPayload)
	}

	// Read-only console sends a keystroke — must be rejected with ErrUpstreamReadOnly.
	roErr := an.SendKeystroke(roKey, sessionName, []byte("blocked"))
	if roErr == nil {
		t.Fatal("SendKeystroke (read-only): expected ErrUpstreamReadOnly, got nil")
	}
	if !errors.Is(roErr, session.ErrUpstreamReadOnly) {
		t.Errorf("SendKeystroke (read-only): got %v, want ErrUpstreamReadOnly", roErr)
	}

	// Sink must still have only 1 call — read-only keystroke must NOT have been forwarded.
	if len(sinkReceived) != 1 {
		t.Errorf("sink: got %d calls after read-only send, want still 1 "+
			"(read-only keystroke must not reach the sink)", len(sinkReceived))
	}

	// Deliver a downstream frame — both consoles must receive it.
	hdr := frame.OuterHeader{}
	an.DeliverFrame(hdr)

	fullFrames := drainN(t, dsFull, 1)
	roFrames := drainN(t, dsRO, 1)

	if len(fullFrames) != 1 {
		t.Errorf("full-access downstream: got %d frames, want 1", len(fullFrames))
	}
	if len(roFrames) != 1 {
		t.Errorf("read-only downstream: got %d frames, want 1", len(roFrames))
	}
	if len(fullFrames) > 0 && len(roFrames) > 0 && fullFrames[0] != roFrames[0] {
		t.Errorf("downstream frames differ: full=%v ro=%v; "+
			"both consoles must receive identical downstream output (BC-2.04.005 invariant 3)",
			fullFrames[0], roFrames[0])
	}

	if err := an.Detach(fullKey, sessionName); err != nil {
		t.Errorf("Detach full-access: %v", err)
	}
	if err := an.Detach(roKey, sessionName); err != nil {
		t.Errorf("Detach read-only: %v", err)
	}
}

// captureSink is a KeystrokeSink that records every payload it receives.
// Not safe for concurrent use without external synchronization, but EC-004
// drives all sends sequentially so no mutex is needed here.
type captureSink struct {
	received *[][]byte
}

func (c *captureSink) SendInput(payload []byte) error {
	// Copy the payload so the test assertion is not affected by later mutations.
	cp := make([]byte, len(payload))
	copy(cp, payload)
	*c.received = append(*c.received, cp)
	return nil
}

// ---------------------------------------------------------------------------
// SessionAuth.Allow direct-path tests (table-driven)
// Exercises the full Allow decision matrix: auth denied, read-only payload
// rejected, read-only empty-tick accepted, full-access accepted.
// ---------------------------------------------------------------------------

// TestSessionAuth_Allow_DecisionMatrix exercises SessionAuth.Allow across the
// full authorization decision matrix. Used in conjunction with the AccessNode
// integration tests above; this unit test verifies Allow in isolation without
// the AccessNode wrapper, so a regression in Allow is localised here rather
// than only caught by integration tests.
func TestSessionAuth_Allow_DecisionMatrix(t *testing.T) {
	t.Parallel()

	const (
		sess    = "agent-01"
		fullKey = session.ConsoleKey("allow-full")
		roKey   = session.ConsoleKey("allow-ro")
		noneKey = session.ConsoleKey("allow-none")
	)

	sa := session.NewSessionAuth()
	sa.RegisterKey(sess, fullKey, session.RoleFull)
	sa.RegisterKey(sess, roKey, session.RoleReadOnly)
	// noneKey is deliberately not registered.

	tests := []struct {
		name        string
		key         session.ConsoleKey
		sessionName string
		payload     []byte
		wantErr     error // nil means expect no error; non-nil means errors.Is match
	}{
		{
			name:        "full-access + payload → accepted",
			key:         fullKey,
			sessionName: sess,
			payload:     []byte("keystroke"),
			wantErr:     nil,
		},
		{
			name:        "full-access + empty-tick → accepted",
			key:         fullKey,
			sessionName: sess,
			payload:     []byte{},
			wantErr:     nil,
		},
		{
			name:        "read-only + payload → ErrUpstreamReadOnly",
			key:         roKey,
			sessionName: sess,
			payload:     []byte("keystroke"),
			wantErr:     session.ErrUpstreamReadOnly,
		},
		{
			name:        "read-only + empty-tick → accepted",
			key:         roKey,
			sessionName: sess,
			payload:     []byte{},
			wantErr:     nil,
		},
		{
			name:        "read-only + nil payload → accepted",
			key:         roKey,
			sessionName: sess,
			payload:     nil,
			wantErr:     nil,
		},
		{
			name:        "unregistered key + payload → ErrSessionAuthDenied",
			key:         noneKey,
			sessionName: sess,
			payload:     []byte("keystroke"),
			wantErr:     session.ErrSessionAuthDenied,
		},
		{
			name:        "unregistered key + empty-tick → ErrSessionAuthDenied",
			key:         noneKey,
			sessionName: sess,
			payload:     []byte{},
			wantErr:     session.ErrSessionAuthDenied,
		},
		{
			name:        "cross-session (registered on sess, queried on other) → ErrSessionAuthDenied",
			key:         fullKey,
			sessionName: "agent-02", // fullKey is only registered on "agent-01"
			payload:     []byte("keystroke"),
			wantErr:     session.ErrSessionAuthDenied,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := sa.Allow(tc.key, tc.sessionName, tc.payload)
			if tc.wantErr == nil {
				if err != nil {
					t.Errorf("Allow: got %v, want nil", err)
				}
			} else {
				if err == nil {
					t.Fatalf("Allow: got nil, want errors.Is(err, %v)", tc.wantErr)
				}
				if !errors.Is(err, tc.wantErr) {
					t.Errorf("Allow: got %v, want errors.Is(err, %v)", err, tc.wantErr)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// RegisterKey idempotency: last-write-wins (ADR-003).
// ---------------------------------------------------------------------------

// TestSessionAuth_RegisterKey_LastWriteWins verifies that RegisterKey is
// idempotent with last-write-wins semantics: re-registering the same key with
// a different role overwrites the previous entry (ADR-003).
func TestSessionAuth_RegisterKey_LastWriteWins(t *testing.T) {
	t.Parallel()

	const (
		sess = "agent-01"
		key  = session.ConsoleKey("console-lww")
	)

	sa := session.NewSessionAuth()
	sa.RegisterKey(sess, key, session.RoleFull)
	sa.RegisterKey(sess, key, session.RoleReadOnly) // overwrite with read-only

	// Key should now be read-only.
	role, err := sa.Authorize(key, sess)
	if err != nil {
		t.Fatalf("Authorize after re-register: %v", err)
	}
	if role != session.RoleReadOnly {
		t.Errorf("Authorize: got %v, want RoleReadOnly (last-write-wins)", role)
	}

	// A payload-bearing upstream frame must be rejected (read-only now).
	allowErr := sa.Allow(key, sess, []byte("payload"))
	if allowErr == nil {
		t.Fatal("Allow: expected ErrUpstreamReadOnly after role downgrade to read-only, got nil")
	}
	if !errors.Is(allowErr, session.ErrUpstreamReadOnly) {
		t.Errorf("Allow: got %v, want ErrUpstreamReadOnly", allowErr)
	}
}

// ---------------------------------------------------------------------------
// Sentinel error identity checks
// Verify ErrSessionAuthDenied and ErrUpstreamReadOnly are distinct sentinels.
// ---------------------------------------------------------------------------

// TestSessionAuth_SentinelErrors_Distinct verifies that ErrSessionAuthDenied
// and ErrUpstreamReadOnly are distinct error values that cannot be conflated
// via errors.Is (E-ADM-006 vs E-ADM-007).
func TestSessionAuth_SentinelErrors_Distinct(t *testing.T) {
	t.Parallel()

	if errors.Is(session.ErrSessionAuthDenied, session.ErrUpstreamReadOnly) {
		t.Error("ErrSessionAuthDenied must not satisfy errors.Is(err, ErrUpstreamReadOnly)")
	}
	if errors.Is(session.ErrUpstreamReadOnly, session.ErrSessionAuthDenied) {
		t.Error("ErrUpstreamReadOnly must not satisfy errors.Is(err, ErrSessionAuthDenied)")
	}

	// Error strings must contain the E-ADM-NNN codes for observability.
	if !strings.Contains(session.ErrSessionAuthDenied.Error(), "E-ADM-006") {
		t.Errorf("ErrSessionAuthDenied.Error() = %q; must contain 'E-ADM-006'",
			session.ErrSessionAuthDenied.Error())
	}
	if !strings.Contains(session.ErrUpstreamReadOnly.Error(), "E-ADM-007") {
		t.Errorf("ErrUpstreamReadOnly.Error() = %q; must contain 'E-ADM-007'",
			session.ErrUpstreamReadOnly.Error())
	}
}

// ---------------------------------------------------------------------------
// Compile-time interface assertion
// SessionAuth must implement Authorizer so the wiring in upstream.go compiles.
// ---------------------------------------------------------------------------

// TestSessionAuth_ImplementsAuthorizer is a compile-time guard: if
// *SessionAuth stops implementing Authorizer, the package will not compile and
// CI catches it immediately. The runtime test body is trivially true; the
// value is in the declaration.
func TestSessionAuth_ImplementsAuthorizer(t *testing.T) {
	t.Parallel()

	var _ session.Authorizer = session.NewSessionAuth()

	// Additional runtime check: the concrete type satisfies the interface
	// method signatures at runtime (belt-and-suspenders over the compile check).
	sa := session.NewSessionAuth()
	if sa == nil {
		t.Fatal("NewSessionAuth returned nil")
	}
	_ = fmt.Sprintf("sa implements Authorizer: %T", sa)
}

// ---------------------------------------------------------------------------
// FINDING C-1 (CRITICAL) — Attach-time Tier-2 enforcement
// BC-2.05.003 PC-2 / EC-001
//
// Current Attach does NOT consult the Authorizer, so an unauthorized console
// is incorrectly admitted. The tests below MUST FAIL against current code
// (Red Gate proving the C-1 defect).
//
// Red Gate expectation:
//   TestAccessNode_Attach_UnauthorizedKey_Rejected — FAIL (Attach returns nil)
//   TestAccessNode_Attach_AuthorizedKey_Succeeds   — PASS (already works)
//   TestAccessNode_Attach_EmptyAuthList_Rejected   — FAIL (Attach returns nil)
// ---------------------------------------------------------------------------

// TestAccessNode_Attach_UnauthorizedKey_Rejected verifies that AccessNode.Attach
// returns ErrSessionAuthDenied when the console's key is not in the session's
// authorization list (BC-2.05.003 PC-2; EC-001).
//
// Proves both the error identity and that the console is NOT added to the
// fan-out set: a downstream frame delivered after the failed attach must NOT
// be received by the unauthorized console.
//
// This test MUST FAIL against current code — Attach does not call the
// Authorizer (Red Gate for C-1).
func TestAccessNode_Attach_UnauthorizedKey_Rejected(t *testing.T) {
	t.Parallel()

	const sessionName = "agent-01"
	const authorizedKey = session.ConsoleKey("console-authorized-001")
	const unauthorizedKey = session.ConsoleKey("console-unauthorized-001")

	sa := session.NewSessionAuth()
	// Register only the authorized key — unauthorized key is deliberately absent.
	sa.RegisterKey(sessionName, authorizedKey, session.RoleFull)

	pub := newAuthPublisher(t, sessionName)
	an := session.NewAccessNode(pub, sa, session.WithKeystrokeSink(session.NoOpSink{}))

	// ASSERTION 1: Attach must return ErrSessionAuthDenied for the unauthorized key.
	ds, us, err := an.Attach(unauthorizedKey, sessionName)
	if err == nil {
		t.Fatal("Attach(unauthorizedKey): expected ErrSessionAuthDenied, got nil; " +
			"Attach does not consult the Authorizer at attach-time (BC-2.05.003 PC-2 / C-1)")
	}
	if !errors.Is(err, session.ErrSessionAuthDenied) {
		t.Errorf("Attach(unauthorizedKey): got %v, want errors.Is(err, ErrSessionAuthDenied)", err)
	}

	// ASSERTION 2: channels must be nil (channel not established).
	if ds != nil {
		t.Error("Attach(unauthorizedKey): downstream channel must be nil after rejection (BC-2.05.003 PC-2)")
	}
	if us != nil {
		t.Error("Attach(unauthorizedKey): upstream channel must be nil after rejection (BC-2.05.003 PC-2)")
	}

	// ASSERTION 3: the unauthorized console must NOT receive downstream frames —
	// it was never added to the fan-out set. We attach an authorized console,
	// deliver a frame, and verify the unauthorized console does not receive it
	// (its channel is nil; attempting to receive would block/panic; the assertion
	// is structural — nil channel is sufficient proof of non-membership).
	//
	// Belt-and-suspenders: also verify the authorized console DOES receive it,
	// confirming fan-out is working and the nil check above is not vacuous.
	dsFull := mustAttachConsole(t, an, authorizedKey, sessionName)

	hdr := frame.OuterHeader{}
	an.DeliverFrame(hdr)

	frames := drainN(t, dsFull, 1)
	if len(frames) != 1 {
		t.Errorf("authorized downstream: expected 1 frame, got %d; "+
			"fan-out must still work for authorized console", len(frames))
	}

	if err := an.Detach(authorizedKey, sessionName); err != nil {
		t.Errorf("Detach: %v", err)
	}
}

// TestAccessNode_Attach_AuthorizedKey_Succeeds is the positive complement of
// TestAccessNode_Attach_UnauthorizedKey_Rejected. It verifies that an
// authorized console CAN attach and receives downstream frames after the
// Tier-2 attach gate is implemented. This makes the rejection test non-vacuous.
//
// BC-2.05.003 PC-1 (registered key attaches normally and receives downstream).
// This test PASSES against current code (Attach allows all) and MUST continue
// to pass after C-1 is fixed.
func TestAccessNode_Attach_AuthorizedKey_Succeeds(t *testing.T) {
	t.Parallel()

	const sessionName = "agent-01"
	const authorizedKey = session.ConsoleKey("console-authorized-002")

	sa := session.NewSessionAuth()
	sa.RegisterKey(sessionName, authorizedKey, session.RoleFull)

	pub := newAuthPublisher(t, sessionName)
	an := session.NewAccessNode(pub, sa, session.WithKeystrokeSink(session.NoOpSink{}))

	ds := mustAttachConsole(t, an, authorizedKey, sessionName)
	if ds == nil {
		t.Fatal("Attach(authorizedKey): downstream channel is nil; authorized key must attach successfully")
	}

	hdr := frame.OuterHeader{}
	an.DeliverFrame(hdr)

	frames := drainN(t, ds, 1)
	if len(frames) != 1 {
		t.Fatalf("downstream: expected 1 frame, got %d", len(frames))
	}

	if err := an.Detach(authorizedKey, sessionName); err != nil {
		t.Errorf("Detach: %v", err)
	}
}

// TestAccessNode_Attach_EmptyAuthList_Rejected verifies EC-002: when the
// session's authorization list is empty (no keys registered), every attach
// request must be rejected with ErrSessionAuthDenied (BC-2.05.003 EC-002).
//
// This test MUST FAIL against current code — Attach does not consult the
// Authorizer (Red Gate for C-1).
func TestAccessNode_Attach_EmptyAuthList_Rejected(t *testing.T) {
	t.Parallel()

	const sessionName = "agent-01"

	// SessionAuth with NO keys registered — empty auth list.
	sa := session.NewSessionAuth()

	pub := newAuthPublisher(t, sessionName)
	an := session.NewAccessNode(pub, sa, session.WithKeystrokeSink(session.NoOpSink{}))

	keys := []session.ConsoleKey{
		"console-empty-a",
		"console-empty-b",
		"console-empty-c",
	}

	for _, k := range keys {
		k := k
		t.Run(string(k), func(t *testing.T) {
			t.Parallel()

			ds, us, err := an.Attach(k, sessionName)
			if err == nil {
				t.Fatalf("Attach(%q): expected ErrSessionAuthDenied for empty auth list, got nil; "+
					"BC-2.05.003 EC-002: empty auth list must reject all attach requests", k)
			}
			if !errors.Is(err, session.ErrSessionAuthDenied) {
				t.Errorf("Attach(%q): got %v, want ErrSessionAuthDenied", k, err)
			}
			if ds != nil {
				t.Errorf("Attach(%q): downstream must be nil after rejection", k)
			}
			if us != nil {
				t.Errorf("Attach(%q): upstream must be nil after rejection", k)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FINDING H-2 (HIGH) — operator-facing error messages must include the
// interpolated fields mandated by the error taxonomy.
//
// error-taxonomy.md line 56:
//   E-ADM-006: "session authorization denied: console <key_fingerprint>
//               not authorized for session <session_name> on <node_addr>"
//
// error-taxonomy.md line 57:
//   E-ADM-007: "upstream rejected: read-only access for console
//               <key_fingerprint> on session <session_name>"
//
// The current sentinel strings are bare ("session: authorization denied
// (E-ADM-006)") and do not include the console fingerprint or session name.
// The Authorize/Allow methods must return interpolated errors (via fmt.Errorf
// or similar) that wrap the sentinel AND include the required fields.
//
// This test MUST FAIL against current code — the interpolated fields are
// absent from the returned error strings (Red Gate for H-1's fix).
//
// NOTE: errors.Is identity checks are preserved — this test adds string
// assertions ONLY for the operator-message contract; it does not replace
// the sentinel-identity checks elsewhere.
// ---------------------------------------------------------------------------

// TestSessionAuth_ErrorMessages_MatchTaxonomy verifies that the FORMATTED error
// returned by Authorize/Allow contains the interpolated fields required by the
// error taxonomy for E-ADM-006 and E-ADM-007 respectively.
//
// The formatted error must:
//   - satisfy errors.Is(err, ErrSessionAuthDenied) or ErrUpstreamReadOnly
//     (sentinel identity preserved), AND
//   - contain the console key fingerprint AND session name in the message
//     (operator observability; error-taxonomy.md §ADM lines 56–57).
//
// This test MUST FAIL against current code (Red Gate for H-2).
func TestSessionAuth_ErrorMessages_MatchTaxonomy(t *testing.T) {
	t.Parallel()

	const sessionName = "agent-01"
	const roKey = session.ConsoleKey("console-ro-taxonomy-001")
	const unknownKey = session.ConsoleKey("console-unknown-taxonomy-001")

	sa := session.NewSessionAuth()
	sa.RegisterKey(sessionName, roKey, session.RoleReadOnly)

	t.Run("E-ADM-006: Authorize returns key and session in error message", func(t *testing.T) {
		t.Parallel()

		_, err := sa.Authorize(unknownKey, sessionName)
		if err == nil {
			t.Fatal("Authorize(unknownKey): expected error, got nil")
		}

		// Sentinel identity must be preserved (errors.Is).
		if !errors.Is(err, session.ErrSessionAuthDenied) {
			t.Errorf("Authorize: got %v, want errors.Is(err, ErrSessionAuthDenied)", err)
		}

		// Operator-message contract: formatted error must contain the console key
		// fingerprint and session name (error-taxonomy.md line 56).
		msg := err.Error()
		if !strings.Contains(msg, string(unknownKey)) {
			t.Errorf("E-ADM-006 message missing console key fingerprint %q: got %q\n"+
				"taxonomy requires: \"session authorization denied: console <key_fingerprint> "+
				"not authorized for session <session_name> on <node_addr>\"", unknownKey, msg)
		}
		if !strings.Contains(msg, sessionName) {
			t.Errorf("E-ADM-006 message missing session name %q: got %q\n"+
				"taxonomy requires: \"session authorization denied: console <key_fingerprint> "+
				"not authorized for session <session_name> on <node_addr>\"", sessionName, msg)
		}
	})

	t.Run("E-ADM-007: Allow returns key and session in error message for read-only console", func(t *testing.T) {
		t.Parallel()

		err := sa.Allow(roKey, sessionName, []byte("keystroke"))
		if err == nil {
			t.Fatal("Allow(roKey, payload): expected error, got nil")
		}

		// Sentinel identity must be preserved.
		if !errors.Is(err, session.ErrUpstreamReadOnly) {
			t.Errorf("Allow: got %v, want errors.Is(err, ErrUpstreamReadOnly)", err)
		}

		// Operator-message contract: formatted error must contain the console key
		// fingerprint and session name (error-taxonomy.md line 57).
		msg := err.Error()
		if !strings.Contains(msg, string(roKey)) {
			t.Errorf("E-ADM-007 message missing console key fingerprint %q: got %q\n"+
				"taxonomy requires: \"upstream rejected: read-only access for console "+
				"<key_fingerprint> on session <session_name>\"", roKey, msg)
		}
		if !strings.Contains(msg, sessionName) {
			t.Errorf("E-ADM-007 message missing session name %q: got %q\n"+
				"taxonomy requires: \"upstream rejected: read-only access for console "+
				"<key_fingerprint> on session <session_name>\"", sessionName, msg)
		}
	})
}

// ---------------------------------------------------------------------------
// FINDING M-3 (MEDIUM) — empty-tick forwarding (BC-2.04.005 EC-004 liveness)
//
// TestReadOnlyConsole_EmptyTickAccepted already exists (AC-006) but uses
// NoOpSink — it proves the empty-tick is not rejected but does NOT prove
// the empty-tick is forwarded to the downstream sink (liveness credited).
//
// This companion test proves that:
//   1. Empty-tick from a read-only console IS forwarded to the captureSink.
//   2. A payload-bearing frame from the same console is NOT forwarded.
//
// If the current code swallows empty-ticks (returns nil but does not call
// sink.SendInput for empty payloads), assertion (1) fails — Red Gate for M-3.
// If forwarded, it passes — report which.
// ---------------------------------------------------------------------------

// TestReadOnlyConsole_EmptyTickForwarded_WithCaptureSink is a companion to
// TestReadOnlyConsole_EmptyTickAccepted. It uses a captureSink instead of
// NoOpSink to prove the empty-tick frame is FORWARDED to the sink (liveness
// probe credited / BC-2.04.005 EC-004), not merely accepted without effect.
//
// A payload-bearing frame from the same console must be rejected AND must NOT
// reach the sink.
func TestReadOnlyConsole_EmptyTickForwarded_WithCaptureSink(t *testing.T) {
	t.Parallel()

	const (
		sessionName = "agent-01"
		roKey       = session.ConsoleKey("console-readonly-emptytick-003")
	)

	var sinkReceived [][]byte
	sink := &captureSink{received: &sinkReceived}

	sa := session.NewSessionAuth()
	sa.RegisterKey(sessionName, roKey, session.RoleReadOnly)

	pub := newAuthPublisher(t, sessionName)
	an := session.NewAccessNode(pub, sa, session.WithKeystrokeSink(sink))

	_ = mustAttachConsole(t, an, roKey, sessionName)

	// STEP 1: payload-bearing frame — must be rejected, must NOT reach sink.
	payloadErr := an.SendKeystroke(roKey, sessionName, []byte("keystroke"))
	if payloadErr == nil {
		t.Fatal("SendKeystroke (payload-bearing): expected ErrUpstreamReadOnly, got nil; " +
			"enforcement check absent — empty-tick forwarding result is not meaningful")
	}
	if !errors.Is(payloadErr, session.ErrUpstreamReadOnly) {
		t.Errorf("SendKeystroke (payload-bearing): got %v, want ErrUpstreamReadOnly", payloadErr)
	}
	if len(sinkReceived) != 0 {
		t.Errorf("sink: got %d call(s) after read-only payload rejection, want 0; "+
			"rejected keystroke must not reach the sink", len(sinkReceived))
	}

	// STEP 2: empty-tick (zero-length payload) — must be accepted AND forwarded.
	// BC-2.04.005 EC-004: "accepted; liveness probe credited".
	// This is the liveness forwarding assertion. If the implementation accepts
	// the empty-tick (returns nil) but does not call sink.SendInput, this fails.
	emptyErr := an.SendKeystroke(roKey, sessionName, []byte{})
	if emptyErr != nil {
		t.Errorf("SendKeystroke (empty-tick): expected nil, got %v; "+
			"empty-tick must be accepted from read-only console (BC-2.04.005 EC-004)", emptyErr)
	}
	if len(sinkReceived) != 1 {
		t.Errorf("sink: got %d call(s) after empty-tick, want 1; "+
			"empty-tick must be FORWARDED to the sink (liveness probe credited, BC-2.04.005 EC-004)",
			len(sinkReceived))
	}
	if len(sinkReceived) == 1 && len(sinkReceived[0]) != 0 {
		t.Errorf("sink: forwarded empty-tick payload has length %d, want 0", len(sinkReceived[0]))
	}

	// STEP 3: nil payload (equivalent to empty-tick) — also forwarded.
	nilErr := an.SendKeystroke(roKey, sessionName, nil)
	if nilErr != nil {
		t.Errorf("SendKeystroke (nil payload): expected nil, got %v; "+
			"nil payload is an empty-tick and must be accepted (BC-2.04.005 EC-004)", nilErr)
	}
	if len(sinkReceived) != 2 {
		t.Errorf("sink: got %d call(s) after nil payload, want 2; "+
			"nil-payload empty-tick must be forwarded (BC-2.04.005 EC-004)", len(sinkReceived))
	}

	if err := an.Detach(roKey, sessionName); err != nil {
		t.Errorf("Detach: %v", err)
	}
}
