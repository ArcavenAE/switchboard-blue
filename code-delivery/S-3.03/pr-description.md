# feat(S-3.03): tier-2 per-session authorization and read-only enforcement (BC-2.04.005/BC-2.05.003)

## Summary

Implements `SessionAuth` in `internal/session` — the Tier-2 per-session authorization layer
that enforces access control at the granularity of individual sessions. A console's public key
must be registered in the named session's authorization list before `AccessNode.Attach` succeeds.
Read-only consoles can attach and receive downstream frames, but any payload-bearing upstream
keystroke is rejected with `E-ADM-007`; empty-tick (liveness probe) frames are accepted.

Resolves drift item **S-3.02-FM1**: the vestigial no-op `Authorizer` is replaced by `SessionAuth`
wired as the live `Authorizer` in the `AccessNode.Attach` path.

Depends on: #13 (S-3.02 — console attach/detach/fan-out, merged).
Blocks: S-6.02 (SVTN lifecycle + key management).

---

## Architecture Changes

```mermaid
graph TD
    subgraph "S-3.03 additions"
        SA[SessionAuth<br/>internal/session/auth.go]
        RK[RegisterKey<br/>sessionName + consoleKey + Role]
        AZ[Authorize<br/>returns Role or E-ADM-006]
        AL[Allow<br/>checks payload + role → E-ADM-007]
    end
    subgraph "S-3.02 (existing, modified)"
        AN[AccessNode<br/>internal/session/upstream.go]
        CS[ConsoleSet<br/>internal/session/fanout.go]
    end
    SA --> RK
    SA --> AZ
    SA --> AL
    AN -->|Authorizer wired here| SA
    AN --> CS
    CS -->|fan-out frames| C1[Console full-access]
    CS -->|fan-out frames| C2[Console read-only]
    C1 -->|keystrokes forwarded| KS[KeystrokeSink]
    C2 -->|payload keystrokes REJECTED E-ADM-007| KS
    C2 -->|empty-tick ACCEPTED| KS
```

Changes in this PR:
- `internal/session/auth.go` — new: `SessionAuth` type with `RegisterKey`, `Authorize`, `Allow`; per-session authorization map; `Role` constants (`RoleFull`, `RoleReadOnly`); sentinel errors `ErrSessionAuthDenied` (E-ADM-006) and `ErrUpstreamReadOnly` (E-ADM-007)
- `internal/session/upstream.go` — modified: `SessionAuth` wired as live `Authorizer`; `AccessNode.Attach` calls `auth.Allow()` before establishing downstream channel and adding to `ConsoleSet`; S-3.02-FM1 resolved
- `internal/session/auth_test.go` — new: Tier-2 authorization tests (AC-001..AC-006) and coverage tests (M-1/M-2)

No changes to `internal/frame`, `internal/admission`, or `internal/routing`.
Import graph invariant holds: `internal/session` imports only `{admission, frame, stdlib}`.

---

## Story Dependencies

```mermaid
graph LR
    S302["#13 S-3.02<br/>console attach/detach<br/>fan-out (merged)"] --> S303
    S303["S-3.03 this PR<br/>Tier-2 auth +<br/>read-only enforcement"]
    S303 --> S602["S-6.02<br/>SVTN lifecycle +<br/>key management"]
    S302a["#11 S-3.01a<br/>tmux control mode<br/>(merged)"] --> S302
    S302b["#12 S-3.01b<br/>PTY proxy fallback<br/>(merged)"] --> S302
    S202["S-2.02<br/>admission SVTN isolation<br/>(merged)"] --> S303
```

---

## Spec Traceability

```mermaid
flowchart LR
    BC205003["BC-2.05.003<br/>Tier-2 per-session auth"] --> AC001["AC-001<br/>RegisteredKey → success + role"]
    BC205003 --> AC002["AC-002<br/>UnregisteredKey → E-ADM-006"]
    BC205003 --> AC003["AC-003<br/>router has no Tier-2 state"]
    BC205003 --> AC004["AC-004<br/>per-session no spillover"]
    BC204005["BC-2.04.005<br/>read-only enforcement"] --> AC005["AC-005<br/>upstream rejected E-ADM-007<br/>downstream continues"]
    BC204005 --> AC006["AC-006<br/>empty-tick accepted"]

    AC001 --> T001["TestSessionAuth_Authorize_RegisteredKey_Succeeds"]
    AC002 --> T002["TestSessionAuth_Authorize_UnregisteredKey_ErrAdmSix"]
    AC003 --> T003["TestSessionAuth_RouterHasNoTier2State"]
    AC004 --> T004["TestSessionAuth_Authorize_PerSession_NoSpillover"]
    AC005 --> T005["TestReadOnlyConsole_UpstreamRejected_DownstreamContinues"]
    AC006 --> T006["TestReadOnlyConsole_EmptyTickAccepted"]
```

---

## Test Evidence

| Metric | Value |
|--------|-------|
| Tests in `internal/session` | 70 PASS lines (top-level tests + subtests), 0 FAIL, 0 SKIP |
| Race detector | CLEAN (`go test -race -v ./internal/session/`) |
| Adversarial passes | 5 (4 consecutive CONVERGED: pass-02 through pass-05) |
| Critical/High findings | 0 |
| Medium findings resolved | 4 (M-1 × 2: attach-time gate + empty-tick forwarding; M-2 × 2: upstream-channel drain + concurrency coverage) |
| Low findings resolved | 2 (O-1: preposition in E-SES-003; L-2: stale comment) |
| Known DEFERRED | S-3.03-L1-REVOKE (no RevokeKey yet); S-3.03-O1-VPSKEL (VP-013/035 skeleton stale) |

Per-AC test evidence in `.factory/demo-evidence/S-3.03/ac-evidence.md` (test-transcript format).

Mapping of ACs to tests:

| AC | Test(s) |
|----|---------|
| AC-001 | `TestSessionAuth_Authorize_RegisteredKey_Succeeds` (3 subtests: full-access key, read-only key, different session) |
| AC-002 | `TestSessionAuth_Authorize_UnregisteredKey_ErrAdmSix` (3 subtests: empty list, different key, different session only); `TestSessionAuth_ErrorMessages_MatchTaxonomy/E-ADM-006` |
| AC-003 | `TestSessionAuth_RouterHasNoTier2State` (VP-012 code-audit grep) |
| AC-004 | `TestSessionAuth_Authorize_PerSession_NoSpillover`; `TestSessionAuth_CrossSession_Rejected`; `TestSessionAuth_Allow_DecisionMatrix/cross-session` |
| AC-005 | `TestReadOnlyConsole_UpstreamRejected_DownstreamContinues`; `TestReadOnlyConsole_FullAndReadOnly_BothAttached`; `TestSessionAuth_Allow_DecisionMatrix/read-only+payload`; `TestSessionAuth_ErrorMessages_MatchTaxonomy/E-ADM-007` |
| AC-006 | `TestReadOnlyConsole_EmptyTickAccepted`; `TestReadOnlyConsole_EmptyTickForwarded_WithCaptureSink`; `TestSessionAuth_Allow_DecisionMatrix/read-only+empty-tick`; `TestSessionAuth_Allow_DecisionMatrix/read-only+nil-payload` |
| Task 7 | `TestSessionAuth_ImplementsAuthorizer` (compile-time interface guard); `TestAccessNode_Attach_ReadOnlyKey_Succeeds` (M-1 coverage); `TestSessionAuth_ConcurrentRegisterAndAuthorize` (M-2 race coverage) |

---

## Demo Evidence

Test transcripts at `.factory/demo-evidence/S-3.03/ac-evidence.md` (branch tip: 0b9b776).
Recording format: Go test transcripts with race detector enabled. VHS/terminal recordings
intentionally omitted per project standing preference.

At least one passing test per AC (AC-001 through AC-006) plus Task 7 integration tests.
Full suite output included in the evidence file.

---

## Adversarial Convergence (Step 4.5)

Four consecutive CONVERGED passes (streak ≥ 4, criterion was 3). Reports at
`.factory/cycles/cycle-1/S-3.03/adversary/pass-0{1..5}.md`.

| Pass | Critical | High | Medium | Low | Verdict | Notes |
|------|----------|------|--------|-----|---------|-------|
| 01 | 0 | 0 | 2 | 3 | RESET | C-1 attach gate; H-1/H-2 error format |
| 02 | 0 | 0 | 2 | 0 | CONVERGED (streak 1) | M-1/M-2 non-blocking |
| 03 | 0 | 0 | 1 | 1 | CONVERGED (streak 2) | M-2 upstream-channel; O-1 doc |
| 04 | 0 | 0 | 0 | 0 | CONVERGED (streak 3) | All surfaces clean |
| 05 | 0 | 0 | 2 | 1 | CONVERGED (streak 4) | M-1/M-2 coverage gaps (non-blocking); fixed by test-writer |

Pass-01 findings fixed: attach-time auth gate added to `AccessNode.Attach` (before `consoles.Add`);
E-ADM-006/E-ADM-007 error messages interpolate `key` and `session` fields per error-taxonomy v1.6.

Spec clarifications made during convergence (on factory-artifacts branch, NOT this code PR):
- Error-taxonomy v1.6: E-SES-005 retired, layering notes added
- BC-2.04.005 v1.3: EC-003/EC-004/PC-5 clarifications (empty-tick forwarding)

---

## Deferred Drift (tracked in STATE.md, non-blocking)

| ID | Description | Resolution |
|----|-------------|-----------|
| S-3.03-L1-REVOKE | No `RevokeKey` on `SessionAuth` | Deferred to operator-provisioning story (S-6.02 or later) |
| S-3.03-O1-VPSKEL | VP-013/035 verification property skeletons stale | Phase-6 formal hardening |

Neither item is blocking for merge. Both are tracked in `.factory/STATE.md`.

---

## Holdout Evaluation

N/A — evaluated at wave gate.

---

## Adversarial Review

Four consecutive CONVERGED passes (pass-02 through pass-05). Pass-01 reset (C-1/H-1/H-2 fixed).

Zero Critical, zero High at any pass. Enforcement core confirmed across 5 independent fresh-context
adversarial passes. Attack surfaces verified: attach gate ordering, read-only SendKeystroke,
per-session isolation, concurrency/lock ordering, error messages, VP-012 routing audit,
test integrity, ACs 001-006 + Task 7.

---

## Security Review

Architecture is pure-Go in-process boundary layer with no network I/O, no external calls,
no serialization of user-controlled input to unsafe sinks. Relevant properties:

- `SessionAuth.mu` is a `sync.RWMutex`; all map reads take `RLock`, writes take `Lock`
- Lock ordering `sinkMu → ConsoleSet.mu → SessionAuth.mu` never reversed; no simultaneous hold
- `SessionAuth` returns `Role` (int) value copies; no internal pointer leaks (go.md rule-12 compliance)
- `AccessNode.Attach` authorizes before `consoles.Add` — denial leaves no partial state, no resource leak
- Fail-closed: unregistered keys get `ErrSessionAuthDenied`; no allow-by-default path
- No `init()` functions; no global mutable state outside constructed types
- `ErrSessionAuthDenied` and `ErrUpstreamReadOnly` are package-level sentinel vars, properly wrapped with `%w`

No OWASP Top-10 attack surface in this diff (no HTTP, no SQL, no template rendering,
no deserialization of untrusted input). No secrets, credentials, or PII in diff.

---

## Risk Assessment

| Dimension | Classification | Notes |
|-----------|---------------|-------|
| Blast radius | LOW | New file `auth.go`; modification to `upstream.go` is additive (wires existing Authorizer interface) |
| Performance | LOW | `SessionAuth` map lookups under RLock; no goroutine allocation |
| Concurrency | LOW | `-race` clean; lock ordering audited (pass-05); `TestSessionAuth_ConcurrentRegisterAndAuthorize` explicit |
| Rollback | SAFE | `SessionAuth` is a new type; `upstream.go` change replaces `NoOpAuthorizer` — callers must now provide auth (expected) |

---

## AI Pipeline Metadata

| Field | Value |
|-------|-------|
| Pipeline mode | greenfield — cycle v1.0.0 |
| Story phase | Phase 3 TDD + Phase 5 adversarial (5 passes, 4 consecutive CONVERGED) |
| Adversarial model | Distinct from implementer (cognitive diversity) |
| Worktree | `/Users/skippy/work/switchboard-blue/.worktrees/S-3.03` |

---

## Pre-Merge Checklist

- [x] PR description complete with traceability
- [x] Demo evidence present (6 ACs + Task 7, all pass, race-clean)
- [x] Adversarial review converged (4 consecutive clean passes; streak ≥ 3 criterion exceeded)
- [x] Security review: no CRITICAL/HIGH findings
- [x] `just fmt` passes
- [x] `just lint` passes with zero warnings
- [x] `go test -race ./internal/session/` passes (70 PASS lines)
- [x] Dependency PR #13 (S-3.02) merged
- [x] Drift item S-3.02-FM1 resolved (SessionAuth wired as live Authorizer)
- [x] Deferred items S-3.03-L1-REVOKE and S-3.03-O1-VPSKEL documented and non-blocking
- [x] PR reviewer clean (cycle 1 — APPROVE, zero BLOCKING findings)
- [x] CI checks green (CodeQL, Analyze(go), Quality Gate, StepSecurity, dependency-review — all PASS)
- [ ] Human merge approval (pending)
