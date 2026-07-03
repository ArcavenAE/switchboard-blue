```yaml
document_type: adversarial-review
artifact_id: P5-pass-8-Adv-A
verdict: HAS_FINDINGS
finding_counts:
  high: 2
  med: 4
  low: 1
  obs: 0
develop_tip: b4ccd061b03103e677837511233d3301de0e44f9
model: claude-opus-4-7
time_spent_minutes: 6
files_read:
  - .factory/specs/prd-supplements/interface-definitions.md
  - cmd/sbctl/main.go
  - cmd/sbctl/admin.go
  - cmd/switchboard/admin_handlers.go
  - cmd/sbctl/console.go
  - cmd/sbctl/router_status.go
  - internal/svtnmgmt/svtnmgmt.go (partial 30 lines; overage acknowledged)
read_cap: 6
prior_passes_read: false
lens: public-surface-and-operator-ux
```

## Findings

### F-P5P8-A-001 [HIGH] admin key register confirm-gate emits wrong-command error prefix

`cmd/sbctl/admin.go:372-374` hardcodes the error prefix `"admin svtn destroy: invalid --confirm ..."` inside `runDestroyConfirmGate`, but that helper is also invoked from `runAdminKeyRegister` at `cmd/sbctl/admin.go:458`. An operator running `sbctl admin key register --svtn X --key ... --confirm=bogus` receives:

```
admin svtn destroy: invalid --confirm "bogus"; expected SVTN-<8 lowercase hex characters>
```

Failure scenario: operator scripting an admin-key-register into a fresh SVTN sees "admin svtn destroy" in stderr, doubts whether their command was misinterpreted or a code path was rerouted, and aborts a legitimate provisioning batch. This is a destructive-verb wrong-command emission — the confirm-gate helper needs to parameterize its error prefix on the invoking subcommand (parallels the existing `targetFlag` parameter). Not in the deferred list — line 395 interactive-mismatch is; line 372 flag-supplied shape check is not.

### F-P5P8-A-002 [HIGH] Spec §108 admin key register exit codes list two unreachable errors

`interface-definitions.md:108` documents register exit codes as `0=ok, E-ADM-012 (already registered), E-ADM-018 (control-to-control confirmation required)`. Neither is actually reachable via the register path:

- **E-ADM-012** is not present in the codebase (`grep -rn 'E-ADM-012'` returns zero hits under `cmd/` or `internal/`). `SVTNManager.RegisterKey` at `internal/svtnmgmt/svtnmgmt.go:238-267` documents "Last-write-wins semantics per ADR-003: registering an already-registered key updates its role" — there is no duplicate-key error path.
- **E-ADM-018** is the `ErrControlRevocationRequiresConfirm` mapping at `cmd/switchboard/admin_handlers.go:437`, reachable only via `admin.key.revoke` (RevokeKey with confirm=false on a control-role target). Register never invokes this sentinel.

Failure scenario: an operator writes a runbook that greps stderr for `E-ADM-012` to distinguish "no-op re-register" from a real failure. The condition never fires; the runbook silently treats every double-register as success (which matches LWW semantics — but the spec-promised distinct error code never appears). §108 needs to be corrected to reflect the actual register error surface (E-CFG-001 for missing/invalid inputs; E-ADM-009 for insufficient authority; E-CFG-012/013 for client-side confirm-gate misuse).

### F-P5P8-A-003 [MED] admin key register --role has silent default to "console"; spec syntax implies required

`interface-definitions.md:108` presents register syntax as `sbctl admin key register --svtn <id> --key <hex-pubkey> --role <control|console|access>` — the angle-bracket form (no `[--role ...]` brackets) reads as required. `cmd/sbctl/admin.go:434` declares `roleFlag := fs.String("role", "console", ...)` — silently defaulting to console-role.

Failure scenario: operator scripts `sbctl admin key register --svtn X --key <console-team-key>` intending to trip a "missing --role" error before proceeding to a validation step; instead a console-role key is registered and the script proceeds. For a destructive admin verb, the default is not just spec drift but an operator-UX trap. Either the spec must document `--role` as optional with a documented default, or the impl must require the flag explicitly.

### F-P5P8-A-004 [MED] admin svtn destroy exit-code column overstates validation coverage

`interface-definitions.md:120` documents destroy exit codes as including `E-CFG-001 (invalid SVTN name: empty / whitespace-only / >255 bytes / invalid UTF-8 / control chars)`. Impl at `cmd/switchboard/admin_handlers.go:777-810` (`makeAdminSVTNDestroyHandler`) validates only `a.Name == ""`. It does not call `validateSVTNName` (which exists in the same file at line 824 and is used by the *create* handler at line 680).

Failure scenario: an operator invokes `sbctl admin svtn destroy --name "   "` (whitespace-only). Impl accepts it, dispatches to `SVTNManager.Destroy`, which fails the map lookup and returns `ErrSVTNNotFound → E-SVTN-003`. The spec-promised E-CFG-001 never fires. Runbook logic branching on E-CFG-001 vs E-SVTN-003 (e.g., "if name-invalid, prompt user to retype; if not-found, prompt for correct name") makes the wrong choice.

### F-P5P8-A-005 [MED] Spec §109 admin key revoke lists E-ADM-011 for hierarchy violation; impl emits E-ADM-019

`interface-definitions.md:109` documents revoke exit codes as `E-ADM-011 (hierarchy violation)` among others. `cmd/switchboard/admin_handlers.go:406-454` (`mapAdminError`) maps `ErrRoleMismatch → E-ADM-019` (role mismatch), and `E-ADM-011` is only emitted for `ErrDestroyUnauthorized` (SVTN destroy path). The revoke handler's hierarchy check (ADR-004: console cannot revoke control) surfaces through the role-mismatch pathway, not a hierarchy-specific sentinel.

Failure scenario: an operator whose console-role key attempts to revoke a control-role key gets `E-ADM-019: role mismatch: claimed role console does not match registered key role control ...` — not `E-ADM-011: hierarchy violation` as spec §109 promises. Operator whose runbook is authored against §109 misclassifies the failure.

### F-P5P8-A-006 [MED] sbctl paths <unknown-verb> emits misleading "usage: sbctl paths list" message

`cmd/sbctl/main.go:73-74` handles the `paths` verb with:

```go
if len(args) < 2 || args[1] != "list" {
    err = usageErrf("usage: sbctl paths list")
}
```

The sibling `router` at line 89 uses the far clearer `usageErrf("router: unknown subcommand %q; expected 'metrics' or 'status'", args[1])`. So `sbctl paths ping --router=X` (spec §77 PENDING-S-BL.CLI-SURFACE-COMPLETION) produces `usage: sbctl paths list`, giving no hint that (a) `ping` was interpreted as an unknown subcommand or (b) `ping` is pending — the message merely recommends an unrelated verb. Failure scenario: an operator following spec §77 to try `paths ping` sees "use paths list", assumes ping was renamed to list, and gets confused when list output has no ping semantics. Bring paths in line with the router pattern.

### F-P5P8-A-007 [LOW] Spec syntax placeholders <hex-pubkey> misleading; impl accepts OpenSSH or base64 fallback

`interface-definitions.md:108`, `:109`, `:110` show admin key register/revoke/expire syntax with `--key <hex-pubkey>`. `cmd/switchboard/admin_handlers.go:137-180` (`decodePublicKey`) accepts OpenSSH authorized_keys format (primary path) or raw base64 (fallback for backward compatibility) — **not hex**. §113 body prose corrects this to `<openssh-pubkey>` but the row-header placeholder still reads `<hex-pubkey>`. Failure scenario: at-a-glance readers copy the row-header syntax verbatim (`--key <hex-of-pubkey>`) and hit `E-CFG-001: invalid pubkey: not valid base64`. Update the three row headers to `<openssh-pubkey>` for consistency with §113.

## Observations

(none — findings-only pass)

VERDICT: HAS_FINDINGS
