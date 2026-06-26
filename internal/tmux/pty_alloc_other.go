//go:build !darwin && !linux

package tmux

import (
	"fmt"
	"io"
)

// defaultPTYAlloc is the unsupported-platform stub. PTY allocation is only
// implemented for darwin and linux (pty_alloc_darwin.go, pty_alloc_linux.go).
// On other platforms, returns ErrPTYDeviceUnavailable at runtime.
//
// Unit tests inject WithPTYAllocFunc and never reach this path.
func defaultPTYAlloc() (io.ReadWriteCloser, int, error) {
	return nil, 0, fmt.Errorf("%w: PTY allocation not implemented for this platform", ErrPTYDeviceUnavailable)
}
