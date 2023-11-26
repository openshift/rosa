package machinepool

import (
	"bytes"
	"encoding/json"
	"fmt"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	ocmOutput "github.com/openshift/rosa/pkg/ocm/output"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

func describeNodePool(r *rosa.Runtime, cluster *cmv1.Cluster, clusterKey string, nodePoolID string) error {
	r.Reporter.Debugf("Fetching node pool '%s' for cluster '%s'", nodePoolID, clusterKey)
	nodePool, exists, err := r.OCMClient.GetNodePool(cluster.ID(), nodePoolID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("Machine pool '%s' not found", nodePoolID)
	}

	_, scheduledUpgrade, err := r.OCMClient.GetHypershiftNodePoolUpgrade(cluster.ID(), clusterKey, nodePoolID)
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

	// Prepare string
	nodePoolOutput := fmt.Sprintf("\n"+
		"ID:                         %s\n"+
		"Cluster ID:                 %s\n"+
		"Autoscaling:                %s\n"+
		"Desired replicas:           %s\n"+
		"Current replicas:           %s\n"+
		"Instance type:              %s\n"+
		"Labels:                     %s\n"+
		"Taints:                     %s\n"+
		"Availability zone:          %s\n"+
		"Subnet:                     %s\n"+
		"Version:                    %s\n"+
		"Autorepair:                 %s\n"+
		"Tuning configs:             %s\n"+
		"Message:                    %s\n",
		nodePool.ID(),
		cluster.ID(),
		ocmOutput.PrintNodePoolAutoscaling(nodePool.Autoscaling()),
		ocmOutput.PrintNodePoolReplicas(nodePool.Autoscaling(), nodePool.Replicas()),
		ocmOutput.PrintNodePoolCurrentReplicas(nodePool.Status()),
		ocmOutput.PrintNodePoolInstanceType(nodePool.AWSNodePool()),
		ocmOutput.PrintLabels(nodePool.Labels()),
		ocmOutput.PrintTaints(nodePool.Taints()),
		nodePool.AvailabilityZone(),
		nodePool.Subnet(),
		ocmOutput.PrintNodePoolVersion(nodePool.Version()),
		ocmOutput.PrintNodePoolAutorepair(nodePool.AutoRepair()),
		ocmOutput.PrintNodePoolTuningConfigs(nodePool.TuningConfigs()),
		ocmOutput.PrintNodePoolMessage(nodePool.Status()),
	)

	// Print scheduled upgrades if existing
	if scheduledUpgrade != nil {
		nodePoolOutput = fmt.Sprintf("%s"+
			"Scheduled upgrade:          %s %s on %s\n",
			nodePoolOutput,
			scheduledUpgrade.State().Value(),
			scheduledUpgrade.Version(),
			scheduledUpgrade.NextRun().Format("2006-01-02 15:04 MST"),
		)
	}
	fmt.Print(nodePoolOutput)

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
