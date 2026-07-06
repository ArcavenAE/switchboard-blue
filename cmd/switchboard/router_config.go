package main

import (
	"time"

	"github.com/arcavenae/switchboard/internal/config"
	"github.com/arcavenae/switchboard/internal/drain"
)

// This file collects the application-point helpers that close the three
// BC-2.09.003 DEFERRED-APPLICATION obligations owned by S-7.04:
//
//   - drain_timeout      → drainTimeoutFor (BC-2.09.003 PC-7, AC-005)
//   - keepalive_interval → keepaliveIntervalFor (BC-2.09.003 PC-8, AC-007)
//   - upstream_routers   → upstreamRoutersFor (BC-2.09.003 PC-9, AC-006)
//
// Each helper mirrors the shape of tickIntervalFor in access.go: a single
// source of truth for the resolved value, with zero-value semantics per
// BC-2.09.003 (zero/absent → daemon default; negative → validation catches
// at config-parse time so this seam never sees a negative value).

// defaultKeepaliveInterval is the node-reconnect keepalive cadence used
// when cfg.KeepaliveInterval is zero (BC-2.09.003 PC-8 zero-value semantics;
// ARCH-06 §Graceful Drain "keepalive_interval, default 1s"; FM-009).
//
// This constant is DISTINCT from sweepDeadline in access.go — sweepDeadline
// governs console eviction (60s inactivity window) and is semantically
// unrelated to node reconnect keepalives. BC-2.09.003 PC-8 explicitly
// forbids wiring keepalive_interval into sweepDeadline; the packages named
// here document that fence.
const defaultKeepaliveInterval = 1 * time.Second

// drainTimeoutFor returns the drain window to hand to drain.New.
//
// When cfg is non-nil and cfg.DrainTimeout > 0, cfg.DrainTimeout is the single
// source of truth. When cfg is nil or cfg.DrainTimeout is zero, drain.DefaultTimeout
// (10s per ARCH-06) is returned. Negative values are impossible in this seam:
// Config.Validate rejects them (E-CFG-006).
//
// BC-2.09.003 PC-7 application point; AC-005.
func drainTimeoutFor(cfg *config.Config) time.Duration {
	if cfg != nil && cfg.DrainTimeout > 0 {
		return cfg.DrainTimeout
	}
	return drain.DefaultTimeout
}

// keepaliveIntervalFor returns the node-reconnect keepalive ticker cadence.
//
// When cfg is non-nil and cfg.KeepaliveInterval > 0, cfg.KeepaliveInterval is
// the single source of truth. When cfg is nil or cfg.KeepaliveInterval is zero,
// defaultKeepaliveInterval (1s per ARCH-06 / FM-009) is returned. Negative values
// are impossible in this seam: Config.Validate rejects them (E-CFG-007).
//
// BC-2.09.003 PC-8 application point; AC-007. MUST NOT be routed into
// sweepDeadline (console eviction, a different semantic).
func keepaliveIntervalFor(cfg *config.Config) time.Duration {
	if cfg != nil && cfg.KeepaliveInterval > 0 {
		return cfg.KeepaliveInterval
	}
	return defaultKeepaliveInterval
}

// upstreamRoutersFor returns the configured upstream router addresses for
// PE-mode operation. An empty return value means E mode (no upstream
// connections attempted).
//
// The return is a fresh slice of Addr strings — the caller does not hold
// a reference into cfg.UpstreamRouters. This isolates the runRouter call
// site from any future runtime reload path that mutates cfg.
//
// BC-2.09.003 PC-9 application point; AC-006.
// BC-2.09.001 PC-1 semantics: non-empty return → PE-mode graduation.
func upstreamRoutersFor(cfg *config.Config) []string {
	if cfg == nil || len(cfg.UpstreamRouters) == 0 {
		return nil
	}
	out := make([]string, 0, len(cfg.UpstreamRouters))
	for _, u := range cfg.UpstreamRouters {
		out = append(out, u.Addr)
	}
	return out
}
