// io.go defines sbctlIO, the explicit output-sink pair threaded through all
// run functions. Using an explicit struct instead of package-level globals
// prevents data races when tests run with t.Parallel() and -race.
//
// Purity classification (ARCH-09): effectful-boundary — wraps OS output sinks.
package main

import (
	"io"
	"os"
)

// sbctlIO bundles the stdout and stderr writers for a single sbctl invocation.
// Callers pass this through run functions instead of reading package-level globals.
type sbctlIO struct {
	out io.Writer
	err io.Writer
}

// defaultIO returns sbctlIO wired to the real OS file descriptors.
// Production callers (main) use this; tests supply their own buffer-backed pair.
func defaultIO() sbctlIO {
	return sbctlIO{out: os.Stdout, err: os.Stderr}
}
