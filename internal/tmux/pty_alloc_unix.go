//go:build darwin || linux

package tmux

import (
	"os"
	"os/exec"
)

// ptyMaster wraps the master FD with the child shell process for lifecycle
// management. Close kills the child shell explicitly before closing the FD,
// preventing orphaned shell processes when the PTY proxy is torn down.
type ptyMaster struct {
	master *os.File
	cmd    *exec.Cmd
}

func (p *ptyMaster) Read(b []byte) (int, error)  { return p.master.Read(b) }
func (p *ptyMaster) Write(b []byte) (int, error) { return p.master.Write(b) }

// Close kills the child shell process and closes the master FD. The reaper
// goroutine (started in defaultPTYAlloc) handles cmd.Wait after Kill returns.
func (p *ptyMaster) Close() error {
	if p.cmd != nil && p.cmd.Process != nil {
		// Kill the child shell explicitly to prevent orphans.
		_ = p.cmd.Process.Kill()
	}
	return p.master.Close()
}
