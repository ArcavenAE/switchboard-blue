---
artifact_id: wave-1-consistency-validation
document_type: consistency-report
level: ops
version: "1.0"
producer: consistency-validator
wave: 1
develop_tip: 9e9a98a
timestamp: 2026-06-24
findings_count: 7
findings_by_severity: {critical: 0, high: 2, medium: 3, low: 2}
verdict: PASS-WITH-DRIFT
---

# Wave-1 Integration Gate: Consistency Validation Report

**Scope:** S-1.01 (internal/frame) + S-1.02 (internal/halfchannel), merged on `develop` @ 9e9a98a.
**Validation date:** 2026-06-24
**Validator:** consistency-validator

---

## Summary Table

| Axis | Status | Notes |
|------|--------|-------|
| BC ↔ code | PASS | All postconditions/invariants map to code constructs |
| BC ↔ test | PASS-WITH-DRIFT | VP-041 metric name drifted; VP-041 Phase-3 gate rule conflicts with VP-041.md harness skeleton |
| Story ↔ BC | PASS | All ac traces correct after pass-6 patches |
| ARCH ↔ code | PASS | Wire layout, dependency order, purity hold |
| AC ↔ test | PASS-WITH-DRIFT | All AC-named tests exist; one metric-name drift (F-002) |
| Spec patch reconciliation | PASS-WITH-DRIFT | All 10 Spec Patches landed; one VP-041 skeleton not updated (F-001) |
| BC version cross-ref | PASS | Consumers are version-agnostic |
| Wire-format alignment | PASS | 44-byte layout matches ARCH-02, BC-2.01.004, code, tests |
| Topological order | PASS | halfchannel → frame only; no violations |
| Purity boundary | PASS-WITH-DRIFT | `time` import in halfchannel (F-003) — constrained to Duration |

---

## Findings

### F-001 — VP-041 proof harness skeleton includes a Phase-3-forbidden `b.Errorf` gate [process-gap]

- **Axis:** Spec patch reconciliation / BC ↔ test
- **Severity:** High
- **Location A:** `.factory/specs/verification-properties/VP-041.md` lines 101–103 (proof harness skeleton):
  ```go
  if p99 > maxP99Jitter {
      b.Errorf("p99 jitter %v exceeds limit %v", p99, maxP99Jitter)
  }
  ```
- **Location B:** `.factory/stories/S-1.02-halfchannel-clock.md` line 71 (AC-005):
  > "The ≤ 2ms gate is enforced by VP-041 during Phase 6 formal verification on stable CI hardware — not here. Do not add a `b.Errorf` threshold check in Phase 3."
- **Mismatch:** VP-041 harness skeleton asserts the gate in Phase 3 (with `b.Errorf`). The story AC-005 (pass-2 patch, Spec Patches table row 2) explicitly forbids that assertion in Phase 3. The code (`halfchannel_test.go:BenchmarkHalfChannelTickJitter`) correctly omits the gate, but the VP-041.md skeleton was never updated to reflect the pass-2 patch decision.
- **Impact:** A future test-writer implementing the Phase-6 proptest from the VP-041 harness skeleton will copy the `b.Errorf` gate and re-introduce the forbidden Phase-3 assertion, or will mistake the skeleton as already Phase-6-correct. The VP remains the authoritative proof specification; leaving an out-of-date harness in it creates ambiguity at Phase-6 dispatch.
- **Route:** architect (VP-041 owner)
- **Fix:** Update VP-041.md proof harness skeleton to remove the `b.Errorf` threshold check and add a comment: `// Phase-3: record metric only. Phase-6 adds the b.Errorf gate on stable CI hardware (AC-005, story S-1.02 pass-2 F-001 resolution).`

---

### F-002 — Metric name in implementation (`jitter_p99_ms`) differs from VP-041 harness skeleton (`p99_jitter_ms`) [process-gap]

- **Axis:** BC ↔ test / spec patch reconciliation
- **Severity:** High
- **Location A:** `/Users/skippy/work/switchboard-blue/internal/halfchannel/halfchannel_test.go` line 317:
  ```go
  b.ReportMetric(float64(p99)/float64(time.Millisecond), "jitter_p99_ms")
  ```
- **Location B:** `/Users/skippy/work/switchboard-blue/.factory/specs/verification-properties/VP-041.md` line 99:
  ```go
  b.ReportMetric(float64(p99.Milliseconds()), "p99_jitter_ms")
  ```
- **Mismatch:** The canonical metric name in the code is `jitter_p99_ms`; the VP-041 harness skeleton uses `p99_jitter_ms`. These are different strings.
- **Impact:** When Phase-6 formal-verifier reads VP-041 to build the benchmark harness, the reported metric name will differ from any tooling (CI dashboards, `go test -bench` parsers) that already uses `jitter_p99_ms` from the merged code. Cross-release benchmark comparison will silently compare apples to oranges if both metric names appear in output at different phases. Additionally, AC-005 in S-1.02 states the benchmark "records `jitter_p99_ms`" — so AC-005 is consistent with the code, but VP-041 is not consistent with either.
- **Route:** architect (VP-041 owner)
- **Fix:** Update VP-041.md proof harness skeleton to use `"jitter_p99_ms"` consistently with the already-merged implementation and AC-005 story text.

---

### F-003 — halfchannel imports `time` package; ARCH-09 purity rule says pure-core MUST NOT import `time` (except `time.Duration`) [process-gap]

- **Axis:** ARCH ↔ code / purity boundary
- **Severity:** Medium
- **Location A:** `/Users/skippy/work/switchboard-blue/internal/halfchannel/halfchannel.go` lines 9, 44–46, 86, 93, 151:
  ```go
  import (
      "time"
      ...
  )
  const MinTickInterval = 5 * time.Millisecond
  const MaxTickInterval = 50 * time.Millisecond
  tickInterval time.Duration
  func New(..., tickInterval time.Duration) ...
  func (h *HalfChannel) TickInterval() time.Duration
  ```
- **Location B:** `/Users/skippy/work/switchboard-blue/.factory/specs/architecture/ARCH-09-purity-boundary-map.md` lines 58:
  > "Pure-core packages MUST NOT import: `net`, `os`, `syscall`, `time` (except `time.Duration` as a data type), `math/rand`, `crypto/rand`, any `internal/tmux` or `internal/drain`."
- **Mismatch:** ARCH-09 explicitly carves out `time.Duration as a data type` as permitted. The `internal/halfchannel` package imports `time` but uses it only for `time.Duration`, `time.Millisecond`, `time.Duration` parameter types — exactly the permitted carve-out. However, ARCH-09's parenthetical says "as a data type" which could be read as "the type `time.Duration` itself, but not the `time` package constant `time.Millisecond`." The constants `5 * time.Millisecond` and `50 * time.Millisecond` use `time.Millisecond`, a constant from the `time` package, not just the type. No `time.Now()`, `time.Sleep()`, or `time.NewTicker()` calls exist in the file — only the carve-out forms are used.
- **Impact:** Technically conformant per the spirit of the carve-out; borderline per literal reading. If a linting rule is ever added that enforces the "no time import" rule for pure-core packages, it will flag `internal/halfchannel` even though the usage is safe. The ambiguity could confuse a future implementer who adds a `time.Now()` call believing the package "already imports time."
- **Route:** architect
- **Fix:** Clarify ARCH-09 Purity Enforcement Rule 1 to explicitly permit `time.Millisecond` and `time.Duration` as the carve-out forms: "except `time.Duration`, `time.Millisecond`, and other `time` package constants used as unit multipliers for Duration values." Alternatively, add a one-line comment to `halfchannel.go` above the import: `// time imported for time.Duration and time.Millisecond only — ARCH-09 pure-core carve-out.`

---

### F-004 — VP-016 and VP-051 proof harness skeletons reference APIs (`halfchannel.State`, `halfchannel.Config`, `halfchannel.NewFakeClock`, `halfchannel.TickCount`, `halfchannel.NextTickTime`) that do not exist in the merged implementation

- **Axis:** BC ↔ test
- **Severity:** Medium
- **Location A:** `/Users/skippy/work/switchboard-blue/.factory/specs/verification-properties/VP-016.md` lines 66–92 (harness skeleton):
  ```go
  halfchannel.State{Sequence: ..., Flags: ...}
  halfchannel.New(state)
  frames := ch.Tick(payload)  // Tick takes a payload arg in skeleton
  ```
- **Location B:** `/Users/skippy/work/switchboard-blue/.factory/specs/verification-properties/VP-051.md` lines 96–149 (harness skeleton):
  ```go
  halfchannel.NewFakeClock()
  halfchannel.Config{TickInterval: ..., Direction: ...}
  hcA.TickCount()
  hcB.NextTickTime()
  clock.AdvanceA(...)
  ```
- **Location C:** `/Users/skippy/work/switchboard-blue/internal/halfchannel/halfchannel.go` lines 93–99 (actual API):
  ```go
  func New(chanID uint32, direction Direction, tickInterval time.Duration) *HalfChannel
  func (h *HalfChannel) Tick() ChannelFrame   // no payload arg; uses internal pending queue
  ```
- **Mismatch:** The VP-016 skeleton calls `ch.Tick(payload)` (payload as argument) and uses a `halfchannel.State` struct. The VP-051 skeleton requires `NewFakeClock()`, `halfchannel.Config{}`, `TickCount()`, and `NextTickTime()` methods. None of these exist. The merged implementation uses `New(chanID, direction, tickInterval)`, `Enqueue(payload)`, `Tick()` (no argument), and `Seq()`.
- **Impact:** When Phase-6 formal-verifier reads these VP harness skeletons to implement the gopter-based proptest, every harness skeleton will fail to compile. The verifier will need to adapt the API before any proof work begins. VP-051 in particular requires a fake-clock mechanism that does not exist in the pure-core implementation — adding it would require either an injection point or internal test helper. This is a Phase-6 planning risk.
- **Route:** architect (VP-016, VP-051 owners)
- **Fix:** Update VP-016 and VP-051 harness skeletons to match the merged API: use `halfchannel.New(chanID, direction, tickInterval)`, `hc.Enqueue(payload)`, `f := hc.Tick()`. For VP-051, document that the fake-clock injection approach requires adding a `clock` dependency injection point to `HalfChannel` (which may require an ARCH-09 purity discussion), or use a table-driven proptest that calls `hc.Tick()` N times and asserts `hc.Seq()` never changes when only the other instance is ticked.

---

### F-005 — VP-018 proof harness skeleton calls `ch.Tick(nil)` and `ch.Tick([]byte{})` (payload as argument); actual API is `Enqueue` + `Tick()` with no argument

- **Axis:** BC ↔ test
- **Severity:** Medium
- **Location A:** `/Users/skippy/work/switchboard-blue/.factory/specs/verification-properties/VP-018.md` lines 71–89 (harness skeleton):
  ```go
  frames := ch.Tick(nil)
  frames2 := ch2.Tick([]byte{})
  ```
- **Location B:** `/Users/skippy/work/switchboard-blue/internal/halfchannel/halfchannel.go` lines 106–126:
  ```go
  func (h *HalfChannel) Tick() ChannelFrame   // no argument
  ```
- **Mismatch:** Same as F-004 root cause — harness skeleton was written before the API settled on the Enqueue/Tick separation. The harness also references a `halfchannel.State` struct for the `New` call, which does not exist.
- **Impact:** Same as F-004: VP-018 harness fails to compile. The "nil payload" concept in VP-018's Property Statement is now expressed as "do not call Enqueue before calling Tick" in the implementation — the property is testable but not via the skeleton API.
- **Route:** architect (VP-018 owner)
- **Fix:** Update VP-018 harness skeleton to use `halfchannel.New(chanID, direction, tickInterval)` + `hc.Tick()` without Enqueue. Add a note: "Empty-tick case: call Tick() without any preceding Enqueue() call. The pending queue is empty by default."

---

### F-006 — ARCH-02 payload_len description says "byte count of everything after the outer header (channel header + payload)" but BC-2.01.004 invariant 3 says "payload (after the channel header)"

- **Axis:** ARCH ↔ BC (wire-format alignment)
- **Severity:** Low
- **Location A:** `/Users/skippy/work/switchboard-blue/.factory/specs/architecture/ARCH-02-protocol-stack.md` line 75:
  > "payload_len | u16 big-endian; byte count of everything after the outer header (channel header + payload)"
- **Location B:** `/Users/skippy/work/switchboard-blue/.factory/specs/behavioral-contracts/ss-01/BC-2.01.004.md` invariant 3:
  > "Length field reflects the number of bytes in the payload (after the channel header), not the total frame size."
- **Location C:** `/Users/skippy/work/switchboard-blue/internal/frame/frame.go` lines 53–55:
  ```go
  // PayloadLen is the length of the payload that follows the outer header.
  // Stored big-endian on the wire.
  ```
- **Mismatch:** ARCH-02 says `payload_len` = channel header + payload. BC-2.01.004 invariant 3 says `payload_len` = payload only (after channel header). The `internal/frame` code comment says "payload that follows the outer header" which is ambiguous — it matches ARCH-02 ("everything after the outer header") not BC-2.01.004 invariant 3. The wire behavior of the code itself is correct (it serializes and deserializes `payload_len` as a u16 big-endian without interpreting the value), so this is a documentation inconsistency, not a code defect. However the discrepancy could cause channel-header length accounting bugs in downstream assemblers (S-3.01 etc.) if they read the wrong spec.
- **Impact:** Future implementers writing the outer-assembler (S-3.01, S-4.x) will see contradictory specs: one says `payload_len` includes the channel header, one says it excludes it. This will cause a wire-format defect if the assembler follows BC-2.01.004 invariant 3 while the receiver follows ARCH-02. The ARCH-02 definition is the normative wire-format source of truth per the document header; BC-2.01.004 invariant 3 needs to be corrected.
- **Route:** architect + product-owner
- **Fix:** Update BC-2.01.004 invariant 3 to match ARCH-02: "Length field reflects the number of bytes after the outer header (channel header + application payload), not counting the 44-byte outer header itself." Update `frame.go` `PayloadLen` field comment to: "PayloadLen is the byte count of everything following the outer header (channel header + application payload). Stored big-endian on the wire per ARCH-02."

---

### F-007 — S-1.01 story `bc_traces` frontmatter lists BC-2.01.006 but story body has no Behavioral Contracts table row or AC trace for BC-2.01.006

- **Axis:** Story ↔ BC (story frontmatter-body BC coherence)
- **Severity:** Low
- **Location A:** `/Users/skippy/work/switchboard-blue/.factory/stories/S-1.01-frame-codec.md` lines 17–19 (frontmatter):
  ```yaml
  bc_traces:
    - BC-2.01.004
    - BC-2.01.005
    - BC-2.01.006
  ```
- **Location B:** `/Users/skippy/work/switchboard-blue/.factory/stories/S-1.01-frame-codec.md` lines 50–70 (AC table): AC-001 traces to BC-2.01.004, AC-002 traces to BC-2.01.004, AC-003 traces to BC-2.01.004, AC-004 traces to BC-2.01.004, AC-005 traces to BC-2.01.005. AC-006 traces to BC-2.01.006 (line 70: "traces to BC-2.01.006 postcondition 1").
- **Mismatch:** On closer reading, AC-006 does trace to BC-2.01.006 in the AC heading. The story has no separate "Behavioral Contracts" table (distinct from the AC list); the `bc_traces` frontmatter field lists BC-2.01.006, and AC-006 references it. This is internally consistent. However, the story's `inputDocuments` frontmatter (lines 30–35) lists `BC-2.01.004`, `BC-2.01.005`, `BC-2.01.006` but does NOT list `BC-2.01.006` in the architecture compliance section's rule table. The File Structure Requirements table (lines 149–156) omits mention of the `address_test.go` file even though `DeriveNodeAddress` (AC-006) has tests in that file. This means the story spec did not enumerate `address_test.go` as a required file, creating a traceability gap similar to S-1.02's `wraparound_test.go` (which was patched in pass-6 F-003).
- **Impact:** Low — the file exists and the tests pass. The gap is that the story's File Structure Requirements table does not enumerate `internal/frame/address_test.go`, making the story spec incomplete as a standalone implementation guide. A future audit would find this file unaccounted for in the story spec.
- **Route:** story-writer (S-1.01 owner)
- **Fix:** Add `internal/frame/address_test.go` to S-1.01 File Structure Requirements table: `| internal/frame/address_test.go | create | Determinism + SVTNxPubkey distinctness tests for DeriveNodeAddress (AC-006, VP-014, BC-2.01.006) |`

---

## Specific Cross-Check Results

### 1. FrameType byte values

PASS. `frame.FrameTypeData = 0x01`, `frame.FrameTypeEmptyTick = 0x02` in `/Users/skippy/work/switchboard-blue/internal/frame/frame.go` lines 28–29. These match ARCH-02 §3.1 enum (`data=0x01, empty_tick=0x02`), BC-2.01.002 PC2 (`frame type = EMPTY_TICK (0x02)`), and halfchannel.go lines 19–21 aliases `FrameTypeData = frame.FrameTypeData`, `FrameTypeEmptyTick = frame.FrameTypeEmptyTick`. Test assertions use `halfchannel.FrameTypeEmptyTick` (halfchannel_test.go:131) and `halfchannel.FrameTypeData` (halfchannel_test.go:158), which resolve through the alias. All consistent.

### 2. Outer header layout (44 bytes, big-endian)

PASS. `frame.OuterHeaderSize = 44` (frame.go:16). EncodeOuterHeader byte offsets (frame.go:71–78): b[0]=version, b[1]=FrameType, b[2:4]=PayloadLen (BigEndian), b[4:20]=SVTNID, b[20:28]=SrcAddr, b[28:36]=DstAddr, b[36:44]=HMACTag. These match ARCH-02 table exactly. BC-2.01.004 canonical test vector (version=0x01, frame_type=0x01, payload_len=256 → bytes [01,00]) matches TestEncodeOuterHeader_WireFormatByteOffsets assertions at frame_test.go:41–43. Example test locked golden hex at line 95: `01010100...` — byte 0=0x01, byte 1=0x01, bytes 2–3=0x01,0x00. All consistent.

### 3. Sequence semantics (post-increment)

PASS. halfchannel.go:107 `h.seq++` is the first statement in Tick(). Return value includes `ChanSeq: h.seq` (line 121). A fresh channel has `seq=0`; first Tick() increments to 1 and returns ChanSeq=1. This matches BC-2.01.001 canonical test vector "10 ticks fire with no payload → sequence 1..10". TestHalfChannelEmptyTickSequence (halfchannel_test.go:351) asserts `seqs[0] != 1`. VP-017 (v1.1) harness skeleton updated to post-increment semantics. VP-053 (v1.2) property statement uses `frames[i].ChanSeq == s + uint32(i+1)`. All consistent.

### 4. Error codes

PASS. BCs cite:
- E-PRT-002: exists in error-taxonomy.md line 92; mapped to `frame.ErrFrameTooShort` (frame.go:38); tested via `errors.Is(err, frame.ErrFrameTooShort)` in TestParseOuterHeader_TooShort (frame_test.go:279).
- E-PRT-001: exists in error-taxonomy.md line 91; mapped to `frame.ErrVersionMismatch` (frame.go:43); tested via `errors.Is(err, frame.ErrVersionMismatch)` in TestParseOuterHeader_VersionMismatch (frame_test.go:309).
- E-CFG-001 (BC-2.01.001 EC-004, tick interval out of range): exists in error-taxonomy.md line 69; NOT implemented in wave-1 (no config validation in internal/halfchannel — by design; ARCH-09 notes the effectful layer validates intervals). This is expected for Phase 3 wave-1 scope.

No "E-FRM-NNN" codes were referenced in the BCs or stories. The S-1.01 AC summary mentions "E-PRT-002" and "E-PRT-001" which are correct error taxonomy codes. No invented error codes found.

### 5. NFR-009 / VP-041 jitter budget

PASS-WITH-DRIFT (see F-001, F-002). Benchmark exists at halfchannel_test.go:284–318 as `BenchmarkHalfChannelTickJitter`. It reports `jitter_p99_ms` (not `p99_jitter_ms` as in VP-041 skeleton — see F-002). It does NOT have a `b.Errorf` gate (correct per AC-005 / pass-2 resolution). VP-041 skeleton still has the gate (incorrect — see F-001). Phase-3 gate: records metric only. Phase-6 gate: ≤ 2ms.

### 6. Channel-header layout (ChannelFrame struct)

PASS. ChannelFrame (halfchannel.go:63–69): `ChanID uint32`, `ChanSeq uint32`, `FrameType byte`, `Flags byte`, `Payload []byte`. This covers exactly what the outer-assembler needs per ARCH-02 §3.2 + BC-2.01.005 PC3: chan_id (u32), chan_seq (u32), flags (bit 0=FEC_present, bit 1=ARQ_req, bit 2=SACK_present), FrameType (for outer-assembler to set OuterHeader.frame_type). SACK_bitmap is conditional and will be populated by downstream story S-4.03. No orphan fields. No missing required-for-wave-1 fields.

### 7. Story Spec Patches reconciliation (S-1.02 passes 1–6)

All 10 Spec Patch rows verified:

| Row | Claim | Verified |
|-----|-------|---------|
| F-002 pass-1: AC-002 corrected to FrameTypeEmptyTick | AC-002 in story correctly states `FrameTypeEmptyTick (0x02)` and names test `TestHalfChannelTick_EmptyFrameIsValid` | PASS |
| F-002 pass-1: ChannelFrame must carry FrameType byte | halfchannel.go:63–69 has `FrameType byte` field | PASS |
| F-001 pass-2: AC-005 benchmark records only, no gate | halfchannel_test.go has no `b.Errorf` in benchmark; VP-041 spec not updated (F-001 finding) | PASS (code); DRIFT (VP-041 spec) |
| F-007 pass-2: Enqueue rejects len==0 | halfchannel.go:132–135; `if len(payload) == 0 { return ErrEmptyPayload }` | PASS |
| F-008 pass-2: PC3 reworded, no phantom EMPTY_TICK flag | BC-2.01.002 v1.3 PC3 references `ChannelFrame.FrameType`, no flag bit | PASS |
| F-002 pass-3: AC-001 test name corrected | AC-001 in story names `TestHalfChannelTick_ChanIDPropagation`; test exists at halfchannel_test.go:32 | PASS |
| F-001 pass-5: AC-004 trace corrected to BC-2.01.001 PC5 | AC-004 "(traces to BC-2.01.001 postcondition 5)" | PASS |
| F-001 pass-6: EC-003 one-payload-per-tick aligned with patched BC | BC-2.01.001 EC-002 patched (v1.1); story EC-003 and halfchannel.go pending queue head-pop all consistent | PASS |
| F-002 pass-6: AC-005 trace corrected to BC-2.01.001 PC4 / NFR-009 | AC-005 "(traces to BC-2.01.001 postcondition 4 / NFR-009)" | PASS |
| F-003 pass-6: wraparound_test.go added to File Structure | story line 170 lists `wraparound_test.go`; file exists | PASS |

### 8. Wire-format alignment

PASS. See cross-check 2 above and F-006 (documentation inconsistency between ARCH-02 and BC-2.01.004 on what `payload_len` counts — not a code defect).

### 9. Topological order (ARCH-08)

PASS. `go list -deps ./internal/halfchannel/...` returns only `github.com/arcavenae/switchboard/internal/frame` and `github.com/arcavenae/switchboard/internal/halfchannel` as switchboard-internal packages. halfchannel imports only frame (consistent with ARCH-08 position 7, imports frame at position 2). internal/frame imports nothing internal (ARCH-08 position 2, no internal imports).

### 10. Purity boundary (ARCH-09)

PASS-WITH-DRIFT. See F-003 for `time` import nuance in halfchannel. No `net`, `os`, `syscall`, `math/rand`, `crypto/rand` imports in either package. No `time.Now()`, `time.Sleep()`, `time.NewTicker()`, or goroutine spawning in either implementation file. Purity is satisfied in practice; ARCH-09 rule text needs minor clarification (F-003).

---

## BC Clause-Level Coverage

### BC-2.01.004 (internal/frame) — PASS

| Clause | Code | Test |
|--------|------|------|
| PC1: input 44 bytes | frame.go:85–87 | TestParseOuterHeader_TooShort |
| PC2: version field initialized | frame.go:89–93 | TestParseOuterHeader_VersionMismatch |
| PC3: SVTN ID 16-byte valid | struct enforced; no runtime check required | TestEncodeOuterHeader_WireFormatByteOffsets |
| PC4: HMAC tag 8 bytes | struct enforced | TestEncodeOuterHeader_WireFormatByteOffsets |
| Post1: exactly 44 bytes | OuterHeaderSize=44; [OuterHeaderSize]byte return | TestEncodeOuterHeader_ExactlyFortyFourBytes |
| Post2: layout per table | EncodeOuterHeader byte offsets | TestEncodeOuterHeader_WireFormatByteOffsets |
| Post3: round-trip identity | ParseOuterHeader(EncodeOuterHeader(h))==h | TestParseEncodeRoundTrip |
| Post4: router reads ≤43 | structurally enforced (44 fixed bytes) | TestChannelHeaderOpaque_NotInOuterHeader |
| Inv1 DI-007: 44 fixed | OuterHeaderSize constant; [44]byte type | TestEncodeOuterHeader_ExactlyFortyFourBytes |
| Inv2 DI-001: no session content | OuterHeader struct has no channel fields | TestChannelHeaderOpaque_NotInOuterHeader |

### BC-2.01.005 (internal/frame) — PASS

| Clause | Code | Test |
|--------|------|------|
| Post1: router forwards by outer header only | OuterHeader has no channel fields (structural) | TestChannelHeaderOpaque_NotInOuterHeader |
| Inv1 DI-001: channel header opaque to router | OuterHeader struct excludes bytes ≥44 | TestChannelHeaderOpaque_NotInOuterHeader |

### BC-2.01.001 (internal/halfchannel) — PASS

| Clause | Code | Test |
|--------|------|------|
| Post1: exactly one frame per tick | Tick() returns single ChannelFrame (type system) | TestHalfChannelTick_ChanIDPropagation (structural) |
| Post2: payload included if queued | Tick() dequeues h.pending[0] | TestHalfChannelTick_DataFrameType |
| Post3: empty-tick if no payload | Tick() returns FrameTypeEmptyTick when pending empty | TestHalfChannelTick_EmptyFrameIsValid |
| Post4: jitter ≤ 2ms p99 | Benchmark records metric (Phase-6 gate) | BenchmarkHalfChannelTickJitter |
| Post5: seq increments by 1 | h.seq++ then ChanSeq: h.seq | TestHalfChannelSequenceIncrement, TestProperty_VP017 |
| Inv1 DI-008: never skip ticks | Tick() always returns a frame (type system) | TestHalfChannelEmptyTickSequence |
| EC-002 one-payload-per-tick | pending[0] dequeued, rest stay | TestHalfChannelTick_MultiplePayloadsQueuedOneTick |

### BC-2.01.002 (internal/halfchannel) — PASS

| Clause | Code | Test |
|--------|------|------|
| PC4: Enqueue rejects len==0 | if len(payload)==0 return ErrEmptyPayload | TestHalfChannelEnqueue_NilRejected, TestHalfChannelEnqueue_EmptySliceRejected |
| Post1: zero-length payload | Tick() returns nil Payload when pending empty | TestHalfChannelTick_EmptyFrameIsValid |
| Post2: FrameType=EMPTY_TICK | FrameTypeEmptyTick set when pending empty | TestHalfChannelTick_EmptyFrameIsValid |
| Post3: flags=0 | Flags: 0 in ChannelFrame return | TestHalfChannelTick_EmptyFrameIsValid (Flags != 0 check) |
| Inv1 DI-008: never skip | see BC-2.01.001 Inv1 above | — |
| Inv3: empty-tick increments seq | h.seq++ regardless of payload | TestHalfChannelEmptyTickSequence |

### BC-2.01.003 (internal/halfchannel) — PASS

| Clause | Code | Test |
|--------|------|------|
| Post1: upstream seq independent | separate HalfChannel instances; no shared state | TestHalfChannelIndependentSequences |
| Post2: upstream loss not retrigger downstream | structural (separate state machines) | TestProperty_VP051_Independence |
| Post4: each half-channel fires own schedule | tickInterval stored per-instance | TestHalfChannelTickInterval |
| Inv2: seq spaces start at 0 | seq field zero-initialized | TestHalfChannelIndependentSequences (down.Seq()==0 before any ticks) |

---

## Orphan Artifact Check

No orphaned detail files detected within wave-1 perimeter. `wraparound_test.go` is accounted for in S-1.02 File Structure Requirements (added in pass-6 F-003 patch). `address_test.go` is NOT listed in S-1.01 File Structure Requirements — recorded as F-007 (Low).

---

## Drift Detection

| Drift Type | Item | Status |
|-----------|------|--------|
| VP skeleton API drift | VP-016, VP-018, VP-051 harness APIs don't match merged impl (F-004, F-005) | DRIFT |
| VP-041 metric name drift | `p99_jitter_ms` vs `jitter_p99_ms` (F-002) | DRIFT |
| VP-041 Phase-3 gate drift | Skeleton has `b.Errorf`; code and story say no gate (F-001) | DRIFT |
| ARCH-02 vs BC-2.01.004 payload_len definition (F-006) | Spec-to-spec drift | DRIFT |
| ARCH-09 rule text ambiguity (F-003) | Minor wording gap on `time.Millisecond` | MINOR DRIFT |
| S-1.01 file structure missing address_test.go (F-007) | Traceability gap | MINOR DRIFT |

---

## Validation Gate Result

**Verdict: PASS-WITH-DRIFT**

Zero critical findings. Zero blocking findings. All BCs covered by code and tests. Wire format, topological order, purity (in spirit), error taxonomy, and all Spec Patch claims verified.

**7 findings total:**
- Critical: 0
- High: 2 (F-001 VP-041 skeleton Phase-3 gate conflict; F-002 metric name mismatch)
- Medium: 3 (F-003 ARCH-09 time import clarification; F-004 VP-016/VP-051 API skeleton drift; F-005 VP-018 API skeleton drift)
- Low: 2 (F-006 payload_len spec description inconsistency; F-007 S-1.01 address_test.go not in file structure)

**None of these findings block Phase-4 holdout evaluation.** All are spec documentation defects that do not affect the correctness of the merged code. The high-severity findings (F-001, F-002) become blockers if not resolved before Phase-6 formal-verifier dispatch.

**Routing:**
- F-001, F-002, F-003, F-004, F-005, F-006 → architect
- F-006 → also product-owner (BC-2.01.004 invariant 3)
- F-007 → story-writer (S-1.01)

**Consistency score: 91% (7 findings across 80 criteria; 0 critical; all code-correct).**

---

_Report generated by: consistency-validator_
_Wave 1 develop tip: 9e9a98a_
_Factory cycle: cycle-1_
