// Package svtnmgmttest provides test helpers for the svtnmgmt package.
// This package is imported only by _test.go files — never by production code.
// It exists as a sub-package (not a _test.go file) so that helpers can be
// called from test packages outside internal/svtnmgmt (e.g., cmd/switchboard).
package svtnmgmttest

import (
	"testing"

	"github.com/arcavenae/switchboard/internal/svtnmgmt"
)

// SeedSVTNWithoutBootstrapKey creates an SVTN record in m without registering
// the bootstrap key as control. This produces the HasAnySVTN()==true /
// BootstrapKeyHasControlRole()==false state used by the Ruling-7 mutation test
// (admin.svtn.create must check RoleControl independently of IsBootstrapKey).
//
// Test scope only — this package is imported by _test.go files.
func SeedSVTNWithoutBootstrapKey(t *testing.T, m *svtnmgmt.SVTNManager, svtnName string) {
	t.Helper()
	if err := m.InsertRawSVTN(svtnName); err != nil {
		t.Fatalf("SeedSVTNWithoutBootstrapKey: InsertRawSVTN(%q): %v", svtnName, err)
	}
}
