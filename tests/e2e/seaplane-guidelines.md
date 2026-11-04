# Seaplane E2E orchestration

`make test-validate-seaplane` runs every Seaplane-validated spec end to end and
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
`make test-validate-seaplane` prepends the build directory and prints the
resolved path. When driving `ginkgo` by hand, check `command -v rosa` first.

## Run all validated specs

```bash
make test-validate-seaplane
```

The endpoint comes from the rosa config written by `rosa login --seaplane-url`,
which is the same value the CLI itself uses. Exporting `SEAPLANE_URL` is
optional; if you do, it must match the config, because the specs read the
variable while the CLI reads the config and a disagreement would point the two
halves of the run at different endpoints. The target prints the endpoint and
the resolved `rosa` binary before phase 1.

This runs the 67 specs labelled `seaplane-validated` in five ordered phases:
`day1`, `day1-readiness`, the main test body, `destroy`, then `destroy-post`.
The phases exist because Ginkgo randomizes top-level containers, so a single
invocation can tear the cluster down before it is created.

Teardown always runs, including when an earlier phase fails, so a failed spec
does not leak a cluster. The command exits with the test failure when there is
one, otherwise the teardown failure.

Optional environment:

- `TEST_PROFILE` — defaults to `rosa-seaplane-basic`.
- `CLUSTER_NAME` — defaults to `sp-e2e-<unix timestamp>`.
- `GINKGO_SEED` — pin the seed to reproduce a failing run.

Two validated specs are gated to their own profiles and report as skipped
otherwise, so full coverage takes three runs:

| `TEST_PROFILE` | Adds |
| --- | --- |
| `rosa-seaplane-basic` | the default set |
| `rosa-seaplane-nocni` | no-CNI cluster creation |
| `rosa-seaplane-disable-capabilities` | disabled cluster capabilities |

`make e2e-seaplane` delegates to the same phases. Set
`LABEL_FILTER=seaplane-sanity` on it to run the self-contained sanity suite
instead, which needs no phasing.

## Profile lifecycle

These are the individual phases that `make test-validate-seaplane` runs for
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
