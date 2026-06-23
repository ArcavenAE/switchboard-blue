---
artifact_id: ARCH-06-deployment-and-ops
document_type: architecture-section
level: L3
version: "1.0"
status: draft
producer: architect
timestamp: 2026-06-23T00:00:00
phase: 1b
traces_to: ARCH-INDEX.md
inputDocuments:
  - '.factory/specs/prd.md'
  - '.factory/specs/prd-supplements/nfr-catalog.md'
  - '.factory/specs/behavioral-contracts/ss-09/BC-2.09.001.md'
  - '.factory/specs/behavioral-contracts/ss-09/BC-2.09.002.md'
  - '.factory/specs/behavioral-contracts/ss-09/BC-2.09.003.md'
kos_anchors:
  - elem-single-binary-three-modes
  - elem-mvp-scope-single-lan
---

# ARCH-06: Deployment & Ops

## ADR-007: P Router Build Target

**Decision:** The P (Provider Core) router is a **separate build target** (build tag
`providercore`), not included in the MVP binary. The E/PE router binary (`switchboard`)
does not contain P router code.

**Rationale:**
- P router is router-facing only (no node connections). Its code paths are fundamentally
  different from E/PE (which connects nodes).
- Including dead code in the MVP binary adds binary size without benefit (NFR-012: ≤20MB).
- Separate build target makes the P scope boundary explicit and prevents accidental
  use of P router APIs in E/PE code.

**BC-2.09.001 (E→PE graduation):** E→PE is a config change, not a build change.
`upstream_routers: []` = E; any entries = PE. Same binary, no reinstall.

## Single Binary Build

The E/PE binary is built with standard `go build`. Release artifacts:

| Platform | Target | Justfile target |
|----------|--------|----------------|
| macOS (arm64) | `switchboard-darwin-arm64` | `just build-all` |
| macOS (amd64) | `switchboard-darwin-amd64` | `just build-all` |
| Linux (amd64) | `switchboard-linux-amd64` | `just build-all` |
| Linux (arm64) | `switchboard-linux-arm64` | `just build-all` |

Build flags: `go build -trimpath -ldflags="-s -w -X main.version=$(VERSION)"`.
`-trimpath` removes local path prefixes from stack traces (security hygiene).
`-s -w` strips debug info to minimize binary size (NFR-012).

macOS binaries are codesigned via `just sign` (uses Developer ID or ad-hoc). Signature
verified via `just verify`. GitHub Actions release workflow runs sign+verify.

## Binary Size Budget (NFR-012: ≤ 20MB)

| Component | Estimated Size |
|-----------|---------------|
| Go runtime + standard library | ~6MB |
| golang.org/x/crypto (SSH/HMAC) | ~500KB |
| YAML config parser (gopkg.in/yaml.v3) | ~300KB |
| Application code | ~2–4MB |
| **Total (estimated)** | **~9–11MB** |

Budget is comfortable. If size exceeds 15MB, investigate with `go tool nm` before
adding UPX compression (UPX adds startup latency; not recommended for interactive tools).

## Config File Validation (BC-2.09.003, NFR-011)

Config validation is the first operation in each daemon mode handler. The binding
sequence is:

```
1. loadConfigFile(path) → Config struct
2. Config.Validate() → []ValidationError
3. if len(errors) > 0: printErrors(errors); os.Exit(1)
4. initLogger() (uses validated config)
5. bindListenSocket() (only after validation passes)
```

Any invalid config exits with code 1 and a message identifying the field, the
constraint violated, and a suggested fix. Example:
```
switchboard: config error: field 'tick_interval_upstream' = 3ms is outside
  allowed range [5ms, 50ms]. Suggestion: set to 10ms for interactive sessions.
```

## Upgrade Model

E Router phase: replace binary, restart daemon. No session migration (single router,
FM-001). Sessions reconnect automatically after restart.

PE phase: `sbctl router drain` → graceful drain (BC-2.09.002) → nodes migrate to
alternate routers → operator replaces binary → restart. Sessions survive if at least
one alternate router remains up.

## E→PE Graduation (BC-2.09.001)

```yaml
# Before (E router)
upstream_routers: []

# After (PE router) — add upstream connections
upstream_routers:
  - addr: "10.0.1.1:9090"
  - addr: "10.0.1.2:9090"
```

On next restart, the binary activates PE mode. The multipath and FEC code paths
(disabled in E mode) become active. No SVTN re-initialization required.

## Graceful Drain (internal/drain, BC-2.09.002)

```
drain sequence:
  1. Router receives SIGTERM or sbctl router drain
  2. Broadcast DRAIN_SIGNAL to all connected nodes
  3. Wait drain_timeout (default 10s) for nodes to migrate
  4. Close all remaining connections
  5. Exit cleanly
```

Nodes receiving DRAIN_SIGNAL initiate reconnection to alternate routers using their
`router_addrs` list. The drained router is removed from the rotation.

**FM-009 (crash without drain):** Nodes detect the absence via keepalive timeout
(after keepalive_interval, default 1s). After timeout, they attempt to reconnect to
the next router in `router_addrs`.

## Platform Support

| Platform | Status | Notes |
|----------|--------|-------|
| macOS (arm64/amd64) | Primary | Development platform |
| Linux (amd64/arm64) | Primary | Production target |
| Docker | Via `just test-docker` | CI isolation |
| Windows | Not supported | No PTY support; not a target persona platform |
