# ROSA CLI Function Verification Testing
This package is the automation package for Function Verification Testing on the ROSA CLI. 

## Structure of tests
```sh
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
|       
|____prow_ci.sh
```

## Contibute to ROSA CLI tests

Please read the structure and contribute code to the correct place

### Contribute to day2

* Create the case in rosa/tests/e2e/<feature name>_test.go
* Label the case with ***CI.Day2***
* Label the case with importance ***CI.Critical*** or ***CI.High***
* Don't need to run creation step, just in BeforeEach step call function  ***config.GetClusterID()*** it will load the clusterID prepared from env or cluster_id file
* Code for day2 actions and check step
* Every case need to recover the cluster after the case run finished unless it's un-recoverable

### Labels

* Label your case with the ***CI.Feature<feature name>*** defined in rosa/tests/ci/labels/features.go
* Label your case with importance defined in rosa/tests/ci/labels/importance.go
* Label your case with ***CI.Day1Post/CI.Day2/CI.Day3*** defined in rosa/tests/ci/labels/runtime.go, according to the case runtime
* Label your case with ***CI.Exclude*** if it fails CI all  the time and you can't fix it in time
* Label you case with ***CI.NonClassicCluster/CI.NonHCPCluster*** if it does not fit a type of cluster

## Running

### Prerequisite

Please read repo's [README.md](../README.md)
For the test cases, we need `make install` to make the rosa command line installed to local

#### Users and Tokens

Please login ocm and aws cli prior to launching the tests.

#### Global variables

To declare the cluster id, use the below variable::
* export CLUSTER_ID = <cluster_id>

### Running a local CI simulation

This feature allows for running tests through a case filter to simulate CI. Anyone can customize the case label filter to select the specific cases that would be run. 

* Run day2 or day1-post cases with profile
  * Run ginkgo run command
    * `ginkgo run --label-filter '(Critical,High)&&(day1-post,day2)&&!Exclude' tests/e2e`
* Run a specified case to debug
  * `ginkgo -focus <case id> tests/e2e`

### Set log level

* Log level defined in rosa/tests/utils/log/logger.go

```golang
Logger.logger.SetLevel()
```
