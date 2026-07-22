# PR Review — #126 S-BL.ADMISSION-SYNC-WIRE

**Reviewer:** pr-reviewer (fresh-eyes, cognitive-diversity model)
**Branch:** feature/S-BL.ADMISSION-SYNC-WIRE → develop
**HEAD:** ab043c5
**Base verified:** origin/develop @ ce06f6a (diff = 7254 insertions, matches PR metadata)

## Verdict: APPROVE — no blocking findings

High-quality, heavily-adjudicated PR. The diff faithfully implements the stated
design. Independent verification on my machine: `go build ./...` clean,
`go vet ./cmd/switchboard/...` clean, `gofmt -l` clean on new files, and a
representative test subset (`PushFullSnapshot`, `RouterAdmissionHandler`,
`Snapshot`, `LoadAndPushFullSnapshot`, `RegisterKey`, `RouterStartup`) passes.
PR description, traceability, adversarial ledger, and demo evidence
(`factory-artifacts@d9a4f46`) all match the diff.

### Verification performed (no rubber-stamp)
- Diff coherence: all 19 files relate to admission-sync. (Note: my local
  `develop` was stale at d249f88, which initially inflated the diff with
  already-merged NODE-ADMISSION-PROVISIONING work; re-based against
  origin/develop the diff is clean and matches the PR's 7254 additions.)
- Fail-closed: `loadSnapshotFromFile`/`unmarshalSnapshot` wrap E-KEY-002 for
  corrupt JSON and unknown schema_version; router refuses to serve on error.
- Loopback guard (AC-012): `buildMgmtListener` restricts console/control/access,
  exempts router only — matches Ruling 9/12.
- Concurrency: control/router persisters serialize {read,marshal,write,rename};
  `AllSVTNEntries` deep-clones entries + PublicKey slices; atomic write uses
  per-write unique temp + rename (go.md locked-accessor rule satisfied).
- Severity-bounding fact: `admission.AdmitNode` and `ReAuthenticate` have ZERO
  production callers (only internal/testenv). Synced router state is durable but
  not yet consulted by any live admission decision until S-BL.NODE-IDENTIFY-WIRE.
  This bounds the severity of every correctness finding below.

---

## Findings (all NON-BLOCKING, priority order)

### F1 — [correctness] Async delta-push reordering can leave a revoked key ACTIVE on a router snapshot
`admin_handlers.go` dispatches each push via `dispatchPush` → one goroutine per
operation, with **no cross-operation ordering/sequencing**. Scenario: register K,
then immediately revoke K on control (both control-side handlers succeed). Two
independent goroutines then race to the router over separate TCP connections,
each with retry-with-backoff (up to 5×~10s). If the `register` push hits a
transient dial failure while `revoke` lands first, the router's
`makeAdmissionRevokeHandler` sees key-not-found → returns success WITHOUT a
tombstone (Ruling 13 semantics) → then the delayed `register` lands and creates
an ACTIVE entry. Net: router durable snapshot has K active while control
considers it revoked — the exact Invariant-6 divergence this story exists to
prevent. Even absent retries, Go gives no happens-before between the two
goroutines. Tests miss this because they run synchronous mode (`pushWG == nil`).
Mitigants: not exploitable until AdmitNode is wired (NODE-IDENTIFY-WIRE); control
restart's PushFullSnapshot eventually reconciles.
**Recommendation:** track as a forward obligation for S-BL.NODE-IDENTIFY-WIRE
alongside O-1 (per-key push serialization or a monotonic generation stamp before
the consumer goes live).
Location: `cmd/switchboard/admin_handlers.go` (dispatchPush sites);
`cmd/switchboard/admission_sync_wire.go` (revoke key-not-found no-op).

### F2 — [availability] Startup PushFullSnapshot is synchronous and gates control from serving
In `runControlWithKey`, `PushFullSnapshot(ctx, ks)` runs BEFORE `serveMgmtServer`.
No-op for an empty keyset (common fresh start), but with a persisted control
keyset AND an unreachable/black-holed endpoint it blocks the full retry budget
per entry per endpoint (~75s/endpoint × N entries) since production's
`signal.NotifyContext` has no deadline. During that window control accepts no
admin RPCs — partially couples control STARTUP availability to router
reachability, in mild tension with the ADR's rationale for rejecting
pull-on-demand. PR performance table understates this (omits the per-entry
retry-budget multiplier against unreachable endpoints).
**Recommendation:** bound startup push with a timeout/deadline, or run it
non-blocking (WG-tracked) like the delta path.
Location: `cmd/switchboard/mgmt_wire.go` (runControlWithKey);
`cmd/switchboard/admission_sync_client.go` (retry budget).

### F3 — [correctness-precision] Expire wire protocol is duration-based; absolute expiry drifts
Control sends `after: ttl.String()`; router recomputes
`expiry = time.Now().UTC().Add(ttl)`. Absolute expiry differs between control and
router by push latency + clock skew, and re-drifts on each full-snapshot resync
(`ttl = time.Until(expiry)`). `TestAdmissionSync_PushFullSnapshot_ExpiryPushed`
acknowledges this with a tolerance rather than exact equality. Bounded and minor
for coarse expiry, but an absolute-timestamp wire field would be exact — worth a
note since expiry is security-relevant once the consumer lands.
Location: `cmd/switchboard/admission_sync_wire.go` (makeAdmissionExpireHandler).

### F4 — [maintainability/nit] `mgmtNetwork` is dead in production
After `mgmtListenAddr` was rewritten to auto-detect TCP/unix, `mgmtNetwork` has
no non-test caller (only its definition + `TestMgmtNetwork_PerMode`). Lint stays
green only because the test references it. Consider removing or folding in.
Location: `cmd/switchboard/mgmt_wire.go`.

### F5 — [maintainability/nit] Test-only production artifacts
`errAdmissionSyncNotImplemented`, `errSnapshotNotImplemented`, and
`marshalSnapshot`'s always-nil error return (with `//nolint:unparam`) exist
solely for frozen-test `errors.Is`/signature compatibility. Documented and
lint-clean, but a residual TDD smell — clean up once the suite can be unfrozen.
Location: `cmd/switchboard/admission_sync_client.go`,
`cmd/switchboard/admission_sync_snapshot.go`.

### F6 — [maintainability/nit] `BuildAdminHandlers` is now test-only + variadic-optional-arg
Production `runControlWithKey` calls `buildAdminHandlersCore` directly; the
exported `BuildAdminHandlers(..., controlSnapshotPath ...string)` is reached only
from tests. Fine to retain for tests, but the variadic-for-one-optional-string on
an exported symbol is a minor smell.
Location: `cmd/switchboard/admin_handlers.go`.

---

## PR description accuracy
Accurate and well-evidenced. Additions count, adversarial-pass ledger, 4 rulings,
security dispositions (1 HIGH accepted-by-design, 3 MED + 3 LOW follow-ups), and
BC→AC→test traceability all match the diff. Demo evidence (13 `.tape` +
evidence-report.md) present on factory-artifacts@d9a4f46 as claimed (POL-004,
no binaries). Only gap: the startup-push performance characterization (see F2).

## Note on posting mechanics
switchboard-blue single-identity constraint: arcavenai authors AND reviews, so
GitHub yields no formal APPROVE/REQUEST_CHANGES reviewDecision. This review is
posted as the formal verdict record; convergence is CI-green + this review.
