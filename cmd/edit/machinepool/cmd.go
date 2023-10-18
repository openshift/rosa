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

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	replicas           int
	autoscalingEnabled bool
	minReplicas        int
	maxReplicas        int
	labels             string
	taints             string
	version            string
	autorepair         bool
	tuningConfigs      string
}

var Cmd = &cobra.Command{
	Use:     "machinepool ID",
	Aliases: []string{"machinepools", "machine-pool", "machine-pools"},
	Short:   "Edit machine pool",
	Long:    "Edit machine pools on a cluster.",
	Example: `  # Set 4 replicas on machine pool 'mp1' on cluster 'mycluster'
  rosa edit machinepool --replicas=4 --cluster=mycluster mp1
  # Enable autoscaling and Set 3-5 replicas on machine pool 'mp1' on cluster 'mycluster'
  rosa edit machinepool --enable-autoscaling --min-replicas=3 --max-replicas=5 --cluster=mycluster mp1`,
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
	flags := Cmd.Flags()

	ocm.AddClusterFlag(Cmd)

	flags.IntVar(
		&args.replicas,
		"replicas",
		0,
		"Count of machines for this machine pool.",
	)

	flags.BoolVar(
		&args.autoscalingEnabled,
		"enable-autoscaling",
		false,
		"Enable autoscaling for the machine pool.",
	)

	flags.IntVar(
		&args.minReplicas,
		"min-replicas",
		0,
		"Minimum number of machines for the machine pool.",
	)

	flags.IntVar(
		&args.maxReplicas,
		"max-replicas",
		0,
		"Maximum number of machines for the machine pool.",
	)

	flags.StringVar(
		&args.labels,
		"labels",
		"",
		"Labels for machine pool. Format should be a comma-separated list of 'key=value'. "+
			"This list will overwrite any modifications made to node labels on an ongoing basis.",
	)

	flags.StringVar(
		&args.taints,
		"taints",
		"",
		"Taints for machine pool. Format should be a comma-separated list of 'key=value:ScheduleType'. "+
			"This list will overwrite any modifications made to node taints on an ongoing basis.",
	)

	flags.StringVar(
		&args.version,
		"version",
		"",
		"Version of OpenShift that will be used to install a machine pool for a hosted cluster,"+
			" for example \"4.12.4\"",
	)

	flags.BoolVar(
		&args.autorepair,
		"autorepair",
		true,
		"Select auto-repair behaviour for a machinepool in a hosted cluster.",
	)

	flags.StringVar(
		&args.tuningConfigs,
		"tuning-configs",
		"",
		"Name of the tuning configs to be applied to the machine pool. Format should be a comma-separated list. "+
			"Tuning config must already exist. "+
			"This list will overwrite any modifications made to node tuning configs on an ongoing basis.",
	)

	flags.MarkHidden("version")
}

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	machinePoolID := argv[0]
	clusterKey := r.GetClusterKey()
	cluster := r.FetchCluster()

	if cluster.Hypershift().Enabled() {
		editNodePool(cmd, machinePoolID, clusterKey, cluster, r)
	} else {
		editMachinePool(cmd, machinePoolID, clusterKey, cluster, r)
	}
}
