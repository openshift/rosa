# ROSA CLI Automation labels
The labels are used to automate ROSA CLI and select the test cases based on the different test profiles. These labels claim what/how/where the testing should be run on CI.

## Runtime
The runtime labels define the execution strategy of the test cases and are always defined on each test case.

#### Presubmit
* labels.E2ECommit: It is used by the image step to grab the ID of the modified cases.

#### Test
The labels here are used for the test cases which are related to the cluster created by [profiles](../data/profiles).

_Pre_
* labels.runtime.Day1: Prepare cluster according to the profiles

_Test_
* labels.runtime.Day1Post: Test cases that configure/validate the cluster to be a ready, healthy cluster. 
* labels.runtime.Day2: Test cases that are related to the operations on the cluster.
* labels.runtime.Upgrade: Test cases that are for  the cluster upgrade.
* labels.runtime.Destructive: Test cases that are related to the operations which introduce the destructive impact on the cluster, e.g. delete the default machinepool on the classic cluster.

_Post_
* labels.runtime.Destroy: Test cases that destroy the cluster.
* labels.runtime.DestroyPost: Test cases that destroy/validate the cluster resources are released.

#### Supplemental
The labels here are used for the test cases which are beyond the cluster created by [profiles](../data/profiles).

* labels.runtime.Day1Supplemental:Test cases that create the cluster that is not supported by the profiles, the cluster will be deleted in a case.
* labels.runtime.OCMResources: Test cases that are around the OCM resources which do not need to prepare a cluster. These test cases are not run with the cluster test profiles but an individual test profile. The OCM resources are supplemental to use rosa CLI, the resources are:
    * Help
    * Config
    * Init
    * Download
    * Whoami
    * Token
    * Gates
    * Instance-types
    * Regions
    * Versions
    * Roles
    * Items under `rosa verify`(except network)
* labels.runtime.Day1Negative: Test cases that validate the preflight check of the cluster creation.

#### Report
* labels.runtime.E2EReport: It is used by the report step to rebuild the Junit file to fit for the report portal format.

## Importance
It defines the importance level of the test case in the customer usage and is always defined on each test case.

* labels.Critical: Test cases that are core checkpoints on the features.
* labels.High: Test cases that are important checkpoints on the features.
* labels.Medium: Test cases that are general checkpoints on the features.
* labels.Low: Test cases that are secondary checkpoints on the features.


## Category
It defines which kind of test profiles the test case could run on.

* labels.Exclude: Test cases with the label will be removed from the CI list.


## Feature
It defines which group the test case belongs to. The feature label is always defined at the top `Description`. Usually, the major subcommands in rosa CLI are considered as features. The feature labels are the items from `rosa create -h` -> `rosa list -h` -> `rosa -h`.

* labels.Feature.AccountRoles
* labels.Feature.Addon
* labels.Feature.Autoscaler
* labels.Feature.Cluster
* labels.Feature.BreakGlassCredential
* labels.Feature.ExternalAuthProvider
* labels.Feature.Gates
* labels.Feature.IDP
* labels.Feature.Ingress
* labels.Feature.InstanceTypes
* labels.Feature.KubeletConfig
* labels.Feature.Machinepool
* labels.Feature.OCMRole
* labels.Feature.OIDCConfig
* labels.Feature.OIDCProvider
* labels.Feature.OperatorRoles
* labels.Feature.Regions
* labels.Feature.Token
* labels.Feature.TuningConfigs
* labels.Feature.UserRole
* labels.Feature.VerifyResources
* labels.Feature.Version
* labels.Feature.ZeroEgress
