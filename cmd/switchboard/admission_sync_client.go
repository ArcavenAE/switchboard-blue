// admission_sync_client.go — control-side admission-state push client.
//
// admissionSyncer is the interface the four admin write handlers use to push
// admission-state changes to configured routers (S-BL.ADMISSION-SYNC-WIRE;
// BC-2.05.009 Rulings 1–2; decisions/S-BL.ADMISSION-SYNC-WIRE-rulings.md v1.2).
//
// A nil admissionSyncer is explicitly permitted — nil means "no routers configured"
// (single-router co-located deployment or router/console/access mode); methods
// are no-ops. Production: *admissionSyncClient. Tests: a mock/stub.
//
// ARCH-08 compliance: this file lives in cmd/switchboard (position 18, the top
// of the import DAG). It imports only internal/admission, internal/mgmt, and
// internal/config — all already imported by mgmt_wire.go.
//
// Purity classification (ARCH-09): boundary — effectful shell that dials TCP,
// sends JSON RPCs, and retries.

package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/config"
)

// errAdmissionSyncNotImplemented is kept for backward compatibility: test
// assertions that check errors.Is(err, errAdmissionSyncNotImplemented) use it
// to confirm the stub has been replaced by a real implementation.
// The real implementation never returns this sentinel.
var errAdmissionSyncNotImplemented = fmt.Errorf("admission sync: not implemented")

// Retry-with-backoff constants (BC-2.05.009 Ruling 2; documented per ruling):
//
//	initial delay: 100ms, multiplier: 2, max delay: 10s, max attempts: 5.
const (
	admissionSyncRetryInitial  = 100 * time.Millisecond
	admissionSyncRetryMaxDelay = 10 * time.Second
	admissionSyncRetryMax      = 5
)

// admissionSyncDialTimeout is the per-dial TCP connect timeout for pushRPC
// (F-P3-03 / S-BL.ADMISSION-SYNC-WIRE). Without a bounded timeout, a black-holed
// endpoint (SYN dropped, no RST) causes the dial to block for the OS TCP connect
// timeout (~127s on macOS/Linux), which stalls pushWG.Wait() at daemon shutdown for
// up to admissionSyncRetryMax * OS_timeout ≈ 10+ minutes.
//
// A 5s per-dial timeout gives: worst-case per-endpoint latency =
// 5 attempts × (5s dial + max_retry_sleep=10s) = 5 × 15s = 75s.
// This is a large reduction from ~127s×5=635s and is bounded, predictable,
// and well within operator-tolerable shutdown latency for a control daemon.
//
// The value is tunable via constant — it is NOT user-configurable because
// the retry policy (Ruling 2) already exposes the relevant knobs (attempts, delays).
const admissionSyncDialTimeout = 5 * time.Second

// Push command name constants used by both the control-side client and the
// router-side handler registration. Defined here so both files can reference
// them without duplication (AC-002 tests also reference these constants).
const (
	// CmdAdmissionRegister is the internal RPC command for RegisterKey push.
	CmdAdmissionRegister = "internal.admission.register"
	// CmdAdmissionRevoke is the internal RPC command for RevokeKey push.
	CmdAdmissionRevoke = "internal.admission.revoke"
	// CmdAdmissionExpire is the internal RPC command for SetKeyExpiry push.
	CmdAdmissionExpire = "internal.admission.expire"
	// CmdAdmissionRemoveSVTN is the internal RPC command for RemoveSVTN push.
	CmdAdmissionRemoveSVTN = "internal.admission.remove-svtn"
)

// admissionSyncer is the interface the four admin write handlers use to push
// admission-state changes to configured routers.
//
// A nil value is explicitly permitted — nil means "no routers configured";
// methods are no-ops. Production: *admissionSyncClient. Tests: a mock/stub.
//
// svtnID is the resolved [16]byte UUID — NOT the human-readable SVTN name.
// The admin handler (which holds *svtnmgmt.SVTNManager) resolves name→[16]byte
// via m.SVTNByName before calling Push*. The router has no SVTNManager and
// therefore no name→ID map; it must receive the [16]byte directly.
//
// Traces to BC-2.05.009; decisions/S-BL.ADMISSION-SYNC-WIRE-rulings.md v1.2
// Decision 5 (corrected interface — svtnID [16]byte, not svtnName string).
type admissionSyncer interface {
	PushRegisterKey(ctx context.Context, svtnID [16]byte, pubkey ed25519.PublicKey, role admission.KeyRole) error
	PushRevokeKey(ctx context.Context, svtnID [16]byte, pubkey ed25519.PublicKey, role admission.KeyRole, confirm bool) error
	PushSetKeyExpiry(ctx context.Context, svtnID [16]byte, pubkey ed25519.PublicKey, ttl time.Duration) error
	PushRemoveSVTN(ctx context.Context, svtnID [16]byte) error
}

// admissionSyncClient is the control-side push client. It dials each configured
// router management endpoint on demand, completes the mgmt challenge-response
// handshake, sends the internal.admission.* RPC, and reads the response.
//
// Dial-on-demand: no persistent idle connection. Retry-with-backoff per
// admissionSyncRetry* constants above (BC-2.05.009 Ruling 2 Decision 4).
//
// Thread-safe: endpoints are protected by mu; Push* methods may be called
// concurrently from different admin handler goroutines.
type admissionSyncClient struct {
	mu         sync.RWMutex
	endpoints  []config.RouterManagementEndpoint
	daemonPriv ed25519.PrivateKey
}

// newAdmissionSyncClient returns an *admissionSyncClient initialised with the
// given endpoints and daemonPriv key. The returned value satisfies admissionSyncer.
//
// endpoints may be empty — push methods become no-ops in that case.
func newAdmissionSyncClient(
	endpoints []config.RouterManagementEndpoint,
	daemonPriv ed25519.PrivateKey,
) *admissionSyncClient {
	return &admissionSyncClient{
		endpoints:  endpoints,
		daemonPriv: daemonPriv,
	}
}

// UpdateEndpoints replaces the client's endpoint list atomically.
// Called from runControl on SIGHUP reload (BC-2.05.009 Invariant 5 / AC-010).
// In-flight pushes are not interrupted; the new list takes effect for the next push.
func (c *admissionSyncClient) UpdateEndpoints(endpoints []config.RouterManagementEndpoint) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.endpoints = endpoints
}

// currentEndpoints returns a snapshot of the endpoint list under read lock.
func (c *admissionSyncClient) currentEndpoints() []config.RouterManagementEndpoint {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]config.RouterManagementEndpoint, len(c.endpoints))
	copy(out, c.endpoints)
	return out
}

// pushRPC dials addr, performs the ADR-012 challenge-response handshake using
// c.daemonPriv, sends the named command with argsJSON, and reads the response.
//
// Returns the first non-nil error encountered (dial, handshake, or RPC error).
// This is a synchronous, dial-on-demand call (Ruling 2 / Decision 4).
func (c *admissionSyncClient) pushRPC(ctx context.Context, addr, command string, argsJSON json.RawMessage) error {
	const handshakeTimeout = 10 * time.Second
	const maxMsg = 1 << 16 // 64 KiB per message

	// F-P3-03: use a bounded per-dial timeout so that a black-holed endpoint
	// (SYN dropped, no RST) fails in ≤ admissionSyncDialTimeout rather than
	// blocking for the OS TCP connect timeout (~127s). This makes pushWG.Wait()
	// bounded at daemon shutdown even against adversarial network conditions.
	dialer := &net.Dialer{Timeout: admissionSyncDialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("dial %s: %w", addr, err)
	}
	defer func() { _ = conn.Close() }()

	if err := conn.SetDeadline(time.Now().Add(handshakeTimeout)); err != nil {
		return fmt.Errorf("set deadline: %w", err)
	}

	// Step 1: read CHALLENGE from server.
	var challenge struct {
		Type      string `json:"type"`
		Nonce     string `json:"nonce"`
		DaemonSig string `json:"daemon_sig"`
	}
	if err := json.NewDecoder(io.LimitReader(conn, maxMsg)).Decode(&challenge); err != nil {
		return fmt.Errorf("read challenge: %w", err)
	}
	if challenge.Type != "challenge" {
		return fmt.Errorf("unexpected message type %q (want challenge)", challenge.Type)
	}

	nonceBytes, err := base64.RawURLEncoding.DecodeString(challenge.Nonce)
	if err != nil {
		return fmt.Errorf("decode nonce: %w", err)
	}

	// Step 2: sign the nonce with our daemon private key.
	nonceSig := ed25519.Sign(c.daemonPriv, nonceBytes)
	pub := c.daemonPriv.Public().(ed25519.PublicKey)

	cresp := struct {
		Type     string `json:"type"`
		NonceSig string `json:"nonce_sig"`
		Pubkey   string `json:"pubkey"`
	}{
		Type:     "challenge_response",
		NonceSig: base64.RawURLEncoding.EncodeToString(nonceSig),
		Pubkey:   base64.RawURLEncoding.EncodeToString([]byte(pub)),
	}
	data, err := json.Marshal(cresp)
	if err != nil {
		return fmt.Errorf("marshal challenge_response: %w", err)
	}
	data = append(data, '\n')
	if _, err := conn.Write(data); err != nil {
		return fmt.Errorf("send challenge_response: %w", err)
	}

	// Step 3: read AUTH_OK or AUTH_FAIL.
	var authResult struct {
		Type    string `json:"type"`
		Code    string `json:"code,omitempty"`
		Message string `json:"message,omitempty"`
	}
	if err := json.NewDecoder(io.LimitReader(conn, maxMsg)).Decode(&authResult); err != nil {
		return fmt.Errorf("read auth result: %w", err)
	}
	if authResult.Type != "auth_ok" {
		return fmt.Errorf("authentication failed: type=%q code=%q msg=%q",
			authResult.Type, authResult.Code, authResult.Message)
	}

	// Step 4: send the RPC request.
	reqID := fmt.Sprintf("sync-%d", time.Now().UnixNano())
	req := struct {
		Type    string          `json:"type"`
		ID      string          `json:"id"`
		Command string          `json:"command"`
		Args    json.RawMessage `json:"args"`
	}{
		Type:    "request",
		ID:      reqID,
		Command: command,
		Args:    argsJSON,
	}
	reqData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal rpc request: %w", err)
	}
	reqData = append(reqData, '\n')
	if _, err := conn.Write(reqData); err != nil {
		return fmt.Errorf("send rpc request: %w", err)
	}

	// Step 5: read the RPC response.
	var resp struct {
		Type  string `json:"type"`
		ID    string `json:"id"`
		OK    bool   `json:"ok"`
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(io.LimitReader(conn, maxMsg)).Decode(&resp); err != nil {
		return fmt.Errorf("read rpc response: %w", err)
	}
	if resp.Error != nil {
		return fmt.Errorf("rpc %s error: code=%q msg=%q", command, resp.Error.Code, resp.Error.Message)
	}
	return nil
}

// pushWithRetry calls pushRPC to all configured endpoints with retry-with-backoff
// (admissionSyncRetryInitial × 2 per attempt, capped at admissionSyncRetryMaxDelay,
// max admissionSyncRetryMax attempts per endpoint).
//
// Returns the last error from any endpoint, or nil if all succeeds. An empty
// endpoint list is a no-op (return nil). BC-2.05.009 Ruling 2 Decision 4.
func (c *admissionSyncClient) pushWithRetry(ctx context.Context, command string, argsJSON json.RawMessage) error {
	endpoints := c.currentEndpoints()
	if len(endpoints) == 0 {
		return nil
	}

	var lastErr error
	for _, ep := range endpoints {
		delay := admissionSyncRetryInitial
		for attempt := 0; attempt < admissionSyncRetryMax; attempt++ {
			err := c.pushRPC(ctx, ep.Addr, command, argsJSON)
			if err == nil {
				lastErr = nil
				break
			}
			lastErr = err
			if attempt+1 < admissionSyncRetryMax {
				select {
				case <-ctx.Done():
					return fmt.Errorf("context cancelled after %d attempts on %s: %w", attempt+1, ep.Addr, ctx.Err())
				case <-time.After(delay):
				}
				delay *= 2
				if delay > admissionSyncRetryMaxDelay {
					delay = admissionSyncRetryMaxDelay
				}
			}
		}
	}
	return lastErr
}

// PushRegisterKey pushes an internal.admission.register RPC to all configured
// router endpoints after a successful admin.key.register on control.
//
// svtn_id is encoded as 32 lowercase hex chars (BC-2.05.009 Inv-4).
// Retry-with-backoff per admissionSyncRetry* constants (Ruling 2 / Decision 4).
// Push error is advisory — callers log WARN and continue (PC-2).
func (c *admissionSyncClient) PushRegisterKey(
	ctx context.Context,
	svtnID [16]byte,
	pubkey ed25519.PublicKey,
	role admission.KeyRole,
) error {
	args := map[string]any{
		"svtn_id":        hex.EncodeToString(svtnID[:]),
		"pubkey_openssh": base64.RawURLEncoding.EncodeToString([]byte(pubkey)),
		"role":           role.String(),
	}
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("PushRegisterKey: marshal args: %w", err)
	}
	return c.pushWithRetry(ctx, CmdAdmissionRegister, argsJSON)
}

// PushRevokeKey pushes an internal.admission.revoke RPC to all configured
// router endpoints after a successful admin.key.revoke on control.
func (c *admissionSyncClient) PushRevokeKey(
	ctx context.Context,
	svtnID [16]byte,
	pubkey ed25519.PublicKey,
	role admission.KeyRole,
	confirm bool,
) error {
	args := map[string]any{
		"svtn_id":        hex.EncodeToString(svtnID[:]),
		"pubkey_openssh": base64.RawURLEncoding.EncodeToString([]byte(pubkey)),
		"role":           role.String(),
		"confirm":        confirm,
	}
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("PushRevokeKey: marshal args: %w", err)
	}
	return c.pushWithRetry(ctx, CmdAdmissionRevoke, argsJSON)
}

// PushSetKeyExpiry pushes an internal.admission.expire RPC to all configured
// router endpoints after a successful admin.key.expire on control.
func (c *admissionSyncClient) PushSetKeyExpiry(
	ctx context.Context,
	svtnID [16]byte,
	pubkey ed25519.PublicKey,
	ttl time.Duration,
) error {
	args := map[string]any{
		"svtn_id":        hex.EncodeToString(svtnID[:]),
		"pubkey_openssh": base64.RawURLEncoding.EncodeToString([]byte(pubkey)),
		"after":          ttl.String(),
	}
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("PushSetKeyExpiry: marshal args: %w", err)
	}
	return c.pushWithRetry(ctx, CmdAdmissionExpire, argsJSON)
}

// PushRemoveSVTN pushes an internal.admission.remove-svtn RPC to all configured
// router endpoints after a successful admin.svtn.destroy on control.
func (c *admissionSyncClient) PushRemoveSVTN(
	ctx context.Context,
	svtnID [16]byte,
) error {
	args := map[string]any{
		"svtn_id": hex.EncodeToString(svtnID[:]),
	}
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("PushRemoveSVTN: marshal args: %w", err)
	}
	return c.pushWithRetry(ctx, CmdAdmissionRemoveSVTN, argsJSON)
}

// pushSnapshotEntries is the inner loop of PushFullSnapshot. It iterates allEntries
// and issues the appropriate admission RPCs via s for each entry. Extracted to allow
// tests to drive the loop with a spy syncer (admissionSyncClient.PushFullSnapshot
// calls its own methods, not the interface, so interception requires this helper).
//
// Per-entry logic (BC-2.05.009 v1.5 PC-7b / Ruling 14):
//
//   - REVOKED entry (Ruling 13 / PC-7c): issue PushRevokeKey ONLY. MUST NOT register.
//
//   - ACTIVE entry: (a) PushRegisterKey; on failure → record lastErr and continue
//     (17f(i): skip expire when router has no entry). (b) If non-zero expiry →
//     PushSetKeyExpiry; on failure → if expiry is in the PAST issue a best-effort
//     compensating PushRevokeKey (17f(ii): prevents active-and-non-expiring entry
//     on router, Invariant 6). For FUTURE-expiry expire-fail → NO compensating
//     revoke (17f(iii): PC-5 permitted staleness; the key is active in control now).
//
// w is the log writer for the advisory WARN emitted when both expire AND compensating
// revoke fail. nil falls back to os.Stderr (F-3 writer pattern).
//
// Push errors are advisory (lastErr records the most recent error; callers log WARN).
func pushSnapshotEntries(ctx context.Context, s admissionSyncer, allEntries map[[16]byte][]admission.AdmittedKey, w io.Writer) error {
	var lastErr error
	for svtnID, entries := range allEntries {
		for _, e := range entries {
			if e.IsRevoked() {
				// Ruling 13 / BC-2.05.009 v1.4 PC-7c: REVOKED entry — revoke only.
				// MUST NOT issue register first. Fresh router treats key-not-found as
				// success (absent = correct non-admissible state).
				if err := s.PushRevokeKey(ctx, svtnID, e.PublicKey, e.Role, true); err != nil {
					lastErr = err
				}
				continue
			}

			// ACTIVE (not revoked) entry: register, then expire if non-zero.

			// (a) Register the key on the router.
			// 17f(i): on register fail, record lastErr and continue — the router has
			// no entry so pushing expire would return E-ADM-013 (wasted dial).
			if err := s.PushRegisterKey(ctx, svtnID, e.PublicKey, e.Role); err != nil {
				lastErr = err
				continue
			}

			// (b) Push expiry if set — including past-expiry entries.
			// (BC-2.05.009 v1.4 PC-7 / Invariant 6 / EC-010: MUST NOT leave past-expiry
			// entries active-and-non-expiring on the router). time.Until returns a
			// negative duration for past expiries; the router expire handler accepts
			// negative durations and marks the entry expired.
			if !e.KeyExpiry().IsZero() {
				ttl := time.Until(e.KeyExpiry())
				// ttl == 0 means expiry is exactly now — treat as expired (negative).
				if ttl == 0 {
					ttl = -1 * time.Millisecond
				}
				if err := s.PushSetKeyExpiry(ctx, svtnID, e.PublicKey, ttl); err != nil {
					lastErr = err
					// 17f(ii): compensating revoke for PAST-expiry entries only.
					// A past-expiry entry whose expire-push failed is now registered
					// as active-and-non-expiring on the router — Invariant 6 violation.
					// Issue a best-effort PushRevokeKey to leave the router non-admissible.
					// 17f(iii): FUTURE-expiry expire-fail → NO compensating revoke
					// (PC-5 permitted staleness; the key is legitimately active in control).
					if e.KeyExpiry().Before(time.Now().UTC()) {
						if rErr := s.PushRevokeKey(ctx, svtnID, e.PublicKey, e.Role, true); rErr != nil {
							// Advisory: compensating revoke also failed. WARN so operators
							// know the router has a transient Invariant-6 violation that
							// will be corrected on the next PushFullSnapshot.
							ww := w
							if ww == nil {
								ww = os.Stderr
							}
							_, _ = fmt.Fprintf(ww,
								"switchboard control: WARN: compensating revoke failed after past-expiry expire-fail: svtn_id=%s err=%v\n",
								hex.EncodeToString(svtnID[:]), rErr)
						}
					}
				}
			}
		}
	}
	return lastErr
}

// PushFullSnapshot iterates all admitted key entries across all SVTNs in ks
// and issues the appropriate internal.admission.* RPCs to each configured router
// endpoint. Delegates to pushSnapshotEntries for the per-entry logic.
//
//   - REVOKED entry (Ruling 13 / BC-2.05.009 v1.4 PC-7c): issue revoke ONLY.
//   - ACTIVE entry: register → (on success) expire if non-zero expiry.
//     Past-expiry expire-fail → compensating revoke (Ruling 14 / 17f).
//     Future-expiry expire-fail → no compensating revoke (PC-5 / 17f(iii)).
//
// Called from runControl on startup, before the management server begins serving
// (BC-2.05.009 v1.4 PC-7, Postcondition 7, Invariant 6 / AC-009 / Decision 10).
//
// An empty keyset is a no-op (return nil). Push errors are advisory — the
// caller (runControl) logs WARN and continues.
func (c *admissionSyncClient) PushFullSnapshot(ctx context.Context, ks *admission.AdmittedKeySet) error {
	allEntries := ks.AllSVTNEntries()
	if len(allEntries) == 0 {
		// Empty keyset — no push attempt (AC-009 EmptyKeyset postcondition).
		return nil
	}
	// Delegate to the shared inner loop, routing Push* calls through c (self).
	// w is nil — compensating-revoke WARN falls back to os.Stderr.
	return pushSnapshotEntries(ctx, c, allEntries, nil)
}
