## Summary

<!-- What does this PR do and why? -->

## Blast Radius

<!--
DECLARATION — this is a required section. Automated check fails PRs that
leave it empty or delete the header. See CONTRIBUTING.md §Blast radius.

Answer THREE questions in a couple of sentences each. Copy from a prior
PR of similar shape if you like; do not delete the labels.

1. Operator-visible surfaces touched:
   (e.g. sbctl subcommand output, switchboard --help/--version, config
    schema, error taxonomy, log format, wire protocol frame layout,
    admission rules, path metric emission, docs/getting-started.md.
    Answer "none" only if this is a truly internal refactor with no
    reachable behaviour change.)

2. Silent-failure risk:
   (Could this ship a defect that the current test suite does NOT
    catch? Cite the classes of regression that would slip through —
    e.g. "banner reads 'dev' in packaged binary because ldflags not
    wired", "help prints to stderr with exit 1", "sbctl <sub> --help
    opens a socket before parsing". Answer "none" only if every
    reachable defect class is unit-covered.)

3. Smoke gate touched:
   (Does this PR add, change, or need a NEW sentinel in
    test/smoke/invariants.sh? If yes, cite the INV-* id and confirm
    the paired docs/architecture.md §Smoke invariants row is included
    in this diff. If no, say "no.")

Purpose (see CONTRIBUTING.md §Blast radius): mechanical merges of
"one-line" changes have shipped operator-boundary regressions three
sessions running (S1/S3/O1/O3, 2026-07-04). The sentinels catch what
they know about; this block catches what they don't.
-->

**1. Operator-visible surfaces touched:**

<!-- your answer -->

**2. Silent-failure risk:**

<!-- your answer -->

**3. Smoke gate touched:**

<!-- your answer -->

## Changes

<!-- Bullet list of what changed. -->

-

## Checklist

- [ ] Tests added/updated
- [ ] `just fmt` -- code is formatted
- [ ] `just lint` -- zero warnings
- [ ] `just test` -- all tests pass
- [ ] `just smoke-quick` -- sentinel invariants pass locally
- [ ] Commit messages follow conventional commits format
- [ ] Blast radius block above answers all three questions (not "TBD")

## Testing

<!-- How was this tested? What test cases were added? -->

## Notes

<!-- Anything out of scope but worth noting for future work. -->
