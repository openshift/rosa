# Seaplane E2E orchestration

`make e2e_test` runs every Seaplane-validated spec end to end and
is the normal entry point. Use the manual phases further down when you need to
drive a single spec, or to re-run one stage against a cluster already running.

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

Then point the CLI at the endpoint:

```bash
rosa login --seaplane-url "$SEAPLANE_URL"
```

This step is required, not optional. It writes `seaplane_url` into the rosa
config, and the config is what the test handler's `usesSeaplane()` actually
reads — it does not look at the `SEAPLANE_URL` environment variable. Without
it the suite silently takes the OCM path and fails somewhere unrelated, such
as a `NAME_PREFIX` panic in `GenerateClusterCreateFlags`.

One more trap: `make install` builds to `$(go env GOPATH)/bin`, but a `rosa`
earlier on `PATH` (a `/usr/local/bin` or Homebrew install is common) will
shadow it, so the suite exercises that binary instead of your working tree.
`make e2e_test` prepends the build directory and prints the
resolved path. When driving `ginkgo` by hand, check `command -v rosa` first.

## Run all validated specs

```bash
make e2e_test
```

The endpoint comes from the rosa config written by `rosa login --seaplane-url`,
which is the same value the CLI itself uses. Exporting `SEAPLANE_URL` is
optional; if you do, it must match the config, because the specs read the
variable while the CLI reads the config and a disagreement would point the two
halves of the run at different endpoints. The target prints the endpoint and
the resolved `rosa` binary before phase 1.

Output is logged to `tests/output/<profile>/e2e-<timestamp>.log` and streamed
to the terminal in real time. Check the log file for detailed failure messages
when debugging test failures.

This runs the 67 specs labeled `seaplane-validated` in five ordered phases:
`day1`, `day1-readiness`, the main test body, `destroy`, then `destroy-post`.
The phases exist because Ginkgo randomizes top-level containers, so a single
invocation can tear the cluster down before it is created.

Phase 4 always runs, including when an earlier phase fails, so a failed spec
does not leak a cluster. Phase 5 only runs when setup and tests succeeded:
with no cluster there is nothing whose deletion can be verified, and running
it would bury the real failure under an unrelated assertion. The command exits
with the test failure when there is one, otherwise the teardown failure.

If a run leaves a cluster behind, the next one fails during setup with
`a cluster named X already exists in this account`, and because that happens
before any state is written the normal teardown phases have nothing to work
from. Remove it directly:

```bash
make clean-seaplane-cluster CLUSTER_NAME=<name>
```

Set `DELETE_OPERATOR_ROLES=1` to also remove operator roles sharing the
cluster-name prefix, and `DELETE_WATCH=1` to wait for uninstall and stream its
logs. `DELETE_OPERATOR_ROLES` implies `DELETE_WATCH`, because the API refuses
to remove roles while any cluster still references the prefix:

```
There are clusters using Operator Roles Prefix 'X', can't delete the IAM roles
```

That message is about the cluster reference, not the roles themselves — the
IAM roles may already be gone while the cluster deletion is still pending.

Note that `CLUSTER_NAME` is honoured when exported, so a stale value in your
shell will be reused across runs and collide with the cluster it created the
first time. Leave it unset unless you mean to pin it.

Optional environment:

- `TEST_PROFILE` — defaults to `rosa-seaplane-basic`.
- `CLUSTER_NAME` — defaults to `sp-e2e-<unix timestamp>`.
- `GINKGO_SEED` — pin the seed to reproduce a failing run.
- `CLUSTER_TIMEOUT` — minutes to wait for the cluster (default 60).
- `NODEPOOL_TIMEOUT` — minutes to wait for node pools to reach Ready
  (default 90). Raise it if day1 fails with
  `timeout waiting for N node pool(s)` while the logged pool status still
  reads `Provisioning`; that means they were progressing, not stuck.
  Note: Node pools can take 45-60 minutes to provision in some environments.

Two validated specs are gated to their own profiles and report as skipped
otherwise, so full coverage takes three runs:

| `TEST_PROFILE` | Adds |
| --- | --- |
| `rosa-seaplane-basic` | the default set |
| `rosa-seaplane-nocni` | no-CNI cluster creation |
| `rosa-seaplane-disable-capabilities` | disabled cluster capabilities |

`make e2e_test` picks this phased mode automatically when the rosa config has
a `seaplane_url` and `LabelFilter` is unset or `seaplane-validated`. Any other
`LabelFilter` takes the historical single-invocation path, so
`make e2e_test LabelFilter=seaplane-sanity` runs the self-contained sanity
suite, which needs no phasing.

## Profile lifecycle

These are the individual phases that `make e2e_test` runs for
you. Drive them by hand when iterating on a single stage.

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
use `latest`).

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

Then run the deletion check, which must follow the destroy above:

```bash
ginkgo run --label-filter='destroy-post && seaplane-validated' --timeout=5h tests/e2e
```
