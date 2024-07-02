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

	"github.com/spf13/cobra"

	mpHelpers "github.com/openshift/rosa/pkg/helper/machinepools"
	"github.com/openshift/rosa/pkg/machinepool"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	replicas             int
	autoscalingEnabled   bool
	minReplicas          int
	maxReplicas          int
	labels               string
	taints               string
	autorepair           bool
	tuningConfigs        string
	kubeletConfigs       string
	nodeDrainGracePeriod string
	maxSurge             string
	maxUnavailable       string
}

var Cmd = &cobra.Command{
	Use:     "machinepool ID",
	Aliases: []string{"machinepools", "machine-pool", "machine-pools"},
	Short:   "Edit machine pool",
	Long:    "Edit machine pools on a cluster.",
	Example: `  # Set 4 replicas on machine pool 'mp1' on cluster 'mycluster'
  rosa edit machinepool --replicas=4 --cluster=mycluster mp1
  # Enable autoscaling and Set 3-5 replicas on machine pool 'mp1' on cluster 'mycluster'
  rosa edit machinepool --enable-autoscaling --min-replicas=3 --max-replicas=5 --cluster=mycluster mp1
  # Set the node drain grace period to 1 hour on machine pool 'mp1' on cluster 'mycluster'
  rosa edit machinepool --node-drain-grace-period="1 hour" --cluster=mycluster mp1`,
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

	flags.StringVar(
		&args.kubeletConfigs,
		"kubelet-configs",
		"",
		"Name of the kubelet config to be applied to the machine pool.  A single kubelet config is allowed. "+
			"Kubelet config must already exist. "+
			"This will overwrite any modifications made to node kubelet configs on an ongoing basis.",
	)

	flags.StringVar(&args.nodeDrainGracePeriod,
		"node-drain-grace-period",
		"",
		"You may set a grace period for how long Pod Disruption Budget-protected workloads will be "+
			"respected when the NodePool is being replaced or upgraded.\nAfter this grace period, all remaining workloads "+
			"will be forcibly evicted.\n"+
			"Valid value is from 0 to 1 week (10080 minutes), and the supported units are 'minute|minutes' or "+
			"'hour|hours'. 0 or empty value means that the NodePool can be drained without any time limitations.\n"+
			"This flag is only supported for Hosted Control Planes.",
	)

	flags.StringVar(&args.maxSurge,
		"max-surge",
		"",
		"The maximum number of nodes that can be provisioned above the desired number of nodes in the machinepool during "+
			"the upgrade. It can be an absolute number i.e. 1, or a percentage i.e. '20%'.",
	)

	flags.StringVar(&args.maxUnavailable,
		"max-unavailable",
		"",
		"The maximum number of nodes in the machinepool that can be unavailable during the upgrade. It can be an "+
			"absolute number i.e. 1, or a percentage i.e. '20%'.",
	)
}

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	machinePoolID := argv[0]
	clusterKey := r.GetClusterKey()
	cluster := r.FetchCluster()

	if cmd.Flags().Changed("labels") {
		_, err := mpHelpers.ParseLabels(args.labels)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
	}

	service := machinepool.NewMachinePoolService()
	err := service.EditMachinePool(cmd, machinePoolID, clusterKey, cluster, r)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}
}
