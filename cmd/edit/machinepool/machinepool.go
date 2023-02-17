package machinepool

import (
	"os"
	"regexp"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/spf13/cobra"
)

// Regular expression to used to make sure that the identifier given by the
// user is safe and that it there is no risk of SQL injection:
var machinePoolKeyRE = regexp.MustCompile(`^[a-z]([-a-z0-9]*[a-z0-9])?$`)

func editMachinePool(cmd *cobra.Command, machinePoolID string, clusterKey string, cluster *cmv1.Cluster,
	r *rosa.Runtime) {
	var err error
	if machinePoolID != "Default" && !machinePoolKeyRE.MatchString(machinePoolID) {
		r.Reporter.Errorf("Expected a valid identifier for the machine pool")
		os.Exit(1)
	}

	isMinReplicasSet := cmd.Flags().Changed("min-replicas")
	isMaxReplicasSet := cmd.Flags().Changed("max-replicas")
	isReplicasSet := cmd.Flags().Changed("replicas")
	isAutoscalingSet := cmd.Flags().Changed("enable-autoscaling")
	isLabelsSet := cmd.Flags().Changed("labels")
	isTaintsSet := cmd.Flags().Changed("taints")

	// if no value set enter interactive mode
	if !(isMinReplicasSet || isMaxReplicasSet || isReplicasSet || isAutoscalingSet || isLabelsSet || isTaintsSet) {
		interactive.Enable()
	}

	// Editing the default machine pool is a different process
	if machinePoolID == "Default" {
		if isTaintsSet {
			r.Reporter.Errorf("Taints are not supported on the Default machine pool")
			os.Exit(1)
		}

		clusterConfig := ocm.Spec{}

		autoscaling, replicas, minReplicas, maxReplicas, scalingUpdated, _, _ :=
			getMachinePoolReplicas(cmd, r.Reporter, machinePoolID, cluster.Nodes().Compute(),
				cluster.Nodes().AutoscaleCompute(), !isLabelsSet)

		if scalingUpdated {
			if cluster.MultiAZ() {
				if !autoscaling && replicas < 3 ||
					(autoscaling && isMinReplicasSet && minReplicas < 3) {
					r.Reporter.Errorf("Default machine pool for AZ cluster requires at least 3 compute nodes")
					os.Exit(1)
				}

				if !autoscaling && replicas%3 != 0 ||
					(autoscaling && (minReplicas%3 != 0 || maxReplicas%3 != 0)) {
					r.Reporter.Errorf("Multi AZ clusters require that the number of compute nodes be a multiple of 3")
					os.Exit(1)
				}
			} else if !autoscaling && replicas < 2 ||
				(autoscaling && isMinReplicasSet && minReplicas < 2) {
				r.Reporter.Errorf("Default machine pool requires at least 2 compute nodes")
				os.Exit(1)
			}

			clusterConfig = ocm.Spec{
				Autoscaling:  autoscaling,
				ComputeNodes: replicas,
				MinReplicas:  minReplicas,
				MaxReplicas:  maxReplicas,
			}
		}

		labelMap := getLabels(cmd, r.Reporter, cluster.Nodes().ComputeLabels())

		if isLabelsSet || interactive.Enabled() {
			clusterConfig.ComputeLabels = labelMap
		}

		r.Reporter.Debugf("Updating machine pool '%s' on cluster '%s'", machinePoolID, clusterKey)
		err = r.OCMClient.UpdateCluster(clusterKey, r.Creator, clusterConfig)
		if err != nil {
			r.Reporter.Errorf("Failed to update machine pool '%s' on cluster '%s': %s",
				machinePoolID, clusterKey, err)
			os.Exit(1)
		}
		r.Reporter.Infof("Updated machine pool '%s' on cluster '%s'", machinePoolID, clusterKey)

		os.Exit(0)
	}

	// Try to find the machine pool:
	r.Reporter.Debugf("Loading machine pools for cluster '%s'", clusterKey)
	machinePools, err := r.OCMClient.GetMachinePools(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("Failed to get machine pools for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	var machinePool *cmv1.MachinePool
	for _, item := range machinePools {
		if item.ID() == machinePoolID {
			machinePool = item
		}
	}
	if machinePool == nil {
		r.Reporter.Errorf("Failed to get machine pool '%s' for cluster '%s'", machinePoolID, clusterKey)
		os.Exit(1)
	}

	autoscaling, replicas, minReplicas, maxReplicas, scalingUpdated, minReplicaUpdated, maxReplicaUpdated :=
		getMachinePoolReplicas(cmd, r.Reporter, machinePoolID, machinePool.Replicas(), machinePool.Autoscaling(),
			!isLabelsSet && !isTaintsSet)

	if scalingUpdated {
		if !autoscaling && replicas < 0 ||
			(autoscaling && isMinReplicasSet && minReplicas < 0) {
			r.Reporter.Errorf("The number of machine pool replicas needs to be a non-negative integer")
			os.Exit(1)
		}

		if cluster.MultiAZ() && isMultiAZMachinePool(machinePool) &&
			(!autoscaling && replicas%3 != 0 ||
				(autoscaling && (minReplicas%3 != 0 || maxReplicas%3 != 0))) {
			r.Reporter.Errorf("Multi AZ clusters require that the number of MachinePool replicas be a multiple of 3")
			os.Exit(1)
		}
	}

	labelMap := getLabels(cmd, r.Reporter, machinePool.Labels())

	taintBuilders := getTaints(cmd, r, machinePool.Taints())

	mpBuilder := cmv1.NewMachinePool().
		ID(machinePool.ID())

	// Check either for an explicit flag or interactive mode. Since
	// interactive will always show both labels and taints we can safely
	// assume that the value entered is the same as the value desired.
	if isLabelsSet || interactive.Enabled() {
		mpBuilder = mpBuilder.Labels(labelMap)
	}
	if isTaintsSet || interactive.Enabled() {
		mpBuilder = mpBuilder.Taints(taintBuilders...)
	}

	if scalingUpdated {
		if autoscaling {
			asBuilder := cmv1.NewMachinePoolAutoscaling()

			if minReplicaUpdated && minReplicas >= 0 {
				asBuilder = asBuilder.MinReplicas(minReplicas)
			}
			if maxReplicaUpdated && maxReplicas >= 0 {
				asBuilder = asBuilder.MaxReplicas(maxReplicas)
			}

			mpBuilder = mpBuilder.Autoscaling(asBuilder)
		} else {
			mpBuilder = mpBuilder.Replicas(replicas)
		}
	}

	machinePool, err = mpBuilder.Build()
	if err != nil {
		r.Reporter.Errorf("Failed to create machine pool for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	r.Reporter.Debugf("Updating machine pool '%s' on cluster '%s'", machinePool.ID(), clusterKey)
	_, err = r.OCMClient.UpdateMachinePool(cluster.ID(), machinePool)
	if err != nil {
		r.Reporter.Errorf("Failed to update machine pool '%s' on cluster '%s': %s",
			machinePool.ID(), clusterKey, err)
		os.Exit(1)
	}
	r.Reporter.Infof("Updated machine pool '%s' on cluster '%s'", machinePool.ID(), clusterKey)
}

func getMachinePoolReplicas(cmd *cobra.Command,
	reporter *rprtr.Object,
	machinePoolID string,
	existingReplicas int,
	existingAutoscaling *cmv1.MachinePoolAutoscaling,
	askForScalingParams bool) (autoscaling bool,
	replicas, minReplicas, maxReplicas int, scalingUpdated bool, minReplicaUpdated bool, maxReplicaUpdated bool) {
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

	scalingUpdated = isMinReplicasSet || isMaxReplicasSet || isReplicasSet || isAutoscalingSet ||
		askForScalingParams || interactive.Enabled()
	minReplicaUpdated = isMinReplicasSet
	maxReplicaUpdated = isMaxReplicasSet

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
		if !isMinReplicasSet && (interactive.Enabled() || !isMaxReplicasSet && askForScalingParams) {
			minReplicaUpdated = true
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
		if !isMaxReplicasSet && (interactive.Enabled() || !isMinReplicasSet && askForScalingParams) {
			maxReplicaUpdated = true
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
	} else if interactive.Enabled() || !isReplicasSet && askForScalingParams {
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

// Single-AZ: AvailabilityZones == []string{"us-east-1a"}
func isMultiAZMachinePool(machinePool *cmv1.MachinePool) bool {
	return len(machinePool.AvailabilityZones()) != 1
}
