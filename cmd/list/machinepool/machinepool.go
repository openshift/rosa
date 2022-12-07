package machinepool

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
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

	// Add default machine pool to the list
	defaultMachinePoolBuilder := cmv1.NewMachinePool().
		ID("Default").
		AvailabilityZones(cluster.Nodes().AvailabilityZones()...).
		InstanceType(cluster.Nodes().ComputeMachineType().ID()).
		Labels(cluster.Nodes().ComputeLabels()).
		Replicas(cluster.Nodes().Compute())
	if cluster.Nodes().AutoscaleCompute() != nil {
		defaultMachinePoolBuilder = defaultMachinePoolBuilder.Autoscaling(
			cmv1.NewMachinePoolAutoscaling().
				MinReplicas(cluster.Nodes().AutoscaleCompute().MinReplicas()).
				MaxReplicas(cluster.Nodes().AutoscaleCompute().MaxReplicas()),
		)
	}
	defaultMachinePool, _ := defaultMachinePoolBuilder.Build()

	machinePools = append([]*cmv1.MachinePool{defaultMachinePool}, machinePools...)

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
		"ID\tAUTOSCALING\tREPLICAS\tINSTANCE TYPE\tLABELS\t\tTAINTS\t\tAVAILABILITY ZONES\t\tSUBNETS\t\tSPOT INSTANCES\n")
	for _, machinePool := range machinePools {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t\t%s\t\t%s\t\t%s\t\t%s\n",
			machinePool.ID(),
			printMachinePoolAutoscaling(machinePool.Autoscaling()),
			printMachinePoolReplicas(machinePool.Autoscaling(), machinePool.Replicas()),
			machinePool.InstanceType(),
			printLabels(machinePool.Labels()),
			printTaints(machinePool.Taints()),
			printStringSlice(machinePool.AvailabilityZones()),
			printStringSlice(machinePool.Subnets()),
			printSpot(machinePool),
		)
	}
	writer.Flush()
}

func printMachinePoolAutoscaling(autoscaling *cmv1.MachinePoolAutoscaling) string {
	if autoscaling != nil {
		return "Yes"
	}
	return "No"
}

func printMachinePoolReplicas(autoscaling *cmv1.MachinePoolAutoscaling, replicas int) string {
	if autoscaling != nil {
		return fmt.Sprintf("%d-%d",
			autoscaling.MinReplicas(),
			autoscaling.MaxReplicas())
	}
	return fmt.Sprintf("%d", replicas)
}

func printLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	output := []string{}
	for k, v := range labels {
		output = append(output, fmt.Sprintf("%s=%s", k, v))
	}

	return strings.Join(output, ", ")
}

func printTaints(taints []*cmv1.Taint) string {
	if len(taints) == 0 {
		return ""
	}
	output := []string{}
	for _, taint := range taints {
		output = append(output, fmt.Sprintf("%s=%s:%s", taint.Key(), taint.Value(), taint.Effect()))
	}

	return strings.Join(output, ", ")
}

func printStringSlice(in []string) string {
	if len(in) == 0 {
		return ""
	}
	return strings.Join(in, ", ")
}

func printSpot(mp *cmv1.MachinePool) string {
	if mp.ID() == "Default" {
		return "N/A"
	}

	if mp.AWS() != nil {
		if spot := mp.AWS().SpotMarketOptions(); spot != nil {
			price := "on-demand"
			if maxPrice, ok := spot.GetMaxPrice(); ok {
				price = fmt.Sprintf("max $%g", maxPrice)
			}
			return fmt.Sprintf("Yes (%s)", price)
		}
	}
	return "No"
}
