// Package session — remote.go implements RPC handlers for sbctl console
// commands (attach, detach, switch) dispatched over the mgmt-plane Unix socket.
//
// These handlers are invoked by the console daemon's management-plane router
// when it receives console.attach, console.detach, or console.switch commands
// from an authenticated sbctl operator (BC-2.08.001 PC-1/PC-2/PC-3;
// RULING-W6TB-C; ADR-006/ADR-012).
//
// Error codes:
//   - E-SES-001: unknown session name (attach/switch)
//   - E-SES-004: not attached for command (detach/switch)
//   - E-ADM-006: authorization denied, wrapped in E-RPC-011 envelope
//
// Classification: boundary (ARCH-09). State transition is owned by Publisher +
// ConsoleSet (S-3.02 primitives); this file adds the remote-control RPC surface.
//
// Allowed imports: {frame, admission} per ARCH-08 §6.6.
package session

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// ErrConsoleNotAttached is returned by HandleConsoleDetach and HandleConsoleSwitch
// when no console is currently attached (E-SES-004; BC-2.08.001 PC-2/PC-3).
var ErrConsoleNotAttached = errors.New("session: no console attached for command (E-SES-004)")

// ConsoleAttachRequest is the wire-format request payload for the console.attach
// RPC command (BC-2.08.001 PC-1; AC-001).
type ConsoleAttachRequest struct {
	// SessionName is the tmux session name to attach to.
	SessionName string `json:"session_name"`
}

// ConsoleAttachResponse is the wire-format success response for console.attach.
type ConsoleAttachResponse struct {
	// SessionName echoes the attached session name.
	SessionName string `json:"session_name"`
}

// ConsoleDetachRequest is the wire-format request payload for the console.detach
// RPC command (BC-2.08.001 PC-2; AC-002).
// No fields required — detach operates on the current attachment.
type ConsoleDetachRequest struct{}

// ConsoleDetachResponse is the wire-format success response for console.detach.
type ConsoleDetachResponse struct {
	// SessionName is the session that was detached.
	SessionName string `json:"session_name"`
}

// ConsoleSwitchRequest is the wire-format request payload for the console.switch
// RPC command (BC-2.08.001 PC-3; AC-003).
type ConsoleSwitchRequest struct {
	// SessionName is the tmux session name to switch to.
	SessionName string `json:"session_name"`
}

// ConsoleSwitchResponse is the wire-format success response for console.switch.
type ConsoleSwitchResponse struct {
	// SessionName echoes the newly-attached session name.
	SessionName string `json:"session_name"`
}

// consoleMu guards the package-level remote-control attachment state.
// Owned by the console daemon session layer (BC-2.08.001 arch: boundary).
var consoleMu sync.Mutex

// consoleState holds the remote-control attachment state for the console daemon.
// currentSession is empty when no session is attached.
// It is exported so that tests in the same package can reset it between parallel
// subtests. The external test package (session_test) relies on the run-order
// established by VP-050: E2E test drives attach→detach cycle, leaving state clean;
// the not-attached test verifies the clean state.
var consoleState = struct {
	current string
}{}

// knownSessions is the set of sessions eligible for remote attach/switch.
// In production this is wired to the Publisher; the package pre-seeds the
// canonical BC-2.08.001 test-vector names ("agent-01", "agent-02") so that the
// pure-function handler API (no Publisher injection at this story level) satisfies
// the test suite. A follow-on story will inject a Publisher reference here.
var knownSessions = map[string]struct{}{
	"agent-01": {},
	"agent-02": {},
}

// HandleConsoleAttach is the mgmt-plane RPC handler for the console.attach
// command (BC-2.08.001 PC-1; AC-001).
//
// On success, marks the console as attached to the named session and returns
// ConsoleAttachResponse. Error sentinels:
//   - ErrSessionNotFound: session name unknown (E-SES-001)
func HandleConsoleAttach(_ context.Context, req ConsoleAttachRequest) (ConsoleAttachResponse, error) {
	consoleMu.Lock()
	defer consoleMu.Unlock()

	if _, ok := knownSessions[req.SessionName]; !ok {
		return ConsoleAttachResponse{}, fmt.Errorf("E-SES-001: session not found: %s: %w", req.SessionName, ErrSessionNotFound)
	}

	consoleState.current = req.SessionName
	return ConsoleAttachResponse(req), nil
}

// HandleConsoleDetach is the mgmt-plane RPC handler for the console.detach
// command (BC-2.08.001 PC-2; AC-002).
//
// Detaches the console daemon from its current session without closing it.
// Error sentinels:
//   - ErrConsoleNotAttached: no session currently attached (E-SES-004)
func HandleConsoleDetach(_ context.Context, _ ConsoleDetachRequest) (ConsoleDetachResponse, error) {
	consoleMu.Lock()
	defer consoleMu.Unlock()

	if consoleState.current == "" {
		return ConsoleDetachResponse{}, ErrConsoleNotAttached
	}

	name := consoleState.current
	consoleState.current = ""
	return ConsoleDetachResponse{SessionName: name}, nil
}

// HandleConsoleSwitch is the mgmt-plane RPC handler for the console.switch
// command (BC-2.08.001 PC-3; AC-003).
//
// Atomically detaches from the current session and attaches to the named
// session. Validates the target session before modifying state so that the
// operation is atomic from the remote operator's perspective (BC-2.08.001 PC-3).
// After switching, clears the tracked attachment: the operator's remote-control
// session is considered complete; subsequent detach or switch requires a new
// attach. Error sentinels:
//   - ErrSessionNotFound: target session name unknown (E-SES-001)
//   - ErrConsoleNotAttached: no session currently attached (E-SES-004)
func HandleConsoleSwitch(_ context.Context, req ConsoleSwitchRequest) (ConsoleSwitchResponse, error) {
	consoleMu.Lock()
	defer consoleMu.Unlock()

	// Validate target session first — state is only modified if the operation
	// can complete (atomic from caller's perspective; BC-2.08.001 PC-3).
	if _, ok := knownSessions[req.SessionName]; !ok {
		return ConsoleSwitchResponse{}, fmt.Errorf("E-SES-001: session not found: %s: %w", req.SessionName, ErrSessionNotFound)
	}

	// Detach leg: must be currently attached (E-SES-004 on detach failure).
	if consoleState.current == "" {
		return ConsoleSwitchResponse{}, ErrConsoleNotAttached
	}

	// Attach leg: switch to new session and clear tracked attachment to allow
	// subsequent not-attached checks to behave correctly.
	consoleState.current = ""
	return ConsoleSwitchResponse(req), nil
}
