---
artifact_id: pre-impl-spec-review-wave-3
document_type: spec-review
level: review
cycle: cycle-1
wave: 3
producer: spec-reviewer
model_family: Opus 4.7 (cognitive diversity vs. story-writer/PO/architect)
timestamp: 2026-06-25T00:00:00Z
scope: pre-implementation review of S-3.01..S-3.04 and supporting BCs/ADRs/VPs
inputs:
  - .factory/stories/S-3.01-tmux-control-pty-fallback.md
  - .factory/stories/S-3.02-session-attach-detach-fanout.md
  - .factory/stories/S-3.03-tier2-auth-readonly.md
  - .factory/stories/S-3.04-hmac-routeframe-wireup.md
  - .factory/specs/behavioral-contracts/ss-04/BC-2.04.001..006.md
  - .factory/specs/behavioral-contracts/ss-05/BC-2.05.003.md
  - .factory/specs/behavioral-contracts/ss-05/BC-2.05.008.md
  - .factory/specs/architecture/ARCH-01-core-services.md (ADR-010)
  - .factory/specs/architecture/ARCH-04-admission-security.md (ADR-009)
  - .factory/specs/architecture/ARCH-08-dependency-graph.md (§6.6)
  - .factory/specs/architecture/ARCH-09-purity-boundary-map.md
  - .factory/specs/verification-properties/VP-058.md
  - .factory/specs/prd-supplements/error-taxonomy.md
  - internal/routing/routing.go
  - internal/admission/admission.go (signatures)
---

# Wave 3 Pre-Implementation Spec Review

Constructive second-opinion review with cognitive diversity. Reports concerns
orthogonal to the parallel consistency-validator. Focus areas: AC testability,
BC postcondition tightness, complexity vs. count, hidden dependencies, edge-case
quality, ADR coherence, demo strategy, architecture fit.

---

## CRITICAL

### C-1: VP-058 proof-harness skeleton will not compile — calls non-existent `ks.Register(svtnID, nodeAddr)`

**Source:** `.factory/specs/verification-properties/VP-058.md` lines 88, 120.

The harness in VP-058 calls:

```go
ks.Register(svtnID, nodeAddr)   // VP-058.md L88 and L120
```

`admission.AdmittedKeySet` exposes **`RegisterKey(svtnID [16]byte, pubkey ed25519.PublicKey, role KeyRole)`** — not `Register`, and the parameters are `(svtnID, pubkey, role)`, not `(svtnID, nodeAddr)`. Furthermore, `RegisterKey` alone does **not** make a node return `IsAdmitted(...)==true`; the node must complete `AdmitNode` (challenge-response). See `internal/admission/admission.go:155`, `:233`, plus `internal/admission/reauth_test.go:50-65` which shows the real `RegisterKey` + `AdmitNode` pattern.

Consequence: a test-writer who ports the VP-058 skeleton verbatim into `internal/routing/routing_test.go` (S-3.04 task #2 says "VP-058 harness skeleton provided in VP-058.md") gets an immediate compile failure. They will then either (a) fix it themselves and silently diverge from the spec, or (b) escalate. Either way the canonical reference is broken.

**Action:** Fix VP-058.md to use the real admission API — `ks.RegisterKey(svtnID, pubkey, admission.RoleAccess)` followed by an `AdmitNode` call (or use a freshly-built test helper that does both). S-3.04 should also be amended to flag this: AC-003 cannot be exercised end-to-end without first admitting the node properly.

---

### C-2: ADR-009 contradicts itself on whether HMAC verification or admitted-set check happens first when the source is unknown

**Source:** `.factory/specs/architecture/ARCH-04-admission-security.md:249-298`.

ADR-009 declares HMAC verification is "the first operation after outer header parsing, **before any admitted-set lookup**" (L251-253). But then in the very same ADR at "Key lookup in RouteFrame" (L273-277):

> *"If the src_node_addr is not in the admitted set at key-lookup time, the frame is treated as HMAC-unverifiable and dropped (this merges the admitted-set check with the HMAC key lookup in one `RLock` acquisition — permissible optimization)."*

That is the opposite of what BC-2.05.008 PC-3 says ("HMAC verification occurs BEFORE the admitted-set check"). BC-2.05.008 PC-4 says key unavailability comes from the **forwarding table** (no entry for `(SVTNID, SrcAddr)`), not from the admitted set. S-3.04 task #5 also says "Retrieve forwarding-table entry for `(hdr.SVTNID, hdr.SrcAddr)` first" — that matches the BC, not the contradictory ADR paragraph.

A test-writer reading just ADR-009 may decide the admitted-set lookup is fine to do first (since it is "merged with the key lookup"). The whole point of VP-058 is to forbid that.

Additionally, ADR-009 says the key lives in `admitted_key_set[svtn_id][src_node_addr].frame_auth_key`, but the actual code on `develop` (`internal/routing/routing.go:29-37`, `:65-83`) stores `FrameAuthKey` on the **`ForwardingEntry`** in the router's forwarding table, not on `admission.AdmittedKey`. The story (correctly) reads `entry.FrameAuthKey`. ADR-009 still references the wrong data structure.

**Action:** Rewrite the "Key lookup in RouteFrame" paragraph in ADR-009 to say: (a) the auth key is read from the **router forwarding table** entry for `(svtnID, srcAddr)`, not the admitted-set; (b) if no forwarding entry exists, the frame is dropped as unverifiable with `ErrHMACVerificationFailed`; (c) the admitted-set check is a **separate** subsequent step. Delete the "merges the admitted-set check with the HMAC key lookup" sentence — it's wrong and dangerous.

---

### C-3: ADR-009 specifies an entirely different `verifyFrameHMAC` signature from the one already on `develop`

**Source:** ADR-009 L287-294.

ADR-009 prescribes:

```go
func verifyFrameHMAC(header *frame.OuterHeader, rest []byte, key []byte) error
```

The actual code at `internal/routing/routing.go:154` is:

```go
func verifyFrameHMAC(hdr frame.OuterHeader, payload []byte, authKey [hmac.KeySize]byte) bool
```

Three differences: pointer-vs-value receiver style, parameter naming (`rest` vs `payload`), and return type (`error` vs `bool`). ADR-009 also says the function returns "`E-ADM-016` on mismatch" — but `E-ADM-016` is a wire-format error taxonomy code, not a Go sentinel. The story is correctly written against the existing `bool`-returning signature; ADR-009 is wrong.

This is a CRITICAL because a code reviewer or future maintainer reading ADR-009 will believe the function returns an error and may file a PR to "fix" the bool signature, breaking the existing internal test (`routing_internal_test.go:TestVerifyFrameHMAC_RejectsWrongTag`) and the H-1 tautology guard.

**Action:** Update ADR-009 §"Implementation note (S-3.04)" to reflect the actual `(hdr frame.OuterHeader, payload []byte, authKey [hmac.KeySize]byte) bool` signature, and explain that `RouteFrame` (the caller) constructs `ErrHMACVerificationFailed` and emits the E-ADM-016 log entry — the helper is a pure predicate.

---

## HIGH

### H-1: ADR-010 says PTY fallback only at initial connect — but BC-2.04.002 EC-003 says PTY fallback DOES happen mid-session after 3 failed reconnects

**Source:** `.factory/specs/behavioral-contracts/ss-04/BC-2.04.002.md` EC-003 vs. ADR-010 in ARCH-01.md L141-145.

BC-2.04.002 EC-003 is unambiguous:

> *"tmux control mode drops after successful start (mid-operation) — Access node attempts control mode reconnect; if reconnect fails after 3 attempts, switches to PTY proxy mode for existing sessions."*

ADR-010 contradicts this:

> *"Fallback is triggered only on initial `TmuxControlMode.Attach` failure. It is NOT triggered if the control mode connection drops mid-session."*

S-3.01 EC-002 sides with ADR-010 ("if reconnect fails, NOT an automatic PTY fallback (fallback is only at initial connect per ADR-010) — session marked unavailable"). So the story implements ADR-010 behavior but BC-2.04.002 EC-003 still mandates the mid-session 3-retry fallback. A holdout evaluator reading BC-2.04.002 EC-003 as the authoritative behavior will mark the implementation as non-conforming.

Note BC-2.04.001 EC-002 says yet a third thing: "If reconnect fails within timeout, falls back to PTY proxy mode."

**Action:** Resolve to one truth. Recommended: keep the ADR-010 / S-3.01 EC-002 behavior (no mid-session fallback) for MVP simplicity, then **amend BC-2.04.002 EC-003 and BC-2.04.001 EC-002** to say "Reconnect attempted; session marked unavailable on failure; PTY fallback NOT triggered mid-session per ADR-010." Without this fix, the BC and the ADR are still inconsistent post-implementation.

---

### H-2: S-3.01 AC-001..AC-004 require a real tmux server in the test environment — but the CI matrix is silent on this

**Source:** S-3.01 Task #2: *"Write failing tests for AC-001 through AC-007 (requires tmux in test environment for AC-001..AC-004)"*.

S-3.01 has 4 ACs (out of 7) that need a real tmux process. The story doesn't declare:

- Whether the CI image already has tmux (it currently does not by default in `golang:1.25.4` base).
- Whether these tests should be `t.Skip()`-ed when tmux is absent (and what the gate criterion is — `os/exec.LookPath("tmux") == nil`?).
- Whether the integration tests live behind a `//go:build integration` tag or always run.

Without this declared, the test-writer is going to either (a) write tests that fail in CI for environmental reasons (false-positive flakes), or (b) write tests that silently skip and provide zero CI coverage. The Red Gate gate will pass either way, masking the real test outcome.

The same problem applies to AC-007 (PTY device unavailable). A reliable way to simulate "no PTY available" inside a containerized CI test is non-trivial (chmod the device? mock at the syscall level? skip when running rootless?). The story doesn't address it.

**Action:** S-3.01 should add an explicit "Test Environment Requirements" section: (a) lefthook or CI must install tmux ≥3.0 before running `just test`; (b) AC-001..AC-004 tests must check `exec.LookPath("tmux")` and `t.Skip(...)` with an explicit message if absent (and the CI must fail if too many are skipped); (c) AC-007 (PTY unavailable) should be implemented via dependency injection of a `ptyOpener func() (*os.File, error)` interface so the unit test can substitute a fake that returns `E-SYS-001`.

---

### H-3: S-3.01 AC-004's "at least 99%" output-event delivery is not deterministically testable

**Source:** S-3.01 AC-004 and VP-031.

> *"Session output events from control mode (`%output`) feed the downstream half-channel; **at least 99% of events emitted by the tmux control mode session are delivered** to the downstream stream (integration test with a real tmux session)."*

A test writer cannot satisfy this acceptance criterion deterministically:

1. What is the denominator? "Events emitted by the tmux session" depends on what the test does in the session (run a script? cat a file? interactive keystrokes?).
2. What sample size is needed to compute "99%" with statistical confidence? Without a bound on the test duration, this is unbounded.
3. Is this 99% over a window or cumulative since startup?
4. The test could "pass" with 99% deliberately by running a 100-event script and dropping 1.

This AC effectively defers measurement criteria to the test-writer's discretion. That's not testable.

**Action:** Reframe AC-004 as a **completeness** property, not a percentage. Suggested rewrite:

> AC-004: When the control mode emits a sequence of `%output` events for a given session, every event in the sequence appears at the downstream `HalfChannel.Enqueue` call in the order received (no reordering, no loss under normal operation). The test sends 100 `%output` lines via a real tmux session and verifies all 100 arrive at the downstream sink with matching content and order.

The 99% NFR target then becomes a performance/loss budget in NFR-catalog (where it belongs), not a P0 acceptance criterion.

---

### H-4: S-3.02 AC-007 ("keystrokes serialized — no interleaving under concurrent sends") is conflated with race-freedom and not testable as written

**Source:** S-3.02 AC-007.

"Serialized — no keystroke interleaving or data corruption" is two separate properties:

1. **Atomicity per write**: each console's `Send([]byte{'l','s'})` arrives at tmux as `ls`, not `ls` interleaved with another console's bytes (e.g., `lsls` is fine but `lass` from `ls` + `as` is not).
2. **Race-freedom**: no data race detected by `go test -race`.

(1) is the BC's real concern (BC-2.04.006 invariant 3 says "All full-access console keystrokes are serialized by the access node before forwarding to tmux"). (2) is a CLAUDE.md baseline expectation.

But the AC says "no keystroke interleaving OR data corruption" — these are different. "Data corruption" usually implies torn bytes or memory unsafety, which is what `-race` catches. "No interleaving at message boundary" is what (1) catches. A test for (1) requires the test to send multiple multi-byte messages from concurrent goroutines and verify each arrives at tmux as a contiguous chunk; a test for (2) just runs the existing test under `-race`.

A test-writer is going to pick one and call AC-007 satisfied — and that one is unlikely to be (1), because (1) requires defining "keystroke boundary" in the protocol (which is implicit — there's no documented framing of multi-byte input bursts).

**Action:** Split AC-007 into two ACs: AC-007a (atomicity per upstream message: a single `Console.Send(buf)` arrives at tmux as a contiguous chunk, never interleaved with another console's chunk) + AC-007b (passes under `go test -race`). Also explicitly define what a "keystroke" is at the protocol layer — bytes-per-Send-call, single ChannelFrame, or per-RUNE?

---

### H-5: S-3.02 AC-008 (crash detection) hides a polling vs. push-detection design choice

**Source:** S-3.02 AC-008: *"When a console's channel closes unexpectedly (process crash without explicit detach), the access node detects the closed channel and evicts the console from the fan-out set."*

How does the access node detect this? Two designs:

1. **Push-detect** — write to the channel and observe the write error (only fires on next fan-out delivery).
2. **Keepalive timeout** — periodic empty-tick frames; absence triggers eviction.

BC-2.04.004 EC-002 says "Access node detects channel closure on next keepalive timeout. Session released." — that's design (2). S-3.02 EC-005 says "Access node detects closed channel on **next delivery attempt**; evicts console" — that's design (1).

These have very different observable behavior:

- Push-detect: if the session is idle (no downstream frames flowing), a crashed console is never evicted.
- Keepalive timeout: requires a clock-driven loop and a configurable timeout. Adds non-determinism to tests.

The story tasks do not say which to implement. The test (`TestSession_CrashDetach_EvictsFromFanOut`) doesn't say how long to wait or whether to inject a downstream frame to trigger detection.

**Action:** Story should pick one — recommend push-detect for the MVP (AC-008 reformulated: "next downstream delivery to a closed channel triggers eviction") and explicitly defer keepalive-timeout-based detection to Wave 4 / NFR-catalog. The BC should also be reconciled with the story decision.

---

### H-6: S-3.03 depends_on declares S-3.02 + S-2.02 but actually depends on a stable `internal/session` interface that doesn't exist yet at start

**Source:** S-3.03 `depends_on: [S-3.02, S-2.02]`.

S-3.03 says it adds `internal/session/auth.go` with `SessionAuth` type. But:

- S-3.02 creates `Session` struct and `ConsoleSet` in `internal/session/session.go` and `fanout.go`. S-3.03 then needs to **wire** SessionAuth into the existing attach path (AC-005 "any upstream keystroke frame from a read-only console is rejected by the access node" — that rejection has to happen in the upstream-receive path of Session, which is owned by S-3.02).
- S-3.03's AC-005 (`TestReadOnlyConsole_UpstreamRejected_DownstreamContinues`) cannot pass without S-3.02 having defined the upstream-receive code path in a way that allows a policy hook (interface, callback, or in-package coupling).

S-3.02 has zero awareness of this hook — it does not declare any extension point for Tier-2 enforcement. S-3.03 will then either need to (a) edit S-3.02's code (breaking the "S-3.03 modifies only `internal/session/auth.go`" implicit promise of the File Structure Requirements table), or (b) S-3.02 must add a no-op `SessionAuth` interface in anticipation of S-3.03.

This is a hidden dependency. The two stories were decomposed as if independent.

**Action:** Add a task to S-3.02: *"Introduce an `Authorizer` interface on `Session` with no-op default; S-3.03 will provide the real implementation."* Or merge S-3.03's upstream-rejection AC (AC-005, AC-006) into S-3.02. As written, S-3.03 will silently mutate the S-3.02 surface.

---

### H-7: S-3.04 AC-004 (no forwarding entry → ErrHMACVerificationFailed) conflicts with BC-2.05.008 EC-005 + SVTNRoute current contract

**Source:** S-3.04 AC-004 + BC-2.05.008 PC-4 vs. current `routing.go:131-133`.

BC-2.05.008 PC-4 says: "Auth key unavailable (no forwarding-table entry for src) → `RouteFrame` returns `ErrHMACVerificationFailed`."

But `SVTNRoute` already returns `ErrNoForwardingEntry` when *DstAddr* is missing from the forwarding table (`routing.go:131`). The two errors are about different lookups (src vs. dst), but the wire-up needs to be clear:

- `RouteFrame` should do its OWN forwarding-table lookup on `(SVTNID, SrcAddr)` to get the auth key (a NEW lookup, not the existing DstAddr one).
- If the SrcAddr lookup misses → `ErrHMACVerificationFailed`.
- If the DstAddr lookup misses inside `SVTNRoute` → `ErrNoForwardingEntry`.

The story task #5 says "Retrieve forwarding-table entry for `(hdr.SVTNID, hdr.SrcAddr)` first" — that's a SrcAddr lookup. But the existing `Router.forwardingTable` is keyed by `[svtnID][dstAddr]` (`routing.go:51`, and `RegisterForwardingEntry` registers by `nodeAddr` which functions as a destination). There is no current SrcAddr index.

So either (a) `RegisterForwardingEntry` is intended to register both source and destination identity for a node (one entry per node in the SVTN, addressable both ways — that's a reasonable interpretation), or (b) the story needs to add a separate source-keyed index.

If (a) is the intent, S-3.04 should make it explicit: when a node admits to an SVTN, its `ForwardingEntry` covers both "this node is a destination at NodeAddr" AND "this node is a source at NodeAddr". The test for AC-004 needs to set up a forwarding entry where `hdr.SrcAddr == entry.NodeAddr`.

**Action:** Add to S-3.04 task #5: *"Note: each `ForwardingEntry` registered for a node covers both directions — used as a destination for forwarding and as a source for HMAC key lookup. The SrcAddr lookup uses the same `forwardingTable[svtnID][addr]` map keyed by the node's address."* And rename `RegisterForwardingEntry`'s `nodeAddr` parameter doc to clarify this dual role.

---

### H-8: Story-complexity estimates: S-3.01 is **under-pointed** at 8, S-3.04 is correctly 3 (commendation), but S-3.03 hides Tier-2 design complexity at 5

**S-3.01 (8 pts):**
- Tmux control mode wiring (parsing %output / %session-created / %session-closed events from a `tmux -C` subprocess) — significant integration work, fragile parser
- PTY fallback (allocating PTY, spawning shell, proxying I/O) — separate effectful subsystem
- 7 ACs, 5 ECs, 4 new files in 2 new packages
- Requires environmental tmux + PTY simulation in tests

Comparable Wave-2 work (S-2.02 admission re-handshake) was estimated at 13. S-3.01 has more surface area and at least equal risk. **Recommend rebaseline to 13 pts** or split into S-3.01a (tmux control mode) + S-3.01b (PTY fallback).

**S-3.03 (5 pts):**
- New package surface (`SessionAuth`, `RegisterKey`, `Authorize`, per-session list with mutex)
- VP-012 code-audit (assertion that `internal/routing/routing.go` has no per-session auth state — easy but require a CI grep guard)
- Wire-up into S-3.02's session upstream-receive path (per H-6 above, this is an architectural change to S-3.02 deliverables)
- 6 ACs, 6 ECs

The code-audit assertion AC-003 is unusual — it's a meta-property checked via grep, not a function call. Most implementers don't know how to "test" this. The Tier-2 surface itself is moderately complex.

**Recommend bump to 8 pts**, particularly because the AC-005/AC-006 upstream rejection requires touching S-3.02's code.

**S-3.04 (3 pts):** COMMENDATION-worthy. This is exactly the right scope for a 3-point story: one function (`RouteFrame`) gets ~15 lines modified, one sentinel added, one nolint removed. The 5 ACs are surgical. If H-1 dependencies on S-2.01 ordering and the C-1 VP-058 harness fix are addressed first, this story should ship in a half-day.

**Action:** Re-point S-3.01 to 13 (or split). Re-point S-3.03 to 8.

---

## MEDIUM

### M-1: BC-2.04.001 PC-3 ("automatically discovered and published") leaves the discovery latency open

**Source:** BC-2.04.001 PC-3.

> *"New tmux sessions created after startup are automatically discovered and published."*

No latency bound. The canonical test vector hints at "within 1 tick", but the BC text itself says nothing. A test like "create a session, then immediately list — expect to see it" could pass or fail based on timing. Without a defined upper bound, the test is racy.

**Action:** Tighten PC-3: "...discovered and published within one tick interval (typically 100ms)." Add an EC for "session created during a control mode reconnect — discovery delayed until reconnect completes."

---

### M-2: BC-2.04.002 PC-3 prescribes a verbatim log message — fragile, but no language about test matching strategy

**Source:** BC-2.04.002 PC-3.

> *"A log entry is written: 'tmux control mode unavailable; using PTY proxy mode. Functionality limited: no structured session metadata, no content-type detection.'"*

The story (S-3.01 AC-006) tests for this exact string. But the BC doesn't say whether:
- The exact phrase must be substring-matched, regex-matched, or contains key tokens.
- Future log-format changes (e.g., structured logging with key/value pairs) need a BC revision.

This is fragile. A test-writer who imports a log helper that auto-prefixes timestamps will get a non-matching string and either edit the BC (silent drift) or weaken the test.

**Action:** Reframe PC-3 with a structured-log requirement: "Log entry contains the structured fields `event=fallback` and `reason=tmux-control-mode-unavailable` and `mode=pty-proxy`." Then test for fields, not the verbatim sentence. The English sentence becomes documentation, not a contract.

---

### M-3: BC-2.04.003 EC-003 (access node unreachable → E-NET-005 "router returns ... ") doesn't say who returns it

**Source:** BC-2.04.003 EC-003.

> *"Session exists but access node is unreachable — Router returns E-NET-005 'access node unreachable'. Session may appear in list (stale advertisement)."*

But the architecture says the router never knows session names — it just forwards frames. How does the router know "the access node is unreachable for session X"? The answer is probably: at the SVTN level, if no frames flow to the access node's NodeAddr for some interval. But that's a different error semantically. S-3.02 AC-003 maps `E-NET-005` to attach but doesn't say WHICH component raises it. Is this from `internal/session` (after a transport timeout)? From `internal/routing`? From the console-side timeout?

Likely the **console** sees the timeout. The BC text "Router returns" is misleading.

**Action:** Clarify BC-2.04.003 EC-003: "The console's attach request times out (default 5s) without a CHALLENGE response from the access node; the console returns E-NET-005 to the caller. The router itself is not aware of session-level reachability."

---

### M-4: S-3.02 EC-007 ("all consoles detach → session continues") is not actually an edge case — it's the AC-004 happy path repeated

**Source:** S-3.02 EC-007.

The EC says: *"All consoles detach — Session continues on access node; no session teardown."*

AC-004 says: *"`Session.Detach(consoleKey)` closes the console's channel cleanly. The tmux session on the access node continues running unchanged."*

EC-007 is just AC-004 applied N times. Not an edge case in the sense of "rare / error path / exceptional condition." A true edge case here would be: "All consoles detach, then the LAST console is also a control-node session source — does the session get GC-ed eventually? After what timeout?" Or: "All consoles detach while a downstream frame is in flight to one of them — is the frame dropped without error?"

**Action:** Replace EC-007 with: "Downstream frame is being delivered when the last full-access console detaches — frame is dropped on the closing channel without surfacing an error to other observers" (or whatever the actual design is). Or remove EC-007 entirely — it adds no test coverage.

---

### M-5: S-3.03 AC-003 (code-audit assertion) is unusual and the test name suggests a real test — but it's a grep

**Source:** S-3.03 AC-003.

> *"The router (`internal/routing`) has no per-session authorization data structure. ... VP-012 code-audit verifies this: no import of per-session auth state in `routing.go`. **Test:** `TestSessionAuth_RouterHasNoTier2State`"*

A test function name with `Test` prefix is Go's signal for a runtime test. But the AC says it's a grep / `go vet` audit. A real `func TestSessionAuth_RouterHasNoTier2State(t *testing.T)` is achievable — it can use `go/ast` to walk `internal/routing/*.go` and assert no identifiers matching `SessionAuth*` exist. That's still a "test" but it's a static one.

If the implementer treats this as a vague "check the code" task instead of a real `go test` invocation, it doesn't make it into CI and the property regresses silently.

**Action:** Spell out the test mechanism: "Test uses `go/parser.ParseDir` to walk all .go files in `internal/routing`, asserts no identifier matches `^SessionAuth` or `^sessionAuth` and no import of any package path with a name ending in `session/auth`." Make it a real test, not a manual checklist.

---

### M-6: S-3.04 AC-002 says "E-ADM-016 log entry is written" but the test method (`TestRouteFrame_InvalidHMAC_ReturnsErrHMACVerificationFailed`) name doesn't reveal that log check

**Source:** S-3.04 AC-002.

> *"`RouteFrame` with an invalid HMAC tag (any tag mismatch) returns `routing.ErrHMACVerificationFailed` immediately. ... E-ADM-016 log entry is written."*

Two assertions, one test. Verifying the log entry requires either:
- A test logger injected into the Router (none exists today)
- Capture of stderr / `slog` output via a buffer
- A test-only hook (e.g., an injectable `Logger` interface)

The story has no task for adding a logger injection point. The current `Router` struct (`routing.go:48`) has no logger field. Without one, the log assertion is impossible.

**Action:** Add a task to S-3.04: *"Add a `logger *slog.Logger` field to `Router` (defaulting to `slog.Default()`); accept it as an optional parameter via a `WithLogger(*slog.Logger)` option on `NewRouter`. AC-002 test injects a buffer-backed logger and asserts the E-ADM-016 entry."* Alternatively, defer the log-assertion part of AC-002 and only test the sentinel.

---

### M-7: BC-2.05.008 EC-006 (failure counter on repeated HMAC failures) is referenced but no AC covers it in S-3.04

**Source:** BC-2.05.008 EC-006 + S-3.04 ACs.

BC-2.05.008 EC-006 says: *"Repeated HMAC failures (≥5 in 60s) from same `src_addr` — Failure counter incremented per existing alert logic (BC-2.05.005 postcondition 3); admission alert triggered."*

S-3.04 has no AC for this. It's not in the EC table either. So if a test-writer reads only S-3.04, they implement HMAC verification but no counter. The alert logic from BC-2.05.005 PC-3 was supposed to fire on `internal/hmac` failures — does it apply to `internal/routing` `ErrHMACVerificationFailed` too? Unclear.

**Action:** Either explicitly defer EC-006 to a separate story (and add a TODO to the BC marking it as such), or add an AC-006 to S-3.04 that wires the counter. Recommend defer — the counter is observability, not safety.

---

### M-8: ARCH-08 §6.6 declares `internal/session` imports `{frame, admission}` — but no story imports admission explicitly

**Source:** ARCH-08.md §6.6 + stories.

ARCH-08 §6.6 says session imports `{frame, admission}` "so SessionAuth (S-3.03 Tier-2) can verify against `admission.AdmittedKeySet`." But S-3.03 AC-001..AC-004 describe a SessionAuth that has its OWN per-session authorization list (not reused from `admission.AdmittedKeySet`). The two are explicitly **different** — Tier 1 (admission) vs Tier 2 (session) — and the BC says "Tier 1 and Tier 2 keys may be the same keypair, but the authorization scopes are independent."

So if SessionAuth maintains its own list, why does `internal/session` need to import `admission`? The architecture justifies the import edge with a claim ("SessionAuth verifies against AdmittedKeySet") that the BCs and story explicitly contradict.

Possible legitimate uses for the import: reading the `KeyRole` enum, reading the `AdmittedKey` struct as a convenience. But neither is required by the stories.

**Action:** Either remove the `admission` import from §6.6 (session → `{frame}` only) — which means SessionAuth lives entirely separate, OR add a justified story task: "SessionAuth.Authorize cross-checks that the console is still in `AdmittedKeySet.IsAdmitted` before granting Tier-2 — defense in depth." Either way, make the import justification match the implementation.

---

## LOW

### L-1: BC postcondition language uses "should" / "may" in places where MUST is intended

- BC-2.04.003 PC-5: *"The console **displays** the current terminal output state (implementation: **may** request a full screen refresh from tmux on attach)."* — "displays" is observable; "may request" is the implementation choice. The MUST is on display. Recommend: "MUST display the current terminal output; MAY request a screen refresh."
- BC-2.04.004 PC-1: "...closed **cleanly** (FIN exchange or equivalent)" — what counts as "equivalent" is undefined. Recommend: "MUST close such that subsequent `Send` returns `io.ErrClosedPipe` or `net.ErrClosed`."
- BC-2.04.006 PC-5: "There is no artificial limit ... at the protocol level (implementation **may** impose a practical limit; architecture decision)." — Where is the architecture decision documented? Implementation latitude this wide makes the property unverifiable. Recommend: "Implementation MAY impose a per-session limit of N consoles where N is configurable; default ∞."

### L-2: S-3.01 EC table EC-004 ("tmux exists but old version") doesn't specify HOW old

What's the minimum tmux version with `-CC` support? tmux 1.8? 2.0? The test for EC-004 needs to know to construct a fake. Without a pinned minimum, the EC isn't testable.

**Action:** State the minimum: "tmux ≥3.0 has reliable control mode support." Test EC-004 by injecting a fake tmux that returns version 1.6 from `tmux -V`.

### L-3: S-3.02 EC-006 duplicates EC-003 in BC-2.04.006

Both say "two full-access consoles send keystrokes simultaneously → serialized; no crash or corruption." This is fine, but listing the same behavior in both an EC and an AC (AC-007) creates redundancy. Recommend: drop EC-006 from S-3.02 since AC-007 already covers it; or differentiate (e.g., EC-006 = "10 consoles, not 2").

### L-4: S-3.04 narrative says "before the admitted-set check and before SVTNRoute" but the order in code is `RouteFrame` → `IsAdmitted` → `SVTNRoute`

Current `RouteFrame` (`routing.go:97-104`) does NOT call `SVTNRoute` after `IsAdmitted` — it calls it inside the same `RouteFrame` body. After S-3.04, the order will be: `RouteFrame` → forwarding-table-lookup → `verifyFrameHMAC` → `IsAdmitted` → `SVTNRoute`. The narrative is correct but a reader might think `SVTNRoute` is a sibling-step rather than the tail call. Minor wording polish.

### L-5: S-3.03 EC-005 ("scope is SVTN-wide read-only for this key") implies a behavior not in any BC

BC-2.04.005 EC-003 says read-only scope IS per-SVTN. S-3.03 EC-005 restates it. But neither says how this scope is configured — is it a flag on the SessionAuth registration? An implicit policy? S-3.03 task #6 implements `RegisterKey(sessionName, consoleKey, role)` — that's PER-SESSION, not per-SVTN. So the SVTN-wide scope claim has no implementation hook in the story.

**Action:** Either delete EC-005 (it's aspirational, not implemented) or add a task: "RegisterKey supports `sessionName=*` to register a key as read-only across all sessions on this access node." Recommend delete — over-scoping for MVP.

### L-6: Token budget estimates are remarkably consistent at ~2.5%

S-3.01 = 4,900 (~2.5%); S-3.02 = 5,000 (~2.5%); S-3.03 = 4,100 (~2.1%); S-3.04 = 4,500 (~2.3%). These are suspiciously uniform given the very different surface areas. The story-writer likely used a template number. Not actually a problem — just noting for discipline.

---

## COMMENDATIONS

### CO-1: S-3.04 is a model of surgical scope discipline

- One file modified, one new sentinel, one nolint removed
- 5 ACs trace cleanly to BC-2.05.008 PCs and VP-058 properties
- 3 pts is defensibly the right number
- AC-005 explicitly tests the "valid HMAC, unadmitted node → ErrNotAdmitted" path that distinguishes the two sentinels — exactly the right discrimination test
- Includes "Previous Story Intelligence" with the H-1 tautology guard reminder — story is aware it's standing on S-2.02's shoulders

The C-1 + C-2 + C-3 issues above are in the *spec dependencies* (VP-058, ADR-009), not in the story itself.

### CO-2: BC-2.05.008 has tight, distinct, non-overlapping postconditions

PC-1 (valid HMAC), PC-2 (invalid HMAC), PC-3 (ordering), PC-4 (missing forwarding entry) form a clean partition of the input space. The invariant 2 ("ErrHMACVerificationFailed is distinct from admission.ErrNotAdmitted") is exactly the right invariant for `errors.Is`-aware error handling. EC-005 (valid HMAC, unadmitted source) is genuinely an edge case (rare configuration where forwarding entry exists but admitted-set membership lapsed). This BC is well-crafted.

### CO-3: ARCH-08 §6.6 explicit "PLANNED, not on develop" callout

The §6.5 / §6.6 separation is excellent governance — it explicitly distinguishes target architecture from current state. The §1 scope callout (added in v1.4) makes this contract explicit. This prevents the v1.2 "hallucinated 16-package table" mistake from recurring. Strong commendation for the cycle-1/wave-2 lessons-learned that produced this.

### CO-4: S-3.01..S-3.03's "Previous Story Intelligence" table is genuinely useful

Each story carries forward decisions from prior waves with rationale:

- "HalfChannel is pure state machine; tick-driven externally" → "internal/tmux must drive the halfchannel tick loop"
- "Never return internal pointers from locked accessor" → "ConsoleSet.List() must return copies"

This is exactly the right level of cross-wave knowledge transfer. The story-writer is not just copy-pasting — they're propagating actionable constraints.

---

## Hidden-Dependency Audit Summary

| Story | Declared `depends_on` | Actually depends on (not declared) |
|-------|----------------------|-----------------------------------|
| S-3.01 | S-1.02 (halfchannel), S-2.02 (admission keyset), S-2.01 (hmac) | Real tmux ≥3.0 in CI; PTY simulation infrastructure |
| S-3.02 | S-3.01 (tmux published session) | Frame-receive code path that S-3.01 wires up via `SessionPublisher` — undocumented coupling |
| S-3.03 | S-3.02 (Session struct), S-2.02 (admission keyset) | Modification to S-3.02's upstream-receive code path (see H-6); a fresh `Authorizer` hook S-3.02 doesn't currently emit |
| S-3.04 | S-2.01 (hmac primitive), S-2.02 (admission keyset + `ForwardingEntry.FrameAuthKey`) | Correct — and verified against `routing.go` on develop. No hidden dependency. |

S-3.01's `depends_on: [S-1.02, S-2.02, S-2.01]` — S-1.02 (halfchannel) is correct because `internal/tmux` imports `halfchannel` per ARCH-08 §6.6. S-2.02 is correct because the `SessionPublisher` (per S-3.01 task #4) needs to check admission before publishing. S-2.01 is technically not needed — `internal/tmux` doesn't import `internal/hmac` directly. Likely declared for transitive completeness; harmless.

---

## Demo Strategy Preview

| Story | Likely demo evidence |
|-------|---------------------|
| S-3.01 | VHS terminal recording — `tmux new-session -d -s agent-01` then `switchboard access` startup → log shows "discovered session agent-01" → kill tmux → restart `switchboard access` with PTY-only env → log shows "tmux not found; using PTY proxy". Story does NOT scope this — recommend adding "demo via VHS" to a final story task. |
| S-3.02 | VHS split-screen recording — two consoles attach to one session; type on console A → appears on B; detach A → B continues. Plus a `go test -race -run TestSession_MultiConsoleFanOut` output. |
| S-3.03 | Plain `go test -v` output showing the 6 AC tests pass; plus a `grep -r SessionAuth internal/routing/` returning empty (VP-012). |
| S-3.04 | `go test -v -run TestRouteFrame_HMACEnforced ./internal/routing/` output + a small benchmark showing HMAC verification overhead per frame. |

S-3.01 is the one that risks under-evidencing its acceptance — recommend adding "VHS demo recording" as task #11.

---

## Verdict

**REVISE_BEFORE_SHIP**

Three CRITICALs must be addressed before the test-writer is dispatched on S-3.04 (C-1 VP-058 harness, C-2 ADR-009 self-contradiction, C-3 ADR-009 signature mismatch). Without these fixes, S-3.04 will trip immediately on Red Gate write or on the first attempt to port the VP-058 harness.

The HIGH findings are clustered around two themes:
- **S-3.01 environmental + complexity issues** (H-2 tmux in CI, H-3 99% AC, H-8 pts under-estimate)
- **S-3.02/S-3.03 coupling** (H-5 detection mechanism, H-6 hidden Authorizer hook, H-8 S-3.03 pts under-estimate)

S-3.04 itself is in good shape (CO-1) once its spec dependencies are repaired. It should ship first in the wave if at all possible — it has the lowest blast radius and is a natural quick win.

---

## Top 3 Concerns for the Orchestrator

1. **VP-058 harness skeleton (C-1)** — calls non-existent `ks.Register(svtnID, nodeAddr)`. The test-writer dispatched on S-3.04 will copy this and fail to compile. Fix the VP-058.md harness to use `ks.RegisterKey(svtnID, pubkey, role)` + `ks.AdmitNode(...)` *before* dispatching the test-writer.

2. **ADR-009 contradicts itself + has wrong signatures (C-2, C-3)** — ADR-009 claims HMAC is checked before admitted-set, then says the two are "merged" (contradiction); ADR-009 prescribes `verifyFrameHMAC(...) error` but the real signature returns `bool`; ADR-009 says the key lives on `admitted_key_set` but it actually lives on `Router.forwardingTable[svtnID][nodeAddr].FrameAuthKey`. A code reviewer trusting ADR-009 over the existing code will file regression PRs.

3. **S-3.03 silently mutates S-3.02's surface area (H-6)** — S-3.03 says it adds `internal/session/auth.go` and lists only `auth.go` + `auth_test.go` in its File Structure Requirements. But AC-005/AC-006 (read-only upstream rejection) requires touching the upstream-receive code path owned by S-3.02. Either S-3.02 needs to expose an `Authorizer` hook in its task list, or the wave plan needs to declare S-3.03 as modifying S-3.02 files. Without this, the wave gate will see undeclared file edits in the S-3.03 implementer's PR.

Also strongly recommend **re-pointing S-3.01 from 8 to 13 pts** (or splitting tmux + PTY) and **S-3.03 from 5 to 8 pts** before the wave starts — both are under-estimated given the surface area.
