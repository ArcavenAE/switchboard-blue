# Red Gate Log — S-W3.04 Daemon Assembly

**Story:** S-W3.04 — Full Daemon Assembly  
**Date:** 2026-06-27  
**Phase:** TDD (test-writer pass)  
**BC-5.38.001 Status:** RED GATE VERIFIED

## Summary

7 new tests written across 2 files. All 7 fail against the current stubs
(panic: not implemented). All 34 existing tests still pass.

## Test Files

| File | New Tests | Status |
|------|-----------|--------|
| `internal/tmux/connector_frames_test.go` | 1 | FAILING (Red Gate) |
| `cmd/switchboard/main_test.go` | 6 | FAILING (Red Gate) |

## Per-Test Red Gate Results

| Test Name | AC/BC Trace | Failure Mode |
|-----------|-------------|--------------|
| `TestSessionConnectorFramesSurviveFailover` | AC-004 / BC-2.04.001 PC-5 + BC-2.04.002 PC-4 | `panic: not implemented: SessionConnector.Frames() relay goroutine (S-W3.04 AC-004)` |
| `TestRouterLoggerEmitsEADM016` | AC-001 / BC-2.05.008 PC-2 | `panic: not implemented: buildRouter (S-W3.04 AC-001)` |
| `TestDaemonAuthRejectsUnregisteredConsole` | AC-002 / BC-2.04.005 PC-3 + BC-2.04.003 PC-3 | `panic: not implemented: buildAccessNode (S-W3.04 AC-002)` |
| `TestDaemonSweepEvictsStaleConsole` | AC-003 / BC-2.04.004 PC-1 + PC-3 | `panic: not implemented: startSweepTicker goroutine (S-W3.04 AC-003)` |
| `TestDaemonFramesDroppedLoggedOnTick` | AC-006 / BC-2.04.006 invariant 4 | `panic: not implemented: startFramesDroppedTicker goroutine (S-W3.04 AC-006)` |
| `TestDaemonConnectFailureExitsNonZero` | AC-007 / BC-2.04.007 PC-1 | `panic: not implemented: runAccess daemon wiring (S-W3.04 AC-001–AC-008)` |
| `TestDaemonCleanShutdown` | AC-008 / BC-2.04.007 PC-2 | `panic: not implemented: runAccess daemon wiring (S-W3.04 AC-001–AC-008)` |

## Existing Test Baseline (still passing)

```
ok  github.com/arcavenae/switchboard/cmd/switchboard    (3 tests)
ok  github.com/arcavenae/switchboard/internal/admission (cached)
ok  github.com/arcavenae/switchboard/internal/frame     (cached)
ok  github.com/arcavenae/switchboard/internal/halfchannel (cached)
ok  github.com/arcavenae/switchboard/internal/hmac      (cached)
ok  github.com/arcavenae/switchboard/internal/routing   (cached)
ok  github.com/arcavenae/switchboard/internal/session   (cached)
ok  github.com/arcavenae/switchboard/internal/tmux      (31 tests pass; new test excluded)
```

## Red Gate Verification Command

```bash
# All new tests fail:
go test ./internal/tmux/ -run TestSessionConnectorFramesSurviveFailover -count=1
go test ./cmd/switchboard/ -run "TestRouterLoggerEmitsEADM016|TestDaemonAuthRejectsUnregisteredConsole|TestDaemonSweepEvictsStaleConsole|TestDaemonFramesDroppedLoggedOnTick|TestDaemonConnectFailureExitsNonZero|TestDaemonCleanShutdown" -count=1

# Existing tests pass (excluding new failing tests):
go test ./cmd/switchboard/ -run "^TestRun$|^TestVersionNonEmpty$|^TestRun_WriteError$" -count=1
go test ./internal/tmux/ -run "TestSessionConnector_|TestPTYProxy_|TestTmuxControlMode_|TestControlMode_|TestClassifyStderr" -count=1
```

## AC-005 Note

AC-005 (Frames→DeliverFrame bridge) has no standalone test per the story spec.
It is covered structurally by AC-004 (TestSessionConnectorFramesSurviveFailover,
which exercises the relay goroutine) and AC-008 (TestDaemonCleanShutdown, which
verifies the bridge goroutine exits cleanly on sc.Close()).

---

## Wave-3 Finding I-1 — Goroutine Join Red Gate (2026-06-27)

**Story:** S-W3.04 — Finding I-1 (ticker wg-tracking)
**Branch:** `fix/W3-i1-ticker-wg-join`
**Commit:** `74a6de2`
**BC-5.38.001 Status:** RED GATE VERIFIED (build failure)

### Test

`cmd/switchboard/access_goroutine_join_test.go`
`TestRunAccessWithConnectorNoGoroutineLeak` (AC-008 / BC-2.04.007 PC-2 postcon-6)

### Red Gate Result

```
# github.com/arcavenae/switchboard/cmd/switchboard [github.com/arcavenae/switchboard/cmd/switchboard.test]
cmd/switchboard/access_goroutine_join_test.go:105:2: cannot assign to framesDroppedInterval (neither addressable nor a map index expression)
cmd/switchboard/access_goroutine_join_test.go:106:21: cannot assign to framesDroppedInterval (neither addressable nor a map index expression)
FAIL    github.com/arcavenae/switchboard/cmd/switchboard [build failed]
```

The test assigns `framesDroppedInterval = time.Millisecond` to inject a fast tick
interval. `framesDroppedInterval` is currently a `const` in `access.go`, which is
not assignable. The build failure is the Red Gate.

### Why This Is The Correct Red Gate

The compile error enforces the full scope of the required fix: the implementer must
change `framesDroppedInterval` from `const` to `var` (testability) AND add `wg.Add(1)`
before both `startSweepTicker` and `startFramesDroppedTicker` (correctness). Neither
change alone is sufficient — both are required for the test to build AND pass.

### Discriminating Mechanism (channel handshake)

Once `framesDroppedInterval` is a `var`:
- `blockingRelayConnector.RelayDropped()` parks the ticker goroutine on first call
- Test cancels ctx, then selects: if `done` closes within 150ms → goroutine not joined → RED
- On fixed code: `wg.Wait()` blocks until goroutine is released → `done` stays open → PASS

### Required Fix

1. In `access.go`: change `const framesDroppedInterval = 30 * time.Second` to
   `var framesDroppedInterval = 30 * time.Second`
2. In `runAccessWithConnector`: add `wg.Add(1)` + `defer wg.Done()` inside both
   `startSweepTicker` and `startFramesDroppedTicker` goroutines

---

## Implementer Handoff

Make each test pass, one at a time, with minimum code:

1. `internal/tmux/connector_frames.go`: implement `Frames()`, `activeFrSource()`,
   `forwardFrames()` per ADR-011 §Concurrency contract.
2. `cmd/switchboard/access.go`: implement `buildRouter`, `buildAccessNode`,
   `startSweepTicker`, `startFramesDroppedTicker`, `runAccess`.

---

# Red Gate Log — S-4.03 Downstream ARQ with TLPKTDROP

**Story:** S-4.03 — Downstream ARQ with Piggybacked ACK/SACK and TLPKTDROP
**Date:** 2026-06-28
**Phase:** TDD (test-writer pass)
**BC-5.38.001 Status:** RED GATE VERIFIED

## Summary

17 tests written (16 failing + 1 GREEN-BY-DESIGN). All 16 AC/BC/VP/EC tests
panic against stubs. `TestBC_2_02_005_VP052_SACKPopCount` passes because
`SACKPopCount` is already implemented in the stub (per stub file comments —
GREEN-BY-DESIGN).

## Test File

| File | Tests | Status |
|------|-------|--------|
| `internal/arq/arq_test.go` | 17 total | 16 FAILING (Red Gate), 1 GREEN-BY-DESIGN |

## Per-Test Red Gate Results

| Test Name | AC/VP/EC Trace | Failure Mode |
|-----------|----------------|--------------|
| `TestARQ_OnAck_NoDuplicateDelivery` | AC-001 / BC-2.02.005 PC-1, VP-019 | `panic: not yet implemented` (EnqueueSend stub) |
| `TestBC_2_02_005_EC001_IdempotentAck` | EC-001 / BC-2.02.005 | `panic: not yet implemented` (EnqueueSend stub) |
| `TestARQ_InOrderDelivery` | AC-002 / BC-2.02.005 PC-2,4, VP-020 | `panic: not yet implemented` (EnqueueSend stub) |
| `TestBC_2_02_005_InOrder_CanonicalVector` | AC-002 / BC-2.02.005 canonical test vector | `panic: not yet implemented` (EnqueueSend stub) |
| `TestARQ_SACKInChannelHeader` | AC-003 / BC-2.02.005 PC-3, ARCH-02 F-P8-007 | `panic: not yet implemented` (SACKFromChannelHeader stub) |
| `TestBC_2_02_005_SACKNotInOuterHeader` | AC-003 / ARCH-02 | `panic: not yet implemented` (SACKFromChannelHeader stub) |
| `TestARQ_TLPKTDROP_TerminatesOverdueFrame` | AC-004 / BC-2.02.006 PC-1,2, VP-021, EC-003 | `panic: not yet implemented` (EnqueueSend stub) |
| `TestBC_2_02_006_TLPKTDROP_FiresExactlyOnce` | VP-021 / BC-2.02.006 | `panic: not yet implemented` (EnqueueSend stub) |
| `TestBC_2_02_006_TLPKTDROP_SessionContinues` | VP-021 / BC-2.02.006 invariant 1 | `panic: not yet implemented` (EnqueueSend stub) |
| `TestARQ_TLPKTDROP_OnlyOverdueFrames` | AC-005 / BC-2.02.006 PC-2 | `panic: not yet implemented` (EnqueueSend stub) |
| `TestBC_2_02_006_OnlyOverdue_TableDriven` | AC-005 / BC-2.02.006 PC-2 (boundary) | `panic: not yet implemented` (EnqueueSend stub) |
| `TestBC_2_02_005_EC002_SACKWholeWindowGap` | EC-002 / BC-2.02.005 | `panic: not yet implemented` (EnqueueSend stub) |
| `TestBC_2_02_006_EC003_TLPKTDROPDuringFailover` | EC-003 / BC-2.02.006, ADR-005 | `panic: not yet implemented` (EnqueueSend stub) |
| `TestBC_2_02_005_VP019_VP020_NoDoubleDelivery` | VP-019, VP-020 (24 permutations) | `panic: not yet implemented` (EnqueueSend stub) |
| `TestBC_2_02_006_VP021_TLPKTDROPNotSessionTermination` | VP-021 (10 drops, session survives) | `panic: not yet implemented` (EnqueueSend stub) |
| `TestBC_2_02_005_VP019_VP020_LargeScale` | VP-019, VP-020 (1024 trials × 8 frames) | `panic: not yet implemented` (EnqueueSend stub) |
| `TestBC_2_02_005_VP052_SACKPopCount` | VP-052 (GREEN-BY-DESIGN) | **PASS** — SACKPopCount already implemented |

## Red Gate Verification Command

```bash
go test ./internal/arq/... 2>&1
# Expected: FAIL with panic: not yet implemented
# SACKPopCount passes by design; all others panic.
```

## Implementer Handoff (S-4.03)

Make each test pass, one at a time, with minimum code:

1. `EnqueueSend(seq, payload, now)`: record frame in inFlight map with deadline = now+dropTimeout
2. `OnAck(ackSeq, sackBitmap)`: advance nextExpected, buffer out-of-order frames (reorderBuf), flush in-order frames to DeliveredFrames, mark acked set for duplicate prevention
3. `SACKFromChannelHeader(channelHeader)`: check flags byte bit 2 at offset 8; return bitmap from bytes [12:20] when set; error if slice too short for 20 bytes when SACK_present=1
4. `TLPKTDROP(overdueSeq, now)`: check inFlight[overdueSeq] exists (ErrSequenceNotInFlight), check now > frame.deadline (ErrFrameNotOverdue), delete from inFlight, send DegradationEvent{DroppedSeq: overdueSeq}

---

# Red Gate Log — S-6.03 sbctl Client Auth

**Story:** S-6.03 v2.0 — sbctl Client Auth, Flag Parsing, JSON Envelope, and Connection Error Reporting
**Date:** 2026-06-28
**Phase:** TDD (test-writer pass)
**BC-5.38.001 Status:** RED GATE VERIFIED

## Summary

15 tests written across 2 files. 12 tests fail against current stubs
(all behavior returns `errors.New("not implemented")`). 3 tests pass
as GREEN-BY-DESIGN (the JSON envelope helpers and negative-assertion tests
are already correctly implemented in the stub or trivially satisfied).

## Test Files

| File | Tests | Status |
|------|-------|--------|
| `cmd/sbctl/client_test.go` | 11 tests (10 sub-cases + 1 helper test) | 10 FAILING, 1 GREEN-BY-DESIGN |
| `cmd/sbctl/main_test.go` | 5 tests (1 skip, 4 integration) | 3 FAILING, 1 GREEN-BY-DESIGN, 1 SKIP |

## Per-Test Red Gate Results

| Test Name | AC/VP/BC Trace | Failure Mode |
|-----------|----------------|--------------|
| `TestAuthenticate_FailClosed_VP067/VP067_a_connection_error_on_challenge_read` | VP-067(a); BC-2.07.002 PC-2 | stub returns "not implemented" — assertProtocolError rejects |
| `TestAuthenticate_FailClosed_VP067/VP067_b_malformed_challenge_json_decode_error` | VP-067(b); BC-2.07.002 PC-2 | stub returns "not implemented" — assertProtocolError rejects |
| `TestAuthenticate_FailClosed_VP067/VP067_b_malformed_challenge_missing_nonce` | VP-067(b); BC-2.07.002 PC-2 | stub returns "not implemented" — assertProtocolError rejects |
| `TestAuthenticate_FailClosed_VP067/VP067_b_malformed_challenge_nonce_not_base64url` | VP-067(b); BC-2.07.002 PC-2 | stub returns "not implemented" — assertProtocolError rejects |
| `TestAuthenticate_FailClosed_VP067/VP067_b_malformed_challenge_nonce_wrong_length` | VP-067(b); BC-2.07.002 PC-2 | stub returns "not implemented" — assertProtocolError rejects |
| `TestAuthenticate_FailClosed_VP067/VP067_e_auth_fail_returns_error` | VP-067(e); BC-2.07.002 PC-4 | stub returns "not implemented" — assertProtocolError rejects; E-ADM-010 not surfaced |
| `TestAuthenticate_FailClosed_VP067/VP067_f_wrong_response_type_returns_error` | VP-067(f); BC-2.07.002 PC-2 | stub returns "not implemented" — assertProtocolError rejects |
| `TestAuthenticate_FailClosed_VP067/VP067_g_truncated_stream_after_challenge` | VP-067(g); BC-2.07.002 PC-2 | stub returns "not implemented" — assertProtocolError rejects |
| `TestAuthenticate_FailClosed_VP067/VP067_h_oversized_auth_response_bounded_by_limit_reader` | VP-067(h); ADR-012 §6 CWE-400 | stub returns "not implemented" — assertProtocolError rejects |
| `TestAuthenticate_FailClosed_VP067/VP067_happy_path_auth_ok_returns_nil` | VP-067 happy path; BC-2.07.002 PC-2 | stub returns non-nil on AUTH_OK |
| `TestSbctl_KeyLoading_Ed25519/well_formed_ed25519_key_loads_to_64_bytes` | AC-001; BC-2.07.002 PC-2; ARCH-12 §OpenSSH Key Loading | stub returns "not implemented" |
| `TestSbctl_ConnectionRefused_ExitsOneWithENET001_VP030` | VP-030; BC-2.07.003 PC-1/PC-2; AC-004 | stub fails at key loading; error lacks target address |
| `TestSbctl_ConnectionTimeout` | BC-2.07.003 Inv-2; AC-007 | stub fails at key loading in <80ms; elapsed time assertion fails |
| `TestSbctl_AuthFailure_ExitsOneWithEADM010` | BC-2.07.002 PC-4; AC-003 | stub exits with E-NET-001 (key loading), not E-ADM-010 |

## GREEN-BY-DESIGN (pass in Red Gate, must continue passing)

| Test Name | Reason |
|-----------|--------|
| `TestSbctl_JSONEnvelopeFormat` (both sub-cases) | `newSuccessEnvelope` and `newErrorEnvelope` are fully implemented in stub |
| `TestSbctl_KeyLoading_Ed25519/file_larger_than_64KiB_is_rejected` | Stub returns error for all inputs; oversized constraint trivially met |
| `TestSbctl_KeyLoading_Ed25519/nonexistent_file_returns_error` | Stub returns error for all inputs |
| `TestSbctl_NoStdoutOnConnectionFailure` | Stub never writes to stdout |
| `TestAuthenticate_PrivKeyNeverTransmitted` | Stub never writes to the pipe; empty wire contains no key bytes |

## Red Gate Verification Command

```bash
go test ./cmd/sbctl/... -timeout 30s
# Expected: FAIL in ~1.4s
# 12 tests fail (see table above); 3 tests + 1 SKIP pass as GREEN-BY-DESIGN.
```

## Testdata Fixture

`cmd/sbctl/testdata/test_ed25519_key` — OpenSSH Ed25519 private key, 387 bytes,
generated at test-write time by a temporary Go program using `ssh.MarshalPrivateKey`.
Used by `TestSbctl_KeyLoading_Ed25519` and as the key path for all subprocess tests.

## Implementer Handoff (S-6.03)

Make each test pass, one at a time, with minimum code:

1. `loadEd25519Key(path string) (ed25519.PrivateKey, error)` — open file, `io.LimitReader(f, maxMessageBytes)`, `ssh.ParsePrivateKey`, extract `ed25519.PrivateKey` via `ssh.CryptoPublicKey`
2. `Authenticate(conn net.Conn, privKey ed25519.PrivateKey) error` — full 7-step ADR-012 handshake: read CHALLENGE, decode base64url nonce (must be exactly 32 bytes), sign nonce, send CHALLENGE_RESPONSE (pubkey + nonce_sig only, never privKey), read AUTH result via `io.LimitReader(conn, maxMessageBytes)`, return nil ONLY on auth_ok, non-nil error for all other outcomes
3. `connectAndRun` already calls these correctly — once they work, VP-030 / E-ADM-010 / timeout tests will pass
