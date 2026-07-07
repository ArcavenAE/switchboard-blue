# finding-001 — examples ladder: what the published alpha actually implements

**Date:** 2026-07-06
**Probe:** build `examples/` — docker-compose proofs of functionality
using the published alpha binaries (`alpha-20260706-203527-62e38d3`,
installed in Linux containers from GitHub Releases, no source build).
**Status:** complete — six examples, all operator containers exit 0.
(The verifying compose service was later renamed driver → operator to
match spec vocabulary.)

## What was proven (distribution-verified, linux/arm64 containers)

1. **Router daemon end-to-end** (ex 01): config validate → data-plane
   TCP listener reachable across network namespaces → management unix
   socket → Ed25519 challenge-response with a configured
   `authorized_operator_keys` key → real RPC round-trips (`router
   status`, `paths list`, `--json` valid). Rogue key → `E-ADM-010`.
2. **Two-layer authority model** (ex 02): authenticated operator ≠
   authorized admin. `admin svtn create` → `E-ADM-009 ... has role
   unregistered` (layer 2) vs `E-ADM-010` (layer 1). Fails closed.
3. **Access mode survives with a live tmux backend on Linux** (ex 03,
   05): first exercise of the session-backend connection anywhere —
   macOS dev machines die on `/dev/ttysNN` permissions (the known
   tier-2 limitation). Four concurrent access daemons hosting
   top/htop/watch/vmstat all stayed healthy (ex 05).
4. **Console session-plane taxonomy** (ex 04): `E-SES-001`,
   `E-SES-004` behind two admission tiers; usage errors exit 2
   client-side; console mgmt socket is loopback-TCP-only (`E-CFG-008`).
5. **Key-based isolation matrix** (ex 06): disjoint operator keys on a
   shared router → 4/4 cross-team denials with `E-ADM-010`, 4/4
   own-team successes. Shared transport ≠ shared authority.

## Gaps found (the honest part)

- **G1 — sbctl rejects PKCS#8 ed25519 private keys:**
  `E-CFG-010 ... not an Ed25519 private key (got ed25519.PrivateKey)`.
  `ssh.ParseRawPrivateKey` returns `*ed25519.PrivateKey` for OPENSSH
  blocks but the value type for PKCS#8; the loader type-asserts only
  the pointer form. Workaround baked into
  `examples/_shared/gen-identity.sh`. → GH issue filed.
- **G2 — no external SVTN bootstrap:** `admin svtn.create` is
  bootstrap-only and the daemon bootstrap key is ephemeral/in-process
  (persistent wiring deferred to S-6.02), so *no external caller can
  create an SVTN in this alpha*. getting-started §3 is target behavior.
- **G3 — no daemon dials any other daemon:** the only network dials in
  the tree are sbctl's mgmt client. access→router→console session
  traversal is unwired; `sessions.list` is not a registered RPC
  anywhere; `sessions.status` exists only on console mode.
- **G4 — docs drift:** getting-started claimed sbctl wasn't on
  Homebrew (sbctl-a formula exists; fixed in this branch).

## Mechanism for the gaps: gated checks

`examples/_shared/harness.sh` adds `check_gated` — target-behavior
assertions that report `GATE-PENDING` today, auto-flip to `GATE-PASS`
when the feature lands, and become hard failures under `GATED=1`.
Examples 05/06 are thereby pre-built acceptance tests for the
connector + bootstrap milestones.

## Key operational facts worth remembering

- Operator identity needs **two formats**: OpenSSH private key for
  `sbctl --key`, SPKI PEM for `authorized_operator_keys`. Portable
  derivation (works on OpenSSH 9.2, no PKCS8 export support needed):
  raw 32-byte ed25519 key = last 32 bytes of the OpenSSH pubkey blob;
  SPKI = fixed 12-byte DER prefix `302a300506032b6570032100` + raw key.
- Console mgmt is loopback-TCP-only → compose operator containers use
  `network_mode: "service:console"`; all other modes use unix sockets
  shareable via a volume.
- Host-side proof of the same commit: all four `test/smoke/` tiers
  pass against the brew binaries symlinked into `bin/` (27/27
  assertions, macOS arm64).
