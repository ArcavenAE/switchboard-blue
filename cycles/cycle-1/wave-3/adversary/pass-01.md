---
Title: Wave 3 Integration Gate — Adversarial Convergence Pass 1
Tree: develop @ b68e498
Verdict: CONVERGED — 0C / 0H / 3M / 2L / 3O (zero CRITICAL, zero HIGH → wave-gate criterion met)

Scope: integrated review of all 5 Wave 3 stories as a working whole (S-3.04 HMAC/RouteFrame, S-3.01a tmux control mode, S-3.01b PTY fallback, S-3.02 console fan-out, S-3.03 Tier-2 auth + read-only). Critical framing: cmd/switchboard/main.go is a version-printing stub — NO production code wires Router + AccessNode + SessionConnector together, so the MEDIUMs are LATENT (no live caller can trigger them yet) but MUST be closed in the story that wires cmd/switchboard.

FINDINGS:
- W3-M-1 (MEDIUM) internal/routing/routing.go:144-146 — BC-2.05.008 PC-2: E-ADM-016 is never logged at the router on HMAC failure. The drop is enforced & tested; only the observability postcondition is unmet (Router has no logger field). Fix: inject a logger into Router (mirror tmux.Logger), emit E-ADM-016 before both ErrHMACVerificationFailed returns. Owner: implementer.
- W3-M-2 (MEDIUM) internal/tmux/pty_fallback.go SessionConnector — ADR-010/BC-2.04.001 PC-5/BC-2.04.002: SessionConnector exposes no Frames(); on control→PTY failover a consumer holding ctrl.Frames() loses output silently (old channel closed, pty.frames undrained). "No lost frames at switch" not structurally guaranteed. Mitigated: no consumer exists yet. Fix: add SessionConnector.Frames() returning a failover-stable aggregated channel, or document re-fetch-on-Err contract. Owner: architect (contract) + implementer.
- W3-M-3 (MEDIUM) internal/session/upstream.go:156-159 — BC-2.05.003 PC-2 (fail-closed): NewAccessNode(pub, nil) silently installs allow-all NoOpAuthorizer (latent fail-OPEN). Polarity opposite to the deliberately fail-LOUD noSink default. A future NewAccessNode(pub, nil, ...) compiles and disables all Tier-2 enforcement with no signal. Fix: default nil-auth to deny-all (fail-closed) or require non-nil. Owner: architect (polarity decision) + implementer.
- W3-L-1 (LOW, verified-inert) upstream.go:213 — attach-time Allow(key,sessionName,nil) is an auth probe only, does NOT call sink.SendInput; no spurious zero-tick on attach. Recorded so a future refactor doesn't route the attach probe through the sink.
- W3-L-2 (LOW) pty_fallback.go:560-566 / control.go:55 — control-mode flag-rejection classification falls back to brittle stderr string-matching (locale/version fragile). Primary path uses errors.Is (good). Fix: prefer exit-code/errors.Is, retire string fallback. Owner: implementer.
- W3-O-1 (OBS) routing.go — BC-2.05.008 EC-006 HMAC failure-rate alert (≥5/60s) not implemented; anchored to BC-2.05.005 PC-3, plausibly out of S-3.04 scope. Architect adjudication.
- W3-O-2 (OBS) cmd/switchboard/main.go — no integrating caller wires the five subsystems; integration is API-level only; the wiring story MUST re-run this gate.
- W3-O-3 (OBS) upstream.go:300 — empty-tick → tmux.SendInput([]byte{}) → stdin.Write([]byte{}) is a correct no-op but untested at the tmux seam.

[process-gap] Fail-loud/fail-open polarity inconsistency between KeystrokeSink default (noSink → fail-loud) and Authorizer default (NoOpAuthorizer → fail-open). No existing review axis catches security-perimeter default polarity. Suggest a constructor-default-polarity rule ("security-perimeter defaults must fail closed unless justified") in go.md/governance. Owner: rules/governance.

POSITIVES (mutation-resistant, verified): HMAC ordering VP-058/BC-2.05.008 PC-3 (tests distinguish ErrHMACVerificationFailed vs ErrNotAdmitted, reorder-killing); VP-012 router-no-Tier2-state grep guard genuine; fan-out Deliver holds RLock across loop (no close-during-send), go.md rule-12 respected; SendKeystroke sinkMu closes detach/evict TOCTOU; empty-tick two-assertion test mutation-resistant; error taxonomy fidelity (E-ADM-006/007 carry codes, E-SES-005 absent from code).

MANDATORY CARRY-FORWARD: all 3 MEDIUMs (esp. M-3 fail-open, M-2 frames seam) MUST be re-reviewed and closed in the cmd/switchboard wiring story.
---
