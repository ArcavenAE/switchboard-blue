# Demo Evidence Report — S-3.01b: PTY Proxy Fallback

**Story:** S-3.01b v1.2  
**Branch:** feature/S-3.01b-pty-proxy-fallback  
**Evidence type:** Go godoc Example functions (hermetic; no real tmux or PTY device invoked)  
**Date:** 2026-06-26

---

## Coverage Map

| Example | AC / EC | BC Anchor | Expected Output Snippet | Pass |
|---------|---------|-----------|------------------------|------|
| `ExampleSessionConnector_initialFallback` | AC-001 | BC-2.04.002 PC-1 + ADR-010 | `connect error: <nil>` · `in PTY mode: true` · `session name: pty-42` | yes |
| `ExamplePTYProxy_publishSession` | AC-002 | BC-2.04.002 PC-2 + PC-3 | `session name: pty-99` · `mandatory log present: true` | yes |
| `ExamplePTYProxy_bothUnavailable` | AC-003 | BC-2.04.002 EC-004 | `is ErrPTYDeviceUnavailable: true` · `operator guidance logged: true` | yes |
| `ExampleSessionConnector_oldTmuxFallback` | EC-001 | BC-2.04.002 EC-001 + ADR-010 | `connect error: <nil>` · `in PTY mode: true` | yes |
| `ExampleSessionConnector_midSessionFallback` | EC-003 | BC-2.04.002 EC-003 + ADR-010 | `connect error: <nil>` · `in PTY mode: true` · `fallback log present: true` | yes |

---

## Files

| File | Package | Purpose |
|------|---------|---------|
| `internal/tmux/example_test.go` | `tmux_test` | 5 new Example functions covering AC-001..003 + EC-001 + EC-003; appended to S-3.01a's 5 pre-existing examples |

---

## Verification Commands

```
go test -run "^Example" ./internal/tmux/... -v
# PASS: 10 examples (5 S-3.01a + 5 S-3.01b), 0 failures

go test ./internal/tmux/... -race -count=1
# ok  github.com/arcavenae/switchboard/internal/tmux

just lint
# 0 issues

just fmt
# no diff
```

---

## Design Constraints Met

- No real tmux binary or PTY device invoked — `WithExecFunc` injects hermetic
  fake streams; `WithPTYAllocFunc` injects an `os.Pipe()` that immediately
  closes its write end so `ioRelay` observes EOF and exits cleanly.
- Every example has a `// Output:` block pinning exact output — deterministic
  across runs.
- Each example is self-contained and idempotent.
- `exampleCapturingLogger` captures mandatory log lines for assertion without
  writing to stderr; satisfies `tmux.Logger` interface.
- Source implementation files not modified — only `example_test.go` extended.
- Pre-existing S-3.01a examples (5) continue to pass alongside new S-3.01b
  examples (5); total 10 examples in `internal/tmux/example_test.go`.
