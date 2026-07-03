---
pass_id: P5P17-Adv-A
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
  file_reads_target: <=6
  file_reads_used: 6
verdict: HAS_FINDINGS
findings_count: 2
anti_findings_count: 7
policies_applied:
  - POL-001 (changelog-completeness)
  - POL-002 (story-index-row-sync)
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
    - F-P5P14-B-002/B-003 SHIPPED .factory
    - F-P5P14-B-004/B-005 SHIPPED PR #70
    - F-P5P15-A-001 SHIPPED .factory 5e42768
    - F-P5P15-B-001 SHIPPED .factory 5120c9e
    - F-P5P16-A-001 SHIPPED .factory 041ea2f
    - VP-077 "Three sub-cases" companion tidy (in-flight, do not raise)
delivered_by: p5-pass17-adv-a (2026-07-03)
adjudication:
  DRIFT-P5P17-A-001: SHIPPED at .factory 2be16e5 — interface-definitions.md v1.27 → v1.28, svtn_id line removed from router.metrics envelope example, Registered Verbs table Response Data column corrected
  DRIFT-P5P17-A-002: SHIPPED at .factory 2be16e5 — same commit, path_distribution example values corrected from fractional ratios (0.52, 0.48) to integer frame counts (900000, 334567) consistent with map[string]uint64 wire type
---

# Adversarial Review — Pass 17 (Adv-A, fresh context)

## Critical Findings

_none_

## Important Findings

### DRIFT-P5P17-A-001 — `router.metrics` response spec claims `svtn_id` echo field the daemon never emits

- **Class:** spec-vs-impl drift (operator-visible RPC response shape)
- **Confidence:** HIGH
- **Severity:** MED (POL-001 candidate — spec envelope example drifted from canonical BC + impl without a reconciling changelog note)
- **Anchors:**
  - Spec claim (envelope example): `.factory/specs/prd-supplements/interface-definitions.md:299` — `"data": { "svtn_id": "...", "frame_count": ..., ... }`
  - Spec claim (Registered Verbs table): `.factory/specs/prd-supplements/interface-definitions.md:403` — Response Data column lists `{"svtn_id", "frame_count", "hmac_fail_count", "drop_cache_hits", "path_distribution"}`
  - Impl (CLI decode shape): `cmd/sbctl/router_metrics.go:25-30` — `type RouterMetrics struct { FrameCount uint64; HMACFailCount uint64; DropCacheHits uint64; PathDistribution map[string]uint64 }` — no SVTNID field
  - Impl (daemon response type): `internal/metrics/types.go:101-112` — `RouterMetricsResponse` has the same four fields, no svtn_id
  - Canonical BC vector (authority): `.factory/specs/behavioral-contracts/ss-06/BC-2.06.003.md:109` — `{"frame_count":<n>,"hmac_fail_count":<n>,"drop_cache_hits":<n>,"path_distribution":{<path_id>:<frame_count>}}` — svtn_id absent
- **Symptom:** An operator reading interface-definitions.md §Router metrics response would code a downstream consumer that reads `data.svtn_id` from the JSON envelope to correlate the metrics back to the queried SVTN. The daemon never writes that field. `json.Unmarshal` into a struct with `SVTNID string` yields zero-value (silent), but any strict-schema consumer (jq path assertion, integration test, JSON-schema validator) will flag the absent field. This is a spec-authored, impl-absent phantom field — the exact class of consumer-facing drift POL-001 targets.
- **Corroboration:** The canonical BC-2.06.003 v1.15 test vector at line 109 omits `svtn_id`; the demo-evidence `stub_daemon.go` (S-5.02 evidence tree) also omits it; the daemon handler at `internal/metrics/handlers.go:79-98` reads `svtn_id` from the request but does not echo it into the response.
- **Remediation shape (spec-side):**
  1. `interface-definitions.md:299` — remove the `"svtn_id": "...",` line from the Router metrics response example.
  2. `interface-definitions.md:403` — remove `"svtn_id"` from the Response Data column for `router.metrics`.
  3. Bump interface-definitions.md changelog to v1.28 with a POL-001-compliant entry.
  4. No BC or impl change required — the spec is the anomaly.

### DRIFT-P5P17-A-002 — `path_distribution` values shown as fractional ratios (`0.52`, `0.48`) contradict the `uint64` frame-count wire type

- **Class:** spec-vs-impl drift (operator-visible value semantics; JSON numeric-type mismatch)
- **Confidence:** HIGH
- **Severity:** MED (POL-001 candidate — same class as A-001; consumer parse breakage under Go/Rust/typed clients)
- **Anchors:**
  - Spec claim: `.factory/specs/prd-supplements/interface-definitions.md:303-305` — `"path_distribution": { "path-001": 0.52, "path-002": 0.48 }`
  - Impl (CLI decode shape): `cmd/sbctl/router_metrics.go:29` — `PathDistribution map[string]uint64 \`json:"path_distribution"\``
  - Impl (daemon response type): `internal/metrics/types.go:110-111` — `PathDistribution map[string]uint64` (per-path frame count for the SVTN)
  - Canonical BC vector (authority): `.factory/specs/behavioral-contracts/ss-06/BC-2.06.003.md:109` — `"path_distribution":{<path_id>:<frame_count>}` — explicitly frame-count, not ratio
  - Demo evidence: `docs/demo-evidence/S-5.02/stub_daemon.go` — emits integer frame counts
- **Symptom:** The `0.52` / `0.48` values in the spec example look like a distribution ratio summing to 1.0. A downstream integration reading the spec will (a) size their percentage-formatting UI code accordingly, (b) fail JSON-decode when they hit a Go/Rust typed consumer wiring `map[string]uint64` against a producer emitting `0.52`, and (c) miscompute traffic-share dashboards. The wire type is per-path integer frame count, not a probability ratio.
- **Corroboration:** BC PC-2 at line 68 describes "per-path frame distribution" (count, not ratio). `internal/metrics/handlers_test.go:360-388` uses integer values (600, 400) confirming the wire contract.
- **Remediation shape (spec-side):**
  1. `interface-definitions.md:303-305` — change example values to integer frame counts consistent with BC and demo-evidence.
  2. Include this change in the same v1.28 POL-001 changelog entry as A-001.
  3. No BC or impl change required.

## Observations

_none — both real drifts have been elevated to Important._

## Anti-findings (things checked that passed)

1. **`admin svtn destroy` short-ID confirm prompt string** — `cmd/sbctl/admin.go:393` emits the literal `"Type the SVTN short-ID (e.g. SVTN-abcd1234) to confirm: "` — matches interface-definitions.md:134 DRIFT-P5P4-PROMPT-SHORTID adjudicated interim rendering; not re-raised.
2. **`admin key revoke` confirm-gate carve-out** — `cmd/sbctl/admin.go:483-518` does NOT route through `runDestroyConfirmGate`; `--role` is required at :501-502 with enum validation at :504-509 — matches §109 v1.23 update and §131 carve-out for BC-2.05.004 EC-005.
3. **`admin key expire --after` flag name (not `--at`)** — `cmd/sbctl/admin.go:531` registers `--after`; pre-validation at :550-555 rejects zero/negative — matches §110 v1.22 correction.
4. **`console attach|detach|switch` flag set** — only `--session` is present at `cmd/sbctl/console.go:91,140`; no `--console` or `--svtn` — matches §86-88 v1.19 F-P5P6-A-004 amendment.
5. **Exit-code taxonomy: usageError → 2, else → 1** — `cmd/sbctl/main.go:104-111` maps `*usageError` via `errors.As` → `os.Exit(2)`; all other errors → `os.Exit(1)`; internal marshal-error at :145 → `os.Exit(3)` — matches §197-202 exit code table.
6. **JSON envelope shape: `{ok, error, data}` with no `$schema`** — `cmd/sbctl/client.go:97-101` jsonEnvelope, marshaled by writeSuccess/writeError at main.go:139-167; no schema pointer field — matches v1.27 changelog note (F-P5P16-A-001 shipped).
7. **`sbctl admin svtn create` wire args = `{"name"}`** — admin.go:96-99 sends adminSVTNCreateArgs{Name string `json:"name"`} — matches §121 and §409 (`{"name": "<svtn-name>"}`).

---

VERDICT: HAS_FINDINGS
