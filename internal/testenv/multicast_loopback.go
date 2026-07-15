// multicast_loopback.go — loopback multicast interface resolution helper
// for discovery wire tests (S-BL.DISCOVERY-WIRE Decision 2(e); Task 3).
//
// Classification (ARCH-09): test helper. This package may import any
// internal package. Nothing in the production tree may import testenv.

package testenv

import (
	"net"
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
//
// STUB — S-BL.DISCOVERY-WIRE (Red Gate, BC-5.38.001). Not yet implemented;
// body panics unconditionally so no test can accidentally pass before
// Task 3's Green step.
func MulticastLoopbackInterface(t testing.TB) *net.Interface {
	t.Helper()
	panic("not implemented: S-BL.DISCOVERY-WIRE MulticastLoopbackInterface")
}
