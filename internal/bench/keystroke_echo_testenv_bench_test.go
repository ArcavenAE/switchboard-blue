//go:build integration

// This file is the testenv-integrated counterpart to the tag-free lower-bound
// benchmark in keystroke_echo_bench_test.go. It is guarded by the `integration`
// build tag (the same convention internal/testenv integration tests use) so the
// tag-free lower-bound bench remains buildable and runnable on its own.
//
// VP-042 status — READ BEFORE USING THIS AS EVIDENCE:
//
//	This benchmark drives the CANONICAL testenv.NewLoopback rig, but that rig,
//	as delivered by S-BL.TESTENV (PR #110), does NOT exercise the full stack the
//	VP-042 property is stated over. Specifically:
//
//	  - testenv.NewLoopback DISCARDS its LoopbackConfig argument. The
//	    TickIntervalUpstream (10ms) and TickIntervalDownstream (50ms) fields are
//	    dead — referenced nowhere but their own struct definition. NewLoopback
//	    calls newEnv(ctx, b, 1), the same single-router env as testenv.New.
//	  - testenv.SendKeystroke calls AccessNode.DeliverFrame directly, a
//	    synchronous in-memory fan-out. AccessNode is goroutine-free; there is no
//	    tick scheduler, no ARQ retransmit path, and no multipath duplicate-and-race.
//	  - internal/halfchannel is pure-core ("no goroutines, no timers, no I/O");
//	    its 10ms/50ms Tick() cadence must be driven by an effectful layer.
//	    testenv does not import halfchannel/arq/multipath and does not drive Tick().
//
//	VP-042's 100ms budget was sized for "10ms upstream tick cadence + 50ms
//	downstream tick cadence + ARQ overhead" (BC-2.01.001, BC-2.02.001). This
//	benchmark measures none of that — it measures the cost of a synchronous
//	DeliverFrame plus a buffer snapshot. It is therefore a testenv-integrated
//	LOWER BOUND, statistically equivalent to BenchmarkKeystrokeEcho_P99, and it
//	does NOT constitute full-stack VP-042 evidence. It MUST NOT be used to flip
//	VP-042 verification_lock. Locking VP-042 on a testenv-integrated measurement
//	requires testenv's loopback path first routing keystrokes through
//	halfchannel.Tick() at the configured cadence + internal/arq + internal/multipath
//	(making LoopbackConfig live) — a testenv/story-scope change with ARCH-08
//	import-set implications, not a benchmark change.
package bench_test

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/testenv"
)

// BenchmarkKeystrokeToEcho_P99 is the testenv-integrated keystroke-to-echo p99
// benchmark from the VP-042 proof-harness skeleton, adapted to the real
// testenv.LoopbackEnv API surface (see the API-divergence notes below).
//
// API divergence from the VP-042.md skeleton:
//   - skeleton: testenv.NewLoopback(b, ctx, cfg) → real: NewLoopback(ctx, b, cfg)
//     (context first) and it returns *LoopbackEnv, whose *Env field carries the
//     session/keystroke/echo helpers (skeleton called them on the return value).
//   - skeleton attaches no console; the real DeliverFrame fan-out only reaches
//     WaitForEcho/CollectFrames through an attached console, so one is attached.
//
// This runs under `-tags integration`. See the package-level comment for why
// this measurement is a lower bound and NOT a VP-042 lock.
//
// Run with:
//
//	go test -tags integration -run '^$' -bench=BenchmarkKeystrokeToEcho_P99 \
//	    -benchtime=1x -count=1 ./internal/bench/
func BenchmarkKeystrokeToEcho_P99(b *testing.B) {
	const (
		upstreamInterval   = 10 * time.Millisecond
		downstreamInterval = 50 * time.Millisecond
		samples            = 500
		maxP99             = 100 * time.Millisecond // NFR-001 / VP-042 floor guard
		echoTimeout        = 500 * time.Millisecond
	)

	ctx := context.Background()
	lb := testenv.NewLoopback(ctx, b, testenv.LoopbackConfig{
		TickIntervalUpstream:   upstreamInterval,
		TickIntervalDownstream: downstreamInterval,
	})
	env := lb.Env
	b.Cleanup(env.Close)

	sessionID := env.CreateSession(b)
	// A console must be attached for DeliverFrame fan-out to be observable by
	// WaitForEcho (which polls CollectFrames over attached consoles/probes).
	_ = env.AttachConsole(b, sessionID)

	latencies := make([]time.Duration, 0, samples)

	b.ResetTimer()
	for i := 0; i < samples; i++ {
		start := time.Now()
		env.SendKeystroke(b, sessionID, "x")
		env.WaitForEcho(b, sessionID, "x", echoTimeout)
		latencies = append(latencies, time.Since(start))
	}
	b.StopTimer()

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	p99idx := int(float64(len(latencies)) * 0.99)
	if p99idx >= len(latencies) {
		p99idx = len(latencies) - 1
	}
	p99 := latencies[p99idx]

	b.ReportMetric(float64(p99)/float64(time.Millisecond), "p99_rtt_ms")

	// Floor guard only. Exceeding 100ms here would indicate a pathological
	// regression in the synchronous fan-out path; it is NOT the VP-042 lock gate
	// (see package comment — this path does not exercise tick scheduling / ARQ /
	// multipath, so passing this guard does not prove the VP-042 property).
	if p99 > maxP99 {
		b.Errorf("keystroke-to-echo p99 %v exceeds NFR-001 floor %v (lower-bound path)", p99, maxP99)
	}
}
