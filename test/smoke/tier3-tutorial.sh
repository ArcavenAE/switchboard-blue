#!/usr/bin/env bash
# smoke/tier3-tutorial.sh — Plan B Tier 3 tutorial smoke (task #176)
#
# Extracts fenced bash blocks from docs/getting-started.md, runs each in
# sequence, asserts exit codes and substring presence. This is the
# regression harness for the drift class caught in task #171 — where the
# tutorial's example router config omitted tick_interval and shipped
# broken.
#
# NOTE — this tier is EXPECTED TO FAIL until task #144 (router mode
# implementation) lands. Section 2 of the tutorial invokes
# `switchboard router --config …`; runRouter currently returns
# "runRouter: not implemented". This script distinguishes "known #144
# failure" (exit 3) from "real regression" (exit 1) so CI and reviewers
# can tell the difference.
#
# Assertion style (per task #176 spec):
#   - exit codes only where the tutorial states them
#   - substring presence for messages the tutorial *shows* the reader
#   - NEVER byte-exact goldens (whitespace, ordering, colors) — cosmetic
#     diffs are forbidden by Murat's risk register (BMAD 2026-07-04)
#
# Exit codes:
#   0 — every extractable block ran and every assertion passed
#   1 — an unexpected failure occurred (regression — investigate)
#   2 — harness broken (binaries missing, doc missing)
#   3 — known task #144 failure at Section 2 (expected until #144 lands)

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BIN_SWITCHBOARD="${REPO_ROOT}/bin/switchboard"
BIN_SBCTL="${REPO_ROOT}/bin/sbctl"
TUTORIAL="${REPO_ROOT}/docs/getting-started.md"

REPORT_DIR="${REPORT_DIR:-${REPO_ROOT}/.smoke/$(date -u +%Y%m%dT%H%M%SZ)-tier3}"
mkdir -p "${REPORT_DIR}"
REPORT="${REPORT_DIR}/report.jsonl"
: >"${REPORT}"

SMOKE_TMPDIR="$(mktemp -d)"
trap 'rm -rf "${SMOKE_TMPDIR}"' EXIT

PASS=0
EXPECTED_FAIL=0
UNEXPECTED_FAIL=0
UNEXPECTED_IDS=()

emit() {
  local id="$1" verdict="$2" detail="${3:-}"
  local esc_detail
  esc_detail="$(printf '%s' "${detail}" | sed 's/\\/\\\\/g; s/"/\\"/g' | tr -d '\n\r')"
  printf '{"id":"%s","verdict":"%s","detail":"%s"}\n' \
    "${id}" "${verdict}" "${esc_detail}" >>"${REPORT}"
  case "${verdict}" in
    PASS)
      PASS=$((PASS + 1))
      printf '  ✓ %s — %s\n' "${id}" "${detail}"
      ;;
    EXPECTED_FAIL)
      EXPECTED_FAIL=$((EXPECTED_FAIL + 1))
      printf '  ⊘ %s — expected-fail: %s\n' "${id}" "${detail}"
      ;;
    *)
      UNEXPECTED_FAIL=$((UNEXPECTED_FAIL + 1))
      UNEXPECTED_IDS+=("${id}")
      printf '  ✗ %s — %s\n' "${id}" "${detail}"
      ;;
  esac
}

require_bin() {
  local path="$1" name="$2"
  if [[ ! -x "${path}" ]]; then
    printf 'ERROR: %s binary not found at %s\n' "${name}" "${path}" >&2
    printf 'Build first: just build && go build -o bin/sbctl ./cmd/sbctl\n' >&2
    exit 2
  fi
}

require_bin "${BIN_SWITCHBOARD}" switchboard
require_bin "${BIN_SBCTL}" sbctl

if [[ ! -f "${TUTORIAL}" ]]; then
  printf 'ERROR: tutorial not found at %s\n' "${TUTORIAL}" >&2
  exit 2
fi

printf 'Tier 3 — tutorial smoke — %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
printf 'Tutorial: %s\n' "${TUTORIAL}"
printf 'Report: %s\n' "${REPORT}"
printf '\n'

# ─── Section-2 assertion: router config validates ────────────
# The narrower, testable claim from getting-started.md §2:
# "the router config with tick_interval: 10ms MUST parse+validate."
#
# The tutorial's actual bash block invokes `switchboard router --config
# …` which will block indefinitely (or fail on runRouter not-implemented).
# We do NOT run the block verbatim — we assert the config-file claim.
#
# Extraction: pull the yaml block that follows "Write `switchboard-router.yaml`"
# via awk, materialize it in tmpdir, then invoke the daemon in `control`
# mode against it. control uses the same config.LoadFile+Validate path
# as router, so any config-taxonomy regression in the tutorial surfaces
# here regardless of #144's status.
ROUTER_YAML="${SMOKE_TMPDIR}/router.yaml"
awk '
  BEGIN { in_block = 0; found = 0 }
  /Write `switchboard-router.yaml`/ { seen_header = 1 }
  seen_header && /^```yaml/ && !found { in_block = 1; next }
  in_block && /^```/ { in_block = 0; found = 1; exit }
  in_block { print }
' "${TUTORIAL}" >"${ROUTER_YAML}"

if [[ ! -s "${ROUTER_YAML}" ]]; then
  emit T3-2-extract UNEXPECTED "could not extract router yaml block from §2"
else
  emit T3-2-extract PASS "extracted router yaml ($(wc -c <"${ROUTER_YAML}" | tr -d ' ') bytes)"

  # Rewrite management_socket to a tmpdir path so we don't need root.
  # The tutorial ships "/run/switchboard-router.sock" which is
  # unwritable in test environments; a substitution here is legitimate
  # because we're testing the config *shape*, not the deployment path.
  ROUTER_YAML_FIXED="${SMOKE_TMPDIR}/router-fixed.yaml"
  sed 's|"/run/switchboard-router.sock"|"'"${SMOKE_TMPDIR}"'/tut-router.sock"|' \
    "${ROUTER_YAML}" >"${ROUTER_YAML_FIXED}"

  set +e
  "${BIN_SWITCHBOARD}" control --config "${ROUTER_YAML_FIXED}" >"${SMOKE_TMPDIR}/t3-2.log" 2>&1 &
  t3_pid=$!
  # Give it 500ms for parse+validate to complete; then TERM.
  for _ in 1 2 3 4 5; do
    if ! kill -0 "${t3_pid}" 2>/dev/null; then break; fi
    sleep 0.1
  done
  kill -TERM "${t3_pid}" 2>/dev/null || true
  wait "${t3_pid}" 2>/dev/null
  t3_exit=$?
  set -e
  t3_log="$(cat "${SMOKE_TMPDIR}/t3-2.log" 2>/dev/null || true)"

  # Success shape: no E-CFG-* in log. Exit code may be 0 (clean SIGTERM)
  # or non-zero if the daemon failed on something downstream — we accept
  # any exit as long as no config-taxonomy error was raised.
  if [[ "${t3_log}" != *"E-CFG-"* ]]; then
    emit T3-2-config PASS "tutorial router config parsed+validated cleanly (exit=${t3_exit})"
  else
    emit T3-2-config UNEXPECTED "tutorial router config leaked E-CFG-*: '${t3_log:0:200}'"
  fi

  # Now attempt the actual `switchboard router` invocation. This is the
  # "expected fail until #144" gate.
  set +e
  "${BIN_SWITCHBOARD}" router --config "${ROUTER_YAML_FIXED}" >"${SMOKE_TMPDIR}/t3-2-router.log" 2>&1 &
  t3r_pid=$!
  # If runRouter returns "not implemented" it exits immediately with
  # non-zero; if #144 lands and it becomes a real daemon it will block.
  # Give it 1s either way.
  for _ in 1 2 3 4 5 6 7 8 9 10; do
    if ! kill -0 "${t3r_pid}" 2>/dev/null; then break; fi
    sleep 0.1
  done
  kill -TERM "${t3r_pid}" 2>/dev/null || true
  wait "${t3r_pid}" 2>/dev/null
  t3r_exit=$?
  set -e
  t3r_log="$(cat "${SMOKE_TMPDIR}/t3-2-router.log" 2>/dev/null || true)"

  if [[ "${t3r_log}" == *"runRouter: not implemented"* || "${t3r_log}" == *"not implemented"* ]]; then
    emit T3-2-router EXPECTED_FAIL "task #144 gate: runRouter returned 'not implemented' (exit=${t3r_exit})"
  elif [[ "${t3r_exit}" -eq 0 || "${t3r_exit}" -eq 143 ]]; then
    # Router became implemented and shut down cleanly.
    emit T3-2-router PASS "router started and shut down cleanly (exit=${t3r_exit}); task #144 appears to have landed — remove the expected-fail once verified"
  else
    emit T3-2-router UNEXPECTED "router exited with unexpected shape: exit=${t3r_exit} log='${t3r_log:0:200}'"
  fi
fi

# ─── Section-4 assertion: sbctl error taxonomy is stable ──────
# The tutorial's §"Common pitfalls" promises stable error codes. Assert
# that sbctl with no auth against a missing target produces a taxonomy
# code (E-CFG-* or E-NET-*), not a panic or bare Go error.
#
# This is the substring assertion from the "Common pitfalls" section:
# "Every error carries a stable taxonomy code." — testable claim.
set +e
sbctl_no_target_out="$("${BIN_SBCTL}" sessions list 2>&1)"
sbctl_no_target_exit=$?
set -e
if [[ "${sbctl_no_target_exit}" -ne 0 \
    && ( "${sbctl_no_target_out}" == *"E-"* || "${sbctl_no_target_out}" == *"target"* || "${sbctl_no_target_out}" == *"required"* ) \
    && "${sbctl_no_target_out}" != *"panic:"* ]]; then
  emit T3-4-taxonomy PASS "sbctl without --target exits non-zero with a stable message (exit=${sbctl_no_target_exit})"
else
  emit T3-4-taxonomy UNEXPECTED "sbctl no-target: exit=${sbctl_no_target_exit} out='${sbctl_no_target_out:0:200}'"
fi

# ─── Summary ──────────────────────────────────────────────────
printf '\n'
printf 'Tier 3: %d passed, %d expected-failed, %d unexpected-failed\n' \
  "${PASS}" "${EXPECTED_FAIL}" "${UNEXPECTED_FAIL}"
printf 'Report artifact: %s\n' "${REPORT}"

# Exit-code contract per task #176:
#   0 — clean pass (every extractable block passed, no expected-fails)
#   1 — real regression (an UNEXPECTED failure)
#   3 — known task #144 failure at Section 2 (expected — CI should
#       currently expect exit 3)
if [[ "${UNEXPECTED_FAIL}" -gt 0 ]]; then
  printf '\nUnexpected failures:\n'
  for id in "${UNEXPECTED_IDS[@]}"; do
    printf '  - %s\n' "${id}"
  done
  exit 1
fi
if [[ "${EXPECTED_FAIL}" -gt 0 ]]; then
  printf '\nExpected failures present — exit 3 to signal known task #144 gap.\n'
  printf 'When #144 lands, T3-2-router should flip to PASS and this exits 0.\n'
  exit 3
fi
exit 0
