---
artifact_id: adv-refactor-frametype-mtu-pass-01
review_target: refactor-frametype-mtu (F-001+F-002 combined PR)
producer: adversary
pass: 1
fresh_context: true
branch: feature/refactor-frametype-mtu
base: develop @ 9e9a98a
tip: 4f4c07c
findings_count: 0
findings_by_severity: {critical: 0, high: 0, medium: 0, low: 0, nitpick: 0}
verdict: CONVERGED
timestamp: 2026-06-24
---

# Adversary Pass 1 — Refactor F-001+F-002

## Verdict: CONVERGED — Zero Findings

Fresh-context re-derivation of the refactor diff found no defects.

## Axes Checked Clean

### A. Refactor scope completeness
- F-002 closure verified: `type FrameType byte` declared; 5 constants typed; `Valid()` method present; `OuterHeader.FrameType` + `ChannelFrame.FrameType` use named type; `ParseOuterHeader` rejects invalid via `ErrInvalidFrameType` with `%w` wrap.
- F-001 closure verified: `MaxPayloadSize = 65515` constant + `ErrPayloadTooLarge` sentinel + Enqueue MTU check after empty check; boundary tests at MaxPayloadSize and MaxPayloadSize+1.
- EncodeOuterHeader leniency intentional and documented; parse-only validation is defensible.

### B. Spec alignment
- BC-2.01.002 v1.4 PC5: implementer collapsed to conservative single constant (65515 SACK=1 worst-case). BC explicitly identifies this as a valid bound. Stricter, not looser. Wire-safe.
- ARCH-02 §3.1 frame-type enum: all 5 canonical values match; Valid() correctly rejects 0x00 and 0x06..0xFF.

### C. Test quality
- All 6 new tests use errors.Is for sentinel-identity assertions.
- Table-driven where >2 cases; t.Parallel() consistent.
- Boundary tests cover both directions (exact MaxPayloadSize accepted; +1 rejected).

### D. Regression risk
- Single mechanical byte() cast in frame_test.go:302 (raw wire buffer construction).
- example_test.go formatting still pinned correctly; benchmark unaffected.

### E. ParseOuterHeader validation ordering
- short-buffer → version → frame_type → others. Sensible precedence; debug message includes offending byte.

### F. Enqueue validation ordering
- empty check (BC PC4) → oversize check (BC PC5). Matches BC numerical order.
- No allocation on the oversize path; OOM concern structurally absent.

### G. Type-system rigor
- Every FrameType site compile-time checked; byte() conversions only at wire-format narrowing/widening boundaries.

### H. ARCH-09 purity
- frame imports only stdlib; halfchannel imports stdlib + frame. No goroutines, no time.Now in production source.

### I. Project rules (.claude/rules/go.md)
- Rules 4, 5, 9, 10, 11 all spot-checked clean. No new init/interface/panics/log.Fatal.

### J. Cross-module FrameType propagation
- halfchannel.FrameTypeData = frame.FrameTypeData (alias inherits named type).
- ChannelFrame.FrameType is frame.FrameType (typed across module boundary).

### K. Process / auditability
- 3 commits in the branch (tests / frame impl / halfchannel impl). git blame traces each substantive change to a clear commit.

## Notable for PR description

`MaxPayloadSize = 65515` is the SACK=1 conservative bound. BC-2.01.002 v1.4 PC5 also documents 65523 as the SACK=0 bound; the implementation deliberately collapses to the stricter single constant, defended in the godoc at halfchannel.go:72-77. Should the outer-assembler ever need SACK-aware sizing, that's a future story.

## Convergence streak: 1/3
Need passes 2 and 3 also clean for BC-5.39.001 closure.
