---
artifact_id: adv-refactor-frametype-mtu-pass-03
review_target: refactor-frametype-mtu (F-001+F-002 combined PR)
producer: adversary
pass: 3
fresh_context: true
branch: feature/refactor-frametype-mtu
base: develop @ 9e9a98a
tip: 4f4c07c
findings_count: 0
findings_by_severity: {critical: 0, high: 0, medium: 0, low: 0, nitpick: 0}
verdict: CONVERGED
timestamp: 2026-06-24
---

# Adversary Pass 3 — Refactor F-001+F-002 (FINAL)

## Verdict: CONVERGED — Zero Findings

Third consecutive clean pass. **Convergence streak 3/3 — BC-5.39.001 SATISFIED.**

## Axes Checked Clean

### F-001 closure (MTU validation in Enqueue) — verified
- halfchannel.go:77 defines MaxPayloadSize = 65535 - 20 = 65515, matching BC-2.01.002 v1.4 PC5 conservative bound (SACK_present=1).
- halfchannel.go:84 declares ErrPayloadTooLarge sentinel.
- halfchannel.go:143-149 Enqueue validates empty-first, oversize-second, wraps via %w (rule go.md #4).
- Tests at halfchannel_test.go:586-595 (constant pin), :603-614 (Max+1 → ErrPayloadTooLarge with errors.Is), :621-632 (Max accept) cover the boundary on both sides.

### F-002 closure (typed FrameType + ParseOuterHeader enum validation) — verified
- frame.go:29 defines `type FrameType byte`.
- frame.go:33-38 types all five constants as FrameType.
- frame.go:42-44 defines (FrameType).Valid() bool with correct boundary 0x01..0x05.
- frame.go:49 declares ErrInvalidFrameType sentinel.
- frame.go:69 OuterHeader.FrameType is FrameType (not byte).
- frame.go:111-114 ParseOuterHeader validates after length+version checks, wraps the sentinel with %w.
- halfchannel.go:67 ChannelFrame.FrameType frame.FrameType propagates the named type across the module boundary (ARCH-08 import direction preserved: halfchannel → frame, no cycles).
- halfchannel.go:19-22 keeps the constant aliases typed via the source-of-truth references.

### Spec alignment — verified
- BC-2.01.002 v1.4 PC5 wording aligns with halfchannel.go:72-77 (conservative bound when SACK=1).
- BC-2.01.004 §postcondition 2 frame_type enum (0x01..0x05) matches FrameType.Valid() range.
- ARCH-02 §"Outer Header Format" frame_type enum matches.

### Test quality — verified
- ParseOuterHeader validation ordering tested in isolation.
- All error inspections use errors.Is.
- Sentinel-wrapping contracts asserted; distinguishes wrapped vs bare sentinel.
- TestFrameType_Valid boundary table covers {0x00, 0x06, 0xFF} + the five canonical values.
- TestParseOuterHeader_AcceptsAllValidFrameTypes is positive-coverage assertion against over-strict validator.

### Project rules (go.md) — verified
- No init(). No log.Fatal/os.Exit. No panics in library code.
- All exported errors are package-level var with godoc.
- Error strings lowercase, no trailing punctuation (ST1005).
- Error wrapping uses %w consistently (4/4 fmt.Errorf sites).
- time.Now/time.Sleep absent from production code (ARCH-09 pure-core preserved).

### Minor process observations (non-blocking)
- EncodeOuterHeader does NOT validate h.FrameType.Valid() before encoding. Wave-1 F-002 explicitly framed the fix as parse-side validation; encode-side silence is per spec. The named-type fence still catches byte→byte accidental assignments at compile time. Not a finding.
- FuzzEncodeParseRoundTrip returns early on err != nil when major == 0 without asserting the error class. Fuzz is not contracted to validate error classes. Not a finding.

## Convergence

**Streak 3/3 → BC-5.39.001 SATISFIED.** Refactor PR is ready for the PR lifecycle.

Trajectory across 3 passes: 0 → 0 → 0.
