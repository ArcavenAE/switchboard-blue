---
artifact_id: S-BL.NODE-ADMISSION-PROVISIONING-delivery-record
document_type: delivery-record
story_id: S-BL.NODE-ADMISSION-PROVISIONING
pr_number: 125
merge_sha: ce06f6a02
merged_at: 2026-07-16
base_sha: d249f88
branch: "develop <- feature/S-BL.NODE-ADMISSION-PROVISIONING"
repo: ArcavenAE/switchboard-blue
reconciled_at: 2026-07-18
reconciled_by: state-manager (retroactive)
---

# Delivery Record: S-BL.NODE-ADMISSION-PROVISIONING

## Summary

Story **S-BL.NODE-ADMISSION-PROVISIONING** was delivered via PR #125 (merged 2026-07-16) by a
parallel session. That session left factory bookkeeping unreconciled. This record documents
the delivery retroactively as of 2026-07-18.

## PR Details

| Field | Value |
|-------|-------|
| PR | #125 |
| Title | feat(access): node admission-identity provisioning — Ed25519 keypair + Discovery.Run lifecycle wiring |
| Merge SHA | ce06f6a02 |
| Base SHA | d249f88 |
| Merged at | 2026-07-16 |
| Branch | develop <- feature/S-BL.NODE-ADMISSION-PROVISIONING |
| Repo | ArcavenAE/switchboard-blue |

## Delivery Scope

Full per-story TDD delivery covering:

- Red Gate stubs established before implementation
- 26 AC test functions covering AC-001..AC-008
- `internal/config/config.go`: `AdmissionKeyFile` field + E-CFG-014 validation (no I/O in Validate)
- `cmd/switchboard/access.go`: `loadOrGenerateAdmissionKeypair` — PKCS#8 PEM, atomic write, mode 0600, fail-closed, startup INFO log
- `cmd/switchboard/access.go`: `Discovery.Run` wired via WaitGroup in `runAccessWithConnector` (Option Y)
- `cmd/switchboard/access_admission_test.go`: +1528 lines, 26 test functions
- `internal/admission/admission.go`: +22 lines
- `internal/svtnmgmt/svtnmgmt.go`: +11 lines
- Adversarial fix bursts: F-3, B-3, M3, F-4, adversary-F-6, umask-race hermeticity
- Demo evidence: 8 AC `.tape` files + `evidence-report.md` under `docs/demo-evidence/S-BL.NODE-ADMISSION-PROVISIONING/` (POL-004 compliant)

## Acceptance Criteria Coverage

| AC | Description | Status |
|----|-------------|--------|
| AC-001 | Config.Validate() — absent/valid accepted; whitespace-only → E-CFG-014; no I/O | DELIVERED |
| AC-002 | First-run keypair generation: atomic write mode 0600, PKCS#8 PEM, parent dir created | DELIVERED |
| AC-003 | Subsequent start: file present and valid → key loaded; public key stable | DELIVERED |
| AC-004 | Fail-closed on corrupt or non-Ed25519 file (E-KEY-001) | DELIVERED |
| AC-005 | File permissions > 0600 → WARNING logged; daemon starts (advisory-not-fatal) | DELIVERED |
| AC-006 | Startup INFO log with base64url pubkey on every start | DELIVERED |
| AC-007 | discovery.Config.LocalNodeAdmissionPubkey wired from loaded/generated keypair | DELIVERED |
| AC-008 | Discovery.Run goroutine WG-tracked; ctx.Canceled is clean shutdown; no goroutine leak | DELIVERED |

## Consequence

With this delivery and S-BL.ADMISSION-SYNC-WIRE (PR #126 @ 92a2c65, 2026-07-18),
**S-BL.NODE-IDENTIFY-WIRE** now has both `depends_on` prerequisites cleared:

- Leg 1 (NODE-SIDE): S-BL.NODE-ADMISSION-PROVISIONING — PR #125 @ ce06f6a (2026-07-16)
- Leg 2 (ROUTER-SIDE): S-BL.ADMISSION-SYNC-WIRE — PR #126 @ 92a2c65 (2026-07-18)

S-BL.NODE-IDENTIFY-WIRE is UNBLOCKED but NOT decomposition-ready: draft v1.4, 0 ACs,
obligations 3/4 open, architect elaboration required before decomposition.

## CI Flake Note

develop tip 92a2c65 (post PR #126 merge) post-push CI run #29659181289 failed on
`internal/discovery.TestDiscovery_Advertise_PeriodicHeartbeat` (0.03s). Dispositioned
as a scheduler-jitter FLAKE — `internal/discovery` was untouched by #126; prior tip
ce06f6a was green; the PR-merge run passed (2m18s); local 20/20 pass plain and 10x
under `-race`. Same class as switchboard-blue#124 (TestLookup_ConcurrentRegisterRace).
NOT a regression from this story or PR #126. Tracked as CI-FLAKE-DISCOVERY-HEARTBEAT
in STATE.md Open Drift Items.

## Historical Note: Input-Hash Inconsistency

The original story draft carried an internal input-hash inconsistency:
- Frontmatter `input-hash`: `05213d5`
- Body POL-005 note citation: `504693c`
- STORY-INDEX v4.115 changelog: `05213d5`

This inconsistency existed in the original draft before delivery. The story was
implemented and delivered from the canonical inputs as specified in rulings v1.0.
The `input-hash` has been updated to `f617617` (current hash of the declared inputs
at reconciliation time) as a post-delivery bookkeeping correction.
