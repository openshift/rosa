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

	fmt.Fprintf(writer, "ID\tAUTOSCALING\tREPLICAS\tINSTANCE TYPE\tAVAILABILITY ZONE\tSUBNET\tNODEPOOL\t\n")
	for _, nodePool := range nodePools {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t\n",
			nodePool.ID(),
			printNodePoolAutoscaling(nodePool.Autoscaling()),
			printNodePoolReplicas(nodePool.Autoscaling(), nodePool.Replicas()),
			printNodePoolInstanceType(nodePool.AWSNodePool()),
			nodePool.AvailabilityZone(),
			nodePool.Subnet(),
			printNodePoolName(nodePool.AWSNodePool()),
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

func printNodePoolName(aws *cmv1.AWSNodePool) string {
	if aws == nil || aws.Tags() == nil {
		return ""
	}
	tags := aws.Tags()
	if nodePoolName, ok := tags["api.openshift.com/nodepool"]; ok {
		return nodePoolName
	}
	return ""
}
