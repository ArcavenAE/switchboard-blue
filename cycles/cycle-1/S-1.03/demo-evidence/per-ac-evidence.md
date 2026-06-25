---
artifact_id: S-1.03-per-ac-evidence
document_type: demo-evidence
story_id: S-1.03
producer: state-manager
timestamp: 2026-06-25T00:00:00Z
source: backfill from Wave-2 fresh-context audit finding LOW-002
story_rev: "1.3"
pr: "#7"
pr_merge_sha: f35e836
---

# S-1.03 Per-AC Evidence Log

**Story:** S-1.03 — implement session continuity via cryptographic re-authentication in internal/admission  
**Branch:** `feature/S-1.03-session-continuity`  
**PR:** #7 — merged into develop @ `f35e836`  
**Story revision:** 1.3  
**Captured:** 2026-06-25 (backfill — original evidence captured at Step 5 of delivery cycle)  
**In-tree evidence report:** `docs/demo-evidence/S-1.03/evidence-report.md` (on develop branch)

---

## AC Evidence

### AC-001 — ExampleAdmittedKeySet_reAuthenticateOnIPChange

- **Trace:** BC-2.01.007 postconditions 3 and 4 — when a node's source IP changes, the node re-authenticates using the same keypair and the session resumes on the new path
- **Test:** `ExampleAdmittedKeySet_reAuthenticateOnIPChange`
- **File:** `internal/admission/example_test.go`
- **Command:**
  ```
  go test -run '^ExampleAdmittedKeySet_reAuthenticateOnIPChange$' -v ./internal/admission/...
  ```
- **Output:**
  ```
  === RUN   ExampleAdmittedKeySet_reAuthenticateOnIPChange
  --- PASS: ExampleAdmittedKeySet_reAuthenticateOnIPChange (0.00s)
  PASS
  ok  	github.com/arcavenae/switchboard/internal/admission	0.275s
  ```
- **Godoc `// Output:` block:**
  ```
  // reauth error: <nil>
  // still admitted: true
  // source addr: 192.0.2.42
  ```
- **Verdict:** PASS

---

### AC-002 — ExampleAdmittedKeySet_reAuthenticateWrongKeyRejected

- **Trace:** BC-2.01.007 precondition 3 — re-authentication is rejected if the keypair presented does not match the originally admitted keypair
- **Test:** `ExampleAdmittedKeySet_reAuthenticateWrongKeyRejected`
- **File:** `internal/admission/example_test.go`
- **Command:**
  ```
  go test -run '^ExampleAdmittedKeySet_reAuthenticateWrongKeyRejected$' -v ./internal/admission/...
  ```
- **Output:**
  ```
  === RUN   ExampleAdmittedKeySet_reAuthenticateWrongKeyRejected
  --- PASS: ExampleAdmittedKeySet_reAuthenticateWrongKeyRejected (0.00s)
  PASS
  ok  	github.com/arcavenae/switchboard/internal/admission	0.275s
  ```
- **Godoc `// Output:` block:**
  ```
  // is ErrSignatureVerificationFailed: true
  ```
- **Verdict:** PASS

---

### AC-003 — ExampleAdmittedKeySet_reAuthenticateNodeAddressStable

- **Trace:** BC-2.01.007 invariant 3 — after successful re-authentication, the node address (derived from SVTN-ID and public key) remains stable; IP change does not alter the logical node address
- **Test:** `ExampleAdmittedKeySet_reAuthenticateNodeAddressStable`
- **File:** `internal/admission/example_test.go`
- **Command:**
  ```
  go test -run '^ExampleAdmittedKeySet_reAuthenticateNodeAddressStable$' -v ./internal/admission/...
  ```
- **Output:**
  ```
  === RUN   ExampleAdmittedKeySet_reAuthenticateNodeAddressStable
  --- PASS: ExampleAdmittedKeySet_reAuthenticateNodeAddressStable (0.00s)
  PASS
  ok  	github.com/arcavenae/switchboard/internal/admission	0.275s
  ```
- **Godoc `// Output:` block:**
  ```
  // addr stable: true
  // still admitted with same addr: true
  ```
- **Verdict:** PASS

---

## EC Evidence

### EC-001 — ExampleAdmittedKeySet_reAuthenticateExpiredKey

- **Trace:** BC-2.01.007 EC-005 + ARCH-04 Key Lifecycle — re-auth attempt with expired key is rejected with `ErrKeyExpired` (E-ADM-015)
- **Test:** `ExampleAdmittedKeySet_reAuthenticateExpiredKey`
- **File:** `internal/admission/example_test.go`
- **Command:**
  ```
  go test -run '^ExampleAdmittedKeySet_reAuthenticateExpiredKey$' -v ./internal/admission/...
  ```
- **Output:**
  ```
  === RUN   ExampleAdmittedKeySet_reAuthenticateExpiredKey
  --- PASS: ExampleAdmittedKeySet_reAuthenticateExpiredKey (0.00s)
  PASS
  ok  	github.com/arcavenae/switchboard/internal/admission	0.275s
  ```
- **Godoc `// Output:` block:**
  ```
  // is ErrKeyExpired: true
  ```
- **Verdict:** PASS

---

### EC-002 — ExampleAdmittedKeySet_reAuthenticateEvictsOldPath

- **Trace:** BC-2.01.007 EC-006 (v1.3) — re-auth attempt while previous session still live on old IP evicts old path; new path accepted
- **Test:** `ExampleAdmittedKeySet_reAuthenticateEvictsOldPath`
- **File:** `internal/admission/example_test.go`
- **Command:**
  ```
  go test -run '^ExampleAdmittedKeySet_reAuthenticateEvictsOldPath$' -v ./internal/admission/...
  ```
- **Output:**
  ```
  === RUN   ExampleAdmittedKeySet_reAuthenticateEvictsOldPath
  --- PASS: ExampleAdmittedKeySet_reAuthenticateEvictsOldPath (0.00s)
  PASS
  ok  	github.com/arcavenae/switchboard/internal/admission	0.275s
  ```
- **Godoc `// Output:` block:**
  ```
  // after first reauth: 192.0.2.10
  // after second reauth: 198.51.100.20
  ```
- **Verdict:** PASS

---

### EC-003 — ExampleAdmittedKeySet_reAuthenticateLastWriteWins

- **Trace:** BC-2.01.007 EC-003 + ADR-003 — two concurrent re-auth attempts from same node; last one wins
- **Test:** `ExampleAdmittedKeySet_reAuthenticateLastWriteWins`
- **File:** `internal/admission/example_test.go`
- **Command:**
  ```
  go test -run '^ExampleAdmittedKeySet_reAuthenticateLastWriteWins$' -v ./internal/admission/...
  ```
- **Output:**
  ```
  === RUN   ExampleAdmittedKeySet_reAuthenticateLastWriteWins
  --- PASS: ExampleAdmittedKeySet_reAuthenticateLastWriteWins (0.00s)
  PASS
  ok  	github.com/arcavenae/switchboard/internal/admission	0.275s
  ```
- **Godoc `// Output:` block:**
  ```
  // last write wins: 10.10.10.2
  ```
- **Verdict:** PASS

---

## Summary

| ID | Example Function | BC Trace | Verdict |
|----|-----------------|----------|---------|
| AC-001 | ExampleAdmittedKeySet_reAuthenticateOnIPChange | BC-2.01.007 PC3+PC4 | PASS |
| AC-002 | ExampleAdmittedKeySet_reAuthenticateWrongKeyRejected | BC-2.01.007 Pre3 | PASS |
| AC-003 | ExampleAdmittedKeySet_reAuthenticateNodeAddressStable | BC-2.01.007 Inv3 | PASS |
| EC-001 | ExampleAdmittedKeySet_reAuthenticateExpiredKey | BC-2.01.007 EC-005 | PASS |
| EC-002 | ExampleAdmittedKeySet_reAuthenticateEvictsOldPath | BC-2.01.007 EC-006 | PASS |
| EC-003 | ExampleAdmittedKeySet_reAuthenticateLastWriteWins | BC-2.01.007 EC-003 | PASS |

**In-tree evidence report:** `docs/demo-evidence/S-1.03/evidence-report.md`  
**Story spec:** `.factory/stories/S-1.03-node-identity-session-continuity.md` (rev 1.3)  
**PR:** #7 — `f35e836` merged into develop
