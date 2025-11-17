package machinepool

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	ocmOutput "github.com/openshift/rosa/pkg/ocm/output"
)

var taintsBuilder *cmv1.TaintBuilder
var labels map[string]string
var taint *cmv1.Taint

var _ = Describe("Output", Ordered, func() {
	Context("Test output for machinepools and nodepools", func() {
		BeforeAll(func() {
			var err error
			taintsBuilder = cmv1.NewTaint().Value("test-taint").Value("test-value")
			taint, err = taintsBuilder.Build()
			labels = map[string]string{"test": "test"}
			Expect(err).ToNot(HaveOccurred())
		})
		It("machinepool output with autoscaling", func() {
			machinePoolBuilder := *cmv1.NewMachinePool().ID("test-mp").Autoscaling(cmv1.NewMachinePoolAutoscaling().
				ID("test-as")).Replicas(4).InstanceType("test-it").
				Labels(labels).Taints(taintsBuilder).AvailabilityZones("test-az").
				Subnets("test-subnet")
			machinePool, err := machinePoolBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			labelsOutput := ocmOutput.PrintLabels(labels)
			taintsOutput := ocmOutput.PrintTaints([]*cmv1.Taint{taint})

			out := fmt.Sprintf(machinePoolOutputString,
				"test-mp", "test-cluster", "Yes", "0-0", "test-it", labelsOutput, taintsOutput,
				"test-az", "test-subnet", ocmOutput.PrintMachinePoolSpot(machinePool),
				ocmOutput.PrintMachinePoolDiskSize(machinePool), "", "")

			result := machinePoolOutput("test-cluster", machinePool)
			Expect(out).To(Equal(result))
		})
		It("machinepool output with additional security groups", func() {
			awsMachinePoolBuilder := cmv1.NewAWSMachinePool().AdditionalSecurityGroupIds("123")
			machinePoolBuilder := *cmv1.NewMachinePool().ID("test-mp").Autoscaling(cmv1.NewMachinePoolAutoscaling().
				ID("test-as")).Replicas(4).InstanceType("test-it").
				Labels(labels).Taints(taintsBuilder).AvailabilityZones("test-az").
				Subnets("test-subnet").
				AWS(awsMachinePoolBuilder)
			machinePool, err := machinePoolBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			labelsOutput := ocmOutput.PrintLabels(labels)
			taintsOutput := ocmOutput.PrintTaints([]*cmv1.Taint{taint})

			out := fmt.Sprintf(machinePoolOutputString,
				"test-mp", "test-cluster", "Yes", "0-0", "test-it", labelsOutput, taintsOutput,
				"test-az", "test-subnet", ocmOutput.PrintMachinePoolSpot(machinePool),
				ocmOutput.PrintMachinePoolDiskSize(machinePool), "123", "")

			result := machinePoolOutput("test-cluster", machinePool)
			Expect(out).To(Equal(result))
		})
		It("machinepool output with aws tags", func() {
			awsMachinePoolBuilder := cmv1.NewAWSMachinePool().Tags(map[string]string{
				"test-tag": "test-value",
			})
			machinePoolBuilder := *cmv1.NewMachinePool().ID("test-mp").Autoscaling(cmv1.NewMachinePoolAutoscaling().
				ID("test-as")).Replicas(4).InstanceType("test-it").
				Labels(labels).Taints(taintsBuilder).AvailabilityZones("test-az").
				Subnets("test-subnet").
				AWS(awsMachinePoolBuilder)
			machinePool, err := machinePoolBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			labelsOutput := ocmOutput.PrintLabels(labels)
			taintsOutput := ocmOutput.PrintTaints([]*cmv1.Taint{taint})

			out := fmt.Sprintf(machinePoolOutputString,
				"test-mp", "test-cluster", "Yes", "0-0", "test-it", labelsOutput, taintsOutput,
				"test-az", "test-subnet", ocmOutput.PrintMachinePoolSpot(machinePool),
				ocmOutput.PrintMachinePoolDiskSize(machinePool), "", "test-tag=test-value")

			result := machinePoolOutput("test-cluster", machinePool)
			Expect(out).To(Equal(result))
		})
		It("machinepool output without autoscaling", func() {
			machinePoolBuilder := *cmv1.NewMachinePool().ID("test-mp2").
				Replicas(4).InstanceType("test-it2").Labels(labels).
				Taints(taintsBuilder).AvailabilityZones("test-az2").Subnets("test-subnet2")
			machinePool, err := machinePoolBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			labelsOutput := ocmOutput.PrintLabels(labels)
			taintsOutput := ocmOutput.PrintTaints([]*cmv1.Taint{taint})

			out := fmt.Sprintf(machinePoolOutputString,
				"test-mp2", "test-cluster", "No", "4", "test-it2", labelsOutput, taintsOutput,
				"test-az2", "test-subnet2", ocmOutput.PrintMachinePoolSpot(machinePool),
				ocmOutput.PrintMachinePoolDiskSize(machinePool), "", "")

			result := machinePoolOutput("test-cluster", machinePool)
			Expect(out).To(Equal(result))
		})
		It("nodepool output with autoscaling", func() {
			awsNodePoolBuilder := cmv1.NewAWSNodePool().RootVolume(cmv1.NewAWSVolume().Size(300))
			npAutoscaling := cmv1.NewNodePoolAutoscaling().ID("test-as").MinReplica(2).MaxReplica(8)
			mgmtUpgradeBuilder := cmv1.NewNodePoolManagementUpgrade().MaxSurge("1").MaxUnavailable("0")
			nodePoolBuilder := *cmv1.NewNodePool().ID("test-mp").Autoscaling(npAutoscaling).Replicas(4).
				AWSNodePool(awsNodePoolBuilder).
				AvailabilityZone("test-az").Subnet("test-subnets").Version(cmv1.NewVersion().
				ID("1")).AutoRepair(false).TuningConfigs("test-tc").
				KubeletConfigs("test-kc").Labels(labels).Taints(taintsBuilder).
				ManagementUpgrade(mgmtUpgradeBuilder)
			nodePool, err := nodePoolBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			labelsOutput := ocmOutput.PrintLabels(labels)
			taintsOutput := ocmOutput.PrintTaints([]*cmv1.Taint{taint})
			replicasOutput := ocmOutput.PrintNodePoolReplicas((*cmv1.NodePoolAutoscaling)(npAutoscaling), 4)
			mgmtUpgrade, err := mgmtUpgradeBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			managementUpgradeOutput := ocmOutput.PrintNodePoolManagementUpgrade(mgmtUpgrade)

			out := fmt.Sprintf(nodePoolOutputString,
				"test-mp", "test-cluster", "Yes", replicasOutput, "", "", "", labelsOutput, "", taintsOutput,
				"test-az", "test-subnets", "300 GiB", "1", "optional", "No", "test-tc", "test-kc", "", "", "",
				managementUpgradeOutput, "")

			result := nodePoolOutput("test-cluster", nodePool)
			Expect(out).To(Equal(result))
		})
		It("nodepool output without autoscaling", func() {
			awsNodePoolBuilder := cmv1.NewAWSNodePool().RootVolume(cmv1.NewAWSVolume().Size(300))
			nodePoolBuilder := *cmv1.NewNodePool().ID("test-mp").Replicas(4).
				AWSNodePool(awsNodePoolBuilder).
				AvailabilityZone("test-az").Subnet("test-subnets").Version(cmv1.NewVersion().
				ID("1")).AutoRepair(false).TuningConfigs("test-tc").
				KubeletConfigs("test-kc").Labels(labels).Taints(taintsBuilder)
			nodePool, err := nodePoolBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			labelsOutput := ocmOutput.PrintLabels(labels)
			taintsOutput := ocmOutput.PrintTaints([]*cmv1.Taint{taint})

			out := fmt.Sprintf(nodePoolOutputString,
				"test-mp", "test-cluster", "No", "4", "", "", "", labelsOutput, "", taintsOutput, "test-az",
				"test-subnets", "300 GiB", "1", "optional", "No", "test-tc", "test-kc", "", "", "", "", "")

			result := nodePoolOutput("test-cluster", nodePool)
			Expect(out).To(Equal(result))
		})
		It("nodepool output with custom disk size", func() {
			awsNodePoolBuilder := cmv1.NewAWSNodePool().RootVolume(cmv1.NewAWSVolume().Size(256))
			nodePoolBuilder := cmv1.NewNodePool().ID("test-mp").Replicas(4).AWSNodePool(awsNodePoolBuilder).
				AvailabilityZone("test-az").Subnet("test-subnets").Version(cmv1.NewVersion().
				ID("1")).AutoRepair(false).TuningConfigs("test-tc").
				KubeletConfigs("test-kc").Labels(labels).Taints(taintsBuilder)
			nodePool, err := nodePoolBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			labelsOutput := ocmOutput.PrintLabels(labels)
			taintsOutput := ocmOutput.PrintTaints([]*cmv1.Taint{taint})

			out := fmt.Sprintf(nodePoolOutputString,
				"test-mp", "test-cluster", "No", "4", "", "", "", labelsOutput, "", taintsOutput, "test-az",
				"test-subnets", "256 GiB", "1", "optional", "No", "test-tc", "test-kc", "", "", "", "", "")

			result := nodePoolOutput("test-cluster", nodePool)
			Expect(out).To(Equal(result))
		})
		It("nodepool output with capacity reservation id", func() {
			awsNodePoolBuilder := cmv1.NewAWSNodePool().RootVolume(cmv1.NewAWSVolume().Size(256)).
				CapacityReservation(cmv1.NewAWSCapacityReservation().Id("test-id").
					MarketType(cmv1.MarketTypeOnDemand))

			nodePoolBuilder := cmv1.NewNodePool().ID("test-mp").Replicas(4).AWSNodePool(awsNodePoolBuilder).
				AvailabilityZone("test-az").Subnet("test-subnets").Version(cmv1.NewVersion().
				ID("1")).AutoRepair(false).TuningConfigs("test-tc").
				KubeletConfigs("test-kc").Labels(labels).Taints(taintsBuilder)
			nodePool, err := nodePoolBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			labelsOutput := ocmOutput.PrintLabels(labels)
			taintsOutput := ocmOutput.PrintTaints([]*cmv1.Taint{taint})

			out := fmt.Sprintf(nodePoolOutputString,
				"test-mp", "test-cluster", "No", "4", "", "", "", labelsOutput, "", taintsOutput, "test-az",
				"test-subnets", "256 GiB", "1", "optional", "No", "test-tc", "test-kc", "", "",
				"\n - ID:                                 test-id\n - Type:                               OnDemand",
				"", "")

			result := nodePoolOutput("test-cluster", nodePool)
			Expect(out).To(Equal(result))
		})
	})
})
