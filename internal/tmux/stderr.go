package tmux

import "regexp"

// flagRejectionRE matches tmux's flag-rejection error messages that indicate
// the -C (control mode) flag is not supported. The pattern anchors to the
// "option" keyword context to prevent false matches on usage banners (which
// may contain "[-C]") or session names that happen to contain "-C".
//
// Matches: "unknown option -- C", "invalid option -- -C", "invalid option -- 'C'"
// Does not match: "usage: tmux [-C] [-f file]", "session not found: -C-test"
var flagRejectionRE = regexp.MustCompile(`(?i)(unknown|invalid|illegal) option[^\n]*\bC\b`)

// ClassifyStderr inspects captured tmux stderr output after cmd exit and returns
// a sentinel error if the output matches a known failure pattern. Returns nil
// if no pattern matches (caller should wrap the cmd.Wait error normally).
//
// Recognized patterns (BC-2.04.001 EC-001; ADR-010):
//   - "unknown option -- C" / "invalid option -- -C" / "invalid option -- 'C'"
//     → ErrControlModeUnsupportedFlag
//
// The regex is anchored to the "option" keyword so that usage banners
// (e.g. "tmux [-C] [-f file]") and session names (e.g. "-C-test") do not
// produce false positives.
//
// This function is pure: it has no side effects and its output depends only on
// its input. Exported for unit testing from the tmux_test package; production
// callers use defaultExecFn which calls this internally.
func ClassifyStderr(captured string) error {
	if flagRejectionRE.MatchString(captured) {
		return ErrControlModeUnsupportedFlag
	}
	return nil
}
