package machinepool

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	ocmOutput "github.com/openshift/rosa/pkg/ocm/output"
	"github.com/openshift/rosa/pkg/output"
)

var policyBuilder cmv1.NodePoolUpgradePolicyBuilder
var date time.Time

var _ = Describe("Machinepool and nodepool", func() {
	Context("Nodepools", Ordered, func() {
		BeforeAll(func() {
			location, err := time.LoadLocation("America/New_York")
			Expect(err).ToNot(HaveOccurred())
			date = time.Date(2024, time.April, 2, 2, 2, 0, 0, location)
			policyBuilder = *cmv1.NewNodePoolUpgradePolicy().ID("test-policy").Version("1").
				ClusterID("test-cluster").State(cmv1.NewUpgradePolicyState().ID("test-state").
				Value(cmv1.UpgradePolicyStateValueScheduled)).
				NextRun(date)
		})
		It("Test printNodePools", func() {
			clusterBuilder := cmv1.NewCluster().ID("test").State(cmv1.ClusterStateReady).
				Hypershift(cmv1.NewHypershift().Enabled(true)).NodePools(cmv1.NewNodePoolList().
				Items(cmv1.NewNodePool().ID("np").Replicas(8).AvailabilityZone("az").
					Subnet("sn").Version(cmv1.NewVersion().ID("1")).AutoRepair(false)))
			cluster, err := clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			Expect(err).ToNot(HaveOccurred())
			out := printNodePools(cluster.NodePools().Slice())
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal(fmt.Sprintf("ID\tAUTOSCALING\tREPLICAS\t"+
				"INSTANCE TYPE\tLABELS\t\tTAINTS\t\tAVAILABILITY ZONE\tSUBNET\tVERSION\tAUTOREPAIR\t\n"+
				"%s\t%s\t%s\t%s\t%s\t\t%s\t\t%s\t%s\t%s\t%s\t\n",
				cluster.NodePools().Get(0).ID(),
				ocmOutput.PrintNodePoolAutoscaling(cluster.NodePools().Get(0).Autoscaling()),
				ocmOutput.PrintNodePoolReplicasShort(
					ocmOutput.PrintNodePoolCurrentReplicas(cluster.NodePools().Get(0).Status()),
					ocmOutput.PrintNodePoolReplicas(cluster.NodePools().Get(0).Autoscaling(),
						cluster.NodePools().Get(0).Replicas()),
				),
				ocmOutput.PrintNodePoolInstanceType(cluster.NodePools().Get(0).AWSNodePool()),
				ocmOutput.PrintLabels(cluster.NodePools().Get(0).Labels()),
				ocmOutput.PrintTaints(cluster.NodePools().Get(0).Taints()),
				cluster.NodePools().Get(0).AvailabilityZone(),
				cluster.NodePools().Get(0).Subnet(),
				ocmOutput.PrintNodePoolVersion(cluster.NodePools().Get(0).Version()),
				ocmOutput.PrintNodePoolAutorepair(cluster.NodePools().Get(0).AutoRepair()))))
		})
		It("Test appendUpgradesIfExist", func() {
			policy, err := policyBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			out := appendUpgradesIfExist(policy, "test\n")
			Expect(out).To(Equal(fmt.Sprintf("test\nScheduled upgrade:                     %s %s on %s\n",
				cmv1.UpgradePolicyStateValueScheduled, "1", date.Format("2006-01-02 15:04 MST"))))
		})
		It("Test appendUpgradesIfExist nil schedule", func() {
			out := appendUpgradesIfExist(nil, "test\n")
			Expect(out).To(Equal("test\n"))
		})
		It("Test func formatNodePoolOutput", func() {
			policy, err := policyBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			nodePool, err := cmv1.NewNodePool().ID("test-np").Version(cmv1.NewVersion().ID("1")).
				Subnet("test-subnet").Replicas(4).AutoRepair(true).Build()
			Expect(err).ToNot(HaveOccurred())

			out, err := formatNodePoolOutput(nodePool, policy)
			Expect(err).ToNot(HaveOccurred())
			expectedOutput := make(map[string]interface{})
			upgrade := make(map[string]interface{})
			upgrade["version"] = policy.Version()
			upgrade["state"] = policy.State().Value()
			upgrade["nextRun"] = policy.NextRun().Format("2006-01-02 15:04 MST")
			expectedOutput["subnet"] = "test-subnet"

			expectedOutput["kind"] = "NodePool"
			expectedOutput["id"] = "test-np"
			expectedOutput["replicas"] = 4.0
			version := make(map[string]interface{})
			version["kind"] = "Version"
			version["id"] = "1"
			expectedOutput["auto_repair"] = true
			expectedOutput["version"] = version
			expectedOutput["scheduledUpgrade"] = upgrade
			fmt.Println(out)
			Expect(fmt.Sprint(out)).To(Equal(fmt.Sprint(expectedOutput)))
		})
	})
	Context("MachinePools", func() {
		It("Test printMachinePools", func() {
			clusterBuilder := cmv1.NewCluster().ID("test").State(cmv1.ClusterStateReady).
				MachinePools(cmv1.NewMachinePoolList().
					Items(cmv1.NewMachinePool().ID("np").Replicas(8).Subnets("sn1", "sn2").
						InstanceType("test instance type").Taints(cmv1.NewTaint().Value("test").
						Key("taint"))))
			cluster, err := clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			Expect(err).ToNot(HaveOccurred())
			out := printMachinePools(cluster.MachinePools().Slice())
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal(fmt.Sprintf("ID\tAUTOSCALING\tREPLICAS\tINSTANCE TYPE\tLABELS\t\tTAINTS\t"+
				"\tAVAILABILITY ZONES\t\tSUBNETS\t\tSPOT INSTANCES\tDISK SIZE\tSG IDs\n"+
				"%s\t%s\t%s\t%s\t%s\t\t%s\t\t%s\t\t%s\t\t%s\t%s\t%s\n",
				cluster.MachinePools().Get(0).ID(),
				ocmOutput.PrintMachinePoolAutoscaling(cluster.MachinePools().Get(0).Autoscaling()),
				ocmOutput.PrintMachinePoolReplicas(cluster.MachinePools().Get(0).Autoscaling(),
					cluster.MachinePools().Get(0).Replicas()),
				cluster.MachinePools().Get(0).InstanceType(),
				ocmOutput.PrintLabels(cluster.MachinePools().Get(0).Labels()),
				ocmOutput.PrintTaints(cluster.MachinePools().Get(0).Taints()),
				output.PrintStringSlice(cluster.MachinePools().Get(0).AvailabilityZones()),
				output.PrintStringSlice(cluster.MachinePools().Get(0).Subnets()),
				ocmOutput.PrintMachinePoolSpot(cluster.MachinePools().Get(0)),
				ocmOutput.PrintMachinePoolDiskSize(cluster.MachinePools().Get(0)),
				output.PrintStringSlice(cluster.MachinePools().Get(0).AWS().AdditionalSecurityGroupIds()))))
		})
	})
})
