---
artifact_id: adv-S-1.01-pass-01
review_target: S-1.01-frame-codec
producer: adversary
pass: 1
fresh_context: true
findings_count: 7
findings_by_severity: {critical: 0, high: 1, medium: 4, low: 1, nitpick: 1}
verdict: NOT_CONVERGED
timestamp: 2026-06-24
---

# Adversary Pass 1 — S-1.01

## High

### F-001 — Fuzz harness is inert: 5 seeds built but never registered via f.Add()
- Location: `internal/frame/frame_test.go:303-406`
- Evidence: `seed1`..`seed5` builders exist, only `f.Add(make([]byte, 44))` is registered. Trailing comment "Add seeds after stubs are implemented" is stale.
- Impact: VP-001/002/003 coverage claim is hollow — fuzz only mutates from a single zeroed input.
- Route: test-writer
- Fix: Replace the trailing `_ = seed1..5` block with `f.Add(seed1()); f.Add(seed2()); f.Add(seed3()); f.Add(seed4()); f.Add(seed5()); f.Add(make([]byte, frame.OuterHeaderSize))`. Delete the stale comment.

## Medium

### F-002 — .golangci.yml SA4006 exclusion is project-wide scope creep masking dead test code
- Location: `.golangci.yml:36-46` + `internal/frame/address_test.go:66-70`
- Real trigger is `var zero [8]byte; _ = zero` (dead code in address_test, admittedly bypassed).
- Story scope is `internal/frame`; modifying project linter config exceeds the declared `architecture_modules`.
- Route: orchestrator-revert-config + implementer
- Fix: (1) revert .golangci.yml to develop state; (2) delete the 5 lines of dead code in address_test.go:66-70.

### F-003 — TestEncodeOuterHeader_ExactlyFortyFourBytes is tautological
- Location: `internal/frame/frame_test.go:15-41`
- `encoded` has type `[OuterHeaderSize]byte` — `len(encoded)` is a compile-time constant. The test cannot fail.
- BC-2.01.004's canonical vector specifies byte offsets (payload_len=256 → bytes 2-3 = 0x01,0x00 big-endian). Test doesn't assert offsets.
- Route: test-writer
- Fix: Replace tautological len-check with byte-level offset assertions from BC-2.01.004 canonical vector.

### F-004 — Round-trip alone doesn't verify big-endian byte order; little-endian regression would silently pass
- Location: entire frame_test.go
- BC-2.01.004 + ARCH-02 specify big-endian for payload_len, but no test asserts the wire-level byte order independently.
- A regression to `binary.LittleEndian.PutUint16/Uint16` would round-trip identically but fail interop with spec-compliant peers.
- Route: test-writer (combine with F-003)
- Fix: Add a test that encodes PayloadLen=256 and asserts `encoded[2]==0x01 && encoded[3]==0x00`.

### F-005 — Wrapped-error contract has no test
- Location: `internal/frame/frame_test.go:192-215, 221-254`
- Implementation uses `fmt.Errorf("...: %w", ErrFrameTooShort)`. Tests verify `errors.Is(err, ErrFrameTooShort)` but a regression to bare-sentinel return would still pass.
- Route: test-writer
- Fix: Add `if err == frame.ErrFrameTooShort { t.Errorf("expected wrapped error, got bare sentinel") }` to both rejection tests.

## Low

### F-006 — address_test.go:66-70 is dead code (root cause of F-002)
- Location: `internal/frame/address_test.go:66-70`
- `var zero [8]byte; _ = zero` admittedly bypassed by `assertSHA256Address` per its own comment.
- Route: implementer
- Fix: Delete lines 66-70.

## Nitpick

### F-007 — ParseOuterHeader doc comment is a comma-splice
- Location: `internal/frame/frame.go:81-83`
- Cosmetic Go doccomment grammar.
- Route: implementer
- Fix: insert "or" before "ErrVersionMismatch".

## Verdict

**NOT_CONVERGED.** F-001 (inert fuzz) and F-002 (config scope creep) must be addressed.
