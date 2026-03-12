#!/usr/bin/env bash

set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

mode=""
dry_run=0
list_steps=0

print_usage() {
  cat <<'USAGE'
Usage:
  make run-checks -- <mode> [--dry-run] [--list-steps]

Modes:
  pre-push                 Steps: format-check, build, lint, coverage, tests
  basic                    Steps: format, format-check, build, lint, coverage, tests

Flags:
  --dry-run                Print planned steps and commands without executing
  --list-steps             Print planned step names without executing
USAGE
}

append_step() {
  local step_name=$1
  local step_command=$2
  step_names+=("$step_name")
  step_commands+=("$step_command")
}

strip_ansi() {
  sed -E $'s/\x1B\[[0-9;]*[[:alpha:]]//g'
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    pre-push|basic)
      if [ -n "$mode" ] && [ "$mode" != "$1" ]; then
        echo "Only one mode can be specified"
        print_usage
        exit 1
      fi
      mode="$1"
      ;;
    --dry-run)
      dry_run=1
      ;;
    --list-steps)
      list_steps=1
      ;;
    -h|--help)
      print_usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1"
      print_usage
      exit 1
      ;;
  esac
  shift
done

if [ -z "$mode" ]; then
  print_usage
  exit 1
fi

declare -a step_names=()
declare -a step_commands=()

case "$mode" in
  pre-push)
    append_step "Format check (imports + gofmt)" "make --no-print-directory fmt-check"
    append_step "Build" "make --no-print-directory rosa"
    append_step "Lint" "make --no-print-directory lint"
    append_step "Coverage (changed files)" "make --no-print-directory coverage-changed-files"
    append_step "Unit and integration tests" "make --no-print-directory test GO_TEST_FLAGS='-count=1'"
    ;;
  basic)
    append_step "Format (imports + gofmt)" "make --no-print-directory fmt"
    append_step "Format check (imports + gofmt)" "make --no-print-directory fmt-check"
    append_step "Build" "make --no-print-directory rosa"
    append_step "Lint" "make --no-print-directory lint"
    append_step "Coverage (changed files)" "make --no-print-directory coverage-changed-files"
    append_step "Unit and integration tests" "make --no-print-directory test GO_TEST_FLAGS='-count=1'"
    ;;
esac

total_steps=${#step_names[@]}
if [ "$list_steps" -eq 1 ] || [ "$dry_run" -eq 1 ]; then
  if [ "$dry_run" -eq 1 ]; then
    echo "Dry-run mode: commands will not execute."
  fi

  echo "Planned $mode checks ($total_steps steps)"
  for i in "${!step_names[@]}"; do
    step_num=$((i + 1))
    echo "[$step_num/$total_steps] ${step_names[$i]}"
    if [ "$dry_run" -eq 1 ]; then
      echo "  COMMAND: ${step_commands[$i]}"
    fi
  done
  exit 0
fi

interrupted_by_signal=0
handle_run_checks_signal() {
  local signal_name=$1
  local signal_exit_code=130

  if [ "$interrupted_by_signal" -eq 1 ]; then
    return 0
  fi
  interrupted_by_signal=1

  if [ "$signal_name" = "TERM" ]; then
    signal_exit_code=143
  fi

  echo
  echo "Execution interrupted by $signal_name"
  exit "$signal_exit_code"
}
trap 'handle_run_checks_signal INT' INT
trap 'handle_run_checks_signal TERM' TERM

declare -a step_status=()
declare -a step_duration_seconds=()
declare -a step_failure_details=()

failed_steps=0
echo "Running $mode checks ($total_steps steps)"

for i in "${!step_names[@]}"; do
  step_num=$((i + 1))
  step_name=${step_names[$i]}
  step_command=${step_commands[$i]}

  started_at=$(date +%s)

  echo "[$step_num/$total_steps] RUNNING: $step_name"

  set +e
  step_output=$(bash -c "$step_command" 2>&1)
  step_exit_code=$?
  set -e

  finished_at=$(date +%s)
  duration_seconds=$((finished_at - started_at))

  step_duration_seconds+=("$duration_seconds")

  if [ "$step_exit_code" -eq 0 ]; then
    step_status+=("PASS")
    step_failure_details+=("")
    echo "[$step_num/$total_steps] PASS: $step_name (${duration_seconds}s)"
    continue
  fi

  failed_steps=$((failed_steps + 1))
  step_status+=("FAIL")

  if [ -n "$step_output" ]; then
    step_failure_details+=("$(printf '%s' "$step_output" | strip_ansi)")
  else
    step_failure_details+=("Command failed with no output")
  fi

  echo "[$step_num/$total_steps] FAIL: $step_name (${duration_seconds}s)"

  remaining_steps=$((total_steps - step_num))
  if [ "$remaining_steps" -gt 0 ]; then
    echo "[$step_num/$total_steps] FAIL-FAST: skipping remaining $remaining_steps step(s)"
  fi
  break
done

executed_steps=${#step_status[@]}
skipped_steps=$((total_steps - executed_steps))

echo
for i in "${!step_status[@]}"; do
  name=${step_names[$i]}
  status=${step_status[$i]}
  duration=${step_duration_seconds[$i]}
  failure_details=${step_failure_details[$i]}

  if [ "$status" = "PASS" ]; then
    echo "  - [PASS] $name (${duration}s)"
    continue
  fi

  echo "  - [FAIL] $name (${duration}s)"
  if [ -n "$failure_details" ]; then
    while IFS= read -r failure_line; do
      echo "            $failure_line"
    done <<< "$failure_details"
  fi
done

if [ "$skipped_steps" -gt 0 ]; then
  echo "  - [SKIP] $skipped_steps step(s) not run (fail-fast)"
fi

echo
if [ "$failed_steps" -ne 0 ]; then
  echo "Result: FAILED ($failed_steps failure, $skipped_steps skipped, $executed_steps/$total_steps executed)"
  exit 1
fi

echo "Result: PASSED ($total_steps/$total_steps checks passed)"
