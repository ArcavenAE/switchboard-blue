# PR Review — S-BL.DISCOVERY-WIRE (PR #123)

**Reviewer:** pr-reviewer (fresh-eyes, cognitive-diversity pass)
**PR:** ArcavenAE/switchboard-blue#123 — feat(discovery): discovery wire boundary (Tasks 1–5, AC-001..016)
**Branch tip reviewed:** `61767d1` (code HEAD `501db03`)
**Posted review:** https://github.com/ArcavenAE/switchboard-blue/pull/123#pullrequestreview-4701547369 (COMMENTED)
**Review type:** `gh pr review --comment` (single-identity project — arcavenai authors AND reviews; GitHub forbids self-APPROVE/REQUEST_CHANGES, so `reviewDecision` stays empty structurally per drbothen/vsdd-factory#626. COMMENTED review + dispositioned findings + CI-green is the convergence record here, NOT a formal APPROVE. `gh pr review` was used — never `gh pr comment`.)

---

## Verdict: HAS_FINDINGS — 1 BLOCKING, 2 forward-guidance (LOW / NIT)

The production crypto/wire surface is genuinely merge-quality and the F-DWIP1-001
fix is real. The blocker is not in the shipped ingest/dispatch logic — it is that
**CI is red** and the failing tests encode a real portability defect. Project
merge gate is CI-green + findings dispositioned, so this must be resolved first.

| # | Severity | Category | Summary |
|---|----------|----------|---------|
| F1 | **BLOCKING** | ci / test-portability | CI Quality Gate red — two new tests fail on the Linux runner; `MulticastLoopbackInterface` returns `lo` without validating the multicast flag it promises. |
| F2 | LOW (forward-guidance) | observability / security | SEC-DW-04/AC-013 HMAC-failure alert is defeatable by NodeAddr rotation (per-source counter on a spoofable key). |
| F3 | NIT (forward-guidance) | concurrency | `Advertise` samples `Sequence` outside the lock that orders it — concurrent callers can produce out-of-order sequences. |
| — | Informational | ci / advisory | StepSecurity Harden-Runner red in `audit` mode (advisory) — likely the new multicast-UDP / `net.Interfaces()` behavior. |

---

## 🔴 BLOCKING — F1: CI Quality Gate is red; two new tests fail on the Linux runner

Quality Gate run
[29394177062](https://github.com/ArcavenAE/switchboard-blue/actions/runs/29394177062/job/87283836521)
fails at PR HEAD `61767d1`:

```
--- FAIL: TestMulticastLoopbackInterface_ResolvesLoopback (0.00s)
    multicast_loopback_test.go:40: MulticastLoopbackInterface returned "lo", which does not support multicast
FAIL  github.com/arcavenae/switchboard/internal/testenv

--- FAIL: TestDiscovery_Advertise_WriteToMulticast_TTL1_NoGroupJoin (4.00s)
    discovery_test.go:2019: AC-003 postcondition 1: no UDP datagram received on the SVTN-derived multicast group — Advertise did not send real UDP
FAIL  github.com/arcavenae/switchboard/internal/discovery
```

**Root cause** — `internal/testenv/multicast_loopback.go`, `MulticastLoopbackInterface`:

```go
name := loopbackInterfaceName()       // "lo" on Linux, "lo0" on macOS
iface, err := net.InterfaceByName(name)
if err == nil {
    return iface                      // ← returns WITHOUT checking the multicast flag
}
// fallback scan below DOES check FlagLoopback && FlagMulticast,
// but only runs when InterfaceByName errored
```

The fast path returns the named interface as soon as lookup succeeds, without
validating the loopback+multicast flags the docstring ("the resolved interface
actually supports both loopback and multicast") promises. On Linux, `lo` exists
but lacks the `MULTICAST` flag by default, so a non-multicast interface is
returned — breaking the testenv self-test (asserts the flag) and the AC-003 test
(joins a multicast group on that interface, then times out waiting for a datagram
that never loops back). Passes on the author's macOS because `lo0` carries
MULTICAST there. This is exactly the platform gap the helper's own cited **B13**
lesson warns about.

A flag check alone is necessary but not sufficient: on a stock GitHub Linux
runner NO interface is both loopback and multicast (`lo` = loopback-not-multicast;
`eth0` = multicast-not-loopback), so a corrected fast-path falling through to the
scan would then `t.Fatalf`. Fix options, in rough order of preference:
- Enable multicast on loopback in the Quality Gate workflow before the test step
  (`sudo ip link set dev lo multicast on`) **and** validate the flag on the
  fast path; or
- `t.Skip` the two tests when no loopback+multicast interface resolves (CI green,
  coverage gap explicit); or
- Restructure AC-003 to not depend on loopback multicast reception.

**Description-accuracy note:** the PR body's "`go test -race -count=1 ./...` — all
packages green, zero FAIL" (checked) is true on macOS but false on the Linux CI
runner. The "CI green" checklist item is (correctly) still unchecked, but the
green-tests claim above it reads as unconditional and should be qualified.

`TestRunRouter_DiscoveryListener_JoinsGroup_RouterModeOnly` passes on Linux —
it uses `net.ListenMulticastUDP("udp4", nil, ...)`, not the loopback helper. The
failure is contained to test infrastructure; production ingest/dispatch is
unaffected.

---

## 🟡 LOW / forward-guidance — F2: HMAC-failure visibility logging (SEC-DW-04 / AC-013) is defeatable by NodeAddr rotation

`RouterIngest.Ingest` records failures with
`ri.failureCounter.RecordHMACFailure(fmt.Sprintf("%x", nodeAddr))`
(`internal/discovery/discovery_wire.go:4233`), where `nodeAddr` is the declared,
unauthenticated, attacker-controlled NodeAddr from the datagram body. The reused
`admission.FailureCounter` fires **per-source** (confirmed:
`counts map[string][]time.Time`, alert at ≥ threshold per key). An attacker
flooding with bad-HMAC datagrams while rotating the declared NodeAddr keeps every
per-source count at 1 → the 5-per-60s threshold never crosses → **no operator
alert ever fires** under a sustained flood. The design deliberately made the rate
limiter aggregate-not-per-source (AC-012) to resist exactly this rotation, but did
not carry that reasoning to the failure-visibility logging. Net: the flood AC-013
exists to surface is the one an adversary silences for free.

Distinct axis from the security reviewer's SEC-101 (per-SVTN rate budget) and
SEC-102 (goroutine leak). Not reachable in the shipped binary (`RouterIngest` has
zero non-test production callers), so forward-guidance for the router-daemon
wiring story: drive the discovery-path threshold alert off an aggregate failure
count, or explicitly adjudicate the visibility gap.

---

## 🟡 NIT / forward-guidance — F3: `Advertise` samples the epoch-qualified Sequence outside the lock that orders it

`Discovery.Advertise` (`internal/discovery/discovery.go`) releases `d.mu` and then
calls `transmitAdvertisement` → `nextSequence()` (atomic increment). Concurrent
`Advertise` on one instance can assign a lower Sequence to a logically-later
advertisement, which the router's replay gate (AC-009) discards as stale
(self-heals on the next heartbeat). Unreachable today (single-goroutine `Run`
loop, no production caller). Fix: sample `nextSequence()` under `d.mu`, or document
`Advertise` as single-goroutine-per-instance.

---

## ℹ️ Informational — StepSecurity Harden-Runner red (advisory)

Runs in `egress-policy: audit` (non-blocking); generated "new alerts," most likely
the genuinely new outbound behavior this PR introduces (multicast UDP sends +
`net.Interfaces()` enumeration). Worth confirming every audited endpoint is
expected before merge; not a code defect and does not, by itself, gate.

---

## What I verified (no rubber-stamp)

- **F-DWIP1-001 fix is real and correct.** `Encode`/`Decode` derive the key via
  `routing.DeriveDiscoveryKey(nodeAdmissionPubkey, svtnID)`; the dead
  `advertisementKey(svtnID) = svtnID` is deleted;
  `TestDiscovery_EncodeThenRouterIngest_AcceptsRealAdmittedNode` exercises the
  real production `Encode` → `RouterIngest.Ingest` round trip (not the test's own
  HKDF re-impl), closing the gap that hid the regression.
- **HKDF domain separation is real.** Distinct `HKDFInfoDiscovery` label;
  `TestDeriveDiscoveryKey_DomainSeparatedFromFrameAuthKey` asserts the two keys
  differ for identical inputs; determinism checked.
- **Hop-1 ingest ordering is fail-closed.** rate-cap → length floor
  (`keySelectorMinRaw`=32) → length ceiling (32768) → fixed-offset key-selector
  extraction → key lookup + `crypto/hmac.Equal` → then body-min recheck +
  `DecodeSessionList` + replay gate under mutex. `lastSeen` mutated only after HMAC
  success, so unauthenticated packets can't grow it (bounded by admission). AC-006
  lookup-miss and tag-mismatch both return the identical `ErrInvalidHMACTag`; the
  `ok &&` short-circuit timing asymmetry is explicitly acknowledged/dispositioned.
- **Wire layouts consistent.** Hop-1 (`SVTNID|NodeAddr|Seq|count|sessions`) and
  hop-2 (`NodeAddr|Seq|count|sessions`, SVTNID in outer header) line up; the
  round-trip test feeds `assembleDiscoveryRelayFrame`'s real output into
  `IngestRelayAdvertisement`. Sequence widening uint32→uint64 threaded
  consistently (body min 26→34, frame min 34→42).
- **Scope boundary holds.** AC-017/AC-018/Task 6 genuinely absent — no partial
  fan-out/dispatch code; relay assembler + ingest are pure/near-pure primitives.
  "Zero HMAC on hop-2 intentional" matches the code (`assembleDiscoveryRelayFrame`
  leaves `HMACTag` zero — tested; `IngestRelayAdvertisement` verifies no per-frame
  HMAC). N-1 PayloadLen guard panics rather than silently wrapping the uint16.
- **Tests are behavioral, not trivia.** Constant-return guard on
  `MulticastAddrFor`; tamper-in-session-list detection for HMAC-covers-full-body;
  honest disclosure that TTL=1 is not wire-verified (no CMSG dependency); real
  reception oracle for AC-001 rather than a bare unerrored `Write`.

**Bottom line:** crypto/wire code is merge-quality; **F1 (CI red + the
`MulticastLoopbackInterface` portability defect) must be fixed before this can go
green and merge.** F2/F3 are non-blocking forward-guidance for the daemon-wiring
follow-on stories.
