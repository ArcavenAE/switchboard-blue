package svtnmgmt_test

import (
	"testing"

	"github.com/arcavenae/switchboard/internal/svtnmgmt"
)

// Compile-time check: ensure the svtnmgmt package is reachable and the key
// sentinel errors are exported correctly. The tests below use panic() stubs
// until the implementer writes real assertions.
var _ = svtnmgmt.ErrSVTNAlreadyExists

// TestSVTNManager_Create_BootstrapsControlKey verifies AC-001:
// SVTNManager.Create(svtn_name) creates a new SVTN with a generated SVTN-ID
// and bootstraps the first control key locally. Returns the SVTN-ID.
//
// Traces to BC-2.07.001 postcondition 1.
func TestSVTNManager_Create_BootstrapsControlKey(t *testing.T) {
	t.Parallel()
	panic("not implemented: TestSVTNManager_Create_BootstrapsControlKey (AC-001, BC-2.07.001)")
}

// TestSVTNManager_Create_DuplicateReturnsError verifies BC-2.07.001 EC-001:
// Create with a SVTN name that already exists returns ErrSVTNAlreadyExists.
func TestSVTNManager_Create_DuplicateReturnsError(t *testing.T) {
	t.Parallel()
	panic("not implemented: TestSVTNManager_Create_DuplicateReturnsError (BC-2.07.001 EC-001)")
}

// TestSbctlAdmin_KeyRegister verifies AC-002:
// RegisterKey adds a key to the admission set; it appears in subsequent
// admission checks.
//
// Traces to BC-2.05.004 postcondition 1.
func TestSbctlAdmin_KeyRegister(t *testing.T) {
	t.Parallel()
	panic("not implemented: TestSbctlAdmin_KeyRegister (AC-002, BC-2.05.004)")
}

// TestSbctlAdmin_KeyRegister_DuplicateLastWriteWins verifies S-6.02 EC-001:
// Registering an already-registered key with a different role applies
// last-write-wins semantics (ADR-003).
func TestSbctlAdmin_KeyRegister_DuplicateLastWriteWins(t *testing.T) {
	t.Parallel()
	panic("not implemented: TestSbctlAdmin_KeyRegister_DuplicateLastWriteWins (EC-001, ADR-003)")
}

// TestSbctlAdmin_KeyRevoke verifies AC-003:
// RevokeKey removes the key from the admission set. Subsequent admission
// attempts with that key return E-ADM-002.
//
// Traces to BC-2.05.004 postcondition 2.
func TestSbctlAdmin_KeyRevoke(t *testing.T) {
	t.Parallel()
	panic("not implemented: TestSbctlAdmin_KeyRevoke (AC-003, BC-2.05.004)")
}

// TestSbctlAdmin_KeyRevoke_NotFound verifies S-6.02 EC-002:
// Revoking a key that is not registered returns ErrKeyNotRegistered (E-ADM-013).
func TestSbctlAdmin_KeyRevoke_NotFound(t *testing.T) {
	t.Parallel()
	panic("not implemented: TestSbctlAdmin_KeyRevoke_NotFound (EC-002, E-ADM-013)")
}

// TestSbctlAdmin_KeyExpire verifies AC-004:
// ExpireKey sets a TTL; after expiry, the key behaves as revoked.
//
// Traces to BC-2.05.004 postcondition 3.
func TestSbctlAdmin_KeyExpire(t *testing.T) {
	t.Parallel()
	panic("not implemented: TestSbctlAdmin_KeyExpire (AC-004, BC-2.05.004)")
}

// TestSbctlAdmin_KeyExpire_ZeroDuration verifies S-6.02 EC-003:
// ExpireKey with a zero TTL returns ErrInvalidDuration (E-CFG-001).
func TestSbctlAdmin_KeyExpire_ZeroDuration(t *testing.T) {
	t.Parallel()
	panic("not implemented: TestSbctlAdmin_KeyExpire_ZeroDuration (EC-003, E-CFG-001)")
}

// TestSbctlAdmin_ControlRevocation_RequiresConfirm verifies AC-005:
// Control-to-control revocation without confirm=true returns
// ErrControlRevocationRequiresConfirm; with confirm=true it succeeds.
//
// Traces to BC-2.05.004 invariant 1 and ADR-004.
func TestSbctlAdmin_ControlRevocation_RequiresConfirm(t *testing.T) {
	t.Parallel()
	panic("not implemented: TestSbctlAdmin_ControlRevocation_RequiresConfirm (AC-005, BC-2.05.004 Inv-1)")
}
