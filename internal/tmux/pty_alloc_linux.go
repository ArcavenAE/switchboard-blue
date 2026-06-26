//go:build linux

package tmux

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/sys/unix"
)

// defaultPTYAlloc allocates a PTY pair on Linux via /dev/ptmx + TIOCGPTN
// and spawns a shell with the slave as its controlling terminal.
//
// Uses the POSIX pseudo-terminal interface:
//  1. Open /dev/ptmx to get the master fd.
//  2. Call TIOCSPTLCK with 0 to unlock the slave.
//  3. Call TIOCGPTN to get the slave index (N), then open /dev/pts/N.
//  4. Spawn a shell with the slave as stdin/stdout/stderr and set it as
//     the process's controlling terminal via SysProcAttr.
//  5. Close the slave in the parent (child holds it via inheritance).
//  6. Return the master as the io.ReadWriteCloser.
//
// VP-032 integration harness exercises this path on real Linux hardware.
func defaultPTYAlloc() (io.ReadWriteCloser, int, error) {
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: open /dev/ptmx: %w", ErrPTYDeviceUnavailable, err)
	}

	// Unlock the slave — required before opening /dev/pts/N.
	lockVal := uint32(0)
	if err := unix.IoctlSetInt(int(master.Fd()), unix.TIOCSPTLCK, int(lockVal)); err != nil {
		_ = master.Close()
		return nil, 0, fmt.Errorf("%w: TIOCSPTLCK unlock: %w", ErrPTYDeviceUnavailable, err)
	}

	// Get the slave index.
	slaveIdx, err := unix.IoctlGetUint32(int(master.Fd()), unix.TIOCGPTN)
	if err != nil {
		_ = master.Close()
		return nil, 0, fmt.Errorf("%w: TIOCGPTN: %w", ErrPTYDeviceUnavailable, err)
	}
	slavePath := fmt.Sprintf("/dev/pts/%d", slaveIdx)

	slave, err := os.OpenFile(slavePath, os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		_ = master.Close()
		return nil, 0, fmt.Errorf("%w: open slave %s: %w", ErrPTYDeviceUnavailable, slavePath, err)
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	cmd := exec.Command(shell)
	cmd.Stdin = slave
	cmd.Stdout = slave
	cmd.Stderr = slave
	// Ctty is in the child's FD namespace. cmd.Stdin=slave, so the child sees
	// slave on FD 0. Ctty: 0 is the correct value — not slave.Fd() which is a
	// parent-process FD number.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true,
		Setctty: true,
		Ctty:    0,
	}

	if err := cmd.Start(); err != nil {
		_ = master.Close()
		_ = slave.Close()
		return nil, 0, fmt.Errorf("%w: start shell: %w", ErrPTYDeviceUnavailable, err)
	}

	// Parent does not need the slave; child inherited it via fork.
	_ = slave.Close()

	// Reap the shell when it exits to prevent zombies. After ptyMaster.Close
	// calls cmd.Process.Kill, this goroutine unblocks and completes cleanly.
	go func() { _ = cmd.Wait() }()

	return &ptyMaster{master: master, cmd: cmd}, cmd.Process.Pid, nil
}
