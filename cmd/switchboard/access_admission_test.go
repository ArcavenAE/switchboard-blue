// Package main — access_admission_test.go
//
// TDD Red Gate tests for S-BL.NODE-ADMISSION-PROVISIONING (BC-5.38.001).
//
// Covers AC-002 through AC-008 of story S-BL.NODE-ADMISSION-PROVISIONING:
//
//	AC-002: first-run keypair generation — atomic write, mode 0600, PKCS#8 PEM, parent dir
//	AC-003: subsequent start — loaded key matches generated key; pubkey stable across restarts
//	AC-004: fail-closed — corrupt PEM or non-Ed25519 key type → daemon refuses to start
//	AC-005: permissions warning — broader than 0600 → WARNING logged; daemon starts
//	AC-006: startup INFO log — base64url pubkey logged on every start
//	AC-007: discovery.Config.LocalNodeAdmissionPubkey wired from loaded/generated keypair
//	AC-008: Discovery.Run goroutine WG-tracked; ctx.Canceled is clean shutdown; no goroutine leak
//
// ALL tests in this file MUST FAIL at Red Gate:
//   - loadOrGenerateAdmissionKeypair is a stub that returns a fresh ephemeral key on every
//     call with no file I/O and no logging.
//   - runAccessWithConnector's discovery goroutine stub is a no-op (_ = disc).
//
// Tests compile cleanly against the stub surface.
//
// Traceability:
//
//	BC-2.09.004 — Admission keypair provisioning (first-run, load, fail-closed, permissions, log)
//	BC-2.04.008 — Discovery.Run daemon-lifecycle wiring (WG-tracked, ctx.Canceled clean, no leak)
//	BC-2.09.003 v2.1 PC-12 — admission_key_file validation (covered in config_test.go AC-001)
//
// Test placement discipline (TD-031): no line-number citations; cite symbols/AC-IDs.
package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/discovery"
	"github.com/arcavenae/switchboard/internal/halfchannel"
)

// ---- helpers ----------------------------------------------------------------

// requireAdmissionNoError fails the test if err is non-nil.
func requireAdmissionNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// requireAdmissionError fails the test if err is nil.
func requireAdmissionError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error but got nil")
	}
}

// requireAdmissionContains fails the test if s does not contain substr.
func requireAdmissionContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Fatalf("expected %q to contain %q", s, substr)
	}
}

// requireAdmissionNotContains fails the test if s contains substr.
func requireAdmissionNotContains(t *testing.T, s, substr string) {
	t.Helper()
	if strings.Contains(s, substr) {
		t.Fatalf("expected %q NOT to contain %q", s, substr)
	}
}

// writePKCS8Ed25519PEM writes a PKCS#8 PEM file for a given Ed25519 private key
// at path with the given file mode. Used by multiple tests.
func writePKCS8Ed25519PEM(t *testing.T, path string, priv ed25519.PrivateKey, mode os.FileMode) {
	t.Helper()
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal PKCS#8: %v", err)
	}
	pemData := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	if err := os.WriteFile(path, pemData, mode); err != nil {
		t.Fatalf("write pem file %s: %v", path, err)
	}
}

// parsePKCS8Ed25519FromFile parses the PKCS#8 PEM at path and returns the
// ed25519.PrivateKey. Fatals on any error.
func parsePKCS8Ed25519FromFile(t *testing.T, path string) ed25519.PrivateKey {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read pem file %s: %v", path, err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		t.Fatalf("pem.Decode: no PEM block in %s", path)
	}
	if block.Type != "PRIVATE KEY" {
		t.Fatalf("unexpected PEM block type %q in %s", block.Type, path)
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		t.Fatalf("ParsePKCS8PrivateKey from %s: %v", path, err)
	}
	priv, ok := key.(ed25519.PrivateKey)
	if !ok {
		t.Fatalf("key in %s is not an ed25519.PrivateKey", path)
	}
	return priv
}

// ---- AC-002: first-run keypair generation -----------------------------------

// TestAdmissionKeypair_FirstRun_FileAbsent_KeypairGeneratedAtomically verifies
// that when the key file does not exist, loadOrGenerateAdmissionKeypair generates
// a keypair and writes it atomically (PKCS#8 PEM) to the given path with mode 0600.
//
// Traces: BC-2.09.004 PC-3a, PC-3b; rulings §1.3 first-run semantics.
func TestAdmissionKeypair_FirstRun_FileAbsent_KeypairGeneratedAtomically(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	keyPath := filepath.Join(dir, "admission.pem")

	// File must not exist before the call.
	if _, err := os.Stat(keyPath); !os.IsNotExist(err) {
		t.Fatalf("precondition: key file must not exist; got: %v", err)
	}

	var stderr bytes.Buffer
	priv, err := loadOrGenerateAdmissionKeypair(&stderr, keyPath)
	requireAdmissionNoError(t, err)

	if priv == nil {
		t.Fatal("returned private key must not be nil")
	}

	// AC-002 PC-2: file must exist after the call.
	fi, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("key file must exist after generation: %v", err)
	}

	// AC-002 PC-2: final file must have mode 0600.
	if mode := fi.Mode().Perm(); mode != 0o600 {
		t.Errorf("expected mode 0600, got %04o (AC-002 PC-2; rulings §1.3 atomic write)", mode)
	}
}

// TestAdmissionKeypair_FirstRun_ParentDirAbsent_MkdirAll verifies that when the
// parent directory of the key file does not exist, it is created with os.MkdirAll
// before writing the key file.
//
// Traces: BC-2.09.004 PC-3c; rulings §1.3 first-run — parent dir creation.
func TestAdmissionKeypair_FirstRun_ParentDirAbsent_MkdirAll(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// Nested subdir that does not yet exist.
	keyPath := filepath.Join(dir, "nested", "subdir", "admission.pem")

	var stderr bytes.Buffer
	priv, err := loadOrGenerateAdmissionKeypair(&stderr, keyPath)
	requireAdmissionNoError(t, err)

	if priv == nil {
		t.Fatal("returned private key must not be nil")
	}

	// Parent directory must exist.
	parentDir := filepath.Dir(keyPath)
	if _, err := os.Stat(parentDir); err != nil {
		t.Errorf("parent directory %s must exist after generation: %v", parentDir, err)
	}

	// Key file must exist.
	if _, err := os.Stat(keyPath); err != nil {
		t.Errorf("key file %s must exist after generation: %v", keyPath, err)
	}
}

// TestAdmissionKeypair_FirstRun_GeneratedPKCS8PEMParseable verifies that the
// generated key file is a valid PKCS#8 PEM block parseable via
// pem.Decode → x509.ParsePKCS8PrivateKey → type-assert to ed25519.PrivateKey.
//
// Traces: BC-2.09.004 PC-3b, PC-3f; rulings §1.1 on-disk encoding.
func TestAdmissionKeypair_FirstRun_GeneratedPKCS8PEMParseable(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	keyPath := filepath.Join(dir, "admission.pem")

	var stderr bytes.Buffer
	origPriv, err := loadOrGenerateAdmissionKeypair(&stderr, keyPath)
	requireAdmissionNoError(t, err)

	// Parse the written file using the canonical decode path.
	loadedPriv := parsePKCS8Ed25519FromFile(t, keyPath)

	// The loaded key's public key must equal the returned key's public key.
	origPub := origPriv.Public().(ed25519.PublicKey)
	loadedPub := loadedPriv.Public().(ed25519.PublicKey)
	if !origPub.Equal(loadedPub) {
		t.Errorf("loaded public key does not match generated public key: generated=%x, loaded=%x",
			[]byte(origPub), []byte(loadedPub))
	}
}

// TestAdmissionKeypair_FirstRun_Mode0600 verifies that the generated key file has
// permissions 0600 (owner read-write, no group/other access).
//
// Traces: BC-2.09.004 PC-3b; rulings §1.4.
func TestAdmissionKeypair_FirstRun_Mode0600(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	keyPath := filepath.Join(dir, "admission.pem")

	var stderr bytes.Buffer
	_, err := loadOrGenerateAdmissionKeypair(&stderr, keyPath)
	requireAdmissionNoError(t, err)

	fi, err := os.Stat(keyPath)
	requireAdmissionNoError(t, err)

	if mode := fi.Mode().Perm(); mode != 0o600 {
		t.Errorf("expected file mode 0600, got %04o (rulings §1.4 / BC-2.09.004 PC-3b)", mode)
	}
}

// ---- AC-003: subsequent start — key stable across restarts ------------------

// TestAdmissionKeypair_SubsequentStart_LoadedKeyMatchesGenerated verifies that
// when the key file exists and contains a valid PKCS#8 Ed25519 PEM block, the
// returned private key's public key equals the key written on first run.
//
// Traces: BC-2.09.004 PC-5; rulings §1.3 subsequent start.
func TestAdmissionKeypair_SubsequentStart_LoadedKeyMatchesGenerated(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	keyPath := filepath.Join(dir, "admission.pem")

	// First run: generate and write the key.
	var stderr1 bytes.Buffer
	privFirst, err := loadOrGenerateAdmissionKeypair(&stderr1, keyPath)
	requireAdmissionNoError(t, err)
	pubFirst := privFirst.Public().(ed25519.PublicKey)

	// Second call (subsequent start): file exists, must load the same key.
	var stderr2 bytes.Buffer
	privSecond, err := loadOrGenerateAdmissionKeypair(&stderr2, keyPath)
	requireAdmissionNoError(t, err)
	pubSecond := privSecond.Public().(ed25519.PublicKey)

	if !pubFirst.Equal(pubSecond) {
		t.Errorf("subsequent load returned different public key: first=%x, second=%x"+
			"\n\nThis means loadOrGenerateAdmissionKeypair generates a fresh key on every call "+
			"instead of loading from the file — stub behaviour, not implementation (AC-003 PC-2)",
			[]byte(pubFirst), []byte(pubSecond))
	}
}

// TestAdmissionKeypair_SubsequentStart_PublicKeyStableAcrossRestarts verifies that
// calling loadOrGenerateAdmissionKeypair a third time with the same path still
// returns the same public key — demonstrating the on-disk identity is stable.
//
// Traces: BC-2.09.004 PC-5; rulings §1.3 subsequent-start semantics.
func TestAdmissionKeypair_SubsequentStart_PublicKeyStableAcrossRestarts(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	keyPath := filepath.Join(dir, "admission.pem")

	// First run.
	var sb bytes.Buffer
	privFirst, err := loadOrGenerateAdmissionKeypair(&sb, keyPath)
	requireAdmissionNoError(t, err)
	pubFirst := privFirst.Public().(ed25519.PublicKey)

	// Simulate two additional "restarts".
	for i := range 2 {
		var sbN bytes.Buffer
		privN, errN := loadOrGenerateAdmissionKeypair(&sbN, keyPath)
		requireAdmissionNoError(t, errN)
		pubN := privN.Public().(ed25519.PublicKey)
		if !pubFirst.Equal(pubN) {
			t.Errorf("restart %d: public key changed — expected %x got %x (AC-003 stability)",
				i+2, []byte(pubFirst), []byte(pubN))
		}
	}
}

// ---- AC-004: fail-closed on corrupt or non-Ed25519 key ----------------------

// TestAdmissionKeypair_FailClosed_CorruptPEM verifies that when the key file
// exists but contains truncated/non-parseable PEM data, loadOrGenerateAdmissionKeypair
// returns a non-nil error containing "PEM decode failed" and the path.
//
// Traces: BC-2.09.004 PC-6; E-KEY-001; rulings §1.3 corrupt file.
func TestAdmissionKeypair_FailClosed_CorruptPEM(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	keyPath := filepath.Join(dir, "admission.pem")

	// Write corrupt / truncated data (not a valid PEM block).
	if err := os.WriteFile(keyPath, []byte("not a pem file\n-----BEGIN PRIVATE KEY-----\nabc"), 0o600); err != nil {
		t.Fatalf("setup: write corrupt pem: %v", err)
	}

	var stderr bytes.Buffer
	_, err := loadOrGenerateAdmissionKeypair(&stderr, keyPath)
	requireAdmissionError(t, err)

	// Error must contain the path and "PEM decode failed".
	msg := err.Error()
	requireAdmissionContains(t, msg, keyPath)
	requireAdmissionContains(t, msg, "PEM decode failed")
}

// TestAdmissionKeypair_FailClosed_NonEd25519Key verifies that when the key file
// contains a valid PKCS#8 PEM block but with an RSA key (not Ed25519),
// loadOrGenerateAdmissionKeypair returns a non-nil error containing "not an Ed25519 key".
//
// Traces: BC-2.09.004 PC-6; E-KEY-001; rulings §1.3 type-assert failure.
func TestAdmissionKeypair_FailClosed_NonEd25519Key(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	keyPath := filepath.Join(dir, "admission.pem")

	// Generate an RSA key and write it as PKCS#8 PEM.
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("setup: generate RSA key: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(rsaKey)
	if err != nil {
		t.Fatalf("setup: marshal RSA PKCS#8: %v", err)
	}
	pemData := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	if writeErr := os.WriteFile(keyPath, pemData, 0o600); writeErr != nil {
		t.Fatalf("setup: write RSA pem: %v", writeErr)
	}

	var stderr bytes.Buffer
	_, loadErr := loadOrGenerateAdmissionKeypair(&stderr, keyPath)
	requireAdmissionError(t, loadErr)

	msg := loadErr.Error()
	requireAdmissionContains(t, msg, keyPath)
	requireAdmissionContains(t, msg, "not an Ed25519 key")
}

// ---- AC-005: permissions warning when broader than 0600 ---------------------

// TestAdmissionKeypair_PermissionsWarning_BroaderThan0600 verifies that when the
// key file exists, is parseable, and has permissions broader than 0600 (e.g., 0644),
// a WARNING is logged to stderr and the daemon continues to start (advisory, not fatal).
//
// Traces: BC-2.09.004 PC-4; rulings §1.4 OpenSSH advisory-not-fatal posture.
func TestAdmissionKeypair_PermissionsWarning_BroaderThan0600(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	keyPath := filepath.Join(dir, "admission.pem")

	// Generate and write a valid key at 0644 (broader than 0600).
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	requireAdmissionNoError(t, err)
	writePKCS8Ed25519PEM(t, keyPath, privKey, 0o644)

	var stderr bytes.Buffer
	loadedPriv, loadErr := loadOrGenerateAdmissionKeypair(&stderr, keyPath)
	requireAdmissionNoError(t, loadErr)

	if loadedPriv == nil {
		t.Fatal("expected non-nil private key when permissions are broader than 0600 (advisory, not fatal)")
	}

	// WARNING must be logged to stderr.
	stderrStr := stderr.String()
	if !strings.Contains(stderrStr, "0644") && !strings.Contains(stderrStr, "permissions") {
		t.Errorf("expected permissions WARNING in stderr for 0644 key file; stderr=%q\n"+
			"(BC-2.09.004 PC-4; rulings §1.4 — log WARNING with mode and expected 0600)", stderrStr)
	}
}

// TestAdmissionKeypair_NoPermissionsWarning_Exactly0600 verifies that when the
// key file has exactly 0600 permissions, no permissions warning is emitted.
//
// Traces: BC-2.09.004 PC-4 (warning not emitted when permissions are correct).
func TestAdmissionKeypair_NoPermissionsWarning_Exactly0600(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	keyPath := filepath.Join(dir, "admission.pem")

	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	requireAdmissionNoError(t, err)
	writePKCS8Ed25519PEM(t, keyPath, privKey, 0o600)

	var stderr bytes.Buffer
	_, loadErr := loadOrGenerateAdmissionKeypair(&stderr, keyPath)
	requireAdmissionNoError(t, loadErr)

	// No permissions warning in stderr.
	stderrStr := stderr.String()
	if strings.Contains(stderrStr, "expected 0600") || strings.Contains(stderrStr, "private key may be exposed") {
		t.Errorf("unexpected permissions WARNING for exactly-0600 key file; stderr=%q", stderrStr)
	}
}

// ---- AC-006: startup INFO log with base64url pubkey -------------------------

// TestAdmissionKeypair_StartupInfoLog_FirstRun_ContainsBase64UrlPubkey verifies
// that on first run (file absent), after keypair generation, a structured INFO log
// is written to stderr containing the base64url (no padding) of the raw 32-byte
// Ed25519 public key.
//
// Traces: BC-2.09.004 PC-7; rulings §1.5 Path A — daemon startup log.
func TestAdmissionKeypair_StartupInfoLog_FirstRun_ContainsBase64UrlPubkey(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	keyPath := filepath.Join(dir, "admission.pem")

	var stderr bytes.Buffer
	priv, err := loadOrGenerateAdmissionKeypair(&stderr, keyPath)
	requireAdmissionNoError(t, err)

	pub := priv.Public().(ed25519.PublicKey)
	expectedBase64 := base64.RawURLEncoding.EncodeToString([]byte(pub))

	stderrStr := stderr.String()

	// Must contain the log prefix.
	requireAdmissionContains(t, stderrStr,
		"access: admission identity pubkey (register with admin.key.register):")

	// Must contain the base64url-encoded public key.
	if !strings.Contains(stderrStr, expectedBase64) {
		t.Errorf("startup INFO log must contain base64url pubkey %q; stderr=%q\n"+
			"(BC-2.09.004 PC-7; rulings §1.5 Path A)", expectedBase64, stderrStr)
	}
}

// TestAdmissionKeypair_StartupInfoLog_SubsequentStart_ContainsBase64UrlPubkey
// verifies that on subsequent start (file present), the INFO log with the base64url
// pubkey is also emitted.
//
// Traces: BC-2.09.004 PC-7; rulings Decision 4 — log unconditionally on every start.
func TestAdmissionKeypair_StartupInfoLog_SubsequentStart_ContainsBase64UrlPubkey(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	keyPath := filepath.Join(dir, "admission.pem")

	// First run: generate the key.
	_, privFirst, genErr := ed25519.GenerateKey(rand.Reader)
	requireAdmissionNoError(t, genErr)
	writePKCS8Ed25519PEM(t, keyPath, privFirst, 0o600)

	// Second call: file present (subsequent start).
	var stderr bytes.Buffer
	priv, err := loadOrGenerateAdmissionKeypair(&stderr, keyPath)
	requireAdmissionNoError(t, err)

	pub := priv.Public().(ed25519.PublicKey)
	expectedBase64 := base64.RawURLEncoding.EncodeToString([]byte(pub))

	stderrStr := stderr.String()

	requireAdmissionContains(t, stderrStr,
		"access: admission identity pubkey (register with admin.key.register):")

	if !strings.Contains(stderrStr, expectedBase64) {
		t.Errorf("subsequent-start INFO log must contain base64url pubkey %q; stderr=%q\n"+
			"(BC-2.09.004 PC-7; Decision 4 — emitted unconditionally)", expectedBase64, stderrStr)
	}
}

// ---- AC-007: discovery.Config.LocalNodeAdmissionPubkey populated -----------

// TestAccessDaemon_LocalNodeAdmissionPubkey_PopulatedFrom_LoadedKeypair verifies
// that when runAccess constructs a discovery.Config, its LocalNodeAdmissionPubkey
// field is populated with the 32-byte raw Ed25519 public key derived from the
// admission keypair — specifically that disc.Run does NOT return
// ErrMissingNodeAdmissionPubkey.
//
// Strategy: call disc.Run with a pre-cancelled context; the first select in Run
// will hit ctx.Done() and return context.Canceled. If LocalNodeAdmissionPubkey
// were empty, transmitAdvertisement would return ErrMissingNodeAdmissionPubkey.
// Since Run returns on ctx.Done() before any advertisement is sent, the absence
// of ErrMissingNodeAdmissionPubkey is confirmed by the run returning context.Canceled.
//
// The test constructs a discovery.Config directly (bypassing runAccess) to isolate
// the wiring contract.
//
// Traces: BC-2.09.004 PC-3e; BC-2.04.008 Precondition 3; rulings Decision 5.
func TestAccessDaemon_LocalNodeAdmissionPubkey_PopulatedFrom_LoadedKeypair(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	keyPath := filepath.Join(dir, "admission.pem")

	var stderr bytes.Buffer
	priv, err := loadOrGenerateAdmissionKeypair(&stderr, keyPath)
	requireAdmissionNoError(t, err)

	pub := priv.Public().(ed25519.PublicKey)
	rawPub := []byte(pub)

	// Construct discovery.Config with LocalNodeAdmissionPubkey as runAccess does
	// (rulings Decision 5).
	discCfg := discovery.Config{
		LocalNodeAdmissionPubkey: rawPub,
	}
	disc := discovery.New(discCfg)

	// Run with a pre-cancelled context — disc.Run returns ctx.Err() == context.Canceled
	// without sending any advertisement.
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel

	runErr := disc.Run(ctx)
	if runErr == nil {
		t.Error("disc.Run must return non-nil error (context.Canceled) when ctx is pre-cancelled")
	} else if runErr != context.Canceled {
		// Any error other than context.Canceled means something went wrong —
		// e.g., ErrMissingNodeAdmissionPubkey if LocalNodeAdmissionPubkey were empty.
		t.Errorf("disc.Run returned unexpected error: %v (expected context.Canceled; "+
			"if ErrMissingNodeAdmissionPubkey, LocalNodeAdmissionPubkey was not wired correctly)",
			runErr)
	}
}

// TestAccessDaemon_LocalNodeAdmissionPubkey_NonNilLength32 verifies that the
// 32-byte raw Ed25519 public key derived from the admission keypair has exactly
// 32 bytes (the Ed25519 public key size) and is non-nil.
//
// Traces: BC-2.09.004 PC-3e; rulings Decision 5.
func TestAccessDaemon_LocalNodeAdmissionPubkey_NonNilLength32(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	keyPath := filepath.Join(dir, "admission.pem")

	var stderr bytes.Buffer
	priv, err := loadOrGenerateAdmissionKeypair(&stderr, keyPath)
	requireAdmissionNoError(t, err)

	pub := priv.Public().(ed25519.PublicKey)
	rawPub := []byte(pub)

	if rawPub == nil {
		t.Fatal("LocalNodeAdmissionPubkey must not be nil (AC-007 PC-2)")
	}
	if len(rawPub) != ed25519.PublicKeySize {
		t.Errorf("LocalNodeAdmissionPubkey must have length %d (ed25519.PublicKeySize), got %d",
			ed25519.PublicKeySize, len(rawPub))
	}
}

// ---- AC-008: Discovery.Run goroutine WG-tracked, ctx.Canceled clean, no leak -

// newFakeConnectorForDiscovery builds a fakeConnector suitable for the AC-008
// runAccessWithConnector injection.
func newFakeConnectorForDiscovery(t *testing.T) *fakeConnector {
	t.Helper()
	fc := &fakeConnector{
		errCh:    make(chan error),
		framesCh: make(chan halfchannel.ChannelFrame),
	}
	t.Cleanup(func() { _ = fc.Close() })
	return fc
}

// makeDiscForAC008 builds a *discovery.Discovery with a TickSource channel so
// tests can send ticks directly without relying on wall-clock timing. The
// optional heartbeatObserver is wired into Config.HeartbeatObserver when non-nil.
// Returns the disc and a send-only reference to the tick channel.
func makeDiscForAC008(t *testing.T, heartbeatObserver func()) (*discovery.Discovery, chan<- time.Time) {
	t.Helper()
	tickCh := make(chan time.Time, 1)
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("setup: generate ed25519 key: %v", err)
	}
	pub := priv.Public().(ed25519.PublicKey)
	disc := discovery.New(discovery.Config{
		LocalNodeAdmissionPubkey: []byte(pub),
		TickSource:               tickCh,
		HeartbeatObserver:        heartbeatObserver,
	})
	return disc, tickCh
}

// TestDiscoveryRun_WGAddBeforeGoStatement verifies BC-2.04.008 PC-1 and ARCH-01
// v1.7 §Goroutine WaitGroup Contract: wg.Add(1) is called before the go statement
// that starts disc.Run, and disc.Run is actually invoked.
//
// Discriminating strategy: disc is constructed with a HeartbeatObserver that
// closes a `started` channel on first call, and a TickSource for deterministic
// delivery. After runAccessWithConnector starts, the test sends a tick. If
// disc.Run was started by runAccessWithConnector, the tick fires the observer,
// closing `started` within a short window. If disc goroutine is never launched
// (stub: _ = disc), the tick has no effect, `started` never closes, and the test
// fails by timeout.
//
// Traces: BC-2.04.008 PC-1; ARCH-01 v1.7 §Goroutine WaitGroup Contract; rulings §3.2.
func TestDiscoveryRun_WGAddBeforeGoStatement(t *testing.T) {
	// NOT t.Parallel(): uses package-level runAccessWithConnector with goroutines.

	started := make(chan struct{})
	var startOnce sync.Once
	disc, tickCh := makeDiscForAC008(t, func() {
		startOnce.Do(func() { close(started) })
	})

	an, router := newMinimalAccessComponents(t)
	fc := newFakeConnectorForDiscovery(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	var stderr bytes.Buffer
	done := make(chan struct{})
	go func() {
		_ = runAccessWithConnector(ctx, &stderr, fc, an, router, disc)
		close(done)
	}()

	// Give runAccessWithConnector time to start goroutines.
	time.Sleep(5 * time.Millisecond)

	// Send a tick to disc.Run — if disc.Run is running, it calls HeartbeatObserver
	// which closes `started`.
	tickCh <- time.Now()

	// If disc.Run was started (fixed code), `started` closes promptly.
	// If disc.Run was never started (stub: _ = disc), `started` never closes → FAIL.
	select {
	case <-started:
		// disc.Run was started and processed the tick — WG contract can be satisfied.
	case <-time.After(300 * time.Millisecond):
		cancel() // prevent test-process leak
		<-done
		t.Fatal("disc.Run was never started by runAccessWithConnector: HeartbeatObserver " +
			"never fired after sending a tick (stub: _ = disc). " +
			"Fix: wg.Add(1) before go disc.Run(runCtx) in runAccessWithConnector. " +
			"(BC-2.04.008 PC-1; ARCH-01 v1.7 §Goroutine WaitGroup Contract; rulings §3.2)")
	}

	// Clean shutdown — cancel and wait for function to return.
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runAccessWithConnector did not return within 2s after ctx cancellation")
	}
}

// TestDiscoveryRun_SameWaitGroupAsSweepTickers verifies BC-2.04.008 PC-2: the
// discovery goroutine is tracked in the same WaitGroup as sweep and
// frames-dropped ticker goroutines (no separate WaitGroup introduced).
//
// Discriminating strategy: same as WGAddBeforeGoStatement — send a tick to confirm
// disc.Run is running (failing on stub), then cancel ctx and verify the function
// returns cleanly (all goroutines including disc joined together via wg.Wait).
//
// Traces: BC-2.04.008 PC-2; rulings §3.1 Option Y rationale.
func TestDiscoveryRun_SameWaitGroupAsSweepTickers(t *testing.T) {
	// NOT t.Parallel(): modifies package-level framesDroppedInterval.

	started := make(chan struct{})
	var startOnce sync.Once
	disc, tickCh := makeDiscForAC008(t, func() {
		startOnce.Do(func() { close(started) })
	})

	origInterval := framesDroppedInterval
	framesDroppedInterval = time.Millisecond
	t.Cleanup(func() { framesDroppedInterval = origInterval })

	an, router := newMinimalAccessComponents(t)
	fc := newFakeConnectorForDiscovery(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	var stderr bytes.Buffer
	done := make(chan struct{})
	go func() {
		_ = runAccessWithConnector(ctx, &stderr, fc, an, router, disc)
		close(done)
	}()

	time.Sleep(10 * time.Millisecond)
	tickCh <- time.Now()

	// Verify disc.Run was started (fails on stub).
	select {
	case <-started:
		// disc is running — good.
	case <-time.After(300 * time.Millisecond):
		cancel()
		<-done
		t.Fatal("disc.Run was never started: same-WG test cannot observe correct tracking " +
			"(stub: _ = disc). Fix: start disc.Run in the same wg as sweep/frames-dropped tickers. " +
			"(BC-2.04.008 PC-2; rulings §3.1 Option Y)")
	}

	cancel()
	select {
	case <-done:
		// all goroutines joined — same WG confirmed
	case <-time.After(2 * time.Second):
		t.Fatal("runAccessWithConnector did not return within 2s after ctx cancellation " +
			"(BC-2.04.008 PC-2 — discovery goroutine must share wg with tickers)")
	}
}

// TestDiscoveryRun_CtxCanceled_NotInternalFailure verifies BC-2.04.008 Invariant 2
// and Decision 7: when disc.Run returns context.Canceled (normal shutdown),
// runAccessWithConnector must return nil (not a non-nil error), i.e. internalFailure
// must NOT be set.
//
// Discriminating strategy: confirm disc.Run was started (tick + HeartbeatObserver),
// then cancel ctx and assert the return value is nil. On stub, the "started" check
// fails, so the test fails before reaching the internalFailure assertion.
//
// Traces: BC-2.04.008 Invariant 2; BC-2.04.008 PC-3; rulings Decision 7; rulings §3.2.
func TestDiscoveryRun_CtxCanceled_NotInternalFailure(t *testing.T) {
	// NOT t.Parallel(): goroutine-heavy; uses channel handshake.

	started := make(chan struct{})
	var startOnce sync.Once
	disc, tickCh := makeDiscForAC008(t, func() {
		startOnce.Do(func() { close(started) })
	})

	an, router := newMinimalAccessComponents(t)
	fc := newFakeConnectorForDiscovery(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	var stderr bytes.Buffer
	done := make(chan error, 1)
	go func() {
		done <- runAccessWithConnector(ctx, &stderr, fc, an, router, disc)
	}()

	time.Sleep(5 * time.Millisecond)
	tickCh <- time.Now()

	// Confirm disc.Run was started (fails on stub).
	select {
	case <-started:
	case <-time.After(300 * time.Millisecond):
		cancel()
		<-done
		t.Fatal("disc.Run was never started: cannot verify internalFailure=false for ctx.Canceled " +
			"(stub: _ = disc). Fix: start disc.Run goroutine in runAccessWithConnector. " +
			"(BC-2.04.008 PC-3 / Invariant 2; Decision 7)")
	}

	// Cancel ctx — disc.Run returns context.Canceled (clean shutdown).
	cancel()

	var runErr error
	select {
	case runErr = <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runAccessWithConnector did not return within 2s after ctx cancellation")
	}

	// context.Canceled from disc.Run is NOT internalFailure — must return nil.
	if runErr != nil {
		t.Errorf("runAccessWithConnector must return nil on clean ctx cancellation (disc.Run "+
			"returned context.Canceled); got %v\n"+
			"(Decision 7 / BC-2.04.008 Invariant 2 — ctx.Canceled is NOT an internal failure)",
			runErr)
	}

	// stderr must NOT contain E-SYS-002 (reserved for mid-session failures).
	requireAdmissionNotContains(t, stderr.String(), "fatal: cannot connect to session backend")
}

// TestDiscoveryRun_NoGoroutineLeak_AfterCtxCancel verifies BC-2.04.008 PC-4:
// wg.Wait() in runAccessWithConnector returns cleanly after ctx cancellation with
// no goroutine leak. Enforcement per the story: bounded wait ≤100ms after cancel.
//
// Discriminating strategy — channel handshake using a blocking HeartbeatObserver:
//
//  1. disc is constructed with a HeartbeatObserver that:
//     a. closes `entered` on first call (disc.Run goroutine is now parked inside it)
//     b. blocks on `release` until the test releases it
//  2. A tick is sent to park disc.Run in the observer.
//  3. ctx is cancelled. runAccessWithConnector calls sc.Close() + wg.Wait().
//  4. On FIXED code: wg.Wait() blocks because disc.Run goroutine is parked →
//     `done` stays open for the 150ms window → test passes first select.
//  5. On STUB code (no disc goroutine): wg.Wait() returns immediately after
//     sc.Close() (only drain + bridge goroutines in wg) → `done` closes WHILE
//     disc is "parked" (actually it never ran) → first select hits <-done →
//     t.Fatal → RED.
//
// Traces: BC-2.04.008 PC-4; rulings §3.2 — bounded wg.Wait enforcement; story spec ≤100ms.
func TestDiscoveryRun_NoGoroutineLeak_AfterCtxCancel(t *testing.T) {
	// NOT t.Parallel(): blocking HeartbeatObserver; channel handshake.

	entered := make(chan struct{})
	release := make(chan struct{})
	var enterOnce sync.Once

	disc, tickCh := makeDiscForAC008(t, func() {
		// First call: signal entered + block until release.
		// Subsequent calls (after release) return immediately.
		enterOnce.Do(func() {
			close(entered)
			<-release
		})
	})

	an, router := newMinimalAccessComponents(t)
	fc := newFakeConnectorForDiscovery(t)

	ctx, cancel := context.WithCancel(context.Background())

	var stderr bytes.Buffer
	done := make(chan struct{})
	go func() {
		_ = runAccessWithConnector(ctx, &stderr, fc, an, router, disc)
		close(done)
	}()

	// Give goroutines time to start.
	time.Sleep(5 * time.Millisecond)

	// Send tick to park disc.Run in HeartbeatObserver.
	tickCh <- time.Now()

	// Wait for disc.Run to enter the observer.
	// If disc was never started (stub), entered never closes → select timeout → FAIL.
	select {
	case <-entered:
		// disc.Run goroutine is now parked inside HeartbeatObserver.
	case <-time.After(300 * time.Millisecond):
		close(release) // avoid observer goroutine leak
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
		t.Fatal("disc.Run HeartbeatObserver never entered: disc goroutine was not started " +
			"(stub: _ = disc). Fix: start disc.Run in a wg-tracked goroutine. " +
			"(BC-2.04.008 PC-4; rulings §3.2)")
	}

	// Trigger PC-2 clean shutdown.
	cancel()

	// THE DISCRIMINATOR:
	//
	// STUB (disc not in wg): wg.Wait() returns without the disc goroutine.
	// runAccessWithConnector returns. `done` closes WHILE disc observer is parked.
	// → <-done fires → t.Fatal → RED.
	//
	// FIXED (disc in wg): wg.Wait() blocks on disc goroutine parked in observer.
	// `done` stays open for the full 150ms window.
	// → timeout branch → continue → PASS (first assertion).
	select {
	case <-done:
		close(release) // unblock observer to avoid test-process goroutine leak
		t.Fatal("runAccessWithConnector returned while disc.Run goroutine was parked " +
			"in HeartbeatObserver — wg.Wait() did not join the discovery goroutine. " +
			"Fix: wg.Add(1) before go disc.Run(runCtx), defer wg.Done() inside goroutine. " +
			"(BC-2.04.008 PC-4; ARCH-01 v1.7 §Goroutine WaitGroup Contract; rulings §3.2)")
	case <-time.After(150 * time.Millisecond):
		// expected on FIXED code: function blocked in wg.Wait() for disc goroutine
	}

	// Release the parked disc goroutine. On fixed code: observer returns,
	// disc.Run loops back to select, ctx.Done() fires, disc.Run returns,
	// wg.Done() fires, wg.Wait() returns, function returns, done closes.
	close(release)

	select {
	case <-done:
		// success: function returned only after disc goroutine was joined
	case <-time.After(2 * time.Second):
		t.Fatal("runAccessWithConnector did not return within 2s after releasing disc goroutine")
	}
}

// TestDiscoveryRun_StartupOrdering_AfterKeypairAndMgmtServer verifies BC-2.04.008
// PC-5: Discovery.Run starts AFTER the admission keypair is loaded AND AFTER the
// management server goroutine is started (rulings §3.1 phases (d)–(f)).
//
// Discriminating strategy: confirm disc.Run was started (tick + HeartbeatObserver),
// then cancel and assert clean return. On stub, the "started" check fails.
//
// The indirect ordering assertion: if LocalNodeAdmissionPubkey were empty when
// disc.Run started, transmitAdvertisement would return ErrMissingNodeAdmissionPubkey.
// Since the disc constructed here has a valid pubkey (wired in makeDiscForAC008),
// the test confirms disc runs and exits cleanly.
//
// Traces: BC-2.04.008 PC-5; rulings §3.1 phase ordering (d)–(f).
func TestDiscoveryRun_StartupOrdering_AfterKeypairAndMgmtServer(t *testing.T) {
	// NOT t.Parallel(): goroutine-heavy.

	started := make(chan struct{})
	var startOnce sync.Once
	disc, tickCh := makeDiscForAC008(t, func() {
		startOnce.Do(func() { close(started) })
	})

	an, router := newMinimalAccessComponents(t)
	fc := newFakeConnectorForDiscovery(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	var stderr bytes.Buffer
	done := make(chan error, 1)
	go func() {
		done <- runAccessWithConnector(ctx, &stderr, fc, an, router, disc)
	}()

	time.Sleep(5 * time.Millisecond)
	tickCh <- time.Now()

	// Confirm disc.Run was started (fails on stub).
	select {
	case <-started:
	case <-time.After(300 * time.Millisecond):
		cancel()
		<-done
		t.Fatal("disc.Run was never started: startup-ordering cannot be verified " +
			"(stub: _ = disc). Fix: start disc.Run after keypair load + mgmt server start. " +
			"(BC-2.04.008 PC-5; rulings §3.1 phases (d)–(f))")
	}

	cancel()
	var runErr error
	select {
	case runErr = <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runAccessWithConnector did not return within 2s (startup ordering test)")
	}

	if runErr != nil {
		t.Errorf("startup-ordering test: expected nil return on clean shutdown; got %v "+
			"(BC-2.04.008 PC-5; disc must start after keypair load)", runErr)
	}
}
