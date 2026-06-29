// testmain_test.go — package-level test setup for cmd/switchboard.
//
// TestMain resets the process umask to 0022 before running the suite.
// listenUnixMgmt temporarily sets umask to 0177 inside umaskMu for the bind(2)
// syscall. When internal/mgmt tests run in a concurrent OS process (parallel
// package execution, default -p=GOMAXPROCS), a listenUnixMgmt call from this
// package can leave the umask at 0177 if the test binary is killed between
// the Umask(0177) and Umask(old) calls — rare but possible. More commonly,
// MkdirTemp called between Umask(0177) and Umask(old) in another goroutine
// receives a 0600 directory (no execute), causing WriteFile in subtests to
// fail with EPERM before the test body can exercise its assertions.
//
// Resetting to 0022 here is a belt-and-suspenders measure: umaskMu already
// serialises the critical section inside listenUnixMgmt; this reset eliminates
// the residual window where MkdirTemp races the restore.
package main

import (
	"os"
	"syscall"
	"testing"
)

func TestMain(m *testing.M) {
	// Reset process umask to 0022 before the suite runs.
	// This eliminates a rare race where a concurrent listenUnixMgmt call
	// (which sets umask=0177 inside umaskMu) coincides with MkdirTemp in a
	// test goroutine, yielding a 0600 (non-executable) directory that blocks
	// subsequent WriteFile calls in test setup (AC-019 / Ruling O).
	syscall.Umask(0o022)
	os.Exit(m.Run())
}
