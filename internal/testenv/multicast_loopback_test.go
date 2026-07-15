// multicast_loopback_test.go is the self-test for MulticastLoopbackInterface
// (S-BL.DISCOVERY-WIRE Decision 2(e)), called for explicitly by the story's
// File-Change List.
package testenv_test

import (
	"net"
	"testing"

	"github.com/arcavenae/switchboard/internal/testenv"
)

// redGateGuard recovers from a not-yet-implemented stub's panic and fails
// the test cleanly (Red Gate discipline, BC-5.38.001) instead of crashing
// the whole test binary.
func redGateGuard(t *testing.T) {
	t.Helper()
	if r := recover(); r != nil {
		t.Fatalf("red gate: stub not yet implemented: %v", r)
	}
}

// TestMulticastLoopbackInterface_ResolvesLoopback verifies
// MulticastLoopbackInterface resolves a real, usable loopback network
// interface on the current platform — macOS "lo0" vs. Linux "lo" (this
// project's own B13 lesson: platform-specific behavior requires
// platform-specific testing) — via net.InterfaceByName, and that the
// resolved interface actually supports both loopback and multicast.
func TestMulticastLoopbackInterface_ResolvesLoopback(t *testing.T) {
	defer redGateGuard(t)

	iface := testenv.MulticastLoopbackInterface(t)
	if iface == nil {
		t.Fatal("MulticastLoopbackInterface returned nil")
	}
	if iface.Flags&net.FlagLoopback == 0 {
		t.Errorf("MulticastLoopbackInterface returned %q, which is not flagged as a loopback interface", iface.Name)
	}
	if iface.Flags&net.FlagMulticast == 0 {
		t.Errorf("MulticastLoopbackInterface returned %q, which does not support multicast", iface.Name)
	}
}
