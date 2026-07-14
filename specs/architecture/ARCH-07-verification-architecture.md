---
artifact_id: ARCH-07-verification-architecture
document_type: architecture-section
level: L3
version: "1.11"
status: draft
producer: architect
timestamp: 2026-06-29T00:00:00
phase: 1b
traces_to: ARCH-INDEX.md
inputDocuments:
  - '.factory/specs/prd.md'
  - '.factory/specs/module-criticality.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '.factory/specs/prd-supplements/nfr-catalog.md'
kos_anchors:
  - elem-timeslice-framing
  - elem-ssh-end-to-end-encryption
  - elem-asymmetric-half-channels
modified:
  - 2026-07-14T00:00:00 # v1.11 — F-DWSP5-002 (MED, spec-adversarial pass 5): VP catalog total refreshed 77→80 (VP-INDEX v2.43). Added VP-078 (integration, cmd/sbctl) and VP-080 (integration, internal/discovery) rows to the Test-Sufficient Properties table; VP-079 (code-audit, internal/mgmt) footnote-only per this document's proof-method-bucket convention (code-audit is not one of the five bucketed methods — VP-058/VP-061 precedent). Footnote block extended for all three. Sibling propagation partner of ARCH-11 v1.24 and ARCH-INDEX v1.13 (same burst).
  - 2026-07-03T00:00:00 # v1.10 — F-P5P20-B-001: VP-043 method column sibling-propagation from VP-INDEX v2.35 (F-P5P3-B-001 close 2026-07-02). Phase-1c-refinement Test-Sufficient table VP-043 row (~L183) Method: proptest → strong-oracle.
  - 2026-07-03T00:00:00 # v1.9 — F-P5P19-B-002: VP catalog total refreshed 76→77 (VP-INDEX v2.36); added footnote block covering VP-075/VP-076/VP-077 (Wave-5 admin-authority triplet for BC-2.05.004); sibling propagation partner of ARCH-11 v1.16 (F-P5P19-B-001).
  - 2026-06-30T00:00:00 # v1.8 — S502-DEFER-3 handoff (commit 7ee5b82): VP-062 bumped v1.2→v1.3 (Property 5a: failed+pending precedence ruling per BC-2.06.003 v1.8 EC-007); VP catalog total corrected 75→76; footnote updated.
  - 2026-06-30T00:00:00 # v1.7 — F-P8L3-001: VP catalog total updated from 74 to 75 (VP-075 was minted in Pass-6/7 but total not incremented)
  - 2026-06-30T00:00:00 # v1.6 — F-P7L3-001: VP-075 module corrected from internal/mgmt to cmd/switchboard in Phase 1c-refinement integration table
  - 2026-06-30T00:00:00 # v1.5 — F-T3-302 (S-6.06 Pass-3 lens-3): add VP-075 to Phase 1c-refinement integration table (admin.key.* handler caller-role check, internal/mgmt)
---

# ARCH-07: Verification Architecture

## Purity Boundary Strategy

The verification strategy is grounded in the purity boundary. Pure-core modules are
deterministic functions over data — they take input structs and return output structs
with no I/O, no globals, no time. These are the formal verification targets.
Effectful modules are tested through integration and race detection.

See ARCH-09 for the complete per-package classification.

## Provable Properties Catalog

> Categorization here is by PROOF-METHOD bucket (proptest/integration/e2e/fuzz/benchmark), not by phase. For canonical phase classification (P0/P1/P2), see `VP-INDEX.md` and per-row "Phase" column in `ARCH-11-verification-coverage-matrix.md`. Phase indicates urgency (P0 = MVP-blocking); proof-method indicates verification technique (proptest = pure-core, integration = boundary, etc.).

### Pure-Core Proptest Catalog (Must Prove — Pure Math Properties)

| VP ID | Property | Module | Method |
|-------|----------|--------|--------|
| VP-001 | `ParseOuterHeader` and `EncodeOuterHeader` are inverses: decode(encode(x)) == x for all valid headers | internal/frame | proptest |
| VP-002 | `ParseOuterHeader` rejects any byte sequence with `version_major != 0` with ErrVersionMismatch | internal/frame | proptest |
| VP-003 | `EncodeOuterHeader` produces exactly 44 bytes for all valid inputs | internal/frame | proptest |
| VP-004 | `ComputeHMAC` and `VerifyHMAC` are consistent: VerifyHMAC(key, frame, ComputeHMAC(key, frame)) == true | internal/hmac | proptest |
| VP-005 | `VerifyHMAC` returns false for any single-bit flip in the frame payload | internal/hmac | fuzz |
| VP-006 | `VerifyHMAC` returns false for any key not used to compute the HMAC | internal/hmac | proptest |
| VP-007 | `AdmissionChallenge` private key bytes never appear in the returned challenge or response structs | internal/admission | proptest |
| VP-008 | `VerifyAdmission` returns false for any public key not in the admitted set | internal/admission | proptest |
| VP-009 | `VerifyAdmission` returns false for a replayed nonce (nonce already in used set) | internal/admission | proptest |
| VP-010 | `SVTNRoute` never delivers a frame to a node in a different SVTN than the frame's svtn_id | internal/routing | proptest |
| VP-011 | `SVTNRoute` never forwards a frame back toward the arrival interface | internal/routing | proptest |
| VP-012 | `SessionAuth.Authorize` returns false for any console key not in the session's authorized set | internal/session | proptest |
| VP-013 | `SessionAuth.Authorize` returns false for a read-only key submitting an upstream frame | internal/session | proptest |
| VP-014 | `DeriveNodeAddress` is deterministic: same (svtn_id, pubkey) always produces same address | internal/frame | proptest |
| VP-015 | Outer header payload field is treated as opaque bytes by all router code paths: no attempt to parse channel header | internal/routing | fuzz (harness + manual audit) |

### Boundary/Integration Proptest Catalog (Should Prove — State + I/O Properties)

| VP ID | Property | Module | Method |
|-------|----------|--------|--------|
| VP-016 | `HalfChannel.Tick` emits exactly one frame per tick regardless of payload availability | internal/halfchannel | proptest |
| VP-017 | `HalfChannel.Tick` increments sequence number by exactly 1 on each call | internal/halfchannel | proptest |
| VP-018 | `HalfChannel.Tick` emits empty-payload frame when no data is queued | internal/halfchannel | proptest |
| VP-019 | `ARQ.OnAck` does not deliver any frame twice for any ACK sequence | internal/arq | proptest |
| VP-020 | `ARQ.OnAck` delivers frames in-order: no frame with seq N is delivered before seq N-1 | internal/arq | proptest |
| VP-021 | `ARQ.TLPKTDROP` triggers when and only when a frame is overdue by > tlpktdrop_timeout | internal/arq | proptest |
| VP-022 | `Replay.OnUpstream` deduplicates: same chan_seq is never delivered twice | internal/replay | proptest |
| VP-023 | `Replay.OnUpstream` delivers in order: keystrokes from the replay window arrive in sequence order | internal/replay | proptest |
| VP-024 | `Multipath.OnFrameArrival` delivers the first copy and discards all subsequent copies for same checksum | internal/multipath | proptest |
| VP-025 | `DropCache` never exceeds its configured capacity | internal/multipath | proptest |
| VP-026 | `PathScore` ranking is transitive: if score(A) < score(B) < score(C) then rank(A) < rank(B) < rank(C) | internal/paths | proptest |
| VP-027 | `QualityIndicator.Update(rttMs, lossPct)` transitions Quality state per BC-2.06.001 OR-form thresholds (Red > Yellow precedence); monotonic under sustained degradation: green→yellow→red only | internal/metrics | proptest |
| VP-028 | `Config.Validate` returns an error for tick_interval outside [5ms, 50ms] | internal/config | proptest |
| VP-029 | `Config.Validate` returns an error for any missing required field | internal/config | proptest |
| VP-030 | `sbctl` exits with code 1 and E-NET-001 when daemon connection is refused | cmd/sbctl | integration |

### Test-Sufficient Properties (Integration / Race Detector / E2E)

| VP ID | Property | Module | Method |
|-------|----------|--------|--------|
| VP-031 | tmux control mode: all `%output` events delivered during sustained 10KB/s session | internal/tmux | integration |
| VP-032 | tmux PTY fallback activates when control mode fails | internal/tmux | integration |
| VP-033 | Session attach/detach: console receives downstream frames after attach | internal/session | e2e |
| VP-034 | Multi-console fan-out: two consoles both receive all frames | internal/session | e2e |
| VP-035 | Read-only console: upstream keystrokes rejected by access node | internal/session | integration |
| VP-036 | Session survives IP address change (wifi handoff simulation) | internal/admission | e2e |
| VP-037 | Router drain: nodes migrate to alternate router within 2s | internal/drain | e2e |
| VP-038 | E→PE graduation: config change only, no session drop | internal/config | e2e |
| VP-039 | SVTN isolation: no cross-SVTN frame delivery with two SVTNs on same router | internal/routing | e2e |
| VP-040 | Multipath failover: session recovers within 2s of path failure (NFR-003) | internal/multipath | e2e |
| VP-041 | Tick regularity: p99 jitter ≤ 2ms over 1,000 ticks (NFR-009) | internal/halfchannel | benchmark |
| VP-042 | Keystroke-to-echo: p99 ≤ 100ms over LAN at tuned tick interval (NFR-001) | internal/halfchannel | benchmark |

> VP catalog total = 80; full BC→VP coverage in ARCH-11. VP-043 through VP-057
> were added in Phase 1c-refinement to close coverage gaps. VP-059 added 2026-06-27
> for BC-2.05.005 PC-3 (Wave 3 gate F-2 remediation — FailureCounter threshold proptest).
> VP-058 added at Wave 3 for BC-2.05.008 (RouteFrame HMAC code-audit).
> VP-060 added 2026-06-27 for BC-2.04.007 (daemon lifecycle — integration/subprocess).
> VP-061–VP-067 added 2026-06-28 (Phase 6 hardening, Wave-5 management plane).
> VP-068–VP-073 added 2026-06-29 for BC-2.07.004 v1.3 Wave-5 Convergence Rulings A–E
> (Invariant 8 key-size panic, PC-10 Serve nil on shutdown, PC-11 E-RPC-010 unknown
> command, PC-12 E-RPC-011 handler error, PC-1 write-deadline CWE-400, EC-013 loopback).
> VP-074 added 2026-06-29 for BC-2.06.001 threshold classification (unit; L-001 disambiguation).
> VP-069 updated to v1.1 (ARCH-12 v1.4 Ruling G): property extended to require non-nil on
> unexpected-close path; coverage expanded to 3 paths; canonical predicate documented.
> VP-069 updated to v1.2 (ARCH-12 v1.5 Ruling P): fatal-accept-error drain obligation added —
> closeAllConns() MUST be called before connWG.Wait() on unexpected-close path; drain budget
> ≤200ms verified by TestServe_FatalAcceptErrorDrainsQuickly; source-contract citation extended
> to Rulings B/G/I/P; error-taxonomy v2.7→v2.8 (E-NET-001 two-case). No count changes.
> VP-073 updated to v1.1 (ARCH-12 v1.4 Ruling L): source-contract extended to cite
> error-taxonomy.md E-CFG-008 Variant 2 / buildMgmtListener canonical message prefix;
> test assertion guidance requires strings.Contains(err.Error(), "E-CFG-008").
> VP-062 updated to v1.3 2026-06-30 (S502-DEFER-3 closure, commit 7ee5b82): Property 5a added
> (failed+pending precedence: Degraded=true AND rttP99Valid=false → quality="pending"; EC-007);
> fuzz corpus seed 8 added; BC pin swept v1.7→v1.8. No count changes.
> VP-075 (integration, cmd/switchboard, P0) added 2026-06-30: admin.key.* (register/revoke/expire)
> handlers reject non-control callers with E-ADM-009; connection kept open; no key store mutation.
> Handler admission-authority write path. Covers BC-2.05.004 caller-role enforcement gate.
> VP-076 (integration, cmd/switchboard, P0) added 2026-06-30: bootstrap-key non-revocable AND
> non-expirable symmetric lockout invariant per BC-2.05.004 EC-007 v1.12. E-ADM-020 / E-ADM-021
> sentinels for any well-formed request on the bootstrap key.
> VP-077 (integration, cmd/switchboard, P0) added 2026-07-03: list-keys admission-gate three-way
> disjunction — any-role OR operator-set OR bootstrap-key; else E-ADM-009. Covers BC-2.05.004
> EC-008 (three admission failure modes for admin.key.list-keys). Closes BC↔VP↔AC triangle for
> BC-2.05.004 EC-008. Orthogonal to VP-075 (write-authority gate on mutating operations).
> VP-078 (integration, cmd/sbctl, P2) added 2026-07-13 for BC-2.06.004: `sbctl paths ping`
> reports `rtt_ms` as `float64`, never emits a quality/status classification field, fast+slow
> round trips (implementing_story: S-BL.CLI-SURFACE-COMPLETION).
> VP-079 (code-audit, internal/mgmt, P2) added 2026-07-13 for BC-2.06.004: `paths.ping` RPC
> handler performs zero per-path metrics reads/writes (no `PathTracker` interaction) — `{}`
> in, `{"pong": true}` out, no other side effect (implementing_story: S-BL.CLI-SURFACE-COMPLETION).
> Code-audit method — no table row per this document's proof-method-bucket convention
> (VP-058/VP-061 precedent: code-audit is not one of the five bucketed methods this catalog
> organizes by); footnote-only, matching those two properties' treatment.
> VP-080 (integration, internal/discovery, P1, status: draft) added 2026-07-13 for BC-2.03.001:
> router-side discovery ingest discards non-increasing `Sequence` (`uint64`, epoch-qualified
> per F-DWSP4-001) per `(SVTNID,NodeAddr)` even after HMAC passes; cold-start accepts
> unconditionally; forward-increasing `Sequence` accepted and advances state; residual replay
> window bounded to ≤1 heartbeat interval. Node-restart liveness gap (F-DWSP4-001) closed with
> two independently-bounded residuals: same-epoch-second crash-loop ≤1s; backward-clock-adjustment
> bounded by the adjustment magnitude N, not ≤1s (F-DWSP5-precision-corrected — see rulings v1.6/
> VP-080.md v1.3). Implementing_story: S-BL.DISCOVERY-WIRE.

### Phase 1c-refinement: Pure-Core Additions

| VP ID | Property | Module | Method |
|-------|----------|--------|--------|
| VP-053 | K empty-tick frames → K frames with contiguous sequence numbers | internal/halfchannel | proptest |
| VP-057 | Node private key bytes absent from all emitted frame types (sampling + HKDF sketch) | internal/admission | proptest |

### Wave 3 Gate F-2 Remediation Additions (2026-06-27)

| VP ID | Property | Module | Method |
|-------|----------|--------|--------|
| VP-059 | `FailureCounter.RecordHMACFailure` fires E-ADM-017 at exactly the ≥5th call within 60s sliding window and not before; fire-once-per-crossing; concurrent-safe (race detector) | internal/admission | proptest |

### BC-2.04.007 Daemon Lifecycle Addition (2026-06-27)

| VP ID | Property | Module | Method |
|-------|----------|--------|--------|
| VP-060 | Connect failure → exit non-zero (E-SYS-002, no relay goroutines); SIGTERM/SIGINT → clean shutdown (all goroutines drain within 500ms, `sc.Close()` once, exit 0, no panic) | cmd/switchboard | integration (subprocess) |

### BC-2.06.001 Threshold Classification Addition (2026-06-29)

| VP ID | Property | Module | Method |
|-------|----------|--------|--------|
| VP-074 | `QualityIndicator` threshold classification maps (RTT, loss) → {Green, Yellow, Red} correctly; enum cardinality = 3; all 8 boundary values correct per BC-2.06.001 PC-2/PC-3/PC-4 | internal/metrics | unit |

VP-027 (proptest) covers transition ordering under sustained degradation. VP-074 (unit) covers single-step threshold mapping and boundary values. The two VPs are orthogonal: VP-027 exercises state-machine sequences; VP-074 exercises the classification table directly. See VP-074.md for the 14-case table-driven harness.

### BC-2.07.004 v1.3 Wave-5 Convergence Rulings A–E Additions (2026-06-29)

| VP ID | Property | Module | Method |
|-------|----------|--------|--------|
| VP-068 | `mgmt.NewServer` panics at construction if `len(daemonKey) != ed25519.PrivateKeySize` (nil or short key; fail-fast remote-panic-DoS guard, Invariant 8) | internal/mgmt | unit |
| VP-069 | `mgmt.Server.Serve` returns `nil` on intentional `Shutdown` or ctx-cancel; returns **non-nil** on unexpected listener failure (fd externally closed, ctx live, Shutdown never called) — canonical predicate: `shuttingDown.Load() \|\| (errors.Is(err, net.ErrClosed) && ctx.Err() != nil)`; 3 paths: Shutdown, ctx-cancel, unexpected-close; **fatal-accept-error drain (Ruling P):** `closeAllConns()` MUST be called before `connWG.Wait()` on unexpected-close path; drain budget ≤200ms (`TestServe_FatalAcceptErrorDrainsQuickly`) (PC-10 Rulings B/G/I/P) | internal/mgmt | integration |
| VP-070 | Unregistered RPC command → in-band `ok:false` response with `E-RPC-010 "unknown command: <cmd>"`; connection NOT closed (PC-11) | internal/mgmt | integration |
| VP-071 | Handler `Fn` returns non-nil error → in-band `ok:false` response with `E-RPC-011 "<err>"` verbatim; connection NOT closed (PC-12) | internal/mgmt | integration |
| VP-072 | Write deadline set before every `sendJSON` (`HandshakeTimeout` for handshake sends, `RPCIdleTimeout` for RPC responses); cleared after each send; closes CWE-400 write-side slowloris (PC-1 Ruling E) | internal/mgmt | integration |
| VP-073 | Console-mode TCP bound to non-loopback address → `buildMgmtListener` aborts with E-CFG-008; daemon startup does not proceed (EC-013 Rulings D/L); canonical message prefix: `"E-CFG-008: management_socket: console mode requires a loopback address …"`; test assertion: `strings.Contains(err.Error(), "E-CFG-008")` (error-taxonomy.md E-CFG-008 Variant 2) | cmd/switchboard | integration |

### Phase 1c-refinement: Boundary/Integration Additions

| VP ID | Property | Module | Method |
|-------|----------|--------|--------|
| VP-051 | HalfChannel independence: half-channel B unaffected by A's frame production | internal/halfchannel | proptest |
| VP-054 | Receiver dedup: first-arriving copy delivered, duplicate discarded silently | internal/multipath | integration |
| VP-055 | Presence advertisement payload round-trip: required fields present and stable | internal/discovery | proptest |

### Test-Sufficient Properties Added in Phase 1c-refinement (Integration / E2E)

| VP ID | Property | Module | Method |
|-------|----------|--------|--------|
| VP-043 | XOR FEC: single loss in group recoverable | internal/arq | strong-oracle |
| VP-044 | Presence advertisement includes required fields | internal/discovery | integration |
| VP-045 | Console session enumeration without hostnames | internal/discovery | e2e |
| VP-046 | Key lifecycle: register/revoke/expire | internal/svtnmgmt | integration |
| VP-075 | admin.key.* handlers reject non-control callers with E-ADM-009; connection kept open; no key store mutation | cmd/switchboard | integration |
| VP-076 | bootstrap-key non-revocable AND non-expirable symmetric management-lockout prevention (E-ADM-020 / E-ADM-021 for any well-formed request on bootstrap key) per BC-2.05.004 EC-007 | cmd/switchboard | integration |
| VP-077 | admin.key.list-keys admission gate: IsAdmittedAnyRole OR OperatorKeySet OR BootstrapKey; else E-ADM-009 (three failure modes: no caller, cross-SVTN, revoked/expired); SVTN-existence check precedes gate (EC-008) | cmd/switchboard | integration |
| VP-047 | Per-path metrics queryable via sbctl | internal/metrics | integration |
| VP-048 | Control node creates/destroys SVTNs | internal/svtnmgmt | integration |
| VP-049 | sbctl unified CLI with OpenSSH auth | cmd/sbctl | e2e |
| VP-050 | Console remotely controllable via sbctl | cmd/sbctl | e2e |
| VP-052 | `OnMissingFrame` call accumulates missed-frame count; N consecutive calls → quality indicator downgrade (one level) per BC-2.06.002 count-based API | internal/metrics | integration |
| VP-056 | Console detach releases session without closing it; observers unaffected | internal/session | integration |
| VP-078 | `sbctl paths ping` reports `rtt_ms` as `float64` and never emits a quality/status classification field, for both fast and slow round trips | cmd/sbctl | integration |
| VP-080 | Router-side discovery ingest discards non-increasing `Sequence` (`uint64`, epoch-qualified) per `(SVTNID,NodeAddr)` even after HMAC passes; cold-start accepts the first frame unconditionally; forward-increasing `Sequence` accepted and advances state; residual replay window bounded to ≤1 heartbeat interval; restarted node's epoch-qualified `Sequence` forward-progresses past its own prior watermark, closing the node-restart liveness gap — same-epoch-second crash-loop residual bounded to ≤1s, backward-clock-adjustment residual bounded by the adjustment magnitude N instead (not ≤1s) | internal/discovery | integration |

## Fuzz Targets (P0 Security Boundaries)

| Fuzz Target | Input | What We're Looking For |
|-------------|-------|----------------------|
| `FuzzParseOuterHeader` | arbitrary `[]byte` | panic, allocation storm, infinite loop |
| `FuzzVerifyHMAC` | arbitrary `(key, frame, tag)` | false positives (tag accepted when wrong) |
| `FuzzParseAdmissionMsg` | arbitrary `[]byte` | panic on malformed wire messages |
| `FuzzConfigParse` | arbitrary YAML bytes | panic, segfault, resource exhaustion |

Fuzz targets are in `*_test.go` files as `FuzzXxx(f *testing.F)` functions, runnable
via `go test -fuzz=FuzzXxx -fuzztime=300s`.

## Mutation Testing

Go mutation testing via `go-mutesting` (or equivalent). Targets:

| Module | Mutation Focus | Kill Rate Target |
|--------|---------------|-----------------|
| internal/frame | Field encoding, version check, length calculation | CRITICAL (≥95%) |
| internal/hmac | HMAC comparison, key derivation | CRITICAL (≥95%) |
| internal/admission | Nonce uniqueness, key set lookup | CRITICAL (≥95%) |
| internal/routing | SVTN partition logic, split-horizon check | CRITICAL (≥95%) |
| internal/session | Authorization check, read-only enforcement | CRITICAL (≥95%) |

Mutation testing is run as a CI gate before Phase 5 (adversarial review). Survivors
below the kill rate target block the gate.

## Race Detector Policy

`go test -race ./...` is run on every commit (via `just test-race`). Race conditions
in any package below the effectful boundary are treated as P0 bugs. The Go race
detector is the backstop for mutex discipline (see go.md rule #12).
