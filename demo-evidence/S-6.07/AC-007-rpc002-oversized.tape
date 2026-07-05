# S-6.07 AC-007: dispatch() wraps io.ErrUnexpectedEOF with E-RPC-002
# ADR-012 §6 transport-layer parity
# Test: TestSbctlAdmin_OversizedRPCResponse_ReturnsE_RPC_002

Output AC-007-rpc002-oversized.gif
Output AC-007-rpc002-oversized.webm

Require go

Set FontFamily "Menlo"
Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set Padding 20
Set PlaybackSpeed 1

Hide
Type "cd /Users/skippy/work/aae-orc/run/switchboard-blue"
Enter
Sleep 500ms
Show

Type "# S-6.07 AC-007: oversized RPC response → E-RPC-002: message too large"
Enter
Sleep 300ms

Type "# error path: io.ErrUnexpectedEOF wrapped before generic error return"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestSbctlAdmin_OversizedRPCResponse_ReturnsE_RPC_002 ./cmd/sbctl/..."
Enter
Sleep 6s
Sleep 2s
