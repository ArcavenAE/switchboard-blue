---
artifact_id: ARCH-09-purity-boundary-map
document_type: architecture-section
level: L3
version: "1.0"
status: draft
producer: architect
timestamp: 2026-06-23T00:00:00
phase: 1b
traces_to: ARCH-INDEX.md
inputDocuments:
  - '.factory/specs/module-criticality.md'
  - '.factory/specs/architecture/ARCH-05-cli-and-api.md'
kos_anchors:
  - elem-timeslice-framing
  - elem-ssh-end-to-end-encryption
---

# ARCH-09: Purity Boundary Map

## Bucket Definitions

| Bucket | Definition | Verification Approach |
|--------|-----------|----------------------|
| **pure-core** | Deterministic functions: same input always produces same output. No I/O, no globals, no `time.Now()`, no `rand`, no OS calls. | Property tests, fuzz tests, mutation testing |
| **boundary** | Parses/serializes external formats, or adapts between pure-core and effectful. May hold mutable state under mutex. | Unit tests + integration tests |
| **infrastructure** | I/O wrappers: network, file system, OS sockets. Pure wrappers with no business logic. | Integration tests |
| **effectful** | Network I/O, disk I/O, clocks, OS signals, goroutine lifecycle. Business logic NOT allowed here. | Integration tests + race detector |

The rule: **business logic lives in pure-core or boundary; effectful packages call
pure-core, they do not implement logic.**

## Per-Package Classification

| Package | Bucket | Rationale | Formal Verification Target |
|---------|--------|-----------|---------------------------|
| `internal/frame` | **pure-core** | Encoding/decoding is a pure transformation over byte slices and structs. No I/O, no state. | Yes — proptest + fuzz (VP-001–003, VP-014) |
| `internal/hmac` | **pure-core** | MAC computation is a pure function: `(key, data) → tag`. No I/O. | Yes — proptest + fuzz (VP-004–006) |
| `internal/config` | **pure-core** | Config parsing and validation are pure: `([]byte YAML) → (Config, []error)`. No file I/O in the pure layer; file reading is in the caller. | Yes — proptest (VP-028–029) |
| `internal/halfchannel` | **pure-core** | The `HalfChannel` state machine is pure: `Tick(state, payload) → (newState, frame)`. The ticker goroutine is in the effectful caller. | Yes — proptest (VP-016–018) |
| `internal/arq` | **pure-core** | ARQ is a pure state machine: `(ARQState, frame/ack) → (newState, [frames_to_deliver], [frames_to_retransmit])`. No I/O. | Yes — proptest (VP-019–021) |
| `internal/replay` | **pure-core** | Replay window is a pure function over a ring buffer: `(state, frame) → (newState, deduplicated_frames)`. | Yes — proptest (VP-022–023) |
| `internal/multipath` | **pure-core** | Drop cache and duplicate detection: `(cache, frame) → (cache', should_deliver)`. Pure over the cache data structure. | Yes — proptest (VP-024–025) |
| `internal/paths` | **pure-core** | Path scoring and ranking: `(path_metrics[]) → ranked_paths[]`. EWMA calculation is a pure function. | Yes — proptest (VP-026) |
| `internal/metrics` | **pure-core** | Quality indicator computation: `(path_metrics, thresholds) → QualityState`. Pure transition function. | Yes — proptest (VP-027) |
| `internal/admission` | **boundary** | Holds admitted key set (mutable under mutex). Admission logic is pure; the key set is mutable state. Nonce verification is deterministic given the nonce store. | Partial — proptest for logic (VP-007–009), integration for key store mutation |
| `internal/session` | **boundary** | Holds per-session authorized key list (mutable under mutex). Authorization logic is pure; the key list is mutable state. | Partial — proptest for auth logic (VP-012–013), integration for lifecycle |
| `internal/routing` | **boundary** | Holds SVTN forwarding table and admitted node map (mutable under mutex). Routing decisions are pure; the forwarding table is mutable state. | Partial — proptest for routing logic (VP-010–011), fuzz for channel header opacity (VP-015), integration for SVTN isolation |
| `internal/discovery` | **boundary** | Parses presence advertisements; maintains session list (mutable). Presence serialization/deserialization is pure. | Integration tests |
| `internal/svtnmgmt` | **boundary** | Manages SVTN lifecycle and key registration. Calls into `internal/admission` for key store mutations. | Integration tests |
| `internal/tmux` | **effectful** | Connects to tmux via Unix socket. Reads `%output` events. Manages the control mode connection lifecycle. No business logic in this package. | Integration + race detector |
| `internal/drain` | **effectful** | Sends DRAIN_SIGNAL over network connections. Manages shutdown timer. No business logic. | Integration tests |
| `cmd/switchboard` | **effectful** | Entry point: parses flags, reads config file, builds dependency graph, starts goroutines. No business logic. | Integration smoke tests |
| `cmd/sbctl` | **effectful** | CLI: parses flags, makes RPC calls over socket, formats output. No business logic. | Integration + e2e |

## Purity Enforcement Rules

1. **Pure-core packages MUST NOT import**: `net`, `os`, `syscall`, `time` (except `time.Duration`
   as a data type), `math/rand`, `crypto/rand`, any `internal/tmux` or `internal/drain`.

2. **Boundary packages MAY import**: pure-core packages. They hold mutable state under
   `sync.RWMutex`. They MUST NOT perform network I/O.

3. **Effectful packages**: Hold all I/O. They MUST NOT contain business logic.
   Logic that is tempted to live here belongs in a pure-core or boundary package.

4. **Testing implication**: Pure-core packages have `_test.go` files using only
   `testing` and `github.com/leanovate/gopter` (proptest). No test helpers that
   open network connections.

## Purity Classification Summary

| Bucket | Count | Packages |
|--------|-------|---------|
| pure-core | 9 | frame, hmac, config, halfchannel, arq, replay, multipath, paths, metrics |
| boundary | 5 | admission, session, routing, discovery, svtnmgmt |
| effectful | 4 | tmux, drain, cmd/switchboard, cmd/sbctl |
| **Total** | **18** | |
