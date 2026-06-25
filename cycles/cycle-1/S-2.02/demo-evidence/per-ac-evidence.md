---
artifact_id: S-2.02-per-ac-evidence
document_type: demo-evidence
story_id: S-2.02
producer: state-manager
timestamp: 2026-06-25T00:00:00Z
source: backfill from Wave-2 consistency-validator gate finding MEDIUM-002
story_rev: "1.3"
pr: "#6"
pr_merge_sha: a06b306
---

# S-2.02 Per-AC Evidence Log

**Story:** S-2.02 — implement tier-1 admission and SVTN isolation in internal/admission and internal/routing  
**Branch:** `feature/S-2.02-admission-svtn-isolation`  
**PR:** #6 — merged into develop @ `a06b306`  
**Story revision:** 1.3  
**Captured:** 2026-06-25 (backfill — original evidence captured at Step 5 of delivery cycle)  
**In-tree evidence report:** `docs/demo-evidence/S-2.02/evidence-report.md` (on develop branch)

---

## AC Evidence

### AC-001 — ExampleAdmittedKeySet_admitNode

- **Trace:** BC-2.05.001 postcondition 1 — `AdmitNode` succeeds when signature over challenge is valid for the presented public key
- **Test:** `ExampleAdmittedKeySet_admitNode`
- **File:** `internal/admission/example_test.go`
- **Command:**
  ```
  go test -run '^ExampleAdmittedKeySet_admitNode$' -v ./internal/admission/...
  ```
- **Output:**
  ```
  === RUN   ExampleAdmittedKeySet_admitNode
  --- PASS: ExampleAdmittedKeySet_admitNode (0.00s)
  PASS
  ok  	github.com/arcavenae/switchboard/internal/admission	0.493s
  ```
- **Godoc `// Output:` block:**
  ```
  // admit error: <nil>
  // is admitted: true
  ```
- **Verdict:** PASS

---

### AC-002 — ExampleAdmittedKeySet_invalidSignature

- **Trace:** BC-2.05.001 postcondition 5 — `AdmitNode` returns `ErrSignatureVerificationFailed` (E-ADM-001) when the signature is invalid
- **Test:** `ExampleAdmittedKeySet_invalidSignature`
- **File:** `internal/admission/example_test.go`
- **Command:**
  ```
  go test -run '^ExampleAdmittedKeySet_invalidSignature$' -v ./internal/admission/...
  ```
- **Output:**
  ```
  === RUN   ExampleAdmittedKeySet_invalidSignature
  --- PASS: ExampleAdmittedKeySet_invalidSignature (0.00s)
  PASS
  ok  	github.com/arcavenae/switchboard/internal/admission	0.493s
  ```
- **Godoc `// Output:` block:**
  ```
  // is ErrSignatureVerificationFailed: true
  ```
- **Verdict:** PASS

---

### AC-003 — ExampleAdmittedKeySet_replayDetection

- **Trace:** BC-2.05.001 invariant 3 — `AdmitNode` returns `ErrNonceReplay` (E-ADM-008) when the challenge nonce has already been used
- **Test:** `ExampleAdmittedKeySet_replayDetection`
- **File:** `internal/admission/example_test.go`
- **Command:**
  ```
  go test -run '^ExampleAdmittedKeySet_replayDetection$' -v ./internal/admission/...
  ```
- **Output:**
  ```
  === RUN   ExampleAdmittedKeySet_replayDetection
  --- PASS: ExampleAdmittedKeySet_replayDetection (0.00s)
  PASS
  ok  	github.com/arcavenae/switchboard/internal/admission	0.493s
  ```
- **Godoc `// Output:` block:**
  ```
  // first admission error: <nil>
  // is ErrNonceReplay: true
  ```
- **Verdict:** PASS

---

### AC-004 — ExampleRouter_dropsUnadmitted

- **Trace:** BC-2.05.002 postcondition 2 — `RouteFrame` drops the frame and returns `E-ADM-003` if the frame's `src_addr` is not in the admitted set for the frame's `svtn_id`
- **Test:** `ExampleRouter_dropsUnadmitted`
- **File:** `internal/routing/example_test.go`
- **Command:**
  ```
  go test -run '^ExampleRouter_dropsUnadmitted$' -v ./internal/routing/...
  ```
- **Output:**
  ```
  === RUN   ExampleRouter_dropsUnadmitted
  --- PASS: ExampleRouter_dropsUnadmitted (0.00s)
  PASS
  ok  	github.com/arcavenae/switchboard/internal/routing	0.766s
  ```
- **Verdict:** PASS

---

### AC-005 — ExampleRouter_svtnIsolation

- **Trace:** BC-2.05.006 postcondition 1 — `SVTNRoute` never delivers a frame to a node on a different SVTN
- **Test:** `ExampleRouter_svtnIsolation`
- **File:** `internal/routing/example_test.go`
- **Command:**
  ```
  go test -run '^ExampleRouter_svtnIsolation$' -v ./internal/routing/...
  ```
- **Output:**
  ```
  === RUN   ExampleRouter_svtnIsolation
  --- PASS: ExampleRouter_svtnIsolation (0.00s)
  PASS
  ok  	github.com/arcavenae/switchboard/internal/routing	0.766s
  ```
- **Verdict:** PASS

---

### AC-006 + AC-007 — ExampleGenerateChallenge_privateKeyAbsent

- **Trace:** BC-2.05.007 invariant 1 (AC-006) + postcondition 1 (AC-007) — no wire struct produced by `internal/admission` contains private key bytes; `GenerateChallenge()` produces a nonce without using or transmitting the node's private key
- **Test:** `ExampleGenerateChallenge_privateKeyAbsent`
- **File:** `internal/admission/example_test.go`
- **Command:**
  ```
  go test -run '^ExampleGenerateChallenge_privateKeyAbsent$' -v ./internal/admission/...
  ```
- **Output:**
  ```
  === RUN   ExampleGenerateChallenge_privateKeyAbsent
  --- PASS: ExampleGenerateChallenge_privateKeyAbsent (0.00s)
  PASS
  ok  	github.com/arcavenae/switchboard/internal/admission	0.493s
  ```
- **Godoc `// Output:` block:**
  ```
  // nonce not in private key: true
  // RouterSig not in private key: true
  // no private key material on wire: true
  ```
- **Verdict:** PASS (both AC-006 and AC-007 satisfied by single property Example)

---

## EC Evidence

### EC-003 — ExampleAdmittedKeySet_revokedKey

- **Trace:** BC-2.05.001 EC-001 — `AdmitNode` returns `ErrKeyRevoked` (E-ADM-005) when the key has been revoked before the handshake
- **Test:** `ExampleAdmittedKeySet_revokedKey`
- **File:** `internal/admission/example_test.go`
- **Command:**
  ```
  go test -run '^ExampleAdmittedKeySet_revokedKey$' -v ./internal/admission/...
  ```
- **Output:**
  ```
  === RUN   ExampleAdmittedKeySet_revokedKey
  --- PASS: ExampleAdmittedKeySet_revokedKey (0.00s)
  PASS
  ok  	github.com/arcavenae/switchboard/internal/admission	0.493s
  ```
- **Godoc `// Output:` block:**
  ```
  // is ErrKeyRevoked: true
  ```
- **Verdict:** PASS

---

## BC Evidence

### BC-2.05.001 PC4 two-state model — ExampleAdmittedKeySet_isAdmitted

- **Trace:** BC-2.05.001 postcondition 4 — `IsAdmitted` returns false after `RegisterKey` alone; true only after `AdmitNode` succeeds
- **Test:** `ExampleAdmittedKeySet_isAdmitted`
- **File:** `internal/admission/example_test.go`
- **Command:**
  ```
  go test -run '^ExampleAdmittedKeySet_isAdmitted$' -v ./internal/admission/...
  ```
- **Output:**
  ```
  === RUN   ExampleAdmittedKeySet_isAdmitted
  --- PASS: ExampleAdmittedKeySet_isAdmitted (0.00s)
  PASS
  ok  	github.com/arcavenae/switchboard/internal/admission	0.493s
  ```
- **Godoc `// Output:` block:**
  ```
  // before register: false
  // after register, before handshake: false
  // after handshake: true
  ```
- **Verdict:** PASS

---

## Summary

| ID | Example Function | BC Trace | Verdict |
|----|-----------------|----------|---------|
| AC-001 | ExampleAdmittedKeySet_admitNode | BC-2.05.001 PC1 | PASS |
| AC-002 | ExampleAdmittedKeySet_invalidSignature | BC-2.05.001 PC5 | PASS |
| AC-003 | ExampleAdmittedKeySet_replayDetection | BC-2.05.001 Inv3 | PASS |
| AC-004 | ExampleRouter_dropsUnadmitted | BC-2.05.002 PC2 | PASS |
| AC-005 | ExampleRouter_svtnIsolation | BC-2.05.006 PC1 | PASS |
| AC-006 | ExampleGenerateChallenge_privateKeyAbsent | BC-2.05.007 Inv1 | PASS |
| AC-007 | ExampleGenerateChallenge_privateKeyAbsent | BC-2.05.007 PC1 | PASS |
| EC-003 | ExampleAdmittedKeySet_revokedKey | BC-2.05.001 EC-001 | PASS |
| BC-2.05.001 PC4 | ExampleAdmittedKeySet_isAdmitted | BC-2.05.001 PC4 | PASS |

**In-tree evidence report:** `docs/demo-evidence/S-2.02/evidence-report.md`  
**Story spec:** `.factory/stories/S-2.02-admission-svtn-isolation.md` (rev 1.3)  
**PR:** #6 — `a06b306` merged into develop
