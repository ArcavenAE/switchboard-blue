//go:build integration

// Package svtnmgmt_test VP-046 e2e test.
//
// TestIntegration_KeyLifecycle discharges VP-046 (key lifecycle:
// register/revoke/expire) using the testenv rig.
//
// VP-046 was PARTIAL: the three key lifecycle properties (register, revoke,
// expire) were discharged by unit tests against real SVTNManager + admission.
// The gap was the end-to-end `env.ConnectWithKey` wrapper that exercises the
// full admission handshake.  This file closes that gap.
//
// Traces to: VP-046, BC-2.05.004
package svtnmgmt_test

import (
	"context"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/testenv"
)

// TestIntegration_KeyLifecycle verifies register, revoke, and expiry semantics
// via the full admission handshake.
//
// Traces to: VP-046, BC-2.05.004
// Sub-properties: (1) registered→admitted; (2) revoked→rejected; (3) expired→rejected.
func TestIntegration_KeyLifecycle(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	env := testenv.New(ctx, t)
	t.Cleanup(env.Close)

	// (1) Registered key is admitted.
	t.Run("registered key admitted", func(t *testing.T) {
		t.Parallel()
		env2 := testenv.New(ctx, t)
		key := env2.GenerateKey(t)
		env2.RegisterKey(t, key)
		if err := env2.ConnectWithKey(t, key); err != nil {
			t.Errorf("expected admission with registered key, got: %v", err)
		}
	})

	// (2) Revoked key is rejected.
	t.Run("revoked key rejected", func(t *testing.T) {
		t.Parallel()
		env2 := testenv.New(ctx, t)
		key := env2.GenerateKey(t)
		env2.RegisterKey(t, key)
		env2.RevokeKey(t, key)
		if err := env2.ConnectWithKey(t, key); err == nil {
			t.Error("expected rejection for revoked key, but connection succeeded")
		}
	})

	// (3) Expired key is rejected after expiry.
	t.Run("expired key rejected after expiry", func(t *testing.T) {
		t.Parallel()
		env2 := testenv.New(ctx, t)
		key := env2.GenerateKeyWithExpiry(t, time.Now().Add(200*time.Millisecond))
		env2.RegisterKey(t, key)

		// Must be admitted before expiry.
		if err := env2.ConnectWithKey(t, key); err != nil {
			t.Errorf("expected admission before expiry, got: %v", err)
		}

		// Wait for expiry.
		time.Sleep(300 * time.Millisecond)

		// Must be rejected after expiry.
		if err := env2.ConnectWithKey(t, key); err == nil {
			t.Error("expected rejection after key expiry, but connection succeeded")
		}
	})
}
