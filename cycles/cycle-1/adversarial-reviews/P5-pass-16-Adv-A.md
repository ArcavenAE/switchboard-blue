---
pass_id: P5P16-Adv-A
adversary_lens: public-surface + operator-UX
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
  file_reads_used: 6 (interface-definitions.md, admin_handlers.go, mgmt_wire.go, policies.yaml, mgmt.go partial, client.go partial + main.go partial via same slot — 4 full reads + 2 partial reads)
  overage_disclosure: none
verdict: HAS_FINDINGS
streak_state:
  adjudicated_deferrals_respected: true
  reopened_deferrals: none
  respected_list:
    - DRIFT-P5P7-O1, DRIFT-P5P7-O4
    - DRIFT-P5P2-B-O003
    - DRIFT-HS006-ROUTER-DAEMON-STUB
    - DRIFT-P5P4-PROMPT-SHORTID
    - DRIFT-P5P9-STALE-RECONCILIATION-COMMENT
    - DRIFT-P5P14-B-001-VP-SOURCE-BC-VERSION-PIN
    - F-P5P13-A-001/A-002, F-P5P13-B-001 SHIPPED PR #69
    - F-P5P14-B-002 SHIPPED .factory 3994fda
    - F-P5P14-B-003 SHIPPED .factory 426e0fa
    - F-P5P14-B-004/B-005 SHIPPED PR #70
    - F-P5P15-A-001 SHIPPED .factory 5e42768
    - F-P5P15-B-001 SHIPPED .factory 5120c9e
    - VP-077 "Three sub-cases" companion tidy (in-flight, do not raise)
delivered_by: p5-pass16-adv-a (2026-07-03T23:05Z)
adjudication:
  F-P5P16-A-001: SHIPPED at .factory 041ea2f — spec path (b), interface-definitions.md v1.26 → v1.27, $schema line removed from envelope success example, "share a common envelope" prose corrected to reflect schemaless envelope
---

# Phase 5 Pass 16 Adv-A — findings

### F-P5P16-A-001 [MED]: `$schema` envelope field documented but never emitted by sbctl

Anchor: `.factory/specs/prd-supplements/interface-definitions.md:206-213` (spec) vs. `cmd/sbctl/client.go:97-101` + `cmd/sbctl/main.go:104-113` (impl), at develop tip 6deda15.

Class: Public-surface drift — spec claims a JSON output field that the implementation never emits (documented API vs. observed API divergence; operator-UX regression for consumers who validate output against a JSON Schema URL).

Symptom: `interface-definitions.md` §202-213 opens with "All JSON responses share a common envelope" and shows the canonical envelope with a top-level `"$schema": "https://switchboard.example/schemas/v1/response.json"` field. `sbctl` with `--json` never emits this field on any code path. Any operator tooling that consumes sbctl JSON output and dispatches / validates on the `$schema` URL sees no such field and either fails validation or falls back to schemaless parsing.

Verify:
- Spec (success envelope): `.factory/specs/prd-supplements/interface-definitions.md:208` — `  "$schema": "https://switchboard.example/schemas/v1/response.json",`
- Impl (envelope struct): `cmd/sbctl/client.go:97-101` — `type jsonEnvelope struct { OK bool ``json:"ok"``; Error *errorDetail ``json:"error"``; Data json.RawMessage ``json:"data"`` }` — no `$schema` field
- Impl (success constructor): `cmd/sbctl/main.go:104-106` — `func newSuccessEnvelope(data json.RawMessage) jsonEnvelope { return jsonEnvelope{OK: true, Error: nil, Data: data} }` — no $schema initialised
- Impl (error constructor): `cmd/sbctl/main.go:108-113` (inferred by grep at cmd/sbctl/client.go:108-113 range referenced by writeError) — no $schema
- Emission sites: `cmd/sbctl/main.go:142` and `:158` — `json.Marshal(env)` where env is `jsonEnvelope`. `$schema` cannot appear in output.

Secondary symptom (spec-internal inconsistency): the error-envelope example at `interface-definitions.md:216-226` omits `$schema` while the success example at :207-213 includes it, but the section prose at :204 says "All JSON responses share a common envelope." The spec itself is unclear whether `$schema` is universal or success-only. Either intent produces a drift with impl: if universal, the sbctl error envelope is also missing it; if success-only, the "common envelope" prose is wrong.

Remediation shape (one of): (a) add `Schema string ``json:"$schema,omitempty"``` to `jsonEnvelope` in `cmd/sbctl/client.go:97` and populate it in both `newSuccessEnvelope` and `newErrorEnvelope` in `cmd/sbctl/main.go` with the canonical URL — publish the schema document at that URL as a follow-up; OR (b) amend `interface-definitions.md` §205-213 to remove the `$schema` line from the success example and drop "share a common envelope" phrasing, formally documenting that sbctl JSON output is a schemaless envelope. Path (b) is the lower-risk choice given no `switchboard.example` domain exists to host the schema; path (a) is the operator-friendly choice if a schema surface is intended for future publication. Either path requires a version bump on `interface-definitions.md` (currently v1.26) with a changelog row per POL-001.

## Anti-findings (checked and passing)

- Registered Verbs Response Data columns (v1.26): `admin.key.register/revoke/expire` return `{"key_fingerprint", "timestamp"}` — matches `adminKeyResult` struct at `cmd/switchboard/admin_handlers.go:84-87` (`Fingerprint string ``json:"key_fingerprint"```, `At time.Time ``json:"timestamp"```) and return sites at :215-218, :257-260, :336-339.
- `admin.svtn.destroy` Response Data `{"status": "destroyed"}` — matches `admin_handlers.go:866-868`.
- `admin.svtn.create` bootstrap-only authority — matches `admin_handlers.go:737-772` (bootstrap check at :738; diagnostic role at :750-753; canonical E-ADM-009 message shape at :754,771).
- `admin.svtn.destroy` uses general control-role gate (NOT bootstrap-only) — matches `admin_handlers.go:857` calling `resolveAndVerifyCallerRole`, consistent with §409 spec row.
- `decodePublicKey` OpenSSH primary + raw base64 fallback — matches `admin_handlers.go:141-180`; §115 spec description accurate.
- `admin.key.revoke` E-ADM-018 emission text — matches `admin_handlers.go:443` verbatim: "E-ADM-018: control-to-control revocation requires explicit confirmation: use --confirm to proceed".
- `admin.key.revoke` E-ADM-019 emission text — matches `admin_handlers.go:429-432,434-437`.
- `admin.key.expire` daemon-side positive/>100y bounds — matches `admin_handlers.go:311-316`; §110 spec accurate.
- `resolveCallerAdmissionAnyRole` for list-keys (F-L2-003 admission gate retained, authority gate bypassed) — matches `admin_handlers.go:363,592-626` and §111/§145 spec.
- runControl wires `BuildAdminHandlers` (control mode only) — matches `mgmt_wire.go:473`; ADR-004 role-exclusion honored: runRouter is not-implemented stub (`mgmt_wire.go:357-359`), runConsole passes `BuildConsoleHandlers` only (`:419`), runAccess (elsewhere) does not register admin. Consistent with §411 authority note.
- `errorDetail` at `cmd/sbctl/client.go:89-93` carries `Field any ``json:"field"``` matching the error-envelope example at spec :219-224.
- Exit codes 0/1/2/3 semantics (§193-200) map cleanly to `main.go` classification (`usageErrf`→2, connect failures→1). No new drift.
- POL-001 changelog-completeness: v1.26 changelog note (line 143) present and describes the F-P5P15-A-001 shipment; POL-001 satisfied for the most recent bump.
- POL-002 story-index-row-sync: not applicable to this file (`interface-definitions.md` is a PRD supplement, not a story).

---

VERDICT: HAS_FINDINGS
