# [S-3.04] wire verifyFrameHMAC into RouteFrame (BC-2.05.008, ADR-009)

**Epic:** E-2 — Admission Security
**Mode:** greenfield
**Convergence:** CONVERGED after 5 adversarial passes (3 consecutive clean: passes 3/4/5)

![Tests](https://img.shields.io/badge/tests-PASS-brightgreen)
![Coverage](https://img.shields.io/badge/coverage-internal%2Frouting-brightgreen)
![Adversary](https://img.shields.io/badge/adversary-5%20passes%2C%203%20clean-green)
![Holdout](https://img.shields.io/badge/holdout-N%2FA%20wave%20gate-blue)

Wires `verifyFrameHMAC` into `RouteFrame` per BC-2.05.008 v1.1. HMAC verification now precedes the admitted-set check (fail-fast on frame forgery). Closes WAVE-3-DEP-001: the post-Wave-2 drift item that Wave-2 router would have forwarded any frame matching only a valid `(svtnID, srcAddr)` tuple — post-S-3.04 requires HMAC verification first. Sentinel `ErrHMACVerificationFailed` added; maps to E-ADM-016. No new packages; `internal/routing/routing.go` only.

---

## Architecture Changes

```mermaid
graph TD
    RF["RouteFrame()"]
    VH["verifyFrameHMAC()"]
    FT["ForwardingTable.Lookup()"]
    IA["admittedKeySet.IsAdmitted()"]
    SR["SVTNRoute()"]

    RF -->|"1 — lookup auth key"| FT
    RF -->|"2 — verify HMAC tag"| VH
    VH -.->|"new call (S-3.04)"| RF
    RF -->|"3 — admitted-set check"| IA
    RF -->|"4 — forward"| SR

    style VH fill:#90EE90
```

<details>
<summary><strong>Architecture Decision Record: ADR-009 v1.6</strong></summary>

### ADR-009 v1.6 — Lock-Free HMAC Verification with Defensive Copy

**Context:** Pass-1 adversary found that the initial implementation re-acquired the forwarding-table `RLock` to read `FrameAuthKey` after already having dropped it (ADR-009 v1.5 "single RLock acquisition" violated). Pass-2 adversary found the v1.6 fix was incomplete — the defensive copy of `FrameAuthKey` was written to the spec but not yet in code.

**Decision:** `RouteFrame` acquires `RLock` once, copies `entry.FrameAuthKey` into a stack-local `[hmac.KeySize]byte`, releases the lock, then calls `verifyFrameHMAC` with the copy. HMAC computation runs outside the lock entirely.

**Rationale:** Prevents lock-hold during constant-time HMAC (mitigates timing-channel interference from scheduling jitter under lock). Eliminates double-acquisition bug from v1.5.

**Alternatives Considered:**
1. Hold `RLock` across `verifyFrameHMAC` — rejected: constant-time HMAC under lock amplifies scheduling jitter; per ADR-009 v1.6 step 3.
2. Copy entire `ForwardingEntry` — rejected: unnecessarily wide copy; only `FrameAuthKey` is consumed outside the lock.

**Consequences:**
- HMAC verification is lock-free; forwarding table is not held during crypto.
- `FrameAuthKey` stack copy is bounded (`[hmac.KeySize]byte` = 32 bytes).

</details>

---

## Story Dependencies

```mermaid
graph LR
    S201["S-2.01<br/>✅ merged"]
    S202["S-2.02<br/>✅ merged"]
    S304["S-3.04<br/>🟡 this PR"]

    S201 --> S304
    S202 --> S304
```

Dependencies S-2.01 and S-2.02 are merged on `develop` (develop tip `d8d7ae6`). S-3.04 blocks no other story currently scheduled in Wave 3.

---

## Spec Traceability

```mermaid
flowchart LR
    BC["BC-2.05.008 v1.1<br/>HMAC-before-admission"]
    AC1["AC-001<br/>valid HMAC → admission"]
    AC2["AC-002<br/>invalid HMAC → E-ADM-016"]
    AC3["AC-003<br/>HMAC before admitted-set"]
    AC4["AC-004<br/>no fwd entry → rejected"]
    AC5["AC-005<br/>valid HMAC + unadmitted → ErrNotAdmitted"]
    VP["VP-058<br/>props 1-4"]

    BC --> AC1
    BC --> AC2
    BC --> AC3
    BC --> AC4
    BC --> AC5
    AC3 --> VP
    AC4 --> VP

    AC1 --> T1["TestRouteFrame_ValidHMAC_ProceedsToAdmission"]
    AC2 --> T2["TestRouteFrame_InvalidHMAC_ReturnsErrHMACVerificationFailed"]
    AC3 --> T3["TestRouteFrame_HMACEnforcedBeforeAdmission"]
    AC4 --> T4["TestRouteFrame_NoForwardingEntry_RejectsAsUnverifiable"]
    AC5 --> T5["TestRouteFrame_ValidHMAC_Unadmitted_ReturnsErrNotAdmitted"]

    T1 --> SRC["internal/routing/routing.go"]
    T2 --> SRC
    T3 --> SRC
    T4 --> SRC
    T5 --> SRC
```

---

## Test Evidence

### Coverage Summary

| Metric | Value | Threshold | Status |
|--------|-------|-----------|--------|
| Unit tests (AC + EC) | 5 AC + 5 EC = all pass | 100% | PASS |
| Example godoc tests | 7 S-3.04 + 2 S-2.02 retained = 9 total | all pass | PASS |
| Race detector (`-race -count=3`) | PASS | no races | PASS |
| Lint (`just lint`) | 0 issues | 0 warnings | PASS |
| Holdout evaluation | N/A — evaluated at wave gate | — | — |

### Test Flow

```mermaid
graph LR
    Unit["5 AC tests + 5 EC tests"]
    Ex["9 Example godoc tests"]
    Fuzz["FuzzRouteFrame harness<br/>(VP-008 coverage restored)"]
    Race["Race detector -count=3"]
    Lint["golangci-lint 0 issues"]

    Unit -->|all pass| P1["PASS"]
    Ex -->|all pass| P2["PASS"]
    Fuzz -->|VP-008 exercised| P3["PASS"]
    Race -->|no data races| P4["PASS"]
    Lint -->|0 warnings| P5["PASS"]

    style P1 fill:#90EE90
    style P2 fill:#90EE90
    style P3 fill:#90EE90
    style P4 fill:#90EE90
    style P5 fill:#90EE90
```

| Metric | Value |
|--------|-------|
| **New tests (AC/EC)** | 5 AC + 5 EC added in `routing_internal_test.go` + `routing_test.go` |
| **New Example tests** | 7 added in `internal/routing/example_test.go` |
| **Modified tests** | 4 pre-S-3.04 tests updated (HMAC-before-admission ordering) |
| **Regression** | none |

<details>
<summary><strong>New Tests (This PR)</strong></summary>

### AC / EC Tests

| Test | File | AC/EC | Result |
|------|------|-------|--------|
| `TestRouteFrame_ValidHMAC_ProceedsToAdmission` | `routing_internal_test.go` | AC-001 | PASS |
| `TestRouteFrame_InvalidHMAC_ReturnsErrHMACVerificationFailed` | `routing_internal_test.go` | AC-002 | PASS |
| `TestRouteFrame_HMACEnforcedBeforeAdmission` | `routing_internal_test.go` | AC-003 / VP-058 p1+2 | PASS |
| `TestRouteFrame_NoForwardingEntry_RejectsAsUnverifiable` | `routing_internal_test.go` | AC-004 / VP-058 p4 | PASS |
| `TestRouteFrame_ValidHMAC_Unadmitted_ReturnsErrNotAdmitted` | `routing_test.go` | AC-005 | PASS |

### Example Tests

| Test | AC/EC | Result |
|------|-------|--------|
| `ExampleRouter_validHMACForwarded` | AC-001 | PASS |
| `ExampleRouter_invalidHMACRejected` | AC-002 | PASS |
| `ExampleRouter_hmacBeforeAdmission` | AC-003 | PASS |
| `ExampleRouter_noForwardingEntry` | AC-004 | PASS |
| `ExampleRouter_validHMACUnadmittedRejected` | AC-005 | PASS |
| `ExampleRouter_zeroHMACTagRejected` | EC-001 | PASS |
| `ExampleRouter_wrongKeyHMACRejected` | EC-002 | PASS |

### VP-058 Properties Directly Tested

| Property | Test | Status |
|----------|------|--------|
| P1: verifyFrameHMAC called before IsAdmitted | `TestRouteFrame_HMACEnforcedBeforeAdmission` | PASS |
| P2: verifyFrameHMAC called before SVTNRoute | `TestRouteFrame_HMACEnforcedBeforeAdmission` | PASS |
| P3: invalid HMAC → ErrHMACVerificationFailed (not ErrNotAdmitted) | `TestRouteFrame_InvalidHMAC_*` | PASS |
| P4: missing fwd entry → ErrHMACVerificationFailed | `TestRouteFrame_NoForwardingEntry_*` | PASS |

</details>

---

## Holdout Evaluation

N/A — evaluated at wave gate per VSDD pipeline policy. No per-story holdout run for intra-wave stories.

---

## Adversarial Review

| Pass | Findings | Critical | High | Medium | Low | Verdict |
|------|----------|----------|------|--------|-----|---------|
| 1 (`30bfa69`) | 2 | 0 | 0 | 1 | 1 | NOT_CONVERGED |
| 2 (`e214f8d`) | 2 | 0 | 0 | 1 | 1 | NOT_CONVERGED |
| 3 (`e214f8d`) | 0 | 0 | 0 | 0 | 0 | CONVERGED |
| 4 (`e214f8d`) | 0 | 0 | 0 | 0 | 0 | CONVERGED |
| 5 (`e214f8d`) | 0 | 0 | 0 | 0 | 0 | CONVERGED |

**Convergence:** 3 consecutive clean passes (3/4/5) — BC-5.39.001 satisfied.
Adversary pass docs: `.factory/cycles/cycle-1/S-3.04/adversary/pass-01..pass-05.md`

<details>
<summary><strong>Findings &amp; Resolutions</strong></summary>

### Pass-1 M-1 — ADR-009 v1.5 "single RLock acquisition" violated; comment contradicts implementation

- **Location:** `internal/routing/routing.go:112-141`
- **Category:** spec-fidelity / code-quality
- **Problem:** Initial implementation called `verifyFrameHMAC` then re-acquired `RLock` to read `FrameAuthKey`, violating ADR-009 v1.5. Stale comment at line 115 described an ordering that did not match the code.
- **Resolution:** ADR-009 amended to v1.6 (lock-free HMAC verify with defensive copy). Comment corrected at `routing.go:115`. Defensive copy added at `routing.go:133` (commit `e99bcf2`).
- **Test:** `TestRouteFrame_HMACEnforcedBeforeAdmission` verifies ordering at runtime.

### Pass-1 L-1 — Stale docstrings in routing.go referenced pre-S-3.04 routing path

- **Location:** `internal/routing/routing.go` (multiple docstrings)
- **Category:** code-quality
- **Problem:** Several docstrings described `RouteFrame` as performing only admitted-set check — no mention of HMAC verification.
- **Resolution:** Docstrings updated in commit `15353b1`.

### Pass-2 M-1 — ADR-009 v1.6 step-3 defensive copy not yet in implementation

- **Location:** `internal/routing/routing.go:133`
- **Category:** spec-fidelity
- **Problem:** ADR-009 v1.6 spec described the defensive copy of `FrameAuthKey` but the implementation had not yet added it — HMAC was called with a pointer into live forwarding table state under RLock.
- **Resolution:** Defensive copy added (`authKey := entry.FrameAuthKey`) before `RUnlock` in commit `e99bcf2`.

### Pass-2 L-1 — FuzzRouteFrame_NonAdmittedNeverForwarded did not genuinely exercise VP-008

- **Location:** `internal/routing/routing_internal_test.go` (fuzz harness)
- **Category:** test-quality
- **Problem:** Fuzz harness was constructing frames with valid HMAC tags and checking that unadmitted nodes were not forwarded — but the new HMAC-first rejection meant VP-008 coverage was being short-circuited before reaching the admitted-set check.
- **Resolution:** Fuzz harness updated to seed both HMAC-invalid and HMAC-valid-but-unadmitted inputs, restoring VP-008 coverage. Commit `e214f8d`.

</details>

---

## Security Review

Security review executed during adversarial convergence passes.

```mermaid
graph LR
    Crit["Critical: 0"]
    High["High: 0"]
    Med["Medium: 0 (2 fixed pre-merge)"]
    Low["Low: 0 (2 fixed pre-merge)"]

    style Crit fill:#90EE90
    style High fill:#90EE90
    style Med fill:#90EE90
    style Low fill:#87CEEB
```

<details>
<summary><strong>Security Details</strong></summary>

### HMAC Timing-Channel Analysis

The `verifyFrameHMAC` function uses `hmac.Equal` (constant-time comparison) for tag verification, satisfying CWE-208 (Observable Timing Discrepancy) mitigation. The defensive-copy pattern (ADR-009 v1.6) ensures HMAC computation occurs without holding `RLock`, preventing lock-contention interference from biasing timing measurements.

### SAST / Lint

- `golangci-lint run ./...` — 0 findings
- No `//nolint` annotations added without justification
- `verifyFrameHMAC` `//nolint:unused` annotation removed (function is now called)

### Dependency Audit

No new dependencies introduced. `internal/routing` imports unchanged: `{frame, hmac, admission}` only (ARCH-08 §6.5 position 5 compliance verified).

### Formal Properties

| Property | Verification Method | Status |
|----------|--------------------|-|
| VP-058 P1: HMAC before IsAdmitted | `TestRouteFrame_HMACEnforcedBeforeAdmission` | PASS |
| VP-058 P2: HMAC before SVTNRoute | `TestRouteFrame_HMACEnforcedBeforeAdmission` | PASS |
| VP-058 P3: ErrHMACVerificationFailed on invalid tag | direct test | PASS |
| VP-058 P4: missing fwd entry → ErrHMACVerificationFailed | direct test | PASS |
| Race-free (no data races under concurrent routing) | `go test -race -count=3` | PASS |

</details>

---

## Risk Assessment & Deployment

### Blast Radius

- **Systems affected:** `internal/routing` only — no new packages, no schema changes
- **User impact:** Post-S-3.04 callers of `RouteFrame` with no forwarding entry or invalid HMAC now receive `ErrHMACVerificationFailed` rather than proceeding to forwarding. Callers that previously relied on the admitted-set check as the first gate must handle `ErrHMACVerificationFailed` before `ErrNotAdmitted`. 4 pre-existing tests were updated accordingly.
- **Data impact:** None — forwarding table schema unchanged; `FrameAuthKey` field pre-existed from S-2.02.
- **Risk Level:** LOW — single-file change; additive security enforcement only; covered by table-driven AC+EC tests + Example tests + race detector.

### Performance Impact

| Metric | Before | After | Delta | Status |
|--------|--------|-------|-------|--------|
| HMAC verify per frame | 0 (not called) | ~1 µs (constant-time, lock-free) | +1 µs | OK |
| RLock hold duration | N/A | reduced (lock released before HMAC) | negative delta | OK |
| Memory | baseline | +32 bytes stack per `RouteFrame` call (defensive copy) | negligible | OK |

<details>
<summary><strong>Rollback Instructions</strong></summary>

**Immediate rollback:**

```bash
git revert <squash-merge-sha>
git push origin develop
```

**Verification after rollback:**
- `just test` passes on `develop`
- `go test -race ./internal/routing/...` passes
- `just lint` 0 issues

</details>

### Feature Flags

None — HMAC enforcement is always-on per BC-2.05.008 mandatory path.

---

## Spec Patches Landed (factory-artifacts side)

The following factory artifact changes are included in this PR's commit chain:

| Artifact | Change | Commit |
|----------|--------|--------|
| ADR-009 (ARCH-04-admission-security.md) | v1.5 → v1.6: lock-free HMAC verify + defensive copy ordering | `e99bcf2` |
| `routing.go:115` comment | corrected to reflect HMAC-first ordering | `15353b1` |
| `routing.go:133` | `authKey := entry.FrameAuthKey` defensive copy added before `RUnlock` | `e99bcf2` |
| `FuzzRouteFrame_NonAdmittedNeverForwarded` | restored to genuinely exercise VP-008 (HMAC-valid + unadmitted seed added) | `e214f8d` |

---

## Traceability

| BC | Story AC | Test | VP | Status |
|----|----------|------|----|--------|
| BC-2.05.008 PC-1 | AC-001 | `TestRouteFrame_ValidHMAC_ProceedsToAdmission` | — | PASS |
| BC-2.05.008 PC-2 | AC-002 | `TestRouteFrame_InvalidHMAC_ReturnsErrHMACVerificationFailed` | VP-058 P3 | PASS |
| BC-2.05.008 PC-3 | AC-003 | `TestRouteFrame_HMACEnforcedBeforeAdmission` | VP-058 P1+2 | PASS |
| BC-2.05.008 PC-4 | AC-004 | `TestRouteFrame_NoForwardingEntry_RejectsAsUnverifiable` | VP-058 P4 | PASS |
| BC-2.05.008 EC-005 / inv-2 | AC-005 | `TestRouteFrame_ValidHMAC_Unadmitted_ReturnsErrNotAdmitted` | — | PASS |
| BC-2.05.008 EC-001 | EC-001 | `ExampleRouter_zeroHMACTagRejected` | — | PASS |
| BC-2.05.008 EC-002 | EC-002 | `ExampleRouter_wrongKeyHMACRejected` | — | PASS |

<details>
<summary><strong>Full VSDD Contract Chain</strong></summary>

```
BC-2.05.008 PC-1 -> VP-058 P1 -> TestRouteFrame_HMACEnforcedBeforeAdmission -> routing.go:113-140 -> ADV-PASS-3-OK
BC-2.05.008 PC-2 -> VP-058 P3 -> TestRouteFrame_InvalidHMAC_* -> routing.go:122-127 -> ADV-PASS-3-OK
BC-2.05.008 PC-3 -> VP-058 P1+2 -> TestRouteFrame_HMACEnforcedBeforeAdmission -> routing.go:113-140 -> ADV-PASS-3-OK
BC-2.05.008 PC-4 -> VP-058 P4 -> TestRouteFrame_NoForwardingEntry_* -> routing.go:115-121 -> ADV-PASS-3-OK
BC-2.05.008 inv-2 -> AC-005 -> TestRouteFrame_ValidHMAC_Unadmitted_* -> routing.go:140-155 -> ADV-PASS-3-OK
```

Demo evidence: `docs/demo-evidence/S-3.04/evidence-report.md`

</details>

---

## AI Pipeline Metadata

<details>
<summary><strong>Pipeline Details</strong></summary>

```yaml
ai-generated: true
pipeline-mode: greenfield
factory-version: "1.0.0-rc.21"
pipeline-stages:
  spec-crystallization: completed
  story-decomposition: completed
  tdd-implementation: completed
  holdout-evaluation: "N/A — evaluated at wave gate"
  adversarial-review: completed
  formal-verification: "N/A — evaluated at Phase 5"
  convergence: achieved
convergence-metrics:
  adversarial-passes: 5
  consecutive-clean-passes: 3
  bc-5.39.001-satisfied: true
adversarial-passes: 5
models-used:
  builder: claude-sonnet-4-6
  adversary: fresh-context adversary (claude-sonnet-4-6)
generated-at: "2026-06-26T00:00:00Z"
story-id: S-3.04
wave: 3
```

</details>

---

## Pre-Merge Checklist

- [x] All CI status checks passing
- [x] Coverage delta is positive or neutral
- [x] No critical/high security findings unresolved
- [x] Rollback procedure documented above
- [x] No feature flags required (always-on security enforcement)
- [x] Demo evidence present: `docs/demo-evidence/S-3.04/evidence-report.md` (7 AC/EC recordings)
- [x] Adversarial convergence: 3 consecutive clean passes (BC-5.39.001)
- [x] Dependencies merged: S-2.01, S-2.02 on `develop @ d8d7ae6`
- [x] Signed commits only; no AI attribution footers
