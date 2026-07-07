package main

import (
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/config"
	"github.com/arcavenae/switchboard/internal/drain"
)

// TestDrainTimeoutFromConfig verifies AC-005 (BC-2.09.003 PC-7 application):
// cfg.DrainTimeout is the single source of truth; zero → drain.DefaultTimeout
// (10s per ARCH-06). Story task list §Tasks item Config-extension is already
// satisfied by S-6.01 (fields + validation live in internal/config); this
// test verifies the runRouter seam that consumes those fields.
func TestDrainTimeoutFromConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cfg  *config.Config
		want time.Duration
	}{
		{"nil cfg -> default 10s", nil, drain.DefaultTimeout},
		{"zero DrainTimeout -> default 10s", &config.Config{}, drain.DefaultTimeout},
		{"positive DrainTimeout -> verbatim", &config.Config{DrainTimeout: 15 * time.Second}, 15 * time.Second},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := drainTimeoutFor(tc.cfg); got != tc.want {
				t.Fatalf("drainTimeoutFor = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestKeepaliveIntervalFromConfig verifies AC-007 (BC-2.09.003 PC-8
// application): cfg.KeepaliveInterval is the single source of truth; zero
// → 1s per FM-009 / ARCH-06.
//
// Critically, keepaliveIntervalFor MUST NOT return sweepDeadline (60s console
// eviction) even for a nil / zero cfg — the two semantic domains are
// disjoint per BC-2.09.003 PC-8 normative note.
func TestKeepaliveIntervalFromConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cfg  *config.Config
		want time.Duration
	}{
		{"nil cfg -> default 1s", nil, defaultKeepaliveInterval},
		{"zero KeepaliveInterval -> default 1s", &config.Config{}, defaultKeepaliveInterval},
		{"positive KeepaliveInterval -> verbatim", &config.Config{KeepaliveInterval: 500 * time.Millisecond}, 500 * time.Millisecond},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := keepaliveIntervalFor(tc.cfg); got != tc.want {
				t.Fatalf("keepaliveIntervalFor = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestKeepaliveIntervalNotSweepDeadline is a normative-fence test:
// BC-2.09.003 PC-8 forbids wiring keepalive_interval into sweepDeadline.
// The two constants must be different by construction; the test asserts
// the constants live at distinct values so a copy-paste conflation is
// caught at test time. Complements AC-007.
func TestKeepaliveIntervalNotSweepDeadline(t *testing.T) {
	t.Parallel()
	if defaultKeepaliveInterval == sweepDeadline {
		t.Fatalf("defaultKeepaliveInterval == sweepDeadline (%v); BC-2.09.003 PC-8 forbids conflating console-eviction inactivity with node-reconnect keepalive cadence",
			defaultKeepaliveInterval)
	}
}

// TestUpstreamRoutersFromConfig verifies AC-006 (BC-2.09.003 PC-9 application):
// cfg.UpstreamRouters flows through to the PE-mode application seam. Empty
// list means E mode (BC-2.09.001 PC-1); non-empty means PE-mode graduation
// eligibility.
func TestUpstreamRoutersFromConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cfg  *config.Config
		want []string
	}{
		{"nil cfg -> E mode (empty)", nil, nil},
		{"empty upstream_routers -> E mode", &config.Config{}, nil},
		{
			"single upstream -> PE mode with one entry",
			&config.Config{UpstreamRouters: []config.UpstreamRouter{{Addr: "10.0.1.1:9090"}}},
			[]string{"10.0.1.1:9090"},
		},
		{
			"two upstreams -> PE mode with both entries preserved in order",
			&config.Config{UpstreamRouters: []config.UpstreamRouter{
				{Addr: "10.0.1.1:9090"},
				{Addr: "10.0.1.2:9090"},
			}},
			[]string{"10.0.1.1:9090", "10.0.1.2:9090"},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := upstreamRoutersFor(tc.cfg)
			if len(got) != len(tc.want) {
				t.Fatalf("upstreamRoutersFor returned %d entries, want %d (%v vs %v)",
					len(got), len(tc.want), got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("entry[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// TestUpstreamRoutersFromConfig_ReturnsFreshSlice verifies that mutating the
// returned slice does not affect subsequent calls or the underlying cfg.
// This is defensive isolation against a future runtime reload path.
func TestUpstreamRoutersFromConfig_ReturnsFreshSlice(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{UpstreamRouters: []config.UpstreamRouter{
		{Addr: "10.0.1.1:9090"},
	}}
	first := upstreamRoutersFor(cfg)
	first[0] = "clobbered:0000"
	second := upstreamRoutersFor(cfg)
	if second[0] != "10.0.1.1:9090" {
		t.Fatalf("second call returned %q, want unchanged '10.0.1.1:9090' (fresh-slice contract broken)", second[0])
	}
}

// TestUpstreamRoutersAsSet verifies the set-equal helper used by the Connector
// reconciler for Q1 set-diff semantics (AC-001, BC-2.09.001 EC-002).
// Two slices with identical addresses in different orders produce the same set.
func TestUpstreamRoutersAsSet(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		addrs []string
		want  map[string]struct{}
	}{
		{"nil input -> empty set", nil, map[string]struct{}{}},
		{"empty input -> empty set", []string{}, map[string]struct{}{}},
		{
			"single address",
			[]string{"10.0.1.1:9090"},
			map[string]struct{}{"10.0.1.1:9090": {}},
		},
		{
			"two addresses — set-equal with any order",
			[]string{"10.0.1.1:9090", "10.0.1.2:9090"},
			map[string]struct{}{"10.0.1.1:9090": {}, "10.0.1.2:9090": {}},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := upstreamRoutersAsSet(tc.addrs)
			if len(got) != len(tc.want) {
				t.Fatalf("upstreamRoutersAsSet(%v) returned %d keys, want %d", tc.addrs, len(got), len(tc.want))
			}
			for k := range tc.want {
				if _, ok := got[k]; !ok {
					t.Fatalf("upstreamRoutersAsSet(%v) missing key %q", tc.addrs, k)
				}
			}
		})
	}
}
