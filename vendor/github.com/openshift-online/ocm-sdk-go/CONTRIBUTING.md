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

When a new ocm-api-model release is published, the **sync-from-model** workflow automatically:

1. Receives a `repository_dispatch` event from ocm-api-model
2. Bumps the ocm-api-model dependency using `./hack/update-model.sh`
3. Regenerates the SDK using `make update`
4. Creates a PR (e.g., `sync-model/v0.0.468`) with the changes

Review and merge the auto-generated PR. This triggers the release process described below.

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

The SDK release process is fully automated through a 3-stage workflow:

#### Stage 1: Code changes merge to main
When any PR that modifies Go code (`*.go`), `go.mod`, or `go.sum` is merged to main (including sync-from-model PRs), the release process begins automatically.

#### Stage 2: Version bump PR creation
The **auto-version-bump** workflow automatically:

1. Calculates the next patch version (e.g., `v0.1.513`)
2. Updates `version.go` with the new version
3. Updates `CHANGES.md` with all commits since the last release
4. Creates a PR (e.g., `release-v0.1.513`) with these changes

**Action required:** Review and merge the version bump PR.

If multiple PRs merge to main before the version bump PR is merged, the workflow intelligently updates the existing version bump PR with all accumulated changes rather than creating duplicate PRs.

#### Stage 3: Tag and release creation
When the version bump PR is merged, the **auto-tag-and-release** workflow automatically:

1. Extracts the version from `version.go`
2. Creates a git tag (e.g., `v0.1.513`)
3. Creates a GitHub Release with the changelog from `CHANGES.md`

**No action required** — the tag and release are created automatically.

### Summary
- **Code PR** → merge → **auto-version-bump creates PR** → merge → **tag & release created automatically**
- Only 2 manual merges required; final release is fully automated
- Works with any code change, not just model syncs

### Manual (fallback)

If the automated workflow is unavailable, you can manually release a new version:

1. **Update version.go**: Increment the `Version` constant (e.g., `"0.1.513"`)

2. **Update CHANGES.md**: Add a new section with the version and changes:
   ```markdown
   ## 0.1.513 Sep 18 2026

   - Update to model 0.0.468:
     - Add `type` attribute to the `ResourceQuota` type.
     - Add `config_managed` attribute to the `RoleBinding` type.
   ```

3. **Create a PR** with these changes:
   ```shell
   git checkout -b release-v0.1.513
   git add version.go CHANGES.md
   git commit -m "chore: bump version to v0.1.513"
   git push origin release-v0.1.513
   # Create PR and merge after review
   ```

4. **Create and push the tag**:
   ```shell
   git checkout main
   git pull
   git tag -a -m 'Release v0.1.513' v0.1.513
   git push origin v0.1.513
   ```

5. **Create the GitHub Release** manually using the tag and changelog from CHANGES.md

Note that a repository administrator may need to push the tag due to access restrictions.
