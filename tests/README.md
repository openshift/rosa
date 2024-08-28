# ROSA CLI Function Verification Testing

This package is the automation package for Function Verification Testing on the ROSA CLI.

## Structure of tests

```bash
tests
|____e2e      
    |____e2e_suite_test.go                        ---- test suite for all the e2e tests
    |____...                                      ---- other tests organized in domain files
|____utils
|    |____common                                  ---- package which contains common methods 
|    |    |____constants                          ---- package for all constants
|    |____config                                  ---- package which contains tests configuration methods 
|    |____exec                                    ---- exec contains the different services to run the commands
|    |    |____rosacli                            ---- ROSA CLI specific services/commands
|    |____log                                     ---- tests logger
|    |____handler                                 ---- package which support create cluster and resources
|       
|____prow_ci.sh
```

## Contibute to ROSA CLI tests

Please read the structure and contribute code to the correct place

### Contribute to day1

1. Enable the configuration in _[profiles](./ci/data/profiles)_
2. Mapping the configuration in _[ClusterConfig](./utils/handler/interface.go)_
3. Define the userdata preparation function in _[data_preparation](./utils/handler/data_preparation.go)_
4. Call the functions in the _[GenerateClusterCreateFlags](./utils/handler/profile_handler.go)_

### Contribute to day2

* Create the case in _rosa/tests/e2e/{feature name}_test.go_
* Label the case with ***Day2***
* Label the case with importance ***Critical*** or ***High***
* Don't need to run creation step, just in BeforeEach step call function  ***config.GetClusterID()*** it will load the clusterID prepared from env or cluster_id file
* Code for day2 actions and check step
* Every case need to recover the cluster after the case run finished unless it's un-recoverable
* Case format should follow 
  * _main feature description in Describe level_ at the same time _testing purpose in It level_.
  * `id` of the case is included which will follow the fmt of _[id:\<id\>]_
  * Use `By("")` to describe the steps
  * An example as below

```golang
var _ = Describe("Create Machine Pool", func() {
  It("to hosted cluster with additional security group IDs will work - [id:72195]",func(){
    By("Prepare security groups")
    // security groups preparation code

    By("Create machinepool with security groups configured")
    // machinepool creation code

    By("Verify the machinepool is created with security groups")
    // machinepool security groups verification code
  })
}
```

  * The commit and PR should follow
    * Only one commit is allowed per PR, if multiple commits created please squash them with command
    `git rebase -i HEAD~N`(_N_ is the commits number you would squashed)
    * The commit and PR title should follow rule of [contributing-to-rosa](../CONTRIBUTING.md#contributing-to-rosa)
    * Case id must be included in the PR/commit title if new automated or updated. Comma-separated if multiple included in same PR/commit. For example
    `<card id> | test: automated cases id:123456,123457`

### Labels
Label Design Doc: [ROSA CLI Automation labels](./ci/labels/labels.md).

* Label your case with the feature defined in [features.go](./ci/labels/features.go). The feature label is always defined at the top `Description`.
* Label your case with the importance defined in [importance.go](./ci/labels/importance.go). The importance lable is always defined on each test case.
* Label your case with the runtime defined in [runtime.go](./ci/labels/runtime.go). The runtime lable is always defined on each test case.

If we meet the case fails CI all the time and we can't fix it in time, we can label the case with ***Exclude*** defined in [category.go](./ci/labels/category.go) to exclude it from the CI list.


## Running

### Prerequisite

Please read repo's [README.md](../README.md)
For the test cases, we need `$ make install` to make the rosa command line installed to local

### Users and Tokens

1. Make local aws configuration finishing to launching the tests.
2. Please login rosacli with the token:
  * `$ rosa login --env staging --token $ROSA_USER_TOKEN`
3. Run rosa init to check all configurations are working well:
  * `$ rosa init`

### Day1 cluster preparation

1. Pick a profile for the cluster creation according to the configurations from 
  - [rosa-classic profiles](./ci/data/profiles/rosa-classic.yaml)
  - [rosa-hcp profiles](./ci/data/profiles/rosa-hcp.yaml)
  - [external team profiles](./ci/data/profiles/external.yaml)

2. Export the profile name as an environment variable
  * `$ export TEST_PROFILE=<PROFILE NAME>`
3. Export the name prefix of the cluster and resources created, no longer than 15 chars.
  * `$ export NAME_PREFIX=<your alias>`

4. Create cluster according to the profile configuration
  * `$ ginkgo run --label-filter day1 tests/e2e --timeout 2h`

5. Wait for the cluster preparation finished

> [!CAUTION]
> **The profiles with a _TODO_ is not supported yet**

> [!NOTE]
> Supported environment variables to override the profile configurations
> * **SHARED_DIR** if you have the env variable set, all output files will be put under it, otherwise it will create a dir _output/${TEST_PROFILE}_
> * **ARTIFACT_DIR** if you configured the env variable, files need to be archived will be recorded in it. Otherwise it will be recorded to dir _output/${TEST_PROFILE}_
> * **CHANNEL_GROUP** if it is set, the *channel_group* in profile configuration will be override
> * **VERSION** if it is set, the _version_ in profile configuration will be override. Supported values 
>    - _`4.15`_ it will pick the latest z-stream version in minor release of _4.15_
>    - _`latest`_ it will pick the latest version in current channel_group
>    - _`4.16.0-rc.0`_ will match the exact version set
>    - _`y-1`_ will pick a minor stream upgrade version
>    - _`z-1`_ will pick a optional stream upgrade version
> * **REGION** if it is set, the _region_ in profile configuration will be override. NOTE: rosa cluster with proxy will fail on region `us-east-1`. It's a known issue.
> * **PROVISION_SHARD** if it is set, a provision shard will be specified for cluster provision
> * **NAME_PREFIX** if it is set, all resources will be generated based with the name prefix to identify the created cluster created by you. Otherwise _`rosacli-ci`_ will be used. For local testing, we should have it be set with your alias
> * **CLUSTER_TIMEOUT** if it is set, the process will exit if cluster cannot be ready in setting time. Unit is minute
> * **USE_LOCAL_CREDENTIALS** if it is set to `true`, then when the cluster is provisioned the `--use-local-credentials` flag will be enabled

### Running a local CI test simulation

This feature allows for running tests through a case filter to simulate CI. Anyone can customize the case label filter to select the specific cases that would be run. 

1. To declare the cluster profile again, use the below variable
  * `$ export TEST_PROFILE=<PROFILE NAME>`

2. Run cases with the profile
    * Running cases based on label filter which to simulate CI jobs
      * `$ ginkgo run --label-filter '(Critical,High)&&(day1-post,day2)&&!Exclude' tests/e2e`
    * Run a specified case to debug
      * `$ ginkgo run -focus <case id> tests/e2e`

### Resources destroy

1. Export the profile name as an environment variable
* `$ export TEST_PROFILE=<PROFILE NAME>`

2. Destroy cluster and prepared user data based on the profile and the information recorded in the creation of the cluster
* `$ ginkgo -label-filter destroy tests/e2e`

3. Wait for the resources destroy finished

> [!NOTE]
> Environment variables setting
> * **SHARED_DIR** if you have the env variable setting, resource destroy will read information from cluster-detail.json and resources.json under it, otherwise it will read the two files from _output/${TEST_PROFILE}_

## Additional configuration

> [!TIP]
> Set log level
> Log level defined in rosa/tests/utils/log/logger.go
> ```golang
> Logger.logger.SetLevel()
> ```

## Running with presubmit jobs

The [presubmit jobs](https://github.com/openshift/release/blob/master/ci-operator/config/openshift/rosa/openshift-rosa-master__e2e-presubmits.yaml) are used for validating the changes in the pull request before merging. The lifecycle of a presubmit job is `create a cluster` -> `do testing` -> `release resources`. The tester needs to select the corresponding configuration to trigger the job manually. These jobs are set to `optional: true` which means even if it is failed, it is not considered as a merging blocker.

Regularly, `do testing` focuses on the test case IDs that are gotten from the commit title in the format `<card id> | test: automated cases id:123456,123457`. If there is no test case ID, it will pick up the `Critical` ones. The testing results will be recorded in a junit.xml under `rosa-test-e2e-pull-request/${ARTIFACT_DIR}`. 

### Help

Type `/test ?` to list the defined jobs.

```text
The following commands are available to trigger optional jobs:

    /test e2e-presubmits-pr-rosa-hcp
    /test e2e-presubmits-pr-rosa-hcp-byo-kms-oidc-auditlog
    /test e2e-presubmits-pr-rosa-hcp-private-proxy
    /test e2e-presubmits-pr-rosa-hcp-security-group
    /test e2e-presubmits-pr-rosa-non-sts
    /test e2e-presubmits-pr-rosa-sts
    /test e2e-presubmits-pr-rosa-sts-byo-kms-oidc-fips
    /test e2e-presubmits-pr-rosa-sts-localzone
    /test e2e-presubmits-pr-rosa-sts-private-proxy
    /test e2e-presubmits-pr-rosa-sts-security-group
    /test e2e-presubmits-pr-rosa-sts-shared-vpc-auto
```
