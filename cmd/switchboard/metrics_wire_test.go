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
	"go/ast"
	"go/parser"
	"go/token"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

// dialAndHandshakeADR012 dials addr and performs the ADR-012 Ed25519
// challenge-response handshake (bootstrap mode: daemon key == operator key),
// returning the authenticated connection. Factored out for
// TestWireMetricsHandlers_RegistersPingOnEveryMode;
// TestMetricsWire_PathsListRegistered predates this helper and is left
// untouched (inlines the same steps).
func dialAndHandshakeADR012(t *testing.T, addr string, daemonPub ed25519.PublicKey, daemonPriv ed25519.PrivateKey) net.Conn {
	t.Helper()

	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		t.Fatalf("dial %s: %v", addr, err)
	}
	_ = conn.SetDeadline(time.Now().Add(8 * time.Second))

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

	var authResult struct {
		Type string `json:"type"`
	}
	if err := json.NewDecoder(conn).Decode(&authResult); err != nil {
		t.Fatalf("decode AUTH result: %v", err)
	}
	if authResult.Type != "auth_ok" {
		t.Fatalf("auth failed; got type=%q", authResult.Type)
	}
	return conn
}

// sendMgmtRPC sends a JSON-RPC request for command over conn and decodes the
// response envelope. Returns ok, the error code (empty if ok), and the raw
// data payload.
func sendMgmtRPC(t *testing.T, conn net.Conn, id, command string) (ok bool, errCode string, data json.RawMessage) {
	t.Helper()

	req := map[string]any{
		"type":    "request",
		"id":      id,
		"command": command,
		"args":    nil,
	}
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		t.Fatalf("send %s RPC: %v", command, err)
	}

	var resp struct {
		Type  string `json:"type"`
		ID    string `json:"id"`
		OK    bool   `json:"ok"`
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		t.Fatalf("decode %s response: %v", command, err)
	}
	if resp.Type != "response" {
		t.Errorf("response.type: got %q; want \"response\"", resp.Type)
	}
	if resp.ID != id {
		t.Errorf("response.id: got %q; want %q", resp.ID, id)
	}
	if resp.Error != nil {
		errCode = resp.Error.Code
	}
	return resp.OK, errCode, resp.Data
}

// TestWireMetricsHandlers_RegistersPingOnEveryMode verifies AC-004 PC-1: a new
// handler (mgmt.RegisterPingHandler) is called from wireMetricsHandlers,
// making paths.ping available on every daemon mode that already wires metrics
// handlers: runRouter, runAccess, runConsole, runControl.
//
// wireMetricsHandlers is architecturally the single choke point where
// paths.ping registration happens — its own doc comment states
// RegisterPingHandler is called once, inside wireMetricsHandlers, not
// duplicated per mode. PC-1's "every mode" claim therefore decomposes into
// two independently-checkable facts, both proven here — neither alone
// establishes the claim:
//
//  1. "wireMetricsHandlers_registers_paths_ping" (dynamic): calling
//     wireMetricsHandlers against a real mgmt.Server makes paths.ping a live,
//     dispatchable RPC command — mirrors TestMetricsWire_PathsListRegistered's
//     proof for paths.list. Catches a regression where RegisterPingHandler is
//     removed from, or never called inside, wireMetricsHandlers (paths.ping
//     would return E-RPC-010 unknown-command). This subtest alone does NOT
//     prove "every mode": it never inspects the four daemon-mode functions, so
//     a wireMetricsHandlers wired into only three of the four modes would
//     still pass it.
//  2. "exactly_four_call_sites_named_by_PC1" (static): parses this package's
//     own non-test source and asserts wireMetricsHandlers is called from
//     exactly the four functions PC-1 names, by name — no more, no fewer.
//     Catches a regression where a new daemon mode is added without wiring
//     metrics/ping, or an existing call site is silently dropped from one of
//     the four current modes. This subtest alone does NOT prove PC-1 either:
//     a wireMetricsHandlers found at all four call sites whose body forgot to
//     call RegisterPingHandler would still pass it, since it never dispatches
//     an RPC.
//
// AC-004 PC-1; BC-2.06.004 Invariant 1, Trigger.
func TestWireMetricsHandlers_RegistersPingOnEveryMode(t *testing.T) {
	t.Run("wireMetricsHandlers_registers_paths_ping", func(t *testing.T) {
		t.Parallel()

		daemonPub, daemonPriv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("GenerateKey: %v", err)
		}

		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("net.Listen: %v", err)
		}
		t.Cleanup(func() { _ = ln.Close() })
		addr := ln.Addr().String()

		ops := mgmt.NewOperatorKeySet(nil) // bootstrap mode
		srv := mgmt.NewServer(
			ln, daemonPriv, ops,
			nil, // handlers registered via wireMetricsHandlers below
			"dev",
			mgmt.WithHandshakeTimeout(2*time.Second),
			mgmt.WithRPCIdleTimeout(5*time.Second),
		)

		// This is exactly what runRouter/runAccess/runConsole/runControl call
		// (AC-004 PC-1 production code path). router=nil exercises the RPC
		// wiring only, matching TestMetricsWire_PathsListRegistered's rationale.
		if err := wireMetricsHandlers(srv, nil); err != nil {
			t.Fatalf("wireMetricsHandlers: %v", err)
		}

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

		conn := dialAndHandshakeADR012(t, addr, daemonPub, daemonPriv)
		defer func() { _ = conn.Close() }()

		ok, errCode, data := sendMgmtRPC(t, conn, "req-ac-004-pc1", "paths.ping")
		if !ok {
			if errCode == "E-RPC-010" {
				t.Fatalf("AC-004 PC-1: paths.ping returned E-RPC-010 (unknown command); " +
					"RegisterPingHandler must be called from wireMetricsHandlers")
			}
			t.Fatalf("paths.ping response ok=false: code=%q", errCode)
		}

		var pong struct {
			Pong bool `json:"pong"`
		}
		if err := json.Unmarshal(data, &pong); err != nil {
			t.Fatalf("unmarshal paths.ping data: %v", err)
		}
		if !pong.Pong {
			t.Errorf("paths.ping data.pong: got false; want true (AC-004 PC-3 / BC-2.06.004 Inv-2)")
		}
	})

	t.Run("exactly_four_call_sites_named_by_PC1", func(t *testing.T) {
		wantCallers := map[string]bool{
			"runRouter":  false,
			"runAccess":  false,
			"runConsole": false,
			"runControl": false,
		}

		_, thisFile, _, ok := runtime.Caller(0)
		if !ok {
			t.Fatal("runtime.Caller(0) failed to resolve this test file's path")
		}
		pkgDir := filepath.Dir(thisFile)

		entries, err := os.ReadDir(pkgDir)
		if err != nil {
			t.Fatalf("ReadDir %s: %v", pkgDir, err)
		}

		fset := token.NewFileSet()
		var foundCallers []string
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			path := filepath.Join(pkgDir, name)
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				t.Fatalf("parse %s: %v", path, err)
			}
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Body == nil {
					continue
				}
				callsWireMetricsHandlers := false
				ast.Inspect(fn.Body, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}
					if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "wireMetricsHandlers" {
						callsWireMetricsHandlers = true
					}
					return true
				})
				if !callsWireMetricsHandlers {
					continue
				}
				foundCallers = append(foundCallers, fn.Name.Name)
				if _, expected := wantCallers[fn.Name.Name]; expected {
					wantCallers[fn.Name.Name] = true
				}
			}
		}

		for name, seen := range wantCallers {
			if !seen {
				t.Errorf("AC-004 PC-1: expected %s to call wireMetricsHandlers; it does not "+
					"(source scan of %s)", name, pkgDir)
			}
		}
		if len(foundCallers) != len(wantCallers) {
			t.Errorf("AC-004 PC-1: expected exactly the four modes named by PC-1 to call "+
				"wireMetricsHandlers; found %d call site(s): %v", len(foundCallers), foundCallers)
		}
	})
}
