// Package session — auth.go defines SessionAuth, the Tier-2 per-session
// authorization component for the access node (BC-2.04.005; BC-2.05.003).
//
// SessionAuth maintains a per-session authorization list guarded by a
// sync.RWMutex. It implements the Authorizer interface declared in upstream.go,
// so that S-3.03's test suite can wire it in place of NoOpAuthorizer.
//
// S-3.03 implementer tasks:
//   - Task 5: implement Authorize (returns role or E-ADM-006)
//   - Task 6: implement RegisterKey (operator provisioning)
//   - Task 7: wire SessionAuth as the live Authorizer in upstream.go
//   - Task 8: confirm empty-tick frames pass Allow without rejection
//
// Classification: boundary (ARCH-09; mutable per-session state).
package session

import (
	"errors"
	"fmt"
	"sync"
)

// Role represents the Tier-2 access scope for a console on a named session.
// A console may be registered as RoleFull (read-write) or RoleReadOnly
// (downstream only; upstream keystrokes rejected per BC-2.04.005 PC-3).
type Role int

const (
	// RoleFull grants full read-write access: upstream keystrokes are forwarded.
	//
	// ZERO-VALUE NOTE: RoleFull == 0 (iota). authEntry values only enter the map
	// through RegisterKey, which always sets the role field explicitly. A zero-value
	// authEntry{} constructed outside RegisterKey would silently grant RoleFull —
	// do not bypass the constructor. If a future story adds a deny/unset sentinel,
	// insert it before RoleFull so the zero value becomes the most restrictive state.
	RoleFull Role = iota

	// RoleReadOnly grants downstream-only access: upstream payload-bearing frames
	// are rejected with E-ADM-007; empty-tick frames are accepted (BC-2.04.005
	// EC-004; AC-006).
	RoleReadOnly
)

// ErrSessionAuthDenied is returned by SessionAuth.Authorize when the console's
// key is not present in the named session's authorization list (E-ADM-006;
// BC-2.05.003 PC-2; AC-002).
//
// Error text must not have trailing punctuation (ST1005).
var ErrSessionAuthDenied = errors.New("session: authorization denied (E-ADM-006)")

// ErrUpstreamReadOnly is returned by SessionAuth.Allow when a payload-bearing
// upstream frame is received from a console whose role is RoleReadOnly
// (E-ADM-007; BC-2.04.005 PC-3; AC-005).
//
// Error text must not have trailing punctuation (ST1005).
var ErrUpstreamReadOnly = errors.New("session: upstream rejected: read-only access (E-ADM-007)")

// authEntry holds a console's Tier-2 authorization record for a single session.
type authEntry struct {
	role Role
}

// SessionAuth is the Tier-2 per-session authorization component. It maintains
// a map[sessionName]map[consoleKey]authEntry guarded by a sync.RWMutex, and
// implements the Authorizer interface (upstream.go) so that the access node's
// upstream-receive path can call Allow before forwarding keystrokes to tmux.
//
// Concurrency: SessionAuth is safe for concurrent use. Readers acquire RLock;
// writers (RegisterKey) acquire the full Lock. Allow and Authorize are reader
// paths.
//
// The zero value is not usable; construct with NewSessionAuth.
//
// Never return internal pointers from locked accessors: all exported read
// methods return value copies (S-3.02 precedent; Go quality rule §12).
type SessionAuth struct {
	mu       sync.RWMutex
	sessions map[string]map[ConsoleKey]authEntry // sessionName → consoleKey → entry
}

// NewSessionAuth constructs a ready-to-use SessionAuth with an empty
// authorization list.
func NewSessionAuth() *SessionAuth {
	return &SessionAuth{
		sessions: make(map[string]map[ConsoleKey]authEntry),
	}
}

// RegisterKey adds or replaces the authorization entry for consoleKey on the
// named session (operator provisioning; task 6). Last-write-wins per ADR-003.
// A previously registered key is overwritten without error.
//
// RegisterKey is safe for concurrent use (takes full write lock).
func (sa *SessionAuth) RegisterKey(sessionName string, key ConsoleKey, role Role) {
	sa.mu.Lock()
	defer sa.mu.Unlock()

	if sa.sessions[sessionName] == nil {
		sa.sessions[sessionName] = make(map[ConsoleKey]authEntry)
	}
	sa.sessions[sessionName][key] = authEntry{role: role}
}

// Authorize checks whether key is in the authorization list for sessionName,
// returning the console's role and a nil error on success, or E-ADM-006
// (ErrSessionAuthDenied) if the key is not present (BC-2.05.003 PC-1, PC-2;
// AC-001, AC-002).
//
// Authorization is per-session: a key authorized for session-A is not
// automatically authorized for session-B (BC-2.05.003 PC-4; AC-004).
//
// Authorize is safe for concurrent use (takes RLock).
func (sa *SessionAuth) Authorize(key ConsoleKey, sessionName string) (Role, error) {
	sa.mu.RLock()
	defer sa.mu.RUnlock()

	sessionKeys, ok := sa.sessions[sessionName]
	if !ok {
		return 0, fmt.Errorf("session authorization denied: console %s not authorized for session %s: %w",
			string(key), sessionName, ErrSessionAuthDenied)
	}
	entry, ok := sessionKeys[key]
	if !ok {
		return 0, fmt.Errorf("session authorization denied: console %s not authorized for session %s: %w",
			string(key), sessionName, ErrSessionAuthDenied)
	}
	return entry.role, nil
}

// Allow implements the Authorizer interface (upstream.go). It is called by
// AccessNode.SendKeystroke for every upstream frame before the frame is
// forwarded to the KeystrokeSink (tmux).
//
// Allow enforces two rules:
//  1. The console's key must be in the session authorization list; if absent,
//     ErrSessionAuthDenied (E-ADM-006) is returned (BC-2.05.003 PC-2).
//  2. If the key is registered as RoleReadOnly AND the payload is non-empty
//     (a keystroke), ErrUpstreamReadOnly (E-ADM-007) is returned (BC-2.04.005
//     PC-3; AC-005). Empty-payload frames (liveness probes / empty-tick) are
//     accepted even for read-only consoles (BC-2.04.005 EC-004; AC-006).
//
// The payload slice is not retained after Allow returns.
//
// Allow must be safe for concurrent calls from multiple goroutines (Authorizer
// interface contract).
func (sa *SessionAuth) Allow(key ConsoleKey, sessionName string, payload []byte) error {
	role, err := sa.Authorize(key, sessionName)
	if err != nil {
		return err
	}
	if role == RoleReadOnly && len(payload) > 0 {
		return fmt.Errorf("upstream rejected: read-only access for console %s on session %s: %w",
			string(key), sessionName, ErrUpstreamReadOnly)
	}
	return nil
}
