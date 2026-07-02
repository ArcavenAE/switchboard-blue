// admin_interactive_prompt_test.go — tests for Phase 5 Pass 4 remediation.
//
// Covers F-A-008 (LOW): the interactive destroy-confirmation prompt must not
// contain the literal string "<short-id>" as a placeholder.
//
// Current implementation at admin.go writes a static example prompt:
//
//	_, _ = fmt.Fprint(sio.err, "Type the SVTN short-ID (e.g. SVTN-abcd1234) to confirm: ")
//
// The prompt uses a static example format rather than substituting the actual
// SVTN short-id of the specific SVTN being destroyed.  Actual short-id
// substitution requires a daemon lookup not available at the CLI layer and is
// tracked as a deferred item (DRIFT-P5P4-PROMPT-SHORTID).
//
// These tests validate the static prompt format:
//   - the prompt must NOT contain the literal "<short-id>" placeholder, and
//   - the prompt must contain the "SVTN-" prefix.
//
// IMPORTANT: DO NOT touch implementation code. This file is tests only.
package main

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

// TestNewInBurst19_InteractivePrompt_NoLiteralPlaceholder verifies that the
// interactive destroy confirmation prompt does NOT write the literal string
// "<short-id>" to stderr.
//
// Path 2 is exercised: stdinIsTTY returns true (injected), a pipe reader is
// injected into stdinReader, and we immediately write a valid-shape response
// so the prompt can complete.
//
// The test validates that the prompt format matches the expected static template
// (no literal "<short-id>" placeholder).  It does not test actual short-id
// substitution, which is deferred (DRIFT-P5P4-PROMPT-SHORTID).
func TestNewInBurst19_InteractivePrompt_NoLiteralPlaceholder(t *testing.T) {
	t.Parallel()

	// Capture stderr to inspect the prompt.
	var errBuf bytes.Buffer
	sio := sbctlIO{out: io.Discard, err: &errBuf}

	// Inject a TTY stub so Path 2 (interactive) is taken.
	origIsTTY := stdinIsTTY
	stdinIsTTY = func() bool { return true }
	t.Cleanup(func() { stdinIsTTY = origIsTTY })

	// Inject a reader that provides a valid SVTN short-ID so the prompt returns
	// success (we test the prompt text, not the validation outcome).
	pr, pw := io.Pipe()
	origReader := stdinReader
	stdinReader = pr
	t.Cleanup(func() {
		stdinReader = origReader
		pr.Close()
		pw.Close()
	})

	// Write the valid response asynchronously so the prompt doesn't block.
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = pw.Write([]byte("SVTN-abcd1234\n"))
	}()

	// Invoke with no --confirm value, --yes=false: exercises Path 2.
	err := runDestroyConfirmGate("", false, sio)

	<-done

	// The call must succeed (valid shape "SVTN-abcd1234" was provided).
	if err != nil {
		t.Fatalf("runDestroyConfirmGate(interactive): unexpected error: %v", err)
	}

	prompt := errBuf.String()

	// The prompt MUST NOT contain the literal "<short-id>" placeholder.
	if strings.Contains(prompt, "<short-id>") {
		t.Errorf("interactive prompt contains literal \"<short-id>\" placeholder; got: %q\n"+
			"  The fix must substitute the actual SVTN short-ID.", prompt)
	}
}

// TestNewInBurst19_InteractivePrompt_ContainsSVTNPrefix verifies that the
// interactive prompt contains the "SVTN-" prefix.
//
// This is a companion assertion to the NoLiteralPlaceholder test, confirming
// the prompt includes a SVTN-prefixed example even in the static format.
func TestNewInBurst19_InteractivePrompt_ContainsSVTNPrefix(t *testing.T) {
	t.Parallel()

	var errBuf bytes.Buffer
	sio := sbctlIO{out: io.Discard, err: &errBuf}

	origIsTTY := stdinIsTTY
	stdinIsTTY = func() bool { return true }
	t.Cleanup(func() { stdinIsTTY = origIsTTY })

	pr, pw := io.Pipe()
	origReader := stdinReader
	stdinReader = pr
	t.Cleanup(func() {
		stdinReader = origReader
		pr.Close()
		pw.Close()
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = pw.Write([]byte("SVTN-abcd1234\n"))
	}()

	_ = runDestroyConfirmGate("", false, sio)
	<-done

	prompt := errBuf.String()
	if !strings.Contains(prompt, "SVTN-") {
		t.Errorf("interactive prompt must contain \"SVTN-\" prefix; got: %q", prompt)
	}
}
