# Demo Evidence Report — S-BL.NODE-IDENTIFY-SVTNID-CONSISTENCY

**Story:** S-BL.NODE-IDENTIFY-SVTNID-CONSISTENCY v1.0 — SVTNID consistency guard: ChallengeResponse svtn_id MUST match the NodeIdentify outer header svtn_id before AdmitNode is reached.
**HEAD:** 8db6c8535e47bee40c10500ee77898bab663b2ae
**BC anchor:** BC-2.01.009 PC-9 / EC-008; error-taxonomy v5.2 E-ADM-024
**E-ADM-024 canonical string:** `node_identify: ChallengeResponse svtn_id mismatch`
**Status:** CONVERGED
**Recorded:** 2026-07-21

## Coverage Matrix

| AC | Title | Test Function | Recording | Pass/Fail |
|----|-------|--------------|-----------|-----------|
| AC-001 | Matching SVTNID in ChallengeResponse → handshake proceeds to AdmitNode, binding recorded (regression: guard MUST NOT fire on valid path) | TestNodeIdentifyHandshake_Success_BindingRecorded | AC-001-matching-svtnid-proceeds-to-admitnode.tape | PASS |
| AC-002 | Mismatched ChallengeResponse svtn_id (admitted keyset) → connection closed BEFORE AdmitNode; LookupInterface returns (0,false) | TestNodeIdentifyHandshake_CRSVTNIDMismatch_ConnectionClosed_BeforeAdmitNode | AC-002-mismatched-svtnid-closed-before-admitnode.tape | PASS |
| AC-003 | Mismatch path → WARN log surfaces canonical E-ADM-024 string via onAccept default arm | TestNodeIdentifyHandshake_CRSVTNIDMismatch_WarnLogContainsE_ADM_024 | AC-003-warn-log-eadm024.tape | PASS |

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

## AC-003 — WARN log contains E-ADM-024 canonical string

**What it demonstrates:** On the mismatch path, the daemon emits a WARN log whose message contains the canonical error-taxonomy string `node_identify: ChallengeResponse svtn_id mismatch` (E-ADM-024). This log is produced via the `onAccept` default arm. The test asserts both that the log entry appears and that it contains the exact canonical string.

**Command:**
```
go test ./cmd/switchboard/ -run 'TestNodeIdentifyHandshake_CRSVTNIDMismatch_WarnLogContainsE_ADM_024' -count=1 -v
```

**Observed output (PASS):**
```
=== RUN   TestNodeIdentifyHandshake_CRSVTNIDMismatch_WarnLogContainsE_ADM_024
--- PASS: TestNodeIdentifyHandshake_CRSVTNIDMismatch_WarnLogContainsE_ADM_024 (0.05s)
PASS
ok  	github.com/arcavenae/switchboard/cmd/switchboard	0.385s
```

**BC trace:** BC-2.01.009 EC-008 / error-taxonomy v5.2 E-ADM-024 — the canonical error string `node_identify: ChallengeResponse svtn_id mismatch` is the normative identifier for this error condition in the error taxonomy; the WARN log assertion verifies the daemon surfaces it correctly.

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
- **Evidence integrity:** All three test runs were verified against HEAD `8db6c8535e47bee40c10500ee77898bab663b2ae` with `go test ... -count=1 -v` immediately before tape creation. Actual captured output is pasted verbatim in each AC section above.
- **AC-002 discriminating property:** The test uses an ADMITTED keyset deliberately — this proves the guard fires on svtn_id mismatch alone, not because the key is unknown. `LookupInterface` returning `(0, false)` is the strict assertion that AdmitNode was never called (no binding was ever recorded).
- **AC-003 onAccept arm:** The canonical E-ADM-024 string is emitted by the `onAccept` default arm in the daemon's connection-accept loop. The test captures the WARN log and asserts the exact substring is present.
