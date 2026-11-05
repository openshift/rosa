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
)

if [[ -n "${E2E_SEAPLANE_PHASES:-}" ]]; then
	read -r -a phases <<<"${E2E_SEAPLANE_PHASES}"
else
	phases=("${default_phases[@]}")
fi

for phase in "${phases[@]}"; do
	case "${phase}" in
	day1 | day1-readiness | day1-post | day1-supplemental | day1-negative | day2 | destroy) ;;
	*)
		echo "unsupported Seaplane E2E phase: ${phase}" >&2
		exit 2
		;;
	esac
done

run_filter() {
	local name="$1"
	local filter="$2"
	echo
	echo "=== Seaplane E2E: ${name} ==="
	echo "label filter: ${filter}"
	"${GINKGO_BIN}" run \
		--label-filter="${filter}" \
		--timeout="${E2E_SEAPLANE_TIMEOUT}" \
		--junit-report="junit-${name}.xml" \
		--output-dir="${ARTIFACT_DIR}" \
		-v \
		./tests/e2e/...
}

run_phase() {
	local phase="$1"
	local filter
	if [[ "${phase}" == "destroy" ]]; then
		filter="destroy && seaplane-validated && !seaplane-na && !seaplane-deferred"
	else
		filter="(${phase}) && seaplane-validated && !seaplane-na && !seaplane-deferred"
	fi
	run_filter "${phase}" "${filter}"
}

destroy_cluster() {
	if [[ " ${phases[*]} " == *" destroy "* ]]; then
		echo "Skipping automatic destroy; explicit destroy phase requested"
		return 0
	fi
	if [[ "${E2E_SEAPLANE_DESTROY}" != "true" ]]; then
		echo "Skipping cluster destroy (E2E_SEAPLANE_DESTROY=${E2E_SEAPLANE_DESTROY})"
		return 0
	fi
	if [[ "${E2E_SEAPLANE_KEEP_CLUSTER}" == "true" ]]; then
		echo "Retaining cluster ${CLUSTER_NAME} (E2E_SEAPLANE_KEEP_CLUSTER=true)"
		return 0
	fi
	if [[ ! -f "${SHARED_DIR}/cluster-detail.json" ]]; then
		echo "No profile-managed cluster state found in ${SHARED_DIR}; skipping destroy"
		return 0
	fi
	run_filter "destroy" "destroy && seaplane-validated && !seaplane-na && !seaplane-deferred"
}

cleanup() {
	local status=$?
	trap - EXIT
	if ! destroy_cluster; then
		echo "Cluster destroy failed; check junit-destroy.xml in ${ARTIFACT_DIR}" >&2
		status=1
	fi
	exit "${status}"
}
trap cleanup EXIT

failed_phases=()
for phase in "${phases[@]}"; do
	if run_phase "${phase}"; then
		continue
	fi
	failed_phases+=("${phase}")
	if [[ "${E2E_SEAPLANE_CONTINUE_ON_FAILURE}" != "true" ]]; then
		exit 1
	fi
	echo "Seaplane E2E phase failed: ${phase}; continuing with remaining phases" >&2
done

if ((${#failed_phases[@]} > 0)); then
	echo "Seaplane E2E failed phases: ${failed_phases[*]}" >&2
	exit 1
fi
