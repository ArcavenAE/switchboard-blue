# AC-006 — `admin.svtn.status` error paths: not-found and admission-denied
#
# Demonstrates:
#   1. `sbctl svtn status --name=doesnotexist` -> E-SVTN-003 "SVTN not found: doesnotexist"; exit 1
#   2. A caller admitted only to a different SVTN -> E-ADM-009 "insufficient authority
#      for operation admin.svtn.status: key <fp> has role <role>"; exit 1. SVTN
#      roster/existence is NOT disclosed — the admission gate fires before status
#      is computed (CWE-862 defense, mirrors BC-2.05.004 EC-008; VP-048 row 2)
#
# BC anchors: BC-2.07.001 PC-4 (not-found and admission-denied Canonical Test Vectors)
#
# Test coverage:
#   TestAdminSVTNStatus_NotFound_ESVTN003                              (cmd/switchboard/admin_handlers_test.go)
#   TestAdminSVTNStatus_AdmissionDenied_EADM009_NoExistenceOracleLeak  (cmd/switchboard/admin_handlers_test.go)
#
# Usage: vhs docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-006-svtn-status-error-paths.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-006-svtn-status-error-paths.gif
Output docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-006-svtn-status-error-paths.webm

Require go

Set Shell "zsh"
Set FontFamily "Menlo"
Set FontSize 14
Set Width 1200
Set Height 720
Set Theme "Catppuccin Mocha"
Set WindowBar Colorful
Set BorderRadius 8
Set Padding 20
Set Framerate 30
Set TypingSpeed 30ms

Hide
Type "clear"
Enter
Sleep 1s
Show

# ── AC-006: admin.svtn.status error paths — not-found, admission-denied ───────

Type `echo "=== AC-006: admin.svtn.status error paths — E-SVTN-003, E-ADM-009 (no existence-oracle leak) ==="`
Enter
Sleep 300ms

Type `go test -race -run 'TestAdminSVTNStatus_NotFound_ESVTN003|TestAdminSVTNStatus_AdmissionDenied_EADM009_NoExistenceOracleLeak' -count=1 -v ./cmd/switchboard/...`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-006 PASS ==="`
Enter
Sleep 2s
