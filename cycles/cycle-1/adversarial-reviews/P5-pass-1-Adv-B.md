---
document_type: adversarial-review
phase: 5
pass: 1
lens: Adv-B
scope: public-surface-test-rigor-traceability
develop_tip: 7fe3e29e4358df16e4e2f1de65a4e0d972540b4a
prior_passes_read: false
verdict: HAS_FINDINGS
finding_high: 1
finding_medium: 2
finding_low: 1
observation_count: 2
timestamp: 2026-07-02
model: opus
---

# Phase 5 Pass 1 — Adversary B (Test Rigor L2 + Traceability/Governance L3, public-surface lens)

## Verdict: HAS_FINDINGS

Three passes of self-validation applied. Findings that failed evidence/actionability grounding were dropped (e.g., an initial "AdvertisementRoundTrip tautology" concern was demoted after re-reading `discovery_test.go:729-748` and confirming field-by-field oracle survives byte-equality tautology).

## F-P5-Adv-B-H-001 [HIGH] E-NET-006 public error code shipped with zero emission site — public-contract-versus-code drift

**Evidence:**
- `.factory/specs/prd-supplements/error-taxonomy.md:119` declares `E-NET-006 | NET | broken | 1 | "router draining; connect to alternate router at <alternates_list>" | BC-2.09.002`
- Grep across the repository for `E-NET-006`: matches only in `docs/demo-evidence/S-4.01/evidence-report.md` (retrospective doc) and the taxonomy row itself. **Zero matches in `cmd/`, `internal/`.**
- Grep across the repository for the message string `router draining`: **zero matches in any Go source.**
- BC-2.09.002 is the drain contract; story S-7.04 is `status: pending` (`.factory/stories/S-7.04-pe-graduation-drain.md:7`).

**Why HIGH:** the error taxonomy is a governance artifact. It is a public promise about what error codes an operator can see. Publishing `E-NET-006` in the taxonomy with no source-of-emission crosses the "code drift vs governance artifact" line. This is not the same finding as HS-006 (which flagged the story being unshipped from the demo-scenario angle) — this is an assertion that the **taxonomy currently lies to operators**: it promises an emission behavior no binary supports. Phase 4 HS-006 saw the missing CLI; this finding sees the missing wire signal.

**Remedy classes:** either (a) mark `E-NET-006` with a `PENDING-S-7.04` annotation in the taxonomy row (same shape as E-CFG-002's "unreachable" defensive-only annotation at line 99), or (b) file a POL-003-adjacent lint that every error-code row in `error-taxonomy.md` has ≥1 grep hit in `cmd/` or `internal/`.

**Related:** Phase 4 HS-006 signal (drain CLI missing) — this finding is orthogonal; different artifact (taxonomy vs CLI).

## F-P5-Adv-B-M-001 [MEDIUM] POL-003 machine-checkability of `source_bc:` version pins is structurally unenforceable — 1/76 VPs conform

**Evidence:**
- Grep of all 76 VP `source_bc:` fields under `.factory/specs/verification-properties/`.
- **Only VP-048** carries a version-pin suffix: `source_bc: BC-2.07.001 v1.12` (`.factory/specs/verification-properties/VP-048.md:14`).
- All 75 other VPs carry bare `source_bc: BC-N.NN.NNN` — no version pin.
- Wave-6-added VPs (VP-043, VP-044, VP-045, VP-047, VP-055, VP-062) — **none** carry the pin.

**Why MEDIUM:** the review-scope prompt notes a `DRIFT-POL003-VP-FRONTMATTER-VERSION-PIN` open item. I confirm the drift extent from the artifact side: it is not a per-VP miss — the version-pin form is present in exactly one artifact out of the whole VP catalog. POL-003's promise of "machine-checkable" is unsupported because there is no canonical shape to check against. If VP-048's shape is the intended form, 75 VPs are drifted; if it's not, VP-048 is drifted. Either way, POL-003 as written cannot pass a lint gate.

**Remedy classes:** POL-003 either (a) requires an amendment stating "if source_bc is BC-2.07.001, pin to v1.12", i.e. a per-BC opt-in list, or (b) makes the pin a per-VP field like `source_bc_version:` that's separately enforceable with `null` allowed.

**Novelty:** this finding is an expansion of the already-open drift — quantifies the gap (1/76 = 1.3% conformance) and identifies that POL-003 as written cannot be a lint gate.

## F-P5-Adv-B-M-002 [MEDIUM] BC-2.09.003 PC-7/PC-8/PC-9 have `DEFERRED-APPLICATION` obligation but no defer-tracking mechanism in place

**Evidence:**
- `.factory/stories/S-7.04-pe-graduation-drain.md:66,69,73` — AC-005, AC-006, AC-007 all cite `BC-2.09.003 PC-7/PC-8/PC-9 DEFERRED-APPLICATION (S-7.04 obligation L-5)`.
- S-7.04 status: `pending` — no red gate, no PR.

**Why MEDIUM:** three postconditions of a shipped BC have deferred-application obligations tied to a single pending story. If S-7.04 is deprioritized indefinitely, three BC PCs will silently remain unenforced. No observable tracking artifact ensures that these three deferred obligations are surfaced as blocking gates on any release milestone.

**Remedy classes:** either (a) add a DRIFT item to STATE.md / STORY-INDEX.md listing BC-2.09.003 PC-7/PC-8/PC-9 as "pending-S-7.04" with a release-gate flag, or (b) mark them in the BC file with an ISO-date TTL that a lint gate can enforce.

## F-P5-Adv-B-L-001 [LOW] E-CFG-002 has a "defensive: emitted if..." annotation but E-NET-006 (identical situation) does not

**Evidence:**
- `.factory/specs/prd-supplements/error-taxonomy.md:99` — E-CFG-002 row: `"private key export not supported: <reason>" | BC-2.05.007 (defensive: emitted if any attempted private-key extraction path is invoked; BC-2.05.007 requires this path to be unreachable. Presence of this code at runtime would indicate a code defect.)`
- `.factory/specs/prd-supplements/error-taxonomy.md:119` — E-NET-006 row: bare mapping to `BC-2.09.002`, no annotation.

**Why LOW:** shape inconsistency in taxonomy annotation convention. E-CFG-002 documents its "defensive" nature (path guaranteed unreachable at runtime). E-NET-006 documents nothing analogous, yet it has no emission site and is dependent on unshipped S-7.04. Consistency in annotation shape reduces reader confusion about which rows are aspirational vs live.

**Remedy:** if F-P5-Adv-B-H-001 is resolved via annotation, use the E-CFG-002 shape as the template.

## Observation-1 [process-gap] positive-coverage assertion pattern absent from error-taxonomy governance

The CI-as-code review axis (positive-coverage assertion for regression-detector jobs) has a natural analog on the spec side: no CI job scans `error-taxonomy.md` rows against a corpus of `cmd/`/`internal/` code and asserts a non-zero emission-site count per row. If such a gate existed, F-P5-Adv-B-H-001 would have been caught pre-merge. `[process-gap]` per S-7.02 tagging convention.

## Observation-2 — test-rigor on the Wave-6 public surface is genuinely high

Positive observation — not a defect. All three merged Wave-6 stories exhibit non-tautological test oracles:
- FEC (S-7.01): independent `xorOracle(payloads)` helper computes expected parity separately from the encoder — the test would fail if the encoder ships identity XOR (`internal/arq/fec_test.go:50-67, 150-179`).
- Discovery (S-7.02): HMAC negative tests use `errors.Is(err, discovery.ErrInvalidHMACTag)` and further assert the tampered session is absent from `Enumerate()` (`internal/discovery/discovery_test.go:788-801, 845-859`). Byte-equality round-trip is followed by field-by-field oracle at `discovery_test.go:729-748` — tautology broken.
- RouterAddr (S-BL.ROUTER-ADDR): `TestVP047_RouterAddrNonEmpty` uses three independent oracles (host:port regex, exact equality, JSON deserialization round-trip) on the same field (`internal/metrics/integration_test.go:412-439`).

Chain the internal-code adversary walked passes rigor from this vantage.

## Novelty Assessment

**Novelty: MEDIUM.** F-P5-Adv-B-H-001 is genuinely new — Phase 4 HS-006 flagged missing drain CLI, but not the taxonomy-vs-code drift for E-NET-006 specifically. F-P5-Adv-B-M-001 expands a known drift with the quantification 1/76, sharpening it into a POL-003 enforceability blocker. F-P5-Adv-B-M-002 is new — the DEFERRED-APPLICATION obligation-tracking gap is not covered by any error-taxonomy or VP-INDEX audit axis I know of. F-P5-Adv-B-L-001 is a shape-consistency observation. Findings are not retreading known nitpicks; they extend Phase 4's signal into governance territory.

## Files load-bearing for these findings

- `.factory/specs/prd-supplements/error-taxonomy.md` (lines 99, 119)
- `.factory/specs/verification-properties/VP-048.md` (line 14)
- `.factory/stories/S-7.04-pe-graduation-drain.md` (lines 7, 66, 69, 73)
- `internal/arq/fec_test.go` (lines 50-67, 150-179)
- `internal/discovery/discovery_test.go` (lines 729-748, 762-859)
- `internal/metrics/integration_test.go` (lines 376-439)
