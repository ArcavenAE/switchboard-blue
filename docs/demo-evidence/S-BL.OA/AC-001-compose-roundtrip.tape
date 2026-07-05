# S-BL.OA AC-001 — outer-assembler composes ChannelFrame + OuterHeader into
# wire bytes that flow through netingress.ReadFrame and are authenticated by
# routing.RouteFrame.
#
# Demonstrates:
#   - The channel-header codec (F-004): 12 bytes fixed, +8 when SACK_present;
#     BC-2.01.005 PC-3 wire order and reserved-bytes-zero rule enforced on
#     decode.
#   - Composed wire-format round-trip (F-003): assembler → netingress →
#     routing verifies end-to-end. The message shape (zeroed-tag outer_header
#     || channel_header || payload) matches routing.verifyFrameHMAC
#     byte-for-byte.
#   - Adversarial duals: any bit-flip inside the MAC region fails HMAC
#     verification; truncation fails parse at the ingress boundary.
#   - Defensive MaxPayloadSize re-check: ErrPayloadTooLarge when
#     channel_header_size + payload would truncate the uint16 PayloadLen.
#
# Traces: S-BL.OA AC-001 (composed wire-format); wave-adv F-003 (composed
# test); wave-adv F-004 (channel-header serializer); BC-2.01.004 (outer
# header); BC-2.01.005 (channel header); BC-2.05.008 PC-2 (wire HMAC).
#
# Usage: vhs AC-001-compose-roundtrip.tape
# Per POL-004 rendered artifacts are gitignored; only this .tape script is
# checked in.

Output docs/demo-evidence/S-BL.OA/AC-001-compose-roundtrip.gif
Output docs/demo-evidence/S-BL.OA/AC-001-compose-roundtrip.webm

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

# The outerassembler package — pure-core composition of ChannelFrame and
# OuterHeader into wire bytes suitable for the receiver's ingress path.
Type "sed -n '1,52p' internal/outerassembler/channelheader.go"
Enter
Sleep 5s

# The channel-header codec (wave-adv F-004): 12/20 byte layout, big-endian
# wire order, conditional SACK bitmap when flag bit 2 is set.
Type "go test -race -run 'TestEncodeChannelHeader_|TestChannelHeader_|TestDecodeChannelHeader_' -v ./internal/outerassembler/ 2>&1 | tail -30"
Enter
Sleep 6s

# Composed wire-format round-trip (wave-adv F-003): assembler → netingress →
# routing.RouteFrame succeeds for DATA, EMPTY_TICK, and SACK-carrying frames.
# The HMAC message shape (zeroed-tag outer_header || channel_header ||
# payload) is byte-for-byte the same routing.verifyFrameHMAC computes.
Type "go test -race -run TestIntegration_Composed_RoutingRouteFrameVerifies -v ./internal/outerassembler/"
Enter
Sleep 6s

# Adversarial duals: any bit flip inside the MAC region fails HMAC verify.
Type "go test -race -run TestIntegration_FlippedBit_FailsHMAC -v ./internal/outerassembler/ 2>&1 | tail -20"
Enter
Sleep 5s

# Adversarial duals: truncated wire bytes fail parse at the ingress boundary
# with io.ErrUnexpectedEOF (CWE-400 fail-closed on truncation).
Type "go test -race -run TestIntegration_Truncation_FailsParse -v ./internal/outerassembler/ 2>&1 | tail -20"
Enter
Sleep 5s

# Defensive re-check: Assemble rejects a payload that would truncate
# PayloadLen, even if a caller bypassed halfchannel.Enqueue.
Type "go test -race -run TestAssemble_RejectsPayloadThatWouldTruncatePayloadLen -v ./internal/outerassembler/ 2>&1 | tail -20"
Enter
Sleep 4s

# Full package test — 3 rounds under -race for concurrency clean.
Type "go test -race -count=3 ./internal/outerassembler/ 2>&1 | tail -5"
Enter
Sleep 5s

Sleep 2s
