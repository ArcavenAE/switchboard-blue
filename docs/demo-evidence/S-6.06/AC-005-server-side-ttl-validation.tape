# AC-005 — server-side duration validation in admin.key.expire handler (DI-003)
# Demonstrates: handler re-validates TTL server-side independent of CLI.
# Valid TTL accepted; ttl<=0 and ttl>100 years rejected with E-CFG-001.
# Traces: BC-2.05.004 PC-3; DI-003 defense-in-depth; AC-005.
#
# Fix landed at commit 660b91a: the three TestE2E_AdminExpire_ServerRejectsTTL*
# tests now authenticate as the SVTNManager bootstrap key (ctrlPriv passed to
# startE2EServerWithOps), so the AC-005 TTL bounds check fires instead of the
# AC-006 authority gate (E-ADM-009). DEMO-ISSUE-001 closed.
#
# Usage: cd .worktrees/S-6.06 && vhs docs/demo-evidence/S-6.06/AC-005-server-side-ttl-validation.tape

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

# --- Unit level: happy path and E-CFG-001 rejections ---
Type "go test ./cmd/switchboard/... -run 'TestBuildAdminHandlers_KeyExpire_HappyPath|TestBuildAdminHandlers_KeyExpire_NegativeTTL|TestBuildAdminHandlers_KeyExpire_MissingAfterField' -v 2>&1 | grep -E 'RUN|PASS|FAIL|^ok|---'"
Enter
Sleep 6s

# --- E2E level (integration build tag): all three TTL-bound tests return E-CFG-001 ---
Type "go test -tags integration ./cmd/switchboard/ -run 'TestE2E_AdminExpire_ServerRejectsTTL' -v 2>&1 | grep -E 'RUN|PASS|FAIL|^ok|---'"
Enter
Sleep 8s

Sleep 2s
