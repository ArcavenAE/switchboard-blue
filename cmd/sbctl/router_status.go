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

// qualityFromPathEntry computes the green/yellow/red/pending quality label for a path entry.
//
// Evaluation order (BC-2.06.003 PC-1, EC-003):
//  1. If rtt_p99_ms == "pending" (< 10 samples), return "pending" immediately.
//  2. If status == "failed", return "red".
//  3. If status == "degraded", return max("yellow", band-classified quality).
//  4. Otherwise, classify by p99 RTT + loss using stateless metrics.Classify.
//
// AC-003 / BC-2.06.003 PC-1, PC-3
func qualityFromPathEntry(entry PathEntry) string {
	// Step 1: pending sentinel — fewer than 10 samples (BC-2.06.003 EC-003).
	if s, ok := entry.P99RTTMs.(string); ok && s == "pending" {
		return "pending"
	}

	// Determine the effective RTT (p99 preferred; fall back to rtt_ms for non-pending strings).
	var rttMs float64
	switch v := entry.P99RTTMs.(type) {
	case float64:
		rttMs = v
	default:
		rttMs = entry.RTTMs
	}

	// Step 2: failed status → unconditional red (BC-2.06.003 PC-1).
	if entry.Status == "failed" {
		return "red"
	}

	// Step 3: stateless band classification (no hysteresis in CLI one-shot; F-C1).
	band := metrics.Classify(rttMs, entry.LossPct).String()

	// Step 4: degraded status applies a yellow floor (max of yellow, band-derived; F-H1).
	if entry.Status == "degraded" && band == "green" {
		return "yellow"
	}

	return band
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
// It appends a `quality` field (green/yellow/red/pending) derived from status + rtt_p99_ms.
//
// The --target flag in args overrides the caller-supplied target address, matching
// the alias semantics in BC-2.06.003 PC-3: `--target <router>` is equivalent to
// `sbctl --target <router> paths list`.
//
// PathEntry fields are passed through as raw JSON (no re-encode) to preserve
// exact wire representation from the daemon (F-H3 / BC-2.06.003 EC-005).
//
// AC-003 / BC-2.06.003 PC-3
func runRouterStatus(ctx context.Context, target, keyPath string, useJSON bool, args []string) error {
	// Parse --target override from args (alias semantics: --target overrides daemon address).
	// F-M1: --target without a following value is a configuration error (E-CFG-010).
	for i, arg := range args {
		if arg == "--target" {
			if i+1 >= len(args) {
				err := fmt.Errorf("E-CFG-010: router status: --target requires a value")
				writeError(useJSON, "E-CFG-010", "router status: --target requires a value")
				return err
			}
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

	// F-L1: single net.Conn variable (no redundant dual-interface vars).
	var netConn net.Conn
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
	defer func() { _ = netConn.Close() }()

	if err = Authenticate(ctx, netConn, privKey); err != nil {
		writeError(useJSON, "E-ADM-010", "authentication failed")
		return err
	}

	data, err := dispatch(ctx, netConn, "paths.list", nil)
	if err != nil {
		writeError(useJSON, "E-RPC-001", err.Error())
		return err
	}

	// Decode paths.list response as a slice of raw JSON objects.
	// Raw passthrough preserves exact wire representation (F-H3 / BC-2.06.003 EC-005):
	// float values like 20.0 remain 20.0 rather than being re-encoded as 20.
	var rawEntries []json.RawMessage
	if err = json.Unmarshal(data, &rawEntries); err != nil {
		writeError(useJSON, "E-RPC-001", fmt.Sprintf("decode paths: %s", err))
		return fmt.Errorf("decode paths: %w", err)
	}

	if useJSON {
		// Inject quality into each raw entry object and write the annotated array.
		annotated := make([]json.RawMessage, len(rawEntries))
		for i, raw := range rawEntries {
			// Decode only enough to compute quality; the raw bytes are preserved.
			var entry PathEntry
			if err = json.Unmarshal(raw, &entry); err != nil {
				writeError(useJSON, "E-RPC-001", fmt.Sprintf("decode entry %d: %s", i, err))
				return fmt.Errorf("decode entry %d: %w", i, err)
			}
			quality := qualityFromPathEntry(entry)
			// Inject quality field into the raw JSON object by stripping the trailing
			// "}" and appending the new field.
			injected := injectJSONField(raw, "quality", quality)
			annotated[i] = injected
		}
		qData, err := json.Marshal(annotated)
		if err != nil {
			writeError(useJSON, "E-RPC-001", fmt.Sprintf("marshal quality entries: %s", err))
			return fmt.Errorf("marshal quality entries: %w", err)
		}
		writeSuccess(useJSON, qData)
		return nil
	}

	// Human-readable: decode into typed entries for tabular formatting.
	withQuality := make([]PathEntryWithQuality, len(rawEntries))
	for i, raw := range rawEntries {
		var entry PathEntry
		if err = json.Unmarshal(raw, &entry); err != nil {
			writeError(useJSON, "E-RPC-001", fmt.Sprintf("decode entry %d: %s", i, err))
			return fmt.Errorf("decode entry %d: %w", i, err)
		}
		withQuality[i] = PathEntryWithQuality{
			PathEntry: entry,
			Quality:   qualityFromPathEntry(entry),
		}
	}
	formatPathsTable(withQuality)
	return nil
}

// injectJSONField appends a string field to a JSON object literal.
// raw must be a valid JSON object (ends with "}"). The field is appended
// before the closing brace. This avoids re-encoding the object through
// Go structs, which would normalise float representations (F-H3).
func injectJSONField(raw json.RawMessage, key, value string) json.RawMessage {
	// Trim trailing whitespace and the closing brace.
	b := []byte(strings.TrimRight(string(raw), " \t\r\n"))
	if len(b) == 0 || b[len(b)-1] != '}' {
		// Malformed object: return as-is (caller will catch decode errors).
		return raw
	}
	// Append the new field: strip trailing "}", add field, close.
	b = b[:len(b)-1]
	fieldJSON, _ := json.Marshal(value)
	keyJSON, _ := json.Marshal(key)
	b = append(b, ',')
	b = append(b, keyJSON...)
	b = append(b, ':')
	b = append(b, fieldJSON...)
	b = append(b, '}')
	return json.RawMessage(b)
}
