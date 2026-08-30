//go:build integration

// This file is the testenv-integrated counterpart to the tag-free lower-bound
// benchmark in keystroke_echo_bench_test.go. It is guarded by the `integration`
// build tag (the same convention internal/testenv integration tests use) so the
// tag-free lower-bound bench remains buildable and runnable on its own.
//
// VP-042 status:
//
//	S-BL.LOOPBACK-FULLSTACK (AC-013) updates this benchmark from its prior
//	two-call env.SendKeystroke/env.WaitForEcho shape (the VP-042.md skeleton
//	shape, now superseded) to the binding token-based RoundTrip API:
//	testenv.LoopbackEnv.SendKeystroke returns a testenv.RoundTrip;
//	testenv.LoopbackEnv.WaitForEcho(t, rt, timeout) returns ([]byte, bool).
//	Both are methods on *LoopbackEnv (Q2), not *Env — CreateSession/
//	SendKeystroke/WaitForEcho are all called on lb directly, not lb.Env.
//
//	Once loopback.go's loopbackDriver is implemented, this benchmark drives
//	the real tick-driven, protocol-accurate loopback stack (internal/
//	halfchannel + internal/arq + internal/multipath + internal/paths)
//	instead of the prior same-goroutine DeliverFrame shortcut. The "lower
//	bound only" framing this file previously carried (disclosing that the
//	measured path bypassed arq/multipath/tick-scheduling) is retired — that
//	divergence no longer exists once the token-based API is backed by the
//	full stack.
//
//	AS OF THIS COMMIT (S-BL.LOOPBACK-FULLSTACK Red Gate ②): loopback.go's
//	loopbackDriver is stub-only (every non-trivial body panics per
//	BC-5.38.001) — CreateSession/SendKeystroke/WaitForEcho below will PANIC
//	when this benchmark actually runs, until the implementer stage fills
//	those stubs in. That is the intended Red Gate signal for this file too;
//	do not treat a failing/panicking run as evidence against the token-based
//	API shape itself — see AC-013's Verification Method (compile-only gate)
//	for what this story's Red Gate requires of this file.
package bench_test

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/testenv"
)

// BenchmarkKeystrokeToEcho_P99 is the testenv-integrated keystroke-to-echo p99
// benchmark from the VP-042 proof-harness skeleton, adapted to
// S-BL.LOOPBACK-FULLSTACK's binding token-based RoundTrip API (Design
// Constraints Q5 — superseding the VP-042.md skeleton's older two-call
// env.SendKeystroke/env.WaitForEcho shape; see this story's Forward
// Obligation section for the follow-up VP-042.md skeleton update).
//
// This runs under `-tags integration`. See the package-level comment for
// current Red Gate status.
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
		maxP99             = 100 * time.Millisecond // NFR-001 / VP-042 ceiling guard
		echoTimeout        = 500 * time.Millisecond
	)

	ctx := context.Background()
	lb := testenv.NewLoopback(ctx, b, testenv.LoopbackConfig{
		TickIntervalUpstream:   upstreamInterval,
		TickIntervalDownstream: downstreamInterval,
	})
	b.Cleanup(lb.Env.Close)

	sessionID := lb.CreateSession(b)

	latencies := make([]time.Duration, 0, samples)

	b.ResetTimer()
	for i := 0; i < samples; i++ {
		start := time.Now()
		rt := lb.SendKeystroke(b, sessionID, "x")
		if _, ok := lb.WaitForEcho(b, rt, echoTimeout); !ok {
			b.Fatalf("BenchmarkKeystrokeToEcho_P99: WaitForEcho timed out on sample %d", i)
		}
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

	// Ceiling guard: NFR-001 / VP-042's 100ms budget, measured against the
	// full tick-driven protocol stack once implemented (see package
	// comment).
	if p99 > maxP99 {
		b.Errorf("keystroke-to-echo p99 %v exceeds NFR-001 limit %v", p99, maxP99)
	}
}
