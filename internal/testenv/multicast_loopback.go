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
// The returned interface is always verified UP+LOOPBACK+MULTICAST before
// being returned — the platform-conventional name alone is not trusted,
// because a name match does not guarantee the flags callers depend on (e.g.
// Linux's "lo" is UP+LOOPBACK but not MULTICAST on stock GitHub Actions
// runners). Callers that want to skip rather than fail when no such
// interface exists in the current environment should call
// RequireMulticastLoopback first.
//
// This is explicitly NOT an extension of NewLoopback (S-BL.DISCOVERY-WIRE
// Decision 2(e)): NewLoopback is a VP-042-scoped compile-shim and is not a
// fit for multicast interface resolution.
func MulticastLoopbackInterface(t testing.TB) *net.Interface {
	t.Helper()

	name := loopbackInterfaceName()
	if iface, err := net.InterfaceByName(name); err == nil && isLoopbackMulticastCapable(*iface) {
		return iface
	}

	// Named lookup either failed (unexpected platform, or a name this
	// project hasn't hardcoded) or found an interface missing the
	// UP+loopback+multicast flag combination (e.g. Linux "lo" without
	// IFF_MULTICAST, the default on stock GitHub Actions runners) — fall
	// back to scanning every interface for the combination the test needs,
	// rather than trusting the name match alone.
	ifaces, listErr := net.Interfaces()
	if listErr != nil {
		t.Fatalf("MulticastLoopbackInterface: named lookup %q unusable; net.Interfaces() fallback: %v", name, listErr)
	}
	for i := range ifaces {
		if isLoopbackMulticastCapable(ifaces[i]) {
			return &ifaces[i]
		}
	}
	t.Fatalf("MulticastLoopbackInterface: no UP+LOOPBACK+MULTICAST interface found (tried named lookup %q, then a full interface scan); call RequireMulticastLoopback(t) first if this environment is expected to lack one", name)
	return nil // unreachable — t.Fatalf stops the goroutine
}

// RequireMulticastLoopback skips the calling test unless the current
// environment has a usable loopback+multicast network interface (UP,
// LOOPBACK, and MULTICAST all set).
//
// Call this as the FIRST line of any test that performs real multicast
// socket I/O (net.ListenMulticastUDP, a net.DialUDP send to a multicast
// group, or any other multicast-join syscall) — before that I/O happens.
// Stock GitHub Actions Linux runners don't flag "lo" as MULTICAST-capable
// (this project's own B13 lesson: platform-specific behavior requires
// platform-specific testing), so real-socket multicast tests fail there;
// worse, the attempted egress also trips StepSecurity Harden-Runner's
// network audit on PRs. Skipping before any socket call avoids both
// failure modes. These tests still run end-to-end on capable environments
// (developer workstations, and a future network-integration CI tier).
func RequireMulticastLoopback(t testing.TB) {
	t.Helper()

	if hasMulticastLoopback() {
		return
	}
	t.Skip("no UP+LOOPBACK+MULTICAST interface in this environment (e.g. stock GitHub Linux runners, where \"lo\" lacks the MULTICAST flag); real-socket discovery tests run on developer workstations and a network-integration CI tier")
}

// hasMulticastLoopback reports whether any interface on the host is
// simultaneously UP, LOOPBACK, and MULTICAST. It never fails the test —
// callers translate a false result into either a skip
// (RequireMulticastLoopback) or a fatal error (MulticastLoopbackInterface),
// depending on whether the absence of such an interface is an expected
// environment limitation or an unexpected setup problem.
func hasMulticastLoopback() bool {
	ifaces, err := net.Interfaces()
	if err != nil {
		return false
	}
	for i := range ifaces {
		if isLoopbackMulticastCapable(ifaces[i]) {
			return true
		}
	}
	return false
}

// isLoopbackMulticastCapable reports whether iface is simultaneously UP,
// LOOPBACK, and MULTICAST — the flag combination every real-socket
// multicast test in this project requires of its loopback interface.
func isLoopbackMulticastCapable(iface net.Interface) bool {
	const want = net.FlagUp | net.FlagLoopback | net.FlagMulticast
	return iface.Flags&want == want
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
