---
artifact_id: wave-3-consistency-audit
document_type: consistency-report
level: ops
version: "1.0"
producer: consistency-validator
timestamp: 2026-06-27T00:00:00Z
traces_to: .factory/cycles/cycle-1/
---

# Wave-3 Consistency Audit — Fresh-Context Cross-Document Report

**Audit date:** 2026-06-27  
**Auditor:** consistency-validator (fresh context, no prior session)  
**Scope:** All Wave-3 stories (S-3.01a, S-3.01b, S-3.03, S-3.04, S-W3.04, S-W3.05)  
**Ground truth:** Ground-truth sources verified independently against git log, file contents, and error strings.

---

## Executive Summary

| Category | PASS / FAIL | Blockers |
|----------|------------|---------|
| 1. Spec↔Code traceability | FAIL | 3 findings (1 HIGH, 2 MEDIUM) |
| 2. Story↔BC anchoring | FAIL | 2 findings (1 HIGH, 1 MEDIUM) |
| 3. Index integrity | FAIL | 3 findings (1 CRITICAL, 1 HIGH, 1 MEDIUM) |
| 4. Cross-spec contradiction | PASS | 0 blockers (1 LOW) |
| 5. Coverage gaps | FAIL | 1 finding (1 HIGH) |

**Overall gate:** FAIL — 4 blocking findings (1 CRITICAL + 3 HIGH). Must be resolved before Wave-3 gate can PASS.

---

## Dimension 1: Spec↔Code Traceability

### F-1.1 — CRITICAL: S-W3.04 (`cmd/switchboard/access.go`, `internal/tmux/connector_frames.go`) NOT on local `develop` branch

**Severity:** CRITICAL  
**Files (spec side):** `.factory/STATE.md` lines 126–129 (`s_w3_04_pr_number: 17`, `s_w3_04_merge_sha: aeb442d`, `s_w3_04_merge_date: 2026-06-27`, `s_w3_04_status: completed`)  
**Files (code side):** `cmd/switchboard/main.go` (current), `cmd/switchboard/access.go` (absent from local develop)  

**Finding:** The audit was told S-W3.04 (PR #17, merge `aeb442d`) is merged into develop. Ground-truth verification shows:

```
$ git merge-base --is-ancestor aeb442d HEAD → exit 1 (NOT ancestor)
$ git log develop..origin/develop → aeb442d feat(S-W3.04): full daemon assembly — wire all Wave-3 subsystems (#17)
```

- **`origin/develop`** is at `aeb442d` (S-W3.04 merged on top of S-W3.05).  
- **Local `develop`** is at `fa6345e` (S-W3.05 is the tip; S-W3.04 not yet pulled).  
- Local `develop` is exactly **1 commit behind** `origin/develop`.  
- `cmd/switchboard/access.go`, `internal/tmux/connector_frames.go`, `internal/tmux/connector_eof_test.go`, `internal/tmux/connector_frames_test.go`, `internal/tmux/connector_toctou_test.go`, `cmd/switchboard/access_test.go` all exist on `origin/develop` but NOT on local `develop`.  

**Canonical source of truth:** `git log origin/develop` (commits `fa6345e` < `aeb442d`).  
**Impact:** All code-side checks for S-W3.04 in this audit are against `origin/develop` content (verified via `git show origin/develop:<path>`). The local working tree does NOT reflect merged state. Any tooling running against the local working tree (CI, linters, tests) will not see S-W3.04 code.  
**Recommended fix:** `git pull origin develop` on the local workstation to advance local `develop` to `aeb442d`.  
**Owner:** state-manager (sync local branch to remote; update any local-branch-dependent tooling).

---

### F-1.2 — HIGH: E-ADM-016 log message format diverges between error-taxonomy.md and `routing.go` PATH-A

**Severity:** HIGH  
**Spec side:** `.factory/specs/prd-supplements/error-taxonomy.md` v2.1, §ADM table, row E-ADM-016, Message Format column:  
`"wire HMAC verification failed at RouteFrame: tag mismatch for SVTN <svtn_id> from src <src_addr> (E-ADM-016)"`  
**Code side:** `internal/routing/routing.go` line 200 (PATH-A — no forwarding entry):  
`"wire HMAC verification failed at RouteFrame: auth key unavailable for SVTN %x from src %x (E-ADM-016)"`  
**Also:** `internal/routing/routing.go` line 215 (PATH-B — tag mismatch):  
`"wire HMAC verification failed at RouteFrame: tag mismatch for SVTN %x from src %x (E-ADM-016)"` — MATCHES taxonomy.

**Finding:** The error taxonomy canonizes a single E-ADM-016 message format (`"tag mismatch …"`). PATH-A uses a substantively different sub-phrase (`"auth key unavailable …"`). BC-2.05.008 EC-003/PC-4 requires logging E-ADM-016 on the PATH-A path but does NOT specify the exact message; the taxonomy does specify a canonical format (only for the tag-mismatch case). The result is two distinct operator-visible strings emitted under the same E-ADM-016 code, one of which does not match the taxonomy entry. Operators relying on grep patterns against the taxonomy's canonical format will miss PATH-A alerts.

**Canonical source of truth:** `.factory/specs/prd-supplements/error-taxonomy.md` v2.1 — the taxonomy is the canonical source for message format strings.  
**Recommended fix (option A):** Add a second row to the E-ADM-016 taxonomy entry for the auth-key-unavailable sub-case: `"wire HMAC verification failed at RouteFrame: auth key unavailable for SVTN <svtn_id> from src <src_addr> (E-ADM-016)"`. This canonizes the existing code behavior.  
**Recommended fix (option B):** Change `routing.go` line 200 to match the taxonomy format: `"wire HMAC verification failed at RouteFrame: tag mismatch for SVTN %x from src %x (E-ADM-016)"` (accepting that PATH-A and PATH-B produce identical messages, with distinction only in the Go return path).  
**Owner:** product-owner (decide option A or B and update taxonomy); implementer (if option B, update routing.go line 200).

---

### F-1.3 — MEDIUM: `ErrUpstreamReadOnly` sentinel message does not match E-ADM-007 taxonomy format

**Severity:** MEDIUM  
**Spec side:** `.factory/specs/prd-supplements/error-taxonomy.md` v2.1, §ADM table, row E-ADM-007, Message Format:  
`"upstream rejected: read-only access for console <key_fingerprint> on session <session_name>"`  
**Code side:** `internal/session/auth.go` line 50:  
`var ErrUpstreamReadOnly = errors.New("session: upstream rejected: read-only access (E-ADM-007)")`

**Finding:** The taxonomy message format includes `<key_fingerprint>` and `<session_name>` parametric fields. The code sentinel is a static string omitting both fields, adding a "session:" prefix and "(E-ADM-007)" suffix. The taxonomy also contains a layering note acknowledging that the session layer omits `<node_addr>`, but the `<key_fingerprint>` and `<session_name>` omissions are not acknowledged in the layering note. The actual emitted message will not satisfy a grep against the taxonomy template.

**Note:** BC-2.04.005 v1.x and error-taxonomy v2.1 both include the layering note for E-ADM-007 stating `<node_addr>` is legitimately omitted at the session layer. However, the taxonomy does NOT grant an exemption for omitting `<key_fingerprint>` or `<session_name>`.  
**Canonical source of truth:** `.factory/specs/prd-supplements/error-taxonomy.md` v2.1 §E-ADM-007.  
**Recommended fix:** Either (a) update the taxonomy layering note to acknowledge that the session-layer sentinel is a static string (static sentinels are idiomatic Go; parametric detail is added by callers via `%w` wrapping), or (b) update `auth.go:50` to use a format string carrying key and session context via `fmt.Errorf`. Option (a) is lower risk for Wave 3.  
**Owner:** product-owner (update taxonomy layering note); implementer (if option b, update auth.go).

---

## Dimension 2: Story↔BC Anchoring

### F-2.1 — HIGH: STORY-INDEX S-W3.04 row omits BC-2.04.007 from BC-traces column; status is stale

**Severity:** HIGH  
**Spec side (story frontmatter):** `.factory/stories/S-W3.04-daemon-assembly.md` frontmatter lines 16–24:  
```yaml
bc_traces:
  - BC-2.04.001 … BC-2.04.007
  - BC-2.05.008
```
(BC-2.04.007 IS present in story frontmatter)  
**Spec side (index):** `.factory/stories/STORY-INDEX.md` line 49 BC-traces column:  
`BC-2.04.001, BC-2.04.002, BC-2.04.003, BC-2.04.004, BC-2.04.005, BC-2.04.006, BC-2.05.008`  
**Missing:** BC-2.04.007 is absent from the STORY-INDEX row for S-W3.04.  
**Also:** STORY-INDEX line 49 Status column: `draft (fix-now F-1; BC-2.04.007 PO decision pending)` — stale; story is `ready` (per story frontmatter `status: ready`) and BC-2.04.007 was authored on 2026-06-27.  
**Also:** STORY-INDEX §BC Coverage Check (line 82): `"All 43 BCs covered (42 original + BC-2.05.008 minted Wave 3)"` — stale; actual count is 44 (BC-2.04.007 added).  
**Also:** STORY-INDEX §BC Coverage Check (line 84): `"Open gap: BC-2.04.007 (daemon lifecycle) does not yet exist"` — stale; BC-2.04.007 was authored and is now active.  

**Canonical source of truth:** `.factory/stories/S-W3.04-daemon-assembly.md` frontmatter (the story file is the BC-trace source of truth per VSDD convention).  
**Recommended fix:** Update STORY-INDEX.md: (1) add BC-2.04.007 to S-W3.04 BC-traces column; (2) change S-W3.04 status to `completed (PR #17, merge aeb442d)`; (3) update BC Coverage Check line to `"44 BCs covered"`; (4) remove "Open gap: BC-2.04.007 does not yet exist" sentence.  
**Owner:** state-manager (STORY-INDEX status update); story-writer (BC-traces column correction).

---

### F-2.2 — MEDIUM: S-W3.05 story (v1.3) references BC-2.05.005 at v1.7; actual BC is v1.8

**Severity:** MEDIUM  
**Spec side (story):** `.factory/stories/S-W3.05-hmac-failure-counter.md` Spec Patches v1.3 (last line): `"update all forward-facing BC-2.05.005 v1.6 references to v1.7"`. Architecture Compliance table line 352: `"BC-2.05.005 v1.7 PC-3"`.  
**Spec side (BC):** `.factory/specs/behavioral-contracts/ss-05/BC-2.05.005.md` frontmatter `version: "1.8"`.  
**Finding:** Story S-W3.05 v1.3 was written when BC-2.05.005 was v1.7. BC-2.05.005 was subsequently advanced to v1.8 (spec-hygiene alignment per adversary OBS-3). The story's BC version citations are now one version behind. No behavioral change was introduced in v1.8 (it is spec-hygiene only per the BC changelog), so this is a reference drift, not a semantic error.  
**Canonical source of truth:** `.factory/specs/behavioral-contracts/ss-05/BC-2.05.005.md` (`version: "1.8"`).  
**Recommended fix:** Update S-W3.05 Spec Patches entry and Architecture Compliance table to reference `BC-2.05.005 v1.8`. Low priority — no behavioral impact.  
**Owner:** story-writer (update version references in story body).

---

## Dimension 3: Index Integrity

### F-3.1 — CRITICAL: Local `develop` branch is behind `origin/develop`; S-W3.04 implementation absent from local working tree

(See F-1.1 above — same root cause. Classified CRITICAL here because it means the local codebase used for any local test run, linting, or `go build` does NOT include the S-W3.04 implementation. Any process relying on the local working tree for Wave-3 gate verification is operating against an incomplete codebase.)

---

### F-3.2 — HIGH: STORY-INDEX states S-W3.04 as `draft`; STATE.md records it as `completed`

**Severity:** HIGH  
**Spec side (STORY-INDEX):** `.factory/stories/STORY-INDEX.md` line 49: Status = `draft (fix-now F-1; BC-2.04.007 PO decision pending)`, Summary line 23: `Pending: 15 (includes S-W3.04 draft, S-W3.05 ready)`.  
**Spec side (STATE.md):** `.factory/STATE.md` lines 127–129: `s_w3_04_pr_number: 17`, `s_w3_04_merge_sha: aeb442d`, `s_w3_04_status: completed`.  
**Finding:** STORY-INDEX was not updated when S-W3.04 was merged. These two indexes are in direct contradiction. STORY-INDEX implies S-W3.04 is in-flight (draft), while STATE.md records it as completed/merged. BC-INDEX line 52 also still shows BC-2.04.007 as `active` (not `implemented (S-W3.04 / PR #17)`) while STATE.md records the story completed.  
**Canonical source of truth:** STATE.md is the pipeline state source of truth. STORY-INDEX must reflect it.  
**Recommended fix:** Update STORY-INDEX.md: change S-W3.04 status to `completed (PR #17, merge aeb442d)`; update Summary line `Complete` count from 11 to 13 (add S-W3.04 and S-W3.05); update `Pending` count from 15 to 13; update Wave 3 summary row to reflect both completed. Update BC-INDEX: change BC-2.04.007 status from `active` to `implemented (S-W3.04 / PR #17)`; change BC-2.05.005 status from `partially implemented` to `implemented (S-W3.05 / PR #16)`.  
**Owner:** state-manager (STORY-INDEX and BC-INDEX status updates post-merge).

---

### F-3.3 — MEDIUM: ARCH-01-core-services.md frontmatter `version: "1.5"` diverges from changelog (v1.6 content present)

**Severity:** MEDIUM  
**File:** `.factory/specs/architecture/ARCH-01-core-services.md` line 5: `version: "1.5"`.  
**Contradicting content:** ARCH-01 changelog entries include a v1.6 entry (line 15 of the `modified` array):  
`# v1.6 — ADR-011 §HIGH-A tightened: TOCTOU fix — {src, srcCh, inPTYMode} MUST be read as a single atomic snapshot…`  
The ADR-011 section body explicitly contains v1.6 content at lines 305, 327, 336, 417, 421, 423, 539.  
**Finding:** The `modified` changelog array and the ADR-011 body content reference v1.6 decisions. The frontmatter `version` field still says `1.5`. The TOCTOU fix (atomic snapshot via `activeSourceSnapshot()`) was documented in the body but the frontmatter was not bumped. STATE.md line 138 correctly references `ARCH-01 v1.6` as a pending check-input-drift item, confirming the version mismatch was known but not yet resolved.  
**Canonical source of truth:** The canonical frontmatter version should match the highest changelog entry present in the document body.  
**Recommended fix:** Bump `ARCH-01-core-services.md` frontmatter `version: "1.5"` to `version: "1.6"`. This is a cosmetic fix; no content change needed.  
**Owner:** architect (frontmatter version bump).

---

### F-3.4 — LOW: ARCH-INDEX §Context Engineering note says "All 42 BCs" — stale count

**Severity:** LOW  
**File:** `.factory/specs/architecture/ARCH-INDEX.md` context engineering note: `"All 42 BCs are covered"`.  
**Actual count:** BC-INDEX Coverage Summary total is 44. BC-2.04.007 (Wave-3) and BC-2.05.008 (Wave-3) were added after ARCH-INDEX was written.  
**Canonical source of truth:** BC-INDEX Coverage Summary total (44).  
**Recommended fix:** Update ARCH-INDEX context engineering note to `"All 44 BCs are covered"`.  
**Owner:** architect (ARCH-INDEX BC count update).

---

## Dimension 4: Cross-Spec Contradiction

### F-4.1 — LOW: E-ADM-016 canonical message (taxonomy v2.1) only registers PATH-B; PATH-A unlabeled

**Severity:** LOW (overlaps with F-1.2; distinct cross-spec angle)  
**Spec side A:** `.factory/specs/prd-supplements/error-taxonomy.md` v2.1 — E-ADM-016 message format row defines only the `"tag mismatch"` variant.  
**Spec side B:** `.factory/specs/behavioral-contracts/ss-05/BC-2.05.008.md` v1.3 — PC-4 states the PATH-A (auth-key-unavailable) path also returns `ErrHMACVerificationFailed` and logs E-ADM-016, but gives no exact message format.  
**Finding:** BC-2.05.008 PC-4 requires an E-ADM-016 log entry for PATH-A but defers the format to the implementer (no prescribed format in the BC). The taxonomy only canonizes the PATH-B format. These two documents are not in direct contradiction (neither claims PATH-A must use the "tag mismatch" wording), but the gap creates an underdefined operator surface that an adversarial reviewer would flag as a traceability hole.  
**No contradiction** — the two specs are consistent (BC delegates message format; taxonomy supplies one canonical form for one sub-case). This is a gap, not a conflict.  
**Recommended fix:** Add a note to error-taxonomy.md E-ADM-016 row or BC-2.05.008 PC-4 specifying the canonical PATH-A format string. See F-1.2 for the concrete text.  
**Owner:** product-owner (extend E-ADM-016 taxonomy entry).

### Check: EC catalogs, invariants, and PC-2.6 path agreement

**Result: PASS (no contradictions found)**

Verified across BC-2.04.002 (EC-002/EC-003/EC-008), BC-2.04.006 (Inv-4 dual counter), BC-2.04.007 (PC-2.6/EC-007/Inv-5), and S-W3.04 story:

- **EC-002** (BC-2.04.001): tmux control mode drops mid-operation → reconnect, fallback. Consistent with BC-2.04.002 EC-003 and S-W3.04 EC-002. ✓  
- **EC-005** (BC-2.04.002): tmux old version → PTY fallback with specific log. Consistent across BC-2.04.001, BC-2.04.002, S-W3.04. ✓  
- **EC-007** (BC-2.04.007): mid-session double-failure or PTY-source EOF → PC-2.6 drain path → E-SYS-002 log + exit 1. Consistent across BC-2.04.002 EC-008, BC-2.04.007 EC-007, S-W3.04 AC-007/AC-009. ✓  
- **EC-008** (BC-2.04.002): PTY-source EOF (hot-spin prevention). Consistent with BC-2.04.007 EC-007 and S-W3.04 AC-009. ✓  
- **Inv-3** (BC-2.04.002, never-silent): PTY-source EOF MUST surface as `ErrPTYSourceEOF` on `sc.Err()` and produce E-SYS-002 log. Consistent with BC-2.04.007 Inv-5 and error-taxonomy v2.1 E-SYS-003. ✓  
- **Inv-4** (BC-2.04.006, counter scope): two counters required (`SessionConnector.RelayDropped()` at relay layer, `AccessNode.FramesDropped()` at ConsoleSet layer); log format `"frames_dropped relay=<N> consoles=<M>"`. Consistent with S-W3.04 AC-006 and access.go (verified on origin/develop). ✓  
- **Inv-5** (BC-2.04.007, Err() drain obligation): drain goroutine must be wg-tracked. Consistent with S-W3.04 AC-007 and access.go on origin/develop. ✓  
- **PC-2.6** (BC-2.04.007): mid-session double-failure path identical to SIGTERM shutdown but exit 1. Consistent across BC-2.04.002 and S-W3.04. ✓  

---

## Dimension 5: Coverage Gaps

### F-5.1 — HIGH: BC-2.04.007 status in BC-INDEX is `active` (unimplemented), but implementation is merged on origin/develop

**Severity:** HIGH  
**Spec side:** `.factory/specs/behavioral-contracts/BC-INDEX.md` line 52: BC-2.04.007 status = `active`.  
**Reality:** STATE.md records S-W3.04 (which implements BC-2.04.007 AC-007/AC-008) as `completed (merge aeb442d)`. `origin/develop` contains `cmd/switchboard/access.go` with `runAccessWithConnector` and the PC-1/PC-2/PC-2.6 lifecycle paths.  
**VP-060 coverage:** VP-060 is correctly registered in VP-INDEX.md (line 86) and ARCH-11 (line 57) for BC-2.04.007. VP-060 file exists at `.factory/specs/verification-properties/VP-060.md`. VP coverage is complete.  
**Finding:** The BC-INDEX `active` status creates a false impression that BC-2.04.007 has no implementation. The VP and story coverage is correctly registered, but the BC-INDEX status field lags reality. This is a coverage-tracking gap, not a true coverage gap.  
**Canonical source of truth:** STATE.md `s_w3_04_status: completed`.  
**Recommended fix:** Update BC-INDEX.md line 52: change BC-2.04.007 status from `active` to `implemented (S-W3.04 / PR #17, merge aeb442d)`.  
**Owner:** state-manager (BC-INDEX post-merge status update; same change as F-3.2).

---

## Verified Correct: No Findings

The following items were explicitly checked and found consistent:

| Item | Check | Result |
|------|-------|--------|
| `ErrPTYSourceEOF` sentinel message | `connector_frames.go` (origin/develop): `errors.New("session connector: PTY source EOF")` matches BC-2.04.007 Error Codes / error-taxonomy E-SYS-003 canonical sentinel `"session connector: PTY source EOF"` | ✓ MATCH |
| E-SYS-002 message format | `access.go` (origin/develop) lines 147/155/186: `fmt.Sprintf("fatal: cannot connect to session backend: %v", err)` matches error-taxonomy E-SYS-002 format `"fatal: cannot connect to session backend: <reason>"` | ✓ MATCH |
| `frames_dropped` log format | `access.go` (origin/develop) line 368: `lg.Printf("frames_dropped relay=%d consoles=%d", …)` matches BC-2.04.006 Inv-4 and S-W3.04 AC-006 format `"frames_dropped relay=<N> consoles=<M>"` | ✓ MATCH |
| E-ADM-017 message format | `failure_counter.go` line 181: `"E-ADM-017 HMAC failure rate alert: ≥%d failures in %.0fs from src %s"` matches error-taxonomy E-ADM-017 canonical format `"E-ADM-017 HMAC failure rate alert: ≥<threshold> failures in <window_seconds>s from src <src_addr>"` | ✓ MATCH |
| E-ADM-016 PATH-B message | `routing.go` line 215: `"wire HMAC verification failed at RouteFrame: tag mismatch for SVTN %x from src %x (E-ADM-016)"` matches error-taxonomy E-ADM-016 canonical `"wire HMAC verification failed at RouteFrame: tag mismatch for SVTN <svtn_id> from src <src_addr> (E-ADM-016)"` | ✓ MATCH |
| E-SYS-001 sentinel message | `pty_fallback.go` line 46: `errors.New("PTY device unavailable: cannot start access node")` matches error-taxonomy E-SYS-001 format (terse sentinel; full guidance logged separately per BC-2.04.002 EC-004) | ✓ MATCH |
| `RecordHMACFailure` call sites | `routing.go` lines 204–207 (PATH-A) and 219–222 (PATH-B): called on BOTH `ErrHMACVerificationFailed` paths, NOT on success path. Matches BC-2.05.008 PC-5/Inv-5, S-W3.05 AC-009. | ✓ MATCH |
| `FailureCounter` constructor nil-logger panic | `failure_counter.go` line 81–83: panics with `"admission: NewFailureCounter: logger must not be nil"`. Matches S-W3.05 AC-013 and BC-2.05.005 v1.7/v1.8 constructor contract. | ✓ MATCH |
| Drain-only re-arm logic | `failure_counter.go` lines 148–153: re-arms when `len(keep) == 0` (drain-only); append-skip active when `!lastFire.IsZero()`. Matches BC-2.05.005 v1.8 PC-3 (drain-only, no post-fire appends). | ✓ MATCH |
| BC-2.04.002 `forwardFrames` hot-spin prevention | `connector_frames.go` (origin/develop) lines 163–187: detects `srcCh == prevSrcCh` in PTY mode, sends `ErrPTYSourceEOF` via `sc.closeErrCh.Do`, returns (no busy-spin). Matches BC-2.04.002 EC-008 / BC-2.04.007 EC-007 / S-W3.04 AC-009. | ✓ MATCH |
| VP-059 registration | `VP-INDEX.md` line 85: VP-059 registered for `internal/admission`, proptest, P0. VP-059 file exists. Matches S-W3.05 AC-017. | ✓ MATCH |
| VP-060 registration | `VP-INDEX.md` line 86: VP-060 registered for `cmd/switchboard`, integration, P0. VP-060 file exists. ARCH-11 line 57 covers it. | ✓ MATCH |
| `maxTrackedSources` constant | `failure_counter.go` line 41: `const maxTrackedSources = 65536`. Matches BC-2.05.005 EC-010 and S-W3.05 AC-011. | ✓ MATCH |
| ARCH-01 v1.5 ADR-011 TOCTOU fix implementation | `connector_frames.go` (origin/develop): `activeSourceSnapshot()` reads `{src, srcCh, inPTYMode}` under single `sc.mu` lock. Matches ARCH-01 v1.6 ADR-011 §HIGH-A requirement. | ✓ MATCH (but ARCH-01 frontmatter version stale — see F-3.3) |
| S-W3.04 story BC-2.04.007 trace | Story frontmatter `bc_traces` includes BC-2.04.007 at lines 16–24. AC-007 traces to BC-2.04.007 PC-1 + PC-2.6 + EC-007 + Inv-5; AC-008 traces to BC-2.04.007 PC-2. | ✓ MATCH |
| VP-INDEX arithmetic | 33 proptest + 2 fuzz + 12 integration + 10 e2e + 2 benchmark + 1 code-audit = 60. Matches `Total VPs: 60`. | ✓ CONSISTENT |

---

## Full Findings Register

| # | Severity | Category | Short Description | Owner |
|---|----------|----------|-------------------|-------|
| F-1.1 | CRITICAL | Spec↔Code | S-W3.04 NOT on local develop; origin/develop is 1 commit ahead | state-manager |
| F-1.2 | HIGH | Spec↔Code | E-ADM-016 PATH-A message (`auth key unavailable`) not in taxonomy; taxonomy only canonizes PATH-B (`tag mismatch`) | product-owner / implementer |
| F-1.3 | MEDIUM | Spec↔Code | `ErrUpstreamReadOnly` static message omits `<key_fingerprint>` and `<session_name>` fields from E-ADM-007 taxonomy format | product-owner |
| F-2.1 | HIGH | Story↔BC | STORY-INDEX S-W3.04 row omits BC-2.04.007; status is stale (`draft` vs actual `completed`); BC-count note stale (43 vs 44) | state-manager / story-writer |
| F-2.2 | MEDIUM | Story↔BC | S-W3.05 story body references BC-2.05.005 v1.7; actual BC is v1.8 (spec-hygiene delta) | story-writer |
| F-3.1 | CRITICAL | Index | Local develop missing S-W3.04 merge (duplicate of F-1.1) | state-manager |
| F-3.2 | HIGH | Index | STORY-INDEX and BC-INDEX statuses stale post S-W3.04 and S-W3.05 merge | state-manager |
| F-3.3 | MEDIUM | Index | ARCH-01 frontmatter `version: "1.5"` but body/changelog contains v1.6 content | architect |
| F-3.4 | LOW | Index | ARCH-INDEX context engineering note says 42 BCs; actual count is 44 | architect |
| F-4.1 | LOW | Cross-spec | E-ADM-016 taxonomy only canonizes PATH-B format; BC-2.05.008 PC-4 allows PATH-A without prescribed format | product-owner |
| F-5.1 | HIGH | Coverage | BC-INDEX BC-2.04.007 status `active` contradicts STATE.md `completed`; creates false impression of zero implementation | state-manager |

---

## Blocking Gate Findings

The following findings BLOCK the Wave-3 consistency gate:

1. **F-1.1 / F-3.1 (CRITICAL):** Local develop is missing S-W3.04 merge. All local tooling (tests, build, linters) operates against an incomplete codebase. Gate cannot be run validly without `git pull` to advance local develop to `origin/develop` HEAD.

2. **F-1.2 (HIGH):** E-ADM-016 PATH-A message format not registered in taxonomy. Operator grep-patterns derived from the taxonomy will miss PATH-A events.

3. **F-2.1 / F-3.2 / F-5.1 (HIGH):** Multiple indexes (STORY-INDEX, BC-INDEX) carry stale statuses that misrepresent the completion state of Wave-3 gate blockers F-1 (S-W3.04) and the BC coverage count.

---

## Non-Blocking Observations

- **F-2.2, F-3.3, F-3.4, F-4.1:** All MEDIUM or LOW severity. None block gate. Should be resolved before Wave-4 starts to prevent accumulation of spec drift.

---

## Consistency Score

| Criteria checked | Passing | Failing |
|-----------------|---------|---------|
| 15 explicit checks | 12 | 3 blocking (F-1.1, F-1.2, F-2.1) + 3 non-blocking (F-1.3, F-2.2, F-3.3) |

**Consistency score: 72 / 90 = 80%** (scoring: CRITICAL = -10, HIGH = -5, MEDIUM = -3, LOW = -1; base 90 for scope audited)

**Gate result: FAIL** — 2 CRITICAL-class findings (F-1.1/F-3.1 are the same root cause) and 3 HIGH findings remain unresolved.
