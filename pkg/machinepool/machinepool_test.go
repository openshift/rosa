package machinepool

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/interactive"
	ocmOutput "github.com/openshift/rosa/pkg/ocm/output"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
	. "github.com/openshift/rosa/pkg/test"
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
		It("editAutoscaling should equal nil if nothing is changed", func() {
			nodepool, err := cmv1.NewNodePool().
				Autoscaling(cmv1.NewNodePoolAutoscaling().MaxReplica(2).MinReplica(1)).
				Build()
			Expect(err).ToNot(HaveOccurred())
			builder := editAutoscaling(nodepool, 1, 2)
			Expect(builder).To(BeNil())
		})
		It("editAutoscaling should equal the exepcted output", func() {
			nodepool, err := cmv1.NewNodePool().
				Autoscaling(cmv1.NewNodePoolAutoscaling().MaxReplica(2).MinReplica(1)).
				Build()
			Expect(err).ToNot(HaveOccurred())
			builder := editAutoscaling(nodepool, 2, 3)
			asBuilder := cmv1.NewNodePoolAutoscaling().MaxReplica(3).MinReplica(2)
			Expect(builder).To(Equal(asBuilder))
		})
		It("editAutoscaling should equal the exepcted output with no min replica value", func() {
			nodepool, err := cmv1.NewNodePool().
				Autoscaling(cmv1.NewNodePoolAutoscaling().MaxReplica(2).MinReplica(1)).
				Build()
			Expect(err).ToNot(HaveOccurred())
			builder := editAutoscaling(nodepool, 0, 3)
			asBuilder := cmv1.NewNodePoolAutoscaling().MaxReplica(3).MinReplica(1)
			Expect(builder).To(Equal(asBuilder))
		})

		It("editAutoscaling should equal the exepcted output with no max replica value", func() {
			nodepool, err := cmv1.NewNodePool().
				Autoscaling(cmv1.NewNodePoolAutoscaling().MaxReplica(4).MinReplica(1)).
				Build()
			Expect(err).ToNot(HaveOccurred())
			builder := editAutoscaling(nodepool, 2, 0)
			asBuilder := cmv1.NewNodePoolAutoscaling().MaxReplica(4).MinReplica(2)
			Expect(builder).To(Equal(asBuilder))
		})
		Context("Prompt For NodePoolNodeRecreate", func() {
			var mockPromptFuncInvoked bool
			var t *TestingRuntime
			BeforeEach(func() {
				t = NewTestRuntime()
				mockPromptFuncInvoked = false
			})

			invoked := func(r *rosa.Runtime) bool {
				mockPromptFuncInvoked = true
				return mockPromptFuncInvoked
			}

			It("Prompts when the user has deleted a kubelet-config", func() {

				original := MockNodePool(func(n *cmv1.NodePoolBuilder) {
					n.KubeletConfigs("test")
				})

				update := MockNodePool(func(n *cmv1.NodePoolBuilder) {
					n.KubeletConfigs("")
				})

				Expect(promptForNodePoolNodeRecreate(original, update, invoked, t.RosaRuntime)).To(BeTrue())
				Expect(mockPromptFuncInvoked).To(BeTrue())
			})

			It("Prompts when the user has changed a kubelet-config", func() {

				original := MockNodePool(func(n *cmv1.NodePoolBuilder) {
					n.KubeletConfigs("test")
				})

				update := MockNodePool(func(n *cmv1.NodePoolBuilder) {
					n.KubeletConfigs("bar")
				})

				Expect(promptForNodePoolNodeRecreate(original, update, invoked, t.RosaRuntime)).To(BeTrue())
				Expect(mockPromptFuncInvoked).To(BeTrue())
			})

			It("Does not prompts when the user has not changed a kubelet-config", func() {
				original := MockNodePool(func(n *cmv1.NodePoolBuilder) {
					n.KubeletConfigs("test")
				})

				update := MockNodePool(func(n *cmv1.NodePoolBuilder) {
					n.KubeletConfigs("test")
				})

				Expect(promptForNodePoolNodeRecreate(original, update, invoked, t.RosaRuntime)).To(BeTrue())
				Expect(mockPromptFuncInvoked).To(BeFalse())
			})
		})
		It("Test printNodePools", func() {
			clusterBuilder := cmv1.NewCluster().ID("test").State(cmv1.ClusterStateReady).
				Hypershift(cmv1.NewHypershift().Enabled(true)).NodePools(cmv1.NewNodePoolList().
				Items(cmv1.NewNodePool().ID("np").Replicas(8).AvailabilityZone("az").
					Subnet("sn").Version(cmv1.NewVersion().ID("1")).AutoRepair(false)))
			cluster, err := clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			out := getNodePoolsString(cluster.NodePools().Slice())
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
		Context("fillAutoScalingAndReplicas", func() {
			var npBuilder *cmv1.NodePoolBuilder
			existingNodepool, err := cmv1.NewNodePool().
				Autoscaling(cmv1.NewNodePoolAutoscaling().MaxReplica(4).MinReplica(1)).
				Build()
			Expect(err).ToNot(HaveOccurred())
			It("Autoscaling set", func() {
				npBuilder = cmv1.NewNodePool()
				fillAutoScalingAndReplicas(npBuilder, true, existingNodepool, 1, 3, 2)
				npPatch, err := npBuilder.Build()
				Expect(err).ToNot(HaveOccurred())
				Expect(npPatch.Autoscaling()).ToNot(BeNil())
				// Default (zero) value
				Expect(npPatch.Replicas()).To(Equal(0))
			})
			It("Replicas set", func() {
				npBuilder = cmv1.NewNodePool()
				fillAutoScalingAndReplicas(npBuilder, false, existingNodepool, 0, 0, 2)
				npPatch, err := npBuilder.Build()
				Expect(err).ToNot(HaveOccurred())
				Expect(npPatch.Autoscaling()).To(BeNil())
				Expect(npPatch.Replicas()).To(Equal(2))
			})

		})
		Describe("Validate management upgrade print output", func() {
			mgmtUpgrade, _ := cmv1.NewNodePoolManagementUpgrade().MaxSurge("10").MaxUnavailable("5").Type("Replace").Build()
			DescribeTable("Validate management upgrade print output",
				func(upgrade *cmv1.NodePoolManagementUpgrade, expectedOutput string) {
					output := ocmOutput.PrintNodePoolManagementUpgrade(upgrade)
					Expect(output).To(Equal(expectedOutput))
				},
				Entry("Should return empty string", nil,
					"",
				),
				Entry("Should return string with type, maxSurge and maxUnavailable",
					mgmtUpgrade,
					fmt.Sprintf("\n - Type:%38s\n - Max surge:%28s\n - Max unavailable:%21s", "Replace", "10", "5"),
				),
			)
		})
	})
	Context("MachinePools", func() {
		Context("editMachinePoolAutoscaling", func() {
			It("editMachinePoolAutoscaling should equal nil if nothing is changed", func() {
				machinepool, err := cmv1.NewMachinePool().
					Autoscaling(cmv1.NewMachinePoolAutoscaling().MaxReplicas(2).MinReplicas(1)).
					Build()
				Expect(err).ToNot(HaveOccurred())
				builder := editMachinePoolAutoscaling(machinepool, 1, 2)
				Expect(builder).To(BeNil())
			})

			It("editMachinePoolAutoscaling should equal the exepcted output", func() {
				machinePool, err := cmv1.NewMachinePool().
					Autoscaling(cmv1.NewMachinePoolAutoscaling().MaxReplicas(2).MinReplicas(1)).
					Build()
				Expect(err).ToNot(HaveOccurred())
				builder := editMachinePoolAutoscaling(machinePool, 2, 3)
				asBuilder := cmv1.NewMachinePoolAutoscaling().MaxReplicas(3).MinReplicas(2)
				Expect(builder).To(Equal(asBuilder))
			})

			It("editMachinePoolAutoscaling should allow 0 min replicas", func() {
				machinePool, err := cmv1.NewMachinePool().
					Autoscaling(cmv1.NewMachinePoolAutoscaling().MaxReplicas(2).MinReplicas(1)).
					Build()
				Expect(err).ToNot(HaveOccurred())
				builder := editMachinePoolAutoscaling(machinePool, 0, 2)
				asBuilder := cmv1.NewMachinePoolAutoscaling().MaxReplicas(2).MinReplicas(0)
				Expect(builder).To(Equal(asBuilder))
			})
		})

		Context("isMultiAZMachinePool", func() {
			It("isMultiAZMachinePool should return true", func() {
				machinePool, err := cmv1.NewMachinePool().Build()
				Expect(err).ToNot(HaveOccurred())
				boolean := isMultiAZMachinePool(machinePool)
				Expect(boolean).To(BeTrue())
			})

			It("isMultiAZMachinePool should return false", func() {
				machinePool, err := cmv1.NewMachinePool().AvailabilityZones("test").Build()
				Expect(err).ToNot(HaveOccurred())
				boolean := isMultiAZMachinePool(machinePool)
				Expect(boolean).To(BeFalse())
			})
		})
		It("Test printMachinePools", func() {
			clusterBuilder := cmv1.NewCluster().ID("test").State(cmv1.ClusterStateReady).
				MachinePools(cmv1.NewMachinePoolList().
					Items(cmv1.NewMachinePool().ID("np").Replicas(8).Subnets("sn1", "sn2").
						InstanceType("test instance type").Taints(cmv1.NewTaint().Value("test").
						Key("taint"))))
			cluster, err := clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			out := getMachinePoolsString(cluster.MachinePools().Slice())
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
		It("Validate invalid regex", func() {
			Expect(MachinePoolKeyRE.MatchString("$%%$%$%^$%^$%^$%^")).To(BeFalse())
			Expect(MachinePoolKeyRE.MatchString("machinepool1")).To(BeTrue())
			Expect(MachinePoolKeyRE.MatchString("1machinepool")).To(BeFalse())
			Expect(MachinePoolKeyRE.MatchString("#1machinepool")).To(BeFalse())
			Expect(MachinePoolKeyRE.MatchString("m123123123123123123123123123")).To(BeTrue())
			Expect(MachinePoolKeyRE.MatchString("m#123")).To(BeFalse())
		})
	})

	Describe("Testing getMachinePoolAvailabilityZones function", func() {
		var (
			r                         *rosa.Runtime
			cluster                   *cmv1.Cluster
			availabilityZoneUserInput string
			subnetUserInput           string
		)

		BeforeEach(func() {
			r = &rosa.Runtime{}
			var err error
			clusterBuilder := cmv1.NewCluster().ID("test").State(cmv1.ClusterStateReady).
				MultiAZ(true).Nodes(cmv1.NewClusterNodes().
				AvailabilityZones("us-east-1a", "us-east-1b"))
			cluster, err = clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			Expect(cluster.MultiAZ()).To(Equal(true))

			availabilityZoneUserInput = "us-east-1a"
			subnetUserInput = ""
		})

		Context("When the machine pool is not multi-AZ", func() {
			It("Should only include the specified availability zone", func() {
				multiAZMachinePool := false
				azs, err := getMachinePoolAvailabilityZones(r, cluster,
					multiAZMachinePool, availabilityZoneUserInput, subnetUserInput)
				Expect(err).ToNot(HaveOccurred())
				Expect(azs).To(Equal([]string{"us-east-1a"}))
			})
		})

		Context("When the machine pool is multi-AZ", func() {
			When("No specific availability zone is preferred", func() {
				It("Should include all available zones for the cluster", func() {
					multiAZMachinePool := true
					azs, err := getMachinePoolAvailabilityZones(r, cluster, multiAZMachinePool, "", subnetUserInput)
					Expect(err).ToNot(HaveOccurred())
					Expect(azs).To(Equal([]string{"us-east-1a", "us-east-1b"}))
				})
			})

			When("A specific availability zone is preferred", func() {
				It("Should still include all available zones for the cluster", func() {
					multiAZMachinePool := true
					azs, err := getMachinePoolAvailabilityZones(r, cluster,
						multiAZMachinePool, availabilityZoneUserInput, subnetUserInput)
					Expect(err).ToNot(HaveOccurred())
					Expect(azs).To(Equal([]string{"us-east-1a", "us-east-1b"}))
				})
			})
		})
	})
})

var _ = Describe("Utility Functions", func() {
	Describe("Split function", func() {
		When("input is '=' rune", func() {
			It("should return true", func() {
				Expect(Split('=')).To(BeTrue())
			})
		})

		When("input is ':' rune", func() {
			It("should return true", func() {
				Expect(Split(':')).To(BeTrue())
			})
		})

		When("input is any other rune", func() {
			It("should return false", func() {
				Expect(Split('a')).To(BeFalse())
			})
		})
	})

	Describe("minReplicaValidator function", func() {
		var validator interactive.Validator

		BeforeEach(func() {
			validator = minReplicaValidator(true) // or false for non-multiAZ
		})

		When("input is non-integer", func() {
			It("should return error", func() {
				err := validator("non-integer")
				Expect(err).To(HaveOccurred())
			})
		})

		When("input is a negative integer", func() {
			It("should return error", func() {
				err := validator(-1)
				Expect(err).To(HaveOccurred())
			})
		})

		When("input is not a multiple of 3 for multiAZ", func() {
			It("should return error", func() {
				err := validator(2)
				Expect(err).To(HaveOccurred())
			})
		})

		When("input is a valid integer", func() {
			It("should not return error", func() {
				err := validator(3)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("maxReplicaValidator function", func() {
		var validator interactive.Validator

		BeforeEach(func() {
			validator = maxReplicaValidator(1, true)
		})

		When("input is non-integer", func() {
			It("should return error", func() {
				err := validator("non-integer")
				Expect(err).To(HaveOccurred())
			})
		})

		When("maxReplicas is less than minReplicas", func() {
			It("should return error", func() {
				err := validator(0)
				Expect(err).To(HaveOccurred())
			})
		})

		When("input is not a multiple of 3 for multiAZ", func() {
			It("should return error", func() {
				err := validator(5)
				Expect(err).To(HaveOccurred())
			})
		})

		When("input is valid", func() {
			It("should not return error", func() {
				err := validator(3)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("spotMaxPriceValidator function", func() {
		When("input is 'on-demand'", func() {
			It("should return nil", func() {
				err := spotMaxPriceValidator("on-demand")
				Expect(err).NotTo(HaveOccurred())
			})
		})

		When("input is non-numeric", func() {
			It("should return error", func() {
				err := spotMaxPriceValidator("not-a-number")
				Expect(err).To(HaveOccurred())
			})
		})

		When("input is a negative price", func() {
			It("should return error", func() {
				err := spotMaxPriceValidator("-1")
				Expect(err).To(HaveOccurred())
			})
		})

		When("input is a positive price", func() {
			It("should not return error", func() {
				err := spotMaxPriceValidator("0.01")
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
