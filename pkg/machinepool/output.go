package machinepool

import (
	"fmt"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	ocmOutput "github.com/openshift/rosa/pkg/ocm/output"
	"github.com/openshift/rosa/pkg/output"
)

var nodePoolOutputString string = "\n" +
	"ID:                                    %s\n" +
	"Cluster ID:                            %s\n" +
	"Autoscaling:                           %s\n" +
	"Desired replicas:                      %s\n" +
	"Current replicas:                      %s\n" +
	"Instance type:                         %s\n" +
	"Labels:                                %s\n" +
	"Tags:                                  %s\n" +
	"Taints:                                %s\n" +
	"Availability zone:                     %s\n" +
	"Subnet:                                %s\n" +
	"Version:                               %s\n" +
	"Autorepair:                            %s\n" +
	"Tuning configs:                        %s\n" +
	"Kubelet configs:                       %s\n" +
	"Additional security group IDs:         %s\n" +
	"Node drain grace period:               %s\n" +
	"Management upgrade:                    %s\n" +
	"Message:                               %s\n"

var machinePoolOutputString = "\n" +
	"ID:                                    %s\n" +
	"Cluster ID:                            %s\n" +
	"Autoscaling:                           %s\n" +
	"Replicas:                              %s\n" +
	"Instance type:                         %s\n" +
	"Labels:                                %s\n" +
	"Taints:                                %s\n" +
	"Availability zones:                    %s\n" +
	"Subnets:                               %s\n" +
	"Spot instances:                        %s\n" +
	"Disk size:                             %s\n" +
	"Additional Security Group IDs:         %s\n" +
	"Tags:                                  %s\n"

func machinePoolOutput(clusterId string, machinePool *cmv1.MachinePool) string {
	return fmt.Sprintf(machinePoolOutputString,
		machinePool.ID(),
		clusterId,
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
		ocmOutput.PrintUserAwsTags(machinePool.AWS().Tags()),
	)
}

func nodePoolOutput(clusterId string, nodePool *cmv1.NodePool) string {
	return fmt.Sprintf(nodePoolOutputString,
		nodePool.ID(),
		clusterId,
		ocmOutput.PrintNodePoolAutoscaling(nodePool.Autoscaling()),
		ocmOutput.PrintNodePoolReplicas(nodePool.Autoscaling(), nodePool.Replicas()),
		ocmOutput.PrintNodePoolCurrentReplicas(nodePool.Status()),
		ocmOutput.PrintNodePoolInstanceType(nodePool.AWSNodePool()),
		ocmOutput.PrintLabels(nodePool.Labels()),
		ocmOutput.PrintUserAwsTags(nodePool.AWSNodePool().Tags()),
		ocmOutput.PrintTaints(nodePool.Taints()),
		nodePool.AvailabilityZone(),
		nodePool.Subnet(),
		ocmOutput.PrintNodePoolVersion(nodePool.Version()),
		ocmOutput.PrintNodePoolAutorepair(nodePool.AutoRepair()),
		ocmOutput.PrintNodePoolConfigs(nodePool.TuningConfigs()),
		ocmOutput.PrintNodePoolConfigs(nodePool.KubeletConfigs()),
		ocmOutput.PrintNodePoolAdditionalSecurityGroups(nodePool.AWSNodePool()),
		ocmOutput.PrintNodeDrainGracePeriod(nodePool.NodeDrainGracePeriod()),
		ocmOutput.PrintNodePoolManagementUpgrade(nodePool.ManagementUpgrade()),
		ocmOutput.PrintNodePoolMessage(nodePool.Status()),
	)
}
