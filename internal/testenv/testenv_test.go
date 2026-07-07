// Package testenv_test exercises the testenv package itself.
// These are unit tests of the harness, not integration tests of the product.
// They run without the 'integration' build tag and execute in < 2s.
package testenv_test

import (
	"context"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/testenv"
)

// TestNew_EnvIsUsable verifies that New returns a functional environment
// with an accessible accessNode (sessions can be created and found).
func TestNew_EnvIsUsable(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	env := testenv.New(t, ctx)
	t.Cleanup(env.Close)

	sid := env.CreateSession(t)
	if sid.String() == "" {
		t.Error("CreateSession returned empty session ID")
	}
	if !env.SessionAlive(t, sid) {
		t.Error("session should be alive immediately after creation")
	}
}

// TestCreateSession_Unique verifies two CreateSession calls return distinct IDs.
func TestCreateSession_Unique(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	env := testenv.New(t, ctx)

	sid1 := env.CreateSession(t)
	sid2 := env.CreateSession(t)
	if sid1.String() == sid2.String() {
		t.Errorf("CreateSession must return unique IDs; got %q twice", sid1)
	}
}

// TestCreateSVTN_Unique verifies two CreateSVTN calls return distinct SVTN IDs.
func TestCreateSVTN_Unique(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	env := testenv.New(t, ctx)

	svtnA := env.CreateSVTN(t)
	svtnB := env.CreateSVTN(t)
	if svtnA.String() == svtnB.String() {
		t.Errorf("CreateSVTN must return unique IDs; got %q twice", svtnA)
	}
}

// TestCreateSessionInSVTN_AliveAfterCreate verifies that sessions created in
// an explicit SVTN are alive and scoped to that SVTN.
func TestCreateSessionInSVTN_AliveAfterCreate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	env := testenv.New(t, ctx)

	svtn := env.CreateSVTN(t)
	sid := env.CreateSessionInSVTN(t, svtn)
	if !env.SessionAlive(t, sid) {
		t.Error("session should be alive after CreateSessionInSVTN")
	}
}

// TestAttachConsole_ReceivesFrames verifies the core frame-delivery contract:
// after AttachConsole + SendKeystroke the console's CollectFrames returns
// at least one frame.
func TestAttachConsole_ReceivesFrames(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	env := testenv.New(t, ctx)

	sid := env.CreateSession(t)
	console := env.AttachConsole(t, sid)

	env.SendKeystroke(t, sid, "hello\n")

	frames := console.CollectFrames(t, 2*time.Second)
	if len(frames) == 0 {
		t.Error("expected at least one frame after SendKeystroke; got none")
	}
}

// TestAttachConsole_Detach_StopsDelivery verifies that after Detach, new
// SendKeystroke calls do not deliver additional frames to the console.
func TestAttachConsole_Detach_StopsDelivery(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	env := testenv.New(t, ctx)

	sid := env.CreateSession(t)
	console := env.AttachConsole(t, sid)

	env.SendKeystroke(t, sid, "before detach\n")
	_ = console.CollectFrames(t, 500*time.Millisecond)
	countBefore := len(console.CollectFrames(t, 0))

	console.Detach(t)
	time.Sleep(50 * time.Millisecond)

	env.SendKeystroke(t, sid, "after detach\n")
	time.Sleep(100 * time.Millisecond)

	countAfter := len(console.CollectFrames(t, 0))
	if countAfter > countBefore {
		t.Errorf("frames grew after Detach: before=%d after=%d", countBefore, countAfter)
	}
}

// TestDetach_SessionSurvives verifies the session is alive after console detach.
func TestDetach_SessionSurvives(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	env := testenv.New(t, ctx)

	sid := env.CreateSession(t)
	console := env.AttachConsole(t, sid)
	console.Detach(t)

	if !env.SessionAlive(t, sid) {
		t.Error("session should still be alive after console detach")
	}
}

// TestMultiConsole_FanOut verifies that two attached consoles both receive
// frames (VP-034 harness prerequisite).
func TestMultiConsole_FanOut(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	env := testenv.New(t, ctx)

	sid := env.CreateSession(t)
	c1 := env.AttachConsole(t, sid)
	c2 := env.AttachConsole(t, sid)

	const messages = 3
	for i := 0; i < messages; i++ {
		env.SendKeystroke(t, sid, "line\n")
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(200 * time.Millisecond)

	f1 := c1.CollectFrames(t, 0)
	f2 := c2.CollectFrames(t, 0)

	if len(f1) == 0 {
		t.Error("console1 received no frames")
	}
	if len(f2) == 0 {
		t.Error("console2 received no frames")
	}
	if len(f1) != len(f2) {
		t.Errorf("frame count mismatch: c1=%d c2=%d", len(f1), len(f2))
	}
}

// TestSVTNIsolation verifies that frames for SVTN-A are not delivered to
// consoles attached to SVTN-B sessions (VP-039 harness prerequisite).
func TestSVTNIsolation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	env := testenv.New(t, ctx)

	svtnA := env.CreateSVTN(t)
	svtnB := env.CreateSVTN(t)
	sidA := env.CreateSessionInSVTN(t, svtnA)
	sidB := env.CreateSessionInSVTN(t, svtnB)

	probeA := env.AttachProbe(t, sidA)
	probeB := env.AttachProbe(t, sidB)

	for i := 0; i < 5; i++ {
		env.SendKeystroke(t, sidA, "data-a\n")
		env.SendKeystroke(t, sidB, "data-b\n")
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(200 * time.Millisecond)

	// Probe A must not have received frames tagged with svtnB.
	crossA := probeA.FramesFromSVTN(svtnB)
	if len(crossA) > 0 {
		t.Errorf("SVTN isolation violated: probeA received %d frames from SVTN-B", len(crossA))
	}
	// Probe B must not have received frames tagged with svtnA.
	crossB := probeB.FramesFromSVTN(svtnA)
	if len(crossB) > 0 {
		t.Errorf("SVTN isolation violated: probeB received %d frames from SVTN-A", len(crossB))
	}
}

// TestConnectWithKey_RegisteredIsAdmitted verifies the happy path (VP-046).
func TestConnectWithKey_RegisteredIsAdmitted(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	env := testenv.New(t, ctx)

	key := env.GenerateKey(t)
	env.RegisterKey(t, key)
	if err := env.ConnectWithKey(t, key); err != nil {
		t.Errorf("expected admission for registered key; got: %v", err)
	}
}

// TestConnectWithKey_RevokedIsRejected verifies that revoked keys fail (VP-046).
func TestConnectWithKey_RevokedIsRejected(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	env := testenv.New(t, ctx)

	key := env.GenerateKey(t)
	env.RegisterKey(t, key)
	env.RevokeKey(t, key)
	if err := env.ConnectWithKey(t, key); err == nil {
		t.Error("expected rejection for revoked key; got nil error")
	}
}

// TestConnectWithKey_ExpiredIsRejected verifies that expired keys fail (VP-046).
func TestConnectWithKey_ExpiredIsRejected(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	env := testenv.New(t, ctx)

	key := env.GenerateKeyWithExpiry(t, time.Now().Add(100*time.Millisecond))
	env.RegisterKey(t, key)

	// Must be admitted before expiry.
	if err := env.ConnectWithKey(t, key); err != nil {
		t.Errorf("expected admission before expiry; got: %v", err)
	}

	// Wait for expiry.
	time.Sleep(200 * time.Millisecond)

	if err := env.ConnectWithKey(t, key); err == nil {
		t.Error("expected rejection after key expiry; got nil error")
	}
}

// TestNewWithRouters_CloseRouterConnection verifies the multipath path-closing
// mechanic (VP-040 harness prerequisite).
func TestNewWithRouters_CloseRouterConnection(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	env := testenv.NewWithRouters(t, ctx, 2)

	// Before close: WaitForPaths should succeed immediately (2 routers).
	if err := env.WaitForPaths(t, testenv.SessionID{}, 2, 1*time.Second); err != nil {
		t.Fatalf("expected 2 active paths before close: %v", err)
	}

	// Close one router.
	env.CloseRouterConnection(t, 0)

	// After close: only 1 router active.
	if err := env.WaitForPaths(t, testenv.SessionID{}, 2, 50*time.Millisecond); err == nil {
		t.Error("expected WaitForPaths(2) to fail after closing one router; got nil")
	}
	if err := env.WaitForPaths(t, testenv.SessionID{}, 1, 100*time.Millisecond); err != nil {
		t.Errorf("expected WaitForPaths(1) to succeed with 1 remaining router: %v", err)
	}
}

// TestConnectWithSourceIP_SessionIDPreserved verifies that reconnecting
// with a different source IP preserves the session ID (VP-036 harness).
func TestConnectWithSourceIP_SessionIDPreserved(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	env := testenv.New(t, ctx)

	creds := env.GenerateCredentials(t)

	conn1 := env.ConnectWithSourceIP(t, "192.0.2.1", creds)
	sid := conn1.SessionID()
	conn1.Close()

	time.Sleep(50 * time.Millisecond)

	conn2 := env.ConnectWithSourceIP(t, "192.0.2.2", creds)
	if conn2.SessionID() != sid {
		t.Errorf("session ID changed after IP change: before=%s after=%s", sid, conn2.SessionID())
	}
}

// TestStartRouter_ModeE verifies a router starts in E mode.
func TestStartRouter_ModeE(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	env := testenv.New(t, ctx)

	r := env.StartRouter(t, testenv.RouterConfig{})
	if r.Mode() != testenv.ModeE {
		t.Errorf("expected E mode; got %v", r.Mode())
	}
}

// TestStartRouter_Restart_EntersPEMode verifies that restarting with
// UpstreamRouters promotes the router to PE mode (VP-038 harness).
func TestStartRouter_Restart_EntersPEMode(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	env := testenv.New(t, ctx)

	peAddr := env.PERouterAddr(t)
	r := env.StartRouter(t, testenv.RouterConfig{})

	svtnBefore := r.SVTNID()
	r.Restart(t, testenv.RouterConfig{UpstreamRouters: []string{peAddr}})

	if r.Mode() != testenv.ModePE {
		t.Errorf("expected PE mode after restart; got %v", r.Mode())
	}
	if r.SVTNID() != svtnBefore {
		t.Errorf("SVTN ID changed after restart: before=%v after=%v", svtnBefore, r.SVTNID())
	}
}

// TestClose_NoGoroutineLeak verifies that Close() causes all background
// goroutines to exit before returning.
func TestClose_NoGoroutineLeak(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	env := testenv.New(t, ctx)

	sid := env.CreateSession(t)
	_ = env.AttachConsole(t, sid)
	_ = env.AttachProbe(t, sid)

	// Closing should drain all goroutines.  The env's wg is internal; we
	// verify indirectly by checking Close() does not block beyond 1s.
	done := make(chan struct{})
	go func() {
		env.Close()
		close(done)
	}()
	select {
	case <-done:
		// OK
	case <-time.After(1 * time.Second):
		t.Error("Close() did not complete within 1s — goroutine leak suspected")
	}
}

// TestRouterHandle_Restart_TwicePE verifies F-P2-001: a connector stopped by
// Restart's oldConn.Stop() and then stopped again by the t.Cleanup registered
// inside Restart must not panic.
//
// Reproduction shape (adversary pass-2, F-P2-001):
//  1. StartRouter (E mode, no connector)
//  2. Restart with PE config → conn1 started, t.Cleanup(conn1.Stop) registered
//  3. Restart again with PE config → conn1.Stop() called (first Stop), conn2
//     started, t.Cleanup(conn2.Stop) registered
//  4. t.Cleanup fires: conn1.Stop() called again (second Stop) → panic without fix
//
// Failure condition (old code): close of closed channel panic.
func TestRouterHandle_Restart_TwicePE(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	env := testenv.New(t, ctx)

	peAddr := env.PERouterAddr(t)

	r := env.StartRouter(t, testenv.RouterConfig{})
	if r.Mode() != testenv.ModeE {
		t.Fatalf("precondition: expected E mode after StartRouter; got %v", r.Mode())
	}

	// First Restart into PE mode: creates conn1, registers t.Cleanup(conn1.Stop).
	r.Restart(t, testenv.RouterConfig{UpstreamRouters: []string{peAddr}})
	if r.Mode() != testenv.ModePE {
		t.Errorf("expected PE mode after first Restart; got %v", r.Mode())
	}

	// Second Restart into PE mode: calls conn1.Stop() (first), creates conn2,
	// registers t.Cleanup(conn2.Stop).
	r.Restart(t, testenv.RouterConfig{UpstreamRouters: []string{peAddr}})
	if r.Mode() != testenv.ModePE {
		t.Errorf("expected PE mode after second Restart; got %v", r.Mode())
	}

	// t.Cleanup will fire conn1.Stop() again and conn2.Stop() when this test
	// returns.  Without the idempotent Stop fix, conn1.Stop() panics here.
}

// TestNewLoopback_Compiles verifies NewLoopback returns a usable LoopbackEnv.
func TestNewLoopback_Compiles(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	lb := testenv.NewLoopback(t, ctx, testenv.LoopbackConfig{})
	if lb == nil {
		t.Fatal("NewLoopback returned nil")
	}
	if lb.Env == nil {
		t.Fatal("NewLoopback.Env is nil")
	}
}
