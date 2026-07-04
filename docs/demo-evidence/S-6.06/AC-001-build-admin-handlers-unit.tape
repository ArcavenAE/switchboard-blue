# AC-001 — BuildAdminHandlers unit tests
# Demonstrates: BuildAdminHandlers returns 4 handlers; register/revoke/expire/list-keys
# happy paths return {key_fingerprint, timestamp}; error mapping unit coverage.
# Traces: BC-2.05.004 PC-1/PC-2/PC-3/PC-4; AC-001.
#
# Usage: vhs AC-001-build-admin-handlers-unit.tape

Output docs/demo-evidence/S-6.06/AC-001-build-admin-handlers-unit.gif
Output docs/demo-evidence/S-6.06/AC-001-build-admin-handlers-unit.webm

Require go

Set Shell "zsh"
Set FontFamily "Menlo"
Set FontSize 14
Set Width 1200
Set Height 600
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

# Happy-path: AC-001 unit tests — 4 handlers, register/revoke/expire/list-keys
# success payloads include key_fingerprint + timestamp (BC-2.05.004 PC-4)
Type "go test ./cmd/switchboard/... -run 'TestBuildAdminHandlers_KeyRegister_HappyPath|TestBuildAdminHandlers_KeyRevoke_HappyPath|TestBuildAdminHandlers_KeyExpire_HappyPath|TestBuildAdminHandlers_ListKeys_HappyPath|TestBuildAdminHandlers_FourHandlers' -v 2>&1 | grep -E 'RUN|PASS|FAIL|^ok|---'"
Enter
Sleep 4s

# Error path: error mapping tests (ErrSVTNNotFound -> E-SVTN-003; role mismatch -> E-ADM-019)
Type "go test ./cmd/switchboard/... -run 'TestBuildAdminHandlers_KeyRegister_ErrorMapping|TestBuildAdminHandlers_KeyRevoke_ErrorMapping|TestBuildAdminHandlers_NilManager' -v 2>&1 | grep -E 'RUN|PASS|FAIL|^ok|---'"
Enter
Sleep 4s
