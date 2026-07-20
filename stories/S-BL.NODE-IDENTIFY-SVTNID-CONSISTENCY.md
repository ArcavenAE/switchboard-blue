---
artifact_id: S-BL.NODE-IDENTIFY-SVTNID-CONSISTENCY
document_type: story
level: ops
story_id: S-BL.NODE-IDENTIFY-SVTNID-CONSISTENCY
version: "1.0"
title: "NODE_IDENTIFY hardening: ChallengeResponse SVTNID-consistency enforcement (BC-2.01.009 PC-9)"
status: ready
producer: story-writer
timestamp: 2026-07-19T00:00:00Z
modified: 2026-07-19T00:00:00Z
phase: 2
epic: E-7
wave: backlog
priority: P2
scope_phase: E
estimated_points: 3
points: 3
inputs:
  - 'specs/behavioral-contracts/ss-01/BC-2.01.009.md'
  - 'specs/prd-supplements/error-taxonomy.md'
input-hash: "1f94fc2"
traces_to: "specs/behavioral-contracts/ss-01/BC-2.01.009.md"
epic_id: E-7
behavioral_contracts:
  - BC-2.01.009
bc_traces:
  - BC-2.01.009
verification_properties: []
subsystems: [session-networking, admission-security]
target_module: "cmd/switchboard"
architecture_modules:
  - cmd/switchboard
tdd_mode: strict
cycle: v1.0.0-greenfield
estimated_days: null
assumption_validations: []
risk_mitigations: []
depends_on: [S-BL.NODE-IDENTIFY-WIRE]
blocks: []
acceptance_criteria_count: 3
provenance:
  origin: "Post-merge security review of PR #127 (S-BL.NODE-IDENTIFY-WIRE); drift item SEC-NIDW-SVTNID-CONSISTENCY (MED); BC-2.01.009 v1.4/v1.5 PC-9 authored 2026-07-19"
  spec_annotation: "BC-2.01.009 v1.5 PC-9 — ChallengeResponse SVTNID-consistency enforcement postcondition"
---

# S-BL.NODE-IDENTIFY-SVTNID-CONSISTENCY: ChallengeResponse SVTNID-Consistency Guard (BC-2.01.009 PC-9)

> **STATUS: READY FOR IMPLEMENTATION.** Prerequisite story `S-BL.NODE-IDENTIFY-WIRE`
> delivered via PR #127 (@ 7fcf0cf). 3 ACs, 3 points, TDD strict.

> **SCOPE NOTE:** This story adds a single, bounded security guard to the existing
> `nodeIdentifyHandshake` function in `cmd/switchboard/node_identify_wire.go`. It
> does NOT add any new infrastructure, new config fields, or changes to
> `internal/admission`. It does NOT address the two LOW findings from the
> post-merge security review (ReadOuterFrame prealloc, per-IP rate limit).

## Narrative

- **As a** router-mode daemon operator
- **I want** the `nodeIdentifyHandshake` driver to reject a `ChallengeResponse`
  whose outer-header `svtn_id` does not match the `svtn_id` from the original
  `NodeIdentify` frame, before calling `admission.AdmitNode`
- **So that** a mismatched-SVTNID `ChallengeResponse` cannot be silently accepted
  and a cross-SVTN credential substitution attack is blocked at the protocol level

## Context

`S-BL.NODE-IDENTIFY-WIRE` (PR #127, 2026-07-19) delivered the full three-message
NODE_IDENTIFY handshake. The post-merge security review identified drift item
`SEC-NIDW-SVTNID-CONSISTENCY (MED)`: the implementation reads the ChallengeResponse
outer header but does not verify that its `svtn_id` matches the `svtn_id` from the
NodeIdentify (message 1). BC-2.01.009 PC-4 states that the ChallengeResponse outer
header has `svtn_id unchanged`, and PC-9 (v1.4/v1.5) makes this an explicit
enforcement postcondition with its own error code (E-ADM-024).

The guard belongs in `nodeIdentifyHandshake` (`cmd/switchboard/node_identify_wire.go`)
immediately after decoding the ChallengeResponse outer header and BEFORE the
`admission.AdmitNode` call — matching the existing fail-closed pattern for all other
pre-admission checks in that function.

## BC Anchors

| BC | Why anchored |
|----|-------------|
| BC-2.01.009 | PC-9 is the direct source for this guard: "Before calling AdmitNode, the router MUST verify that the ChallengeResponse (message 3) outer-header svtn_id equals the svtn_id from the NodeIdentify (message 1) outer header; mismatch closes the connection (E-ADM-024)." EC-008 documents the edge case. The E-ADM-024 row in error-taxonomy v5.2 provides the canonical string and WARN-log placement. |

## Previous Story Intelligence (MANDATORY)

| Predecessor | Key Decisions | Patterns Established | Lessons Carried Forward |
|-------------|--------------|---------------------|------------------------|
| `S-BL.NODE-IDENTIFY-WIRE` (PR #127 @ 7fcf0cf) | Three-message handshake driver in `cmd/switchboard/node_identify_wire.go`; `nodeIdentifyHandshake` function signature `(conn net.Conn, r *routing.Router, routerPrivKey ed25519.PrivateKey, ks *admission.AdmittedKeySet, h netingress.NodeHandle) (svtnID [16]byte, nodeAddr [8]byte, err error)`; outer SVTNID is read from `NodeIdentify` (message 1) and captured as `hdr.SVTNID` / local `svtnID`; that value is echoed in Challenge (message 2) and must be unchanged in ChallengeResponse (message 3). | `onAccept` classification switch pattern in `mgmt_wire.go`; fail-closed on ANY pre-AdmitNode error (close conn + return error immediately); WARN log pattern for E-ADM-* codes (e.g. E-ADM-003, E-ADM-022, E-ADM-023) logged before or alongside the close; F-1 adjudication: AdmitNode verifies against the STORED registered key, not the frame-supplied pubKey; conn.SetDeadline cleared only on full success. | The `svtnID` variable captured from the NodeIdentify outer header (`hdr.SVTNID`) is the authoritative reference for all subsequent SVTNID comparisons in the same handshake. The ChallengeResponse outer header is read into its own `crHdr` struct — compare `crHdr.SVTNID` against the captured `svtnID` immediately after the read, before any downstream call. The error path must close `conn` and return a non-nil error (same as every other failure path in this function) so that `onAccept` receives the error and returns a no-op cleanup without calling `sendMap.Store` or `r.BindInterface`. |

## Adjudicated Design Decisions

### Decision 1 — Guard placement: after ChallengeResponse outer-header read, before AdmitNode

The guard is a two-line check inserted at one point in `nodeIdentifyHandshake`:

```go
// After: io.ReadFull(conn, crOuterBuf) + decoding crHdr
// Before: admission.AdmitNode(...)
if crHdr.SVTNID != svtnID {
    _ = conn.Close()
    return [16]byte{}, [8]byte{}, fmt.Errorf("node_identify: ChallengeResponse svtn_id mismatch")
}
```

The WARN log containing E-ADM-024 is emitted by the `onAccept` caller when it
receives this error (consistent with how `onAccept` handles the other E-ADM-* errors
returned from `nodeIdentifyHandshake` — the logging happens at the `onAccept`
dispatch switch, not inside the handshake driver itself). Alternatively, the WARN
may be emitted directly in `nodeIdentifyHandshake` before returning, mirroring how
E-ADM-022 is logged. The implementer follows whichever pattern exists in the PR #127
delivery for other pre-AdmitNode failure paths. The canonical string is
`"node_identify: ChallengeResponse svtn_id mismatch"` — it MUST be byte-identical
to the error-taxonomy v5.2 E-ADM-024 row.

### Decision 2 — No AdmitNode call on mismatch (discriminating property)

The guard short-circuits BEFORE `admission.AdmitNode` is reached. This means:
- No keyset lookup occurs.
- No nonce is consumed.
- A key that _would_ otherwise be admitted is NOT admitted if the SVTNID in the
  ChallengeResponse outer header was tampered with.
- AC-002 tests this discriminating property explicitly: use an admitted key (so
  AdmitNode _would_ return nil) but send a mismatched SVTNID in the ChallengeResponse;
  assert the connection closes AND that AdmitNode was NOT reached.

## Acceptance Criteria

### AC-001 — Matching SVTNID in ChallengeResponse outer header proceeds to AdmitNode (no regression) (traces to BC-2.01.009 PC-9 success branch)

**BC Anchor:** BC-2.01.009 Postcondition 9 (success branch: svtn_id identical → proceed to AdmitNode); BC-2.01.009 PC-5 (AdmitNode call).

**Postconditions:**
1. When `crHdr.SVTNID == svtnID` (the SVTNID from the NodeIdentify outer header),
   the handshake driver does NOT close the connection on account of the SVTNID check.
2. Execution proceeds past the guard to `admission.AdmitNode`. For an admitted key
   with a valid NonceSig, AdmitNode returns nil and the full handshake completes
   successfully (BindInterface called, ServeConn begins).
3. This is a regression guard: the PR #127 happy-path behavior is preserved
   byte-for-byte on the matching-SVTNID path.

**Test name:**
- `TestNodeIdentifyHandshake_CRSVTNIDMatch_ProceedsToAdmitNode` — extend or
  verify an existing PR #127 success-path test: send a ChallengeResponse with
  `svtn_id == NodeIdentify.svtn_id`; assert handshake completes, binding is
  recorded, no connection closure from the guard.

---

### AC-002 — Mismatched SVTNID in ChallengeResponse outer header → connection closed before AdmitNode (traces to BC-2.01.009 PC-9 / EC-008)

**BC Anchor:** BC-2.01.009 Postcondition 9 (mismatch → close connection, return E-ADM-024); BC-2.01.009 EC-008.

**Postconditions:**
1. When `crHdr.SVTNID != svtnID`, the handshake driver closes the connection
   immediately and returns a non-nil error.
2. `admission.AdmitNode` is NOT called. No nonce is consumed. No binding is
   recorded (`Router.BindInterface` is not called).
3. **Discriminating constraint:** the admitted keyset MUST contain the connecting
   node's public key (i.e., AdmitNode _would_ have returned nil on the matching path)
   so the test proves the guard fires BEFORE AdmitNode, not because AdmitNode
   rejected the key.
4. The returned error carries the canonical text
   `"node_identify: ChallengeResponse svtn_id mismatch"` (substring match
   sufficient; must be byte-identical to error-taxonomy v5.2 E-ADM-024).
5. After the close, `LookupInterface(svtnID, nodeAddr)` returns `(0, false)` —
   no binding was recorded.

**Test name:**
- `TestNodeIdentifyHandshake_CRSVTNIDMismatch_ConnectionClosed_BeforeAdmitNode`
  — construct a valid handshake except mutate the ChallengeResponse outer-header
  `svtn_id` to any value different from the NodeIdentify `svtn_id`; use an
  admitted keyset that would otherwise grant admission; assert: (a) connection is
  closed, (b) the error message contains the canonical E-ADM-024 string, (c) no
  binding exists in the router's identity map.

---

### AC-003 — Daemon WARN log for the mismatch path contains the E-ADM-024 canonical string (traces to BC-2.01.009 EC-008; error-taxonomy v5.2 E-ADM-024)

**BC Anchor:** BC-2.01.009 EC-008 (expected behavior: "Connection closed with E-ADM-024 before AdmitNode is called"); error-taxonomy v5.2 E-ADM-024 (canonical string: `"node_identify: ChallengeResponse svtn_id mismatch"`, WARN level).

**Postconditions:**
1. The WARN log emitted on the mismatch path (whether from inside
   `nodeIdentifyHandshake` or from the `onAccept` caller, consistent with how
   PR #127 handles other E-ADM-* paths) contains the substring
   `"node_identify: ChallengeResponse svtn_id mismatch"` (byte-identical to
   error-taxonomy v5.2 E-ADM-024 canonical string) and/or the substring
   `"E-ADM-024"`.
2. The log is at WARN severity (not ERROR or INFO) — matching E-ADM-022 and
   E-ADM-023 log-level precedent.
3. The log includes the SVTNID context from the NodeIdentify outer header (the
   expected value) to aid operator diagnosis — following the `svtn={svtnID}`
   pattern from E-ADM-022 or equivalent per the PR #127 log format convention.

**Test name:**
- `TestNodeIdentifyHandshake_CRSVTNIDMismatch_WarnLogContainsE_ADM_024`
  — capture log output during the mismatch scenario (AC-002 setup); assert the
  captured log contains `"node_identify: ChallengeResponse svtn_id mismatch"`
  or `"E-ADM-024"` at WARN level.

---

## Architecture Mapping

| Component | Module | Pure/Effectful | Justification |
|-----------|--------|---------------|---------------|
| SVTNID-consistency guard (2-line check) | `cmd/switchboard/node_identify_wire.go` | effectful-shell | Inserted into the existing effectful `nodeIdentifyHandshake` function; reads `crHdr.SVTNID` (already decoded from the live TCP connection); closes `conn` on mismatch — TCP I/O |
| WARN log emission | `cmd/switchboard/node_identify_wire.go` or `cmd/switchboard/mgmt_wire.go` | effectful-shell | Log I/O; follows existing E-ADM-* log placement in `onAccept` switch |

No changes to `internal/admission`, `internal/routing`, or any other package.
No new imports required beyond what `nodeIdentifyHandshake` already uses.

## Non-Goals

- **ReadOuterFrame prealloc (LOW finding)** — separate drift item; not in scope.
- **Per-IP rate limiting (LOW finding)** — separate drift item; not in scope.
- **Changes to internal/admission** — BC-2.01.009 PC-9 is enforced at the wire layer before AdmitNode; no admission package edits needed.
- **Changes to internal/routing** — no new bind/unbind behavior; the guard prevents reaching BindInterface on the mismatch path.
- **New config fields or management RPCs** — none needed.

## Edge Cases

| ID | Scenario | Expected Behavior |
|----|----------|-------------------|
| EC-008 (from BC-2.01.009) | ChallengeResponse outer-header `svtn_id` ≠ NodeIdentify outer-header `svtn_id` | Connection closed with E-ADM-024 before AdmitNode is called. No admission attempted. Covered by AC-002 and AC-003. |
| (regression) | ChallengeResponse outer-header `svtn_id` == NodeIdentify outer-header `svtn_id` (all bytes identical) | Guard does NOT fire. Handshake proceeds to AdmitNode as in PR #127. Covered by AC-001. |
| (edge) | ChallengeResponse `svtn_id` is all-zero bytes (different from a non-zero NodeIdentify `svtn_id`) | Guard fires (all-zero != non-zero); closed with E-ADM-024. The earlier zero-SVTNID check (BC-2.01.009 PC-5 / AC-003 of parent story) already rejected NodeIdentify with all-zero SVTNID, so this case can only arise if the ChallengeResponse has all-zero SVTNID while the NodeIdentify had a valid non-zero one. Same treatment: mismatch → E-ADM-024. |

## Token Budget Estimate (MANDATORY)

| Context Source | Estimated Tokens |
|---------------|-----------------|
| This story spec | ~400 |
| `cmd/switchboard/node_identify_wire.go` (nodeIdentifyHandshake function — the ChallengeResponse decode + AdmitNode call region, ~50 relevant lines) | ~80 |
| `cmd/switchboard/mgmt_wire.go` (onAccept E-ADM logging pattern, ~20 relevant lines) | ~40 |
| BC-2.01.009 (PC-9 section + EC-008 + E-ADM-024 error table row, ~30 lines) | ~60 |
| error-taxonomy.md (E-ADM-024 row, ~10 lines) | ~25 |
| Existing test file `cmd/switchboard/node_identify_wire_test.go` (reference for test patterns, ~50 relevant lines) | ~80 |
| New test functions (3 ACs, ~3-5 test functions, ~60 lines) | ~120 |
| Tool outputs overhead | ~80 |
| **Total** | **~885 tokens — well under 20% of agent context window** |

## Tasks (MANDATORY)

Red Gate discipline: all test functions must be written FIRST (test-writer step)
and FAIL before any implementation code is written (implementer step).

1. [ ] Write failing test for AC-001: `TestNodeIdentifyHandshake_CRSVTNIDMatch_ProceedsToAdmitNode` — confirm matching SVTNID in ChallengeResponse outer header does not trigger the new guard; handshake completes; binding recorded. (May be satisfied by an existing PR #127 success-path test if it already validates the full matching path end-to-end; the test-writer verifies and adds if absent.) — test-writer
2. [ ] Write failing test for AC-002: `TestNodeIdentifyHandshake_CRSVTNIDMismatch_ConnectionClosed_BeforeAdmitNode` — construct a valid handshake except mutate the ChallengeResponse outer-header `svtn_id`; use an admitted keyset that would grant admission on the matching path; assert: (a) connection closed, (b) error message contains canonical E-ADM-024 string, (c) `LookupInterface` returns `(0, false)` — AdmitNode was NOT reached. — test-writer
3. [ ] Write failing test for AC-003: `TestNodeIdentifyHandshake_CRSVTNIDMismatch_WarnLogContainsE_ADM_024` — capture WARN log on the mismatch path; assert log contains `"node_identify: ChallengeResponse svtn_id mismatch"` or `"E-ADM-024"`. — test-writer
4. [ ] Verify Red Gate: `go test ./cmd/switchboard/... -run TestNodeIdentifyHandshake_CRSVTNID` fails with compile errors or test failures for all 3 new AC tests — implementer
5. [ ] Add the SVTNID-consistency guard in `nodeIdentifyHandshake` (`cmd/switchboard/node_identify_wire.go`): after reading and decoding the ChallengeResponse outer header (`crHdr`), immediately before the `admission.AdmitNode(...)` call, insert: `if crHdr.SVTNID != svtnID { _ = conn.Close(); return [16]byte{}, [8]byte{}, fmt.Errorf("node_identify: ChallengeResponse svtn_id mismatch") }`. Emit a WARN log consistent with the E-ADM-022/E-ADM-023 log pattern in `onAccept` (either from inside `nodeIdentifyHandshake` before returning, or from the `onAccept` error dispatch switch after receiving the error — follow the existing PR #127 convention). Canonical string MUST be byte-identical to error-taxonomy v5.2 E-ADM-024: `"node_identify: ChallengeResponse svtn_id mismatch"`. — implementer [BC-2.01.009 PC-9; error-taxonomy v5.2 E-ADM-024]
6. [ ] Run `go test ./cmd/switchboard/... -race`; confirm all 3 new AC test functions pass and no regressions in existing PR #127 tests
7. [ ] Update STATE.md (state-manager)

## Architecture Compliance Rules (MANDATORY)

| Rule | Requirement | Enforcement |
|------|-------------|-------------|
| ARCH-08 §Import DAG — `cmd/switchboard` position 18 | The guard adds no new imports; uses only the `svtnID` and `crHdr.SVTNID` values already in scope within `nodeIdentifyHandshake`. No new package dependencies. | Compile-time |
| F-P2L1-001 register-before-serve | Guard fires BEFORE AdmitNode — consistent with the fail-closed posture established by PR #127 for all pre-admission checks. The guard is purely additive and does not change the existing sequence for the matching-SVTNID path. | AC-001 regression test |
| DI-002 — private keys never transit | Guard operates on the ChallengeResponse outer-header `svtn_id` field (a 16-byte SVTN identifier), not on any key material. No security-perimeter impact. | Code review |
| Canonical error string (error-taxonomy v5.2 E-ADM-024) | The error returned from `nodeIdentifyHandshake` and/or the WARN log string MUST contain `"node_identify: ChallengeResponse svtn_id mismatch"` byte-identically. No paraphrase, no extra wrapping text that obscures the canonical string. | AC-002 and AC-003 tests (substring match) |
| go.md rule 3 — always check error returns | The `conn.Close()` call on the mismatch path: `_ = conn.Close()` (blank-identifier discard is acceptable here because the connection is already being abandoned; this mirrors the PR #127 fail-closed error-path pattern for this same function). | Code review |
| Forbidden dependency: no new imports in `cmd/switchboard` beyond what PR #127 introduced | `fmt.Errorf` and the comparison `crHdr.SVTNID != svtnID` use only packages already imported by `node_identify_wire.go`. | `go list -deps ./cmd/switchboard` |

## Library & Framework Requirements (MANDATORY)

| Tool / Package | Version | Purpose |
|----------------|---------|---------|
| Go | 1.25.4 (per `go.mod`) | Language runtime |
| `fmt` | stdlib | `fmt.Errorf` for canonical error message |
| `net` | stdlib | `conn.Close()` on mismatch path |
| (all other imports) | unchanged from PR #127 | The guard uses no new packages; all types (`[16]byte` SVTNID comparison) are primitive Go built-ins |

## File Structure Requirements (MANDATORY)

| File | Action | Purpose |
|------|--------|---------|
| `cmd/switchboard/node_identify_wire.go` | **modify** | Add 2-line SVTNID-consistency guard in `nodeIdentifyHandshake` after ChallengeResponse outer-header decode, before `admission.AdmitNode` call. Add WARN log consistent with E-ADM-022/E-ADM-023 precedent. |
| `cmd/switchboard/node_identify_wire_test.go` | **modify** | Add `TestNodeIdentifyHandshake_CRSVTNIDMatch_ProceedsToAdmitNode` (AC-001), `TestNodeIdentifyHandshake_CRSVTNIDMismatch_ConnectionClosed_BeforeAdmitNode` (AC-002), `TestNodeIdentifyHandshake_CRSVTNIDMismatch_WarnLogContainsE_ADM_024` (AC-003) |

No other files are modified by this story.

## Provenance

- **Origin:** Post-merge security review of PR #127 (`S-BL.NODE-IDENTIFY-WIRE`);
  drift item `SEC-NIDW-SVTNID-CONSISTENCY (MED)`.
- **BC source:** BC-2.01.009 v1.4 (PC-9 added 2026-07-19); v1.5 (changelog number
  corrected to PC-9, E-ADM-024 registered in error-taxonomy).
- **Error code:** error-taxonomy.md v5.2, E-ADM-024 (`"node_identify: ChallengeResponse
  svtn_id mismatch"`; added in v5.2 changelog entry 2026-07-19).
- **Prerequisite:** `S-BL.NODE-IDENTIFY-WIRE` (PR #127 @ 7fcf0cf) — delivers the
  `nodeIdentifyHandshake` function this story patches.

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.0 | 2026-07-19 | Initial creation — SEC-NIDW-SVTNID-CONSISTENCY follow-up story per post-merge review of PR #127; BC-2.01.009 PC-9 source. |
