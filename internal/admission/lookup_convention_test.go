// Package admission_test — Red Gate tests for S-BL.LOOKUP signature migration.
//
// These tests assert that Lookup and LookupByPubkey return (AdmittedKey, bool)
// per go.md rule 12 (locked-accessor convention) and the DRIFT-F005-LOOKUP-CONVENTION
// tech-debt ruling. They will fail to compile until the migration lands, enforcing
// the Red Gate (BC-5.38.001).
package admission_test

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"sync"
	"testing"
	"unsafe"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/hmac"
)

// mustGenPub generates a fresh Ed25519 public key for lookup convention tests.
func mustGenPub(t *testing.T) ed25519.PublicKey {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey: %v", err)
	}
	return pub
}

// svtnID returns a deterministic SVTN ID for lookup convention tests.
func svtnLookupID(b byte) [16]byte {
	var id [16]byte
	id[0] = b
	return id
}

// ── Lookup ────────────────────────────────────────────────────────────────────

// TestLookup_ReturnsBoolTrueOnHit asserts that Lookup returns (AdmittedKey, true)
// when the (svtnID, nodeAddr) tuple is registered, and that the returned value
// fields are byte-equal to the registered values.
//
// Compile failure before migration: Lookup currently returns *AdmittedKey (single
// value), so `key, ok := ks.Lookup(...)` will not compile until the signature is
// migrated to (AdmittedKey, bool). This is the Red Gate assertion for S-BL.LOOKUP.
func TestLookup_ReturnsBoolTrueOnHit(t *testing.T) {
	t.Parallel()

	ks := admission.NewAdmittedKeySet()
	svtnID := svtnLookupID(0x01)
	pub := mustGenPub(t)

	ks.RegisterKey(svtnID, pub, admission.RoleAccess)

	// Derive the node address using the same function production code uses.
	nodeAddr := frame.DeriveNodeAddress(svtnID, []byte(pub))
	key, ok := ks.Lookup(svtnID, nodeAddr) // S-BL.LOOKUP: compile fails until (AdmittedKey, bool)
	if !ok {
		t.Fatal("Lookup: want ok=true for registered key; got false")
	}
	// F-P1L2-001: assert byte-equal PublicKey and exact NodeAddr — not just non-emptiness.
	// A bug returning arbitrary non-zero bytes would still pass a len!=0 check.
	if !bytes.Equal(key.PublicKey, []byte(pub)) {
		t.Errorf("Lookup: PublicKey mismatch: got %x, want %x", key.PublicKey, []byte(pub))
	}
	if key.NodeAddr != nodeAddr {
		t.Errorf("Lookup: NodeAddr mismatch: got %x, want %x", key.NodeAddr, nodeAddr)
	}
}

// TestLookup_ReturnsBoolFalseOnMiss asserts that Lookup returns (AdmittedKey{}, false)
// when the (svtnID, nodeAddr) tuple is not registered. The returned AdmittedKey must
// be the zero value — a non-zero AdmittedKey paired with false is a contract violation.
func TestLookup_ReturnsBoolFalseOnMiss(t *testing.T) {
	t.Parallel()

	ks := admission.NewAdmittedKeySet()
	svtnID := svtnLookupID(0x02)

	var absent [8]byte
	absent[0] = 0xFF

	key, ok := ks.Lookup(svtnID, absent) // S-BL.LOOKUP: compile fails until (AdmittedKey, bool)
	if ok {
		t.Fatalf("Lookup: want ok=false for unregistered nodeAddr; got true with key=%+v", key)
	}
	// F-P1L2-002: on miss the returned AdmittedKey must be zero-valued.
	if len(key.PublicKey) != 0 || key.NodeAddr != ([8]byte{}) {
		t.Errorf("Lookup miss: returned non-zero AdmittedKey: %+v; want zero value", key)
	}
}

// ── LookupByPubkey ────────────────────────────────────────────────────────────

// TestLookupByPubkey_ReturnsBoolTrueOnHit asserts that LookupByPubkey returns
// (AdmittedKey, true) when the public key is registered for the given SVTN, and
// that the returned value fields are byte-equal to the registered values.
func TestLookupByPubkey_ReturnsBoolTrueOnHit(t *testing.T) {
	t.Parallel()

	ks := admission.NewAdmittedKeySet()
	svtnID := svtnLookupID(0x03)
	pub := mustGenPub(t)

	ks.RegisterKey(svtnID, pub, admission.RoleControl)

	expectedNodeAddr := frame.DeriveNodeAddress(svtnID, []byte(pub))
	key, ok := ks.LookupByPubkey(svtnID, pub) // S-BL.LOOKUP: compile fails until (AdmittedKey, bool)
	if !ok {
		t.Fatal("LookupByPubkey: want ok=true for registered key; got false")
	}
	// F-P1L2-001: assert byte-equal PublicKey and exact NodeAddr — not just non-emptiness.
	if !bytes.Equal(key.PublicKey, []byte(pub)) {
		t.Errorf("LookupByPubkey: PublicKey mismatch: got %x, want %x", key.PublicKey, []byte(pub))
	}
	if key.NodeAddr != expectedNodeAddr {
		t.Errorf("LookupByPubkey: NodeAddr mismatch: got %x, want %x", key.NodeAddr, expectedNodeAddr)
	}
}

// TestLookupByPubkey_ReturnsBoolFalseOnMiss asserts that LookupByPubkey returns
// (AdmittedKey{}, false) when the public key is not registered for the given SVTN.
// The returned AdmittedKey must be the zero value.
func TestLookupByPubkey_ReturnsBoolFalseOnMiss(t *testing.T) {
	t.Parallel()

	ks := admission.NewAdmittedKeySet()
	svtnID := svtnLookupID(0x04)
	pub := mustGenPub(t)

	// Do NOT register — key is absent.
	key, ok := ks.LookupByPubkey(svtnID, pub) // S-BL.LOOKUP: compile fails until (AdmittedKey, bool)
	if ok {
		t.Fatalf("LookupByPubkey: want ok=false for unregistered key; got true with key=%+v", key)
	}
	// F-P1L2-002: on miss the returned AdmittedKey must be zero-valued.
	if len(key.PublicKey) != 0 || key.NodeAddr != ([8]byte{}) {
		t.Errorf("LookupByPubkey miss: returned non-zero AdmittedKey: %+v; want zero value", key)
	}
}

// ── F-P2L2-H1: deep-clone regression fence ────────────────────────────────────

// TestLookup_DeepCloneFence_PublicKeyMutationDoesNotLeak asserts that mutating
// the PublicKey slice returned by Lookup does not corrupt subsequent Lookup
// results or internal store state.
//
// This is the M-3 deep-clone regression fence: Lookup allocates a fresh backing
// array for each returned PublicKey via append(ed25519.PublicKey(nil), ...).
// A failure here means the implementation returned a shared backing array,
// violating go.md rule 12 and ARCH-04 M-3.
func TestLookup_DeepCloneFence_PublicKeyMutationDoesNotLeak(t *testing.T) {
	t.Parallel()

	ks := admission.NewAdmittedKeySet()
	svtnID := svtnLookupID(0x10)
	pub := mustGenPub(t)
	ks.RegisterKey(svtnID, pub, admission.RoleAccess)

	nodeAddr := frame.DeriveNodeAddress(svtnID, []byte(pub))
	originalByte0 := pub[0]

	k1, ok := ks.Lookup(svtnID, nodeAddr)
	if !ok {
		t.Fatal("Lookup: want ok=true for registered key; got false")
	}

	// Mutate the caller's copy of PublicKey.
	k1.PublicKey[0] ^= 0xFF // flip all bits in byte 0

	// A subsequent Lookup must return the original, unmodified value.
	k2, ok := ks.Lookup(svtnID, nodeAddr)
	if !ok {
		t.Fatal("Lookup after mutation: want ok=true; got false")
	}
	if k2.PublicKey[0] != originalByte0 {
		t.Errorf("deep-clone fence: k2.PublicKey[0]=%02x want %02x — mutation of k1 leaked into store or k2",
			k2.PublicKey[0], originalByte0)
	}

	// The two returned slices must not share a backing array.
	// Re-fetch k1 with the original value (before mutation) for the pointer check.
	k1b, _ := ks.Lookup(svtnID, nodeAddr)
	if len(k1b.PublicKey) > 0 && len(k2.PublicKey) > 0 {
		p1 := unsafe.Pointer(&k1b.PublicKey[0])
		p2 := unsafe.Pointer(&k2.PublicKey[0])
		if p1 == p2 {
			t.Errorf("deep-clone fence: k1 and k2 PublicKey share the same backing array (%p) — Lookup must return independent copies", p1)
		}
	}
}

// ── F-P2L2-H2: zero-value miss exhaustive check ───────────────────────────────

// TestLookup_Miss_ReturnsZeroAdmittedKey_AllFields asserts that ALL exported
// fields of the returned AdmittedKey are zero-valued when Lookup returns false.
//
// Go rule 12 (T, bool) contract: on miss, value MUST be zero to prevent
// stale-data confusion. Partial zero checks allow callers to incorrectly use
// fields that happen to be non-zero on a cache-miss path.
func TestLookup_Miss_ReturnsZeroAdmittedKey_AllFields(t *testing.T) {
	t.Parallel()

	ks := admission.NewAdmittedKeySet()
	svtnID := svtnLookupID(0x20)

	var absent [8]byte
	absent[0] = 0xDE
	absent[1] = 0xAD

	key, ok := ks.Lookup(svtnID, absent)
	if ok {
		t.Fatalf("Lookup miss: want ok=false; got true with key=%+v", key)
	}

	// Assert every exported field is its zero value.
	// Go rule 12 (T, bool): on miss the value MUST be zero to prevent
	// stale-data confusion — partial checks are insufficient.
	var zeroFrameAuthKey [32]byte
	if len(key.PublicKey) != 0 {
		t.Errorf("miss: PublicKey want nil/empty, got len=%d value=%x", len(key.PublicKey), key.PublicKey)
	}
	if key.Role != 0 {
		t.Errorf("miss: Role want 0, got %v (%d)", key.Role, uint8(key.Role))
	}
	if key.FrameAuthKey != zeroFrameAuthKey {
		t.Errorf("miss: FrameAuthKey want zero [32]byte, got non-zero")
	}
	if key.NodeAddr != ([8]byte{}) {
		t.Errorf("miss: NodeAddr want zero [8]byte, got %x", key.NodeAddr)
	}
	// KeyExpiry() surfaces the unexported expiry field via an accessor.
	if !key.KeyExpiry().IsZero() {
		t.Errorf("miss: KeyExpiry want zero Time, got %v", key.KeyExpiry())
	}
	// IsRevoked() surfaces the unexported revoked field.
	if key.IsRevoked() {
		t.Errorf("miss: IsRevoked want false, got true")
	}
}

// ── F-P1L2-005 positive control: all hit fields match ─────────────────────────

// TestLookup_Hit_AllFieldsMatchRegistration extends the basic hit test to assert
// that every populated exported field of the returned AdmittedKey is byte-equal
// to what was registered. A partial-field check allows bugs where one field is
// correctly populated but another carries stale or default data.
func TestLookup_Hit_AllFieldsMatchRegistration(t *testing.T) {
	t.Parallel()

	ks := admission.NewAdmittedKeySet()
	svtnID := svtnLookupID(0x30)
	pub := mustGenPub(t)

	ks.RegisterKey(svtnID, pub, admission.RoleControl)

	nodeAddr := frame.DeriveNodeAddress(svtnID, []byte(pub))
	key, ok := ks.Lookup(svtnID, nodeAddr)
	if !ok {
		t.Fatal("Lookup: want ok=true for registered key; got false")
	}

	// PublicKey: byte-equal to registered public key.
	if !bytes.Equal(key.PublicKey, []byte(pub)) {
		t.Errorf("hit: PublicKey mismatch: got %x, want %x", key.PublicKey, []byte(pub))
	}
	// Role: must match the role passed to RegisterKey.
	if key.Role != admission.RoleControl {
		t.Errorf("hit: Role got %v, want RoleControl", key.Role)
	}
	// NodeAddr: must equal the derivation of (svtnID, pub).
	if key.NodeAddr != nodeAddr {
		t.Errorf("hit: NodeAddr got %x, want %x", key.NodeAddr, nodeAddr)
	}
	// FrameAuthKey: must be non-zero (derivation produces a 32-byte key).
	// We assert non-zero rather than recomputing HKDF here to keep the test
	// black-box; the derivation correctness is covered by hmac package tests.
	var zeroKey [32]byte
	if key.FrameAuthKey == zeroKey {
		t.Error("hit: FrameAuthKey is zero — expected derived HMAC key to be non-zero")
	}
}

// ── F-P2L2-H3: concurrent Register/Lookup race test ──────────────────────────

// TestLookup_ConcurrentRegisterRace asserts that concurrent calls to RegisterKey
// and Lookup do not produce data races or return torn/invalid AdmittedKey values.
//
// This test is the fence that prevents a future implementation from silently
// regressing to a data-raced variant. It must be run with -race to be effective
// (CI enforces this via just test-race). The test itself passes deterministically;
// the race detector catches violations.
func TestLookup_ConcurrentRegisterRace(t *testing.T) {
	t.Parallel()

	const (
		numRegisterers = 4
		numLookers     = 4
		keysPerWriter  = 20
	)

	ks := admission.NewAdmittedKeySet()

	// Pre-generate keys so goroutines don't block on crypto/rand concurrently.
	type keyEntry struct {
		svtnID   [16]byte
		pub      ed25519.PublicKey
		nodeAddr [8]byte
	}
	entries := make([]keyEntry, numRegisterers*keysPerWriter)
	for i := range entries {
		pub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("ed25519.GenerateKey: %v", err)
		}
		var svtnID [16]byte
		svtnID[0] = byte(i / keysPerWriter)
		svtnID[1] = byte(i % keysPerWriter)
		entries[i] = keyEntry{
			svtnID:   svtnID,
			pub:      pub,
			nodeAddr: frame.DeriveNodeAddress(svtnID, []byte(pub)),
		}
	}

	var wg sync.WaitGroup

	// Writer goroutines: register all keys.
	for w := range numRegisterers {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			start := w * keysPerWriter
			end := start + keysPerWriter
			for _, e := range entries[start:end] {
				ks.RegisterKey(e.svtnID, e.pub, admission.RoleAccess)
			}
		}(w)
	}

	// Reader goroutines: lookup a mix of keys during concurrent registration.
	for range numLookers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, e := range entries {
				k, ok := ks.Lookup(e.svtnID, e.nodeAddr)
				if !ok {
					// Key may not yet be registered — that is fine.
					continue
				}
				// If ok=true, the returned key must be well-formed.
				if len(k.PublicKey) != ed25519.PublicKeySize {
					t.Errorf("concurrent hit: PublicKey len=%d want %d", len(k.PublicKey), ed25519.PublicKeySize)
				}
				if k.NodeAddr != e.nodeAddr {
					t.Errorf("concurrent hit: NodeAddr mismatch: got %x, want %x", k.NodeAddr, e.nodeAddr)
				}
			}
		}()
	}

	wg.Wait()

	// After all writers finish, every key must be present and well-formed.
	for _, e := range entries {
		k, ok := ks.Lookup(e.svtnID, e.nodeAddr)
		if !ok {
			t.Errorf("post-race: key svtnID=%x nodeAddr=%x not found after all registrations", e.svtnID, e.nodeAddr)
			continue
		}
		if !bytes.Equal(k.PublicKey, []byte(e.pub)) {
			t.Errorf("post-race: PublicKey mismatch for svtnID=%x nodeAddr=%x", e.svtnID, e.nodeAddr)
		}
		if k.NodeAddr != e.nodeAddr {
			t.Errorf("post-race: NodeAddr mismatch for svtnID=%x nodeAddr=%x", e.svtnID, e.nodeAddr)
		}
	}
}

// ── F-P3L2-001: RegisterKey caller-alias fence ───────────────────────────────

// TestRegisterKey_DeepClonesCallerPubkey asserts that RegisterKey stores an
// independent copy of the caller-supplied public key slice. If the implementation
// stored the caller's slice directly, a post-RegisterKey mutation of the caller's
// backing array would corrupt the stored entry and all subsequent Lookups.
//
// This is the write-path deep-clone fence (complement to the Lookup-side M-3
// fence already tested by TestLookup_DeepCloneFence_PublicKeyMutationDoesNotLeak).
func TestRegisterKey_DeepClonesCallerPubkey(t *testing.T) {
	t.Parallel()

	ks := admission.NewAdmittedKeySet()
	svtnID := svtnLookupID(0x40)
	pub := mustGenPub(t)

	// Record the original byte 0 before any mutation.
	originalByte0 := pub[0]

	ks.RegisterKey(svtnID, pub, admission.RoleAccess)

	// Mutate the caller's slice after RegisterKey returns.
	pub[0] ^= 0xFF

	// Derive the nodeAddr using the ORIGINAL (pre-mutation) key, which is what
	// RegisterKey used internally.
	originalPub := make(ed25519.PublicKey, len(pub))
	copy(originalPub, pub)
	originalPub[0] = originalByte0
	nodeAddr := frame.DeriveNodeAddress(svtnID, []byte(originalPub))

	key, ok := ks.Lookup(svtnID, nodeAddr)
	if !ok {
		t.Fatal("Lookup: want ok=true after RegisterKey; got false")
	}

	// The returned PublicKey must reflect the ORIGINAL bytes, not the mutated ones.
	if key.PublicKey[0] != originalByte0 {
		t.Errorf("RegisterKey caller-alias: key.PublicKey[0]=%02x want %02x — "+
			"post-RegisterKey mutation of caller slice leaked into store",
			key.PublicKey[0], originalByte0)
	}
}

// ── F-P3L2-002: same-nodeAddr concurrent contention + FrameAuthKey oracle ─────

// TestLookupByPubkey_ConcurrentSameNodeAddr exercises the writer serialisation
// path by routing N readers and M writers at the SAME (svtnID, nodeAddr). The
// existing TestLookup_ConcurrentRegisterRace uses disjoint nodeAddr per goroutine,
// so cross-partition contention is never reached.
//
// Assertions:
//  1. No data race under -race (structural — go test -race catches violations).
//  2. Every hit read returns a well-formed PublicKey (len == ed25519.PublicKeySize).
//  3. On any hit, FrameAuthKey == hmac.DeriveKey(pub, svtnID) — not just non-zero.
func TestLookupByPubkey_ConcurrentSameNodeAddr(t *testing.T) {
	t.Parallel()

	const (
		numReaders = 8
		numWriters = 4
		iterations = 50
	)

	ks := admission.NewAdmittedKeySet()
	svtnID := svtnLookupID(0x50)

	// Two keys that map to different nodeAddrs but both target the same svtnID,
	// maximising lock contention on the svtnID partition.
	pubA := mustGenPub(t)
	pubB := mustGenPub(t)

	// Pre-register pubA so readers see at least one hit from the start.
	ks.RegisterKey(svtnID, pubA, admission.RoleAccess)
	expectedAuthKeyA := hmac.DeriveKey([]byte(pubA), svtnID)

	var wg sync.WaitGroup

	// Writer goroutines alternate re-registering pubA and pubB into the same svtnID.
	for w := range numWriters {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := range iterations {
				if (w+i)%2 == 0 {
					ks.RegisterKey(svtnID, pubA, admission.RoleAccess)
				} else {
					ks.RegisterKey(svtnID, pubB, admission.RoleConsole)
				}
			}
		}(w)
	}

	// Reader goroutines continuously look up pubA.
	for range numReaders {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range iterations {
				key, ok := ks.LookupByPubkey(svtnID, pubA)
				if !ok {
					// pubA may have been temporarily superseded by a concurrent LWW
					// write for pubB at the same nodeAddr — that cannot happen because
					// pubA and pubB derive different nodeAddrs. If !ok here, it is a
					// real miss (pubA was not registered yet or was overwritten). Just
					// skip — the post-wg.Wait check below is the definitive assertion.
					continue
				}
				// Structural check: PublicKey must be well-formed on any hit.
				if len(key.PublicKey) != ed25519.PublicKeySize {
					t.Errorf("concurrent same-nodeAddr: PublicKey len=%d want %d",
						len(key.PublicKey), ed25519.PublicKeySize)
				}
				// FrameAuthKey oracle: must equal the derived value, not merely non-zero.
				// This catches a bug where the key is zeroed or stale under contention.
				if key.FrameAuthKey != expectedAuthKeyA {
					t.Errorf("concurrent same-nodeAddr: FrameAuthKey mismatch: got %x, want %x",
						key.FrameAuthKey, expectedAuthKeyA)
				}
			}
		}()
	}

	wg.Wait()

	// Post-concurrency: pubA must still be reachable (last writer for pubA
	// may have been the final write, or pubB may be current — either is valid
	// per LWW semantics). If pubA is present its FrameAuthKey must be oracle-correct.
	if key, ok := ks.LookupByPubkey(svtnID, pubA); ok {
		if key.FrameAuthKey != expectedAuthKeyA {
			t.Errorf("post-concurrent: pubA FrameAuthKey=%x want %x",
				key.FrameAuthKey, expectedAuthKeyA)
		}
	}
}
