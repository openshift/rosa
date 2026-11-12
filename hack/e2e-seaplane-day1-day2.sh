#!/usr/bin/env bash
# Copyright Red Hat
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "${REPO_ROOT}"

GINKGO_BIN="${GINKGO_BIN:-ginkgo}"
if ! command -v "${GINKGO_BIN}" >/dev/null 2>&1; then
	echo "ginkgo is required on PATH; install it or set GINKGO_BIN" >&2
	exit 1
fi
ROSA_BIN="${ROSA_BIN:-$(go env GOPATH)/bin/rosa}"
if [[ ! -x "${ROSA_BIN}" ]]; then
	echo "rosa binary not found at ${ROSA_BIN}; run 'make install' or set ROSA_BIN" >&2
	exit 1
fi
export PATH="$(dirname "${ROSA_BIN}"):${PATH}"

: "${SEAPLANE_URL:?SEAPLANE_URL is required}"
: "${TEST_PROFILE:?TEST_PROFILE is required}"

CLUSTER_NAME="${CLUSTER_NAME:-sp-e2e-$(date +%s)}"
OPERATOR_ROLES_PREFIX="${OPERATOR_ROLES_PREFIX:-${CLUSTER_NAME}}"
SHARED_DIR="${SHARED_DIR:-${REPO_ROOT}/tests/output/${TEST_PROFILE}}"
ARTIFACT_DIR="${ARTIFACT_DIR:-${SHARED_DIR}}"
E2E_SEAPLANE_TIMEOUT="${E2E_SEAPLANE_TIMEOUT:-4h}"
E2E_SEAPLANE_DESTROY="${E2E_SEAPLANE_DESTROY:-true}"
E2E_SEAPLANE_KEEP_CLUSTER="${E2E_SEAPLANE_KEEP_CLUSTER:-false}"
E2E_SEAPLANE_CONTINUE_ON_FAILURE="${E2E_SEAPLANE_CONTINUE_ON_FAILURE:-false}"
E2E_SEAPLANE_RUN_SHARED="${E2E_SEAPLANE_RUN_SHARED:-true}"

mkdir -p "${SHARED_DIR}" "${ARTIFACT_DIR}"
export OCM_CONFIG="${OCM_CONFIG:-${SHARED_DIR}/rosa-config.json}"
# Ginkgo talks to the Platform API in-process via SEAPLANE_URL, but the rosa
# subprocesses it launches only use --seaplane-url or the saved CLI config.
"${ROSA_BIN}" login --seaplane-url="${SEAPLANE_URL}" >/dev/null
export SEAPLANE_URL TEST_PROFILE CLUSTER_NAME OPERATOR_ROLES_PREFIX
export SHARED_DIR ARTIFACT_DIR OCM_CONFIG GOTOOLCHAIN=auto
if [[ -n "${AWS_DEFAULT_REGION:-}" ]]; then
	export AWS_DEFAULT_REGION
fi
if [[ -n "${CLUSTER_ID:-}" ]]; then
	export CLUSTER_ID
fi

default_phases=(
	day1
	day1-readiness
	day1-post
	day1-supplemental
	day1-negative
	day2
	destroy-post
)

explicit_phases=false
if [[ -n "${E2E_SEAPLANE_PHASES:-}" ]]; then
	read -r -a phases <<<"${E2E_SEAPLANE_PHASES}"
	explicit_phases=true
else
	phases=("${default_phases[@]}")
fi

for phase in "${phases[@]}"; do
	case "${phase}" in
	day1 | day1-readiness | day1-post | day1-supplemental | day1-negative | day2 | destroy | destroy-post) ;;
	*)
		echo "unsupported Seaplane E2E phase: ${phase}" >&2
		exit 2
		;;
	esac
done

# A spec that passed on an earlier profile is appended to a ledger shared by the
# whole matrix and skipped on the profiles that follow, so identical work is not
# repeated once it is known to pass. The Makefile creates the ledger per matrix
# run; when it is unset this script runs a single profile and dedupes nothing.
passed_ledger="${E2E_SEAPLANE_PASSED_LEDGER:-}"
dedupe_passed="${E2E_SEAPLANE_DEDUPE_PASSED:-true}"

# Only the phases that assert against a cluster someone else set up are deduped.
# day1 and day1-readiness create each profile's cluster and destroy tears it
# down, so those must run once per profile no matter what an earlier profile
# recorded.
dedupe_phases=(day1-post day1-supplemental day1-negative day2)

dedupes_phase() {
	local phase="$1"
	local candidate
	for candidate in "${dedupe_phases[@]}"; do
		if [[ "${candidate}" == "${phase}" ]]; then
			return 0
		fi
	done
	return 1
}

# skip_file_args prints one --skip-file flag per file holding passed specs.
# Ginkgo matches --skip-file against spec code locations, which avoids depending
# on spec text or on the labels Ginkgo appends to report names.
skip_file_args() {
	if ! dedupes_phase "$1" || [[ "${dedupe_passed}" != "true" || ! -s "${passed_ledger}" ]]; then
		return 0
	fi
	awk -F: '
		{ lines[$1] = lines[$1] "," $2 }
		END {
			for (file in lines) {
				sub(/^,/, "", lines[file])
				print "--skip-file=" file ":" lines[file]
			}
		}' "${passed_ledger}"
}

# record_passed_specs appends the code location of every passed spec to the
# ledger. Failed specs stay out of it so a later profile retries them.
record_passed_specs() {
	local phase="$1"
	local report="$2"
	if ! dedupes_phase "${phase}" || [[ "${dedupe_passed}" != "true" || -z "${passed_ledger}" || ! -s "${report}" ]]; then
		return 0
	fi
	if ! command -v jq >/dev/null 2>&1; then
		echo "WARNING: jq is required to dedupe passed specs; not updating ${passed_ledger}" >&2
		return 0
	fi
	local new_entries="${passed_ledger}.new"
	{
		if [[ -f "${passed_ledger}" ]]; then
			cat "${passed_ledger}"
		fi
		jq -r '
			.[]
			| .SpecReports[]
			| select(.LeafNodeType == "It" and .State == "passed")
			| "\(.LeafNodeLocation.FileName):\(.LeafNodeLocation.LineNumber)"' "${report}"
	} | sort -u >"${new_entries}"
	mv "${new_entries}" "${passed_ledger}"
}

run_filter() {
	local name="$1"
	local filter="$2"
	local status=0
	echo
	echo "=== Seaplane E2E: ${name} ==="
	echo "label filter: ${filter}"
	local skip_args=()
	while IFS= read -r skip_arg; do
		skip_args+=("${skip_arg}")
	done < <(skip_file_args "${name}")
	"${GINKGO_BIN}" run \
		${skip_args[@]+"${skip_args[@]}"} \
		--label-filter="${filter}" \
		--timeout="${E2E_SEAPLANE_TIMEOUT}" \
		--junit-report="junit-${name}.xml" \
		--json-report="json-${name}.json" \
		--output-dir="${ARTIFACT_DIR}" \
		-v \
		./tests/e2e/... || status=$?
	record_passed_specs "${name}" "${ARTIFACT_DIR}/json-${name}.json"
	return "${status}"
}

# Profile-scoped specs carry a seaplane-profile-* label. Specs without one run
# on every profile; specs with one run only on the matching TEST_PROFILE. The
# harness excludes the labels of every other known profile, so a spec written
# for one profile never runs against another.
known_profile_labels=(
	seaplane-profile-basic
	seaplane-profile-fips
	seaplane-profile-advanced
	seaplane-profile-private
	seaplane-profile-nocni
	seaplane-profile-disable-capabilities
)
current_profile_label="seaplane-profile-${TEST_PROFILE#rosa-seaplane-}"
profile_filter=""
profile_label_known=false
for label in "${known_profile_labels[@]}"; do
	if [[ "${label}" == "${current_profile_label}" ]]; then
		profile_label_known=true
	else
		profile_filter+=" && !${label}"
	fi
done
if [[ "${profile_label_known}" != "true" ]]; then
	echo "WARNING: ${TEST_PROFILE} has no seaplane-profile-* label; running only profile-agnostic specs" >&2
fi

phase_filter() {
	local phase="$1"
	local shared_filter=""
	if [[ "${E2E_SEAPLANE_RUN_SHARED}" != "true" ]]; then
		shared_filter=" && !seaplane-shared"
	fi
	if [[ "${phase}" == "destroy" ]]; then
		printf 'destroy && seaplane-validated && !seaplane-na && !seaplane-deferred%s%s' "${shared_filter}" "${profile_filter}"
	else
		printf '(%s) && seaplane-validated && !seaplane-na && !seaplane-deferred%s%s' "${phase}" "${shared_filter}" "${profile_filter}"
	fi
}

run_phase() {
	run_filter "$1" "$(phase_filter "$1")"
}

# cluster_destroy_state records what happened during teardown so that
# destroy-post runs exactly once, and only after the cluster is actually gone.
cluster_destroy_state=pending

destroy_cluster() {
	if [[ "${E2E_SEAPLANE_DESTROY}" != "true" ]]; then
		echo "Skipping cluster destroy (E2E_SEAPLANE_DESTROY=${E2E_SEAPLANE_DESTROY})"
		cluster_destroy_state=skipped
		return 0
	fi
	if [[ "${E2E_SEAPLANE_KEEP_CLUSTER}" == "true" ]]; then
		echo "Retaining cluster ${CLUSTER_NAME} (E2E_SEAPLANE_KEEP_CLUSTER=true)"
		cluster_destroy_state=skipped
		return 0
	fi
	if [[ ! -f "${SHARED_DIR}/cluster-detail.json" ]]; then
		echo "No profile-managed cluster state found in ${SHARED_DIR}; skipping destroy"
		cluster_destroy_state=skipped
		return 0
	fi
	if run_filter "destroy" "$(phase_filter destroy)"; then
		cluster_destroy_state=destroyed
		return 0
	fi
	cluster_destroy_state=failed
	return 1
}

cleanup() {
	local status=$?
	trap - EXIT
	case "${cluster_destroy_state}" in
	pending)
		if ! destroy_cluster; then
			echo "Cluster destroy failed; check junit-destroy.xml in ${ARTIFACT_DIR}" >&2
			[[ "${status}" -eq 0 ]] && status=1
		fi
		;;
	failed)
		echo "Cluster destroy failed; check junit-destroy.xml in ${ARTIFACT_DIR}" >&2
		[[ "${status}" -eq 0 ]] && status=1
		;;
	esac
	exit "${status}"
}
trap cleanup EXIT

failed_phases=()
for phase in "${phases[@]}"; do
	case "${phase}" in
	destroy)
		if run_phase destroy; then
			cluster_destroy_state=destroyed
			continue
		fi
		cluster_destroy_state=failed
		failed_phases+=("destroy")
		;;
	destroy-post)
		# An explicit phase list means each phase may run in its own CI step, so
		# teardown can have happened in an earlier process. Only the default
		# sequence needs to prove the cluster was destroyed here.
		if [[ "${explicit_phases}" != "true" && "${cluster_destroy_state}" == "pending" ]]; then
			destroy_cluster || true
		fi
		if [[ "${explicit_phases}" != "true" && "${cluster_destroy_state}" != "destroyed" ]]; then
			echo "Skipping destroy-post: cluster teardown state is ${cluster_destroy_state}" >&2
			if [[ "${cluster_destroy_state}" == "failed" ]]; then
				failed_phases+=("destroy")
			fi
			continue
		fi
		if run_phase destroy-post; then
			# With an explicit phase list, teardown ran in an earlier step and the
			# post-check just proved the cluster is gone, so the exit-time safety
			# net would only try to delete it a second time.
			if [[ "${explicit_phases}" == "true" ]]; then
				cluster_destroy_state=destroyed
			fi
			continue
		fi
		failed_phases+=("destroy-post")
		;;
	*)
		if run_phase "${phase}"; then
			continue
		fi
		failed_phases+=("${phase}")
		;;
	esac
	if [[ "${E2E_SEAPLANE_CONTINUE_ON_FAILURE}" != "true" ]]; then
		exit 1
	fi
	echo "Seaplane E2E phase failed: ${phase}; continuing with remaining phases" >&2
done

if ((${#failed_phases[@]} > 0)); then
	echo "Seaplane E2E failed phases: ${failed_phases[*]}" >&2
	exit 1
fi
