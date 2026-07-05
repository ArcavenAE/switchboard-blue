// security_hardening_test.go — S601-SEC-001..002 hardening for
// internal/config.  Deferred-LOW findings from S-6.01 PR review
// (.factory/code-delivery/S-6.01/pr-review.md §106-111).
//
// S601-SEC-001 (CWE-117): the --config path is echoed unsanitized into
// E-CFG-004 error strings from LoadFile.  A path containing raw control
// characters (\n, \r, \x1b, …) survives into stderr/logs and can be used
// to forge log lines or corrupt terminal output.  Fix: strip control
// characters from `path` before interpolation (same treatment addr fields
// already get via sanitizeAddrForError; F-SEC-002).
//
// S601-SEC-002 (CWE-400): Validate collects per-field failures into an
// unbounded []string slice.  While the input is implicitly bounded by
// maxConfigFileSize (1 MiB), an explicit cap yields (a) a clean operator
// error naming the truncation instead of a wall of concatenated failure
// messages and (b) a predictable upper bound on Validate's memory
// footprint independent of the file-size guard.
//
// Package config_test (external) for the same package boundary as the
// rest of the config test surface.
package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	"github.com/arcavenae/switchboard/internal/config"
)

// ---- S601-SEC-001 (CWE-117): --config path sanitization -----------------

// TestLoadFile_ConfigPathControlCharsAreSanitizedInError verifies that
// LoadFile's E-CFG-004 error string does NOT carry raw control characters
// through from the path argument.  This closes the CWE-117 log-injection
// vector for operator-supplied --config values.
//
// Attack model: operator invokes `switchboard access --config
// $'/tmp/nope\r\x1b[31m[FAKE ERROR]'` and the error message emits the
// raw sequence to stderr where it can forge log lines or repaint the
// terminal (F-SEC-002 sibling; local-access-gated, still LOW).
//
// The fix must strip control characters (unicode.IsControl) exactly like
// sanitizeAddrForError does for listen_addr / upstream_routers[N].addr.
// Printable characters must survive so the operator can still see what
// they mistyped.
//
// Refs: S601-SEC-001, CWE-117, BC-2.09.003 EC-001, F-SEC-002 (sibling).
func TestLoadFile_ConfigPathControlCharsAreSanitizedInError(t *testing.T) {
	t.Parallel()

	// Build a path that contains a variety of control characters — CR, LF,
	// ESC (start of an ANSI sequence), and a C1 byte.  The path must
	// resolve to os.ErrNotExist so we hit the E-CFG-004 branch.
	dir := t.TempDir()
	base := "not-there\r\n\x1b[31m\x85"
	badPath := filepath.Join(dir, base)

	_, err := config.LoadFile(badPath)
	if err == nil {
		t.Fatal("S601-SEC-001: expected E-CFG-004 for missing file; got nil")
	}
	var ce *config.ConfigError
	if !errors.As(err, &ce) {
		t.Fatalf("S601-SEC-001: expected *config.ConfigError, got %T: %v", err, err)
	}
	if ce.Code != "E-CFG-004" {
		t.Fatalf("S601-SEC-001: expected E-CFG-004, got %s", ce.Code)
	}

	msg := ce.Error()
	// The full msg must not contain any Unicode control character.  This
	// mirrors the assertion pattern used at internal/config/config_test.go:1447
	// for listen_addr sanitization.
	for _, r := range msg {
		if unicode.IsControl(r) {
			t.Errorf("S601-SEC-001: error message contains raw control char %q "+
				"(CWE-117 / --config path); full message: %q", r, msg)
		}
	}
	// Sanity: the printable portion of the path must still appear so the
	// operator can diagnose the typo — parity with sanitizeAddrForError.
	if !strings.Contains(msg, "not-there") {
		t.Errorf("S601-SEC-001: sanitized path lost its printable segment; "+
			"operator diagnosis broken.  message: %q", msg)
	}
}

// ---- S601-SEC-002 (CWE-400): explicit cap on Validate() failure slice ---

// TestValidate_UpstreamRoutersRespectExplicitCap verifies that Validate
// caps the number of collected upstream_routers[N] failures at a bounded,
// documented ceiling.  Without a cap, a malicious YAML file could produce
// up to ~100K invalid entries within the 1 MiB file guard, each becoming
// a formatted failure string in memory.  With the cap, the caller sees a
// clear "too many upstream_routers failures" signal instead of unbounded
// growth.
//
// Contract asserted:
//   - c.Validate() returns *ConfigError with code E-CFG-001.
//   - The Detail string is bounded — it does NOT interpolate every one of
//     N invalid entries verbatim (where N ≫ cap).
//   - It DOES emit an explicit truncation marker naming the cap so the
//     operator knows the list was clipped rather than silently squashed.
//
// Note on cap size: config.UpstreamRoutersFailureCap is exposed as a
// package constant so the test and the implementation share the same
// authoritative value.
//
// Refs: S601-SEC-002, CWE-400.
func TestValidate_UpstreamRoutersRespectExplicitCap(t *testing.T) {
	t.Parallel()

	// Build a Config with far more invalid upstream_routers than the cap.
	// Every entry has a malformed addr so each triggers a validate failure.
	// Using an oversubscription factor (cap × 3) guarantees a truncation
	// event regardless of cap value.
	cap := config.UpstreamRoutersFailureCap
	if cap <= 0 {
		t.Fatalf("S601-SEC-002: config.UpstreamRoutersFailureCap must be positive; got %d", cap)
	}
	over := cap * 3

	cfg := &config.Config{
		ListenAddr:   "0.0.0.0:9090",
		TickInterval: 10_000_000, // 10ms in ns; inside [5ms, 50ms]
	}
	cfg.UpstreamRouters = make([]config.UpstreamRouter, 0, over)
	for i := 0; i < over; i++ {
		cfg.UpstreamRouters = append(cfg.UpstreamRouters, config.UpstreamRouter{
			Addr: "not-a-host-port", // malformed → validateHostPort fails
		})
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("S601-SEC-002: expected E-CFG-001 for oversubscribed upstream_routers; got nil")
	}
	var ce *config.ConfigError
	if !errors.As(err, &ce) {
		t.Fatalf("S601-SEC-002: expected *config.ConfigError, got %T: %v", err, err)
	}
	if ce.Code != "E-CFG-001" {
		t.Fatalf("S601-SEC-002: expected E-CFG-001, got %s", ce.Code)
	}

	// The message must contain no more than `cap` per-entry failure lines.
	// Every upstream_routers failure includes the literal "upstream_routers["
	// prefix, so counting occurrences is a lower bound on collected failures.
	msg := ce.Error()
	occurrences := strings.Count(msg, "upstream_routers[")
	if occurrences > cap {
		t.Errorf("S601-SEC-002: Validate emitted %d upstream_routers failure lines; "+
			"cap is %d — unbounded slice growth (CWE-400)", occurrences, cap)
	}

	// A truncation marker must be present so the operator sees WHY the list
	// was clipped (parity with io.LimitReader's E-CFG-005 truncation msg).
	if !strings.Contains(msg, "truncated") &&
		!strings.Contains(msg, "additional upstream_routers") {
		t.Errorf("S601-SEC-002: Validate silently truncated failure list — "+
			"must emit an explicit \"...truncated\" or \"...additional upstream_routers\" "+
			"marker so operators see the cap fired.  message: %q", msg)
	}
}

// TestUpstreamRoutersFailureCap_IsExportedAndPositive is a compile-time
// contract that the cap is a package-visible constant so downstream
// callers (main, tests, ops docs) can reference the authoritative value.
func TestUpstreamRoutersFailureCap_IsExportedAndPositive(t *testing.T) {
	t.Parallel()
	if config.UpstreamRoutersFailureCap <= 0 {
		t.Errorf("S601-SEC-002: UpstreamRoutersFailureCap must be positive; got %d",
			config.UpstreamRoutersFailureCap)
	}
}

// _ silences the "unused import" flag when go tooling doesn't yet see os
// used — os is imported for future test-fixture extensions that touch the
// filesystem.  Removing this line if os stops being needed is safe.
var _ = os.Stat
