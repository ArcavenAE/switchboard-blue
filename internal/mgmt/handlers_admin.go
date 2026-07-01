// handlers_admin.go — admin RPC handler types and helpers for the management
// plane (internal/mgmt package).
//
// This file defines the SVTNCreator interface and AdminSVTNCreateResult type
// so that the management plane can dispatch admin.svtn.create RPCs without
// importing internal/svtnmgmt directly (ARCH-08 §6.6 / ARCH-12 §Package DAG
// Constraints: internal/mgmt MUST NOT import internal/svtnmgmt).
//
// The concrete wiring — binding *svtnmgmt.SVTNManager to the interface and
// registering the handler in BuildAdminHandlers — lives in
// cmd/switchboard/admin_handlers.go, which is permitted to import both packages.
//
// Purity classification (ARCH-09): boundary — defines handler interface seam
// only; no business logic.
//
// S-6.07: admin.svtn.create handler + CLI subcommand (BC-2.07.001 PC-1 + Inv-3).
package mgmt

import (
	"context"
	"encoding/json"
)

// SVTNCreator is the interface through which admin.svtn.create dispatches SVTN
// creation. *svtnmgmt.SVTNManager satisfies this interface.
//
// Using an interface here keeps internal/mgmt free of the internal/svtnmgmt
// import (ARCH-12 §Package DAG Constraints). The concrete binding is done by
// cmd/switchboard/admin_handlers.go at wiring time.
//
// S-6.07 BC-2.07.001 PC-1 (Create handler RPC-reachability gap closure).
type SVTNCreator interface {
	// Create creates a new SVTN with the given name. Returns an error if the
	// SVTN already exists (ErrSVTNAlreadyExists / E-SVTN-001) or if name is
	// empty. The caller is responsible for mapping sentinel errors to wire
	// error codes.
	//
	// The returned SVTNCreateResult carries the SVTN ID (hex-encoded) and
	// the bootstrap fingerprint (SHA256:<base64> canonical format per
	// BC-2.05.004 PC-4 / BC-2.07.001 PC-2).
	Create(svtnName string) (SVTNCreateResult, error)
}

// SVTNCreateResult is the result returned by SVTNCreator.Create. It carries
// the fields required for the AC-004 success response.
//
// SVTNID is the hex-encoded 16-byte SVTN identifier (BC-2.07.001 postcondition 1).
// BootstrapFingerprint is the SHA256:<base64> fingerprint of the bootstrap
// control key registered at creation time (BC-2.07.001 postcondition 2;
// BC-2.05.004 PC-4 canonical format — do NOT re-encode to hex).
type SVTNCreateResult struct {
	// SVTNID is the hex-encoded SVTN identifier.
	SVTNID string
	// BootstrapFingerprint is the "SHA256:<base64>" fingerprint of the bootstrap
	// control key. Verbatim from svtnmgmt.keyFingerprint output.
	BootstrapFingerprint string
}

// adminSVTNCreateArgs is the wire-format JSON args for admin.svtn.create.
//
// Name is the human-readable SVTN label provided by the operator via
// `sbctl admin svtn create --name=<name>` (AC-002 / BC-2.07.001 PC-1).
type adminSVTNCreateArgs struct {
	// Name is the SVTN name supplied by the operator.
	Name string `json:"name"`
}

// adminSVTNCreateResponse is the success data payload for the admin.svtn.create
// response (AC-004 / BC-2.07.001 PC-1 + PC-2).
//
// JSON field names match AC-004 wire contract: svtn_id and bootstrap_fingerprint.
type adminSVTNCreateResponse struct {
	// SVTNID is the hex-encoded SVTN identifier.
	SVTNID string `json:"svtn_id"`
	// BootstrapFingerprint is the "SHA256:<base64>" fingerprint of the bootstrap
	// control key (BC-2.05.004 PC-4 canonical format).
	BootstrapFingerprint string `json:"bootstrap_fingerprint"`
}

// MakeAdminSVTNCreateHandler returns the admin.svtn.create handler function
// for use in BuildAdminHandlers. creator must not be nil.
//
// Authority check (BC-2.07.001 Inv-3 / AC-003): the handler reads the
// authenticated caller's pubkey from ctx (set by handleConnection after a
// successful ADR-012 challenge-response handshake), resolves its role via
// the provided roleChecker, and rejects non-control-role callers with E-ADM-009
// before invoking creator.Create. The request is NEVER dispatched to
// creator.Create if the caller does not hold control authority.
//
// The roleChecker is injected separately so that the mgmt package can perform
// the role gate without importing svtnmgmt (ARCH-12 §Package DAG). The
// concrete binding uses resolveAndVerifyCallerRole from cmd/switchboard.
//
// Traces to BC-2.07.001 PC-1 + PC-2 + Inv-3; AC-001; AC-003; AC-004; AC-005.
func MakeAdminSVTNCreateHandler(
	creator SVTNCreator,
	roleChecker func(ctx context.Context, name string) error,
) func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		panic("TODO: S-6.07 MakeAdminSVTNCreateHandler not yet implemented")
	}
}
