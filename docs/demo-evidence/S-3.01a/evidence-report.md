# Demo Evidence Report — S-3.01a: Tmux Control Mode Integration

**Story:** S-3.01a v1.1  
**Branch:** feature/S-3.01a-tmux-control-mode  
**Evidence type:** Go godoc Example functions (hermetic; no real tmux invoked)  
**Date:** 2026-06-26

---

## Coverage Map

| Example | AC / EC | BC Anchor | Expected Output Snippet | Pass |
|---------|---------|-----------|------------------------|------|
| `ExampleControlMode_connect` | AC-001 | BC-2.04.001 PC-1 | `connect error: <nil>` · `is ErrAlreadyConnected: true` | yes |
| `ExampleControlMode_enumerateSessions` | AC-002 | BC-2.04.001 PC-2 | `session count: 2` · `session: alpha` · `session: beta` | yes |
| `ExampleControlMode_sessionLifecycle` | AC-003 | BC-2.04.001 PC-3 + PC-4 | `session count: 1` · `session: delta` | yes |
| `ExampleControlMode_outputFramesDelivered` | AC-004 | BC-2.04.001 PC-5 | `frames received: 1` | yes |
| `ExampleControlMode_tmuxUnavailable` | EC-001 | BC-2.04.001 EC-001 / EC-004 + ADR-010 | `is ErrControlModeUnavailable: true` | yes |
| `ExamplePublisher_publishUnpublish` | Publisher lifecycle (AC-002..AC-003) | BC-2.04.001 PC-2 + PC-3 + PC-4 | `published count: 2` · `remaining count: 1` · `is ErrSessionNotFound: true` | yes |

---

## Files

| File | Package | Purpose |
|------|---------|---------|
| `internal/tmux/example_test.go` | `tmux_test` | 5 Example functions covering AC-001..AC-004 + EC-001 |
| `internal/session/example_test.go` | `session_test` | 1 Example function covering Publisher lifecycle |

---

## Verification Commands

```
go test -run "^Example" ./internal/tmux/... ./internal/session/... -v
# PASS: 6 examples, 0 failures

go test ./internal/tmux/... ./internal/session/... -race -count=1
# ok  github.com/arcavenae/switchboard/internal/tmux
# ok  github.com/arcavenae/switchboard/internal/session

just lint
# 0 issues

just fmt
# no diff
```

---

## Design Constraints Met

- No real tmux binary invoked — all examples use `WithExecFunc` to inject a
  hermetic fake stream (`exampleFakeExec` + `exampleStream`).
- Every example has a `// Output:` block pinning exact output — deterministic
  across runs.
- Each example is self-contained and idempotent.
- Source files not modified — only new `example_test.go` files added.
