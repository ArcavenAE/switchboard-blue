---
artifact_id: adv-S-3.01b-pass-05
review_target: S-3.01b-pty-proxy-fallback
producer: adversary
pass: 5
fresh_context: true
branch: feature/S-3.01b-pty-proxy-fallback
base: develop @ 43208ab
tip: 3c18590
findings_count: 8
findings_by_severity: {critical: 0, high: 3, medium: 3, low: 2, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-26
note: Returned inline by adversary; persisted via state-manager.
---

# Adversarial Review — Pass 5 — S-3.01b

## High Findings

### H-01 — SysProcAttr.Ctty set to parent FD instead of child FD

**Files:** `internal/tmux/pty_alloc_darwin.go:77`, `internal/tmux/pty_alloc_linux.go:66`

Both files set `Ctty: int(slave.Fd())`. `slave.Fd()` returns the parent process FD number for /dev/ptmx's slave (typically 7+). `SysProcAttr.Ctty` is interpreted in the **child's** descriptor namespace. Since cmd.Stdin=slave; Stdout=slave; Stderr=slave, the child sees slave on FDs 0/1/2 — so `Ctty: 0` is the correct value.

Effect: in production, cmd.Start may fail (EBADF) or succeed without correctly making slave the controlling tty. Shell child runs without controlling terminal; SIGHUP-on-master-close semantics break; child process leak.

Not caught by unit tests because WithPTYAllocFunc injects fakes. VP-032 would catch.

### H-02 — Child shell process leak; PTYProxy.Close cannot signal child

**Files:** `internal/tmux/pty_alloc_darwin.go:90`, `pty_alloc_linux.go:79`, `pty_fallback.go:292-322`

Shell spawned via cmd.Start; only reaped via `go func() { _ = cmd.Wait() }()`. The *exec.Cmd reference is captured ONLY by the reaper goroutine; NOT returned to PTYProxy.

PTYProxy.Close closes master FD but cannot send SIGTERM/SIGKILL. If child shell ignores SIGHUP or (combined with H-01) has no controlling tty, Close returns while shell continues orphaned.

CRITICAL criterion ("child process leak") borderline; classified HIGH because failure is observable (orphan accumulation) not immediate crash.

### H-03 — go.mod lists golang.org/x/sys as // indirect but it's directly imported

**File:** `go.mod:5`

`require golang.org/x/sys v0.46.0 // indirect`. But `pty_alloc_linux.go:12` directly imports `golang.org/x/sys/unix`. `go mod tidy` would correct; CI gates may flag.

## Medium Findings

### M-01 — classifyCh-monitoring goroutine leaked across ControlMode.Close

**File:** `internal/tmux/control.go:321-333`

Goroutine spawned but NOT registered with c.wg (only dispatchLoop is at line 311). On Close, c.wg.Wait returns once dispatchLoop exits but classification goroutine may still be blocked on <-classifyCh. The reaper eventually closes classifyCh, but there's a window where the goroutine is leaked relative to documented "Close returns" lifecycle boundary.

go test -race with goleak would flag.

### M-02 — ClassifyStderr over-broad patterns produce false positives

**File:** `internal/tmux/stderr.go:16-20`

`strings.Contains(captured, "-C")` matches ANY occurrence in stderr — including tmux's own usage output (always documents `[-C]` as valid flag), unrelated error tokens, session names containing "-C". Other patterns ("unknown option", "invalid option") also generic.

Effect: unrelated tmux error → prints usage → contains [-C] → classified as ErrControlModeUnsupportedFlag instead of ErrControlModeDropped/Unavailable → wrong log + possibly wrong fallback.

Unit tests only exercise positive cases. A more targeted regex would limit false positives.

### M-03 — Publish-failure path leaves p.pid and p.sessionName set

**File:** `internal/tmux/pty_fallback.go:200-209`

When publisher.Publish fails non-idempotently, code closes master and nils p.master, but p.pid and p.sessionName remain set; p.closed not set.

Subsequent Connect would pass guards and overwrite. Subsequent Close calls Unpublish(sessionName) even though Publish failed — relying on ignore-ErrSessionNotFound path. Loose state-machine; latent regression risk.

## Low Findings

### L-01 — Orphaned godoc comment at end of pty_fallback.go

**File:** `internal/tmux/pty_fallback.go:668-675`

Doc comment describing defaultPTYAlloc has no symbol below it. The function defined in platform files. Confusing to readers; lint tools may flag dangling doc comment.

### L-02 — AC-002 log assertion too loose

**File:** `internal/tmux/pty_fallback_test.go:229`

`wantLogSubstr = "Functionality limited"` (21 chars). Canonical message is ~150 chars: "tmux control mode unavailable; using PTY proxy mode. Functionality limited: no structured session metadata, no content-type detection."

Test would pass if message were rewritten to anything containing "Functionality limited". Regression-detection coverage of canonical wording incomplete.

## Observations

- ARCH-08 §6.5 line 253 describes internal/tmux allowed imports as {halfchannel, session}. New external import golang.org/x/sys/unix is allowed (external deps not §6.5-scoped) but worth recording in a future spec patch.
- closeErrCh.Do default arm has misleading comment about "buffered-1 invariant" but sync.Once.Do guarantees first-execution semantics.

## Resolution decisions (mechanical)

- H-01: change Ctty to 0 in both pty_alloc_darwin.go and pty_alloc_linux.go.
- H-02: retain *exec.Cmd in defaultPTYAlloc return path; pass through to PTYProxy; Close calls cmd.Process.Kill() before closing master.
- H-03: run `go mod tidy` to remove // indirect marker.
- M-01: register classifyCh goroutine with c.wg.
- M-02: tighten ClassifyStderr patterns (regex or more-specific substrings; add negative test cases).
- M-03: clear p.pid and p.sessionName on publish failure.
- L-01: relocate or remove orphaned godoc.
- L-02: assert full canonical message.

## Novelty Assessment

Novelty: MEDIUM-HIGH. New cross-platform PTY allocation surface (pass-4 L-005) introduced real production defects (H-01 Ctty + H-02 cmd retention) that weren't visible from the test surface (fakes injected). Deep systems analysis paid off.
