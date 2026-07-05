#!/usr/bin/env bash
# smoke/spec-runner.sh — Plan D spec-as-test runner (task #178)
#
# Generic executor for test/smoke/spec-assertions.json: a machine-readable
# catalog of spec acceptance criteria projected into behavioral assertions.
# The inversion Plan D proposed (BMAD party 2026-07-04): instead of a human
# translating each AC into bespoke bash (Tier 1 sentinels), the AC's claim
# lives as data — anchor, prose claim, command, expected behavior — and this
# runner executes every entry identically. Adding spec coverage = adding a
# JSON object; no new shell.
#
# Assertion vocabulary (deliberately small, per the Tier 1 contract rules in
# docs/architecture.md §Smoke invariants — behavioral only, never cosmetic):
#   exit               required int — expected exit code
#   stdout_empty       bool — stdout must be empty
#   stderr_empty       bool — stderr must be empty
#   stdout_contains    substring that must appear on stdout
#   stderr_contains    substring that must appear on stderr
#   stderr_contains_2  second required stderr substring
#   stdout_jq          jq boolean expression evaluated against stdout as JSON
#   stderr_jq          jq boolean expression evaluated against stderr as JSON
#
# Expected-fail: an entry may carry `expected_fail: "<issue ref + reason>"`.
# A failing expected-fail entry reports XFAIL and does not fail the run; a
# PASSING expected-fail entry reports XPASS and FAILS the run — the defect
# was fixed, so the annotation must be removed in the fixing PR (same
# discipline as Tier 3's task-#144 gate).
#
# Command template variables: $BIN_SWITCHBOARD, $BIN_SBCTL, $SMOKE_KEY
# (ephemeral valid ed25519 key generated per run), $SMOKE_TMPDIR.
#
# Usage:
#   just smoke-spec               # from repo root, builds and runs
#   test/smoke/spec-runner.sh     # requires bin/switchboard, bin/sbctl
#
# Exit codes (same contract as invariants.sh):
#   0 — all spec assertions passed
#   1 — one or more assertions failed
#   2 — harness broken (jq missing, binaries missing, catalog unparseable)

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BIN_SWITCHBOARD="${REPO_ROOT}/bin/switchboard"
BIN_SBCTL="${REPO_ROOT}/bin/sbctl"
CATALOG="${REPO_ROOT}/test/smoke/spec-assertions.json"

REPORT_DIR="${REPORT_DIR:-${REPO_ROOT}/.smoke/$(date -u +%Y%m%dT%H%M%SZ)-spec}"
mkdir -p "${REPORT_DIR}"
REPORT="${REPORT_DIR}/report.jsonl"
: >"${REPORT}"

SMOKE_TMPDIR="$(mktemp -d)"
trap 'rm -rf "${SMOKE_TMPDIR}"' EXIT

# ─── Harness preconditions (exit 2 on breakage) ───────────────

for tool in jq ssh-keygen; do
  if ! command -v "${tool}" >/dev/null 2>&1; then
    printf 'spec-runner: harness broken: %s not found\n' "${tool}" >&2
    exit 2
  fi
done
for bin in "${BIN_SWITCHBOARD}" "${BIN_SBCTL}"; do
  if [[ ! -x "${bin}" ]]; then
    printf 'spec-runner: harness broken: %s missing (run just build first)\n' "${bin}" >&2
    exit 2
  fi
done
if ! jq -e '.assertions | length > 0' "${CATALOG}" >/dev/null 2>&1; then
  printf 'spec-runner: harness broken: %s missing or unparseable\n' "${CATALOG}" >&2
  exit 2
fi

# Ephemeral valid operator key — lets assertions reach past key-loading to
# the behavior actually under test (e.g. E-NET-001 daemon-unreachable).
SMOKE_KEY="${SMOKE_TMPDIR}/spec-key"
ssh-keygen -t ed25519 -f "${SMOKE_KEY}" -N '' -q

export BIN_SWITCHBOARD BIN_SBCTL SMOKE_KEY SMOKE_TMPDIR

PASS=0
FAIL=0
XFAIL=0
FAIL_IDS=()

emit() {
  local id="$1" verdict="$2" detail="${3:-}"
  local esc_detail
  esc_detail="$(printf '%s' "${detail}" | sed 's/\\/\\\\/g; s/"/\\"/g' | tr -d '\n\r')"
  printf '{"id":"%s","verdict":"%s","detail":"%s"}\n' \
    "${id}" "${verdict}" "${esc_detail}" >>"${REPORT}"
  case "${verdict}" in
  PASS)
    PASS=$((PASS + 1))
    printf '  ✓ %s\n' "${id}"
    ;;
  XFAIL)
    XFAIL=$((XFAIL + 1))
    printf '  ⊘ %s — expected-fail: %s\n' "${id}" "${detail}"
    ;;
  *)
    FAIL=$((FAIL + 1))
    FAIL_IDS+=("${id}")
    printf '  ✗ %s — %s\n' "${id}" "${detail}"
    ;;
  esac
}

printf 'Spec assertions (Plan D) — %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
printf 'Catalog: %s\n' "${CATALOG}"
printf 'Report:  %s\n\n' "${REPORT}"

count="$(jq -r '.assertions | length' "${CATALOG}")"

for i in $(seq 0 $((count - 1))); do
  entry="$(jq -c ".assertions[${i}]" "${CATALOG}")"
  id="$(jq -r '.id' <<<"${entry}")"
  anchor="$(jq -r '.anchor' <<<"${entry}")"
  cmd_template="$(jq -r '.cmd' <<<"${entry}")"

  # Expand $VAR references from the exported environment. eval is confined
  # to the checked-in catalog (reviewed like code); no external input.
  cmd="$(eval "printf '%s' \"${cmd_template}\"")"

  stdout_f="${SMOKE_TMPDIR}/${id}.out"
  stderr_f="${SMOKE_TMPDIR}/${id}.err"
  set +e
  # shellcheck disable=SC2086
  # (word-splitting the expanded command is intended)
  ${cmd} >"${stdout_f}" 2>"${stderr_f}"
  actual_exit=$?
  set -e

  failures=()

  want_exit="$(jq -r '.want.exit' <<<"${entry}")"
  if [[ "${actual_exit}" -ne "${want_exit}" ]]; then
    failures+=("exit=${actual_exit} want=${want_exit}")
  fi

  if [[ "$(jq -r '.want.stdout_empty // false' <<<"${entry}")" == "true" && -s "${stdout_f}" ]]; then
    failures+=("stdout not empty ($(wc -c <"${stdout_f}" | tr -d ' ') bytes)")
  fi
  if [[ "$(jq -r '.want.stderr_empty // false' <<<"${entry}")" == "true" && -s "${stderr_f}" ]]; then
    failures+=("stderr not empty ($(wc -c <"${stderr_f}" | tr -d ' ') bytes)")
  fi

  for key in stdout_contains stderr_contains stderr_contains_2; do
    needle="$(jq -r ".want.${key} // empty" <<<"${entry}")"
    [[ -z "${needle}" ]] && continue
    stream_f="${stdout_f}"
    [[ "${key}" == stderr_* ]] && stream_f="${stderr_f}"
    if ! grep -qF -- "${needle}" "${stream_f}"; then
      failures+=("${key} '${needle}' not found")
    fi
  done

  for key in stdout_jq stderr_jq; do
    expr="$(jq -r ".want.${key} // empty" <<<"${entry}")"
    [[ -z "${expr}" ]] && continue
    stream_f="${stdout_f}"
    [[ "${key}" == stderr_* ]] && stream_f="${stderr_f}"
    if ! jq -e "${expr}" "${stream_f}" >/dev/null 2>&1; then
      failures+=("${key} '${expr}' false or unparseable")
    fi
  done

  expected_fail="$(jq -r '.expected_fail // empty' <<<"${entry}")"
  if [[ ${#failures[@]} -eq 0 ]]; then
    if [[ -n "${expected_fail}" ]]; then
      emit "${id}" XPASS "expected-fail annotation is stale (${expected_fail}) — the defect appears fixed; remove the annotation in the fixing PR"
    else
      emit "${id}" PASS "${anchor}"
    fi
  else
    if [[ -n "${expected_fail}" ]]; then
      emit "${id}" XFAIL "${expected_fail}"
    else
      emit "${id}" FAIL "${anchor}: $(IFS='; '; echo "${failures[*]}")"
    fi
  fi
done

printf '\n%d passed, %d expected-fail, %d failed (of %d spec assertions)\n' "${PASS}" "${XFAIL}" "${FAIL}" "${count}"
if [[ "${FAIL}" -gt 0 ]]; then
  printf 'Failed: %s\n' "${FAIL_IDS[*]}"
  exit 1
fi
exit 0
