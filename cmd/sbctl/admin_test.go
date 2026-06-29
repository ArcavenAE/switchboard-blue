package main

import (
	"testing"
)

// TestSbctlAdmin_KeyRegister_CLI verifies AC-002 at the CLI layer:
// `sbctl admin key register --key <pubkey> --svtn <id>` sends
// admin.key.register to the daemon and confirms the key appears in subsequent
// admission checks.
//
// Uses an in-process fake mgmt.Server to avoid requiring a live daemon.
// Traces to BC-2.05.004 postcondition 1.
func TestSbctlAdmin_KeyRegister_CLI(t *testing.T) {
	t.Parallel()
	panic("not implemented: TestSbctlAdmin_KeyRegister_CLI (AC-002, BC-2.05.004)")
}

// TestSbctlAdmin_KeyRevoke_CLI verifies AC-003 at the CLI layer:
// `sbctl admin key revoke --key <pubkey> --svtn <id>` sends
// admin.key.revoke to the daemon and the key no longer appears in admission.
//
// Traces to BC-2.05.004 postcondition 2.
func TestSbctlAdmin_KeyRevoke_CLI(t *testing.T) {
	t.Parallel()
	panic("not implemented: TestSbctlAdmin_KeyRevoke_CLI (AC-003, BC-2.05.004)")
}

// TestSbctlAdmin_KeyExpire_CLI verifies AC-004 at the CLI layer:
// `sbctl admin key expire --key <pubkey> --svtn <id> --after <duration>`
// sends admin.key.expire to the daemon.
//
// Traces to BC-2.05.004 postcondition 3.
func TestSbctlAdmin_KeyExpire_CLI(t *testing.T) {
	t.Parallel()
	panic("not implemented: TestSbctlAdmin_KeyExpire_CLI (AC-004, BC-2.05.004)")
}

// TestSbctlAdmin_ControlRevocation_RequiresConfirm_CLI verifies AC-005 at the
// CLI layer: `sbctl admin key revoke` without --confirm when the target key
// is a control key returns E-ADM-004; with --confirm it succeeds.
//
// Traces to BC-2.05.004 invariant 1 and ADR-004.
func TestSbctlAdmin_ControlRevocation_RequiresConfirm_CLI(t *testing.T) {
	t.Parallel()
	panic("not implemented: TestSbctlAdmin_ControlRevocation_RequiresConfirm_CLI (AC-005, ADR-004)")
}

// TestSbctlAdmin_KeyRegister_UnknownSubcommand verifies that supplying an
// unknown subcommand to `sbctl admin` exits with code 2 and a usage hint.
func TestSbctlAdmin_KeyRegister_UnknownSubcommand(t *testing.T) {
	t.Parallel()
	panic("not implemented: TestSbctlAdmin_KeyRegister_UnknownSubcommand")
}
