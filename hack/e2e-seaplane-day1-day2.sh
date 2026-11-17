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

if [[ "${E2E_SEAPLANE_PREFLIGHT:-true}" == "true" ]]; then
	echo "=== Seaplane E2E: preflight ==="
	if ! preflight_output=$("${ROSA_BIN}" list oidc-config -o json 2>&1); then
		echo "ERROR: Platform API preflight failed: rosa list oidc-config did not succeed." >&2
		echo "This is an endpoint, authentication, or network failure, not a test failure." >&2
		echo "Endpoint: ${SEAPLANE_URL}" >&2
		printf '%s\n' "${preflight_output}" >&2
		exit 3
	fi
fi

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
	local shared_filter=""
	if [[ "${E2E_SEAPLANE_RUN_SHARED}" != "true" ]]; then
		shared_filter=" && !seaplane-shared"
	fi
	if [[ "${phase}" == "destroy" ]]; then
		filter="destroy && seaplane-validated && !seaplane-na && !seaplane-deferred${shared_filter}"
	else
		filter="(${phase}) && seaplane-validated && !seaplane-na && !seaplane-deferred${shared_filter}"
	fi
	run_filter "${phase}" "${filter}"
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
	if run_filter "destroy" "destroy && seaplane-validated && !seaplane-na && !seaplane-deferred"; then
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

# phase_failed_on_oidc_create reports whether a phase failed because the Platform
# API rejected managed OIDC config creation. That is an environment failure: every
# later phase fails the same way, so continuing only buries the cause under
# unrelated assertions.
phase_failed_on_oidc_create() {
	local phase="$1"
	local junit_report="${ARTIFACT_DIR}/junit-${phase}.xml"
	[[ -f "${junit_report}" ]] || return 1
	grep -q 'post oidc_configs' "${junit_report}" || return 1
	echo "ERROR: Platform API rejected managed OIDC config creation (POST /oidc_configs)." >&2
	echo "This is an endpoint or deployment failure, not a test failure; aborting remaining phases." >&2
	echo "Endpoint: ${SEAPLANE_URL}" >&2
	return 0
}

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
	if phase_failed_on_oidc_create "${phase}"; then
		exit 3
	fi
	echo "Seaplane E2E phase failed: ${phase}; continuing with remaining phases" >&2
done

if ((${#failed_phases[@]} > 0)); then
	echo "Seaplane E2E failed phases: ${failed_phases[*]}" >&2
	exit 1
fi
