---
artifact_id: ARCH-10-tooling-selection
document_type: architecture-section
level: L3
version: "1.0"
status: draft
producer: architect
timestamp: 2026-06-23T00:00:00
phase: 1b
traces_to: ARCH-INDEX.md
inputDocuments:
  - '.factory/specs/prd-supplements/nfr-catalog.md'
  - '.factory/specs/module-criticality.md'
kos_anchors: []
---

# ARCH-10: Tooling Selection

## Constraint: Go-Native Toolchain Only

This is a Go project. Rust-specific tools (Kani, cargo-fuzz, cargo-mutants) are
explicitly excluded. All verification tooling must be compatible with Go 1.25.4
and the existing justfile/lefthook CI setup.

## Verification Tool Registry

| Tool | Version | Purpose | Scope |
|------|---------|---------|-------|
| `go test` | go1.25.4 (stdlib) | Unit tests, table-driven tests, integration tests | all packages |
| `go test -race` | go1.25.4 (stdlib) | Race condition detection | all packages |
| `go test -fuzz` | go1.25.4 (stdlib) | Native fuzz testing (introduced in Go 1.18) | frame, hmac, admission, config |
| `gopter` | v0.2.9+ | Property-based testing (analogous to proptest/Hypothesis) | all pure-core packages |
| `golangci-lint` | v1.59+ (pinned in .golangci.yml) | Static analysis, security rules | all packages |
| `staticcheck` | latest (via golangci-lint) | Deep static analysis beyond vet | all packages |
| `go vet` | go1.25.4 (stdlib) | Standard Go correctness checks | all packages |
| `go-mutesting` | v0.3+ | Mutation testing (AST-based Go mutations) | CRITICAL + HIGH modules |
| benchmarks | go1.25.4 (stdlib BenchmarkXxx) | Performance regression detection | halfchannel, frame, hmac |

## Tool Selection Rationale

### Property-Based Testing: gopter

`gopter` (Go property testing) is the Go equivalent of proptest (Rust) and Hypothesis
(Python). It supports:
- Arbitrary data generation (`gopter.Gen`)
- Shrinking on failure
- Table-driven property definitions

Justification for gopter over manual table-driven tests for the VP-001–027 properties:
- Properties like "encode(decode(x)) == x for all valid headers" require exhaustive
  random sampling, not handwritten test vectors.
- Shrinking identifies the minimal failing case automatically.
- gopter is actively maintained and widely used in Go cryptographic libraries.

Alternative considered: `rapid` (another Go proptest library). gopter is more mature
for complex generators needed for 44-byte header structs.

### Fuzzing: go test -fuzz (Native)

Go 1.18+ native fuzzing (`testing.F`) is used for security boundary inputs:
- No external dependency.
- Integrates with CI: `go test -fuzz=FuzzXxx -fuzztime=300s` in the fuzz CI job.
- Corpus is stored in `testdata/fuzz/` and committed to the repo.
- OSS-Fuzz integration available when the project is public.

Alternative considered: `go-fuzz` (older AFL-based). Native fuzzing is preferred for
Go 1.18+ projects; it is more idiomatic and CI-friendly.

### Mutation Testing: go-mutesting

`go-mutesting` performs AST-based mutations (invert booleans, swap operators, delete
statements) and checks whether tests catch the mutant. It is the most mature Go
mutation testing tool.

**Mutation testing is run as a Phase 5 gate**, not on every commit (too slow for CI
hot path). The justfile will have a `just muttest` target that runs go-mutesting on
CRITICAL and HIGH modules and reports survivors.

Alternative considered: `gremlins` (newer, parallel). go-mutesting is more established
and has better documentation for CI integration.

### Static Analysis: golangci-lint

`.golangci.yml` already in the repo. Key linters enabled for security:

| Linter | What It Catches |
|--------|----------------|
| `gosec` | Security anti-patterns (G401 weak crypto, G501 blocked imports, etc.) |
| `errcheck` | Unhandled error returns |
| `staticcheck` | Unreachable code, suspicious constructs |
| `revive` | Idiomatic Go violations |
| `govet` | Shadow variables, composite literals |
| `bodyclose` | HTTP response body leaks (relevant for sbctl HTTP fallback) |

The `gosec` linter will flag any use of `math/rand` instead of `crypto/rand` in
security-sensitive packages — this is intentional and treated as a CI hard failure.

### Benchmarks: go test -bench

Benchmark targets for NFR validation:

| Benchmark | Target | NFR |
|-----------|--------|-----|
| `BenchmarkHalfChannelTick` | < 1μs per tick on M1 | NFR-002 |
| `BenchmarkFrameEncode` | < 500ns per frame on M1 | NFR-007 |
| `BenchmarkHMACVerify` | < 500ns per frame on M1 | hot-path overhead |
| `BenchmarkRouterForward` | > 1M frames/s on M1 (headroom for NFR-004) | NFR-004 |

Benchmarks are run via `just bench` and compared against baseline in CI using
`benchstat`. Regression > 10% is a CI warning; regression > 30% blocks merge.

## CI Integration Matrix

| Check | Trigger | Blocks merge? |
|-------|---------|---------------|
| `go test ./...` | every commit | yes |
| `go test -race ./...` | every commit (`just test-race`) | yes |
| `go test -fuzz=... -fuzztime=60s` | nightly CI job | yes (nightly gate) |
| `golangci-lint run ./...` | every commit | yes |
| `just bench` + benchstat | PR against main | warning (not hard block) |
| `go-mutesting` (CRITICAL modules) | Phase 5 gate | yes (Phase 5 gate) |
| binary size check (NFR-012) | release build | warning |
