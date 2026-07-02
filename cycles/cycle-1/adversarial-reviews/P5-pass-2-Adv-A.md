---
document_type: adversarial-review
artifact_id: P5-pass-2-Adv-A
version: "1.0"
phase: 5
pass: 2
lens: public-surface-operator-ux
adversary_variant: A
verdict: HAS_FINDINGS
finding_high: 0
finding_medium: 2
finding_low: 1
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

# Phase 5 Pass 2 Adv-A Public-Surface Review

**Verdict:** HAS_FINDINGS
**Develop tip:** 7fe3e29e4358df16e4e2f1de65a4e0d972540b4a
**Model:** opus
**Time spent:** ~5 minutes
**Files read:** 4 / 6
**Scope lens:** public-surface / operator-UX

## Findings

### F-P5P2-A-001 [MEDIUM]: `sbctl version` dispatches wire command `version` with no daemon-side handler

- **What:** `cmd/sbctl/main.go:80` dispatches `connectAndRun(ctx, *target, *key, *jsonOut, "version", nil, sio)` for the `sbctl version` subcommand, but no daemon registers a handler for the wire command `"version"`. `internal/mgmt/register_metrics.go` registers `paths.list`, `router.metrics`, `router.status`; `cmd/switchboard/admin_handlers.go` registers the six `admin.*` commands; `cmd/switchboard/console_handlers.go` registers the three `console.*` commands. A ripgrep for `"version"` as a handler `Command:` field across `internal/mgmt` and `cmd/switchboard` returns only the doc-comment reference in `mgmt.go:82`.
- **Where:** `cmd/sbctl/main.go:80` (dispatch); no handler registration under `cmd/switchboard/*.go` or `internal/mgmt/register_metrics.go`. BC-2.07.002 EC-004 references "Daemon returns version info; sbctl prints warning if version differs" — the spec anticipates a version RPC.
- **Why it's a finding:** New operator running `sbctl version` — a first-class top-level subcommand advertised via `main.go`'s usage-adjacent flow — will complete the ADR-012 handshake, then receive `E-RPC-010: unknown command: version` in-band and exit 1 with a confusing error. Same failure shape as the annotated `svtn.list`/`sessions.list` gaps from Pass 1, but not documented as a `PENDING-<story>` in any BC or in the error-taxonomy row. Operators cannot distinguish "gap" from "defect" from the error surface.
- **Suggested remediation:** annotate BC-2.07.002 EC-004 (or the appropriate BC) with `PENDING-<story>`, or register a trivial version handler in a shared registration path. Prefer annotation if version-RPC is deferred; the code shape mirrors S-BL.SVTN-LIST-WIRE / S-BL.DISCOVERY-WIRE precisely.

### F-P5P2-A-002 [MEDIUM]: `sbctl ping` dispatches wire command `ping` with no daemon-side handler

- **What:** `cmd/sbctl/main.go:82` dispatches `connectAndRun(ctx, *target, *key, *jsonOut, "ping", nil, sio)` for the `sbctl ping` subcommand. No daemon registers a handler for `"ping"`. Ripgrep for `"ping"` as a handler `Command:` field returns zero matches across the daemon-side registration surface.
- **Where:** `cmd/sbctl/main.go:81-82` (dispatch); no handler registration anywhere in `cmd/switchboard/*.go` or `internal/mgmt/register_metrics.go`. Ping is not documented in any BC that I sampled (BC-2.07.002, BC-2.07.003, admin/console BCs) as a spec'd RPC — this may be a bare `sbctl` convenience command whose semantics were never anchored to a BC.
- **Why it's a finding:** Operator smoke-testing connectivity with `sbctl ping` — the natural first-command an operator reaches for after `--target` setup — completes AUTH_OK and then receives `E-RPC-010: unknown command: ping`. This is a strictly worse operator experience than the daemon simply not existing (which returns E-NET-001 with a clear "unreachable" message). Same pattern as F-P5P2-A-001 but with the added twist that ping isn't anchored to any BC at all — no BC exists to annotate. Either an unshipped feature was left dangling in `main.go`, or ping needs a BC + handler.
- **Suggested remediation:** decide: (a) implement a `ping` handler that returns `{"pong":true}` (cheapest — no state, no auth-specific logic); (b) delete the `case "ping":` arm from `main.go`; (c) mint a BC + register a `PENDING-<story>` annotation. Option (a) is by far the smallest fix and gives operators the connectivity smoke-test they expect.

### F-P5P2-A-003 [LOW]: `admin.key.list-keys` test helper uses wrong wire name (`admin.key.list`) — divergence from shipped surface

- **What:** `cmd/sbctl/e2e_helpers_test.go:191` registers a mock handler for wire command `"admin.key.list"` (no `-keys` suffix), whereas the shipped sbctl dispatches to `"admin.key.list-keys"` (`cmd/sbctl/admin.go:132`) and the daemon-side handler registers `"admin.key.list-keys"` (`cmd/switchboard/admin_handlers.go:130`). This is test-scaffolding drift, not shipped-surface breakage — but the test helper is misleading about what wire command is under test.
- **Where:** `cmd/sbctl/e2e_helpers_test.go:191` vs `cmd/switchboard/admin_handlers.go:130` and `cmd/sbctl/admin.go:132`.
- **Why it's a finding:** The e2e helper appears to exercise `list-keys` from an operator's perspective but registers the wrong command name; the mocked handler would never fire against the real shipped call path. Not a live operator-UX failure, but the test does not defend the shipped wire contract. Flagged LOW because operator-visible surface is correct — this is a testing/observation-layer concern.
- **Suggested remediation:** rename the mock's `Command` field from `"admin.key.list"` to `"admin.key.list-keys"` in `e2e_helpers_test.go:191`. One-line fix.

## Observations (non-blocking)

### O-P5P2-A-001: PENDING annotations for `svtn.list` and `sessions.list` correctly implemented as tracked deferrals

Verified BC-2.07.002 v1.6 (2026-07-02) contains the `PENDING-S-BL.SVTN-LIST-WIRE` block on the Canonical Test Vectors section explaining the happy-path row is unreachable through the shipped surface — reads as tracked deferral, not fresh finding. Verified BC-2.03.002 v1.4 (2026-07-02) analogously annotates PC-1 with `PENDING-S-BL.DISCOVERY-WIRE`. Both correctly avoid claiming shipped behavior. Both would be flagged by a fresh operator-UX pass except for the annotation shape. Adv-A Pass 1 remediation appears sound.

### O-P5P2-A-002: `mapAdminError` default-arm E-INT-999 shape looks correct

Verified `admin_handlers.go:428` — default arm returns `"E-INT-999: unmapped admin error: %w"` and the wire wrapper (mgmt.go stamps E-RPC-011 as envelope code). Matches the v4.1 error-taxonomy row: wire envelope carries E-RPC-011 with E-INT-999 embedded in the message. No shape defect.

### O-P5P2-A-003: `admin.svtn.list` explicitly noted as absent — operator has no admin-scoped SVTN listing

BC-2.07.002 PENDING block explicitly acknowledges `admin.svtn.list` is NOT registered — only `admin.svtn.create` and `admin.svtn.destroy` exist on the admin surface. This gap is now visible and traced to `S-BL.SVTN-LIST-WIRE`. A control-mode operator can create+destroy SVTNs but has no way to list what they've created via the wire. Documented, not silently broken — hence non-blocking. Included here for orientation, not as a finding.
