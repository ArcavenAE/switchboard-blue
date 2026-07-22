# Demo Evidence Report — S-BL.NODE-IDENTIFY-SVTNID-CONSISTENCY

**Story:** S-BL.NODE-IDENTIFY-SVTNID-CONSISTENCY v1.2 — SVTNID consistency guard: ChallengeResponse svtn_id MUST match the NodeIdentify outer header svtn_id before AdmitNode is reached.
**Code-complete at:** 8b667ce (guard + tests); demo/doc refinements in later commits on branch fix/node-identify-eadm024-log-context
**BC anchor:** BC-2.01.009 PC-9 / EC-008; error-taxonomy v5.2 E-ADM-024
**E-ADM-024 canonical string:** `node_identify: ChallengeResponse svtn_id mismatch`
**Status:** CONVERGED
**Recorded:** 2026-07-21

## Coverage Matrix

| AC | Title | Test Function | Recording | Pass/Fail |
|----|-------|--------------|-----------|-----------|
| AC-001 | Matching SVTNID in ChallengeResponse → handshake proceeds to AdmitNode, binding recorded (regression: guard MUST NOT fire on valid path) | TestNodeIdentifyHandshake_Success_BindingRecorded | AC-001-matching-svtnid-proceeds-to-admitnode.tape | PASS |
| AC-002 | Mismatched ChallengeResponse svtn_id (admitted keyset) → connection closed BEFORE AdmitNode; LookupInterface returns (0,false) | TestNodeIdentifyHandshake_CRSVTNIDMismatch_ConnectionClosed_BeforeAdmitNode | AC-002-mismatched-svtnid-closed-before-admitnode.tape | PASS |
| AC-003 (PC-1) | Mismatch path → WARN log contains canonical E-ADM-024 substring via dedicated errCRSVTNIDMismatch arm | TestNodeIdentifyHandshake_CRSVTNIDMismatch_WarnLogContainsE_ADM_024 | AC-003-warn-log-eadm024.tape | PASS |
| AC-003 (PC-3) | Mismatch path → WARN log includes real svtn_id hex context + greppable E-ADM-024 code literal | TestNodeIdentifyHandshake_CRSVTNIDMismatch_WarnLog_IncludesSVTNContextAndCode | AC-003-warn-log-eadm024.tape | PASS |

---

## AC-001 — Matching SVTNID proceeds to AdmitNode (regression guard)

**What it demonstrates:** When the ChallengeResponse svtn_id matches the NodeIdentify outer header svtn_id, the SVTNID consistency guard does NOT fire. The handshake continues to AdmitNode, the binding is recorded (LookupInterface returns true), and ServeConn begins. This is the non-false-positive regression test: BC-2.01.009 PC-9 must not block the valid path.

**Command:**
```
go test ./cmd/switchboard/ -run 'TestNodeIdentifyHandshake_Success_BindingRecorded' -count=1 -v
```

**Observed output (PASS):**
```
=== RUN   TestNodeIdentifyHandshake_Success_BindingRecorded
=== PAUSE TestNodeIdentifyHandshake_Success_BindingRecorded
=== CONT  TestNodeIdentifyHandshake_Success_BindingRecorded
--- PASS: TestNodeIdentifyHandshake_Success_BindingRecorded (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/cmd/switchboard	0.271s
```

**BC trace:** BC-2.01.009 PC-9 — the guard condition is `ChallengeResponse.svtn_id != NodeIdentify.svtn_id`; on the matching path the condition is false, so AdmitNode is reached and the binding is recorded.

---

## AC-002 — Mismatched SVTNID closes connection BEFORE AdmitNode

**What it demonstrates:** When the ChallengeResponse svtn_id does NOT match the NodeIdentify outer header svtn_id — even when the keyset IS admitted — the daemon closes the connection before calling AdmitNode. The test uses an admitted keyset to prove the discrimination is purely on the svtn_id mismatch, not on key rejection. `LookupInterface` returning `(0, false)` is the discriminating assertion that AdmitNode was never reached: no binding was recorded.

**Command:**
```
go test ./cmd/switchboard/ -run 'TestNodeIdentifyHandshake_CRSVTNIDMismatch_ConnectionClosed_BeforeAdmitNode' -count=1 -v
```

**Observed output (PASS):**
```
=== RUN   TestNodeIdentifyHandshake_CRSVTNIDMismatch_ConnectionClosed_BeforeAdmitNode
=== PAUSE TestNodeIdentifyHandshake_CRSVTNIDMismatch_ConnectionClosed_BeforeAdmitNode
=== CONT  TestNodeIdentifyHandshake_CRSVTNIDMismatch_ConnectionClosed_BeforeAdmitNode
--- PASS: TestNodeIdentifyHandshake_CRSVTNIDMismatch_ConnectionClosed_BeforeAdmitNode (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/cmd/switchboard	0.480s
```

**BC trace:** BC-2.01.009 PC-9 — guard condition fires; EC-008 — connection is closed with the E-ADM-024 error string before the handshake completes. `LookupInterface` returning `(0, false)` proves AdmitNode was NOT called.

---

## AC-003 — WARN log contains E-ADM-024 canonical string (PC-1 + PC-3)

**What it demonstrates:** On the mismatch path, the daemon emits a WARN log from the dedicated `errCRSVTNIDMismatch` arm introduced at `mgmt_wire.go:724`. This replaced the former `onAccept` default arm. The log message emits: `node_identify: ChallengeResponse svtn_id mismatch E-ADM-024 svtn=<hex>` — canonical substring first, then the greppable E-ADM-024 code literal, then the real NodeIdentify svtn as hex.

Two test functions cover the two post-conditions:
- **PC-1** (`TestNodeIdentifyHandshake_CRSVTNIDMismatch_WarnLogContainsE_ADM_024`): asserts the canonical substring `node_identify: ChallengeResponse svtn_id mismatch` is present.
- **PC-3** (`TestNodeIdentifyHandshake_CRSVTNIDMismatch_WarnLog_IncludesSVTNContextAndCode`): asserts the real svtn hex context (e.g. `svtn=ab00...`) AND the greppable `E-ADM-024` code literal are both present in the same log entry.

**Command (both tests):**
```
go test ./cmd/switchboard/ -run 'CRSVTNID' -count=1 -v
```

**Observed output (PASS):**
```
=== RUN   TestNodeIdentifyHandshake_CRSVTNIDMismatch_ConnectionClosed_BeforeAdmitNode
=== PAUSE TestNodeIdentifyHandshake_CRSVTNIDMismatch_ConnectionClosed_BeforeAdmitNode
=== RUN   TestNodeIdentifyHandshake_CRSVTNIDMismatch_WarnLogContainsE_ADM_024
--- PASS: TestNodeIdentifyHandshake_CRSVTNIDMismatch_WarnLogContainsE_ADM_024 (0.05s)
=== RUN   TestNodeIdentifyHandshake_CRSVTNIDMismatch_WarnLog_IncludesSVTNContextAndCode
--- PASS: TestNodeIdentifyHandshake_CRSVTNIDMismatch_WarnLog_IncludesSVTNContextAndCode (0.05s)
=== CONT  TestNodeIdentifyHandshake_CRSVTNIDMismatch_ConnectionClosed_BeforeAdmitNode
--- PASS: TestNodeIdentifyHandshake_CRSVTNIDMismatch_ConnectionClosed_BeforeAdmitNode (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/cmd/switchboard	0.591s
```

**BC trace:** BC-2.01.009 EC-008 / error-taxonomy v5.2 E-ADM-024 — the dedicated `errCRSVTNIDMismatch` arm at `mgmt_wire.go:724` is the normative emission point; PC-1 verifies the canonical substring; PC-3 verifies the svtn hex context and greppable code literal are co-present in the same log entry.

---

## Files

```
AC-001-matching-svtnid-proceeds-to-admitnode.tape
AC-002-mismatched-svtnid-closed-before-admitnode.tape
AC-003-warn-log-eadm024.tape
evidence-report.md
```

## Notes

- **Headless daemon story:** S-BL.NODE-IDENTIFY-SVTNID-CONSISTENCY is an internal wire-protocol guard on a headless daemon. There is no operator-facing CLI command or TUI surface. The only honest demo medium is `go test -run <TestName> -v` showing each AC's test passing with its discriminating assertion — consistent with the S-BL.NODE-IDENTIFY-WIRE precedent.
- **POL-004 compliance:** Only `.tape` scripts and `evidence-report.md` are committed. No `.gif`/`.webm`/`.mp4`/`.png`/`.jpg`/`.jpeg` binaries. The `.gitignore` at lines 58-63 excludes rendered artifacts from `docs/demo-evidence/**/*`; this directory structure ensures compliance.
- **Evidence integrity:** All test runs were verified against the code-complete state (guard commit 8b667ce; tests unchanged since). Subsequent commits on this branch are documentation-only (evidence-report + comment refreshes) and do not affect test behavior. Actual captured output is pasted verbatim in each AC section above.
- **AC-002 discriminating property:** The test uses an ADMITTED keyset deliberately — this proves the guard fires on svtn_id mismatch alone, not because the key is unknown. `LookupInterface` returning `(0, false)` is the strict assertion that AdmitNode was never called (no binding was ever recorded).
- **AC-003 dedicated arm:** The canonical E-ADM-024 string is emitted by the dedicated `errCRSVTNIDMismatch` arm at `mgmt_wire.go:724`, which replaced the former `onAccept` default arm. Two test functions cover PC-1 (canonical substring) and PC-3 (svtn hex context + greppable code literal) independently.
