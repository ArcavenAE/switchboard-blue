# AC-002: sbctl admin svtn create --name wire envelope + happy-path output
#
# Demonstrates: CLI subcommand sends {"command":"admin.svtn.create","args":{"name":"..."}}
# and prints svtn_id (hex) + bootstrap_fingerprint (SHA256:<base64>) on success.
#
# Usage: vhs AC-002-cli-wire-envelope.tape
Output docs/demo-evidence/S-6.07/AC-002-cli-wire-envelope.gif
Output docs/demo-evidence/S-6.07/AC-002-cli-wire-envelope.webm

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
Type "cd /Users/skippy/work/switchboard-blue/.worktrees/S-6.07 && clear"
Enter
Sleep 1s
Show

# Success path: CLI happy-path — wire envelope shape + output format
Type "go test ./cmd/sbctl/... -run 'TestSbctlAdmin_SvtnCreate_CLI|TestSbctlAdmin_SvtnCreate_SuccessOutputsSVTNIDAndFingerprint|TestSbctlAdmin_SvtnCreate_JSONRoundTrip' -v 2>&1 | grep -E 'RUN|PASS|FAIL|svtn_id|bootstrap|^ok|---'"
Enter
Sleep 6s

# Error path: missing --name flag rejected before connecting to daemon
Type "go test ./cmd/sbctl/... -run 'TestSbctlAdmin_SvtnCreate_MissingName' -v 2>&1 | grep -E 'RUN|PASS|FAIL|^ok|---'"
Enter
Sleep 5s
