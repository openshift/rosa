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

package install

import (
	"fmt"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	rprtr "github.com/openshift/rosa/pkg/reporter"
	"github.com/spf13/cobra"
	"os"
	"time"
)

var Cmd = &cobra.Command{
	Use:   "install",
	Short: "Waits for install to finish",
	Long:  "Waits for install to finish",
	Run:   run,
}

var args struct {
	clusterKey   string
	pollInterval int
	maxInterval  int
}

func init() {
	flags := Cmd.Flags()

	output.AddFlag(Cmd)

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID of the cluster to describe.",
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

	Cmd.MarkFlagRequired("cluster")
}

func run(cmd *cobra.Command, argv []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	// Gets cluster key argument
	clusterKey := args.clusterKey
	pollInterval := args.pollInterval
	maxInterval := args.maxInterval

	// Check that the cluster key (name, identifier or external identifier) given by the user
	// is reasonably safe so that there is no risk of SQL injection:
	if !ocm.IsValidClusterKey(clusterKey) {
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
	n := 0
	for !done {
		// Try to find the cluster:
		reporter.Debugf("Loading cluster '%s'", clusterKey)
		cluster, err := ocmClient.GetCluster(clusterKey, awsCreator)
		if err != nil {
			reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
			os.Exit(1)
		}

		clusterState := cluster.State()
		clusterName := cluster.DisplayName()
		fmt.Fprintf(os.Stdout, "Cluster %s state is %s\n", clusterName, clusterState)

		switch cluster.State() {
		case cmv1.ClusterStateReady,
			cmv1.ClusterStateUninstalling,
			cmv1.ClusterStateResuming,
			cmv1.ClusterStatePoweringDown,
			cmv1.ClusterStateHibernating,
			cmv1.ClusterStateError,
			cmv1.ClusterStateUnknown:
			done = true
		}
		if n > maxInterval {
			done = true
		}
		if !done {
			fmt.Fprintf(os.Stdout, "Waiting (%d s/%d s) for install on cluster %s ...\n", n, maxInterval, clusterKey)
			time.Sleep(time.Duration(pollInterval) * time.Second)
			n += pollInterval
		}
	}
}
