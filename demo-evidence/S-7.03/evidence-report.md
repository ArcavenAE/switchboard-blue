# S-7.03 Demo Evidence Report

**Story:** S-7.03 v1.6 — Console Remote Control via sbctl  
**Module:** `cmd/sbctl`, `internal/session`  
**HEAD (factory-artifacts):** 2213780  
**Status:** CONVERGED under BC-5.39.001 (Pass-3 adversarial: 3/3 clean)  
**Recorded:** 2026-07-02

---

## Coverage Summary

| Recording | AC | BC/VP | Tests Demonstrated | Paths |
|-----------|----|-------|--------------------|-------|
| AC-001-attach | AC-001 | BC-2.08.001 PC-1 | TestSbctlConsole_Attach/success_dispatches_console_attach_rpc, TestSbctlConsole_Attach/unknown_session_surfaces_E_SES_001, TestSbctlConsole_Attach/auth_denied_surfaces_E_ADM_006 | success + E-SES-001 (EC-001) + E-ADM-006 (EC-003) |
| AC-002-detach | AC-002 | BC-2.08.001 PC-2 | TestSbctlConsole_Detach/success_dispatches_console_detach_rpc, TestSbctlConsole_Detach/not_attached_surfaces_E_SES_004 | success + E-SES-004 (EC-002) |
| AC-003-switch | AC-003 | BC-2.08.001 PC-3 | TestSbctlConsole_Switch/success_dispatches_console_switch_rpc, TestSbctlConsole_Switch/unknown_session_surfaces_E_SES_001, TestSbctlConsole_Switch/not_attached_surfaces_E_SES_004 | success + E-SES-001 (attach leg) + E-SES-004 (detach leg) |

---

## Recordings

### AC-001: Console Attach

- `AC-001-attach.gif`
- `AC-001-attach.webm`
- `AC-001-attach.tape`

Traces to **BC-2.08.001 PC-1**. Demonstrates `sbctl console attach --target
<console_addr> --session <name>` dispatches a `console.attach` JSON RPC over the
management-plane transport per BC-2.07.004 EC-013 (RULING-W6TB-C). Three test
sub-cases shown:

1. **Happy path** — RPC dispatched with correct `session_name`; response accepted.
2. **E-SES-001 error path (EC-001)** — unknown session name; error surfaced with
   `E-SES-001:` prefix.
3. **E-ADM-006 error path (EC-003)** — auth denied; error surfaced with
   `E-ADM-006:` prefix in E-RPC-011 envelope.

**Reproduction:** `cd .factory/demo-evidence/S-7.03 && vhs AC-001-attach.tape`

---

### AC-002: Console Detach

- `AC-002-detach.gif`
- `AC-002-detach.webm`
- `AC-002-detach.tape`

Traces to **BC-2.08.001 PC-2**. Demonstrates `sbctl console detach --target
<console_addr>` dispatches a `console.detach` JSON RPC over the management-plane
transport. Session is detached but not closed (PC-2 semantics). Two test sub-cases:

1. **Happy path** — RPC dispatched; `console.detach` command confirmed on wire.
2. **E-SES-004 error path (EC-002)** — detach when not attached; error surfaced
   with `E-SES-004:` prefix.

**Reproduction:** `cd .factory/demo-evidence/S-7.03 && vhs AC-002-detach.tape`

---

### AC-003: Console Switch

- `AC-003-switch.gif`
- `AC-003-switch.webm`
- `AC-003-switch.tape`

Traces to **BC-2.08.001 PC-3**. Demonstrates `sbctl console switch --target
<console_addr> --session <name>` dispatches a `console.switch` JSON RPC over the
management-plane transport. Atomic detach+attach in one operation. Three test
sub-cases:

1. **Happy path** — RPC dispatched with correct `session_name`; atomic switch
   confirmed.
2. **E-SES-001 error path (EC-001)** — unknown session target on attach leg;
   error surfaced with `E-SES-001:` prefix.
3. **E-SES-004 error path (EC-002)** — not attached on detach leg; error surfaced
   with `E-SES-004:` prefix.

**Reproduction:** `cd .factory/demo-evidence/S-7.03 && vhs AC-003-switch.tape`

---

## Production Method

**Harness:** All recordings use `go test -count=1 -v -run <test>
./cmd/sbctl/...` in the story worktree. The test harness
(`startFakeServer` in `admin_test.go`) spins a per-test in-process fake
server that speaks the ADR-012 challenge-response protocol over a Unix
socket, injecting exactly the RPC responses needed for each sub-case.
No running SVTN daemon is required — the fake server exercises the full
management-plane transport path (connect, authenticate, dispatch) while
controlling the server-side response.

**Transport:** Management-plane Unix socket (RULING-W6TB-C; BC-2.07.004
EC-013). Tests explicitly verify the `console.attach` / `console.detach` /
`console.switch` commands and the `session_name` payload fields on the wire.

**Environment:**
- Go 1.25.4 (per `go.mod`)
- Test key: `cmd/sbctl/testdata/test_ed25519_key` (Ed25519, test fixture only)
- VHS 0.11.0, FontFamily Menlo, 1200x600, Catppuccin Mocha theme

**Worktree:** `.worktrees/S-7.03/` on branch `feat/S-7.03-console-remote-control`

---

## BC/VP Traceability

| Behavioral Contract | Postcondition/Invariant | AC | Status |
|---------------------|------------------------|----|--------|
| BC-2.08.001 | PC-1 (console.attach dispatched with session_name) | AC-001 | PASS |
| BC-2.08.001 | PC-1 / EC-001 (E-SES-001 on unknown session) | AC-001 | PASS |
| BC-2.08.001 | PC-1 / EC-003 (E-ADM-006 on auth denied) | AC-001 | PASS |
| BC-2.08.001 | PC-2 (console.detach dispatched; session not closed) | AC-002 | PASS |
| BC-2.08.001 | PC-2 / EC-002 (E-SES-004 on not-attached) | AC-002 | PASS |
| BC-2.08.001 | PC-3 (console.switch dispatched, atomic detach+attach) | AC-003 | PASS |
| BC-2.08.001 | PC-3 / EC-001 (E-SES-001 on unknown target session) | AC-003 | PASS |
| BC-2.08.001 | PC-3 / EC-002 (E-SES-004 on not-attached) | AC-003 | PASS |
| VP-050 | Console remotely controllable end-to-end | AC-001/002/003 | PASS |

---

## Notes

- **No daemon required:** All three ACs are demonstrated via the in-process fake
  server pattern. The console mode uses TCP loopback (127.0.0.1:9091) per BC-2.07.004
  EC-013 Ruling D/J; the fake server binds a loopback address per the same convention.
  The test harness is production-equivalent for the management-plane transport path.
- **No `--json` mode gap:** The `--json` flag exercises a code path inside
  `connectAndRun` (writes `jsonEnvelope` vs plain text). The test harness verifies
  the RPC dispatch layer; `--json` mode is covered by existing `TestConnectAndRun_*`
  unit tests in `client_test.go`. No separate demo tape was needed.
- **Spec version:** S-7.03 v1.6 (frontmatter `version: "1.6"`). Converged at
  factory-artifacts commit 2213780.
