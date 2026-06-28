---
artifact_id: adjudication-W3-R3-F1
document_type: adjudication
finding_id: W3-R3-F1
finding_label: cmd-wiring
status: RESOLVED
disposition: RESOLVED
owner: architect
adjudicated_at: 2026-06-27
adversary_pass: wave-3/adversary/pass-r3.md
adversary_tree: develop @ 10dd880
current_tree: develop @ 144d094
arch_version_changed: false
---

# Adjudication: W3-R3-F1 — CMD Wiring / nopLogger in Production Builds

## Finding Restatement

Pass-r3 (run against develop @ `10dd880`) found:

> **F-1 (HIGH)** — `cmd/switchboard/main.go` is a version-printing stub. None of
> the 5 Wave-3 subsystems are referenced anywhere in `cmd/`. The production
> `Logger` for `RouteFrame` E-ADM-016 paths is never instantiated; the `nopLogger`
> inside `routing.Router` is therefore the only logger present in any real build.
> The `AccessNode.Sweep` eviction ticker is never started. "Wave 3 as a working
> whole" is not demonstrable from the entrypoint.

Prior passes (r1, r2) rated this OBSERVATION / in-scope-deferred. Pass-r3 escalated
to HIGH and flagged it as ADJUDICATION-DEPENDENT because ARCH-08 placed
`cmd/switchboard` at "position 18 / target–planned" with no Wave-3 story explicitly
scoping the wiring.

## Code Evidence

### At the time of the adversary run (10dd880)

The adversary correctly observed that `cmd/switchboard/main.go` at `10dd880`
contained only a version-printing stub:

```
cmd/switchboard/main.go:12-28 — no references to Router / AccessNode /
SessionConnector / SessionAuth / ControlMode / PTYProxy
```

This was the last commit before S-W3.04 was dispatched. The story S-W3.04
(`S-W3.04-daemon-assembly.md`, status: `ready`, v1.0 dated 2026-06-27) explicitly
lists W3-R3-F1 as the trigger and first line of its classification block:

> "Closes drift items W3-R3-F1, W3-M-3, W3-R2-M3, W3-R2-M4, and the
> SessionConnector half of W3-M-2."

### Current state (develop @ 144d094)

S-W3.04 has been fully implemented and merged via a chain of PRs:

| PR / Commit | Content |
|-------------|---------|
| aeb442d (PR #17) | `feat(S-W3.04)`: full daemon assembly — all Wave-3 subsystems wired |
| e9421d8 (PR #18) | `fix(access)`: join ticker goroutines into WaitGroup (I-1 fix) |
| 418de54 (PR #20) | `feat(access)`: wire HMAC failure counter into daemon router |
| 849bd86 (PR #19) | `test(tmux)`: TOCTOU misclassification branch regression test |

The current `cmd/switchboard/` contains:

- `cmd/switchboard/main.go` — subcommand dispatch; `case "access": → runAccess(ctx, os.Stderr)` (line 48–55). The `"access"` case wires a real `signal.NotifyContext` and delegates to `runAccess`.
- `cmd/switchboard/access.go` — all six ARCH-08 §6.5.1 wiring obligations implemented:
  1. `buildRouter(keys, stdLogger{log.New(stderr, "", 0)})` — real `log.Logger` wrapped as `stdLogger` satisfying `routing.Logger`; NOT a nop (access.go:105, 305–308)
  2. `session.NewAccessNode(pub, auth, session.WithKeystrokeSink(sc))` with live `*session.SessionAuth` (access.go:279–281)
  3. `startSweepTicker` called with `wg`, real `time.Ticker` (access.go:219–221)
  4. `startFramesBridge` goroutine: `sc.Frames() → an.DeliverFrame` (access.go:211–215)
  5. `sc.Err()` drain goroutine tracked in `sync.WaitGroup`; mid-session failure → E-SYS-002 + cancel (access.go:188–208)
  6. `startFramesDroppedTicker` emitting `"frames_dropped relay=<N> consoles=<M>"` (access.go:226–228)
  7. `admission.NewFailureCounter(5, 60s, rl)` wired via `routing.WithFailureCounter(fc)` in `buildRouter` (access.go:305–308; PR #20)

The E-ADM-016 logger path: `routerLogger := stdLogger{log.New(stderr, "", 0)}` at
`access.go:105` — this is NOT a nop. It wraps the stdlib `log.Logger` pointed at
`os.Stderr` (the real process stderr in production). When `RouteFrame` is called
with an HMAC-bad frame, the log event reaches `os.Stderr` unconditionally.

The AC-001 test `TestRouterLoggerEmitsEADM016` in `cmd/switchboard/main_test.go`
exercises the daemon's own router instance (the one returned by `buildRouter` inside
`runAccess`) — it is non-tautological because the router shares the daemon's single
`*admission.AdmittedKeySet`.

## Disposition

**RESOLVED**

W3-R3-F1 is fully resolved in the current codebase. The finding was a legitimate
observation against develop @ `10dd880`, where `cmd/switchboard` was still a stub.
Story S-W3.04 was the planned vehicle for closing it (the story's v1.0 classification
block explicitly named W3-R3-F1 as the trigger). S-W3.04 merged via PR #17 (aeb442d)
plus follow-up fixes in PRs #18 and #20, all of which are present on develop.

There is no gap between the finding and the current state of the codebase.

## Rationale

The adversary ran against `10dd880`, which was the pre-S-W3.04 stub. S-W3.04 was
already written and ready at that point — it was in the Wave-3 story queue waiting
for dispatch. The escalation from OBSERVATION to HIGH in pass-r3 was correct given
that no story was yet in-flight at the time of the pass; the ADJUDICATION-DEPENDENT
flag was appropriate because the question was "is this scope-deferred or a real gap."

The answer was: it was scope-deferred to S-W3.04, and S-W3.04 has now landed.

The overlap with "S-BL.NI network-ingress deferral" noted in the drift context is
also confirmed: ARCH-08 v2.3 §6.5.1 explicitly records that the only remaining
deferral at the `cmd/switchboard` boundary is the **network-ingress listener**
(story S-BL.NI). The router is constructed with a live logger and FailureCounter;
it simply has no live non-test caller for `RouteFrame` until S-BL.NI lands. This
is an accepted, documented deferral — not a wiring gap.

## ARCH-08 Version Pin Update

No ARCH-08 version update required. ARCH-08 v2.3 already records the C-1
resolution (WithFailureCounter landed in PR #20 / 418de54) and the S-BL.NI
remaining deferral. The architectural decisions did not change as a result of
this adjudication; the code simply caught up to the spec.

## Wave-4 Story Required?

No. This finding requires no Wave-4 story. The wiring is complete. The S-BL.NI
network-ingress story is already registered in the backlog (referenced in ARCH-08
v2.3 §6.5.1 and `S-BL.OA-outer-assembler.md`); it is the natural successor but
is not created by this adjudication.
