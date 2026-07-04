# AC-007: dispatch() E-RPC-002 wrap for oversized RPC responses (Ruling-14)
#
# Demonstrates: io.ErrUnexpectedEOF from the mgmt response decode path is
# wrapped as "E-RPC-002: message too large" and the error chain preserves
# errors.Is(err, io.ErrUnexpectedEOF).
#
# Usage: vhs AC-007-rpc002-oversized-response.tape
Output docs/demo-evidence/S-6.07/AC-007-rpc002-oversized-response.gif
Output docs/demo-evidence/S-6.07/AC-007-rpc002-oversized-response.webm

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

# Error path: oversized response stream → E-RPC-002: message too large
Type "go test ./cmd/sbctl/... -run 'TestSbctlAdmin_OversizedRPCResponse_ReturnsE_RPC_002' -v 2>&1 | grep -E 'RUN|PASS|FAIL|E-RPC-002|message too large|^ok|---'"
Enter
Sleep 6s
