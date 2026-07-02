// admin_handlers_pubkey_test.go — RED tests for Phase 5 Pass 4 remediation.
//
// Covers F-A-010 (HIGH): decodePublicKey only accepts base64; must also accept
// OpenSSH authorized_keys format ("ssh-ed25519 AAAA... comment").
//
// These tests MUST FAIL until decodePublicKey is updated to parse OpenSSH
// format via golang.org/x/crypto/ssh.ParseAuthorizedKey.
//
// IMPORTANT: DO NOT touch implementation code. This file is tests only.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

// generateOpenSSHPubkey creates a real ed25519 keypair and returns the public
// key in OpenSSH authorized_keys format: "ssh-ed25519 <base64> <comment>".
func generateOpenSSHPubkey(t *testing.T, comment string) (ed25519.PublicKey, string) {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate ed25519 key: %v", err)
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatalf("convert to ssh.PublicKey: %v", err)
	}
	encoded := base64.StdEncoding.EncodeToString(sshPub.Marshal())
	openssh := "ssh-ed25519 " + encoded
	if comment != "" {
		openssh += " " + comment
	}
	return pub, openssh
}

// TestNewInBurst19_DecodePublicKey_AcceptsOpenSSH verifies that decodePublicKey
// accepts the OpenSSH authorized_keys format used by real ssh key files.
//
// MUST FAIL with current base64-only decodePublicKey implementation.
func TestNewInBurst19_DecodePublicKey_AcceptsOpenSSH(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		comment string
	}{
		{name: "no_comment", comment: ""},
		{name: "with_comment", comment: "operator@example.com"},
		{name: "multi_word_comment", comment: "alice bob charlie"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, openssh := generateOpenSSHPubkey(t, tc.comment)

			// Sanity check: the openssh string has the expected prefix.
			if !strings.HasPrefix(openssh, "ssh-ed25519 ") {
				t.Fatalf("generateOpenSSHPubkey produced unexpected format: %q", openssh)
			}

			// This call FAILS with current code because decodePublicKey only accepts
			// raw base64, not the "ssh-ed25519 <b64> <comment>" format.
			got, err := decodePublicKey(openssh)
			if err != nil {
				t.Fatalf("decodePublicKey(OpenSSH format) returned error: %v\n  input: %q\n  (decodePublicKey must accept OpenSSH format; currently only accepts raw base64)", err, openssh)
			}
			if len(got) != ed25519.PublicKeySize {
				t.Errorf("decodePublicKey returned key of length %d; want %d", len(got), ed25519.PublicKeySize)
			}
		})
	}
}

// TestNewInBurst19_DecodePublicKey_OpenSSH_ReturnsCorrectBytes verifies that
// the ed25519.PublicKey returned by decodePublicKey for an OpenSSH-format input
// equals the original public key bytes.
//
// MUST FAIL with current code.
func TestNewInBurst19_DecodePublicKey_OpenSSH_ReturnsCorrectBytes(t *testing.T) {
	t.Parallel()

	origPub, openssh := generateOpenSSHPubkey(t, "test-comment")

	got, err := decodePublicKey(openssh)
	if err != nil {
		t.Fatalf("decodePublicKey(OpenSSH format) returned error: %v\n  (must accept OpenSSH format)", err)
	}

	// The returned key bytes must match the original ed25519 public key.
	if !origPub.Equal(got) {
		t.Errorf("decoded key does not match original:\n  want: %x\n  got:  %x", []byte(origPub), []byte(got))
	}
}

// TestNewInBurst19_DecodePublicKey_OpenSSH_WrongKeyType verifies that a
// non-ed25519 OpenSSH key (e.g. rsa or ecdsa prefix) is rejected.
//
// This is a negative test to ensure that OpenSSH parsing does NOT silently
// accept unsupported key types. SHOULD FAIL until the parser is in place
// (because currently ANY non-base64 string returns an error rather than a
// type-specific error, so this may pass vacuously — but once the OpenSSH
// parser is added, the type check must be tested explicitly).
//
// MUST FAIL with current code because the error message won't say "must be
// ed25519" — it will say "not valid base64".
func TestNewInBurst19_DecodePublicKey_OpenSSH_WrongKeyType_RejectsWithTypeError(t *testing.T) {
	t.Parallel()

	// Construct a fake "ssh-rsa" prefixed string to simulate wrong key type.
	// We don't need a real RSA key — any non-ed25519 type prefix is sufficient.
	fakeRSA := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC3 test@host"

	_, err := decodePublicKey(fakeRSA)
	if err == nil {
		t.Fatal("decodePublicKey(ssh-rsa key): expected error for wrong key type, got nil")
	}
	// The error must not say "not valid base64" once OpenSSH parsing is in place.
	// Instead it should say something about key type or ed25519.
	// Until then (current code), this test fails because the code DOES say "not valid base64".
	if strings.Contains(err.Error(), "not valid base64") {
		t.Errorf("decodePublicKey(ssh-rsa key): error message says 'not valid base64' — "+
			"once OpenSSH parsing is added, wrong-type keys must return a type-specific error; got: %v", err)
	}
}

// TestNewInBurst19_DecodePublicKey_RawBase64_StillAccepted verifies that raw
// base64 encoding (the pre-existing accepted form) continues to work after
// the OpenSSH parser is added. This is a GREEN guard test to ensure backward
// compatibility — it should PASS before and after the fix.
//
// This test is GREEN (it passes with current code). It is included here to
// document the non-regression requirement.
func TestNewInBurst19_DecodePublicKey_RawBase64_StillAccepted(t *testing.T) {
	t.Parallel()

	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	encoded := base64.StdEncoding.EncodeToString(pub)

	got, err := decodePublicKey(encoded)
	if err != nil {
		t.Fatalf("decodePublicKey(base64) returned error: %v", err)
	}
	if !pub.Equal(got) {
		t.Errorf("base64 decoded key does not match original")
	}
}

// TestNewInBurst19_DecodePublicKey_Empty_ReturnsECFG001 verifies that an
// empty string still returns E-CFG-001. GREEN guard — passes with current code.
func TestNewInBurst19_DecodePublicKey_Empty_ReturnsECFG001(t *testing.T) {
	t.Parallel()

	_, err := decodePublicKey("")
	if err == nil {
		t.Fatal("decodePublicKey(empty): expected E-CFG-001 error, got nil")
	}
	if !strings.Contains(err.Error(), "E-CFG-001") {
		t.Errorf("decodePublicKey(empty): expected E-CFG-001 in error; got: %v", err)
	}
}
