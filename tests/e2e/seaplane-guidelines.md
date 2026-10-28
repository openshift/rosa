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
