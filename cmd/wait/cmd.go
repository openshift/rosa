/*
Copyright (c) 2020 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package wait

import (
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	rprtr "github.com/openshift/rosa/pkg/reporter"
	"github.com/spf13/cobra"
	"os"
	"strings"
	"time"
)

var Cmd = &cobra.Command{
	Use:   "wait-for -c cluster -s ready -s error",
	Short: "Waits for desired cluster states",
	Long:  "Waits for desired cluster states, useful for CI/CD automation",
	Run:   run,
}

var args struct {
	clusterKey   string
	targetStates []string
	pollInterval int
	maxInterval  int
}

func init() {
	flags := Cmd.Flags()
	output.AddFlag(Cmd)
	ocm.AddClusterFlag(Cmd)

	flags.StringSliceVarP(
		&args.targetStates,
		"targetState",
		"s",
		[]string{"ready", "error"},
		"States to wait for",
	)

	flags.IntVarP(
		&args.pollInterval,
		"interval",
		"i",
		60,
		"Polling interval in seconds.",
	)

	flags.IntVarP(
		&args.maxInterval,
		"max",
		"m",
		14400,
		"Max interval in seconds",
	)
}

func run(cmd *cobra.Command, argv []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	clusterKey, err := ocm.GetClusterKey()
	pollInterval := args.pollInterval
	maxInterval := args.maxInterval
	targetStates := args.targetStates
	targetStatesStr := strings.Join(targetStates, ", ")

	// Check that the cluster key (name, identifier or external identifier) given by the user
	// is reasonably safe so that there is no risk of SQL injection:
	if err != nil {
		reporter.Errorf(
			"Cluster name, identifier or external identifier '%s' isn't valid: it "+
				"must contain only letters, digits, dashes and underscores",
			clusterKey,
		)
		os.Exit(1)
	}

	// Create the AWS client:
	awsClient, err := aws.NewClient().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create AWS client: %v", err)
		os.Exit(1)
	}

	awsCreator, err := awsClient.GetCreator()
	if err != nil {
		reporter.Errorf("Failed to get AWS creator: %v", err)
		os.Exit(1)
	}

	// Create the client for the OCM API:
	ocmClient, err := ocm.NewClient().
		Logger(logger).
		Build()

	if err != nil {
		reporter.Errorf("Failed to create OCM connection: %v", err)
		os.Exit(1)
	}

	done := false
	elapsed := 0
	for !done {
		// Try to find the cluster:
		reporter.Debugf("Loading cluster '%s'", clusterKey)
		cluster, err := ocmClient.GetCluster(clusterKey, awsCreator)
		if err != nil {
			reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
			os.Exit(1)
		}

		clusterState := cluster.State()
		stateName := string(clusterState)
		clusterName := cluster.DisplayName()
		reporter.Infof("Cluster %s state is %s", clusterName, stateName)

		done = elapsed > maxInterval || contains(targetStates, stateName)

		if clusterState == cmv1.ClusterStateError {
			reporter.Errorf("Exiting with cluster on error state.")
			os.Exit(1)
		}

		if !done {
			reporter.Infof("Waiting for state [%s] on cluster %s. (%d/%d s)",
				targetStatesStr, clusterKey, elapsed, maxInterval)
			time.Sleep(time.Duration(pollInterval) * time.Second)
			elapsed += pollInterval
		}
	}
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
