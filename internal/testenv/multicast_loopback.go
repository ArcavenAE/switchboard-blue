// multicast_loopback.go — loopback multicast interface resolution helper
// for discovery wire tests (S-BL.DISCOVERY-WIRE Decision 2(e); Task 3).
//
// Classification (ARCH-09): test helper. This package may import any
// internal package. Nothing in the production tree may import testenv.

package testenv

import (
	"net"
	"runtime"
	"testing"
)

// MulticastLoopbackInterface resolves the platform-appropriate loopback
// network interface for multicast UDP tests.
//
// net.ListenMulticastUDP works on loopback on both macOS and Linux, but the
// loopback interface name differs ("lo0" vs "lo", per this project's own B13
// lesson — platform-specific behavior requires platform-specific testing)
// and must be resolved via net.InterfaceByName.
//
// This is explicitly NOT an extension of NewLoopback (S-BL.DISCOVERY-WIRE
// Decision 2(e)): NewLoopback is a VP-042-scoped compile-shim and is not a
// fit for multicast interface resolution.
func MulticastLoopbackInterface(t testing.TB) *net.Interface {
	t.Helper()

	name := loopbackInterfaceName()
	iface, err := net.InterfaceByName(name)
	if err == nil {
		return iface
	}

	// Named lookup failed (unexpected platform, or a name this project
	// hasn't hardcoded) — fall back to scanning every interface for the
	// loopback+multicast flag combination the test needs, rather than
	// failing outright on a name mismatch alone.
	ifaces, listErr := net.Interfaces()
	if listErr != nil {
		t.Fatalf("MulticastLoopbackInterface: net.InterfaceByName(%q): %v; net.Interfaces() fallback: %v", name, err, listErr)
	}
	for i := range ifaces {
		f := ifaces[i].Flags
		if f&net.FlagLoopback != 0 && f&net.FlagMulticast != 0 {
			return &ifaces[i]
		}
	}
	t.Fatalf("MulticastLoopbackInterface: no loopback+multicast interface found (tried named lookup %q: %v)", name, err)
	return nil // unreachable — t.Fatalf stops the goroutine
}

// loopbackInterfaceName returns the platform-conventional loopback
// interface name — "lo0" on macOS, "lo" on Linux (this project's own B13
// lesson: platform-specific behavior requires platform-specific testing).
func loopbackInterfaceName() string {
	if runtime.GOOS == "darwin" {
		return "lo0"
	}
	return "lo"
}
