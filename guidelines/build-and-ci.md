# Build and CI

Use this file for CI image roles, Go toolchain alignment, and release boundaries.
For exact local commands and hooks, use [CONTRIBUTING.md](../CONTRIBUTING.md),
the [Makefile](../Makefile), and the referenced hook scripts.

ROSA uses OpenShift ci-operator (configured in `openshift/release`) and Konflux
(`.tekton/`). Each Dockerfile serves a different job type.

| Dockerfile | Built by | Used for |
|------------|----------|----------|
| `Dockerfile` | Konflux | Shipping CLI zips (product) |
| `Dockerfile.clients` | Prow (main config) | Presubmits: lint, build, test, commits, coverage |
| `images/Dockerfile.e2e` | Prow (e2e/images variants) | E2E runner; promoted as `ci/rosa-aws-cli:latest` |
| `images/Dockerfile.release` | Prow (images-release) | E2E against latest GitHub release (`ci/rosa-aws-cli:release`) |
| `images/Dockerfile.konflux` | Konflux | Konflux E2E only |

Presubmit jobs run inside `rosa-clients` from `Dockerfile.clients`, which
copies the repository during image build. ci-operator does not re-clone into
custom images unless `clone: true`. Prow E2E runs inside `rosa-aws-cli` from
`images/Dockerfile.e2e`. `.ci-operator.yaml` pins the ci-operator build root
(`ocp/builder`) for pipeline plumbing, not presubmit execution.

## Go version changes

Keep compilation, presubmits, E2E builds, and product images aligned:

- Update the `go` directive in `go.mod`.
- Update the `ubi9/go-toolset` tag in `Dockerfile`, `Dockerfile.clients`,
  `images/Dockerfile.e2e`, and `images/Dockerfile.konflux`.
- Match `.ci-operator.yaml` `build_root_image.tag` to an available
  `ocp/builder:rhel-9-golang-*` for pipeline plumbing.
- Regenerate `vendor/` when required by `CONTRIBUTING.md`.

`images/Dockerfile.release` downloads a released binary and is not tied to
the source Go version. Release-side ci-operator configs are under
`openshift/release/ci-operator/config/openshift/rosa/`. When adding
`.ci-operator.yaml`, set `build_root: from_repository: true` there instead of
duplicating the builder tag.

Do not perform release work or release automation from an agent session.
Treat monthly-release codepaths and user-facing commands with care.
