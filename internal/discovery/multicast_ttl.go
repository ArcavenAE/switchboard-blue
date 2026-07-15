package discovery

import (
	"fmt"
	"net"
	"syscall"
)

// setMulticastTTL1 sets conn's outbound IPv4 multicast TTL to 1
// (AC-003 postcondition 2; SEC-DW-08's LAN-segment-scoped delivery model —
// a TTL of 1 means the advertisement never crosses a router hop).
//
// Mechanism: stdlib net.UDPConn exposes no multicast-TTL setter, and this
// story's Library & Framework Requirements commit to zero new third-party
// dependencies (specifically ruling out golang.org/x/net/ipv4, which is the
// conventional way to reach this socket option from Go). Instead this
// reaches the underlying file descriptor via SyscallConn/RawConn.Control —
// stdlib-only — and calls setsockopt(IPPROTO_IP, IP_MULTICAST_TTL, 1)
// directly via the syscall package. Both of this project's release targets
// (darwin, linux; see justfile build-all) define IP_MULTICAST_TTL and
// IPPROTO_IP in the syscall package (darwin: 0xa, linux: 0x21 — different
// numeric values, same semantic option, both resolved by the stdlib
// per-platform zerrors_*.go constant tables at compile time), so no
// build-tag branching is needed here.
func setMulticastTTL1(conn *net.UDPConn) error {
	rawConn, err := conn.SyscallConn()
	if err != nil {
		return fmt.Errorf("obtain raw connection: %w", err)
	}

	var sockErr error
	if err := rawConn.Control(func(fd uintptr) {
		sockErr = syscall.SetsockoptByte(int(fd), syscall.IPPROTO_IP, syscall.IP_MULTICAST_TTL, 1)
	}); err != nil {
		return fmt.Errorf("control raw connection: %w", err)
	}
	if sockErr != nil {
		return fmt.Errorf("setsockopt IP_MULTICAST_TTL: %w", sockErr)
	}
	return nil
}

// setMulticastOutgoingInterface pins conn's outbound multicast traffic to
// the local IPv4 address ifaceAddr (setsockopt IP_MULTICAST_IF).
//
// Without this, the kernel selects the outgoing interface for a multicast
// destination via ordinary unicast route lookup — on a multi-homed host
// this is not necessarily the interface the intended LAN peers are
// reachable on (verified empirically: this project's dev hosts route
// 239.0.0.0/8 to their default egress interface, never to loopback, even
// when a loopback-bound listener has joined the group). sendMulticastAdvertisement
// calls this once per UP+multicast-capable local interface so the
// advertisement reaches peers regardless of which interface they're on —
// the same fan-out-on-every-interface pattern common LAN discovery
// protocols (mDNS, SSDP) use on multi-homed hosts. Explicitly stdlib-only,
// same rationale as setMulticastTTL1.
func setMulticastOutgoingInterface(conn *net.UDPConn, ifaceAddr net.IP) error {
	v4 := ifaceAddr.To4()
	if v4 == nil {
		return fmt.Errorf("interface address %v is not IPv4", ifaceAddr)
	}

	rawConn, err := conn.SyscallConn()
	if err != nil {
		return fmt.Errorf("obtain raw connection: %w", err)
	}

	var mreq [4]byte
	copy(mreq[:], v4)

	var sockErr error
	if err := rawConn.Control(func(fd uintptr) {
		sockErr = syscall.SetsockoptInet4Addr(int(fd), syscall.IPPROTO_IP, syscall.IP_MULTICAST_IF, mreq)
	}); err != nil {
		return fmt.Errorf("control raw connection: %w", err)
	}
	if sockErr != nil {
		return fmt.Errorf("setsockopt IP_MULTICAST_IF: %w", sockErr)
	}
	return nil
}
