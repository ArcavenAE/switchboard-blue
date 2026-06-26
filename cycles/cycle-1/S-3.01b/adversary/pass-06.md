---
artifact_id: adv-S-3.01b-pass-06
review_target: S-3.01b-pty-proxy-fallback
producer: adversary
pass: 6
fresh_context: true
branch: feature/S-3.01b-pty-proxy-fallback
base: develop @ 43208ab
tip: 3d880fd
findings_count: 5
findings_by_severity: {critical: 0, high: 1, medium: 0, low: 4, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-26
note: Returned inline by adversary; persisted via state-manager.
---

# Adversarial Review — Pass 6 — S-3.01b

## High Findings

### F-H-01 — Data race on sc.ctrl between Close (unlocked read) and watchAndFallback (locked write on reconnect success)

**Files:** `internal/tmux/pty_fallback.go:547` (Close), `:611-612` (watchAndFallback)

Close holds sc.mu only briefly (lines 527-537) to set closed=true and capture connectCancel, then drops the lock and reads sc.ctrl unprotected at line 547. Meanwhile, after a successful factory reconnect, watchAndFallback writes `sc.ctrl = newCtrl` under sc.mu. The two accesses are unsynchronized per the Go memory model — `go test -race` will flag this once a reconnect-success path coexists with a concurrent Close.

Why latent: no existing test exercises factory reconnect SUCCESS followed by Close (both factory-success tests force failures; line 612 never reached in tests).

Fix: snapshot `ctrl := sc.ctrl; pty := sc.pty` inside the same critical section that sets sc.closed = true (lines 527-537), then use snapshots at lines 547-555.

Confidence: HIGH. Severity: HIGH.

## Low Findings

### F-L-01 — control.go package-doc references stale §6.6 anchor

**File:** `internal/tmux/control.go:1-2, :9`

Package doc says "ARCH-08 §6.6 position 7" and "per ARCH-08 §6.6". Per ARCH-08 v1.6, internal/tmux was PROMOTED from §6.6 PLANNED to §6.5 CURRENT at S-3.01a merge; §6.6 is now Wave 4+ planning placeholder.

pty_fallback.go:3 correctly references §6.5; only control.go is stale.

### F-L-02 — ptyAllocFunc docstring inaccurate vs actual signature and implementation

**File:** `internal/tmux/pty_fallback.go:53-60`

Doc says "calls golang.org/x/sys/unix.Openpty" — but pty_alloc_darwin.go uses raw `syscall.SYS_IOCTL` with TIOCPTYGNAME; pty_alloc_linux.go uses unix.IoctlSetInt(TIOCSPTLCK) + unix.IoctlGetUint32(TIOCGPTN). Neither calls unix.Openpty.

Doc also says signature is `(masterFD, slaveFD io.ReadWriteCloser, pid int, err error)` — actual signature is `(master io.ReadWriteCloser, pid int, err error)`. No slave returned.

### F-L-03 — Story spec compliance-table cites test names that diverge from actual

**File:** `.factory/stories/S-3.01b-pty-proxy-fallback.md:121-122`

References `TestPTYProxy_FallbackOnMidSessionLoss` and `TestPTYProxy_NoAutoUpgrade`. Actual: `TestSessionConnector_MidSessionFallback_ReconnectAttempts` and `TestSessionConnector_NoAutoUpgrade_AfterFallback`. Behavior covered but compliance table cannot be greppable against the code.

### F-L-04 — errors.Is(ErrControlModeBinaryNotFound) branch untested

**File:** `internal/tmux/pty_fallback.go:477-478`

Production wrap path in defaultExecFn (control.go:153) emits `fmt.Errorf("%w: %w: %w", ErrControlModeUnavailable, ErrControlModeBinaryNotFound, lookErr)` — satisfies this branch. But the only EC-002 test wraps only ErrControlModeUnavailable + literal "no such file" substring, traversing the string-match fallback. Production errors.Is branch covered only incidentally.

## Observations

- Pass-5 fixes H-01/H-02/H-03/M-01/M-02/M-03/L-02 all verified in place.
- Orphan top-of-file comment in pty_fallback.go between `package tmux` and `import` is non-idiomatic but golangci-lint doesn't flag; not a finding.

## Resolution decisions (mechanical)

- H-01: snapshot ctrl/pty under sc.mu before unlocking, then use snapshots in Close cleanup.
- L-01: update control.go package doc anchors §6.6 → §6.5.
- L-02: fix ptyAllocFunc docstring to match actual signature + implementation (raw ioctls; (master, pid, err) — no slave).
- L-03: align story compliance table test names with actual test names.
- L-04: add test that injects an EC-002 error wrapping ErrControlModeBinaryNotFound via errors.Is path.

## Novelty Assessment

Novelty: MEDIUM. F-H-01 is genuine concurrency defect untouched by listed pass-5 fixes. F-L-01..L-04 are doc/test-anchor accuracy items.
