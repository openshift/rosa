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
