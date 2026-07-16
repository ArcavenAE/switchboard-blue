# AC-008 — Discovery.Run goroutine WG-tracked; ctx.Canceled is clean shutdown; no goroutine leak
#
# Demonstrates:
#   1. wg.Add(1) before go disc.Run(runCtx) — runAccessWithConnector starts disc.Run in a
#      WG-tracked goroutine. Verified by HeartbeatObserver channel handshake: a tick fires
#      the observer, confirming disc.Run is live inside runAccessWithConnector.
#   2. disc.Run shares the SAME sync.WaitGroup as sweep and frames-dropped tickers —
#      wg.Wait() in runAccessWithConnector joins all goroutines together.
#   3. ctx.Canceled from disc.Run is NOT internalFailure — runAccessWithConnector returns
#      nil on clean context cancellation (Decision 7 / BC-2.04.008 Invariant 2).
#   4. No goroutine leak after ctx cancellation — wg.Wait() blocks until disc.Run exits
#      (blocking HeartbeatObserver discriminator); bounded wait ≤100ms confirmed.
#   5. Discovery.Run starts AFTER the admission keypair is loaded AND AFTER the mgmt server
#      goroutine is started (startup-ordering: phases (d)–(f) per rulings §3.1).
#
# BC anchors: BC-2.04.008 PC-1 through PC-5; ARCH-01 v1.7 §Goroutine WaitGroup Contract;
#             BC-2.04.008 Invariant 2; rulings Decision 7
#
# Test coverage:
#   TestDiscoveryRun_WGAddBeforeGoStatement
#   TestDiscoveryRun_SameWaitGroupAsSweepTickers
#   TestDiscoveryRun_CtxCanceled_NotInternalFailure
#   TestDiscoveryRun_NoGoroutineLeak_AfterCtxCancel
#   TestDiscoveryRun_StartupOrdering_AfterKeypairAndMgmtServer
#
# Usage: vhs docs/demo-evidence/S-BL.NODE-ADMISSION-PROVISIONING/AC-008-discovery-run-wg-tracked-clean-shutdown.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is checked in.

Output docs/demo-evidence/S-BL.NODE-ADMISSION-PROVISIONING/AC-008-discovery-run-wg-tracked-clean-shutdown.gif
Output docs/demo-evidence/S-BL.NODE-ADMISSION-PROVISIONING/AC-008-discovery-run-wg-tracked-clean-shutdown.webm

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

# ── AC-008: Discovery.Run WG-tracked, ctx.Canceled clean, no goroutine leak ─────

Type `echo "=== AC-008: disc.Run WG-tracked in runAccessWithConnector; ctx.Canceled → clean shutdown (not internalFailure); no goroutine leak ==="`
Enter
Sleep 300ms

Type `go test -race -run "TestDiscoveryRun_WGAddBeforeGoStatement|TestDiscoveryRun_SameWaitGroupAsSweepTickers|TestDiscoveryRun_CtxCanceled_NotInternalFailure|TestDiscoveryRun_NoGoroutineLeak_AfterCtxCancel|TestDiscoveryRun_StartupOrdering_AfterKeypairAndMgmtServer" -count=1 -v ./cmd/switchboard/`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-008 PASS: WG-tracked goroutine, clean ctx.Canceled shutdown, no goroutine leak, startup ordering — all verified ==="`
Enter
Sleep 2s
