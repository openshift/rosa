package machinepool

import (
	"fmt"
	"os"
	"slices"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/helper/machinepools"
	"github.com/openshift/rosa/pkg/interactive"
	. "github.com/openshift/rosa/pkg/kubeletconfig"
	"github.com/openshift/rosa/pkg/machinepool"
	rprtr "github.com/openshift/rosa/pkg/reporter"
	"github.com/openshift/rosa/pkg/rosa"
)

func editNodePool(cmd *cobra.Command, nodePoolID string,
	clusterKey string, cluster *cmv1.Cluster, r *rosa.Runtime) error {
	var err error

	isMinReplicasSet := cmd.Flags().Changed("min-replicas")
	isMaxReplicasSet := cmd.Flags().Changed("max-replicas")
	isReplicasSet := cmd.Flags().Changed("replicas")
	isAutoscalingSet := cmd.Flags().Changed("enable-autoscaling")
	isLabelsSet := cmd.Flags().Changed("labels")
	isTaintsSet := cmd.Flags().Changed("taints")
	isAutorepairSet := cmd.Flags().Changed("autorepair")
	isTuningsConfigSet := cmd.Flags().Changed("tuning-configs")
	isKubeletConfigSet := cmd.Flags().Changed("kubelet-configs")
	isNodeDrainGracePeriodSet := cmd.Flags().Changed("node-drain-grace-period")
	isUpgradeMaxSurgeSet := cmd.Flags().Changed("max-surge")
	isUpgradeMaxUnavailableSet := cmd.Flags().Changed("max-unavailable")

	// isAnyAdditionalParameterSet is true if at least one parameter not related to replicas and autoscaling is set
	isAnyAdditionalParameterSet := isLabelsSet || isTaintsSet || isAutorepairSet || isTuningsConfigSet ||
		isKubeletConfigSet || isUpgradeMaxSurgeSet || isUpgradeMaxUnavailableSet
	isAnyParameterSet := isMinReplicasSet || isMaxReplicasSet || isReplicasSet ||
		isAutoscalingSet || isAnyAdditionalParameterSet

	// if no value set enter interactive mode
	if !isAnyParameterSet {
		interactive.Enable()
	}

	// Try to find the node pool
	r.Reporter.Debugf("Loading machine pool for hosted cluster '%s'", clusterKey)
	nodePool, exists, err := r.OCMClient.GetNodePool(cluster.ID(), nodePoolID)
	if err != nil {
		return fmt.Errorf("Failed to get machine pools for hosted cluster '%s': %v", clusterKey, err)
	}
	if !exists {
		return fmt.Errorf("Machine pool '%s' does not exist for hosted cluster '%s'", nodePoolID, clusterKey)
	}

	autoscaling, replicas, minReplicas, maxReplicas, err := getNodePoolReplicas(cmd, r.Reporter, nodePoolID,
		nodePool.Replicas(), nodePool.Autoscaling(), isAnyAdditionalParameterSet)
	if err != nil {
		return fmt.Errorf("Failed to get autoscaling or replicas: '%s'", err)
	}

	if !autoscaling && replicas < 0 {
		return fmt.Errorf("The number of machine pool replicas needs to be a non-negative integer")
	}

	if autoscaling && cmd.Flags().Changed("min-replicas") && minReplicas < 1 {
		return fmt.Errorf("The number of machine pool min-replicas needs to be greater than zero")
	}

	labelMap := machinepools.GetLabelMap(cmd, r, nodePool.Labels(), args.labels)

	taintBuilders := machinepools.GetTaints(cmd, r, nodePool.Taints(), args.taints)

	npBuilder := cmv1.NewNodePool().
		ID(nodePool.ID())

	// Check either for an explicit flag or interactive mode. Since
	// interactive will always show both labels and taints we can safely
	// assume that the value entered is the same as the value desired.
	if isLabelsSet || interactive.Enabled() {
		npBuilder = npBuilder.Labels(labelMap)
	}
	if isTaintsSet || interactive.Enabled() {
		npBuilder = npBuilder.Taints(taintBuilders...)
	}

	fillAutoScalingAndReplicas(npBuilder, autoscaling, nodePool, minReplicas, maxReplicas, replicas)

	if isAutorepairSet || interactive.Enabled() {
		autorepair := args.autorepair
		if interactive.Enabled() {
			autorepair, err = interactive.GetBool(interactive.Input{
				Question: "Autorepair",
				Help:     cmd.Flags().Lookup("autorepair").Usage,
				Default:  autorepair,
				Required: false,
			})
			if err != nil {
				return fmt.Errorf("Expected a valid value for autorepair: %s", err)
			}
		}

		npBuilder.AutoRepair(autorepair)
	}

	if isTuningsConfigSet || interactive.Enabled() {
		var inputTuningConfig []string
		tuningConfigs := args.tuningConfigs
		// Get the list of available tuning configs
		availableTuningConfigs, err := r.OCMClient.GetTuningConfigsName(cluster.ID())
		if err != nil {
			return fmt.Errorf("%s", err)
		}
		if tuningConfigs != "" {
			if len(availableTuningConfigs) > 0 {
				inputTuningConfig = strings.Split(tuningConfigs, ",")
			} else {
				// Parameter will be ignored
				r.Reporter.Warnf("No tuning config available for cluster '%s'. "+
					"Any tuning config in input will be ignored", cluster.ID())
			}
		}

		if interactive.Enabled() {
			if !isTuningsConfigSet {
				// Interactive mode without explicit input parameter. Take the existing value
				inputTuningConfig = nodePool.TuningConfigs()
			}

			// Skip if no tuning configs are available
			if len(availableTuningConfigs) > 0 {
				inputTuningConfig, err = interactive.GetMultipleOptions(interactive.Input{
					Question: "Tuning configs",
					Help:     cmd.Flags().Lookup("tuning-configs").Usage,
					Options:  availableTuningConfigs,
					Default:  inputTuningConfig,
					Required: false,
				})
				if err != nil {
					return fmt.Errorf("Expected a valid value for tuning configs: %s", err)
				}
			}
		}

		npBuilder.TuningConfigs(inputTuningConfig...)
	}

	if isKubeletConfigSet || interactive.Enabled() {
		var inputKubeletConfig []string
		kubeletConfigs := args.kubeletConfigs
		// Get the list of available tuning configs
		availableKubeletConfigs, err := r.OCMClient.ListKubeletConfigNames(cluster.ID())
		if err != nil {
			return fmt.Errorf("%s", err)
		}
		if kubeletConfigs != "" {
			if len(availableKubeletConfigs) > 0 {
				inputKubeletConfig = strings.Split(kubeletConfigs, ",")
			} else {
				// Parameter will be ignored
				r.Reporter.Warnf("No kubelet config available for cluster '%s'. "+
					"Any kubelet config in input will be ignored", cluster.ID())
			}
		}

		if interactive.Enabled() {
			if !isKubeletConfigSet {
				// Interactive mode without explicit input parameter. Take the existing value
				inputKubeletConfig = nodePool.KubeletConfigs()
			}

			// Skip if no kubelet configs are available
			if len(availableKubeletConfigs) > 0 {
				inputKubeletConfig, err = interactive.GetMultipleOptions(interactive.Input{
					Question: "Kubelet config",
					Help:     cmd.Flags().Lookup("kubelet-configs").Usage,
					Options:  availableKubeletConfigs,
					Default:  inputKubeletConfig,
					Required: false,
					Validators: []interactive.Validator{
						machinepool.ValidateKubeletConfig,
					},
				})
				if err != nil {
					return fmt.Errorf("Expected a valid value for kubelet config: %s", err)
				}
			}
		}
		err = machinepool.ValidateKubeletConfig(inputKubeletConfig)
		if err != nil {
			r.Reporter.Errorf(err.Error())
			os.Exit(1)
		}
		npBuilder.KubeletConfigs(inputKubeletConfig...)
		isKubeletConfigSet = true
	}

	if isNodeDrainGracePeriodSet || interactive.Enabled() {
		nodeDrainGracePeriod := args.nodeDrainGracePeriod
		if nodeDrainGracePeriod == "" && nodePool.NodeDrainGracePeriod() != nil &&
			nodePool.NodeDrainGracePeriod().Value() != 0 {
			nodeDrainGracePeriod = fmt.Sprintf("%d minutes", int(nodePool.NodeDrainGracePeriod().Value()))
		}

		if interactive.Enabled() {
			nodeDrainGracePeriod, err = interactive.GetString(interactive.Input{
				Question: "Node drain grace period",
				Help:     cmd.Flags().Lookup("node-drain-grace-period").Usage,
				Default:  nodeDrainGracePeriod,
				Required: false,
				Validators: []interactive.Validator{
					machinepools.ValidateNodeDrainGracePeriod,
				},
			})
			if err != nil {
				return fmt.Errorf("Expected a valid value for Node drain grace period: %s", err)
			}
		}

		if nodeDrainGracePeriod != "" {
			nodeDrainBuilder, err := machinepools.CreateNodeDrainGracePeriodBuilder(nodeDrainGracePeriod)
			if err != nil {
				return fmt.Errorf(err.Error())
			}
			npBuilder.NodeDrainGracePeriod(nodeDrainBuilder)
		}
	}

	if isUpgradeMaxSurgeSet && isUpgradeMaxUnavailableSet {
		maxSurge := args.maxSurge
		if maxSurge == "" && nodePool.ManagementUpgrade().MaxSurge() != "" {
			maxSurge = nodePool.ManagementUpgrade().MaxSurge()
		}

		maxUnavailable := args.maxUnavailable
		if maxUnavailable == "" && nodePool.ManagementUpgrade().MaxUnavailable() != "" {
			maxUnavailable = nodePool.ManagementUpgrade().MaxUnavailable()
		}
		if interactive.Enabled() {
			maxSurge, err = interactive.GetString(interactive.Input{
				Question: "Max surge",
				Help:     cmd.Flags().Lookup("max-surge").Usage,
				Default:  maxSurge,
				Required: false,
				Validators: []interactive.Validator{
					machinepools.ValidateUpgradeMaxSurgeUnavailable,
				},
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid value for max surge: %s", err)
				os.Exit(1)
			}

			maxUnavailable, err = interactive.GetString(interactive.Input{
				Question: "Max unavailable",
				Help:     cmd.Flags().Lookup("max-unavailable").Usage,
				Default:  maxUnavailable,
				Required: false,
				Validators: []interactive.Validator{
					machinepools.ValidateUpgradeMaxSurgeUnavailable,
				},
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid value for max unavailable: %s", err)
				os.Exit(1)
			}
		}

		if maxSurge != "" || maxUnavailable != "" {
			mgmtUpgradeBuilder := cmv1.NewNodePoolManagementUpgrade()
			if maxSurge != "" {
				mgmtUpgradeBuilder.MaxSurge(maxSurge)
			}
			if maxUnavailable != "" {
				mgmtUpgradeBuilder.MaxUnavailable(maxUnavailable)
			}
			npBuilder.ManagementUpgrade(mgmtUpgradeBuilder)
		}
	}

	update, err := npBuilder.Build()
	if err != nil {
		return fmt.Errorf("Failed to create machine pool for hosted cluster '%s': %v", clusterKey, err)
	}

	if isKubeletConfigSet && !promptForNodePoolNodeRecreate(nodePool, update, PromptToAcceptNodePoolNodeRecreate, r) {
		return nil
	}

	r.Reporter.Debugf("Updating machine pool '%s' on hosted cluster '%s'", nodePool.ID(), clusterKey)
	_, err = r.OCMClient.UpdateNodePool(cluster.ID(), update)
	if err != nil {
		return fmt.Errorf("Failed to update machine pool '%s' on hosted cluster '%s': %s",
			nodePool.ID(), clusterKey, err)
	}
	r.Reporter.Infof("Updated machine pool '%s' on hosted cluster '%s'", nodePool.ID(), clusterKey)
	return nil
}

// fillAutoScalingAndReplicas is filling either autoscaling or replicas value in the builder
func fillAutoScalingAndReplicas(npBuilder *cmv1.NodePoolBuilder, autoscaling bool, existingNodepool *cmv1.NodePool,
	minReplicas int, maxReplicas int, replicas int) {
	if autoscaling {
		npBuilder.Autoscaling(editAutoscaling(existingNodepool, minReplicas, maxReplicas))
	} else {
		npBuilder.Replicas(replicas)
	}
}

// promptForNodePoolNodeRecreate - prompts the user to accept that their changes will cause the nodes
// in their nodepool to be recreated. This primarily applies to KubeletConfig modifications.
func promptForNodePoolNodeRecreate(
	original *cmv1.NodePool,
	update *cmv1.NodePool,
	promptFunc func(r *rosa.Runtime) bool, r *rosa.Runtime) bool {
	if len(original.KubeletConfigs()) != len(update.KubeletConfigs()) {
		return promptFunc(r)
	}

	for _, s := range update.KubeletConfigs() {
		if !slices.Contains(original.KubeletConfigs(), s) {
			return promptFunc(r)
		}
	}

	return true
}

func getNodePoolReplicas(cmd *cobra.Command,
	reporter *rprtr.Object,
	nodePoolID string,
	existingReplicas int,
	existingAutoscaling *cmv1.NodePoolAutoscaling, isAnyAdditionalParameterSet bool) (autoscaling bool,
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
			nodePoolID)
		return

	}

	// if the user set replicas but enabled autoscaling or hasn't disabled existing autoscaling
	if isReplicasSet && existingAutoscaling != nil && (!isAutoscalingSet || autoscaling) {
		err = fmt.Errorf("Autoscaling enabled on machine pool '%s'. can't set replicas",
			nodePoolID)
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
		if !isMinReplicasSet && (interactive.Enabled() || !isMaxReplicasSet && !isAnyAdditionalParameterSet) {
			minReplicas, err = interactive.GetInt(interactive.Input{
				Question: "Min replicas",
				Help:     cmd.Flags().Lookup("min-replicas").Usage,
				Default:  existingAutoscaling.MinReplica(),
				Required: replicasRequired,
				Validators: []interactive.Validator{
					machinepools.MinNodePoolReplicaValidator(true),
				},
			})
			if err != nil {
				err = fmt.Errorf("Expected a valid number of min replicas: %s", err)
				return
			}
		}

		// Prompt for max replicas if neither min or max is set or interactive mode
		if !isMaxReplicasSet && (interactive.Enabled() || !isMinReplicasSet && !isAnyAdditionalParameterSet) {
			maxReplicas, err = interactive.GetInt(interactive.Input{
				Question: "Max replicas",
				Help:     cmd.Flags().Lookup("max-replicas").Usage,
				Default:  existingAutoscaling.MaxReplica(),
				Required: replicasRequired,
				Validators: []interactive.Validator{
					machinepools.MaxNodePoolReplicaValidator(minReplicas),
				},
			})
			if err != nil {
				err = fmt.Errorf("Expected a valid number of max replicas: %s", err)
				return
			}
		}
	} else if interactive.Enabled() || !isReplicasSet {
		if !isReplicasSet {
			replicas = existingReplicas
		}
		if !interactive.Enabled() && isAnyAdditionalParameterSet {
			// Not interactive mode and we have at least an additional parameter set, just keep the existing replicas
			return
		}
		replicas, err = interactive.GetInt(interactive.Input{
			Question: "Replicas",
			Help:     cmd.Flags().Lookup("replicas").Usage,
			Default:  replicas,
			Required: true,
			Validators: []interactive.Validator{
				machinepools.MinNodePoolReplicaValidator(false),
			},
		})
		if err != nil {
			err = fmt.Errorf("Expected a valid number of replicas: %s", err)
			return
		}
	}
	return
}

func editAutoscaling(nodePool *cmv1.NodePool, minReplicas int, maxReplicas int) *cmv1.NodePoolAutoscalingBuilder {
	existingMinReplica := nodePool.Autoscaling().MinReplica()
	existingMaxReplica := nodePool.Autoscaling().MaxReplica()

	min := existingMinReplica
	max := existingMaxReplica

	if minReplicas != 0 {
		min = minReplicas
	}
	if maxReplicas != 0 {
		max = maxReplicas
	}

	if existingMinReplica != minReplicas || existingMaxReplica != maxReplicas {
		if min >= 1 && max >= 1 {
			return cmv1.NewNodePoolAutoscaling().MinReplica(min).MaxReplica(max)
		}
	}

	return nil
}
