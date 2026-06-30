// router_status.go implements `sbctl router status --target <router>` as a
// convenience alias for `sbctl paths list`.
//
// This is a CLI dispatch shim only. There is exactly one code path in the
// underlying query layer serving both commands — no divergent implementations
// (BC-2.06.003 PC-3 + EC-005; F-P8-002 ruling).
//
// The `quality` column (green/yellow/red) is added by this formatter on top of
// the PathEntry output. The underlying data source is identical to `sbctl paths list`.
//
// Flags:
//
//	--target <router>   overrides the default daemon address
//
// Output format: identical to `sbctl paths list` plus a `quality` field
// (green/yellow/red derived from status + rtt_p99_ms).
//
// AC-003 / BC-2.06.003 PC-3
//
// Purity classification (ARCH-09): effectful-boundary — network I/O to daemon socket.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/arcavenae/switchboard/internal/metrics"
)

// PathEntryWithQuality extends PathEntry with the quality field for the
// `sbctl router status` alias output.
//
// AC-003 / BC-2.06.003 PC-3
type PathEntryWithQuality struct {
	PathEntry
	Quality string `json:"quality"`
}

// qualityFromPathEntry computes the green/yellow/red quality label for a path entry.
// Uses the metrics.classify thresholds (BC-2.06.001 v1.3):
//   - p99 RTT > 500 ms OR loss > 20 % → red
//   - p99 RTT > 100 ms OR loss > 5 % → yellow
//   - otherwise → green
//
// When rtt_p99_ms is "pending" (< 10 samples; BC-2.06.003 EC-003), falls back
// to rtt_ms for the classification.
//
// AC-003 / BC-2.06.003 PC-3
func qualityFromPathEntry(entry PathEntry) string {
	// Determine the effective RTT to classify against.
	var rttMs float64
	switch v := entry.P99RTTMs.(type) {
	case float64:
		rttMs = v
	case string:
		// "pending" — fewer than 10 samples; fall back to rtt_ms.
		rttMs = entry.RTTMs
	default:
		rttMs = entry.RTTMs
	}

	// Instantiate a fresh indicator and classify directly via a single Update.
	// We do not track hysteresis for the one-shot CLI display.
	qi := metrics.NewQualityIndicator()
	qi.Update(rttMs, entry.LossPct)
	return qi.Current().String()
}

// formatPathsTable renders a slice of PathEntryWithQuality as a human-readable
// tab-separated table to stdout.
//
// AC-003 / BC-2.06.003 PC-3
func formatPathsTable(entries []PathEntryWithQuality) {
	fmt.Printf("%-20s %-22s %10s %10s %10s %-10s %-8s\n",
		"PATH_ID", "ROUTER_ADDR", "RTT_MS", "P99_MS", "LOSS_PCT", "STATUS", "QUALITY")
	for _, e := range entries {
		p99 := fmt.Sprintf("%v", e.P99RTTMs)
		fmt.Printf("%-20s %-22s %10.2f %10s %10.2f %-10s %-8s\n",
			e.PathID, e.RouterAddr, e.RTTMs, p99, e.LossPct, e.Status, e.Quality)
	}
}

// runRouterStatus implements `sbctl router status --target <router>`.
// This is a CLI shim that calls the same underlying query path as runPathsList.
// It appends a `quality` field (green/yellow/red) derived from status + rtt_p99_ms.
//
// The --target flag in args overrides the caller-supplied target address, matching
// the alias semantics in BC-2.06.003 PC-3: `--target <router>` is equivalent to
// `sbctl --target <router> paths list`.
//
// AC-003 / BC-2.06.003 PC-3
func runRouterStatus(ctx context.Context, target, keyPath string, useJSON bool, args []string) error {
	// Parse --target override from args (alias semantics: --target overrides daemon address).
	for i, arg := range args {
		if arg == "--target" && i+1 < len(args) {
			target = args[i+1]
		} else if strings.HasPrefix(arg, "--target=") {
			target = strings.TrimPrefix(arg, "--target=")
		}
	}

	// Dial and authenticate, then dispatch "paths.list" — same RPC as runPathsList.
	// We handle the response ourselves to inject the quality field.
	privKey, err := loadEd25519Key(keyPath, os.UserHomeDir)
	if err != nil {
		writeError(useJSON, "E-CFG-010", err.Error())
		return err
	}

	var conn interface {
		Close() error
	}
	var netConn interface {
		net.Conn
	}

	if len(target) > 0 && target[0] == '/' {
		netConn, err = (&net.Dialer{}).DialContext(ctx, "unix", target)
	} else {
		netConn, err = (&net.Dialer{}).DialContext(ctx, "tcp", target)
	}
	if err != nil {
		msg := fmt.Sprintf("daemon unreachable: %s: %s", target, err)
		writeError(useJSON, "E-NET-001", msg)
		return fmt.Errorf("E-NET-001: %s", msg)
	}
	conn = netConn
	defer func() { _ = conn.Close() }()

	if err = Authenticate(ctx, netConn, privKey); err != nil {
		writeError(useJSON, "E-ADM-010", "authentication failed")
		return err
	}

	data, err := dispatch(ctx, netConn, "paths.list", nil)
	if err != nil {
		writeError(useJSON, "E-RPC-001", err.Error())
		return err
	}

	// Decode paths.list response into []PathEntry.
	var entries []PathEntry
	if err = json.Unmarshal(data, &entries); err != nil {
		writeError(useJSON, "E-RPC-001", fmt.Sprintf("decode paths: %s", err))
		return fmt.Errorf("decode paths: %w", err)
	}

	// Annotate each entry with quality.
	withQuality := make([]PathEntryWithQuality, len(entries))
	for i, e := range entries {
		withQuality[i] = PathEntryWithQuality{
			PathEntry: e,
			Quality:   qualityFromPathEntry(e),
		}
	}

	if useJSON {
		// Marshal annotated entries and wrap in the standard success envelope.
		qData, err := json.Marshal(withQuality)
		if err != nil {
			writeError(useJSON, "E-RPC-001", fmt.Sprintf("marshal quality entries: %s", err))
			return fmt.Errorf("marshal quality entries: %w", err)
		}
		writeSuccess(useJSON, qData)
		return nil
	}

	// Human-readable: tabular output with quality column.
	formatPathsTable(withQuality)
	return nil
}
