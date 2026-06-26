---
artifact_id: wave-3-post-fix-consistency-report
document_type: consistency-report
level: ops
version: "1.0"
producer: consistency-validator
timestamp: 2026-06-25T00:00:00Z
cycle: cycle-1
wave: 3
traces_to: pre-impl-consistency-report.md
---

# Wave 3 Post-Fix Consistency Report

**Scope:** Re-validation after fix burst (tip `15f944a`). Verifies closure of
prior findings F-W3-H-001/002/003 and F-W3-M-001/002/003/004, spec-reviewer
criticals C-1, C-2, C-3, and H-6, plus regression check on new drift.

**Artifacts read:**
- `.factory/stories/S-3.01a-tmux-control-mode.md` (v1.0)
- `.factory/stories/S-3.01b-pty-proxy-fallback.md` (v1.0)
- `.factory/stories/S-3.02-session-attach-detach-fanout.md` (v1.2)
- `.factory/stories/S-3.03-tier2-auth-readonly.md` (v1.1)
- `.factory/stories/S-3.04-hmac-routeframe-wireup.md` (v1.0)
- `.factory/stories/STORY-INDEX.md` (v1.2)
- `.factory/specs/behavioral-contracts/ss-05/BC-2.05.003.md` (v1.2)
- `.factory/specs/verification-properties/VP-058.md` (v1.1)
- `.factory/specs/verification-properties/VP-INDEX.md`
- `.factory/specs/behavioral-contracts/BC-INDEX.md`
- `.factory/specs/architecture/ARCH-04-admission-security.md` (v1.5)
- `.factory/specs/architecture/ARCH-08-dependency-graph.md` (v1.5)
- `.factory/specs/architecture/ARCH-01-core-services.md` (v1.2)
- `.factory/STATE.md`

---

## Prior-Finding Closure Table

| Finding ID | Severity | Description | Status |
|-----------|---------|-------------|--------|
| F-W3-H-001 | HIGH | S-3.02 AC-003 cited EC-001 instead of EC-002 | RESOLVED |
| F-W3-H-002 | HIGH | S-3.02 AC-007 cited PC-3 instead of Invariant 3 | RESOLVED |
| F-W3-H-003 | HIGH | ADR-010 v1.1 forbade mid-session PTY fallback, conflicting with BC ECs | RESOLVED |
| F-W3-M-001 | MAJOR | BC-2.05.003 VP table had three duplicate VP-012 rows | RESOLVED |
| F-W3-M-002 | MAJOR | ADR-009 `verifyFrameHMAC` signature incorrect | RESOLVED |
| F-W3-M-003 | MAJOR | STATE.md l3_bc_count: 42, l4_vp_count: 57 (stale) | RESOLVED |
| F-W3-M-004 | MAJOR | ARCH-08 Mermaid layer groupings vs. import positions unexplained | RESOLVED |

### Spec-reviewer CRITICAL and HIGH findings

| Finding ID | Severity | Description | Status |
|-----------|---------|-------------|--------|
| C-1 | CRITICAL | VP-058 harness used non-existent `ks.Register(svtnID, nodeAddr)` call | RESOLVED |
| C-2 | CRITICAL | ADR-009 ordering text contradicted itself (HMAC before admitted but description implied merge) | RESOLVED |
| C-3 | CRITICAL | ADR-009 auth key location said `admitted_key_set` not `Router.forwardingTable[svtnID][nodeAddr].FrameAuthKey` | RESOLVED |
| H-6 | HIGH | S-3.03 hidden coupling: Authorizer interface declared in S-3.02 but S-3.03 scope gave no wiring path | RESOLVED |

---

## Detailed Verification

### F-W3-H-001: S-3.02 AC-003 EC trace

**Verification:** S-3.02 v1.2, line 56:
```
### AC-003 (traces to BC-2.04.003 EC-002)
`Session.Attach` returns `E-SES-001` ("session not found: <session_name>") when the named session does not exist.
```
The trace now correctly cites `EC-002` (session not found) rather than `EC-001` (wrong tier). The Spec Patches table (line 179) records the fix at v1.1 with explicit attribution to F-W3-H-001.

**Result: RESOLVED**

---

### F-W3-H-002: S-3.02 AC-007 PC/Inv trace

**Verification:** S-3.02 v1.2, line 72:
```
### AC-007 (traces to BC-2.04.006 Invariant 3)
Keystrokes from multiple full-access consoles attached simultaneously are serialized...
```
The trace now correctly cites `Invariant 3` (keystroke serialization) rather than `PC-3`. Spec Patches records this fix at v1.1 alongside F-W3-H-001.

**Result: RESOLVED**

---

### F-W3-H-003: ADR-010 vs BC EC conflict

**Verification:** ARCH-01 v1.2 §ADR-010 (changelog lines 175-176) shows:
- v1.1: initial decision (initial-connect-only fallback)
- v1.2: "ADR-010: revised fallback semantics to allow mid-session PTY fallback on control-mode loss"

ADR-010 body (ARCH-01 lines 145-154) now reads:
> "PTY fallback is triggered by any control-mode failure: initial `TmuxControlMode.Attach` failure OR mid-session control-mode loss (e.g., tmux server crash, control socket destroyed)."

Rejected alternatives section (line 161) explicitly records: "Initial-connect-only fallback (prior v1.1 decision): rejected... Restricting fallback to initial connect only would leave mid-session control-mode loss unhandled."

S-3.01b Architecture Compliance Rules (line 120) confirm: "Mid-session PTY fallback IS allowed (ADR-010 reverted at commit 1aedebc)."

BC-2.04.001 EC-002 and BC-2.04.002 EC-003 both describe mid-session fallback as expected behavior. ADR-010 now aligns with the BCs.

**Result: RESOLVED**

---

### F-W3-M-001: BC-2.05.003 VP table deduplication

**Verification:** BC-2.05.003 v1.2 Verification Properties section:
```
| VP-012 | SessionAuth rejects unauthorized console key        | proptest     |
| VP-013 | SessionAuth rejects upstream from read-only key     | proptest     |
| VP-035 | Read-only console upstream rejected by access node  | integration  |
```
Three distinct rows, each appearing exactly once. VP-INDEX confirms:
- VP-012 line 38: "SessionAuth rejects unauthorized console key | ... | proptest"
- VP-013 line 39: "SessionAuth rejects upstream from read-only key | ... | proptest"
- VP-035 line 61: "Read-only console: upstream rejected by access node | ... | integration"

Property text in BC-2.05.003 matches VP-INDEX H1 titles. No duplicates.

**Result: RESOLVED**

---

### F-W3-M-002: ADR-009 `verifyFrameHMAC` signature

**Verification:** ARCH-04 v1.5 §ADR-009 (lines 306-309):
```go
func verifyFrameHMAC(hdr frame.OuterHeader, payload []byte, authKey [hmac.KeySize]byte) bool
```
- Return type: `bool` (not `error`) — CORRECT
- `hdr`: passed by value (not pointer) — CORRECT
- `authKey`: `[hmac.KeySize]byte` fixed-size array (not slice) — CORRECT

The ADR text (lines 317-320) explicitly notes the `error` return was considered and rejected: "the function has exactly two outcomes (valid / invalid); `bool` is unambiguous."

**Result: RESOLVED**

---

### F-W3-M-003: STATE.md counts

**Verification:** STATE.md frontmatter lines 41-44:
```yaml
l3_bc_count: 43
l4_vp_count: 58
```
Both fields match the stated post-fix targets. VP-INDEX arithmetic check (line 92): "32 + 2 + 11 + 10 + 2 + 1 = 58. Consistent." VP-INDEX per-phase total (line 101): "Total | 58". BC-INDEX confirms 43 rows of active BCs (42 original + BC-2.05.008 minted Wave 3).

**Result: RESOLVED**

---

### F-W3-M-004: ARCH-08 Mermaid prose note

**Verification:** ARCH-08 v1.5 changelog (line 11): "v1.5 — Add prose note below Mermaid: positions in §6.5/§6.6 are authoritative for import-order layering; Mermaid groupings reflect functional domain."

The prose note (lines 111-122) reads in full:
> "The Mermaid diagram above groups packages into named layers (Layer 0: Foundation, Layer 1: Security, etc.) for visual readability by functional domain. These groupings do NOT represent strict import-order positions. The authoritative topological positions are in §6.5 (packages present on develop) and §6.6 (planned Wave 3+ packages). In particular, `internal/session` is shown in the Mermaid 'Layer 1: Security' group... but its import-order position is 6 (§6.6)... Always consult §6.5/§6.6 for import-ordering decisions; consult the Mermaid only for functional domain context. (Finding F-W3-M-004 from consistency-validator Wave-3 audit.)"

The note addresses the exact confusion vector identified in the prior report.

**Result: RESOLVED**

---

### C-1: VP-058 harness compile correctness

**Verification:** VP-058 v1.1 proof harness uses:
- `ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)` — correct signature (no return value per comment "void per admission.go line 155")
- `admission.GenerateChallenge(routerPriv)` — returns `(challenge, error)`
- `admission.AdmitNode(challenge, resp, nodePub, svtnID, ks)` — package-level function, 5-arg signature
- `bytes.NewReader(seed[:])` for deterministic Ed25519 keypair — satisfies `io.Reader` contract

The VP-058 v1.1 modification record states: "Corrected proof harness skeleton: replaced non-existent `ks.Register(svtnID, nodeAddr)` call with `RegisterKey(svtnID, pubkey, RoleNode)` + `GenerateChallenge` + `AdmitNode` using actual admission.go API. Added `bytes.NewReader` seed for deterministic Ed25519 keypair generation."

The harness does use `admission.RoleAccess` rather than `admission.RoleNode` as mentioned in the modification note, but `RoleAccess` is the correct role for a console/node in this test context (the modification note contains a minor naming inconsistency in its description only — the code is internally consistent and the test logic is correct as written).

**Result: RESOLVED**

---

### C-2 / C-3: ADR-009 contradictions

**Verification:** ARCH-04 v1.5 §ADR-009 ordering section (lines 282-295):

C-2 (HMAC before admitted, single RLock): Lines 283-295 explicitly state:
> "Although the forwarding-table lookup and the admitted-set check share one RLock acquisition (permissible performance optimization), the **sequential check order is strict and non-negotiable**:" followed by the numbered 6-step sequence showing HMAC (steps 3-4) gates routing (step 5).

Line 295: "The HMAC check (steps 3–4) completes and gates steps 5–6. Sharing one lock acquisition does NOT mean the checks execute simultaneously or that admitted-set logic runs before the HMAC result is known."

C-3 (key location): Lines 275-279:
> "`verifyFrameHMAC` receives the per-node frame_auth_key from `Router.forwardingTable[svtnID][srcNodeAddr].FrameAuthKey`. This is an O(1) read under a single RLock on the forwarding table."

Both contradictions resolved. The ADR now unambiguously states: (1) sequential check order is mandatory even under shared lock; (2) key lives in `forwardingTable`, not `admitted_key_set`.

**Result: RESOLVED**

---

### H-6: S-3.03 hidden coupling / Authorizer interface wiring

**Verification:**

S-3.02 v1.2 declares the `Authorizer` interface in two places:
- Task 11 (line 133): "Declare `Authorizer` interface in `internal/session/upstream.go`; upstream-receive path calls `Authorizer.Allow()` before forwarding the frame; default no-op allows all (so AC-001..AC-008 pass without auth wired)"
- File Structure Requirements (line 172): `internal/session/upstream.go` | create | "`Authorizer` interface (`Allow(consoleKey, sessionName, frame) error`); upstream-receive path; default no-op authorizer (allow-all); S-3.03 wires `SessionAuth` as the `Authorizer`"

S-3.03 v1.1 wires the interface:
- Task 7 (line 119): "Wire `SessionAuth` as the `Authorizer` in `internal/session/upstream.go` (created by S-3.02): call `SessionAuth.Allow()` on every upstream frame before forwarding; reject payload-bearing frames from read-only consoles with E-ADM-007"
- File Structure Requirements (line 157): `internal/session/upstream.go` | modify | "Wire `SessionAuth` as the live `Authorizer` in the upstream-receive path (file created by S-3.02; S-3.03 replaces the no-op with `SessionAuth`)"
- S-3.03 also cites `upstream.go` in the Token Budget table (line 104)

The Spec Patches entry (line 164) records the fix: "add `internal/session/upstream.go` modify entry and task 7 for Authorizer wiring — spec-reviewer H-6 + user decision."

The contract is fully explicit: S-3.02 creates `upstream.go` with a no-op `Authorizer`; S-3.03 replaces it with `SessionAuth`. The coupling path from S-3.02 to S-3.03 is documented in both stories.

**Result: RESOLVED**

---

## New Drift Check (Regression from Fix Burst)

### DRIFT-1: ARCH-08 §6.5 matches `ls internal/`

`ls internal/` returns: `admission, frame, halfchannel, hmac, routing` (5 packages).

ARCH-08 §6.5 "Current import positions (post-Wave-2, develop @ `d8d7ae6`)" table lists exactly 5 packages: frame (1), hmac (2), halfchannel (3), admission (4), routing (5). Match is exact.

**Result: CLEAN**

---

### DRIFT-2: ARCH-08 §6.6 lists only session + tmux for Wave 3

ARCH-08 §6.6 lists exactly 2 rows:
- Position 6: `internal/session` (Wave 3, S-3.01/02/03, PLANNED)
- Position 7: `internal/tmux` (Wave 3, S-3.01, PLANNED)

No phantom packages re-introduced. No packages from the previously-corrected hallucinated 16-package table appear in §6.6.

**Result: CLEAN**

---

### DRIFT-3: Wave 3 stories cite only §6.5/§6.6 packages

Checking each story's Architecture Compliance Rules and File Structure Requirements:

- S-3.01a: cites `internal/tmux` (pos 7) and `internal/session` (pos 6) — both in §6.6. CLEAN.
- S-3.01b: cites `internal/tmux` (pos 7) — in §6.6. CLEAN.
- S-3.02: cites `internal/session` (pos 6) — in §6.6. CLEAN.
- S-3.03: cites `internal/session` (pos 6) — in §6.6. CLEAN.
- S-3.04: cites `internal/routing` (pos 5) — in §6.5. CLEAN.

No story references packages outside §6.5 or §6.6.

**Result: CLEAN**

---

### DRIFT-4: STORY-INDEX has all 5 Wave 3 stories with correct point totals

STORY-INDEX master table rows for Wave 3:
| Story | Points | Status |
|-------|--------|--------|
| S-3.01a | 8 | pending |
| S-3.01b | 5 | pending |
| S-3.02 | 8 | pending (v1.2) |
| S-3.03 | 8 | pending (v1.1) |
| S-3.04 | 3 | pending |

Wave summary row (line 67): "S-3.01a, S-3.01b, S-3.02, S-3.03, S-3.04 | 32 | Session access MVP + HMAC wire-up." Arithmetic: 8 + 5 + 8 + 8 + 3 = 32. Correct.

All 5 story files exist in `.factory/stories/`. All have `status: pending`.

**Result: CLEAN**

---

### DRIFT-5: BC-INDEX shows BC-2.05.008 as pending (S-3.04 / Wave 3)

BC-INDEX line 59:
```
| BC-2.05.008 | RouteFrame wire-layer HMAC enforcement (Fail-Closed for Writes) | admission-security | CAP-020 | P0 | E | pending (S-3.04 / Wave 3) | ss-05/BC-2.05.008.md |
```
Status is `pending (S-3.04 / Wave 3)` — not `active`. This is correct: BC status should advance to `active` only when the story implementing it is merged. No drift introduced here.

**Result: CLEAN**

---

### DRIFT-6: Wire-layer HMAC failure error code (E-ADM-016 not E-ADM-002)

Checking all Wave 3 stories for E-ADM-002 references in the HMAC failure path:

S-3.04 AC-002 (line 58): "E-ADM-016 log entry is written"
S-3.04 EC-001 (line 77): "E-ADM-016 logged"
S-3.04 EC-002 (line 78): "E-ADM-016 logged"
S-3.04 task 5 (line 122): "log E-ADM-016"

No story cites `E-ADM-002` for the wire-layer HMAC failure path. E-ADM-016 is used consistently throughout S-3.04.

**Result: CLEAN**

---

### DRIFT-7: depends_on graph resolves correctly

Dependency chain:
- S-3.01a: depends_on [S-1.02, S-2.02, S-2.01] — all Wave 1-2 completed stories. VALID.
- S-3.01b: depends_on [S-3.01a] — correct (PTY fallback extends tmux control mode). VALID.
- S-3.02: depends_on [S-3.01b] — correct (session attach requires tmux/PTY layer). S-3.01a also blocks S-3.02 (line 24 of S-3.01a: `blocks: [S-3.01b, S-3.02]`). The depends_on points to S-3.01b which already depends_on S-3.01a, so the chain is acyclic and transitive. VALID.
- S-3.03: depends_on [S-3.02, S-2.02] — correct (Tier-2 auth builds on session attach + admission). VALID.
- S-3.04: depends_on [S-2.01, S-2.02] — correct (HMAC wire-up independent of session layer). VALID.

No cycles. Topological order: S-3.01a → S-3.01b → S-3.02 → S-3.03 (S-3.04 is independent).

**Result: CLEAN**

---

### DRIFT-8: Spec Patches tables log the version bumps

- S-3.02 Spec Patches: three rows (v1.0 initial, v1.1 EC/Inv trace fixes, v1.2 Authorizer + depends_on update). All changes from the fix burst are recorded. COMPLETE.
- S-3.03 Spec Patches: two rows (v1.0 initial, v1.1 points repoint + upstream.go + task 7). Fix recorded. COMPLETE.
- S-3.01b Spec Patches: one row (v1.0 initial, notes ADR-010 reversion propagated). COMPLETE.
- S-3.01a: no Spec Patches table present (v1.0 only, no patches applied). ACCEPTABLE — no patches were applied to S-3.01a.
- S-3.04: one row (v1.0 initial). ACCEPTABLE — no patches were applied to S-3.04.

**Result: CLEAN**

---

## New Findings

### NEW-M-001: STORY-INDEX summary count stale (MAJOR)

**Artifact:** `.factory/stories/STORY-INDEX.md` frontmatter + Summary section.

**Finding:** The Summary table declares:
```
| Total stories | 23 |
```
However, the master story table contains 24 non-backlog entries (S-3.01 was split into S-3.01a + S-3.01b, adding one story). The summary also declares `| Pending | 17 |` but Wave 3 adds 5 pending stories; with 6 complete, pending should be 18. The Wave Summary note at line 73 acknowledges the split: "Wave 3 total: 5 stories, 32 pts. Total points including Wave 0: 141." The note also states "Total points including Wave 0: 141" but the per-wave total in the table sums to 140 (13 + 18 + 32 + 29 + 21 + 29 = 142 wave points, or 1 + 13 + 18 + 32 + 29 + 21 + 29 = 143 including Wave 0). There is count drift throughout the Summary section.

**Specifics:**
- `Total stories: 23` should be `24` (S-3.01 split into S-3.01a + S-3.01b)
- `Pending: 17` should be `18` (24 total - 6 complete = 18 pending)
- Wave 0 (1 pt) + Wave 1 (13 pt) + Wave 2 (18 pt) + Wave 3 (32 pt) + Wave 4 (29 pt) + Wave 5 (21 pt) + Wave 6 (29 pt) = 143 pts, not 140 or 141. The discrepancy is likely because Wave 1 is 13 pts in the summary row (includes refactor PR) but story-file points for S-1.01 (5) + S-1.02 (8) = 13, and the refactor is not a story file — so wave story points are 13. Wave sum: 1 + 13 + 18 + 32 + 29 + 21 + 29 = 143. The `Total points: 140` figure appears to be pre-split (before S-3.01 5→8+5 repoint and S-3.03 5→8 repoint). After the fix burst: 140 + 3 (S-3.01a split adds 3: 5→8+5 = net +8 pts vs original 5-pt story) = actually original S-3.01 was 8 pts (per story-writer task notes); S-3.03 was 5 pts, now 8 pts (+3); S-3.04 is 3 pts (newly minted). Pre-fix: 132 (Phase 2 gate) + 8 (S-3.04 new) = 140. Post-fix: +3 (S-3.03 repoint) = 143 pts total, not 140.

**Severity:** MAJOR — implementer who reads summary metrics will see wrong counts. Does not block individual story delivery but could mislead wave planning tooling.

**Remediation:** Update STORY-INDEX Summary section: `Total stories: 24`, `Pending: 18`, `Total points: 143`. Update Wave 3 summary note to remove the contradictory "Total points including Wave 0: 141" statement and replace with "Total story points: 143 (includes S-3.03 repoint 5→8 and S-3.04 mint at 3 pts)."

---

### NEW-O-001: S-3.01a blocks S-3.02 but S-3.02 depends_on only S-3.01b (OBSERVATION)

**Finding:** S-3.01a declares `blocks: [S-3.01b, S-3.02]`. S-3.02 declares `depends_on: [S-3.01b]`. This is logically consistent (S-3.01b depends_on S-3.01a, so S-3.02 transitively requires S-3.01a), but the direct link from S-3.01a to S-3.02 in the blocks field creates a minor asymmetry: S-3.02's depends_on doesn't list S-3.01a, only S-3.01b. The dependency is satisfied transitively, and the dependency-graph.md may reflect this, but the direct S-3.01a→S-3.02 blocks link is not mirrored in S-3.02's depends_on.

**Severity:** OBSERVATION — no blocking risk; transitive resolution is correct.

---

### NEW-O-002: VP-058 harness uses `admission.RoleAccess` but modification note says `RoleNode` (OBSERVATION)

**Finding:** VP-058 v1.1 harness calls `ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)`. The modification note in the frontmatter states "RegisterKey(svtnID, pubkey, RoleNode)". The code itself uses `RoleAccess`. For the purposes of this VP (testing HMAC enforcement ordering, not role semantics), either role constant would produce equivalent behavior. The discrepancy is between the prose description and the code — the code is self-consistent and correct for the test intent.

**Severity:** OBSERVATION — modification note is imprecise but the harness code is correct.

---

## Summary Table

| Check Category | Result | Findings |
|---------------|--------|---------|
| Prior findings closed (7 total) | PASS | All 7 RESOLVED |
| Spec-reviewer criticals/highs (4 total) | PASS | All 4 RESOLVED |
| §6.5 matches actual `internal/` packages | PASS | CLEAN |
| §6.6 Wave 3 package declarations | PASS | CLEAN |
| Story package citations vs §6.5/§6.6 | PASS | CLEAN |
| STORY-INDEX Wave 3 story presence (5 stories, 32 pts) | PASS | CLEAN |
| BC-INDEX BC-2.05.008 pending status | PASS | CLEAN |
| E-ADM-016 for HMAC failures (not E-ADM-002) | PASS | CLEAN |
| depends_on graph acyclicity | PASS | CLEAN |
| Spec Patches completeness | PASS | CLEAN |
| STORY-INDEX summary counts | FAIL | NEW-M-001 (summary stale: 23 vs 24 stories, 140 vs 143 pts) |

---

## Consistency Score

- Criteria checked: all relevant per-finding + new drift checks
- Blocking (CRITICAL) violations: 0
- Major violations: 1 (NEW-M-001, summary metrics only — does not affect story deliverability)
- Observations: 2 (non-blocking)

**Score: 97/100** (1 MAJOR finding on non-critical summary metadata; all spec/story/BC/VP/ADR content clean)

---

## Wave Gate Readiness

### S-3.04 as first worktree target

S-3.04 satisfies all pre-launch checklist items:
- Status: `pending` — PASS
- Frontmatter complete: artifact_id, document_type, level, version, producer, timestamp, cycle, depends_on, bc_traces, vp_traces — PASS
- Depends only on completed stories (S-2.01 PR #5, S-2.02 PR #6) — PASS
- File structure references only `internal/routing/routing.go` (exists on develop) — PASS
- No phantom packages — PASS
- BC-2.05.008 and VP-058 both exist and are internally consistent — PASS
- ADR-009 (the architectural decision governing the implementation) is correct and unambiguous — PASS
- 5 ACs fully traced to BC-2.05.008 postconditions/invariants — PASS
- WAVE-3-DEP-001 resolution: S-3.04 is specifically the story that resolves the open drift item — PASS

S-3.04 is the smallest (3 pts), fully self-contained story in Wave 3. It touches only `internal/routing` (already exists), introduces no new packages, and its VP-058 harness skeleton is ready to port into the test file.

**Wave 3 is GREEN to launch S-3.04 as the first implementation worktree.**

The MAJOR finding (NEW-M-001, STORY-INDEX summary counts) does not block S-3.04 delivery or any other Wave 3 story delivery. It should be corrected by the state-manager before the Wave 3 gate report is written but is not a pre-launch blocker.

---

## Verdict

**PASS_WITH_OBSERVATIONS**

All prior-report findings (F-W3-H-001, F-W3-H-002, F-W3-H-003, F-W3-M-001, F-W3-M-002, F-W3-M-003, F-W3-M-004) and spec-reviewer findings (C-1, C-2, C-3, H-6) are RESOLVED. No new CRITICAL findings. One new MAJOR finding (NEW-M-001) on STORY-INDEX summary metadata only — does not block story delivery. Two non-blocking observations (NEW-O-001, NEW-O-002).

Wave 3 is clear to proceed. Recommended first worktree: S-3.04 (HMAC wire-up, 3 pts, no new packages, all dependencies satisfied).

---

## Action Items

| Priority | Owner | Action |
|---------|-------|--------|
| Pre-gate | state-manager | Update STORY-INDEX Summary: `Total stories: 24`, `Pending: 18`, `Total points: 143`; remove contradictory Wave 0 total note |
| Pre-gate | state-manager | Update STATE.md `phase_2_stories: 21` → `22` to reflect the S-3.01 split (if that field is meant to count story files; current value of 21 predates the split) |
| Observation | story-writer | S-3.01a `blocks` field lists S-3.02 — consider whether S-3.02 `depends_on` should explicitly list S-3.01a as well as S-3.01b for clarity |
| Observation | product-owner | VP-058 v1.1 modification note says "RoleNode" but harness code uses `admission.RoleAccess` — minor cleanup to align description with code |
