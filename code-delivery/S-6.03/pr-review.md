# PR #32 Review — S-6.03 sbctl client auth

**Verdict: APPROVE**

Fresh-eyes review of the diff, PR description, and tests. Reviewed all 9 changed
files line-by-line. No blocking findings. The implementation is correct against the
stated contracts; security properties (fail-closed, bounded reads, no private-key
transmission) hold; wire-protocol semantics match ADR-012 §3 step 6; error-code
segregation is clean. Findings below are LOW / INFO only.

## What I verified (no rubber-stamp)

- **Fail-closed `Authenticate()`**: returns `nil` ONLY on `type:"auth_ok"`. Every
  other branch — challenge read error, missing/invalid/wrong-length nonce, encode
  failure, auth-result read error, `auth_fail`, unknown type — returns non-nil.
  The terminal `switch` has an explicit `default` that rejects unknown types. Good.
- **Bounded reads (CWE-400)**: every decode wraps the conn in
  `io.LimitReader(conn, maxMessageBytes)` — both handshake reads and the dispatch
  response. Key file read uses `io.LimitReader(f, maxMessageBytes+1)` with an
  explicit `len(raw) > maxMessageBytes` overflow check. Correct boundary handling.
- **Private key never transmitted (DI-002)**: only `privKey.Public()` (32-byte
  Ed25519 public key) and the nonce signature go on the wire. The seed is never
  serialized. `TestAuthenticate_PrivKeyNeverTransmitted` asserts neither base64url
  nor base64std encoding of the seed appears in captured bytes.
- **Wire types (ADR-012 §3 step 6)**: `dispatch()` emits `type:"request"`,
  requires `type:"response"` on the reply (Ruling U), and the type check precedes
  the `ok` check (guard ordering locked by a dedicated test).
- **ID echo (Ruling X)**: request `id` is per-call non-constant
  (`fmt.Sprintf("%x", time.Now().UnixNano())`); `dispatch()` rejects a mismatched
  `resp.ID`. Verified by both the mismatch and non-constant sub-cases.
- **Read deadlines (Ruling V)**: `Authenticate()` and `dispatch()` both derive the
  deadline from `ctx.Deadline()` with a fallback (10s / 30s), call
  `conn.SetReadDeadline`, and `dispatch()` clears it via `defer SetReadDeadline(time.Time{})`.
  The deadline-clear is verified by a `recordingConn` wrapper.
- **Error-code segregation**: E-CFG-010 (key load, before any dial), E-NET-001
  (dial failure + auth-time net timeout), E-ADM-010 (AUTH_FAIL), E-RPC-001
  (post-auth dispatch). Subprocess tests assert each code appears in isolation
  (e.g. key-load failure must NOT emit E-NET-001, proving load-before-dial ordering).
- **Go idioms**: `context.Context` is first param on `Authenticate`/`dispatch`;
  no `os.Exit`/`log.Fatal` outside `main()` (`connectAndRun` returns error, the
  subprocess entrypoint maps to exit codes only inside the re-exec'd child);
  errors wrapped with `%w`; `time.Now().UTC()` for deadline math; `fmt.Fprintf`
  used for stderr writes. Clean.
- **Test reliability**: `t.Fatalf`-bearing helpers (`wellFormedChallenge`,
  `freshNonce`) are hoisted into the test goroutine before `go func()` (Ruling W).
  Goroutine-internal failures use channels / non-fatal paths. `homeDir` injected
  per-call (no shared global) so tilde tests are `-race`-safe. The one test that
  mutates the package var `rpcResponseFallbackTimeout` is deliberately NOT
  `t.Parallel()` at the top level and restores via `t.Cleanup`. Correct.

## Findings

| # | Severity | Category | Finding | Suggestion |
|---|----------|----------|---------|------------|
| 1 | LOW | coherence | `Authenticate()` doc comment (step 4) states "daemon_sig is decoded above but NOT verified". `daemon_sig` is never base64-decoded — it is only unmarshaled as a struct string field and otherwise unused. The "decoded" wording overstates what the code does. | Reword to "daemon_sig is received but neither decoded nor verified (TOFU deferral)" to match the code. Non-blocking; the deferral itself is a settled ruling. |
| 2 | INFO | coherence | `errorDetail` carries a `Field any` JSON tag (`"field"`) that `newErrorEnvelope` never populates, so error envelopes always serialize `"field":null`. `TestSbctl_JSONEnvelopeFormat` only asserts `code`/`message`, so an unintended `field` key would pass unnoticed. | Confirm `interface-definitions.md` JSON Output Schema expects (or tolerates) `field`. If not, add `omitempty` or drop the field. Cannot verify against the schema from the diff alone. |
| 3 | INFO | coherence | `go.mod` adds `golang.org/x/crypto v0.53.0` in its own trailing `require` block rather than merged into an existing block. `go mod tidy` typically consolidates these. Lint reportedly passed, so this is cosmetic. | Optionally run `go mod tidy` to consolidate. No functional impact. |
| 4 | INFO | coherence | The diff includes one non-code artifact: `.factory/cycles/.../red-gate-log.md` (+74). It is internal pipeline state rather than shippable product code. | Confirm intent to commit pipeline logs to the product repo. Not a correctness issue. |

## Checklist

1. Diff coherence — PASS (one pipeline-log file noted, finding #4)
2. Description accuracy — PASS (body matches: cmd/sbctl only, no daemon/internal/mgmt)
3. Test coverage — PASS (all 12 ACs + VP-067/VP-030 + Rulings U/V/X/W covered)
4. Demo evidence — N/A here (CLI story; deferred live happy-path is a settled ruling to S-W5.02)
5. Commit quality — PASS (22 commits, conventional format, story IDs, clear messages)
6. Diff size — 2634 additions, but ~2066 are tests + fixtures; production code is
   `client.go` (375) + `main.go` (104) = 479 lines. Acceptable for the AC count.
7. Missing changes — none detected; scope matches story
8. Dependency status — client-only; no upstream merge dependency

## Settled rulings NOT re-litigated

daemon_sig TOFU deferral; live-daemon happy-path deferred to S-W5.02; CWE-400
write-deadline deferred to S-HRD.01. Out of scope per story spec v2.6.
