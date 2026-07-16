---
artifact_id: BC-2.04.008
document_type: behavioral-contract
level: L3
version: "1.0"
status: draft
producer: product-owner
timestamp: 2026-07-15T00:00:00Z
phase: 1a
inputs:
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '.factory/specs/domain-spec/failure-modes.md'
  - '_bmad-output/planning-artifacts/prd.md'
  - 'decisions/S-BL.NODE-ADMISSION-PROVISIONING-rulings.md'
  - 'decisions/identity-cluster-architecture.md'
input-hash: "893d434"
extracted_from: null
bc_id: BC-2.04.008
subsystem: session-access
architecture_module: cmd/switchboard
capability: CAP-013
priority: P0
criticality: critical
scope_phase: PE
origin: greenfield
lifecycle_status: active
introduced: v0.1.0
modified:
  - date: 2026-07-15
    version: "1.0"
    change: >
      Initial draft — Discovery.Run() daemon-lifecycle wiring in runAccess:
      goroutine WG-tracked per ARCH-01, ctx.Canceled is clean shutdown not
      internalFailure, startup ordering (after mgmt server), Option Y placement
      in runAccessWithConnector. Authored per S-BL.NODE-ADMISSION-PROVISIONING
      BC groundwork list item N2 (rulings.md v1.0 §3, §6).
deprecated: null
deprecated_by: null
replacement: null
retired: null
removed: null
removal_reason: null
inputDocuments:
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '.factory/specs/domain-spec/failure-modes.md'
  - '_bmad-output/planning-artifacts/prd.md'
  - 'decisions/S-BL.NODE-ADMISSION-PROVISIONING-rulings.md'
  - 'decisions/identity-cluster-architecture.md'
traces_to: [CAP-013]
kos_anchors:
  - elem-node-router-architecture
---

# Behavioral Contract BC-2.04.008: Discovery Sender Goroutine (`Discovery.Run`) Wired into Access Daemon Lifecycle — WG-Tracked, ctx-Driven Shutdown

## Description

The access-mode daemon (`switchboard access`) must run `discovery.Discovery.Run(ctx)` as a
WaitGroup-tracked goroutine inside `runAccessWithConnector` so that the discovery heartbeat
sender is started with the daemon and cleanly joined on shutdown. This BC governs the
daemon-lifecycle integration of `Discovery.Run` — specifically: when it starts (after the
management server is up, keyed on the admission keypair being available), how it is tracked
(same `sync.WaitGroup` as sweep and frames-dropped tickers per ARCH-01 §Goroutine WaitGroup
Contract), and how it shuts down (`context.Canceled` from `disc.Run` is clean shutdown, not
an internal failure). `Discovery.Run` has zero production callers before
`S-BL.NODE-ADMISSION-PROVISIONING`; this BC specifies the wiring obligation.

## Preconditions

1. The access daemon is starting.
2. The admission keypair has been successfully loaded or generated (BC-2.09.004 postconditions
   satisfied) and `admissionPubKey` is available.
3. `discovery.New(discoveryCfg)` has been called with a valid `discovery.Config` including a
   non-nil `LocalNodeAdmissionPubkey` (the 32-byte raw Ed25519 public key from BC-2.09.004 PC-7).
4. The management server goroutine is already started (`newMgmtServer` + `serveMgmtServer`
   phases completed — startup-ordering rule per ARCH-12 §Daemon Mode Startup).
5. `runAccessWithConnector` is entered with the `disc *discovery.Discovery` instance (or
   equivalent — see Invariant 5 for the acceptable signature shapes).

## Postconditions

1. **WG-tracked start (ARCH-01 §Goroutine WaitGroup Contract):** `wg.Add(1)` is called in the
   caller scope BEFORE the `go` statement that launches `disc.Run(runCtx)`. The goroutine body
   calls `defer wg.Done()` as its first statement after entry.

2. **Same WaitGroup as peer goroutines:** The discovery goroutine is tracked in the same
   `sync.WaitGroup` that tracks the sweep ticker and frames-dropped ticker goroutines (those
   goroutines are also `wg.Add(1)` / `defer wg.Done()` per BC-2.04.007 PC-2 postcon-6 and
   ARCH-01 v1.7). A separate WaitGroup for the discovery goroutine is not used.

3. **ctx.Canceled is clean shutdown:** When `runCtx` is cancelled (SIGTERM/SIGINT or internal
   failure), `disc.Run(runCtx)` returns `context.Canceled` (or `context.DeadlineExceeded`).
   This return value MUST NOT set `internalFailure` and MUST NOT call `cancel()` — it is a
   clean, expected shutdown signal, indistinguishable from any other goroutine observing
   `<-runCtx.Done()`. Only unexpected non-context errors from `disc.Run` (if any) are
   candidates for setting `internalFailure`.

4. **No goroutine leak:** `wg.Wait()` in `runAccessWithConnector` (which joins the sweep,
   frames-dropped, and now discovery goroutines) returns cleanly with no goroutine leak after
   `runCtx` is cancelled. Test verification follows the BC-2.04.007 pattern: `t.Cleanup` +
   a bounded `wg.Wait()` timeout (≤100ms after cancellation).

5. **Startup ordering:** The `Discovery.Run` goroutine is started AFTER the management server
   is up and AFTER the admission keypair is loaded (phases (a)–(d) in the lifecycle ordering
   defined by `S-BL.NODE-ADMISSION-PROVISIONING-rulings.md` §3.1). It is NOT started before
   the keypair is available — doing so would cause `transmitAdvertisement` to fail with
   `ErrMissingNodeAdmissionPubkey`.

## Invariants

1. **ARCH-01 §Goroutine WaitGroup Contract (F-DWIP3-001):** `wg.Add(1)` is called synchronously
   in the launching goroutine's scope before the `go` statement. The launched goroutine calls
   `defer wg.Done()`. This is the same invariant enforced for sweep and frames-dropped tickers
   (BC-2.04.007 v1.3 / ARCH-01 v1.7 §I-1 adjudication).
2. **ctx.Canceled ≠ internalFailure:** `context.Canceled` returned by `disc.Run` is a normal
   shutdown signal. It MUST NOT trigger `internalFailure = true` or a redundant `cancel()` call.
3. **No duplicate WaitGroup:** The discovery goroutine is in exactly one WaitGroup, the same one
   used by the other post-connect goroutines in `runAccessWithConnector`. Parallel WaitGroups
   would require a separate `wg.Wait()` and would violate the "one clean `wg.Wait()` accounts
   for all goroutines" invariant.
4. **Admission keypair precedes Run:** `discovery.Config.LocalNodeAdmissionPubkey` MUST be
   populated before `discovery.New` is called and before `disc.Run` is started. Calling
   `disc.Run` without a populated `LocalNodeAdmissionPubkey` results in
   `ErrMissingNodeAdmissionPubkey` from `transmitAdvertisement` — this is a programming error
   caught at startup, not a runtime condition to handle gracefully.
5. **Signature flexibility (story-writer decision):** The exact extension to
   `runAccessWithConnector`'s signature (pass `disc *discovery.Discovery`, pass
   `discovery.Config` and construct inside, or pass `admissionPrivKey ed25519.PrivateKey` and
   let the function build both) is an implementation detail left to the story-writer. All three
   are architecturally equivalent for the purpose of this BC. The key point is that the
   `Discovery.Run` goroutine is WG-tracked in the same WaitGroup, regardless of where
   `discovery.New` is called.

## Trigger

`runAccessWithConnector` entered with a valid `*discovery.Discovery` instance (or equivalent
per Invariant 5); daemon startup after admission keypair is loaded.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | `runCtx` is already cancelled when `disc.Run` is called | `disc.Run(runCtx)` returns `context.Canceled` immediately; `wg.Done()` fires; no goroutine leak; no `internalFailure` set. |
| EC-002 | SIGTERM received while `disc.Run` is blocked waiting for the next heartbeat interval | `runCtx.Done()` fires; `disc.Run` returns `context.Canceled` at its next `<-ctx.Done()` select; goroutine drains within one heartbeat interval (30s max by default, shorter in tests with injected tick). |
| EC-003 | `runAccessWithConnector` returns before `disc.Run` goroutine has exited | `wg.Wait()` blocks until `disc.Run` exits; prevents goroutine leak. Test enforces a bounded deadline: `t.Cleanup` with `time.AfterFunc(100ms, func(){ t.Fatal("wg.Wait timed out") })`. |
| EC-004 | `disc.Run` returns an unexpected non-context error (hypothetical future path) | Error is logged at ERROR level; decision on whether to set `internalFailure` is left to the implementation. `context.Canceled` and `context.DeadlineExceeded` MUST NOT be treated as unexpected. |
| EC-005 | `disc.Run` called without `LocalNodeAdmissionPubkey` populated | `transmitAdvertisement` returns `ErrMissingNodeAdmissionPubkey`; `disc.Run` propagates the error on the first tick; `runAccess` startup sequence prevents this by loading the keypair before constructing `discovery.Config` (Invariant 4). |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| `runAccessWithConnector` starts with valid `disc`; `runCtx` cancelled immediately | `disc.Run` returns `context.Canceled`; `wg.Done()` fires; `wg.Wait()` returns within 100ms; no `internalFailure` | happy-path (PC-3, PC-4) |
| SIGTERM delivered after `disc.Run` is running | `runCtx` cancelled; `disc.Run` returns `context.Canceled` at next select; `wg.Wait()` returns cleanly; exit 0 | happy-path (PC-3, PC-4) |
| `wg.Add(1)` called before `go disc.Run(runCtx)` | Test (using `sync.WaitGroup` inspection or race detector) verifies no `wg.Add(1)` inside the goroutine body | happy-path (PC-1, Inv-1) |
| Discovery goroutine joins same WaitGroup as sweep ticker | `wg.Wait()` on a single WaitGroup joins all post-connect goroutines (discovery + sweep + frames-dropped); one `wg.Wait()` call suffices | happy-path (PC-2) |

## Verification Properties

| VP-NNN | Property | Proof Method | Notes |
|--------|----------|-------------|-------|
| test-as-evidence | `disc.Run` goroutine tracked in same WaitGroup as sweep and frames-dropped tickers; `wg.Wait()` joins all | unit (wg.Wait + deadline) | S-BL.NODE-ADMISSION-PROVISIONING AC |
| test-as-evidence | `context.Canceled` from `disc.Run` does not set `internalFailure` or call `cancel()` | unit | S-BL.NODE-ADMISSION-PROVISIONING AC |
| test-as-evidence | `wg.Add(1)` in caller before `go`; `defer wg.Done()` inside goroutine (ARCH-01 §Goroutine WaitGroup Contract) | unit (race detector + wg.Wait timeout) | S-BL.NODE-ADMISSION-PROVISIONING AC |
| test-as-evidence | Discovery goroutine does not start before keypair is loaded (`LocalNodeAdmissionPubkey` non-nil at `discovery.New` time) | unit | S-BL.NODE-ADMISSION-PROVISIONING AC |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-013 ("Access node tmux session publishing") per capabilities.md §CAP-013 |
| L2 Domain Invariants | (none directly; derives from DI-002 via keypair precondition) |
| Architecture Module | cmd/switchboard (runAccess, runAccessWithConnector) |
| Architecture Doc | ARCH-01-core-services.md §Goroutine WaitGroup Contract (F-DWIP3-001, v1.7 §I-1 adjudication) |
| Stories | S-BL.NODE-ADMISSION-PROVISIONING (all postconditions and ACs) |
| Capability Anchor Justification | CAP-013 ("Access node tmux session publishing") — the discovery sender's lifecycle is a prerequisite for CAP-013's access node to advertise sessions over the SVTN. Without `Discovery.Run` running in `runAccess`, advertisements are never sent regardless of keypair availability. The daemon lifecycle framing parallels BC-2.04.007's CAP-013 anchor. |

## Related BCs

- BC-2.04.007 — parallel: same ARCH-01 WaitGroup contract; discovery goroutine is the fourth goroutine (alongside Err-drain, frames-bridge, sweep ticker, frames-dropped ticker) in the post-connect WaitGroup; same test pattern (`t.Cleanup` + bounded `wg.Wait()`)
- BC-2.09.004 — depends on: `Discovery.Run` MUST NOT be started until the admission keypair from BC-2.09.004 is loaded
- BC-2.03.001 — downstream: `Discovery.Run` is the production caller of `Discovery.Advertise` / `transmitAdvertisement`; this BC wires the loop; BC-2.03.001 specifies what the loop does

## Architecture Anchors

- ARCH-01-core-services.md §Goroutine WaitGroup Contract (F-DWIP3-001, v1.7 — `wg.Add(1)` in caller before `go`, `defer wg.Done()` inside goroutine)
- decisions/S-BL.NODE-ADMISSION-PROVISIONING-rulings.md §3 (daemon-lifecycle wiring, Option Y ruling, startup ordering phases (a)–(f))

## Story Anchor

S-BL.NODE-ADMISSION-PROVISIONING — all postconditions in this BC trace to acceptance criteria for this story.

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.0 | 2026-07-15 | Initial draft — `Discovery.Run()` goroutine WG-tracked in same WaitGroup as sweep/frames-dropped tickers; `ctx.Canceled` is clean shutdown not `internalFailure`; startup ordering (after mgmt server + after keypair load); Option Y placement in `runAccessWithConnector`. Authored per S-BL.NODE-ADMISSION-PROVISIONING BC groundwork item N2. |
