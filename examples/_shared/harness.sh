#!/usr/bin/env bash
# harness.sh — tiny assertion harness shared by the examples' assert.sh
# scripts. Same reporting style as test/smoke: behavioral assertions only
# (exit codes + substrings), never byte-exact goldens.
#
# Source this file, then use:
#   check <id> <expected_exit> <substring> -- cmd args...
#     Runs cmd, asserts exit code and (if non-empty) that the substring
#     appears in combined stdout+stderr.
#   check_gated <id> <expected_exit> <substring> -- cmd args...
#     Same, but only counts as a failure when GATED=1 — used for
#     assertions that document the TARGET behavior of features the
#     current alpha has not wired yet. Without GATED=1 they report
#     their live verdict as informational (GATE-PASS / GATE-PENDING).
#   summary
#     Prints totals; exits 1 if any non-gated check failed.

PASS=0
FAIL=0
GPASS=0
GPEND=0
FAIL_IDS=()

_run_capture() {
  set +e
  CHECK_OUT="$("$@" 2>&1)"
  CHECK_EXIT=$?
  set -e
}

check() {
  local id="$1" want_exit="$2" want_sub="$3"
  shift 3
  [[ "$1" == "--" ]] && shift
  _run_capture "$@"
  if [[ "${CHECK_EXIT}" -eq "${want_exit}" && ( -z "${want_sub}" || "${CHECK_OUT}" == *"${want_sub}"* ) ]]; then
    PASS=$((PASS + 1))
    printf '  ✓ %s\n' "${id}"
  else
    FAIL=$((FAIL + 1))
    FAIL_IDS+=("${id}")
    printf '  ✗ %s — exit=%s (want %s) out=%.200s\n' "${id}" "${CHECK_EXIT}" "${want_exit}" "${CHECK_OUT}"
  fi
}

check_gated() {
  local id="$1" want_exit="$2" want_sub="$3"
  shift 3
  [[ "$1" == "--" ]] && shift
  _run_capture "$@"
  if [[ "${CHECK_EXIT}" -eq "${want_exit}" && ( -z "${want_sub}" || "${CHECK_OUT}" == *"${want_sub}"* ) ]]; then
    GPASS=$((GPASS + 1))
    printf '  ★ %s — GATE-PASS (target behavior now works; ungate this check)\n' "${id}"
  else
    if [[ "${GATED:-0}" == "1" ]]; then
      FAIL=$((FAIL + 1))
      FAIL_IDS+=("${id}")
      printf '  ✗ %s — GATED FAIL exit=%s out=%.200s\n' "${id}" "${CHECK_EXIT}" "${CHECK_OUT}"
    else
      GPEND=$((GPEND + 1))
      printf '  ⊘ %s — GATE-PENDING (expected: not yet wired in this alpha) out=%.120s\n' "${id}" "${CHECK_OUT}"
    fi
  fi
}

wait_for_socket() {
  local path="$1" tries="${2:-50}"
  for _ in $(seq 1 "${tries}"); do
    [[ -S "${path}" ]] && return 0
    sleep 0.1
  done
  echo "timeout waiting for socket ${path}" >&2
  return 1
}

wait_for_tcp() {
  local host="$1" port="$2" tries="${3:-50}"
  for _ in $(seq 1 "${tries}"); do
    (exec 3<>"/dev/tcp/${host}/${port}") 2>/dev/null && { exec 3>&- 3<&-; return 0; }
    sleep 0.1
  done
  echo "timeout waiting for tcp ${host}:${port}" >&2
  return 1
}

summary() {
  printf '\n%d passed, %d failed' "${PASS}" "${FAIL}"
  [[ $((GPASS + GPEND)) -gt 0 ]] && printf ', gated: %d now-passing, %d pending' "${GPASS}" "${GPEND}"
  printf '\n'
  if [[ "${FAIL}" -gt 0 ]]; then
    printf 'failures:\n'
    for id in "${FAIL_IDS[@]}"; do printf '  - %s\n' "${id}"; done
    exit 1
  fi
  exit 0
}
