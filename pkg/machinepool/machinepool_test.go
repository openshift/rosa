package machinepool

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"time"

	"go.uber.org/mock/gomock"

	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	sdk "github.com/openshift-online/ocm-sdk-go"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift-online/ocm-sdk-go/logging"
	. "github.com/openshift-online/ocm-sdk-go/testing"
	"github.com/spf13/cobra"

	mock "github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/securitygroups"
	"github.com/openshift/rosa/pkg/ocm"
	ocmOutput "github.com/openshift/rosa/pkg/ocm/output"
	mpOpts "github.com/openshift/rosa/pkg/options/machinepool"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/test"
)

var policyBuilder cmv1.NodePoolUpgradePolicyBuilder
var date time.Time

var _ = Describe("Machinepool and nodepool", func() {
	var (
		mockClient  *mock.MockClient
		mockCtrl    *gomock.Controller
		testCommand *cobra.Command
		maxReplicas int
		minReplicas int
	)
	Context("Nodepools", Ordered, func() {
		BeforeAll(func() {
			location, err := time.LoadLocation("America/New_York")
			Expect(err).ToNot(HaveOccurred())
			date = time.Date(2024, time.April, 2, 2, 2, 0, 0, location)
			policyBuilder = *cmv1.NewNodePoolUpgradePolicy().ID("test-policy").Version("1").
				ClusterID("test-cluster").State(cmv1.NewUpgradePolicyState().ID("test-state").
				Value(cmv1.UpgradePolicyStateValueScheduled)).
				NextRun(date)
			mockCtrl = gomock.NewController(GinkgoT())
			mockClient = mock.NewMockClient(mockCtrl)
			testCommand = &cobra.Command{}
			flags := testCommand.Flags()
			flags.IntVar(
				&maxReplicas,
				"max-replicas",
				0,
				"Maximum number of machines for the machine pool.",
			)
			flags.IntVar(
				&minReplicas,
				"min-replicas",
				0,
				"Minimum number of machines for the machine pool.",
			)
			testCommand.Flags().Set("min-replicas", "0")
			testCommand.Flags().Set("max-replicas", "1")
			testCommand.Flags().Lookup("min-replicas").Changed = true
			testCommand.Flags().Lookup("max-replicas").Changed = true
		})
		It("editAutoscaling should not be nil if nothing is changed", func() {
			nodepool, err := cmv1.NewNodePool().
				Autoscaling(cmv1.NewNodePoolAutoscaling().MaxReplica(2).MinReplica(1)).
				Build()
			Expect(err).ToNot(HaveOccurred())
			builder := editAutoscaling(nodepool, 1, 2)
			Expect(builder).To(Not(BeNil()))
		})
		It("editAutoscaling should equal the expected output", func() {
			nodepool, err := cmv1.NewNodePool().
				Autoscaling(cmv1.NewNodePoolAutoscaling().MaxReplica(2).MinReplica(1)).
				Build()
			Expect(err).ToNot(HaveOccurred())
			builder := editAutoscaling(nodepool, 2, 3)
			asBuilder := cmv1.NewNodePoolAutoscaling().MaxReplica(3).MinReplica(2)
			Expect(builder).To(Equal(asBuilder))
		})
		It("editAutoscaling should equal the expected output with no min replica value", func() {
			nodepool, err := cmv1.NewNodePool().
				Autoscaling(cmv1.NewNodePoolAutoscaling().MaxReplica(2).MinReplica(1)).
				Build()
			Expect(err).ToNot(HaveOccurred())
			builder := editAutoscaling(nodepool, 0, 3)
			asBuilder := cmv1.NewNodePoolAutoscaling().MaxReplica(3).MinReplica(1)
			Expect(builder).To(Equal(asBuilder))
		})
		It("editAutoscaling should equal the expected output with no max replica value", func() {
			nodepool, err := cmv1.NewNodePool().
				Autoscaling(cmv1.NewNodePoolAutoscaling().MaxReplica(4).MinReplica(1)).
				Build()
			Expect(err).ToNot(HaveOccurred())
			builder := editAutoscaling(nodepool, 2, 0)
			asBuilder := cmv1.NewNodePoolAutoscaling().MaxReplica(4).MinReplica(2)
			Expect(builder).To(Equal(asBuilder))
		})
		It("Test edit nodepool min-replicas < 1 when autoscaling is set", func() {
			err := validateNodePoolEdit(testCommand, true, 0, 0, 1)
			Expect(err.Error()).To(Equal("min-replicas must be greater than zero."))
		})
		It("Test edit nodepool !autoscaling and replicas < 0 for nodepools", func() {
			err := validateNodePoolEdit(testCommand, false, -1, 0, 0)
			Expect(err.Error()).To(Equal("The number of machine pool replicas needs to be a non-negative integer"))
		})
		It("Test edit nodepool autoscaling and minReplicas > maxReplicas", func() {
			err := validateNodePoolEdit(testCommand, true, 0, 5, 1)
			Expect(err.Error()).To(Equal("The number of machine pool min-replicas needs to be less " +
				"than the number of machine pool max-replicas"))
		})
		It("Test edit nodepool autoscaling and maxReplicas < 1", func() {
			err := validateNodePoolEdit(testCommand, true, 0, 1, 0)
			Expect(err.Error()).To(Equal("max-replicas must be greater than zero."))
		})

		Context("Prompt For NodePoolNodeRecreate", func() {
			var mockPromptFuncInvoked bool
			var t *test.TestingRuntime

			BeforeEach(func() {
				t = test.NewTestRuntime()
				mockPromptFuncInvoked = false
			})

			invoked := func(r *rosa.Runtime) bool {
				mockPromptFuncInvoked = true
				return mockPromptFuncInvoked
			}

			It("Prompts when the user has deleted a kubelet-config", func() {

				original := test.MockNodePool(func(n *cmv1.NodePoolBuilder) {
					n.KubeletConfigs("test")
				})

				update := test.MockNodePool(func(n *cmv1.NodePoolBuilder) {
					n.KubeletConfigs("")
				})

				Expect(promptForNodePoolNodeRecreate(original, update, invoked, t.RosaRuntime)).To(BeTrue())
				Expect(mockPromptFuncInvoked).To(BeTrue())
			})

			It("Prompts when the user has changed a kubelet-config", func() {

				original := test.MockNodePool(func(n *cmv1.NodePoolBuilder) {
					n.KubeletConfigs("test")
				})

				update := test.MockNodePool(func(n *cmv1.NodePoolBuilder) {
					n.KubeletConfigs("bar")
				})

				Expect(promptForNodePoolNodeRecreate(original, update, invoked, t.RosaRuntime)).To(BeTrue())
				Expect(mockPromptFuncInvoked).To(BeTrue())
			})

			It("Does not prompts when the user has not changed a kubelet-config", func() {
				original := test.MockNodePool(func(n *cmv1.NodePoolBuilder) {
					n.KubeletConfigs("test")
				})

				update := test.MockNodePool(func(n *cmv1.NodePoolBuilder) {
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
					AWSNodePool(cmv1.NewAWSNodePool().RootVolume(cmv1.NewAWSVolume().Size(256))).
					Subnet("sn").Version(cmv1.NewVersion().ID("1")).AutoRepair(false)))
			cluster, err := clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			out := getNodePoolsString(cluster.NodePools().Slice())
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal(fmt.Sprintf("ID\tAUTOSCALING\tREPLICAS\t"+
				"INSTANCE TYPE\tLABELS\t\tTAINTS\t\tAVAILABILITY ZONE\tSUBNET\tDISK SIZE\tVERSION\tAUTOREPAIR\t\n"+
				"%s\t%s\t%s\t%s\t%s\t\t%s\t\t%s\t%s\t%s\t%s\t%s\t\n",
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
				ocmOutput.PrintNodePoolDiskSize(cluster.NodePools().Get(0).AWSNodePool()),
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
		subnet := "subnet-12345"
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
		It("Test printMachinePools with basic functionality", func() {
			clusterBuilder := cmv1.NewCluster().ID("test").State(cmv1.ClusterStateReady).
				MachinePools(cmv1.NewMachinePoolList().
					Items(cmv1.NewMachinePool().ID("mp-1").Replicas(3).
						InstanceType("m5.large").Subnets("subnet-1", "subnet-2").
						Taints(cmv1.NewTaint().Value("test-value").Key("test-key"))))
			cluster, err := clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())

			r := &rosa.Runtime{}
			args := ListMachinePoolArgs{}
			out := getMachinePoolsString(r, cluster.MachinePools().Slice(), args)

			expectedOutput := "ID\tAUTOSCALING\tREPLICAS\tINSTANCE TYPE\tTAINTS\tSUBNETS\tSPOT INSTANCES\tDISK SIZE\n" +
				"mp-1\tNo\t3\tm5.large\ttest-key=test-value:\tsubnet-1, subnet-2\tNo\tdefault\n"
			Expect(out).To(Equal(expectedOutput))
		})

		It("Test printMachinePools with no machine pools", func() {
			clusterBuilder := cmv1.NewCluster().ID("test").State(cmv1.ClusterStateReady).
				MachinePools(cmv1.NewMachinePoolList())
			cluster, err := clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())

			r := &rosa.Runtime{}
			args := ListMachinePoolArgs{}
			out := getMachinePoolsString(r, cluster.MachinePools().Slice(), args)

			// When there are no machine pools, only headers are returned
			expectedOutput := "ID\tAUTOSCALING\tREPLICAS\tINSTANCE TYPE\tLABELS\tTAINTS\tAVAILABILITY ZONES\tSUBNETS\tSPOT INSTANCES\tDISK SIZE\tSG IDS\n"

			Expect(out).To(Equal(expectedOutput))
		})

		It("Test printMachinePools with showAll flag", func() {
			clusterBuilder := cmv1.NewCluster().ID("test").State(cmv1.ClusterStateReady).
				MachinePools(cmv1.NewMachinePoolList().
					Items(cmv1.NewMachinePool().ID("mp-1").Replicas(3).
						InstanceType("m5.large")))
			cluster, err := clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())

			r := &rosa.Runtime{}
			args := ListMachinePoolArgs{ShowAll: true}
			out := getMachinePoolsString(r, cluster.MachinePools().Slice(), args)

			expectedOutput := "ID\tAUTOSCALING\tREPLICAS\tINSTANCE TYPE\tLABELS\tTAINTS\tAVAILABILITY ZONES\tSUBNETS\tSPOT INSTANCES\tDISK SIZE\tSG IDS\tAZ TYPE\tWIN-LI ENABLED\tDEDICATED HOST\n" +
				"mp-1\tNo\t3\tm5.large\t\t\t\t\tNo\tdefault\t\tN/A\tNo\tNo\n"
			Expect(out).To(Equal(expectedOutput))
		})

		It("Test printMachinePools with showAZType flag", func() {
			clusterBuilder := cmv1.NewCluster().ID("test").State(cmv1.ClusterStateReady).
				MachinePools(cmv1.NewMachinePoolList().
					Items(cmv1.NewMachinePool().ID("mp-1").Replicas(3).
						InstanceType("m5.large")))
			cluster, err := clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())

			r := &rosa.Runtime{}
			args := ListMachinePoolArgs{ShowAZType: true}
			out := getMachinePoolsString(r, cluster.MachinePools().Slice(), args)

			expectedOutput := "ID\tAUTOSCALING\tREPLICAS\tINSTANCE TYPE\tSPOT INSTANCES\tDISK SIZE\tAZ TYPE\n" +
				"mp-1\tNo\t3\tm5.large\tNo\tdefault\tN/A\n"
			Expect(out).To(Equal(expectedOutput))
		})

		It("Test printMachinePools with showDedicated flag", func() {
			clusterBuilder := cmv1.NewCluster().ID("test").State(cmv1.ClusterStateReady).
				MachinePools(cmv1.NewMachinePoolList().
					Items(cmv1.NewMachinePool().ID("mp-1").Replicas(3).
						InstanceType("m5.large")))
			cluster, err := clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())

			r := &rosa.Runtime{}
			args := ListMachinePoolArgs{ShowDedicated: true}
			out := getMachinePoolsString(r, cluster.MachinePools().Slice(), args)

			expectedOutput := "ID\tAUTOSCALING\tREPLICAS\tINSTANCE TYPE\tSPOT INSTANCES\tDISK SIZE\tDEDICATED HOST\n" +
				"mp-1\tNo\t3\tm5.large\tNo\tdefault\tNo\n"
			Expect(out).To(Equal(expectedOutput))
		})

		It("Test printMachinePools with showWindowsLI flag", func() {
			clusterBuilder := cmv1.NewCluster().ID("test").State(cmv1.ClusterStateReady).
				MachinePools(cmv1.NewMachinePoolList().
					Items(cmv1.NewMachinePool().ID("mp-1").Replicas(3).
						InstanceType("m5.large")))
			cluster, err := clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())

			r := &rosa.Runtime{}
			args := ListMachinePoolArgs{ShowWindowsLI: true}
			out := getMachinePoolsString(r, cluster.MachinePools().Slice(), args)

			expectedOutput := "ID\tAUTOSCALING\tREPLICAS\tINSTANCE TYPE\tSPOT INSTANCES\tDISK SIZE\tWIN-LI ENABLED\n" +
				"mp-1\tNo\t3\tm5.large\tNo\tdefault\tNo\n"
			Expect(out).To(Equal(expectedOutput))
		})

		It("Test printMachinePools with multiple flags", func() {
			clusterBuilder := cmv1.NewCluster().ID("test").State(cmv1.ClusterStateReady).
				MachinePools(cmv1.NewMachinePoolList().
					Items(cmv1.NewMachinePool().ID("mp-1").Replicas(3).
						InstanceType("m5.large")))
			cluster, err := clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())

			r := &rosa.Runtime{}
			args := ListMachinePoolArgs{ShowAZType: true, ShowDedicated: true}
			out := getMachinePoolsString(r, cluster.MachinePools().Slice(), args)

			expectedOutput := "ID\tAUTOSCALING\tREPLICAS\tINSTANCE TYPE\tSPOT INSTANCES\tDISK SIZE\tAZ TYPE\tDEDICATED HOST\n" +
				"mp-1\tNo\t3\tm5.large\tNo\tdefault\tN/A\tNo\n"
			Expect(out).To(Equal(expectedOutput))
		})

		It("Test printMachinePools with autoscaling machine pool", func() {
			autoscaling := cmv1.NewMachinePoolAutoscaling().MinReplicas(2).MaxReplicas(10)
			clusterBuilder := cmv1.NewCluster().ID("test").State(cmv1.ClusterStateReady).
				MachinePools(cmv1.NewMachinePoolList().
					Items(cmv1.NewMachinePool().ID("mp-autoscale").
						Autoscaling(autoscaling).
						InstanceType("m5.xlarge")))
			cluster, err := clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())

			r := &rosa.Runtime{}
			args := ListMachinePoolArgs{}
			out := getMachinePoolsString(r, cluster.MachinePools().Slice(), args)

			expectedOutput := "ID\tAUTOSCALING\tREPLICAS\tINSTANCE TYPE\tSPOT INSTANCES\tDISK SIZE\n" +
				"mp-autoscale\tYes\t2-10\tm5.xlarge\tNo\tdefault\n"
			Expect(out).To(Equal(expectedOutput))
		})

		It("Test printMachinePools with multiple machine pools", func() {
			autoscaling := cmv1.NewMachinePoolAutoscaling().MinReplicas(1).MaxReplicas(5)

			clusterBuilder := cmv1.NewCluster().ID("test").State(cmv1.ClusterStateReady).
				MachinePools(cmv1.NewMachinePoolList().
					Items(
						cmv1.NewMachinePool().ID("mp-1").Replicas(3).InstanceType("m5.large"),
						cmv1.NewMachinePool().ID("mp-2").Autoscaling(autoscaling).InstanceType("c5.xlarge"),
						cmv1.NewMachinePool().ID("mp-3").Replicas(1).InstanceType("t3.medium"),
					))
			cluster, err := clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())

			r := &rosa.Runtime{}
			args := ListMachinePoolArgs{}
			out := getMachinePoolsString(r, cluster.MachinePools().Slice(), args)

			expectedOutput := "ID\tAUTOSCALING\tREPLICAS\tINSTANCE TYPE\tSPOT INSTANCES\tDISK SIZE\n" +
				"mp-1\tNo\t3\tm5.large\tNo\tdefault\n" +
				"mp-2\tYes\t1-5\tc5.xlarge\tNo\tdefault\n" +
				"mp-3\tNo\t1\tt3.medium\tNo\tdefault\n"
			Expect(out).To(Equal(expectedOutput))
		})

		It("Test printMachinePools column filtering logic", func() {
			clusterBuilder := cmv1.NewCluster().ID("test").State(cmv1.ClusterStateReady).
				MachinePools(cmv1.NewMachinePoolList().
					Items(
						// Machine pool with minimal data
						cmv1.NewMachinePool().ID("mp-minimal").Replicas(1).InstanceType("t3.small"),
						// Machine pool with more complete data
						cmv1.NewMachinePool().ID("mp-complete").Replicas(3).InstanceType("m5.large").
							Subnets("subnet-1").AvailabilityZones("us-east-1a"),
					))
			cluster, err := clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())

			r := &rosa.Runtime{}
			args := ListMachinePoolArgs{}
			out := getMachinePoolsString(r, cluster.MachinePools().Slice(), args)

			expectedOutput := "ID\tAUTOSCALING\tREPLICAS\tINSTANCE TYPE\tAVAILABILITY ZONES\tSUBNETS\tSPOT INSTANCES\tDISK SIZE\n" +
				"mp-minimal\tNo\t1\tt3.small\t\t\tNo\tdefault\n" +
				"mp-complete\tNo\t3\tm5.large\tus-east-1a\tsubnet-1\tNo\tdefault\n"
			Expect(out).To(Equal(expectedOutput))
		})

		It("Test printMachinePools with empty data columns filtered out", func() {
			clusterBuilder := cmv1.NewCluster().ID("test").State(cmv1.ClusterStateReady).
				MachinePools(cmv1.NewMachinePoolList().
					Items(cmv1.NewMachinePool().ID("mp-1").Replicas(2).
						InstanceType("m5.medium")))
			cluster, err := clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())

			r := &rosa.Runtime{}
			args := ListMachinePoolArgs{}
			out := getMachinePoolsString(r, cluster.MachinePools().Slice(), args)

			// Only columns with actual data should be included
			expectedOutput := "ID\tAUTOSCALING\tREPLICAS\tINSTANCE TYPE\tSPOT INSTANCES\tDISK SIZE\n" +
				"mp-1\tNo\t2\tm5.medium\tNo\tdefault\n"
			Expect(out).To(Equal(expectedOutput))
		})

		It("Validate invalid regex", func() {
			Expect(MachinePoolKeyRE.MatchString("$%%$%$%^$%^$%^$%^")).To(BeFalse())
			Expect(MachinePoolKeyRE.MatchString("machinepool1")).To(BeTrue())
			Expect(MachinePoolKeyRE.MatchString("1machinepool")).To(BeFalse())
			Expect(MachinePoolKeyRE.MatchString("#1machinepool")).To(BeFalse())
			Expect(MachinePoolKeyRE.MatchString("m123123123123123123123123123")).To(BeTrue())
			Expect(MachinePoolKeyRE.MatchString("m#123")).To(BeFalse())
		})
		It("Tests getMachinePoolAvailabilityZones", func() {
			r := &rosa.Runtime{}
			r.AWSClient = mockClient
			var expectedAZs []string
			clusterBuilder := cmv1.NewCluster().ID("test").State(cmv1.ClusterStateReady).
				MultiAZ(true).Nodes(cmv1.NewClusterNodes().
				AvailabilityZones("us-east-1a", "us-east-1b"))
			cluster, err := clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			isMultiAZ := cluster.MultiAZ()
			Expect(isMultiAZ).To(Equal(true))

			multiAZMachinePool := false
			availabilityZoneUserInput := "us-east-1a"
			subnetUserInput := ""

			azs, err := getMachinePoolAvailabilityZones(r, cluster,
				multiAZMachinePool, availabilityZoneUserInput, subnetUserInput)
			Expect(err).ToNot(HaveOccurred())

			expectedAZs = append(expectedAZs, "us-east-1a")
			Expect(azs).To(Equal(expectedAZs))

			multiAZMachinePool = true
			expectedAZs = append(expectedAZs, "us-east-1b")
			azs, err = getMachinePoolAvailabilityZones(r, cluster,
				multiAZMachinePool, availabilityZoneUserInput, subnetUserInput)
			Expect(err).ToNot(HaveOccurred())

			Expect(azs).To(Equal(expectedAZs))

			// Test with subnet input
			newAvailabilityZoneUserInput := "us-east-1a"
			subnetUserInput = "subnet-12345"
			multiAZMachinePool = true
			mockClient.EXPECT().GetSubnetAvailabilityZone(subnetUserInput).
				Return(newAvailabilityZoneUserInput, nil)

			azs, err = getMachinePoolAvailabilityZones(r, cluster,
				multiAZMachinePool, newAvailabilityZoneUserInput, subnetUserInput)
			Expect(err).ToNot(HaveOccurred())

			Expect(azs).To(Equal([]string{newAvailabilityZoneUserInput}))
		})

		It("Tests getSubnetFromAvailabilityZone", func() {
			r := &rosa.Runtime{AWSClient: mockClient}
			cmd := &cobra.Command{}
			isAvailabilityZoneSet := false
			args := &mpOpts.CreateMachinepoolUserOptions{}
			az := "us-east-1a"

			subnetId2 := "subnet-456"

			// Mocking private subnet retrieval
			privateSubnets := []ec2types.Subnet{
				{AvailabilityZone: &az, SubnetId: &subnet},
			}
			mockClient.EXPECT().GetVPCPrivateSubnets(gomock.Any()).Return(privateSubnets, nil)

			// Building a mock cluster
			clusterBuilder := cmv1.NewCluster().ID("test-cluster").State(cmv1.ClusterStateReady).
				Nodes(cmv1.NewClusterNodes().AvailabilityZones("us-east-1a")).AWS(cmv1.NewAWS().SubnetIDs(subnet, subnetId2))
			cluster, err := clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())

			// Test when no availability zone is set and only one subnet is returned
			subnet, err := getSubnetFromAvailabilityZone(cmd, r, isAvailabilityZoneSet, cluster, args)
			Expect(err).ToNot(HaveOccurred())
			Expect(subnet).To(Equal("subnet-12345"))
		})
		It("Tests error case for getSubnetFromAvailabilityZone", func() {
			r := &rosa.Runtime{AWSClient: mockClient}
			cmd := &cobra.Command{}
			isAvailabilityZoneSet := true
			args := &mpOpts.CreateMachinepoolUserOptions{
				AvailabilityZone: "us-west-1a",
			}
			az := "us-east-1a"

			privateSubnets := []ec2types.Subnet{
				{AvailabilityZone: &az, SubnetId: &subnet},
			}
			mockClient.EXPECT().GetVPCPrivateSubnets(gomock.Any()).Return(privateSubnets, nil)

			// Building a mock cluster
			clusterBuilder := cmv1.NewCluster().ID("test-cluster").State(cmv1.ClusterStateReady).
				Nodes(cmv1.NewClusterNodes().AvailabilityZones(az)).AWS(cmv1.NewAWS().SubnetIDs(subnet))
			cluster, err := clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())

			// Attempt to get a subnet from a non-existent availability zone
			subnet, err := getSubnetFromAvailabilityZone(cmd, r, isAvailabilityZoneSet, cluster, args)
			Expect(err).To(HaveOccurred())
			Expect(subnet).To(Equal(""))
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
		var replicaSizeValidation *ReplicaSizeValidation

		BeforeEach(func() {
			replicaSizeValidation = &ReplicaSizeValidation{
				ClusterVersion: "openshift-v4.14.14",
				MultiAz:        true,
				IsHostedCp:     false,
				Autoscaling:    false,
			}
			validator = replicaSizeValidation.MinReplicaValidator() // or false for non-multiAZ
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
		var replicaSizeValidation *ReplicaSizeValidation

		BeforeEach(func() {
			replicaSizeValidation = &ReplicaSizeValidation{
				MinReplicas:    1,
				ClusterVersion: "openshift-v4.14.14",
				MultiAz:        true,
				IsHostedCp:     false,
				Autoscaling:    false,
			}
			validator = replicaSizeValidation.MaxReplicaValidator() // or false for non-multiAZ
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

func returnMockCluster(version string) *cmv1.Cluster {
	region := "us-east-1"
	subnet := "subnet-12345"
	v := cmv1.VersionBuilder{}
	v.ID(version).ChannelGroup("stable").RawID(version).Default(true).
		Enabled(true).ROSAEnabled(true).HostedControlPlaneDefault(true)
	cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
		c.State(cmv1.ClusterStateReady)
		b := cmv1.HypershiftBuilder{}
		a := cmv1.AWSBuilder{}
		s := cmv1.STSBuilder{}
		n := cmv1.ClusterNodesBuilder{}
		r := cmv1.CloudRegionBuilder{}
		cloud := cmv1.CloudProviderBuilder{}
		cloud.Name("aws").Regions(&r).Name("aws")
		r.ID(region).Name(region)
		n.AvailabilityZones("a1")
		s.RoleARN("arn:aws:iam::123456789012:role/SampleRole")
		a.STS(&s).AccountID("123456789012").SubnetIDs(subnet)
		b.Enabled(true)
		c.Hypershift(&b)
		c.ID("test").State(cmv1.ClusterStateReady).AWS(&a).MultiAZ(true).
			Version(&v).Nodes(&n).Region(&r).CloudProvider(&cloud)
	})

	return cluster
}

var _ = Describe("MachinePools", func() {
	Context("AddMachinePool validation errors", func() {
		var (
			cmd        *cobra.Command
			clusterKey string
			args       mpOpts.CreateMachinepoolUserOptions
			cluster    *cmv1.Cluster
			err        error
			t          *TestRuntime
			mockClient *mock.MockClient
			mockCtrl   *gomock.Controller
			subnet     string
			region     string
			version    string
		)

		JustBeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			mockClient = mock.NewMockClient(mockCtrl)
			t = NewTestingRuntime(mockClient)
			args = mpOpts.CreateMachinepoolUserOptions{}
			clusterKey = "test-cluster-key"
			cmd = &cobra.Command{}
			subnet = "subnet-12345"
			version = "4.15.0"
			region = "us-east-1"
		})

		It("should error when 'multi-availability-zone' flag is set for non-multi-AZ clusters", func() {
			machinePool := &machinePool{}
			cmd.Flags().Bool("multi-availability-zone", true, "")
			cmd.Flags().Set("multi-availability-zone", "true")
			err = machinePool.CreateMachinePool(t.RosaRuntime, cmd, clusterKey, cluster, &args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Setting the `multi-availability-zone` flag is only allowed for multi-AZ clusters"))
		})

		It("should error when 'availability-zone' flag is set for non-multi-AZ clusters", func() {
			machinePool := &machinePool{}
			cmd.Flags().StringVar(&args.AvailabilityZone, "availability-zone", "", "")
			cmd.Flags().Set("availability-zone", "az")
			err = machinePool.CreateMachinePool(t.RosaRuntime, cmd, clusterKey, cluster, &args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Setting the `availability-zone` flag is only allowed for multi-AZ clusters"))
		})

		It("should error when 'subnet' flag is set for non-BYOVPC clusters", func() {
			machinePool := &machinePool{}
			cmd.Flags().StringVar(&args.Subnet, "subnet", "", "")
			cmd.Flags().Set("subnet", "test-subnet")
			err = machinePool.CreateMachinePool(t.RosaRuntime, cmd, clusterKey, cluster, &args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Setting the `subnet` flag is only allowed for BYO VPC clusters"))
		})

		It("should error when the security group IDs flag is set for non-BYOVPC clusters", func() {
			machinePool := &machinePool{}
			v := cmv1.VersionBuilder{}
			v.RawID(version).ChannelGroup("stable")
			cluster = test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.ID("test")
				c.State(cmv1.ClusterStateReady)
				c.Version(&v)
			})
			cmd.Flags().StringSliceVar(&args.SecurityGroupIds, "additional-security-group-ids",
				[]string{}, "comma-separated list of security group IDs")
			cmd.Flags().Set("additional-security-group-ids", "sg-12345")
			err = machinePool.CreateMachinePool(t.RosaRuntime, cmd, clusterKey, cluster, &args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf(
				"Setting the `%s` flag is only allowed for BYOVPC clusters",
				securitygroups.MachinePoolSecurityGroupFlag)))
		})

		It("should error checking version compatibility", func() {
			machinePool := &machinePool{}
			incompatibleVersion := "2.5.0"
			v := cmv1.VersionBuilder{}
			v.ID(incompatibleVersion).ChannelGroup("stable")
			cluster = test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.ID("test")
				c.State(cmv1.ClusterStateReady)
				c.Version(&v)
			})

			err = machinePool.CreateMachinePool(t.RosaRuntime, cmd, clusterKey, cluster, &args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("There was a problem checking version compatibility:"))
		})

		It("should error when setting flag that is only allowed for BYOVPC clusters", func() {
			machinePool := &machinePool{}
			v := cmv1.VersionBuilder{}
			v.ID(version).ChannelGroup("stable").RawID(version)
			cluster = test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.ID("test")
				c.State(cmv1.ClusterStateReady)
				c.Version(&v)
			})

			cmd.Flags().StringSliceVar(&args.SecurityGroupIds, "additional-security-group-ids",
				[]string{}, "comma-separated list of security group IDs")
			cmd.Flags().Set("additional-security-group-ids", "sg-12345")
			err = machinePool.CreateMachinePool(t.RosaRuntime, cmd, clusterKey, cluster, &args)
			Expect(err).To(HaveOccurred())

			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(
				"Setting the `%s` flag is only allowed for BYOVPC clusters",
				securitygroups.MachinePoolSecurityGroupFlag)))
		})

		It("should error when the security group IDs flag is set for clusters with incompatible versions", func() {
			machinePool := &machinePool{}
			incompatibleVersion := "2.5.0"
			v := cmv1.VersionBuilder{}
			v.ID(incompatibleVersion).ChannelGroup("stable").RawID(incompatibleVersion).Default(true).
				Enabled(true).ROSAEnabled(true).HostedControlPlaneDefault(true)
			cluster = test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.ID("test")
				a := cmv1.AWSBuilder{}
				n := cmv1.ClusterNodesBuilder{}
				n.AvailabilityZones("a1")
				a.AccountID("123456789012").SubnetIDs(subnet)
				c.State(cmv1.ClusterStateReady).AWS(&a).Version(&v).MultiAZ(true).Nodes(&n)
			})

			cmd.Flags().StringSliceVar(&args.SecurityGroupIds, "additional-security-group-ids",
				[]string{}, "comma-separated list of security group IDs")
			cmd.Flags().Set("additional-security-group-ids", "sg-12345")
			err = machinePool.CreateMachinePool(t.RosaRuntime, cmd, clusterKey, cluster, &args)
			Expect(err).To(HaveOccurred())

			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(
				"Parameter '%s' is not supported prior to version ", securitygroups.MachinePoolSecurityGroupFlag)))
		})

		It("should error when both 'subnet' and 'availability-zone' flags are set", func() {
			machinePool := &machinePool{}
			cluster = returnMockCluster(version)

			cmd.Flags().StringSliceVar(&args.SecurityGroupIds, "additional-security-group-ids",
				[]string{}, "comma-separated list of security group IDs")
			cmd.Flags().Set("additional-security-group-ids", "sg-12345")
			cmd.Flags().StringVar(&args.Subnet, "subnet", "", "")
			cmd.Flags().Set("subnet", "test-subnet")
			cmd.Flags().StringVar(&args.AvailabilityZone, "availability-zone", "", "")
			cmd.Flags().Set("availability-zone", "az")
			err = machinePool.CreateMachinePool(t.RosaRuntime, cmd, clusterKey, cluster, &args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Setting both `subnet` and `availability-zone`" +
				" flag is not supported. Please select `subnet` or `availability-zone` " +
				"to create a single availability zone machine pool"))
		})

		It("should error when 'availability-zone' flag is set for a single AZ machine pool in a multi-AZ cluster", func() {
			machinePool := &machinePool{}
			cluster = returnMockCluster(version)

			cmd.Flags().StringSliceVar(&args.SecurityGroupIds, "additional-security-group-ids",
				[]string{}, "comma-separated list of security group IDs")
			cmd.Flags().Set("additional-security-group-ids", "sg-12345")
			cmd.Flags().Bool("multi-availability-zone", true, "")
			cmd.Flags().Set("multi-availability-zone", "true")
			cmd.Flags().StringVar(&args.AvailabilityZone, "availability-zone", "", "")
			cmd.Flags().Set("availability-zone", "az")
			args.MultiAvailabilityZone = true
			err = machinePool.CreateMachinePool(t.RosaRuntime, cmd, clusterKey, cluster, &args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(
				"Setting the `availability-zone` flag is only supported for creating a" +
					" single AZ machine pool in a multi-AZ cluster"))
		})

		It("should error when setting an invalid name", func() {
			machinePool := &machinePool{}
			invalidName := "998 .-"
			cluster = returnMockCluster(version)

			cmd.Flags().StringVar(&args.Name, "name", "", "Name of the machine pool")
			cmd.Flags().Set("name", invalidName)
			cmd.Flags().Bool("multi-availability-zone", true, "")
			cmd.Flags().Set("multi-availability-zone", "true")
			err = machinePool.CreateMachinePool(t.RosaRuntime, cmd, clusterKey, cluster, &args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected a valid name for the machine pool"))
		})

		It("should error when autoscaling and replicas are enabled", func() {
			machinePool := &machinePool{}
			cluster = returnMockCluster(version)

			cmd.Flags().StringVar(&args.Name, "name", "", "Name of the machine pool")
			cmd.Flags().Set("name", "mp-1")
			cmd.Flags().Bool("multi-availability-zone", true, "")
			cmd.Flags().Set("multi-availability-zone", "true")
			cmd.Flags().Bool("enable-autoscaling", true, "")
			cmd.Flags().Set("enable-autoscaling", "true")
			cmd.Flags().IntVar(&args.Replicas, "replicas", 0, "Replicas of the machine pool")
			cmd.Flags().Set("replicas", "3")
			args.AutoscalingEnabled = true
			args.Replicas = 3
			err = machinePool.CreateMachinePool(t.RosaRuntime, cmd, clusterKey, cluster, &args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Replicas can't be set when autoscaling is enabled"))
		})

		It("should error when not supplying an instance type", func() {
			machinePool := &machinePool{}
			cluster = returnMockCluster(version)

			cmd.Flags().StringVar(&args.Name, "name", "", "Name of the machine pool")
			cmd.Flags().Set("name", "mp-1")
			cmd.Flags().Bool("multi-availability-zone", true, "")
			cmd.Flags().Set("multi-availability-zone", "true")
			cmd.Flags().Bool("enable-autoscaling", true, "")
			cmd.Flags().Set("enable-autoscaling", "true")
			cmd.Flags().Int32("min-replicas", 1, "Replicas of the machine pool")
			cmd.Flags().Set("min-replicas", "1")
			cmd.Flags().Int32("max-replicas", 3, "Replicas of the machine pool")
			cmd.Flags().Set("max-replicas", "3")
			args.AutoscalingEnabled = true
			args.MinReplicas = 1
			args.MaxReplicas = 3
			err = machinePool.CreateMachinePool(t.RosaRuntime, cmd, clusterKey, cluster, &args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("You must supply a valid instance type"))
		})

		It("should error when not supplying min and max replicas but not autoscaling", func() {
			machinePool := &machinePool{}
			cluster = returnMockCluster(version)

			cmd.Flags().StringVar(&args.Name, "name", "", "Name of the machine pool")
			cmd.Flags().Set("name", "mp-1")
			cmd.Flags().Bool("multi-availability-zone", true, "")
			cmd.Flags().Set("multi-availability-zone", "true")
			cmd.Flags().Int32("min-replicas", 1, "Replicas of the machine pool")
			cmd.Flags().Set("min-replicas", "1")
			cmd.Flags().Int32("max-replicas", 3, "Replicas of the machine pool")
			cmd.Flags().Set("max-replicas", "3")
			args.MinReplicas = 1
			args.MaxReplicas = 3
			err = machinePool.CreateMachinePool(t.RosaRuntime, cmd, clusterKey, cluster, &args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Autoscaling must be enabled in order to set min and max replicas"))
		})

		It("Should error when can't set max price when not using spot instances", func() {
			machinePool := &machinePool{}
			cluster = returnMockCluster(version)

			cmd.Flags().StringVar(&args.Name, "name", "", "Name of the machine pool")
			cmd.Flags().Set("name", "mp-1")
			cmd.Flags().Bool("multi-availability-zone", true, "")
			cmd.Flags().Set("multi-availability-zone", "true")
			cmd.Flags().IntVar(&args.Replicas, "replicas", 3, "Replicas of the machine pool")
			cmd.Flags().Set("replicas", "3")
			args.Replicas = 3
			cmd.Flags().BoolVar(&args.UseSpotInstances, "use-spot-instances", false, "")
			cmd.Flags().Set("use-spot-instances", "false")
			cmd.Flags().Changed("use-spot-instances")
			cmd.Flags().StringVar(&args.SpotMaxPrice, "spot-max-price", "0.01", "")
			cmd.Flags().Set("spot-max-price", "0.01")
			args.InstanceType = "t3.small"
			mt, err := cmv1.NewMachineType().ID("t3.small").Name("t3.small").Build()
			Expect(err).ToNot(HaveOccurred())
			acc, err := amsv1.NewAccount().ID("123456789012").Build()
			Expect(err).ToNot(HaveOccurred())
			qc, err := amsv1.NewQuotaCost().QuotaID("test-quota").OrganizationID("123456789012").Version("4.15.0").Build()
			Expect(err).ToNot(HaveOccurred())
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatMachineTypeList([]*cmv1.MachineType{mt})))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatMachineTypeList([]*cmv1.MachineType{mt})))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatResources(acc)))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatQuotaCostList([]*amsv1.QuotaCost{qc})))
			err = machinePool.CreateMachinePool(t.RosaRuntime, cmd, clusterKey, cluster, &args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Can't set max price when not using spot instances"))
		})
		It("Should error when instances are set for local zones", func() {
			machinePool := &machinePool{}
			args.Subnet = subnet
			cluster = returnMockCluster(version)

			cmd.Flags().StringVar(&args.Name, "name", "", "Name of the machine pool")
			cmd.Flags().Set("name", "mp-1")
			cmd.Flags().Bool("multi-availability-zone", true, "")
			cmd.Flags().Set("multi-availability-zone", "true")
			cmd.Flags().IntVar(&args.Replicas, "replicas", 3, "Replicas of the machine pool")
			cmd.Flags().Set("replicas", "3")
			args.Replicas = 3
			args.InstanceType = "t3.small"
			args.UseSpotInstances = true
			mt, err := cmv1.NewMachineType().ID("t3.small").Name("t3.small").Build()
			Expect(err).ToNot(HaveOccurred())
			acc, err := amsv1.NewAccount().ID("123456789012").Build()
			Expect(err).ToNot(HaveOccurred())
			qc, err := amsv1.NewQuotaCost().QuotaID("test-quota").OrganizationID("123456789012").Version("4.15.0").Build()
			Expect(err).ToNot(HaveOccurred())
			mockClient.EXPECT().GetSubnetAvailabilityZone(subnet).Return(region, nil)
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatMachineTypeList([]*cmv1.MachineType{mt})))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatMachineTypeList([]*cmv1.MachineType{mt})))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatResources(acc)))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatQuotaCostList([]*amsv1.QuotaCost{qc})))
			mockClient.EXPECT().IsLocalAvailabilityZone(region).Return(true, nil)
			err = machinePool.CreateMachinePool(t.RosaRuntime, cmd, clusterKey, cluster, &args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Spot instances are not supported for local zones"))
		})
		It("should error when parsing invalid root disk size", func() {
			machinePool := &machinePool{}
			args.Subnet = subnet
			cluster = returnMockCluster(version)

			cmd.Flags().StringVar(&args.Name, "name", "", "Name of the machine pool")
			cmd.Flags().Set("name", "mp-1")
			cmd.Flags().Bool("multi-availability-zone", true, "")
			cmd.Flags().Set("multi-availability-zone", "true")
			cmd.Flags().IntVar(&args.Replicas, "replicas", 3, "Replicas of the machine pool")
			cmd.Flags().Set("replicas", "3")
			args.Replicas = 3
			args.InstanceType = "t3.small"
			args.UseSpotInstances = true
			args.SpotMaxPrice = "1.00"
			args.RootDiskSize = "opj99i"
			mt, err := cmv1.NewMachineType().ID("t3.small").Name("t3.small").Build()
			Expect(err).ToNot(HaveOccurred())
			acc, err := amsv1.NewAccount().ID("123456789012").Build()
			Expect(err).ToNot(HaveOccurred())
			qc, err := amsv1.NewQuotaCost().QuotaID("test-quota").OrganizationID("123456789012").Version("4.15.0").Build()
			Expect(err).ToNot(HaveOccurred())
			mockClient.EXPECT().GetSubnetAvailabilityZone(subnet).Return(region, nil)
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatMachineTypeList([]*cmv1.MachineType{mt})))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatMachineTypeList([]*cmv1.MachineType{mt})))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatResources(acc)))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatQuotaCostList([]*amsv1.QuotaCost{qc})))
			mockClient.EXPECT().IsLocalAvailabilityZone(region).Return(false, nil)
			err = machinePool.CreateMachinePool(t.RosaRuntime, cmd, clusterKey, cluster, &args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Expected a valid machine pool root disk size value"))
		})
		It("should fail when there is an invalid root disk size set", func() {
			machinePool := &machinePool{}
			args.Subnet = subnet
			cluster = returnMockCluster(version)

			cmd.Flags().StringVar(&args.Name, "name", "", "Name of the machine pool")
			cmd.Flags().Set("name", "mp-1")
			cmd.Flags().Bool("multi-availability-zone", true, "")
			cmd.Flags().Set("multi-availability-zone", "true")
			cmd.Flags().IntVar(&args.Replicas, "replicas", 3, "Replicas of the machine pool")
			cmd.Flags().Set("replicas", "3")
			args.Replicas = 3
			args.InstanceType = "t3.small"
			args.UseSpotInstances = true
			args.SpotMaxPrice = "1.00"
			args.RootDiskSize = "1000"
			mt, err := cmv1.NewMachineType().ID("t3.small").Name("t3.small").Build()
			Expect(err).ToNot(HaveOccurred())
			acc, err := amsv1.NewAccount().ID("123456789012").Build()
			Expect(err).ToNot(HaveOccurred())
			qc, err := amsv1.NewQuotaCost().QuotaID("test-quota").OrganizationID("123456789012").Version("4.15.0").Build()
			Expect(err).ToNot(HaveOccurred())
			mockClient.EXPECT().GetSubnetAvailabilityZone(subnet).Return(region, nil)
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatMachineTypeList([]*cmv1.MachineType{mt})))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatMachineTypeList([]*cmv1.MachineType{mt})))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatResources(acc)))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatQuotaCostList([]*amsv1.QuotaCost{qc})))
			mockClient.EXPECT().IsLocalAvailabilityZone(region).Return(false, nil)
			err = machinePool.CreateMachinePool(t.RosaRuntime, cmd, clusterKey, cluster, &args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Invalid root disk size"))
		})
		It("Fails to add a machine pool to cluster", func() {
			machinePool := &machinePool{}
			args.Subnet = subnet
			cluster = returnMockCluster(version)

			cmd.Flags().StringVar(&args.Name, "name", "", "Name of the machine pool")
			cmd.Flags().Set("name", "mp-1")
			cmd.Flags().Bool("multi-availability-zone", true, "")
			cmd.Flags().Set("multi-availability-zone", "true")
			cmd.Flags().IntVar(&args.Replicas, "replicas", 3, "Replicas of the machine pool")
			cmd.Flags().Set("replicas", "3")
			args.Replicas = 3
			args.InstanceType = "t3.small"
			args.UseSpotInstances = true
			args.SpotMaxPrice = "1.00"
			args.RootDiskSize = "1000GB"
			mt, err := cmv1.NewMachineType().ID("t3.small").Name("t3.small").Build()
			Expect(err).ToNot(HaveOccurred())
			acc, err := amsv1.NewAccount().ID("123456789012").Build()
			Expect(err).ToNot(HaveOccurred())
			qc, err := amsv1.NewQuotaCost().QuotaID("test-quota").OrganizationID("123456789012").Version("4.15.0").Build()
			Expect(err).ToNot(HaveOccurred())
			mockClient.EXPECT().GetSubnetAvailabilityZone(subnet).Return(region, nil)
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatMachineTypeList([]*cmv1.MachineType{mt})))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatMachineTypeList([]*cmv1.MachineType{mt})))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatResources(acc)))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatQuotaCostList([]*amsv1.QuotaCost{qc})))
			mockClient.EXPECT().IsLocalAvailabilityZone(region).Return(false, nil)
			err = machinePool.CreateMachinePool(t.RosaRuntime, cmd, clusterKey, cluster, &args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Failed to add machine pool to cluster"))
		})
		It("Successfully create a machine pool", func() {
			machinePool := &machinePool{}
			args.Subnet = subnet
			cluster = returnMockCluster(version)

			cmd.Flags().StringVar(&args.Name, "name", "", "Name of the machine pool")
			cmd.Flags().Set("name", "mp-1")
			cmd.Flags().Bool("multi-availability-zone", true, "")
			cmd.Flags().Set("multi-availability-zone", "true")
			cmd.Flags().Bool("enable-autoscaling", true, "")
			cmd.Flags().Set("enable-autoscaling", "true")
			cmd.Flags().Int32("min-replicas", 1, "Replicas of the machine pool")
			cmd.Flags().Set("min-replicas", "1")
			cmd.Flags().Int32("max-replicas", 3, "Replicas of the machine pool")
			cmd.Flags().Set("max-replicas", "3")
			args.MinReplicas = 1
			args.MaxReplicas = 3
			args.InstanceType = "t3.small"
			args.UseSpotInstances = true
			args.SpotMaxPrice = "1.00"
			args.AutoscalingEnabled = true
			mt, err := cmv1.NewMachineType().ID("t3.small").Name("t3.small").Build()
			Expect(err).ToNot(HaveOccurred())
			acc, err := amsv1.NewAccount().ID("123456789012").Build()
			Expect(err).ToNot(HaveOccurred())
			qc, err := amsv1.NewQuotaCost().QuotaID("test-quota").OrganizationID("123456789012").Version("4.15.0").Build()
			Expect(err).ToNot(HaveOccurred())
			machinePoolObj, err := cmv1.NewMachinePool().ID("mp-1").InstanceType("t3.small").Build()
			Expect(err).ToNot(HaveOccurred())
			mockClient.EXPECT().GetSubnetAvailabilityZone(subnet).Return(region, nil)
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatMachineTypeList([]*cmv1.MachineType{mt})))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatMachineTypeList([]*cmv1.MachineType{mt})))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatResources(acc)))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatQuotaCostList([]*amsv1.QuotaCost{qc})))
			mockClient.EXPECT().IsLocalAvailabilityZone(region).Return(false, nil)
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatResources(machinePoolObj)))
			err = machinePool.CreateMachinePool(t.RosaRuntime, cmd, clusterKey, cluster, &args)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

var _ = Describe("NodePools", func() {
	Context("AddNodePool validation errors", func() {
		var (
			cmd        *cobra.Command
			clusterKey string
			args       mpOpts.CreateMachinepoolUserOptions
			cluster    *cmv1.Cluster
			err        error
			t          *TestRuntime
			mockClient *mock.MockClient
			mockCtrl   *gomock.Controller
			subnet     string
			version    string
		)

		JustBeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			mockClient = mock.NewMockClient(mockCtrl)
			t = NewTestingRuntime(mockClient)
			args = mpOpts.CreateMachinepoolUserOptions{}
			clusterKey = "test-cluster-key"
			cmd = &cobra.Command{}
			subnet = "subnet-12345"
			version = "4.15.0"
		})

		It("should return an error if both `subnet` and `availability-zone` flags are set", func() {
			cmd.Flags().Bool("availability-zone", true, "")
			cmd.Flags().Bool("subnet", true, "")
			cluster = test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				c.ID("test")
			})

			cmd.Flags().Set("availability-zone", "true")
			cmd.Flags().Set("subnet", "true")

			machinePool := &machinePool{}
			err = machinePool.CreateNodePools(t.RosaRuntime, cmd, clusterKey, cluster, nil, &args)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Setting both `subnet` and " +
				"`availability-zone` flag is not supported. Please select `subnet` " +
				"or `availability-zone` to create a single availability zone machine pool"))
		})
		It("should fail name validation", func() {
			machinePool := &machinePool{}
			cluster = test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				c.ID("test")
			})

			cmd.Flags().StringVar(&args.Name, "name", "", "Name of the machine pool")
			invalidName := "0909+===..3"
			cmd.Flags().Set("name", invalidName)

			err = machinePool.CreateNodePools(t.RosaRuntime, cmd, clusterKey, cluster, nil, &args)
			Expect(err).To(HaveOccurred())

			Expect(err.Error()).To(Equal("Expected a valid name for the machine pool"))
		})
		It("should fail version validation", func() {
			machinePool := &machinePool{}
			v := cmv1.VersionBuilder{}
			v.ID(version).ChannelGroup("stable").RawID(version).Default(true).
				Enabled(true).ROSAEnabled(true).HostedControlPlaneDefault(true)
			cluster = returnMockCluster(version)

			cmd.Flags().StringVar(&args.Name, "name", "", "Name of the machine pool")
			cmd.Flags().Set("name", "test")

			cmd.Flags().StringVar(&args.Version, "version", "", "Version of the machine pool")
			cmd.Flags().Set("version", "aaaa")
			isVersionSet := cmd.Flags().Changed("version")
			Expect(isVersionSet).To(BeTrue())

			versionObj, err := v.Build()
			Expect(err).ToNot(HaveOccurred())

			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatVersionList([]*cmv1.Version{versionObj})))
			err = machinePool.CreateNodePools(t.RosaRuntime, cmd, clusterKey, cluster, nil, &args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Expected a valid OpenShift version"))
		})
		It("should fail when not providing a valid instance type", func() {
			machinePool := &machinePool{}
			version := "4.15.0"
			v := cmv1.VersionBuilder{}
			v.ID(version).ChannelGroup("stable").RawID(version).Default(true).
				Enabled(true).ROSAEnabled(true).HostedControlPlaneDefault(true)
			az := "a1"
			cluster = returnMockCluster(version)
			privateSubnets := []ec2types.Subnet{
				{AvailabilityZone: &az, SubnetId: &subnet},
			}

			cmd.Flags().StringVar(&args.Name, "name", "", "Name of the machine pool")
			cmd.Flags().Set("name", "test")

			cmd.Flags().StringVar(&args.Version, "version", "", "Version of the machine pool")
			cmd.Flags().Set("version", version)
			isVersionSet := cmd.Flags().Changed("version")
			Expect(isVersionSet).To(BeTrue())

			cmd.Flags().Bool("enable-autoscaling", true, "")
			cmd.Flags().Set("enable-autoscaling", "true")
			cmd.Flags().Int32("min-replicas", 1, "Replicas of the machine pool")
			cmd.Flags().Set("min-replicas", "1")
			cmd.Flags().Int32("max-replicas", 3, "Replicas of the machine pool")
			cmd.Flags().Set("max-replicas", "3")
			args.MinReplicas = 1
			args.MaxReplicas = 3
			args.AutoscalingEnabled = true

			versionObj, err := v.Build()
			Expect(err).ToNot(HaveOccurred())

			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatVersionList([]*cmv1.Version{versionObj})))
			mockClient.EXPECT().GetVPCPrivateSubnets(gomock.Any()).Return(privateSubnets, nil)
			err = machinePool.CreateNodePools(t.RosaRuntime, cmd, clusterKey, cluster, nil, &args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("You must supply a valid instance type"))
		})
		It("fails to add the node pool to the hosted cluster", func() {
			machinePool := &machinePool{}
			version := "4.15.0"
			v := cmv1.VersionBuilder{}
			v.ID(version).ChannelGroup("stable").RawID(version).Default(true).
				Enabled(true).ROSAEnabled(true).HostedControlPlaneDefault(true)
			az := "a1"

			cluster = returnMockCluster(version)
			privateSubnets := []ec2types.Subnet{
				{AvailabilityZone: &az, SubnetId: &subnet},
			}

			cmd.Flags().StringVar(&args.Name, "name", "", "Name of the machine pool")
			cmd.Flags().Set("name", "test")

			cmd.Flags().StringVar(&args.Version, "version", "", "Version of the machine pool")
			cmd.Flags().Set("version", version)
			isVersionSet := cmd.Flags().Changed("version")
			Expect(isVersionSet).To(BeTrue())

			cmd.Flags().StringVar(&args.MaxSurge, "max-surge", "", "Max surge of the machine pool")
			cmd.Flags().Set("max-surge", "1")
			isMaxSurgeSet := cmd.Flags().Changed("max-surge")
			Expect(isMaxSurgeSet).To(BeTrue())

			cmd.Flags().StringVar(&args.MaxUnavailable, "max-unavailable", "", "Max unavailable of the machine pool")
			cmd.Flags().Set("max-unavailable", "1")
			isMaxUnavailableSet := cmd.Flags().Changed("max-unavailable")
			Expect(isMaxUnavailableSet).To(BeTrue())

			cmd.Flags().Bool("enable-autoscaling", true, "")
			cmd.Flags().Set("enable-autoscaling", "true")
			cmd.Flags().Int32("min-replicas", 1, "Replicas of the machine pool")
			cmd.Flags().Set("min-replicas", "1")
			cmd.Flags().Int32("max-replicas", 3, "Replicas of the machine pool")
			cmd.Flags().Set("max-replicas", "3")
			args.MinReplicas = 1
			args.MaxReplicas = 3
			args.AutoscalingEnabled = true
			args.InstanceType = "t3.small"
			args.TuningConfigs = "test"
			args.KubeletConfigs = "test"
			args.MaxSurge = "1"
			args.MaxUnavailable = "1"

			versionObj, err := v.Build()
			Expect(err).ToNot(HaveOccurred())

			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatVersionList([]*cmv1.Version{versionObj})))
			mockClient.EXPECT().GetVPCPrivateSubnets(gomock.Any()).Return(privateSubnets, nil)
			mockClient.EXPECT().GetSubnetAvailabilityZone(subnet).Return(az, nil)
			mtBuilder := cmv1.NewMachineType().ID("t3.small").Name("t3.small")
			machineType, err := mtBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatMachineTypeList(
				[]*cmv1.MachineType{machineType})))
			acc, err := amsv1.NewAccount().ID("123456789012").Build()
			Expect(err).ToNot(HaveOccurred())
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatResources(acc)))
			qc, err := amsv1.NewQuotaCost().
				QuotaID("test-quota").OrganizationID("123456789012").Version("4.15.0").Build()
			Expect(err).ToNot(HaveOccurred())
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatQuotaCostList([]*amsv1.QuotaCost{qc})))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatTuningConfigList([]*cmv1.TuningConfig{})))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatKubeletConfigList([]*cmv1.KubeletConfig{})))
			err = machinePool.CreateNodePools(t.RosaRuntime, cmd, clusterKey, cluster, nil, &args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Failed to add machine pool to hosted cluster"))
		})
		It("Successfully creates a node pool with capacity-reservation-id", func() {
			machinePool := &machinePool{}
			version := "4.19.0"
			az := "a1"

			cluster = returnMockCluster(version)
			privateSubnets := []ec2types.Subnet{
				{AvailabilityZone: &az, SubnetId: &subnet},
			}

			cmd.Flags().StringVar(&args.Name, "name", "", "Name of the machine pool")
			cmd.Flags().Set("name", "test")

			cmd.Flags().StringVar(&args.Version, "version", "", "Version of the machine pool")
			cmd.Flags().Set("version", version)
			isVersionSet := cmd.Flags().Changed("version")
			Expect(isVersionSet).To(BeTrue())

			cmd.Flags().Int32("replicas", 3, "Replicas of the machine pool")
			cmd.Flags().Set("replicas", "3")

			cmd.Flags().StringVar(&args.CapacityReservationId, "capacity-reservation-id", "fake-capacity-reservation-id", "capacity-reservation-id")
			cmd.Flags().Set("capacity-reservation-id", "fake-capacity-reservation-id")
			args.Replicas = 3
			args.InstanceType = "t3.small"

			v := cmv1.VersionBuilder{}
			v.ID(version).ChannelGroup("stable").RawID(version).Default(true).
				Enabled(true).ROSAEnabled(true).HostedControlPlaneDefault(true)
			versionObj, err := v.Build()
			Expect(err).ToNot(HaveOccurred())
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatVersionList(
				[]*cmv1.Version{versionObj})))

			mockClient.EXPECT().GetVPCPrivateSubnets(gomock.Any()).Return(privateSubnets, nil)
			mockClient.EXPECT().GetSubnetAvailabilityZone(subnet).Return(az, nil)

			mtBuilder := cmv1.NewMachineType().ID("t3.small").Name("t3.small")
			machineType, err := mtBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatMachineTypeList(
				[]*cmv1.MachineType{machineType})))
			acc, err := amsv1.NewAccount().ID("123456789012").Build()
			Expect(err).ToNot(HaveOccurred())
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatResources(acc)))
			qc, err := amsv1.NewQuotaCost().QuotaID("test-quota").
				OrganizationID("123456789012").Version("4.19.0").Build()
			Expect(err).ToNot(HaveOccurred())
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatQuotaCostList([]*amsv1.QuotaCost{qc})))
			tuningConfig, err := cmv1.NewTuningConfig().ID("test").Build()
			Expect(err).ToNot(HaveOccurred())
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatTuningConfigList(
				[]*cmv1.TuningConfig{tuningConfig})))
			kubeConfig, err := cmv1.NewKubeletConfig().Name("test").ID("test").PodPidsLimit(5000).Build()
			Expect(err).ToNot(HaveOccurred())
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatKubeletConfigList(
				[]*cmv1.KubeletConfig{kubeConfig})))
			flavour, err := cmv1.NewFlavour().AWS(cmv1.NewAWSFlavour().ComputeInstanceType("x-large").
				WorkerVolume(cmv1.NewAWSVolume().Size(100))).
				Network(cmv1.NewNetwork().MachineCIDR("").PodCIDR("").ServiceCIDR("").HostPrefix(1)).Build()
			Expect(err).ToNot(HaveOccurred())
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatResources(flavour)))
			nodePoolObj, err := cmv1.NewNodePool().ID("np-1").Build()
			Expect(err).ToNot(HaveOccurred())
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatResources(nodePoolObj)))
			nodePoolResponse := test.FormatNodePoolList([]*cmv1.NodePool{nodePoolObj})
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolResponse))

			err = machinePool.CreateNodePools(t.RosaRuntime, cmd, clusterKey, cluster, nil, &args)
			Expect(err).To(Not(HaveOccurred()))
		})

		It("Successfully creates a node pool", func() {
			machinePool := &machinePool{}
			version := "4.15.0"
			az := "a1"

			cluster = returnMockCluster(version)
			privateSubnets := []ec2types.Subnet{
				{AvailabilityZone: &az, SubnetId: &subnet},
			}

			cmd.Flags().StringVar(&args.Name, "name", "", "Name of the machine pool")
			cmd.Flags().Set("name", "test")

			cmd.Flags().StringVar(&args.Version, "version", "", "Version of the machine pool")
			cmd.Flags().Set("version", version)
			isVersionSet := cmd.Flags().Changed("version")
			Expect(isVersionSet).To(BeTrue())

			cmd.Flags().Bool("enable-autoscaling", true, "")
			cmd.Flags().Set("enable-autoscaling", "true")
			cmd.Flags().Int32("min-replicas", 1, "Replicas of the machine pool")
			cmd.Flags().Set("min-replicas", "1")
			cmd.Flags().Int32("max-replicas", 3, "Replicas of the machine pool")
			cmd.Flags().Set("max-replicas", "3")

			args.MinReplicas = 1
			args.MaxReplicas = 3
			args.AutoscalingEnabled = true
			args.InstanceType = "t3.small"
			args.TuningConfigs = "test"
			args.KubeletConfigs = "test"
			args.NodeDrainGracePeriod = "30"

			args.RootDiskSize = "256GB"

			v := cmv1.VersionBuilder{}
			v.ID(version).ChannelGroup("stable").RawID(version).Default(true).
				Enabled(true).ROSAEnabled(true).HostedControlPlaneDefault(true)
			versionObj, err := v.Build()
			Expect(err).ToNot(HaveOccurred())

			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatVersionList(
				[]*cmv1.Version{versionObj})))
			mockClient.EXPECT().GetVPCPrivateSubnets(gomock.Any()).Return(privateSubnets, nil)
			mockClient.EXPECT().GetSubnetAvailabilityZone(subnet).Return(az, nil)
			mtBuilder := cmv1.NewMachineType().ID("t3.small").Name("t3.small")
			machineType, err := mtBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatMachineTypeList(
				[]*cmv1.MachineType{machineType})))
			acc, err := amsv1.NewAccount().ID("123456789012").Build()
			Expect(err).ToNot(HaveOccurred())
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatResources(acc)))
			qc, err := amsv1.NewQuotaCost().QuotaID("test-quota").
				OrganizationID("123456789012").Version("4.15.0").Build()
			Expect(err).ToNot(HaveOccurred())
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatQuotaCostList([]*amsv1.QuotaCost{qc})))
			tuningConfig, err := cmv1.NewTuningConfig().ID("test").Build()
			Expect(err).ToNot(HaveOccurred())
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatTuningConfigList(
				[]*cmv1.TuningConfig{tuningConfig})))
			kubeConfig, err := cmv1.NewKubeletConfig().Name("test").ID("test").PodPidsLimit(5000).Build()
			Expect(err).ToNot(HaveOccurred())
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatKubeletConfigList(
				[]*cmv1.KubeletConfig{kubeConfig})))
			flavour, err := cmv1.NewFlavour().AWS(cmv1.NewAWSFlavour().ComputeInstanceType("x-large").
				WorkerVolume(cmv1.NewAWSVolume().Size(100))).
				Network(cmv1.NewNetwork().MachineCIDR("").PodCIDR("").ServiceCIDR("").HostPrefix(1)).Build()
			Expect(err).ToNot(HaveOccurred())
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatResources(flavour)))
			nodePoolObj, err := cmv1.NewNodePool().ID("np-1").Build()
			Expect(err).ToNot(HaveOccurred())
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatResources(nodePoolObj)))
			nodePoolResponse := test.FormatNodePoolList([]*cmv1.NodePool{nodePoolObj})
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolResponse))
			err = machinePool.CreateNodePools(t.RosaRuntime, cmd, clusterKey, cluster, nil, &args)
			Expect(err).To(Not(HaveOccurred()))
		})
		It("should fail if disk size is invalid", func() {
			machinePool := &machinePool{}
			version := "4.15.0"
			az := "a1"

			cluster = returnMockCluster(version)
			privateSubnets := []ec2types.Subnet{
				{AvailabilityZone: &az, SubnetId: &subnet},
			}

			cmd.Flags().StringVar(&args.Name, "name", "", "Name of the machine pool")
			cmd.Flags().Set("name", "test")

			cmd.Flags().StringVar(&args.Version, "version", "", "Version of the machine pool")
			cmd.Flags().Set("version", version)
			isVersionSet := cmd.Flags().Changed("version")
			Expect(isVersionSet).To(BeTrue())

			cmd.Flags().Bool("enable-autoscaling", true, "")
			cmd.Flags().Set("enable-autoscaling", "true")
			cmd.Flags().Int32("min-replicas", 1, "Replicas of the machine pool")
			cmd.Flags().Set("min-replicas", "1")
			cmd.Flags().Int32("max-replicas", 3, "Replicas of the machine pool")
			cmd.Flags().Set("max-replicas", "3")
			args.AutoscalingEnabled = true
			args.InstanceType = "t3.small"
			args.TuningConfigs = "test"
			args.KubeletConfigs = "test"
			args.NodeDrainGracePeriod = "30"
			args.MinReplicas = 1
			args.MaxReplicas = 3

			args.RootDiskSize = "200000000000GB"

			v := cmv1.VersionBuilder{}
			v.ID(version).ChannelGroup("stable").RawID(version).Default(true).
				Enabled(true).ROSAEnabled(true).HostedControlPlaneDefault(true)
			versionObj, err := v.Build()
			Expect(err).ToNot(HaveOccurred())

			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatVersionList(
				[]*cmv1.Version{versionObj})))
			mockClient.EXPECT().GetVPCPrivateSubnets(gomock.Any()).Return(privateSubnets, nil)
			mockClient.EXPECT().GetSubnetAvailabilityZone(subnet).Return(az, nil)
			mtBuilder := cmv1.NewMachineType().ID("t3.small").Name("t3.small")
			machineType, err := mtBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatMachineTypeList(
				[]*cmv1.MachineType{machineType})))
			acc, err := amsv1.NewAccount().ID("123456789012").Build()
			Expect(err).ToNot(HaveOccurred())
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatResources(acc)))
			qc, err := amsv1.NewQuotaCost().QuotaID("test-quota").
				OrganizationID("123456789012").Version("4.15.0").Build()
			Expect(err).ToNot(HaveOccurred())
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatQuotaCostList([]*amsv1.QuotaCost{qc})))
			tuningConfig, err := cmv1.NewTuningConfig().ID("test").Build()
			Expect(err).ToNot(HaveOccurred())
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatTuningConfigList(
				[]*cmv1.TuningConfig{tuningConfig})))
			kubeConfig, err := cmv1.NewKubeletConfig().Name("test").ID("test").PodPidsLimit(5000).Build()
			Expect(err).ToNot(HaveOccurred())
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatKubeletConfigList(
				[]*cmv1.KubeletConfig{kubeConfig})))
			flavour, err := cmv1.NewFlavour().AWS(cmv1.NewAWSFlavour().ComputeInstanceType("x-large").
				WorkerVolume(cmv1.NewAWSVolume().Size(100))).
				Network(cmv1.NewNetwork().MachineCIDR("").PodCIDR("").ServiceCIDR("").HostPrefix(1)).Build()
			Expect(err).ToNot(HaveOccurred())
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatResources(flavour)))
			nodePoolObj, err := cmv1.NewNodePool().ID("np-1").Build()
			Expect(err).ToNot(HaveOccurred())
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatResources(nodePoolObj)))
			err = machinePool.CreateNodePools(t.RosaRuntime, cmd, clusterKey, cluster, nil, &args)
			Expect(err).To(HaveOccurred())

			Expect(err.Error()).To(ContainSubstring("Expected a valid node pool root disk size value"))
		})
	})
})

var _ = Describe("ManageReplicas", func() {
	var cmd *cobra.Command
	var args *mpOpts.CreateMachinepoolUserOptions
	var replicaSizeValidation *ReplicaSizeValidation
	BeforeEach(func() {
		cmd = &cobra.Command{}
		args = &mpOpts.CreateMachinepoolUserOptions{}
		replicaSizeValidation = &ReplicaSizeValidation{
			ClusterVersion: "openshift-v4.14.14",
			MultiAz:        true,
			IsHostedCp:     false,
			Autoscaling:    false,
		}
	})

	When("when autoscaling is enabled", func() {
		It("should not allow setting replicas directly", func() {
			args.AutoscalingEnabled = true
			cmd.Flags().Int32("replicas", 1, "Replicas of the machine pool")
			cmd.Flags().Set("replicas", "1")
			_, _, _, autoscaling, err := manageReplicas(cmd, args, replicaSizeValidation)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Replicas can't be set when autoscaling is enabled"))
			Expect(autoscaling).To(BeTrue())
		})
		It("should pass successfully", func() {
			args.AutoscalingEnabled = true
			cmd.Flags().Int32("min-replicas", 3, "Replicas of the machine pool")
			cmd.Flags().Set("min-replicas", "3")
			cmd.Flags().Int32("max-replicas", 6, "Replicas of the machine pool")
			cmd.Flags().Set("max-replicas", "6")
			args.MinReplicas = 3
			args.MaxReplicas = 6
			_, _, _, _, err := manageReplicas(cmd, args, replicaSizeValidation)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	When("when autoscaling is not enabled", func() {
		It("should not allow setting min and max replicas", func() {
			args.AutoscalingEnabled = false
			cmd.Flags().Int32("min-replicas", 1, "Replicas of the machine pool")
			cmd.Flags().Set("min-replicas", "1")
			cmd.Flags().Int32("max-replicas", 3, "Replicas of the machine pool")
			cmd.Flags().Set("max-replicas", "3")
			args.MinReplicas = 1
			args.MaxReplicas = 3
			_, _, _, autoscaling, err := manageReplicas(cmd, args, replicaSizeValidation)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Autoscaling must be enabled in order to set min and max replicas"))
			Expect(autoscaling).To(BeFalse())
		})
		It("should pass successfully", func() {
			args.AutoscalingEnabled = false
			cmd.Flags().Int32("replicas", 1, "Replicas of the machine pool")
			cmd.Flags().Set("replicas", "1")
			_, _, _, autoscaling, err := manageReplicas(cmd, args, replicaSizeValidation)
			Expect(err).ToNot(HaveOccurred())
			Expect(autoscaling).To(BeFalse())
		})
	})
})

var _ = Describe("Utility Functions", func() {
	Describe("Split function", func() {
		It("should return true for '=' rune", func() {
			Expect(Split('=')).To(BeTrue())
		})

		It("should return true for ':' rune", func() {
			Expect(Split(':')).To(BeTrue())
		})

		It("should return false for any other rune", func() {
			Expect(Split('a')).To(BeFalse())
		})
	})

	Describe("minReplicaValidator function", func() {
		var validator interactive.Validator
		var replicaSizeValidation *ReplicaSizeValidation

		BeforeEach(func() {
			replicaSizeValidation = &ReplicaSizeValidation{
				ClusterVersion: "openshift-v4.14.14",
				MultiAz:        true,
				IsHostedCp:     false,
				Autoscaling:    false,
			}
			validator = replicaSizeValidation.MinReplicaValidator() // or false for non-multiAZ
		})

		It("should return error for non-integer input", func() {
			err := validator("non-integer")
			Expect(err).To(HaveOccurred())
		})

		It("should return error for negative input", func() {
			err := validator(-1)
			Expect(err).To(HaveOccurred())
		})

		It("should return error if not multiple of 3 for multiAZ", func() {
			err := validator(2)
			Expect(err).To(HaveOccurred())
		})

		It("should not return error for valid input", func() {
			err := validator(3)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("maxReplicaValidator function", func() {
		var validator interactive.Validator
		var replicaSizeValidation *ReplicaSizeValidation

		BeforeEach(func() {
			replicaSizeValidation = &ReplicaSizeValidation{
				MinReplicas:    1,
				ClusterVersion: "openshift-v4.14.14",
				MultiAz:        true,
				IsHostedCp:     false,
				Autoscaling:    false,
			}
			validator = replicaSizeValidation.MaxReplicaValidator() // or false for non-multiAZ
		})

		It("should return error for non-integer input", func() {
			err := validator("non-integer")
			Expect(err).To(HaveOccurred())
		})

		It("should return error if maxReplicas less than minReplicas", func() {
			err := validator(0)
			Expect(err).To(HaveOccurred())
		})

		It("should return error if not multiple of 3 for multiAZ", func() {
			err := validator(5)
			Expect(err).To(HaveOccurred())
		})

		It("should not return error for valid input", func() {
			err := validator(3)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("spotMaxPriceValidator function", func() {
		It("should return nil for 'on-demand'", func() {
			err := spotMaxPriceValidator("on-demand")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return error for non-numeric input", func() {
			err := spotMaxPriceValidator("not-a-number")
			Expect(err).To(HaveOccurred())
		})

		It("should return error for negative price", func() {
			err := spotMaxPriceValidator("-1")
			Expect(err).To(HaveOccurred())
		})

		It("should not return error for positive price", func() {
			err := spotMaxPriceValidator("0.01")
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

func NewTestingRuntime(mockClient *mock.MockClient) *TestRuntime {
	t := &TestRuntime{}
	t.InitRuntime(mockClient)
	return t
}

// TestingRuntime is a wrapper for the structure used for testing
type TestRuntime struct {
	SsoServer    *ghttp.Server
	ApiServer    *ghttp.Server
	RosaRuntime  *rosa.Runtime
	StdOutReader stdOutReader
}

func (t *TestRuntime) InitRuntime(mockClient *mock.MockClient) {
	// Create the servers:
	t.SsoServer = MakeTCPServer()
	t.ApiServer = MakeTCPServer()
	t.ApiServer.SetAllowUnhandledRequests(true)
	t.ApiServer.SetUnhandledRequestStatusCode(http.StatusInternalServerError)
	// Create the token:
	accessToken := MakeTokenString("Bearer", 15*time.Minute)

	// Prepare the server:
	t.SsoServer.AppendHandlers(
		RespondWithAccessToken(accessToken),
	)
	// Prepare the logger:
	logger, err := logging.NewGoLoggerBuilder().
		Debug(false).
		Build()
	Expect(err).ToNot(HaveOccurred())
	// Set up the connection with the fake config
	connection, err := sdk.NewConnectionBuilder().
		Logger(logger).
		Tokens(accessToken).
		URL(t.ApiServer.URL()).
		Build()
	// Initialize client object
	Expect(err).ToNot(HaveOccurred())
	ocmClient := ocm.NewClientWithConnection(connection)
	ocm.SetClusterKey("cluster1")
	t.RosaRuntime = rosa.NewRuntime()
	t.RosaRuntime.OCMClient = ocmClient
	t.RosaRuntime.Creator = &mock.Creator{
		ARN:       "fake",
		AccountID: "123",
		IsSTS:     false,
	}
	t.RosaRuntime.AWSClient = mockClient

	DeferCleanup(t.RosaRuntime.Cleanup)
	DeferCleanup(t.SsoServer.Close)
	DeferCleanup(t.ApiServer.Close)
	DeferCleanup(t.Close)
}

func (t *TestRuntime) Close() {
	ocm.SetClusterKey("")
}

func (t *TestRuntime) SetCluster(clusterKey string, cluster *cmv1.Cluster) {
	ocm.SetClusterKey(clusterKey)
	t.RosaRuntime.Cluster = cluster
	t.RosaRuntime.ClusterKey = clusterKey
}

type stdOutReader struct {
	w           *os.File
	r           *os.File
	stdOutState *os.File
}

// Record pipes Stdout to a reader for returning all Stdout output with Read and saves the state of
// stdout to later return to normal. These two functions should be called in series
func (s *stdOutReader) Record() error {
	var err error
	s.stdOutState = os.Stdout
	s.r, s.w, err = os.Pipe()
	os.Stdout = s.w
	return err
}

// Read reads the output using the information gathered from Record, then returns Stdout to printing
// normally at the end of this function using the state captured from Record
func (s *stdOutReader) Read() (string, error) {
	err := s.w.Close()
	if err != nil {
		return "", err
	}
	out, err := io.ReadAll(s.r)
	os.Stdout = s.stdOutState

	return string(out), err
}

func FormatResources(resource interface{}) string {
	var outputJson bytes.Buffer
	var err error
	switch reflect.TypeOf(resource).String() {
	case "*v1.Version":
		if res, ok := resource.(*cmv1.Version); ok {
			err = cmv1.MarshalVersion(res, &outputJson)
		}
	case "*v1.Account":
		if res, ok := resource.(*amsv1.Account); ok {
			err = amsv1.MarshalAccount(res, &outputJson)
		}
	case "*v1.MachinePool":
		if res, ok := resource.(*cmv1.MachinePool); ok {
			err = cmv1.MarshalMachinePool(res, &outputJson)
		}
	case "*v1.NodePool":
		if res, ok := resource.(*cmv1.NodePool); ok {
			err = cmv1.MarshalNodePool(res, &outputJson)
		}
	case "*v1.Flavour":
		if res, ok := resource.(*cmv1.Flavour); ok {
			err = cmv1.MarshalFlavour(res, &outputJson)
		}
	default:
		{
			return "NOTIMPLEMENTED"
		}
	}
	if err != nil {
		return err.Error()
	}

	return outputJson.String()
}
