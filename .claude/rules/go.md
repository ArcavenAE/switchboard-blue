# Go Quality Rules

These rules apply to everything under `cmd/` and `internal/`. They are a
condensation of community practice and what `.golangci.yml` enforces —
follow the rules up front; lint is the backstop.

## Idiomatic Go — MUST Follow

**1. Use `fmt.Fprintf` — never `WriteString` + `Sprintf`**
```go
// WRONG — allocates intermediate string
s.WriteString(fmt.Sprintf("Task: %s", name))

// RIGHT — writes directly to the writer
fmt.Fprintf(&s, "Task: %s", name)
```

**2. Never nil-check before `len`**
```go
// WRONG — len handles nil slices/maps (returns 0)
if tasks != nil && len(tasks) > 0 { ... }

// RIGHT
if len(tasks) > 0 { ... }
```

**3. Always check error returns**
```go
// WRONG — silently ignoring error
data, _ := json.Marshal(task)

// RIGHT — handle or propagate every error
data, err := json.Marshal(task)
if err != nil {
    return fmt.Errorf("marshal task %s: %w", task.ID, err)
}
```

**4. Wrap errors with context using `%w`**
```go
// WRONG — loses error chain
return fmt.Errorf("failed to save: %v", err)

// RIGHT — preserves chain for errors.Is/errors.As
return fmt.Errorf("save task %s: %w", id, err)
```

**5. Error strings: no trailing punctuation or newlines (ST1005)**
```go
// WRONG
return fmt.Errorf("cannot do the thing.")
return fmt.Errorf("cannot do the thing!\n")

// RIGHT
return fmt.Errorf("cannot do the thing")
```

**6. Accept interfaces, return concrete types**
```go
// WRONG — returning interface hides implementation
func NewProvider() Provider { ... }

// RIGHT — return the concrete type
func NewProvider(path string) *FileProvider { ... }
```

**7. `context.Context` is always the first parameter**
```go
// WRONG
func Load(path string, ctx context.Context) error

// RIGHT
func Load(ctx context.Context, path string) error
```

**8. Don't use `interface{}`/`any` without justification**
- Prefer specific types or generics over `any`
- If `any` is needed, document why in a comment

**9. Prefer value receivers unless mutation is needed**
- Use pointer receiver only when mutating, when the struct is large
  (>~64 bytes) and copying is expensive, or when consistency demands it
  (one pointer method → all pointer methods)

**10. No `init()` functions**
- Pass dependencies explicitly via constructors
- Configuration belongs in `main()` or factory functions

**11. Timestamps always in UTC**
```go
time.Now().UTC()   // yes
time.Now()         // no
```

**12. Never return internal pointers from a locked accessor**

If a type owns state behind a mutex, its `Get*`/`List*`/`Lookup*` methods
return value copies (deep enough — clone nested slices/maps/pointers).
Mutation goes through methods on that owning type; those methods take the
lock and mutate in place. A returned `*T` into concurrently-mutated state
is a bug — the lock protects the map, not the values.

For single-item lookups that may not find a result, return `(T, bool)` —
the idiomatic Go map-lookup pattern. Never return `*T` as a found/not-found
signal from a locked accessor; the pointer return type falsely implies the
caller may hold live state, and a nil-check is weaker than a bool-check at
the call site.

```go
// WRONG — returns pointer; nil used as "not found" sentinel; caller could
// hold a stale pointer into internal state
func (s *Store) Lookup(key string) *Session {
    s.mu.RLock()
    defer s.mu.RUnlock()
    sess, ok := s.sessions[key]
    if !ok {
        return nil
    }
    cp := *sess
    return &cp // misleading: looks like live state, signals "may be mutated"
}

// RIGHT — value + present-flag; caller pattern is `sess, ok := ...; if !ok`
func (s *Store) Lookup(key string) (Session, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    sess, ok := s.sessions[key]
    if !ok {
        return Session{}, false
    }
    return cloneSession(sess), true
}

// WRONG — caller holds a pointer into the store's live object,
// mutates it after the lock is dropped, other callers see torn writes
func (s *Store) ListSessions() []*Session {
    s.mu.RLock()
    defer s.mu.RUnlock()
    out := make([]*Session, 0, len(s.sessions))
    for _, sess := range s.sessions {
        out = append(out, sess)   // leaks internal pointer
    }
    return out
}

// RIGHT — snapshots fully decoupled from internal state
func (s *Store) ListSessions() []Session {
    s.mu.RLock()
    defer s.mu.RUnlock()
    out := make([]Session, 0, len(s.sessions))
    for _, sess := range s.sessions {
        out = append(out, cloneSession(sess))
    }
    return out
}

// RIGHT — mutation goes through the owner under the lock
func (s *Store) UpdateSession(key string, fn func(*Session) error) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    sess, ok := s.sessions[key]
    if !ok {
        return fmt.Errorf("session %s: %w", key, ErrNotFound)
    }
    return fn(sess)
}
```

Note: if a struct contains a slice field (e.g., `PublicKey []byte`), a
value copy of the struct shares the slice's backing array. Deep-clone
such fields explicitly: `cp.PublicKey = append([]byte(nil), src.PublicKey...)`.

The Kubernetes lister/informer pattern is the canonical example: cached
reads give immutable snapshots, writes go through the API. `go test -race`
is the backstop; the type signature is the fence. See orc
finding-032-store-sync-contract-leak.

**13. Security-perimeter constructor defaults must fail closed**

When a constructor parameter guards a security-enforcing boundary (an
authorization check, a HMAC verifier, a rate-limiter, a sandbox), its
nil/zero default MUST fail closed (deny-all, panic, or require an explicit
opt-in value), not fail open (allow-all, no-op). A nil-or-zero default that
silently disables enforcement compiles without warning and disables security
invariants with no operator-visible signal.

The canonical failure: `NewAccessNode(pub, nil)` defaulting to `NoOpAuthorizer`
(allow-all) while the deliberate no-op sink (`noSink`) panics on misuse — the
same codebase uses opposite polarities with no rationale for the asymmetry
(W3-PG-001, Wave-3 adversary pass-01 F-6).

Rule: if nil/zero is accepted for a security-perimeter parameter, document
the explicit rationale in the constructor comment. If no rationale exists,
require a non-nil value (panic or return error), or provide a test-only
`WithAllowAll` opt-in that callers must name explicitly. The default remains
deny-all.

```go
// WRONG — nil auth silently disables all Tier-2 enforcement
func NewAccessNode(pub PublicKey, auth Authorizer) *AccessNode {
    if auth == nil {
        auth = NoOpAuthorizer{} // latent fail-OPEN: compiles, enforces nothing
    }
    ...
}

// RIGHT — nil auth is a programmer error; require explicit allow-all opt-in
func NewAccessNode(pub PublicKey, auth Authorizer) *AccessNode {
    if auth == nil {
        panic("NewAccessNode: auth must not be nil; use WithAllowAllAuthorizer() for tests")
    }
    ...
}
```

## Error handling

- Every exported function that can fail returns `error` last
- Use `errors.Is()` and `errors.As()` for inspection — never string
  matching
- Define sentinel errors as package-level `var` with documentation
- No `log.Fatal` / `os.Exit` outside `main()`; let errors propagate
- No panics in library code

## Testing

- **Table-driven tests** for any function with >2 test cases
- **stdlib `testing`** — no testify. Use `t.Fatal`, `t.Errorf`, `t.Helper()`
- **`t.Helper()`** in test helpers so failures report the caller's line
- **`t.Cleanup()`** instead of `defer` for test resource cleanup
- Test files alongside source: `foo.go` → `foo_test.go`
- Fixtures in `testdata/` directories
- Mark independent tests with `t.Parallel()` where safe

## Code organisation

- Package naming: lowercase, single word. No underscores or camelCase
- File naming: lowercase snake_case (`task_pool.go`, `handler.go`)
- One primary type per file when practical
- Import order enforced by gofumpt: stdlib → external → internal
- Keep packages small — split when one exceeds ~10 files

## Common AI mistakes to avoid

1. Don't create unnecessary abstractions — three similar lines are
   better than a premature helper
2. Don't add unused parameters "for future use" — YAGNI
3. Don't shadow imports — `var errors = ...` shadows the `errors` pkg
4. Don't use `log.Fatal` / `os.Exit` outside `main()`
5. Don't buffer channels without justification — unbuffered is the
   default for a reason
6. Don't use `sync.Mutex` when `atomic` suffices for simple counters
7. Don't create `utils` or `helpers` packages — put functions where
   they're used
8. Don't add comments that restate the code — only comment the *why*
9. Don't use `strings.Builder` then call `Sprintf` into it — use
   `fmt.Fprintf` directly
10. Don't return `(bool, error)` as a substitute for `error`

## Formatting & linting

- Formatter: `gofumpt` — run via `just fmt`
- Linter: `golangci-lint run ./...` pinned in `.golangci.yml`
- Both are enforced in CI (`.github/workflows/ci.yml`) and by lefthook
  (`lefthook.yml`)
- Never disable linter rules with `//nolint` without a justifying
  comment on the same line
