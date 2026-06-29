// Package mgmt implements the daemon-side management server per ADR-012.
// It listens on a Unix socket or TCP address, performs the Ed25519 challenge-response
// handshake, and dispatches authenticated RPC commands to registered handlers.
//
// Purity classification (ARCH-09): boundary — owns listener I/O and socket state;
// pure-core logic (challenge generation, signature verify) lives in crypto/ed25519
// and crypto/rand directly (not re-wrapped here).
//
// Package DAG: internal/mgmt MUST NOT import internal/admission or any data-plane
// package (routing, multipath, arq, replay, paths, halfchannel, session, tmux,
// discovery). Only stdlib is imported here. See ARCH-12 §Package DAG Constraints.
package mgmt

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// MaxMessageBytes is the maximum JSON message size accepted on the management
// socket (server or client side). io.LimitReader MUST be applied before any
// json.Decoder.Decode call. 64 KiB is generous for all management RPCs.
// Defined here per ADR-012 §6 Bounded Read (CWE-400); BC-2.07.004 PC-6.
const MaxMessageBytes = 1 << 16 // 64 KiB

// HandshakeTimeout is the read deadline applied during the challenge-response
// handshake (from CHALLENGE sent to CHALLENGE_RESPONSE received). Default 10s.
// Closes EC-001 CWE-400 gap (ADR-012 §7 / BC-2.07.004 PC-1 / Ruling 1).
const HandshakeTimeout = 10 * time.Second

// RPCIdleTimeout is the read deadline applied after AUTH_OK is sent, while
// waiting for the first RPC request. Default 30s (ADR-012 §7 / Ruling 1).
const RPCIdleTimeout = 30 * time.Second

// MaxConcurrentConnections is the default semaphore size for concurrent
// per-connection goroutines. Excess connections back-pressure into the OS
// accept backlog. Prevents CWE-770 fd/goroutine exhaustion (ADR-012 §8 / Ruling 3).
const MaxConcurrentConnections = 128

// Option is a functional option for NewServer.
type Option func(*Server)

// WithHandshakeTimeout overrides the HandshakeTimeout used for the challenge-response
// phase. The default is HandshakeTimeout (10s). This option exists to allow tests
// to use a short timeout without waiting 10s (AC-001, VP-064 sub-case a).
func WithHandshakeTimeout(d time.Duration) Option {
	return func(s *Server) {
		s.handshakeTimeout = d
	}
}

// WithMaxConnections overrides the MaxConcurrentConnections semaphore size.
// The default is MaxConcurrentConnections (128). This option exists to allow
// tests to verify connection-cap back-pressure with a small cap (AC-013).
func WithMaxConnections(n int) Option {
	return func(s *Server) {
		s.sem = make(chan struct{}, n)
	}
}

// WithRPCIdleTimeout overrides the per-handler execution timeout (Ruling R /
// BC-2.07.004 PC-6 / AC-020). The default is RPCIdleTimeout (30s). This
// option exists to allow tests to verify handler-timeout behaviour without
// waiting 30s for the production default.
func WithRPCIdleTimeout(d time.Duration) Option {
	return func(s *Server) {
		s.rpcIdleTimeout = d
	}
}

// Handler is a registered command handler. Command is the RPC command name
// (e.g. "svtn.list"). Fn receives the authenticated connection context and the
// raw args JSON, and returns a data value (marshaled into the response envelope)
// or an error.
type Handler struct {
	Command string
	Fn      func(ctx context.Context, args json.RawMessage) (any, error)
}

// OperatorKeySet holds the set of authorized operator public keys for this daemon.
// IsAuthorized is safe for concurrent use.
type OperatorKeySet struct {
	keys []ed25519.PublicKey
}

// NewOperatorKeySet creates an OperatorKeySet from a slice of authorized public keys.
// Keys are copied; the caller's slice is not retained. When keys is empty or nil,
// the set is in bootstrap mode — IsBootstrap() returns true.
func NewOperatorKeySet(keys []ed25519.PublicKey) *OperatorKeySet {
	copied := make([]ed25519.PublicKey, len(keys))
	copy(copied, keys)
	return &OperatorKeySet{keys: copied}
}

// IsBootstrap reports whether no authorized operator keys were configured.
// In bootstrap mode the caller authorizes by comparing against the daemon's own key.
func (o *OperatorKeySet) IsBootstrap() bool {
	return len(o.keys) == 0
}

// IsAuthorized reports whether pubkey appears in the authorized set.
// Uses constant-time comparison to prevent timing oracle on key enumeration
// (BC-2.07.004 PC-8 / Inv-5 / AC-008).
func (o *OperatorKeySet) IsAuthorized(pubkey ed25519.PublicKey) bool {
	for _, k := range o.keys {
		if subtle.ConstantTimeCompare([]byte(k), []byte(pubkey)) == 1 {
			return true
		}
	}
	return false
}

// Server is the management plane server. It is started once per daemon mode.
// Construct via NewServer; never copy after first use.
type Server struct {
	ln               net.Listener
	daemonKey        ed25519.PrivateKey
	ops              *OperatorKeySet
	handlers         []Handler
	daemonVersion    string
	handshakeTimeout time.Duration
	rpcIdleTimeout   time.Duration // per-handler execution timeout (Ruling R / AC-020)
	sem              chan struct{} // bounded accept semaphore (CWE-770)
	connWG           sync.WaitGroup
	// shuttingDown is set to true before s.ln.Close() in both Shutdown and the
	// ctx-watcher goroutine. The Accept-error path in Serve checks this flag so
	// it can return nil on intentional shutdown vs. the real error on a fatal
	// Accept failure (BC-2.07.004 PC-10 / AC-017 / VP-069).
	shuttingDown atomic.Bool

	// mu protects the active connection set. Never hold mu while calling conn.Close().
	mu    sync.Mutex
	conns map[net.Conn]struct{}
}

// NewServer constructs a Server with the given listener, operator key set,
// and registered handlers. No init() functions — all dependencies injected.
// daemonKey is the daemon's own Ed25519 private key, used to sign challenges.
// daemonVersion is the semver string embedded in AUTH_OK messages — must be
// non-empty (e.g. "1.2.3" from ldflags, or "dev" for unreleased builds).
// NewServer panics if daemonVersion is empty; this enforces the invariant that
// the build system always injects a version string (ADR-012 §Ruling 6 / AC-007).
func NewServer(
	ln net.Listener,
	daemonKey ed25519.PrivateKey,
	ops *OperatorKeySet,
	handlers []Handler,
	daemonVersion string,
	opts ...Option,
) *Server {
	// Panic on empty daemonVersion: enforces that the build system injects a version.
	// The sentinel "dev" is accepted — it is the defined unreleased-build value.
	// An empty string is a defect (hardcoded "" or a missing ldflags wiring).
	if daemonVersion == "" {
		panic("mgmt.NewServer: daemonVersion must not be empty (ADR-012 §Ruling 6)")
	}
	// Panic on wrong-sized daemonKey: guards against nil key and public-key-for-private-key
	// mistakes at construction time (BC-2.07.004 Invariant 8 / AC-016 / VP-068).
	// A nil or short key would panic mid-connection inside handleConnection — a remote DoS
	// vector. Fail at NewServer instead.
	if len(daemonKey) != ed25519.PrivateKeySize {
		panic("mgmt.NewServer: daemonKey must be ed25519.PrivateKeySize bytes (ADR-012 §Invariant 8 / AC-016)")
	}

	s := &Server{
		ln:               ln,
		daemonKey:        daemonKey,
		ops:              ops,
		handlers:         handlers,
		daemonVersion:    daemonVersion,
		handshakeTimeout: HandshakeTimeout,
		rpcIdleTimeout:   RPCIdleTimeout,
		sem:              make(chan struct{}, MaxConcurrentConnections),
		conns:            make(map[net.Conn]struct{}),
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// trackConn registers conn in the active connection set.
func (s *Server) trackConn(conn net.Conn) {
	s.mu.Lock()
	s.conns[conn] = struct{}{}
	s.mu.Unlock()
}

// untrackConn removes conn from the active connection set.
func (s *Server) untrackConn(conn net.Conn) {
	s.mu.Lock()
	delete(s.conns, conn)
	s.mu.Unlock()
}

// closeAllConns snapshots the connection set under the lock, then closes each
// connection outside the lock. This prevents holding mu while calling Close(),
// which would deadlock if Close() triggers untrackConn (go.md rule 12).
func (s *Server) closeAllConns() {
	s.mu.Lock()
	snapshot := make([]net.Conn, 0, len(s.conns))
	for conn := range s.conns {
		snapshot = append(snapshot, conn)
	}
	s.mu.Unlock()
	for _, conn := range snapshot {
		_ = conn.Close()
	}
}

// Serve accepts connections and handles them until ctx is cancelled or the
// listener is closed. Returns nil on normal Shutdown, or a non-nil error on
// unexpected listener failure. Safe to call from a wg-tracked goroutine in the
// daemon lifecycle per ARCH-01 §Goroutine WaitGroup Contract.
//
// Serve is the sole owner of connWG.Wait(). Shutdown does NOT call connWG.Wait()
// (doing so would create a concurrent Wait race). Callers observe full drain
// completion by waiting for Serve to return (e.g., via mgmtWG.Wait() in the
// daemon lifecycle). Shutdown returns as soon as it has signalled the shutdown
// (marked shuttingDown, closed listener, force-closed connections).
func (s *Server) Serve(ctx context.Context) error {
	// ctx-watcher goroutine: closes the listener when the context is cancelled so
	// Accept unblocks. Uses a done channel to ensure this goroutine exits when
	// Serve returns (preventing a goroutine leak if Shutdown is never called).
	done := make(chan struct{})
	defer close(done)

	go func() {
		select {
		case <-ctx.Done():
			// Mark as shutting down BEFORE closing the listener so the Accept-error
			// path in Serve returns nil (VP-069 / PC-10 / AC-017).
			s.shuttingDown.Store(true)
			_ = s.ln.Close()
			s.closeAllConns()
		case <-done:
			// Serve returned (e.g. Shutdown called) — exit cleanly.
		}
	}()

	var backoff time.Duration
	for {
		// Acquire semaphore slot before Accept so we block here (not after spawning
		// a goroutine) when at capacity. This provides back-pressure into the OS
		// accept queue without spawning unbounded goroutines (CWE-770 / AC-013).
		select {
		case s.sem <- struct{}{}:
		case <-ctx.Done():
			// ctx cancelled — drain in-flight connections and exit. Serve is the
			// sole owner of connWG.Wait() (Shutdown does NOT call Wait — that would
			// race with this call). closeAllConns was already called by the
			// ctx-watcher goroutine above, so Wait should return quickly.
			s.connWG.Wait()
			return nil
		}

		conn, err := s.ln.Accept()
		if err != nil {
			// Release the semaphore slot we pre-acquired.
			<-s.sem

			// If we are shutting down (Shutdown called or ctx cancelled), the
			// listener close is intentional — return nil, not net.ErrClosed
			// (BC-2.07.004 PC-10 / AC-017 / VP-069 / Ruling G).
			// The ctx.Err() conjunct is required: an unexpected external listener
			// close (ctx live, Shutdown never called) must return the real error.
			// Drain in-flight connections before returning (Serve is the sole
			// owner of connWG.Wait — Shutdown does not call it).
			if s.shuttingDown.Load() || (errors.Is(err, net.ErrClosed) && ctx.Err() != nil) {
				s.connWG.Wait()
				return nil
			}

			// Transient error: back off exponentially (5ms→1s) and retry.
			// Non-temporary fatal errors cause Serve to return.
			if ne, ok := err.(net.Error); ok && ne.Temporary() { //nolint:staticcheck // net.Error.Temporary deprecated but still needed for EMFILE
				if backoff == 0 {
					backoff = 5 * time.Millisecond
				} else {
					backoff *= 2
					if backoff > time.Second {
						backoff = time.Second
					}
				}
				select {
				case <-time.After(backoff):
				case <-ctx.Done():
					s.connWG.Wait()
					return nil
				}
				continue
			}

			// Fatal accept error — force-close in-flight connections then drain.
			// closeAllConns() must precede connWG.Wait() so that in-flight
			// connections are signalled to close; without this, Wait() can stall
			// for up to RPCIdleTimeout (30s) waiting for idle connections to time
			// out naturally (Ruling P / BC-2.07.004 PC-10 / VP-069 v1.2).
			s.closeAllConns()
			s.connWG.Wait()
			return err
		}
		backoff = 0

		// Ruling I: drop connections accepted in the shutdown window before they
		// enter the WaitGroup. This prevents connWG.Add(1) racing connWG.Wait()
		// (Add-after-Wait panic) and ensures closeAllConns sees every tracked conn.
		if s.shuttingDown.Load() {
			_ = conn.Close()
			<-s.sem
			continue
		}

		// Track the connection BEFORE connWG.Add and BEFORE the goroutine spawn
		// so that closeAllConns sees it immediately and drain is complete within
		// the shutdown budget (Ruling I).
		s.trackConn(conn)
		s.connWG.Add(1)
		go func() {
			defer s.connWG.Done()
			defer func() { <-s.sem }() // release semaphore when connection goroutine exits
			defer s.untrackConn(conn)
			s.handleConnection(ctx, conn)
		}()
	}
}

// Shutdown signals the management server to stop accepting new connections,
// force-closes all in-flight connections, and returns. Full drain completion
// is observed by waiting for Serve to return (e.g., via mgmtWG.Wait() in the
// daemon lifecycle — see startMgmtServer).
//
// Shutdown does NOT call connWG.Wait() because Serve is the sole owner of
// connWG.Wait(). Concurrent Wait() calls from both Shutdown and Serve would
// race on WaitGroup internals (Ruling H). Serve calls connWG.Wait() after
// its accept loop exits and returns; the caller's mgmtWG.Wait() then confirms
// all goroutines are complete.
//
// Called by the daemon on SIGTERM/context cancel. Returns nil on success.
func (s *Server) Shutdown(_ context.Context) error {
	// Mark as shutting down BEFORE closing the listener so the Accept-error path
	// in Serve returns nil instead of net.ErrClosed (VP-069 / PC-10).
	s.shuttingDown.Store(true)
	// Close the listener so Serve's Accept loop unblocks and returns.
	_ = s.ln.Close()
	// Force-close all in-flight connections so their handleConnection goroutines
	// return quickly, allowing Serve's connWG.Wait() to unblock promptly.
	s.closeAllConns()
	return nil
}

// challengeMsg is the CHALLENGE message sent by the server on each new connection
// per ADR-012 §3 step 2.
type challengeMsg struct {
	Type      string `json:"type"`
	Nonce     string `json:"nonce"`
	DaemonSig string `json:"daemon_sig"`
}

// challengeResponseMsg is the CHALLENGE_RESPONSE message sent by the client
// per ADR-012 §3 step 3.
type challengeResponseMsg struct {
	Type     string `json:"type"`
	NonceSig string `json:"nonce_sig"`
	Pubkey   string `json:"pubkey"`
}

// authOKMsg is the AUTH_OK message sent on successful authentication per ADR-012 §3 step 5a.
type authOKMsg struct {
	Type          string `json:"type"`
	DaemonVersion string `json:"daemon_version"`
}

// authFailMsg is the AUTH_FAIL message sent on authentication failure per ADR-012 §3 step 5b.
type authFailMsg struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// rpcRequestMsg is an authenticated RPC request from the client per ADR-012 §3 step 6.
type rpcRequestMsg struct {
	Type    string          `json:"type"`
	ID      string          `json:"id"`
	Command string          `json:"command"`
	Args    json.RawMessage `json:"args"`
}

// rpcResponseMsg wraps a handler result in the standard JSON envelope per
// interface-definitions.md §JSON Output Schema.
type rpcResponseMsg struct {
	Type  string    `json:"type"`
	ID    string    `json:"id"`
	OK    bool      `json:"ok"`
	Error *rpcError `json:"error"`
	Data  any       `json:"data"`
}

// rpcError is the error object in the response envelope.
type rpcError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// sendJSON encodes v as a newline-terminated JSON object and writes it to conn.
func sendJSON(conn net.Conn, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	data = append(data, '\n')
	_, err = conn.Write(data)
	return err
}

// sendAuthFail sends an AUTH_FAIL message (with write deadline d) and closes conn.
// Helper to avoid repetition across all auth failure paths.
// d is applied as a write deadline before the send so that a non-draining client
// cannot pin the goroutine indefinitely (PC-1 write-deadline / Ruling E / VP-072).
func sendAuthFail(conn net.Conn, d time.Duration) {
	_ = conn.SetWriteDeadline(time.Now().Add(d))
	_ = sendJSON(conn, authFailMsg{
		Type:    "auth_fail",
		Code:    "E-ADM-010",
		Message: "authentication failed",
	})
	_ = conn.SetWriteDeadline(time.Time{})
	_ = conn.Close()
}

// handleConnection performs the ADR-012 auth handshake on a single connection:
// send CHALLENGE, read CHALLENGE_RESPONSE (via io.LimitReader), verify signature,
// send AUTH_OK or AUTH_FAIL, then dispatch authenticated RPCs.
//
// All reads use io.LimitReader(conn, MaxMessageBytes) — CWE-400 / BC-2.07.004 PC-6.
func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
	defer func() {
		_ = conn.Close()
	}()

	// Step 1: Generate fresh 32-byte nonce from crypto/rand (per-connection, never reused).
	var nonceBytes [32]byte
	if _, err := rand.Read(nonceBytes[:]); err != nil {
		return
	}

	// Step 2: Sign the nonce with the daemon's private key to prevent nonce forgery by MITM.
	pub, ok := s.daemonKey.Public().(ed25519.PublicKey)
	if !ok {
		return
	}
	_ = pub // public key available for bootstrap comparison below
	daemonSig := ed25519.Sign(s.daemonKey, nonceBytes[:])

	// Step 3: Send CHALLENGE message immediately before reading any client data (PC-1).
	// Set write deadline before send to defend against slow/non-draining clients
	// pinning the goroutine forever (slowloris-on-write / PC-1 amended / VP-072 /
	// Ruling E). Use HandshakeTimeout for all handshake-phase writes.
	challenge := challengeMsg{
		Type:      "challenge",
		Nonce:     base64.RawURLEncoding.EncodeToString(nonceBytes[:]),
		DaemonSig: base64.RawURLEncoding.EncodeToString(daemonSig),
	}
	if err := conn.SetWriteDeadline(time.Now().Add(s.handshakeTimeout)); err != nil {
		return
	}
	if err := sendJSON(conn, challenge); err != nil {
		return
	}
	_ = conn.SetWriteDeadline(time.Time{})

	// Step 4: Apply HandshakeTimeout read deadline before reading CHALLENGE_RESPONSE.
	// Closes EC-001 CWE-400 gap (ADR-012 §7 / BC-2.07.004 PC-1 / Ruling 1).
	if err := conn.SetReadDeadline(time.Now().Add(s.handshakeTimeout)); err != nil {
		return
	}

	// Step 5: Read CHALLENGE_RESPONSE via io.LimitReader (CWE-400, PC-6).
	// Any message exceeding MaxMessageBytes causes decode error → connection close.
	dec := json.NewDecoder(io.LimitReader(conn, MaxMessageBytes))

	var cresp challengeResponseMsg
	if err := dec.Decode(&cresp); err != nil {
		// EOF, timeout, oversized message, or malformed JSON → fail closed.
		// On timeout (HandshakeTimeout expiry / silent stall) just close — no point
		// sending AUTH_FAIL to a non-responsive client (AC-001 VP-064 sub-case a).
		// On other errors (malformed JSON, oversized) send AUTH_FAIL.
		_ = conn.SetReadDeadline(time.Time{})
		if ne, ok := err.(net.Error); !ok || !ne.Timeout() {
			sendAuthFail(conn, s.handshakeTimeout)
		}
		return
	}

	// Clear the handshake read deadline.
	_ = conn.SetReadDeadline(time.Time{})

	// Step 6: Validate that type=="challenge_response".
	// Post-auth structural guard (AC-003 / VP-065 / BC-2.07.004 PC-3 / Ruling 7):
	// The authenticated bool is managed per-connection in this function. On entry
	// (first CHALLENGE_RESPONSE decode), authenticated is false. If any
	// challenge_response arrives after auth succeeds, the guard fires below.
	// Here we also reject non-challenge_response types (AC-005: RPC without auth).
	if cresp.Type != "challenge_response" {
		// Wrong type (e.g. "request") at the auth step → AUTH_FAIL + close.
		sendAuthFail(conn, s.handshakeTimeout)
		return
	}

	// Step 7: Decode nonce_sig and pubkey from base64url.
	nonceSig, err := base64.RawURLEncoding.DecodeString(cresp.NonceSig)
	if err != nil {
		sendAuthFail(conn, s.handshakeTimeout)
		return
	}
	pubkeyBytes, err := base64.RawURLEncoding.DecodeString(cresp.Pubkey)
	if err != nil {
		sendAuthFail(conn, s.handshakeTimeout)
		return
	}

	// Step 8: Validate pubkey length (ed25519 requires exactly 32 bytes).
	if len(pubkeyBytes) != ed25519.PublicKeySize {
		sendAuthFail(conn, s.handshakeTimeout)
		return
	}

	pubkey := ed25519.PublicKey(pubkeyBytes)

	// Step 9: Determine the effective authorized key set.
	// Bootstrap mode: daemon's own public key is the sole authorized key (PC-9).
	var authorized bool
	if s.ops.IsBootstrap() {
		daemonPub := s.daemonKey.Public().(ed25519.PublicKey)
		// Constant-time comparison for bootstrap check too (Inv-5).
		authorized = subtle.ConstantTimeCompare([]byte(pubkey), []byte(daemonPub)) == 1
	} else {
		authorized = s.ops.IsAuthorized(pubkey)
	}

	// Verify signature: ed25519.Verify(pubkey, nonceBytes, nonceSig).
	sigValid := ed25519.Verify(pubkey, nonceBytes[:], nonceSig)

	// Both checks must pass. Fail closed — same AUTH_FAIL for all failures (Inv-5, PC-8).
	if !authorized || !sigValid {
		sendAuthFail(conn, s.handshakeTimeout)
		return
	}

	// Step 10: Send AUTH_OK with the injected daemonVersion (PC-7 / Ruling 6).
	// daemonVersion comes from NewServer — never hardcoded here.
	// Write deadline guards against a slow client pinning the goroutine (Ruling E / VP-072).
	if err := conn.SetWriteDeadline(time.Now().Add(s.handshakeTimeout)); err != nil {
		return
	}
	if err := sendJSON(conn, authOKMsg{
		Type:          "auth_ok",
		DaemonVersion: s.daemonVersion,
	}); err != nil {
		return
	}
	_ = conn.SetWriteDeadline(time.Time{})

	// Connection is now authenticated.
	authenticated := true

	// Step 11: Apply s.rpcIdleTimeout before reading the first RPC.
	if err := conn.SetReadDeadline(time.Now().Add(s.rpcIdleTimeout)); err != nil {
		return
	}

	// Step 12: Dispatch authenticated RPCs until connection closes or ctx is done.
	// Each RPC read is bounded by a fresh io.LimitReader (PC-6 applies to all reads).
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Fresh LimitReader for each RPC message — CWE-400 / PC-6 / Inv-3.
		rpcDec := json.NewDecoder(io.LimitReader(conn, MaxMessageBytes))

		var req rpcRequestMsg
		if err := rpcDec.Decode(&req); err != nil {
			// EOF, deadline, oversized message → clean disconnect.
			_ = conn.SetReadDeadline(time.Time{})
			return
		}

		// Reset read deadline after successful decode.
		_ = conn.SetReadDeadline(time.Time{})

		// Post-auth structural guard (AC-003 / VP-065 / BC-2.07.004 PC-3 / Ruling 7):
		// After AUTH_OK, any further "challenge_response" message triggers E-ADM-010 + close.
		// The authenticated boolean is true at this point; this guard enforces that
		// the protocol state machine does not allow re-authentication on a live connection.
		if authenticated && req.Type == "challenge_response" {
			// Security event: post-auth challenge_response protocol violation.
			sendAuthFail(conn, s.handshakeTimeout)
			return
		}

		if req.Type != "request" {
			// Unknown type after auth → close (clean disconnect).
			return
		}

		// Find handler for the command.
		var handlerFn func(ctx context.Context, args json.RawMessage) (any, error)
		for _, h := range s.handlers {
			if h.Command == req.Command {
				handlerFn = h.Fn
				break
			}
		}

		var resp rpcResponseMsg
		resp.Type = "response"
		resp.ID = req.ID

		if handlerFn == nil {
			// PC-11 (Ruling C): unknown command → E-RPC-010; connection stays OPEN.
			resp.OK = false
			resp.Error = &rpcError{Code: "E-RPC-010", Message: "unknown command: " + req.Command}
			resp.Data = nil
		} else {
			// Wrap handler invocation with a per-call deadline derived from
			// rpcIdleTimeout. A blocked handler is cancelled after the timeout and
			// the connection stays open for subsequent requests (Ruling R / AC-020 /
			// BC-2.07.004 PC-6 / Inv-3). cancel is called explicitly at the end of
			// each iteration (not deferred to function end) to avoid context-leak
			// accumulation across loop iterations (govet lostcancel).
			callCtx, callCancel := context.WithTimeout(ctx, s.rpcIdleTimeout)
			data, err := handlerFn(callCtx, req.Args)
			callCancel()
			if err != nil {
				// PC-12 (Ruling C / Ruling R): handler error or timeout → E-RPC-011;
				// connection stays OPEN.
				resp.OK = false
				resp.Error = &rpcError{Code: "E-RPC-011", Message: err.Error()}
				resp.Data = nil
			} else {
				resp.OK = true
				resp.Error = nil
				resp.Data = data
			}
		}

		// Write deadline for RPC response (Ruling E / VP-072): use s.rpcIdleTimeout.
		if err := conn.SetWriteDeadline(time.Now().Add(s.rpcIdleTimeout)); err != nil {
			return
		}
		if err := sendJSON(conn, resp); err != nil {
			return
		}
		_ = conn.SetWriteDeadline(time.Time{})

		// Re-apply s.rpcIdleTimeout before reading the next RPC.
		if err := conn.SetReadDeadline(time.Now().Add(s.rpcIdleTimeout)); err != nil {
			return
		}
	}
}
