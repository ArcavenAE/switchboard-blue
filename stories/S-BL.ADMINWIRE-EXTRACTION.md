---
artifact_id: S-BL.ADMINWIRE-EXTRACTION
document_type: story
level: ops
story_id: S-BL.ADMINWIRE-EXTRACTION
version: "1.0"
title: "Extract admin wire arg structs to internal/adminwire shared package"
status: backlog
producer: pr-manager
timestamp: 2026-07-03T00:00:00
modified: 2026-07-03T00:00:00
phase: 2
epic: E-6
wave: backlog
priority: P3
scope_phase: M
estimated_points: 2
bc_traces: []
vp_traces: []
subsystems: [network-management]
architecture_modules: [cmd/switchboard, cmd/sbctl, internal/adminwire]
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on: []
blocks: []
acceptance_criteria_count: 3
drift_origin: DRIFT-P5P4-ADMINWIRE-EXTRACTION
---

# S-BL.ADMINWIRE-EXTRACTION — Extract admin wire arg structs to internal/adminwire

## Background

Wire arg struct types (`KeyRegisterArgs`, `KeyRevokeArgs`, `SVTNDestroyArgs`) are
currently defined inline in `cmd/switchboard/admin_handlers.go`. Both the sbctl-side
tests (`cmd/sbctl/admin_test.go:2093`) and the switchboard-side tests
(`cmd/switchboard/admin_handlers_wire_shared_pkg_test.go`) cross-assert wire-field
JSON tag shapes to prevent silent wire breakage.

A comment in `admin_handlers_wire_shared_pkg_test.go:7,33` notes the extraction path:
> "a future refactor may extract them to internal/adminwire"

This story formalizes that refactor. No behavior change — pure package restructure.

## Acceptance Criteria

### AC-001: New internal/adminwire package
A new `internal/adminwire` package is created containing `KeyRegisterArgs`,
`KeyRevokeArgs`, and `SVTNDestroyArgs` with unchanged JSON tags (`json:"svtn_id"`,
`json:"pubkey_openssh"`, `json:"name"`). The package carries no non-struct logic.

### AC-002: Import in cmd/switchboard and cmd/sbctl
`cmd/switchboard/admin_handlers.go` and `cmd/sbctl/admin.go` import from
`internal/adminwire` rather than defining the structs inline. Wire contracts
(JSON tags) are unchanged — no wire-breaking change.

### AC-003: Tests migrate and stay green
`admin_handlers_wire_shared_pkg_test.go` migrates its assertions to reference the
`internal/adminwire` types directly. All existing wire-contract tests pass with
`go test ./... -race -count=1`. `golangci-lint` is clean.

## Out of Scope

- Changing any wire field names or JSON tags (would be a wire-breaking change)
- Adding new RPC types beyond the three named above
- Any behavior changes to admin handlers

## Notes

Deferred from Burst 19 (Phase 5 Pass 4 wire-contract remediation). The current
dual-test approach (both sides assert the same JSON tags) provides adequate contract
protection until extraction is scheduled. This story is a maintenance convenience,
not a correctness gap.
