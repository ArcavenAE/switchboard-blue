---
artifact_id: S-BL.CLI-SURFACE-COMPLETION
document_type: story
level: ops
story_id: S-BL.CLI-SURFACE-COMPLETION
epic_id: E-7
title: "CLI surface completion: dispatch + wire for paths ping, admin.svtn.status, router reload/drain, svtn destroy shim"
status: ready
producer: story-writer
timestamp: 2026-07-12T00:00:00Z
modified:
  - date: 2026-07-13
    version: "2.8"
    change: >
      Remediated step-4.5 impl pass 3 (fresh adversary) finding F-CS-I3-001 (LOW,
      File-Change-List completeness gap, orchestrator-verified). The File-Change List omitted two
      files present in the feature diff against `develop` (`git diff --stat`:
      `cmd/sbctl/main_test.go` +42/-10, `cmd/sbctl/phase5_pass8_test.go` +9/-3) — zero mentions of
      either in the story. Both are correct, necessary existing-test accommodations forced by this
      story's own scope, not scope creep: `main_test.go`'s `TestSbctl_OrphanSubcommands` asserted
      `svtn` was an orphan subcommand — AC-010's new `case "svtn":` dispatch re-points those
      sub-cases to AC-010 PC-3 behavior (exit 2 naming `svtn`, `wantNoPanic` guard);
      `phase5_pass8_test.go`'s `TestPathsUnknownVerb` used `ping` as its unknown-`paths`-verb
      example — AC-001 makes `ping` a real verb, so the exemplar swaps to `trace`. Fixed: two new
      rows added to the File-Change List, in the existing table's style, grouped with the other
      `cmd/sbctl` rows. No AC, PC, Decision, Forward Obligation, or point content touched — this is
      a documentation-completeness fix for accommodations the implementation already had to make;
      `acceptance_criteria_count` (16) and `estimated_points` (5) unchanged. `input-hash`
      unchanged — no declared `inputs:` file changed, story-body-only edit; `--check` confirms no
      drift. Frontmatter `version` 2.7 → 2.8; new `modified:` entry appended (newest-first).
  - date: 2026-07-13
    version: "2.7"
    change: >
      Propagated the step-4.5 impl pass 2 remediation burst (finding F-CS-I2-001, nitpick
      N-CS-I2-01) plus a version-pin refresh. **VP propagation (F-CS-I2-001, FO(d)):** every live
      `VP-TBD-PING-A`/`VP-TBD-PING-B` reference replaced with `VP-078`/`VP-079` — frontmatter
      `verification_properties:` and `vp_traces:` lists, AC-004 PC-3's parenthetical. The two
      v2.0-era historical references (frontmatter `modified:` entry, Changelog table row) and the
      unrelated BC-2.06.003 `VP-TBD-A`/`VP-TBD-B` precedent citation (Previous Story Intelligence
      table) are left untouched per the layered-decision-record convention — accurate records of
      what was true at authorship time / a different BC's own already-resolved placeholders, not
      this story's. **Forward Obligations table:** row (b) → DISCHARGED (`ARCH-INDEX.md` v1.10,
      F-CS-I2-001 — SS-06 gains `internal/mgmt`); row (d) → DISCHARGED (`VP-TBD-PING-A`→`VP-078`,
      `VP-TBD-PING-B`→`VP-079` minted in BC-2.06.004 v1.4, `VP-INDEX.md` v2.40 synced); row (a) →
      DISCHARGED (`CAP-029` minted in `capabilities.md` v1.1, BC-2.06.004 v1.5 re-anchored
      `CAP-022`→`CAP-029`, `BC-INDEX.md` v3.5 synced, per architect recommendation + PO
      concurrence) — all four Forward Obligations are now DISCHARGED, rows (a)/(b)/(d) rewritten
      to match row (c)'s existing wording style (Gate: "None — discharged"; Status: "DISCHARGED —
      landed in \<artifact\> v\<N\> (\<date\>, \<burst\>)"); Obligation-column description text left
      unchanged, same treatment as row (c) received at its own discharge. File-Change List's
      `ARCH-INDEX.md` row updated to the same discharged form already used on the
      `error-taxonomy.md` row. **N-CS-I2-01 (adversary nitpick, pass 2, taken):** AC-015 PC-1 and
      AC-016 PC-1 wrongly showed `sbctl router reload --router=<addr>` / `sbctl router drain
      --router=<addr>` — neither verb takes a `--router` sub-flag; the daemon address comes from
      the global `--target` flag (`interface-definitions.md` v1.31 §82/§83), and the frozen
      implementation + tests (feature/S-BL.CLI-SURFACE-COMPLETION @ 1b0e010) already dispatch via
      the bare command. Fixed both PC-1 exemplars to the bare form and both ACs' Precondition
      lines (`--router=<addr>` → `--target=<addr>`, correctly describing daemon reachability);
      swept the rest of both AC blocks, the Task Breakdown, and the File-Change List for any other
      `--router=` occurrence attached to reload/drain — none found. The four remaining `--router=`
      hits in the story are all `paths ping`'s own correct flag (Decision 1, AC-001, Architecture
      Mapping) — `--router` is a real, distinct flag there, overriding `--target`; untouched.
      **Version-pin refresh:** BC-2.06.004's `inputDocuments` comment v1.1 → v1.5 (reflects the
      FO(a)/(d) discharge content now live in that file). `ARCH-INDEX.md` v1.10, `capabilities.md`
      v1.1, and `BC-INDEX.md` v3.5 are newly cited (first version-pinned citations of these files
      in this story) at the Forward Obligations table's row (a)/(b) Status cells introduced by
      this same edit. `acceptance_criteria_count` (16), `estimated_points` (5), and all AC/PC
      semantics are unchanged — the N-CS-I2-01 fix corrects two exemplars to match the
      already-authoritative `interface-definitions.md`/frozen implementation, not a behavior
      change. `input-hash` recomputed via `compute-input-hash --update` (BC-2.06.004 input
      content changed, v1.1 → v1.5). Frontmatter `version` 2.6 → 2.7; new `modified:` entry
      appended (newest-first).
  - date: 2026-07-13
    version: "2.6"
    change: >
      Delivery-phase governance addition (NON-BEHAVIORAL): added the missing "Token Budget
      Estimate (MANDATORY)" section, one of the template sections `validate-template-compliance`
      has flagged since Round 1 — the per-story-delivery playbook's Token Budget Check reads this
      section before every test-writer/implementer spawn and mandates story-writer add it if
      absent. No AC, PC, Decision, or Forward Obligation content touched; the spec-adversarial
      convergence (3/3 clean passes as of pass 9, achieved on v2.5) covers behavioral content and
      STANDS unaffected. Section broken into the three per-story-delivery dispatch passes (stub-
      architect, test-writer, implementer), each a fresh-context dispatch: Pass 1 (stub) ~55k
      tokens (~28% of a 200K window), Pass 2 (failing-test) ~75k (~38%), Pass 3 (TDD
      implementation, the heaviest) ~98k (~49%) — none breaches the 60% split-discussion
      threshold. Methodology: `wc -c`/4 chars-per-token on files as they exist at develop@4c276d9
      for everything already on disk (story spec, precedent production files, `interface-
      definitions.md`, `error-taxonomy.md`, the four BC anchor files, the six production files
      being extended, `mgmt_wire_test.go`); line-count-based estimates, called out explicitly, for
      not-yet-written content (6 new-file stub bodies, ~37 test functions across 5 new + 2
      extended test files implied by the story's cited test names). Noted honestly that Passes 2
      and 3 exceed the template's nominal 20-30% target band — driven by this story's real scope
      (16 ACs, 4 BC anchors, all 3 `error-taxonomy.md` E-CFG-004 variants, 13 `runRouter` call
      sites across 6 files) rather than padding — but the heaviest pass stays under half the
      window; no story split warranted at 5 points. Section inserted between File-Change List and
      Task Breakdown, matching the template's ordering. `input-hash` unchanged — story-body-only
      edit; `--check` confirms no drift. Frontmatter `version` 2.5 → 2.6; new `modified:` entry
      appended (newest-first).
  - date: 2026-07-13
    version: "2.5"
    change: >
      Remediated pass-8 spec-adversarial nitpick N-CS-SP8-01 (orchestrator-adopted as fix; pass 8
      verdict was NITPICK_ONLY, zero findings — 3-clean-pass streak sits at 2/3, unaffected by this
      fix). N-CS-SP8-01: AC-015 PC-1 and AC-016 PC-1 both cited "`router status`/`router metrics`"
      as the connectAndRun exemplars for the reload/drain dispatch shape. Verified against develop@
      4c276d9 (`grep connectAndRun cmd/sbctl/*.go`): `runRouterMetrics` (router_metrics.go) and
      `runPathsList` (paths_list.go) both call `connectAndRun` directly; `runRouterStatus`
      (router_status.go) does NOT — it hand-rolls its own `net.Dialer` + `Authenticate` + dispatch
      of `paths.list`, and its own source comment says it "mirrors connectAndRun's discrimination"
      rather than using it. Citing `router status` as a connectAndRun user was factually wrong.
      Fixed both parentheticals to "(same dial+auth+dispatch shape `router metrics` and `paths
      list` already use)". Swept the rest of the story (Decisions, Task 4, File-Change List, all
      `dial+auth+dispatch` and `net.Dialer`/`hand-roll` hits) for the same mispairing — none found;
      the other `paths list`/`router status` co-mentions (Decision 1's RPC-semantics contrast,
      the drain UX-parity note, Decision 3's bare-top-level-read CLI-shape note) are unrelated to
      connectAndRun and needed no change. Also applied N-CS-SP8-02 (STORY-INDEX.md frontmatter
      `modified` scalar trailed at 2026-07-12; refreshed to 2026-07-13). `input-hash` unchanged —
      story-body-only edit; `--check` confirms no drift. Frontmatter `version` 2.4 → 2.5; new
      `modified:` entry appended (newest-first).
  - date: 2026-07-13
    version: "2.4"
    change: >
      Remediated pass-6 spec-adversarial finding F-CS-SP6-001 (MED, AC-coverage/test-file-
      assignment gap, orchestrator-verified) plus nitpick N-CS-SP6-01. F-CS-SP6-001: `router
      reload`/`router drain` were the only verbs in the story whose client-side CLI dispatch had
      no acceptance criterion and no `cmd/sbctl` test file — the File-Change List already created
      `cmd/sbctl/router_reload.go`/`router_drain.go` and added `reload`/`drain` sub-verb arms to
      `main.go`'s `router` case (a real behavior change: that arm today dispatches only
      `metrics`/`status`, everything else falls through to `usageErrf` exit 2), but no AC and no
      test file existed for either. AC-014 PC-3's E-NET-001/E-ADM-010 client-observed codes were
      also mis-assigned to `cmd/switchboard/router_control_wire_test.go`, a server-side package
      that cannot exercise `cmd/sbctl`'s dispatch. Fixed via four moves: (a) two new client
      happy-path ACs added — AC-015 (`sbctl router reload`, same BC-2.09.001 v1.2 PC-1 anchor as
      AC-011) and AC-016 (`sbctl router drain`, same BC-2.09.002 v1.3 Trigger/PC-1 anchor as
      AC-012), each with a "sub-verb transition pin" postcondition asserting the known sub-verb
      now dispatches (exit 0) while a still-unknown sibling sub-verb (`router bogus`) continues to
      exit 2 via the unchanged default arm — proves the specific transition, not just the end
      state; (b) AC-014 PC-3's test re-homed to a new `cmd/sbctl/router_control_test.go` file, its
      Test names/level/file block split per-postcondition (PC-1/PC-2 stay in
      `router_control_wire_test.go` as genuinely server-side; PC-3 moves); (c) the transition
      pinned explicitly in both new ACs' postcondition 3, tying to the pre-existing exit-2
      baseline; (d) consistency sweep — `acceptance_criteria_count` 14 → 16; File-Change List
      gained the new test-file row and the `router_control_wire_test.go` row's AC-014 scope
      narrowed to "(PC-1, PC-2 only)"; Task 4 retitled and its Red step rewritten to enumerate all
      six ACs and explain both failure modes (server-side no-op select-arm vs. client-side
      not-yet-wired sub-verbs); Anchors Consumed table's reload/drain rows gained AC-015/AC-016;
      Architecture Mapping needed no change (it never named test files). (e) Points: kept at 5 —
      the client CLI functions and sub-verb wiring were always scoped in the File-Change List and
      Architecture Mapping; the gap was AC/test documentation of existing scope, not new
      implementation scope. **N-CS-SP6-01 (nitpick, optional, taken):** AC-011 PC-3's parenthetical
      quoted the `runRouter` entry-guard message in the pre-v4.9 short form
      (`E-CFG-004: --config is required for router mode`) — marked abbreviated and the full
      wrapped literal (`runRouter: E-CFG-004: --config is required for router mode (no config
      loaded)`, error-taxonomy.md v4.9 Variant 2) inlined, per error-taxonomy.md's own v4.9
      changelog (F-CS-SP4-001) confirming the story's Variant 3 literal was already correct and
      only Variant 2's story-side quote needed the disclaimer. `input-hash` unchanged —
      story-body-only edit; `--check` confirms no drift. Frontmatter `version` 2.3 → 2.4; new
      `modified:` entry appended (newest-first).
  - date: 2026-07-12
    version: "2.3"
    change: >
      Remediated pass-3 spec-adversarial findings (F-CS-SP3-001..003) and discharged Forward
      Obligation (c). Context: architect filed Ruling 2 Addendum (rulings doc v1.1 → v1.2 —
      AC-008 PC-3 VINDICATED, stands unchanged, optional traceability citation offered); PO
      landed error-taxonomy.md v4.8 with both the E-CFG-001 client-side variant note
      (F-CS-SP3-003) and the E-CFG-004 `router.reload` defense-in-depth variant (FO(c) discharge,
      F-CS-SP1-001) in the same remediation burst. F-CS-SP3-001 (FO table row (c)): Gate cell
      `Before implementation of AC-011's E-CFG-004 postcondition` → `None — discharged (was
      non-blocking per Ruling 4 Addendum v1.1)`; Status cell `OPEN — non-blocking` →
      `DISCHARGED — landed in error-taxonomy.md v4.8 (2026-07-12, pass-3 remediation burst)`; the
      "Downgraded by Ruling 4 Addendum v1.1" paragraph rewritten to record the discharge.
      F-CS-SP3-002 (Decision 4 reload-bridging bullet): reworded from "must land before this AC
      ships" to the discharged form citing the v4.8 landing. F-CS-SP3-003 (AC-008 PC-3): text
      stands unchanged per the addendum; appended the architect's optional traceability
      parenthetical citing `usageErrf`, the §110/§111 siblings, and the error-taxonomy.md
      E-CFG-001 client-side variant note. Semantic reference-site sweep (searched "Obligation
      (c)", "FO(c)", "E-CFG-004", "error-taxonomy", "taxonomy" and read every hit's sentence,
      not just its tokens — the token-grep approach missed this class twice, F-CS-SP2-002 then
      this pass): AC-011 PC-3's trailing sentence updated from "non-blocking... should still
      land" to "DISCHARGED... landed in v4.8"; the File-Change List's error-taxonomy.md row
      updated to "DISCHARGED, landed in v4.8"; Task 4's note rewritten from a non-blocking
      caveat to a discharged confirmation; the two v2.0/v2.2-era historical `modified:`/Changelog
      references to the old "hard gate"/"non-blocking" language left untouched as accurate
      records of what was true at each prior version; the Delivery Plan Note (POL-005) does not
      mention FO(c) and needed no change; the FO table's general intro sentence ("each gates a
      specific AC...") remains accurate as a statement about all four FOs, not (c) specifically.
      Live rulings-doc pin refreshed v1.1 → v1.2 at all three live binding-source citations
      (frontmatter `inputDocuments` comment, Adjudicated Design Decisions section intro,
      Provenance section) — each now also notes the Ruling 2 Addendum's scope. error-taxonomy.md
      was not previously cited with a version anywhere live in the story (filename-only
      references); the new discharge language introduces its first version-pinned citations,
      all at v4.8. `input-hash` recomputed via `compute-input-hash --update` — the rulings doc
      input changed content (v1.1 → v1.2). Frontmatter `version` 2.2 → 2.3; new `modified:`
      entry appended (newest-first).
  - date: 2026-07-12
    version: "2.2"
    change: >
      Remediated two MED spec-adversarial pass 2 findings (F-CS-SP2-001, F-CS-SP2-002).
      F-CS-SP2-001 (premise/doc-drift): the `runRouter` call-site enumeration was incomplete —
      the Design Constraint parenthetical, File-Change List, and Task 3 named only `main.go`,
      `mgmt_wire_test.go`, `router_drain_test.go`, but a `runRouter(` grep against
      `cmd/switchboard` at develop @ `4c276d9` found thirteen call sites across six files
      (`router_sighup_test.go`, `router_pe_receive_test.go`, `router_pe_connector_test.go` were
      omitted; all package `main`, all would fail to compile once the `drainRequestCh` trailing
      parameter lands). Fixed at all three loci: Design Constraint parenthetical now enumerates
      all six files with per-file call counts (five/one/one/one/one/four); File-Change List
      gained three new rows for the omitted files, each with a call count, and the two existing
      test-file rows gained counts too; Task 3's call-site sentence rewritten to the open,
      drift-durable form — enumerates today's six files but instructs the implementer to
      re-grep `runRouter(` under `cmd/switchboard` at implementation time, since new call sites
      may land before delivery. F-CS-SP2-002 (contradiction): the File-Change List's
      `error-taxonomy.md` row still read "(PO edit, gates AC-011; not a story-writer edit)" —
      the one locus the v2.1 FO(c) downgrade missed. Fixed to "(PO edit, non-blocking per
      Ruling 4 Addendum v1.1; not a story-writer edit)". Grepped the whole story for residual
      "gate"/"gates"/"hard gate" phrasing tied to FO(c); found no other contradictions — the
      Forward Obligations table intro's "each gates a specific AC or a downstream artifact's
      correctness" is a general statement about all four FOs (not specific to (c)'s blocking
      status) and remains accurate; Task 4's "no longer gates this task" note already correctly
      states the downgrade. `input-hash` unchanged — no input file (rulings doc, BC files,
      interface-definitions.md) was touched by this remediation; `--check` confirms no drift.
  - date: 2026-07-12
    version: "2.1"
    change: >
      Propagated architect Ruling 4 Addendum (v1.1, F-CS-SP1-001, spec-adversarial pass 1) into
      AC-011 and its dependents. AC-011 PC-3 reframed as an explicit defense-in-depth guard
      (unreachable via any real daemon startup path — `runRouter`'s entry guard plus `main.go`'s
      `"router"` case together guarantee `configPath != ""` for every router instance reaching
      `wireRouterControlHandlers` registration; mirrors the `E-CFG-011` defensive-annotation
      shape). PC-3's test level downgraded integration → unit (test name unchanged); invocation
      pattern note added (calls `wireRouterControlHandlers` directly with `configPath = ""`, no
      live daemon). Mechanism correction: `wireRouterControlHandlers` gains a `configPath string`
      second parameter — updated at both literal-signature occurrences (Decision 4 registration
      point, AC-013 postcondition 1) plus the Architecture Mapping table row. Forward Obligation
      (c) disposition downgraded `OPEN — hard gate on AC-011` → `OPEN — non-blocking (does not
      gate Task 4 implementation)`; the "only hard implementation gate" paragraph and Task 4's
      gate-check note rewritten to match. Rulings-doc citation pinned to v1.1 at the two locations
      asserting it as binding source. `interface-definitions.md` pin bumped v1.30 → v1.31
      (F-CS-SP1-002 §60 `usage:` prefix fix; AC-009 text itself needed no change) at all
      live-reference citations. BC-2.09.001 (v1.2) / BC-2.09.002 (v1.3) pins reviewed and
      retained per the governance-leaf convention (N-CS-SP1-01) — both files' subsequent bumps
      (v1.2→v1.3, v1.3→v1.4) are traceability-only Stories-cell fills, no PC/AC behavior change,
      so the existing pins are not factually wrong. `input-hash` recomputed via
      `compute-input-hash --update` (the rulings doc input changed).
  - date: 2026-07-12
    version: "2.0"
    change: >
      Elaborated from backlog stub (v1.0, draft, 0 ACs) to sprint-ready. Status: draft → ready;
      wave: backlog → steady-state (S-7.04-FU-SIGHUP-RELOAD lifecycle precedent). All four Open
      Design Obligations resolved by architect ruling
      `.factory/decisions/S-BL.CLI-SURFACE-COMPLETION-rulings.md` (2026-07-12) and transcribed
      here verbatim — story-writer's job was transcription, not re-derivation. 14 ACs firmed
      across five verbs: `paths.ping` (new BC-2.06.004), `admin.svtn.status` (BC-2.07.001 v1.14
      PC-4), `router.reload`/`router.drain` (BC-2.09.001 v1.2 / BC-2.09.002 v1.3 governance
      addenda, resolves `DRIFT-HS006-DRAIN-CLI-MISSING`), `svtn destroy` top-level migration shim
      (no BC change). `bc_traces` += BC-2.06.004. `estimated_points` TBD → 5 (sizing rationale
      below). Four Forward Obligations from the ruling encoded as explicit story tasks the
      adversary MUST police: (a) CAP-022 anchor on BC-2.06.004 provisional pending
      architect/PO confirm-or-mint-CAP-029; (b) ARCH-INDEX SS-06 needs an `internal/mgmt` module
      row; (c) error-taxonomy.md needs the E-CFG-004 "reload not applicable" message variant
      before the reload AC is implemented; (d) VP-TBD-PING-A/B placeholders need real VP numbers
      minted by architect (BC-2.06.003 VP-061/062 precedent). Frontmatter conformed to
      template-mandated superset keys per S-BL.LOOPBACK-FULLSTACK precedent (`epic_id`,
      `inputs`, `input-hash`, `traces_to`, `behavioral_contracts`, `verification_properties`,
      `target_module`, `estimated_days`, `assumption_validations`, `risk_mitigations`).
      Traceability Stories cells filled in BC-2.06.004, BC-2.07.001 (PC-4), BC-2.09.001,
      BC-2.09.002 with this story id (separate BC edits, each version-bumped + changelogged per
      POL-001). `interface-definitions.md` v1.30 already carries the adjudicated CLI listing and
      Registered Verbs rows (PO/architect edit, not touched by this elaboration). No
      line-number citations in story prose (S-BL.PE-RECEIVE-LOOP / S-BL.LOOPBACK-FULLSTACK
      convention) — mechanism-anchor descriptions only; symbols grep-resolved against
      develop@4c276d9.
version: "2.8"
phase: 2
epic: E-7
wave: steady-state
priority: P2
scope_phase: E
estimated_points: 5
inputs:
  - '.factory/decisions/S-BL.CLI-SURFACE-COMPLETION-rulings.md'
  - '.factory/specs/behavioral-contracts/ss-06/BC-2.06.004.md'
  - '.factory/specs/behavioral-contracts/ss-07/BC-2.07.001.md'
  - '.factory/specs/behavioral-contracts/ss-09/BC-2.09.001.md'
  - '.factory/specs/behavioral-contracts/ss-09/BC-2.09.002.md'
  - '.factory/specs/prd-supplements/interface-definitions.md'
input-hash: "d95ecfe"
traces_to: .factory/decisions/S-BL.CLI-SURFACE-COMPLETION-rulings.md
behavioral_contracts:
  - BC-2.06.004
  - BC-2.07.001
  - BC-2.09.001
  - BC-2.09.002
verification_properties:
  - VP-078          # BC-2.06.004 — minted v1.4 (integration); was VP-TBD-PING-A, Forward Obligation (d), DISCHARGED
  - VP-079          # BC-2.06.004 — minted v1.4 (code-audit); was VP-TBD-PING-B, Forward Obligation (d), DISCHARGED
  - VP-048          # BC-2.07.001 PC-4 — two new sibling rows added by Ruling 2
  - VP-038          # BC-2.09.001 — unaffected by the governance-only PC-1 addendum
  - VP-037          # BC-2.09.002 — unaffected by the governance-only Trigger addendum
bc_traces:
  - BC-2.09.001
  - BC-2.09.002
  - BC-2.07.001
  - BC-2.06.004
vp_traces:
  - VP-078
  - VP-079
  - VP-048
  - VP-038
  - VP-037
subsystems: [quality-observability, network-management, deployment-operations]
target_module: "cmd/sbctl, cmd/switchboard, internal/mgmt"
architecture_modules:
  - cmd/sbctl
  - cmd/switchboard
  - internal/mgmt
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on: []
blocks: []
estimated_days: null
assumption_validations: []
risk_mitigations: []   # the ruling's four follow-ups are captured as explicit story obligations below (Forward Obligations), not ASM/R-registry IDs
inputDocuments:
  - '.factory/decisions/S-BL.CLI-SURFACE-COMPLETION-rulings.md'   # BINDING — v1.2 (v1.1 Ruling 4 Addendum, F-CS-SP1-001, AC-011 PC-3 defense-in-depth reframe; v1.2 Ruling 2 Addendum, F-CS-SP3-003, AC-008 PC-3 confirmed unchanged) — 4 rulings, wire contracts, error codes, authority tiers, implementation constraints
  - '.factory/specs/behavioral-contracts/ss-06/BC-2.06.004.md'    # v1.5 — new BC, paths.ping (was v1.1; FO(a)/(d) discharge — CAP-029 re-anchor v1.5, VP-078/VP-079 mint v1.4 — step-4.5 impl pass 2 remediation burst)
  - '.factory/specs/behavioral-contracts/ss-07/BC-2.07.001.md'    # v1.14 — PC-4 admin.svtn.status
  - '.factory/specs/behavioral-contracts/ss-09/BC-2.09.001.md'    # v1.2 — governance addendum, router.reload (pin retained per governance-leaf convention; file now at v1.3, traceability-only)
  - '.factory/specs/behavioral-contracts/ss-09/BC-2.09.002.md'    # v1.3 — governance addendum, router.drain (pin retained per governance-leaf convention; file now at v1.4, traceability-only)
  - '.factory/specs/prd-supplements/interface-definitions.md'     # v1.31 — Registered Verbs rows + CLI listing corrections already executed by PO (bumped from v1.30, F-CS-SP1-002 §60 usage: prefix fix; no AC text change)
  - '.factory/stories/S-7.04-FU-SIGHUP-RELOAD.md'                 # v1.7 — lifecycle/status/versioning convention precedent; shipped sighupCh shape this story bridges into
  - '.factory/stories/S-7.04-FU-DRAIN-WIRE.md'                    # v1.11 — shipped drainCoord/shutdown-sequence shape this story bridges into
  - '.factory/stories/S-BL.LOOPBACK-FULLSTACK.md'                 # v1.1 — template-mandated superset-keys precedent, no-line-number-citation convention
acceptance_criteria_count: 16
backlog_origin:
  source: "F-P5P6-A-005 (Phase 5 Pass 6 Adv-A, 2026-07-03)"
  deferred_from: null
  drift_items_consumed:
    - DRIFT-HS006-DRAIN-CLI-MISSING
  notes: >
    Backlog stub v1.0 collectively annotated five unimplemented sbctl verbs (paths ping, router
    reload, router drain, svtn destroy, svtn status) per F-P5P6-A-005 adjudication
    (annotate-and-defer). Two of five (paths ping, svtn status) carried no governing BC; four
    open design obligations blocked scheduling. All four resolved by architect ruling
    S-BL.CLI-SURFACE-COMPLETION-rulings.md (2026-07-12) — this elaboration transcribes the
    rulings into sprint-ready ACs; story-writer's job here is transcription, not re-derivation,
    per the ruling's own framing ("It does not edit the story... those edits belong to the
    product-owner / story-writer").
---

# S-BL.CLI-SURFACE-COMPLETION: CLI Surface Completion — Dispatch + Wire for Five Verbs

> **Status note:** All four Open Design Obligations from the v1.0 stub are ADJUDICATED
> (`.factory/decisions/S-BL.CLI-SURFACE-COMPLETION-rulings.md`, 2026-07-12) and the BC surface is
> settled (BC-2.06.004 commissioned; BC-2.07.001 v1.14 PC-4; BC-2.09.001 v1.2 / BC-2.09.002 v1.3
> governance addenda). This story is elaborated to sprint-ready and awaits its spec-adversarial
> convergence cycle (not yet dispatched) before TDD implementation begins — mirroring
> `S-7.04-FU-SIGHUP-RELOAD`'s lifecycle (backlog stub → `ready` v1.0 → adversarial version climb
> to v1.7 → merged). Do not implement from the v1.0 stub's "Open Design Obligations" section — it
> is superseded below by "Adjudicated Design Decisions."

## Narrative

- **As an** operator managing a running switchboard fleet via `sbctl`
- **I want** `paths ping`, `svtn status`, `router reload`, `router drain` to actually dispatch
  (not return usage errors), and `svtn destroy`'s top-level form to redirect clearly to its
  canonical form
- **So that** I get a one-shot reachability probe, an SVTN status query, and RPC-triggered
  config-reload/drain — without needing raw OS-signal access to the daemon host — while the
  destructive `svtn destroy` surface stays single-path (no duplicated confirm-gate)

## Context

Five `sbctl` verbs were specified in `interface-definitions.md` but had no CLI dispatch case arm
— they returned unknown-subcommand usage errors (exit 2) as of PR #65. The v1.0 backlog stub
(F-P5P6-A-005, 2026-07-03) annotated all five collectively as `PENDING-S-BL.CLI-SURFACE-COMPLETION`
and logged four open design obligations blocking scheduling: no governing BC for `paths ping` or
`svtn status`; unresolved `--id`-vs-`--name`/confirm-gate ambiguity for `svtn destroy`; unconfirmed
wire verb names for `router reload`/`router drain`.

The architect ruling (2026-07-12, grep-verified against `develop@4c276d9`) resolved all four. The
underlying mechanisms for `router reload`/`router drain` are **already shipped** —
`S-7.04-FU-SIGHUP-RELOAD` (PR #113) built the SIGHUP-triggered reload path; `S-7.04-FU-DRAIN-WIRE`
(PR #120) built the DRAIN-broadcast + SIGTERM shutdown sequence. Both placement notes explicitly
named the RPC-trigger gap as deferred, out-of-scope work for a "follow-on ops-UX story" — this is
that story. `interface-definitions.md` v1.31 already carries the adjudicated CLI listing and the
four new Registered Verbs rows (`admin.svtn.status`, `paths.ping`, `router.reload`, `router.drain`)
— that spec-side edit is done; this story is the implementation-side closure.

**Explicitly out of scope (unchanged from v1.0 stub):**

- `sbctl svtn list` — won't-fix (`S-BL.SVTN-LIST-WIRE`); surface removed.
- `sbctl sessions attach/detach/status` — covered by `S-BL.DISCOVERY-WIRE`.
- `sbctl admin recover` — covered by `S-BL.ADMIN-RECOVER-WIRE`.
- `sbctl version` / `sbctl ping` — covered by `S-BL.PING-VERSION-WIRE`.

## Previous Story Intelligence (MANDATORY)

| Predecessor | Lesson carried forward |
|-------------|--------------------------|
| `S-7.04-FU-SIGHUP-RELOAD` (merged PR #113 @ 950285c) | Ships the `sighupCh` reload path this story's `router.reload` bridges into. `runRouter`'s current signature is `func runRouter(ctx context.Context, w io.Writer, cfg *config.Config, configPath string, sighupCh <-chan os.Signal) error`; every existing call site (production and test) already constructs a **bidirectional** `make(chan os.Signal, 1)` — only `runRouter`'s own parameter type needs to widen to `chan os.Signal`, no call site needs to change. Lifecycle/status/versioning convention (backlog stub → `ready` v1.0 → adversarial version climb) followed here. |
| `S-7.04-FU-DRAIN-WIRE` (merged PR #120 @ f73676d) | Ships the `drainCoord`/shutdown-sequence this story's `router.drain` bridges into. The `shutdown:` label sequence (drain broadcast → per-node flush → `ingressCancel()` → `mgmtSrv.Shutdown`) is reached today only via `ctx.Done()`. |
| `S-BL.LOOPBACK-FULLSTACK` (v1.1, draft/unscheduled) | Template-mandated frontmatter superset-keys precedent (`epic_id`, `inputs`, `input-hash`, `traces_to`, `behavioral_contracts`, `verification_properties`, `target_module`, `estimated_days`, `assumption_validations`, `risk_mitigations`) adopted here. Also: "story-writer's job here is transcription, not re-derivation" framing when a binding architect ruling exists — applied identically to this story's relationship with its rulings doc. |
| `S-BL.PE-RECEIVE-LOOP` (merged PR #118) | House convention: every new symbol claim must be grep-resolved or marked "(new — defined by this story)"; **line-number citations are forbidden in story prose** — use mechanism-anchor descriptions. Followed throughout this story. |
| BC-2.05.004 / `admin.key.list-keys` (S-6.06) | Direct precedent for Ruling 2's authority carve-out: a read-only accessor living inside a destructive-lifecycle BC as an added precondition/authority carve-out (F-L2-003), reusing `resolveCallerAdmissionAnyRole` rather than the control-only gate. |
| BC-2.06.003 VP-TBD-A/VP-TBD-B → VP-061/VP-062 (architect, v1.3) | Direct precedent for Forward Obligation (d): placeholder VP IDs are legitimate at story-authorship time and do not block implementation; the architect mints real VP numbers in a later BC version, "not blocking implementation." |
| `interface-definitions.md` §59 `svtn create` REMOVED (PR #62) | Direct precedent for Ruling 3 / AC-009: a destructive top-level alias that duplicates `sbctl admin svtn <verb>` is retired to a redirect/removal, not maintained as a parallel code path. |

## Adjudicated Design Decisions

Transcribed from `.factory/decisions/S-BL.CLI-SURFACE-COMPLETION-rulings.md` v1.2 (binding — v1.1
added the Ruling 4 Addendum, F-CS-SP1-001, reframing AC-011 PC-3 as a defense-in-depth guard; v1.2
adds the Ruling 2 Addendum, F-CS-SP3-003, confirming AC-008 PC-3 stands unchanged). Where this
story and the rulings doc appear to diverge, the rulings doc governs. Each entry below carries
the load-bearing constraints inline — the implementer should not need to re-open the rulings doc
for the common path.

### Decision 1 (Ruling 1) — `paths ping`: new RPC `paths.ping`, new BC-2.06.004

**Not** a reuse of `paths.list`. `paths ping` is a one-shot, on-demand reachability + latency probe
of an arbitrarily-dialed target; `paths list`/`router status` report historical, EWMA-smoothed
per-path metrics accumulated by a `PathTracker` over time. Reusing `paths.list` and discarding its
body to derive a timing figure would be a category mismatch in the RPC-name-based audit trail.

- **Wire verb:** `paths.ping`. Request args `{}` (empty — the daemon dialed via `--router=<addr>`
  IS the probe target by construction). Response `{"pong": true}`.
- **CLI-synthesized output** (not on the wire, computed by `cmd/sbctl` around the dial+auth+dispatch
  sequence): `{"router": "<addr>", "rtt_ms": <float64>}` — measured client-side, dial-start to
  response-decode-complete.
- **Authority:** Tier-1 operator-key auth only — same bar as `paths.list`/`router.metrics`/
  `router.status`; no additional Tier-2 role gate.
- **Registration:** new handler (e.g. `mgmt.RegisterPingHandler`) called from `wireMetricsHandlers`
  — available on every daemon mode that already wires metrics handlers (`runRouter`, `runAccess`,
  `runConsole`, `runControl`), since `paths ping` targets an arbitrary daemon and is not
  router-mode-exclusive (contrast Decision 4).
- **CLI:** new `runPathsPing(ctx, target, keyPath, useJSON, args, sio)` wired into the existing
  `paths` case arm in `cmd/sbctl/main.go` alongside `list`.
- **Reachability vs. slow semantics:** unreachable-before-connection → E-NET-001, exit 1. Auth
  failure after connection → E-ADM-010, exit 1. A connection that succeeds but is slow is **not an
  error** — `rtt_ms` simply reports a larger number, exactly like `ping(8)`. `paths ping` performs
  **no** quality classification (no green/yellow/red field) — that remains `router.status`'s job.

### Decision 2 (Ruling 2) — `svtn status`: extend BC-2.07.001 with PC-4, wire `admin.svtn.status`

Extend, don't commission — direct precedent is `admin.key.list-keys` living inside BC-2.05.004
alongside destructive key-lifecycle ops as an added authority carve-out (F-L2-003). `svtn status`
is the symmetric case: same manager (`SVTNManager`), same boundary package
(`cmd/switchboard/admin_handlers.go`), a new read accessor over existing state.

- **Wire verb:** `admin.svtn.status`, registered in `BuildAdminHandlers` alongside create/destroy —
  needs `*svtnmgmt.SVTNManager`, which exists only on the control-mode daemon (`runControl`).
  Router/access/console pass nil admin handlers (ADR-004) and correctly return E-RPC-010.
- **Request args:** `{"name": "<svtn-name>"}`. **Response data:**
  `{"svtn_id": "<hex>", "name": "<svtn-name>", "created_at": "<RFC3339>", "key_counts": {"control": <n>, "console": <n>, "access": <n>}}`.
- **Authority:** any admitted role (control, console, or access) in the target SVTN, OR
  operator-set member, OR bootstrap key — reuse `resolveCallerAdmissionAnyRole` verbatim (the same
  function `admin.key.list-keys` already uses). The admission gate still applies (CWE-862 defense
  against cross-SVTN roster/existence enumeration — mirrors BC-2.05.004 EC-008); only the
  control-only **authority** gate is skipped.
- **Error codes:** E-SVTN-003 (not found — reuse the existing `mapAdminError` `ErrSVTNNotFound` arm),
  E-CFG-001 (missing `--name`), E-ADM-009 (admission failure).
- **Why NOT session/health data:** `admin_handlers.go`'s own package header states the purity
  boundary explicitly — `internal/session` is a forbidden import. The response schema uses only
  fields `SVTNManager` already exposes (`SVTN{ID, Name, CreatedAt}` via `SVTNByName`, plus
  role-grouped counts derived from `ListKeys`, already used by `admin.key.list-keys`). No health
  indicator is proposed — there is no accessible signal to compute one from at this boundary.
- **`--id` vs `--name`:** `SVTNManager` is exclusively name-keyed (`m.svtns map[string]SVTN`, looked
  up via `SVTNByName`) — no hex-ID reverse index exists anywhere in the package. CLI flag is
  `--name=<svtn-name>`, matching every other `admin svtn`/`admin key` command family.
- **CLI dispatch:** read-only and non-destructive, so none of Decision 3's confirm-gate duplication
  risk applies. `sbctl svtn status --name=<svtn-name>` (top-level `svtn` case arm) is a genuine
  standalone dispatch directly to `admin.svtn.status` — **not** routed through `sbctl admin`
  framing, exactly as `paths list`/`router status` are already bare top-level reads.

### Decision 3 (Ruling 3) — `svtn destroy` top-level form: migration shim, not a parallel alias

`sbctl svtn destroy` (top-level) is a migration shim. It does **not** implement `--id`, does
**not** dispatch `admin.svtn.destroy`, and does **not** duplicate the confirm-gate. It always
returns a usage error (exit 2) redirecting to the canonical form.

- **Direct precedent:** `sbctl svtn create` was **removed entirely**, not aliased, for the same
  reason (`interface-definitions.md` §59, PR #62).
- **`--id=<svtn_id>` cannot be honored literally:** same name-keyed-only constraint as Decision 2.
  Silently reinterpreting `--id` to mean "name" would be a footgun on a **destructive** command.
- **Duplicating `runDestroyConfirmGate` doubles a security-sensitive surface for no operator
  benefit** — `sbctl admin svtn destroy` already implements it correctly and is the documented
  canonical form.
- **Implementation:** in the new `runSvtn` dispatch function's `destroy` sub-verb:
  ```go
  case "destroy":
      return usageErrf("svtn destroy: use 'sbctl admin svtn destroy --name=<svtn-name> [--confirm=<svtn-short-id>|--yes]'")
  ```
  No RPC dispatch, no `--id`/`--name` flag parsing at all — the shim never parses either flag, so
  the `--id`-vs-`--name` discrepancy is moot in the implementation.
- **No BC change** — BC-2.07.001 PC-3 already fully governs `admin.svtn.destroy`; this ruling only
  concerns the top-level CLI alias surface, never itself a BC anchor point.

### Decision 4 (Ruling 4) — `router reload` / `router drain`: new router-mode RPCs, in scope

New wire verbs `router.reload` / `router.drain`, registered on the router daemon only, via a new
`wireRouterControlHandlers` function called from `runRouter` alongside `wireMetricsHandlers`. Both
handlers bridge into the **already-shipped** SIGHUP-reload and SIGTERM-drain code paths via new
channels threaded the same way `sighupCh` already is — **no reload/drain logic is duplicated**.
This closes `DRIFT-HS006-DRAIN-CLI-MISSING`. Descoping was considered and rejected: the missing
piece is bounded, low-risk, and directly named as this story's job by two prior architect
placement notes (`S-7.04-FU-DRAIN-WIRE-placement-note.md`, `S-7.04-FU-SIGHUP-RELOAD-placement-note.md`).

- **Wire verb names:** `router.reload`, `router.drain` — match the CLI sub-verb names already
  dispatched from the `router` case arm (alongside `metrics`/`status`).
- **Registration point:** new **router-mode-exclusive** function
  `wireRouterControlHandlers(srv *mgmt.Server, configPath string, sighupCh chan os.Signal, drainRequestCh chan struct{}) error`,
  called from `runRouter` at the same phase as `wireMetricsHandlers`, **before** `serveMgmtServer`
  (register-before-serve invariant, F-P2L1-001). `runAccess`/`runConsole`/`runControl` never call
  it — meaningless on those modes (no `sighupCh`/drain-coordinator concept). The `configPath`
  parameter lets `router.reload`'s handler check `configPath == ""` synchronously for AC-011
  PC-3's defense-in-depth guard, without touching `sighupCh` or the select loop (mechanism
  correction, Ruling 4 Addendum v1.1). See the dedicated Design Constraint section below for the
  exact `runRouter` signature change.
- **Reload bridging (no new channel):** `router.reload`'s handler synthesizes the exact signal the
  SIGHUP path already consumes: `select { case sighupCh <- syscall.SIGHUP: default: }` (matches
  `signal.Notify`'s own coalescing semantics — a reload already pending silently drops the second
  request). Unlike a bare SIGHUP (which silently no-ops when `configPath == ""`), the RPC handler
  has a response channel and surfaces that case synchronously as **E-CFG-004: reload not
  applicable: daemon started without --config** rather than a silent `{"accepted": true}` no-op
  (Forward Obligation (c) — the error-taxonomy.md reload variant landed in v4.8, DISCHARGED;
  F-CS-SP1-001 / Ruling 4 Addendum v1.1).
- **Drain bridging (genuinely new channel):** `drainRequestCh chan struct{}` (buffered 1), threaded
  into `runRouter` the same way `sighupCh` was threaded by `S-7.04-FU-SIGHUP-RELOAD`. A third
  select-loop arm: `case <-drainRequestCh: goto shutdown`. Handler:
  `select { case drainRequestCh <- struct{}{}: default: }` (already-in-flight drain → no-op).
  Rejected alternative: threading `cancel func()` directly into the RPC layer — considered and
  rejected, would hand `main.go`'s exclusive cancel ownership to an RPC handler closure with no
  testing benefit over the channel approach.
- **Wire contract (both):** request `{}`, response `{"accepted": true}` — fire-and-forget, matching
  UX parity with sending a raw OS signal (a `kill -HUP`/`kill -TERM` sender gets no synchronous
  completion confirmation either; operator confirms via logs / `router status` afterward). A
  synchronous wait-for-completion variant is a future enhancement, out of proportion to this
  story's P2 priority.
- **`router.drain` connection-teardown note (binding on the implementer/test-writer):** because
  drain triggers the full shutdown sequence, the RPC connection itself will likely be severed as
  the daemon exits shortly after — treat "connection reset" following (or even without) a
  `{"accepted": true}` as an **expected outcome, not a protocol error**. Mirrors BC-2.09.002 PC-3's
  existing best-effort-delivery framing extended to the triggering RPC itself.
- **Authority:** Tier-1 operator-key auth only — the same (and only) gate `paths.list`/
  `router.metrics`/`router.status` already use on this daemon. Router mode has no
  `SVTNManager`/`RoleControl` concept at all; introducing a new "router-operator" role would be
  disproportionate to this story and neither BC's Trigger requests it.
- **Error codes:** E-NET-001 (unreachable), E-ADM-010 (auth failure) — shared connection-error
  codes. Reload adds E-CFG-004 for the no-config-loaded case. No new error codes for drain.

## Design Constraint: `runRouter` Signature Widening (Ruling 4)

**Binding.** Current signature (shipped by `S-7.04-FU-SIGHUP-RELOAD`):

```go
func runRouter(ctx context.Context, w io.Writer, cfg *config.Config,
               configPath string, sighupCh <-chan os.Signal) error
```

After this story:

```go
func runRouter(ctx context.Context, w io.Writer, cfg *config.Config,
               configPath string, sighupCh chan os.Signal, drainRequestCh chan struct{}) error
```

Two changes: (1) `sighupCh` widens from receive-only (`<-chan os.Signal`) to bidirectional
(`chan os.Signal`) — every existing call site (production `main.go` and every test) already
constructs a bidirectional `make(chan os.Signal, 1)` or passes `nil`, either of which is valid for
both the old and new parameter type, so **only the parameter type itself needs to change; no call
site needs to change** for `sighupCh`. (2) `drainRequestCh chan struct{}` is a new trailing
parameter — **every** call site (production and test) DOES need to add this argument, mirroring
the exact call-site-update pattern `S-7.04-FU-SIGHUP-RELOAD` used when it added
`configPath`/`sighupCh`. Full enumeration as of a `runRouter(` grep against `cmd/switchboard` at
develop @ `4c276d9` — thirteen call sites across six files: `mgmt_wire_test.go` (five), `main.go`
(one), `router_drain_test.go` (one), `router_sighup_test.go` (one), `router_pe_receive_test.go`
(one), `router_pe_connector_test.go` (four). Every one of these files is package `main` and will
fail to compile once the trailing parameter lands, so all six are load-bearing for Task 3's "all
existing tests remain green" gate, not just the three cited in earlier drafts of this story.

`main.go`'s `"router"` case body constructs `drainRequestCh := make(chan struct{}, 1)` alongside
the existing `sighupCh` construction, and passes both into `runRouter`. The select loop
(currently two cases: `ctx.Done()`, `sighupCh`) gains a third arm:

```go
for {
    select {
    case <-ctx.Done():
        goto shutdown
    case <-sighupCh:
        // existing reload logic, unchanged
    case <-drainRequestCh:
        goto shutdown
    }
}
```

## Acceptance Criteria

### AC-001 — `paths ping` happy path: dial, authenticate, measure RTT

**BC Anchor:** BC-2.06.004 PC-1, Invariant 1

**Precondition:** A daemon is running and reachable at `--router=<addr>`; the operator's key
Tier-1-authenticates.

**Postconditions:**

1. `sbctl paths ping --router=<addr>` dials `<addr>` directly, overriding `--target`.
2. The daemon Tier-1-authenticates the caller (no additional Tier-2 gate).
3. `paths.ping` is issued with empty request args (`{}`); the daemon returns `{"pong": true}`.
4. sbctl reports `{"router": "<addr>", "rtt_ms": <float64>}`, `rtt_ms` measured client-side from
   dial-start to response-decode-complete; exit code 0.

**Test name:** `TestPathsPing_HappyPath_ReportsRTT`
**Test level:** integration
**Test file:** `cmd/sbctl/paths_ping_test.go` (new)

---

### AC-002 — `paths ping` error paths: unreachable and auth failure

**BC Anchor:** BC-2.06.004 PC-2, PC-3, EC-001, EC-002

**Postconditions:**

1. Target daemon unreachable before connection → E-NET-001 "daemon unreachable: <address>"; exit 1.
2. Connection succeeds but Tier-1 authentication fails → E-ADM-010; exit 1. No `paths.ping` RPC is
   dispatched (auth failure occurs before command dispatch).

**Test names:** `TestPathsPing_Unreachable_ENET001`, `TestPathsPing_AuthFailure_EADM010`
**Test level:** integration
**Test file:** `cmd/sbctl/paths_ping_test.go`

---

### AC-003 — `paths ping` slow round trip is not an error; no quality classification

**BC Anchor:** BC-2.06.004 PC-4, EC-003, Invariant 2

**Postconditions:**

1. A connection that succeeds but measures high latency is **not** an error — `rtt_ms` reports the
   measured (larger) value; exit 0.
2. `paths.ping`'s response and sbctl's synthesized output never carry a quality/status field
   (no green/yellow/red) — `router.status` (BC-2.06.003 PC-3) remains the exclusive owner of
   quality classification.

**Test name:** `TestPathsPing_SlowRoundTrip_NotAnError_NoQualityField`
**Test level:** integration
**Test file:** `cmd/sbctl/paths_ping_test.go`

---

### AC-004 — `paths.ping` RPC handler registration and authority

**BC Anchor:** BC-2.06.004 Invariant 1, Trigger

**Postconditions:**

1. A new handler (e.g. `mgmt.RegisterPingHandler`) is called from `wireMetricsHandlers`, making
   `paths.ping` available on **every** daemon mode that already wires metrics handlers: `runRouter`,
   `runAccess`, `runConsole`, `runControl`.
2. `paths.ping` requires no additional Tier-2 authority beyond standard Tier-1 operator-key
   authentication — the same bar as `paths.list`/`router.metrics`/`router.status`.
3. The handler performs zero per-path metrics reads/writes — no `PathTracker` interaction; request
   `{}` in, response `{"pong": true}` out, no other side effect (VP-079).

**Test names:** `TestWireMetricsHandlers_RegistersPingOnEveryMode`,
`TestPingHandler_EmptyArgsIn_PongOut_ZeroPathTrackerInteraction`
**Test level:** unit (handler) + integration (per-mode registration)
**Test file:** `internal/mgmt/register_metrics_test.go` (extended) or `register_ping_test.go` (new);
`cmd/switchboard/metrics_wire_test.go` (extended)

---

### AC-005 — `admin.svtn.status` happy path

**BC Anchor:** BC-2.07.001 PC-4 (happy-path Canonical Test Vector)

**Precondition:** SVTN `mynet` exists; caller is admitted to `mynet` in any role.

**Postconditions:**

1. `sbctl svtn status --name=mynet` returns
   `{"svtn_id":"<hex>","name":"mynet","created_at":"<RFC3339>","key_counts":{"control":1,"console":0,"access":2}}`;
   exit 0.
2. `key_counts` are grouped by role, scoped exclusively to the target SVTN (VP-048 sibling row 1).

**Test name:** `TestAdminSVTNStatus_HappyPath_KeyCounts`
**Test level:** integration
**Test file:** `cmd/switchboard/admin_handlers_test.go` (extended)

---

### AC-006 — `admin.svtn.status` error paths: not-found and admission-denied

**BC Anchor:** BC-2.07.001 PC-4 (not-found and admission-denied Canonical Test Vectors)

**Postconditions:**

1. `sbctl svtn status --name=doesnotexist` → E-SVTN-003 "SVTN not found: doesnotexist"; exit 1.
2. A caller with a valid operator key admitted only to a **different** SVTN (not `mynet`, not
   operator-set, not bootstrap) → E-ADM-009 "insufficient authority for operation
   admin.svtn.status: key <fp> has role <role>"; exit 1. SVTN roster/existence is **not**
   disclosed — the admission gate fires before status is computed (CWE-862 defense, mirrors
   BC-2.05.004 EC-008; VP-048 sibling row 2).

**Test names:** `TestAdminSVTNStatus_NotFound_ESVTN003`,
`TestAdminSVTNStatus_AdmissionDenied_EADM009_NoExistenceOracleLeak`
**Test level:** integration
**Test file:** `cmd/switchboard/admin_handlers_test.go`

---

### AC-007 — `admin.svtn.status` purity boundary and mode exclusion

**BC Anchor:** BC-2.07.001 PC-4 (ARCH-09 purity note); ADR-004

**Postconditions:**

1. The response schema (`svtn_id`, `name`, `created_at`, `key_counts`) never carries session or
   health-indicator fields — `internal/session` remains a forbidden import for
   `cmd/switchboard/admin_handlers.go`.
2. `admin.svtn.status` is registered in `BuildAdminHandlers`, control-mode-daemon-only (needs
   `*svtnmgmt.SVTNManager`). Router, access, and console modes pass nil admin handlers and
   correctly return E-RPC-010 (unknown command) for `admin.svtn.status`.

**Test names:** `TestAdminSVTNStatus_ResponseExcludesSessionHealthFields`,
`TestAdminSVTNStatus_NonControlMode_NilAdminHandlers_ERPC010`
**Test level:** unit (schema) + integration (mode exclusion)
**Test file:** `cmd/switchboard/admin_handlers_test.go`

---

### AC-008 — `sbctl svtn status` CLI dispatch: bare top-level, `--name` flag

**BC Anchor:** BC-2.07.001 PC-4 (CLI dispatch note)

**Postconditions:**

1. `sbctl svtn status --name=<svtn-name>` dispatches directly to `admin.svtn.status` — **not**
   routed through `sbctl admin` framing (matches the `paths list`/`router status` bare top-level
   read shape).
2. The flag is `--name`, not `--id` (`SVTNManager` is exclusively name-keyed).
3. Missing `--name` → **E-CFG-001** (client-side flag validation via `usageErrf`, exit 2 — same
   pattern as `sbctl admin list-keys --svtn` (interface-definitions.md §111) and
   `sbctl admin key expire --after` (§110); see error-taxonomy.md E-CFG-001 client-side variant
   note, F-CS-SP3-003).

**Test name:** `TestSvtnStatus_CLIDispatch_BareTopLevel_NameFlag`
**Test level:** integration
**Test file:** `cmd/sbctl/svtn_test.go` (new)

---

### AC-009 — `sbctl svtn destroy` top-level migration shim

**BC Anchor:** none (Decision 3 — CLI-surface documentation only, not a BC anchor point)

**Postconditions:**

1. `sbctl svtn destroy` (any arguments) recognizes the `destroy` sub-verb and returns a usage error
   (exit 2) with the exact redirect text: `svtn destroy: use 'sbctl admin svtn destroy --name=<svtn-name> [--confirm=<svtn-short-id>|--yes]'`.
2. No `--id`/`--name` flag parsing occurs — the shim never inspects either flag.
3. No RPC is dispatched; `admin.svtn.destroy` is never called from this code path.
4. `runDestroyConfirmGate` is never invoked from the top-level `svtn destroy` shim — the confirm
   gate remains exclusively owned by `sbctl admin svtn destroy`.

**Test names:** `TestSvtnDestroy_TopLevelShim_UsageErrorRedirect_Exit2`,
`TestSvtnDestroy_TopLevelShim_NoRPCDispatch`
**Test level:** unit
**Test file:** `cmd/sbctl/svtn_test.go`

---

### AC-010 — `sbctl svtn` top-level case arm dispatch

**BC Anchor:** none (Scope item 1 — CLI dispatch structure)

**Postconditions:**

1. `cmd/sbctl/main.go` gains a new top-level `case "svtn":` (alongside `sessions`, `paths`,
   `router`, `console`, `admin`) dispatching to a new `runSvtn` function.
2. `runSvtn` routes `status` → AC-005..AC-008 dispatch, `destroy` → AC-009 shim.
3. An unknown sub-verb under `svtn` returns a usage error, exit 2 (same shape as the existing
   `paths`/`router` case arms' default arms).

**Test name:** `TestSvtn_UnknownSubVerb_UsageErrorExit2`
**Test level:** unit
**Test file:** `cmd/sbctl/svtn_test.go`

---

### AC-011 — `router.reload` bridges into the shipped SIGHUP-reload path

**BC Anchor:** BC-2.09.001 v1.2 PC-1 (RPC-trigger note)

**Postconditions:**

1. The `router.reload` handler synthesizes a signal onto the (now-bidirectional) `sighupCh` —
   `select { case sighupCh <- syscall.SIGHUP: default: }` — coalescing exactly like
   `signal.Notify`'s own semantics when a reload is already pending.
2. From that synthesis point forward, the RPC-triggered and SIGHUP-OS-signal-triggered reload
   paths are code-path-identical (same `sighupCh` consumer, same fail-closed reload-dispatch
   logic shipped by `S-7.04-FU-SIGHUP-RELOAD`).
3. **Defense-in-depth guard (unreachable via any real daemon startup path — presence at runtime
   would indicate a code defect, not an operator condition).** `runRouter`'s entry guard in
   `cmd/switchboard/mgmt_wire.go` (`cfg == nil` → `E-CFG-004: --config is required for router
   mode` — abbreviated; the full wrapped literal is `runRouter: E-CFG-004: --config is required
   for router mode (no config loaded)`, error-taxonomy.md v4.9 Variant 2) and the `"router"` case
   in `cmd/switchboard/main.go` (`cfg` set iff
   `*configPath != ""`) together guarantee `configPath != ""` for every router instance that
   reaches `wireRouterControlHandlers` registration. `router.reload`'s handler nonetheless
   checks `configPath == ""` before synthesizing onto `sighupCh`, returning **E-CFG-004: reload
   not applicable: daemon started without --config** synchronously if that invariant is ever
   violated (e.g. by a future refactor decoupling `cfg` construction from `configPath`). Mirrors
   the `E-CFG-011` defensive-annotation shape (the E-CFG-011 row of error-taxonomy.md). Forward
   Obligation (c) — the error-taxonomy.md E-CFG-004 message-variant documentation — is
   **DISCHARGED**: the variant landed in error-taxonomy.md v4.8 (2026-07-12, pass-3 remediation
   burst; see Forward Obligations table below). Nothing gates this postcondition's implementation.

**Invocation pattern (PC-3):** `TestRouterReload_NoConfigLoaded_ECFG004` calls
`wireRouterControlHandlers` (or its registered `router.reload` handler) directly with
`configPath = ""` — no live `runRouter`/daemon required.

**Test names:** `TestRouterReload_BridgesToSighupCh_CodePathIdentical` (PC-1, PC-2),
`TestRouterReload_NoConfigLoaded_ECFG004` (PC-3)
**Test level:** integration (PC-1, PC-2) + unit (PC-3)
**Test file:** `cmd/switchboard/router_control_wire_test.go` (new)

---

### AC-012 — `router.drain` bridges into the shipped shutdown sequence

**BC Anchor:** BC-2.09.002 v1.3 Trigger/PC-1 (RPC-trigger note)

**Postconditions:**

1. The `router.drain` handler sends on the new `drainRequestCh` —
   `select { case drainRequestCh <- struct{}{}: default: }` (already-in-flight drain → no-op).
2. The select loop's third arm (`case <-drainRequestCh: goto shutdown`) reaches the same
   `shutdown:` label as `ctx.Done()`/SIGTERM — same drain-broadcast, per-node-flush, exit sequence;
   same exit parity as the OS-signal path.
3. The RPC connection is expected to be severed as the daemon exits shortly after — a
   "connection reset" observed by the client following (or even without) a `{"accepted": true}`
   response is treated as an **expected outcome, not a protocol error** (extends BC-2.09.002 PC-3's
   best-effort-delivery framing to the triggering RPC itself).

**Test names:** `TestRouterDrain_BridgesToShutdownSequence_ViaDrainRequestCh`,
`TestRouterDrain_ConnectionSeveredAfterAccepted_NotAnError`
**Test level:** integration
**Test file:** `cmd/switchboard/router_control_wire_test.go`

---

### AC-013 — `router.reload`/`router.drain` registration: router-mode-exclusive, register-before-serve

**BC Anchor:** Decision 4 (registration point); F-P2L1-001

**Postconditions:**

1. A new `wireRouterControlHandlers(srv *mgmt.Server, configPath string, sighupCh chan os.Signal, drainRequestCh chan struct{}) error`
   is called from `runRouter` at the same phase as `wireMetricsHandlers`, **before**
   `serveMgmtServer` starts the `Serve` goroutine (register-before-serve invariant). `runRouter`
   passes its own (already-guard-verified-non-empty) `configPath` argument through unchanged —
   this is not a further widening of `runRouter`'s own signature beyond item 3 below.
2. `runAccess`, `runConsole`, `runControl` never call `wireRouterControlHandlers`. Both
   `router.reload` and `router.drain` return E-RPC-010 (unknown command) when dispatched against
   those modes.
3. `runRouter`'s `sighupCh` parameter widens from `<-chan os.Signal` to `chan os.Signal`; a new
   trailing `drainRequestCh chan struct{}` parameter is added. `main.go`'s `"router"` case body
   constructs `drainRequestCh := make(chan struct{}, 1)` alongside the existing `sighupCh`
   construction and passes both into `runRouter`. Every existing test call site is updated with
   the new trailing argument (mirroring the `S-7.04-FU-SIGHUP-RELOAD` call-site-update pattern).

**Test names:** `TestWireRouterControlHandlers_RegisterBeforeServe`,
`TestWireRouterControlHandlers_RouterModeExclusive_OtherModesERPC010`,
`TestRunRouter_DrainRequestChThirdSelectArm_ReachesShutdown_SameExitParityAsSIGTERM`
**Test level:** integration
**Test file:** `cmd/switchboard/router_control_wire_test.go`; `cmd/switchboard/mgmt_wire_test.go` (extended)

---

### AC-014 — `router.reload`/`router.drain` wire contract

**BC Anchor:** Decision 4 (wire contract); BC-2.09.001 v1.2, BC-2.09.002 v1.3

**Postconditions:**

1. Both verbs require Tier-1 operator-key authentication only — no stricter Tier-2 gate is
   available or introduced (router mode has no `SVTNManager`/`RoleControl` concept).
2. Request args for both: `{}`. Response data for both: `{"accepted": true}` — fire-and-forget,
   no synchronous completion confirmation.
3. Standard shared connection-error codes apply: E-NET-001 (unreachable), E-ADM-010 (auth
   failure).

**Test names:** `TestRouterReloadDrain_TierOneAuthOnly_FireAndForgetAcceptedTrue` (PC-1, PC-2),
`TestRouterReloadDrain_Unreachable_ENET001` (PC-3),
`TestRouterReloadDrain_AuthFailure_EADM010` (PC-3)
**Test level:** integration
**Test file:** `cmd/switchboard/router_control_wire_test.go` (PC-1, PC-2 — genuinely server-side:
Tier-1-only authority requirement and the RPC wire contract itself);
`cmd/sbctl/router_control_test.go` (new) (PC-3 — re-homed per F-CS-SP6-001: E-NET-001
"daemon unreachable" and E-ADM-010 auth failure are codes the *client* observes when
`connectAndRun`'s dial/auth step fails before any RPC is dispatched; `router_control_wire_test.go`
is a server-side package that structurally cannot exercise `cmd/sbctl`'s dispatch or these
client-side emissions)

---

### AC-015 — `sbctl router reload` CLI dispatch: happy path + sub-verb transition pin

**BC Anchor:** BC-2.09.001 v1.2 PC-1 (RPC-trigger note) — same anchor as AC-011

**Precondition:** A router-mode daemon is running and reachable at `--target=<addr>`; the
operator's key Tier-1-authenticates.

**Postconditions:**

1. `sbctl router reload` dispatches `router.reload` via the existing
   `connectAndRun` pattern (same dial+auth+dispatch shape `router metrics` and `paths list`
   already use).
2. sbctl prints the `{"accepted": true}` response; exit code 0.
3. **Sub-verb transition pin.** Before this story, the `router` case arm's sub-verb switch
   dispatches only `metrics`/`status`; any other sub-verb — including `reload` — falls through
   to the default arm's usage error (`usageErrf`), exit 2. This AC moves `reload` out of that
   default bucket. The test asserts both sides of the boundary in one run: `sbctl router reload`
   exits 0 via real dispatch, while `sbctl router bogus` (a still-genuinely-unknown sub-verb)
   continues to exit 2 via the unchanged default arm — proving `reload`'s specific transition,
   not a change to the case arm's default behavior for everything else.

**Test names:** `TestRouterReload_CLIDispatch_HappyPath_AcceptedTrue`,
`TestRouterReload_SubVerbTransition_KnownDispatchesUnknownStillExit2`
**Test level:** integration
**Test file:** `cmd/sbctl/router_control_test.go` (new)

---

### AC-016 — `sbctl router drain` CLI dispatch: happy path + sub-verb transition pin

**BC Anchor:** BC-2.09.002 v1.3 Trigger/PC-1 (RPC-trigger note) — same anchor as AC-012

**Precondition:** A router-mode daemon is running and reachable at `--target=<addr>`; the
operator's key Tier-1-authenticates.

**Postconditions:**

1. `sbctl router drain` dispatches `router.drain` via the existing
   `connectAndRun` pattern (same dial+auth+dispatch shape `router metrics` and `paths list`
   already use).
2. sbctl prints the `{"accepted": true}` response; exit code 0. Per AC-012 PC-3 and BC-2.09.002
   PC-3's best-effort-delivery framing, a "connection reset" observed instead of (or after) the
   response is an **expected outcome, not a protocol error** — this AC's test tolerates either
   observed outcome as passing.
3. **Sub-verb transition pin.** Before this story, the `router` case arm's sub-verb switch
   dispatches only `metrics`/`status`; any other sub-verb — including `drain` — falls through to
   the default arm's usage error (`usageErrf`), exit 2. This AC moves `drain` out of that default
   bucket. The test asserts both sides of the boundary in one run: `sbctl router drain` exits 0
   (or tolerates connection-reset per postcondition 2) via real dispatch, while `sbctl router
   bogus` (a still-genuinely-unknown sub-verb) continues to exit 2 via the unchanged default arm.

**Test names:** `TestRouterDrain_CLIDispatch_HappyPath_AcceptedTrueOrConnReset`,
`TestRouterDrain_SubVerbTransition_KnownDispatchesUnknownStillExit2`
**Test level:** integration
**Test file:** `cmd/sbctl/router_control_test.go`

## Forward Obligations (tracked as story tasks — the adversary MUST police these)

These four follow-ups originate directly from the rulings doc's per-ruling "BC action for PO" /
"Implementation constraints" notes. They are not optional cleanup — each gates a specific AC or a
downstream artifact's correctness, and each is a distinct owner/timing combination.

| # | Obligation | Owner | Gate | Status |
|---|-----------|-------|------|--------|
| (a) | BC-2.06.004's `CAP-022` capability anchor is provisional — Ruling 1 did not mint a dedicated capability. Architect/PO must confirm CAP-022 as the correct anchor or mint `CAP-029`. | architect / PO | None — discharged | DISCHARGED — `CAP-029` minted in `capabilities.md` v1.1; BC-2.06.004 v1.5 re-anchored `CAP-022`→`CAP-029`; `BC-INDEX.md` v3.5 synced (2026-07-13, step-4.5 impl pass 2 remediation burst, per architect recommendation + PO concurrence) |
| (b) | `ARCH-INDEX.md`'s SS-06 (quality-observability) subsystem row lists Implementing Modules as `internal/metrics, internal/paths` — does not yet include `internal/mgmt`, which BC-2.06.004 names as its `architecture_module`. | architect | None — discharged | DISCHARGED — landed in `ARCH-INDEX.md` v1.10 (2026-07-13, step-4.5 impl pass 2 remediation burst, finding F-CS-I2-001) |
| (c) | `error-taxonomy.md`'s E-CFG-004 row currently reads `"config file not found: <path>"` (BC-2.09.003 scope). Ruling 4's reload variant needs a documented second message variant — `"reload not applicable: daemon started without --config"` — mirroring the existing E-NET-001/E-CFG-008 multi-variant catalog pattern. | PO | None — discharged (was non-blocking per Ruling 4 Addendum v1.1) | DISCHARGED — landed in error-taxonomy.md v4.8 (2026-07-12, pass-3 remediation burst) |
| (d) | BC-2.06.004's `VP-TBD-PING-A`/`VP-TBD-PING-B` are placeholder IDs — Ruling 1 did not mint real VP numbers. Architect mints real numbers following the BC-2.06.003 `VP-TBD-A`/`VP-TBD-B` → `VP-061`/`VP-062` precedent (v1.3, "not blocking implementation"). | architect | None — discharged | DISCHARGED — `VP-TBD-PING-A`→`VP-078` (integration), `VP-TBD-PING-B`→`VP-079` (code-audit) minted in BC-2.06.004 v1.4; `VP-INDEX.md` v2.40 synced (2026-07-13, step-4.5 impl pass 2 remediation burst) |

**Downgraded by Ruling 4 Addendum v1.1 (F-CS-SP1-001), then DISCHARGED (pass-3 remediation
burst, 2026-07-12):** Obligation (c) no longer hard-gates TDD implementation. AC-011 PC-3 was
reframed as a defense-in-depth guard — `configPath == ""` is confirmed unreachable via any real
daemon startup path, so the E-CFG-004 message it returns is operator-unreachable, unit-tested
directly against `wireRouterControlHandlers` with `configPath = ""` rather than gated on a
live-daemon integration path. The error-taxonomy.md message-variant documentation has now
**landed** — PO shipped error-taxonomy.md v4.8 with the `router.reload` defense-in-depth message
variant (Variant 3 of the E-CFG-004 row), discharging Obligation (c) in full; nothing remains to
land at or before delivery for this obligation. None of the four Forward Obligations block TDD
implementation of the remaining ACs.

**Non-binding architect recommendation, also from Ruling 4 (not tracked as a Forward Obligation —
informational only):** wherever ADR-004's disambiguation table enumerates per-mode handler sets,
add a row for the new router-mode-exclusion pattern `wireRouterControlHandlers` introduces, so it
doesn't silently drift from the `admin.*` handler exclusion it parallels.

## Non-Goals

- **Literal `--id=<svtn_id>` implementation** for `svtn destroy` or `svtn status` — `SVTNManager`
  is exclusively name-keyed; adding a hex-ID reverse index is a real data-structure change,
  disproportionate to this story (Decisions 2, 3).
- **A synchronous wait-for-reload/drain-completion RPC variant.** `router.reload`/`router.drain`
  are fire-and-forget (`{"accepted": true}`), matching raw-signal UX parity. A response-channel
  variant with real completion confirmation is a future enhancement (Decision 4).
- **A new "router-operator" Tier-2 role/gate.** Router mode has no `SVTNManager`/`RoleControl`
  concept; neither governing BC's Trigger text requests a role qualifier (Decision 4).
- **Duplicating `runDestroyConfirmGate`** in the top-level `svtn destroy` shim (Decision 3).
- **`sbctl svtn list`, `sessions attach/detach/status`, `admin recover`, `version`/`ping`** — each
  covered by a separate backlog story (see Context).

## Architecture Mapping

| Component | Package | New / Modified | Notes |
|-----------|---------|-----------------|-------|
| `runSvtn`, `runSvtnStatus`, `runSvtnDestroyShim` (new) | `cmd/sbctl` (new file, e.g. `svtn.go`) | New | Top-level `svtn` case arm dispatch + status query + destroy redirect shim |
| `runPathsPing` (new) | `cmd/sbctl` (new file, e.g. `paths_ping.go`) | New | Dials `--router=<addr>`, measures client-side RTT, synthesizes CLI output |
| `runRouterReload`, `runRouterDrain` (new) | `cmd/sbctl` (new file(s), e.g. `router_reload.go`/`router_drain.go`) | New | Dispatch `router.reload`/`router.drain` via the existing `connectAndRun` pattern |
| `mgmt.RegisterPingHandler` (new) | `internal/mgmt` (`register_metrics.go` or sibling) | New | `paths.ping` handler — empty request, `{"pong": true}` response |
| `wireMetricsHandlers` | `cmd/switchboard/metrics_wire.go` | Modified | Calls `mgmt.RegisterPingHandler(srv)` alongside `mgmt.RegisterMetricsHandlers` |
| `makeAdminSVTNStatusHandler` (new) | `cmd/switchboard/admin_handlers.go` | New | Uses `resolveCallerAdmissionAnyRole` + `SVTNByName` + role-grouped `ListKeys` counts |
| `BuildAdminHandlers` | `cmd/switchboard/admin_handlers.go` | Modified | Registers `admin.svtn.status` alongside create/destroy |
| `wireRouterControlHandlers` (new) | `cmd/switchboard` (new file, e.g. `router_control_wire.go`) | New | Registers `router.reload`/`router.drain`; router-mode-exclusive; takes `configPath` for AC-011 PC-3's defense-in-depth guard |
| `runRouter` | `cmd/switchboard/mgmt_wire.go` | Modified | Signature widening (Design Constraint above); third select-loop arm |
| `"router"` case body | `cmd/switchboard/main.go` | Modified | Constructs `drainRequestCh`; passes to `runRouter` |
| `svtnmgmt.SVTNManager` (`SVTNByName`, `ListKeys`) | `internal/svtnmgmt` | Read-only consumer | No source changes |
| `mgmt.Server`, `mgmt.Handler` | `internal/mgmt` | Read-only consumer (`Register`) beyond the new ping handler | No structural changes |

## File-Change List

| File | Change |
|------|--------|
| `cmd/sbctl/main.go` | New top-level `case "svtn":` dispatching to `runSvtn`; `ping` sub-verb added to the existing `paths` case arm; `reload`/`drain` sub-verbs added to the existing `router` case arm |
| `cmd/sbctl/svtn.go` (new) | `runSvtn` dispatch (status/destroy/unknown sub-verb); `runSvtnStatus`; `runSvtnDestroyShim` |
| `cmd/sbctl/paths_ping.go` (new) | `runPathsPing` |
| `cmd/sbctl/router_reload.go` / `router_drain.go` (new) | `runRouterReload`, `runRouterDrain` |
| `cmd/sbctl/svtn_test.go` (new) | AC-008, AC-009, AC-010 tests |
| `cmd/sbctl/paths_ping_test.go` (new) | AC-001, AC-002, AC-003 tests |
| `cmd/sbctl/router_control_test.go` (new) | AC-014 (PC-3, re-homed client-side E-NET-001/E-ADM-010 tests, F-CS-SP6-001), AC-015, AC-016 tests |
| `cmd/sbctl/main_test.go` (extended) | Existing-test accommodation (F-CS-I3-001) — `TestSbctl_OrphanSubcommands` asserted `svtn` was an orphan subcommand; AC-010's new `case "svtn":` dispatch re-points those sub-cases to AC-010 PC-3 behavior (exit 2 naming `svtn`, `wantNoPanic` guard) |
| `cmd/sbctl/phase5_pass8_test.go` (extended) | Existing-test accommodation (F-CS-I3-001) — `TestPathsUnknownVerb` used `ping` as its unknown-`paths`-verb example; AC-001 makes `ping` a real verb, so the exemplar swaps to `trace` |
| `internal/mgmt/register_metrics.go` (or new `register_ping.go`) | `RegisterPingHandler` |
| `internal/mgmt/register_metrics_test.go` (extended) or `register_ping_test.go` (new) | AC-004 handler-level tests |
| `cmd/switchboard/metrics_wire.go` | `wireMetricsHandlers` calls `mgmt.RegisterPingHandler` |
| `cmd/switchboard/metrics_wire_test.go` (extended) | AC-004 per-mode registration tests |
| `cmd/switchboard/admin_handlers.go` | New `admin.svtn.status` handler; `BuildAdminHandlers` registration |
| `cmd/switchboard/admin_handlers_test.go` (extended) | AC-005, AC-006, AC-007 tests |
| `cmd/switchboard/router_control_wire.go` (new) | `wireRouterControlHandlers` |
| `cmd/switchboard/router_control_wire_test.go` (new) | AC-011, AC-012, AC-013 (registration half), AC-014 (PC-1, PC-2 only — PC-3 re-homed to `cmd/sbctl/router_control_test.go`, F-CS-SP6-001) tests |
| `cmd/switchboard/mgmt_wire.go` | `runRouter` signature widening; third select-loop arm; `wireRouterControlHandlers` call site |
| `cmd/switchboard/mgmt_wire_test.go` (extended) | Call-site updates for the new `drainRequestCh` parameter — five call sites (mirrors the five-call-site `S-7.04-FU-SIGHUP-RELOAD` update pattern); AC-013 shutdown-parity test |
| `cmd/switchboard/router_drain_test.go` (extended) | Call-site updates for the new `drainRequestCh` parameter — one call site |
| `cmd/switchboard/router_sighup_test.go` (extended) | Call-site update for the new `drainRequestCh` parameter — one call site |
| `cmd/switchboard/router_pe_receive_test.go` (extended) | Call-site update for the new `drainRequestCh` parameter — one call site |
| `cmd/switchboard/router_pe_connector_test.go` (extended) | Call-site updates for the new `drainRequestCh` parameter — four call sites |
| `cmd/switchboard/main.go` | `"router"` case body constructs `drainRequestCh`; passes to `runRouter` — one call site |
| `.factory/specs/prd-supplements/error-taxonomy.md` | **Forward Obligation (c)** — E-CFG-004 message-variant addition — **DISCHARGED**, landed in v4.8 (PO edit, 2026-07-12 pass-3 remediation burst; not a story-writer edit) |
| `.factory/specs/architecture/ARCH-INDEX.md` | **Forward Obligation (b)** — SS-06 Implementing Modules row gains `internal/mgmt` — **DISCHARGED**, landed in v1.10 (architect edit, 2026-07-13 step-4.5 impl pass 2 remediation burst, finding F-CS-I2-001; not a story-writer edit) |

**No ARCH-08 §6.4 registration obligation** — no new `internal/` package is introduced (`internal/mgmt`
already exists at position 20; only its exported surface grows).

## Token Budget Estimate (MANDATORY)

**Methodology:** token counts below are `wc -c` byte counts of the files as they exist at
`develop@4c276d9`, divided by 4 chars/token, for every file that currently exists. Content that
does not yet exist (new-file stub bodies, the 16 ACs' worth of new/extended test code) is a
line-count-based estimate, called out explicitly below. Broken out per dispatch pass per the
per-story-delivery TDD sequence (stub-architect → test-writer → implementer); each pass runs in
its own fresh-context dispatch.

### Pass 1 — Stub pass (stub-architect)

| Context Source | Estimated Tokens |
|---|---|
| This story spec (full — File-Change List, Architecture Mapping, all 16 ACs' signatures) | ~21k |
| Precedent production files (`client.go` [`connectAndRun`], `router_status.go`, `router_metrics.go`, `paths_list.go`, `admin.go` — signature/style conventions for the 6 new production files) | ~14k |
| `interface-definitions.md` (full — CLI verb/flag signatures for the new dispatch surface) | ~15k |
| Tool-call overhead (Read/Glob/Grep envelopes, ~10%) | ~5k |
| **Total** | **~55k** |
| Agent context window | 200K (Sonnet-class) |
| **Budget usage** | **~28%** |

### Pass 2 — Failing-test pass (test-writer)

| Context Source | Estimated Tokens |
|---|---|
| This story spec (full — all 16 ACs' postcondition/test-name blocks, BC anchors) | ~21k |
| BC-2.09.001, BC-2.09.002, BC-2.07.001, BC-2.06.004 (full — exact PC/EC wording to test against) | ~13k |
| `error-taxonomy.md` (full — exact error-string literals for assertions, incl. all 3 E-CFG-004 variants) | ~17k |
| `interface-definitions.md` (full — exact usage/flag strings for CLI-dispatch tests) | ~15k |
| Stub code from Pass 1 (6 new-file skeletons/signatures) | ~3k |
| Tool-call overhead | ~6k |
| **Total** | **~75k** |
| Agent context window | 200K (Sonnet-class) |
| **Budget usage** | **~38%** |

### Pass 3 — TDD implementation pass (implementer)

| Context Source | Estimated Tokens |
|---|---|
| This story spec (full) | ~21k |
| Stub code + failing-test content from Pass 1/2 (6 stub files + ~37 test functions across 5 new + 2 extended test files — estimated from the story's cited test names, not yet written) | ~12k |
| Production files being extended, before-state (`cmd/sbctl/main.go`, `cmd/switchboard/mgmt_wire.go`, `cmd/switchboard/admin_handlers.go`, `cmd/switchboard/metrics_wire.go`, `cmd/switchboard/main.go`, `internal/mgmt/register_metrics.go` — 6 files, full) | ~30k |
| `cmd/switchboard/mgmt_wire_test.go` (full — hosts 5 of the 13 `runRouter` call sites plus the new AC-013 shutdown-parity test) | ~15k |
| Remaining 7 `runRouter` call sites across 4 files (`router_drain_test.go`, `router_sighup_test.go`, `router_pe_receive_test.go`, `router_pe_connector_test.go`) — targeted grep + ~15-line context per site, not full-file reads | ~2k |
| BC/error-taxonomy spot-checks during implementation (exact literals already encoded in the Pass-2 failing tests; lighter touch than Pass 2) | ~8k |
| Tool-call overhead (heaviest pass — most edits + `just test-race` cycles) | ~10k |
| **Total** | **~98k** |
| Agent context window | 200K (Sonnet-class) |
| **Budget usage** | **~49%** |

**Overall:** no pass breaches the 60% split-discussion threshold. Passes 2 (~38%) and 3 (~49%)
exceed the template's nominal 20-30% target band — driven honestly by this story's real scope
(16 ACs, 4 BC anchors, all 3 `error-taxonomy.md` E-CFG-004 variants, and 13 `runRouter` call
sites across 6 files from the signature-widening constraint) rather than padding. The heaviest
pass (implementation, ~49%) stays under half the window with room to spare. No story split
required at 5 points.

## Task Breakdown (Strict TDD — Stubs → Red → Green → Gate)

All tasks execute in a single worktree on a feature branch cut from `develop@HEAD`. Each task gate
is `just test-race` green + `just lint` clean before proceeding to the next.

### Task 1 — `paths.ping`: handler + registration + CLI (AC-001..AC-004)

Red: write the four AC-001..AC-004 tests against stub/no-op implementations. Green: implement
`mgmt.RegisterPingHandler`, wire into `wireMetricsHandlers`, implement `runPathsPing`, wire into
`cmd/sbctl/main.go`'s `paths` case arm. Gate: `just test-race`, `just lint`.

### Task 2 — `admin.svtn.status` + `svtn` top-level dispatch (AC-005..AC-010)

Red: write AC-005..AC-010 tests. Green: implement the `admin.svtn.status` handler (reusing
`resolveCallerAdmissionAnyRole`/`SVTNByName`/`ListKeys`), register in `BuildAdminHandlers`,
implement `runSvtn`/`runSvtnStatus`/`runSvtnDestroyShim`, wire the new `svtn` case arm into
`cmd/sbctl/main.go`. Gate: `just test-race`, `just lint`.

### Task 3 — `runRouter` signature widening + `wireRouterControlHandlers` scaffolding (AC-013 registration half)

Stub-first (mirrors `S-7.04-FU-SIGHUP-RELOAD` Task 1 pattern): widen `runRouter`'s `sighupCh`
parameter, add `drainRequestCh` parameter, add the third select-loop arm as a no-op stub, update
every `runRouter` call site in `cmd/switchboard` (enumerated as of develop @ `4c276d9`:
`main.go`, `mgmt_wire_test.go`, `router_drain_test.go`, `router_sighup_test.go`,
`router_pe_receive_test.go`, `router_pe_connector_test.go` — implementer MUST re-grep
`runRouter(` under `cmd/switchboard` at implementation time, since new call sites may land
before delivery) to pass the new argument. Gate: all **existing** tests remain green (no new
test files yet); `just lint` clean.

### Task 4 — `router.reload`/`router.drain` handlers, server + client (AC-011, AC-012, AC-013 remainder, AC-014, AC-015, AC-016)

**Note (DISCHARGED, pass-3 remediation burst):** Forward Obligation (c) — error-taxonomy.md's
E-CFG-004 "reload not applicable" variant has landed (v4.8, 2026-07-12); nothing remains to gate
this task. AC-011 PC-3 is a unit-tested defense-in-depth guard
(`TestRouterReload_NoConfigLoaded_ECFG004` calls `wireRouterControlHandlers` directly with
`configPath = ""`).

**Client-coverage note (pass-6 remediation, F-CS-SP6-001):** AC-015/AC-016 and AC-014 PC-3 close a
gap the pass-6 adversary found by auditing ACs against the File-Change List — `runRouterReload`/
`runRouterDrain` and the `router` case arm's new `reload`/`drain` sub-verb arms were already
scoped as this task's Green-step deliverables, but had no Red-step test coverage in `cmd/sbctl`
before this fix. That gap is closed below; no task boundary moved, since the client CLI
implementation was already correctly assigned to this task's Green step.

Red: write AC-011, AC-012, AC-013, AC-014, AC-015, AC-016 tests against the Task 3 stub. The
server-side tests (AC-011, AC-012, AC-013, AC-014 PC-1/PC-2) fail because the select-arm is a
no-op. The client-side tests (AC-014 PC-3, AC-015, AC-016, all in the new
`cmd/sbctl/router_control_test.go`) fail because `runRouterReload`/`runRouterDrain` don't exist
yet and the `router` case arm's `reload`/`drain` sub-verbs aren't wired — today that arm dispatches
only `metrics`/`status`, so `sbctl router reload`/`sbctl router drain` currently fall through to
the default arm's usage error, exit 2 (the sub-verb transition AC-015/AC-016 pin). Green:
implement `wireRouterControlHandlers`, replace the select-loop stub arm with real `goto shutdown`
dispatch, implement `runRouterReload`/`runRouterDrain` CLI (via the existing `connectAndRun`
pattern), wire the `router` case arm's `reload`/`drain` sub-verbs. Gate: `just test-race`,
`just lint`.

### Task 5 — Quality gate

```sh
just fmt
just lint
just test-race
```

All packages pass. Zero lint warnings. Then open PR targeting `develop`.

## Delivery Plan Note — POL-005

Any adversarial or evaluation dispatch for this story (per-story pass, wave-gate Perimeter-2, or
any other evaluation dispatch) **MUST embed the POL-005 (`adversary-dispatch-integrity`, HIGH)
verification tuple** in the dispatch prompt — `{repo path, branch, expected HEAD SHA at dispatch
time, artifact IDs + versions under review}` — per `.factory/policies.yaml` POL-005. The dispatched
agent's first action must verify its observed `git rev-parse HEAD` and artifact versions against
the tuple before proceeding; on mismatch, it must ABORT the pass and report the divergence as the
pass result rather than reviewing stale state.

## Anchors Consumed

| Anchor | Verbatim ID | Source | Disposition |
|--------|-------------|--------|-------------|
| One-shot reachability + RTT probe | BC-2.06.004 PC-1..PC-4, Invariant 1, Invariant 2 | Ruling 1 | TO DISCHARGE — AC-001..AC-004 |
| SVTN status query with role-grouped key counts | BC-2.07.001 v1.14 PC-4 | Ruling 2 | TO DISCHARGE — AC-005..AC-008 |
| SVTN destroy top-level migration shim | (no BC — CLI-surface documentation) | Ruling 3 | TO DISCHARGE — AC-009, AC-010 |
| RPC-triggered reload, code-path-identical to SIGHUP | BC-2.09.001 v1.2 PC-1 | Ruling 4 | TO DISCHARGE — AC-011, AC-013, AC-014, AC-015 |
| RPC-triggered drain, same shutdown sequence as SIGTERM | BC-2.09.002 v1.3 Trigger/PC-1 | Ruling 4 | TO DISCHARGE — AC-012, AC-013, AC-014, AC-016 |
| `DRIFT-HS006-DRAIN-CLI-MISSING` | drift item | `S-7.04-FU-DRAIN-WIRE-placement-note.md`, `S-7.04-FU-SIGHUP-RELOAD-placement-note.md` | RESOLVED by AC-011/AC-012 — tag PR with `Resolves: DRIFT-HS006-DRAIN-CLI-MISSING` per this repo's non-`closes`/`fixes` convention for prior-architect-note-reported items |

## Provenance

- **Finding:** F-P5P6-A-005 (Phase 5 Pass 6 Adv-A, 2026-07-03) — seven `sbctl` verbs specified
  without PENDING annotations; five collective-annotated here.
- **Spec annotation:** `interface-definitions.md` v1.31 — CLI listing and Registered Verbs rows
  already adjudicated and updated by PO/architect per the rulings doc (this story does not edit
  that file).
- **Adjudication:** `.factory/decisions/S-BL.CLI-SURFACE-COMPLETION-rulings.md` v1.2 (2026-07-12)
  — all four Open Design Obligations resolved, plus the Ruling 4 Addendum (F-CS-SP1-001,
  spec-adversarial pass 1) reframing AC-011 PC-3 as a defense-in-depth guard, plus the Ruling 2
  Addendum (F-CS-SP3-003, spec-adversarial pass 3) confirming AC-008 PC-3 stands unchanged. This
  elaboration (v2.3) is the story-writer transcription of that ruling into sprint-ready ACs.

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 2.8 | 2026-07-13 | Remediated step-4.5 impl pass 3 (fresh adversary) finding F-CS-I3-001 (LOW, File-Change-List completeness gap, orchestrator-verified). The File-Change List omitted two files present in the feature diff against `develop` (`git diff --stat`: `cmd/sbctl/main_test.go` +42/-10, `cmd/sbctl/phase5_pass8_test.go` +9/-3) — both correct, necessary existing-test accommodations forced by this story's own scope, not scope creep: `main_test.go`'s `TestSbctl_OrphanSubcommands` asserted `svtn` was an orphan subcommand — AC-010's new `case "svtn":` dispatch re-points those sub-cases to AC-010 PC-3 behavior (exit 2 naming `svtn`, `wantNoPanic` guard); `phase5_pass8_test.go`'s `TestPathsUnknownVerb` used `ping` as its unknown-`paths`-verb example — AC-001 makes `ping` a real verb, so the exemplar swaps to `trace`. Fixed: two new rows added to the File-Change List in the existing table's style, grouped with the other `cmd/sbctl` rows. No AC/PC/Decision/FO/point content touched — a documentation-completeness fix for accommodations the implementation already had to make; `acceptance_criteria_count` (16) and `estimated_points` (5) unchanged. `input-hash` unchanged — no declared `inputs:` file changed, story-body-only edit; `--check` confirms no drift. Frontmatter `version` 2.7 → 2.8; new `modified:` entry appended (newest-first). |
| 2.7 | 2026-07-13 | Propagated the step-4.5 impl pass 2 remediation burst (finding F-CS-I2-001, nitpick N-CS-I2-01) plus a version-pin refresh. **VP propagation (F-CS-I2-001, FO(d)):** every live `VP-TBD-PING-A`/`VP-TBD-PING-B` reference replaced with `VP-078`/`VP-079` — frontmatter `verification_properties:`/`vp_traces:` lists, AC-004 PC-3's parenthetical; the two v2.0-era historical references and the unrelated BC-2.06.003 `VP-TBD-A`/`VP-TBD-B` precedent citation left untouched per the layered-decision-record convention. **Forward Obligations table:** row (b) → DISCHARGED (`ARCH-INDEX.md` v1.10, F-CS-I2-001 — SS-06 gains `internal/mgmt`); row (d) → DISCHARGED (`VP-TBD-PING-A`→`VP-078`, `VP-TBD-PING-B`→`VP-079` minted BC-2.06.004 v1.4, `VP-INDEX.md` v2.40 synced); row (a) → DISCHARGED (`CAP-029` minted `capabilities.md` v1.1, BC-2.06.004 v1.5 re-anchored `CAP-022`→`CAP-029`, `BC-INDEX.md` v3.5 synced, per architect recommendation + PO concurrence) — all four FOs now DISCHARGED, rows (a)/(b)/(d) rewritten to match row (c)'s existing wording style; Obligation-column description text left unchanged. File-Change List's `ARCH-INDEX.md` row updated to the same discharged form already used on the `error-taxonomy.md` row. **N-CS-I2-01** (adversary nitpick, pass 2, taken): AC-015 PC-1 and AC-016 PC-1 wrongly showed `sbctl router reload --router=<addr>` / `sbctl router drain --router=<addr>` — neither verb takes a `--router` sub-flag; the daemon address comes from the global `--target` flag (`interface-definitions.md` v1.31 §82/§83), matching the frozen implementation + tests (feature/S-BL.CLI-SURFACE-COMPLETION @ 1b0e010). Fixed both PC-1 exemplars to the bare form and both ACs' Precondition lines (`--router=<addr>` → `--target=<addr>`); swept both AC blocks, the Task Breakdown, and the File-Change List for other `--router=` occurrences attached to reload/drain — none found (the four remaining `--router=` hits are all `paths ping`'s own correct, distinct flag — untouched). **Version-pin refresh:** BC-2.06.004's `inputDocuments` comment v1.1 → v1.5; `ARCH-INDEX.md` v1.10, `capabilities.md` v1.1, `BC-INDEX.md` v3.5 newly cited at the FO table's row (a)/(b) Status cells. `acceptance_criteria_count` (16), `estimated_points` (5), and all AC/PC semantics unchanged — N-CS-I2-01 corrects exemplars to match already-authoritative spec/implementation, not a behavior change. `input-hash` recomputed via `compute-input-hash --update` (BC-2.06.004 input content changed, v1.1 → v1.5). Frontmatter `version` 2.6 → 2.7; new `modified:` entry appended (newest-first). |
| 2.6 | 2026-07-13 | Delivery-phase governance addition (NON-BEHAVIORAL): added the missing "Token Budget Estimate (MANDATORY)" section, one of the template sections `validate-template-compliance` has flagged since Round 1 — the per-story-delivery playbook's Token Budget Check reads this section before every test-writer/implementer spawn and mandates story-writer add it if absent. No AC, PC, Decision, or Forward Obligation content touched; the spec-adversarial convergence (3/3 clean passes as of pass 9, achieved on v2.5) covers behavioral content and STANDS unaffected. Section broken into the three per-story-delivery dispatch passes (stub-architect, test-writer, implementer), each a fresh-context dispatch: Pass 1 (stub) ~55k tokens (~28% of a 200K window), Pass 2 (failing-test) ~75k (~38%), Pass 3 (TDD implementation, the heaviest) ~98k (~49%) — none breaches the 60% split-discussion threshold. Methodology: `wc -c`/4 chars-per-token on files as they exist at develop@4c276d9 for everything already on disk (story spec, precedent production files, `interface-definitions.md`, `error-taxonomy.md`, the four BC anchor files, the six production files being extended, `mgmt_wire_test.go`); line-count-based estimates, called out explicitly, for not-yet-written content (6 new-file stub bodies, ~37 test functions across 5 new + 2 extended test files implied by the story's cited test names). Noted honestly that Passes 2 and 3 exceed the template's nominal 20-30% target band — driven by this story's real scope (16 ACs, 4 BC anchors, all 3 `error-taxonomy.md` E-CFG-004 variants, 13 `runRouter` call sites across 6 files) rather than padding — but the heaviest pass stays under half the window; no story split warranted at 5 points. Section inserted between File-Change List and Task Breakdown, matching the template's ordering. `input-hash` unchanged — story-body-only edit; `--check` confirms no drift. Frontmatter `version` 2.5 → 2.6; new `modified:` entry appended (newest-first). |
| 2.5 | 2026-07-13 | Remediated pass-8 spec-adversarial nitpick N-CS-SP8-01 (orchestrator-adopted as fix; pass 8 verdict was NITPICK_ONLY, zero findings — 3-clean-pass streak sits at 2/3, unaffected by this fix). AC-015 PC-1 and AC-016 PC-1 both cited "`router status`/`router metrics`" as the `connectAndRun` exemplars for the reload/drain dispatch shape. Verified against develop@4c276d9 (`grep connectAndRun cmd/sbctl/*.go`): `runRouterMetrics` (router_metrics.go) and `runPathsList` (paths_list.go) both call `connectAndRun` directly; `runRouterStatus` (router_status.go) does **not** — it hand-rolls its own `net.Dialer` + `Authenticate` + dispatch of `paths.list`, and its own source comment says it "mirrors connectAndRun's discrimination" rather than using it. Citing `router status` as a `connectAndRun` user was factually wrong. Fixed both parentheticals to "(same dial+auth+dispatch shape `router metrics` and `paths list` already use)". Swept the rest of the story (Decisions, Task 4, File-Change List, all `dial+auth+dispatch`/`net.Dialer`/`hand-roll` hits) for the same mispairing — none found; the other `paths list`/`router status` co-mentions (Decision 1's RPC-semantics contrast, the drain UX-parity note, Decision 3's bare-top-level-read CLI-shape note) are unrelated to `connectAndRun` and needed no change. Also applied N-CS-SP8-02 (STORY-INDEX.md frontmatter `modified` scalar trailed at 2026-07-12; refreshed to 2026-07-13). `input-hash` unchanged — story-body-only edit; `--check` confirms no drift. Frontmatter `version` 2.4 → 2.5; new `modified:` entry appended (newest-first). |
| 2.4 | 2026-07-13 | Remediated pass-6 spec-adversarial finding F-CS-SP6-001 (MED, AC-coverage/test-file-assignment gap, orchestrator-verified) plus nitpick N-CS-SP6-01. `router reload`/`router drain` were the only verbs with no client-side CLI dispatch acceptance criterion and no `cmd/sbctl` test file, despite the File-Change List already creating `cmd/sbctl/router_reload.go`/`router_drain.go` and adding `reload`/`drain` sub-verb arms to `main.go`'s `router` case — a real behavior change (that arm dispatches only `metrics`/`status` today; other sub-verbs exit 2 via `usageErrf`). AC-014 PC-3's client-observed E-NET-001/E-ADM-010 codes were also mis-assigned to a server-side test file that structurally cannot exercise `cmd/sbctl`. **Fixed:** two new client happy-path ACs added — **AC-015** (`sbctl router reload`, BC-2.09.001 v1.2 PC-1, same anchor as AC-011) and **AC-016** (`sbctl router drain`, BC-2.09.002 v1.3 Trigger/PC-1, same anchor as AC-012) — each with a sub-verb-transition-pin postcondition proving the known sub-verb now dispatches while an adjacent still-unknown sub-verb (`router bogus`) continues to exit 2; AC-014 PC-3 re-homed to new `cmd/sbctl/router_control_test.go`, its Test names/level/file block split per-postcondition (PC-1/PC-2 stay server-side, PC-3 moves); `acceptance_criteria_count` 14 → 16; File-Change List gained the new test-file row plus a narrowed AC-014 scope note on the `router_control_wire_test.go` row; Task 4 retitled and its Red step rewritten to cover all six ACs and both failure modes; Anchors Consumed's reload/drain rows gained AC-015/AC-016; Architecture Mapping needed no change (never named test files); points kept at 5 (the gap was AC/test documentation of already-scoped work, not new implementation scope). **N-CS-SP6-01** (nitpick, taken): AC-011 PC-3's abbreviated quote of the `runRouter` entry-guard message marked as abbreviated with the full wrapped literal inlined, per error-taxonomy.md v4.9's own changelog (F-CS-SP4-001) confirming the story's Variant 3 literal needed no change. `input-hash` unchanged — story-body-only edit, `--check` confirms no drift. Frontmatter `version` 2.3 → 2.4; new `modified:` entry appended (newest-first). |
| 2.3 | 2026-07-12 | Remediated pass-3 spec-adversarial findings (F-CS-SP3-001, F-CS-SP3-002, F-CS-SP3-003) and discharged Forward Obligation (c). Architect filed the Ruling 2 Addendum (rulings doc v1.1 → v1.2, F-CS-SP3-003 — AC-008 PC-3 VINDICATED, stands unchanged); PO landed `error-taxonomy.md` v4.8 with both the E-CFG-001 client-side variant note and the E-CFG-004 `router.reload` defense-in-depth variant in the same burst, which discharges FO(c). **F-CS-SP3-001** (FO table row (c)): Gate cell `Before implementation of AC-011's E-CFG-004 postcondition` → `None — discharged (was non-blocking per Ruling 4 Addendum v1.1)`; Status cell → `DISCHARGED — landed in error-taxonomy.md v4.8 (2026-07-12, pass-3 remediation burst)`; the "Downgraded by Ruling 4 Addendum v1.1" paragraph rewritten to record the discharge. **F-CS-SP3-002** (Decision 4 reload-bridging bullet): "must land before this AC ships" → discharged form citing the v4.8 landing. **F-CS-SP3-003** (AC-008 PC-3): text stands unchanged per the addendum; appended the architect's optional traceability parenthetical (`usageErrf`, §110/§111 siblings, error-taxonomy.md E-CFG-001 client-side variant note). **Semantic reference-site sweep** (searched "Obligation (c)"/"FO(c)"/"E-CFG-004"/"error-taxonomy"/"taxonomy" and read every hit's sentence — the token-grep approach missed this class twice already, F-CS-SP2-002 then this pass): AC-011 PC-3's trailing sentence, the File-Change List's error-taxonomy.md row, and Task 4's note all updated from non-blocking/pending phrasing to discharged/landed phrasing; the v2.0/v2.2-era historical `modified:`/Changelog rows describing the old language left untouched as accurate period records; the Delivery Plan Note (POL-005) doesn't mention FO(c), no change needed; the FO table's general intro sentence remains accurate (describes all four FOs, not (c) specifically). Live rulings-doc pin refreshed v1.1 → v1.2 at all three live binding-source citations (frontmatter `inputDocuments` comment, Adjudicated Design Decisions intro, Provenance section). `error-taxonomy.md` gained its first version-pinned citations in this story (v4.8), at the discharge sites — it was previously cited by filename only. `input-hash` recomputed via `compute-input-hash --update` (the rulings doc input changed content, v1.1 → v1.2). Frontmatter `version` 2.2 → 2.3; new `modified:` entry appended (newest-first). |
| 2.2 | 2026-07-12 | Remediated two MED spec-adversarial pass 2 findings, both story-side. **F-CS-SP2-001** (premise/doc-drift): the `runRouter` call-site enumeration was incomplete — Design Constraint parenthetical, File-Change List, and Task 3 named only `main.go`, `mgmt_wire_test.go`, `router_drain_test.go`, but a `runRouter(` grep against `cmd/switchboard` at develop @ `4c276d9` found thirteen call sites across six files (`router_sighup_test.go` one call, `router_pe_receive_test.go` one call, `router_pe_connector_test.go` four calls, all previously omitted; all six files are package `main` and would fail to compile once the `drainRequestCh` trailing parameter lands, making Task 3's "all existing tests remain green" gate unmeetable against the old closed list). Fixed at all three loci: Design Constraint parenthetical now enumerates all six files with per-file call counts; File-Change List gained three new rows for the omitted files (each with its call count) and the two pre-existing test-file rows gained counts too; Task 3's call-site sentence rewritten to the open, drift-durable form — enumerates today's six files but requires the implementer to re-grep `runRouter(` under `cmd/switchboard` at implementation time, since new call sites may land before delivery. **F-CS-SP2-002** (contradiction): the File-Change List's `error-taxonomy.md` row still read `(PO edit, gates AC-011; not a story-writer edit)` — the one locus the v2.1 FO(c) downgrade missed, letting an implementer re-derive the blocking dependency v2.1 removed. Fixed to `(PO edit, non-blocking per Ruling 4 Addendum v1.1; not a story-writer edit)`. Grepped the whole story for residual "gate"/"gates"/"hard gate" phrasing tied to FO(c); found no other contradictions (the Forward Obligations table's intro sentence and Task 4's existing non-blocking note both already read correctly; changelog rows describing history are exempt as accurate records). `input-hash` unchanged — this was a story-body-only fix; no input file (rulings doc, BC files, `interface-definitions.md`) was touched; `--check` confirms no drift. Frontmatter `version` 2.1 → 2.2; new `modified:` entry appended (newest-first). |
| 2.1 | 2026-07-12 | Propagated architect Ruling 4 Addendum (`S-BL.CLI-SURFACE-COMPLETION-rulings.md` v1.1, F-CS-SP1-001, spec-adversarial pass 1) into AC-011 and its dependents. **AC-011 PC-3 reframed** from an operator-reachable-but-untested guard to an explicit **defense-in-depth guard** (unreachable via any real daemon startup path — `runRouter`'s entry guard in `cmd/switchboard/mgmt_wire.go` plus the `"router"` case in `cmd/switchboard/main.go` together guarantee `configPath != ""` for every router instance reaching `wireRouterControlHandlers` registration; presence at runtime would indicate a code defect, mirrors the `E-CFG-011` defensive-annotation shape). PC-3's test level downgraded `integration` → `unit` (test name unchanged: `TestRouterReload_BridgesToSighupCh_CodePathIdentical` stays integration for PC-1/PC-2; `TestRouterReload_NoConfigLoaded_ECFG004` for PC-3 is now unit); invocation-pattern note added — calls `wireRouterControlHandlers`/its registered handler directly with `configPath = ""`, no live daemon. **Mechanism correction:** `wireRouterControlHandlers` gains a `configPath string` second parameter (was missing entirely in the original signature — PC-3 as drafted had no way to observe `configPath`); updated at both literal-signature occurrences (Decision 4 registration-point bullet, AC-013 postcondition 1) plus the Architecture Mapping table row's Notes cell, each with a one-line rationale pointer back to AC-011 PC-3. **Forward Obligation (c) downgraded** from `OPEN — hard gate on AC-011` to `OPEN — non-blocking (does not gate Task 4 implementation)`; the "Obligation (c) is the only hard implementation gate" paragraph and Task 4's "Gate check before this task" note both rewritten to match — none of the four Forward Obligations now hard-gate TDD implementation. Rulings-doc citation pinned to v1.1 at the two locations asserting it as binding source (Adjudicated Design Decisions section intro, Provenance section) — previously cited by filename+date only. `interface-definitions.md` pin bumped v1.30 → v1.31 (PO fixed §60's `usage:` prefix under F-CS-SP1-002; AC-009's own text was already correct, no AC change) at all live-reference citations (frontmatter `inputDocuments` comment, Context section prose, Provenance section) — the v2.0 historical `modified:` narrative entry left untouched as an accurate record of what was true at that time. BC-2.09.001 (v1.2) / BC-2.09.002 (v1.3) pins reviewed and **retained** per the governance-leaf convention (N-CS-SP1-01) — both files' subsequent bumps (v1.2→v1.3, v1.3→v1.4) are traceability-only Stories-cell fills, `governance_leaf: true`, no PC/AC behavior change, so the story's existing pins are not factually wrong. `input-hash` recomputed via `compute-input-hash --update` (`88c13c8`, was `2af06c0` — the rulings doc input changed). Frontmatter `version` 2.0 → 2.1; new `modified:` entry appended (newest-first). |
| 2.0 | 2026-07-12 | Elaborated from backlog stub (v1.0, draft, 0 ACs) to sprint-ready (`ready`, 14 ACs, 5 points) per architect ruling `S-BL.CLI-SURFACE-COMPLETION-rulings.md`. Replaced "Open Design Obligations" with "Adjudicated Design Decisions" (four decisions, one per ruling, load-bearing constraints transcribed inline). Added Design Constraint section for the `runRouter` signature widening. 14 ACs traced to BC-2.06.004 PC-1..4, BC-2.07.001 PC-4, BC-2.09.001 v1.2 PC-1 RPC-trigger note, BC-2.09.002 v1.3 Trigger/PC-1 RPC-trigger note, plus CLI dispatch/flag-parse ACs per `interface-definitions.md` §§60/62/77/82-83. Four Forward Obligations encoded as explicit story-tracked tasks (CAP-022/CAP-029 confirmation, ARCH-INDEX SS-06 `internal/mgmt` row, error-taxonomy.md E-CFG-004 variant [hard gate on AC-011], VP-TBD-PING-A/B real VP-number minting). `bc_traces` gained BC-2.06.004. `estimated_points` TBD → 5 (Ruling 4 is the largest plumbing — signature widening + new channel + registration function + router-mode-exclusive wiring, comparable alone to `S-7.04-FU-SIGHUP-RELOAD`'s full 3-point scope; Rulings 1-2 each add a full handler+CLI wire pair; Ruling 3 is a near-zero usage-error shim). Frontmatter conformed to `S-BL.LOOPBACK-FULLSTACK` template-mandated superset keys. Full File-Change List, Architecture Mapping, Task Breakdown, and POL-005 Delivery Plan Note added. `input-hash` to be computed via `compute-input-hash --update` in the same burst as commit. |
| 1.0 | 2026-07-03 | Draft backlog stub created per F-P5P6-A-005 adjudication (annotate-and-defer). `interface-definitions.md` v1.19 PENDING-S-BL.CLI-SURFACE-COMPLETION annotation is the spec-side closure; this stub is the backlog-side closure. BC anchors: BC-2.09.001 (router reload), BC-2.09.002 (router drain), BC-2.07.001 (svtn destroy). Two verbs (paths ping, svtn status) had no governing BC — open design obligations noted. Four open design obligations logged. |
