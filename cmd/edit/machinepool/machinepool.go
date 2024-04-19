package machinepool

import (
	"fmt"
	"regexp"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	mpHelpers "github.com/openshift/rosa/pkg/helper/machinepools"
	"github.com/openshift/rosa/pkg/interactive"
	rprtr "github.com/openshift/rosa/pkg/reporter"
	"github.com/openshift/rosa/pkg/rosa"
)

// Regular expression to used to make sure that the identifier given by the
// user is safe and that it there is no risk of SQL injection:
var machinePoolKeyRE = regexp.MustCompile(`^[a-z]([-a-z0-9]*[a-z0-9])?$`)

func editMachinePool(cmd *cobra.Command, machinePoolID string, clusterKey string, cluster *cmv1.Cluster,
	r *rosa.Runtime) error {
	if !machinePoolKeyRE.MatchString(machinePoolID) {
		return fmt.Errorf("Expected a valid identifier for the machine pool")
	}

	mpHelpers.HostedClusterOnlyFlag(r, cmd, "version")
	mpHelpers.HostedClusterOnlyFlag(r, cmd, "autorepair")
	mpHelpers.HostedClusterOnlyFlag(r, cmd, "tuning-configs")

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

	// Try to find the machine pool:
	r.Reporter.Debugf("Loading machine pools for cluster '%s'", clusterKey)
	machinePools, err := r.OCMClient.GetMachinePools(cluster.ID())
	if err != nil {
		return fmt.Errorf("Failed to get machine pools for cluster '%s': %v", clusterKey, err)
	}

	var machinePool *cmv1.MachinePool
	for _, item := range machinePools {
		if item.ID() == machinePoolID {
			machinePool = item
		}
	}
	if machinePool == nil {
		return fmt.Errorf("Failed to get machine pool '%s' for cluster '%s'", machinePoolID, clusterKey)
	}

	autoscaling, replicas, minReplicas, maxReplicas, err :=
		getMachinePoolReplicas(cmd, r.Reporter, machinePoolID, machinePool.Replicas(), machinePool.Autoscaling(),
			!isLabelsSet && !isTaintsSet)

	if err != nil {
		return fmt.Errorf("Failed to get autoscaling or replicas: '%s'", err)
	}

	if !autoscaling && replicas < 0 ||
		(autoscaling && isMinReplicasSet && minReplicas < 0) {
		return fmt.Errorf("The number of machine pool replicas needs to be a non-negative integer")
	}

	if cluster.MultiAZ() && isMultiAZMachinePool(machinePool) &&
		(!autoscaling && replicas%3 != 0 ||
			(autoscaling && (minReplicas%3 != 0 || maxReplicas%3 != 0))) {
		return fmt.Errorf("Multi AZ clusters require that the number of MachinePool replicas be a multiple of 3")
	}

	labelMap := mpHelpers.GetLabelMap(cmd, r, machinePool.Labels(), args.labels)

	taintBuilders := mpHelpers.GetTaints(cmd, r, machinePool.Taints(), args.taints)

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

	if autoscaling {
		mpBuilder.Autoscaling(editMachinePoolAutoscaling(machinePool, minReplicas, maxReplicas))
	} else {
		if machinePool.Replicas() != replicas {
			mpBuilder.Replicas(replicas)
		}
	}

	machinePool, err = mpBuilder.Build()
	if err != nil {
		return fmt.Errorf("Failed to create machine pool for cluster '%s': %v", clusterKey, err)
	}

	r.Reporter.Debugf("Updating machine pool '%s' on cluster '%s'", machinePool.ID(), clusterKey)
	_, err = r.OCMClient.UpdateMachinePool(cluster.ID(), machinePool)
	if err != nil {
		return fmt.Errorf("Failed to update machine pool '%s' on cluster '%s': %s",
			machinePool.ID(), clusterKey, err)
	}
	r.Reporter.Infof("Updated machine pool '%s' on cluster '%s'", machinePool.ID(), clusterKey)
	return nil
}

func getMachinePoolReplicas(cmd *cobra.Command,
	reporter *rprtr.Object,
	machinePoolID string,
	existingReplicas int,
	existingAutoscaling *cmv1.MachinePoolAutoscaling,
	askForScalingParams bool) (autoscaling bool,
	replicas, minReplicas, maxReplicas int, err error) {
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
		err = fmt.Errorf("Autoscaling is not enabled on machine pool '%s'. can't set min or max replicas",
			machinePoolID)
		return
	}

	// if the user set replicas but enabled autoscaling or hasn't disabled existing autoscaling
	if isReplicasSet && existingAutoscaling != nil && (!isAutoscalingSet || autoscaling) {
		err = fmt.Errorf("Autoscaling enabled on machine pool '%s'. can't set replicas",
			machinePoolID)
		return
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
				err = fmt.Errorf("Expected a valid value for enable-autoscaling: %s", err)
				return
			}
		}
	}

	if autoscaling {
		// Prompt for min replicas if neither min or max is set or interactive mode
		if !isMinReplicasSet && (interactive.Enabled() || !isMaxReplicasSet && askForScalingParams) {
			minReplicas, err = interactive.GetInt(interactive.Input{
				Question: "Min replicas",
				Help:     cmd.Flags().Lookup("min-replicas").Usage,
				Default:  existingAutoscaling.MinReplicas(),
				Required: replicasRequired,
			})
			if err != nil {
				err = fmt.Errorf("Expected a valid number of min replicas: %s", err)
				return
			}
		}

		// Prompt for max replicas if neither min or max is set or interactive mode
		if !isMaxReplicasSet && (interactive.Enabled() || !isMinReplicasSet && askForScalingParams) {
			maxReplicas, err = interactive.GetInt(interactive.Input{
				Question: "Max replicas",
				Help:     cmd.Flags().Lookup("max-replicas").Usage,
				Default:  existingAutoscaling.MaxReplicas(),
				Required: replicasRequired,
			})
			if err != nil {
				err = fmt.Errorf("Expected a valid number of max replicas: %s", err)
				return
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
			err = fmt.Errorf("Expected a valid number of replicas: %s", err)
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

func editMachinePoolAutoscaling(machinePool *cmv1.MachinePool,
	minReplicas int, maxReplicas int) *cmv1.MachinePoolAutoscalingBuilder {
	asBuilder := cmv1.NewMachinePoolAutoscaling()
	changed := false

	if machinePool.Autoscaling().MinReplicas() != minReplicas && minReplicas >= 1 {
		asBuilder = asBuilder.MinReplicas(minReplicas)
		changed = true
	}
	if machinePool.Autoscaling().MaxReplicas() != maxReplicas && maxReplicas >= 1 {
		asBuilder = asBuilder.MaxReplicas(maxReplicas)
		changed = true
	}

	if changed {
		return asBuilder
	}
	return nil
}
