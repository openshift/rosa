package machinepool

import (
	"fmt"
	"os"
	"text/tabwriter"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/rosa"
)

func listNodePools(r *rosa.Runtime, clusterKey string, cluster *cmv1.Cluster) {
	// Load any existing machine pools for this cluster
	r.Reporter.Debugf("Loading machine pools for cluster '%s'", clusterKey)
	nodePools, err := r.OCMClient.GetNodePools(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("Failed to get machine pools for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	// Create the writer that will be used to print the tabulated results:
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Fprintf(writer, "ID\tAUTOSCALING\tDESIRED REPLICAS\tCURRENT REPLICAS\t"+
		"INSTANCE TYPE\tAVAILABILITY ZONE\tSUBNET\tMESSAGE\t\n")
	for _, nodePool := range nodePools {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t\n",
			nodePool.ID(),
			printNodePoolAutoscaling(nodePool.Autoscaling()),
			printNodePoolReplicas(nodePool.Autoscaling(), nodePool.Replicas()),
			printNodePoolCurrentReplicas(nodePool.Status()),
			printNodePoolInstanceType(nodePool.AWSNodePool()),
			nodePool.AvailabilityZone(),
			nodePool.Subnet(),
			printNodePoolMessage(nodePool.Status()),
		)
	}
	writer.Flush()
}

func printNodePoolAutoscaling(autoscaling *cmv1.NodePoolAutoscaling) string {
	if autoscaling != nil {
		return "Yes"
	}
	return "No"
}

func printNodePoolReplicas(autoscaling *cmv1.NodePoolAutoscaling, replicas int) string {
	if autoscaling != nil {
		return fmt.Sprintf("%d-%d",
			autoscaling.MinReplica(),
			autoscaling.MaxReplica())
	}
	return fmt.Sprintf("%d", replicas)
}

func printNodePoolInstanceType(aws *cmv1.AWSNodePool) string {
	if aws == nil {
		return ""
	}
	return aws.InstanceType()
}

func printNodePoolCurrentReplicas(status *cmv1.NodePoolStatus) string {
	if status != nil {
		return fmt.Sprintf("%d", status.CurrentReplicas())
	}
	return ""
}

func printNodePoolMessage(status *cmv1.NodePoolStatus) string {
	if status != nil {
		return status.Message()
	}
	return ""
}
