# AC-001: BuildAdminHandlers registers admin.svtn.create (control-mode only)
#
# Demonstrates: TestBuildAdminHandlers_SVTNCreate_Registered passes —
# the handler map contains the "admin.svtn.create" key;
# TestBuildAdminHandlers_FiveHandlers confirms exactly 5 handlers.
#
# Usage: vhs AC-001-handler-registration.tape
Output docs/demo-evidence/S-6.07/AC-001-handler-registration.gif
Output docs/demo-evidence/S-6.07/AC-001-handler-registration.webm

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

# Success path: handler registration unit tests
Type "go test ./cmd/switchboard/... -run 'TestBuildAdminHandlers_SVTNCreate_Registered|TestBuildAdminHandlers_FiveHandlers' -v 2>&1 | grep -E 'RUN|PASS|FAIL|^ok|---'"
Enter
Sleep 6s
