// admin_interactive_prompt_test.go — RED tests for Phase 5 Pass 4 remediation.
//
// Covers F-A-008 (LOW): interactive destroy-confirmation prompt contains the
// literal string "<short-id>" instead of substituting the actual SVTN short-ID.
//
// Current code at admin.go:335:
//
//	_, _ = fmt.Fprint(sio.err, "Type SVTN-<short-id> to confirm: ")
//
// The fix should substitute the actual short-id of the SVTN being destroyed.
// After the fix the prompt must NOT contain "<short-id>" and MUST contain a
// "SVTN-" prefix followed by 8 lowercase hex characters.
//
// Note: the current confirm-gate API (runDestroyConfirmGate) does not accept the
// SVTN name/id as a parameter, so the test exercises the observable prompt text
// written to sio.err and asserts the absence of the literal placeholder.
//
// MUST FAIL with current code because the prompt writes "<short-id>" literally.
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
// MUST FAIL with current code which writes "<short-id>" literally.
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
	// FAILS with current code which writes "Type SVTN-<short-id> to confirm: ".
	if strings.Contains(prompt, "<short-id>") {
		t.Errorf("interactive prompt contains literal \"<short-id>\" placeholder; got: %q\n"+
			"  The fix must substitute the actual SVTN short-ID.", prompt)
	}
}

// TestNewInBurst19_InteractivePrompt_ContainsSVTNPrefix verifies that the
// interactive prompt contains the "SVTN-" prefix (confirming the format
// exists, even if the short-ID substitution needs implementation).
//
// This is a companion assertion to the NoLiteralPlaceholder test.
// Both FAIL with current code (one because it has the placeholder;
// this one passes vacuously if the prompt contains "SVTN-<short-id>"
// — but we add it so that after the fix, both pass).
//
// GREEN guard test: this PASSES with current code (the prompt contains
// "SVTN-") but documents the expected format for post-fix verification.
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
