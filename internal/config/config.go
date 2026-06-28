// Package config provides YAML config parsing and validation for the switchboard daemon.
// This package is a DAG root: it has no internal imports (ARCH-08 position 1).
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

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

	// UpstreamRouters lists upstream router addresses for PE mode.
	// An empty slice means E mode (BC-2.09.001).
	UpstreamRouters []string `yaml:"upstream_routers"`

	// DrainTimeout is the maximum time allowed for graceful drain (BC-2.09.002).
	// Optional; defaults applied by the daemon, not by Validate.
	DrainTimeout time.Duration `yaml:"drain_timeout"`

	// KeepaliveInterval is the node keepalive cadence (FM-009).
	// Optional; defaults applied by the daemon, not by Validate.
	KeepaliveInterval time.Duration `yaml:"keepalive_interval"`
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
// sees every problem in one pass (BC-2.09.003 postcondition 2, AC-002).
//
// Returns *ConfigError with code E-CFG-001 wrapping all ValidationErrors on failure,
// or nil on success.
func (c *Config) Validate() error {
	var failures []ValidationError

	// Required: listen_addr must be non-empty and non-whitespace.
	if strings.TrimSpace(c.ListenAddr) == "" {
		failures = append(failures, ValidationError{
			Field:      "listen_addr",
			Problem:    "required field missing",
			Suggestion: "add 'listen_addr: <ip>:<port>' to config, e.g. 'listen_addr: 0.0.0.0:9090'",
		})
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
		failures = append(failures, ValidationError{
			Field:      "tick_interval",
			Value:      val,
			Problem:    fmt.Sprintf("is outside allowed range [%s, %s]", TickIntervalMin, TickIntervalMax),
			Suggestion: fmt.Sprintf("set to a value in [%s, %s], e.g. 'tick_interval: 10ms' for interactive sessions", TickIntervalMin, TickIntervalMax),
		})
	}

	if len(failures) == 0 {
		return nil
	}

	// Build combined detail: all field errors in one message so the operator
	// sees every problem at once (AC-002 / BC-2.09.003 postcondition 2).
	var parts []string
	for _, f := range failures {
		parts = append(parts, f.Error())
	}
	return &ConfigError{
		Code:   "E-CFG-001",
		Detail: strings.Join(parts, "; "),
	}
}

// LoadFile reads and parses a YAML config file at path, returning a *Config.
//
// Returns *ConfigError with code E-CFG-004 if the file does not exist.
// Returns *ConfigError with code E-CFG-005 if the YAML is malformed.
// Does NOT call Validate — the caller is responsible for calling Validate
// after LoadFile (see ARCH-06 binding sequence).
func LoadFile(path string) (*Config, error) {
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
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, &ConfigError{
			Code:   "E-CFG-005",
			Detail: yamlParseDetail(err),
		}
	}

	return &cfg, nil
}

// yamlParseDetail formats a yaml.Unmarshal error into the canonical E-CFG-005
// detail string: "config parse error: invalid YAML at line N: <detail>".
//
// yaml.v3 encodes parse errors as strings like "yaml: line N: <detail>".
// We extract N and reformat into the canonical BC-2.09.003 EC-003 form so the
// operator can navigate directly to the bad line.
func yamlParseDetail(err error) string {
	raw := err.Error()
	// yaml.v3 format: "yaml: line N: <detail>" or "yaml: <detail>" (no line).
	const prefix = "yaml: line "
	if strings.HasPrefix(raw, prefix) {
		rest := raw[len(prefix):]
		colonIdx := strings.Index(rest, ":")
		if colonIdx > 0 {
			lineStr := rest[:colonIdx]
			if _, parseErr := strconv.Atoi(lineStr); parseErr == nil {
				detail := strings.TrimSpace(rest[colonIdx+1:])
				return fmt.Sprintf("config parse error: invalid YAML at line %s: %s", lineStr, detail)
			}
		}
	}
	// Fallback: no line number available; still use canonical prefix.
	return fmt.Sprintf("config parse error: invalid YAML at line ?: %s", raw)
}
