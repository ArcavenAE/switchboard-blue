package tmux

import "strings"

// ClassifyStderr inspects captured tmux stderr output after cmd exit and returns
// a sentinel error if the output matches a known failure pattern. Returns nil
// if no pattern matches (caller should wrap the cmd.Wait error normally).
//
// Recognized patterns (BC-2.04.001 EC-001; ADR-010):
//   - "-C" / "unknown option -- C" / "invalid option" → ErrControlModeUnsupportedFlag
//
// This function is pure: it has no side effects and its output depends only on
// its input. Exported for unit testing from the tmux_test package; production
// callers use defaultExecFn which calls this internally.
func ClassifyStderr(captured string) error {
	if strings.Contains(captured, "-C") ||
		strings.Contains(captured, "unknown option") ||
		strings.Contains(captured, "invalid option") {
		return ErrControlModeUnsupportedFlag
	}
	return nil
}
