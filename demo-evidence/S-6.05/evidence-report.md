# S-6.05 Demo Evidence Report

**Story:** S-6.05 v1.11 — SVTN Destroy Lifecycle: SVTNManager.Destroy + sbctl admin svtn destroy
**Module:** `internal/svtnmgmt`, `cmd/sbctl`
**HEAD (factory-artifacts convergence commit):** 26ce8f0
**impl\_tip:** d0b4923 (stacked: ba735c9 → 3f54b3e → d0b4923)
**Status:** CONVERGED under BC-5.39.001 (Pass-7 adversarial: 3/3 clean, F-P7L1-MED-1 CLOSED)
**Recorded:** 2026-07-02

---

## Coverage Summary

| Recording | AC | BC/VP | Tests Demonstrated | Paths |
|-----------|-----|-------|-------------------|-------|
| AC-001-destroy-text | AC-003 (text mode) | BC-2.07.001 PC-3 | TestSbctlAdmin_SVTNDestroy_HappyPath, TestSbctlAdmin_SVTNDestroy_NotFound | success (stdout "destroyed SVTN: \<name\>") + E-SVTN-003 (not-found, non-zero exit) |
| AC-002-destroy-json | AC-003 (JSON mode) | BC-2.07.001 PC-3; interface-definitions.md §162/§164 | TestSbctlAdmin_SVTNDestroy_HappyPath_JSON | single-envelope integrity (F-P7L1-MED-1 closure) + no trailing plain-text |
| AC-003-confirm-gate | AC-003 confirm gate | BC-2.07.001 PC-3; interface-definitions.md §125/§127/§129; ADR-004 | TestSbctlAdmin_SVTNDestroy_ConfirmGate (7/7 paths) | P1-valid, P1-invalid, P2-TTY-match, P2-TTY-mismatch, P3-non-TTY, P4-yes, P5-conflict |

---

## Recordings

### AC-001-destroy-text: Text Mode Destroy

- `AC-001-destroy-text.gif`
- `AC-001-destroy-text.webm`
- `AC-001-destroy-text.tape`
- `AC-001-destroy-text.txt`

Traces to **BC-2.07.001 postcondition 3** and story AC-003. Demonstrates
`sbctl admin svtn destroy --name <name> --confirm SVTN-<8hex>` in text mode:

1. **Happy path** — `runAdminSvtnDestroy` dispatches `admin.svtn.destroy` RPC
   to the fake server; stdout contains `"destroyed SVTN: destroy-happy-svtn"`.
   Wire format verified: JSON command field = `"admin.svtn.destroy"`, `args.name`
   field matches `--name` argument (ADR-006, ADR-012).
2. **E-SVTN-003 error path** — fake server returns `"SVTN not found: ghost-svtn"`;
   CLI exits non-zero; error surfaces `E-SVTN-003` prefix.

**Reproduction:**
```
cd .worktrees/S-6.05
go test -count=1 -v -run TestSbctlAdmin_SVTNDestroy_HappyPath ./cmd/sbctl/...
go test -count=1 -v -run TestSbctlAdmin_SVTNDestroy_NotFound ./cmd/sbctl/...
```
Or replay the tape:
```
cd .factory/demo-evidence/S-6.05 && vhs AC-001-destroy-text.tape
```

---

### AC-002-destroy-json: JSON Envelope Integrity (F-P7L1-MED-1 Closure)

- `AC-002-destroy-json.gif`
- `AC-002-destroy-json.webm`
- `AC-002-destroy-json.tape`
- `AC-002-destroy-json.txt`

Traces to **BC-2.07.001 postcondition 3** and **interface-definitions.md §162/§164**
(universal `--json` envelope contract). Demonstrates **F-P7L1-MED-1 closure** at
impl\_tip `d0b4923`:

Prior to `d0b4923`, `runAdminSvtnDestroy` unconditionally printed
`"destroyed SVTN: <name>"` after `connectAndRun` returned — in `--json` mode this
appended a plain-text line trailing the canonical envelope, violating §164. All peer
commands (`svtn create`, `key register`, `key revoke`, `key expire`) use `return
connectAndRun(...)` directly with no post-print; `svtn destroy` was the sole outlier.
The fix wraps the `Fprintf` in `if !useJSON { ... }`.

1. **Single-envelope integrity** — `json.Decoder.Decode` on stdout returns one
   valid envelope (`{"ok":true,"error":null,"data":{"status":"destroyed"}}`).
2. **No trailing plain-text** — second `json.Decoder.Decode` call returns `io.EOF`
   (not a parse error), confirming exactly one envelope on stdout.

**Reproduction:**
```
cd .worktrees/S-6.05
go test -count=1 -v -run TestSbctlAdmin_SVTNDestroy_HappyPath_JSON ./cmd/sbctl/...
```
Or replay the tape:
```
cd .factory/demo-evidence/S-6.05 && vhs AC-002-destroy-json.tape
```

---

### AC-003-confirm-gate: Five-Path Confirm Gate

- `AC-003-confirm-gate.gif`
- `AC-003-confirm-gate.webm`
- `AC-003-confirm-gate.tape`
- `AC-003-confirm-gate.txt`

Traces to **BC-2.07.001 postcondition 3**, **interface-definitions.md §125/§127/§129**,
and **ADR-004** (destructive-op guard). Demonstrates all five confirm-gate paths
(7 subtests):

| Subtest | Path | Observable |
|---------|------|------------|
| `path1_valid_confirm_flag` | Path 1 — valid `--confirm SVTN-<8hex>` | static shape check passes; RPC dispatched |
| `path1_invalid_confirm_flag` | Path 1 — invalid `--confirm` value | error "invalid --confirm"; no RPC |
| `path2_tty_matching_input` | Path 2 — TTY, user types correct short-ID | interactive prompt; match; RPC dispatched |
| `path2_tty_invalid_shape_input` | Path 2 — TTY, user types invalid input | error "interactive confirmation failed"; no RPC |
| `path3_non_tty_no_confirm` | Path 3 — non-TTY, `--confirm` omitted | error "non-interactive session"; no RPC |
| `path4_yes_alone_bypasses` | Path 4 — `--yes` alone | RPC dispatched; stderr "WARNING: --yes bypasses confirmation" |
| `path5_yes_plus_confirm_is_usage_error` | Path 5 — `--yes` + `--confirm` combined | error "E-CFG-006"; exit 2; no RPC |

`stdinIsTTY` and `stdinReader` package-level seams are swapped per subtest
(impl commit `ba735c9`, reconciled at v1.9).

**Reproduction:**
```
cd .worktrees/S-6.05
go test -count=1 -v -run TestSbctlAdmin_SVTNDestroy_ConfirmGate ./cmd/sbctl/...
```
Or replay the tape:
```
cd .factory/demo-evidence/S-6.05 && vhs AC-003-confirm-gate.tape
```

---

## Production Method

**Harness:** All recordings use `go test -count=1 -v -run <test> ./cmd/sbctl/...`
in the story worktree (`.worktrees/S-6.05/`). The test harness (`startFakeServer`
in `cmd/sbctl/admin_test.go:401`) spins a per-test in-process fake server that speaks
the ADR-012 challenge-response protocol over a Unix socket (TCP loopback for console
tests, Unix socket for admin tests). No running SVTN daemon is required — the fake
server exercises the full management-plane transport path (connect, authenticate,
dispatch) while controlling server-side responses.

**Environment:**
- Go 1.25.4 (per `go.mod`)
- Test key: `cmd/sbctl/testdata/test_ed25519_key` (Ed25519, test fixture only)
- VHS 0.11.0, FontFamily Menlo, 1200x600, Catppuccin Mocha theme

**Worktree:** `.worktrees/S-6.05/` on branch `feat/S-6.05-svtn-destroy`

---

## BC/VP Traceability

| Behavioral Contract | Postcondition/Invariant | AC | Recording | Status |
|---------------------|------------------------|----|-----------|--------|
| BC-2.07.001 v1.12 | PC-3 (Destroy) — RPC dispatch, stdout confirmation | AC-003 (text) | AC-001-destroy-text | PASS |
| BC-2.07.001 v1.12 | PC-3 / EC-001 (E-SVTN-003 not found) | AC-003 (text) | AC-001-destroy-text | PASS |
| BC-2.07.001 v1.12 | PC-3 — `--json` single canonical envelope (§164) | AC-003 (JSON) | AC-002-destroy-json | PASS |
| BC-2.07.001 v1.12 | PC-3 — confirm gate P1 (flag supplied, valid) | AC-003 (confirm) | AC-003-confirm-gate | PASS |
| BC-2.07.001 v1.12 | PC-3 — confirm gate P1 (flag supplied, invalid) | AC-003 (confirm) | AC-003-confirm-gate | PASS |
| BC-2.07.001 v1.12 | PC-3 — confirm gate P2 (TTY interactive, match) | AC-003 (confirm) | AC-003-confirm-gate | PASS |
| BC-2.07.001 v1.12 | PC-3 — confirm gate P2 (TTY interactive, mismatch) | AC-003 (confirm) | AC-003-confirm-gate | PASS |
| BC-2.07.001 v1.12 | PC-3 — confirm gate P3 (non-TTY guard) | AC-003 (confirm) | AC-003-confirm-gate | PASS |
| BC-2.07.001 v1.12 | PC-3 — confirm gate P4 (`--yes` bypass) | AC-003 (confirm) | AC-003-confirm-gate | PASS |
| BC-2.07.001 v1.12 | PC-3 — confirm gate P5 (`--yes`+`--confirm` E-CFG-006) | AC-003 (confirm) | AC-003-confirm-gate | PASS |
| VP-048 v1.9 | Property 2 (destroy → SVTN removed + keys purged) | AC-001/AC-002 | AC-001-destroy-text | PASS (svtnmgmt unit tests) |
| VP-048 v1.9 | Property 3 (non-control key cannot destroy) | AC-004 (RPC gate) | — (svtnmgmt + admin_test) | PASS (existing tests) |

**Note on AC-001/AC-002 (SVTNManager.Destroy unit tests):** These are the internal
Go API layer (key purge, genesis re-open, concurrency, defense-in-depth). The recordings
cover the CLI/RPC layer (AC-003). The unit tests (`TestSVTNManager_Destroy_*`) run via
`go test ./internal/svtnmgmt/...` and produce clean output at HEAD `d0b4923`; they are
not separately recorded because the demo audience for this story is the CLI surface.

---

## Notes

- **F-P7L1-MED-1 CLOSED:** impl\_tip `d0b4923` closes the sole outstanding Pass-7
  medium finding. The `AC-002-destroy-json` recording provides visual evidence.
  The fix is a one-line guard `if !useJSON { fmt.Fprintf(sio.out, ...) }` added to
  `runAdminSvtnDestroy` (admin.go:269).
- **Confirm-gate seams:** `stdinIsTTY` and `stdinReader` are package-level vars
  in `cmd/sbctl/admin.go` that allow the interactive TTY prompt to be exercised in
  tests without actually attaching a TTY. This pattern is compatible with the
  `startFakeServer` harness.
- **No daemon required:** All demos are driven by the in-process fake server.
  The management-plane Unix socket is the only real I/O.
- **Spec version:** S-6.05 v1.11 (frontmatter `version: "1.11"`).
  Converged at factory-artifacts commit 26ce8f0.
