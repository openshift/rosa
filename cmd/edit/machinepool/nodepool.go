package machinepool

import (
	"fmt"
	"os"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/helper/machinepools"
	"github.com/openshift/rosa/pkg/interactive"
	rprtr "github.com/openshift/rosa/pkg/reporter"
	"github.com/openshift/rosa/pkg/rosa"
)

func editNodePool(cmd *cobra.Command, nodePoolID string, clusterKey string, cluster *cmv1.Cluster, r *rosa.Runtime) {
	var err error

	isMinReplicasSet := cmd.Flags().Changed("min-replicas")
	isMaxReplicasSet := cmd.Flags().Changed("max-replicas")
	isReplicasSet := cmd.Flags().Changed("replicas")
	isAutoscalingSet := cmd.Flags().Changed("enable-autoscaling")
	isLabelsSet := cmd.Flags().Changed("labels")
	isTaintsSet := cmd.Flags().Changed("taints")
	isVersionSet := cmd.Flags().Changed("version")
	isAutorepairSet := cmd.Flags().Changed("autorepair")
	isTuningsConfigSet := cmd.Flags().Changed("tuning-configs")
	isNodeDrainGracePeriodSet := cmd.Flags().Changed("node-drain-grace-period")
	isTagsSet := cmd.Flags().Changed("tags")

	// we don't support anymore the version parameter
	if isVersionSet {
		r.Reporter.Errorf("Editing versions is not supported, for upgrades please use " +
			"'rosa upgrade machinepool'")
		os.Exit(1)
	}

	// isAnyAdditionalParameterSet is true if at least one parameter not related to replicas and autoscaling is set
	isAnyAdditionalParameterSet := isLabelsSet || isTaintsSet || isAutorepairSet || isTuningsConfigSet || isTagsSet
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
		r.Reporter.Errorf("Failed to get machine pools for hosted cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}
	if !exists {
		r.Reporter.Errorf("Machine pool '%s' does not exist for hosted cluster '%s'", nodePoolID, clusterKey)
		os.Exit(1)
	}

	autoscaling, replicas, minReplicas, maxReplicas := getNodePoolReplicas(cmd, r.Reporter, nodePoolID,
		nodePool.Replicas(), nodePool.Autoscaling(), isAnyAdditionalParameterSet)

	if !autoscaling && replicas < 0 {
		r.Reporter.Errorf("The number of machine pool replicas needs to be a non-negative integer")
		os.Exit(1)
	}

	if autoscaling && cmd.Flags().Changed("min-replicas") && minReplicas < 1 {
		r.Reporter.Errorf("The number of machine pool min-replicas needs to be greater than zero")
		os.Exit(1)
	}

	labelMap := machinepools.GetLabelMap(cmd, r, nodePool.Labels(), args.labels)

	taintBuilders := machinepools.GetTaints(cmd, r, nodePool.Taints(), args.taints)

	awsTags := machinepools.GetAwsTags(cmd, r, map[string]string{}, args.tags)

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
	var awsNodePoolBuilder *cmv1.AWSNodePoolBuilder
	if isTagsSet || interactive.Enabled() {
		awsNodePoolBuilder = cmv1.NewAWSNodePool().Tags(awsTags)
	}

	if awsNodePoolBuilder != nil {
		npBuilder = npBuilder.AWSNodePool(awsNodePoolBuilder)
	}

	if autoscaling {
		asBuilder := cmv1.NewNodePoolAutoscaling()

		if minReplicas >= 1 {
			asBuilder = asBuilder.MinReplica(minReplicas)
		}
		if maxReplicas >= 1 {
			asBuilder = asBuilder.MaxReplica(maxReplicas)
		}

		npBuilder = npBuilder.Autoscaling(asBuilder)
	} else {
		npBuilder = npBuilder.Replicas(replicas)
	}

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
				r.Reporter.Errorf("Expected a valid value for autorepair: %s", err)
				os.Exit(1)
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
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
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
					r.Reporter.Errorf("Expected a valid value for tuning configs: %s", err)
					os.Exit(1)
				}
			}
		}

		npBuilder.TuningConfigs(inputTuningConfig...)
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
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid value for Node drain grace period: %s", err)
				os.Exit(1)
			}
		}

		if nodeDrainGracePeriod != "" {
			nodeDrainBuilder, err := machinepools.CreateNodeDrainGracePeriodBuilder(nodeDrainGracePeriod)
			if err != nil {
				r.Reporter.Errorf(err.Error())
				os.Exit(1)
			}
			npBuilder.NodeDrainGracePeriod(nodeDrainBuilder)
		}
	}

	nodePool, err = npBuilder.Build()
	if err != nil {
		r.Reporter.Errorf("Failed to create machine pool for hosted cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	r.Reporter.Debugf("Updating machine pool '%s' on hosted cluster '%s'", nodePool.ID(), clusterKey)
	_, err = r.OCMClient.UpdateNodePool(cluster.ID(), nodePool)
	if err != nil {
		r.Reporter.Errorf("Failed to update machine pool '%s' on hosted cluster '%s': %s",
			nodePool.ID(), clusterKey, err)
		os.Exit(1)
	}
	r.Reporter.Infof("Updated machine pool '%s' on hosted cluster '%s'", nodePool.ID(), clusterKey)
}

func getNodePoolReplicas(cmd *cobra.Command,
	reporter *rprtr.Object,
	nodePoolID string,
	existingReplicas int,
	existingAutoscaling *cmv1.NodePoolAutoscaling, isAnyAdditionalParameterSet bool) (autoscaling bool,
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
			nodePoolID)
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
				Default:  existingAutoscaling.MinReplica(),
				Required: replicasRequired,
				Validators: []interactive.Validator{
					machinepools.MinNodePoolReplicaValidator(true),
				},
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
				Default:  existingAutoscaling.MaxReplica(),
				Required: replicasRequired,
				Validators: []interactive.Validator{
					machinepools.MaxNodePoolReplicaValidator(minReplicas),
				},
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
			reporter.Errorf("Expected a valid number of replicas: %s", err)
			os.Exit(1)
		}
	}
	return
}
