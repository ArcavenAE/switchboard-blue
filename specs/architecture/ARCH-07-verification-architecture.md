---
artifact_id: ARCH-07-verification-architecture
document_type: architecture-section
level: L3
version: "1.0"
status: draft
producer: architect
timestamp: 2026-06-23T00:00:00
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
---

# ARCH-07: Verification Architecture

## Purity Boundary Strategy

The verification strategy is grounded in the purity boundary. Pure-core modules are
deterministic functions over data — they take input structs and return output structs
with no I/O, no globals, no time. These are the formal verification targets.
Effectful modules are tested through integration and race detection.

See ARCH-09 for the complete per-package classification.

## Provable Properties Catalog

### P0 Properties (Must Prove — Security + Protocol Correctness)

| VP ID | Property | Module | Method |
|-------|----------|--------|--------|
| VP-001 | `ParseOuterHeader` and `EncodeOuterHeader` are inverses: decode(encode(x)) == x for all valid headers | internal/frame | proptest |
| VP-002 | `ParseOuterHeader` rejects any byte sequence with `version_major != 1` with ErrVersionMismatch | internal/frame | proptest |
| VP-003 | `EncodeOuterHeader` produces exactly 44 bytes for all valid inputs | internal/frame | proptest |
| VP-004 | `ComputeHMAC` and `VerifyHMAC` are consistent: VerifyHMAC(key, frame, ComputeHMAC(key, frame)) == true | internal/hmac | proptest |
| VP-005 | `VerifyHMAC` returns false for any single-bit flip in the frame payload | internal/hmac | proptest/fuzz |
| VP-006 | `VerifyHMAC` returns false for any key not used to compute the HMAC | internal/hmac | proptest |
| VP-007 | `AdmissionChallenge` private key bytes never appear in the returned challenge or response structs | internal/admission | proptest |
| VP-008 | `VerifyAdmission` returns false for any public key not in the admitted set | internal/admission | proptest |
| VP-009 | `VerifyAdmission` returns false for a replayed nonce (nonce already in used set) | internal/admission | proptest |
| VP-010 | `SVTNRoute` never delivers a frame to a node in a different SVTN than the frame's svtn_id | internal/routing | proptest |
| VP-011 | `SVTNRoute` never forwards a frame back toward the arrival interface | internal/routing | proptest |
| VP-012 | `SessionAuth.Authorize` returns false for any console key not in the session's authorized set | internal/session | proptest |
| VP-013 | `SessionAuth.Authorize` returns false for a read-only key submitting an upstream frame | internal/session | proptest |
| VP-014 | `DeriveNodeAddress` is deterministic: same (svtn_id, pubkey) always produces same address | internal/frame | proptest |
| VP-015 | Outer header payload field is treated as opaque bytes by all router code paths: no attempt to parse channel header | internal/routing | fuzz + code audit |

### P1 Properties (Should Prove — Session Correctness)

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
| VP-027 | `QualityIndicator.Compute` transitions are monotonic under sustained degradation: green→yellow→red only | internal/metrics | proptest |
| VP-028 | `Config.Validate` returns an error for tick_interval outside [5ms, 50ms] | internal/config | unit |
| VP-029 | `Config.Validate` returns an error for any missing required field | internal/config | proptest |
| VP-030 | `sbctl` exits with code 1 and E-NET-001 when daemon connection is refused | cmd/sbctl | integration |

### Test-Sufficient Properties (Integration / Race Detector / E2E)

| VP ID | Property | Module | Method |
|-------|----------|--------|--------|
| VP-031 | tmux control mode: all `%output` events delivered during sustained 10KB/s session | internal/tmux | integration |
| VP-032 | tmux PTY fallback activates when control mode fails | internal/tmux | integration |
| VP-033 | Session attach/detach: console receives downstream frames after attach | integration | e2e |
| VP-034 | Multi-console fan-out: two consoles both receive all frames | integration | e2e |
| VP-035 | Read-only console: upstream keystrokes rejected by access node | internal/session | integration |
| VP-036 | Session survives IP address change (wifi handoff simulation) | integration | e2e |
| VP-037 | Router drain: nodes migrate to alternate router within 2s | integration | e2e |
| VP-038 | E→PE graduation: config change only, no session drop | integration | e2e |
| VP-039 | SVTN isolation: no cross-SVTN frame delivery with two SVTNs on same router | integration | e2e |
| VP-040 | Multipath failover: session recovers within 2s of path failure (NFR-003) | integration | e2e |
| VP-041 | Tick regularity: p99 jitter ≤ 2ms over 1,000 ticks (NFR-009) | internal/halfchannel | benchmark |
| VP-042 | Keystroke-to-echo: p99 ≤ 100ms over LAN at tuned tick interval (NFR-001) | integration | benchmark |

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
