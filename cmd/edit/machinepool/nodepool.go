package machinepool

import (
	"os"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/helper/machinepools"
	"github.com/openshift/rosa/pkg/helper/versions"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/spf13/cobra"
)

func editNodePool(cmd *cobra.Command, nodePoolID string, clusterKey string, cluster *cmv1.Cluster, r *rosa.Runtime) {
	var err error

	isMinReplicasSet := cmd.Flags().Changed("min-replicas")
	isMaxReplicasSet := cmd.Flags().Changed("max-replicas")
	isReplicasSet := cmd.Flags().Changed("replicas")
	isAutoscalingSet := cmd.Flags().Changed("enable-autoscaling")
	isLabelsSet := cmd.Flags().Changed("labels")
	isTaintsSet := cmd.Flags().Changed("taints")
	isLabelOrTaintSet := isLabelsSet || isTaintsSet
	isVersionSet := cmd.Flags().Changed("version")

	// if no value set enter interactive mode
	if !(isMinReplicasSet || isMaxReplicasSet || isReplicasSet || isAutoscalingSet || isLabelsSet || isTaintsSet) {
		interactive.Enable()
	}

	// Try to find the node pool
	r.Reporter.Debugf("Loading machine pool for hosted cluster '%s'", clusterKey)
	nodePool, err := r.OCMClient.GetNodePool(cluster.ID(), nodePoolID)
	if err != nil {
		r.Reporter.Errorf("Failed to get machine pools for hosted cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	autoscaling, replicas, minReplicas, maxReplicas := getNodePoolReplicas(cmd, r.Reporter, nodePoolID,
		nodePool.Replicas(), nodePool.Autoscaling(), isLabelOrTaintSet)

	if !autoscaling && replicas < 1 ||
		(autoscaling && cmd.Flags().Changed("min-replicas") && minReplicas < 1) {
		r.Reporter.Errorf("The number of machine pool replicas needs to be greater than zero")
		os.Exit(1)
	}

	labelMap := getLabels(cmd, r.Reporter, nodePool.Labels())

	taintBuilders := getTaints(cmd, r, nodePool.Taints())

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

	if autoscaling {
		asBuilder := cmv1.NewNodePoolAutoscaling()

		if minReplicas > 1 {
			asBuilder = asBuilder.MinReplica(minReplicas)
		}
		if maxReplicas > 1 {
			asBuilder = asBuilder.MaxReplica(maxReplicas)
		}

		npBuilder = npBuilder.Autoscaling(asBuilder)
	} else {
		npBuilder = npBuilder.Replicas(replicas)
	}

	if isVersionSet || interactive.Enabled() {
		version := args.version
		channelGroup := cluster.Version().ChannelGroup()
		clusterVersion := cluster.Version().RawID()
		nodePoolVersion := ocm.GetRawVersionId(nodePool.Version().ID())
		versionList, err := versions.GetVersionList(r, channelGroup, true, true)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}

		// Filter the available list of versions for a hosted machine pool
		filteredVersionList := versions.GetFilteredVersionList(versionList, nodePoolVersion, clusterVersion)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}

		if interactive.Enabled() {
			version, err = interactive.GetOption(interactive.Input{
				Question: "OpenShift version",
				Help:     cmd.Flags().Lookup("version").Usage,
				Options:  filteredVersionList,
				Default:  version,
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid OpenShift version: %s", err)
				os.Exit(1)
			}
		}
		version, err = versions.ValidateVersion(version, filteredVersionList, channelGroup, true, true)
		if err != nil {
			r.Reporter.Errorf("Expected a valid OpenShift version: %s", err)
			os.Exit(1)
		}
		npBuilder.Version(cmv1.NewVersion().ID(version))
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
	existingAutoscaling *cmv1.NodePoolAutoscaling, isLabelOrTaintSet bool) (autoscaling bool,
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
					machinepools.MinNodePoolReplicaValidator(),
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
		if !interactive.Enabled() && isLabelOrTaintSet {
			// Not interactive mode and Label or taints set, just keep the existing replicas
			return
		}
		replicas, err = interactive.GetInt(interactive.Input{
			Question: "Replicas",
			Help:     cmd.Flags().Lookup("replicas").Usage,
			Default:  replicas,
			Required: true,
			Validators: []interactive.Validator{
				machinepools.MinNodePoolReplicaValidator(),
			},
		})
		if err != nil {
			reporter.Errorf("Expected a valid number of replicas: %s", err)
			os.Exit(1)
		}
	}
	return
}
