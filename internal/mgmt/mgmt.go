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
	"fmt"
	"io"
	"net"
	"sync"
)

// MaxMessageBytes is the maximum JSON message size accepted on the management
// socket (server or client side). io.LimitReader MUST be applied before any
// json.Decoder.Decode call. 64 KiB is generous for all management RPCs.
// Defined here per ADR-012 §6 Bounded Read (CWE-400); BC-2.07.004 PC-6.
const MaxMessageBytes = 1 << 16 // 64 KiB

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
//
// GREEN-BY-DESIGN: zero branching beyond the return expression, no I/O, no helpers,
// 1 line. Test for this passes immediately against the stub — expected and documented.
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
	ln        net.Listener
	daemonKey ed25519.PrivateKey
	ops       *OperatorKeySet
	handlers  []Handler
	connWG    sync.WaitGroup // tracks per-connection goroutines; used by Shutdown

	// mu protects the active connection set.
	mu    sync.Mutex
	conns map[net.Conn]struct{}
}

// NewServer constructs a Server with the given listener, operator key set,
// and registered handlers. No init() functions — all dependencies injected.
// daemonKey is the daemon's own Ed25519 private key, used to sign challenges.
func NewServer(
	ln net.Listener,
	daemonKey ed25519.PrivateKey,
	ops *OperatorKeySet,
	handlers []Handler,
) *Server {
	return &Server{
		ln:        ln,
		daemonKey: daemonKey,
		ops:       ops,
		handlers:  handlers,
		conns:     make(map[net.Conn]struct{}),
	}
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

// closeAllConns closes all tracked connections, unblocking their goroutines.
func (s *Server) closeAllConns() {
	s.mu.Lock()
	for conn := range s.conns {
		_ = conn.Close()
	}
	s.mu.Unlock()
}

// Serve accepts connections and handles them until ctx is cancelled or the
// listener is closed. Returns when all in-flight connections have terminated
// (WaitGroup-tracked). Safe to call from a wg-tracked goroutine in the daemon
// lifecycle per ARCH-01 §Goroutine WaitGroup Contract.
func (s *Server) Serve(ctx context.Context) error {
	// Close the listener when ctx is cancelled so Accept unblocks and returns.
	// Shutdown also calls ln.Close; the second close is a no-op error that's ignored.
	go func() {
		<-ctx.Done()
		_ = s.ln.Close()
		// Also force-close all in-flight connections so their goroutines exit promptly.
		s.closeAllConns()
	}()

	for {
		conn, err := s.ln.Accept()
		if err != nil {
			// Listener closed — drain in-flight connections, then return.
			s.connWG.Wait()
			return err
		}
		s.connWG.Add(1)
		go func() {
			defer s.connWG.Done()
			s.trackConn(conn)
			defer s.untrackConn(conn)
			s.handleConnection(ctx, conn)
		}()
	}
}

// Shutdown drains in-flight connections and closes the listener.
// Called by the daemon on SIGTERM/context cancel. Blocks until drained or
// ctx expires (mirrors drain timeout semantics from internal/drain).
func (s *Server) Shutdown(ctx context.Context) error {
	// Close the listener so Serve's Accept loop unblocks and returns.
	_ = s.ln.Close()
	// Force-close all in-flight connections so their goroutines return.
	s.closeAllConns()
	// Wait for all in-flight connection goroutines to finish.
	done := make(chan struct{})
	go func() {
		s.connWG.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("mgmt shutdown: %w", ctx.Err())
	}
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
	daemonSig := ed25519.Sign(s.daemonKey, nonceBytes[:])

	// Step 3: Send CHALLENGE message immediately before reading any client data (PC-1).
	challenge := challengeMsg{
		Type:      "challenge",
		Nonce:     base64.RawURLEncoding.EncodeToString(nonceBytes[:]),
		DaemonSig: base64.RawURLEncoding.EncodeToString(daemonSig),
	}
	if err := sendJSON(conn, challenge); err != nil {
		return
	}

	// Step 4: Read CHALLENGE_RESPONSE via io.LimitReader (CWE-400, PC-6).
	// Any message exceeding MaxMessageBytes causes decode error → connection close.
	dec := json.NewDecoder(io.LimitReader(conn, MaxMessageBytes))

	var cresp challengeResponseMsg
	if err := dec.Decode(&cresp); err != nil {
		// EOF, timeout, oversized message, or malformed JSON → fail closed, no AUTH_FAIL
		// (no point sending if connection is broken/oversized).
		_ = sendJSON(conn, authFailMsg{
			Type:    "auth_fail",
			Code:    "E-ADM-010",
			Message: "authentication failed",
		})
		return
	}

	// Step 5: Validate that type=="challenge_response" (PC-5: wrong type → AUTH_FAIL).
	if cresp.Type != "challenge_response" {
		_ = sendJSON(conn, authFailMsg{
			Type:    "auth_fail",
			Code:    "E-ADM-010",
			Message: "authentication failed",
		})
		return
	}

	// Step 6: Decode nonce_sig and pubkey from base64url.
	nonceSig, err := base64.RawURLEncoding.DecodeString(cresp.NonceSig)
	if err != nil {
		_ = sendJSON(conn, authFailMsg{
			Type:    "auth_fail",
			Code:    "E-ADM-010",
			Message: "authentication failed",
		})
		return
	}
	pubkeyBytes, err := base64.RawURLEncoding.DecodeString(cresp.Pubkey)
	if err != nil {
		_ = sendJSON(conn, authFailMsg{
			Type:    "auth_fail",
			Code:    "E-ADM-010",
			Message: "authentication failed",
		})
		return
	}

	// Step 7: Validate pubkey length before using (ed25519 requires exactly 32 bytes).
	if len(pubkeyBytes) != ed25519.PublicKeySize {
		_ = sendJSON(conn, authFailMsg{
			Type:    "auth_fail",
			Code:    "E-ADM-010",
			Message: "authentication failed",
		})
		return
	}

	pubkey := ed25519.PublicKey(pubkeyBytes)

	// Determine the effective authorized key set. In bootstrap mode (no operator keys
	// configured), the daemon's own public key is the sole authorized key (PC-9).
	var authorized bool
	if s.ops.IsBootstrap() {
		daemonPub := s.daemonKey.Public().(ed25519.PublicKey)
		// Use constant-time comparison for bootstrap check too (Inv-5).
		authorized = subtle.ConstantTimeCompare([]byte(pubkey), []byte(daemonPub)) == 1
	} else {
		authorized = s.ops.IsAuthorized(pubkey)
	}

	// Verify signature: ed25519.Verify(pubkey, nonceBytes, nonceSig).
	sigValid := ed25519.Verify(pubkey, nonceBytes[:], nonceSig)

	// Both checks must pass. Fail closed — same AUTH_FAIL for all failure modes (Inv-5, PC-8).
	if !authorized || !sigValid {
		_ = sendJSON(conn, authFailMsg{
			Type:    "auth_fail",
			Code:    "E-ADM-010",
			Message: "authentication failed",
		})
		return
	}

	// Step 8: Send AUTH_OK (PC-7).
	authOK := authOKMsg{
		Type:          "auth_ok",
		DaemonVersion: "dev",
	}
	if err := sendJSON(conn, authOK); err != nil {
		return
	}

	// Step 9: Dispatch authenticated RPCs until connection closes or ctx is done.
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
			// EOF, deadline, oversized message → clean disconnect
			return
		}

		if req.Type != "request" {
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
			resp.OK = false
			resp.Error = &rpcError{Code: "E-RPC-001", Message: "unknown command"}
			resp.Data = nil
		} else {
			data, err := handlerFn(ctx, req.Args)
			if err != nil {
				resp.OK = false
				resp.Error = &rpcError{Code: "E-RPC-002", Message: err.Error()}
				resp.Data = nil
			} else {
				resp.OK = true
				resp.Error = nil
				resp.Data = data
			}
		}

		if err := sendJSON(conn, resp); err != nil {
			return
		}
	}
}
