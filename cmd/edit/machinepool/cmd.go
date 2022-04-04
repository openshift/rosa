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
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

// Regular expression to used to make sure that the identifier given by the
// user is safe and that it there is no risk of SQL injection:
var machinePoolKeyRE = regexp.MustCompile(`^[a-z]([-a-z0-9]*[a-z0-9])?$`)

var args struct {
	replicas           int
	autoscalingEnabled bool
	minReplicas        int
	maxReplicas        int
	labels             string
	taints             string
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
}

func run(cmd *cobra.Command, argv []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

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
	cluster, err := ocmClient.GetCluster(clusterKey, awsCreator.AccountID)
	if err != nil {
		reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	// Editing the default machine pool is a different process
	if machinePoolID == "Default" {
		if cmd.Flags().Changed("labels") {
			reporter.Errorf("Labels cannot be updated on the Default machine pool")
			os.Exit(1)
		}
		if cmd.Flags().Changed("taints") {
			reporter.Errorf("Taints are not supported on the Default machine pool")
			os.Exit(1)
		}

		autoscaling, replicas, minReplicas, maxReplicas := getReplicas(cmd, reporter, machinePoolID,
			cluster.Nodes().Compute(), cluster.Nodes().AutoscaleCompute())

		if cluster.MultiAZ() {
			if !autoscaling && replicas < 3 ||
				(autoscaling && cmd.Flags().Changed("min-replicas") && minReplicas < 3) {
				reporter.Errorf("Default machine pool for AZ cluster requires at least 3 compute nodes")
				os.Exit(1)
			}

			if !autoscaling && replicas%3 != 0 ||
				(autoscaling && (minReplicas%3 != 0 || maxReplicas%3 != 0)) {
				reporter.Errorf("Multi AZ clusters require that the number of compute nodes be a multiple of 3")
				os.Exit(1)
			}
		} else if !autoscaling && replicas < 2 ||
			(autoscaling && cmd.Flags().Changed("min-replicas") && minReplicas < 2) {
			reporter.Errorf("Default machine pool requires at least 2 compute nodes")
			os.Exit(1)
		}

		clusterConfig := ocm.Spec{
			Autoscaling:  autoscaling,
			ComputeNodes: replicas,
			MinReplicas:  minReplicas,
			MaxReplicas:  maxReplicas,
		}

		reporter.Debugf("Updating machine pool '%s' on cluster '%s'", machinePoolID, clusterKey)
		err = ocmClient.UpdateCluster(clusterKey, awsCreator.AccountID, clusterConfig)
		if err != nil {
			reporter.Errorf("Failed to update machine pool '%s' on cluster '%s': %s",
				machinePoolID, clusterKey, err)
			os.Exit(1)
		}
		reporter.Infof("Updated machine pool '%s' on cluster '%s'", machinePoolID, clusterKey)

		os.Exit(0)
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

	autoscaling, replicas, minReplicas, maxReplicas := getReplicas(cmd, reporter, machinePoolID,
		machinePool.Replicas(), machinePool.Autoscaling())

	if !autoscaling && replicas < 0 ||
		(autoscaling && cmd.Flags().Changed("min-replicas") && minReplicas < 0) {
		reporter.Errorf("The number of machine pool replicas needs to be a non-negative integer")
		os.Exit(1)
	}

	if cluster.MultiAZ() &&
		(!autoscaling && replicas%3 != 0 ||
			(autoscaling && (minReplicas%3 != 0 || maxReplicas%3 != 0))) {
		reporter.Errorf("Multi AZ clusters require that the number of MachinePool replicas be a multiple of 3")
		os.Exit(1)
	}

	labels := args.labels
	labelMap := make(map[string]string)
	if interactive.Enabled() {
		if labels == "" {
			for lk, lv := range machinePool.Labels() {
				if labels != "" {
					labels += ","
				}
				labels += fmt.Sprintf("%s=%s", lk, lv)
			}
		}
		labels, err = interactive.GetString(interactive.Input{
			Question: "Labels",
			Help:     cmd.Flags().Lookup("labels").Usage,
			Default:  labels,
		})
		if err != nil {
			reporter.Errorf("Expected a valid comma-separated list of attributes: %s", err)
			os.Exit(1)
		}
	}
	labels = strings.Trim(labels, " ")
	if labels != "" {
		for _, label := range strings.Split(labels, ",") {
			if !strings.Contains(label, "=") {
				reporter.Errorf("Expected key=value format for labels")
				os.Exit(1)
			}
			tokens := strings.Split(label, "=")
			labelMap[strings.TrimSpace(tokens[0])] = strings.TrimSpace(tokens[1])
		}
	}

	taints := args.taints
	taintBuilders := []*cmv1.TaintBuilder{}
	if interactive.Enabled() {
		if taints == "" {
			for _, taint := range machinePool.Taints() {
				if taints != "" {
					taints += ","
				}
				taints += fmt.Sprintf("%s=%s:%s", taint.Key(), taint.Value(), taint.Effect())
			}
		}
		taints, err = interactive.GetString(interactive.Input{
			Question: "Taints",
			Help:     cmd.Flags().Lookup("taints").Usage,
			Default:  taints,
		})
		if err != nil {
			reporter.Errorf("Expected a valid comma-separated list of attributes: %s", err)
			os.Exit(1)
		}
	}
	taints = strings.Trim(taints, " ")
	if taints != "" {
		for _, taint := range strings.Split(taints, ",") {
			if !strings.Contains(taint, "=") || !strings.Contains(taint, ":") {
				reporter.Errorf("Expected key=value:scheduleType format for taints")
				os.Exit(1)
			}
			tokens := strings.FieldsFunc(taint, Split)
			taintBuilders = append(taintBuilders, cmv1.NewTaint().Key(tokens[0]).Value(tokens[1]).Effect(tokens[2]))
		}
	}

	mpBuilder := cmv1.NewMachinePool().
		ID(machinePool.ID())

	// Check either for an explicit flag or interactive mode. Since
	// interactive will always show both labels and taints we can safely
	// assume that the value entered is the same as the value desired.
	if cmd.Flags().Changed("labels") || interactive.Enabled() {
		mpBuilder = mpBuilder.Labels(labelMap)
	}
	if cmd.Flags().Changed("taints") || interactive.Enabled() {
		mpBuilder = mpBuilder.Taints(taintBuilders...)
	}

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
	_, err = ocmClient.UpdateMachinePool(cluster.ID(), machinePool)
	if err != nil {
		reporter.Errorf("Failed to update machine pool '%s' on cluster '%s': %s",
			machinePool.ID(), clusterKey, err)
		os.Exit(1)
	}
	reporter.Infof("Updated machine pool '%s' on cluster '%s'", machinePool.ID(), clusterKey)
}

func getReplicas(cmd *cobra.Command,
	reporter *rprtr.Object,
	machinePoolID string,
	existingReplicas int,
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
	replicasRequired := existingAutoscaling == nil

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
				Required: replicasRequired,
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
				Required: replicasRequired,
			})
			if err != nil {
				reporter.Errorf("Expected a valid number of max replicas: %s", err)
				os.Exit(1)
			}
		}
	} else if interactive.Enabled() || !isReplicasSet {
		if !isReplicasSet {
			replicas = existingReplicas
		}
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

func Split(r rune) bool {
	return r == '=' || r == ':'
}
