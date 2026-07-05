# S-BL.ARQ-TX — ARQ retransmit-SEND path wired to authenticated wire bytes.
#
# The pre-story surface: internal/arq is a pure-core state machine with
# OnAck/GapsToRetransmit/EnqueueSend/TLPKTDROP primitives. The queue holds
# payloads under their send seq, gaps are detected against a SACK bitmap,
# but nothing turned a "retransmit this payload" decision into an
# HMAC-authenticated wire frame the receiver could verify.
#
# What this story delivers: internal/arqsend — a small boundary-layer
# package that composes arq.PayloadForInFlight + outerassembler.Assemble +
# a caller-supplied Dispatch callback into a single Retransmit seam. The
# retransmit carries the ORIGINAL payload but under a NEW frame seq number
# (BC-2.02.005 PC-5, QUIC retransmit model). On dispatch success, the new
# seq is enqueued and the old seq is released; on dispatch failure, ARQ
# state is left unchanged so the retransmit is re-tryable on the next
# GapsToRetransmit pass (no-orphan-state invariant).
#
# Traces: S-BL.ARQ-TX AC-001..005; BC-2.02.005 PC-3 (retransmit on gap)
# and PC-5 (new seq); ARCH-02 wire format; ARCH-03 routing; ARCH-09
# pure-core boundary; drift lineage S403-H1-DEFER (Wave 4 audit).
#
# Usage: vhs AC-001-retransmit-wiring.tape
# Per POL-004 rendered artifacts are gitignored; only this .tape script is
# checked in.

Output docs/demo-evidence/S-BL.ARQ-TX/AC-001-retransmit-wiring.gif
Output docs/demo-evidence/S-BL.ARQ-TX/AC-001-retransmit-wiring.webm

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

# The pure-core ARQ accessors that make the boundary layer possible.
# PayloadForInFlight returns a defensive copy (go.md rule 12);
# RemoveInFlight is idempotent — both are single-writer.
Type "sed -n '485,525p' internal/arq/arq.go"
Enter
Sleep 6s

Type "go test -race -run 'TestARQ_PayloadForInFlight_|TestARQ_RemoveInFlight_' -v ./internal/arq/ 2>&1 | tail -20"
Enter
Sleep 5s

# The boundary-layer seam: internal/arqsend.Retransmit. Composes
# arq.PayloadForInFlight + outerassembler.Assemble + a caller Dispatch
# into a single call, ordering side effects so ARQ state stays
# consistent whether Dispatch succeeds or fails.
Type "sed -n '112,164p' internal/arqsend/arqsend.go"
Enter
Sleep 6s

# Unit-level contract: happy-path emits wire for newSeq, unknown oldSeq
# returns ErrSequenceNotInFlight, and — the load-bearing invariant — a
# Dispatch error leaves ARQ state UNCHANGED (no orphan queue entry).
Type "go test -race -run 'TestRetransmit_' -v ./internal/arqsend/ 2>&1 | tail -30"
Enter
Sleep 6s

# End-to-end integration: SACK bitmap marks one of three in-flight
# frames received; GapsToRetransmit returns the two gaps; arqsend
# retransmits each under a fresh new seq; each wire frame is parsed
# by netingress.ReadFrame and verified by routing.RouteFrame
# (HMAC + admitted + SVTNRoute all succeed).
Type "go test -race -run TestIntegration_GapWalkToRoutedRetransmit -v ./internal/arqsend/ 2>&1 | tail -20"
Enter
Sleep 6s

# BC-2.02.005 PC-5 across a batch: four gaps, monotonic new-seq
# assignment; every wire ChanSeq is the assigned new seq, never the
# old seq.
Type "go test -race -run TestIntegration_MultipleGapsGetMonotonicNewSeqs -v ./internal/arqsend/ 2>&1 | tail -15"
Enter
Sleep 5s

# Full package test — 3 rounds under -race for concurrency clean.
Type "go test -race -count=3 ./internal/arqsend/ 2>&1 | tail -5"
Enter
Sleep 5s

Sleep 2s
