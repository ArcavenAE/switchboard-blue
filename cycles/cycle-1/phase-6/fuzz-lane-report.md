---
artifact_id: P6-FUZZ-LANE-1
document_type: phase-6-evidence
phase: phase-6-formal-hardening
lane: fuzz-and-property
producer: formal-verifier (phase6-fuzz)
coordinator: team-lead
timestamp: 2026-07-06T03:30:00Z
develop_head_at_start: 18fd2fe
merged_as: f09fe73 (PR #105, squash)
status: complete
---

# Phase 6 — Fuzz & Property Lane Report (Burst 1)

## Scope

Five never-hardened surfaces: internal/netingress, internal/outerassembler,
internal/arqsend, internal/drain, cmd/switchboard router-config helpers.

## Fuzz harnesses (Go native, ≥90s each, ZERO crashes)

| Target | Execs | Property pinned |
|--------|-------|-----------------|
| netingress.FuzzReadFrame | 16.8M | decode never panics/over-allocates on malformed bytes |
| netingress.FuzzServeConnDispatch | 12K (ctx-bounded) | serve loop resilient to arbitrary connection bytes |
| outerassembler.FuzzChannelHeaderRoundTrip | 11.6M | header codec round-trips or fails cleanly |
| outerassembler.FuzzAssembleReadFrameRoundTrip | 10.7M | assemble→ingress-decode byte-exact round-trip |

Decoder/assembler behavior conforms to BC-2.01.004 / BC-2.01.005 /
BC-2.02.005 / BC-2.09.002 as written. No spec findings.

## Property tests (all under -race)

- arqsend (6): no-orphan-state ×100 iters, unknown-oldSeq idempotence,
  chained-retransmit monotonicity, gap-walk termination, error taxonomy,
  BC-2.02.005 PC-5 ChanSeq stamping.
- drain (6): exactly-once observer notification (32×16 concurrent), race-free
  Register+Signal (64 registrants), same-terminal-result concurrent Wait,
  Wait-before-Signal no-deadlock, monotonic timedOut, post-signal registration.

## Mutation-kill tests (from secscan lane survivors — cross-lane routing)

- TestReadFrame_MaxFrameBytesWireBound_MaxPayloadDecodes — CWE-400 (VP-066):
  formula assertion MaxFrameBytes == OuterHeaderSize + 65535 + boundary decode
  through ServeConn's LimitReader wrap.
- TestServe_MaxConcurrentConnections_SheddingCap — CWE-770 (VP-070):
  deterministic sem-full via blocking handler (128 held + observed), 3 excess
  shed with immediate read error + cap log; held-conn routed counter
  undisturbed. Mutation-kill self-check: 4/4 hypothetical mutants traced to a
  failing assertion.

## 5th target

Router-config helpers (drainTimeoutFor / keepaliveIntervalFor /
upstreamRoutersFor) already fully covered by router_config_test.go — no top-up.

## Gates at merge

-race full suite ok; smoke-quick 14/14; golangci-lint 0; Declaration +
Quality Gate + CodeQL + dependency-review green. Coordinator re-ran smoke
post-merge on develop f09fe73: 14/14.

## Incidents

Encountered the secscan lane's live mutation samples in the shared checkout
(inverted HMAC verify + gutted E-ADM-017 re-arm), correctly flagged them as
foreign security-relevant modifications, reverted safely, excluded from PR.
See secscan-lane-report.md §Incident for the coordinator lesson
(mutation sampling must be worktree-isolated).

Process positive: agent explicitly declared its two-commit deviation from the
one-commit instruction (lint fixes had raced ahead) instead of force-pushing —
the declared-divergence protocol working as intended post-CONSOLE-OBS.
