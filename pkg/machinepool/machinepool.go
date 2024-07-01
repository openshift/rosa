package machinepool

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	mpHelpers "github.com/openshift/rosa/pkg/helper/machinepools"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	. "github.com/openshift/rosa/pkg/kubeletconfig"
	ocmOutput "github.com/openshift/rosa/pkg/ocm/output"
	"github.com/openshift/rosa/pkg/output"
	rprtr "github.com/openshift/rosa/pkg/reporter"
	"github.com/openshift/rosa/pkg/rosa"
)

var fetchMessage string = "Fetching %s '%s' for cluster '%s'"
var notFoundMessage string = "Machine pool '%s' not found"

//go:generate mockgen -source=machinepool.go -package=machinepool -destination=machinepool_mock.go
type MachinePoolService interface {
	DescribeMachinePool(r *rosa.Runtime, cluster *cmv1.Cluster, clusterKey string, machinePoolId string) error
	ListMachinePools(r *rosa.Runtime, clusterKey string, cluster *cmv1.Cluster) error
	DeleteMachinePool(r *rosa.Runtime, machinePoolId string, clusterKey string, cluster *cmv1.Cluster) error
	EditMachinePool(cmd *cobra.Command, machinePoolID string, clusterKey string, cluster *cmv1.Cluster,
		r *rosa.Runtime) error
}

type machinePool struct {
}

var _ MachinePoolService = &machinePool{}

func NewMachinePoolService() MachinePoolService {
	return &machinePool{}
}

// ListMachinePools lists all machinepools (or, nodepools if hypershift) in a cluster
func (m *machinePool) ListMachinePools(r *rosa.Runtime, clusterKey string, cluster *cmv1.Cluster) error {
	// Load any existing machine pools for this cluster
	r.Reporter.Debugf("Loading machine pools for cluster '%s'", clusterKey)
	isHypershift := cluster.Hypershift().Enabled()
	var err error
	var machinePools []*cmv1.MachinePool
	var nodePools []*cmv1.NodePool
	if isHypershift {
		nodePools, err = r.OCMClient.GetNodePools(cluster.ID())
		if err != nil {
			return err
		}
	} else {
		machinePools, err = r.OCMClient.GetMachinePools(cluster.ID())
		if err != nil {
			return err
		}
	}

	if output.HasFlag() {
		if isHypershift {
			return output.Print(nodePools)
		}
		return output.Print(machinePools)
	}

	// Create the writer that will be used to print the tabulated results:
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	finalStringToOutput := getMachinePoolsString(machinePools)
	if isHypershift {
		finalStringToOutput = getNodePoolsString(nodePools)
	}
	fmt.Fprint(writer, finalStringToOutput)
	writer.Flush()
	return nil
}

// DescribeMachinePool describes either a machinepool, or, a nodepool (if hypershift)
func (m *machinePool) DescribeMachinePool(r *rosa.Runtime, cluster *cmv1.Cluster, clusterKey string,
	machinePoolId string) error {
	if cluster.Hypershift().Enabled() {
		return m.describeNodePool(r, cluster, clusterKey, machinePoolId)
	}

	r.Reporter.Debugf(fetchMessage, "machine pool", machinePoolId, clusterKey)
	machinePool, exists, err := r.OCMClient.GetMachinePool(cluster.ID(), machinePoolId)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf(notFoundMessage, machinePoolId)
	}

	if output.HasFlag() {
		return output.Print(machinePool)
	}

	fmt.Print(machinePoolOutput(cluster.ID(), machinePool))

	return nil
}

func (m *machinePool) describeNodePool(r *rosa.Runtime, cluster *cmv1.Cluster, clusterKey string,
	nodePoolId string) error {
	r.Reporter.Debugf(fetchMessage, "node pool", nodePoolId, clusterKey)
	nodePool, exists, err := r.OCMClient.GetNodePool(cluster.ID(), nodePoolId)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf(notFoundMessage, nodePoolId)
	}

	_, scheduledUpgrade, err := r.OCMClient.GetHypershiftNodePoolUpgrade(cluster.ID(), clusterKey, nodePoolId)
	if err != nil {
		return err
	}

	if output.HasFlag() {
		var formattedOutput map[string]interface{}
		formattedOutput, err = formatNodePoolOutput(nodePool, scheduledUpgrade)
		if err != nil {
			return err
		}
		return output.Print(formattedOutput)
	}

	// Attach and print scheduledUpgrades if they exist, otherwise, print output normally
	fmt.Print(appendUpgradesIfExist(scheduledUpgrade, nodePoolOutput(cluster.ID(), nodePool)))

	return nil
}

// Regular expression to used to make sure that the identifier given by the
// user is safe and that it there is no risk of SQL injection:
var MachinePoolKeyRE = regexp.MustCompile(`^[a-z]([-a-z0-9]*[a-z0-9])?$`)

// DeleteMachinePool deletes a machinepool from a cluster if it is possible- this function also calls the hypershift
// equivalent, deleteNodePool if it is a hypershift cluster
func (m *machinePool) DeleteMachinePool(r *rosa.Runtime, machinePoolId string, clusterKey string,
	cluster *cmv1.Cluster) error {
	if cluster.Hypershift().Enabled() {
		return deleteNodePool(r, machinePoolId, clusterKey, cluster)
	}

	// Try to find the machine pool:
	r.Reporter.Debugf("Loading machine pools for cluster '%s'", clusterKey)
	machinePools, err := r.OCMClient.GetMachinePools(cluster.ID())
	if err != nil {
		return fmt.Errorf("Failed to get machine pools for cluster '%s': %v", clusterKey, err)
	}

	var machinePool *cmv1.MachinePool
	for _, item := range machinePools {
		if item.ID() == machinePoolId {
			machinePool = item
		}
	}
	if machinePool == nil {
		return fmt.Errorf("Failed to get machine pool '%s' for cluster '%s'", machinePoolId, clusterKey)
	}

	if confirm.Confirm("delete machine pool '%s' on cluster '%s'", machinePoolId, clusterKey) {
		r.Reporter.Debugf("Deleting machine pool '%s' on cluster '%s'", machinePool.ID(), clusterKey)
		err = r.OCMClient.DeleteMachinePool(cluster.ID(), machinePool.ID())
		if err != nil {
			return fmt.Errorf("Failed to delete machine pool '%s' on cluster '%s': %s",
				machinePool.ID(), clusterKey, err)
		}
		r.Reporter.Infof("Successfully deleted machine pool '%s' from cluster '%s'", machinePoolId, clusterKey)
	}
	return nil
}

// deleteNodePool is the hypershift version of DeleteMachinePool - deleteNodePool is called in DeleteMachinePool
// if the cluster is hypershift
func deleteNodePool(r *rosa.Runtime, nodePoolID string, clusterKey string, cluster *cmv1.Cluster) error {
	// Try to find the machine pool:
	r.Reporter.Debugf("Loading machine pools for hosted cluster '%s'", clusterKey)
	nodePool, exists, err := r.OCMClient.GetNodePool(cluster.ID(), nodePoolID)
	if err != nil {
		return fmt.Errorf("Failed to get machine pools for hosted cluster '%s': %v", clusterKey,
			err)
	}
	if !exists {
		return fmt.Errorf("Machine pool '%s' does not exist for hosted cluster '%s'", nodePoolID,
			clusterKey)
	}

	if confirm.Confirm("delete machine pool '%s' on hosted cluster '%s'", nodePoolID, clusterKey) {
		r.Reporter.Debugf("Deleting machine pool '%s' on hosted cluster '%s'", nodePool.ID(), clusterKey)
		err = r.OCMClient.DeleteNodePool(cluster.ID(), nodePool.ID())
		if err != nil {
			return fmt.Errorf("Failed to delete machine pool '%s' on hosted cluster '%s': %s",
				nodePool.ID(), clusterKey, err)
		}
		r.Reporter.Infof("Successfully deleted machine pool '%s' from hosted cluster '%s'", nodePoolID,
			clusterKey)
	}
	return nil
}

func formatNodePoolOutput(nodePool *cmv1.NodePool,
	scheduledUpgrade *cmv1.NodePoolUpgradePolicy) (map[string]interface{}, error) {

	var b bytes.Buffer
	err := cmv1.MarshalNodePool(nodePool, &b)
	if err != nil {
		return nil, err
	}
	ret := make(map[string]interface{})
	err = json.Unmarshal(b.Bytes(), &ret)
	if err != nil {
		return nil, err
	}
	if scheduledUpgrade != nil &&
		scheduledUpgrade.State() != nil &&
		len(scheduledUpgrade.Version()) > 0 &&
		len(scheduledUpgrade.State().Value()) > 0 {
		upgrade := make(map[string]interface{})
		upgrade["version"] = scheduledUpgrade.Version()
		upgrade["state"] = scheduledUpgrade.State().Value()
		upgrade["nextRun"] = scheduledUpgrade.NextRun().Format("2006-01-02 15:04 MST")
		ret["scheduledUpgrade"] = upgrade
	}

	return ret, nil
}

func appendUpgradesIfExist(scheduledUpgrade *cmv1.NodePoolUpgradePolicy, output string) string {
	if scheduledUpgrade != nil {
		return fmt.Sprintf("%s"+
			"Scheduled upgrade:                     %s %s on %s\n",
			output,
			scheduledUpgrade.State().Value(),
			scheduledUpgrade.Version(),
			scheduledUpgrade.NextRun().Format("2006-01-02 15:04 MST"),
		)
	}
	return output
}

func getMachinePoolsString(machinePools []*cmv1.MachinePool) string {
	outputString := "ID\tAUTOSCALING\tREPLICAS\tINSTANCE TYPE\tLABELS\t\tTAINTS\t" +
		"\tAVAILABILITY ZONES\t\tSUBNETS\t\tSPOT INSTANCES\tDISK SIZE\tSG IDs\n"
	for _, machinePool := range machinePools {
		outputString += fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t\t%s\t\t%s\t\t%s\t\t%s\t%s\t%s\n",
			machinePool.ID(),
			ocmOutput.PrintMachinePoolAutoscaling(machinePool.Autoscaling()),
			ocmOutput.PrintMachinePoolReplicas(machinePool.Autoscaling(), machinePool.Replicas()),
			machinePool.InstanceType(),
			ocmOutput.PrintLabels(machinePool.Labels()),
			ocmOutput.PrintTaints(machinePool.Taints()),
			output.PrintStringSlice(machinePool.AvailabilityZones()),
			output.PrintStringSlice(machinePool.Subnets()),
			ocmOutput.PrintMachinePoolSpot(machinePool),
			ocmOutput.PrintMachinePoolDiskSize(machinePool),
			output.PrintStringSlice(machinePool.AWS().AdditionalSecurityGroupIds()),
		)
	}
	return outputString
}

func getNodePoolsString(nodePools []*cmv1.NodePool) string {
	outputString := "ID\tAUTOSCALING\tREPLICAS\t" +
		"INSTANCE TYPE\tLABELS\t\tTAINTS\t\tAVAILABILITY ZONE\tSUBNET\tVERSION\tAUTOREPAIR\t\n"
	for _, nodePool := range nodePools {
		outputString += fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t\t%s\t\t%s\t%s\t%s\t%s\t\n",
			nodePool.ID(),
			ocmOutput.PrintNodePoolAutoscaling(nodePool.Autoscaling()),
			ocmOutput.PrintNodePoolReplicasShort(
				ocmOutput.PrintNodePoolCurrentReplicas(nodePool.Status()),
				ocmOutput.PrintNodePoolReplicasInline(nodePool.Autoscaling(), nodePool.Replicas()),
			),
			ocmOutput.PrintNodePoolInstanceType(nodePool.AWSNodePool()),
			ocmOutput.PrintLabels(nodePool.Labels()),
			ocmOutput.PrintTaints(nodePool.Taints()),
			nodePool.AvailabilityZone(),
			nodePool.Subnet(),
			ocmOutput.PrintNodePoolVersion(nodePool.Version()),
			ocmOutput.PrintNodePoolAutorepair(nodePool.AutoRepair()),
		)
	}
	return outputString
}

func (m *machinePool) EditMachinePool(cmd *cobra.Command, machinePoolId string, clusterKey string,
	cluster *cmv1.Cluster, r *rosa.Runtime) error {
	if !MachinePoolKeyRE.MatchString(machinePoolId) {
		return fmt.Errorf("Expected a valid identifier for the machine pool")
	}
	if cluster.Hypershift().Enabled() {
		return editNodePool(cmd, machinePoolId, clusterKey, cluster, r)
	}
	editMachinePool(cmd, machinePoolId, clusterKey, cluster, r)
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

	replicas, err = cmd.Flags().GetInt("replicas")
	if err != nil {
		err = fmt.Errorf("Failed to get inputted replicas: %s", err)
		return
	}
	minReplicas, err = cmd.Flags().GetInt("min-replicas")
	if err != nil {
		err = fmt.Errorf("Failed to get inputted min replicas: %s", err)
		return
	}
	maxReplicas, err = cmd.Flags().GetInt("max-replicas")
	if err != nil {
		err = fmt.Errorf("Failed to get inputted max replicas: %s", err)
		return
	}
	autoscaling, err = cmd.Flags().GetBool("autoscaling")
	if err != nil {
		err = fmt.Errorf("Failed to get inputted autoscaling: %s", err)
		return
	}
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

		// set default values from previous autoscaling values
		if !isMinReplicasSet {
			minReplicas = existingAutoscaling.MinReplicas()
		}
		if !isMaxReplicasSet {
			maxReplicas = existingAutoscaling.MaxReplicas()
		}

		// Prompt for min replicas if neither min or max is set or interactive mode
		if !isMinReplicasSet && (interactive.Enabled() || !isMaxReplicasSet && askForScalingParams) {
			minReplicas, err = interactive.GetInt(interactive.Input{
				Question: "Min replicas",
				Help:     cmd.Flags().Lookup("min-replicas").Usage,
				Default:  minReplicas,
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
				Default:  maxReplicas,
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

	if machinePool.Autoscaling().MinReplicas() != minReplicas && minReplicas >= 0 {
		changed = true
	}
	if machinePool.Autoscaling().MaxReplicas() != maxReplicas && maxReplicas >= 0 {
		changed = true
	}

	if changed {
		asBuilder = asBuilder.MinReplicas(minReplicas).MaxReplicas(maxReplicas)
		return asBuilder
	}
	return nil
}

func editMachinePool(cmd *cobra.Command, machinePoolId string,
	clusterKey string, cluster *cmv1.Cluster, r *rosa.Runtime) error {
	mpHelpers.HostedClusterOnlyFlag(r, cmd, "autorepair")
	mpHelpers.HostedClusterOnlyFlag(r, cmd, "tuning-configs")
	mpHelpers.HostedClusterOnlyFlag(r, cmd, "kubelet-configs")

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
		if item.ID() == machinePoolId {
			machinePool = item
		}
	}
	if machinePool == nil {
		return fmt.Errorf("Failed to get machine pool '%s' for cluster '%s'", machinePoolId, clusterKey)
	}

	autoscaling, replicas, minReplicas, maxReplicas, err :=
		getMachinePoolReplicas(cmd, r.Reporter, machinePoolId, machinePool.Replicas(), machinePool.Autoscaling(),
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

	labels, err := cmd.Flags().GetString("labels")
	if err != nil {
		return fmt.Errorf("Failed to get inputted labels: '%s'", err)
	}
	labelMap := mpHelpers.GetLabelMap(cmd, r, machinePool.Labels(), labels)

	taints, err := cmd.Flags().GetString("taints")
	if err != nil {
		return fmt.Errorf("Failed to get inputted taints: '%s'", err)
	}
	taintBuilders := mpHelpers.GetTaints(cmd, r, machinePool.Taints(), taints)

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

	// isAnyAdditionalParameterSet is true if at least one parameter not related to replicas and autoscaling is set
	isAnyAdditionalParameterSet := isLabelsSet || isTaintsSet || isAutorepairSet || isTuningsConfigSet ||
		isKubeletConfigSet
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

	autoscaling, replicas, minReplicas, maxReplicas, err := getNodePoolReplicas(cmd, r, nodePoolID,
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

	labels := cmd.Flags().Lookup("labels").Value.String()
	labelMap := mpHelpers.GetLabelMap(cmd, r, nodePool.Labels(), labels)

	taints := cmd.Flags().Lookup("taints").Value.String()
	taintBuilders := mpHelpers.GetTaints(cmd, r, nodePool.Taints(), taints)

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
		autorepair, err := strconv.ParseBool(cmd.Flags().Lookup("autorepair").Value.String())
		if err != nil {
			return fmt.Errorf("Failed to parse autorepair flag: %s", err)
		}
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
		tuningConfigs := cmd.Flags().Lookup("tuning-configs").Value.String()
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
		kubeletConfigs := cmd.Flags().Lookup("kubelet-configs").Value.String()
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
						ValidateKubeletConfig,
					},
				})
				if err != nil {
					return fmt.Errorf("Expected a valid value for kubelet config: %s", err)
				}
			}
		}
		err = ValidateKubeletConfig(inputKubeletConfig)
		if err != nil {
			r.Reporter.Errorf(err.Error())
			os.Exit(1)
		}
		npBuilder.KubeletConfigs(inputKubeletConfig...)
		isKubeletConfigSet = true
	}

	if isNodeDrainGracePeriodSet || interactive.Enabled() {
		nodeDrainGracePeriod := cmd.Flags().Lookup("node-drain-grace-period").Value.String()
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
					mpHelpers.ValidateNodeDrainGracePeriod,
				},
			})
			if err != nil {
				return fmt.Errorf("Expected a valid value for Node drain grace period: %s", err)
			}
		}

		if nodeDrainGracePeriod != "" {
			nodeDrainBuilder, err := mpHelpers.CreateNodeDrainGracePeriodBuilder(nodeDrainGracePeriod)
			if err != nil {
				return fmt.Errorf(err.Error())
			}
			npBuilder.NodeDrainGracePeriod(nodeDrainBuilder)
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
	r *rosa.Runtime,
	nodePoolID string,
	existingReplicas int,
	existingAutoscaling *cmv1.NodePoolAutoscaling, isAnyAdditionalParameterSet bool) (autoscaling bool,
	replicas, minReplicas, maxReplicas int, err error) {

	isMinReplicasSet := cmd.Flags().Changed("min-replicas")
	isMaxReplicasSet := cmd.Flags().Changed("max-replicas")
	isReplicasSet := cmd.Flags().Changed("replicas")
	isAutoscalingSet := cmd.Flags().Changed("enable-autoscaling")

	replicas, err = cmd.Flags().GetInt("replicas")
	if err != nil {
		err = fmt.Errorf("Failed to get inputted replicas: %s", err)
		return
	}
	minReplicas, err = cmd.Flags().GetInt("min-replicas")
	if err != nil {
		err = fmt.Errorf("Failed to get inputted min replicas: %s", err)
		return
	}
	maxReplicas, err = cmd.Flags().GetInt("max-replicas")
	if err != nil {
		err = fmt.Errorf("Failed to get inputted max replicas: %s", err)
		return
	}
	autoscaling, err = cmd.Flags().GetBool("autoscaling")
	if err != nil {
		err = fmt.Errorf("Failed to get inputted autoscaling: %s", err)
		return
	}
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
					mpHelpers.MinNodePoolReplicaValidator(true),
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
					mpHelpers.MaxNodePoolReplicaValidator(minReplicas),
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
				mpHelpers.MinNodePoolReplicaValidator(false),
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
