# PR #61 — S-6.05 SVTN destroy lifecycle — Fresh-eyes Review

**Verdict:** LGTM (approve)

**Head:** `d0b4923e74ab75369f79aee0afb3b0dbe9b65df9` (matches the convergence baseline)
**Base:** `develop`
**Merge state:** CLEAN / MERGEABLE
**CI:** Quality Gate SUCCESS, CodeQL SUCCESS, Dependency Review SUCCESS, StepSecurity Harden-Runner SUCCESS. Alpha release / sign / notarize skipped as expected on PR.

---

## Checklist walk

### 1. Diff coherence — PASS

Every path in the diff maps to S-6.05's scope: SVTNManager.Destroy + destroy handler + `sbctl admin svtn destroy` CLI (including the five-path confirm gate) + the RemoveSVTN primitive it needs + go.mod entry for `golang.org/x/term`. No stray unrelated churn.

Files (+1611/-39, 9 files):
- `internal/svtnmgmt/svtnmgmt.go` — `Destroy(caller, name)` under write lock; `ErrDestroyUnauthorized` sentinel; wraps `ErrSVTNNotFound` with name.
- `internal/admission/admission.go` — `AdmittedKeySet.RemoveSVTN` primitive.
- `cmd/switchboard/admin_handlers.go` — `makeAdminSVTNDestroyHandler`, wired into `BuildAdminHandlers`, `mapAdminError` grew arms for `ErrDestroyUnauthorized` → `E-ADM-011` and `ErrSVTNNotFound` → `E-SVTN-003`; `resolveAndVerifyCallerRole` refactored to return the real `AdmittedKey` so the inner defense-in-depth check sees the actual role (F-P3L1-001 closure).
- `cmd/sbctl/admin.go` — `runAdminSvtnDestroy`, `confirmSVTNShortIDValid`, `runDestroyConfirmGate` (five paths), `stdinIsTTY`/`stdinReader` seams.
- Test files exercising the handler, CLI, and manager surfaces.
- `go.mod` — `golang.org/x/term v0.44.0` promoted to direct (matches the `go mod tidy` commit at `3f54b3e`).

### 2. Description accuracy — PASS

PR body describes exactly what shipped: BC/VP coverage, convergence sidecars, demo evidence layout, per-test coverage, F-P7L1-MED-1 closure at `admin.go:262-276` plus `admin_test.go:1770-1843`. I confirmed each of those anchors on the head SHA:

- `cmd/sbctl/admin.go:272-274` (in the "wrap `Fprintf` under `!useJSON`" block) — present and correctly gated with a cited rationale comment linking F-P7L1-MED-1 and interface-definitions.md:164.
- `cmd/sbctl/admin_test.go:1780+` — `TestSbctlAdmin_SVTNDestroy_HappyPath_JSON` uses the double-`json.Decoder.Decode` idiom; second `.Decode` must return `io.EOF`. Also asserts `env.OK==true`, `env.Error==nil`, `env.Data` contains `"status":"destroyed"`.
- Seven-subtest `TestSbctlAdmin_SVTNDestroy_ConfirmGate` covers Path 1 valid/invalid, Path 2 TTY match/mismatch, Path 3 non-TTY, Path 4 `--yes` (with stderr warning check), Path 5 `--yes + --confirm` → `E-CFG-006`. Docstring is post-retraction (references `ba735c9`) and matches the impl.

### 3. Test coverage — PASS

Changed lines have coverage at three altitudes:
- Manager: `TestSVTNManager_Destroy_ErrDestroyUnauthorized`, `_KeyPurgePostcondition`, `_ConcurrentAdmissionIsRaceFree` (race oracle), `_Idempotent` (EC-001 double-destroy).
- Handler: `TestAdminSVTNDestroy_ArgsValidation_E_CFG_001`, `_ControlRoleSucceeds`, `_NonControlCallerReturnsEADM009`, `_UnknownSVTN`, `_ErrDestroyUnauthorizedMapsToEADM011` (exercises `mapAdminError` directly with `errors.Is` chain preserved).
- CLI: `TestSbctlAdmin_SVTNDestroy_HappyPath` (text, unchanged pre-fix), `_HappyPath_JSON` (F-P7L1-MED-1 regression guard), `_NotFound` (E-SVTN-003), `_ConfirmGate` (5 paths × 7 cases), `_RequiresControlRole` (RPC-path E-ADM-009).

`just test-race` and `just lint` pass per the PR body; CI Quality Gate SUCCESS at head confirms.

### 4. Demo evidence — PASS

`factory-artifacts` branch at commit `238cfd4` carries `demo-evidence/S-6.05/` with per-AC recordings — GIF + WebM + tape + txt trace — for AC-001 (destroy text), AC-002 (destroy JSON single-envelope integrity), AC-003 (confirm gate 5-path). `evidence-report.md` explains the F-P7L1-MED-1 closure and cross-references each recording to its `Test*` function.

### 5. Commit quality — PASS

Twelve conventional commits, all with `feat(S-6.05)`, `fix(S-6.05)`, or `test(S-6.05)` prefixes. Bodies explain the "why" and reference finding IDs, spec sections, and rulings. Story ID present throughout. Signed authorship consistent.

### 6. Diff size — NOTE

+1611/-39 is above the 500-line hint but the split is honest: **1235 of those lines are net-new test coverage** across four test files, and the remainder is the necessary implementation surface for a new admin RPC + CLI subcommand + five-path confirm gate. Not a red flag.

### 7. Missing changes — PASS

Every BC-2.07.001 PC-1..PC-4 element that this story owns is present:
- **PC-3** (Destroy): `SVTNManager.Destroy` removes admitted keys via `RemoveSVTN` then frees the SVTN ID from the registry (ARCH-04 ordering documented in the doc comment).
- **Inv-3** (control-role gate): outer gate at handler (`resolveAndVerifyCallerRole`); inner Go-API guard in `SVTNManager.Destroy` returning `ErrDestroyUnauthorized` before any state is consulted; W6TB-A destroy-authority ruling honored (any control-role key, not bootstrap-only).
- **EC-001** (idempotent second destroy): `TestSVTNManager_Destroy_Idempotent` asserts second call returns `ErrSVTNNotFound`.
- **AC-003 confirm gate (interface-definitions.md §125/§127/§129, ADR-004)**: five paths implemented with the TTY seam. E-CFG-006 for `--yes + --confirm`. §162/§164 JSON envelope contract preserved via `!useJSON` gate (F-P7L1-MED-1).

**Notes on scope for PC-1/PC-2/PC-4:**
- PC-1 (Create) and PC-2 (Bootstrap) are governed by earlier stories (S-6.02, S-6.07); this PR touches them only where necessary (`RemoveSVTN` is the destroy-side primitive for the register they own).
- BC-2.07.001 as documented has three explicit postconditions (Create, Bootstrap, Destroy) — the PR body's "PC-1..PC-4" phrasing is a broad umbrella but every postcondition that S-6.05 owns is covered.
- Session-drain (BC-2.07.001 postcondition 3's "all active sessions terminated" clause) is explicitly deferred to `S-BL.SESSION-DRAIN` per the doc comment on `Destroy`. Test `TestSVTNManager_Destroy_KeyPurgePostcondition` covers the key-purge invariant that this story is scoped to.

### 8. Dependency status — PASS

Depends_on = [S-6.02, S-6.07] per BC-2.07.001 Stories row. Both are in-tree (S-6.02 provides SVTNManager Create + key primitives; S-6.07 provides the admin.svtn.create handler pattern that destroy mirrors). Nothing external is required for merge.

---

## F-P7L1-MED-1 closure verification (per task ask #5)

Fully closed at `d0b4923`. Verified independently on the head SHA:

```
cmd/sbctl/admin.go:272-274
    if !useJSON {
        _, _ = fmt.Fprintf(sio.out, "destroyed SVTN: %s\n", *nameFlag)
    }
```

Peer parity was the diagnosis for the finding — I confirmed all four peer admin commands (`svtn create`, `key register`, `key revoke`, `key expire`) `return connectAndRun(...)` directly with no post-print. `svtn destroy` was the sole outlier; the gate closes the outlier.

Regression guard at `cmd/sbctl/admin_test.go:1780+` uses `json.NewDecoder(&outBuf).Decode(&env)` then asserts a second `.Decode` returns `io.EOF` (idiom that catches any trailing byte, plain-text or otherwise). Envelope shape asserted: `env.OK==true`, `env.Error==nil`, `env.Data` contains `"status"` and `"destroyed"`. Text-mode test unchanged and still passes (`useJSON=false` path preserved).

---

## Non-blocking (per task guidance)

- **F-P8L3-LOW-1** (class-C narrative drift, v1.11 changelog line-number citations): factory task #60 tracks the tidy sweep. Grep-recoverable, no behavioral defect. **Not flagged** per your instructions.

---

## Verdict

**APPROVE / LGTM.** The PR is at the convergence baseline, F-P7L1-MED-1 is genuinely closed with peer-parity restored and a robust regression test, and all 8 checklist items pass. CI is green. No blocking findings.
