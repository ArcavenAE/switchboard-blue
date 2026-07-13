DISPOSITION: APPROVE

## PR Reviewer — re-review of fix delta (`dc8c88c..95a9d6a`)

Round-1 blockers B1 and B2 and the security LOW finding are all resolved. Head `95a9d6a`. This re-review is scoped to the delta only.

### B1 — `## Blast Radius` declaration — RESOLVED
The PR body now carries the required literal `## Blast Radius` section; the "Declaration present" CI check is green on this head. Verified independently via `gh pr checks`.

### B2 — `-race` flake in `TestRunRouter_NodeConnClose_CleansUpSendMap` — RESOLVED (genuine, not masked)
`d2a6208` replaces the single un-retried `net.Dial` with a poll-retry loop:

- It is a byte-for-byte mirror of the suite's own established readiness pattern in `TestRunRouter_DataListenerBinds` (`mgmt_wire_test.go:451-462`): `net.DialTimeout("tcp", cfg.ListenAddr, 50ms)` retried on a 1s deadline with a 20ms backoff.
- The readiness signal is a **real TCP accept against the actual data-plane bind** (`cfg.ListenAddr`) — this is not a `time.Sleep` masking the race. It waits on the exact thing that was racing.
- Root cause is correctly diagnosed in the comment: this story's `wireRouterControlHandlers` adds two `srv.Register` calls before the data-plane bind (AC-013 register-before-serve), which widens the window between `startRunRouterWithConfig`'s mgmt-socket-ready return and `cfg.ListenAddr`'s bind. The old single dial raced that window under CI `-race` load.
- The failure path is preserved: on an exhausted 1s budget the test still `t.Fatalf`s, so a router that genuinely never binds still fails — the fix hardens against timing, it does not hide a real regression.
- Correct adaptation vs the reference: the successful `conn` is **kept open** (the test needs it for the registered/removed event assertions below), whereas `DataListenerBinds` closes immediately.

Quality Gate is green on `95a9d6a`.

### Security LOW (CWE-20/150) — `validateSVTNName` missing in `admin.svtn.status` — RESOLVED
`95a9d6a` adds `validateSVTNName(a.Name)` to `makeAdminSVTNStatusHandler`. Two things I want to call out, both to the fix's credit:

1. **Placement is correct and deliberately more careful than "mirror the siblings."** create/destroy validate *before* their authority gate; status validates *after* the admission gate (`resolveCallerAdmissionAnyRole`) and before `SVTNByName`. The comment explains why, and it is the right call: putting validation before admission would leak an oracle — an unauthorized caller sending a control-character name would see E-CFG-001 while one sending a well-formed name would see E-ADM-009. Post-fix, every unauthorized caller uniformly sees E-ADM-009 regardless of name shape, preserving the AC-006 byte-identical denied-path oracle.
2. **The byte-echo vector is fully closed.** The raw-name echo happens only in the not-found path (`SVTNByName` → `mapAdminError` → E-SVTN-003 with the name interpolated). That path is reachable only by an admitted caller, who now hits E-CFG-001 first. Unauthorized callers never reach it.

The RED test `TestAdminSVTNStatus_ArgsValidation_ControlCharacterName_E_CFG_001_NoByteEcho` genuinely proves the fix rather than asserting a tautology:
- It uses the bootstrap key, which I confirmed is the unconditional admission trust anchor in `resolveCallerAdmissionAnyRole` — so admission always clears and the test truly exercises the validation gate (not the admission gate).
- Its assertions are load-bearing: (a) an error is returned; (b) it is E-CFG-001; (c) it is **not** E-SVTN-003 — proving validation fired *before* the lookup that would echo the name; (d) the error does **not** contain the raw control bytes (`\x00`, `\x02`) — the actual no-byte-echo guarantee. Table-driven over NUL and STX.

### Regression check — clean
- Delta touches exactly 3 files. The only production change is the 3-line `validateSVTNName` call (+ comment) in `admin_handlers.go`; everything else is test code.
- All helpers used by the new test (`BuildAdminHandlers`, `newSVTNManagerForStatusTest`, `findAdminSVTNStatusHandler`, `callAdminSVTNStatusSafely`, `mgmt.WithCallerPubkey`) pre-exist and are shared with the existing AC-005/AC-006 status tests.
- Behavior change is strictly additive: admitted callers with malformed names now get E-CFG-001 (was E-SVTN-003); unauthorized callers are wholly unaffected (still E-ADM-009); well-formed names behave exactly as before.
- All CI checks green on `95a9d6a` (Analyze, CodeQL, Declaration present, Quality Gate, dependency-review, Harden-Runner).

### Residual observations (non-blocking, informational — no fix requested)
- **NIT (pre-existing, out of delta scope):** the top-of-handler `a.Name == ""` fast-path returns E-CFG-001 *before* the admission gate, so the empty-name case is the one input shape that still differs from the uniform E-ADM-009 denied response. This is not an existence oracle (empty string can never name a real SVTN, so it reveals nothing about existence) and it predates this PR, so it is not a regression. If the denied-path oracle is ever tightened further, moving that empty-check below admission would make the story fully uniform. Noting only for completeness.

Nothing here blocks merge. Both blockers and the security finding are genuinely fixed with correct, well-tested reasoning.

DISPOSITION: APPROVE
