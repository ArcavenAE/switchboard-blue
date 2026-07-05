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
	"io"
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
//  1. If rtt_p99_ms.Kind == PendingKind, return "pending" immediately — fewer
//     than 10 samples or indeterminate state.
//  2. If status == "failed", return "red".
//  3. If status == "degraded", return max("yellow", band-classified quality).
//  4. Otherwise, classify by p99 RTT + loss using stateless metrics.Classify.
//
// Per BC-2.06.003 v1.5 F-M3: pending p99 must yield pending quality; the pre-v1.5
// fallback-to-rtt_ms behaviour is removed.
//
// Kind is authoritative post-#54: RTTValue.UnmarshalJSON populates Kind from
// the wire form (float → FloatKind, "pending" → PendingKind, null → decode
// error) and rejects legacy null shapes (F-P2L1-004).
//
// AC-003 / BC-2.06.003 PC-1, PC-3
func qualityFromPathEntry(entry PathEntry) string {
	// Step 1: pending p99 → pending (regardless of status).
	// BC-2.06.003 v1.5 F-M3 + degraded+pending precedence (S502-DEFER-3).
	if entry.P99RTTMs.Kind == metrics.PendingKind {
		return "pending"
	}

	// Step 2: failed status → unconditional red (BC-2.06.003 PC-1).
	if entry.Status == "failed" {
		return "red"
	}

	// Step 3: stateless band classification (no hysteresis in CLI one-shot; F-C1).
	band := metrics.Classify(entry.P99RTTMs.Value, entry.LossPct).String()

	// Step 4: degraded status applies a yellow floor (max of yellow, band-derived; F-H1).
	if entry.Status == "degraded" && band == "green" {
		return "yellow"
	}

	return band
}

// formatP99 converts a PathEntry.P99RTTMs value to its human-readable string
// for table output.
//
// When p99.Kind == PendingKind, formatP99 returns "pending" — the AC-004
// sentinel per BC-2.06.003 EC-003. When p99.Kind == FloatKind it returns the
// formatted number (e.g. "22.00").
func formatP99(p99 metrics.RTTValue) string {
	if p99.Kind == metrics.PendingKind {
		return "pending"
	}
	return fmt.Sprintf("%.2f", p99.Value)
}

// formatPathsTable renders a slice of PathEntryWithQuality as a human-readable
// tab-separated table to out.
//
// AC-003 / BC-2.06.003 PC-3
func formatPathsTable(out io.Writer, entries []PathEntryWithQuality) {
	_, _ = fmt.Fprintf(out, "%-20s %-22s %10s %10s %10s %-10s %-8s\n",
		"PATH_ID", "ROUTER_ADDR", "RTT_MS", "P99_MS", "LOSS_PCT", "STATUS", "QUALITY")
	for _, e := range entries {
		_, _ = fmt.Fprintf(out, "%-20s %-22s %10.2f %10s %10.2f %-10s %-8s\n",
			e.PathID, e.RouterAddr, e.RTTMs, formatP99(e.P99RTTMs), e.LossPct, e.Status, e.Quality)
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
func runRouterStatus(ctx context.Context, target, keyPath string, useJSON bool, args []string, sio sbctlIO) error {
	// Parse --target override from args (alias semantics: --target overrides daemon address).
	// F-M1: --target without a following value is a configuration error (E-CFG-010).
	for i, arg := range args {
		if arg == "--target" {
			if i+1 >= len(args) {
				_ = writeError(useJSON, "E-CFG-010", "router status: --target requires a value", sio)
				return reported(usageErrf("E-CFG-010: router status: --target requires a value"))
			}
			target = args[i+1]
		} else if strings.HasPrefix(arg, "--target=") {
			target = strings.TrimPrefix(arg, "--target=")
		}
	}

	// F-C3: --target= (empty value after equals) is a configuration error (E-CFG-010).
	if target == "" {
		_ = writeError(useJSON, "E-CFG-010", "router status: --target requires a value", sio)
		return reported(usageErrf("E-CFG-010: router status: --target requires a value"))
	}

	// Dial and authenticate, then dispatch "paths.list" — same RPC as runPathsList.
	// We handle the response ourselves to inject the quality field.
	privKey, err := loadEd25519Key(keyPath, os.UserHomeDir)
	if err != nil {
		_ = writeError(useJSON, "E-CFG-010", err.Error(), sio)
		return reported(err)
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
		return writeError(useJSON, "E-NET-001", msg, sio)
	}
	defer func() { _ = netConn.Close() }()

	if err = Authenticate(ctx, netConn, privKey); err != nil {
		_ = writeError(useJSON, "E-ADM-010", "authentication failed", sio)
		return reported(err)
	}

	data, err := dispatch(ctx, netConn, "paths.list", nil)
	if err != nil {
		_ = writeError(useJSON, "E-RPC-001", err.Error(), sio)
		return reported(err)
	}

	// F-C4: sniff the first non-whitespace byte to detect EC-001 object form.
	// BC-2.06.003 EC-001: empty-paths response is an object {"paths":[],...}, not an array.
	// Pass object responses through directly — no quality column to inject.
	trimmed := strings.TrimLeft(string(data), " \t\r\n")
	if len(trimmed) > 0 && trimmed[0] == '{' {
		writeSuccess(useJSON, data, sio)
		return nil
	}

	// Decode paths.list response as a slice of raw JSON objects.
	// Raw passthrough preserves exact wire representation (F-H3 / BC-2.06.003 EC-005):
	// float values like 20.0 remain 20.0 rather than being re-encoded as 20.
	var rawEntries []json.RawMessage
	if err = json.Unmarshal(data, &rawEntries); err != nil {
		_ = writeError(useJSON, "E-RPC-001", fmt.Sprintf("decode paths: %s", err), sio)
		return reported(fmt.Errorf("decode paths: %w", err))
	}

	if useJSON {
		// Inject quality into each raw entry object and write the annotated array.
		annotated := make([]json.RawMessage, len(rawEntries))
		for i, raw := range rawEntries {
			// Decode only enough to compute quality; the raw bytes are preserved.
			var entry PathEntry
			if err = json.Unmarshal(raw, &entry); err != nil {
				_ = writeError(useJSON, "E-RPC-001", fmt.Sprintf("decode entry %d: %s", i, err), sio)
				return reported(fmt.Errorf("decode entry %d: %w", i, err))
			}
			quality := qualityFromPathEntry(entry)
			// Inject quality field into the raw JSON object by stripping the trailing
			// "}" and appending the new field.
			injected, injectErr := injectJSONField(raw, "quality", quality)
			if injectErr != nil {
				_ = writeError(useJSON, "E-RPC-001", fmt.Sprintf("inject quality entry %d: %s", i, injectErr), sio)
				return reported(fmt.Errorf("inject quality entry %d: %w", i, injectErr))
			}
			annotated[i] = injected
		}
		qData, err := json.Marshal(annotated)
		if err != nil {
			_ = writeError(useJSON, "E-RPC-001", fmt.Sprintf("marshal quality entries: %s", err), sio)
			return reported(fmt.Errorf("marshal quality entries: %w", err))
		}
		writeSuccess(useJSON, qData, sio)
		return nil
	}

	// Human-readable: decode into typed entries for tabular formatting.
	withQuality := make([]PathEntryWithQuality, len(rawEntries))
	for i, raw := range rawEntries {
		var entry PathEntry
		if err = json.Unmarshal(raw, &entry); err != nil {
			_ = writeError(useJSON, "E-RPC-001", fmt.Sprintf("decode entry %d: %s", i, err), sio)
			return reported(fmt.Errorf("decode entry %d: %w", i, err))
		}
		withQuality[i] = PathEntryWithQuality{
			PathEntry: entry,
			Quality:   qualityFromPathEntry(entry),
		}
	}
	formatPathsTable(sio.out, withQuality)
	return nil
}

// injectJSONField appends a string field to a JSON object literal.
// raw must be a valid JSON object (ends with "}"). The field is appended
// before the closing brace. This avoids re-encoding the object through
// Go structs, which would normalise float representations (F-H3).
//
// Returns an error if json.Marshal of key or value fails (go.md rule 3).
func injectJSONField(raw json.RawMessage, key, value string) (json.RawMessage, error) {
	// Trim trailing whitespace and the closing brace.
	b := []byte(strings.TrimRight(string(raw), " \t\r\n"))
	if len(b) == 0 || b[len(b)-1] != '}' {
		// Malformed object: return as-is (caller will catch decode errors).
		return raw, nil
	}
	// Append the new field: strip trailing "}", add field, close.
	b = b[:len(b)-1]
	fieldJSON, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal field value %q: %w", value, err)
	}
	keyJSON, err := json.Marshal(key)
	if err != nil {
		return nil, fmt.Errorf("marshal field key %q: %w", key, err)
	}
	b = append(b, ',')
	b = append(b, keyJSON...)
	b = append(b, ':')
	b = append(b, fieldJSON...)
	b = append(b, '}')
	return json.RawMessage(b), nil
}
