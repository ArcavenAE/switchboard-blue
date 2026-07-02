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
var ErrConsoleNotAttached = errors.New("E-SES-004: no console attached for command")

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

// SessionRegistry is the interface the ConsoleServer uses to validate whether a
// named session exists before attaching or switching to it (L1-C2; BC-2.08.001
// PC-1/PC-3). In production this is wired to *Publisher; in tests it can be a
// stub.
//
// Allowed imports: no dependency on internal/mgmt; pure interface (ARCH-08 §6.6).
type SessionRegistry interface {
	// Exists reports whether the named session is currently published.
	Exists(name string) bool
}

// ConsoleState holds the remote-control attachment state for the console daemon.
// It replaces the previous package-level vars to eliminate the parallel-test data
// race (L2-T1). Construct with NewConsoleState; the zero value is not usable.
//
// ConsoleState is safe for concurrent use.
type ConsoleState struct {
	mu      sync.Mutex
	current string // empty when no session is attached
}

// NewConsoleState returns a ConsoleState with no session attached.
func NewConsoleState() *ConsoleState {
	return &ConsoleState{}
}

// Current returns the name of the currently-attached session, or "" if no
// session is attached. Safe for concurrent use (go.md rule 12: value copy
// returned under lock).
func (cs *ConsoleState) Current() string {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.current
}

// ConsoleServer groups a SessionRegistry and a *ConsoleState for handler dispatch
// (L1-C2). Construct with NewConsoleServer.
type ConsoleServer struct {
	reg   SessionRegistry
	state *ConsoleState
}

// NewConsoleServer returns a ConsoleServer wired to the given registry and state.
// Both reg and state must be non-nil.
func NewConsoleServer(reg SessionRegistry, state *ConsoleState) *ConsoleServer {
	return &ConsoleServer{reg: reg, state: state}
}

// HandleConsoleAttach is the mgmt-plane RPC handler for the console.attach
// command (BC-2.08.001 PC-1; AC-001).
//
// On success, marks the console as attached to the named session and returns
// ConsoleAttachResponse. Error sentinels:
//   - ErrSessionNotFound: session name unknown (E-SES-001)
//
// Design: state-only, no ConsoleSet.Add (F-P2L1-002 intentional).
// The console daemon is a SINGLE physical console; it has one ConsoleState
// tracking which session it is "tuned to." ConsoleSet.Add wires a channel pair
// for frame fanout — that is the job of the actual console rendering process
// (S-3.02), not of this remote-control RPC. The remote-control RPC records the
// operator's intent (which session to follow); the rendering path acts on it.
//
// TOCTOU acknowledgment (F-P2L1-003): cs.reg.Exists is called without holding
// cs.state.mu. Between the check and the lock acquisition, the session could be
// unpublished. The consequence is harmless: cs.state.current is set to a name
// string, not to a live pointer. Any subsequent operation that uses this name
// (e.g., the rendering path) will re-validate against the live registry at that
// time, at which point the session's absence will be detected. Adding an atomic
// "attach-if-exists" operation to SessionRegistry would couple session and
// ConsoleState synchronization unnecessarily for this benefit; accepted.
func (cs *ConsoleServer) HandleConsoleAttach(_ context.Context, req ConsoleAttachRequest) (ConsoleAttachResponse, error) {
	if !cs.reg.Exists(req.SessionName) {
		return ConsoleAttachResponse{}, fmt.Errorf("E-SES-001: session not found: %s: %w", req.SessionName, ErrSessionNotFound)
	}

	cs.state.mu.Lock()
	defer cs.state.mu.Unlock()

	cs.state.current = req.SessionName
	return ConsoleAttachResponse(req), nil
}

// HandleConsoleDetach is the mgmt-plane RPC handler for the console.detach
// command (BC-2.08.001 PC-2; AC-002).
//
// Detaches the console daemon from its current session without closing it.
// Error sentinels:
//   - ErrConsoleNotAttached: no session currently attached (E-SES-004)
func (cs *ConsoleServer) HandleConsoleDetach(_ context.Context, _ ConsoleDetachRequest) (ConsoleDetachResponse, error) {
	cs.state.mu.Lock()
	defer cs.state.mu.Unlock()

	if cs.state.current == "" {
		return ConsoleDetachResponse{}, ErrConsoleNotAttached
	}

	name := cs.state.current
	cs.state.current = ""
	return ConsoleDetachResponse{SessionName: name}, nil
}

// HandleConsoleSwitch is the mgmt-plane RPC handler for the console.switch
// command (BC-2.08.001 PC-3; AC-003).
//
// Atomically detaches from the current session and attaches to the named
// session. Validates the target session before modifying state so that the
// operation is atomic from the remote operator's perspective (BC-2.08.001 PC-3).
// After switching, the tracked attachment is updated to the new session name
// (L1-C3 fix: previously incorrectly cleared to ""). Error sentinels:
//   - ErrSessionNotFound: target session name unknown (E-SES-001)
//   - ErrConsoleNotAttached: no session currently attached (E-SES-004)
//
// State-only design and TOCTOU: see HandleConsoleAttach godoc (F-P2L1-002/003).
func (cs *ConsoleServer) HandleConsoleSwitch(_ context.Context, req ConsoleSwitchRequest) (ConsoleSwitchResponse, error) {
	// Validate target session first — state is only modified if the operation
	// can complete (atomic from caller's perspective; BC-2.08.001 PC-3).
	if !cs.reg.Exists(req.SessionName) {
		return ConsoleSwitchResponse{}, fmt.Errorf("E-SES-001: session not found: %s: %w", req.SessionName, ErrSessionNotFound)
	}

	cs.state.mu.Lock()
	defer cs.state.mu.Unlock()

	// Detach leg: must be currently attached (E-SES-004 on detach failure).
	if cs.state.current == "" {
		return ConsoleSwitchResponse{}, ErrConsoleNotAttached
	}

	// Attach leg: switch to new session (L1-C3: set to req.SessionName, not "").
	cs.state.current = req.SessionName
	return ConsoleSwitchResponse(req), nil
}
