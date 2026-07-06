// routing_pathtrackers_test.go — Router forwarding-entry hook contract
// (S-BL.PATH-TRACKER-WIRING; S-BL.PATH-TRACKER-WRITER folded).
//
// BC-2.06.003 PC-1 requires paths.list to enumerate live per-path metrics from
// the routing subsystem. S-W5.04 shipped the handler surface and the
// PathsListSource adapter interface; production population was deferred to
// S-BL.PATH-TRACKER-WIRING per wave-6-tranche-a-scope-rulings Ruling-6.
//
// ARCH-08 §6 forbids `internal/routing` from importing `internal/paths`
// (routing is DAG position 5; paths is 8). The registry itself therefore lives
// in cmd/switchboard (the sole package that already sits above both). Router
// exposes only a typed hook signature and calls the hook under its own lock —
// the caller (cmd/switchboard) supplies the PathTracker registry side.
//
// The concurrent-writer sanction from S-BL.PATH-TRACKER-WRITER folds in here
// (Ruling-11): the hook fires under Router's write lock, and the writer's
// registry maintains its own mutex — the composition is race-clean.

package routing_test

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/hmac"
	"github.com/arcavenae/switchboard/internal/routing"
)

// TestRouter_RegisterForwardingEntry_FiresHook verifies that the hook installed
// via WithForwardingEntryHook is called exactly once per RegisterForwardingEntry
// with the (svtnID, nodeAddr) pair the caller supplied.
//
// S-BL.PATH-TRACKER-WIRING AC-1; BC-2.06.003 PC-1.
func TestRouter_RegisterForwardingEntry_FiresHook(t *testing.T) {
	t.Parallel()

	type registered struct {
		svtnID   [16]byte
		nodeAddr [8]byte
	}
	var got []registered
	var mu sync.Mutex
	hook := func(svtnID [16]byte, nodeAddr [8]byte) {
		mu.Lock()
		defer mu.Unlock()
		got = append(got, registered{svtnID, nodeAddr})
	}

	r := routing.NewRouter(admission.NewAdmittedKeySet(), routing.WithForwardingEntryHook(hook))

	var svtnID [16]byte
	copy(svtnID[:], []byte("svtn-alpha-00000"))
	var nodeAddr [8]byte
	copy(nodeAddr[:], []byte("node-001"))
	var authKey [hmac.KeySize]byte
	for i := range authKey {
		authKey[i] = byte(i)
	}

	r.RegisterForwardingEntry(svtnID, nodeAddr, authKey)

	mu.Lock()
	defer mu.Unlock()
	if len(got) != 1 {
		t.Fatalf("hook fired %d times; want 1", len(got))
	}
	if got[0].svtnID != svtnID || got[0].nodeAddr != nodeAddr {
		t.Errorf("hook received (svtnID=%x, nodeAddr=%x); want (%x, %x)",
			got[0].svtnID, got[0].nodeAddr, svtnID, nodeAddr)
	}
}

// TestRouter_RegisterForwardingEntry_HookFiresOnReplace verifies that repeated
// registrations for the same (svtnID, nodeAddr) each fire the hook. The
// registry side (cmd/switchboard pathTrackerSource) uses this signal to
// determine whether a tracker exists — repeated calls MUST NOT construct a
// new tracker each time (see cmd/switchboard tests) but the hook itself is a
// pure notification, so it fires per registration.
//
// This test does NOT assert LWW at the hook level — LWW is enforced on the
// forwarding table itself (see W3-R2-M2 tests) and on tracker identity on the
// pathTrackerSource side. The hook is a fire-and-forget notification.
//
// S-BL.PATH-TRACKER-WIRING AC-4.
func TestRouter_RegisterForwardingEntry_HookFiresOnReplace(t *testing.T) {
	t.Parallel()

	var fires int64
	hook := func(_ [16]byte, _ [8]byte) {
		atomic.AddInt64(&fires, 1)
	}

	r := routing.NewRouter(admission.NewAdmittedKeySet(), routing.WithForwardingEntryHook(hook))

	var svtnID [16]byte
	var nodeAddr [8]byte
	var k1, k2 [hmac.KeySize]byte
	k2[0] = 0xff

	r.RegisterForwardingEntry(svtnID, nodeAddr, k1)
	r.RegisterForwardingEntry(svtnID, nodeAddr, k2)

	if got := atomic.LoadInt64(&fires); got != 2 {
		t.Errorf("hook fires after two RegisterForwardingEntry calls: got %d; want 2", got)
	}
}

// TestRouter_WithoutHook_RegisterForwardingEntryStillWorks verifies backwards
// compatibility: RegisterForwardingEntry MUST NOT panic or otherwise misbehave
// when no hook is installed. Existing call sites (test doubles, non-metrics
// paths) continue to work unchanged.
//
// S-BL.PATH-TRACKER-WIRING AC-2 (backwards compat).
func TestRouter_WithoutHook_RegisterForwardingEntryStillWorks(t *testing.T) {
	t.Parallel()

	r := routing.NewRouter(admission.NewAdmittedKeySet())

	var svtnID [16]byte
	var nodeAddr [8]byte
	var authKey [hmac.KeySize]byte

	// Must not panic.
	r.RegisterForwardingEntry(svtnID, nodeAddr, authKey)
}

// TestRouter_ConcurrentRegisterForwardingEntry_HookRaceClean exercises the
// RWMutex sanction folded in from S-BL.PATH-TRACKER-WRITER. The hook fires
// while the writer holds Router.mu — the hook implementation must not attempt
// to acquire Router.mu recursively (which would deadlock), and concurrent
// callers must produce race-detector-clean behaviour.
//
// Ruling-11 (wave-6-tranche-a-scope-rulings): pathTracker-registry writer
// protection folds into this story since the writer path lands here.
//
// S-BL.PATH-TRACKER-WIRING AC-5; S-BL.PATH-TRACKER-WRITER (folded).
func TestRouter_ConcurrentRegisterForwardingEntry_HookRaceClean(t *testing.T) {
	t.Parallel()

	// Registry maintained inside the hook — same shape as cmd/switchboard's
	// pathTrackerSource will use in production.
	var (
		registryMu sync.Mutex
		registry   = make(map[string]struct{})
	)
	hook := func(svtnID [16]byte, nodeAddr [8]byte) {
		pathID := fmt.Sprintf("%x-%x", svtnID, nodeAddr)
		registryMu.Lock()
		defer registryMu.Unlock()
		registry[pathID] = struct{}{}
	}

	r := routing.NewRouter(admission.NewAdmittedKeySet(), routing.WithForwardingEntryHook(hook))

	const workers = 8
	const perWorker = 64

	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		w := w
		go func() {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				var svtnID [16]byte
				svtnID[0] = byte(w)
				svtnID[1] = byte(i)
				var nodeAddr [8]byte
				nodeAddr[0] = byte(w)
				nodeAddr[1] = byte(i)
				var authKey [hmac.KeySize]byte
				r.RegisterForwardingEntry(svtnID, nodeAddr, authKey)
			}
		}()
	}
	wg.Wait()

	registryMu.Lock()
	defer registryMu.Unlock()
	if got := len(registry); got != workers*perWorker {
		t.Errorf("registry size after concurrent writes: got %d; want %d", got, workers*perWorker)
	}
}
