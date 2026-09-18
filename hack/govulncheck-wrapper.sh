#!/usr/bin/env bash
# Copyright Red Hat
# SPDX-License-Identifier: Apache-2.0


# govulncheck-wrapper.sh - Run govulncheck while ignoring specified vulnerabilities.
#
# Adapted from openshift/ci-chat-bot hack/govulncheck-wrapper.sh with source/binary
# mode support and optional yq-less YAML parsing for CI environments.
#
# Usage:
#   ./hack/govulncheck-wrapper.sh --mode source [govulncheck patterns...]
#   ./hack/govulncheck-wrapper.sh --mode binary <path-to-binary>
#
# Configuration file format (YAML):
#   ignored_vulnerabilities:
#     - id: GO-2024-12345
#       module: github.com/example/module
#       reason: "No fix available - ..."

set -euo pipefail

print_usage() {
	cat <<'EOF'
Usage: govulncheck-wrapper.sh --mode <source|binary> [target...]

Options:
  --mode MODE       Scan mode: source (reachable dependency CVEs) or binary (stdlib/toolchain CVEs)
  --config FILE     Path to YAML config file (default: .govulncheck-ignore.yaml)
  --verbose         Enable verbose output
  -h, --help        Show this help message

Environment:
  GOVULNCHECK_BIN   Path to the govulncheck binary (default: govulncheck on PATH)

Requires jq to parse govulncheck JSON output. yq is optional; an awk fallback parses
the ignore file when yq is not installed.
EOF
}

CONFIG_FILE=".govulncheck-ignore.yaml"
MODE=""
VERBOSE=0
GOVULNCHECK_BIN="${GOVULNCHECK_BIN:-govulncheck}"
TARGETS=()

while [[ $# -gt 0 ]]; do
	case "$1" in
	--mode)
		MODE="$2"
		shift 2
		;;
	--config)
		CONFIG_FILE="$2"
		shift 2
		;;
	--verbose)
		VERBOSE=1
		shift
		;;
	-h | --help)
		print_usage
		exit 0
		;;
	*)
		TARGETS+=("$1")
		shift
		;;
	esac
done

log_info() {
	echo "[INFO] $*"
}

log_error() {
	echo "[ERROR] $*" >&2
}

if [[ -z "$MODE" ]]; then
	log_error "--mode is required"
	print_usage
	exit 1
fi

if [[ "$MODE" != "source" && "$MODE" != "binary" ]]; then
	log_error "unsupported mode: $MODE"
	exit 1
fi

if [[ ${#TARGETS[@]} -eq 0 ]]; then
	if [[ "$MODE" == "source" ]]; then
		TARGETS=("./...")
	else
		log_error "binary mode requires a path to the compiled binary"
		exit 1
	fi
fi

if [[ "$MODE" == "binary" && ${#TARGETS[@]} -ne 1 ]]; then
	log_error "binary mode supports exactly one binary path"
	exit 1
fi

if [[ ! -x "$GOVULNCHECK_BIN" ]] && ! command -v "$GOVULNCHECK_BIN" >/dev/null; then
	log_error "govulncheck not found: $GOVULNCHECK_BIN"
	exit 1
fi

if [[ ! -f "$CONFIG_FILE" ]]; then
	log_error "config file not found: $CONFIG_FILE"
	exit 1
fi

if ! command -v jq >/dev/null; then
	log_error "jq not found"
	exit 1
fi

load_ignored_list() {
	if command -v yq >/dev/null 2>&1; then
		yq -r '.ignored_vulnerabilities[]? | "\(.id)|\(.module)"' "$CONFIG_FILE" 2>/dev/null || true
		return
	fi

	log_info "yq not found; parsing $CONFIG_FILE with awk fallback"

	awk '
		function trim(value) {
			gsub(/^[ \t]+|[ \t]+$/, "", value)
			gsub(/^"|"$/, "", value)
			return value
		}
		/^[ \t]*-[ \t]*id:/ {
			sub(/^[ \t]*-[ \t]*id:[ \t]*/, "")
			id = trim($0)
			next
		}
		/^[ \t]*module:/ {
			sub(/^[ \t]*module:[ \t]*/, "")
			module = trim($0)
			if (id != "") {
				print id "|" module
				id = ""
			}
			next
		}
	' "$CONFIG_FILE"
}

run_govulncheck() {
	local -a cmd=("$GOVULNCHECK_BIN" -json)
	if [[ "$MODE" == "binary" ]]; then
		cmd+=(-mode binary)
	fi
	cmd+=("${TARGETS[@]}")

	[[ $VERBOSE -eq 1 ]] && log_info "Running: ${cmd[*]}"

	local stdout_file stderr_file
	stdout_file=$(mktemp)
	stderr_file=$(mktemp)
	# shellcheck disable=SC2064
	trap 'rm -f "$stdout_file" "$stderr_file"' RETURN

	set +e
	"${cmd[@]}" >"$stdout_file" 2>"$stderr_file"
	local exit_code=$?
	set -e

	if [[ -s "$stderr_file" ]]; then
		cat "$stderr_file" >&2
	fi

	local output
	output=$(<"$stdout_file")

	if [[ $exit_code -ne 0 ]]; then
		log_error "govulncheck failed (exit $exit_code)"
		[[ -n "$output" ]] && echo "$output" >&2
		return 1
	fi

	if grep -qE '^govulncheck:' <<<"$output"; then
		log_error "govulncheck reported errors"
		echo "$output" >&2
		return 1
	fi

	if [[ -n "$output" ]]; then
		if ! jq_stderr=$(echo "$output" | jq -e -n 'inputs' 2>&1 >/dev/null); then
			log_error "govulncheck output is not valid JSON"
			log_error "$jq_stderr"
			echo "$output" >&2
			return 1
		fi
	fi

	printf '%s' "$output"
}

extract_findings() {
	local vuln_json="$1"
	local jq_filter

	if [[ "$MODE" == "source" ]]; then
		jq_filter='select(.finding) | select(.finding.trace | length > 1) | {id: .finding.osv, module: .finding.trace[0].module, fixed: .finding.fixed_version}'
	else
		jq_filter='select(.finding) | {id: .finding.osv, module: (.finding.trace[0].module // "stdlib"), fixed: .finding.fixed_version}'
	fi

	echo "$vuln_json" | jq -c "$jq_filter"
}

warn_stale_ignore_entries() {
	local matched_ignored="$1"
	local scan_label="$2"

	[[ $VERBOSE -eq 1 ]] || return 0

	local ignored_list entry
	ignored_list=$(load_ignored_list)

	while IFS= read -r entry; do
		[[ -z "$entry" ]] && continue
		if ! grep -qxF "$entry" <<<"$matched_ignored"; then
			log_info "$scan_label: stale ignore entry ${entry%%|*} in ${entry#*|} (not reported by govulncheck)"
		fi
	done <<<"$ignored_list"
}

evaluate_findings() {
	local findings="$1"
	local scan_label="$2"

	if [[ -z "$findings" ]]; then
		log_info "$scan_label: no vulnerabilities found"
		warn_stale_ignore_entries "" "$scan_label"
		return 0
	fi

	local unique_vulns
	unique_vulns=$(echo "$findings" | jq -c -s 'unique_by(.id + .module)' | jq -c '.[]')

	local ignored_list
	ignored_list=$(load_ignored_list)

	local ignored_count=0
	local unignored_count=0
	local unignored_vulns=""
	local matched_ignored=""

	while IFS= read -r vuln; do
		[[ -z "$vuln" ]] && continue

		local vuln_id module fixed
		vuln_id=$(echo "$vuln" | jq -r '.id')
		module=$(echo "$vuln" | jq -r '.module')
		fixed=$(echo "$vuln" | jq -r '.fixed // "N/A"')

		if grep -qxF "${vuln_id}|${module}" <<<"$ignored_list"; then
			((ignored_count++)) || true
			matched_ignored="${matched_ignored}${vuln_id}|${module}"$'\n'
			[[ $VERBOSE -eq 1 ]] && log_info "$scan_label: ignored $vuln_id in $module (listed in $CONFIG_FILE)"
		else
			((unignored_count++)) || true
			unignored_vulns="${unignored_vulns} - $vuln_id in $module (fixed: $fixed)\n"
			[[ $VERBOSE -eq 1 ]] && log_error "$scan_label: found $vuln_id in $module (fixed: $fixed)"
		fi
	done <<<"$unique_vulns"

	log_info "$scan_label: $unignored_count unignored, $ignored_count ignored"
	warn_stale_ignore_entries "$matched_ignored" "$scan_label"

	if [[ $unignored_count -gt 0 ]]; then
		log_error "$scan_label: unignored vulnerabilities found:"
		echo -e "$unignored_vulns" >&2
		return 1
	fi

	return 0
}

[[ $VERBOSE -eq 1 ]] && log_info "Using config file: $CONFIG_FILE"

vuln_json=$(run_govulncheck)
findings=$(extract_findings "$vuln_json")
scan_label="govulncheck ($MODE)"
evaluate_findings "$findings" "$scan_label"
