// Package config provides YAML config parsing and validation for the switchboard daemon.
// This package is a DAG root: it has no internal imports (ARCH-08 position 1).
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"gopkg.in/yaml.v3"
)

// maxConfigFileSize is the upper bound on config file size accepted by LoadFile.
// A router config is tiny; 1 MiB is a generous cap that protects against
// accidental --config /dev/zero or a mispointed path to a large file (CWE-400).
const maxConfigFileSize = 1 << 20 // 1 MiB

// ErrValidation (E-CFG-001) is returned when one or more config fields fail validation
// (range violation, constraint violation, or a required field is missing).
var ErrValidation = &ConfigError{Code: "E-CFG-001"}

// ErrConfigFileNotFound (E-CFG-004) is returned when the config file cannot be found
// at the expected path (BC-2.09.003 EC-001).
var ErrConfigFileNotFound = &ConfigError{Code: "E-CFG-004"}

// ErrParseError (E-CFG-005) is returned when the config file contains malformed YAML
// (BC-2.09.003 EC-003 / FM-010).
var ErrParseError = &ConfigError{Code: "E-CFG-005"}

// ConfigError is the sentinel error type for config package errors.
// Code is one of E-CFG-001, E-CFG-004, E-CFG-005.
type ConfigError struct {
	// Code is the machine-readable error code (e.g., "E-CFG-001").
	Code string
	// Detail carries a human-readable description of the specific problem.
	Detail string
}

// Error implements the error interface.
// Format: "<Code>: <Detail>" when Detail is set, "<Code>" otherwise.
func (e *ConfigError) Error() string {
	if e.Detail == "" {
		return e.Code
	}
	return e.Code + ": " + e.Detail
}

// Is supports errors.Is comparisons against sentinel ConfigError values by
// matching on Code alone, so callers can write errors.Is(err, config.ErrValidation).
func (e *ConfigError) Is(target error) bool {
	var t *ConfigError
	if errors.As(target, &t) {
		return e.Code == t.Code
	}
	return false
}

// ValidationError describes a single field-level validation failure.
// Validate() collects all failures and returns them as a slice rather than
// stopping at the first error (BC-2.09.003 postcondition 2, AC-002).
type ValidationError struct {
	// Field is the YAML key of the invalid or missing config field.
	Field string
	// Value is the string representation of the invalid value, or empty if the
	// field is missing.
	Value string
	// Problem is a human-readable description of the constraint violated.
	Problem string
	// Suggestion is an actionable fix hint shown to the operator.
	Suggestion string
}

// Error implements the error interface.
// Format: "config error: <field>: <problem>. Fix: <suggestion>" per BC-2.09.003 postcondition 2.
// The field name appears exactly once (in the <field>: slot); the problem describes the value/range.
func (v *ValidationError) Error() string {
	if v.Suggestion != "" {
		if v.Value != "" {
			return fmt.Sprintf("config error: %s: value %s %s. Fix: %s",
				v.Field, v.Value, v.Problem, v.Suggestion)
		}
		return fmt.Sprintf("config error: %s: %s. Fix: %s", v.Field, v.Problem, v.Suggestion)
	}
	if v.Value != "" {
		return fmt.Sprintf("config error: %s: value %s %s", v.Field, v.Value, v.Problem)
	}
	return fmt.Sprintf("config error: %s: %s", v.Field, v.Problem)
}

// Config is the top-level switchboard daemon configuration.
// The file format is YAML (gopkg.in/yaml.v3); ARCH-06 binary budget table
// lists "YAML config parser (gopkg.in/yaml.v3)" explicitly.
//
// Required fields: ListenAddr, TickInterval.
type Config struct {
	// ListenAddr is the TCP address the router listens on, e.g. "0.0.0.0:9090".
	// Required.
	ListenAddr string `yaml:"listen_addr"`

	// TickInterval is the routing-tick cadence. Valid range: [5ms, 50ms] (ADR-008).
	// Required.
	TickInterval time.Duration `yaml:"tick_interval"`

	// UpstreamRouters lists upstream router entries for PE mode.
	// An empty slice means E mode (BC-2.09.001).
	// Each entry has an Addr field validated as host:port (BC-2.09.003 PC-6 / AC-006).
	UpstreamRouters []UpstreamRouter `yaml:"upstream_routers"`

	// DrainTimeout is the maximum time allowed for graceful drain (BC-2.09.002).
	// Optional; defaults applied by the daemon, not by Validate.
	DrainTimeout time.Duration `yaml:"drain_timeout"`

	// KeepaliveInterval is the node keepalive cadence (FM-009).
	// Optional; defaults applied by the daemon, not by Validate.
	KeepaliveInterval time.Duration `yaml:"keepalive_interval"`
}

// UpstreamRouter is a single entry in the upstream_routers list.
// Addr is the TCP address of the upstream router, validated as host:port
// (BC-2.09.003 PC-6 / AC-006 / E-CFG-003).
type UpstreamRouter struct {
	// Addr is the TCP address of the upstream router, e.g. "10.0.0.1:9090".
	Addr string `yaml:"addr"`
}

// TickIntervalMin is the lower bound of the valid tick_interval range (ADR-008).
const TickIntervalMin = 5 * time.Millisecond

// TickIntervalMax is the upper bound of the valid tick_interval range (ADR-008).
const TickIntervalMax = 50 * time.Millisecond

// Validate checks all fields in c and returns an error if any field is invalid
// or a required field is missing.
//
// Validate is pure-core: it performs no I/O and opens no sockets. It must be
// called before any socket is opened (BC-2.09.003 invariant 1, AC-004).
//
// All validation failures are collected and returned together so the operator
// sees every problem in one pass (BC-2.09.003 postcondition 2 / invariant 4, AC-002).
//
// Returns *ConfigError with code E-CFG-001 wrapping all ValidationErrors on failure,
// or nil on success.
func (c *Config) Validate() error {
	var failures []string

	// Required: listen_addr must be non-empty and non-whitespace.
	if strings.TrimSpace(c.ListenAddr) == "" {
		failures = append(failures, (&ValidationError{
			Field:      "listen_addr",
			Problem:    "required field missing",
			Suggestion: "add 'listen_addr: <ip>:<port>' to config, e.g. 'listen_addr: 0.0.0.0:9090'",
		}).Error())
	} else {
		// AC-005 / PC-5 / E-CFG-002: listen_addr must be a valid host:port.
		// net.SplitHostPort validates the host:port structure; additionally we
		// require the port to be a non-empty numeric string.
		if err := validateHostPort(c.ListenAddr); err != nil {
			failures = append(failures, fmt.Sprintf(
				"config error: listen_addr: '%s' is not a valid host:port. Fix: use '<ip>:<port>' format, e.g. '0.0.0.0:9090'",
				sanitizeAddrForError(c.ListenAddr),
			))
		}
	}

	// Required: tick_interval must be in [5ms, 50ms].
	// Zero is treated as missing/invalid per BC-2.09.003 EC-002.
	if c.TickInterval < TickIntervalMin || c.TickInterval > TickIntervalMax {
		var val string
		if c.TickInterval == 0 {
			val = "0s"
		} else {
			val = c.TickInterval.String()
		}
		failures = append(failures, (&ValidationError{
			Field:      "tick_interval",
			Value:      val,
			Problem:    fmt.Sprintf("is outside allowed range [%s, %s]", TickIntervalMin, TickIntervalMax),
			Suggestion: fmt.Sprintf("set to a value in [%s, %s], e.g. 'tick_interval: 10ms' for interactive sessions", TickIntervalMin, TickIntervalMax),
		}).Error())
	}

	// AC-006 / PC-6 / E-CFG-003: each upstream_routers[N].addr must be host:port.
	for i, r := range c.UpstreamRouters {
		if err := validateHostPort(r.Addr); err != nil {
			failures = append(failures, fmt.Sprintf(
				"config error: upstream_routers[%d].addr: '%s' is not a valid host:port. Fix: use '<ip>:<port>' format, e.g. '10.0.0.1:9090'",
				i, sanitizeAddrForError(r.Addr),
			))
		}
	}

	// AC-007 / PC-7 / E-CFG-006: drain_timeout must not be negative when set.
	// Zero means absent — the daemon applies the default (10s) at startup (S-7.04).
	if c.DrainTimeout < 0 {
		failures = append(failures, fmt.Sprintf(
			"config error: drain_timeout: must not be negative; got '%s'. Fix: remove the field to use the daemon default (10s), or set to a positive duration, e.g. '10s'",
			c.DrainTimeout,
		))
	}

	// AC-008 / PC-8 / E-CFG-007: keepalive_interval must not be negative when set.
	// Zero means absent — the daemon applies the default (1s) at startup (S-7.04).
	if c.KeepaliveInterval < 0 {
		failures = append(failures, fmt.Sprintf(
			"config error: keepalive_interval: must not be negative; got '%s'. Fix: remove the field to use the daemon default (1s), or set to a positive duration, e.g. '1s'",
			c.KeepaliveInterval,
		))
	}

	if len(failures) == 0 {
		return nil
	}

	// Build combined detail: all field errors in one message so the operator
	// sees every problem at once (AC-002 / BC-2.09.003 postcondition 2 / invariant 4).
	return &ConfigError{
		Code:   "E-CFG-001",
		Detail: strings.Join(failures, "; "),
	}
}

// stripControlChars removes all Unicode control characters from s —
// C0 (U+0000–U+001F), DEL (U+007F), and C1 (U+0080–U+009F) — using
// unicode.IsControl so the predicate matches the full Unicode definition.
//
// Embedding raw control bytes in error strings allows an attacker to forge
// log lines or corrupt terminal output (CWE-117 / F-SEC-002 / F-SEC-V1).
// Printable characters are never altered, so operator-visible values survive
// intact for diagnosis.
func stripControlChars(s string) string {
	var b strings.Builder
	for _, r := range s {
		if !unicode.IsControl(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// sanitizeAddrForError strips control characters from an operator-supplied
// address value before interpolating it into an error message.
//
// Embedding raw control characters (\n, \r, \x1b, etc.) into an error string
// allows an attacker to forge log lines or corrupt terminal output (CWE-117 /
// F-SEC-002). We strip all Unicode control characters — C0 (U+0000–U+001F),
// DEL (U+007F), and C1 (U+0080–U+009F) — using unicode.IsControl so the
// predicate matches the full Unicode definition rather than a hand-rolled range.
//
// Legitimate values (e.g. "not-a-host-port") pass through unchanged so the
// operator can see what they mistyped (BC-2.09.003 PC-5/PC-6).
func sanitizeAddrForError(addr string) string {
	return stripControlChars(addr)
}

// validateHostPort returns nil if addr is a valid host:port as defined by
// net.SplitHostPort, with an additional requirement that the port is a non-empty
// numeric string in [0, 65535]. An empty host is allowed (e.g. ":9090").
func validateHostPort(addr string) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	_ = host // empty host is valid (listen on all interfaces)
	if port == "" {
		return fmt.Errorf("missing port in address")
	}
	n, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("non-numeric port %q", port)
	}
	if n < 0 || n > 65535 {
		return fmt.Errorf("port %d out of range [0, 65535]", n)
	}
	return nil
}

// LoadFile reads and parses a YAML config file at path, returning a *Config.
//
// Returns *ConfigError with code E-CFG-004 if the file does not exist.
// Returns *ConfigError with code E-CFG-005 if the YAML is malformed.
// Does NOT call Validate — the caller is responsible for calling Validate
// after LoadFile (see ARCH-06 binding sequence).
func LoadFile(path string) (*Config, error) {
	// Guard: stat before reading to reject non-regular files (e.g. /dev/zero,
	// directories) and files that exceed maxConfigFileSize (CWE-400).
	fi, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, &ConfigError{
				Code:   "E-CFG-004",
				Detail: fmt.Sprintf("config file not found: %s", path),
			}
		}
		return nil, &ConfigError{
			Code:   "E-CFG-004",
			Detail: fmt.Sprintf("config file not found: %s: %v", path, err),
		}
	}
	if !fi.Mode().IsRegular() {
		return nil, &ConfigError{
			Code:   "E-CFG-005",
			Detail: fmt.Sprintf("config parse error: %s is not a regular file", path),
		}
	}
	if fi.Size() > maxConfigFileSize {
		return nil, &ConfigError{
			Code:   "E-CFG-005",
			Detail: fmt.Sprintf("config parse error: config file too large (%d bytes, max %d)", fi.Size(), maxConfigFileSize),
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, &ConfigError{
				Code:   "E-CFG-004",
				Detail: fmt.Sprintf("config file not found: %s", path),
			}
		}
		return nil, &ConfigError{
			Code:   "E-CFG-004",
			Detail: fmt.Sprintf("config file not found: %s: %v", path, err),
		}
	}

	var cfg Config
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		// An empty file produces io.EOF from the decoder — that is valid YAML
		// syntax (the document is simply absent), yielding a zero-value Config.
		// All other errors are genuine parse failures → E-CFG-005.
		if !errors.Is(err, io.EOF) {
			return nil, &ConfigError{
				Code:   "E-CFG-005",
				Detail: yamlParseDetail(err),
			}
		}
	}

	return &cfg, nil
}

// yamlParseDetail formats a yaml.Decode/Unmarshal error into the canonical
// E-CFG-005 detail string: "config parse error: invalid YAML at line N: <detail>".
//
// yaml.v3 produces two distinct error shapes:
//
//  1. Syntax errors: "yaml: line N: <detail>"
//  2. Multi-error (KnownFields / type errors): "yaml: unmarshal errors:\n  line N: <detail>"
//
// We extract N from whichever shape applies and reformat into the canonical
// BC-2.09.003 EC-003 form so the operator can navigate directly to the bad line.
func yamlParseDetail(err error) string {
	// Strip control characters before embedding in the E-CFG-005 detail string.
	// yaml.v3 TypeError messages include the offending decoded value verbatim —
	// a double-quoted YAML scalar like "\e[31m" decodes to raw 0x1B bytes that
	// would survive into the error message unchanged (F-SEC-V1 / CWE-117).
	raw := stripControlChars(err.Error())

	// Shape 1: "yaml: line N: <detail>"
	const linePrefix = "yaml: line "
	if strings.HasPrefix(raw, linePrefix) {
		rest := raw[len(linePrefix):]
		colonIdx := strings.Index(rest, ":")
		if colonIdx > 0 {
			lineStr := rest[:colonIdx]
			if _, parseErr := strconv.Atoi(lineStr); parseErr == nil {
				detail := strings.TrimSpace(rest[colonIdx+1:])
				return fmt.Sprintf("config parse error: invalid YAML at line %s: %s", lineStr, detail)
			}
		}
	}

	// Shape 2: "yaml: unmarshal errors:\n  line N: <detail>[; line M: ...]"
	// KnownFields(true) and type-mismatch errors use this multi-error envelope.
	const multiPrefix = "yaml: unmarshal errors:\n"
	if strings.HasPrefix(raw, multiPrefix) {
		rest := strings.TrimPrefix(raw, multiPrefix)
		// Each sub-error is "  line N: <detail>"; take the first one.
		first := strings.SplitN(strings.TrimSpace(rest), "\n", 2)[0]
		const subPrefix = "line "
		if strings.HasPrefix(first, subPrefix) {
			sub := first[len(subPrefix):]
			colonIdx := strings.Index(sub, ":")
			if colonIdx > 0 {
				lineStr := sub[:colonIdx]
				if _, parseErr := strconv.Atoi(lineStr); parseErr == nil {
					detail := strings.TrimSpace(sub[colonIdx+1:])
					return fmt.Sprintf("config parse error: invalid YAML at line %s: %s", lineStr, detail)
				}
			}
		}
	}

	// Fallback: no line number available; still use canonical prefix.
	return fmt.Sprintf("config parse error: invalid YAML at line ?: %s", raw)
}
