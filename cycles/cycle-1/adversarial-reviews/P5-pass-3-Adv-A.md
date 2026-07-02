---
document_type: adversarial-review
artifact_id: P5-pass-3-Adv-A
version: "1.0"
phase: 5
pass: 3
lens: public-surface-operator-ux
adversary_variant: A
verdict: HAS_FINDINGS
finding_high: 3
finding_medium: 4
finding_low: 2
observation_count: 3
develop_tip: 7fe3e29e4358df16e4e2f1de65a4e0d972540b4a
model: opus
time_spent_minutes: 5
files_read: 4
read_cap: 6
prior_passes_read: false
producer: adversary
timestamp: 2026-07-02T00:00:00Z
---

# Phase 5 Pass 3 Adv-A Public-Surface Review

**Verdict:** HAS_FINDINGS
**Tip:** `7fe3e29e4358df16e4e2f1de65a4e0d972540b4a`
**Model:** opus
**Time spent:** ~5 min
**Files read:** 4/6
**Lens:** public-surface / operator-UX

## Findings

### F-P5P3-A-001 [HIGH]: `sbctl svtn list` — canonical happy-path test vector is unreachable through the shipped operator surface

**What.** BC-2.07.002 documents `sbctl svtn list` as the first canonical happy-path Test Vector (line 151). `cmd/sbctl/main.go:49-50` dispatches the `svtn` subcommand to wire command `svtn.list`. No daemon (control / router / access / console) registers a handler for `svtn.list` — `cmd/switchboard/admin_handlers.go:126-133` shows `admin.svtn.create` and `admin.svtn.destroy` but no listing surface.

**Where.**
- `cmd/sbctl/main.go:49-50` (dispatch site)
- `cmd/switchboard/admin_handlers.go:126-133` (handler registry — no list-ing verb)
- `.factory/specs/behavioral-contracts/ss-07/BC-2.07.002.md:151` (canonical vector)
- `.factory/specs/behavioral-contracts/ss-07/BC-2.07.002.md:156` (PENDING-S-BL.SVTN-LIST-WIRE annotation)

**Why it's a finding.** From a first-time-operator's perspective this is the **canonical smoke command** — the one command that BC-2.07.002's own Test Vectors section names as the exemplar happy-path. Instead of returning a list, the operator receives `E-RPC-010: unknown command: svtn.list`. The failure mode is indistinguishable from a wire-protocol fault, so the operator will reasonably conclude "the daemon is broken / I mis-configured something." The PENDING annotation is an honest paper trail but it does not change what the fingers actually experience. Documenting a shipping public-surface defect and calling it a backlog story does not make it non-shipping.

**Suggested remediation.** One of:
(a) Register `admin.svtn.list` in `BuildAdminHandlers` and have `sbctl svtn` dispatch to it (fix the wire gap for real, in this cycle);
(b) Remove the `case "svtn"` arm from `main.go` and remove the row from BC-2.07.002 Canonical Test Vectors so the shipped surface does not advertise an operator smoke test that cannot succeed. The current state — ship the case-arm, annotate the BC, defer indefinitely — is the worst of all three.

### F-P5P3-A-002 [HIGH]: `sbctl version` and `sbctl ping` case-arms ship pointing at non-existent daemon handlers

**What.** `cmd/sbctl/main.go:79-82` includes:
```
version + ping case arms
```
Neither `version` nor `ping` has a daemon-side handler. Any operator running the two most conventional smoke commands after installing the tool — `sbctl version` and `sbctl ping` — gets `E-RPC-010: unknown command: version` and `E-RPC-010: unknown command: ping`. BC-2.07.002 EC-004 (line 144) and EC-005 (line 145) both acknowledge this with `PENDING-S-BL.PING-VERSION-WIRE`. EC-005 explicitly documents that a product-owner decision is still owed: "Product-owner decision at delivery: implement trivial handler returning `{"pong":true}`, or remove the sbctl case-arm."

**Where.**
- `cmd/sbctl/main.go:79-82`
- `.factory/specs/behavioral-contracts/ss-07/BC-2.07.002.md:144-145` (EC-004 + EC-005)

**Why it's a finding.** `version` and `ping` are the two conventional zero-cost operator smoke tests. Their `E-RPC-010: unknown command` responses masquerade as a wire-protocol fault — the operator has no way to distinguish "the daemon doesn't implement this" from "the daemon is misbehaving." Shipping the case-arms points a first-time user at a documented dead end. The PENDING annotation is a governance signal, not an operator UX; a first-time reader of the shipping binary does not read backlog stories.

**Suggested remediation.** Land the trivial `{"pong": true, "version": "<build>"}` handler pair in `internal/mgmt` (a two-hour job) OR remove the two case-arms from `main.go` and delete EC-004/EC-005 from the BC. As with F-001 the split state (advertise + annotate + defer) is the operator-hostile choice.

### F-P5P3-A-003 [HIGH]: E-ADM-018 shipping message does not tell the operator what value to pass to `--confirm` — spec-vs-impl drift

**What.** `.factory/specs/prd-supplements/error-taxonomy.md:90` (E-ADM-018 canonical message):
`"control-to-control revocation requires explicit confirmation: use --confirm=<svtn-id> to proceed"`
`cmd/switchboard/admin_handlers.go:413` (actual emitted message):
`return fmt.Errorf("E-ADM-018: control-to-control revocation requires explicit confirmation: pass --confirm to proceed (revoking control key from SVTN %q): %w", svtnName, err)`

The wire message the operator actually sees says `pass --confirm to proceed`. That phrasing does not tell the operator that `--confirm` takes a value, let alone that the value must be the SVTN identifier. An operator following the emitted message will type `sbctl admin key revoke ... --confirm` (bare flag) and get… well, either the same error a second time or a different flag-parsing error depending on the flag-parsing library.

**Where.**
- `cmd/switchboard/admin_handlers.go:413` (emitting site)
- `.factory/specs/prd-supplements/error-taxonomy.md:90` (canonical message)
- Also see the taxonomy changelog entries v3.4 and v3.5 (lines 232-233) — the exact phrasing `use --confirm=<svtn-id> to proceed` was itself a Pass-11 fix, so this drift has already been re-worked once, and the code did not track.

**Why it's a finding.** The taxonomy fix landed to specifically eliminate operator ambiguity ("use `--confirm=<svtn-id>`") and the shipping message reverted the fix. This is a self-directed footgun: the operator hits a friction wall, follows the message verbatim, and hits it a second time. The severity is HIGH from the operator-UX lens because the message is the only feedback path — there is no interactive help — and it currently promises less information than the spec canonically requires.

**Suggested remediation.** Replace the `%w`-wrapping fmt.Errorf at line 413 with a canonical-message emitter that produces byte-identical text to the taxonomy row (mirror the `svtnAlreadyExistsErr` / `svtnNotFoundErr` typed-error pattern already used in the same file). Add a unit test that asserts `strings.Contains(err.Error(), "use --confirm=<svtn-id> to proceed")` to prevent regression.

### F-P5P3-A-004 [MEDIUM]: `sbctl svtn <arbitrary-args>` silently discards trailing arguments and dispatches to `svtn.list`

**What.** `cmd/sbctl/main.go:49-50`:
`case "svtn": err = connectAndRun(ctx, *target, *key, *jsonOut, "svtn.list", nil, sio)`
No check on `len(args) >= 2` or on `args[1]`. Compare `sbctl paths list` (main.go:53-59) which requires the `list` subcommand argument and prints usage + exits 2 when absent, and `sbctl router <verb>` (main.go:60-74) which validates the verb. `sbctl svtn` alone is the only "list-shorthand" verb in the CLI.

**Where.** `cmd/sbctl/main.go:49-50`

**Why it's a finding.** An operator who types `sbctl svtn create my-svtn`, `sbctl svtn destroy prod`, or any other reasonable subcommand — expecting the CLI to complain that these aren't real subcommands — gets a silent dispatch to `svtn.list`, arguments discarded. Because the real create/destroy verbs live under `sbctl admin svtn create` and `sbctl admin svtn destroy` (line 77-78), the surface is inconsistent: `svtn` alone is list-shorthand, but `svtn create` looks like it should be `admin svtn create` and instead becomes a stealth list. This is the exact class of silent-misdispatch that becomes dangerous the moment someone types `sbctl svtn destroy some-svtn` and interprets the "SVTN listing" that comes back as evidence that the destroy succeeded (or, less catastrophically, spends minutes trying to figure out why "destroy" showed them a list).

**Suggested remediation.** Either:
(a) require an explicit `list` subcommand — `sbctl svtn list` — matching the `paths` pattern (this also aligns with BC-2.07.002's own canonical test vector wording, which says `sbctl svtn list` verbatim);
(b) reject unknown trailing args with exit 2 and a "unknown svtn subcommand" message.
Option (a) has better spec alignment.

### F-P5P3-A-005 [MEDIUM]: E-INT-999 shipping message drifts from taxonomy canonical

**What.** Error taxonomy v4.1 (line 196) canonical message:
`"unmapped internal condition, programmer error, please report"`
`cmd/switchboard/admin_handlers.go:428`:
`return fmt.Errorf("E-INT-999: unmapped admin error: %w", err)`
The wire message the operator sees carries `"unmapped admin error"`, not the documented `"unmapped internal condition, programmer error, please report"`.

**Where.**
- `cmd/switchboard/admin_handlers.go:428`
- `.factory/specs/prd-supplements/error-taxonomy.md:196` (canonical)

**Why it's a finding.** E-INT-999 is the last-resort "we didn't map this properly" signal. Its purpose is (a) to be conspicuous in logs and (b) to instruct the operator on what to do (report to the maintainer). The shipping message drops the `programmer error, please report` guidance — the whole point of the canonical text. Ruling-12 §7 in the taxonomy explicitly requires that any new handler-code family carry a taxonomy row + universality row + Ruling-12 §1 amendment "all in the same fix-burst" — the process discipline is not matched by the emission-site message.

**Suggested remediation.** Change line 428 to `fmt.Errorf("E-INT-999: unmapped internal condition, programmer error, please report: %w", err)`. Add a unit test on the message text.

### F-P5P3-A-006 [MEDIUM]: E-ADM-011 Variant 2 shipping message drops role and SVTN-name discriminators required by the taxonomy

**What.** Error taxonomy (line 85) canonical Variant 2 message:
`"permission denied: <role> key cannot destroy SVTN <svtn_name>"`
`cmd/switchboard/admin_handlers.go:419`:
`return fmt.Errorf("E-ADM-011: destroy unauthorized: %w", err)`

**Where.**
- `cmd/switchboard/admin_handlers.go:419`
- `.factory/specs/prd-supplements/error-taxonomy.md:85` (Variant 2)

**Why it's a finding.** The operator hits an authorization deny at destroy time and needs both discriminators (which role their key resolved to, and which SVTN they were trying to destroy) to figure out whether they used the wrong key or targeted the wrong SVTN. The shipping message strips both. The `%w` wrapping of the underlying sentinel may or may not carry them depending on whether `ErrDestroyUnauthorized`'s `Error()` text embeds those values (it is a `svtnmgmt` package internal — outside this pass's read budget). The operator-visible top-line loses the diagnostics.

**Suggested remediation.** Have `mapAdminError` route `ErrDestroyUnauthorized` through a typed error (mirror `svtnAlreadyExistsErr` at line 562) that composes `<role>` and `<svtn_name>` into the canonical Variant 2 message.

### F-P5P3-A-007 [MEDIUM]: Two long-standing un-reconciled error-code collisions in the operator taxonomy — E-CFG-002 and E-CFG-006

**What.** The error taxonomy carries two explicitly-flagged code collisions that predate this cycle and remain unreconciled:
- E-CFG-002 (line 100-101): "private key export not supported" in this taxonomy vs. `listen_addr invalid host:port` in BC-2.09.003 v1.2.
- E-CFG-006 (line 105-106): sbctl `--yes cannot be combined with --confirm` in this taxonomy vs. `drain_timeout` and `keepalive_interval` negative validation in BC-2.09.003 v1.4.
Both rows contain **KNOWN INCONSISTENCY** blocks noting "reconciliation is needed in a maintenance pass" and neither has been reconciled.

**Where.** `.factory/specs/prd-supplements/error-taxonomy.md:100-106`

**Why it's a finding.** An operator seeing `E-CFG-002` in the field cannot tell whether the daemon rejected their listen_addr or their key-export attempt. The whole purpose of a code taxonomy is to give the operator a stable, unambiguous identifier for the failure class; a collision reduces the code to a diagnostic hint. The taxonomy itself documents these as maintenance debt. From the public-surface lens the un-reconciled state is a MEDIUM operator UX defect — the collision itself is diagnostic noise, and the acknowledgment-without-fix pattern is repeating (see F-001, F-002).

**Suggested remediation.** Assign a next-free CFG slot to whichever collision fires less often in operator flows (E-CFG-011 / E-CFG-012 are free per the v2.5 changelog note) and update BC-2.09.003 in the same fix-burst.

### F-P5P3-A-008 [LOW]: BC-2.07.002 EC-004 documents a version-mismatch operator UX that does not ship

**What.** BC-2.07.002 EC-004 line 144:
> "Daemon returns version info; sbctl prints warning if version differs; command may still succeed if protocol is compatible."
The BC's own PENDING annotation acknowledges the wire command `version` has no daemon handler. Even if it did, there is no evidence in `cmd/sbctl/main.go` of any version-comparison-and-warn path invoked on other subcommands. The EC-004 description reads like a feature specification for a feature that has not been built.

**Where.** `.factory/specs/behavioral-contracts/ss-07/BC-2.07.002.md:144`

**Why it's a finding.** EC-004 is not describing "shipped behavior" — it is describing an intent. A first-time reader of the BC will assume the described version-mismatch warning exists and will be confused when it does not fire.

**Suggested remediation.** Convert EC-004 to a `DEFERRED:` annotation (or move it into a separate "planned edge cases" section) so a reader can distinguish "this ships" from "this is planned."

### F-P5P3-A-009 [LOW]: `sbctl <unknown-subcommand>` error gives no discovery path

**What.** `cmd/sbctl/main.go:83-85`:
`default: fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", subcommand); os.Exit(2)`
No hint about how to see the list of subcommands (no "run 'sbctl' or 'sbctl --help' for usage"). The `no-args` path (line 34-37) prints the usage line, but the `unknown-subcommand` path does not point at it.

**Where.** `cmd/sbctl/main.go:83-85`

**Why it's a finding.** Low-severity operator friction; a first-time user who typos a subcommand gets a dead-end. Trivial fix.

**Suggested remediation.** Append `"\nrun 'sbctl' with no args to see the usage line, or 'sbctl --help' for full flag help"` to the error message.

## Observations (non-blocking)

### O-P5P3-A-001: PENDING-annotation pattern is being applied to shipping operator-visible defects, not just to spec-only gaps

The BC-2.07.002 v1.6 / v1.7 change history shows a recurring pattern: an operator-facing defect is discovered, a `PENDING-<story-id>` annotation is added to the BC and taxonomy, and the fix is deferred to a backlog story. That policy is defensible for internal invariants and non-visible edge cases; from a public-surface lens it is questionable for defects that fire on the very first commands an operator types (F-001, F-002). This is a governance observation, not a defect — worth naming so the pattern is visible to future PO decisions.

### O-P5P3-A-002: [process-gap] There is no "canonical-message drift detector" between error-taxonomy and shipping code

F-003, F-005, and F-006 are the same underlying class: the emission site in `admin_handlers.go` drifts from the canonical text in `error-taxonomy.md`, and nothing catches the drift. This is exactly the class of defect a small test file — one `strings.Contains` assertion per error-code row — would prevent. A generation-once check would take under an hour and would eliminate an entire recurring finding class. Not writing a report file, just naming the process gap.

### O-P5P3-A-003: Positive callout on taxonomy governance discipline

The error-taxonomy.md changelog (v2.5 through v4.2, ~30 versioned entries) is unusually well-kept — every version bump has a matching row, cross-references named findings, and cites the source BC/ruling. This is a clear positive against POL-001 for this artifact. It also makes it much easier for a fresh reviewer (me) to reason about the state; that is worth preserving in the review record.

## Closing note

Read budget: 4 of 6 (BC-2.07.002, error-taxonomy.md, cmd/sbctl/main.go, cmd/switchboard/admin_handlers.go). I intentionally did not spend the last two reads on ss-08 / ss-03 or on `cmd/sbctl/admin.go` because the four files I did read were already producing HIGH findings with concrete file:line evidence, and further reading was more likely to expand the finding-count than to change the verdict. Uncovered surface for a future pass: `sbctl admin key revoke --confirm` flag-parser behavior (F-003 remediation will want a real integration test); `sessions.list` dispatch (main.go:51-52) — likely the same class as F-001 but not verified against a handler-registry read in this pass.
