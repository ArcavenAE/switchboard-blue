# Evidence Report — S-7.01: XOR Parity FEC for Single-Loss Recovery

**Story:** S-7.01 v1.4
**BC trace:** BC-2.02.007 (XOR parity FEC, single-loss recovery)
**Convergence status:** BC-5.39.001 CLOSED — 3 consecutive clean fresh-context adversarial passes (Pass-6, Pass-7, Pass-8)
**Evidence date:** 2026-07-01

---

## Coverage Map

| AC | Acceptance Criterion | BC Trace | Test Name | Recording | Status |
|----|---------------------|----------|-----------|-----------|--------|
| AC-001 | `FEC.Encode` produces one parity frame per group; `frame_type=fec=0x05` | BC-2.02.007 PC-1 | `TestFEC_Encode_ProducesParityFrame` | [AC-001-parity-frame-emission.gif](AC-001-parity-frame-emission.gif) | PASS |
| AC-002 | `FEC.Recover` recovers missing frame via XOR when exactly one frame is absent | BC-2.02.007 PC-3 | `TestFEC_Recover_SingleLoss` | [AC-002-single-loss-recovery.gif](AC-002-single-loss-recovery.gif) | PASS |
| AC-003 | `FEC.Recover` returns `ErrTooManyLosses` when >1 frame missing | BC-2.02.007 PC-4 | `TestFEC_Recover_TwoLossesFail` | [AC-003-too-many-losses-error.gif](AC-003-too-many-losses-error.gif) | PASS |
| AC-004 | `ErrTooManyLosses` triggers ARQ retransmit; caller MUST NOT drop silently | BC-2.02.007 PC-4 + VP-043 | `TestFEC_FallbackToARQ_OnMultiLoss` | [AC-004-arq-fallback-on-multi-loss.gif](AC-004-arq-fallback-on-multi-loss.gif) | PASS |
| AC-005 | `FEC.Encode` does not emit parity for incomplete last group (session ends mid-group) | BC-2.02.007 EC-001 | `TestFEC_Encode_IncompleteLastGroup_NoParity` | [AC-005-incomplete-last-group-no-parity.gif](AC-005-incomplete-last-group-no-parity.gif) | PASS |

---

## Recordings

### AC-001: Parity Frame Emission

- **Tape:** [AC-001-parity-frame-emission.tape](AC-001-parity-frame-emission.tape)
- **GIF:** [AC-001-parity-frame-emission.gif](AC-001-parity-frame-emission.gif)
- **WebM:** [AC-001-parity-frame-emission.webm](AC-001-parity-frame-emission.webm)
- **Demonstrates:** `FEC.Encode` produces exactly one parity frame for a complete group of N=4 data frames with `frame_type=fec=0x05`

### AC-002: Single-Loss Recovery

- **Tape:** [AC-002-single-loss-recovery.tape](AC-002-single-loss-recovery.tape)
- **GIF:** [AC-002-single-loss-recovery.gif](AC-002-single-loss-recovery.gif)
- **WebM:** [AC-002-single-loss-recovery.webm](AC-002-single-loss-recovery.webm)
- **Demonstrates:** `FEC.Recover` correctly reconstructs each of the 4 possible single-loss positions (lossIdx=0,1,2,3) via XOR

### AC-003: Too-Many-Losses Error

- **Tape:** [AC-003-too-many-losses-error.tape](AC-003-too-many-losses-error.tape)
- **GIF:** [AC-003-too-many-losses-error.gif](AC-003-too-many-losses-error.gif)
- **WebM:** [AC-003-too-many-losses-error.webm](AC-003-too-many-losses-error.webm)
- **Demonstrates:** `FEC.Recover` returns `ErrTooManyLosses` for all 4 distinct 2-loss position pairs; decoder constructed fresh per subtest (no state leak)

### AC-004: ARQ Fallback on Multi-Loss

- **Tape:** [AC-004-arq-fallback-on-multi-loss.tape](AC-004-arq-fallback-on-multi-loss.tape)
- **GIF:** [AC-004-arq-fallback-on-multi-loss.gif](AC-004-arq-fallback-on-multi-loss.gif)
- **WebM:** [AC-004-arq-fallback-on-multi-loss.webm](AC-004-arq-fallback-on-multi-loss.webm)
- **Demonstrates:** Positive path (ErrTooManyLosses -> ARQ engaged, gaps non-empty, lost seqs present) AND negative path (nil err -> ARQ NOT invoked) per RULING-W6TB-E §H-1

### AC-005: Incomplete Last Group No Parity

- **Tape:** [AC-005-incomplete-last-group-no-parity.tape](AC-005-incomplete-last-group-no-parity.tape)
- **GIF:** [AC-005-incomplete-last-group-no-parity.gif](AC-005-incomplete-last-group-no-parity.gif)
- **WebM:** [AC-005-incomplete-last-group-no-parity.webm](AC-005-incomplete-last-group-no-parity.webm)
- **Demonstrates:** Groups of 1, 2, and 3 frames (all < N=4) produce zero parity frames; tested with subtests `one_of_four`, `two_of_four`, `three_of_four`

---

## Race Detector Evidence

- **File:** [race-test-transcript.txt](race-test-transcript.txt)
- **Command:** `go test -race -v ./internal/arq/...`
- **Result:** PASS — no data races detected; 35,000 VP-043 single-loss recovery assertions confirmed (7 group sizes × 35 positions × 1,000 trials)

---

## Property Test Coverage (VP-043)

`TestFEC_VP043_SingleLossRecovery_Property` executed 35,000 recovery assertions across group sizes 2–8 with runtime counter verification. The `count_verify` subtest asserts the exact count matches expected (35,000 recovery + 7,000 parity oracle checks).

---

## Summary

All 5 acceptance criteria are GREEN. Recordings produced with VHS 0.11.0 using Menlo font, Catppuccin Mocha theme. Race detector shows zero races across the full `internal/arq` package. Story S-7.01 is convergence-closed under BC-5.39.001.
