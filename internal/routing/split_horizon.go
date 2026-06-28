// Package routing — split-horizon loop prevention (BC-2.02.008).
//
// SplitHorizon enforces the rule that a frame is never forwarded back toward
// the interface it arrived on. This is the first-line loop prevention
// mechanism; the drop cache (BC-2.02.009, on_frame_arrival.go) is the second.
//
// Architecture constraints:
//   - The router NEVER parses channel header payload (BC-2.01.005 / VP-015).
//     Frame bytes are treated as opaque beyond the outer header.
//   - Split-horizon applies to all SVTN frame types (BC-2.02.008 invariant 1).
//   - Split-horizon is stateless per-frame (BC-2.02.008 invariant 2).
package routing

import "errors"

// ErrAllPathsSplitHorizon is returned by SplitHorizon.Forward when every
// interface in the set is the arrival interface, leaving no eligible
// output interface. Maps to E-FWD-001 in the error taxonomy.
//
// BC-2.02.008 postcondition 3: "if the only eligible interface is the
// arrival interface, the frame is dropped and an E-FWD-001 event is logged."
var ErrAllPathsSplitHorizon = errors.New("routing: split-horizon: no eligible output interface (E-FWD-001)")

// InterfaceID is the logical identifier of a network interface. Split-horizon
// operates at the interface level (BC-2.02.008): a frame received on interface
// A is never forwarded back out on interface A.
type InterfaceID uint64

// SplitHorizon enforces split-horizon forwarding for a router.
//
// A SplitHorizon value does not hold mutable state: split-horizon is
// stateless per-frame (BC-2.02.008 invariant 2). The zero value is usable.
type SplitHorizon struct{}

// ForwardResult describes the dispatch outcome for a single interface after
// split-horizon filtering.
type ForwardResult struct {
	// InterfaceID is the interface on which the frame was (or would be) forwarded.
	InterfaceID InterfaceID
	// Forwarded is true when the frame was dispatched on this interface.
	Forwarded bool
}

// ForwardFunc is the caller-supplied function that writes a frame to a
// specific output interface. SplitHorizon.Forward calls it once per eligible
// interface without holding any internal lock.
type ForwardFunc func(iface InterfaceID, frameBytes []byte) error

// Forward applies split-horizon filtering then dispatches frameBytes on
// every interface in interfaceSet except arrivalIface (BC-2.02.008
// postcondition 1 and 2).
//
// frameBytes is the raw frame including outer header and payload. The router
// treats these bytes as opaque — the channel header section is NEVER parsed
// (BC-2.01.005 / VP-015; AC-003 / FuzzSplitHorizon_ChannelHeaderOpaque).
//
// If interfaceSet is empty or all interfaces equal arrivalIface, the frame is
// dropped and ErrAllPathsSplitHorizon is returned (BC-2.02.008 postcondition 3;
// EC-001 — only one interface in set).
//
// fn is called once per eligible interface with the full frameBytes slice.
// Errors from fn are collected but do not stop iteration over remaining
// interfaces; the first non-nil error is returned alongside the results.
func (s SplitHorizon) Forward(
	frameBytes []byte,
	arrivalIface InterfaceID,
	interfaceSet []InterfaceID,
	fn ForwardFunc,
) ([]ForwardResult, error) {
	// Build the eligible output set: all interfaces except arrivalIface.
	// frameBytes is treated as opaque — never parsed here (AC-003 / VP-015).
	eligible := make([]InterfaceID, 0, len(interfaceSet))
	for _, iface := range interfaceSet {
		if iface != arrivalIface {
			eligible = append(eligible, iface)
		}
	}

	// BC-2.02.008 postcondition 3 / EC-001: no eligible interface → drop.
	if len(eligible) == 0 {
		return nil, ErrAllPathsSplitHorizon
	}

	results := make([]ForwardResult, 0, len(eligible))
	var firstErr error
	for _, iface := range eligible {
		err := fn(iface, frameBytes)
		results = append(results, ForwardResult{InterfaceID: iface, Forwarded: err == nil})
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return results, firstErr
}
