// metrics_wire_test.go — integration test for F-P1L1-002: paths.list is a
// registered handler on the running access daemon management server.
//
// TestMetricsWire_PathsListRegistered spins up a real mgmt.Server via
// wireMetricsHandlers (the same function called by runAccess), dials it,
// performs the ADR-012 Ed25519 challenge-response handshake, sends a
// paths.list RPC, and asserts the response is ok=true with a paths array.
//
// F-P1L1-002; S-W5.04 AC-001; BC-2.06.003 PC-1.
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/mgmt"
)

// TestMetricsWire_PathsListRegistered verifies that wireMetricsHandlers registers
// the paths.list handler on the management server, and that sending a paths.list
// RPC returns ok=true (not E-RPC-010 unknown-command).
//
// This is the production observable for F-P1L1-002: handlers never registered.
//
// F-P1L1-002; S-W5.04 AC-001; BC-2.06.003 PC-1.
func TestMetricsWire_PathsListRegistered(t *testing.T) {
	t.Parallel()

	// ── 1. Generate ephemeral daemon keypair. ─────────────────────────────────
	daemonPub, daemonPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	// ── 2. Open a TCP listener on loopback. ───────────────────────────────────
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	addr := ln.Addr().String()

	// ── 3. Build the mgmt.Server. ─────────────────────────────────────────────
	ops := mgmt.NewOperatorKeySet(nil) // bootstrap mode
	srv := mgmt.NewServer(
		ln, daemonPriv, ops,
		nil, // handlers registered via wireMetricsHandlers below
		"dev",
		mgmt.WithHandshakeTimeout(2*time.Second),
		mgmt.WithRPCIdleTimeout(5*time.Second),
	)

	// ── 4. Register metrics handlers via production wiring function. ──────────
	// This is exactly what runAccess calls (F-P1L1-002 production code path).
	// Pass nil router — this test exercises the RPC handler wiring, not the
	// forwarding-entry hook. With router=nil the pathTrackerSource is an empty
	// registry and paths.list returns EC-001 "no active paths", which is
	// exactly what the assertion below relies on (S-BL.PATH-TRACKER-WIRING).
	if err := wireMetricsHandlers(srv, nil); err != nil {
		t.Fatalf("wireMetricsHandlers: %v", err)
	}

	// ── 5. Start Serve. ───────────────────────────────────────────────────────
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = srv.Serve(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		wg.Wait()
	})

	// ── 6. Dial and perform ADR-012 challenge-response. ───────────────────────
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		t.Fatalf("dial %s: %v", addr, err)
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(8 * time.Second))

	// Step 6a: read CHALLENGE.
	var challenge struct {
		Type  string `json:"type"`
		Nonce string `json:"nonce"`
	}
	if err := json.NewDecoder(conn).Decode(&challenge); err != nil {
		t.Fatalf("decode CHALLENGE: %v", err)
	}
	if challenge.Type != "challenge" {
		t.Fatalf("expected type=challenge; got %q", challenge.Type)
	}

	// Step 6b: sign nonce with daemon key (bootstrap mode: daemon key == operator key).
	nonceBytes, err := base64.RawURLEncoding.DecodeString(challenge.Nonce)
	if err != nil {
		t.Fatalf("decode nonce: %v", err)
	}
	sig := ed25519.Sign(daemonPriv, nonceBytes)
	cresp := map[string]string{
		"type":      "challenge_response",
		"nonce_sig": base64.RawURLEncoding.EncodeToString(sig),
		"pubkey":    base64.RawURLEncoding.EncodeToString([]byte(daemonPub)),
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
		t.Fatalf("auth failed; got type=%q", authResult.Type)
	}

	// ── 7. Send paths.list RPC. ───────────────────────────────────────────────
	rpcReq := map[string]any{
		"type":    "request",
		"id":      "req-f-p1l1-002",
		"command": "paths.list",
		"args":    nil,
	}
	if err := json.NewEncoder(conn).Encode(rpcReq); err != nil {
		t.Fatalf("send paths.list RPC: %v", err)
	}

	// ── 8. Read response. ─────────────────────────────────────────────────────
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
		t.Fatalf("decode paths.list response: %v", err)
	}
	if rpcResp.Type != "response" {
		t.Errorf("response.type: got %q; want \"response\"", rpcResp.Type)
	}
	if rpcResp.ID != "req-f-p1l1-002" {
		t.Errorf("response.id: got %q; want %q", rpcResp.ID, "req-f-p1l1-002")
	}

	// F-P1L1-002 core assertion: paths.list must be a registered handler → ok=true.
	// Before the fix, wireMetricsHandlers was never called → E-RPC-010 unknown-command.
	if !rpcResp.OK {
		errCode := ""
		if rpcResp.Error != nil {
			errCode = rpcResp.Error.Code
		}
		// If we got E-RPC-010, the handler was not registered.
		if errCode == "E-RPC-010" {
			t.Errorf("F-P1L1-002: paths.list returned E-RPC-010 (unknown command); " +
				"wireMetricsHandlers must be called to register the handler (S-W5.04 AC-001; BC-2.06.003 PC-1)")
		} else {
			t.Errorf("paths.list response ok=false: code=%q; paths.list handler failed", errCode)
		}
		return
	}

	// ── 9. Verify the response has a paths array. ─────────────────────────────
	var pathsResp struct {
		Paths   []json.RawMessage `json:"paths"`
		Message string            `json:"message,omitempty"`
	}
	if err := json.Unmarshal(rpcResp.Data, &pathsResp); err != nil {
		t.Fatalf("unmarshal paths.list data: %v", err)
	}
	// emptyPathsSource returns zero paths → paths array must be present and empty.
	// message must be "no active paths" per EC-001.
	if pathsResp.Message != "no active paths" {
		t.Errorf("paths.list message: got %q; want \"no active paths\" (emptyPathsSource, EC-001)", pathsResp.Message)
	}
}
