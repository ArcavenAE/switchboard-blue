# Demo Evidence Report — S-6.03: sbctl Client Auth, Connection Error Reporting

## Header

| Field | Value |
|-------|-------|
| Story | S-6.03 |
| Story spec version | 2.6 |
| HEAD SHA | ff1d14655e11d5e0d2bb409fe4a2824b01ed538e |
| Branch | feature/S-6.03-sbctl-client-auth |
| Date | 2026-06-29 |
| `go test -race` (cmd/sbctl) | PASS — all 26 test functions pass |
| Binary built | /tmp/sbctl-demo (go build ./cmd/sbctl) |

Overall race result:
```
ok  	github.com/arcavenae/switchboard/cmd/sbctl	2.722s
```

---

## Per-AC Evidence

### AC-001 — Ed25519 key loading (BC-2.07.002 PC-2)

**Criterion:** `--key <path>` loads Ed25519 key via `ssh.ParseRawPrivateKey` with `io.LimitReader(f, 1<<16)`.

**Proving tests:** `TestSbctl_KeyLoading_Ed25519` (client_test.go)

**Transcript:** [AC-001-008-key-loading-tilde-expansion.txt](AC-001-008-key-loading-tilde-expansion.txt)

```
--- PASS: TestSbctl_KeyLoading_Ed25519 (0.00s)
    --- PASS: TestSbctl_KeyLoading_Ed25519/well_formed_ed25519_key_loads_to_64_bytes (0.00s)
    --- PASS: TestSbctl_KeyLoading_Ed25519/file_larger_than_64KiB_is_rejected (0.01s)
    --- PASS: TestSbctl_KeyLoading_Ed25519/nonexistent_file_returns_error (0.00s)
```

---

### AC-002 — Authenticate() fail-closed (VP-067, BC-2.07.002 PC-2)

**Criterion:** `Authenticate()` returns nil ONLY on AUTH_OK; all other outcomes (AUTH_FAIL, malformed challenge, missing nonce, oversized response, deadline expiry, connection closed) return non-nil error. ctx-first signature per go.md rule 7.

**Proving tests:** `TestAuthenticate_FailClosed_VP067` (client_test.go) — 11 sub-cases

**Transcript:** [AC-003-auth-failure-eadm010.txt](AC-003-auth-failure-eadm010.txt)

```
--- PASS: TestAuthenticate_FailClosed_VP067 (0.01s)
    --- PASS: TestAuthenticate_FailClosed_VP067/VP067_a_connection_error_on_challenge_read (0.00s)
    --- PASS: TestAuthenticate_FailClosed_VP067/VP067_b_malformed_challenge_missing_nonce (0.00s)
    --- PASS: TestAuthenticate_FailClosed_VP067/VP067_b_malformed_challenge_nonce_not_base64url (0.00s)
    --- PASS: TestAuthenticate_FailClosed_VP067/VP067_b_malformed_challenge_json_decode_error (0.00s)
    --- PASS: TestAuthenticate_FailClosed_VP067/VP067_b_malformed_challenge_nonce_wrong_length (0.00s)
    --- PASS: TestAuthenticate_FailClosed_VP067/VP067_e_auth_fail_returns_error (0.00s)
    --- PASS: TestAuthenticate_FailClosed_VP067/VP067_f_wrong_response_type_returns_error (0.00s)
    --- PASS: TestAuthenticate_FailClosed_VP067/VP067_g_truncated_stream_after_challenge (0.00s)
    --- PASS: TestAuthenticate_FailClosed_VP067/VP067_happy_path_auth_ok_returns_nil (0.00s)
    --- PASS: TestAuthenticate_FailClosed_VP067/VP067_h_oversized_auth_response_bounded_by_limit_reader (0.01s)
    --- PASS: TestAuthenticate_FailClosed_VP067/VP067_i_deadline_expiry_server_silent (0.05s)
```

---

### AC-003 — Auth failure → E-ADM-010, exit 1 (BC-2.07.002 PC-4; BC-2.07.003 EC-005)

**Criterion:** AUTH_FAIL → `E-ADM-010 "authentication failed"` on stderr, exit 1, no stdout. Key load failure → `E-CFG-010 "key load failed: <path>: <reason>"`, no dial.

**Proving tests:** `TestSbctl_AuthFailure_ExitsOneWithEADM010`, `TestSbctl_KeyLoadFailure_ExitsOneWithECFG010` (main_test.go)

**Transcript:** [AC-003-auth-failure-eadm010.txt](AC-003-auth-failure-eadm010.txt), [AC-001-008-key-loading-tilde-expansion.txt](AC-001-008-key-loading-tilde-expansion.txt)

**Binary demo (key absent, E-CFG-010 fires before dial):**
```
$ /tmp/sbctl-demo --target=127.0.0.1:19995 --key=/nonexistent/key.pem router
stdout: (empty)
stderr: E-CFG-010 key load failed: /nonexistent/key.pem: open /nonexistent/key.pem: no such file or directory
Exit code: 1
```

```
--- PASS: TestSbctl_AuthFailure_ExitsOneWithEADM010 (0.03s)
--- PASS: TestSbctl_KeyLoadFailure_ExitsOneWithECFG010 (0.00s)
    --- PASS: TestSbctl_KeyLoadFailure_ExitsOneWithECFG010/missing_key_file (0.02s)
    --- PASS: TestSbctl_KeyLoadFailure_ExitsOneWithECFG010/oversized_key_file (0.01s)
    --- PASS: TestSbctl_KeyLoadFailure_ExitsOneWithECFG010/wrong_key_type_rsa (0.01s)
    --- PASS: TestSbctl_KeyLoadFailure_ExitsOneWithECFG010/malformed_pem_key_file (0.02s)
```

---

### AC-004 — Connection refused → E-NET-001, exit 1 (BC-2.07.003 PC-1, VP-030)

**Criterion:** Dial failure → `E-NET-001 "daemon unreachable: <address>: <reason>"` on stderr, exit 1, no stdout. Also covers E-RPC-001 for post-auth dispatch failure.

**Proving tests:** `TestSbctl_ConnectionRefused_ExitsOneWithENET001_VP030`, `TestSbctl_RPCDispatchFailure_ExitsOneWithERPC001` (main_test.go)

**Transcript:** [AC-004-007-connection-refused-timeout-enet001.txt](AC-004-007-connection-refused-timeout-enet001.txt)

**Binary demo:**
```
$ /tmp/sbctl-demo --target=127.0.0.1:19995 --key=<testdata/test_ed25519_key> router
stdout: (empty)
stderr: E-NET-001 daemon unreachable: 127.0.0.1:19995: dial tcp 127.0.0.1:19995: connect: connection refused
Exit code: 1
```

```
--- PASS: TestSbctl_ConnectionRefused_ExitsOneWithENET001_VP030 (0.02s)
--- PASS: TestSbctl_RPCDispatchFailure_ExitsOneWithERPC001 (0.03s)
```

---

### AC-005 — Zero stdout on failure (BC-2.07.003 PC-3)

**Criterion:** No stdout when daemon unreachable.

**Proving test:** `TestSbctl_NoStdoutOnConnectionFailure` (main_test.go)

**Transcript:** [AC-005-006-009-010-011-remaining-acs.txt](AC-005-006-009-010-011-remaining-acs.txt)

```
--- PASS: TestSbctl_NoStdoutOnConnectionFailure (0.02s)
```

---

### AC-006 — JSON envelope output (BC-2.07.002 PC-3, interface-definitions.md)

**Criterion:** `--json` produces `{"ok":true/false,"error":...,"data":...}` for both success and error.

**Proving test:** `TestSbctl_JSONEnvelopeFormat` (client_test.go)

**Transcript:** [AC-005-006-009-010-011-remaining-acs.txt](AC-005-006-009-010-011-remaining-acs.txt)

```
--- PASS: TestSbctl_JSONEnvelopeFormat (0.00s)
    --- PASS: TestSbctl_JSONEnvelopeFormat/success_envelope_ok_true_error_null_data_present (0.00s)
    --- PASS: TestSbctl_JSONEnvelopeFormat/error_envelope_ok_false_error_present_data_null (0.00s)
```

---

### AC-007 — Timeout → E-NET-001, no hang (BC-2.07.003 Inv-2, Ruling Y)

**Criterion:** Listener accepts TCP but never completes handshake → E-NET-001 "connection timed out", exit 1, within --timeout budget.

**Proving test:** `TestSbctl_ConnectionTimeout` (main_test.go)

**Transcript:** [AC-004-007-connection-refused-timeout-enet001.txt](AC-004-007-connection-refused-timeout-enet001.txt)

**Binary demo (--timeout=150ms, listener accepts but never responds):**
```
stdout: (empty)
stderr: E-NET-001 daemon unreachable: 127.0.0.1:29876: connection timed out
Exit code: 1
Elapsed: ~173ms (within 500ms budget)
```

```
--- PASS: TestSbctl_ConnectionTimeout (0.12s)
```

---

### AC-008 — Tilde expansion for --key default (BC-2.07.003 EC-007 + Precondition 3)

**Criterion:** `~/` expanded via `os.UserHomeDir()` before file-open; two failure sub-cases; `~username` treated as literal.

**Proving test:** `TestSbctl_TildeExpansion_DefaultKey` (client_test.go) — 4 sub-cases

**Transcript:** [AC-001-008-key-loading-tilde-expansion.txt](AC-001-008-key-loading-tilde-expansion.txt)

```
--- PASS: TestSbctl_TildeExpansion_DefaultKey (0.00s)
    --- PASS: TestSbctl_TildeExpansion_DefaultKey/happy_path_tilde_slash_expands_to_home (0.01s)
    --- PASS: TestSbctl_TildeExpansion_DefaultKey/sub_case_a_homedir_error_uses_original_path (0.00s)
    --- PASS: TestSbctl_TildeExpansion_DefaultKey/sub_case_b_expansion_ok_but_file_missing_uses_expanded_path (0.01s)
    --- PASS: TestSbctl_TildeExpansion_DefaultKey/tilde_username_treated_as_literal (0.00s)
```

---

### AC-009 — os.Exit only in main() (go.md rule)

**Criterion:** `connectAndRun` returns error, never calls os.Exit. Only main() maps errors to exit codes.

**Proving test:** `TestSbctl_ConnectAndRun_ReturnsError` (main_test.go)

**Transcript:** [AC-005-006-009-010-011-remaining-acs.txt](AC-005-006-009-010-011-remaining-acs.txt)

```
--- PASS: TestSbctl_ConnectAndRun_ReturnsError (0.00s)
```

---

### AC-010 — RPC wire type "request"/"response" (BC-2.07.002 PC-3, Rulings M/U/X)

**Criterion:** `dispatch()` sets `Type: "request"` (not "rpc_request"); validates `resp.Type == "response"`; validates `resp.ID == req.ID`; non-constant per-call IDs.

**Proving tests:** `TestDispatch_EmitsCorrectWireType`, `TestDispatch_AcceptsResponseType`, `TestDispatch_RejectsWrongResponseType`, `TestDispatch_IDEchoEnforced` (client_test.go)

**Transcript:** [AC-005-006-009-010-011-remaining-acs.txt](AC-005-006-009-010-011-remaining-acs.txt)

```
--- PASS: TestDispatch_EmitsCorrectWireType (0.00s)
--- PASS: TestDispatch_AcceptsResponseType (0.00s)
--- PASS: TestDispatch_RejectsWrongResponseType (0.00s)
    --- PASS: TestDispatch_RejectsWrongResponseType/rpc_response_type_with_ok_true_is_rejected (0.00s)
    --- PASS: TestDispatch_RejectsWrongResponseType/ok_false_with_correct_type_is_erpc001_not_type_mismatch (0.00s)
--- PASS: TestDispatch_IDEchoEnforced (0.00s)
    --- PASS: TestDispatch_IDEchoEnforced/id_mismatch_returns_error (0.00s)
    --- PASS: TestDispatch_IDEchoEnforced/id_is_non_constant_across_calls (0.00s)
```

---

### AC-011 — dispatch() ctx-first + read deadline (BC-2.07.003 Inv-2, Ruling V)

**Criterion:** `dispatch(ctx, ...)` ctx-first; sets `conn.SetReadDeadline` from `ctx.Deadline()` or 30s fallback; deadline cleared on return.

**Proving test:** `TestDispatch_RespReadDeadlineEnforced` (client_test.go) — 3 sub-cases

**Transcript:** [AC-005-006-009-010-011-remaining-acs.txt](AC-005-006-009-010-011-remaining-acs.txt)

```
--- PASS: TestDispatch_RespReadDeadlineEnforced (0.11s)
    --- PASS: TestDispatch_RespReadDeadlineEnforced/ctx_with_deadline_fires_before_silent_server (0.20s)
    --- PASS: TestDispatch_RespReadDeadlineEnforced/no_deadline_ctx_fallback_arms_deadline (0.10s)
    --- PASS: TestDispatch_RespReadDeadlineEnforced/deadline_cleared_after_dispatch_returns (0.00s)
```

---

### AC-012 — No-subcommand and --help/-h → stdout, exit 0 (BC-2.07.002 EC-003)

**Criterion:** No args → usage to stdout, exit 0. `--help`/`-h` → flag usage to stdout (via `flag.CommandLine.SetOutput(os.Stdout)`), exit 0. stderr empty.

**Proving tests:** `TestSbctl_NoSubcommand_ExitsZero`, `TestSbctl_HelpFlag_ExitsZeroStdout` (main_test.go)

**Transcript:** [AC-012-no-subcommand-help-exit-zero.txt](AC-012-no-subcommand-help-exit-zero.txt)

**Binary demos:**
```
$ /tmp/sbctl-demo
usage: sbctl [--target=<addr>] [--key=<path>] [--json] [--timeout=<dur>] <subcommand> [args...]
Exit code: 0   (stdout non-empty, stderr empty)

$ /tmp/sbctl-demo --help
Usage of /tmp/sbctl-demo:
  -json ...  -key ...  -target ...  -timeout ...
Exit code: 0   (stdout non-empty, stderr empty)
```

```
--- PASS: TestSbctl_NoSubcommand_ExitsZero (1.02s)
--- PASS: TestSbctl_HelpFlag_ExitsZeroStdout (0.00s)
    --- PASS: TestSbctl_HelpFlag_ExitsZeroStdout/short-h (1.02s)
    --- PASS: TestSbctl_HelpFlag_ExitsZeroStdout/double-dash-help (1.02s)
```

---

## Deferred ACs

| AC | Happy-path scenario | Deferral reason |
|----|---------------------|-----------------|
| AC-003 (auth_ok e2e) | Live daemon sends AUTH_OK for correct key | Requires running daemon (`internal/mgmt` — S-W5.01) and full E2E harness — deferred to **S-W5.02** per story scope note. |
| AC-004 (E-RPC-001 e2e) | AUTH_OK + RPC dispatch failure from live daemon | Same — requires running `internal/mgmt`. Exercised in-process via mock in `TestSbctl_RPCDispatchFailure_ExitsOneWithERPC001`. |

All 12 ACs are covered by passing tests. The two deferred items are live-daemon happy-path verifications that require S-W5.01 daemon implementation — mock-based tests confirm the client-side logic is correct per the story boundary.
