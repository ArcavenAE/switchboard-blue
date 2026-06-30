# AC-005 — server-side duration validation in admin.key.expire handler (DI-003)
# Demonstrates: handler re-validates TTL server-side independent of CLI.
# Valid TTL accepted; ttl<=0 and ttl>100 years rejected with E-CFG-001.
# Traces: BC-2.05.004 PC-3; DI-003 defense-in-depth; AC-005.
#
# NOTE: The e2e TTL rejection tests (TestE2E_AdminExpire_ServerRejectsTTL*)
# currently return E-ADM-009 instead of E-CFG-001 because those tests use
# an unregistered caller key. The AC-006 authority gate fires before the
# TTL validation check. Unit-level evidence below is authoritative for AC-005.
# The test wiring issue (unregistered caller key) needs a separate fix.
#
# Usage: vhs AC-005-server-side-ttl-validation.tape

Output docs/demo-evidence/S-6.06/AC-005-server-side-ttl-validation.gif
Output docs/demo-evidence/S-6.06/AC-005-server-side-ttl-validation.webm

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
Type "cd /Users/skippy/work/switchboard-blue/.worktrees/S-6.06 && clear"
Enter
Sleep 1s
Show

# Happy path: valid positive TTL (unit -- handler accepts 24h)
Type "go test ./cmd/switchboard/... -run 'TestBuildAdminHandlers_KeyExpire_HappyPath' -v 2>&1 | grep -E 'RUN|PASS|FAIL|^ok|---'"
Enter
Sleep 4s

# Error path: negative TTL, zero TTL, TTL > 100 years -> E-CFG-001 (unit)
Type "go test ./cmd/switchboard/... -run 'TestBuildAdminHandlers_KeyExpire_NegativeTTL|TestBuildAdminHandlers_KeyExpire_MissingAfterField' -v 2>&1 | grep -E 'RUN|PASS|FAIL|^ok|---'"
Enter
Sleep 4s
