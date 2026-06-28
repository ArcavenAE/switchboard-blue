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
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
			ListenAddr:   "0.0.0.0:9090",
			TickInterval: 10 * time.Millisecond,
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

// TestLoadFile_MissingFile verifies that LoadFile returns E-CFG-004 (or the
// canonical E-CFG-002 alias per story EC-001) with an actionable message that
// includes the expected path when the config file does not exist.
//
// Traces: BC-2.09.003 EC-001.
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

	// Accept either canonical E-CFG-004 (per BC EC-001) or E-CFG-002 (per story EC-001).
	// The implementer may consolidate; both codes are acceptable here.
	var ce *config.ConfigError
	if !errors.As(err, &ce) {
		t.Fatalf("expected *config.ConfigError, got %T: %v", err, err)
	}
	if ce.Code != "E-CFG-004" && ce.Code != "E-CFG-002" {
		t.Errorf("expected error code E-CFG-004 or E-CFG-002, got %q", ce.Code)
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
