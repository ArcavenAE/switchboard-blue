#!/usr/bin/env bash
# smoke/invariants.sh — Plan A sentinel invariants (task #175 / BMAD party 2026-07-04)
#
# Behavioral assertions that MUST hold on every merge. Runs in <5 seconds.
# Guards the class of operator-boundary regressions caught by the tutorial
# smoke of 2026-07-04 (S1/S3/O1/O3 in .factory/STATE.md drift register) and
# would have blocked the shipping of the three fixes that followed.
#
# Rules (per Murat's risk register, BMAD 2026-07-04):
#   - Assertions MUST be behavioral: exit code, stream direction, substring
#     presence. Cosmetic diffs (exact whitespace, exact ordering, color
#     codes) are FORBIDDEN. Reviewers reject cosmetic sentinels in PR review.
#   - New invariants require a paired docs/architecture.md §Smoke Invariants
#     note.
#
# Usage:
#   just smoke-quick                # from repo root, builds and runs
#   test/smoke/invariants.sh        # requires bin/switchboard, bin/sbctl
#
# Exit codes:
#   0 — all invariants passed
#   1 — one or more invariants failed
#   2 — harness itself is broken (binary missing, tmpdir unwritable, etc.)

set -euo pipefail

# ─── Setup ────────────────────────────────────────────────────

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BIN_SWITCHBOARD="${REPO_ROOT}/bin/switchboard"
BIN_SBCTL="${REPO_ROOT}/bin/sbctl"

# Timestamped report directory — Priya's audit trail requirement.
# Reproducibility: no timestamps in the assertions themselves, only in the
# report path so multiple runs coexist for post-mortem.
REPORT_DIR="${REPORT_DIR:-${REPO_ROOT}/.smoke/$(date -u +%Y%m%dT%H%M%SZ)}"
mkdir -p "${REPORT_DIR}"
REPORT="${REPORT_DIR}/report.jsonl"
: >"${REPORT}"

# Working tmpdir — isolated, no touching ~/.switchboard.
SMOKE_TMPDIR="$(mktemp -d)"
trap 'rm -rf "${SMOKE_TMPDIR}"' EXIT

# ─── Helpers ──────────────────────────────────────────────────

PASS=0
FAIL=0
FAIL_IDS=()

emit() {
  # emit <id> <verdict> <detail>
  local id="$1" verdict="$2" detail="${3:-}"
  # Escape backslashes and quotes for JSON.
  local esc_detail
  esc_detail="$(printf '%s' "${detail}" | sed 's/\\/\\\\/g; s/"/\\"/g' | tr -d '\n\r')"
  printf '{"id":"%s","verdict":"%s","detail":"%s"}\n' \
    "${id}" "${verdict}" "${esc_detail}" >>"${REPORT}"
  if [[ "${verdict}" == "PASS" ]]; then
    PASS=$((PASS + 1))
    printf '  ✓ %s\n' "${id}"
  else
    FAIL=$((FAIL + 1))
    FAIL_IDS+=("${id}")
    printf '  ✗ %s — %s\n' "${id}" "${detail}"
  fi
}

require_bin() {
  local path="$1" name="$2"
  if [[ ! -x "${path}" ]]; then
    printf 'ERROR: %s binary not found at %s\n' "${name}" "${path}" >&2
    printf 'Build first: just build && go build -o bin/sbctl ./cmd/sbctl\n' >&2
    exit 2
  fi
}

# Run a command capturing stdout, stderr, and exit code separately.
# Sets: SMOKE_STDOUT, SMOKE_STDERR, SMOKE_EXIT
run_capture() {
  local out_file err_file
  out_file="$(mktemp -p "${SMOKE_TMPDIR}")"
  err_file="$(mktemp -p "${SMOKE_TMPDIR}")"
  set +e
  "$@" >"${out_file}" 2>"${err_file}"
  SMOKE_EXIT=$?
  set -e
  SMOKE_STDOUT="$(cat "${out_file}")"
  SMOKE_STDERR="$(cat "${err_file}")"
}

# ─── Preflight ────────────────────────────────────────────────

require_bin "${BIN_SWITCHBOARD}" switchboard
require_bin "${BIN_SBCTL}" sbctl

printf 'Sentinel invariants — %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
printf 'Report: %s\n' "${REPORT}"
printf '\n'

# ─── INV-1: switchboard --help exits 0 to stdout with no stderr ─
# BC-2.07.002 EC-003 Ruling A. Shipped in PR #77 (7e7af92).
run_capture "${BIN_SWITCHBOARD}" --help
if [[ "${SMOKE_EXIT}" -eq 0 && -n "${SMOKE_STDOUT}" && -z "${SMOKE_STDERR}" ]]; then
  emit INV-1 PASS "switchboard --help exit=0 stdout non-empty stderr empty"
else
  emit INV-1 FAIL "switchboard --help exit=${SMOKE_EXIT} stdout_bytes=${#SMOKE_STDOUT} stderr_bytes=${#SMOKE_STDERR}"
fi

# ─── INV-2: switchboard --version prints basename prefix and exits 0 ─
# BC-2.07.002 EC-003 Ruling A analog. Prevents S3-style regression where
# the version banner is a hardcoded literal instead of args[0]-derived.
run_capture "${BIN_SWITCHBOARD}" --version
basename_switchboard="$(basename "${BIN_SWITCHBOARD}")"
if [[ "${SMOKE_EXIT}" -eq 0 && "${SMOKE_STDOUT}" == "${basename_switchboard} "* ]]; then
  emit INV-2 PASS "switchboard --version starts with '${basename_switchboard} '"
else
  emit INV-2 FAIL "switchboard --version exit=${SMOKE_EXIT} stdout='${SMOKE_STDOUT}'"
fi

# ─── INV-3: sbctl --help exits 0 to stdout with no stderr ─
# BC-2.07.002 EC-003 Ruling A. Shipped in PR #77.
run_capture "${BIN_SBCTL}" --help
if [[ "${SMOKE_EXIT}" -eq 0 && -n "${SMOKE_STDOUT}" && -z "${SMOKE_STDERR}" ]]; then
  emit INV-3 PASS "sbctl --help exit=0 stdout non-empty stderr empty"
else
  emit INV-3 FAIL "sbctl --help exit=${SMOKE_EXIT} stdout_bytes=${#SMOKE_STDOUT} stderr_bytes=${#SMOKE_STDERR}"
fi

# ─── INV-4: sbctl --version prints basename prefix and exits 0 ─
# Guards O3 class regression (sbctl --version flag was missing pre-PR #77).
run_capture "${BIN_SBCTL}" --version
basename_sbctl="$(basename "${BIN_SBCTL}")"
if [[ "${SMOKE_EXIT}" -eq 0 && "${SMOKE_STDOUT}" == "${basename_sbctl} "* ]]; then
  emit INV-4 PASS "sbctl --version starts with '${basename_sbctl} '"
else
  emit INV-4 FAIL "sbctl --version exit=${SMOKE_EXIT} stdout='${SMOKE_STDOUT}'"
fi

# ─── INV-5: sbctl (no args) exits 2 with usage on stderr ─
# interface-definitions.md v1.18 §174 — usage error contract.
run_capture "${BIN_SBCTL}"
if [[ "${SMOKE_EXIT}" -eq 2 && "${SMOKE_STDERR}" == *"available subcommands:"* ]]; then
  emit INV-5 PASS "sbctl no-args exit=2 stderr contains 'available subcommands:'"
else
  emit INV-5 FAIL "sbctl no-args exit=${SMOKE_EXIT} stderr='${SMOKE_STDERR}'"
fi

# ─── INV-6: sbctl unknown-subcommand exits 2 with hint on stderr ─
# interface-definitions.md v1.18 §174 — unknown-subcommand contract.
run_capture "${BIN_SBCTL}" definitely-not-a-real-subcommand
if [[ "${SMOKE_EXIT}" -eq 2 && "${SMOKE_STDERR}" == *"unknown subcommand"* ]]; then
  emit INV-6 PASS "sbctl unknown-subcommand exit=2 stderr contains 'unknown subcommand'"
else
  emit INV-6 FAIL "sbctl unknown-subcommand exit=${SMOKE_EXIT} stderr='${SMOKE_STDERR}'"
fi

# ─── INV-7: every switchboard subcommand accepts --help and exits 0 ─
# Guards subcommand-scoped help regressions. Currently registered
# subcommands: access, router, console, control.
# access requires config; --help must short-circuit BEFORE any I/O.
for sub in access router console control; do
  run_capture "${BIN_SWITCHBOARD}" "${sub}" --help
  if [[ "${SMOKE_EXIT}" -eq 0 && -n "${SMOKE_STDOUT}" ]]; then
    emit "INV-7:${sub}" PASS "switchboard ${sub} --help exit=0"
  else
    emit "INV-7:${sub}" FAIL "switchboard ${sub} --help exit=${SMOKE_EXIT} stdout_bytes=${#SMOKE_STDOUT} stderr='${SMOKE_STDERR:0:120}'"
  fi
done

# ─── INV-8: version banner contains ldflags-injected version, NOT 'dev' ─
# Guards the S3-tail case: ldflags wiring missing → binary reports "dev"
# in production. This is the invariant that would have caught the sbctl-a
# packaging gap (task #163) at pre-merge time.
#
# Two-part assertion:
#   (a) VERSION env var was set by CI (fail-fast if unset — this is a CI
#       contract, not a local dev contract; local runs of `just smoke-quick`
#       set VERSION=smoke-sentinel automatically via the recipe).
#   (b) Both banners contain the exact VERSION token.
if [[ -z "${VERSION:-}" ]]; then
  emit INV-8 SKIP "VERSION env not set — INV-8 asserts ldflags wiring, must be run via 'just smoke-quick' or set VERSION explicitly"
else
  run_capture "${BIN_SWITCHBOARD}" --version
  if [[ "${SMOKE_STDOUT}" == *"${VERSION}"* ]]; then
    emit "INV-8:switchboard" PASS "switchboard --version contains VERSION=${VERSION}"
  else
    emit "INV-8:switchboard" FAIL "switchboard --version='${SMOKE_STDOUT}' does not contain VERSION=${VERSION} (ldflags wiring missing?)"
  fi
  run_capture "${BIN_SBCTL}" --version
  if [[ "${SMOKE_STDOUT}" == *"${VERSION}"* ]]; then
    emit "INV-8:sbctl" PASS "sbctl --version contains VERSION=${VERSION}"
  else
    emit "INV-8:sbctl" FAIL "sbctl --version='${SMOKE_STDOUT}' does not contain VERSION=${VERSION} (ldflags wiring missing — this is the task #163 defect at pre-merge time)"
  fi
fi

# ─── Summary ──────────────────────────────────────────────────

printf '\n'
printf 'Sentinels: %d passed, %d failed\n' "${PASS}" "${FAIL}"
printf 'Report artifact: %s\n' "${REPORT}"

if [[ "${FAIL}" -gt 0 ]]; then
  printf '\nFailed invariants:\n'
  for id in "${FAIL_IDS[@]}"; do
    printf '  - %s\n' "${id}"
  done
  printf '\nSee report for details. Fix or explain BEFORE merging.\n'
  exit 1
fi

exit 0
