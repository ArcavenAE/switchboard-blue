---
artifact_id: pass-12
document_type: adversarial-review
story_id: S-7.04-FU-SIGHUP-RELOAD
pass: 12
verdict: NO_FINDINGS
code_sha: 48e3271
story_version: "1.7"
novelty: LOW
streak_before: 0
streak_after: 1
streak_note: "0/3 → 1/3 — first clean since pass 6; pass-6's 1/3 was reset by pass-7"
anti_findings_count: 12
findings_count: 0
observations_count: 4
---

# Adversarial Review — S-7.04-FU-SIGHUP-RELOAD Pass 12

**Code lane:** 48e3271 (10th consecutive pass with zero code findings)
**Story version at review:** v1.7
**Verdict:** NO_FINDINGS

---

## Observations (non-actionable)

**O-P12-1 (drainCoord-untouched-no-direct-observable):** `drainCoord` is created and wired
in `runRouter` but the SIGHUP-reload path never calls `drainCoord.RegisterObserver` or any
drain-side method. The hook is intentionally absent — no per-node identity exists yet; observer
registration and its panic-recovery obligations are S-7.04-FU-DRAIN-WIRE era work. Anchored to
that story stub; non-defect.

**O-P12-2 (VP-038-test-thin-by-design):** `TestRunRouter_VP038_EtoPEViaConfigOnly` drives a
config reload through `SetSighupCh` but does not assert that downstream transport goroutines
actually reconnect to the new upstream. This thinness was deliberate — per v1.2 deferral
(VP-038 activation deferred to PE-CONNECTOR, which supplies the live dial loop) and v1.5's
explicit note that the test validates the config-propagation path only. Non-defect; scope is
intentionally bounded to the signal→config-update path.

**O-P12-3 (inert-non-upstream-reload):** A valid SIGHUP that changes only non-upstream fields
(e.g. `drain_timeout`, `keepalive_interval`) is silently processed — no observable feedback to
the operator that the reload succeeded. This is the already-tracked
`DRIFT-SIGHUP-INERT-RELOAD-UX` drift item (LOW, anchor S-BL.CLI-SURFACE-COMPLETION). No new
item warranted; folds into the existing row.

**O-P12-4 (isNetError-idiom):** `router_sighup_test.go` uses `//nolint:errorlint` on the
`isNetError` type-assertion path. The nolint annotation is justified — `net.Error` is an
interface type-assertion for timeout/temporary semantics, not an `errors.Is` chain; the linter
suppression is deliberate and documented in the inline comment. Non-defect.

---

## Anti-Findings (12)

1. **Fail-closed atomicity (all reload paths incl. non-empty immutability):** `upstreamRouters`
   is assigned only after `LoadFile` + `Validate` + `equalStringSlices` diff all succeed;
   non-empty deep-copy asserts in `TestRunRouter_SIGHUPReload_PEtoPE` and
   `TestRunRouter_SIGHUPReload_EtoPE` verify that the original slice remains unaffected after
   the reload mutates the local copy — pass-8 F-P8-001 closure holds under re-verification.

2. **EC-004 verbatim single-line with control-char-strip robustness:** The error-emission path
   uses `fmt.Fprintf(stderr, "config reload failed: %s; continuing with previous config\n", err)`
   matching the spec exactly; the `--config` path is stripped of Unicode control chars before
   interpolation (S601-SEC-001 closure), ensuring the EC-004 line is a single parseable record
   even for adversarially crafted config paths.

3. **Emission byte-parity:** Both happy-path mode-line emissions (`modeELine` / `modePELine`)
   and the reload failure emission are asserted character-for-character by the
   `scanForExactModeLine` / `modeELine` / `modePELine` helpers introduced in pass-4 F-P4-001;
   the helpers are verified present and unchanged at 48e3271.

4. **Q1 real-signal guard:** `TestRunRouterRun_RealSIGHUP_DoesNotExit` (pass-5 F-P5-001
   closure) sends a genuine `syscall.Kill(os.Getpid(), syscall.SIGHUP)` through the live
   `run()` entrypoint and asserts the daemon does not exit — OS-signal delivery path remains
   covered; the channel-vs-OS-signal distinction is exercised.

5. **Race-clean:** `just test-race` clean at 48e3271 — no data races detected across all 10
   tests in `router_sighup_test.go`. The `sighupCh` cap-1 buffered channel pattern eliminates
   signal-delivery races; `upstreamRouters` is single-goroutine-local through all currently
   merged paths.

6. **Diff-guard all transitions incl. nil==empty:** `equalStringSlices` correctly treats both
   `nil` and `[]string{}` as empty (anti-finding 6 from pass-11 re-verified independently);
   PEtoPE, EtoPE, PEtoE, and idempotent-resend transitions all route through this guard.

7. **Signal directionality:** `signal.Notify` registers only `syscall.SIGHUP` on `sighupCh`;
   termination signals route exclusively through `signal.NotifyContext`; no cross-signal
   conflation present.

8. **Three fail-closed arms:** LoadFileNotFound, MalformedYAML, and InvalidUpstreamAddr_FailClosed
   each assert that the daemon emits EC-004 and retains its previous config without mode
   transition — all three failure-class tests green at 48e3271.

9. **FCL 8-row independent re-sweep — all accurate:** The pass-11 class-closure upgrade
   (full-surface sweep baseline) holds under independent re-verification at code SHA 48e3271:
   - `cmd/switchboard/main.go`: sighupCh cap-1 + Notify + defer Stop + runRouter call ✓
   - `cmd/switchboard/mgmt_wire.go`: two-case select + reload dispatch ✓
   - `cmd/switchboard/router_config.go`: equalStringSlices helper ✓
   - `cmd/switchboard/router_sighup_test.go`: ten tests ✓
   - `cmd/switchboard/main_test.go`: RealSIGHUP_DoesNotExit ✓
   - `cmd/switchboard/mgmt_wire_test.go`: five call-site updates ✓
   - `cmd/switchboard/router_drain_test.go`: existing call-sites updated ✓
   - `internal/testenv/testenv.go`: SetSighupCh post-hoc setter + 1-arg SendReloadSignal ✓
   All 8 rows match. No prose-shape drift remaining.

10. **POL-001/002/004 compliance:** All four ACs trace to BC-2.09.001 PC-1 and BC-2.09.003
    EC-004; story v1.7 version is consistent with STORY-INDEX v4.00 backlog row; no new
    internal package introduced (ARCH-08 registration obligation correctly absent).

11. **AC-003 positive liveness — both probes:** `TestRunRouter_SIGHUPReload_SessionsNotInterrupted`
    (AC-003) holds a live TCP connection through the reload and asserts (a) the connection
    remains alive via a read probe, and (b) the goroutine running `runRouter` does not return —
    the no-return `select{}` assert added in pass-9 F-P9-001 closure verified present and
    exercising both liveness probes.

12. **go.md hygiene:** No `interface{}`/`any` without justification; all error returns checked;
    no `log.Fatal`/`os.Exit` outside `main()`; context.Context first-param discipline observed
    in the reload path; nolint annotations carry inline rationale comments.

---

## Novelty Assessment

**LOW** — "confirmation, not discovery." All 12 anti-findings independently re-derive and
confirm prior-pass closures. The four observations surface no new defect classes: O1 is an
intentional architectural absence (DRAIN-WIRE era), O2 is a recorded scope deferral, O3 folds
into an existing drift row, and O4 is a justified nolint. The full-surface FCL sweep from pass-11
holds without exception — the class-closure technique introduced in pass-11 has reached steady
state.

## Streak

Before: 0/3
After: 1/3 (first clean since pass 6; pass-6's 1/3 was reset by pass-7 process-gap finding)
Next pass: 13
