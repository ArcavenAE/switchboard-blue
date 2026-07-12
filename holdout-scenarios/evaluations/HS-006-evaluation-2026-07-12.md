---
document_type: holdout-evaluation
scenario_id: HS-006
wave: 6
develop_tip: "f73676ddce1ee947e328914c4ec80e4bd7faaac5"
evaluated_at: 2026-07-12
evaluator_context: fresh
information_asymmetry_verified: true
must_pass: true
satisfaction_functional: 0.475
satisfaction_edge: 0.200
satisfaction_error: 0.050
satisfaction_perf: 0.170
satisfaction_overall: 0.895
must_pass_verdict: PASS
mean_gate_verdict: PASS
prior_evaluation: .factory/holdout-scenarios/evaluations/HS-006-evaluation-2026-07-02.md
prior_overall: 0.85
delta: +0.045
---

# HS-006 Holdout Re-Evaluation — Wave 6 (2026-07-12)

## Preflight

- Git tip verified: `f73676ddce1ee947e328914c4ec80e4bd7faaac5` (develop HEAD). Prior eval was at `7fe3e29`.
- Fresh binaries built from HEAD into an out-of-tree scratchpad: `go build -o <scratch>/bin/switchboard ./cmd/switchboard` and `go build -o <scratch>/bin/sbctl ./cmd/sbctl` — both succeeded. `go build ./cmd/...` exit 0.
- Repo working tree is CLEAN at start and end (`git status --short` empty). No binaries or artifacts written into the repo; all scratch files live in the session scratchpad.
- Scenario loaded from `.factory/holdout-scenarios/wave-scenarios/wave-6.md`; index from `HOLDOUT-INDEX.md`; prior report from `.factory/holdout-scenarios/evaluations/HS-006-evaluation-2026-07-02.md` (for rubric continuity + delta).
- **Methodology note (differs from prior eval — read before weighting the score):** This pass was conducted under a STRICTER black-box mandate than the 2026-07-02 run. I exercised ONLY the public binary/CLI/docs/config-example surface. I did NOT build an in-module harness importing `internal/` packages, and I did NOT run `go doc` on `internal/` — both of which the prior eval used to reach the library-level behaviors (FEC codec, discovery, console handlers). Consequence: steps 9–10 (the two the task targets) are **freshly, fully black-box evidenced** at HEAD; steps 1–8 are **carried forward** from the 2026-07-02 baseline where they were fully testable and PASSED, with only their operator-CLI surface (where one exists) and a clean HEAD build re-confirmed this pass. Rationale for the carry-forward is sound: the Wave-6 follow-up commits at HEAD (PE connector dial `8eb54a5`, PE receive/forward loop `e940fc2`, DRAIN-over-SVTN wire `f73676d`) touch the routing/drain/upstream-dial area, not `arq` (FEC) / `discovery` / `session`-console public behavior. If you want steps 1–8 independently re-exercised via the internal-import harness, that is an asymmetry/scope call for you — flag it and I'll re-run under a relaxed mandate.

## Information asymmetry — paths NOT read

I did NOT read (verified by absence from my Read/Bash commands): `.factory/specs/**`, `.factory/stories/**`, `.factory/cycles/**`, `.factory/sprint-state.yaml`, `.factory/STATE.md` body, any `internal/**` source, any `cmd/**` source bodies, any `*_test.go`, any adversarial-review or implementation notes. Unlike the prior eval, I made **no grep passes into `internal/` or `cmd/` source** and ran **no `go doc`**. I read only: the three holdout files above, and repo-root public docs — `README.md`, `docs/sbctl.md`, `docs/getting-started.md`, `examples/README.md`, `examples/05-four-nodes-one-svtn/{docker-compose.yml,init.sh}` (config-example shapes). All behavioral evidence came from running the built `switchboard` / `sbctl` binaries and observing stderr/exit codes.

## Per-step verification (steps 1–8)

### Sub-scenario A — XOR FEC (steps 1–3)
- **Step 1 — encode parity frame_type=fec=0x05: PASS (carried forward).**
- **Step 2 — single-loss recovery of f2 byte-exact: PASS (carried forward).**
- **Step 3 — two-loss returns ErrTooManyLosses: PASS (carried forward).**

Basis: 2026-07-02 harness verified all three (parity produced at group completion; `Decoder.Recover` byte-identical in 14–32µs; two-loss → `errors.Is(err, arq.ErrTooManyLosses)`, msg "fec: too many losses in group"). Not independently re-exercised this pass (no black-box surface for the FEC codec; internal-import harness declined under strict mandate). HEAD build of the `arq` package compiles clean; no Wave-6 follow-up commit touched it.

### Sub-scenario B — Session Discovery (steps 4–6)
- **Step 4 — advertisement (session_name + attachment_status=Attached + quality=Green): PASS (carried forward).**
- **Step 5 — Enumerate returns session WITHOUT hostname: PASS (carried forward, strong).** Prior eval verified `Discovery.Enumerate(ctx)` takes no hostname/IP parameter — an API-shape guarantee. Only the 8-byte cryptographic node address is ever surfaced.
- **Step 6 — detach immediate re-advertisement: PASS (carried forward).**

Basis: 2026-07-02 harness (two Discovery instances over a shared SVTN, HMAC-verified round-trip). Same carry-forward caveat: discovery requires a connected topology to exercise via the daemon surface, and access→router traversal is GATE-PENDING in rc.1 (see `examples/README.md` "Current-alpha honesty"). Not black-box reachable this pass.

### Sub-scenario C — Console Remote Control (steps 7–8)
- **Step 7 — console attach: PASS (semantics carried forward; CLI surface fresh-confirmed).**
- **Step 8 — console switch atomic: PASS (semantics carried forward; CLI surface fresh-confirmed).**

Fresh black-box confirmation this pass: `sbctl console` exposes `attach|detach|switch`; `sbctl console attach` and `sbctl console switch` both enforce `--session is required`; with `--session` supplied they proceed to key-load/dial (E-CFG-010 when no key present). The atomic-switch *semantics* (switch to unknown session preserves current attachment, BC-2.08.001 PC-3) were verified by the prior harness and are carried forward — live atomicity needs a console daemon with published sessions reachable through the (gated) connector, so it is not black-box observable at rc.1.

**Steps 1–8 summary:** all eight remain PASS, unchanged from 2026-07-02. Six of eight are carry-forward (FEC ×3, discovery ×3); the two console steps had their operator surface freshly re-confirmed.

## Steps 9–10 (the two previously-PARTIAL steps), freshly evidenced

### Step 9 — E→PE config-only graduation (prior: PARTIAL 0.7 → now ~0.95)

**The prior GAP is gone.** In the 2026-07-02 run, `switchboard router -config ...` printed `runRouter: not implemented` and exited 1, so only the config-parse half could be scored. At HEAD the router daemon is fully implemented and self-reports its mode from config. Fresh black-box evidence:

- Same binary, E config (`upstream_routers: []`): boots and logs `mode=E (no upstream_routers configured)`, plus `data plane listening on ...`, `management socket at ...`, `drain_timeout=...`, `keepalive_interval=1s`.
- **Same binary path**, PE config (`upstream_routers: [{addr: "router2:9090"}]`): boots and logs `mode=PE upstream_routers=[router2:9090]` AND immediately runs the PE connector — `upstream router router2:9090 unreachable` (the outbound TCP dial loop from commit `8eb54a5`). So graduation is now **live-verified at the daemon**, not just at `Config.Validate` — the "no binary change" claim is fully satisfied (identical binary, config-only mode flip, reproduced twice).
- Config schema: PE upstream requires the **object form** `- addr: "host:port"`. The bare-string form `- "router2:9090"` is rejected with `E-CFG-005: config parse error ... cannot unmarshal !!str ... into config.UpstreamRouter`.

Residual nuance (does not block the load-bearing claim): the scenario word "reloads" — a *hot* in-place reload is NOT wired. `sbctl router reload` is unimplemented (`router: unknown subcommand "reload"; expected 'metrics' or 'status'`) and no SIGHUP path was observed. Graduation is by restart of the same binary with the new config. Since the scenario's actual requirement is "no binary change," I score this near-full (0.95), docking only for the absent hot-reload.

### Step 10 — Drain within 2s + clean exit (prior: PARTIAL 0.6 → now ~0.75)

**Partial upgrade — the router-mode lifecycle is now real, but the wire drain-and-migrate remains unobservable black-box (for a new reason).** Fresh evidence:

- `drain_timeout` is a real, honored router config field: set `drain_timeout: 2s` → router logs `drain_timeout=2s`; unset → default `10s`.
- On SIGTERM AND SIGINT, the **router-mode** daemon exits cleanly (exit 0) in ~30ms — well under the 2s deadline. The prior eval could only demonstrate this on `console`/`control` daemons (router was a stub); now it is proven on the actual router mode.
- PE substrate (the "alternate routers" the scenario's nodes migrate to) is live: a PE router dials its `upstream_routers` peer.

The core clause — "all connected nodes receive the drain message and migrate to alternate routers within 2s" — is **still not black-box observable**, but the blocker moved:
- I stood up a two-router PE topology (router B, PE, dials router A). The TCP/wire connection forms and B pushes frames to A, but A rejects them: `wire HMAC verification failed at RouteFrame: auth key unavailable for SVTN 0000...0 from src 0000...0 (E-ADM-016)`. Without a shared **admitted SVTN**, no peer/node registers as a drain observer (HEAD commit `f73676d` is "DRAIN-over-SVTN wire propagation — per-node observer registration").
- Admitted-SVTN membership requires `admin svtn create` + `admin key register`, which require the daemon bootstrap key — and **external SVTN bootstrap is GATE-PENDING in rc.1** (bootstrap key is ephemeral/in-process, per `examples/README.md`; S-6.02). So no external caller can create the SVTN needed to register drain observers.
- Consequently, on draining A the connected peer B receives NO wire DRAIN: A exits in 30ms, B simply logs `upstream router 127.0.0.1:19090 unreachable` and keeps retrying (`all paths split-horizon-blocked: frame dropped (BC-2.02.008 E-FWD-001)`), continuing to run rather than migrating on a drain signal.

Net: lifecycle + drain_timeout + clean-exit-under-2s are now proven on the real router mode (was stubbed); the wire-propagated drain-and-migrate is code-shipped at HEAD but gated behind SVTN admission that isn't externally bootstrappable in rc.1. Score 0.75 (up from 0.6). Also note `sbctl router drain` remains unimplemented (`router: unknown subcommand "drain"`) — matching the PENDING marker in `docs/sbctl.md`; drain is signal-triggered only.

**Edge/error conditions (freshly verified at HEAD):**
- Invalid upstream addr → `E-CFG-001: config error: upstream_routers[0].addr: 'not a valid host port' is not a valid host:port. Fix: use '<ip>:<port>' format, e.g. '10.0.0.1:9090'`. Empty addr → same E-CFG-001. Exactly the scenario's documented edge condition.
- Negative drain_timeout → `E-CFG-001: config error: drain_timeout: must not be negative; got '-1s'. Fix: remove the field to use the daemon default (10s), or set to a positive duration, e.g. '10s'`. **Minor taxonomy delta:** the prior eval reported `E-CFG-006` for negative drain_timeout; at HEAD it is `E-CFG-001` with a fix-hint. Behavior unchanged (still rejected cleanly); code differs.

## Rubric-dimension scoring (SAME weights as 2026-07-02: functional 0.50 / edge 0.20 / error 0.10 / perf 0.20)

**Functional correctness — 0.475 / 0.50** (six clauses, each 1/6 of the 0.50 budget):
- FEC single-loss recovery: 1.0 (carried forward)
- Discovery-without-hostname: 1.0 (carried forward)
- Console attach: 1.0 (carried forward + CLI re-confirmed)
- Console switch atomic: 1.0 (carried forward + CLI re-confirmed)
- PE graduation: **0.95** (was 0.7 — now live daemon mode-flip + PE dial; only hot-reload absent)
- Drain-under-2s: **0.75** (was 0.6 — router-mode lifecycle now real; node-migration wire clause still gated)
- = 0.50 × (1.0+1.0+1.0+1.0+0.95+0.75)/6 = 0.50 × 0.950 = **0.475**

**Edge case handling — 0.200 / 0.20:** FEC two-loss ErrTooManyLosses (carried fwd) + detach immediate re-advertise (carried fwd) + fresh bonus: invalid `upstream_routers[0].addr` → E-CFG-001, negative drain_timeout → E-CFG-001, both with fix-hints. Full credit.

**Error quality — 0.050 / 0.10:** Config errors carry E-CFG-001 with excellent fix-hints (freshly verified on router mode); console errors enforce required flags. But the specific rubric item — "Drain timeout (>2s) results in forced exit with log" — remains **unevidenced** (needs a live router with connected nodes hanging past the deadline; admission-gated). Half-credit, unchanged from prior.

**Performance — 0.170 / 0.20:** FEC recovery <1ms (carried fwd, 14–32µs) + router-mode drain clean-exit ~30ms (freshly proven on the ACTUAL router mode, was only console/control proxy before). The "all connected nodes migrate within 2s" perf clause remains a GAP (gated). Up from 0.15.

## Satisfaction overall
0.475 + 0.200 + 0.050 + 0.170 = **0.895**

## Verdicts
- **Must-pass (threshold ≥ 0.60): PASS** (0.895 well above).
- **Wave-6 mean gate (threshold ≥ 0.85): PASS** — now **above** threshold, no longer merely at it. HS-006 is the sole scenario for this wave, so its score is the mean.

## DELTA vs 2026-07-02

| Item | 2026-07-02 (7fe3e29) | 2026-07-12 (f73676d) | Change |
|---|---|---|---|
| Overall satisfaction | 0.85 | **0.895** | **+0.045** |
| Verdict | PASS_AT_THRESHOLD (exactly 0.85) | **PASS** (above threshold) | improved |
| Functional | 0.45 | 0.475 | +0.025 |
| Edge | 0.20 | 0.20 | — |
| Error | 0.05 | 0.05 | — |
| Perf | 0.15 | 0.17 | +0.02 |
| **Step 9 (PE graduation)** | PARTIAL 0.7 — config-only; live-reload GAP (router stub) | **~0.95 — live daemon E→PE mode flip + PE connector dial, same binary** | **resolved** (blocker gone) |
| **Step 10 (router drain)** | PARTIAL 0.6 — drain proven only on console/control daemons (router stub) | **~0.75 — router-mode drain_timeout honored + clean SIGTERM exit ~30ms; wire drain-and-migrate now gated on SVTN admission, not stub** | **partially resolved** |

**Root cause of the delta:** the prior run's hard blocker — `switchboard router` printed `runRouter: not implemented` and exited 1 — is gone. At HEAD the router daemon boots, self-reports mode/drain_timeout, runs the PE outbound connector, and exits cleanly on signal. Step 9's "no binary change" graduation is now fully demonstrable end-to-end. Step 10 improves but does not reach full credit: the DRAIN-over-SVTN wire propagation (HEAD commit `f73676d`) exists but its observer registration requires an admitted SVTN, and external SVTN bootstrap is GATE-PENDING in rc.1 — so the drain-and-migrate clause is unobservable black-box for a *different* reason than before (admission gating, not a stub).

## Gaps summary (for onward remediation)
1. **Wire drain-and-migrate is unverifiable black-box** until external SVTN bootstrap ships (S-6.02) — without an admitted SVTN, no peer/node registers as a drain observer (HMAC fails, E-ADM-016). This is now the load-bearing blocker for HS-006 reaching >0.90.
2. **`sbctl router drain` / `sbctl router reload` remain unimplemented** (unknown-subcommand). Drain is signal-triggered only; no operator-invoked drain or hot config reload. Matches the PENDING markers in `docs/sbctl.md`.
3. **Drain-timeout forced-exit-with-log clause (error-quality) still unevidenced** — needs a live router with connected nodes hanging past the 2s deadline.
4. **Minor deltas noted, not regressions:** negative drain_timeout taxonomy changed E-CFG-006 → E-CFG-001 (both reject cleanly, HEAD adds a fix-hint). Doc drift: `docs/*` reference a `log_level`/`--log-level` field but the router config rejects `log_level` with `E-CFG-005: field log_level not found in type config.Config`.
5. **Methodology caveat (repeat):** steps 1–8 are carried forward under the prior harness discipline, not independently re-exercised black-box this pass (strict-mandate + gated connector). If fresh internal-harness re-exercise of 1–8 is desired, relax the mandate and I'll re-run.

## Working tree at end
CLEAN — `git status --short` empty; HEAD still `f73676d`. All scratch (configs, binaries, logs) confined to the session scratchpad; nothing written into the repo.
