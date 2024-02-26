/*
Copyright (c) 2023 Red Hat, Inc.

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

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "machinepool",
	Aliases: []string{"machine-pool"},
	Short:   "Show details of a machine pool on a cluster",
	Long:    "Show details of a machine pool on a cluster.",
	Example: `  # Show details of a machine pool named "mymachinepool"" on a cluster named "mycluster"
  rosa describe machinepool --cluster=mycluster --machinepool=mymachinepool`,
	Run:  run,
	Args: cobra.MaximumNArgs(1),
}

var args struct {
	machinePool string
}

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false
	ocm.AddClusterFlag(Cmd)
	output.AddFlag(Cmd)
	flags.StringVar(
		&args.machinePool,
		"machinepool",
		"",
		"Machine pool of the cluster to target",
	)
}

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()
	err := runWithRuntime(r, cmd, argv)
	if err != nil {
		r.Reporter.Errorf(err.Error())
		os.Exit(1)
	}
}

func runWithRuntime(r *rosa.Runtime, cmd *cobra.Command, argv []string) error {
	machinePool := args.machinePool
	// Allow the use also directly the machine pool id as positional parameter
	if len(argv) == 1 && !cmd.Flag("machinepool").Changed {
		machinePool = argv[0]
	}
	if machinePool == "" {
		return fmt.Errorf("You need to specify a machine pool name")
	}
	clusterKey := r.GetClusterKey()
	cluster := r.FetchCluster()
	if cluster.State() != cmv1.ClusterStateReady {
		r.Reporter.Errorf("Cluster '%s' is not yet ready", clusterKey)
		os.Exit(1)
	}
	isHypershift := cluster.Hypershift().Enabled()

	if isHypershift {
		return describeNodePool(r, cluster, clusterKey, machinePool)
	} else {
		return describeMachinePool(r, cluster, clusterKey, machinePool)
	}
}
