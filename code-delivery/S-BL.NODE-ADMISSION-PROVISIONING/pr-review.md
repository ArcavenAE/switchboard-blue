## PR Reviewer — Fresh-Eyes Review (COMMENTED)

**Verdict:** Code is sound and ready to merge on its own merits. **One required CI check is currently RED**, which blocks merge until fixed (fix is a PR-description edit only — no code change). Reviewed all changed files against the 8-item checklist.

Single-identity constraint acknowledged: delivered as a COMMENTED review (no formal APPROVE/REQUEST-CHANGES verdict available).

---

### What I verified (independent inspection)

- `go build ./...` — clean. `go test -race` for `TestAdmissionKeypair* / TestConfig_Validate_AdmissionKeyFile* / TestDiscoveryRun* / TestAccessDaemon_LocalNode* / TestRunAccess_*` — all PASS (`cmd/switchboard`, `internal/config`). `go vet` — clean. CodeQL + Quality Gate + dependency-review — green.
- **Security (key material):** private key is held only as a local `ed25519.PrivateKey` in `runAccess`, never stored in a struct crossing a package boundary, never logged (only the base64url *public* key is emitted). Atomic write is correct: `os.CreateTemp` (O_EXCL + random suffix) → `Write` → `Close` → explicit `os.Chmod(0600)` (umask-independent) → `os.Rename`; parent dir `MkdirAll(…, 0700)`. Error paths `os.Remove` the temp. Fail-closed on corrupt PEM / non-Ed25519 type.
- **Permissions predicate:** `perm&0o077 != 0` (OpenSSH semantics) correctly warns on world-readable `0444`, which the naive `perm > 0o600` predicate would have missed (292 < 384). The AC-005 table explicitly covers 0644/0444/0440 and 0600-no-warn.
- **WaitGroup contract:** `wg.Add(1)` before `go`, `defer wg.Done()` inside the goroutine; `disc.Run` shares the same `wg` as the sweep/frames-dropped tickers and is joined by `wg.Wait()` inside `runAccessWithConnector` before mgmt-server shutdown. `context.Canceled` / `DeadlineExceeded` from `disc.Run` are treated as clean shutdown (not `internalFailure`).
- **No partial-startup leak (F-2):** keypair load runs after the mgmt Serve goroutine starts, but a once-guarded `defer shutdownMgmt()` joins the mgmt goroutine on any early return, and the explicit post-`runAccessWithConnector` shutdown is de-duplicated by the same `sync.Once`.
- **Config purity:** `Validate()` performs no file I/O; only rejects a present-but-whitespace-only `admission_key_file` (E-CFG-014), collected exhaustively alongside other field errors.
- **Test quality:** tests are genuinely discriminating, not trivially passing — the AC-008 blocking-`HeartbeatObserver` channel handshake distinguishes fixed-vs-stub (function must block in `wg.Wait` while disc is parked); the AC-007 through-`runAccess` `newDiscovery` capturing seam asserts byte-equality of the wired pubkey (and the diff honestly notes the naive "never returns ErrMissingNodeAdmissionPubkey" postcondition is vacuous). Commit history shows a real Red-Gate → green progression.
- **Demo evidence:** present and POL-004-compliant — 8 `.tape` scripts + `evidence-report.md` (source-of-truth, no committed binaries, regeneration commands included). Error/negative paths covered (AC-004 fail-closed, AC-005 warning). Compliant with the project's ratified demo policy.
- **Diff coherence / commits:** all changes relate to this story; conventional-commit format with story ID; production code is ~276 lines (`access.go`) + 23 (`config.go`), the bulk of the 2677-line diff is tests (1528) + evidence. Reasonable for a plumbing story.

---

### Findings

| # | Severity | Category | Location | Finding |
|---|----------|----------|----------|---------|
| 1 | **BLOCKING (merge)** | ci / description | PR body `### Blast Radius` | Required check **"Declaration present / Verify blast-radius block"** is FAILING. |
| 2 | suggestion | correctness / durability | `cmd/switchboard/access.go` `loadOrGenerateAdmissionKeypair` | No `fsync` before `os.Rename`. |
| 3 | nit | hygiene | `cmd/switchboard/access.go` `loadOrGenerateAdmissionKeypair` | Orphaned `admission-*.pem.tmp` (private-key material) can accumulate on hard crash. |
| 4 | nit | style | `cmd/switchboard/access.go` `runAccessWithConnector` | `disc ...*discovery.Discovery` variadic-of-one for an optional single arg. |

---

#### 1. [BLOCKING for merge] Required CI check "Verify blast-radius block" is red

The `Declaration present` workflow requires a **top-level** `## Blast Radius` section whose regex is `^##\s+blast radius\b`, with at least one of three labelled prompts carrying ≥8 chars of substance:
`**1. Operator-visible surfaces touched:**`, `**2. Silent-failure risk:**`, `**3. Smoke gate touched:**`.

The current PR body has `### Blast Radius` (H3, nested under "Risk Assessment & Deployment") — an H3 does not match `^##\s+`, and the three labelled prompts are absent. The gate therefore fails.

This is **not a code defect** — the fix is entirely in the PR description. Add a top-level section, e.g.:

```
## Blast Radius

**1. Operator-visible surfaces touched:** access daemon startup path; `internal/config.Config` gains `admission_key_file`. A new key file is created at the default path on first start (INFO-logged).
**2. Silent-failure risk:** none — keypair load is fail-closed (corrupt/wrong-type → daemon refuses to start); no network I/O in this story.
**3. Smoke gate touched:** no new sentinel required.
```

Per the convergence record ("CI green"), the PR cannot merge while this check is red.

#### 2. [SUGGESTION] fsync before rename in the atomic write

`loadOrGenerateAdmissionKeypair` does `Write` → `Close` → `Chmod` → `Rename` without an `f.Sync()`. On a hard crash (power loss) after the rename metadata is persisted but before the file data is flushed, the canonical path could hold a truncated/empty key. This is **safe** because the fail-closed load path means the daemon refuses to start on a corrupt key (operator deletes + regenerates) rather than using bad key material silently — so it is a durability nicety, not a security or silent-corruption issue. Consider `f.Sync()` before `Close()` for crash-durability.

#### 3. [NIT] Orphaned temp files accumulate on hard crash

The temp file (`admission-*.pem.tmp`, containing a freshly generated private key, mode 0600) is `os.Remove`d on every clean error path, but a `SIGKILL`/power-loss between `CreateTemp` and the rename leaves one orphaned in the key directory indefinitely. Mode 0600 limits exposure, but they contain key material and never get swept. Consider a startup glob-and-remove of stale `admission-*.pem.tmp`, or documenting the expectation.

#### 4. [NIT] Variadic-of-one for an optional discovery arg

`runAccessWithConnector(..., disc ...*discovery.Discovery)` uses a variadic to keep the 3 existing test callers compiling unchanged, guarded by `len(disc) > 0 && disc[0] != nil`. Idiomatic Go generally prefers an explicit `disc *discovery.Discovery` parameter with `nil` passed at the (few) existing call sites, which makes the "at most one, may be nil" contract type-enforced rather than convention. The current form is well-documented and mirrors the existing `newHalfChannel` seam pattern, so this is a judgment call — noting for consistency, not correctness.

---

### Recommendation

**Merge: YES, once the blast-radius CI check passes** (finding #1 — PR-description edit only). Findings #2–#4 are non-blocking; disposition at author's discretion. Code, tests, security handling, and demo evidence are all sound and independently verified.
