package machinepool

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	ocmOutput "github.com/openshift/rosa/pkg/ocm/output"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var fetchMessage string = "Fetching %s '%s' for cluster '%s'"
var notFoundMessage string = "Machine pool '%s' not found"

//go:generate mockgen -source=machinepool.go -package=mocks -destination=machinepool_mock.go
type MachinePoolService interface {
	DescribeMachinePool(r *rosa.Runtime, cluster *cmv1.Cluster, clusterKey string, machinePoolId string) error
	ListMachinePools(r *rosa.Runtime, clusterKey string, cluster *cmv1.Cluster) error
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
