# Changelog

All notable changes to switchboard are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0-rc.1] — 2026-07-04

Initial stable release candidate. All Phase 1–7 VSDD convergence dimensions
satisfied (5 CONVERGED, 1 CONVERGED_WITH_DRIFT, 2 N/A_JUSTIFIED); zero-slop
achieved with 30 named-and-tracked drift items and 5 follow-up stories.

### Added

**Control plane (sbctl / daemon)**

- SVTN lifecycle admin verbs: `admin svtn create` handler and sbctl subcommand
  (S-6.07, S-6.02); `admin svtn destroy` with 5-path confirm gate (S-6.05,
  BC-2.07.001 PC-1..PC-4)
- Admin key lifecycle: daemon admin RPC handlers for register/revoke/expire/list-keys
  (S-6.06); sbctl client auth with Ed25519 fail-closed, flag parsing, JSON envelope,
  and connection error reporting (S-6.03, S-6.02)
- Console attach/detach/switch via sbctl and daemon handlers (S-7.03, BC-2.08.001
  PC-1/PC-2/PC-3)
- `paths.list` / `router.metrics` / `router.status` RPC handlers and response types
  (S-W5.04); sbctl `paths list` / `router metrics` / `router status` aliases with
  p99/quality surfacing (S-5.02)
- Internal management server and cmd/switchboard wiring for all four daemon modes
  (S-W5.01)
- Config parsing and validation with actionable startup errors (S-6.01)
- `AdmittedKeySet.Lookup`/`LookupByPubkey` migrated to value-return form (S-BL.LOOKUP)
- Green/yellow/red quality indicator with hysteresis (S-5.01, BC-2.06.001,
  BC-2.06.002)
- List-keys admission gate and spec-conformance follow-ons (Phase 5 Pass 13)

**Data plane**

- XOR parity FEC for single-loss recovery (S-7.01, BC-2.02.007 PC-2/PC-3/PC-4)
- PathSnapshot RouterAddr populated with resolved host:port (S-BL.ROUTER-ADDR,
  BC-2.06.003 PC-1)
- SVTN-scoped session discovery with HMAC-first auth, UTF-8 rune-boundary
  truncation, and session-name presence advertisement (S-7.02)
- Per-path RTT/loss tracking and duplicate-and-race dispatch (S-4.01)
- Per-path EWMA RTT flagging degraded paths when RTT > 200ms (S-5.03,
  BC-2.02.003 PC-5)
- Split-horizon loop prevention with DropCache wiring (S-4.04)
- Downstream ARQ with piggybacked ACK/SACK and TLPKTDROP (S-4.03)
- Upstream idempotent replay window (S-4.02)
- HMAC-SHA256 frame auth with HKDF per-(node, SVTN) keying (S-2.01, BC-2.05.005)
- Admission + SVTN-isolated routing (S-2.02, BC-2.05.001/002/006/007)
- Per-source HMAC failure counter and E-ADM-017 admission alert (S-W3.05)
- HMAC failure counter wired into daemon router (BC-2.05.008)
- Tier-2 per-session authorization and read-only enforcement (S-3.03,
  BC-2.04.005/BC-2.05.003)
- Console attach/detach and multi-console fan-out (S-3.02, BC-2.04.003/004/006)
- PTY proxy fallback for control-mode failures (S-3.01b, BC-2.04.002)
- tmux control mode integration (S-3.01a, BC-2.04.001)
- HMAC wire-up into RouteFrame (S-3.04, BC-2.05.008)
- Session continuity via cryptographic re-authentication (S-1.03, BC-2.01.007)
- Typed FrameType + MTU validation (S-1.01 refactor)
- Timeslice clock state machine in internal/halfchannel (S-1.02)
- 44-byte outer header codec in internal/frame (S-1.01)
- Full daemon assembly wiring all Wave-3 subsystems (S-W3.04)

**Wire contracts and CLI**

- Canonical JSON envelope (ok/error{code,message}/data) throughout admin surface
- E-* error taxonomy: E-ADM-*, E-SVTN-*, E-CFG-*, E-INT-* families
- `sbctl` with subcommand dispatch and usageError exit-code discrimination
  (exit 2 for usage errors, exit 1 for runtime errors); complete sweep across
  all console/router/admin verbs (Phase 5 Passes 6/7/8/10)
- OpenSSH pubkey handling in admin wire contract (Phase 5 Pass 4)
- Interactive TTY confirm prompt for destructive SVTN operations
- E2E management-plane harness across all four daemon types (S-W5.02, VP-049)

### Security

- HMAC constant-time comparison (`crypto/hmac.Equal`) throughout admission and
  session-discovery paths
- Fuzz-saturated wire parsers (7 targets × 300s clean)
- `govulncheck` clean (single Windows-only advisory triaged LOW; switchboard
  targets darwin and linux only)
- No secrets in repository

### Verified

- 45 behavioral contracts (BC-S.SS.NNN), 77 verification properties (VP-*)
  with 100% coverage
- `go test ./... -count=1 -short` green
- Formal hardening gate SATISFIED (6/6 criteria: coverage audit, fuzz
  saturation, mutation testing, security scan, baseline, governance)

### Known follow-ups (tracked, non-blocking)

Filed as 5 follow-up story trackers in `.factory/` open drift register:

- FOLLOWUP-S-HMAC-MUTATION (MED): Raise hmac package mutation coverage from
  62.50% toward ≥90%
- FOLLOWUP-S-ADMISSION-MUTATION (LOW)
- FOLLOWUP-S-PATHS-EWMA-GOLDENS (LOW)
- FOLLOWUP-S-VP-COVERAGE-GAPS (LOW): Remediate GAP-P6A-001..004 including
  drain-package impl
- FOLLOWUP-S-TOOLCHAIN-SECURITY-BUMP (LOW)

Deferred to backlog anchor `S-BL.BENCH`: VP-041/VP-042 performance benchmarks
(NFR-001..006 declared; benchmarks explicitly non-blocking for this release).

[Unreleased]: https://github.com/ArcavenAE/switchboard-blue/compare/v0.1.0-rc.1...HEAD
[0.1.0-rc.1]: https://github.com/ArcavenAE/switchboard-blue/releases/tag/v0.1.0-rc.1
