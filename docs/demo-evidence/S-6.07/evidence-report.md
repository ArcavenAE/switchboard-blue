---
document_type: demo-evidence-report
product: "Switchboard — sbctl admin svtn create"
story_id: S-6.07
story_version: "1.13"
pipeline_run: "2026-07-01"
demo_type: cli
recording_tool: vhs
status: recorded
---

# Demo Evidence Report — S-6.07

## Product: Switchboard — `sbctl admin svtn create` handler + CLI subcommand
## Story: S-6.07 v1.13
## Pipeline Run: 2026-07-01
## Convergence: 3/3 clean fresh 3-lens passes (BC-5.39.001)

---

## Per-AC Demo Recordings

| AC | Description | Tape | GIF | WebM | Paths Covered | Status |
|----|-------------|------|-----|------|---------------|--------|
| AC-001 | `BuildAdminHandlers` registers `"admin.svtn.create"` (control-mode only, 5 total) | [AC-001-handler-registration.tape](AC-001-handler-registration.tape) | [.gif](AC-001-handler-registration.gif) | [.webm](AC-001-handler-registration.webm) | success | recorded |
| AC-002 | `sbctl admin svtn create --name` wire envelope + `svtn_id`/`bootstrap_fingerprint` output | [AC-002-cli-wire-envelope.tape](AC-002-cli-wire-envelope.tape) | [.gif](AC-002-cli-wire-envelope.gif) | [.webm](AC-002-cli-wire-envelope.webm) | success + missing-flag error | recorded |
| AC-003 | Bootstrap-only authority: non-bootstrap key → `E-ADM-009`; genesis carve-out; cross-SVTN denial; mutation test | [AC-003-bootstrap-only-authority.tape](AC-003-bootstrap-only-authority.tape) | [.gif](AC-003-bootstrap-only-authority.gif) | [.webm](AC-003-bootstrap-only-authority.webm) | error (authority) + mutation | recorded |
| AC-004 | `adminSVTNCreateResult` wire shape: `svtn_id` hex, `bootstrap_fingerprint` SHA256:base64 verbatim | [AC-004-wire-shape.tape](AC-004-wire-shape.tape) | [.gif](AC-004-wire-shape.gif) | [.webm](AC-004-wire-shape.webm) | success + field-name assertions | recorded |
| AC-005 | Duplicate name → `E-SVTN-001` stutter-free; non-duplicate `Create()` failure → `E-INT-001` | [AC-005-duplicate-name-stutter-free.tape](AC-005-duplicate-name-stutter-free.tape) | [.gif](AC-005-duplicate-name-stutter-free.gif) | [.webm](AC-005-duplicate-name-stutter-free.webm) | error (duplicate) + error (non-duplicate) | recorded |
| AC-006 | Exhaustive args validation: `E-CFG-001` for control chars, U+2028/U+2029, whitespace-only, >255 bytes | [AC-006-args-validation.tape](AC-006-args-validation.tape) | [.gif](AC-006-args-validation.gif) | [.webm](AC-006-args-validation.webm) | error (validation matrix) | recorded |
| AC-007 | `dispatch()` wraps `io.ErrUnexpectedEOF` → `E-RPC-002: message too large` (Ruling-14) | [AC-007-rpc002-oversized-response.tape](AC-007-rpc002-oversized-response.tape) | [.gif](AC-007-rpc002-oversized-response.gif) | [.webm](AC-007-rpc002-oversized-response.webm) | error (transport decode) | recorded |

---

## Coverage Summary

- **Criteria demonstrated:** 7/7
- **Success paths recorded:** AC-001, AC-002, AC-004
- **Error paths recorded:** AC-002 (missing flag), AC-003 (authority), AC-005 (duplicate + non-duplicate), AC-006 (validation matrix), AC-007 (transport decode)
- **Mutation test recorded:** AC-003 (`TestAdminSVTNCreate_MutationTest_RoleControlCheckMustFireIndependently`)

---

## Wire Envelope Contract Evidence

Tests confirm the two-level envelope scheme (Ruling-11 / Ruling-12):

| Error Origin | Envelope `code` | `message` prefix | Test |
|-------------|-----------------|------------------|------|
| Authority failure (handler) | `E-RPC-011` | `E-ADM-009: ...` | `TestAdminSVTNCreate_NonBootstrapControlKey_RejectsWithEADM009` |
| Duplicate name (handler) | `E-RPC-011` | `E-SVTN-001: SVTN already exists: <name>` | `TestAdminSVTNCreate_DuplicateName_E_SVTN_001` |
| Args validation (handler) | `E-RPC-011` | `E-CFG-001: ...` | `TestAdminSVTNCreate_ArgsValidation_E_CFG_001_Exhaustive` |
| crypto/rand failure (handler) | `E-RPC-011` | `E-INT-001: ...` | `TestAdminSVTNCreate_CryptoRandFailure_E_INT_001` |
| Transport decode (mgmt layer) | `E-RPC-002` | (no prefix — transport code is authoritative) | `TestSbctlAdmin_OversizedRPCResponse_ReturnsE_RPC_002` |

CLI re-wraps daemon envelope under `E-RPC-001` top-level code (Ruling-13); discrimination via message prefix.

---

## Toolchain

| Tool | Version | Status |
|------|---------|--------|
| VHS | 0.11.0 | installed |
| Go | 1.25.4 | installed |
| Menlo | system | installed (used as FontFamily) |

---

## PR Embedding Snippet

```markdown
## Demo Evidence — S-6.07

| AC | Recording |
|----|-----------|
| AC-001 Handler registration | ![AC-001](docs/demo-evidence/S-6.07/AC-001-handler-registration.gif) |
| AC-002 CLI wire envelope | ![AC-002](docs/demo-evidence/S-6.07/AC-002-cli-wire-envelope.gif) |
| AC-003 Bootstrap-only authority | ![AC-003](docs/demo-evidence/S-6.07/AC-003-bootstrap-only-authority.gif) |
| AC-004 Wire shape | ![AC-004](docs/demo-evidence/S-6.07/AC-004-wire-shape.gif) |
| AC-005 Duplicate stutter-free | ![AC-005](docs/demo-evidence/S-6.07/AC-005-duplicate-name-stutter-free.gif) |
| AC-006 Args validation | ![AC-006](docs/demo-evidence/S-6.07/AC-006-args-validation.gif) |
| AC-007 E-RPC-002 oversized | ![AC-007](docs/demo-evidence/S-6.07/AC-007-rpc002-oversized-response.gif) |
```

---

## Notes

- All recordings run the actual Go test suite in the worktree — no mocked outputs.
- VHS `FontFamily "Menlo"` used (Menlo available at `/System/Library/Fonts/SFNSMono.ttf`-family; JetBrains Mono and FiraCode Nerd Font Mono not installed on this host).
- Both `.gif` (PR embed) and `.webm` (archival) produced for every AC.
- Dead symbols `MakeAdminSVTNCreateHandler`, `SVTNCreator`, `SVTNCreateResult`, `adminSVTNCreateArgs`, `adminSVTNCreateResponse` deleted from `internal/mgmt/handlers_admin.go` (F-P1L3-002); live path is `cmd/switchboard/admin_handlers.go`.
