---
document_type: adversarial-review
phase: 5
pass: 1
lens: Adv-A
scope: public-surface-operator-ux
develop_tip: 7fe3e29e4358df16e4e2f1de65a4e0d972540b4a
prior_passes_read: false
verdict: HAS_FINDINGS
finding_high: 2
finding_medium: 1
finding_low: 0
observation_count: 2
timestamp: 2026-07-02
model: opus
---

# Phase 5 Pass 1 — Adversary A (Correctness / Coverage, public-surface / operator-UX lens)

## Verdict

**HAS_FINDINGS** — 2 HIGH, 1 MEDIUM, 2 observations. Genuinely novel — orthogonal to the Phase 4 HS-006 router-daemon-stub + drain-CLI drift items.

The finding pattern is identical in structure to the Phase 4 router-daemon-stub finding (documented capability → orphaned surface → E-RPC-010 on operator invocation) but at a different location: the sbctl→daemon wire boundary. Three top-level `sbctl` subcommands wired in `cmd/sbctl/main.go:48-86` dispatch to wire commands that **no daemon anywhere in this tree registers**. The internal-code adversary chain missed these because internal-code coverage is fine — each side of the wire compiles and tests; the gap is only visible when you enumerate `sbctl` subcommands against the union of `Command:` registrations across all daemon modes.

## F-P5P1-A-001 (HIGH) — `sbctl svtn` orphaned: no daemon registers `svtn.list`

**Public-surface evidence.** `cmd/sbctl/main.go:49-50` dispatches subcommand `svtn` to wire command `svtn.list` via `connectAndRun`. Exhaustive grep of all `Command:\s*"` literals in the tree (excluding `_test.go` fixtures) yields the complete daemon-registered set: `console.attach|detach|switch` (`cmd/switchboard/console_handlers.go:49-51`), `admin.key.{register,revoke,expire,list-keys}` + `admin.svtn.{create,destroy}` (`cmd/switchboard/admin_handlers.go:127-132`), and `paths.list|router.metrics|router.status` (`internal/mgmt/register_metrics.go:46-52`). **`svtn.list` is not among them.** Server dispatch at `internal/mgmt/mgmt.go:677-680` returns `E-RPC-010: unknown command: svtn.list` for any command not in `s.handlers`. Every `sbctl svtn` invocation against a live daemon fails with E-RPC-010 regardless of mode.

**BC clause promising the missing behavior.** BC-2.07.002 (BC-INDEX line 65, `active`): "sbctl unified CLI for all four daemon types with OpenSSH key authentication." Canonical Test Vectors at `.factory/specs/behavioral-contracts/ss-07/BC-2.07.002.md:135-136`:

- `sbctl svtn list` with registered key → "List of SVTNs returned" (happy-path)
- `sbctl svtn list` with unregistered key → E-ADM-010 (error)

The happy-path row is unreachable on develop@7fe3e29: even with a registered key the response is E-RPC-010, not "List of SVTNs".

**Merged Wave-6 story anchor.** BC-2.07.002 is `active` in BC-INDEX and referenced through S-6.03/BC-2.07.003 chain. `sbctl svtn` is also the operator command used to *smoke-test* the BC-2.07.003 connect-error path (BC-2.07.003:135-138 all use `sbctl svtn list --target=…`).

## F-P5P1-A-002 (HIGH) — `sbctl sessions` orphaned: no daemon registers `sessions.list`

**Public-surface evidence.** `cmd/sbctl/main.go:51-52` dispatches `sessions` → wire command `sessions.list`. Exhaustive `Command:\s*"` grep confirms **no `sessions.list` (or `session.list`) handler is registered anywhere**. Result: E-RPC-010 on every `sbctl sessions` invocation.

**BC clause promising the missing behavior.** BC-2.03.002 (BC-INDEX line 44, `active`, subsystem `session-discovery`): "Console enumerates all SVTN sessions without specifying hostnames or IP addresses." Description at `.factory/specs/behavioral-contracts/ss-03/BC-2.03.002.md:40`: *"This is the core operator experience: `sbctl sessions list` returns all sessions across the fleet."* PC-1 (line 50): "`sbctl sessions list` (or equivalent API call) returns a list of all sessions currently known to the console." Test vectors (lines 79-80) both invoke `sbctl sessions list`.

**Merged Wave-6 story anchor.** S-7.02 (merged in commit c54a8ad, PR #55) declares `bc_traces: [BC-2.03.001, BC-2.03.002, BC-2.03.003]` (`.factory/stories/S-7.02-session-discovery.md:23`). The story implements `Discovery.Enumerate()` internally (AC-002) but no daemon RPC handler exposes it. S-BL.DISCOVERY-WIRE is the backlog story for the wire boundary, but BC-2.03.002 postcondition 1 promises the CLI command works today — and the CLI dispatcher promises `sessions.list` at the wire — so the public surface currently returns E-RPC-010 while both the spec and the CLI code claim the capability exists. This is worse than the router-daemon stub: the router subcommand at least prints an honest error; `sbctl sessions` completes handshake, dispatches, and returns a domain-code error masquerading as a wire error.

## F-P5P1-A-003 (MEDIUM) — `sbctl ping` and `sbctl version` orphaned

**Public-surface evidence.** `cmd/sbctl/main.go:79-82` dispatch `version` → wire `version`, and `ping` → wire `ping`. Grep for `Command:\s*"version"` and `Command:\s*"ping"` in all non-test Go source files returns **zero matches**. Both return E-RPC-010 on invocation against a live daemon.

**Countervailing evidence.** The AUTH_OK response envelope does carry a `daemon_version` field (`internal/mgmt/mgmt.go:425`, tested at `internal/mgmt/mgmt_test.go:713`), so a client that AUTHed and inspected the handshake response could read the version — but that's not what `sbctl version` is wired to do. `sbctl version` sends a post-handshake `version` RPC that has no handler.

**Impact.** These are the operator's canonical liveness probes. A user running `sbctl ping` to smoke-test a fresh daemon deployment receives `E-RPC-010: unknown command: ping` — a domain-code error that suggests "unknown command" rather than "daemon is live and auth works." This is a first-touch operator UX defect: two subcommands appear in the `sbctl` help/default output but neither reaches a working daemon path. Severity MEDIUM (not HIGH) because no BC directly promises these; the promise is only in the CLI's own dispatcher table.

## Observation-1 — BC-2.07.003 connect-error test vectors use an orphaned command

`.factory/specs/behavioral-contracts/ss-07/BC-2.07.003.md:135-138`: **all four** canonical test vectors invoke `sbctl svtn list --target=…` as the trigger. The BC's postcondition is about the *error path* (daemon unreachable → E-NET-001), and that path is genuinely reachable. But this couples the observable spec-conformance test to a command whose happy path is broken (see F-P5P1-A-001). The spec is not wrong — the trigger command happens to also fail on connect — but any operator following the spec verbatim to sanity-check E-NET-001 will get their true positive obscured by a downstream E-RPC-010 as soon as they *do* successfully connect.

## Observation-2 — No `admin.svtn.list` handler; operator has no listing surface

`cmd/switchboard/admin_handlers.go:127-132` registers `admin.svtn.create` and `admin.svtn.destroy` but **not** `admin.svtn.list`. `admin.key.list-keys` is exposed (`cmd/switchboard/admin_handlers.go:130`) via `sbctl admin list-keys`, giving parity for the key side of the admin namespace. There is no equivalent for SVTNs — an operator who has created several SVTNs via `sbctl admin svtn create` has no public-API path to enumerate them. BC-2.07.001 promises create/destroy but does not name a listing postcondition, so this is not a HIGH finding — it's a symmetric gap noted for post-Wave-6 planning. If BC-2.07.002 "sbctl unified CLI" is interpreted as requiring `sbctl svtn list` (see F-P5P1-A-001), then the wire boundary needs *both* an admin-scoped and a broader enumeration surface.

## Novelty Assessment

**Novelty: HIGH** for F-P5P1-A-001 and F-P5P1-A-002 — they extend the router-daemon-stub pattern (documented-capability / orphaned-surface / E-code masking) to two additional merged-Wave-6 story commands that the internal-code adversary chain apparently did not enumerate cross-cut. Grep across `Command:\s*"` literals is a lens no internal-code review would run because internal code is complete on each side of the wire. The gap is only observable from the operator's shell.

F-P5P1-A-003 is a moderate refinement — same pattern but weaker BC anchoring (CLI-declared, not BC-declared).

Observations are pattern-adjacent and worth documenting for Wave-7 planning.

## Files load-bearing for these findings

- `cmd/sbctl/main.go`
- `cmd/switchboard/admin_handlers.go`
- `cmd/switchboard/console_handlers.go`
- `internal/mgmt/register_metrics.go`
- `internal/mgmt/mgmt.go`
- `.factory/specs/behavioral-contracts/ss-03/BC-2.03.002.md`
- `.factory/specs/behavioral-contracts/ss-07/BC-2.07.002.md`
- `.factory/specs/behavioral-contracts/ss-07/BC-2.07.003.md`
- `.factory/stories/S-7.02-session-discovery.md`
