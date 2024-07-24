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
	"os"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	mpHelpers "github.com/openshift/rosa/pkg/helper/machinepools"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/securitygroups"
	"github.com/openshift/rosa/pkg/machinepool"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/properties"
	"github.com/openshift/rosa/pkg/rosa"
)

var args machinepool.MachinePoolArgs

var Cmd = &cobra.Command{
	Use:     "machinepool",
	Aliases: []string{"machinepools", "machine-pool", "machine-pools"},
	Short:   "Add machine pool to cluster",
	Long:    "Add a machine pool to the cluster.",
	Example: `  # Interactively add a machine pool to a cluster named "mycluster"
  rosa create machinepool --cluster=mycluster --interactive

  # Add a machine pool mp-1 with 3 replicas of m5.xlarge to a cluster
  rosa create machinepool --cluster=mycluster --name=mp-1 --replicas=3 --instance-type=m5.xlarge

  # Add a machine pool mp-1 with autoscaling enabled and 3 to 6 replicas of m5.xlarge to a cluster
  rosa create machinepool --cluster=mycluster --name=mp-1 --enable-autoscaling \
	--min-replicas=3 --max-replicas=6 --instance-type=m5.xlarge

  # Add a machine pool with labels to a cluster
  rosa create machinepool -c mycluster --name=mp-1 --replicas=2 --instance-type=r5.2xlarge --labels=foo=bar,bar=baz,

  # Add a machine pool with spot instances to a cluster
  rosa create machinepool -c mycluster --name=mp-1 --replicas=2 --instance-type=r5.2xlarge --use-spot-instances \
    --spot-max-price=0.5

  # Add a machine pool to a cluster and set the node drain grace period
  rosa create machinepool -c mycluster --name=mp-1 --node-drain-grace-period="90 minutes"`,
	Run:  run,
	Args: cobra.NoArgs,
}

func init() {
	flags := Cmd.Flags()

	ocm.AddClusterFlag(Cmd)

	flags.StringVar(
		&args.Name,
		"name",
		"",
		"Name for the machine pool (required).",
	)

	flags.IntVar(
		&args.Replicas,
		"replicas",
		0,
		"Count of machines for the machine pool (required when autoscaling is disabled).",
	)

	flags.BoolVar(
		&args.AutoscalingEnabled,
		"enable-autoscaling",
		false,
		"Enable autoscaling for the machine pool.",
	)

	flags.IntVar(
		&args.MinReplicas,
		"min-replicas",
		0,
		"Minimum number of machines for the machine pool.",
	)

	flags.IntVar(
		&args.MaxReplicas,
		"max-replicas",
		0,
		"Maximum number of machines for the machine pool.",
	)

	flags.StringVar(
		&args.InstanceType,
		"instance-type",
		"m5.xlarge",
		"Instance type that should be used.",
	)

	flags.StringVar(
		&args.Labels,
		"labels",
		"",
		"Labels for machine pool. Format should be a comma-separated list of 'key=value'. "+
			"This list will overwrite any modifications made to Node labels on an ongoing basis.",
	)

	flags.StringVar(
		&args.Taints,
		"taints",
		"",
		"Taints for machine pool. Format should be a comma-separated list of 'key=value:ScheduleType'. "+
			"This list will overwrite any modifications made to Node taints on an ongoing basis.",
	)

	flags.BoolVar(
		&args.UseSpotInstances,
		"use-spot-instances",
		false,
		"Use spot instances for the machine pool.",
	)

	flags.StringVar(
		&args.SpotMaxPrice,
		"spot-max-price",
		"on-demand",
		"Max price for spot instance. If empty use the on-demand price.",
	)

	flags.BoolVar(
		&args.MultiAvailabilityZone,
		"multi-availability-zone",
		true,
		"Create a multi-AZ machine pool for a multi-AZ cluster")

	flags.StringVar(
		&args.AvailabilityZone,
		"availability-zone",
		"",
		"Select availability zone to create a single AZ machine pool for a multi-AZ cluster")

	flags.StringVar(
		&args.Subnet,
		"subnet",
		"",
		"Select subnet to create a single AZ machine pool for BYOVPC cluster")

	flags.StringVar(
		&args.Version,
		"version",
		"",
		"Version of OpenShift that will be used to install a machine pool for a hosted cluster,"+
			" for example \"4.12.4\"",
	)

	flags.BoolVar(
		&args.Autorepair,
		"autorepair",
		true,
		"Select auto-repair behaviour for a machinepool in a hosted cluster.",
	)

	flags.StringVar(
		&args.TuningConfigs,
		"tuning-configs",
		"",
		"Name of the tuning configs to be applied to the machine pool. Format should be a comma-separated list. "+
			"Tuning config must already exist. "+
			"This list will overwrite any modifications made to node tuning configs on an ongoing basis.",
	)

	flags.StringVar(
		&args.KubeletConfigs,
		"kubelet-configs",
		"",
		"Name of the kubelet config to be applied to the machine pool. A single kubelet config is allowed. "+
			"Kubelet config must already exist. "+
			"This will overwrite any modifications made to node kubelet configs on an ongoing basis.",
	)

	flags.StringVar(&args.RootDiskSize,
		"disk-size",
		"",
		"Root disk size with a suffix like GiB or TiB",
	)

	flags.StringSliceVar(&args.SecurityGroupIds,
		securitygroups.MachinePoolSecurityGroupFlag,
		nil,
		"The additional Security Group IDs to be added to the machine pool. "+
			"Format should be a comma-separated list.",
	)

	flags.StringVar(&args.NodeDrainGracePeriod,
		"node-drain-grace-period",
		"",
		"You may set a grace period for how long Pod Disruption Budget-protected workloads will be "+
			"respected when the NodePool is being replaced or upgraded.\nAfter this grace period, all remaining workloads "+
			"will be forcibly evicted.\n"+
			"Valid value is from 0 to 1 week (10080 minutes), and the supported units are 'minute|minutes' or "+
			"'hour|hours'. 0 or empty value means that the NodePool can be drained without any time limitations.\n"+
			"This flag is only supported for Hosted Control Planes.",
	)

	flags.StringSliceVar(
		&args.Tags,
		"tags",
		nil,
		"Apply user defined tags to all resources created by ROSA in AWS. "+
			"Tags are comma separated, for example: 'key value, foo bar'",
	)

	flags.StringVar(
		&args.EC2MetadataHttpTokens,
		"ec2-metadata-http-tokens",
		"",
		"Should cluster nodes use both v1 and v2 endpoints or just v2 endpoint "+
			"of EC2 Instance Metadata Service (IMDS)"+
			"This flag is only supported for Hosted Control Planes.",
	)

	flags.StringVar(&args.MaxSurge,
		"max-surge",
		"1",
		"The maximum number of nodes that can be provisioned above the desired number of nodes in the machinepool during "+
			"the upgrade. It can be an absolute number i.e. 1, or a percentage i.e. '20%'.",
	)

	flags.StringVar(&args.MaxUnavailable,
		"max-unavailable",
		"0",
		"The maximum number of nodes in the machinepool that can be unavailable during the upgrade. It can be an "+
			"absolute number i.e. 1, or a percentage i.e. '20%'.",
	)

	interactive.AddFlag(flags)
	output.AddFlag(Cmd)
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithOCM()
	defer r.Cleanup()

	clusterKey := r.GetClusterKey()

	cluster := r.FetchCluster()
	if cluster.State() != cmv1.ClusterStateReady {
		r.Reporter.Errorf("Cluster '%s' is not yet ready", clusterKey)
		os.Exit(1)
	}

	val, ok := cluster.Properties()[properties.UseLocalCredentials]
	useLocalCredentials := ok && val == "true"

	if cmd.Flags().Changed("labels") {
		_, err := mpHelpers.ParseLabels(args.Labels)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
	}

	// Initiate the AWS client with the cluster's region
	var err error
	r.AWSClient, err = aws.NewClient().
		Region(cluster.Region().ID()).
		Logger(r.Logger).
		UseLocalCredentials(useLocalCredentials).
		Build()
	if err != nil {
		r.Reporter.Errorf("Failed to create awsClient: %s", err)
		os.Exit(1)
	}

	service := machinepool.NewMachinePoolService()
	if cluster.Hypershift().Enabled() {
		err = service.AddNodePool(cmd, clusterKey, cluster, r, &args)
	} else {
		err = service.AddMachinePool(cmd, clusterKey, cluster, r, &args)
	}
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}
}
