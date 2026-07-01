# Evidence Report — S-5.02: sbctl per-path metrics (paths list / router metrics / router status)

**Story:** S-5.02 v1.10  
**Branch:** feat/S-5.02-sbctl-metrics-query  
**Recorded:** 2026-06-30  
**Toolchain:** VHS 0.11.0 (terminal recordings)  
**Stub daemon:** `docs/demo-evidence/S-5.02/stub_daemon.go` — ADR-012 handshake + canned JSON per mode

---

## Coverage Summary

| AC | Description | Coverage | Demo File |
|----|-------------|----------|-----------|
| AC-001 | `sbctl paths list` PC-1 schema (float64 rtt_p99_ms >=10 samples) | FULL | AC-001-paths-list-canonical-fields.gif |
| AC-002 | `sbctl router metrics --svtn=<id>` PC-2 schema | FULL | AC-002-router-metrics-svtn.gif |
| AC-003 | `sbctl router status` alias + quality column | FULL | AC-003-router-status-alias.gif |
| AC-004 | rtt_p99_ms string "pending" when <10 samples | FULL | AC-004-p99-pending-less-than-10-samples.gif |
| AC-005 | p99 histogram accuracy TestBC_2_06_003_P99HistogramAccuracy | FULL | AC-005-p99-histogram-accuracy.gif |
| AC-006 | --json flag + E-NET-001 unreachable daemon (exit 1) | FULL | AC-006-json-flag-and-daemon-unreachable.gif |
| AC-008 | quality="pending" when rtt_p99_ms="pending" | FULL | AC-008-quality-pending-when-p99-pending.gif |

**All 7 ACs: FULL coverage. No DEMO-ISSUE flags.**

---

## Per-AC Detail

### AC-001 — paths list canonical schema (BC-2.06.003 PC-1)

**Recording:** `AC-001-paths-list-canonical-fields.gif` / `.webm`  
**Demo shows:** `sbctl --json paths list` against stub daemon returning 3 paths with >=10 samples. JSON envelope contains all 6 required fields: `path_id`, `router_addr`, `rtt_ms`, `rtt_p99_ms` (float64 e.g. `22.7`), `loss_pct`, `status`. Output formatted with `python3 -m json.tool` for readability.  
**Stub mode:** `paths-full`

---

### AC-002 — router metrics --svtn schema (BC-2.06.003 PC-2)

**Recording:** `AC-002-router-metrics-svtn.gif` / `.webm`  
**Demo shows:** `sbctl --json router metrics --svtn=abc123` against stub daemon. JSON envelope data contains all 4 required PC-2 fields: `frame_count` (12345), `hmac_fail_count` (3), `drop_cache_hits` (7), `path_distribution` (map).  
**Stub mode:** `router-metrics`

---

### AC-003 — router status alias + quality column (BC-2.06.003 PC-3)

**Recording:** `AC-003-router-status-alias.gif` / `.webm`  
**Demo shows:** Two views — (1) human-readable table with `QUALITY` column showing `green`/`yellow` computed from p99+loss+status; (2) `--json` output with injected `quality` field beside all canonical PathEntry fields. Confirms single `paths.list` RPC dispatch (same as paths list).  
**Stub mode:** `router-status`

---

### AC-004 — rtt_p99_ms string "pending" when <10 samples (BC-2.06.003 EC-003)

**Recording:** `AC-004-p99-pending-less-than-10-samples.gif` / `.webm`  
**Demo shows:** `sbctl --json paths list` against stub returning `rtt_p99_ms: "pending"` (string). JSON output confirms `"pending"` is a string not null, not 0.  
**Stub mode:** `paths-pending`

---

### AC-005 — p99 histogram accuracy (internal/paths, ARCH-03 v1.6)

**Recording:** `AC-005-p99-histogram-accuracy.gif` / `.webm`  
**Demo shows:** Live execution of `go test -v -run TestBC_2_06_003_P99HistogramAccuracy ./internal/paths/`. All 5 sub-cases pass: all-samples-in-0-25ms-bucket, samples-across-0-25ms-and-100-150ms, p99-in-150-200ms-bucket, p99-in-200-300ms-coarse-bucket, p99-in-unbounded-bucket. PASS shown on screen.  
**Note:** This AC is purely internal/paths accuracy — no daemon RPC involved. The demo uses `go test` as evidence rather than a CLI demo because there is no CLI surface to exercise the histogram directly.

---

### AC-006 — --json flag + E-NET-001 unreachable daemon (BC-2.06.003 PC-4, PC-5)

**Recording:** `AC-006-json-flag-and-daemon-unreachable.gif` / `.webm`  
**Demo shows two paths:**
- **Success path:** `sbctl --json paths list` returns `{"ok":true,"error":null,"data":[...]}` envelope
- **Error path:** unreachable socket returns `{"ok":false,"error":{"code":"E-NET-001",...},"data":null}` to stderr; process exits with code 1

---

### AC-008 — quality="pending" when rtt_p99_ms="pending" (BC-2.06.003 v1.7 PC-3 F-M3 + EC-006)

**Recording:** `AC-008-quality-pending-when-p99-pending.gif` / `.webm`  
**Demo shows:** `sbctl router status` against stub daemon returning `rtt_p99_ms: "pending"`. Both human-readable table (QUALITY column shows `pending`) and `--json` output (quality field is string `"pending"`) are demonstrated. Client-side `qualityFromPathEntry` derivation confirmed.  
**Stub mode:** `router-status-pending`

---

## Recordings Index

| File | Size | AC |
|------|------|----|
| AC-001-paths-list-canonical-fields.gif | 146K | AC-001 |
| AC-001-paths-list-canonical-fields.webm | 100K | AC-001 |
| AC-002-router-metrics-svtn.gif | 106K | AC-002 |
| AC-002-router-metrics-svtn.webm | 93K | AC-002 |
| AC-003-router-status-alias.gif | 172K | AC-003 |
| AC-003-router-status-alias.webm | 176K | AC-003 |
| AC-004-p99-pending-less-than-10-samples.gif | 88K | AC-004 |
| AC-004-p99-pending-less-than-10-samples.webm | 62K | AC-004 |
| AC-005-p99-histogram-accuracy.gif | 242K | AC-005 |
| AC-005-p99-histogram-accuracy.webm | 120K | AC-005 |
| AC-006-json-flag-and-daemon-unreachable.gif | 133K | AC-006 |
| AC-006-json-flag-and-daemon-unreachable.webm | 141K | AC-006 |
| AC-008-quality-pending-when-p99-pending.gif | 153K | AC-008 |
| AC-008-quality-pending-when-p99-pending.webm | 166K | AC-008 |

---

## DEMO-ISSUE Log

No issues. All 7 ACs recorded at FULL coverage level.
