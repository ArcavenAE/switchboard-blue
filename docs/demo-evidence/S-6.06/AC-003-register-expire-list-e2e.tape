# AC-003 — admin.key.register, admin.key.expire, admin.key.list-keys e2e tests
# Demonstrates: register key -> appears in list-keys; expire sets TTL; list-keys returns both.
# Uses real mgmt.Server + real Unix socket.
# Traces: BC-2.05.004 PC-1/PC-3; AC-003.
#
# Usage: vhs AC-003-register-expire-list-e2e.tape

Output docs/demo-evidence/S-6.06/AC-003-register-expire-list-e2e.gif
Output docs/demo-evidence/S-6.06/AC-003-register-expire-list-e2e.webm

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

# Happy paths: register+list, expire+inspect, list-keys with multiple keys
Type "go test -tags integration ./cmd/switchboard/... -run 'TestE2E_AdminRegister_HappyPath|TestE2E_AdminExpire_HappyPath|TestE2E_AdminListKeys_HappyPath' -v 2>&1 | grep -E 'RUN|PASS|FAIL|^ok|---'"
Enter
Sleep 6s
