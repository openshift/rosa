package machinepool

import (
	"fmt"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/helper"
	ocmOutput "github.com/openshift/rosa/pkg/ocm/output"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

func describeMachinePool(r *rosa.Runtime, cluster *cmv1.Cluster, clusterKey string, machinePoolID string) error {
	r.Reporter.Debugf("Fetching machine pool '%s' for cluster '%s'", machinePoolID, clusterKey)
	machinePool, exists, err := r.OCMClient.GetMachinePool(cluster.ID(), machinePoolID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("Machine pool '%s' not found", machinePoolID)
	}

	if output.HasFlag() {
		return output.Print(machinePool)
	}

	// Prepare string
	machinePoolOutput := fmt.Sprintf("\n"+
		"ID:                         %s\n"+
		"Cluster ID:                 %s\n"+
		"Autoscaling:                %s\n"+
		"Replicas:                   %s\n"+
		"Instance type:              %s\n"+
		"Labels:                     %s\n"+
		"Taints:                     %s\n"+
		"Availability zones:         %s\n"+
		"Subnets:                    %s\n"+
		"Spot instances:             %s\n"+
		"Disk size:                  %s\n"+
		"Security Group IDs:         %s\n",
		machinePool.ID(),
		cluster.ID(),
		ocmOutput.PrintMachinePoolAutoscaling(machinePool.Autoscaling()),
		ocmOutput.PrintMachinePoolReplicas(machinePool.Autoscaling(), machinePool.Replicas()),
		machinePool.InstanceType(),
		ocmOutput.PrintLabels(machinePool.Labels()),
		ocmOutput.PrintTaints(machinePool.Taints()),
		ocmOutput.PrintStringSlice(machinePool.AvailabilityZones()),
		ocmOutput.PrintStringSlice(machinePool.Subnets()),
		ocmOutput.PrintMachinePoolSpot(machinePool),
		ocmOutput.PrintMachinePoolDiskSize(machinePool),
		helper.SliceToSortedString(machinePool.AWS().AdditionalSecurityGroupIds()),
	)
	fmt.Print(machinePoolOutput)

	return nil
}
