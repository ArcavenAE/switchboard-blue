package tmux_test

import (
	"errors"
	"testing"

	"github.com/arcavenae/switchboard/internal/tmux"
)

// TestClassifyStderr verifies that classifyStderr correctly identifies tmux
// flag-rejection patterns in stderr output and returns the appropriate sentinel.
//
// classifyStderr is pure: no side effects; output depends only on input.
// These tests exercise each recognized pattern in isolation, plus the
// no-match case (returns nil).
//
// Traces: BC-2.04.001 EC-001 (old tmux -C flag rejection); pass-4 M-004.
func TestClassifyStderr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		captured string
		wantErr  error
	}{
		{
			name:     "empty stderr — no classification",
			captured: "",
			wantErr:  nil,
		},
		{
			name:     "unrelated output — no classification",
			captured: "starting server on /tmp/tmux-1000/default",
			wantErr:  nil,
		},
		{
			name:     "dash-C flag in output",
			captured: "tmux: invalid option -- -C",
			wantErr:  tmux.ErrControlModeUnsupportedFlag,
		},
		{
			name:     "unknown option pattern",
			captured: "tmux: unknown option -- C",
			wantErr:  tmux.ErrControlModeUnsupportedFlag,
		},
		{
			name:     "invalid option pattern",
			captured: "tmux: invalid option -- 'C'",
			wantErr:  tmux.ErrControlModeUnsupportedFlag,
		},
		{
			// Usage banners contain "[-C]" but must NOT trigger a false positive:
			// the regex is anchored to the "option" keyword context.
			name:     "usage banner with [-C] — no match",
			captured: "usage: tmux [-2CDlLNuvV] [-C] ...",
			wantErr:  nil,
		},
		{
			// Session names containing "-C" must NOT match.
			name:     "session name containing -C — no match",
			captured: "session not found: -C-test",
			wantErr:  nil,
		},
		{
			// Generic "invalid option" without the C flag must NOT match.
			name:     "invalid option without C flag — no match",
			captured: "tmux 1.8\ninvalid option passed\nusage: tmux",
			wantErr:  nil,
		},
		{
			name:     "error unrelated to flags",
			captured: "failed to connect to server",
			wantErr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tmux.ClassifyStderr(tt.captured)
			if tt.wantErr == nil {
				if got != nil {
					t.Errorf("classifyStderr(%q) = %v; want nil", tt.captured, got)
				}
				return
			}
			if got == nil {
				t.Errorf("classifyStderr(%q) = nil; want %v", tt.captured, tt.wantErr)
				return
			}
			if !errors.Is(got, tt.wantErr) {
				t.Errorf("classifyStderr(%q) = %v; want errors.Is(_, %v)", tt.captured, got, tt.wantErr)
			}
		})
	}
}
