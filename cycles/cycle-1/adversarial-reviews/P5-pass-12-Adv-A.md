---
document_type: adversarial-review
artifact_id: P5-pass-12-Adv-A
verdict: HAS_FINDINGS
finding_counts:
  high: 0
  med: 2
  low: 0
  obs: 2
develop_tip: 66e9ddcd12f1c515fe1839b858452191d1472d8c
model: opus-4.7
time_spent_minutes: 5
files_read:
  - .factory/specs/prd-supplements/interface-definitions.md
  - cmd/sbctl/main.go
  - cmd/sbctl/admin.go
  - cmd/sbctl/console.go
  - cmd/switchboard/admin_handlers.go
read_cap: 6
prior_passes_read: false
---

## Findings

### F-P5P12-A-001 [MED] — `admin list-keys` exit-code column omits reachable E-SVTN-003

**Where:** interface-definitions.md §111 (row for `sbctl admin list-keys --svtn <id>`) vs `cmd/switchboard/admin_handlers.go:361` and `admin_handlers.go:413-414`.

**Claim:** Spec §111 exit-code column reads "0=ok" only.

**Impl reality:** `makeListKeysHandler` at `admin_handlers.go:361` invokes `m.ListKeys(a.SVTNName)`. When the requested SVTN does not exist, this returns `svtnmgmt.ErrSVTNNotFound`, which `mapAdminError:413-414` wraps as `&svtnNotFoundErr{name: svtnName, cause: err}` — rendered on the wire as `"E-SVTN-003: SVTN not found: <name>"`. That is a mapped, tokened error surfaced to the operator on exit 1.

**Scope note:** The v1.20–v1.22 adjudicated "register/revoke/expire error surfaces reachability-audited" umbrella (deferral list) enumerates only three verbs — `admin.key.register`, `admin.key.revoke`, `admin.key.expire`. `admin.key.list-keys` is not covered by that adjudication.

**Failure scenario:** Operator reads §111, sees "0=ok", assumes the read-only verb has no operational failure path. Runs `sbctl admin list-keys --svtn typo-in-svtn-name`, receives `E-SVTN-003: SVTN not found: typo-in-svtn-name` on exit 1 — a tokened error the row promised did not exist.

**Suggested spec fix:** Extend §111 exit-code column to `0=ok, E-SVTN-003 (SVTN not found — reachable via admin_handlers.go:361 → mapAdminError:413-414) and E-CFG-001 (missing --svtn, client-side, exit 2 — cmd/sbctl/admin.go:167-169)`. Consider a parallel note that E-CFG-001 is reachable from `cmd/sbctl/admin.go:167-169` when `--svtn` is empty (exit 2 on client-side usage), for symmetry with §108/§109/§110.

---

### F-P5P12-A-002 [MED] — CLI syntax cells §108/§109/§110 use `--svtn <id>` while wire schema §396-398 documents `<svtn-name>`

**Where:** interface-definitions.md §108, §109, §110 (CLI syntax cells for `admin key register`, `admin key revoke`, `admin key expire`) vs §396-398 (Registered Verbs rows for the corresponding admin.key.* wire verbs, corrected in v1.14).

**Claim:** §108 — `sbctl admin key register --svtn <id> --key <openssh-pubkey> [--role ...]`. §109 — `sbctl admin key revoke --svtn <id> --key <openssh-pubkey> --role ...`. §110 — `sbctl admin key expire --svtn <id> --key <openssh-pubkey> --after <duration>`. The placeholder `<id>` implies a machine identifier.

**Impl reality:** The wire args struct at `cmd/switchboard/admin_handlers.go:49-54` uses Go field name `SVTNName` (JSON tag `svtn_id`). §396-398 (updated in the v1.14 changelog note) document the field carrying `<svtn-name>` — the human-readable label passed to `admin svtn create --name=<svtn-name>`. The daemon's lookup is name-keyed. Meanwhile §400 (admin.svtn.create response) returns `{"svtn_id": "<hex>", "bootstrap_fingerprint": "SHA256:<base64>"}` — the value at that key is the 16-byte hex identifier. So `svtn_id` in a create response is hex, but `svtn_id` in a key-lifecycle request is a name.

**Failure scenario:** Operator creates an SVTN, receives `{svtn_id: "a1b2c3d4e5f60708..."}` in the create response, and pastes that hex string into `sbctl admin key register --svtn a1b2c3d4... --key ...`. The daemon interprets the value as an SVTN name, the lookup fails, the operator sees `E-SVTN-003: SVTN not found: a1b2c3d4...` — reprinting the "correct" identifier they just received from the daemon. Confusing at best; support-ticket-generating at worst.

**Suggested spec fix:** In §108, §109, §110 syntax cells, change `--svtn <id>` → `--svtn <svtn-name>` (or `--svtn <name>`), matching the v1.14-corrected Registered Verbs placeholders. CLI flag help at `admin.go:438,486,530` ("SVTN identifier (required)") is likewise ambiguous but subordinate.

**Note:** I did not verify daemon SVTN-lookup semantics directly (would exceed read budget); the name-keying inference is from the `SVTNName` Go field, the v1.14 changelog wording, and the value in §396-398. If the daemon actually accepts both name and hex (dual-key lookup), this finding downgrades to LOW / OBS.

---

## Observations

### OBS-P5P12-A-001 — admin.go:5-9 doc header shows `admin list-keys [--svtn <id>]` optional, but code requires it

The internal doc comment at `cmd/sbctl/admin.go:5-9` shows `sbctl admin list-keys [--svtn <id>]`, brackets indicating optionality. But `admin.go:167-169` explicitly rejects the empty case with `usageErrf("admin list-keys: --svtn <id> is required")`. Spec §111 is correct (no brackets → required); only the internal doc comment is drifted. Not user-facing help text. Distinct from the deferral-list adjudicated item ("admin.go:5-9 doc header omits svtn destroy") which is about a missing row; this is a bracketing discrepancy on an existing row.

### OBS-P5P12-A-002 — Confirm-family flags absent from §108/§120 syntax cells

§109 revoke row header shows `[--confirm]` in the syntax cell. §108 register and §120 destroy syntax cells omit `[--confirm]` and `[--yes]` even though register and destroy ARE part of the `runDestroyConfirmGate` family per §131/§135/§137 (verified at `cmd/sbctl/admin.go:306` and `cmd/sbctl/admin.go:463`), with corresponding E-CFG-012 / E-CFG-013 exit codes listed in their exit-codes columns. Operators cross-referencing the row header for quick-syntax will miss `--yes`/`--confirm` on register/destroy; §131-§137 prose is the only place they surface. Consistency observation, not a hard defect (§131-§137 prose is authoritative).

---

VERDICT: HAS_FINDINGS
