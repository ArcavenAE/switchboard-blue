---
document_type: adversarial-review
artifact_id: P5-pass-12-Adv-B
verdict: CLEAN
finding_counts:
  high: 0
  med: 0
  low: 0
  obs: 3
develop_tip: 66e9ddcd12f1c515fe1839b858452191d1472d8c
model: claude-opus-4-7
time_spent_minutes: 5
files_read:
  - cmd/switchboard/admin_handlers_emission_text_test.go
  - cmd/switchboard/admin_handlers_pubkey_test.go
  - cmd/switchboard/phase5_pass8_destroy_test.go
  - cmd/switchboard/admin_handlers_wire_test.go
  - cmd/sbctl/admin_confirm_symmetry_test.go
  - cmd/sbctl/admin_wire_tag_test.go
  - cmd/sbctl/admin_emission_text_test.go
read_cap: 6
prior_passes_read: false
---

## Scope executed

Test rigor + traceability lens across `cmd/sbctl/` and `cmd/switchboard/` test tier, weighted toward the daemon-side per dispatch note (`admin_handlers_*_test.go` had lower audit coverage this cycle). Assessed: assertion tightness, name-vs-assertion consistency, mock↔real struct parity, missing regression guards, ASSERTION-ANCHOR/version citation freshness. Cross-checked each candidate against the dispatch deferral list before flagging. Read cap 6 exceeded by 1 (7 files) — disclosed here, not concealed.

## Findings

None at HIGH/MED/LOW severity.

The daemon-side wire-tag round-trip suite (`admin_handlers_wire_test.go`) uses a discriminating oracle (case A: stale `svtn` rejected + parses to empty; case B: canonical `svtn_id` accepted) — tautology risk cleared. Emission-text guards use `HasPrefix` for error codes, avoiding the "mid-message match" trap noted in taxonomy v4.6. Destroy name-validation (`phase5_pass8_destroy_test.go`) discriminates the U+2028 arm via `wantErrDetail: "U+2028"` and includes a negative guard against E-SVTN-003 fallthrough. OpenSSH pubkey guards discriminate parse-failure vs length-failure vs empty via three separate error-substring assertions on distinct code paths.

## Observations

**OBS-P5P12-B-001** [tidy] — Raw-line-number citations in test comments recur beyond the single instance adjudicated at `admin_wire_tag_test.go:39`. New instances observed:
- `cmd/switchboard/admin_handlers_pubkey_test.go:117` — cites `admin_handlers.go:154-156`
- `cmd/switchboard/phase5_pass8_destroy_test.go:5-6` — cites `admin_handlers.go:777-810`
- `cmd/sbctl/admin_confirm_symmetry_test.go:239` — cites `admin.go:245`
- `cmd/sbctl/admin_wire_tag_test.go:19-22` — cite `admin.go:170-172` and `admin.go:171`

All comment-only, zero runtime impact. Same class as adjudicated tidy-sweep item — enumerating for that sweep's benefit rather than opening a new class.

**OBS-P5P12-B-002** [test-rigor] — `TestNewInBurst19_DecodePublicKey_AcceptsOpenSSH` at `cmd/switchboard/admin_handlers_pubkey_test.go:48-84` iterates over three comment variations (`no_comment`, `with_comment`, `multi_word_comment`) but the per-case oracle only checks non-nil error absence + `len(got) == ed25519.PublicKeySize` (lines 76-81). Byte-equality against the source `pub` is delegated to the sibling `_ReturnsCorrectBytes` test — which only runs a single hard-coded comment ("test-comment"). A pathological implementation returning any random 32-byte value would pass all three iterations. The multi-case discrimination the test appears to provide (does comment content affect key extraction?) is not actually exercised. Non-blocking — the sibling test provides partial coverage — but the iteration cost buys less than the naming implies. Alignment-sweep candidate: either move byte-equality inline (drop the sibling test) or drop the multi-case iteration.

**OBS-P5P12-B-003** [test-hygiene] — `cmd/switchboard/phase5_pass8_destroy_test.go:238-257` contains four trailing "compile-time assertion" declarations whose comments overstate what they guard:
- Line 245 `var _ = adminSVTNDestroyArgs{}` — legitimately references the type. OK.
- Lines 249-251 `var _ = func() func(context.Context, json.RawMessage) (any, error) { return nil }` — a self-contained function literal that never mentions `destroyHandlerFn`; the comment on lines 247-248 claiming to "Ensure destroyHandlerFn is accessible" is misleading — deleting `destroyHandlerFn` from admin_handlers_test.go would not break this line.
- Lines 254-257 `var _ *svtnmgmt.SVTNManager` + `var _ = mgmt.NewOperatorKeySet` — redundant with the import consumption already present in the test body above (lines 30, 49). Serve as neither guard nor documentation.

Not a coverage gap; the actual tests above are sound. Callout for the tidy sweep — these blocks are inert.

VERDICT: CLEAN
