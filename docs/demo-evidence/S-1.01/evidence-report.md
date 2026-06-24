---
artifact_id: evidence-report-S-1.01
story_id: S-1.01
producer: demo-recorder
timestamp: 2026-06-24T00:00:00Z
phase: 3
---

# S-1.01 Demo Evidence Report

## AC-to-Artifact Mapping

| AC | Test name | Evidence file | Status |
|---|---|---|---|
| AC-001 | TestEncodeOuterHeader_ExactlyFortyFourBytes | AC-001-TestEncodeOuterHeader_ExactlyFortyFourBytes.log | PASS |
| AC-001 | TestEncodeOuterHeader_WireFormatByteOffsets | AC-001b-TestEncodeOuterHeader_WireFormatByteOffsets.log | PASS |
| AC-002 | TestParseEncodeRoundTrip (11 subtests) | AC-002-TestParseEncodeRoundTrip.log | PASS |
| AC-003 | TestParseOuterHeader_TooShort (4 sizes) | AC-003-TestParseOuterHeader_TooShort.log | PASS |
| AC-004 | TestParseOuterHeader_VersionMismatch (3 subtests) | AC-004-TestParseOuterHeader_VersionMismatch.log | PASS |
| AC-005 | TestChannelHeaderOpaque_NotInOuterHeader | AC-005-TestChannelHeaderOpaque_NotInOuterHeader.log | PASS |
| AC-006 | TestDeriveNodeAddress_Deterministic (5 cases) | AC-006-TestDeriveNodeAddress_Deterministic.log | PASS |
| AC-006 supporting | TestDeriveNodeAddress_DifferentSVTNYieldsDifferentAddress | AC-006b-TestDeriveNodeAddress_DifferentSVTNYieldsDifferentAddress.log | PASS |
| AC-006 supporting | TestDeriveNodeAddress_DifferentPubkeyYieldsDifferentAddress | AC-006c-TestDeriveNodeAddress_DifferentPubkeyYieldsDifferentAddress.log | PASS |
| AC-006 supporting | TestDeriveNodeAddress_ReturnsExpectedSHA256Prefix | AC-006d-TestDeriveNodeAddress_ReturnsExpectedSHA256Prefix.log | PASS |

## Quality gates

- Race detector: race-detector.log (PASS — exit 0, no data races)
- Fuzz harness (30s): fuzz-30s.log (no crash, corpus exercised)
- golangci-lint: lint.log (0 issues)
- gofumpt: gofumpt.log (empty diff — no formatting violations)

## End-to-end demo

Example test `Example_encodeParseRoundTrip` in `internal/frame/example_test.go`
exercises the public API end-to-end using the BC-2.01.004 canonical test vectors:

- Encodes a data frame with Version=0x01, FrameType=0x01, PayloadLen=256
- Parses it back and verifies round-trip identity (decoded == h)
- Demonstrates E-PRT-002: `ParseOuterHeader(make([]byte, 30))` returns `ErrFrameTooShort`
- Demonstrates E-PRT-001: version byte 0x20 (major=2) returns `ErrVersionMismatch`
- Derives node address twice from identical inputs: both calls return `c341fbfa8e968c5f`
- Derives address for different pubkey (last byte 0xBF vs 0xBE): returns `a2fbf85017cc4999`

Output captured in demo-output.txt. The `// Output:` assertion is verified by
`go test` — a wrong output line causes test failure.

## Provenance

- Branch: feature/S-1.01-frame-codec
- HEAD SHA: dbc90dc3f26f660df5a524e0c06116c4f06079d8
- Go version: go1.26.2 darwin/arm64
- Adversary convergence: BC-5.39.001 satisfied — 3 clean consecutive passes (passes 6, 7, 8)
- Convergence state: `.factory/cycles/cycle-1/S-1.01/adversary-convergence-state.json`

## Source artifacts

- Story spec: .factory/stories/S-1.01-frame-codec.md
- BCs realized: BC-2.01.004, BC-2.01.005, BC-2.01.006
- VPs covered: VP-001, VP-002, VP-003, VP-014 (VP-015 deferred to S-2.01 routing scope)
