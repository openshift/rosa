# Seaplane E2E orchestration

Use this runbook for profile-based Seaplane E2E orchestration: prepare a
cluster, run one or more focused specs, then destroy the cluster and its
profile-managed infrastructure.

## Prerequisites

- Run from the ROSA CLI repository root with `ginkgo` available on `PATH`.
- Set `SEAPLANE_URL` to the Platform API v2 endpoint.
- Set `TEST_PROFILE` to a profile defined in `tests/ci/data/profiles/rosa-seaplane.yaml`.
- Configure AWS credentials through the standard AWS SDK credential chain; the
  caller must be authorized to create and destroy resources for the profile.
- Use the same `SEAPLANE_URL`, `TEST_PROFILE`, `SHARED_DIR`, and
  `ARTIFACT_DIR` for setup, focused tests, and teardown.

The Make targets create an isolated `OCM_CONFIG` under the output directory and
run `rosa login --seaplane-url` before starting Ginkgo. This lets the CLI
subprocesses launched by the tests route to the Platform API without modifying
your user-level ROSA configuration. They also prepend `GOBIN` (default
`$(go env GOPATH)/bin`) to `PATH`, so the CLI binary installed by
`make install` is used instead of any older `rosa` already on `PATH`.

Seaplane uses AWS SigV4; an OCM token is not required. Set `AWS_DEFAULT_REGION`
if the region cannot be read from the profile or derived from the endpoint URL.
Install the local CLI before starting:

```bash
make install
```

## Profile lifecycle

Run the profile-based cluster preparation spec:

```bash
ginkgo run --focus-file='tests/e2e/e2e_setup_test.go' --focus='by profile' --label-filter='day1 && seaplane-validated' --timeout=5h tests/e2e
```

Run the cluster readiness spec after preparation:

```bash
ginkgo run --focus-file='tests/e2e/e2e_setup_test.go' --focus='to wait for cluster ready' --label-filter='day1-readiness && seaplane-validated' --timeout=5h tests/e2e
```

The profile test stores cluster details in the profile output directory. Set
`CLUSTER_ID` explicitly if the cluster ID is not available from that state.

## Day 1 then Day 2

Run the complete profile-backed lifecycle with one command:

```bash
SEAPLANE_URL=https://example.execute-api.us-east-1.amazonaws.com \
  TEST_PROFILE=rosa-seaplane-basic \
  make e2e-seaplane-day1-day2
```

To run the lifecycle once for every profile that can back the shared cluster,
use the profile matrix target:

```bash
SEAPLANE_URL=https://example.execute-api.us-east-1.amazonaws.com \
  OCP_IMAGE=quay.io/openshift-release-dev/ocp-release:5.0.0-rc.5-multi \
  make e2e-seaplane-profiles
```

Each profile runs in its own Ginkgo process with a generated cluster name and
an isolated directory under `tests/output/<profile>/`. `OCP_IMAGE` or
`SEAPLANE_VERSION` is required for profiles whose `version` is not a release
image pullspec (`rosa-seaplane-advanced` and `rosa-seaplane-private` currently
use `latest`). See `tests/e2e/seaplane-profile-matrix.md` for profile-specific
test targets.

This runs these phases in order, reusing one cluster and the same
`SHARED_DIR`/`ARTIFACT_DIR`:

1. `day1`
2. `day1-readiness`
3. `day1-post`
4. `day1-supplemental`
5. `day1-negative`
6. `day2`

The profile-managed cluster is destroyed on exit, including after a failure.
Set `E2E_SEAPLANE_KEEP_CLUSTER=true` to retain it for debugging, or
`E2E_SEAPLANE_DESTROY=false` when another system owns teardown. To run a subset
in a CI step, set `E2E_SEAPLANE_PHASES`, for example
`E2E_SEAPLANE_PHASES="day1 day1-readiness"`.

The equivalent direct Ginkgo sequence is:

```bash
set -euo pipefail

export SEAPLANE_URL=https://example.execute-api.us-east-1.amazonaws.com
export TEST_PROFILE=rosa-seaplane-basic
export CLUSTER_NAME=sp-e2e-$(date +%s)
export OPERATOR_ROLES_PREFIX="${CLUSTER_NAME}"
export SHARED_DIR="${PWD}/tests/output/${TEST_PROFILE}"
export ARTIFACT_DIR="${SHARED_DIR}"
mkdir -p "${SHARED_DIR}" "${ARTIFACT_DIR}"

run_phase() {
  local phase="$1"
  ginkgo run \
    --label-filter="(${phase}) && seaplane-validated && !seaplane-na && !seaplane-deferred" \
    --timeout=4h \
    --junit-report="junit-${phase}.xml" \
    --output-dir="${ARTIFACT_DIR}" \
    -v \
    ./tests/e2e/...
}

cleanup() {
  local rc=$?
  trap - EXIT
  if [[ -f "${SHARED_DIR}/cluster-detail.json" ]]; then
    ginkgo run \
      --label-filter="destroy && seaplane-validated && !seaplane-na && !seaplane-deferred" \
      --timeout=4h \
      --junit-report=junit-destroy.xml \
      --output-dir="${ARTIFACT_DIR}" \
      -v \
      ./tests/e2e/... || rc=1
  fi
  exit "${rc}"
}
trap cleanup EXIT

for phase in day1 day1-readiness day1-post day1-supplemental day1-negative day2; do
  run_phase "${phase}"
done
```

Each phase is a separate Ginkgo run because Ginkgo randomizes top-level
containers, cluster preparation must precede readiness and Day 2, and the
`destroy` spec shares the `seaplane-validated` label. Specs that require a
different profile (for example the capability-disabled or no-CNI cases) are
skipped under `rosa-seaplane-basic`; run those with the focused make targets:

```bash
make e2e-seaplane-nocni
make e2e-seaplane-disable-capabilities
```

### Prow

Prefer a Seaplane-specific chain beside
`ci-operator/step-registry/rosa/tests/e2e/fullcycle/rosa-tests-e2e-fullcycle-chain.yaml`
in `openshift/release`. Use one `openshift/rosa`-sourced step per phase, for
example:

```text
day1 -> day1-readiness -> day1-post -> day1-supplemental -> day1-negative
     -> day2 -> destroy -> check-failures
```

Each phase step can call this target with `E2E_SEAPLANE_PHASES=<phase>` and
`E2E_SEAPLANE_DESTROY=false`; use `E2E_SEAPLANE_PHASES=destroy` for a final
destroy step. The step image must have `ginkgo` on `PATH`, or set `GINKGO_BIN`.
All steps must share `SHARED_DIR`, and each should get its own timeout and JUnit
artifact. Add destroy as a `best_effort` post step as well so canceled jobs do
not leak clusters.

The single-step alternative is to call `make e2e-seaplane-day1-day2` from a
Seaplane e2e step that checks out the ROSA CLI, but raise its timeout: the
current `rosa-seaplane-e2e-ref.yaml` value is 7200s, which is not enough for the
full sequence. A single step also gives less granular retries and failure reporting.

## Focused tests

Run a single E2E spec by matching both its source file and its `It` description,
and select its labels with `--label-filter`:

```bash
ginkgo run --focus-file='tests/e2e/<source-file>.go' --focus='<It description>' --label-filter='<label expression>' --timeout=5h tests/e2e
```

For local CLI unit tests, use the relevant package and Ginkgo focus:

```bash
go test ./cmd/<command>/<resource> -ginkgo.focus='<spec description>'
```

## Teardown

Always run the profile-based destroy spec after testing, including when a
focused spec fails after cluster creation:

```bash
ginkgo run --focus-file='tests/e2e/e2e_tear_down_test.go' --focus='by profile' --label-filter='destroy && seaplane-validated' --timeout=5h tests/e2e
```

The destroy handler tears down the profile-managed cluster and associated
infrastructure. Confirm that it reports no destroy errors before considering
the run complete.
