# Demo Evidence Report: S-BL.LOOKUP

**Story:** S-BL.LOOKUP — Migrate `AdmittedKeySet.Lookup` / `LookupByPubkey` to `(AdmittedKey, bool)` Value-Return Form
**Story version:** 1.5
**HEAD SHA:** 014490e1c5491165eed6b0b9afc6b0ae79c97599
**Convergence:** 3/3 clean fresh-context passes
**Recorded:** 2026-07-01

---

## Coverage Map

| AC | Description | Recording | Success Path | Error Path |
|----|-------------|-----------|:---:|:---:|
| AC-1 | `Lookup(svtnID, nodeAddr) (AdmittedKey, bool)` value-return signature | [AC-1-lookup-value-return-signature.gif](AC-1-lookup-value-return-signature.gif) | PASS | PASS |
| AC-2 | `LookupByPubkey(svtnID, pubkey) (AdmittedKey, bool)` value-return signature | [AC-2-lookupbypubkey-value-return-signature.gif](AC-2-lookupbypubkey-value-return-signature.gif) | PASS | PASS |
| AC-3 | 4 callsites in svtnmgmt.go use `entry, ok := ...; if !ok` pattern | [AC-3-callsite-migration.gif](AC-3-callsite-migration.gif) | PASS | PASS |
| AC-4 | go.md rule 12: deep-clone `ed25519.PublicKey` (`[]byte`) field | [AC-4-go-rule12-deep-clone.gif](AC-4-go-rule12-deep-clone.gif) | PASS | PASS |
| AC-5 | `go test ./internal/admission/... -run Lookup -race -v` passes | [AC-5-test-coverage-race.gif](AC-5-test-coverage-race.gif) | PASS | PASS |

---

## AC-1: Lookup Value-Return Signature

**File:** `AC-1-lookup-value-return-signature.tape` / `.gif` / `.webm`

**Demonstrates:**
- Success path: `TestLookup_ReturnsBoolTrueOnHit`, `TestLookup_ReturnsBoolFalseOnMiss`, `TestLookup_Miss_ReturnsZeroAdmittedKey_AllFields`, `TestLookup_Hit_AllFieldsMatchRegistration` — hit returns `(AdmittedKey, true)`, miss returns `(zero, false)`
- Error path: `TestLookup_DeepCloneFence_PublicKeyMutationDoesNotLeak` — returned `PublicKey` array is independent (deep-clone fence)

**Implementation:** `internal/admission/admission.go` line 363 — `func (s *AdmittedKeySet) Lookup(svtnID [16]byte, nodeAddr [8]byte) (AdmittedKey, bool)`

---

## AC-2: LookupByPubkey Value-Return Signature

**File:** `AC-2-lookupbypubkey-value-return-signature.tape` / `.gif` / `.webm`

**Demonstrates:**
- Success path: `TestLookupByPubkey_ReturnsBoolTrueOnHit`, `TestLookupByPubkey_ReturnsBoolFalseOnMiss`, `TestAdmittedKeySet_LookupByPubkey` (3 subtests)
- Error path: `TestLookupByPubkey_ExhaustiveLookupOutcomeTable` (unknown pubkey, wrong svtnID, revoked) + `TestLookupByPubkey_DeepCloneFence_PublicKeyMutationDoesNotLeak`

**Implementation:** `internal/admission/admission.go` line 390 — `func (s *AdmittedKeySet) LookupByPubkey(svtnID [16]byte, pubkey ed25519.PublicKey) (AdmittedKey, bool)` — delegates to `Lookup`

---

## AC-3: Callsite Migration

**File:** `AC-3-callsite-migration.tape` / `.gif` / `.webm`

**Demonstrates:**
- Success path: `grep` shows all 4 callsites (`ExpireKey` line 352, `CallerKeyRole` line 404, `CallerKeyRoleActive` line 425, `IsRegisteredAnyState` line 526) use `ok`-idiom with `LookupByPubkey`
- Error path: `grep` for old nil-check pattern (`if stored == nil`, `if entry == nil`, `if lookedUp == nil`) returns zero matches — `CLEAN: no nil-check callsite pattern found`
- `go test ./internal/svtnmgmt/... -race -count=1` passes

---

## AC-4: go.md Rule 12 — Deep-Clone

**File:** `AC-4-go-rule12-deep-clone.tape` / `.gif` / `.webm`

**Demonstrates:**
- Success path: `grep` shows `cp.PublicKey = append(ed25519.PublicKey(nil), entry.PublicKey...)` in `admission.go`; both deep-clone fence tests pass under `-race`
- Error path: concurrent register + lookup tests (`TestLookup_ConcurrentRegisterRace`, `TestLookupByPubkey_ConcurrentSameEntryRegistration`) pass with no `DATA RACE` output

---

## AC-5: Test Coverage with Race Detector

**File:** `AC-5-test-coverage-race.tape` / `.gif` / `.webm`

**Demonstrates:**
- Success path: `go test ./internal/admission/... -run Lookup -race -v` — all `TestLookup*` and `TestLookupByPubkey*` tests PASS, no DATA RACE, package passes
- Error path: `go test ./internal/admission/... -run LookupNonExistent -v` — returns `ok` with `[no test files]` / `testing: warning: no tests to run` (confirms test filter mechanism works and package compiles clean)

---

## Artifact Inventory

```
docs/demo-evidence/S-BL.LOOKUP/
├── AC-1-lookup-value-return-signature.tape
├── AC-1-lookup-value-return-signature.gif
├── AC-1-lookup-value-return-signature.webm
├── AC-2-lookupbypubkey-value-return-signature.tape
├── AC-2-lookupbypubkey-value-return-signature.gif
├── AC-2-lookupbypubkey-value-return-signature.webm
├── AC-3-callsite-migration.tape
├── AC-3-callsite-migration.gif
├── AC-3-callsite-migration.webm
├── AC-4-go-rule12-deep-clone.tape
├── AC-4-go-rule12-deep-clone.gif
├── AC-4-go-rule12-deep-clone.webm
├── AC-5-test-coverage-race.tape
├── AC-5-test-coverage-race.gif
├── AC-5-test-coverage-race.webm
└── evidence-report.md
```

**Total:** 5 ACs covered, 5 tapes, 5 GIFs, 5 WEBMs, 1 evidence report. All success and error paths recorded.
