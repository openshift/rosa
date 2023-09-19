package machinepool

import (
	"fmt"
	"os"
	"text/tabwriter"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/helper"
	ocmOutput "github.com/openshift/rosa/pkg/ocm/output"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

func listMachinePools(r *rosa.Runtime, clusterKey string, cluster *cmv1.Cluster) {
	// Load any existing machine pools for this cluster
	r.Reporter.Debugf("Loading machine pools for cluster '%s'", clusterKey)
	machinePools, err := r.OCMClient.GetMachinePools(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("Failed to get machine pools for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	if output.HasFlag() {
		err = output.Print(machinePools)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Create the writer that will be used to print the tabulated results:
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Fprintf(writer,
		"ID\tAUTOSCALING\tREPLICAS\tINSTANCE TYPE\tLABELS\t\tTAINTS\t"+
			"\tAVAILABILITY ZONES\t\tSUBNETS\t\tSPOT INSTANCES\tDISK SIZE\tSG IDs\n")
	for _, machinePool := range machinePools {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t\t%s\t\t%s\t\t%s\t\t%s\t%s\t%s\n",
			machinePool.ID(),
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
	}
	writer.Flush()
}
