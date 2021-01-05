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
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	c "github.com/openshift/rosa/pkg/cluster"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

// Regular expression to used to make sure that the identifier given by the
// user is safe and that it there is no risk of SQL injection:
var machinePoolKeyRE = regexp.MustCompile(`^[a-z]([-a-z0-9]*[a-z0-9])?$`)

var args struct {
	clusterKey         string
	replicas           int
	autoscalingEnabled bool
	minReplicas        int
	maxReplicas        int
}

var Cmd = &cobra.Command{
	Use:     "machinepool",
	Aliases: []string{"machinepools", "machine-pool", "machine-pools"},
	Short:   "Edit machine pool",
	Long:    "Edit the additional machine pool from a cluster.",
	Example: `  # Set 4 replicas on machine pool 'mp1' on cluster 'mycluster'
  rosa edit machinepool --replicas=4 --cluster=mycluster mp1`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID of the cluster to add the machine pool to (required).",
	)
	Cmd.MarkFlagRequired("cluster")

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
}

func run(cmd *cobra.Command, argv []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	// Check command line arguments:
	if len(argv) != 1 {
		reporter.Errorf(
			"Expected exactly one command line parameter containing the id of the machine pool",
		)
		os.Exit(1)
	}

	machinePoolID := argv[0]
	if !machinePoolKeyRE.MatchString(machinePoolID) {
		reporter.Errorf("Expected a valid identifier for the machine pool")
		os.Exit(1)
	}

	// Check that the cluster key (name, identifier or external identifier) given by the user
	// is reasonably safe so that there is no risk of SQL injection:
	clusterKey := args.clusterKey
	if !c.IsValidClusterKey(clusterKey) {
		reporter.Errorf(
			"Cluster name, identifier or external identifier '%s' isn't valid: it "+
				"must contain only letters, digits, dashes and underscores",
			clusterKey,
		)
		os.Exit(1)
	}

	// Create the AWS client:
	var err error
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
	ocmConnection, err := ocm.NewConnection().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create OCM connection: %v", err)
		os.Exit(1)
	}
	defer func() {
		err = ocmConnection.Close()
		if err != nil {
			reporter.Errorf("Failed to close OCM connection: %v", err)
		}
	}()

	// Get the client for the OCM collection of clusters:
	clustersCollection := ocmConnection.ClustersMgmt().V1().Clusters()

	// Try to find the cluster:
	reporter.Debugf("Loading cluster '%s'", clusterKey)
	cluster, err := ocm.GetCluster(clustersCollection, clusterKey, awsCreator.ARN)
	if err != nil {
		reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	// Editing the default machine pool is a different process
	if machinePoolID == "default" {
		autoscaling, replicas, minReplicas, maxReplicas := getReplicas(cmd, reporter, machinePoolID,
			cluster.Nodes().AutoscaleCompute())

		if !autoscaling && replicas < 2 ||
			(autoscaling && cmd.Flags().Changed("min-replicas") && minReplicas < 2) {
			reporter.Errorf("Default machine pool requires at least 2 compute nodes")
			os.Exit(1)
		}
		if cluster.MultiAZ() &&
			(!autoscaling && replicas%3 != 0 ||
				(autoscaling && (minReplicas%3 != 0 || maxReplicas%3 != 0))) {
			reporter.Errorf("Multi AZ clusters require that the number of compute nodes be a multiple of 3")
			os.Exit(1)
		}

		clusterConfig := c.Spec{
			Autoscaling:  autoscaling,
			ComputeNodes: replicas,
			MinReplicas:  minReplicas,
			MaxReplicas:  maxReplicas,
		}

		reporter.Debugf("Updating machine pool '%s' on cluster '%s'", machinePoolID, clusterKey)
		err = c.UpdateCluster(clustersCollection, clusterKey, awsCreator.ARN, clusterConfig)
		if err != nil {
			reporter.Errorf("Failed to update machine pool '%s' on cluster '%s': %s",
				machinePoolID, clusterKey, err)
			os.Exit(1)
		}

		os.Exit(0)
	}

	// Try to find the machine pool:
	reporter.Debugf("Loading machine pools for cluster '%s'", clusterKey)
	machinePools, err := ocm.GetMachinePools(clustersCollection, cluster.ID())
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

	autoscaling, replicas, minReplicas, maxReplicas := getReplicas(cmd, reporter, machinePoolID,
		machinePool.Autoscaling())

	if !autoscaling && replicas < 0 ||
		(autoscaling && cmd.Flags().Changed("min-replicas") && minReplicas < 1) {
		reporter.Errorf("The number of machine pool replicas needs to be a positive integer")
		os.Exit(1)
	}

	if cluster.MultiAZ() &&
		(!autoscaling && replicas%3 != 0 ||
			(autoscaling && (minReplicas%3 != 0 || maxReplicas%3 != 0))) {
		reporter.Errorf("Multi AZ clusters require that the number of MachinePool replicas be a multiple of 3")
		os.Exit(1)
	}

	mpBuilder := cmv1.NewMachinePool().
		ID(machinePool.ID())

	if autoscaling {
		asBuilder := cmv1.NewMachinePoolAutoscaling()

		if minReplicas > 0 {
			asBuilder = asBuilder.MinReplicas(minReplicas)
		}
		if maxReplicas > 0 {
			asBuilder = asBuilder.MaxReplicas(maxReplicas)
		}

		mpBuilder = mpBuilder.Autoscaling(asBuilder)
	} else {
		mpBuilder = mpBuilder.Replicas(replicas)
	}

	machinePool, err = mpBuilder.Build()
	if err != nil {
		reporter.Errorf("Failed to create machine pool for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	reporter.Debugf("Updating machine pool '%s' on cluster '%s'", machinePool.ID(), clusterKey)
	res, err := clustersCollection.
		Cluster(cluster.ID()).
		MachinePools().
		MachinePool(machinePool.ID()).
		Update().
		Body(machinePool).
		Send()
	if err != nil {
		reporter.Debugf(err.Error())
		reporter.Errorf("Failed to update machine pool '%s' on cluster '%s': %s",
			machinePool.ID(), clusterKey, res.Error().Reason())
		os.Exit(1)
	}
}

func getReplicas(cmd *cobra.Command,
	reporter *rprtr.Object,
	machinePoolID string,
	existingAutoscaling *cmv1.MachinePoolAutoscaling) (autoscaling bool,
	replicas, minReplicas, maxReplicas int) {

	var err error
	isMinReplicasSet := cmd.Flags().Changed("min-replicas")
	isMaxReplicasSet := cmd.Flags().Changed("max-replicas")
	isReplicasSet := cmd.Flags().Changed("replicas")
	isAutoscalingSet := cmd.Flags().Changed("enable-autoscaling")

	replicas = args.replicas
	minReplicas = args.minReplicas
	maxReplicas = args.maxReplicas
	autoscaling = args.autoscalingEnabled

	// if the user set min/max replicas and hasn't enabled autoscaling, or existing is disabled
	if (isMinReplicasSet || isMaxReplicasSet) && !autoscaling && existingAutoscaling == nil {
		reporter.Errorf("Autoscaling is not enabled on machine pool '%s'. can't set min or max replicas",
			machinePoolID)
		os.Exit(1)
	}

	// if the user set replicas but enabled autoscaling or hasn't disabled existing autoscaling
	if isReplicasSet && existingAutoscaling != nil && (!isAutoscalingSet || autoscaling) {
		reporter.Errorf("Autoscaling enabled on machine pool '%s'. can't set replicas",
			machinePoolID)
		os.Exit(1)
	}

	if !isAutoscalingSet {
		autoscaling = existingAutoscaling != nil
		if interactive.Enabled() {
			autoscaling, err = interactive.GetBool(interactive.Input{
				Question: "Enable autoscaling",
				Help:     cmd.Flags().Lookup("enable-autoscaling").Usage,
				Default:  autoscaling,
				Required: false,
			})
			if err != nil {
				reporter.Errorf("Expected a valid value for enable-autoscaling: %s", err)
				os.Exit(1)
			}
		}
	}

	if autoscaling {
		// Prompt for min replicas if neither min or max is set or interactive mode
		if !isMinReplicasSet && (interactive.Enabled() || !isMaxReplicasSet) {
			minReplicas, err = interactive.GetInt(interactive.Input{
				Question: "Min replicas",
				Help:     cmd.Flags().Lookup("min-replicas").Usage,
				Default:  existingAutoscaling.MinReplicas(),
				Required: false,
			})
			if err != nil {
				reporter.Errorf("Expected a valid number of min replicas: %s", err)
				os.Exit(1)
			}
		}

		// Prompt for max replicas if neither min or max is set or interactive mode
		if !isMaxReplicasSet && (interactive.Enabled() || !isMinReplicasSet) {
			maxReplicas, err = interactive.GetInt(interactive.Input{
				Question: "Max replicas",
				Help:     cmd.Flags().Lookup("max-replicas").Usage,
				Default:  existingAutoscaling.MaxReplicas(),
				Required: false,
			})
			if err != nil {
				reporter.Errorf("Expected a valid number of max replicas: %s", err)
				os.Exit(1)
			}
		}
	} else if interactive.Enabled() || !isReplicasSet {
		replicas, err = interactive.GetInt(interactive.Input{
			Question: "Replicas",
			Help:     cmd.Flags().Lookup("replicas").Usage,
			Default:  replicas,
			Required: true,
		})
		if err != nil {
			reporter.Errorf("Expected a valid number of replicas: %s", err)
			os.Exit(1)
		}
	}
	return
}
