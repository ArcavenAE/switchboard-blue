// write_success_test.go — S502-DEFER-2 contract tests for writeSuccess.
//
// writeSuccess must return an error rather than terminating the process
// with os.Exit(3), so the go.md rule "no os.Exit outside main()" holds and
// so callers stay composable. The exit-3 mapping now lives in main() via
// the *internalError sentinel wrapper, alongside the *usageError → exit-2
// and *reportedError → skip-reprint mappings.
//
// RED (S502-DEFER-2): current impl calls os.Exit(3) inline; test can't run.
// GREEN: writeSuccess returns *internalError on json.Marshal failure; all
// call sites propagate; main() maps *internalError → exit 3.
//
// Package main (internal test file) for access to writeSuccess,
// internalError, and sbctlIO.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// TestWriteSuccess_MarshalFailure_ReturnsInternalErrorAndDoesNotExit verifies
// the S502-DEFER-2 contract: on json.Marshal failure of the success envelope,
// writeSuccess must return an *internalError sentinel and must not call
// os.Exit.  The sentinel is the exit-3 signal for main().
//
// Failure mode: pass an invalid json.RawMessage (unterminated object) — the
// MarshalJSON method on RawMessage rejects malformed bytes.
//
// Refs: S502-DEFER-2, go.md "no os.Exit outside main()".
func TestWriteSuccess_MarshalFailure_ReturnsInternalErrorAndDoesNotExit(t *testing.T) {
	t.Parallel()

	// Invalid RawMessage: RawMessage.MarshalJSON validates by calling
	// json.Compact, which errors on malformed input.
	bad := json.RawMessage([]byte("{not-json"))

	var outBuf, errBuf bytes.Buffer
	sio := sbctlIO{out: &outBuf, err: &errBuf}

	err := writeSuccess(true, bad, sio)

	if err == nil {
		t.Fatal("S502-DEFER-2: writeSuccess must return error on marshal failure; got nil")
	}
	var ie *internalError
	if !errors.As(err, &ie) {
		t.Errorf("S502-DEFER-2: writeSuccess must return *internalError sentinel "+
			"for the exit-3 mapping; got %T: %v", err, err)
	}
	// Optional but stable: stderr should still carry the diagnostic line so
	// operators see WHY the envelope couldn't be emitted (parity with old
	// os.Exit(3) behaviour which printed to stderr first).
	if !strings.Contains(errBuf.String(), "marshal") {
		t.Errorf("S502-DEFER-2: stderr should include a marshal diagnostic; got %q",
			errBuf.String())
	}
}

// TestWriteSuccess_ValidJSON_ReturnsNilAndWritesEnvelope verifies the happy
// path — a valid JSON payload is wrapped in the success envelope and written
// to sio.out, and writeSuccess returns nil.
func TestWriteSuccess_ValidJSON_ReturnsNilAndWritesEnvelope(t *testing.T) {
	t.Parallel()

	good := json.RawMessage([]byte(`{"foo":"bar"}`))

	var outBuf, errBuf bytes.Buffer
	sio := sbctlIO{out: &outBuf, err: &errBuf}

	if err := writeSuccess(true, good, sio); err != nil {
		t.Fatalf("writeSuccess returned non-nil for valid JSON: %v", err)
	}
	if errBuf.Len() != 0 {
		t.Errorf("stderr must be empty on success; got %q", errBuf.String())
	}

	// stdout should contain a single JSON envelope with ok:true and the raw data.
	var env struct {
		OK   bool            `json:"ok"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(outBuf.Bytes()), &env); err != nil {
		t.Fatalf("stdout is not a valid JSON envelope: %v\nraw: %q", err, outBuf.String())
	}
	if !env.OK {
		t.Error("envelope ok field must be true on success path")
	}
	if !bytes.Equal(env.Data, good) {
		t.Errorf("envelope data mismatch: want %s got %s", good, env.Data)
	}
}

// TestWriteSuccess_PlainText_ReturnsNilAndWritesRaw verifies that when
// --json is false, writeSuccess writes the raw data bytes directly (no
// envelope) and returns nil.
func TestWriteSuccess_PlainText_ReturnsNilAndWritesRaw(t *testing.T) {
	t.Parallel()

	// Plain-text branch never marshals — data is emitted verbatim.
	// Even a malformed json.RawMessage should be written unchanged.
	raw := json.RawMessage([]byte("hello world\n"))

	var outBuf, errBuf bytes.Buffer
	sio := sbctlIO{out: &outBuf, err: &errBuf}

	if err := writeSuccess(false, raw, sio); err != nil {
		t.Fatalf("writeSuccess(useJSON=false) returned non-nil: %v", err)
	}
	if errBuf.Len() != 0 {
		t.Errorf("stderr must be empty on plain-text success; got %q", errBuf.String())
	}
	if !strings.HasPrefix(outBuf.String(), "hello world") {
		t.Errorf("stdout should carry raw data; got %q", outBuf.String())
	}
}
