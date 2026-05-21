# ROSA Helper Scripts

These scripts help in the process of releasing new versions of `rosa`. For
full details on how to use these scripts in our release process, refer to our
[wiki](https://source.redhat.com/groups/public/ocm_team/ocm_wiki/rosa_release_process)

## release-list-jiras.sh

	./hack/release-list-jiras.sh $release-version

**Note**: This script has a dependency on the [Jira CLI](https://github.com/ankitpokhrel/jira-cli/releases)

This script will create a list of Jira issues that have been included in the current release. Using the Jira CLI, it
will allow you to open each issue and ensure that they have the correct release label.

Required parameters:

* `release-version`: The version of ROSA being released

Example usage: 

```shell
./hack/release-list-jiras.sh v1.2.35 v1.2.34
```

## changelog-generate.sh

This helper generates the historical `CHANGELOG.md` content using
[`git-cliff`](https://git-cliff.org/) and the repository `cliff.toml` config.

Generate the full historical changelog from all stable tags:

```shell
./hack/changelog-generate.sh --bootstrap
```

Generate or update the entry for a specific stable tag:

```shell
./hack/changelog-generate.sh --tag v1.2.63
```

The helper auto-installs a pinned `git-cliff` binary when `git-cliff` is not
already available locally.

## changelog-pr.sh

This helper is the manual fallback for the GitHub Actions changelog automation. It:

1. generates the historical changelog update for a stable tag
2. creates or updates a dedicated changelog branch
3. pushes that branch
4. opens or updates a reviewable PR back to the repository default branch

It requires `GITHUB_TOKEN` to be set.

Example usage:

```shell
GITHUB_TOKEN=... ./hack/changelog-pr.sh --tag v1.2.63
```
