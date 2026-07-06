// Package testenv provides an in-process multi-daemon test rig for
// switchboard integration tests (ARCH-08 position 22 — top of DAG,
// test-helper composition root; imported only by _test.go files).
//
// Classification: test helper (ARCH-09).  This package may import any
// internal package.  Nothing in the production tree may import testenv.
//
// Unblocked VPs: VP-033, VP-034, VP-036, VP-037, VP-038, VP-039,
//
//	VP-031, VP-032, VP-040, VP-046.
//
// Allowed internal imports: all — position 22 (top composition root).
//
// # SVTN isolation design
//
// Each SVTN in the test environment is backed by its own (publisher,
// auth, accessNode) triple so that DeliverFrame on one SVTN's accessNode
// is invisible to consoles attached to a different SVTN's accessNode.
// This mirrors the real switchboard router's SVTN-aware forwarding without
// requiring a live network stack, and makes VP-039's isolation claim
// testable in-process without faking it.
package testenv

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/drain"
	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/session"
)

// SessionID is an opaque session identifier within a test environment.
// Internally it is a tmux session name string.
type SessionID struct{ name string }

func (s SessionID) String() string { return s.name }

// SVTNID is an opaque SVTN identifier within a test environment.
type SVTNID struct{ id [16]byte }

func (s SVTNID) String() string {
	return fmt.Sprintf("svtn-%x", s.id[:8])
}

// svtnShard holds the per-SVTN in-process stack.
type svtnShard struct {
	svtnID    SVTNID
	keySet    *admission.AdmittedKeySet
	publisher *session.Publisher
	auth      *session.SessionAuth
	access    *session.AccessNode
}

// Conn is a simulated network connection to the test environment.
// It holds the session ID established during connection.
type Conn struct {
	sessionID  SessionID
	shard      *svtnShard
	key        ed25519.PublicKey
	consoleKey session.ConsoleKey
	env        *Env
	closed     atomic.Bool

	mu     sync.Mutex
	frames []frame.OuterHeader
}

// SessionID returns the session this connection joined.
func (c *Conn) SessionID() SessionID { return c.sessionID }

// CollectFrames drains all frames currently buffered, blocking up to timeout
// for the first frame when timeout > 0.  With timeout == 0 returns immediately.
func (c *Conn) CollectFrames(t testing.TB, timeout time.Duration) []frame.OuterHeader {
	t.Helper()
	return collectFromSlice(c, timeout)
}

func (c *Conn) appendFrame(f frame.OuterHeader) {
	c.mu.Lock()
	c.frames = append(c.frames, f)
	c.mu.Unlock()
}

func (c *Conn) snapshot() []frame.OuterHeader {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]frame.OuterHeader, len(c.frames))
	copy(out, c.frames)
	return out
}

// Close terminates the simulated connection.
func (c *Conn) Close() {
	if c.closed.CompareAndSwap(false, true) && c.shard != nil {
		_ = c.shard.access.Detach(c.consoleKey, c.sessionID.name)
	}
}

// frameStore is the common collector interface for poll-based CollectFrames.
type frameStore interface {
	snapshot() []frame.OuterHeader
}

// collectFromSlice is the shared polling loop for CollectFrames variants.
func collectFromSlice(fs frameStore, timeout time.Duration) []frame.OuterHeader {
	snap := fs.snapshot()
	if len(snap) > 0 || timeout <= 0 {
		return snap
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
		if snap = fs.snapshot(); len(snap) > 0 {
			return snap
		}
	}
	return nil
}

// Console is a simulated console attached to a session (VP-033/034).
type Console struct {
	key       session.ConsoleKey
	sessionID SessionID
	shard     *svtnShard
	env       *Env

	mu       sync.Mutex
	frames   []frame.OuterHeader
	detached atomic.Bool
}

func (c *Console) appendFrame(f frame.OuterHeader) {
	c.mu.Lock()
	c.frames = append(c.frames, f)
	c.mu.Unlock()
}

func (c *Console) snapshot() []frame.OuterHeader {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]frame.OuterHeader, len(c.frames))
	copy(out, c.frames)
	return out
}

// CollectFrames returns all frames received so far, blocking up to timeout
// for the first frame when timeout > 0.  With timeout == 0 returns buffered.
func (c *Console) CollectFrames(t testing.TB, timeout time.Duration) []frame.OuterHeader {
	t.Helper()
	return collectFromSlice(c, timeout)
}

// Detach removes this console from the session.  Further frames will not
// be delivered to this Console handle.  The remote session continues.
func (c *Console) Detach(t testing.TB) {
	t.Helper()
	if c.detached.CompareAndSwap(false, true) {
		if err := c.shard.access.Detach(c.key, c.sessionID.name); err != nil {
			t.Logf("testenv: Console.Detach: %v (non-fatal)", err)
		}
		c.env.mu.Lock()
		delete(c.env.consoles, c.key)
		c.env.mu.Unlock()
	}
}

// Probe is a passive frame observer attached to a session for SVTN
// isolation testing (VP-039).
type Probe struct {
	sessionID SessionID
	env       *Env

	mu  sync.Mutex
	all []frame.OuterHeader
}

// FramesFromSVTN returns all frames whose SVTN field matches svtnID.
func (p *Probe) FramesFromSVTN(svtnID SVTNID) []frame.OuterHeader {
	p.mu.Lock()
	defer p.mu.Unlock()
	var out []frame.OuterHeader
	for _, f := range p.all {
		if f.SVTNID == svtnID.id {
			out = append(out, f)
		}
	}
	return out
}

func (p *Probe) record(f frame.OuterHeader) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.all = append(p.all, f)
}

func (p *Probe) snapshot() []frame.OuterHeader {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]frame.OuterHeader, len(p.all))
	copy(out, p.all)
	return out
}

// RouterConfig is the configuration for a router started via StartRouter.
type RouterConfig struct {
	// UpstreamRouters is the list of PE-router addresses this router
	// connects to on startup.  Populated → PE mode; empty → E mode.
	UpstreamRouters []string
}

// RouterMode is the operating mode of a router handle.
type RouterMode int

const (
	// ModeE is edge mode (no upstream router connections).
	ModeE RouterMode = iota
	// ModePE is provider-edge mode (has upstream router connections).
	ModePE
	// modeClose is an internal sentinel for a closed router.
	modeClose RouterMode = -1
)

func (m RouterMode) String() string {
	switch m {
	case ModeE:
		return "E"
	case ModePE:
		return "PE"
	default:
		return fmt.Sprintf("RouterMode(%d)", int(m))
	}
}

// RouterHandle is a live router started by StartRouter.
type RouterHandle struct {
	env    *Env
	mu     sync.RWMutex
	cfg    RouterConfig
	svtnID SVTNID
	mode   RouterMode
}

// Mode returns the current operating mode of the router (E or PE).
func (r *RouterHandle) Mode() RouterMode {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.mode
}

// SVTNID returns the SVTN ID of the router's default SVTN.
func (r *RouterHandle) SVTNID() SVTNID {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.svtnID
}

// Restart reconfigures the router to the new config in place.
// If cfg.UpstreamRouters is non-empty the router enters PE mode; otherwise E.
func (r *RouterHandle) Restart(t testing.TB, cfg RouterConfig) {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cfg = cfg
	if len(cfg.UpstreamRouters) > 0 {
		r.mode = ModePE
	} else {
		r.mode = ModeE
	}
}

// LoopbackConfig configures a NewLoopback environment (VP-042 / S-BL.BENCH).
type LoopbackConfig struct {
	// TickIntervalUpstream is the tick interval for the upstream path.
	TickIntervalUpstream time.Duration
	// TickIntervalDownstream is the tick interval for the downstream path.
	TickIntervalDownstream time.Duration
}

// LoopbackEnv is a minimal single-session loopback environment for
// benchmark use (VP-042, S-BL.BENCH).  Frames sent via SendKeystroke are
// immediately reflected as downstream frames via DeliverFrame.
type LoopbackEnv struct {
	Env *Env
}

// NewLoopback creates a minimal single-session environment optimised for
// benchmarks.  The loopback config controls tick intervals for the
// halfchannel paths.
//
// Required by: VP-042 (S-BL.BENCH).
func NewLoopback(b testing.TB, ctx context.Context, cfg LoopbackConfig) *LoopbackEnv {
	b.Helper()
	env := newEnv(b, ctx, 1)
	return &LoopbackEnv{Env: env}
}

// Env is the in-process test environment.  Construct with New or NewWithRouters.
// All goroutines started by Env are torn down when env.Close() is called or
// when the test's t.Cleanup fires.
type Env struct {
	t   testing.TB
	ctx context.Context

	// routerPriv is the router's ed25519 private key used to generate admission
	// challenges.  It is generated once per Env and used by RegisterKey to
	// complete the challenge-response handshake so that admitted=true.
	routerPub  ed25519.PublicKey
	routerPriv ed25519.PrivateKey

	// Default SVTN shard (used when sessions are created without an explicit SVTN).
	defaultShard *svtnShard

	// All shards indexed by SVTN ID string.
	mu     sync.Mutex
	shards map[string]*svtnShard // svtnID.String() → shard

	// Session → shard mapping (for SendKeystroke routing).
	sessionShard map[SessionID]*svtnShard

	// Drain controller shared across router instances.
	drainCtrl *drain.Drain

	// Router topology for multi-path tests.
	nRouters int
	routers  []*RouterHandle

	// Simulated PE router address (for VP-038).
	peRouterAddr string

	// Console and probe registries.
	consoles    map[session.ConsoleKey]*Console
	probes      map[SessionID][]*Probe
	connsByKey  map[string]*Conn
	keyExpiries map[string]time.Time          // public-key hex → expiry time
	keyPrivates map[string]ed25519.PrivateKey // public-key hex → private key (for admission)

	// Sequence counters.
	sessionSeq atomic.Uint64
	svtnSeq    atomic.Uint64

	// Lifecycle.
	closeCh   chan struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup
}

// New creates a minimal single-router in-process switchboard environment.
// The environment is torn down when t.Cleanup fires or Close() is called.
//
// Required by: VP-033, VP-034, VP-036, VP-038, VP-039, VP-046.
func New(t testing.TB, ctx context.Context) *Env {
	t.Helper()
	return newEnv(t, ctx, 1)
}

// NewWithRouters creates an in-process switchboard environment with n routers
// in a multi-hop topology suitable for drain/failover/multipath tests.
//
// Required by: VP-037, VP-040.
func NewWithRouters(t testing.TB, ctx context.Context, n int) *Env {
	t.Helper()
	if n < 1 {
		t.Fatalf("testenv.NewWithRouters: n must be >= 1, got %d", n)
	}
	return newEnv(t, ctx, n)
}

func newEnv(t testing.TB, ctx context.Context, nRouters int) *Env {
	t.Helper()

	routerPub, routerPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("testenv: generate router keypair: %v", err)
	}

	e := &Env{
		t:            t,
		ctx:          ctx,
		routerPub:    routerPub,
		routerPriv:   routerPriv,
		drainCtrl:    drain.New(drain.DefaultTimeout),
		nRouters:     nRouters,
		shards:       make(map[string]*svtnShard),
		sessionShard: make(map[SessionID]*svtnShard),
		consoles:     make(map[session.ConsoleKey]*Console),
		probes:       make(map[SessionID][]*Probe),
		connsByKey:   make(map[string]*Conn),
		keyExpiries:  make(map[string]time.Time),
		keyPrivates:  make(map[string]ed25519.PrivateKey),
		closeCh:      make(chan struct{}),
		peRouterAddr: "127.0.0.1:9999",
	}

	// Create default SVTN shard.
	defaultSVTN := e.newSVTNIDNoLock()
	e.defaultShard = e.newShard(defaultSVTN)

	// Pre-create router handles for multi-router topologies.
	for i := 0; i < nRouters; i++ {
		svtn := e.newSVTNIDNoLock()
		e.routers = append(e.routers, &RouterHandle{
			env:    e,
			cfg:    RouterConfig{},
			mode:   ModeE,
			svtnID: svtn,
		})
	}

	t.Cleanup(e.Close)
	return e
}

// newShard creates a fresh per-SVTN stack and registers it.  Must be called
// under e.mu or during construction.
func (e *Env) newShard(svtnID SVTNID) *svtnShard {
	ks := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(ks)
	auth := session.NewSessionAuth()
	access := session.NewAccessNode(pub, auth, session.WithKeystrokeSink(session.NoOpSink{}))
	sh := &svtnShard{
		svtnID:    svtnID,
		keySet:    ks,
		publisher: pub,
		auth:      auth,
		access:    access,
	}
	e.shards[svtnID.String()] = sh
	return sh
}

// shardFor returns the shard for a SVTN ID, creating it lazily.
func (e *Env) shardFor(svtnID SVTNID) *svtnShard {
	e.mu.Lock()
	defer e.mu.Unlock()
	if sh, ok := e.shards[svtnID.String()]; ok {
		return sh
	}
	return e.newShard(svtnID)
}

// Close tears down the environment.  Idempotent.
func (e *Env) Close() {
	e.closeOnce.Do(func() {
		close(e.closeCh)
		e.wg.Wait()
	})
}

// PERouterAddr returns the address string of the PE router in this environment.
// Used by VP-038 to supply the UpstreamRouters list.
func (e *Env) PERouterAddr(t testing.TB) string {
	t.Helper()
	return e.peRouterAddr
}

// --- Session helpers -------------------------------------------------------

// CreateSession publishes a new session in the default SVTN and returns its ID.
func (e *Env) CreateSession(t testing.TB) SessionID {
	t.Helper()
	return e.createSessionInShard(t, e.defaultShard)
}

// CreateSVTN creates a new virtual tenant network and returns its ID.
// The optional name parameter is ignored; SVTN IDs are auto-generated.
// Required by VP-039.
func (e *Env) CreateSVTN(t testing.TB, names ...string) SVTNID {
	t.Helper()
	svtn := e.newSVTNID()
	_ = e.shardFor(svtn) // ensure shard exists
	return svtn
}

// CreateSessionInSVTN creates a session scoped to the specified SVTN.
// Required by VP-039.
func (e *Env) CreateSessionInSVTN(t testing.TB, svtnID SVTNID) SessionID {
	t.Helper()
	sh := e.shardFor(svtnID)
	return e.createSessionInShard(t, sh)
}

func (e *Env) createSessionInShard(t testing.TB, sh *svtnShard) SessionID {
	t.Helper()
	name := fmt.Sprintf("test-session-%d", e.sessionSeq.Add(1))
	if err := sh.publisher.Publish(name); err != nil {
		t.Fatalf("testenv.CreateSession: Publish(%q): %v", name, err)
	}
	sid := SessionID{name: name}
	e.mu.Lock()
	e.sessionShard[sid] = sh
	e.mu.Unlock()
	t.Cleanup(func() {
		_ = sh.publisher.Unpublish(name)
		e.mu.Lock()
		delete(e.sessionShard, sid)
		e.mu.Unlock()
	})
	return sid
}

// shardForSession returns the shard for the session, defaulting to defaultShard.
func (e *Env) shardForSession(sessionID SessionID) *svtnShard {
	e.mu.Lock()
	defer e.mu.Unlock()
	if sh, ok := e.sessionShard[sessionID]; ok {
		return sh
	}
	return e.defaultShard
}

// SessionAlive returns true if the session is still published.
func (e *Env) SessionAlive(t testing.TB, sessionID SessionID) bool {
	t.Helper()
	sh := e.shardForSession(sessionID)
	return sh.publisher.Exists(sessionID.name)
}

// --- Console helpers -------------------------------------------------------

// AttachConsole attaches a new console to the given session.
// The returned Console collects downstream frames asynchronously.
// Required by VP-033, VP-034.
func (e *Env) AttachConsole(t testing.TB, sessionID SessionID) *Console {
	t.Helper()
	sh := e.shardForSession(sessionID)
	key := e.newConsoleKey()
	sh.auth.RegisterKey(sessionID.name, key, session.RoleFull)

	downstream, _, err := sh.access.Attach(key, sessionID.name)
	if err != nil {
		t.Fatalf("testenv.AttachConsole: Attach(%q): %v", sessionID, err)
	}

	c := &Console{
		key:       key,
		sessionID: sessionID,
		shard:     sh,
		env:       e,
	}

	e.mu.Lock()
	e.consoles[key] = c
	e.mu.Unlock()

	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		for {
			select {
			case f, ok := <-downstream:
				if !ok {
					return
				}
				if !c.detached.Load() {
					c.appendFrame(f)
				}
			case <-e.closeCh:
				return
			}
		}
	}()

	t.Cleanup(func() {
		if !c.detached.Load() {
			c.Detach(t)
		}
	})
	return c
}

// AttachProbe attaches a passive frame observer to a session.
// Frames are recorded with their SVTN ID so FramesFromSVTN can filter them.
// Required by VP-039.
func (e *Env) AttachProbe(t testing.TB, sessionID SessionID) *Probe {
	t.Helper()
	sh := e.shardForSession(sessionID)
	key := e.newConsoleKey()
	sh.auth.RegisterKey(sessionID.name, key, session.RoleFull)

	downstream, _, err := sh.access.Attach(key, sessionID.name)
	if err != nil {
		t.Fatalf("testenv.AttachProbe: Attach(%q): %v", sessionID, err)
	}

	p := &Probe{
		sessionID: sessionID,
		env:       e,
	}

	e.mu.Lock()
	e.probes[sessionID] = append(e.probes[sessionID], p)
	e.mu.Unlock()

	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		for {
			select {
			case f, ok := <-downstream:
				if !ok {
					return
				}
				p.record(f)
			case <-e.closeCh:
				return
			}
		}
	}()

	t.Cleanup(func() {
		_ = sh.access.Detach(key, sessionID.name)
	})
	return p
}

// --- Keystroke / frame delivery -------------------------------------------

// SendKeystroke injects a frame into the access node for the given session.
// All attached consoles and probes on that session will receive the frame.
// The SVTN ID is stamped onto the frame header to allow isolation probing.
//
// Because each SVTN has its own accessNode, frames stamped with svtnA.id
// are only delivered to consoles attached to svtnA sessions — never to
// svtnB consoles.  This provides in-process SVTN isolation for VP-039.
func (e *Env) SendKeystroke(t testing.TB, sessionID SessionID, key string) {
	t.Helper()
	sh := e.shardForSession(sessionID)
	hdr := frame.OuterHeader{
		Version:   frame.VersionByte,
		FrameType: frame.FrameTypeData,
		SVTNID:    sh.svtnID.id,
	}
	copy(hdr.SrcAddr[:], []byte(sessionID.name))
	sh.access.DeliverFrame(hdr)
}

// CollectFrames collects frames delivered to a session from all attached
// consoles and probes.  Blocks up to timeout for the first frame.
func (e *Env) CollectFrames(t testing.TB, sessionID SessionID, timeout time.Duration) []frame.OuterHeader {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		e.mu.Lock()
		var all []frame.OuterHeader
		for _, p := range e.probes[sessionID] {
			all = append(all, p.snapshot()...)
		}
		for _, c := range e.consoles {
			if c.sessionID == sessionID && !c.detached.Load() {
				all = append(all, c.snapshot()...)
			}
		}
		e.mu.Unlock()
		if len(all) > 0 {
			return all
		}
		if timeout <= 0 || time.Now().After(deadline) {
			return nil
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// --- Connectivity helpers -------------------------------------------------

// GenerateCredentials generates a fresh ed25519 key pair, registers it,
// and performs admission so that IsAdmitted returns true immediately.
// Returns the public key.  Required by VP-036.
func (e *Env) GenerateCredentials(t testing.TB) ed25519.PublicKey {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("testenv.GenerateCredentials: GenerateKey: %v", err)
	}
	svtnID := e.defaultShard.svtnID.id
	e.defaultShard.keySet.RegisterKey(svtnID, pub, admission.RoleAccess)
	e.performAdmission(t, e.defaultShard.keySet, svtnID, pub, priv)
	return pub
}

// ConnectWithSourceIP simulates a connection to the environment from the
// given source IP address using the provided ed25519 public key credential.
// Reconnecting with new source IP preserves the session ID (VP-036).
//
// The returned Conn exposes CollectFrames to receive downstream traffic.
// Required by VP-036.
func (e *Env) ConnectWithSourceIP(t testing.TB, srcIP string, creds ed25519.PublicKey) *Conn {
	t.Helper()

	keyHex := fmt.Sprintf("%x", creds)

	e.mu.Lock()
	existing, ok := e.connsByKey[keyHex]
	e.mu.Unlock()

	var sessionID SessionID
	var sh *svtnShard
	if ok {
		sessionID = existing.sessionID
		sh = existing.shard
		// Detach the old connection so the console key can be reused.
		if !existing.closed.Load() {
			_ = sh.access.Detach(existing.consoleKey, existing.sessionID.name)
			existing.closed.Store(true)
		}
	} else {
		// First connection: create a new session in the default SVTN.
		sh = e.defaultShard
		sessionID = e.createSessionInShard(t, sh)
	}

	// Source-IP is encoded into the console key so IP-A and IP-B get distinct
	// console registrations, simulating reconnection from a different address.
	consoleKey := session.ConsoleKey(fmt.Sprintf("conn-%s-%x", srcIP, creds[:4]))
	sh.auth.RegisterKey(sessionID.name, consoleKey, session.RoleFull)

	downstream, _, err := sh.access.Attach(consoleKey, sessionID.name)
	if err != nil {
		t.Fatalf("testenv.ConnectWithSourceIP: Attach from %s: %v", srcIP, err)
	}

	conn := &Conn{
		sessionID:  sessionID,
		shard:      sh,
		key:        creds,
		consoleKey: consoleKey,
		env:        e,
	}

	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		for {
			select {
			case f, ok := <-downstream:
				if !ok {
					return
				}
				conn.appendFrame(f)
			case <-e.closeCh:
				return
			}
		}
	}()

	e.mu.Lock()
	e.connsByKey[keyHex] = conn
	e.mu.Unlock()

	t.Cleanup(conn.Close)
	return conn
}

// GenerateKey generates a fresh ed25519 public key and stores its private key
// internally so RegisterKey can perform the full admission handshake.
// Required by VP-046.
func (e *Env) GenerateKey(t testing.TB) ed25519.PublicKey {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("testenv.GenerateKey: GenerateKey: %v", err)
	}
	e.mu.Lock()
	e.keyPrivates[fmt.Sprintf("%x", pub)] = priv
	e.mu.Unlock()
	return pub
}

// GenerateKeyWithExpiry generates a key that expires at the given time.
// Call RegisterKey separately before ConnectWithKey.  Required by VP-046.
func (e *Env) GenerateKeyWithExpiry(t testing.TB, expiry time.Time) ed25519.PublicKey {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("testenv.GenerateKeyWithExpiry: GenerateKey: %v", err)
	}
	keyHex := fmt.Sprintf("%x", pub)
	e.mu.Lock()
	e.keyExpiries[keyHex] = expiry
	e.keyPrivates[keyHex] = priv
	e.mu.Unlock()
	return pub
}

// RegisterKey registers and admits the given public key in the environment's
// default SVTN so that ConnectWithKey returns nil.  Requires that pub was
// produced by GenerateKey() or GenerateKeyWithExpiry() (so the private key
// is available for the challenge-response handshake).  Required by VP-046.
func (e *Env) RegisterKey(t testing.TB, pub ed25519.PublicKey) {
	t.Helper()
	e.admitKey(t, e.defaultShard.keySet, e.defaultShard.svtnID.id, pub)
}

// admitKey registers a key and immediately admits it via the challenge-response
// protocol, so that AdmittedKeySet.IsAdmitted returns true.
// Requires that GenerateKey() was used to produce pub (so the private key
// is stored in e.keyPrivates).
func (e *Env) admitKey(t testing.TB, ks *admission.AdmittedKeySet, svtnID [16]byte, pub ed25519.PublicKey) {
	t.Helper()
	ks.RegisterKey(svtnID, pub, admission.RoleAccess)

	e.mu.Lock()
	priv, hasPriv := e.keyPrivates[fmt.Sprintf("%x", pub)]
	e.mu.Unlock()
	if !hasPriv {
		// Key was generated externally; skip admission (IsAdmitted stays false).
		// Callers should use GenerateKey() to produce keys for ConnectWithKey.
		return
	}
	e.performAdmission(t, ks, svtnID, pub, priv)
}

// performAdmission performs the full challenge-response admission handshake,
// setting admitted=true in the key set for the given (svtnID, pub) pair.
func (e *Env) performAdmission(t testing.TB, ks *admission.AdmittedKeySet, svtnID [16]byte, pub ed25519.PublicKey, priv ed25519.PrivateKey) {
	t.Helper()
	ch, err := admission.GenerateChallenge(e.routerPriv)
	if err != nil {
		t.Fatalf("testenv.performAdmission: GenerateChallenge: %v", err)
	}
	sig := ed25519.Sign(priv, ch.Nonce[:])
	resp := admission.ChallengeResponse{NonceSig: sig}
	if err := admission.AdmitNode(ch, resp, pub, svtnID, ks); err != nil {
		t.Fatalf("testenv.performAdmission: AdmitNode: %v", err)
	}
}

// RevokeKey revokes the given key so that subsequent ConnectWithKey calls fail.
// Required by VP-046.
func (e *Env) RevokeKey(t testing.TB, pub ed25519.PublicKey) {
	t.Helper()
	svtnID := e.defaultShard.svtnID.id
	nodeAddr := frame.DeriveNodeAddress(svtnID, pub)
	if err := e.defaultShard.keySet.RevokeKey(svtnID, nodeAddr); err != nil {
		t.Fatalf("testenv.RevokeKey: %v", err)
	}
}

// ConnectWithKey attempts admission of the given key.  Returns nil on success,
// a non-nil error on rejection (revoked / expired / not registered).
// Required by VP-046.
func (e *Env) ConnectWithKey(t testing.TB, pub ed25519.PublicKey) error {
	t.Helper()
	svtnID := e.defaultShard.svtnID.id
	nodeAddr := frame.DeriveNodeAddress(svtnID, pub)

	keyHex := fmt.Sprintf("%x", pub)
	e.mu.Lock()
	expiry, hasExpiry := e.keyExpiries[keyHex]
	e.mu.Unlock()
	if hasExpiry && time.Now().After(expiry) {
		return fmt.Errorf("testenv: key expired at %v", expiry)
	}

	if !e.defaultShard.keySet.IsAdmitted(svtnID, nodeAddr) {
		return fmt.Errorf("testenv: key not admitted (not registered or revoked)")
	}
	return nil
}

// --- Router control helpers (VP-037, VP-038, VP-040) ----------------------

// StartRouter starts a router with the given config.  Returns a handle for
// inspecting and restarting the router.  Required by VP-038.
func (e *Env) StartRouter(t testing.TB, cfg RouterConfig) *RouterHandle {
	t.Helper()
	h := &RouterHandle{
		env:    e,
		cfg:    cfg,
		svtnID: e.newSVTNID(),
	}
	if len(cfg.UpstreamRouters) > 0 {
		h.mode = ModePE
	} else {
		h.mode = ModeE
	}
	e.mu.Lock()
	e.routers = append(e.routers, h)
	e.mu.Unlock()
	return h
}

// SendDrainSignal triggers the drain controller.  Connected nodes should
// migrate to an alternate router within the drain timeout.
// Required by VP-037.
func (e *Env) SendDrainSignal(t testing.TB, idx int) {
	t.Helper()
	e.drainCtrl.Signal(e.ctx)
}

// WaitForPaths blocks until the environment has at least n active routers,
// or timeout elapses.  Returns an error describing the shortfall.
// Required by VP-040.
func (e *Env) WaitForPaths(t testing.TB, sessionID SessionID, n int, timeout time.Duration) error {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if e.activeRouterCount() >= n {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("WaitForPaths: only %d routers active after %v (wanted %d)",
		e.activeRouterCount(), timeout, n)
}

func (e *Env) activeRouterCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	active := 0
	for _, r := range e.routers {
		r.mu.RLock()
		if r.mode != modeClose {
			active++
		}
		r.mu.RUnlock()
	}
	return active
}

// CloseRouterConnection simulates closing the connection to router[idx].
// Traffic previously using that router must fail over to remaining routers.
// Required by VP-040.
func (e *Env) CloseRouterConnection(t testing.TB, idx int) {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	if idx < 0 || idx >= len(e.routers) {
		t.Fatalf("testenv.CloseRouterConnection: index %d out of range (have %d routers)", idx, len(e.routers))
	}
	e.routers[idx].mu.Lock()
	e.routers[idx].mode = modeClose
	e.routers[idx].mu.Unlock()
}

// WaitForEcho blocks until the session produces at least one downstream frame.
// Used by VP-042 (S-BL.BENCH) for benchmark teardown.
func (e *Env) WaitForEcho(b testing.TB, sessionID SessionID, text string, timeout time.Duration) {
	b.Helper()
	if frames := e.CollectFrames(b, sessionID, timeout); len(frames) == 0 {
		b.Errorf("testenv.WaitForEcho: no frame received within %v", timeout)
	}
}

// --- internal helpers -----------------------------------------------------

func (e *Env) newConsoleKey() session.ConsoleKey {
	var buf [8]byte
	_, _ = io.ReadFull(rand.Reader, buf[:])
	return session.ConsoleKey(fmt.Sprintf("tc-%x", buf[:]))
}

func (e *Env) newSVTNID() SVTNID {
	return svtnIDFromSeq(e.svtnSeq.Add(1))
}

func (e *Env) newSVTNIDNoLock() SVTNID {
	return svtnIDFromSeq(e.svtnSeq.Add(1))
}

func svtnIDFromSeq(seq uint64) SVTNID {
	var id [16]byte
	binary.BigEndian.PutUint64(id[:8], seq)
	binary.BigEndian.PutUint64(id[8:], seq^0xdeadbeefcafe0000)
	return SVTNID{id: id}
}
