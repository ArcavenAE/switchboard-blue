// Package bench_test contains integration benchmarks that span multiple
// internal packages. Benchmarks here are DIAGNOSTIC only — not wired to any
// required CI check per ADR-007. Run them manually on stable hardware to
// produce VP evidence.
//
// VP-042 — BenchmarkKeystrokeEcho_P99:
//
//	500 keystroke-to-echo round trips over an in-process loopback stack
//	(session.AccessNode + echo sink). Reports p99_rtt_ms; enforces VP-042
//	gate (≤ 100ms p99) via b.Errorf.
//
// Ruling Divergence (S-BL.BENCH):
//
//	VP-042.md proof harness skeleton calls testenv.NewLoopback, which
//	requires S-BL.TESTENV (backlog, not yet delivered). This benchmark
//	delivers equivalent measurement semantics using session.AccessNode +
//	an inline echo sink. When S-BL.TESTENV ships, the harness may be
//	migrated to the canonical testenv.NewLoopback API; the VP-042 lock can
//	be evaluated at that point. The loopback here is fully in-process with
//	the same statistical interpretation: 500 samples, p99 gate ≤ 100ms.
//
// Architecture note:
//
//	The echo sink calls AccessNode.DeliverFrame synchronously from within
//	SendInput, which is called synchronously from SendKeystroke. This
//	creates an immediate same-goroutine delivery path: the WaitForEcho
//	receive on the downstream channel is unblocked before SendKeystroke
//	returns. This in-process path is faster than the real network path
//	(no arq, no multipath, no wire encoding) and therefore represents a
//	lower bound on latency. On the real stack the 100ms budget accommodates
//	10ms upstream tick cadence + 50ms downstream tick cadence + ARQ overhead.
//	The loopback demonstrates the pure in-process overhead; VP-042 on
//	the full stack is gated on S-BL.TESTENV.
package bench_test

import (
	"sort"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/session"
)

// echoSink is a KeystrokeSink that echoes received keystrokes back as a
// FrameTypeData frame via the injected AccessNode.DeliverFrame. This creates
// the loopback: SendKeystroke → echoSink.SendInput → DeliverFrame → downstream
// channel unblocked.
//
// The AccessNode reference is set after construction (set field after New) to
// avoid the circular construction dependency.
type echoSink struct {
	node *session.AccessNode
}

// SendInput echoes payload back as a FrameTypeData frame with PayloadLen set to
// the payload length. The frame is structurally minimal — its fields satisfy
// frame.OuterHeader but carry no real wire encoding. The benchmark measures
// the delivery pipeline latency, not wire-format correctness.
func (s *echoSink) SendInput(payload []byte) error {
	echo := frame.OuterHeader{
		Version:    frame.VersionByte,
		FrameType:  frame.FrameTypeData,
		PayloadLen: uint16(len(payload)),
	}
	s.node.DeliverFrame(echo)
	return nil
}

// BenchmarkKeystrokeEcho_P99 measures the keystroke-to-echo round-trip
// latency over 500 samples on an in-process loopback stack. It reports
// p99_rtt_ms as a custom metric and enforces the VP-042 gate (≤ 100ms p99)
// via b.Errorf.
//
// Loopback path:
//
//	SendKeystroke → echoSink.SendInput → DeliverFrame → downstream channel
//
// Hardware note: this is a lower-bound measurement (no network, no arq, no
// tick scheduling). The gate is expected to pass trivially on any hardware.
// VP-042 on the full stack (with tick intervals) requires S-BL.TESTENV.
//
// Run with: go test -bench=BenchmarkKeystrokeEcho_P99 -benchtime=500x ./internal/bench/
// or via:   just bench
func BenchmarkKeystrokeEcho_P99(b *testing.B) {
	const (
		sessionName = "bench-session"
		consoleKey  = session.ConsoleKey("bench-console")
		samples     = 500
		maxP99      = 100 * time.Millisecond // NFR-001 / VP-042
	)

	// Build the in-process loopback stack.
	ks := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(ks)

	sink := &echoSink{}
	node := session.NewAccessNode(pub, nil,
		session.WithKeystrokeSink(sink),
	)
	sink.node = node // wire echo loop

	if err := pub.Publish(sessionName); err != nil {
		b.Fatalf("Publish: %v", err)
	}

	downstream, _, err := node.Attach(consoleKey, sessionName)
	if err != nil {
		b.Fatalf("Attach: %v", err)
	}
	b.Cleanup(func() {
		_ = node.Detach(consoleKey, sessionName)
	})

	payload := []byte("x")
	latencies := make([]time.Duration, 0, samples)

	// Warm-up: one round trip before measurement starts to exercise caches
	// and goroutine scheduling paths.
	if err := node.SendKeystroke(consoleKey, sessionName, payload); err != nil {
		b.Fatalf("warmup SendKeystroke: %v", err)
	}
	select {
	case <-downstream:
	case <-time.After(500 * time.Millisecond):
		b.Fatal("warmup: echo not received within 500ms")
	}

	b.ResetTimer()
	for i := 0; i < samples; i++ {
		start := time.Now()

		if err := node.SendKeystroke(consoleKey, sessionName, payload); err != nil {
			b.Fatalf("SendKeystroke[%d]: %v", i, err)
		}
		// WaitForEcho: block until the echo frame arrives on the downstream
		// channel (written by echoSink.SendInput → DeliverFrame).
		select {
		case <-downstream:
		case <-time.After(500 * time.Millisecond):
			b.Fatalf("sample %d: echo not received within 500ms", i)
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

	// VP-042 gate (S-BL.BENCH AC-002): enforce ≤ 100ms p99 (NFR-001).
	// This loopback is lower-bound only; the full-stack gate requires S-BL.TESTENV.
	if p99 > maxP99 {
		b.Errorf("keystroke-to-echo p99 %v exceeds NFR-001 limit %v (VP-042)", p99, maxP99)
	}
}
