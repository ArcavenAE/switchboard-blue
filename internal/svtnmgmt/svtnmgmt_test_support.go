package svtnmgmt

import (
	"fmt"
	"io"
)

// SeedSVTNWithoutBootstrapKeyForTest seeds an SVTN record without registering
// the bootstrap key as control. This is a test-only method used to construct
// the "demoted bootstrap key" state for the Ruling-7 mutation test
// (admin.svtn.create must check RoleControl independently of IsBootstrapKey).
//
// DO NOT call this from production code. It intentionally violates the
// Create() invariant that the bootstrap key is always registered as RoleControl.
//
// This lives in a dedicated file (not svtnmgmt.go) to make its test-only nature
// visible at a glance. It cannot be moved to a _test.go file because it must
// be callable from cmd/switchboard/admin_handlers_test.go (cross-package).
func (m *SVTNManager) SeedSVTNWithoutBootstrapKeyForTest(svtnName string) error {
	var id [16]byte
	if _, err := io.ReadFull(m.randSource, id[:]); err != nil {
		return fmt.Errorf("generate SVTN ID for test seed: %w", err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.svtns[svtnName]; exists {
		return ErrSVTNAlreadyExists
	}
	m.svtns[svtnName] = SVTN{ID: id, Name: svtnName}
	// Intentionally does NOT call m.keySet.RegisterKey — bootstrap key is absent.
	return nil
}
