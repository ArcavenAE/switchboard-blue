```yaml
document_type: adversarial-review
artifact_id: P5-pass-9-Adv-A
verdict: HAS_FINDINGS
finding_counts:
  high: 1
  med: 2
  low: 3
  obs: 3
develop_tip: 32ea461cd1c50a32e17e42a7f678f701b4dfa04b
model: claude-opus-4-7
time_spent_minutes: 5
files_read:
  - .factory/specs/prd-supplements/interface-definitions.md
  - cmd/sbctl/main.go
  - cmd/sbctl/admin.go
  - cmd/sbctl/console.go
  - cmd/switchboard/admin_handlers.go
read_cap: 6
prior_passes_read: false
```

## Findings

### F-P5P9-A-001 [HIGH] — `sbctl version` and `sbctl ping` are un-annotated spec promises that dispatch to exit-2 unknown-subcommand

**Spec cite:** `interface-definitions.md` §94-95 (v1.20):
```
sbctl version                                   # Print daemon version
sbctl ping                                      # Connectivity check to daemon
```
Neither carries a `PENDING-*` annotation, changelog note, or footnote — the spec presents both as working, implemented diagnostic commands.

**Impl cite:** `cmd/sbctl/main.go:64-102` — the top-level `switch subcommand` has case arms for `sessions`, `paths`, `router`, `console`, `admin` only. Both `version` and `ping` fall through to the `default` arm at line 100-101: `usageErrf("unknown subcommand: %s\nrun 'sbctl' with no args for usage", subcommand)` → `os.Exit(2)`.

**Failure scenario:** An operator following spec §94-95 runs `sbctl version` expecting daemon version output; receives `unknown subcommand: version\nrun 'sbctl' with no args for usage` on stderr and exit 2. The bare-invocation usage line at `main.go:55` (`available subcommands: sessions, paths, router, console, admin`) confirms the omission — but the spec never says these two verbs are unimplemented. This is the same class of spec-impl gap F-P5P6-A-005 resolved for seven other verbs (§60-83) by adding `PENDING-*` annotations; §94-95 was overlooked in that sweep. The pre-adjudicated deferral list names "version/ping" only as backlog wire handlers, which reads as a runtime-side deferral, not a green light for un-annotated spec text.

### F-P5P9-A-002 [MED] — `--target` default (`/run/switchboard-router.sock`) is undocumented in the spec

**Spec cite:** `interface-definitions.md` §48-54 global flags table:
```
--target=<addr>   Daemon address (host:port or unix socket path)
--key=<path>      Path to operator private key file (default: ~/.ssh/id_ed25519)
--json            Machine-readable JSON output
--timeout=<dur>   Connection timeout (default: 5s)
```
`--target` is the only global flag with no documented default. §370 "Flag Interactions" mentions `--target` vs `config daemon.address` precedence but not the fall-through default.

**Impl cite:** `cmd/sbctl/main.go:21`: `const defaultTarget = "/run/switchboard-router.sock"`. `main.go:41`: `target := flag.String("target", defaultTarget, ...)`. The comment at `main.go:20-21` explicitly says "EC-001: when --target is absent and the default socket is absent, E-NET-001 is returned" — the daemon-unreachable E-NET-001 path (exit 1) is triggered by the silent default, invisible to spec readers.

**Failure scenario:** Operator on a machine where no router daemon is running invokes `sbctl sessions list` without `--target`; expects a "missing required flag" usage error (exit 2) based on the spec's silence about a default; instead receives `E-NET-001 daemon unreachable: /run/switchboard-router.sock: connection refused` (exit 1) and cannot understand where the socket path came from. This is doubly a documentation gap and a footgun: `--key` and `--timeout` document their defaults; `--target` alone does not.

### F-P5P9-A-003 [MED] — Spec §110 `admin key expire` exit-code column omits E-ADM-021, E-ADM-009, and E-SVTN-003 — all reachable via the daemon handler

**Spec cite:** `interface-definitions.md` §110 (v1.20), exit-codes cell:
> `0=ok, E-ADM-013 (key not found), E-CFG-001 (invalid after duration: zero, negative, or >100 years)`

**Impl cite:**
- `cmd/switchboard/admin_handlers.go:440-441` — `mapAdminError` returns `E-ADM-021: bootstrap-key-expire-forbidden` when `ExpireKey` returns `ErrBootstrapKeyExpireForbidden`. This arm is unquestionably reachable — the impl explicitly guards against expiring the bootstrap key.
- `cmd/switchboard/admin_handlers.go:290-292` — the expire handler calls `resolveAndVerifyCallerRole(ctx, m, ops, svtnName, "", "admin.key.expire")` before validation; non-control-role callers receive `E-ADM-009: insufficient authority for operation admin.key.expire`.
- `cmd/switchboard/admin_handlers.go:413-414` — `mapAdminError` maps `ErrSVTNNotFound` → `E-SVTN-003: SVTN not found: <name>`, which fires when the operator expires a key in a nonexistent SVTN.

**Failure scenario:** Operator uses `sbctl admin key expire --svtn=<id> --key=<bootstrap-pubkey> --after=1h`, expecting either 0 (success) or E-ADM-013 (key not found). Instead receives an undocumented `E-ADM-021: bootstrap-key-expire-forbidden` error at exit 1 with no reference in the spec. Automation that parses expected exit-codes cannot distinguish this from a bug. The v1.20 changelog documents §108 register with five codes and §109 revoke's audited three; §110 expire was not similarly audited despite equivalent adjudication authority.

### F-P5P9-A-004 [LOW] — Spec §120 `admin svtn destroy` exit-code column omits E-SVTN-003

**Spec cite:** `interface-definitions.md` §120 (v1.20), exit-codes cell:
> `0=ok, E-ADM-011 (unauthorized), E-ADM-009 (insufficient role), E-CFG-001 (invalid SVTN name: empty / whitespace-only / >255 bytes / invalid UTF-8 / control chars)`

**Impl cite:** `cmd/switchboard/admin_handlers.go:812-814` — after `resolveAndVerifyCallerRole` succeeds, `m.Destroy(callerKey, a.Name)` is called; if the SVTN does not exist, `mapAdminError` (line 413) maps `ErrSVTNNotFound` → `svtnNotFoundErr{name: a.Name}` producing `E-SVTN-003: SVTN not found: <name>`.

**Failure scenario:** Operator invokes `sbctl admin svtn destroy --name=typo --confirm=SVTN-abcd1234` for a nonexistent SVTN; expects an E-CFG-001 or "unauthorized" path per spec; instead receives `E-SVTN-003: SVTN not found: typo` at exit 1. Not blocking, but the destroy exit-code enumeration is incomplete relative to its create sibling (§119 correctly lists E-SVTN-001 for the corresponding not-found-analog).

### F-P5P9-A-005 [LOW] — Bare `sbctl` usage line omits `--timeout` synopsis parity with spec §48

**Spec cite:** `interface-definitions.md` §48 synopsis:
```
sbctl [--target=<addr>] [--key=<path>] [--json] <subcommand>
```
This one-liner is missing `[--timeout=<dur>]` despite §54 documenting `--timeout` as a first-class global flag with a default.

**Impl cite:** `cmd/sbctl/main.go:54`:
```
usage: sbctl [--target=<addr>] [--key=<path>] [--json] [--timeout=<dur>] <subcommand> [args...]
```
Impl's usage line is more complete than the spec synopsis — includes `--timeout` and `[args...]`.

**Failure scenario:** Reader consulting spec §48 does not realize `--timeout` is available as a global flag. Non-blocking but drift: impl's usage text is authoritative and diverges from the spec synopsis. Recommend the spec's §48 be reflowed to match the actual output at `main.go:54`.

### F-P5P9-A-006 [LOW] — `admin key register`'s `--yes` warning targetFlag diverges from spec §128 sample text

**Spec cite:** `interface-definitions.md` §128:
> `--yes` — ... Emits a warning to stderr: `"WARNING: --yes bypasses confirmation; ensure correct --name target before scripting"`.

The spec text uses the literal `--name` in a documented warning string. The v1.17 changelog note (§145) says: "F-11A-3: §128 --yes warning corrects --svtn→--name (destroy uses --name)". This ratifies `--name` for destroy but does not carve out register's differing flag.

**Impl cite:** `cmd/sbctl/admin.go:462-465` — register calls `runDestroyConfirmGate("admin key register", *confirmFlag, *yesFlag, "--svtn", sio)`. The warning printer at `admin.go:370` interpolates `targetFlag` (which is `--svtn` for register, `--name` for destroy). So register emits: `WARNING: --yes bypasses confirmation; ensure correct --svtn target before scripting`.

**Failure scenario:** Operator scripting `sbctl admin key register --yes ...` sees a `--svtn`-flavored warning; if they cross-check against spec §128 they find a `--name`-flavored template and briefly suspect a bug or misconfiguration. Consider adding a footnote to §128 that the flag-name interpolation is command-specific (destroy: `--name`; register: `--svtn`). Behavior is correct; documentation is just parochially destroy-shaped.

## Observations

### OBS-P5P9-A-001 — `sbctl paths` unknown sub-verb error hard-codes "expected 'list'" without acknowledging §77 `paths ping`

`cmd/sbctl/main.go:78`: `usageErrf("paths: unknown sub-verb %q; expected 'list'", args[1])`. Spec §77 does have `sbctl paths ping --router=<addr>` under `PENDING-S-BL.CLI-SURFACE-COMPLETION`. The error naming "expected 'list'" is consistent with the adjudicated deferral but reads slightly aggressively — an operator following the spec might expect the CLI to at least name `ping` as pending. Not a bug (adjudicated); noting for future doc-CLI alignment.

### OBS-P5P9-A-002 — `admin.key.expire` daemon-side rejection of `after <= 0` predicated on `time.ParseDuration` accepting negative values

`cmd/switchboard/admin_handlers.go:305-312`: server parses `after` with `time.ParseDuration`, then rejects `ttl <= 0`. `time.ParseDuration` DOES accept negative durations (`-1h` → `-1 * time.Hour`), so the guard fires as intended. Spec §110 promising E-CFG-001 for "zero, negative, or >100 years" is impl-consistent — but only because the daemon does the sign check, not the parser. Documentation-worthy in a defense-in-depth section, not a bug.

### OBS-P5P9-A-003 — `sbctl` bare-invocation stderr help does not list `svtn` (correctly, since §59 documents removal); no operator-friendly hint about migration

`cmd/sbctl/main.go:54-55`: bare invocation prints `available subcommands: sessions, paths, router, console, admin`. Spec §59 records `sbctl svtn create` as REMOVED with a migration target (`sbctl admin svtn create`). The default arm at `main.go:101` returns `unknown subcommand: svtn` on `sbctl svtn ...` with no hint pointing to `sbctl admin svtn`. Not a spec violation (spec explicitly says exit 2 unknown-subcommand); UX hint opportunity only.

VERDICT: HAS_FINDINGS
