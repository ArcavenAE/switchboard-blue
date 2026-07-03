---
pass_id: P5P15-Adv-A
adversary_lens: public-surface + operator-UX
prior_passes_read: false
worktree_preflight:
  target_sha: 6deda15def9326f28e96f133e237aff5ecb74d7b
  branch: develop
  HEAD_sha: 6deda15def9326f28e96f133e237aff5ecb74d7b
  refs/heads/develop_sha: 6deda15def9326f28e96f133e237aff5ecb74d7b
  origin/develop_sha: 6deda15def9326f28e96f133e237aff5ecb74d7b
  worktree_root: /Users/skippy/work/aae-orc/run/switchboard-blue
  preflight_result: PASS
budget:
  wall_clock_target: <=6 min
  wall_clock_used: ~5 min
  file_reads_target: <=6
  file_reads_used: 5 substantive (admin_handlers.go full; interface-definitions.md 2 windows; error-taxonomy.md 2 windows; policies.yaml full; plus grep-only scans)
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
    - F-P5P13-A-001, F-P5P13-A-002, F-P5P13-B-001 (shipped PR #69)
    - F-P5P14-B-002 (shipped .factory 3994fda)
    - F-P5P14-B-003 (shipped .factory 426e0fa)
    - F-P5P14-B-004, F-P5P14-B-005 (shipped PR #70)
delivered_by: p5-pass15-adv-a (2026-07-03T22:55Z)
---

# Phase 5 Pass 15 Adv-A — findings

### F-P5P15-A-001 [MED]: interface-definitions.md Response Data for `admin.key.register` / `.revoke` / `.expire` says `{"ok": true}` but wire actually carries `key_fingerprint` + `timestamp` confirmation payload

Anchor: `.factory/specs/prd-supplements/interface-definitions.md:402-404` at develop tip 6deda15 (Response Data column values `{"ok": true}` for `admin.key.register`, `.revoke`, `.expire`) versus `cmd/switchboard/admin_handlers.go:84-87` (adminKeyResult struct with JSON fields `key_fingerprint` and `timestamp`) and returns at `admin_handlers.go:215-218` (register), `:257-260` (revoke), `:336-339` (expire).

Class: public-API drift; interface-definitions.md ↔ mgmt wire divergence; POL-001 changelog completeness (v1.14/v1.15 changelog notes on line 163/165 corrected wire-request fields for these rows but did not correct the Response Data column).

Symptom: The interface-definitions.md Registered Verbs table documents `admin.key.register`, `admin.key.revoke`, and `admin.key.expire` as returning `{"ok": true}` — the same shorthand as truly empty-data responses. The actual daemon response envelope carries `data: {"key_fingerprint": "SHA256:<base64>", "timestamp": "<RFC3339>"}` per BC-2.05.004 PC-4 (confirmation with fingerprint and timestamp). Compare row 399 (`paths.list`) which correctly enumerates the data payload structure — the column is meant to describe the `data` payload, so `{"ok": true}` in these rows reads as "no data payload" to an operator writing a client. An operator who wants to log the key fingerprint or the exact server-side timestamp of a register/revoke/expire has no way to know from the spec that both fields are emitted. The test at `cmd/switchboard/admin_handlers_test.go:123-124` further pins the wire contract to `key_fingerprint` (canonical `SHA256:<base64>`) + `timestamp`. This is a documentation drift on the public wire surface for three of the six admin verbs.

Verify:
- `cmd/switchboard/admin_handlers.go:84-87` — `adminKeyResult` struct declares `Fingerprint string \`json:"key_fingerprint"\`` and `At time.Time \`json:"timestamp"\``.
- `cmd/switchboard/admin_handlers.go:215-218, 257-260, 336-339` — all three key-lifecycle handlers return `adminKeyResult{Fingerprint, At}`.
- `cmd/switchboard/admin_handlers_test.go:123-124` — pins wire fields to `key_fingerprint`/`timestamp` and enforces canonical `SHA256:...` prefix.
- `.factory/specs/prd-supplements/interface-definitions.md:399` — the shorthand convention: `paths.list` Response Data column shows the data payload structure (not just `{"ok": true}`).
- `.factory/specs/prd-supplements/interface-definitions.md:402-404` — the three drifting rows.
- `internal/mgmt/mgmt.go:443-451` — `rpcResponseMsg{OK, Error, Data any}` — `Data` is the marshalled handler return value.

Remediation shape: Update the Response Data column for `admin.key.register`, `admin.key.revoke`, and `admin.key.expire` to `{"key_fingerprint": "SHA256:<base64>", "timestamp": "<RFC3339>"}` (matching the BC-2.05.004 PC-4 confirmation surface and the existing AC-001 wire contract). Add a v1.16 changelog row under the header changelog table citing the correction. Consider aligning `admin.svtn.destroy` row (`.md:407`) similarly: it documents `{"ok": true, "status": "destroyed"}` in a mixed shorthand, but the wire actually emits `data: {"status": "destroyed"}` — spec-shorthand of `{"status": "destroyed"}` in the Response Data column would be canonical with the `paths.list` and `admin.svtn.create` rows.

Adjudication: **SHIPPED** at `.factory` commit `5e42768` (Burst 42a) — interface-definitions.md v1.25 → v1.26. All 4 rows (admin.key.register/revoke/expire + admin.svtn.destroy) corrected. Changelog row added.

## Anti-findings (checked and passing)

- E-ADM-020 canonical text at `error-taxonomy.md:96` (`"cannot revoke the bootstrap key in SVTN <svtn_id> (permanent trust anchor)"`) is byte-identical to impl at `admin_handlers.go:445` — POL-001 clean for the v3.7 rewrite.
- E-ADM-021 canonical text at `error-taxonomy.md:97` matches impl at `admin_handlers.go:447` — bootstrap-key-expire-forbidden format string aligned.
- E-INT-999 catch-all: impl at `admin_handlers.go:458` emits `"E-INT-999: unmapped internal condition, programmer error, please report: %w"` matching taxonomy line 201 canonical text and Ruling-12 §1.
- E-SVTN-001 message shape ("SVTN already exists: <name>") — impl at `admin_handlers.go:642` matches taxonomy line 176 (v4.5 <svtn_id>→<svtn_name> placeholder correction).
- E-SVTN-003 message shape ("SVTN not found: <name>") — impl at `admin_handlers.go:659` matches taxonomy line 178.
- E-ADM-018 canonical text at `error-taxonomy.md:94` (`"use --confirm to proceed"`, no `=<svtn-id>` suffix) matches impl at `admin_handlers.go:443` — v3.5 + v4.4 corrections held.
- E-ADM-013 impl message ("no key with fingerprint <fp> registered in SVTN <name>") at `admin_handlers.go:676` aligns with taxonomy line 91 v4.5 substitution semantics.
- E-CFG-001 validation ordering: `admin.svtn.destroy` handler correctly validates UTF-8 (`admin_handlers.go:832`), then `validateSVTNName` (`:843`), then authority — matches F-P5P8-A-004 wire-contract expectation.
- `admin.key.list-keys` correctly emits `[]` not `null` — `admin_handlers.go:373` (`make([]adminKeyEntry, 0, ...)`); EC-003 non-null-array invariant held.
- `admin.svtn.create` bootstrap-only authority gate (Ruling-5): defense-in-depth at `admin_handlers.go:737-772` — pubkey identity + bootstrap RoleControl check — matches interface-definitions.md:406 "bootstrap-only" wording and BC-2.07.001 Inv-3.
- CWE-862 defense on `admin.key.list-keys`: `resolveCallerAdmissionAnyRole` (`admin_handlers.go:592-626`) — any-role admission enforcement present; cross-SVTN enumeration blocked; matches F-L2-003 amendment.
- CWE-200: E-ADM-010 wire response format ("authentication failed" without `key_fingerprint`) documented at `error-taxonomy.md:88` to prevent enumeration oracle; sendAuthFail at `mgmt.go:474-483` matches.

---

VERDICT: HAS_FINDINGS
