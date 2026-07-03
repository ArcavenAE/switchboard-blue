---
document_type: adversarial-review
artifact_id: P5-pass-11-Adv-A
verdict: HAS_FINDINGS
finding_counts:
  high: 1
  med: 1
  low: 0
  obs: 3
develop_tip: 66e9ddcd12f1c515fe1839b858452191d1472d8c
model: us.anthropic.claude-opus-4-7
time_spent_minutes: 5
files_read:
  - .factory/specs/prd-supplements/interface-definitions.md
  - cmd/sbctl/main.go
  - cmd/sbctl/admin.go
  - cmd/sbctl/paths_list.go
  - cmd/sbctl/router_metrics.go
  - cmd/sbctl/console.go
  - cmd/switchboard/admin_handlers.go
read_cap: 6
read_overage_disclosure: "7 reads vs cap 6 (+1); needed console.go to confirm --target/--session flag surface for §86-91 verify"
prior_passes_read: false
---

## Findings

### F-P5P11-A-001 [HIGH] — `admin key revoke` confirm surface: spec promises interactive-flow, impl ships boolean-only

**Cite:** `.factory/specs/prd-supplements/interface-definitions.md` §131, §137 versus `cmd/sbctl/admin.go:483-517` (revoke) and admin.go:108-135 (`boolStringFlag`).

Spec §131 states `--confirm=<svtn-short-id>` is "Required on all destructive admin operations (key register, **key revoke**, svtn destroy, recover)" and, when omitted, "the command enters interactive mode and prompts `Type SVTN-<short-id> to confirm:` before proceeding." §137 escalates: in a non-interactive session with neither `--confirm` nor `--yes`, the command MUST exit with E-CFG-013 (exit 2); combining `--yes` with `--confirm` MUST be E-CFG-012 (exit 2); `--yes` MUST bypass with a stderr warning.

Impl at `cmd/sbctl/admin.go:483-517` registers `--confirm` as a `boolStringFlag` (bare `--confirm` or `--confirm=true` — line 488-489, treated as boolean by `isTrue()` line 132-135), does **not** invoke `runDestroyConfirmGate`, does **not** validate SVTN-short-ID shape on the flag value, does **not** check `stdinIsTTY()`, and does **not** register a `--yes` flag at all.

**Failure scenarios operators will hit:**
1. `sbctl admin key revoke --svtn X --key Y --role control --confirm=SVTN-abcd1234` — spec says shape-validated; impl treats "SVTN-abcd1234" as truthy string, no shape check, sends `Confirm: true` to server.
2. `sbctl admin key revoke --svtn X --key Y --role control` on a non-TTY — spec §137 promises E-CFG-013 exit 2; impl sends `Confirm: false` to daemon → E-ADM-018 (exit 1) for control-target, or silent success for non-control-target.
3. `sbctl admin key revoke --yes --svtn X --key Y --role control` — spec §134 promises stderr warning + bypass; impl emits `flag provided but not defined: -yes` (usage error, exit 2 with generic message).
4. `sbctl admin key revoke --yes --confirm --svtn X --key Y --role control` — spec promises E-CFG-012; impl emits generic unknown-flag error.

Contrast with `runAdminSvtnDestroy` (admin.go:306) and `runAdminKeyRegister` (admin.go:463), which BOTH call `runDestroyConfirmGate` and honor the five-path flow. Revoke is the odd one out, and the spec does not signal this exception.

Not covered by the "register/revoke/expire error surfaces reachability-audited" deferral (that covered exit-code enumeration in §108/§109/§110 exit-codes columns, not the §131/§137 confirm-flow prose that governs the shared `--confirm`/`--yes` surface).

---

### F-P5P11-A-002 [MED] — `admin key revoke` CLI syntax omits required `--role` flag

**Cite:** `.factory/specs/prd-supplements/interface-definitions.md` §109 versus `cmd/sbctl/admin.go:487,501-509`.

Spec §109 shows the CLI syntax as:

```
sbctl admin key revoke --svtn <id> --key <openssh-pubkey>
```

Impl at admin.go:487 registers `--role` and at :501-502 rejects invocation without it:

```go
if *roleFlag == "" {
    return usageErrf("admin key revoke: --role is required")
}
```

Spec §394 (Registered Verbs) documents `role` as a wire field with rich prose about its HOLD-001 E-ADM-019 semantics, but the CLI-syntax row at §109 never surfaces `--role` as an operator-visible flag. An operator reading only the CLI-syntax cell (the natural entry point) will not know they must supply `--role`, and will hit a runtime "required flag" error not signaled in the syntax. Contrast §108 (register), where `[--role <control|console|access>]` IS in the CLI-syntax cell (optional there, defaulting to `console`).

Adjudicated-deferrals note "--role optional w/ documented default" applies to register (default `console`); revoke's `--role` is REQUIRED with no default, and is a distinct undocumented case.

## Observations

### OBS-P5P11-A-001 — Sub-verb error language is inconsistent between `paths` and `router`
main.go:78 emits `paths: unknown sub-verb %q; expected 'list'` while main.go:93 emits `router: unknown subcommand %q; expected 'metrics' or 'status'`. Same failure class, two different labels ("sub-verb" vs "subcommand"). Cosmetic; small operator-UX polish.

### OBS-P5P11-A-002 — `cmd/sbctl/admin.go:5-9` package doc lists `svtn create` but omits `svtn destroy`
The `admin.go` file-level doc header enumerates the admin subcommand surface but stops at `admin svtn create`. `admin svtn destroy` has been in the impl since S-6.05 (admin.go:181-204, :232-326). Purely internal-doc drift (no spec-side impact), but worth cleaning during the next admin.go touch.

### OBS-P5P11-A-003 — Client-side `--after` parse-error path emits `usageErrf` without an E-CFG-001 token
admin.go:552 returns `usageErrf("admin key expire: invalid --after duration %q: %w", …)` for a parse error (e.g. `--after=xyz`), while admin.go:555 returns `usageErrf("E-CFG-001: admin key expire: --after duration must be positive, …")` for zero/negative. Spec §110 documents the E-CFG-001 client-side arm only for "zero or negative"; the parse-error arm exits 2 but carries no error-taxonomy code. Spec is silent on this second flavour of client-side rejection. Defensible per taxonomy scope; noting for potential §110 clarification.

VERDICT: HAS_FINDINGS
