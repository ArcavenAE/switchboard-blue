// console_test.go tests the `sbctl console` subcommand tree at the
// CLI layer, covering arg parsing, wire-level JSON dispatch for
// attach/detach/switch, and error-code surfacing.
//
// Traceability:
//
//	BC-2.08.001 — Console Remotely Controllable via sbctl
//	AC-001      — sbctl console attach --target --session
//	AC-002      — sbctl console detach --target
//	AC-003      — sbctl console switch --target --session
//	RULING-W6TB-C — transport is mgmt-plane Unix socket, not SVTN data plane
//
// Red Gate: all tests MUST fail before implementation (runConsole panics).
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestSbctlConsole_Attach verifies AC-001:
// `sbctl console attach --target <console_addr> --session <name>` sends a JSON
// attach-request over the mgmt-plane Unix socket, and surfaces:
//   - success path (session attached),
//   - E-SES-001 (unknown session),
//   - E-ADM-006 wrapped in E-RPC-011 (auth denied).
//
// BC-2.08.001 PC-1 — console.attach RPC dispatched with correct session_name.
// AC-001.
// RULING-W6TB-C — transport is mgmt-plane Unix socket.
func TestSbctlConsole_Attach(t *testing.T) {
	t.Parallel()

	t.Run("success_dispatches_console_attach_rpc", func(t *testing.T) {
		t.Parallel()

		// BC-2.08.001 PC-1 — console.attach RPC dispatched with correct session_name.
		// AC-001 success path.
		const wantSession = "agent-01"

		requestCh := make(chan adminRPCRequest, 1)
		addr := startFakeServer(t, requestCh, func(cmd string, args json.RawMessage) (any, error) {
			if cmd != "console.attach" {
				return nil, fmt.Errorf("unexpected command: %q; want console.attach", cmd)
			}
			var a consoleAttachArgs
			if err := json.Unmarshal(args, &a); err != nil {
				return nil, fmt.Errorf("unmarshal consoleAttachArgs: %w", err)
			}
			if a.SessionName != wantSession {
				return nil, fmt.Errorf("session_name: got %q; want %q", a.SessionName, wantSession)
			}
			return map[string]string{"session_name": a.SessionName}, nil
		})

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		err := runConsoleAttach(ctx, addr, testdataKeyPath(t), false, []string{
			"--session", wantSession,
		}, defaultIO())
		if err != nil {
			t.Fatalf("AC-001 success — runConsoleAttach: %v", err)
		}

		select {
		case req := <-requestCh:
			if req.Command != "console.attach" {
				t.Errorf("AC-001 — dispatched command: got %q; want console.attach", req.Command)
			}
			var a consoleAttachArgs
			if err := json.Unmarshal(req.Args, &a); err != nil {
				t.Fatalf("AC-001 — unmarshal args: %v", err)
			}
			if a.SessionName != wantSession {
				t.Errorf("AC-001 — wire session_name: got %q; want %q", a.SessionName, wantSession)
			}
		case <-time.After(2 * time.Second):
			t.Error("AC-001: timed out waiting for console.attach RPC")
		}
	})

	t.Run("unknown_session_surfaces_E_SES_001", func(t *testing.T) {
		t.Parallel()

		// BC-2.08.001 PC-1 / EC-001 — unknown session → E-SES-001.
		addr := startFakeServer(t, nil, func(cmd string, _ json.RawMessage) (any, error) {
			if cmd != "console.attach" {
				return nil, fmt.Errorf("unexpected command: %q", cmd)
			}
			return nil, fmt.Errorf("E-SES-001: session not found: no-such-session")
		})

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		err := runConsoleAttach(ctx, addr, testdataKeyPath(t), false, []string{
			"--session", "no-such-session",
		}, defaultIO())

		if err == nil {
			t.Fatal("AC-001 / EC-001 — expected E-SES-001 error; got nil")
		}
		// F-P2L2-007: assert "E-SES-001:" (colon suffix) — stricter than Contains("E-SES-001")
		// so a hypothetical E-SES-0010 would not spuriously match.
		if !strings.Contains(err.Error(), "E-SES-001:") {
			t.Errorf("AC-001 / EC-001 — expected E-SES-001: in error; got: %v", err)
		}
	})

	t.Run("auth_denied_surfaces_E_ADM_006", func(t *testing.T) {
		t.Parallel()

		// BC-2.08.001 Inv-1 / EC-003 — auth denied → E-ADM-006 in E-RPC-011 envelope.
		addr := startFakeServer(t, nil, func(cmd string, _ json.RawMessage) (any, error) {
			if cmd != "console.attach" {
				return nil, fmt.Errorf("unexpected command: %q", cmd)
			}
			return nil, fmt.Errorf("E-ADM-006: authorization denied for console.attach")
		})

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		err := runConsoleAttach(ctx, addr, testdataKeyPath(t), false, []string{
			"--session", "agent-01",
		}, defaultIO())

		if err == nil {
			t.Fatal("AC-001 / EC-003 — expected E-ADM-006 error; got nil")
		}
		// F-P2L2-007: assert "E-ADM-006:" (colon suffix) — stricter than Contains("E-ADM-006").
		if !strings.Contains(err.Error(), "E-ADM-006:") {
			t.Errorf("AC-001 / EC-003 — expected E-ADM-006: in error; got: %v", err)
		}
	})
}

// TestSbctlConsole_Detach verifies AC-002:
// `sbctl console detach --target <console_addr>` sends a JSON detach-request
// over the mgmt-plane Unix socket, and surfaces:
//   - success path (session detached, not closed),
//   - E-SES-004 (not attached for command).
//
// BC-2.08.001 PC-2 — console.detach RPC dispatched; session not closed.
// AC-002.
// RULING-W6TB-C — transport is mgmt-plane Unix socket.
func TestSbctlConsole_Detach(t *testing.T) {
	t.Parallel()

	t.Run("success_dispatches_console_detach_rpc", func(t *testing.T) {
		t.Parallel()

		// BC-2.08.001 PC-2 — console.detach RPC dispatched; session not closed.
		// AC-002 success path.
		requestCh := make(chan adminRPCRequest, 1)
		addr := startFakeServer(t, requestCh, func(cmd string, _ json.RawMessage) (any, error) {
			if cmd != "console.detach" {
				return nil, fmt.Errorf("unexpected command: %q; want console.detach", cmd)
			}
			return map[string]string{"session_name": "agent-01"}, nil
		})

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		err := runConsoleDetach(ctx, addr, testdataKeyPath(t), false, nil, defaultIO())
		if err != nil {
			t.Fatalf("AC-002 success — runConsoleDetach: %v", err)
		}

		select {
		case req := <-requestCh:
			if req.Command != "console.detach" {
				t.Errorf("AC-002 — dispatched command: got %q; want console.detach", req.Command)
			}
		case <-time.After(2 * time.Second):
			t.Error("AC-002: timed out waiting for console.detach RPC")
		}
	})

	t.Run("not_attached_surfaces_E_SES_004", func(t *testing.T) {
		t.Parallel()

		// BC-2.08.001 PC-2 / EC-002 — not attached → E-SES-004.
		addr := startFakeServer(t, nil, func(cmd string, _ json.RawMessage) (any, error) {
			if cmd != "console.detach" {
				return nil, fmt.Errorf("unexpected command: %q", cmd)
			}
			return nil, fmt.Errorf("E-SES-004: no console attached for command")
		})

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		err := runConsoleDetach(ctx, addr, testdataKeyPath(t), false, nil, defaultIO())

		if err == nil {
			t.Fatal("AC-002 / EC-002 — expected E-SES-004 error; got nil")
		}
		// F-P2L2-007: assert "E-SES-004:" (colon suffix) — stricter than Contains("E-SES-004").
		if !strings.Contains(err.Error(), "E-SES-004:") {
			t.Errorf("AC-002 / EC-002 — expected E-SES-004: in error; got: %v", err)
		}
	})
}

// TestSbctlConsole_Switch verifies AC-003:
// `sbctl console switch --target <console_addr> --session <name>` sends a JSON
// switch-request over the mgmt-plane Unix socket, and surfaces:
//   - success path (atomic detach+attach),
//   - E-SES-001 (unknown session — attach leg),
//   - E-SES-004 (not attached for command — detach leg),
//   - E-ADM-006 wrapped in E-RPC-011 (auth denied).
//
// BC-2.08.001 PC-3 — console.switch RPC dispatched with correct session_name.
// AC-003.
// RULING-W6TB-C — transport is mgmt-plane Unix socket.
func TestSbctlConsole_Switch(t *testing.T) {
	t.Parallel()

	t.Run("success_dispatches_console_switch_rpc", func(t *testing.T) {
		t.Parallel()

		// BC-2.08.001 PC-3 — console.switch RPC dispatched with correct session_name.
		// AC-003 success path: atomic detach+attach.
		const wantSession = "agent-02"

		requestCh := make(chan adminRPCRequest, 1)
		addr := startFakeServer(t, requestCh, func(cmd string, args json.RawMessage) (any, error) {
			if cmd != "console.switch" {
				return nil, fmt.Errorf("unexpected command: %q; want console.switch", cmd)
			}
			var a consoleSwitchArgs
			if err := json.Unmarshal(args, &a); err != nil {
				return nil, fmt.Errorf("unmarshal consoleSwitchArgs: %w", err)
			}
			if a.SessionName != wantSession {
				return nil, fmt.Errorf("session_name: got %q; want %q", a.SessionName, wantSession)
			}
			return map[string]string{"session_name": a.SessionName}, nil
		})

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		err := runConsoleSwitch(ctx, addr, testdataKeyPath(t), false, []string{
			"--session", wantSession,
		}, defaultIO())
		if err != nil {
			t.Fatalf("AC-003 success — runConsoleSwitch: %v", err)
		}

		select {
		case req := <-requestCh:
			if req.Command != "console.switch" {
				t.Errorf("AC-003 — dispatched command: got %q; want console.switch", req.Command)
			}
			var a consoleSwitchArgs
			if err := json.Unmarshal(req.Args, &a); err != nil {
				t.Fatalf("AC-003 — unmarshal args: %v", err)
			}
			if a.SessionName != wantSession {
				t.Errorf("AC-003 — wire session_name: got %q; want %q", a.SessionName, wantSession)
			}
		case <-time.After(2 * time.Second):
			t.Error("AC-003: timed out waiting for console.switch RPC")
		}
	})

	t.Run("unknown_session_surfaces_E_SES_001", func(t *testing.T) {
		t.Parallel()

		// BC-2.08.001 PC-3 / EC-001 — unknown session → E-SES-001 (attach leg).
		addr := startFakeServer(t, nil, func(cmd string, _ json.RawMessage) (any, error) {
			if cmd != "console.switch" {
				return nil, fmt.Errorf("unexpected command: %q", cmd)
			}
			return nil, fmt.Errorf("E-SES-001: session not found: no-such-session")
		})

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		err := runConsoleSwitch(ctx, addr, testdataKeyPath(t), false, []string{
			"--session", "no-such-session",
		}, defaultIO())

		if err == nil {
			t.Fatal("AC-003 / EC-001 — expected E-SES-001 error; got nil")
		}
		// F-P2L2-007: assert "E-SES-001:" (colon suffix) — stricter than Contains("E-SES-001").
		if !strings.Contains(err.Error(), "E-SES-001:") {
			t.Errorf("AC-003 / EC-001 — expected E-SES-001: in error; got: %v", err)
		}
	})

	t.Run("not_attached_surfaces_E_SES_004", func(t *testing.T) {
		t.Parallel()

		// BC-2.08.001 PC-3 / EC-002 — not attached → E-SES-004 (detach leg).
		addr := startFakeServer(t, nil, func(cmd string, _ json.RawMessage) (any, error) {
			if cmd != "console.switch" {
				return nil, fmt.Errorf("unexpected command: %q", cmd)
			}
			return nil, fmt.Errorf("E-SES-004: no console attached for command")
		})

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		err := runConsoleSwitch(ctx, addr, testdataKeyPath(t), false, []string{
			"--session", "agent-02",
		}, defaultIO())

		if err == nil {
			t.Fatal("AC-003 / EC-002 — expected E-SES-004 error; got nil")
		}
		// F-P2L2-007: assert "E-SES-004:" (colon suffix) — stricter than Contains("E-SES-004").
		if !strings.Contains(err.Error(), "E-SES-004:") {
			t.Errorf("AC-003 / EC-002 — expected E-SES-004: in error; got: %v", err)
		}
	})

	t.Run("auth_denied_surfaces_E_ADM_006", func(t *testing.T) {
		t.Parallel()

		// BC-2.08.001 Inv-1 / EC-003 — auth denied → E-ADM-006 in E-RPC-011 envelope.
		addr := startFakeServer(t, nil, func(cmd string, _ json.RawMessage) (any, error) {
			if cmd != "console.switch" {
				return nil, fmt.Errorf("unexpected command: %q", cmd)
			}
			return nil, fmt.Errorf("E-ADM-006: authorization denied for console.switch")
		})

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		err := runConsoleSwitch(ctx, addr, testdataKeyPath(t), false, []string{
			"--session", "agent-02",
		}, defaultIO())

		if err == nil {
			t.Fatal("AC-003 / EC-003 — expected E-ADM-006 error; got nil")
		}
		// F-P2L2-007: assert "E-ADM-006:" (colon suffix) — stricter than Contains("E-ADM-006").
		if !strings.Contains(err.Error(), "E-ADM-006:") {
			t.Errorf("AC-003 / EC-003 — expected E-ADM-006: in error; got: %v", err)
		}
	})
}
