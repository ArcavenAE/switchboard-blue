package config_test

// Test suite for internal/config.
//
// All tests are FAILING (Red Gate) because config.go stubs panic("not implemented").
// This file must compile; every test must fail before any implementation exists.
//
// Traceability:
//   BC-2.09.003 — Router Startup Fails Cleanly on Malformed Config
//   VP-028, VP-029 — Startup with any config error always exits non-zero;
//                    error message includes field name and fix suggestion.
//
// AC-003 NOTE FOR IMPLEMENTER:
//   TestRouterStartup_ExitsWithActionableError is tested here at the
//   internal/config layer — it verifies that Validate() returns an error
//   with code E-CFG-001 and that the error message names the field and
//   provides a fix suggestion. The cmd/switchboard wiring (calling
//   LoadFile + Validate, printing to stderr, and calling os.Exit(1)) is
//   a separate responsibility that must be implemented in cmd/switchboard/main.go
//   (or a dedicated startup.go). The ARCH-06 binding sequence is:
//       1. loadConfigFile(path)  →  Config struct
//       2. Config.Validate()     →  []ValidationError
//       3. if errors: printErrors(errors); os.Exit(1)
//       4. initLogger()
//       5. bindListenSocket()
//   This sequence must be added to the "access" subcommand handler in run().
//   A cmd-level integration test (e.g., in cmd/switchboard/) should additionally
//   exercise the binary exit-code and stderr output once that wiring exists.

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/arcavenae/switchboard/internal/config"
)

// ---- helpers ----------------------------------------------------------------

// requireError calls t.Fatal if err is nil.
func requireError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error but got nil")
	}
}

// requireNoError calls t.Fatal if err is non-nil.
func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// requireContains calls t.Fatalf if s does not contain substr (case-sensitive).
func requireContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Fatalf("expected %q to contain %q", s, substr)
	}
}

// requireECFG001 asserts that err wraps a *ConfigError with code E-CFG-001.
func requireECFG001(t *testing.T, err error) {
	t.Helper()
	var ce *config.ConfigError
	if !errors.As(err, &ce) {
		t.Fatalf("expected *config.ConfigError with E-CFG-001, got %T: %v", err, err)
	}
	if ce.Code != "E-CFG-001" {
		t.Fatalf("expected error code %q, got %q", "E-CFG-001", ce.Code)
	}
}

// ---- AC-001: tick_interval range validation ---------------------------------

// TestConfigValidate_RejectsOutOfRangeTickInterval verifies that Validate()
// returns a descriptive error identifying the field and value when tick_interval
// is outside [5ms, 50ms].
//
// Traces: BC-2.09.003 postcondition 1, AC-001, VP-028, VP-029.
func TestConfigValidate_RejectsOutOfRangeTickInterval(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		tickInterval time.Duration
		wantInMsg    []string // fragments that must appear in the error message
	}{
		{
			name:         "below_minimum_3ms",
			tickInterval: 3 * time.Millisecond,
			// BC-2.09.003 postcondition 2: error must name the field AND the value.
			// ARCH-06 example format: "field 'tick_interval' = 3ms is outside allowed range [5ms, 50ms]"
			wantInMsg: []string{"tick_interval", "3ms"},
		},
		{
			name:         "above_maximum_100ms",
			tickInterval: 100 * time.Millisecond,
			wantInMsg:    []string{"tick_interval", "100ms"},
		},
		{
			name:         "zero_tick_interval",
			tickInterval: 0,
			wantInMsg:    []string{"tick_interval"},
		},
		{
			name:         "negative_tick_interval",
			tickInterval: -1 * time.Millisecond,
			wantInMsg:    []string{"tick_interval"},
		},
		{
			name:         "just_below_minimum_4ms999us",
			tickInterval: 4*time.Millisecond + 999*time.Microsecond,
			wantInMsg:    []string{"tick_interval"},
		},
		{
			name:         "just_above_maximum_50ms001us",
			tickInterval: 50*time.Millisecond + 1*time.Microsecond,
			wantInMsg:    []string{"tick_interval"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := &config.Config{
				ListenAddr:   "0.0.0.0:9090",
				TickInterval: tc.tickInterval,
			}
			err := cfg.Validate()
			requireError(t, err)

			// VP-028, VP-029: error must include field name and value.
			msg := err.Error()
			for _, want := range tc.wantInMsg {
				requireContains(t, msg, want)
			}

			// Must carry E-CFG-001 (VP-028, VP-029: any config error → E-CFG-001).
			requireECFG001(t, err)
		})
	}
}

// ---- AC-002: all missing required fields reported at once -------------------

// TestConfigValidate_RejectsMissingRequiredFields verifies that Validate()
// returns an error listing ALL missing required fields in a single pass —
// not just the first one found.
//
// Traces: BC-2.09.003 postcondition 2, AC-002, VP-028, VP-029.
func TestConfigValidate_RejectsMissingRequiredFields(t *testing.T) {
	t.Parallel()

	t.Run("both_required_fields_missing", func(t *testing.T) {
		t.Parallel()

		// Zero-value Config: ListenAddr="" and TickInterval=0 are both missing/invalid.
		cfg := &config.Config{}
		err := cfg.Validate()
		requireError(t, err)

		// BC-2.09.003 postcondition 2: ALL missing fields must be reported together.
		// Both "listen_addr" and "tick_interval" must appear in the single error.
		msg := err.Error()
		requireContains(t, msg, "listen_addr")
		requireContains(t, msg, "tick_interval")

		requireECFG001(t, err)
	})

	t.Run("listen_addr_missing_tick_valid", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{
			ListenAddr:   "",
			TickInterval: 10 * time.Millisecond,
		}
		err := cfg.Validate()
		requireError(t, err)

		msg := err.Error()
		requireContains(t, msg, "listen_addr")
		requireECFG001(t, err)
	})

	t.Run("tick_interval_missing_listen_valid", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{
			ListenAddr:   "0.0.0.0:9090",
			TickInterval: 0, // zero → missing/invalid
		}
		err := cfg.Validate()
		requireError(t, err)

		msg := err.Error()
		requireContains(t, msg, "tick_interval")
		requireECFG001(t, err)
	})
}

// ---- AC-003: actionable error for startup config failure -------------------

// TestRouterStartup_ExitsWithActionableError verifies that when Config.Validate()
// fails, the returned error is E-CFG-001, its message names the specific field,
// and the message includes a fix suggestion (Suggestion field non-empty).
//
// This test exercises the contract at the internal/config layer.
//
// IMPLEMENTER NOTE: The cmd/switchboard "access" subcommand handler in run()
// must be updated to:
//
//  1. Accept a --config flag (or default path) pointing to the config file.
//  2. Call config.LoadFile(path).
//  3. Call cfg.Validate().
//  4. On error: fmt.Fprintf(stderr, "switchboard: %v\n", err); os.Exit(1).
//  5. Proceed only after validation passes.
//
// A separate test in cmd/switchboard/ should verify exit code 1 and stderr
// output by building the binary and running it with a bad config.
//
// Traces: BC-2.09.003 postcondition 3, AC-003, VP-028, VP-029.
func TestRouterStartup_ExitsWithActionableError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		cfg          *config.Config
		wantInMsg    []string
		wantNotInMsg []string
	}{
		{
			name: "invalid_tick_interval_has_suggestion",
			cfg: &config.Config{
				ListenAddr:   "0.0.0.0:9090",
				TickInterval: 3 * time.Millisecond,
			},
			// BC-2.09.003 postcondition 3 / ARCH-06:
			// "config error: <field>: <problem>. Fix: <suggestion>"
			wantInMsg: []string{
				"tick_interval",
				"3ms",
				// A fix suggestion must be present (non-empty Suggestion).
				// We check for a keyword that any reasonable suggestion would contain
				// given ADR-008 range [5ms, 50ms].
				"5ms",
			},
		},
		{
			name: "missing_listen_addr_has_suggestion",
			cfg: &config.Config{
				ListenAddr:   "",
				TickInterval: 10 * time.Millisecond,
			},
			wantInMsg: []string{"listen_addr"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.cfg.Validate()
			requireError(t, err)
			requireECFG001(t, err)

			msg := err.Error()
			for _, want := range tc.wantInMsg {
				requireContains(t, msg, want)
			}
			for _, notWant := range tc.wantNotInMsg {
				if strings.Contains(msg, notWant) {
					t.Errorf("error message must NOT contain %q; got: %q", notWant, msg)
				}
			}
		})
	}
}

// ---- AC-004: Validate called before any socket opens ----------------------

// TestConfigValidate_BeforeSocketOpen verifies the purity invariant:
// Validate() performs no I/O and opens no sockets. It must be callable
// before any network initialization (BC-2.09.003 invariant 1, ARCH-06
// binding sequence step 2).
//
// This test verifies the pure-core contract by confirming Validate returns
// an error on bad config without blocking (a socket-opening implementation
// would stall or fail differently) and succeeds on good config without
// side effects.
//
// Traces: BC-2.09.003 invariant 1, AC-004.
func TestConfigValidate_BeforeSocketOpen(t *testing.T) {
	t.Parallel()

	t.Run("invalid_config_returns_error_without_io", func(t *testing.T) {
		t.Parallel()

		// Validate must return promptly with an error — no socket bind, no I/O.
		cfg := &config.Config{
			ListenAddr:   "",
			TickInterval: 0,
		}
		// If Validate incorrectly attempts to open a socket on ListenAddr="",
		// it would either panic or return a different error type. We assert
		// only E-CFG-001 is returned (a pure validation error, not a network error).
		err := cfg.Validate()
		requireError(t, err)
		requireECFG001(t, err)
	})

	t.Run("valid_config_returns_nil_without_io", func(t *testing.T) {
		t.Parallel()

		// Valid config must return nil — no side effects, no sockets opened.
		cfg := &config.Config{
			ListenAddr:        "0.0.0.0:9090",
			TickInterval:      10 * time.Millisecond,
			DrainTimeout:      10 * time.Second,
			KeepaliveInterval: 1 * time.Second,
		}
		err := cfg.Validate()
		requireNoError(t, err)
	})

	t.Run("validate_does_not_mutate_config", func(t *testing.T) {
		t.Parallel()

		// Validate must not mutate the Config struct (pure function).
		// Config contains a []string field so we snapshot comparable scalar fields.
		cfg := &config.Config{
			ListenAddr:   "0.0.0.0:9090",
			TickInterval: 10 * time.Millisecond,
		}
		beforeAddr := cfg.ListenAddr
		beforeTick := cfg.TickInterval
		_ = cfg.Validate()
		if cfg.ListenAddr != beforeAddr {
			t.Errorf("Validate() must not mutate ListenAddr; before=%q after=%q", beforeAddr, cfg.ListenAddr)
		}
		if cfg.TickInterval != beforeTick {
			t.Errorf("Validate() must not mutate TickInterval; before=%v after=%v", beforeTick, cfg.TickInterval)
		}
	})
}

// ---- EC-001: config file missing entirely ----------------------------------

// TestLoadFile_MissingFile verifies that LoadFile returns E-CFG-004 with an
// actionable message that includes the expected path when the config file does
// not exist.
//
// v1.1 correction (SP-001): E-CFG-002 is no longer an acceptable code for this
// case. The canonical code per BC-2.09.003 EC-001 is E-CFG-004.
//
// Traces: BC-2.09.003 EC-001, S-6.01 EC-001 (v1.1).
func TestLoadFile_MissingFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	missingPath := filepath.Join(dir, "does-not-exist.yaml")

	_, err := config.LoadFile(missingPath)
	requireError(t, err)

	// BC-2.09.003 EC-001: "config file not found: <path>"; actionable message
	// must include the expected path so the operator knows where to place the file.
	msg := err.Error()
	requireContains(t, msg, missingPath)

	// v1.1 (SP-001): ONLY E-CFG-004 is acceptable. E-CFG-002 was the pre-revision
	// code and must no longer be returned for file-not-found.
	var ce *config.ConfigError
	if !errors.As(err, &ce) {
		t.Fatalf("expected *config.ConfigError, got %T: %v", err, err)
	}
	if ce.Code != "E-CFG-004" {
		t.Errorf("expected error code E-CFG-004 (BC-2.09.003 EC-001 v1.1), got %q", ce.Code)
	}
}

// ---- EC-002: config file present but empty ---------------------------------

// TestLoadFile_EmptyFile verifies that an empty config file causes Validate()
// to return E-CFG-001 listing ALL required fields as missing.
//
// Traces: BC-2.09.003 EC-002.
func TestLoadFile_EmptyFile(t *testing.T) {
	t.Parallel()

	cfg, err := config.LoadFile("testdata/empty.yaml")
	// LoadFile itself may succeed (empty YAML is valid syntax but zero-value struct).
	// The validation error emerges from Validate().
	if err != nil {
		// If LoadFile returns an error, it must be E-CFG-001 (validation) not a parse error.
		requireECFG001(t, err)
		requireContains(t, err.Error(), "listen_addr")
		requireContains(t, err.Error(), "tick_interval")
		return
	}

	// LoadFile succeeded; Validate must catch all missing required fields.
	valErr := cfg.Validate()
	requireError(t, valErr)
	requireECFG001(t, valErr)
	msg := valErr.Error()
	requireContains(t, msg, "listen_addr")
	requireContains(t, msg, "tick_interval")
}

// ---- EC-003: config file present but malformed YAML syntax -----------------

// TestLoadFile_MalformedYAML_ReturnsECFG005WithLineNumber verifies that LoadFile
// on a syntactically malformed YAML file returns an error with code E-CFG-005
// and that the error message includes a real numeric line number.
//
// BC-2.09.003 EC-003 / FM-010 canonical format:
//
//	"config parse error: invalid YAML at line N: <detail>"
//
// FIX 3 (L-3): the "at line N" fragment must contain a real digit — not the
// degraded fallback "at line ?:" which also satisfies "at line" alone.
// yaml.v3 reports line 2 for this fixture (the tab is detected while streaming
// after parsing the tick_interval line).
//
// Traces: BC-2.09.003 EC-003 (FM-010), S-6.01 EC-003 (v1.1).
func TestLoadFile_MalformedYAML_ReturnsECFG005WithLineNumber(t *testing.T) {
	t.Parallel()

	// Write a YAML file with a deliberate syntax error.
	// Tab characters are illegal as indentation in YAML.
	// yaml.v3 reports line 2 when it detects the tab character during streaming.
	malformedContent := "listen_addr: 0.0.0.0:9090\ntick_interval: 10ms\n\tfoo: bad-tab-indent\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "malformed.yaml")
	if err := os.WriteFile(path, []byte(malformedContent), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := config.LoadFile(path)
	requireError(t, err)

	// Must be E-CFG-005.
	var ce *config.ConfigError
	if !errors.As(err, &ce) {
		t.Fatalf("expected *config.ConfigError with E-CFG-005, got %T: %v", err, err)
	}
	if ce.Code != "E-CFG-005" {
		t.Errorf("expected error code E-CFG-005 (BC-2.09.003 EC-003), got %q", ce.Code)
	}

	msg := err.Error()
	if !strings.Contains(msg, "at line") {
		t.Errorf("E-CFG-005 message must contain 'at line' "+
			"(BC-2.09.003 EC-003 canonical format 'config parse error: invalid YAML at line N: <detail>'); "+
			"got: %q", msg)
	}
	// L-3 fix: require a real numeric line, not the degraded "at line ?:".
	// yaml.v3 reports line 2 for this fixture.
	if !strings.Contains(msg, "at line 2") {
		t.Errorf("E-CFG-005 message must contain real line number 'at line 2' "+
			"(not degraded 'at line ?'); got: %q", msg)
	}
}

// ---- EC-003: tick_interval exactly at lower boundary (5ms) -----------------

// TestLoadFile_TickInterval_ExactlyMinBoundary verifies that tick_interval=5ms
// is accepted as valid (inclusive lower bound per ADR-008).
//
// Traces: BC-2.09.003 EC-003, AC-001.
func TestLoadFile_TickInterval_ExactlyMinBoundary(t *testing.T) {
	t.Parallel()

	cfg, err := config.LoadFile("testdata/valid.yaml")
	requireNoError(t, err)

	// Override tick_interval to exactly TickIntervalMin.
	cfg.TickInterval = config.TickIntervalMin // 5ms
	err = cfg.Validate()
	requireNoError(t, err)
}

// ---- EC-004: tick_interval exactly at upper boundary (50ms) ----------------

// TestLoadFile_TickInterval_ExactlyMaxBoundary verifies that tick_interval=50ms
// is accepted as valid (inclusive upper bound per ADR-008).
//
// Traces: BC-2.09.003 EC-004, AC-001.
func TestLoadFile_TickInterval_ExactlyMaxBoundary(t *testing.T) {
	t.Parallel()

	cfg, err := config.LoadFile("testdata/valid.yaml")
	requireNoError(t, err)

	// Override tick_interval to exactly TickIntervalMax.
	cfg.TickInterval = config.TickIntervalMax // 50ms
	err = cfg.Validate()
	requireNoError(t, err)
}

// ---- EC-003 / EC-004 via fixture file loading ------------------------------

// TestLoadFile_OutOfRangeFixture loads out-of-range.yaml (tick_interval=3ms)
// and confirms the round-trip through LoadFile+Validate produces E-CFG-001.
//
// Traces: BC-2.09.003 postcondition 1, AC-001.
func TestLoadFile_OutOfRangeFixture(t *testing.T) {
	t.Parallel()

	cfg, err := config.LoadFile("testdata/out-of-range.yaml")
	requireNoError(t, err) // LoadFile does not validate; parse must succeed.

	valErr := cfg.Validate()
	requireError(t, valErr)
	requireECFG001(t, valErr)
	requireContains(t, valErr.Error(), "tick_interval")
}

// TestLoadFile_MissingFieldsFixture loads missing-fields.yaml (no listen_addr,
// no tick_interval) and confirms LoadFile+Validate reports both fields.
//
// Traces: BC-2.09.003 postcondition 2, AC-002, EC-002 variant.
func TestLoadFile_MissingFieldsFixture(t *testing.T) {
	t.Parallel()

	cfg, err := config.LoadFile("testdata/missing-fields.yaml")
	requireNoError(t, err)

	valErr := cfg.Validate()
	requireError(t, valErr)
	requireECFG001(t, valErr)
	msg := valErr.Error()
	requireContains(t, msg, "listen_addr")
	requireContains(t, msg, "tick_interval")
}

// TestLoadFile_ValidFixture verifies that the canonical valid.yaml fixture
// round-trips cleanly through LoadFile+Validate.
func TestLoadFile_ValidFixture(t *testing.T) {
	t.Parallel()

	cfg, err := config.LoadFile("testdata/valid.yaml")
	requireNoError(t, err)

	valErr := cfg.Validate()
	requireNoError(t, valErr)
}

// ---- VP-028 property: any config error always returns E-CFG-001 -----------

// TestVP028_AnyValidationErrorIsECFG001 is a property test verifying that
// every possible validation failure on Config.Validate() carries error code
// E-CFG-001 (VP-028: "Startup with any config error always exits non-zero").
//
// The test exercises the full Cartesian product of invalid field combinations
// to ensure no edge case silently returns nil or a different error code.
//
// Traces: VP-028.
func TestVP028_AnyValidationErrorIsECFG001(t *testing.T) {
	t.Parallel()

	// Invalid tick_interval values to test exhaustively.
	invalidTicks := []time.Duration{
		-1 * time.Millisecond,
		0,
		1 * time.Microsecond,
		4*time.Millisecond + 999*time.Microsecond,
		50*time.Millisecond + 1*time.Microsecond,
		100 * time.Millisecond,
		1 * time.Second,
	}

	invalidAddrs := []string{
		"",
		"   ",
	}

	// Case 1: invalid tick_interval with valid addr.
	for _, tick := range invalidTicks {
		tick := tick
		t.Run("invalid_tick_"+tick.String(), func(t *testing.T) {
			t.Parallel()
			cfg := &config.Config{
				ListenAddr:   "0.0.0.0:9090",
				TickInterval: tick,
			}
			err := cfg.Validate()
			requireError(t, err)
			requireECFG001(t, err)
		})
	}

	// Case 2: invalid addr with valid tick.
	for _, addr := range invalidAddrs {
		addr := addr
		t.Run("invalid_addr_"+addr, func(t *testing.T) {
			t.Parallel()
			cfg := &config.Config{
				ListenAddr:   addr,
				TickInterval: 10 * time.Millisecond,
			}
			err := cfg.Validate()
			requireError(t, err)
			requireECFG001(t, err)
		})
	}

	// Case 3: both invalid.
	for _, tick := range invalidTicks {
		for _, addr := range invalidAddrs {
			tick, addr := tick, addr
			t.Run("both_invalid_tick_"+tick.String()+"_addr_"+addr, func(t *testing.T) {
				t.Parallel()
				cfg := &config.Config{
					ListenAddr:   addr,
					TickInterval: tick,
				}
				err := cfg.Validate()
				requireError(t, err)
				requireECFG001(t, err)
			})
		}
	}
}

// ---- VP-029 property: error message always names field and fix suggestion --

// TestVP029_ErrorMessageNamesFieldAndSuggestion is a property test verifying
// that every validation error message names the specific field that failed
// and provides a non-empty fix suggestion (VP-029: "Error message includes
// field name and fix suggestion").
//
// Traces: VP-029.
func TestVP029_ErrorMessageNamesFieldAndSuggestion(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		cfg       *config.Config
		wantField string
	}{
		{
			name: "tick_interval_below_min",
			cfg: &config.Config{
				ListenAddr:   "0.0.0.0:9090",
				TickInterval: 1 * time.Millisecond,
			},
			wantField: "tick_interval",
		},
		{
			name: "tick_interval_above_max",
			cfg: &config.Config{
				ListenAddr:   "0.0.0.0:9090",
				TickInterval: 200 * time.Millisecond,
			},
			wantField: "tick_interval",
		},
		{
			name: "listen_addr_empty",
			cfg: &config.Config{
				ListenAddr:   "",
				TickInterval: 10 * time.Millisecond,
			},
			wantField: "listen_addr",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.cfg.Validate()
			requireError(t, err)

			// VP-029: message must name the field.
			msg := err.Error()
			requireContains(t, msg, tc.wantField)

			// VP-029: the ValidationError for this field must carry a non-empty Suggestion.
			// We verify this by unwrapping to *config.ConfigError and checking its Detail
			// contains actionable language. The implementer must populate ValidationError.Suggestion
			// per BC-2.09.003 postcondition 3 / ARCH-06 example format.
			var ce *config.ConfigError
			if !errors.As(err, &ce) {
				t.Fatalf("expected *config.ConfigError, got %T", err)
			}
			// The Detail or the combined error string must contain "Fix" or a
			// suggestion verb, per ARCH-06: "Suggestion: set to 10ms for interactive sessions."
			if !strings.Contains(msg, "Fix") && !strings.Contains(msg, "fix") &&
				!strings.Contains(msg, "set") && !strings.Contains(msg, "add") &&
				!strings.Contains(msg, "use") {
				t.Errorf("VP-029: error message must contain a fix suggestion; got: %q", msg)
			}
		})
	}
}

// ---- ValidationError type contract -----------------------------------------

// TestValidationError_ImplementsError verifies that *ValidationError implements
// the error interface and that its Error() method is callable (not panicking
// with an unexpected nil dereference).
func TestValidationError_ImplementsError(t *testing.T) {
	t.Parallel()

	// We exercise Error() indirectly through Validate() — the stub will panic,
	// which is the expected Red Gate behavior. This test documents that
	// ValidationError.Error() must produce a non-empty string in production.
	cfg := &config.Config{
		ListenAddr:   "",
		TickInterval: 0,
	}
	err := cfg.Validate()
	// In production: err must be non-nil and err.Error() must be non-empty.
	// Under stubs: this panics → Red Gate confirmed.
	if err != nil && err.Error() == "" {
		t.Errorf("error.Error() must return a non-empty string")
	}
}

// ---- M-1 (CWE-400): config file size guard ----------------------------------

// TestLoadFile_FileTooLarge verifies that LoadFile rejects a config file that
// exceeds maxConfigFileSize with E-CFG-005 (CWE-400 defence).
//
// Traces: M-1 security finding, BC-2.09.003 EC-003 family.
func TestLoadFile_FileTooLarge(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "huge.yaml")
	// Write slightly more than 1 MiB (maxConfigFileSize = 1 << 20).
	huge := make([]byte, (1<<20)+1)
	if err := os.WriteFile(path, huge, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := config.LoadFile(path)
	requireError(t, err)

	var ce *config.ConfigError
	if !errors.As(err, &ce) {
		t.Fatalf("expected *config.ConfigError, got %T: %v", err, err)
	}
	if ce.Code != "E-CFG-005" {
		t.Errorf("expected E-CFG-005 for oversized file, got %q", ce.Code)
	}
	requireContains(t, err.Error(), "too large")
}

// TestLoadFile_RejectsFileExceedingSizeCapViaBoundedRead pins the bounded-read
// rejection path: even if os.Stat were bypassed, io.LimitReader enforces the
// cap so a file of maxConfigFileSize+1 bytes is always rejected (F-SEC-L1,
// CWE-400 / CWE-367).
//
// It also verifies the boundary: a file of exactly maxConfigFileSize bytes that
// contains valid YAML is accepted (the cap is inclusive).
//
// Traces: F-SEC-L1, CWE-400, CWE-367.
func TestLoadFile_RejectsFileExceedingSizeCapViaBoundedRead(t *testing.T) {
	t.Parallel()

	t.Run("exactly_max_plus_one_rejected", func(t *testing.T) {
		t.Helper()
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "over-cap.yaml")
		// maxConfigFileSize+1 bytes — must be rejected regardless of stat.
		over := make([]byte, (1<<20)+1)
		if err := os.WriteFile(path, over, 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		_, err := config.LoadFile(path)
		if err == nil {
			t.Fatal("expected an error for over-cap file, got nil")
		}

		var ce *config.ConfigError
		if !errors.As(err, &ce) {
			t.Fatalf("expected *config.ConfigError, got %T: %v", err, err)
		}
		if ce.Code != "E-CFG-005" {
			t.Errorf("expected E-CFG-005 for over-cap file, got %q", ce.Code)
		}
		requireContains(t, err.Error(), "too large")
	})

	t.Run("exactly_max_valid_yaml_accepted", func(t *testing.T) {
		t.Helper()
		t.Parallel()

		// Build a file of exactly maxConfigFileSize bytes containing valid YAML at
		// the front and padded with '#' comments (valid YAML) to fill the rest.
		// This confirms the cap is inclusive: len(data) == maxConfigFileSize is allowed.
		const header = "listen_addr: 0.0.0.0:9090\ntick_interval: 10ms\n# "
		const maxSize = 1 << 20
		pad := make([]byte, maxSize-len(header))
		for i := range pad {
			pad[i] = 'x' // 'x' characters after the "# " comment marker — valid YAML
		}
		content := append([]byte(header), pad...)
		if len(content) != maxSize {
			t.Fatalf("test setup error: content length %d != maxConfigFileSize %d", len(content), maxSize)
		}

		dir := t.TempDir()
		path := filepath.Join(dir, "at-cap.yaml")
		if err := os.WriteFile(path, content, 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		_, err := config.LoadFile(path)
		if err != nil {
			t.Errorf("expected no error for file at maxConfigFileSize boundary, got: %v", err)
		}
	})
}

// TestLoadFile_NonRegularFile verifies that LoadFile rejects a non-regular file
// (e.g. a directory) with E-CFG-005 (fail-closed, CWE-400 defence).
//
// Traces: M-1 security finding, BC-2.09.003 EC-003 family.
func TestLoadFile_NonRegularFile(t *testing.T) {
	t.Parallel()

	// Pass a directory path — directories are not regular files.
	dir := t.TempDir()

	_, err := config.LoadFile(dir)
	requireError(t, err)

	var ce *config.ConfigError
	if !errors.As(err, &ce) {
		t.Fatalf("expected *config.ConfigError, got %T: %v", err, err)
	}
	if ce.Code != "E-CFG-005" {
		t.Errorf("expected E-CFG-005 for non-regular file, got %q", ce.Code)
	}
	requireContains(t, err.Error(), "not a regular file")
}

// ---- L-1 (CWE-20): strict YAML decoding rejects unknown keys ----------------

// TestLoadFile_UnknownKey_ReturnsECFG005 verifies that LoadFile rejects a config
// file containing an unknown (typo'd) key with E-CFG-005 (L-1 finding, CWE-20).
// This prevents silent misconfiguration when an operator misspells an optional key.
//
// Traces: L-1 security finding, BC-2.09.003 FM-010.
func TestLoadFile_UnknownKey_ReturnsECFG005(t *testing.T) {
	t.Parallel()

	// keepalive_intervall is a misspelling of keepalive_interval.
	content := "listen_addr: 0.0.0.0:9090\ntick_interval: 10ms\nkeepalive_intervall: 1s\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "unknown-key.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := config.LoadFile(path)
	requireError(t, err)

	var ce *config.ConfigError
	if !errors.As(err, &ce) {
		t.Fatalf("expected *config.ConfigError with E-CFG-005, got %T: %v", err, err)
	}
	if ce.Code != "E-CFG-005" {
		t.Errorf("expected E-CFG-005 for unknown key, got %q", ce.Code)
	}
	// Error must mention the unknown key name so the operator can identify the typo.
	requireContains(t, err.Error(), "keepalive_intervall")
}

// TestLoadFile_ValidFixture_StrictDecode verifies that the canonical valid.yaml
// fixture is accepted cleanly after switching to strict (KnownFields) decoding —
// no false rejection of known keys.
//
// Traces: L-1 security finding regression guard.
func TestLoadFile_ValidFixture_StrictDecode(t *testing.T) {
	t.Parallel()

	cfg, err := config.LoadFile("testdata/valid.yaml")
	requireNoError(t, err)

	// valid.yaml must still decode and validate cleanly.
	valErr := cfg.Validate()
	requireNoError(t, valErr)
}

// ---- AC-005 / PC-5: listen_addr host:port format validation -----------------

// TestConfigValidate_RejectsInvalidListenAddrFormat verifies that Validate()
// rejects a listen_addr that is not a valid host:port with E-CFG-002.
//
// The error message must match the canonical E-CFG-002 format:
//
//	"config error: listen_addr: '<value>' is not a valid host:port. Fix: use '<ip>:<port>' format, e.g. '0.0.0.0:9090'"
//
// Specifically: the error must (a) carry E-CFG-001 (Validate wraps field errors in
// E-CFG-001), (b) contain the prefix "config error: listen_addr:", (c) contain the
// offending value verbatim, and (d) contain "is not a valid host:port".
//
// A valid host:port (e.g. "0.0.0.0:9090") with a valid tick_interval must pass.
//
// Traces: BC-2.09.003 postcondition 5, AC-005, EC-006, EC-007.
func TestConfigValidate_RejectsInvalidListenAddrFormat(t *testing.T) {
	t.Parallel()

	invalidCases := []struct {
		name       string
		listenAddr string
		wantInMsg  []string // fragments that must appear in error message
	}{
		{
			// EC-006: missing port
			name:       "missing_port",
			listenAddr: "0.0.0.0",
			wantInMsg:  []string{"config error: listen_addr:", "'0.0.0.0'", "is not a valid host:port"},
		},
		{
			// EC-007: non-numeric port
			name:       "non_numeric_port",
			listenAddr: "0.0.0.0:notaport",
			wantInMsg:  []string{"config error: listen_addr:", "'0.0.0.0:notaport'", "is not a valid host:port"},
		},
		{
			// Bare hostname with no port
			name:       "hostname_no_port",
			listenAddr: "not-a-host-port",
			wantInMsg:  []string{"config error: listen_addr:", "'not-a-host-port'", "is not a valid host:port"},
		},
		{
			// Empty string is the "missing required field" case — already covered by
			// AC-001/AC-002 but included here to confirm it is caught under the same
			// E-CFG-001 umbrella (the exact sub-code / message varies by implementation).
			name:       "empty_string",
			listenAddr: "",
			wantInMsg:  []string{"listen_addr"},
		},
	}

	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := &config.Config{
				ListenAddr:   tc.listenAddr,
				TickInterval: 10 * time.Millisecond,
			}
			err := cfg.Validate()
			requireError(t, err)

			// Must carry E-CFG-001 (Validate wraps field errors in E-CFG-001 envelope).
			requireECFG001(t, err)

			msg := err.Error()
			for _, want := range tc.wantInMsg {
				requireContains(t, msg, want)
			}
		})
	}

	// Valid host:port must pass validation (regression guard).
	t.Run("valid_host_port_passes", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{
			ListenAddr:        "0.0.0.0:9090",
			TickInterval:      10 * time.Millisecond,
			DrainTimeout:      10 * time.Second,
			KeepaliveInterval: 1 * time.Second,
		}
		requireNoError(t, cfg.Validate())
	})

	t.Run("valid_loopback_port_passes", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{
			ListenAddr:        "127.0.0.1:8080",
			TickInterval:      10 * time.Millisecond,
			DrainTimeout:      10 * time.Second,
			KeepaliveInterval: 1 * time.Second,
		}
		requireNoError(t, cfg.Validate())
	})
}

// ---- AC-006 / PC-6: upstream_routers[N].addr host:port validation -----------

// TestConfigValidate_RejectsInvalidUpstreamRouterAddr verifies that Validate()
// rejects an upstream_routers entry whose addr is not a valid host:port with
// E-CFG-003, naming the 0-based index N and the offending value.
//
// The error message must match the canonical E-CFG-003 format:
//
//	"config error: upstream_routers[<N>].addr: '<value>' is not a valid host:port. Fix: use '<ip>:<port>' format, e.g. '10.0.0.1:9090'"
//
// Traces: BC-2.09.003 postcondition 6, AC-006, EC-008.
func TestConfigValidate_RejectsInvalidUpstreamRouterAddr(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name            string
		upstreamRouters []config.UpstreamRouter
		wantInMsg       []string // fragments that must appear in error message
	}{
		{
			// Single invalid entry at index 0 (canonical test vector from BC).
			name: "single_invalid_at_index_0",
			upstreamRouters: []config.UpstreamRouter{
				{Addr: "notvalid"},
			},
			wantInMsg: []string{
				"config error: upstream_routers[0].addr:",
				"'notvalid'",
				"is not a valid host:port",
			},
		},
		{
			// EC-008: first valid, second invalid — must name index 1.
			name: "first_valid_second_invalid_at_index_1",
			upstreamRouters: []config.UpstreamRouter{
				{Addr: "10.0.0.1:9090"},
				{Addr: "badaddr"},
			},
			// Must name index 1 (0-based).
			wantInMsg: []string{
				"upstream_routers[1].addr",
				"'badaddr'",
				"is not a valid host:port",
			},
		},
		{
			// Missing port on an upstream addr.
			name: "missing_port_on_upstream",
			upstreamRouters: []config.UpstreamRouter{
				{Addr: "10.0.0.1"},
			},
			wantInMsg: []string{
				"upstream_routers[0].addr",
				"'10.0.0.1'",
				"is not a valid host:port",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := &config.Config{
				ListenAddr:      "0.0.0.0:9090",
				TickInterval:    10 * time.Millisecond,
				UpstreamRouters: tc.upstreamRouters,
			}
			err := cfg.Validate()
			requireError(t, err)

			// Must carry E-CFG-001 (Validate wraps all field errors in E-CFG-001).
			requireECFG001(t, err)

			msg := err.Error()
			for _, want := range tc.wantInMsg {
				requireContains(t, msg, want)
			}
		})
	}

	// Valid upstream addr must pass (regression guard).
	t.Run("valid_upstream_addr_passes", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{
			ListenAddr:   "0.0.0.0:9090",
			TickInterval: 10 * time.Millisecond,
			UpstreamRouters: []config.UpstreamRouter{
				{Addr: "10.0.0.1:9090"},
				{Addr: "10.0.0.2:9091"},
			},
			DrainTimeout:      10 * time.Second,
			KeepaliveInterval: 1 * time.Second,
		}
		requireNoError(t, cfg.Validate())
	})
}

// ---- AC-007 / PC-7: drain_timeout — negative rejected, zero/absent accepted --

// TestConfigValidate_RejectsNegativeDrainTimeout verifies that Validate()
// rejects drain_timeout ONLY when negative (E-CFG-006), and ACCEPTS zero or
// absent (Go yaml / time.Duration zero-value semantics: absent == zero).
//
// The error message must match the canonical E-CFG-006 format (BC-2.09.003 v1.4):
//
//	"config error: drain_timeout: must not be negative; got '<value>'. Fix: remove the field to use the daemon default (10s), or set to a positive duration, e.g. '10s'"
//
// Red Gate note: the CURRENT config.go rejects zero with "must be > 0".
// These tests are written for the NEW v1.4 contract:
//   - Zero-accepted cases FAIL against current code (it rejects zero).
//   - "must not be negative" assertions FAIL against current code (it emits "must be > 0").
//
// Traces: BC-2.09.003 postcondition 7 (v1.4), AC-007, EC-009, EC-010, VP-028, VP-029.
func TestConfigValidate_RejectsNegativeDrainTimeout(t *testing.T) {
	t.Parallel()

	// Negative values must be rejected with the canonical E-CFG-006 message.
	negativeCases := []struct {
		name         string
		drainTimeout time.Duration
		wantInMsg    []string // fragments that must appear in error message
	}{
		{
			// EC-010: drain_timeout = -5s must be rejected with E-CFG-006.
			name:         "negative_5s",
			drainTimeout: -5 * time.Second,
			wantInMsg: []string{
				"config error: drain_timeout:",
				"must not be negative",
				"-5s",
			},
		},
		{
			// Very small negative value (-1ns) must also be rejected.
			name:         "negative_1ns",
			drainTimeout: -1,
			wantInMsg: []string{
				"config error: drain_timeout:",
				"must not be negative",
			},
		},
	}

	for _, tc := range negativeCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Set KeepaliveInterval to a positive value so the only failing field is
			// drain_timeout, and the message check is not obscured by a second error.
			cfg := &config.Config{
				ListenAddr:        "0.0.0.0:9090",
				TickInterval:      10 * time.Millisecond,
				DrainTimeout:      tc.drainTimeout,
				KeepaliveInterval: 1 * time.Second,
			}
			err := cfg.Validate()
			requireError(t, err)

			// Must carry E-CFG-001 (Validate wraps all field errors in E-CFG-001).
			requireECFG001(t, err)

			msg := err.Error()
			for _, want := range tc.wantInMsg {
				requireContains(t, msg, want)
			}
		})
	}

	// EC-009: drain_timeout = 0s (or absent — Go yaml / time.Duration cannot
	// distinguish absent from explicit-zero) must be ACCEPTED. Validate() returns nil.
	// Red Gate: current code rejects zero, so this subtest FAILS against current config.go.
	t.Run("zero_drain_timeout_accepted", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{
			ListenAddr:        "0.0.0.0:9090",
			TickInterval:      10 * time.Millisecond,
			DrainTimeout:      0,               // zero == absent per Go yaml / time.Duration semantics
			KeepaliveInterval: 1 * time.Second, // positive — not under test
		}
		requireNoError(t, cfg.Validate())
	})

	// Positive drain_timeout must pass (regression guard).
	t.Run("positive_drain_timeout_passes", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{
			ListenAddr:        "0.0.0.0:9090",
			TickInterval:      10 * time.Millisecond,
			DrainTimeout:      10 * time.Second,
			KeepaliveInterval: 1 * time.Second,
		}
		requireNoError(t, cfg.Validate())
	})
}

// ---- AC-008 / PC-8: keepalive_interval — negative rejected, zero/absent accepted --

// TestConfigValidate_RejectsNegativeKeepaliveInterval verifies that Validate()
// rejects keepalive_interval ONLY when negative (E-CFG-007), and ACCEPTS zero or
// absent (Go yaml / time.Duration zero-value semantics: absent == zero).
//
// The error message must match the canonical E-CFG-007 format (BC-2.09.003 v1.4):
//
//	"config error: keepalive_interval: must not be negative; got '<value>'. Fix: remove the field to use the daemon default (1s), or set to a positive duration, e.g. '1s'"
//
// Red Gate note: the CURRENT config.go rejects zero with "must be > 0".
// These tests are written for the NEW v1.4 contract:
//   - Zero-accepted cases FAIL against current code (it rejects zero).
//   - "must not be negative" assertions FAIL against current code (it emits "must be > 0").
//
// Traces: BC-2.09.003 postcondition 8 (v1.4), AC-008, EC-011, EC-012, VP-028, VP-029.
func TestConfigValidate_RejectsNegativeKeepaliveInterval(t *testing.T) {
	t.Parallel()

	// Negative values must be rejected with the canonical E-CFG-007 message.
	negativeCases := []struct {
		name              string
		keepaliveInterval time.Duration
		wantInMsg         []string
	}{
		{
			// EC-012: keepalive_interval = -1s must be rejected with E-CFG-007.
			name:              "negative_1s",
			keepaliveInterval: -1 * time.Second,
			wantInMsg: []string{
				"config error: keepalive_interval:",
				"must not be negative",
				"-1s",
			},
		},
		{
			// Very small negative value (-1ns) must also be rejected.
			name:              "negative_1ns",
			keepaliveInterval: -1,
			wantInMsg: []string{
				"config error: keepalive_interval:",
				"must not be negative",
			},
		},
	}

	for _, tc := range negativeCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Set DrainTimeout to a positive value so the only failing field is
			// keepalive_interval, and the message check is not obscured by a second error.
			cfg := &config.Config{
				ListenAddr:        "0.0.0.0:9090",
				TickInterval:      10 * time.Millisecond,
				DrainTimeout:      10 * time.Second,
				KeepaliveInterval: tc.keepaliveInterval,
			}
			err := cfg.Validate()
			requireError(t, err)

			// Must carry E-CFG-001 (Validate wraps all field errors in E-CFG-001).
			requireECFG001(t, err)

			msg := err.Error()
			for _, want := range tc.wantInMsg {
				requireContains(t, msg, want)
			}
		})
	}

	// EC-011: keepalive_interval = 0s (or absent) must be ACCEPTED. Validate() returns nil.
	// Red Gate: current code rejects zero, so this subtest FAILS against current config.go.
	t.Run("zero_keepalive_interval_accepted", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{
			ListenAddr:        "0.0.0.0:9090",
			TickInterval:      10 * time.Millisecond,
			DrainTimeout:      10 * time.Second, // positive — not under test
			KeepaliveInterval: 0,                // zero == absent per Go yaml / time.Duration semantics
		}
		requireNoError(t, cfg.Validate())
	})

	// Positive keepalive_interval must pass (regression guard).
	t.Run("positive_keepalive_passes", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{
			ListenAddr:        "0.0.0.0:9090",
			TickInterval:      10 * time.Millisecond,
			DrainTimeout:      10 * time.Second,
			KeepaliveInterval: 1 * time.Second,
		}
		requireNoError(t, cfg.Validate())
	})
}

// ---- EC-009 / EC-011: zero/absent drain_timeout and keepalive_interval accepted --

// TestConfigValidate_AcceptsZeroOrAbsentDurationFields verifies that a fully valid
// config with drain_timeout and keepalive_interval both zero (== absent per Go yaml
// / time.Duration zero-value semantics) passes Validate() with no error.
//
// This is the EC-009 + EC-011 happy path: the daemon will apply the documented
// defaults (10s / 1s) at startup (S-7.04). Validate() must not reject these fields
// when zero.
//
// Red Gate: current config.go rejects zero for both fields ("must be > 0").
// These tests FAIL against current code — they pass only after the v1.4 fix lands.
//
// Traces: BC-2.09.003 PC-7 v1.4, PC-8 v1.4, EC-009, EC-011, AC-007, AC-008.
func TestConfigValidate_AcceptsZeroOrAbsentDurationFields(t *testing.T) {
	t.Parallel()

	t.Run("both_zero_drain_and_keepalive_accepted", func(t *testing.T) {
		t.Parallel()

		// Both optional duration fields absent/zero: Validate() must return nil.
		cfg := &config.Config{
			ListenAddr:        "0.0.0.0:9090",
			TickInterval:      10 * time.Millisecond,
			DrainTimeout:      0, // absent/zero — daemon default 10s applied by S-7.04
			KeepaliveInterval: 0, // absent/zero — daemon default 1s applied by S-7.04
		}
		requireNoError(t, cfg.Validate())
	})

	t.Run("zero_drain_positive_keepalive_accepted", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{
			ListenAddr:        "0.0.0.0:9090",
			TickInterval:      10 * time.Millisecond,
			DrainTimeout:      0,               // absent/zero — accepted
			KeepaliveInterval: 1 * time.Second, // explicit positive — accepted
		}
		requireNoError(t, cfg.Validate())
	})

	t.Run("positive_drain_zero_keepalive_accepted", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{
			ListenAddr:        "0.0.0.0:9090",
			TickInterval:      10 * time.Millisecond,
			DrainTimeout:      10 * time.Second, // explicit positive — accepted
			KeepaliveInterval: 0,                // absent/zero — accepted
		}
		requireNoError(t, cfg.Validate())
	})
}

// ---- F-SEC-002 (CWE-117): log/error injection via control chars in addr ------

// TestConfigValidate_RejectsControlCharsInAddrError verifies that Validate()
// does NOT propagate raw control characters from attacker-controlled field
// values into the returned error message (F-SEC-002, CWE-117).
//
// Current code (config.go line 164-167 and 191-194) interpolates the raw
// value verbatim with fmt.Sprintf("%s", c.ListenAddr). A listen_addr or
// upstream_routers[N].addr that embeds a newline, carriage return, or ANSI
// escape sequence therefore produces a multi-line error string that an
// attacker could use to forge log lines or corrupt terminal output.
//
// Security assertions:
//  1. The error message must still identify a listen_addr / upstream_routers
//     validation failure (the value is not a valid host:port — both because
//     embedded control chars make the value structurally invalid AND because
//     even without them the base value is not host:port format).
//  2. The raw newline (\n), carriage return (\r), and ESC (\x1b) characters
//     from the attacker-supplied value MUST NOT appear in the error message.
//     The implementer must neutralize the value before interpolation (e.g.,
//     strconv.Quote or a strip function).
//
// Positive regression case: a plain invalid value with no control characters
// (e.g. "not-a-host-port") MUST still appear verbatim (or quoted) in the
// error message so the operator can identify what they mistyped. This pins
// the BC 'value' format for the common case while neutralizing control chars.
//
// RED Gate: current code interpolates the raw value verbatim — the \n
// character WILL appear in the error string, causing the "must not contain
// raw newline" assertion to fail.
//
// Traces: BC-2.09.003 PC-5 / PC-6, F-SEC-002, CWE-117.
func TestConfigValidate_RejectsControlCharsInAddrError(t *testing.T) {
	t.Parallel()

	// ---- listen_addr injection cases ----------------------------------------

	type addrCase struct {
		name           string
		addr           string
		wantFieldInMsg string   // must appear in error
		mustNotContain []string // raw control chars that must NOT appear in error
	}

	listenCases := []addrCase{
		{
			// Primary injection: embedded newline after a plausible address.
			// An attacker sets listen_addr to "0.0.0.0:9090\nswitchboard: FORGED LOG LINE"
			// and hopes the error message leaks the newline, forging a second log line.
			name:           "listen_addr_embedded_newline",
			addr:           "0.0.0.0:9090\nswitchboard: FORGED LOG LINE",
			wantFieldInMsg: "listen_addr",
			mustNotContain: []string{"\n"},
		},
		{
			// Bare carriage return.
			name:           "listen_addr_carriage_return",
			addr:           "0.0.0.0:9090\r",
			wantFieldInMsg: "listen_addr",
			mustNotContain: []string{"\r"},
		},
		{
			// ANSI escape sequence (terminal colour attack).
			name:           "listen_addr_ansi_escape",
			addr:           "0.0.0.0:9090\x1b[31mRED",
			wantFieldInMsg: "listen_addr",
			mustNotContain: []string{"\x1b"},
		},
	}

	for _, tc := range listenCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := &config.Config{
				ListenAddr:   tc.addr,
				TickInterval: 10 * time.Millisecond,
			}
			err := cfg.Validate()
			requireError(t, err)
			requireECFG001(t, err)

			msg := err.Error()

			// Assertion 1: error still identifies the field.
			requireContains(t, msg, tc.wantFieldInMsg)

			// Assertion 2 (security): raw control chars must NOT appear in message.
			for _, forbidden := range tc.mustNotContain {
				if strings.Contains(msg, forbidden) {
					t.Errorf("error message contains raw control char %q (CWE-117 / F-SEC-002); "+
						"value must be sanitized before interpolation; got: %q",
						forbidden, msg)
				}
			}
		})
	}

	// ---- upstream_routers injection cases ------------------------------------

	upstreamCases := []addrCase{
		{
			// Embedded newline in upstream_routers[0].addr.
			name:           "upstream_routers_addr_embedded_newline",
			addr:           "10.0.0.1:9090\nswitchboard: FORGED",
			wantFieldInMsg: "upstream_routers",
			mustNotContain: []string{"\n"},
		},
		{
			// Carriage return in upstream_routers[0].addr.
			name:           "upstream_routers_addr_carriage_return",
			addr:           "10.0.0.1:9090\r",
			wantFieldInMsg: "upstream_routers",
			mustNotContain: []string{"\r"},
		},
	}

	for _, tc := range upstreamCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := &config.Config{
				ListenAddr:   "0.0.0.0:9090",
				TickInterval: 10 * time.Millisecond,
				UpstreamRouters: []config.UpstreamRouter{
					{Addr: tc.addr},
				},
			}
			err := cfg.Validate()
			requireError(t, err)
			requireECFG001(t, err)

			msg := err.Error()

			// Assertion 1: error still identifies the field.
			requireContains(t, msg, tc.wantFieldInMsg)

			// Assertion 2 (security): raw control chars must NOT appear.
			for _, forbidden := range tc.mustNotContain {
				if strings.Contains(msg, forbidden) {
					t.Errorf("error message contains raw control char %q (CWE-117 / F-SEC-002); "+
						"upstream addr value must be sanitized; got: %q",
						forbidden, msg)
				}
			}
		})
	}

	// ---- Positive regression: plain invalid value must appear in message -----
	//
	// When the value has no control characters, the error message must still
	// contain the offending value (or a quoted form) so the operator knows what
	// they mistyped. This guards against over-eager sanitization that swallows
	// the value entirely.
	t.Run("plain_invalid_value_appears_in_message", func(t *testing.T) {
		t.Parallel()

		const plainInvalid = "not-a-host-port"
		cfg := &config.Config{
			ListenAddr:   plainInvalid,
			TickInterval: 10 * time.Millisecond,
		}
		err := cfg.Validate()
		requireError(t, err)
		requireECFG001(t, err)

		msg := err.Error()
		requireContains(t, msg, "listen_addr")
		// The offending value (or its quoted form) must appear so the operator
		// can identify the problem. strconv.Quote("not-a-host-port") ==
		// `"not-a-host-port"` — either the raw value or the quoted form is fine;
		// both contain the key substring.
		if !strings.Contains(msg, plainInvalid) && !strings.Contains(msg, `"not-a-host-port"`) {
			t.Errorf("error message does not contain the offending value %q (or its quoted form); "+
				"sanitization must not swallow the value entirely; got: %q", plainInvalid, msg)
		}
	})
}

// ---- AC-002 (Inv-4) amendment: exhaustive multi-field error reporting -------

// TestConfigValidate_ReportsAllErrorsTogether verifies that when multiple fields
// are simultaneously invalid, Validate() collects ALL errors and reports them in
// a single E-CFG-001 error — not just the first one encountered.
//
// This tests the amended AC-002 requirement (S-6.01 v1.5 / BC-2.09.003 v1.4 Inv-4):
// exhaustive reporting must cover E-CFG-002 (bad listen_addr), E-CFG-003 (bad
// upstream addr), E-CFG-006 (negative drain_timeout), and E-CFG-007
// (negative keepalive_interval) together.
//
// BC-2.09.003 v1.4 note: drain_timeout and keepalive_interval are triggered ONLY
// by NEGATIVE values. Zero is now accepted (daemon default). The multi-field cases
// below use -5s / -1s (not zero) so they still trigger E-CFG-006 / E-CFG-007.
//
// Traces: BC-2.09.003 invariant 4, AC-002 (amendment), S-6.01 v1.5.
func TestConfigValidate_ReportsAllErrorsTogether(t *testing.T) {
	t.Parallel()

	t.Run("bad_listen_addr_and_negative_drain_timeout_together", func(t *testing.T) {
		t.Parallel()

		// Config with two distinct invalid fields: bad listen_addr (E-CFG-002) and
		// negative drain_timeout (E-CFG-006). Both must appear in a single error.
		cfg := &config.Config{
			ListenAddr:   "no-port-here", // invalid host:port → E-CFG-002
			TickInterval: 10 * time.Millisecond,
			DrainTimeout: -5 * time.Second, // negative → E-CFG-006 (zero would be valid per v1.4)
		}
		err := cfg.Validate()
		requireError(t, err)
		requireECFG001(t, err)

		// Inv-4: BOTH offending fields must appear in the single combined error.
		msg := err.Error()
		requireContains(t, msg, "listen_addr")
		requireContains(t, msg, "drain_timeout")
	})

	t.Run("bad_upstream_addr_and_negative_keepalive_together", func(t *testing.T) {
		t.Parallel()

		// keepalive_interval: -1s is negative → E-CFG-007.
		// (Previously this case used zero, which is now accepted per BC-2.09.003 v1.4.)
		cfg := &config.Config{
			ListenAddr:   "0.0.0.0:9090",
			TickInterval: 10 * time.Millisecond,
			UpstreamRouters: []config.UpstreamRouter{
				{Addr: "notvalid"}, // invalid → E-CFG-003 at index 0
			},
			KeepaliveInterval: -1 * time.Second, // negative → E-CFG-007
		}
		err := cfg.Validate()
		requireError(t, err)
		requireECFG001(t, err)

		msg := err.Error()
		// Both upstream_routers[0].addr and keepalive_interval must appear.
		requireContains(t, msg, "upstream_routers[0].addr")
		requireContains(t, msg, "keepalive_interval")
	})

	t.Run("three_invalid_fields_all_reported", func(t *testing.T) {
		t.Parallel()

		// Maximum coverage: bad listen_addr + negative drain_timeout + bad upstream addr.
		// All three must appear in a single E-CFG-001 error (Inv-4: exhaustive reporting).
		// drain_timeout is -5s (negative) so E-CFG-006 is triggered; zero would be valid.
		cfg := &config.Config{
			ListenAddr:   "0.0.0.0", // missing port → E-CFG-002
			TickInterval: 10 * time.Millisecond,
			UpstreamRouters: []config.UpstreamRouter{
				{Addr: "badupstream"}, // invalid host:port → E-CFG-003 at [0]
			},
			DrainTimeout: -5 * time.Second, // negative → E-CFG-006 (zero would be valid per v1.4)
		}
		err := cfg.Validate()
		requireError(t, err)
		requireECFG001(t, err)

		msg := err.Error()
		// All three offending fields must appear in the single aggregated error.
		requireContains(t, msg, "listen_addr")
		requireContains(t, msg, "upstream_routers[0].addr")
		requireContains(t, msg, "drain_timeout")
	})
}

// ---- F-SEC-005 (CWE-117): control chars in yaml parse-error path (unknown key / parse detail) --

// TestLoadFile_StripsControlCharsFromParseError verifies that LoadFile does NOT
// propagate raw control characters from attacker-controlled YAML key names into
// the returned E-CFG-005 error message (F-SEC-005, CWE-117).
//
// Background: config.go yamlParseDetail (~line 349) builds the E-CFG-005 detail
// by embedding raw := err.Error() with only strings.TrimSpace — NO control-char
// stripping. With dec.KnownFields(true), an unknown YAML key appears in yaml.v3's
// error message as "field <KEY> not found in type config.Config". If the YAML key
// carries control runes (C0, DEL, C1), they will appear verbatim in the detail
// string unless the parse path sanitizes them.
//
// This test settles EMPIRICALLY whether yaml.v3 preserves or escapes raw control
// runes in its error strings (the adversary had MEDIUM confidence on this).
//
// Empirical determination:
//   - If any control-char subtest FAILS: yaml.v3 preserves the raw bytes →
//     F-SEC-005 is a CONFIRMED CWE-117 vulnerability. Routes to implementer to
//     sanitize yamlParseDetail (or the E-CFG-005 ConfigError construction site).
//   - If all control-char subtests PASS as-is: yaml.v3 escapes/quotes the key →
//     F-SEC-005 is NOT exploitable today. Tests are kept as regression fence.
//     Defense-in-depth sanitization of the parse path is still recommended.
//
// Positive regression: a plain misspelled key like "keepalive_intervall" must
// still appear in the error so the operator can identify the typo. This guards
// against over-eager sanitization that swallows the key name entirely.
//
// Traces: F-SEC-005, CWE-117, BC-2.09.003 EC-003.
func TestLoadFile_StripsControlCharsFromParseError(t *testing.T) {
	t.Parallel()

	// requireNoControlRune fails if any rune in msg satisfies unicode.IsControl.
	// unicode.IsControl returns true for C0 (U+0000–U+001F), DEL (U+007F), and
	// C1 (U+0080–U+009F) — the full Unicode control-character definition.
	requireNoControlRune := func(t *testing.T, msg string) {
		t.Helper()
		for i, r := range msg {
			if unicode.IsControl(r) {
				t.Errorf("error message contains control rune U+%04X at byte offset %d "+
					"(F-SEC-005 / CWE-117); yamlParseDetail must strip control chars from "+
					"the yaml.v3 error string before embedding it in the E-CFG-005 detail; "+
					"got message: %q", r, i, msg)
			}
		}
	}

	// writeConfig writes content to a temp file and returns the path.
	writeConfig := func(t *testing.T, content string) string {
		t.Helper()
		dir := t.TempDir()
		path := filepath.Join(dir, "config.yaml")
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		return path
	}

	// requireECFG005 asserts that err wraps a *ConfigError with code E-CFG-005.
	requireECFG005 := func(t *testing.T, err error) {
		t.Helper()
		var ce *config.ConfigError
		if !errors.As(err, &ce) {
			t.Fatalf("expected *config.ConfigError with E-CFG-005, got %T: %v", err, err)
		}
		if ce.Code != "E-CFG-005" {
			t.Errorf("expected error code E-CFG-005 (BC-2.09.003 EC-003), got %q", ce.Code)
		}
	}

	// ---- Unknown-key path (Shape-2 via KnownFields): primary injection site ---
	//
	// With dec.KnownFields(true), an unknown YAML key causes yaml.v3 to return:
	//   "yaml: unmarshal errors:\n  line N: field <KEY> not found in type config.Config"
	// If the key carries control runes, they appear in the <KEY> slot verbatim
	// (or escaped, depending on yaml.v3 internals — this test determines which).

	type unknownKeyCase struct {
		name string
		// yamlContent is a valid config with one unknown field whose key carries
		// an embedded control character. The minimal valid surrounding fields
		// (listen_addr + tick_interval) ensure the ONLY error is the unknown key.
		yamlContent string
		// desc describes the injected control char for diagnostics.
		desc string
	}

	unknownKeyCases := []unknownKeyCase{
		{
			// U+009B (8-bit CSI — Control Sequence Introducer): highest-risk C1 rune.
			// On UTF-8/8-bit terminals "\x9b" == "\x1b[" — the ANSI SGR escape prefix.
			// YAML quoted key: "x\u009bmFORGED" (U+009B is embedded inside the quoted key).
			name:        "unknown_key_C1_U009B_CSI",
			yamlContent: "listen_addr: 0.0.0.0:9090\ntick_interval: 10ms\n\"x\u009bmFORGED\": bad\n",
			desc:        "C1 U+009B (8-bit CSI)",
		},
		{
			// U+001B (ESC — C0 escape): classic ANSI terminal-injection byte.
			name:        "unknown_key_C0_ESC",
			yamlContent: "listen_addr: 0.0.0.0:9090\ntick_interval: 10ms\n\"x\x1bmFORGED\": bad\n",
			desc:        "C0 U+001B (ESC)",
		},
		{
			// Embedded carriage return (U+000D) — can forge log line endings.
			name:        "unknown_key_C0_CR",
			yamlContent: "listen_addr: 0.0.0.0:9090\ntick_interval: 10ms\n\"forged\rkey\": bad\n",
			desc:        "C0 U+000D (CR)",
		},
	}

	for _, tc := range unknownKeyCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			path := writeConfig(t, tc.yamlContent)
			_, err := config.LoadFile(path)

			// NOTE: if yaml.v3 cannot parse the quoted key at all (treating the
			// control byte as invalid), the error might be a syntax error (Shape-1)
			// rather than an unknown-field error (Shape-2). Either way it must be
			// E-CFG-005 and must not carry raw control runes.
			requireError(t, err)
			requireECFG005(t, err)

			msg := err.Error()

			// Log the observed yaml.v3 error string for empirical determination.
			t.Logf("observed error for %s case: %q", tc.desc, msg)

			// Security assertion: no control rune (C0, DEL, or C1) may appear in
			// the returned error message (F-SEC-005 / CWE-117).
			requireNoControlRune(t, msg)
		})
	}

	// ---- Positive regression: plain misspelled key must still appear ----------
	//
	// "keepalive_intervall" carries no control characters and must survive in the
	// error message so the operator can identify the typo. This guards against
	// over-eager sanitization that silences the key name entirely.
	t.Run("plain_misspelled_key_preserved", func(t *testing.T) {
		t.Parallel()

		content := "listen_addr: 0.0.0.0:9090\ntick_interval: 10ms\nkeepalive_intervall: 1s\n"
		path := writeConfig(t, content)
		_, err := config.LoadFile(path)
		requireError(t, err)
		requireECFG005(t, err)

		msg := err.Error()
		t.Logf("plain misspelled key error: %q", msg)

		// The misspelled key name must appear in the error.
		requireContains(t, msg, "keepalive_intervall")
		// No control rune either (regression fence for the sanitized case).
		requireNoControlRune(t, msg)
	})
}

// ---- F-SEC-V1 (CWE-117): control chars via escape sequences in double-quoted scalar values --

// TestLoadFile_StripsControlCharsFromEscapedValue verifies empirically whether
// yaml.v3 preserves raw control bytes that were introduced via YAML escape sequences
// inside double-quoted scalar values of known typed fields (F-SEC-V1, CWE-117).
//
// This is a DISTINCT path from F-SEC-005 (which tested control chars in YAML KEY names,
// where yaml.v3's raw-stream scanner rejects them with "control characters are not allowed").
//
// The F-SEC-V1 path:
//  1. A double-quoted YAML value like "\e[31mRED" or "\x9b31m" is decoded by yaml.v3
//     into a string containing the raw control byte (0x1B or 0x9B).
//  2. yaml.v3 tries to unmarshal that decoded string into time.Duration — this fails.
//  3. yaml.v3 returns a TypeError whose message includes the offending value
//     (truncated, e.g. the first 7 bytes), VERBATIM — with the raw control byte intact.
//  4. That TypeError flows through:
//     dec.Decode(&cfg) → KnownFields(true) error → yamlParseDetail (strings.TrimSpace only,
//     NO control stripping) → ConfigError{Code: "E-CFG-005", Detail: yamlParseDetail(err)}.
//
// The sanitizeAddrForError chokepoint (lines 240-248 in config.go) is applied ONLY at
// the two Validate() addr sites, NEVER in the parse/decode path.
//
// Empirical determination:
//   - If any control-char subtest FAILS: yaml.v3 embeds the raw decoded control byte into its
//     TypeError message, and that byte survives all the way to err.Error() →
//     F-SEC-V1 CONFIRMED as a real CWE-117 vector. Routes to implementer to add
//     control-char stripping in yamlParseDetail (or the E-CFG-005 ConfigError construction).
//   - If all subtests PASS: yaml.v3 escapes/quotes the value in its error message, or
//     rejects the escape sequence before decoding, so no raw control rune survives →
//     F-SEC-V1 NOT exploitable today. Tests are kept as a regression fence.
//     Defense-in-depth sanitization of the parse path is still advisable.
//
// Positive regression: a plain invalid duration like "abc" (no control chars) must still
// appear in the error so the operator can identify the problem — guards against any
// future over-eager sanitization.
//
// Traces: F-SEC-V1, CWE-117, BC-2.09.003 EC-003 / FM-010.
func TestLoadFile_StripsControlCharsFromEscapedValue(t *testing.T) {
	t.Parallel()

	// requireNoControlRune fails if any rune in msg satisfies unicode.IsControl.
	// unicode.IsControl covers C0 (U+0000–U+001F), DEL (U+007F), and C1 (U+0080–U+009F).
	requireNoControlRune := func(t *testing.T, msg string) {
		t.Helper()
		for i, r := range msg {
			if unicode.IsControl(r) {
				t.Errorf("error message contains control rune U+%04X at byte offset %d "+
					"(F-SEC-V1 / CWE-117); yamlParseDetail must strip control chars from "+
					"the yaml.v3 TypeError before embedding it in the E-CFG-005 detail; "+
					"got message: %q", r, i, msg)
			}
		}
	}

	// writeConfig writes content to a temp file and returns the path.
	writeConfig := func(t *testing.T, content string) string {
		t.Helper()
		dir := t.TempDir()
		path := filepath.Join(dir, "config.yaml")
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		return path
	}

	// requireECFG005 asserts that err wraps a *ConfigError with code E-CFG-005.
	requireECFG005 := func(t *testing.T, err error) {
		t.Helper()
		var ce *config.ConfigError
		if !errors.As(err, &ce) {
			t.Fatalf("expected *config.ConfigError with E-CFG-005, got %T: %v", err, err)
		}
		if ce.Code != "E-CFG-005" {
			t.Errorf("expected error code E-CFG-005 (BC-2.09.003 EC-003), got %q", ce.Code)
		}
	}

	// These cases use a YAML file with valid surrounding fields (listen_addr + tick_interval
	// are NOT set because we want tick_interval to be the failing field) plus a
	// tick_interval whose double-quoted value contains an escape sequence that yaml.v3
	// decodes into a raw control byte. The typed unmarshal into time.Duration fails →
	// yaml.v3 returns a TypeError that embeds the decoded (raw control byte) value.
	//
	// listen_addr is set to a valid value so that the ONLY yaml.v3 decode error is the
	// tick_interval TypeError. This keeps the error message focused on a single field.
	//
	// Note: "\e" is a YAML double-quoted escape for ESC (U+001B). "\x9b" decodes to the
	// C1 CSI byte (U+009B as a raw byte 0x9B, which is NOT valid UTF-8 — yaml.v3 may
	// handle this as a raw byte). "\r" decodes to CR (U+000D).
	type escapeCase struct {
		name        string
		yamlContent string // full config file content
		desc        string // description of the injected control char
	}

	cases := []escapeCase{
		{
			// Primary vector: ESC via YAML \e escape in double-quoted Duration value.
			// yaml.v3 decodes "\e[31mRED" → []byte{0x1B, '[', '3', '1', 'm', 'R', 'E', 'D'}
			// time.Duration unmarshal fails → TypeError embeds ~first 7 decoded bytes verbatim.
			name:        "tick_interval_ESC_via_backslash_e",
			yamlContent: "listen_addr: 0.0.0.0:9090\ntick_interval: \"\\e[31mRED\"\n",
			desc:        "ESC (U+001B) via YAML \\e escape",
		},
		{
			// C1 CSI via YAML \x9b escape. 0x9B is a raw non-UTF-8 byte; yaml.v3 may
			// decode \x9b as the Unicode codepoint U+009B (C1 CSI) or as the raw byte.
			name:        "tick_interval_C1_CSI_via_backslash_x9b",
			yamlContent: "listen_addr: 0.0.0.0:9090\ntick_interval: \"\\x9b31m\"\n",
			desc:        "C1 CSI (U+009B / 0x9B) via YAML \\x9b escape",
		},
		{
			// CR via YAML \r escape in double-quoted Duration value.
			name:        "tick_interval_CR_via_backslash_r",
			yamlContent: "listen_addr: 0.0.0.0:9090\ntick_interval: \"\\rINJECT\"\n",
			desc:        "CR (U+000D) via YAML \\r escape",
		},
		{
			// ESC via YAML \x1b hex escape — same byte as \e, different syntax.
			name:        "tick_interval_ESC_via_backslash_x1b",
			yamlContent: "listen_addr: 0.0.0.0:9090\ntick_interval: \"\\x1b[0mRESET\"\n",
			desc:        "ESC (U+001B) via YAML \\x1b hex escape",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			path := writeConfig(t, tc.yamlContent)
			_, err := config.LoadFile(path)

			// The typed unmarshal of a non-Duration string always fails → E-CFG-005.
			requireError(t, err)
			requireECFG005(t, err)

			msg := err.Error()

			// Log the observed yaml.v3 error string verbatim for empirical determination.
			// The %q verb shows escape sequences for any non-printable bytes, making the
			// presence of raw control bytes immediately visible in test output.
			t.Logf("observed yaml.v3 error string for %s case: %q", tc.desc, msg)

			// Security assertion: no control rune (C0, DEL, or C1) may survive to the
			// caller's error message (F-SEC-V1 / CWE-117).
			//
			// If this assertion fires, the raw decoded control byte survived the entire
			// pipeline (yaml.v3 TypeError → yamlParseDetail → ConfigError.Detail → Error())
			// and F-SEC-V1 is CONFIRMED as a real vulnerability.
			requireNoControlRune(t, msg)
		})
	}

	// Positive regression: a plain invalid duration ("abc") must still appear in the
	// error message. This guards against any future over-eager sanitization that
	// silently swallows the offending value, leaving operators unable to diagnose
	// misconfiguration.
	t.Run("plain_invalid_duration_preserved_in_error", func(t *testing.T) {
		t.Parallel()

		content := "listen_addr: 0.0.0.0:9090\ntick_interval: \"abc\"\n"
		path := writeConfig(t, content)
		_, err := config.LoadFile(path)
		requireError(t, err)
		requireECFG005(t, err)

		msg := err.Error()
		t.Logf("plain invalid duration error: %q", msg)

		// "abc" has no control chars; it must survive into the error message.
		requireContains(t, msg, "abc")
		// And naturally no control rune either.
		requireNoControlRune(t, msg)
	})
}

// ---- F-SEC-C1 (CWE-117): C1 control block U+0080–U+009F passes through sanitizer --

// TestConfigValidate_StripsC1ControlChars verifies that sanitizeAddrForError strips
// the entire C1 control block (U+0080–U+009F) before interpolating an address into
// an error message.
//
// Background: the existing predicate `r >= 0x20 && r != 0x7F` strips C0 (U+0000–U+001F)
// and the lone DEL (U+007F) but lets the C1 block (U+0080–U+009F) through unchanged.
// U+009B is the 8-bit CSI (Control Sequence Introducer) — on UTF-8/8-bit terminals it
// initiates an ANSI escape exactly like ESC[. This is the same log/terminal-injection
// vector that the round-1 fix (F-SEC-002) was meant to close, left half-open.
//
// The doc comment on sanitizeAddrForError claims it strips "U+0000–U+001F AND
// U+007F–U+009F". These tests enforce that claim.
//
// Red Gate: the current predicate DOES NOT strip 0x80–0x9F, so the C1 rune in each
// addr case survives and appears verbatim in the error message. Every C1 subtest
// therefore FAILS against current code.
//
// Traces: F-SEC-C1, CWE-117, BC-2.09.003 PC-5/PC-6.
// ── AC-011 / PC-10 / E-CFG-008: management_socket validation ─────────────────

// TestConfig_Validate_ManagementSocket_E_CFG_008_AC011 verifies that Validate()
// rejects management_socket when it is present but empty or whitespace-only, and
// accepts it when absent or a non-empty, non-whitespace string.
//
// Canonical E-CFG-008 message from BC-2.09.003 PC-10:
//
//	"config error: management_socket: must not be empty. Fix: set to a valid Unix
//	socket path, e.g. '/run/switchboard-router.sock', or remove the field to use
//	the daemon default"
//
// Traces: BC-2.09.003 PC-10, AC-011, E-CFG-008.
func TestConfig_Validate_ManagementSocket_E_CFG_008_AC011(t *testing.T) {
	t.Parallel()

	// baseValid is a minimal valid config for all sub-cases that do NOT test
	// management_socket errors in isolation. The new management_socket field
	// must not cause regressions when absent/valid.
	makeBase := func() *config.Config {
		return &config.Config{
			ListenAddr:   "0.0.0.0:9090",
			TickInterval: 10 * time.Millisecond,
		}
	}

	cases := []struct {
		name             string
		managementSocket string
		wantErr          bool   // true = expect E-CFG-008
		wantInMsg        string // fragment that must appear in error message
	}{
		{
			// (a) Empty string == Go zero-value == absent from YAML == accepted.
			// BC-2.09.003 PC-10: "When present, it must be non-empty." Go yaml cannot
			// distinguish absent from explicit empty string; empty maps to "absent" →
			// validator returns nil (no E-CFG-008).
			name:             "empty_string_accepted_as_absent",
			managementSocket: "",
			wantErr:          false,
		},
		{
			// (b) Whitespace-only → E-CFG-008.
			name:             "whitespace_only_rejected",
			managementSocket: "   ",
			wantErr:          true,
			wantInMsg:        "management_socket",
		},
		{
			// (c) Tab-only whitespace → E-CFG-008.
			name:             "tab_only_rejected",
			managementSocket: "\t",
			wantErr:          true,
			wantInMsg:        "management_socket",
		},
		{
			// (d) Valid socket path → accepted.
			name:             "valid_unix_path_accepted",
			managementSocket: "/run/switchboard-router.sock",
			wantErr:          false,
		},
		{
			// (e) Valid TCP address (console mode) → accepted.
			name:             "valid_tcp_address_accepted",
			managementSocket: "127.0.0.1:9091",
			wantErr:          false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := makeBase()
			cfg.ManagementSocket = tc.managementSocket
			err := cfg.Validate()

			if tc.wantErr {
				requireError(t, err)
				requireECFG001(t, err)
				msg := err.Error()
				requireContains(t, msg, tc.wantInMsg)
				// E-CFG-008 canonical fragment check.
				requireContains(t, msg, "management_socket")
				requireContains(t, msg, "must not be empty")
			} else {
				requireNoError(t, err)
			}
		})
	}

	// Exhaustive error collection: management_socket whitespace error must appear
	// ALONGSIDE other config errors in a single E-CFG-001 response.
	t.Run("exhaustive_reporting_with_other_errors", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{
			ListenAddr:       "",    // E-CFG-001 (missing required field)
			TickInterval:     0,     // E-CFG-001 (missing required field)
			ManagementSocket: "   ", // E-CFG-008
		}
		err := cfg.Validate()
		requireError(t, err)
		requireECFG001(t, err)

		msg := err.Error()
		// All three errors must appear in a single combined E-CFG-001.
		requireContains(t, msg, "listen_addr")
		requireContains(t, msg, "tick_interval")
		requireContains(t, msg, "management_socket")
	})
}

// ── AC-012 / PC-11 / E-CFG-009: authorized_operator_keys validation ───────────

// mustEd25519PEMPublicKey generates an Ed25519 keypair and marshals the public
// key into PKIX PEM format ("PUBLIC KEY" type). Used to construct valid fixtures.
func mustEd25519PEMPublicKey(t *testing.T) string {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		t.Fatalf("MarshalPKIXPublicKey: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: der,
	}))
}

// TestConfig_Validate_AuthorizedOperatorKeys_E_CFG_009_AC012 verifies that
// Validate() reports E-CFG-009 for each invalid authorized_operator_keys entry,
// collects all errors exhaustively, and accepts empty lists or valid Ed25519 PEM keys.
//
// Canonical E-CFG-009 message from BC-2.09.003 PC-11:
//
//	"config error: authorized_operator_keys[<N>]: entry is not a valid Ed25519 PEM
//	PUBLIC KEY block. Fix: provide a PEM-encoded Ed25519 public key (type 'PUBLIC
//	KEY', 32-byte key length)"
//
// Traces: BC-2.09.003 PC-11, AC-012, E-CFG-009.
func TestConfig_Validate_AuthorizedOperatorKeys_E_CFG_009_AC012(t *testing.T) {
	t.Parallel()

	makeValid := func() *config.Config {
		return &config.Config{
			ListenAddr:   "0.0.0.0:9090",
			TickInterval: 10 * time.Millisecond,
		}
	}

	t.Run("invalid_pem_at_index_0", func(t *testing.T) {
		t.Parallel()

		cfg := makeValid()
		cfg.AuthorizedOperatorKeys = []string{"not-pem-at-all"}
		err := cfg.Validate()
		requireError(t, err)
		requireECFG001(t, err)
		msg := err.Error()
		requireContains(t, msg, "authorized_operator_keys[0]")
		requireContains(t, msg, "not a valid Ed25519 PEM PUBLIC KEY")
	})

	t.Run("valid_at_0_invalid_at_1", func(t *testing.T) {
		t.Parallel()

		validPEM := mustEd25519PEMPublicKey(t)
		cfg := makeValid()
		cfg.AuthorizedOperatorKeys = []string{validPEM, "garbage"}
		err := cfg.Validate()
		requireError(t, err)
		requireECFG001(t, err)
		msg := err.Error()
		// index 1 must be reported.
		requireContains(t, msg, "authorized_operator_keys[1]")
		// index 0 must NOT be reported (it is valid).
		if strings.Contains(msg, "authorized_operator_keys[0]") {
			t.Errorf("E-CFG-009: valid entry at index 0 must not produce an error; got: %q", msg)
		}
	})

	t.Run("empty_list_accepted", func(t *testing.T) {
		t.Parallel()

		cfg := makeValid()
		cfg.AuthorizedOperatorKeys = []string{}
		requireNoError(t, cfg.Validate())
	})

	t.Run("nil_list_accepted", func(t *testing.T) {
		t.Parallel()

		cfg := makeValid()
		cfg.AuthorizedOperatorKeys = nil
		requireNoError(t, cfg.Validate())
	})

	t.Run("valid_ed25519_pem_accepted", func(t *testing.T) {
		t.Parallel()

		validPEM := mustEd25519PEMPublicKey(t)
		cfg := makeValid()
		cfg.AuthorizedOperatorKeys = []string{validPEM}
		requireNoError(t, cfg.Validate())
	})

	t.Run("multiple_valid_pem_accepted", func(t *testing.T) {
		t.Parallel()

		cfg := makeValid()
		cfg.AuthorizedOperatorKeys = []string{
			mustEd25519PEMPublicKey(t),
			mustEd25519PEMPublicKey(t),
		}
		requireNoError(t, cfg.Validate())
	})

	t.Run("rsa_pem_rejected_wrong_key_type", func(t *testing.T) {
		// A valid PEM block of type "PUBLIC KEY" but containing an RSA key (not Ed25519)
		// must be rejected with E-CFG-009. We simulate this by using a PEM block
		// whose DER content is not a valid Ed25519 public key.
		//
		// For simplicity in a test, we use a well-known invalid DER payload by placing
		// random bytes in a "PUBLIC KEY" PEM block — the PKIX parse will fail, which
		// is sufficient to trigger E-CFG-009.
		t.Parallel()

		invalidDER := make([]byte, 128) // 128 random bytes — not a valid PKIX key
		_, err := rand.Read(invalidDER)
		if err != nil {
			t.Fatalf("rand.Read: %v", err)
		}
		fakePEM := string(pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: invalidDER,
		}))

		cfg := makeValid()
		cfg.AuthorizedOperatorKeys = []string{fakePEM}
		valErr := cfg.Validate()
		requireError(t, valErr)
		requireECFG001(t, valErr)
		msg := valErr.Error()
		requireContains(t, msg, "authorized_operator_keys[0]")
		requireContains(t, msg, "not a valid Ed25519 PEM PUBLIC KEY")
	})

	t.Run("wrong_pem_block_type_rejected", func(t *testing.T) {
		// A PEM block of type "CERTIFICATE" (not "PUBLIC KEY") must be rejected.
		t.Parallel()

		wrongTypePEM := string(pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: make([]byte, 32),
		}))

		cfg := makeValid()
		cfg.AuthorizedOperatorKeys = []string{wrongTypePEM}
		valErr := cfg.Validate()
		requireError(t, valErr)
		requireECFG001(t, valErr)
		msg := valErr.Error()
		requireContains(t, msg, "authorized_operator_keys[0]")
	})

	// Exhaustive error collection: two bad entries at indices 0 and 2; one valid
	// at index 1. E-CFG-009[0] and E-CFG-009[2] must both appear in a single error.
	t.Run("exhaustive_reporting_both_bad_entries_collected", func(t *testing.T) {
		t.Parallel()

		validPEM := mustEd25519PEMPublicKey(t)
		cfg := makeValid()
		cfg.AuthorizedOperatorKeys = []string{
			"bad-entry-zero", // index 0 → E-CFG-009
			validPEM,         // index 1 → OK
			"bad-entry-two",  // index 2 → E-CFG-009
		}
		err := cfg.Validate()
		requireError(t, err)
		requireECFG001(t, err)
		msg := err.Error()
		requireContains(t, msg, "authorized_operator_keys[0]")
		requireContains(t, msg, "authorized_operator_keys[2]")
		if strings.Contains(msg, "authorized_operator_keys[1]") {
			t.Errorf("valid entry at index 1 must not produce an error; got: %q", msg)
		}
	})
}

// TestConfig_Validate_ManagementFields_DoNotBreakExistingValidCases verifies
// that adding management_socket and authorized_operator_keys fields to an
// otherwise valid config does not break existing validation. Regression guard.
func TestConfig_Validate_ManagementFields_DoNotBreakExistingValidCases(t *testing.T) {
	t.Parallel()

	validPEM := mustEd25519PEMPublicKey(t)

	cfg := &config.Config{
		ListenAddr:             "0.0.0.0:9090",
		TickInterval:           10 * time.Millisecond,
		DrainTimeout:           10 * time.Second,
		KeepaliveInterval:      1 * time.Second,
		ManagementSocket:       "/run/switchboard-router.sock",
		AuthorizedOperatorKeys: []string{validPEM},
	}
	requireNoError(t, cfg.Validate())
}

// TestConfig_Validate_StripsC1ControlChars is defined later in this file.
// The next test continues the CWE-117 / F-SEC-C1 coverage:

func TestConfigValidate_StripsC1ControlChars(t *testing.T) {
	t.Parallel()

	// requireNoControlRune fails if any rune in msg satisfies unicode.IsControl.
	// unicode.IsControl returns true for U+0000–U+001F, U+007F–U+009F — exactly
	// the range the sanitizer doc comment promises to strip.
	requireNoControlRune := func(t *testing.T, msg string) {
		t.Helper()
		for i, r := range msg {
			if unicode.IsControl(r) {
				t.Errorf("error message contains control rune U+%04X at byte offset %d "+
					"(F-SEC-C1 / CWE-117); sanitizeAddrForError must strip U+0080–U+009F; "+
					"got message: %q", r, i, msg)
			}
		}
	}

	// ---- listen_addr C1 injection cases ----------------------------------------

	type c1Case struct {
		name string
		addr string
	}

	listenC1Cases := []c1Case{
		{
			// U+0080 (PAD — Padding Character) — first C1 codepoint.
			name: "listen_addr_C1_U0080_PAD",
			addr: "0.0.0.0:9090\u0080injected",
		},
		{
			// U+009B (CSI — 8-bit Control Sequence Introducer).
			// On many terminals "\x9b" == "\x1b[" — the ANSI SGR escape prefix.
			// This is the highest-risk C1 codepoint for terminal injection.
			name: "listen_addr_C1_U009B_CSI",
			addr: "0.0.0.0:9090\u009b31mRED",
		},
		{
			// U+009F (APC — Application Program Command) — last C1 codepoint.
			name: "listen_addr_C1_U009F_APC",
			addr: "0.0.0.0:9090\u009f",
		},
		{
			// Spread across the full C1 block: U+0081, U+008D, U+0090, U+009B, U+009C.
			// A sanitizer that strips only individual codepoints rather than the full
			// range would let at least one of these through.
			name: "listen_addr_C1_spread_across_block",
			addr: "0.0.0.0:9090\u0081\u008d\u0090\u009b\u009c",
		},
	}

	for _, tc := range listenC1Cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := &config.Config{
				ListenAddr:   tc.addr,
				TickInterval: 10 * time.Millisecond,
			}
			err := cfg.Validate()
			requireError(t, err)
			requireECFG001(t, err)

			msg := err.Error()

			// Assertion 1: error still identifies the field.
			requireContains(t, msg, "listen_addr")

			// Assertion 2 (security): no control rune — C0, DEL, or C1 — may appear
			// in the error message (F-SEC-C1 / CWE-117).
			requireNoControlRune(t, msg)
		})
	}

	// ---- upstream_routers C1 injection cases ------------------------------------

	upstreamC1Cases := []c1Case{
		{
			// U+009B (CSI) in upstream_routers[0].addr — same terminal-injection risk.
			name: "upstream_routers_addr_C1_U009B_CSI",
			addr: "10.0.0.1:9090\u009b31mRED",
		},
		{
			// U+0085 (NEL — Next Line) — a C1 line-terminator; may split log lines
			// on some parsers just like \n.
			name: "upstream_routers_addr_C1_U0085_NEL",
			addr: "10.0.0.1:9090\u0085",
		},
	}

	for _, tc := range upstreamC1Cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := &config.Config{
				ListenAddr:   "0.0.0.0:9090",
				TickInterval: 10 * time.Millisecond,
				UpstreamRouters: []config.UpstreamRouter{
					{Addr: tc.addr},
				},
			}
			err := cfg.Validate()
			requireError(t, err)
			requireECFG001(t, err)

			msg := err.Error()

			// Assertion 1: error still identifies the upstream field.
			requireContains(t, msg, "upstream_routers")

			// Assertion 2 (security): no control rune may appear.
			requireNoControlRune(t, msg)
		})
	}

	// ---- Positive regression: plain printable invalid value must still appear ---
	//
	// Confirm that the sanitizer does not over-strip printable characters.
	// "not-a-host-port" has no control characters and must survive verbatim
	// (or quoted) so the operator can identify the problem.
	//
	// This subtest PASSES against current code — it guards against over-sanitization.
	t.Run("plain_printable_value_preserved_after_sanitization", func(t *testing.T) {
		t.Parallel()

		const plainInvalid = "not-a-host-port"
		cfg := &config.Config{
			ListenAddr:   plainInvalid,
			TickInterval: 10 * time.Millisecond,
		}
		err := cfg.Validate()
		requireError(t, err)
		requireECFG001(t, err)

		msg := err.Error()
		requireContains(t, msg, "listen_addr")
		// The offending value (or its quoted form) must appear so the operator
		// can identify the problem; sanitization must not swallow printable chars.
		if !strings.Contains(msg, plainInvalid) && !strings.Contains(msg, `"not-a-host-port"`) {
			t.Errorf("sanitization must preserve printable value %q in error message; got: %q",
				plainInvalid, msg)
		}
	})
}

// ---- AC-001 (S-BL.NODE-ADMISSION-PROVISIONING): admission_key_file validation ----
//
// Traces:
//   BC-2.09.003 v2.1 Postcondition 12
//   BC-2.09.004 Postconditions 1–2
//   E-CFG-014
//   ARCH-06 §Config purity contract (no file I/O in Validate)
//
// All four named tests must be present per the story AC-001 spec.

// TestConfig_Validate_AdmissionKeyFile_AbsentAccepted verifies that when
// admission_key_file is absent (empty string), Validate() accepts the config
// and returns no error for this field (BC-2.09.004 PC-1; EC-001).
func TestConfig_Validate_AdmissionKeyFile_AbsentAccepted(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		ListenAddr:       "0.0.0.0:9090",
		TickInterval:     10 * time.Millisecond,
		AdmissionKeyFile: "", // absent / empty string
	}
	err := cfg.Validate()
	requireNoError(t, err)
}

// TestConfig_Validate_AdmissionKeyFile_ValidPathAccepted verifies that when
// admission_key_file is a non-whitespace string, Validate() accepts it regardless
// of whether the file exists on disk (BC-2.09.004 PC-2; ARCH-06 §Config purity —
// Validate performs no file I/O).
func TestConfig_Validate_AdmissionKeyFile_ValidPathAccepted(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		path string
	}{
		{
			name: "absolute_path_no_file_exists",
			// A path that does not exist — Validate() must not stat/open it.
			path: "/nonexistent/path/that/will/never/exist/admission.pem",
		},
		{
			name: "default_path_value",
			path: "/var/lib/switchboard/access-admission-identity.pem",
		},
		{
			name: "relative_path_accepted",
			path: "some/relative/path.pem",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := &config.Config{
				ListenAddr:       "0.0.0.0:9090",
				TickInterval:     10 * time.Millisecond,
				AdmissionKeyFile: tc.path,
			}
			err := cfg.Validate()
			requireNoError(t, err)
		})
	}
}

// TestConfig_Validate_AdmissionKeyFile_WhitespaceOnlyRejectsE_CFG_014 verifies
// that when admission_key_file is present but whitespace-only, Validate() returns
// an error containing E-CFG-014 with the canonical error message
// (BC-2.09.003 v2.1 PC-12; BC-2.09.004 PC-1; E-CFG-014).
//
// This test MUST FAIL at Red Gate because the current stub in config.go does not
// implement the whitespace-rejection check (see the TODO comment).
func TestConfig_Validate_AdmissionKeyFile_WhitespaceOnlyRejectsE_CFG_014(t *testing.T) {
	t.Parallel()

	// Canonical E-CFG-014 message per rulings v1.0 §1.2 and story AC-001 PC-3.
	const wantMsg = "config error: admission_key_file: must not be empty. Fix: set to a valid file path, e.g. '/var/lib/switchboard/access-admission-identity.pem', or remove the field to use the daemon default"

	cases := []struct {
		name  string
		value string
	}{
		{name: "single_space", value: " "},
		{name: "multiple_spaces", value: "   "},
		{name: "tab_only", value: "\t"},
		{name: "mixed_whitespace", value: " \t\n  "},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := &config.Config{
				ListenAddr:       "0.0.0.0:9090",
				TickInterval:     10 * time.Millisecond,
				AdmissionKeyFile: tc.value,
			}
			err := cfg.Validate()
			requireError(t, err)

			// Must carry E-CFG-001 outer code (all validation failures wrapped).
			requireECFG001(t, err)

			// Must contain the canonical E-CFG-014 message verbatim.
			msg := err.Error()
			requireContains(t, msg, wantMsg)
		})
	}
}

// TestConfig_Validate_AdmissionKeyFile_NoIOPerformed verifies that Validate()
// performs NO file I/O for admission_key_file — it must not stat, open, or read
// the path (ARCH-06 §Config purity contract; BC-2.09.004 PC-2).
//
// Method: supply a non-whitespace path pointing to a directory entry that, if
// stat'd or opened, would cause an OS error. Validate() must return nil (path
// is non-whitespace and valid from Validate's perspective). If Validate() were
// to open the path it would get an error and either panic or fail — this test
// would then fail with an unexpected error, catching the I/O violation.
//
// Also verifies that a valid path to a file that does not exist is accepted
// (Validate must not check existence).
func TestConfig_Validate_AdmissionKeyFile_NoIOPerformed(t *testing.T) {
	t.Parallel()

	// Create a tempdir for a path that exists as a directory — any attempt to
	// open it as a key file would produce an OS error (IsDir). We use t.TempDir
	// so the directory is real; if Validate() opens or stats it, it would see
	// a directory, not a PEM file, and could only succeed by ignoring the result.
	// The actual invariant is that Validate() does NO I/O whatsoever.
	dir := t.TempDir()
	dirPath := filepath.Join(dir, "a_directory_not_a_pem_file")
	if mkErr := os.Mkdir(dirPath, 0o755); mkErr != nil {
		t.Fatalf("setup: mkdir %s: %v", dirPath, mkErr)
	}

	cfg := &config.Config{
		ListenAddr:       "0.0.0.0:9090",
		TickInterval:     10 * time.Millisecond,
		AdmissionKeyFile: dirPath, // exists and is a dir — any I/O would reveal this
	}
	// Validate() MUST accept this value — it's non-empty, non-whitespace.
	// If it opens the path and checks its type, it would behave differently from
	// the spec. The spec says: only reject whitespace-only values.
	err := cfg.Validate()
	requireNoError(t, err)

	// Also verify a path that does not exist at all is accepted.
	cfg2 := &config.Config{
		ListenAddr:       "0.0.0.0:9090",
		TickInterval:     10 * time.Millisecond,
		AdmissionKeyFile: filepath.Join(dir, "does_not_exist.pem"),
	}
	err2 := cfg2.Validate()
	requireNoError(t, err2)
}

// TestConfig_Validate_AdmissionKeyFile_ExhaustiveErrorCollection verifies that
// when admission_key_file is whitespace-only AND another field is also invalid,
// both errors are returned together — exhaustive collection is preserved
// (BC-2.09.003 Invariant 4; AC-001 PC-5).
//
// This test MUST FAIL at Red Gate (admission_key_file whitespace check not yet
// implemented, so only the tick_interval error would appear, not E-CFG-014).
func TestConfig_Validate_AdmissionKeyFile_ExhaustiveErrorCollection(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		ListenAddr:       "0.0.0.0:9090",
		TickInterval:     0,     // invalid — triggers tick_interval error
		AdmissionKeyFile: "   ", // whitespace-only — triggers E-CFG-014
	}
	err := cfg.Validate()
	requireError(t, err)
	requireECFG001(t, err)

	msg := err.Error()
	// Both errors must appear.
	requireContains(t, msg, "tick_interval")
	requireContains(t, msg, "admission_key_file")
}
