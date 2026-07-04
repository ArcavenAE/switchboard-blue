//go:build darwin

package tmux

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"unsafe"
)

// defaultPTYAlloc allocates a PTY pair on macOS via /dev/ptmx + TIOCPTYGNAME
// and spawns a shell with the slave as its controlling terminal.
//
// Uses the POSIX pseudo-terminal interface:
//  1. Open /dev/ptmx to get the master fd.
//  2. Call TIOCPTYGNAME to retrieve the slave device path.
//  3. Open the slave device.
//  4. Spawn a shell with the slave as stdin/stdout/stderr and set it as
//     the process's controlling terminal via SysProcAttr.
//  5. Close the slave in the parent (child holds it via inheritance).
//  6. Return the master as the io.ReadWriteCloser.
//
// VP-032 integration harness exercises this path on real macOS hardware.
func defaultPTYAlloc() (io.ReadWriteCloser, int, error) {
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: open /dev/ptmx: %w", ErrPTYDeviceUnavailable, err)
	}

	// TIOCPTYGNAME returns the slave device path (e.g. "/dev/ttys003").
	// The buffer must be at least 128 bytes per the macOS ioctl contract.
	// Use syscall.Syscall from stdlib — the standard ioctl path on darwin.
	var slaveName [128]byte
	if _, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		master.Fd(),
		syscall.TIOCPTYGNAME,
		uintptr(unsafe.Pointer(&slaveName[0])),
	); errno != 0 {
		_ = master.Close()
		return nil, 0, fmt.Errorf("%w: TIOCPTYGNAME: %w", ErrPTYDeviceUnavailable, errno)
	}

	// Find the NUL terminator to convert [128]byte → string.
	slaveNameStr := ""
	for i, b := range slaveName {
		if b == 0 {
			slaveNameStr = string(slaveName[:i])
			break
		}
	}
	if slaveNameStr == "" {
		_ = master.Close()
		return nil, 0, fmt.Errorf("%w: TIOCPTYGNAME returned empty name", ErrPTYDeviceUnavailable)
	}

	slave, err := os.OpenFile(slaveNameStr, os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		_ = master.Close()
		return nil, 0, fmt.Errorf("%w: open slave %s: %w", ErrPTYDeviceUnavailable, slaveNameStr, err)
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
