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
	"reflect"
	"sync"
	"sync/atomic"
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
//
// F-P3L2-003: uses reflection over exported fields to future-proof against
// new field additions — a hardcoded list would silently miss new fields.
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

	// Reflection pass: assert every exported field is its type's zero value.
	// This future-proofs the check — adding a new exported field to AdmittedKey
	// without zeroing it on miss will cause this test to fail automatically.
	//
	// reflect.Value.Equal panics on non-comparable kinds (e.g. slice), so we
	// use reflect.DeepEqual(fv.Interface(), zero.Interface()) which handles all
	// kinds safely.
	rv := reflect.ValueOf(key)
	rt := rv.Type()
	for i := range rv.NumField() {
		f := rt.Field(i)
		if !f.IsExported() {
			continue
		}
		fv := rv.Field(i)
		zero := reflect.Zero(f.Type)
		if !reflect.DeepEqual(fv.Interface(), zero.Interface()) {
			t.Errorf("miss: exported field %s want zero value %v, got %v", f.Name, zero, fv)
		}
	}

	// Accessor checks for unexported fields surfaced via methods.
	// KeyExpiry() surfaces the unexported expiry field.
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
	// FrameAuthKey: must be byte-equal to the canonical derivation
	// hmac.DeriveKey(pub, svtnID). Asserting non-zero (the previous check) is
	// insufficient — a non-zero but wrong key would pass. F-P3L2-004.
	expectedAuthKey := hmac.DeriveKey([]byte(pub), svtnID)
	if key.FrameAuthKey != expectedAuthKey {
		t.Errorf("hit: FrameAuthKey got %x, want derived %x", key.FrameAuthKey, expectedAuthKey)
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
	// hits counts reader iterations that observed ok=true. After wg.Wait() we
	// assert hits > 0: if every reader iteration misses, the test is vacuously
	// passing and cannot detect a broken RLock or an always-miss regression
	// (F-P5L2-02). The continue-on-!ok per-iteration behaviour is preserved so
	// a single miss does not mask race-detector output.
	var hits int64

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
				atomic.AddInt64(&hits, 1)
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
	if atomic.LoadInt64(&hits) == 0 {
		t.Errorf("no reader observed a hit during concurrent registration; readers may be silently missing")
	}

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

	// Snapshot the original public key bytes BEFORE any mutation so that the
	// oracle does not depend on which byte was flipped (F-P5L2-01).
	originalPub := append(ed25519.PublicKey(nil), pub...)

	ks.RegisterKey(svtnID, pub, admission.RoleAccess)

	// Mutate the caller's slice after RegisterKey returns.
	pub[0] ^= 0xFF

	// Derive the nodeAddr from the pre-mutation snapshot — matching what
	// RegisterKey used internally.
	nodeAddr := frame.DeriveNodeAddress(svtnID, []byte(originalPub))

	key, ok := ks.Lookup(svtnID, nodeAddr)
	if !ok {
		t.Fatal("Lookup: want ok=true after RegisterKey; got false")
	}

	// The returned PublicKey must be byte-equal to the original snapshot, not the
	// mutated slice. Comparing the full key (not just byte 0) ensures the oracle
	// is not silently broken by mutations at other offsets (F-P5L2-01).
	if !bytes.Equal(key.PublicKey, []byte(originalPub)) {
		t.Errorf("RegisterKey caller-alias: key.PublicKey=%x want %x — "+
			"post-RegisterKey mutation of caller slice leaked into store",
			key.PublicKey, []byte(originalPub))
	}
}

// ── F-P3L2-002: same-entry concurrent registration contention + FrameAuthKey oracle ──

// TestLookupByPubkey_ConcurrentSameEntryRegistration exercises the writer
// serialisation path by routing N readers and M writers all operating on the
// SAME (svtnID, pubkey) pair. The existing TestLookup_ConcurrentRegisterRace
// uses disjoint nodeAddr per goroutine, so single-entry lock contention is
// never reached by that test.
//
// Design note: in this admission model, nodeAddr is deterministically derived
// from (svtnID, pubkey) via frame.DeriveNodeAddress. Two distinct public keys
// always map to distinct nodeAddrs — there is no code path where two different
// pubkeys share a nodeAddr ("same-nodeAddr LWW"). The meaningful concurrent
// contention to test is therefore N goroutines concurrently re-registering the
// SAME (pubkey, nodeAddr) pair, exercising the write-lock critical section and
// the idempotent LWW overwrite path under the race detector.
//
// Assertions:
//  1. No data race under -race (structural — go test -race catches violations).
//  2. Every hit read returns a well-formed PublicKey (len == ed25519.PublicKeySize).
//  3. On any hit, FrameAuthKey == hmac.DeriveKey(pub, svtnID) — not just non-zero.
//  4. Post-concurrency: the entry is always present and oracle-correct.
func TestLookupByPubkey_ConcurrentSameEntryRegistration(t *testing.T) {
	t.Parallel()

	const (
		numReaders = 8
		numWriters = 4
		iterations = 50
	)

	ks := admission.NewAdmittedKeySet()
	svtnID := svtnLookupID(0x50)

	// Single shared key — all writers register the SAME (pubkey, nodeAddr) pair,
	// exercising write-lock contention on a single map slot.
	pub := mustGenPub(t)
	expectedAuthKey := hmac.DeriveKey([]byte(pub), svtnID)

	// Pre-register so readers see at least one hit from the start.
	ks.RegisterKey(svtnID, pub, admission.RoleAccess)

	var wg sync.WaitGroup
	// hits counts reader iterations that observed ok=true. After wg.Wait() we
	// assert hits > 0 to prevent a vacuous-pass: if every reader iteration
	// returns !ok the test passes silently even when LookupByPubkey is broken
	// (F-P5L2-03).
	var hits int64

	// Writer goroutines all re-register the same (pubkey, nodeAddr) pair.
	// Each call is an idempotent LWW overwrite; the entry must remain
	// consistent throughout.
	for range numWriters {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range iterations {
				ks.RegisterKey(svtnID, pub, admission.RoleAccess)
			}
		}()
	}

	// Reader goroutines continuously look up the same pubkey under concurrent writes.
	// The entry is pre-registered above so !ok is always a regression, not a
	// timing window. t.Errorf (non-fatal) is used instead of t.Fatal so the loop
	// keeps running and the race detector continues to observe concurrent accesses.
	for range numReaders {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range iterations {
				key, ok := ks.LookupByPubkey(svtnID, pub)
				if !ok {
					// Entry was pre-registered; every iteration must find it.
					t.Errorf("concurrent same-entry: LookupByPubkey returned ok=false for pre-registered key")
					continue
				}
				atomic.AddInt64(&hits, 1)
				// Structural check: PublicKey must be well-formed on any hit.
				if len(key.PublicKey) != ed25519.PublicKeySize {
					t.Errorf("concurrent same-entry: PublicKey len=%d want %d",
						len(key.PublicKey), ed25519.PublicKeySize)
				}
				// FrameAuthKey oracle: must equal the derived value, not merely non-zero.
				// This catches a bug where the key is zeroed or stale under contention.
				if key.FrameAuthKey != expectedAuthKey {
					t.Errorf("concurrent same-entry: FrameAuthKey mismatch: got %x, want %x",
						key.FrameAuthKey, expectedAuthKey)
				}
			}
		}()
	}

	wg.Wait()
	if atomic.LoadInt64(&hits) == 0 {
		t.Errorf("no reader observed a hit during concurrent same-entry registration; readers may be silently missing")
	}

	// Post-concurrency: entry must be present and oracle-correct after all
	// concurrent registrations complete.
	key, ok := ks.LookupByPubkey(svtnID, pub)
	if !ok {
		t.Fatal("post-concurrent: entry not found after all concurrent registrations")
	}
	if key.FrameAuthKey != expectedAuthKey {
		t.Errorf("post-concurrent: FrameAuthKey=%x want %x", key.FrameAuthKey, expectedAuthKey)
	}
	if !bytes.Equal(key.PublicKey, []byte(pub)) {
		t.Errorf("post-concurrent: PublicKey mismatch: got %x, want %x", key.PublicKey, []byte(pub))
	}
}

// ── F-P3L2-005: exhaustive miss table ─────────────────────────────────────────

// assertMissZero is a test helper that verifies LookupByPubkey returned a miss
// and that the returned AdmittedKey is entirely zero-valued.
//
// Uses reflect.DeepEqual(key, admission.AdmittedKey{}) rather than a
// field-by-field loop filtered by IsExported(). The DeepEqual form covers ALL
// fields — exported and unexported alike — so a broken LookupByPubkey that
// returns a non-zero unexported field (e.g. revoked=true or expiry set) on a
// miss will be caught without any changes to this helper. F-L2-02.
func assertMissZero(t *testing.T, label string, key admission.AdmittedKey, ok bool) {
	t.Helper()
	if ok {
		t.Errorf("%s: want ok=false (miss), got ok=true with key=%+v", label, key)
		return
	}
	if !reflect.DeepEqual(key, admission.AdmittedKey{}) {
		t.Errorf("%s: want zero AdmittedKey on miss; got non-zero value (exported or unexported fields populated)", label)
	}
}

// TestLookupByPubkey_ExhaustiveMissTable exercises every structurally distinct
// miss scenario for LookupByPubkey. Each case hits a different code path:
//
//	(a) svtnID exists in store, but pubkey is not registered → inner-map miss
//	(b) svtnID does not exist in store at all → outer-map miss
//	    Note: "known pubkey in wrong svtnID" and "known pubkey in never-registered
//	    svtnID" both reach the same outer-map miss branch. They are merged here
//	    because the code path is identical — a distinct case would add no coverage
//	    value. F-L2-03.
//	(c) revoked pubkey — LookupByPubkey returns the entry with IsRevoked()==true
//	    (revocation is enforced at admission time, not at lookup time).
//
// Each true-miss row asserts (AdmittedKey{}, false) with all-zero fields via
// reflect.DeepEqual (F-L2-02), future-proofing against new field additions.
// F-P3L2-005.
func TestLookupByPubkey_ExhaustiveMissTable(t *testing.T) {
	t.Parallel()

	ks := admission.NewAdmittedKeySet()
	svtnRegistered := svtnLookupID(0x60)

	pub := mustGenPub(t)
	pubUnknown := mustGenPub(t)
	pubRevoked := mustGenPub(t)

	ks.RegisterKey(svtnRegistered, pub, admission.RoleAccess)
	ks.RegisterKey(svtnRegistered, pubRevoked, admission.RoleAccess)
	nodeAddrRevoked := frame.DeriveNodeAddress(svtnRegistered, []byte(pubRevoked))
	if err := ks.RevokeKey(svtnRegistered, nodeAddrRevoked); err != nil {
		t.Fatalf("RevokeKey: %v", err)
	}

	tests := []struct {
		name    string
		svtnID  [16]byte
		pub     ed25519.PublicKey
		wantHit bool // true for case (c) which is a hit with IsRevoked=true
	}{
		{
			// (a) inner-map miss: svtnID is registered but pubkey is not
			name:   "(a) unknown pubkey in registered svtnID",
			svtnID: svtnRegistered,
			pub:    pubUnknown,
		},
		{
			// (b) outer-map miss: svtnID has never been registered in the store.
			// "Known pubkey in wrong svtnID" and "known pubkey in never-registered
			// svtnID" are the same code path — merged into one case. F-L2-03.
			name:   "(b) known pubkey in svtnID not present in store",
			svtnID: svtnLookupID(0x61),
			pub:    pub,
		},
		{
			// (c) revoked pubkey: entry IS in the store; LookupByPubkey returns
			// (entry, true) — revocation is checked at admission time, not here.
			name:    "(c) revoked pubkey — returns hit with IsRevoked=true",
			svtnID:  svtnRegistered,
			pub:     pubRevoked,
			wantHit: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			key, ok := ks.LookupByPubkey(tc.svtnID, tc.pub)

			if tc.wantHit {
				// Case (c): revoked key is in the store; must return ok=true with IsRevoked.
				if !ok {
					t.Fatalf("%s: want ok=true for revoked-but-registered key; got false", tc.name)
				}
				if !key.IsRevoked() {
					t.Errorf("%s: want IsRevoked=true; got false", tc.name)
				}
				return
			}

			assertMissZero(t, tc.name, key, ok)
		})
	}
}
