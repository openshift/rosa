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

package machinepool

import (
	"fmt"
	"os"
	"regexp"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

// Regular expression to used to make sure that the identifier given by the
// user is safe and that it there is no risk of SQL injection:
var machinePoolKeyRE = regexp.MustCompile(`^[a-z]([-a-z0-9]*[a-z0-9])?$`)

var Cmd = &cobra.Command{
	Use:     "machinepool ID",
	Aliases: []string{"machinepools", "machine-pool", "machine-pools"},
	Short:   "Delete machine pool",
	Long:    "Delete the additional machine pool from a cluster.",
	Example: `  # Delete machine pool with ID mp-1 from a cluster named 'mycluster'
  rosa delete machinepool --cluster=mycluster mp-1`,
	Run: run,
	Args: func(_ *cobra.Command, argv []string) error {
		if len(argv) != 1 {
			return fmt.Errorf(
				"Expected exactly one command line parameter containing the id of the machine pool",
			)
		}
		return nil
	},
}

func init() {
	ocm.AddClusterFlag(Cmd)
}

func run(_ *cobra.Command, argv []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.NewLogger()

	machinePoolID := argv[0]
	if machinePoolID != "Default" && !machinePoolKeyRE.MatchString(machinePoolID) {
		reporter.Errorf("Expected a valid identifier for the machine pool")
		os.Exit(1)
	}

	clusterKey, err := ocm.GetClusterKey()
	if err != nil {
		reporter.Errorf("%s", err)
		os.Exit(1)
	}

	if machinePoolID == "Default" {
		reporter.Errorf("Machine pool '%s' cannot be deleted from cluster '%s'", machinePoolID, clusterKey)
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
	defer func() {
		err = ocmClient.Close()
		if err != nil {
			reporter.Errorf("Failed to close OCM connection: %v", err)
		}
	}()

	// Try to find the cluster:
	reporter.Debugf("Loading cluster '%s'", clusterKey)
	cluster, err := ocmClient.GetCluster(clusterKey, awsCreator)
	if err != nil {
		reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	// Try to find the machine pool:
	reporter.Debugf("Loading machine pools for cluster '%s'", clusterKey)
	machinePools, err := ocmClient.GetMachinePools(cluster.ID())
	if err != nil {
		reporter.Errorf("Failed to get machine pools for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	var machinePool *cmv1.MachinePool
	for _, item := range machinePools {
		if item.ID() == machinePoolID {
			machinePool = item
		}
	}
	if machinePool == nil {
		reporter.Errorf("Failed to get machine pool '%s' for cluster '%s'", machinePoolID, clusterKey)
		os.Exit(1)
	}

	if confirm.Confirm("delete machine pool '%s' on cluster '%s'", machinePoolID, clusterKey) {
		reporter.Debugf("Deleting machine pool '%s' on cluster '%s'", machinePool.ID(), clusterKey)
		err = ocmClient.DeleteMachinePool(cluster.ID(), machinePool.ID())
		if err != nil {
			reporter.Errorf("Failed to delete machine pool '%s' on cluster '%s': %s",
				machinePool.ID(), clusterKey, err)
			os.Exit(1)
		}
		reporter.Infof("Successfully deleted machine pool '%s' from cluster '%s'", machinePoolID, clusterKey)
	}
}
