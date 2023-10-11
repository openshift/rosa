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
	"regexp"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/properties"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
)

// Regular expression to used to make sure that the identifier given by the
// user is safe and that it there is no risk of SQL injection:
var machinePoolKeyRE = regexp.MustCompile(`^[a-z]([-a-z0-9]*[a-z0-9])?$`)

var args struct {
	name                  string
	instanceType          string
	replicas              int
	autoscalingEnabled    bool
	minReplicas           int
	maxReplicas           int
	labels                string
	taints                string
	useSpotInstances      bool
	spotMaxPrice          string
	multiAvailabilityZone bool
	availabilityZone      string
	subnet                string
	version               string
	autorepair            bool
	tuningConfigs         string
	rootDiskSize          string
	securityGroupIds      []string
}

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
    --spot-max-price=0.5`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()

	ocm.AddClusterFlag(Cmd)

	flags.StringVar(
		&args.name,
		"name",
		"",
		"Name for the machine pool (required).",
	)

	flags.IntVar(
		&args.replicas,
		"replicas",
		0,
		"Count of machines for the machine pool (required when autoscaling is disabled).",
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
		&args.instanceType,
		"instance-type",
		"m5.xlarge",
		"Instance type that should be used.",
	)

	flags.StringVar(
		&args.labels,
		"labels",
		"",
		"Labels for machine pool. Format should be a comma-separated list of 'key=value'. "+
			"This list will overwrite any modifications made to Node labels on an ongoing basis.",
	)

	flags.StringVar(
		&args.taints,
		"taints",
		"",
		"Taints for machine pool. Format should be a comma-separated list of 'key=value:ScheduleType'. "+
			"This list will overwrite any modifications made to Node taints on an ongoing basis.",
	)

	flags.BoolVar(
		&args.useSpotInstances,
		"use-spot-instances",
		false,
		"Use spot instances for the machine pool.",
	)

	flags.StringVar(
		&args.spotMaxPrice,
		"spot-max-price",
		"on-demand",
		"Max price for spot instance. If empty use the on-demand price.",
	)

	flags.BoolVar(
		&args.multiAvailabilityZone,
		"multi-availability-zone",
		true,
		"Create a multi-AZ machine pool for a multi-AZ cluster")

	flags.StringVar(
		&args.availabilityZone,
		"availability-zone",
		"",
		"Select availability zone to create a single AZ machine pool for a multi-AZ cluster")

	flags.StringVar(
		&args.subnet,
		"subnet",
		"",
		"Select subnet to create a single AZ machine pool for BYOVPC cluster")

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

	flags.StringVar(&args.rootDiskSize,
		"disk-size",
		"",
		"Root disk size with a suffix like GiB or TiB",
	)

	flags.StringSliceVar(&args.securityGroupIds,
		securityGroupIdsFlag,
		nil,
		"The additional Security Group IDs to be added to the machine pool. "+
			"Format should be a comma-separated list.",
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

	if cluster.Hypershift().Enabled() {
		addNodePool(cmd, clusterKey, cluster, r)
	} else {
		addMachinePool(cmd, clusterKey, cluster, r)
	}
}
