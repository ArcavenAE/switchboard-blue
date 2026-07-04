---
pass_id: P5P18-Adv-A
adversary_lens: public-surface + operator-UX drift
prior_passes_read: false
worktree_preflight:
  target_sha: 6deda15def9326f28e96f133e237aff5ecb74d7b
  branch: develop
  HEAD_sha: <not-verified-read-only-no-bash>
  refs/heads/develop_sha: <not-verified-read-only-no-bash>
  origin/develop_sha: <not-verified-read-only-no-bash>
  worktree_root: /Users/skippy/work/aae-orc/run/switchboard-blue
  preflight_result: PASS (orchestrator-verified out-of-band before dispatch)
budget:
  wall_clock_target: <=6 min
  wall_clock_used: ~5 min
  file_reads_target: <=6
  file_reads_used: 5
verdict: HAS_FINDINGS
findings_count: 1
anti_findings_count: 6
policies_applied:
  - POL-001
  - POL-002
streak_state:
  adjudicated_deferrals_respected: true
  reopened_deferrals: none
  respected_list:
    - DRIFT-P5P7-O1
    - DRIFT-P5P7-O4
    - DRIFT-P5P2-B-O003
    - DRIFT-HS006-ROUTER-DAEMON-STUB
    - DRIFT-P5P4-PROMPT-SHORTID
    - DRIFT-P5P9-STALE-RECONCILIATION-COMMENT
    - DRIFT-P5P14-B-001-VP-SOURCE-BC-VERSION-PIN
    - F-P5P13-A-001, F-P5P13-A-002, F-P5P13-B-001 (SHIPPED PR #69)
    - F-P5P14-B-002, F-P5P14-B-003 (SHIPPED .factory)
    - F-P5P14-B-004, F-P5P14-B-005 (SHIPPED PR #70)
    - F-P5P15-A-001 (SHIPPED .factory 5e42768)
    - F-P5P15-B-001 (SHIPPED .factory 5120c9e)
    - F-P5P16-A-001 (SHIPPED .factory 041ea2f)
    - F-P5P17-A-001, F-P5P17-A-002 (SHIPPED .factory 2be16e5)
    - VP-077 Coverage-cell tidy nit (in-flight)
delivered_by: p5-pass18-adv-a (2026-07-03)
adjudication:
  F-P5P18-A-001: SHIPPED at .factory f8b2d7e — 8 story-frontmatter status fields swept to canonical `status: merged` (S-6.05, S-6.07, S-7.03, S-1.02, S-1.03, S-2.02, S-W3.04, S-W3.05); STORY-INDEX v3.74 changelog row (commit bc79621) documents the sweep and preserves mixed master-table cell vocabulary intentionally.
---

## Critical Findings

*(none)*

## Important Findings

### F-P5P18-A-001 — STORY-INDEX rows systematically out-of-sync with story frontmatter status (POL-002)

- **Class:** POL-002 story-index-row-sync (systematic pattern)
- **Confidence:** HIGH
- **Severity:** MED (pattern flag: 8+ stories affected → policy elevation criterion met)
- **Anchor lines (primary drift trilogy — completed-work stories whose spec still reads draft):**
  - `.factory/stories/STORY-INDEX.md:73` marks S-6.05 as `merged (PR #61, 7fe3e29)` — but `.factory/stories/S-6.05-svtn-destroy.md:7` shows `status: draft`.
  - `.factory/stories/STORY-INDEX.md:74` marks S-6.07 as `merged (PR #42, 446efce)` — but `.factory/stories/S-6.07-svtn-admin-create.md:6` shows `status: draft`.
  - `.factory/stories/STORY-INDEX.md:77` marks S-7.03 as `merged (PR #60, 7142146)` — but `.factory/stories/S-7.03-console-remote-control.md:7` shows `status: draft-with-po-refit`.
- **Anchor lines (secondary drift — "completed" in STORY-INDEX but story frontmatter still `ready`/`pending`):**
  - S-1.02 (STORY-INDEX line 47 "completed (PR #2, merge 9e9a98a)") vs `.factory/stories/S-1.02-halfchannel-clock.md:7` `status: ready`.
  - S-1.03 (STORY-INDEX line 48 "completed (PR #7, merge f35e836)") vs `.factory/stories/S-1.03-node-identity-session-continuity.md:7` `status: pending`.
  - S-2.02 (STORY-INDEX line 50 "completed (PR #6, merge a06b306)") vs `.factory/stories/S-2.02-admission-svtn-isolation.md:7` `status: pending`.
  - S-W3.04 (STORY-INDEX line 56 "completed (PR #17)") vs `.factory/stories/S-W3.04-daemon-assembly.md:7` `status: ready`.
  - S-W3.05 (STORY-INDEX line 57 "completed (PR #16, merge fa6345e)") vs `.factory/stories/S-W3.05-hmac-failure-counter.md:7` `status: ready`.
- **Symptom (operator/reader impact):** An operator/agent reading a story spec to determine whether the work is done — the canonical VSDD workflow — is misled by a `draft` / `ready` / `pending` / `draft-with-po-refit` status on the story file. The STORY-INDEX is authoritative for macro sprint tracking, but the story frontmatter is authoritative for per-story lifecycle and is what downstream tooling (spec-steward, adversary preflight, wave-gate agents) reads first. The mismatch means the two authorities disagree on the same fact.
- **Corroboration:** STORY-INDEX v3.73 (line 5) reports `Complete: 34` with an explicit list including all three primary-drift stories at line 24; frontmatter for those three stories still carries pre-merge lifecycle labels. Some peer stories DO carry post-merge status labels (S-BL.ROUTER-ADDR `status: merged`, S-7.01 `status: merged`, S-5.02 `status: merged`, S-6.06 `status: merged`), which confirms a `merged` label is available and used — the affected stories were simply not updated. The drift is systematic (≥8 stories), not an isolated slip.
- **Remediation shape:** For each drifted story, update the story-file frontmatter `status:` field to reflect the merged/completed state that STORY-INDEX already records. Post-merge lifecycle vocabulary is currently mixed (`completed` vs `merged`); recommend a single canonical value (either `completed` or `merged`) and one-shot pass across all 34 currently-listed complete rows. If the mixed vocabulary is intentional (e.g., `completed` = pre-`merged`-vocabulary Wave-0..4 stories, `merged` = later waves), document that convention in STORY-INDEX or the story template — otherwise every future pass will re-raise this drift.

## Observations

*(none)*

## Anti-findings (things checked that passed)

- **F-P5P16-A-001 stayed shipped** — envelope success example at `.factory/specs/prd-supplements/interface-definitions.md:210-216` shows the three-field envelope `{ok, error, data}` with no `$schema` field; `cmd/sbctl/client.go:97` `jsonEnvelope` struct definition (verified via grep) has three fields matching. Prose at line 208 correctly reads "no top-level schema field is emitted." No regression.
- **F-P5P17-A-001 stayed shipped** — router.metrics response example at `.factory/specs/prd-supplements/interface-definitions.md:297-310` shows `{frame_count, hmac_fail_count, drop_cache_hits, path_distribution}` with NO `svtn_id` field. Registered Verbs Response Data column at line 404 also omits `svtn_id`. Matches `internal/metrics/types.go:101-112` `RouterMetricsResponse` wire type and `cmd/sbctl/router_metrics.go:25-30` decode struct.
- **F-P5P17-A-002 stayed shipped** — `path_distribution` example values at `.factory/specs/prd-supplements/interface-definitions.md:305-306` are integer frame counts (`900000, 334567`), not fractional ratios. Matches wire type `map[string]uint64` at `internal/metrics/types.go:110-111` and `cmd/sbctl/router_metrics.go:29`. Corroborated by BC-2.06.003 test vector at demo-evidence `stub_daemon.go:125` emitting integer counts.
- **POL-001 changelog completeness for v1.28** — spec version bump v1.27 → v1.28 at line 5 is accompanied by an explicit v1.28 changelog entry at line 143 naming WHAT (svtn_id phantom field removal + path_distribution integer-count correction), WHY (Phase 5 Pass 17 adversarial remediation, deserialization safety under typed consumers), and TRACEABILITY (F-P5P17-A-001, F-P5P17-A-002; wire types at `internal/metrics/types.go:101-112` and `cmd/sbctl/router_metrics.go:25-30`; BC-2.06.003 test vector reference).
- **VP-077 v1.1 traceability chain intact** — `.factory/specs/verification-properties/VP-077.md:14` `source_bc: BC-2.05.004@v14` uses version-pin form (POL-003 candidate satisfied); `.factory/specs/verification-properties/VP-077.md:21` `implementing_story: S-6.06` matches Story Trace table at line 277 (S-6.06); STORY-INDEX line 70 confirms S-6.06 is complete + owns BC-2.05.004; VP-077 v1.1 lifecycle event at line 284 anchors to F-P5P15-B-001 and cites develop tip `6deda15` — matches target SHA.
- **Console CLI wire-format matches spec (F-P5P6-A-004 amendment stayed correct)** — spec at `.factory/specs/prd-supplements/interface-definitions.md:89-91` documents `sbctl console attach --session=<name>`, `sbctl console detach` (no flags), `sbctl console switch --session=<name>`. `cmd/switchboard/console_handlers.go:49-51` registers `console.attach`, `console.detach`, `console.switch` verbs; e2e test at `cmd/switchboard/console_handlers_e2e_test.go:96` sends `console.attach` with `{"session_name": "agent-01"}` payload. Verb names, flag names, and payload key (`session_name` wire field maps to `--session=<name>` CLI flag) are all consistent.
- **Bare `sbctl sessions` default-to-list behavior (F-P5P6-A-003) matches spec** — `.factory/specs/prd-supplements/interface-definitions.md:70` documents "bare `sbctl sessions` also dispatches sessions.list RPC"; `cmd/sbctl/main.go:129` confirms `runSessions` dispatches `sessions.list` when no sub-verb given; production_exit_code_test.go:552-567 guards this contract with an explicit test.

VERDICT: HAS_FINDINGS
