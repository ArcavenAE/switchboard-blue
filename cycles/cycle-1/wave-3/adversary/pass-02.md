---
Title: Wave 3 Integration Gate — Adversarial Convergence Pass 2 (fresh context, independent)
Tree: develop @ b68e498
Verdict: NOT_CONVERGED — 0C / 1H / 2M / 4 OBS

BLOCKING HIGH:
- F-1 (HIGH) internal/routing/routing.go:138-146 (Router struct 61-65) — BC-2.05.008 PC-2: spec mandates "E-ADM-016 is logged at the router before return" on every HMAC failure. Router has NO logger field; RouteFrame returns ErrHMACVerificationFailed at lines 139 & 145 with no log emission. Comment at line 105 falsely claims "log E-ADM-016 → return". Forged/unverifiable frame dropped SILENTLY — no observability for active wire-layer forgery against the security boundary. Violates SOUL #4 (no silent failure on security path). Fix: add injectable logger to Router (mirror tmux.Logger; not init()), emit E-ADM-016 with svtn_id/src_addr before both returns (implementer) + log-assertion test (test-writer). If logging deliberately deferred, PC-2+VP-058 must be amended (product-owner/architect).
MEDIUM/OBS:
- F-2 (MEDIUM) routing_test.go:717,751; example_test.go:162 — tests carry "E-ADM-016 logged" comments but assert ONLY errors.Is(err, ErrHMACVerificationFailed); logging postcondition has zero test coverage. Fix: inject buffer logger, assert E-ADM-016 line present with src/svtn fields (test-writer).
- F-3 (MEDIUM) tmux SessionConnector → session DeliverFrame seam — tmux emits halfchannel.ChannelFrame; AccessNode.DeliverFrame consumes frame.OuterHeader; no production code bridges them and SessionConnector exposes no unified Frames(). Per ARCH-08 the bridge belongs in cmd/switchboard (position 18, planned) → later-wave obligation, NOT a Wave-3 violation, but absence of mode-transparent Frames() is a latent seam defect. Owner: architect (deferred-integration → phase-5).
- F-4 (OBS) cmd/switchboard/main.go — version stub; no subsystems wired (no NewRouter/NewAccessNode/SessionConnector in cmd/). Within ARCH-08 scope (position 18 target/planned). No E2E path exercised yet; wiring story must re-run this gate.
- F-5 (OBS) routing — BC-2.05.008 EC-006 (≥5 HMAC fails/60s alert) delegated by spec to BC-2.05.005 PC-3 counter; transitively unsatisfiable until F-1's emit path feeds the counter.
- F-6 (OBS) auth.go:30 RoleFull==iota==0 (zero value most-permissive); documented + entries only via RegisterKey so currently safe; latent fail-open if authEntry{} ever constructed directly. Place a deny sentinel before RoleFull if added later.
- F-7 (OBS) auth_test.go:217 VP-012 guard greps routing source for Tier2/TierTwo — brittle (passes if differently-named); low risk.
POSITIVES (mutation-resistant, verified): HMAC ordering VP-058/BC-2.05.008 PC-3 (reorder-killing tests, distinct ErrHMACVerificationFailed vs ErrNotAdmitted); wire-tag-before-zeroing anti-tautology guard (routing_internal_test.go:57); Tier-2 fail-closed (Authorize denies missing-session+missing-key; Allow empty-tick accepted, payload rejected); SendKeystroke sinkMu-first TOCTOU fix; go.md rule-12 respected (Snapshot/ListSessions value copies; Deliver RLock across loop); E-SES-005 absent from code; E-ADM-006/007/016 sentinels correct.
Novelty: MODERATE. F-1 is a substantive spec-anchored P0 security-observability gap, not a reword.
---
