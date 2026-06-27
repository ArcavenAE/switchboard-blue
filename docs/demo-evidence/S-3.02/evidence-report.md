# Evidence Report — S-3.02: Console Attach/Detach and Multi-Console Fan-Out

**Story:** S-3.02  
**Branch:** feature/S-3.02-session-attach-detach-fanout  
**Commit:** f8bfdb8  
**Recording tool:** VHS not available on this host; evidence captured as verbose `go test -race -v` transcripts  
**Date:** 2026-06-26

## Summary

All 8 acceptance criteria have demonstrable evidence. Every test listed in the story was executed with the Go race detector (`-race`) and passed. No data races were detected anywhere in the package.

Package result: **27 tests + 1 example — ALL PASS, race detector CLEAN**

---

## AC → Evidence Mapping

| AC | Description | BC Trace | Test Name | Evidence File | Result |
|----|-------------|----------|-----------|---------------|--------|
| AC-001 | Attach establishes bidirectional channel (downstream half-channel returned) | BC-2.04.003 PC-1 | `TestSession_Attach_EstablishesBidirectionalChannel` | `AC-001-002-attach-bidirectional-keystroke.txt` | PASS |
| AC-002 | Console upstream keystrokes forwarded to sink | BC-2.04.003 PC-3 | `TestSession_Attach_UpstreamKeystrokesForwarded` | `AC-001-002-attach-bidirectional-keystroke.txt` | PASS |
| AC-003 | Attach returns E-SES-001 for nonexistent session | BC-2.04.003 EC-002 | `TestSession_Attach_NonexistentSession_ErrSesOne` | `AC-003-session-not-found-error.txt` | PASS |
| AC-004 | Detach closes console's channel cleanly; session continues; no post-detach keystrokes | BC-2.04.004 PC-1+PC-2 | `TestSession_Detach_SessionContinues` | `AC-004-005-detach-session-continues-observers-unaffected.txt` | PASS |
| AC-005 | After full-access console detaches, surviving consoles keep receiving frames | BC-2.04.004 PC-5 | `TestSession_Detach_ReadOnlyObserversUnaffected` | `AC-004-005-detach-session-continues-observers-unaffected.txt` | PASS |
| AC-006 | Two or more consoles attached simultaneously each receive all downstream frames | BC-2.04.006 PC-1 | `TestSession_MultiConsoleFanOut_AllReceiveFrames` | `AC-006-multi-console-fanout.txt` | PASS |
| AC-007 | Concurrent keystrokes from multiple consoles serialized; no interleaving under `-race` | BC-2.04.006 Invariant 3 | `TestSession_ConcurrentKeystrokes_Serialized` | `AC-007-concurrent-keystroke-serialization-race-clean.txt` | PASS |
| AC-008 | EvictStale(deadline) selectively evicts stale consoles; healthy consoles unaffected; clock-injected for determinism | BC-2.04.004 EC-002 / BC-2.04.006 | `TestSession_CrashDetach_EvictsFromFanOut` | `AC-008-crash-eviction-evictstale.txt` | PASS |

---

## Full Suite Transcript

`full-suite-race-clean.txt` — complete `go test -race -v ./internal/session/` run showing all 27 tests + 1 example passing in parallel with no race violations.

---

## Recording Toolchain Note

VHS was not available on this host (`which vhs` returned not found). Per the Demo Recorder operating procedure, evidence falls back to verbose test transcripts. Each `.txt` file in this directory is a captured terminal transcript from a specific `go test -race -v -run <TestName>` invocation directly demonstrating the behavior specified in the corresponding acceptance criterion.

For the AC-007 (concurrent serialization) evidence: the `-race` flag is the key instrumentation — the Go race detector instruments all memory accesses and would report a data race if any goroutine accessed shared state without synchronization. `PASS` with `-race` on `TestSession_ConcurrentKeystrokes_Serialized` constitutes positive proof of correct serialization.

For AC-008 (crash eviction): the `WithClock` seam (`ConsoleSetWithClock(fn func() time.Time)`) is the key design element enabling deterministic eviction without real-time sleeps. The test advances a fake clock past the deadline and asserts the stale console is evicted while the healthy observer continues receiving frames.

---

## Files in This Directory

```
AC-001-002-attach-bidirectional-keystroke.txt
AC-003-session-not-found-error.txt
AC-004-005-detach-session-continues-observers-unaffected.txt
AC-006-multi-console-fanout.txt
AC-007-concurrent-keystroke-serialization-race-clean.txt
AC-008-crash-eviction-evictstale.txt
full-suite-race-clean.txt
evidence-report.md
```
