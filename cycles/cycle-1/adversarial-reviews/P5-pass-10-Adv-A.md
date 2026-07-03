```yaml
document_type: adversarial-review
artifact_id: P5-pass-10-Adv-A
verdict: HAS_FINDINGS
finding_counts:
  high: 1
  med: 1
  low: 0
  obs: 0
develop_tip: 32ea461cd1c50a32e17e42a7f678f701b4dfa04b
model: us.anthropic.claude-opus-4-7
time_spent_minutes: 5
files_read:
  - .factory/specs/prd-supplements/interface-definitions.md
  - cmd/sbctl/main.go
  - cmd/sbctl/admin.go
read_cap: 6
prior_passes_read: false
```

## Findings

### F-P5P10-A-001 [HIGH] — `sbctl admin key expire`: spec-documented `--at <RFC3339-timestamp>` flag does not exist in the CLI; impl accepts `--after <duration>`

**Spec cite:** interface-definitions.md §110 (row 3 of Key management table), v1.6 changelog note (§168–169):
> `sbctl admin key expire --svtn <id> --key <openssh-pubkey> --at <RFC3339-timestamp>` | Set automatic expiry on an admission key. CLI translates `--at <RFC3339-timestamp>` to a Go duration string (`after` wire field) before sending: `after = timestamp - time.Now()`.

The spec declares the operator-facing flag is `--at`, and the CLI is responsible for the RFC3339→duration translation before the wire hop.

**Impl cite:** `cmd/sbctl/admin.go:527-563` (`runAdminKeyExpire`)
- Line 531: `afterFlag := fs.String("after", "", "TTL duration (required; e.g. \"24h\")")`
- Line 543-545: requires `--after`; no `--at` flag registered.
- Line 550: `d, err := time.ParseDuration(*afterFlag)` — parses Go duration syntax, not RFC3339.

There is no `--at` flag registered, and no RFC3339 parsing anywhere in `runAdminKeyExpire`. An operator who follows the spec verbatim and types `sbctl admin key expire --svtn <id> --key <pk> --at 2026-08-01T12:00:00Z` will hit Go's `flag` package with `flag provided but not defined: -at`, producing a parse error routed through `usageErrf` and `os.Exit(2)`. The advertised operator-facing CLI surface is unreachable.

**Failure scenario:** Operator reads §110 of the shipped interface spec, types the documented invocation, gets exit 2 with "flag provided but not defined: -at" and no hint that they should be using `--after <duration>` instead. Every operator following the spec fails their first key-expire attempt; scripts written against the spec break.

### F-P5P10-A-002 [MED] — `sbctl admin key expire`: spec-promised `E-CFG-001` for zero/negative duration is neither emitted as a code nor mapped to the exit class it implies

**Spec cite:** interface-definitions.md §110:
> 0=ok, E-ADM-013 (key not found), **E-CFG-001** (invalid `after` duration: zero, negative, or >100 years), E-ADM-021 …

Spec §186 (exit code table): row-2 (exit 2, usage error) explicitly enumerates the E-CFG-* codes routed there as "E-CFG-012; E-CFG-013". E-CFG-001 is not listed under exit 2. Every other `E-CFG-` and `E-ADM-` code shipped by the impl appears in stderr as a prefixed token (`E-CFG-012: …`, `E-CFG-013: …`, `E-ADM-019: …`), which is how scripts pattern-match error taxonomy.

**Impl cite:** `cmd/sbctl/admin.go:554-556`:
```go
if d <= 0 {
    return usageErrf("admin key expire: --after duration must be positive, got %q", *afterFlag)
}
```

The zero/negative branch:
1. Emits no `E-CFG-001` token in the error message — the string is plain prose.
2. Wraps the error in `usageError`, so `main.go:106-109` maps it to `os.Exit(2)`, not the operational-error class where `E-CFG-*` codes conventionally live.

The `>100 years` arm has no client-side check at admin.go:554-556 and reaches the server, so *that* subcase produces `E-CFG-001` via the daemon's `mapAdminError`. The zero/negative subcase, which is the one an operator most naturally hits, silently drops the taxonomy code.

**Failure scenario:** An operator script wraps `sbctl admin key expire` with `if [[ $? -eq 1 ]] && grep -q E-CFG-001 stderr`; passing `--after 0s` or `--after -1h` hits exit 2 with no matching stderr token, so the script's config-error handler never fires. Meanwhile passing `--after 200000h` (>100 years) hits E-CFG-001 with exit 1. The same "invalid duration" family documented under one code (E-CFG-001) fragments in the wire into two exit codes and two different stderr shapes depending on which side of zero the operator's typo lands on.

VERDICT: HAS_FINDINGS
