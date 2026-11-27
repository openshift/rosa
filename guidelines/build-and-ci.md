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

- Update the `go` and `toolchain` directives in `go.mod`. Raising the `go`
  directive also makes `go mod tidy` reflow the `require` blocks; commit that
  reflow, because the presubmit `test` job runs `make verify`.
- Update the `ubi10/go-toolset` tag in `Dockerfile`, `Dockerfile.clients`,
  `images/Dockerfile.e2e`, and `images/Dockerfile.konflux`.
- Keep the `images/Dockerfile.konflux` runtime stage on the same RHEL major as
  its builder stage: it reuses the builder's Python tree (`/awscli`) and Go
  toolchain, so a mismatch breaks the bundled `aws` CLI.
- Update `renovate.json` `gomod.constraints.go` so it allows the new `go`
  directive, otherwise Renovate's gomod runs fail.
- Drop `.govulncheck-ignore.yaml` stdlib entries the new toolchain fixes.
  Confirm each one at `https://vuln.go.dev/ID/<GO-ID>.json` first.
- Re-pin `.bingo` `golangci-lint` with `bingo get` when the current pin cannot
  parse the new release (`export data version ... greater than maximum
  supported version`). Never hand-edit `.bingo` files.
- Match `.ci-operator.yaml` `build_root_image.tag` to an available
  `ocp/builder:rhel-*-golang-*` for pipeline plumbing. `ocp/builder` publishes
  no `golang-1.27` or `rhel-10-golang-*` tag yet, so it stays on
  `rhel-9-golang-1.26-openshift-4.23`. No rosa ci-operator config sets
  `binary_build_commands`, and every `tests:` entry runs with
  `container.from: rosa-clients` rather than a build-root-derived `src` or
  `root` image, so the build root only clones and drives image builds; each
  image compiles with the `FROM` in its own Dockerfile. Move the tag forward
  once a RHEL 10 / Go 1.27 builder exists.

`images/Dockerfile.release` downloads a released binary and is not tied to
the source Go version. Release-side ci-operator configs are under
`openshift/release/ci-operator/config/openshift/rosa/`. When adding
`.ci-operator.yaml`, set `build_root: from_repository: true` there instead of
duplicating the builder tag.

Do not perform release work or release automation from an agent session.
Treat monthly-release codepaths and user-facing commands with care.
