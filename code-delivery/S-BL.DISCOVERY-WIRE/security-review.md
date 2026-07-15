# Security Review — S-BL.DISCOVERY-WIRE (PR #123)

**Reviewer:** vsdd-factory:security-reviewer (fresh-eyes pass, dedicated dispatch)
**Scope:** `git diff develop..feature/S-BL.DISCOVERY-WIRE` at code HEAD `501db03` (branch tip at
time of review `61767d1`)
**Repo/branch:** ArcavenAE/switchboard-blue, `feature/S-BL.DISCOVERY-WIRE` -> `develop`

---

## Verdict

No CRITICAL or HIGH findings. Core cryptographic surface (HKDF domain separation,
constant-time comparison, fail-closed lookup-miss/tag-mismatch unification, fixed-offset
pre-auth parsing, buffer bounds) is sound and matches the adjudicated design. 2 LOW
findings, both informational/forward-guidance on code paths that are not yet wired into
any production caller — neither blocks merge.

Verified as correct (not re-flagged, per the story's adjudicated design list):
`DeriveDiscoveryKey` derives from the admitted node's pubkey
(`internal/routing/advertisement_hmac.go:45`, `55`) with a distinct HKDF info label
(`HKDFInfoDiscovery = "switchboard-discovery-auth"`, `internal/hmac/hmac.go:36`) from
the frame-auth key, and `TestDeriveDiscoveryKey_DomainSeparatedFromFrameAuthKey`
empirically asserts the two outputs differ for identical inputs — the F-DWIP1-001 fix is
confirmed correct, not just documented. `VerifyHMAC`/`VerifyAdvertisementHMAC` use
`crypto/hmac.Equal` (constant-time). `RouterIngest.Ingest` extracts the SVTNID/NodeAddr
key selector via fixed-offset slicing before any variable-length, attacker-controlled
session-list walk, and every slice access (`body[0:16]`, `body[16:24]`, `body[24:32]`,
`DecodeSessionList`'s per-entry offsets) is preceded by an explicit length check sized to
that exact access — no out-of-bounds slice/panic path found. Lookup-miss and
HMAC-mismatch both resolve to `ErrInvalidHMACTag` with no other observable
differential (confirmed by
`TestRouterIngest_LookupMissAndTagMismatch_IndistinguishableRejection`).
`FailureCounter` (reused, not part of this diff) already bounds its per-source map at
`maxTrackedSources = 65536` with LRU eviction (CWE-770 already mitigated pre-existing),
so reusing it for the new, cheaper-to-spoof UDP surface doesn't reopen unbounded map
growth.

---

## Findings

### SEC-101: Discovery ingest rate limiter and failure counter are per-`RouterIngest` instance, not per-SVTN — cross-tenant starvation risk depends on how the (not-yet-written) production wiring instantiates it
- **Severity:** LOW
- **CWE:** CWE-770 (Allocation of Resources Without Limits or Throttling — present, but
  not partitioned along the trust boundary that matters)
- **OWASP:** A04:2021 – Insecure Design
- **Attack Vector:** `RouterIngest.Ingest` (`internal/discovery/discovery_wire.go:252`)
  authenticates the packet's own declared `svtnID` field against the admitted-key set,
  but the aggregate token bucket (`aggregateRateBurst=500`,
  `aggregateRateFillPerSec=100.0`) and the shared `FailureCounter` live on the
  `RouterIngest` struct itself, with no SVTN-keyed partitioning.
  `wireDiscoveryListener` takes a `*discovery.RouterIngest` as a parameter rather than
  constructing one per SVTN group it joins. If a router hosts multiple SVTNs and a
  follow-on story (gated on `S-BL.NODE-IDENTIFY-WIRE`) wires one shared `RouterIngest`
  across every `wireDiscoveryListener` goroutine — which nothing in today's API
  prevents — a node admitted to SVTN A can flood SVTN A's multicast group at the
  sustained rate cap indefinitely, consuming the entire process-wide 100 tokens/sec
  budget and starving legitimate discovery traffic for unrelated, co-hosted SVTN B on
  the same router.
- **Impact:** Availability degradation for discovery advertisements of co-hosted SVTNs
  sharing a router process — bounded (no memory exhaustion, no authentication bypass),
  but a real cross-tenant blast-radius expansion if the eventual wiring shares one
  instance.
- **Evidence:** `internal/discovery/discovery_wire.go:128-137` (bucket sizing, no SVTN
  key), `197-205` (single shared `rateLimiter`/`failureCounter` fields on
  `RouterIngest`), `302-311` (per-packet SVTN is read from the wire, not used to scope
  the limiter).
- **Proposed Mitigation:** Not a defect in this PR — `wireDiscoveryListener` and
  `RouterIngest.Ingest` are correctly implemented and unwired into any production
  daemon path today (no caller in `runRouter`'s lifecycle yet; verified by grep — zero
  non-test callers of `wireDiscoveryListener`). This is forward guidance for the Task 6
  follow-on story: either construct one `RouterIngest` per admitted SVTN (rate/failure
  state naturally isolated), or explicitly adjudicate a shared-budget-across-SVTNs
  trade-off the way the aggregate-vs-per-source rate-cap decision (SEC-DW-03(a)) was
  adjudicated for this story. **Accepted, not blocking.**

### SEC-102: `wireDiscoveryListener`'s ctx-cancellation closer goroutine can outlive the read loop on a non-context socket error
- **Severity:** LOW
- **CWE:** CWE-772 (Missing Release of Resource after Effective Lifetime)
- **OWASP:** A04:2021 – Insecure Design (resource lifecycle)
- **Attack Vector:** `wireDiscoveryListener` (`cmd/switchboard/discovery_wire.go:91-126`)
  spawns a second goroutine (`go func() { <-ctx.Done(); _ = conn.Close() }()`, line
  103-106) to unblock the blocking `ReadFromUDP` call on shutdown — a standard,
  necessary idiom since `net.Conn` has no context-aware read. On the function's *other*
  exit path — a real socket read error unrelated to `ctx` cancellation (line 111-117,
  e.g. an unexpected fatal errno) — the function returns and `defer wg.Done()` fires,
  but that closer goroutine remains parked on `<-ctx.Done()` until the caller's root
  context is eventually cancelled (in practice, process/daemon shutdown). No unbounded
  accumulation is possible from a single invocation (exactly one extra goroutine,
  self-terminating at shutdown), and this is not remotely triggerable per-packet — it
  requires an actual fatal socket-level error, not attacker-controlled datagram content.
- **Impact:** Negligible today. Worth a note only because a plausible future
  enhancement — an outer retry/reconnect wrapper around `wireDiscoveryListener` for
  resilience against transient socket errors — would leak one goroutine per retry cycle
  rather than one total, since each retry re-enters the function and spawns a fresh
  closer goroutine that never terminates until the *outer* ctx (not the retry loop)
  cancels.
- **Evidence:** `cmd/switchboard/discovery_wire.go:103-106` (spawn), `110-117`
  (non-ctx error return path with no explicit signal to the closer goroutine). No
  current caller exists (`wireDiscoveryListener` has zero non-test callers — router
  daemon wiring is explicitly deferred to a follow-on story per the file's own doc
  comment), so this is not reachable in the shipped binary today.
- **Proposed Mitigation:** No action required for this PR. If/when a retry wrapper is
  added around `wireDiscoveryListener`, either derive a child context per attempt (so
  each closer goroutine is scoped to that attempt's lifetime, not the daemon's) or
  signal the closer goroutine explicitly on the error return path (e.g. a
  `done chan struct{}` closed via `defer`). **Accepted, not blocking.**

### Accepted-as-adjudicated (re-verified, not re-litigated)
- Hop-2 zero HMACTag, TCP-connection-as-trust-boundary
  (`cmd/switchboard/discovery_relay_wire.go:94-97`) — confirmed present and matches the
  DRAIN-wire precedent (`S-7.04-FU-DRAIN-WIRE`, PR #120). `Discovery.IngestRelayAdvertisement`
  (`internal/discovery/discovery.go:522`) has zero non-test production callers today,
  consistent with Task 6 being gated.
- Discovery HMAC key derived from admitted pubkey, not on-wire SVTNID — confirmed
  correct (see above), F-DWIP1-001 fix verified both by code inspection and by
  dedicated regression test.
- Sequence-based, non-per-source replay gate and aggregate (not per-source) rate cap —
  confirmed as implemented; also independently re-checked the epoch-qualified `uint64`
  sequence math (`nextSequence`, `internal/discovery/discovery.go:238-241`) for
  overflow/underflow: no wraparound is reachable within the epoch range in use
  (2038-safe until 2106), and the widened low-32-bit counter would only wrap after ~4.3
  billion `Advertise` calls from a single unrestarted instance — not a practical concern
  and not a new finding.
- Lookup-miss / HMAC-mismatch outcome unification — confirmed identical
  `RouterIngestDecision{}, ErrInvalidHMACTag` return, including the documented (and
  correctly-labeled-as-accepted) processing-time asymmetry from the `ok &&`
  short-circuit.
- `FailureCounter` visibility-only, never gates admission — confirmed;
  `RecordHMACFailure` has no return value consulted for gating anywhere in `Ingest`.
- AC-017/AC-018 fan-out dispatch and dispatch rate cap — absent, as expected; correctly
  scoped out to `S-BL.NODE-IDENTIFY-WIRE`.

---

## Follow-up (found by the subsequent PR-reviewer pass, not this dispatch)

The independent `pr-reviewer` fresh-eyes pass (see `pr-review.md` in this directory)
surfaced one additional LOW finding this dispatch did not cover — **F2: SEC-DW-04/AC-013
HMAC-failure visibility logging is defeatable by NodeAddr rotation** (per-source
`FailureCounter` keyed on an attacker-controlled, unauthenticated `NodeAddr` never
crosses its per-source alert threshold under a NodeAddr-rotating flood). Accepted as
forward guidance alongside SEC-101/SEC-102 — same non-blocking rationale (zero
production callers for `RouterIngest` today). See `pr-review.md` for full detail and
`pr-description.md`'s "PR Review" section for the disposition table.

---

**Files reviewed:** `internal/discovery/discovery.go`, `discovery_wire.go`,
`multicast_ttl.go`; `internal/hmac/hmac.go`; `internal/routing/advertisement_hmac.go`,
`discovery_failure_counter.go`; `internal/admission/failure_counter.go` (pre-existing,
reused — read for context, not part of the diff); `cmd/switchboard/discovery_wire.go`,
`discovery_relay_wire.go`; `internal/testenv/multicast_loopback.go`. Test files were
read to confirm claimed properties are empirically asserted, not just documented, but
are not independently in scope for CWE findings.
