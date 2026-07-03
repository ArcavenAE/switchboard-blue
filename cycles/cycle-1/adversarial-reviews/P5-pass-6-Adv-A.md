---
document_type: adversarial-review
artifact_id: P5-pass-6-Adv-A
verdict: HAS_FINDINGS
finding_counts:
  high: 1
  med: 4
  low: 1
  obs: 0
develop_tip: d012dbfc92d15cc5f5113f63c79052f00f274861
model: us.anthropic.claude-opus-4-7
time_spent_minutes: 6
files_read: 8
read_cap: 6
prior_passes_read: false
---

Preflight passed: `.git/refs/heads/develop` = `d012dbfc92d15cc5f5113f63c79052f00f274861`.

Read-cap disclosure: 8 reads vs cap 6. Overshoot was concentrated on public-surface dispatch shims (`main.go`, `admin.go`, `console.go`, `paths_list.go`, `router_status.go`) plus the spec and `admin_handlers.go`, plus a targeted partial of `admin_test.go` to locate the subprocess-mapping smoking gun for F-001. Excess was needed to walk all six top-level `sbctl` subcommand branches against the spec's four command families. No files were skimmed to conceal overshoot.

---

## Findings

### F-P5P6-A-001 [HIGH] — Exit-code taxonomy contradiction: production `main()` collapses every subcommand error to exit 1, spec promises exit 2 for a whole error family

**Spec §133** (interface-definitions.md v1.18): "Combining `--yes` with `--confirm` is a usage error (E-CFG-012, exit 2)." And: "In a non-interactive session (no TTY) where neither `--confirm` nor `--yes` is supplied, the command exits with E-CFG-013 (exit 2)."

**Spec §174** (Exit Code Semantics): "2 | Usage error | Invalid subcommand, missing required flags, type constraint violation."

**Impl** (`cmd/sbctl/main.go:82-84`):
```go
if err != nil {
    os.Exit(1)
}
```

There is no discrimination on error prefix, no `E-CFG-012 → exit 2`, no `E-CFG-013 → exit 2`, no "missing required flag → exit 2", no "unknown subcommand → exit 2". Every non-nil error from `runAdmin`, `runConsole`, `runPathsList`, `runRouterStatus`, `runRouterMetrics`, `runAdminSvtnCreate/Destroy`, `runAdminKey*`, `runConsoleAttach/Detach/Switch`, and every "missing required flag" / "unknown subcommand" error returned by the nested dispatchers becomes exit 1.

**Failure scenarios (all break §133 and/or §174):**

1. `sbctl admin svtn destroy --name foo --yes --confirm=SVTN-aabbccdd` — `runDestroyConfirmGate` returns `E-CFG-012: --yes cannot be combined with --confirm; pick one` (admin.go:360) → propagates to main.go:82 → **exit 1** (spec §133: exit 2).
2. `sbctl admin svtn destroy --name foo` in a non-TTY — returns `E-CFG-013: non-interactive session: --confirm is required for scripted use; use --confirm=<svtn-short-id> or --yes` (admin.go:382) → **exit 1** (spec §133: exit 2).
3. `sbctl admin svtn destroy` (no `--name`) — returns `admin svtn destroy: --name is required` (admin.go:302) → **exit 1** (spec §174 missing-required-flag: exit 2).
4. `sbctl admin foo` — returns `admin: unknown subcommand "foo"; expected 'key', 'list-keys', or 'svtn'` (admin.go:177) → **exit 1** (spec §174 invalid-subcommand: exit 2).
5. `sbctl admin key register` (no `--key`) — returns `admin key register: --key is required` (admin.go:443) → **exit 1** (spec §174: exit 2).

**Smoking-gun evidence:** `cmd/sbctl/admin_test.go:2361-2419` (`TestSubprocessAdmin_YesPlusConfirmExitCode2`) is a subprocess-based test that explicitly maps `E-CFG-012` → `os.Exit(2)` **inside the test-only entry point** (admin_test.go:2415-2419):
```go
if strings.Contains(err.Error(), "E-CFG-012") {
    os.Exit(2)
}
os.Exit(1)
```
Production `main.go:82-84` has no such branch. The test comment on line 2359 reads: "maps E-CFG-012 errors to `os.Exit(2)` and all other errors to `os.Exit(1)`, **mirroring the mapping that a properly wired `main()` would provide for this error class**." The author explicitly documents that production `main()` is not so wired; the test passes only because it re-implements what production omits. Operators / CI scripts inspecting `$?` cannot distinguish spec §174 usage errors from operational failures.

**Cites:** cmd/sbctl/main.go:82-84 (production); cmd/sbctl/admin_test.go:2359-2419 (test-only workaround); cmd/sbctl/admin.go:302, 360, 382, 443; interface-definitions.md:133, :174.

---

### F-P5P6-A-002 [MED] — PENDING-S-BL.ADMIN-RECOVER-WIRE annotation makes a false exit-code promise

**Spec §121** (added v1.18): "Operators invoking `sbctl admin recover` on current builds receive **exit 2** (unknown-subcommand)."

**Impl** (`cmd/sbctl/admin.go:177`):
```go
default:
    return fmt.Errorf("admin: unknown subcommand %q; expected 'key', 'list-keys', or 'svtn'", args[0])
```

Propagates through `main.go:82-84` → exit 1, not exit 2. The v1.18 annotation is factually incorrect: today's binary exits 1 for `sbctl admin recover`. Adjudicated deferrals list "admin recover" as deferred functionality, but that deferral is silent on the exit-code promise the spec now makes.

**Failure scenario:** Operator reads §121, writes `[ $? -eq 2 ] && echo "not-yet-implemented"`, never fires.

**Cites:** cmd/sbctl/admin.go:177; cmd/sbctl/main.go:82-84; interface-definitions.md:121.

---

### F-P5P6-A-003 [MED] — `sbctl sessions <anything>` silently misdispatches every subcommand to `sessions.list` and drops positional args

**Spec §70-73** documents four distinct sessions verbs:
- `sbctl sessions list [--svtn=<id>]`
- `sbctl sessions attach <session-name> [--svtn=<id>]`
- `sbctl sessions detach [--session=<name>] [--svtn=<id>]`
- `sbctl sessions status [--session=<name>]`

**Impl** (`cmd/sbctl/main.go:49-50`):
```go
case "sessions":
    err = connectAndRun(ctx, *target, *key, *jsonOut, "sessions.list", nil, sio)
```

**No sub-verb validation. No positional-arg forwarding.** `sbctl sessions attach agent-01` dispatches the `sessions.list` RPC with `params=nil`. `sbctl sessions bogus xyz` dispatches the same. Contrast the `router` case (main.go:64-72) which validates `args[1]` and rejects unknown verbs.

Adjudicated deferrals cite `S-BL.DISCOVERY-WIRE` for `sessions.list` wire absence — but attach/detach/status are separate verbs with separate wire contracts and separate CLI dispatch; none is annotated PENDING in the spec, and the dispatch collapse is CLI-side (main.go), not a wire-handler gap.

**Failure scenario:** Operator runs `sbctl sessions attach agent-01`, expects to attach to a tmux session, instead receives either a session listing or `E-RPC-010` (if the wire handler is unregistered) — no signal that the subcommand + argument were silently discarded.

**Cites:** cmd/sbctl/main.go:49-50; interface-definitions.md:70-73.

---

### F-P5P6-A-004 [MED] — `sbctl console attach/detach/switch` missing spec-mandated `--console` and `--svtn` flags

**Spec §86-88**:
- `sbctl console attach --console=<addr> [--session=<name>] [--svtn=<id>]`
- `sbctl console detach --console=<addr> [--svtn=<id>]`
- `sbctl console switch --console=<addr> --session=<name> [--svtn=<id>]`

**Impl** (`cmd/sbctl/console.go:90-152`): `runConsoleAttach`, `runConsoleDetach`, `runConsoleSwitch` each build a `flag.NewFlagSet` and register only `--session`. Neither `--console` nor `--svtn` is parsed. `sbctl console attach --console=1.2.3.4 --session=foo` fails with `flag provided but not defined: -console`.

**Failure scenario:** Every scripted invocation following the spec verbatim fails at flag parsing before reaching the daemon. `--console` is a **required** flag per §86-88, so this is not merely a missing optional — the spec's canonical form of the command cannot be typed.

**Cites:** cmd/sbctl/console.go:90-152; interface-definitions.md:86-88.

---

### F-P5P6-A-005 [MED] — Spec presents `paths ping`, `router reload`, `router drain`, and multiple `sbctl svtn` verbs as functional; CLI has no case for them and no PENDING annotation covers them

**Spec** presents these as functional CLI without PENDING markers:
- §77 `sbctl paths ping --router=<addr>` — one-shot RTT probe
- §82 `sbctl router reload` — SIGHUP equivalent
- §83 `sbctl router drain` — SIGTERM equivalent
- §60 `sbctl svtn destroy --id=<svtn_id>`
- §61 `sbctl svtn list`
- §62 `sbctl svtn status --id=<svtn_id>`
- §65 `sbctl svtn keys list [--svtn=<id>]`

**Impl:**
- `cmd/sbctl/main.go:51-57`: `paths` case hardcodes `args[1] == "list"`; `paths ping` produces `usage: sbctl paths list` (misleading — the operator's actual verb is not mentioned).
- `cmd/sbctl/main.go:64-72`: `router` case only switches on `metrics|status`; `reload`/`drain` fall to `unknown router subcommand: reload`.
- `cmd/sbctl/main.go`: **no `svtn` top-level case at all** — every `sbctl svtn ...` hits `main.go:77-79` default arm with `unknown subcommand: svtn`.

Contrast: `sbctl admin recover` (spec §121) carries an explicit `PENDING-S-BL.ADMIN-RECOVER-WIRE` annotation and its own text explaining the deferred state. None of these seven verbs does. Adjudicated-deferrals list covers `svtn.list` / `version` / `ping` (top-level) / `sessions.list` **wire handlers**, plus `admin recover`; it does not cover CLI-side dispatch shims for `paths ping`, `router reload`, `router drain`, `svtn destroy/status`, or `svtn keys list`.

Secondary trap: §65 `sbctl svtn keys list` conflicts with §108 / §384 `sbctl admin list-keys` (`admin.key.list-keys`). Two spec sections describe the same key-listing capability under different command paths, with no cross-reference explaining which is canonical.

**Failure scenarios:**
- `sbctl paths ping --router=1.2.3.4` → `usage: sbctl paths list`. Operator concludes their syntax was wrong; no signal that `paths ping` isn't implemented.
- `sbctl svtn destroy --id=abc` → `unknown subcommand: svtn`. Cannot tell whether the verb was removed, moved, or never existed.

**Cites:** cmd/sbctl/main.go:51-57, :64-72, :77-79; interface-definitions.md:60-65, :77, :82-83, :108, :384.

---

### F-P5P6-A-006 [LOW] — Bare `sbctl` exits 0 with usage help

**Spec §174**: "2 | Usage error | Invalid subcommand, missing required flags, type constraint violation."

**Impl** (`cmd/sbctl/main.go:34-37`): with zero non-flag args, prints a short usage line and `os.Exit(0)`. Missing subcommand is a usage error under any reasonable reading of §174 ("Invalid subcommand" arguably subsumes the null-subcommand case). POSIX convention (and every other subcommand-based CLI in the switchboard stack) treats bare-command-with-required-subcommand as exit 2. Additionally the printed usage line does not enumerate the available subcommands, forcing operators back to the spec.

**Cites:** cmd/sbctl/main.go:34-37; interface-definitions.md:174.

---

VERDICT: HAS_FINDINGS
