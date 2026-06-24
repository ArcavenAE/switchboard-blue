---
artifact_id: red-gate-log-S-1.01
story_id: S-1.01
cycle: cycle-1
producer: orchestrator
timestamp: 2026-06-24T09:25:00Z
phase: 3
inputDocuments:
  - '.factory/stories/S-1.01-frame-codec.md'
  - '.factory/specs/behavioral-contracts/ss-01/BC-2.01.004.md'
  - '.factory/specs/behavioral-contracts/ss-01/BC-2.01.005.md'
  - '.factory/specs/behavioral-contracts/ss-01/BC-2.01.006.md'
---

# Red Gate Verification Log — S-1.01

## Outcome

**PASSED** — verified independently by orchestrator after test-writer dispatch.

## Step 2 — Stub Architect

**Agent:** vsdd-factory:stub-architect
**Commit:** `07a0827` (SSH-signed, branch `feature/S-1.01-frame-codec`)

Files created:
- `internal/frame/frame.go` — OuterHeader struct (Version, FrameType, PayloadLen, SVTNID, SrcAddr, DstAddr, HMACTag), constants (OuterHeaderSize=44, VersionByte=0x01, frame_type enum), sentinel errors (ErrFrameTooShort, ErrVersionMismatch), 2 stub functions (EncodeOuterHeader, ParseOuterHeader) — both panic("not implemented: S-1.01 ...")
- `internal/frame/address.go` — 1 stub function (DeriveNodeAddress) — panics

Verification:
- `go build ./...` exit 0
- `go vet ./...` exit 0
- 3 total stub functions; all panic on call

Anti-precedent guard (deliver-story SKILL): not applicable — that guard refers to Rust sibling crates from another project (Prism commits). Switchboard is Go; the equivalent guard ("don't pre-implement the body") was honored — only struct/const/error declarations exist, no business logic.

## Step 3 — Test Writer

**Agent:** vsdd-factory:test-writer
**Commit:** `52b1db5` (SSH-signed)

Files created:
- `internal/frame/frame_test.go` — 5 unit tests + 1 fuzz harness
- `internal/frame/address_test.go` — 3 unit tests

Test-to-AC traceability:

| AC | Test Function | BC Reference |
|---|---|---|
| AC-001 | TestEncodeOuterHeader_ExactlyFortyFourBytes | BC-2.01.004 postcondition 1 |
| AC-002 | TestParseEncodeRoundTrip (10 table cases) | BC-2.01.004 postcondition 2 |
| AC-003 | TestParseOuterHeader_TooShort (4 length cases) | BC-2.01.004 precondition 1 (ErrFrameTooShort) |
| AC-004 | TestParseOuterHeader_VersionMismatch (major=1, 15; minor-only mismatch passes) | BC-2.01.004 precondition 2 (ErrVersionMismatch) |
| AC-005 | TestChannelHeaderOpaque_NotInOuterHeader (reflection-based) | BC-2.01.005 invariant 1 |
| AC-006 | TestDeriveNodeAddress_Deterministic (5 pairs) + TestDeriveNodeAddress_OutputIsEightBytes + TestDeriveNodeAddress_DifferentSVTNYieldsDifferentAddress | BC-2.01.006 postcondition 1 |
| VP stretch | FuzzEncodeParseRoundTrip (structural seed) | VP-001/002/003 |

## Red Gate Verification (orchestrator, independent)

Command: `go test ./internal/frame/... -count=1`
Exit code: **1** (FAIL — expected)

Failure mode: every behavioral test panics with the stub panic message (`panic: not implemented: S-1.01 EncodeOuterHeader` / `DeriveNodeAddress` / `ParseOuterHeader`). The Go runtime captures the panic via testing's recover and surfaces it as a FAIL. This satisfies the Red Gate requirement:

- ✅ Tests compile (go build clean)
- ✅ All behavioral tests fail
- ✅ Tests fail with panics (Go idiom for unimplemented body — equivalent to Rust's `todo!()`)
- ✅ Failure messages reference the behavior under test (`S-1.01 EncodeOuterHeader` etc.)

**Exception:** `TestChannelHeaderOpaque_NotInOuterHeader` passes against the stubs. This is a structural reflection-based test that verifies the `OuterHeader` struct has exactly the 7 expected fields (Version, FrameType, PayloadLen, SVTNID, SrcAddr, DstAddr, HMACTag) and no channel-header fields. The struct was correctly shaped by stub-architect at commit `07a0827`, so the test passes — and would catch a future regression where channel-header fields leaked in. This is acceptable Red Gate behavior: the AC tests struct shape (a static property), not runtime behavior. The remaining 7 behavioral tests fail as required.

## Authorization

Red Gate verified. Implementer may proceed to Step 4 (TDD implementation).
