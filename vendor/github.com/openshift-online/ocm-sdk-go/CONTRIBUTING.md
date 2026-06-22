# Contributing to the OCM SDK

## Releasing a new OCM API Model version

This section describes the release process in the [ocm-api-model](https://github.com/openshift-online/ocm-api-model) repository. Once a model release is published, it triggers the SDK update process described in the next section.

First, all changes to the model have been defined and reviewed. Then, the client types for the model need to be generated via `make update` target in the `ocm-api-model` project.

Once all changes have been committed to the main branch, the automated release pipeline in **ocm-api-model** handles the rest:

1. **Auto-tag** (runs in ocm-api-model) — a GitHub Action automatically bumps the patch version, regenerates `clientapi/` and `openapi/`, updates `CHANGES.md`, and pushes all sub-module tags.
2. **Release** (runs in ocm-api-model) — the tag push triggers a GitHub Release.
3. **SDK sync** (runs in ocm-api-model) — the release sends a `repository_dispatch` event to this repository (ocm-sdk-go), triggering the SDK update below.

If the automation is not available, you can manually tag and release in ocm-api-model:

```shell
make update
git add -A
git commit -m "Release vX.Y.Z"
git tag vX.Y.Z
git tag clientapi/vX.Y.Z
git tag model/vX.Y.Z
git tag metamodel_generator/vX.Y.Z
git push origin main --tags
```

### Validating model updates

If you would like to test the SDK against a *local version* use the following instructions:

Ensure ocm-sdk-go is cloned locally alongside your cloned ocm-api-model directory where changes are made.

Use the following commands to test you're locally generated client types:
```
go mod edit -replace=github.com/openshift-online/ocm-api-model/clientapi=/path/to/your/local/ocm-api-model/clientapi

go mod edit -replace=github.com/openshift-online/ocm-api-model/model=/path/to/your/local/ocm-api-model/model

make update
```

## Updating the OCM SDK

### Automated (recommended)

When a new ocm-api-model release is published, a GitHub Action automatically:

1. Receives a `repository_dispatch` event from ocm-api-model
2. Bumps the ocm-api-model dependency using `./hack/update-model.sh`
3. Regenerates the SDK using `make update`
4. Opens a PR with the changes

Review and merge the auto-generated PR. On merge, a new SDK version tag is created automatically.

### Manual

The OCM SDK can be generated simply by running the following after all changes have been made:

```shell
./hack/update-model.sh
make update
```

The `./hack/update-model.sh` script will ensure the `ocm-api-model` modules are all up to date with the latest version across the OCM-SDK project.
To verify that they are all in-sync one can use the `./hack/verify-model-version.sh` script.

One can add an optional commit SHA or version to the `./update-model.sh <vX.Y.Z>` script to update the go modules to a specific version.

Whenever an update is made, ensure that the corresponding example in [examples](examples) is also updated where
necessary. It is *highly recommended* that new endpoints have a new example created.

## Releasing a new OCM SDK Version

### Automated (recommended)

On merge to main, a GitHub Action automatically bumps the patch version and pushes a new tag. The existing `publish-release` workflow then creates the GitHub Release.

### Manual

Releasing a new version requires submitting an MR for review/merge with an update to the `Version` constant in
[version.go](version.go). Additionally, update the [CHANGES.md](CHANGES.md) file to include the new version and
describe all changes included.

Below is an example CHANGES.md update:

```
== 0.1.39 Oct 7 2019

- Update to model 0.0.9:
  - Add `type` attribute to the `ResourceQuota` type.
  - Add `config_managed` attribute to the `RoleBinding` type.
```

Submit an MR for review/merge with the CHANGES.md and version.go update.

Finally, create and submit a new tag with the new version following the below example:

```shell
git checkout main
git pull
git tag -a -m 'Release 0.1.39' v0.1.39
git push origin v0.1.39
```

Note that a repository administrator may need to push the tag to the repository due to access restrictions.
