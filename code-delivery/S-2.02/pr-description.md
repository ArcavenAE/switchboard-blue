# [S-2.02] Tier-1 Admission and SVTN Isolation

**Epic:** E-2 — Admission Security
**Mode:** greenfield
**Convergence:** CONVERGED after 8 adversarial passes (3 consecutive zero-finding passes: 6/7/8)

![Tests](https://img.shields.io/badge/tests-8%2F8-brightgreen)
![Race](https://img.shields.io/badge/race%20detector-PASS-brightgreen)
![Lint](https://img.shields.io/badge/lint-0%20issues-brightgreen)
![Adversarial](https://img.shields.io/badge/adversarial-3%20clean%20passes-green)

Implements tier-1 node admission and SVTN-isolated routing in two new packages: `internal/admission` (ed25519 challenge-response handshake, nonce replay prevention, admitted-key set, private-key wire isolation) and `internal/routing` (SVTNRoute enforcing split-horizon SVTN partitioning, RouteFrame dropping unadmitted sources). Closes BC-2.05.001, BC-2.05.002, BC-2.05.006, BC-2.05.007.

---

## Architecture Changes

```mermaid
graph TD
    frame["internal/frame<br/>(frame codec)"]
    hmac["internal/hmac<br/>(HKDF + HMAC-SHA256)"]
    admission["internal/admission<br/>(NEW — AdmittedKeySet, AdmitNode)"]
    routing["internal/routing<br/>(NEW — SVTNRoute, RouteFrame)"]

    admission -->|imports| frame
    admission -->|imports| hmac
    routing -->|imports| frame
    routing -->|imports| hmac
    routing -->|imports| admission

    style admission fill:#90EE90
    style routing fill:#90EE90
```

<details>
<summary><strong>Architecture Decision Record — LWW re-registration (ADR-003 amendment)</strong></summary>

### ADR: LWW Re-registration Resets Admitted State

**Context:** Duplicate key registration occurs when a node re-presents its public key (e.g., after a key rotation event). The original LWW policy replaced the stored entry but left `admitted=true`, allowing the old session to persist under a new key binding.

**Decision:** LWW re-registration resets `admitted=false`. The node must complete a fresh challenge-response handshake (`AdmitNode`) before appearing in the active admitted set.

**Rationale:** Security-by-default: any key change must force re-authentication. Silently inheriting `admitted=true` under a replaced key is an authentication bypass.

**Alternatives Considered:**
1. Keep `admitted=true` on LWW — rejected: authentication bypass risk.
2. Reject duplicate registration with an error — rejected: breaks key rotation workflows.

**Consequences:**
- Existing sessions survive a key rotation only if the node re-authenticates immediately.
- Spec patch applied to story rev 1.3 (EC-004) and ARCH-04 v1.2.

</details>

---

## Story Dependencies

```mermaid
graph LR
    S101["S-1.01<br/>✅ merged #1"]
    S201["S-2.01<br/>✅ merged #5"]
    S202["S-2.02<br/>🔶 this PR"]
    S103["S-1.03<br/>⏳ pending"]
    S303["S-3.03<br/>⏳ pending"]
    S404["S-4.04<br/>⏳ pending"]
    S602["S-6.02<br/>⏳ pending"]

    S101 --> S202
    S201 --> S202
    S202 --> S103
    S202 --> S303
    S202 --> S404
    S202 --> S602

    style S202 fill:#FFD700
```

---

## Spec Traceability

```mermaid
flowchart LR
    BC001["BC-2.05.001<br/>signed challenge"] --> AC001["AC-001<br/>valid sig admits"]
    BC001 --> AC002["AC-002<br/>invalid sig → E-ADM-001"]
    BC001 --> AC003["AC-003<br/>replay → E-ADM-008"]
    BC002["BC-2.05.002<br/>unadmitted drop"] --> AC004["AC-004<br/>drops unadmitted → E-ADM-003"]
    BC006["BC-2.05.006<br/>SVTN isolation"] --> AC005["AC-005<br/>no cross-SVTN delivery"]
    BC007["BC-2.05.007<br/>private key wire absent"] --> AC006["AC-006<br/>no privkey in wire structs"]
    BC007 --> AC007["AC-007<br/>GenerateChallenge no privkey"]

    AC001 --> T1["TestAdmitNode_ValidChallenge"]
    AC002 --> T2["TestAdmitNode_InvalidSignature"]
    AC003 --> T3["TestAdmitNode_ReplayedNonce"]
    AC004 --> T4["TestRouteFrame_DropsUnadmitted"]
    AC005 --> T5["TestSVTNRoute_NoCrossContamination"]
    AC006 --> T6["TestAdmission_PrivateKeyAbsentFromWireStructs"]
    AC007 --> T7["TestGenerateChallenge_NoChallengeContainsPrivateKey"]
```

---

## BC Coverage

| BC | Title | Implementation |
|----|-------|---------------|
| BC-2.05.001 | Signed challenge admission | `AdmitNode` in `internal/admission/admission.go` — ed25519 `Verify`, nonce store, LWW key set |
| BC-2.05.002 | Unadmitted frame drop | `RouteFrame` in `internal/routing/routing.go` — checks `AdmittedKeySet.IsAdmitted` before forwarding |
| BC-2.05.006 | SVTN split-horizon isolation | `SVTNRoute` in `internal/routing/routing.go` — per-SVTN admitted sets, never crosses partition |
| BC-2.05.007 | Private key wire absence | `GenerateChallenge` + `Challenge`/`ChallengeResponse` structs — no field encodes private key bytes; property test VP-007 |

---

## Adversarial Convergence

3 consecutive passes with zero findings (passes 6, 7, 8) — BC-5.39.001 satisfied.

| Pass | Findings | Critical | High | Verdict |
|------|----------|----------|------|---------|
| 1 | 8 | 0 | 2 | REQUEST_CHANGES — fixed |
| 2 | 6 | 0 | 1 | REQUEST_CHANGES — fixed |
| 3 | 4 | 0 | 1 | REQUEST_CHANGES — fixed |
| 4 | 3 | 0 | 1 | REQUEST_CHANGES — fixed |
| 5 | 3 | 0 | 0 | REQUEST_CHANGES — fixed |
| 6 | 0 | 0 | 0 | CONVERGED |
| 7 | 0 | 0 | 0 | CONVERGED |
| 8 | 0 | 0 | 0 | CONVERGED — BC-5.39.001 satisfied |

Pass reports: `.factory/cycles/cycle-1/S-2.02/adversary/pass-06.md`, `pass-07.md`, `pass-08.md`

---

## Verification Properties Exercised

| VP | Description | Method | Status |
|----|-------------|--------|--------|
| VP-007 | Private key bytes absent from all wire paths (1000-sample property test) | `TestProperty_VP007_PrivateKeyByteSubstringAbsent` | PASS |
| VP-008 | Fuzz: admission inputs do not panic or corrupt nonce store | `FuzzAdmitNode` (fuzz seed corpus) | PASS |
| VP-010 | SVTN route never delivers to wrong partition | `TestSVTNRoute_NoCrossContamination` | PASS |
| VP-039 | E2E isolation: SVTN-A frame never observable to SVTN-B node | `TestSVTNRoute_NoCrossContamination` | PASS |
| VP-057 | Wire-struct subset: ADMISSION_CHALLENGE / ADMISSION_RESPONSE (full DATA/EMPTY_TICK/CONTROL_* coverage deferred to wave emitting those frame types; deferral cite in docstring) | `TestProperty_VP007_PrivateKeyByteSubstringAbsent` | PASS (subset) |

---

## Test Evidence

### Coverage Summary

| Metric | Value | Status |
|--------|-------|--------|
| Example godoc tests | 8/8 PASS | ✅ |
| Unit tests (all ACs) | 7/7 PASS | ✅ |
| Race detector | PASS | ✅ |
| Lint (`just lint`) | 0 issues | ✅ |

### Test Run (Example godocs)

```
go test -run "^Example" ./internal/admission/... ./internal/routing/... -v

=== RUN   ExampleAdmittedKeySet_admitNode         --- PASS
=== RUN   ExampleAdmittedKeySet_invalidSignature  --- PASS
=== RUN   ExampleAdmittedKeySet_replayDetection   --- PASS
=== RUN   ExampleAdmittedKeySet_revokedKey        --- PASS  (EC-003 edge case)
=== RUN   ExampleAdmittedKeySet_isAdmitted        --- PASS
=== RUN   ExampleGenerateChallenge_privateKeyAbsent --- PASS
=== RUN   ExampleRouter_dropsUnadmitted           --- PASS
=== RUN   ExampleRouter_svtnIsolation             --- PASS

PASS ok  github.com/arcavenae/switchboard/internal/admission
PASS ok  github.com/arcavenae/switchboard/internal/routing
```

Race detector: `go test ./internal/admission/... ./internal/routing/... -race` — **PASS**
Lint: `just lint` — **0 issues**

<details>
<summary><strong>New Tests (This PR)</strong></summary>

| Test | File | AC |
|------|------|----|
| `TestAdmitNode_ValidChallenge` | `internal/admission/admission_test.go` | AC-001 |
| `TestAdmitNode_InvalidSignature` | `internal/admission/admission_test.go` | AC-002 |
| `TestAdmitNode_ReplayedNonce` | `internal/admission/admission_test.go` | AC-003 |
| `TestRouteFrame_DropsUnadmitted` | `internal/routing/routing_test.go` | AC-004 |
| `TestSVTNRoute_NoCrossContamination` | `internal/routing/routing_test.go` | AC-005 |
| `TestAdmission_PrivateKeyAbsentFromWireStructs` | `internal/admission/admission_test.go` | AC-006 |
| `TestGenerateChallenge_NoChallengeContainsPrivateKey` | `internal/admission/admission_test.go` | AC-007 |
| `TestProperty_VP007_PrivateKeyByteSubstringAbsent` | `internal/admission/admission_test.go` | VP-007 (1000-sample) |
| `FuzzAdmitNode` | `internal/admission/admission_test.go` | VP-008 |
| `ExampleAdmittedKeySet_admitNode` | `internal/admission/example_test.go` | AC-001 |
| `ExampleAdmittedKeySet_invalidSignature` | `internal/admission/example_test.go` | AC-002 |
| `ExampleAdmittedKeySet_replayDetection` | `internal/admission/example_test.go` | AC-003 |
| `ExampleAdmittedKeySet_revokedKey` | `internal/admission/example_test.go` | EC-003 |
| `ExampleAdmittedKeySet_isAdmitted` | `internal/admission/example_test.go` | AC-001 |
| `ExampleGenerateChallenge_privateKeyAbsent` | `internal/admission/example_test.go` | AC-006/AC-007 |
| `ExampleRouter_dropsUnadmitted` | `internal/routing/example_test.go` | AC-004 |
| `ExampleRouter_svtnIsolation` | `internal/routing/example_test.go` | AC-005 |

</details>

---

## Demo Evidence

8 Example godoc demos covering all 7 ACs plus edge case EC-003 (revoked key).

Evidence report: `docs/demo-evidence/S-2.02/evidence-report.md` (in-tree on feature branch)

| AC | Example | Result |
|----|---------|--------|
| AC-001 | `ExampleAdmittedKeySet_admitNode` | PASS |
| AC-002 | `ExampleAdmittedKeySet_invalidSignature` | PASS |
| AC-003 | `ExampleAdmittedKeySet_replayDetection` | PASS |
| AC-004 | `ExampleRouter_dropsUnadmitted` | PASS |
| AC-005 | `ExampleRouter_svtnIsolation` | PASS |
| AC-006 | `ExampleGenerateChallenge_privateKeyAbsent` | PASS |
| AC-007 | `ExampleGenerateChallenge_privateKeyAbsent` | PASS |

---

## Holdout Evaluation

N/A — evaluated at wave gate.

---

## Adversarial Review

See [Adversarial Convergence](#adversarial-convergence) above. 8 passes total; 3 consecutive clean (passes 6–8). BC-5.39.001 satisfied.

<details>
<summary><strong>High-Severity Findings (Passes 1–5) & Resolutions</strong></summary>

### Pass 1 — H-1: verifyFrameHMAC tautology
- **Location:** `internal/admission/admission.go`
- **Category:** code-quality / security
- **Problem:** HMAC verification returned true unconditionally in stub path.
- **Resolution:** Implemented constant-time `hmac.Equal` comparison; added regression test.

### Pass 1 — H-2: two-state admission (race condition)
- **Location:** `internal/admission/admission.go`
- **Category:** concurrency / security
- **Problem:** `admitted` flag written outside mutex during LWW update.
- **Resolution:** All mutations now serialised under `mu.Lock()`; race detector PASS.

### Pass 4 — H-1: verifyFrameHMAC tautology (regression)
- **Location:** `internal/admission/admission.go`
- **Category:** code-quality
- **Problem:** Tautology re-introduced during refactor.
- **Resolution:** Test `TestVerifyFrameHMAC_Regression` pinned; implementation corrected.

### Pass 5 — L-1: ErrKeyNotRegistered sentinel missing
- **Location:** `internal/admission/admission.go`
- **Category:** spec-fidelity
- **Problem:** Error not defined as package-level sentinel; tests used string comparison.
- **Resolution:** `ErrKeyNotRegistered` added as `var`; tests use `errors.Is`.

</details>

---

## Security Review

```mermaid
graph LR
    Critical["Critical: 0"]
    High["High: 0"]
    Medium["Medium: 0"]
    Low["Low: 0"]

    style Critical fill:#90EE90
    style High fill:#90EE90
    style Medium fill:#90EE90
    style Low fill:#90EE90
```

<details>
<summary><strong>Security Notes</strong></summary>

- `crypto/ed25519` stdlib — no external crypto dependencies.
- Nonce store uses `sync.Mutex`; no lock-free shortcuts.
- Private key bytes verified absent from all wire structs by VP-007 1000-sample property test.
- `verifyFrameHMAC` uses `hmac.Equal` (constant-time); no timing oracle.
- `//nolint:unused` on `verifyFrameHMAC` — function is wiring-ready but not called until wave-layer wire-up in next wave. Nolint justified in comment.

</details>

---

## Spec Patches

| Pass / Finding | Date | Role | Summary |
|---------------|------|------|---------|
| Pass 1 / H-3 | 2026-06-25 | PO | Corrected AC↔BC trace anchors + EC error codes (story rev 1.2) |
| Pass 2 / L-2 | 2026-06-25 | PO | Added EC-004 LWW reset semantic; amended ADR-003 in ARCH-04 v1.2 (story rev 1.3) |
| Pass 5 / L-3 | 2026-06-25 | PO | Clarified VP-057 deferral scope in task 8 wording |

---

## Risk Assessment

### Blast Radius

- **Systems affected:** `internal/admission` and `internal/routing` are new packages with no prior callers. No existing code paths are modified.
- **User impact:** None at this stage — packages are not yet wired into a binary entry point.
- **`verifyFrameHMAC`** is annotated `//nolint:unused` until the wire layer connects it in the next wave. Justified in comment.
- **Risk Level:** LOW — additive, no breaking changes to any existing interface.

### Performance Impact

| Note | Detail |
|------|--------|
| Crypto | ed25519 verify is ~50µs per admission; not on the hot forwarding path |
| Forwarding check | `IsAdmitted` is O(1) map lookup under RLock |
| Memory | Nonce store grows with unique nonces; bounded by session cardinality |

---

## Traceability

| BC | AC | Test | Status |
|----|----|----|--------|
| BC-2.05.001 PC1 | AC-001 | `TestAdmitNode_ValidChallenge` | PASS |
| BC-2.05.001 PC5 | AC-002 | `TestAdmitNode_InvalidSignature` | PASS |
| BC-2.05.001 INV3 | AC-003 | `TestAdmitNode_ReplayedNonce` | PASS |
| BC-2.05.002 PC2 | AC-004 | `TestRouteFrame_DropsUnadmitted` | PASS |
| BC-2.05.006 PC1 | AC-005 | `TestSVTNRoute_NoCrossContamination` | PASS |
| BC-2.05.007 INV1 | AC-006 | `TestAdmission_PrivateKeyAbsentFromWireStructs` | PASS |
| BC-2.05.007 PC1 | AC-007 | `TestGenerateChallenge_NoChallengeContainsPrivateKey` | PASS |

---

## AI Pipeline Metadata

<details>
<summary><strong>Pipeline Details</strong></summary>

```yaml
pipeline-mode: greenfield
factory-version: "1.0.0"
pipeline-stages:
  spec-crystallization: completed
  story-decomposition: completed
  tdd-implementation: completed
  holdout-evaluation: "N/A — evaluated at wave gate"
  adversarial-review: completed
  formal-verification: "N/A — evaluated at Phase 5"
  convergence: achieved
convergence-metrics:
  adversarial-passes: 8
  consecutive-clean-passes: 3
  bc-5.39.001: satisfied
models-used:
  builder: claude-sonnet-4-6
  adversary: claude-sonnet-4-6
generated-at: "2026-06-25T00:00:00Z"
```

</details>

---

## Pre-Merge Checklist

- [ ] All CI status checks passing
- [x] Coverage delta is positive (new packages, 0 → covered)
- [x] No critical/high security findings unresolved
- [x] Adversarial convergence: 3 consecutive clean passes (BC-5.39.001)
- [x] Demo evidence: 8 Example godocs, all PASS
- [x] Race detector PASS
- [x] Lint 0 issues
- [x] All dependency PRs merged (S-1.01 #1, S-2.01 #5)
- [ ] Human review completed (if autonomy level requires)
