package metrics_test

// integration_test.go — VP-047 end-to-end integration test for AC-006.
//
// TestVP047_SbctlPathsList_EndToEnd spins up a real mgmt.Server (the production
// entry point for daemon management-plane RPCs) with two synthetic paths — one
// pending (SampleCount<10) and one green (SampleCount≥10) — and invokes the
// paths.list handler through the server's dispatch loop. It then parses the
// JSON response and asserts all required fields are present per VP-047.
//
// AC-006 / VP-047 / BC-2.06.003 PC-1
//
// --- Why this approach (AC-006 oracle rationale) ---
// VP-047 requires "a real (non-stub) daemon with PathTracker state". The sbctl
// binary is not available as a compiled artifact during `go test ./internal/...`
// runs (the binary lives in cmd/switchboard). Instead, we enter the daemon through
// its production wiring path by:
//  1. constructing a mgmt.Server with the production Register()+Serve() plumbing
//  2. calling metrics.RegisterMetricsHandlers (same function cmd/switchboard calls)
//     to wire PathsList into the dispatch table
//  3. dialling the server and performing the full ADR-012 challenge-response handshake
//  4. sending a paths.list RPC and asserting the response fields
// This gives a genuine oracle: if RegisterMetricsHandlers or PathsList panics
// (current stub state), the test fails with a non-nil error or panic recovery;
// when implemented, the test validates field presence on the real response.
//
// The sbctl-binary oracle (exec.CommandContext("sbctl", ...)) is the Wave-6
// stretch goal per story v1.4 AC-006, blocked until the binary is compiled with
// S-W5.04 handlers. That binary-level test is tracked in VP-047.md v1.2 harness.

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/metrics"
	"github.com/arcavenae/switchboard/internal/mgmt"
	"github.com/arcavenae/switchboard/internal/paths"
)

// routerAddrPattern is the strong structural oracle for the router_addr field:
// must be a non-empty host:port string matching ^[^:]+:[0-9]+$.
// Used by TestVP047_RouterAddrNonEmpty (AC-005 / VP-047 / BC-2.06.003 v1.15 PC-1).
var routerAddrPattern = regexp.MustCompile(`^[^:]+:[0-9]+$`)

// pathTrackerListSource is a PathsListSource backed by a map of
// pathID → *paths.PathTracker. It calls PathTracker.Snapshot() to produce
// consistent value copies per go.md rule 12.
//
// F-P2L1-003: exercises the production wiring path (real PathTracker state)
// rather than injecting a synthetic snapshot map directly.
type pathTrackerListSource struct {
	trackers map[string]*paths.PathTracker
}

func (p *pathTrackerListSource) AllSnapshots() map[string]paths.PathSnapshot {
	out := make(map[string]paths.PathSnapshot, len(p.trackers))
	for id, t := range p.trackers {
		out[id] = t.Snapshot()
	}
	return out
}

// newTestPathTrackerSource creates a pathTrackerListSource seeded with:
//   - "pending-path": 5 OnProbe calls → SampleCount < 10 → rtt_p99_ms:"pending"
//   - "green-path": 15 OnProbe calls → SampleCount ≥ 10 → rtt_p99_ms:float64
//
// VP-047 requires "at least one pending + one green path". Using real PathTracker
// objects exercises the production Snapshot() method (F-P2L1-003 Ruling-3 Option A).
func newTestPathTrackerSource(t *testing.T) *pathTrackerListSource {
	t.Helper()

	pending := paths.NewPathTracker(50.0, 0.125)
	for i := 0; i < 5; i++ { // SampleCount = 5 < 10
		pending.OnProbe(12.0, false)
	}

	green := paths.NewPathTracker(50.0, 0.125)
	for i := 0; i < 15; i++ { // SampleCount = 15 ≥ 10
		green.OnProbe(22.0, false)
	}

	return &pathTrackerListSource{
		trackers: map[string]*paths.PathTracker{
			"pending-path": pending,
			"green-path":   green,
		},
	}
}

// b64Decode decodes a base64url-encoded string. Calls t.Fatalf on error so it
// can be used inline in test setup.
func b64Decode(t *testing.T, s string) []byte {
	t.Helper()
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		t.Fatalf("base64url decode %q: %v", s, err)
	}
	return b
}

func b64Encode(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

// TestVP047_SbctlPathsList_EndToEnd is the VP-047 integration test.
//
// It spins up a real mgmt.Server (production dispatch loop), registers the
// paths.list handler via RegisterMetricsHandlers, dials the management socket,
// performs the ADR-012 Ed25519 challenge-response handshake, sends a
// paths.list RPC, and asserts:
//   - All required fields present: path_id, router_addr, rtt_ms, rtt_p99_ms, loss_pct, status
//   - Pending path has rtt_p99_ms == "pending" (string)
//   - Green path has rtt_p99_ms as float64
//   - At least one pending + one green entry
//
// AC-006 / VP-047 / BC-2.06.003 PC-1.
func TestVP047_SbctlPathsList_EndToEnd(t *testing.T) {
	t.Parallel()

	// ── 1. Generate a daemon Ed25519 keypair. ─────────────────────────────────
	// Bootstrap mode: daemon key == authorized operator key (no separate operator key).
	daemonPub, daemonPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate daemon keypair: %v", err)
	}

	// ── 2. Open a TCP listener on loopback (avoids Unix socket path-length limits). ──
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	addr := ln.Addr().String()

	// ── 3. Build the mgmt.Server with no initial handlers. ────────────────────
	ops := mgmt.NewOperatorKeySet(nil) // bootstrap mode
	srv := mgmt.NewServer(
		ln, daemonPriv, ops,
		nil, // handlers will be registered via RegisterMetricsHandlers
		"dev",
		mgmt.WithHandshakeTimeout(2*time.Second),
		mgmt.WithRPCIdleTimeout(5*time.Second),
	)

	// ── 4. Register paths.list via RegisterMetricsHandlers (production path). ─
	// Use real PathTracker-backed source per F-P2L1-003 (Ruling-3 Option A).
	// This exercises the production Snapshot() read path rather than injecting
	// a static snapshot map.
	pathsSrc := newTestPathTrackerSource(t)
	routerSrc := &fakeRouterMetricsSource{metrics: map[string]metrics.RouterMetricsResponse{}}
	if err := mgmt.RegisterMetricsHandlers(srv, pathsSrc, routerSrc); err != nil {
		t.Fatalf("RegisterMetricsHandlers: %v", err)
	}

	// ── 5. Start Serve in a background goroutine. ─────────────────────────────
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	var serveWG sync.WaitGroup
	serveWG.Add(1)
	go func() {
		defer serveWG.Done()
		_ = srv.Serve(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		serveWG.Wait()
	})

	// ── 6. Dial and perform the ADR-012 Ed25519 challenge-response handshake. ─
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		t.Fatalf("dial management server at %s: %v", addr, err)
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(8 * time.Second))

	// Step 6a: read CHALLENGE.
	var challenge struct {
		Type      string `json:"type"`
		Nonce     string `json:"nonce"`
		DaemonSig string `json:"daemon_sig"`
	}
	if err := json.NewDecoder(conn).Decode(&challenge); err != nil {
		t.Fatalf("decode CHALLENGE: %v", err)
	}
	if challenge.Type != "challenge" {
		t.Fatalf("expected type=challenge; got %q", challenge.Type)
	}

	// Step 6b: sign nonce with daemon's own key (bootstrap: daemon key is operator key).
	nonceBytes := b64Decode(t, challenge.Nonce)
	nonceSig := ed25519.Sign(daemonPriv, nonceBytes)

	cresp := map[string]string{
		"type":      "challenge_response",
		"nonce_sig": b64Encode(nonceSig),
		"pubkey":    b64Encode([]byte(daemonPub)),
	}
	if err := json.NewEncoder(conn).Encode(cresp); err != nil {
		t.Fatalf("send CHALLENGE_RESPONSE: %v", err)
	}

	// Step 6c: read AUTH_OK.
	var authResult struct {
		Type string `json:"type"`
	}
	if err := json.NewDecoder(conn).Decode(&authResult); err != nil {
		t.Fatalf("decode AUTH result: %v", err)
	}
	if authResult.Type != "auth_ok" {
		t.Fatalf("expected auth_ok; got %q (auth failed — check key bootstrap setup)", authResult.Type)
	}

	// ── 7. Send paths.list RPC. ───────────────────────────────────────────────
	rpcReq := map[string]any{
		"type":    "request",
		"id":      "req-vp047-001",
		"command": "paths.list",
		"args":    nil,
	}
	if err := json.NewEncoder(conn).Encode(rpcReq); err != nil {
		t.Fatalf("send paths.list RPC: %v", err)
	}

	// ── 8. Read RPC response. ─────────────────────────────────────────────────
	var rpcResp struct {
		Type  string `json:"type"`
		ID    string `json:"id"`
		OK    bool   `json:"ok"`
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(conn).Decode(&rpcResp); err != nil {
		t.Fatalf("decode RPC response: %v", err)
	}
	if rpcResp.Type != "response" {
		t.Errorf("response type: got %q; want \"response\"", rpcResp.Type)
	}
	if rpcResp.ID != "req-vp047-001" {
		t.Errorf("response id: got %q; want %q", rpcResp.ID, "req-vp047-001")
	}
	if !rpcResp.OK {
		errCode, errMsg := "", ""
		if rpcResp.Error != nil {
			errCode = rpcResp.Error.Code
			errMsg = rpcResp.Error.Message
		}
		t.Fatalf("RPC response ok=false: code=%q message=%q (paths.list handler not implemented or panicked)", errCode, errMsg)
	}

	// ── 9. Parse PathsListResponse from data. ─────────────────────────────────
	var pathsResp metrics.PathsListResponse
	if err := json.Unmarshal(rpcResp.Data, &pathsResp); err != nil {
		t.Fatalf("unmarshal PathsListResponse: %v\nraw data: %s", err, string(rpcResp.Data))
	}

	if len(pathsResp.Paths) < 2 {
		t.Fatalf("VP-047: expected ≥2 paths (one pending + one green); got %d", len(pathsResp.Paths))
	}

	// ── 10. Assert required fields per VP-047 using raw JSON for type inspection. ─
	var rawData struct {
		Paths []json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(rpcResp.Data, &rawData); err != nil {
		t.Fatalf("unmarshal raw paths array: %v", err)
	}

	type rawEntry struct {
		PathID     *string         `json:"path_id"`
		RouterAddr *string         `json:"router_addr"`
		RTTMs      *float64        `json:"rtt_ms"`
		RTTP99Ms   json.RawMessage `json:"rtt_p99_ms"`
		LossPct    *float64        `json:"loss_pct"`
		Status     *string         `json:"status"`
	}

	pendingCount := 0
	greenCount := 0

	for i, rawPath := range rawData.Paths {
		var e rawEntry
		if err := json.Unmarshal(rawPath, &e); err != nil {
			t.Errorf("path[%d]: unmarshal error: %v", i, err)
			continue
		}

		// VP-047 required fields: path_id, router_addr, rtt_ms, rtt_p99_ms, loss_pct, status.
		if e.PathID == nil || *e.PathID == "" {
			t.Errorf("path[%d]: missing or empty path_id", i)
		}
		// router_addr: field must be present. Non-empty for PathTrackers constructed
		// via NewPathTrackerWithAddr; "" for addr-less PathTrackers (NewPathTracker).
		// DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER closed by S-BL.ROUTER-ADDR (BC-2.06.003 v1.15).
		// The integration paths here use NewPathTracker (addr-less) so "" is expected;
		// non-empty oracle requires production wiring via S-BL.PATH-TRACKER-WIRING.
		// AC-005 / VP-047 / BC-2.06.003 v1.15 PC-1.
		if e.RouterAddr == nil {
			t.Errorf("path[%d]: missing router_addr field", i)
		}
		if e.RTTMs == nil {
			t.Errorf("path[%d]: missing rtt_ms", i)
		}
		if len(e.RTTP99Ms) == 0 {
			t.Errorf("path[%d]: missing rtt_p99_ms", i)
		}
		if e.LossPct == nil {
			t.Errorf("path[%d]: missing loss_pct", i)
		}
		if e.Status == nil || *e.Status == "" {
			t.Errorf("path[%d]: missing or empty status", i)
		}

		// Classify rtt_p99_ms as pending (string) or green (float64).
		p99Raw := string(e.RTTP99Ms)
		if p99Raw == `"pending"` {
			pendingCount++
		} else {
			var p99Float float64
			if jsonErr := json.Unmarshal(e.RTTP99Ms, &p99Float); jsonErr != nil {
				t.Errorf("path[%d]: rtt_p99_ms is neither \"pending\" nor float64: %s", i, p99Raw)
			} else {
				greenCount++
			}
		}
	}

	// VP-047: at least one pending + one green path required.
	if pendingCount == 0 {
		t.Errorf("VP-047: expected ≥1 pending path (rtt_p99_ms==\"pending\"); got 0")
	}
	if greenCount == 0 {
		t.Errorf("VP-047: expected ≥1 green path (rtt_p99_ms==float64); got 0")
	}
}

// TestVP047_RouterAddrNonEmpty is structured as two parts (RULING-W6TB-K F-P4L2-03):
//
// Part A (GREEN-BY-DESIGN): exercises the handler seam using fakePathsListSource
// with an injected PathSnapshot carrying a non-empty RouterAddr. It verifies that
// PathsList forwards RouterAddr through to the JSON response. Part A exists
// separately from TestPathsList_PassesRouterAddr (handlers_test.go) because it
// adds a second oracle: the host:port regex assertion (^[^:]+:[0-9]+$) that
// TestPathsList_PassesRouterAddr does not include. Both tests use fakePathsListSource
// with an injected PathSnapshot; Part A is not redundant because the regex check
// is an additional structural constraint required by VP-047. Both parts MUST remain
// collocated under the VP-047 oracle name per RULING-W6TB-K F-P4L2-03.
//
// Part B: exercises NewPathTrackerWithAddr → Snapshot() → router_addr end-to-end.
//
// Both parts MUST remain in this test. Together they constitute the AC-005 oracle
// for VP-047 router_addr traceability.
//
// This test exercises the VP-047 AC-006 oracle flip from AC-005:
//   - The original integration test (TestVP047_SbctlPathsList_EndToEnd) constructs
//     trackers via NewPathTracker (addr-less) and therefore expects router_addr=="".
//   - This test constructs a tracker via NewPathTrackerWithAddr and asserts that
//     router_addr is non-empty and structurally valid (host:port pattern).
//
// End-to-end observability through a running daemon is deferred to
// S-BL.PATH-TRACKER-WIRING per RULING-W6TB-B.
//
// AC-005 / VP-047 / BC-2.06.003 v1.15 PC-1 (S-BL.ROUTER-ADDR); RULING-W6TB-B.
func TestVP047_RouterAddrNonEmpty(t *testing.T) {
	t.Parallel()

	// ── Part A: handler seam — fakePathsListSource with non-empty RouterAddr ──────
	// This part verifies that PathsList forwards a non-empty RouterAddr through to
	// the JSON response. It bypasses the PathTracker constructor so it is
	// GREEN-BY-DESIGN at Red Gate (tests the handler, not the constructor).
	t.Run("handler_seam_non_empty", func(t *testing.T) {
		t.Parallel()

		const stubAddr = "10.0.0.1:9000"
		snap := paths.PathSnapshot{
			EWMARTTMs:   20.0,
			LossPct:     0.0,
			Active:      true,
			Degraded:    false,
			P99RTTMs:    20.0,
			SampleCount: 10,
			RouterAddr:  stubAddr,
		}
		src := &fakePathsListSource{
			snaps: map[string]paths.PathSnapshot{
				"path-with-addr": snap,
			},
		}

		resp, err := metrics.PathsList(context.Background(), nil, src)
		if err != nil {
			t.Fatalf("PathsList error: %v", err)
		}
		if len(resp.Paths) != 1 {
			t.Fatalf("expected 1 path; got %d", len(resp.Paths))
		}

		entry := resp.Paths[0]

		// Oracle: router_addr must be non-empty.
		if entry.RouterAddr == "" {
			t.Errorf("PathEntry.RouterAddr: got \"\"; want non-empty (snap.RouterAddr=%q must be forwarded)", stubAddr)
		}

		// Oracle: router_addr must match host:port pattern ^[^:]+:[0-9]+$.
		if !routerAddrPattern.MatchString(entry.RouterAddr) {
			t.Errorf("PathEntry.RouterAddr=%q: does not match ^[^:]+:[0-9]+$ (invalid host:port format)", entry.RouterAddr)
		}

		// Oracle: router_addr must equal the stub addr exactly (no truncation/mutation).
		if entry.RouterAddr != stubAddr {
			t.Errorf("PathEntry.RouterAddr: got %q; want %q", entry.RouterAddr, stubAddr)
		}

		// Verify via JSON deserialization as well.
		data, err := json.Marshal(entry)
		if err != nil {
			t.Fatalf("marshal PathEntry: %v", err)
		}
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("unmarshal raw: %v", err)
		}
		var gotRouterAddr string
		if err := json.Unmarshal(raw["router_addr"], &gotRouterAddr); err != nil {
			t.Fatalf("unmarshal router_addr from JSON: %v", err)
		}
		if !routerAddrPattern.MatchString(gotRouterAddr) {
			t.Errorf("JSON router_addr=%q: does not match ^[^:]+:[0-9]+$ pattern", gotRouterAddr)
		}
	})

	// ── Part B: constructor-through-Snapshot — NewPathTrackerWithAddr path ────────
	// This part constructs a real PathTracker via NewPathTrackerWithAddr and asserts
	// that Snapshot().RouterAddr is non-empty and matches the host:port pattern.
	//
	// RED GATE (originally at commit 4a4efed): stub panicked per BC-5.38.001.
	// Now GREEN: implemented in paths.go (commit 27d7717).
	t.Run("constructor_through_snapshot", func(t *testing.T) {
		t.Parallel()

		const stubAddr = "h:9000"

		tracker := paths.NewPathTrackerWithAddr(stubAddr, 50.0, 0.125)
		for i := 0; i < 10; i++ {
			tracker.OnProbe(22.0, false)
		}

		snap := tracker.Snapshot()

		// Oracle: RouterAddr must be non-empty.
		if snap.RouterAddr == "" {
			t.Errorf("Snapshot().RouterAddr: got \"\"; want non-empty (constructed with addr=%q)", stubAddr)
		}

		// Oracle: RouterAddr must match ^[^:]+:[0-9]+$ (strong structural oracle).
		if !routerAddrPattern.MatchString(snap.RouterAddr) {
			t.Errorf("Snapshot().RouterAddr=%q: does not match ^[^:]+:[0-9]+$ (invalid host:port format)", snap.RouterAddr)
		}

		// Oracle: value must equal the constructor arg exactly.
		if snap.RouterAddr != stubAddr {
			t.Errorf("Snapshot().RouterAddr: got %q; want %q (constructor arg must be preserved verbatim)", snap.RouterAddr, stubAddr)
		}

		// Verify PathsList forwards it through to JSON.
		src := &pathTrackerListSource{
			trackers: map[string]*paths.PathTracker{
				"path-with-addr": tracker,
			},
		}
		resp, err := metrics.PathsList(context.Background(), nil, src)
		if err != nil {
			t.Fatalf("PathsList error: %v", err)
		}
		if len(resp.Paths) != 1 {
			t.Fatalf("expected 1 path; got %d", len(resp.Paths))
		}
		entry := resp.Paths[0]
		if !routerAddrPattern.MatchString(entry.RouterAddr) {
			t.Errorf("PathsList entry RouterAddr=%q: does not match ^[^:]+:[0-9]+$ pattern", entry.RouterAddr)
		}
		if entry.RouterAddr != stubAddr {
			t.Errorf("PathsList entry RouterAddr: got %q; want %q", entry.RouterAddr, stubAddr)
		}
	})
}
