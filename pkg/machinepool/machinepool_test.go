package machinepool

import (
	"bytes"
	// "encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	reflect "reflect"

	// "net/url"
	"time"

	gomock "go.uber.org/mock/gomock"

	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	sdk "github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift-online/ocm-sdk-go/logging"
	. "github.com/openshift-online/ocm-sdk-go/testing"
	"github.com/spf13/cobra"

	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	mock "github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/securitygroups"
	"github.com/openshift/rosa/pkg/ocm"
	ocmOutput "github.com/openshift/rosa/pkg/ocm/output"
	mpOpts "github.com/openshift/rosa/pkg/options/machinepool"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/test"
)

var policyBuilder cmv1.NodePoolUpgradePolicyBuilder
var date time.Time

var _ = Describe("Machinepool and nodepool", func() {
	var (
		mockClient           *mock.MockClient
		mockCtrl             *gomock.Controller
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
			subnetUserInput = "subnet-123"
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
			subnetId1 := "subnet-123"
			subnetId2 := "subnet-456"

			// Mocking private subnet retrieval
			privateSubnets := []ec2types.Subnet{
				{AvailabilityZone: &az, SubnetId: &subnetId1},
			}
			mockClient.EXPECT().GetVPCPrivateSubnets(gomock.Any()).Return(privateSubnets, nil)

			// Building a mock cluster
			clusterBuilder := cmv1.NewCluster().ID("test-cluster").State(cmv1.ClusterStateReady).
				Nodes(cmv1.NewClusterNodes().AvailabilityZones("us-east-1a")).AWS(cmv1.NewAWS().SubnetIDs(subnetId1, subnetId2))
			cluster, err := clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())

			// Test when no availability zone is set and only one subnet is returned
			subnet, err := getSubnetFromAvailabilityZone(cmd, r, isAvailabilityZoneSet, cluster, args)
			Expect(err).ToNot(HaveOccurred())
			Expect(subnet).To(Equal("subnet-123"))
		})
		It("Tests error case for getSubnetFromAvailabilityZone", func() {
			r := &rosa.Runtime{AWSClient: mockClient}
			cmd := &cobra.Command{}
			isAvailabilityZoneSet := true 
			args := &mpOpts.CreateMachinepoolUserOptions{
				AvailabilityZone: "us-west-1a",
			}
			az := "us-east-1a"
			subnetId1 := "subnet-123"

			privateSubnets := []ec2types.Subnet{
				{AvailabilityZone: &az, SubnetId: &subnetId1},
			}
			mockClient.EXPECT().GetVPCPrivateSubnets(gomock.Any()).Return(privateSubnets, nil)

			// Building a mock cluster
			clusterBuilder := cmv1.NewCluster().ID("test-cluster").State(cmv1.ClusterStateReady).
				Nodes(cmv1.NewClusterNodes().AvailabilityZones(az)).AWS(cmv1.NewAWS().SubnetIDs(subnetId1))
			cluster, err := clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())

			// Attempt to get a subnet from a non-existent availability zone
			subnet, err := getSubnetFromAvailabilityZone(cmd, r, isAvailabilityZoneSet, cluster, args)
			Expect(err).To(HaveOccurred()) 
			Expect(subnet).To(Equal(""))
		})
	})
})

var _ = Describe("MachinePools", func() {
	Context("AddMachinePool validation errors", func() {
		var (
			cmd                  *cobra.Command
			clusterKey           string
			args                 mpOpts.CreateMachinepoolUserOptions
			cluster              *cmv1.Cluster
			err                  error
			t                    *TestingRuntime
			mockClient           *mock.MockClient
			mockCtrl             *gomock.Controller
		)

		JustBeforeEach(func() {
			t = NewTestRuntime()
			args = mpOpts.CreateMachinepoolUserOptions{}
			clusterKey = "test-cluster-key"
			cmd = &cobra.Command{}
			mockCtrl = gomock.NewController(GinkgoT())
			mockClient = mock.NewMockClient(mockCtrl)
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
			cmd.Flags().StringSliceVar(&args.SecurityGroupIds, "additional-security-group-ids", []string{}, "comma-separated list of security group IDs")
			cmd.Flags().Set("additional-security-group-ids", "sg-12345")
			compatibleVersion := "4.15.0"
			clusterBuilder := cmv1.NewCluster().ID("test").State(cmv1.ClusterStateReady).Version(cmv1.NewVersion().RawID(compatibleVersion))
			cluster, err = clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			err = machinePool.CreateMachinePool(t.RosaRuntime, cmd, clusterKey, cluster, &args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf("Setting the `%s` flag is only allowed for BYOVPC clusters", securitygroups.MachinePoolSecurityGroupFlag)))
		})

		It("should error checking version compatibility", func() {
			machinePool := &machinePool{}
			incompatibleVersion := "2.5.0"
			clusterBuilder := cmv1.NewCluster().ID("test").State(cmv1.ClusterStateReady).Version(cmv1.NewVersion().ID(incompatibleVersion))
			cluster, err = clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())

			err = machinePool.CreateMachinePool(t.RosaRuntime, cmd, clusterKey, cluster, &args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("There was a problem checking version compatibility:")))
		})

		It("should error when setting flag that is only allowed for BYOVPC clusters", func() {
			machinePool := &machinePool{}
			incompatibleVersion := "2.5.0"
			cmd.Flags().StringSliceVar(&args.SecurityGroupIds, "additional-security-group-ids", []string{}, "comma-separated list of security group IDs")
			cmd.Flags().Set("additional-security-group-ids", "sg-12345")
			clusterBuilder := cmv1.NewCluster().ID("test").State(cmv1.ClusterStateReady).Version(cmv1.NewVersion().RawID(incompatibleVersion))
			cluster, err = clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())

			err = machinePool.CreateMachinePool(t.RosaRuntime, cmd, clusterKey, cluster, &args)
			Expect(err).To(HaveOccurred())

			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Setting the `%s` flag is only allowed for BYOVPC clusters", securitygroups.MachinePoolSecurityGroupFlag)))
		})

		It("should error when the security group IDs flag is set for clusters with incompatible versions", func() {
			machinePool := &machinePool{}
			incompatibleVersion := "2.5.0"
			cmd.Flags().StringSliceVar(&args.SecurityGroupIds, "additional-security-group-ids", []string{}, "comma-separated list of security group IDs")
			cmd.Flags().Set("additional-security-group-ids", "sg-12345")

			subnetIDs := []string{"subnet-12345"}
			awsBuilder := cmv1.NewAWS().SubnetIDs(subnetIDs...)
			clusterBuilder := cmv1.NewCluster().ID("test").State(cmv1.ClusterStateReady).AWS(awsBuilder).Version(cmv1.NewVersion().RawID(incompatibleVersion))
			cluster, err = clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())

			err = machinePool.CreateMachinePool(t.RosaRuntime, cmd, clusterKey, cluster, &args)
			Expect(err).To(HaveOccurred())

			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Parameter '%s' is not supported prior to version ", securitygroups.MachinePoolSecurityGroupFlag)))
		})

		It("should error when both 'subnet' and 'availability-zone' flags are set", func() {
			machinePool := &machinePool{}
			compatibleVersion := "4.15.0"
			cmd.Flags().StringSliceVar(&args.SecurityGroupIds, "additional-security-group-ids", []string{}, "comma-separated list of security group IDs")
			cmd.Flags().Set("additional-security-group-ids", "sg-12345")

			subnetIDs := []string{"subnet-12345"}
			awsBuilder := cmv1.NewAWS().SubnetIDs(subnetIDs...)
			clusterBuilder := cmv1.NewCluster().ID("test").State(cmv1.ClusterStateReady).AWS(awsBuilder).Version(cmv1.NewVersion().RawID(compatibleVersion)).MultiAZ(true)
			cluster, err = clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			cmd.Flags().StringVar(&args.Subnet, "subnet", "", "")
			cmd.Flags().Set("subnet", "test-subnet")
			cmd.Flags().StringVar(&args.AvailabilityZone, "availability-zone", "", "")
			cmd.Flags().Set("availability-zone", "az")
			err = machinePool.CreateMachinePool(t.RosaRuntime, cmd, clusterKey, cluster, &args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Setting both `subnet` and `availability-zone` flag is not supported. Please select `subnet` or `availability-zone` to create a single availability zone machine pool"))
		})

		It("should error when 'availability-zone' flag is set for a single AZ machine pool in a multi-AZ cluster", func() {
			machinePool := &machinePool{}
			compatibleVersion := "4.15.0"
			cmd.Flags().StringSliceVar(&args.SecurityGroupIds, "additional-security-group-ids", []string{}, "comma-separated list of security group IDs")
			cmd.Flags().Set("additional-security-group-ids", "sg-12345")

			subnetIDs := []string{"subnet-12345"}
			awsBuilder := cmv1.NewAWS().SubnetIDs(subnetIDs...)
			clusterBuilder := cmv1.NewCluster().ID("test").State(cmv1.ClusterStateReady).AWS(awsBuilder).Version(cmv1.NewVersion().RawID(compatibleVersion)).MultiAZ(true)
			cluster, err = clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			cmd.Flags().Bool("multi-availability-zone", true, "")
			cmd.Flags().Set("multi-availability-zone", "true")
			cmd.Flags().StringVar(&args.AvailabilityZone, "availability-zone", "", "")
			cmd.Flags().Set("availability-zone", "az")
			args.MultiAvailabilityZone = true
			err = machinePool.CreateMachinePool(t.RosaRuntime, cmd, clusterKey, cluster, &args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Setting the `availability-zone` flag is only supported for creating a single AZ machine pool in a multi-AZ cluster"))
		})

		It("should error when setting an invalid name", func() {
			machinePool := &machinePool{}
			compatibleVersion := "4.15.0"
			subnetIDs := []string{"subnet-12345"}
			awsBuilder := cmv1.NewAWS().SubnetIDs(subnetIDs...)
			clusterBuilder := cmv1.NewCluster().ID("test").State(cmv1.ClusterStateReady).AWS(awsBuilder).Version(cmv1.NewVersion().RawID(compatibleVersion)).MultiAZ(true)
			cluster, err = clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			cmd.Flags().StringVar(&args.Name, "name", "", "Name of the machine pool")
			invalidName := "998 .-"
			cmd.Flags().Set("name", invalidName)
			cmd.Flags().Bool("multi-availability-zone", true, "")
			cmd.Flags().Set("multi-availability-zone", "true")
			err = machinePool.CreateMachinePool(t.RosaRuntime, cmd, clusterKey, cluster, &args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected a valid name for the machine pool"))
		})

		It("should error when autoscaling and replicas are enabled", func() {
			machinePool := &machinePool{}
			compatibleVersion := "4.15.0"
			subnetIDs := []string{"subnet-12345"}
			awsBuilder := cmv1.NewAWS().SubnetIDs(subnetIDs...)
			nodeBuilder := cmv1.NewClusterNodes().AvailabilityZones("a1")
			clusterBuilder := cmv1.NewCluster().ID("test").State(cmv1.ClusterStateReady).AWS(awsBuilder).Version(cmv1.NewVersion().RawID(compatibleVersion)).MultiAZ(true).Nodes(nodeBuilder)
			cluster, err = clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			cmd.Flags().StringVar(&args.Name, "name", "", "Name of the machine pool")
			cmd.Flags().Set("name", "mp-1")
			cmd.Flags().Bool("multi-availability-zone", true, "")
			cmd.Flags().Set("multi-availability-zone", "true")
			cmd.Flags().Bool("enable-autoscaling", true, "")
			cmd.Flags().Set("enable-autoscaling", "true")
			cmd.Flags().IntVar(&args.Replicas, "replicas", 0, "Replicas of the machine pool")
			cmd.Flags().Set("replicas", "3")
			args.AutoscalingEnabled = true
			
			err = machinePool.CreateMachinePool(t.RosaRuntime, cmd, clusterKey, cluster, &args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Replicas can't be set when autoscaling is enabled"))
		})

		It("should error when not supplying an instance type", func() {
			machinePool := &machinePool{}
			compatibleVersion := "4.15.0"
			subnetIDs := []string{"subnet-12345"}
			awsBuilder := cmv1.NewAWS().SubnetIDs(subnetIDs...)
			nodeBuilder := cmv1.NewClusterNodes().AvailabilityZones("a1")
			clusterBuilder := cmv1.NewCluster().ID("test").State(cmv1.ClusterStateReady).AWS(awsBuilder).Version(cmv1.NewVersion().RawID(compatibleVersion)).MultiAZ(true).Nodes(nodeBuilder)
			cluster, err = clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			cmd.Flags().StringVar(&args.Name, "name", "", "Name of the machine pool")
			cmd.Flags().Set("name", "mp-1")
			cmd.Flags().Bool("multi-availability-zone", true, "")
			cmd.Flags().Set("multi-availability-zone", "true")
			cmd.Flags().Bool("enable-autoscaling", true, "")
			cmd.Flags().Set("enable-autoscaling", "true")
			cmd.Flags().Int32("min-replicas", 0, "Replicas of the machine pool")
			cmd.Flags().Set("min-replicas", "1")
			cmd.Flags().Int32("max-replicas", 0, "Replicas of the machine pool")
			cmd.Flags().Set("max-replicas", "3")
			args.AutoscalingEnabled = true
			err = machinePool.CreateMachinePool(t.RosaRuntime, cmd, clusterKey, cluster, &args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("You must supply a valid instance type"))
		})

		It("should error when not supplying min and max replicas but not autoscaling", func() {
			machinePool := &machinePool{}
			compatibleVersion := "4.15.0"
			subnetIDs := []string{"subnet-12345"}
			awsBuilder := cmv1.NewAWS().SubnetIDs(subnetIDs...)
			nodeBuilder := cmv1.NewClusterNodes().AvailabilityZones("a1")
			clusterBuilder := cmv1.NewCluster().ID("test").State(cmv1.ClusterStateReady).AWS(awsBuilder).Version(cmv1.NewVersion().RawID(compatibleVersion)).MultiAZ(true).Nodes(nodeBuilder)
			cluster, err = clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			cmd.Flags().StringVar(&args.Name, "name", "", "Name of the machine pool")
			cmd.Flags().Set("name", "mp-1")
			cmd.Flags().Bool("multi-availability-zone", true, "")
			cmd.Flags().Set("multi-availability-zone", "true")
			cmd.Flags().Int32("min-replicas", 0, "Replicas of the machine pool")
			cmd.Flags().Set("min-replicas", "1")
			cmd.Flags().Int32("max-replicas", 0, "Replicas of the machine pool")
			cmd.Flags().Set("max-replicas", "3")
			err = machinePool.CreateMachinePool(t.RosaRuntime, cmd, clusterKey, cluster, &args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Autoscaling must be enabled in order to set min and max replicas"))
		})

		It("should error when not providing a valid instance type", func() {
			machinePool := &machinePool{}
			
			compatibleVersion := "4.15.0"
			subnetIDs := []string{"subnet-12345"}
			region := "us-east-1"
			regionBuilder := cmv1.NewCloudRegion().Name(region).ID(region)
			cloudProvider := cmv1.NewCloudProvider().ID("aws").Regions(regionBuilder)
			stsBuilder := cmv1.NewSTS().RoleARN("arn:aws:iam::123456789012:role/SampleRole")
			awsBuilder := cmv1.NewAWS().SubnetIDs(subnetIDs...).STS(stsBuilder).AccountID("123456789012")
			nodeBuilder := cmv1.NewClusterNodes().AvailabilityZones("a1")
			clusterBuilder := cmv1.NewCluster().ID("cluster-test").State(cmv1.ClusterStateReady).AWS(awsBuilder).Version(cmv1.NewVersion().RawID(compatibleVersion)).MultiAZ(true).Nodes(nodeBuilder).Region(regionBuilder).CloudProvider(cloudProvider)
			cluster, err = clusterBuilder.Build()
		
			Expect(err).ToNot(HaveOccurred())
			cmd.Flags().StringVar(&args.Name, "name", "", "Name of the machine pool")
			cmd.Flags().Set("name", "mp-1")
			cmd.Flags().Bool("multi-availability-zone", true, "")
			cmd.Flags().Set("multi-availability-zone", "true")
			cmd.Flags().IntVar(&args.Replicas, "replicas", 0, "Replicas of the machine pool")
			cmd.Flags().Set("replicas", "3")
			args.InstanceType = "test"
			machineTypeList, err := cmv1.NewMachineTypeList().Items(cmv1.NewMachineType().ID("t3.small").CloudProvider(cmv1.NewCloudProvider().ID("aws").Regions(cmv1.NewCloudRegion().ID(region).Name(region)))).Build()
			Expect(err).ToNot(HaveOccurred())
			acc, err := amsv1.NewAccount().ID("123456789012").Build()
			Expect(err).ToNot(HaveOccurred())
			sq, err := amsv1.NewQuotaCostList().Items(amsv1.NewQuotaCost().QuotaID("test").OrganizationID("123456789012").Version("4.15.0")).Build()
			Expect(err).ToNot(HaveOccurred())
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatClusterList([]*cmv1.Cluster{cluster})))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatResource(machineTypeList)))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatResource(acc)))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatResource(sq)))
			err = machinePool.CreateMachinePool(t.RosaRuntime, cmd, clusterKey, cluster, &args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Expected a valid instance type"))
		})

		It("Should error when can't set max price when not using spot instances", func() {
			machinePool := &machinePool{}
			compatibleVersion := "4.15.0"
			subnetIDs := []string{"subnet-12345"}
			region := "us-east-1"
			regionBuilder := cmv1.NewCloudRegion().Name(region).ID(region)
			cloudProvider := cmv1.NewCloudProvider().ID("aws").Regions(regionBuilder)
			stsBuilder := cmv1.NewSTS().RoleARN("arn:aws:iam::123456789012:role/SampleRole")
			awsBuilder := cmv1.NewAWS().SubnetIDs(subnetIDs...).STS(stsBuilder).AccountID("123456789012")
			mtBuilder := cmv1.NewMachineType().ID("t3.small").Name("t3.small").CloudProvider(cloudProvider)
			nodeBuilder := cmv1.NewClusterNodes().AvailabilityZones("a1")
			clusterBuilder := cmv1.NewCluster().ID("test").State(cmv1.ClusterStateReady).AWS(awsBuilder).Version(cmv1.NewVersion().RawID(compatibleVersion)).MultiAZ(true).Nodes(nodeBuilder).Region(regionBuilder).CloudProvider(cloudProvider)
			cluster, err = clusterBuilder.Build()
		
			Expect(err).ToNot(HaveOccurred())
			cmd.Flags().StringVar(&args.Name, "name", "", "Name of the machine pool")
			cmd.Flags().Set("name", "mp-1")
			cmd.Flags().Bool("multi-availability-zone", true, "")
			cmd.Flags().Set("multi-availability-zone", "true")
			cmd.Flags().IntVar(&args.Replicas, "replicas", 0, "Replicas of the machine pool")
			cmd.Flags().Set("replicas", "3")
			cmd.Flags().BoolVar(&args.UseSpotInstances, "use-spot-instances", false, "")
			cmd.Flags().Set("use-spot-instances", "false")
			cmd.Flags().Changed("use-spot-instances")
			cmd.Flags().StringVar(&args.SpotMaxPrice, "spot-max-price", "0.01", "")
			cmd.Flags().Set("spot-max-price", "0.01")
			args.InstanceType = "test"
			machineTypeList, err := cmv1.NewMachineTypeList().Items(mtBuilder).Build()
			Expect(err).ToNot(HaveOccurred())
			acc, err := amsv1.NewAccount().ID("123456789012").Build()
			Expect(err).ToNot(HaveOccurred())
			qcList, err := amsv1.NewQuotaCostList().Items(amsv1.NewQuotaCost().QuotaID("test-quota").OrganizationID("123456789012").Version("4.15.0")).Build()
			Expect(err).ToNot(HaveOccurred())
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatClusterList([]*cmv1.Cluster{cluster})))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatResource(machineTypeList)))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatResource(acc)))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatResource(qcList)))
			err = machinePool.CreateMachinePool(t.RosaRuntime, cmd, clusterKey, cluster, &args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Can't set max price when not using spot instances"))
		})
		It("Should error when instances are set for local zones", func() {
			machinePool := &machinePool{}
			compatibleVersion := "4.15.0"
			region := "us-east-1"
			subnet := "subnet-12345"
			args.Subnet = subnet
			regionBuilder := cmv1.NewCloudRegion().Name(region).ID(region)
			cloudProvider := cmv1.NewCloudProvider().ID("aws").Regions(regionBuilder)
			stsBuilder := cmv1.NewSTS().RoleARN("arn:aws:iam::123456789012:role/SampleRole")
			awsBuilder := cmv1.NewAWS().STS(stsBuilder).AccountID("123456789012").SubnetIDs(subnet)
			mtBuilder := cmv1.NewMachineType().ID("t3.small").Name("t3.small").CloudProvider(cloudProvider)
			nodeBuilder := cmv1.NewClusterNodes().AvailabilityZones("a1")
			clusterBuilder := cmv1.NewCluster().ID("test").State(cmv1.ClusterStateReady).AWS(awsBuilder).Version(cmv1.NewVersion().RawID(compatibleVersion)).MultiAZ(true).Nodes(nodeBuilder).Region(regionBuilder).CloudProvider(cloudProvider)
			cluster, err = clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())

			cmd.Flags().StringVar(&args.Name, "name", "", "Name of the machine pool")
			cmd.Flags().Set("name", "mp-1")
			cmd.Flags().Bool("multi-availability-zone", true, "")
			cmd.Flags().Set("multi-availability-zone", "true")
			cmd.Flags().IntVar(&args.Replicas, "replicas", 0, "Replicas of the machine pool")
			cmd.Flags().Set("replicas", "3")
			args.InstanceType = "test"
			machineTypeList, err := cmv1.NewMachineTypeList().Items(mtBuilder).Build()
			Expect(err).ToNot(HaveOccurred())
			acc, err := amsv1.NewAccount().ID("123456789012").Build()
			Expect(err).ToNot(HaveOccurred())
			qcList, err := amsv1.NewQuotaCostList().Items(amsv1.NewQuotaCost().QuotaID("test-quota").OrganizationID("123456789012").Version("4.15.0")).Build()
			Expect(err).ToNot(HaveOccurred())
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatClusterList([]*cmv1.Cluster{cluster})))
			// t.RosaRuntime.AWSClient.EXPECT().GetSubnetAvailabilityZone(subnet).Return(region, nil)
			t.RosaRuntime.AWSClient.Expect().GetSubnetAvailabilityZone(subnet).Return(region, nil)
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatResource(machineTypeList)))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatResource(acc)))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatResource(qcList)))
			mockClient.EXPECT().IsLocalAvailabilityZone(region).Return(true, nil).Times(1)
			err = machinePool.CreateMachinePool(t.RosaRuntime, cmd, clusterKey, cluster, &args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Spot instances are not supported for local zones"))
		})
	})
})

var _ = Describe("NodePools", func() {
	Context("AddNodePool validation errors", func() {
		var (
			cmd                  *cobra.Command
			clusterKey           string
			args                 mpOpts.CreateMachinepoolUserOptions
			cluster              *cmv1.Cluster
			err                  error
			t                    *TestingRuntime
			//mockClient           *mock.MockClient
			//mockCtrl             *gomock.Controller
		)

		JustBeforeEach(func() {
			t = NewTestRuntime()
			args = mpOpts.CreateMachinepoolUserOptions{}
			clusterKey = "test-cluster-key"
			cmd = &cobra.Command{}
			//mockCtrl = gomock.NewController(GinkgoT())
			//mockClient = mock.NewMockClient(mockCtrl)
		})

		It("should return an error if both `subnet` and `availability-zone` flags are set", func() {
			cmd.Flags().Bool("availability-zone", true, "")
			cmd.Flags().Bool("subnet", true, "")
			clusterBuilder := cmv1.NewCluster().ID("test").State(cmv1.ClusterStateReady)
			cluster, err = clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())

			cmd.Flags().Set("availability-zone", "true")
			cmd.Flags().Set("subnet", "true")

			machinePool := &machinePool{}
			err = machinePool.CreateNodePools(t.RosaRuntime, cmd, clusterKey, cluster, &args)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Setting both `subnet` and " +
				"`availability-zone` flag is not supported. Please select `subnet` " +
				"or `availability-zone` to create a single availability zone machine pool"))
		})
		It("should fail name validation", func() {

			machinePool := &machinePool{}

			clusterKey := "test-cluster-key"
			clusterBuilder := cmv1.NewCluster().ID("test").State(cmv1.ClusterStateReady)
			cluster, err := clusterBuilder.Build()
			Expect(err).ToNot(HaveOccurred())

			cmd.Flags().StringVar(&args.Name, "name", "", "Name of the machine pool")
			invalidName := "0909+===..3"
			cmd.Flags().Set("name", invalidName)

			err = machinePool.CreateNodePools(t.RosaRuntime, cmd, clusterKey, cluster, &args)
			Expect(err).To(HaveOccurred())

			Expect(err.Error()).To(Equal("Expected a valid name for the machine pool"))
		})
		// It("", func() {
		// 	version := "4.15.0"
		// 	machinePool := &machinePool{}
		// 	t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatClusterList([]*cmv1.Cluster{cluster})))
		// 	clusterKey := "test-cluster-key"
		// 	awsBuilder := cmv1.NewAWS().STS(cmv1.NewSTS().Enabled(false))
		// 	versionBuilder := cmv1.NewVersion().ID(version).ChannelGroup("stable").RawID(version).Default(true).Enabled(true).ROSAEnabled(true).RawID(version)
		// 	clusterBuilder := cmv1.NewCluster().ID("test").State(cmv1.ClusterStateReady).Version(versionBuilder).AWS(awsBuilder)
		// 	cluster, err := clusterBuilder.Build()
		// 	Expect(err).ToNot(HaveOccurred())

		// 	cmd.Flags().StringVar(&args.Name, "name", "", "Name of the machine pool")
		// 	cmd.Flags().Set("name", "test")

		// 	cmd.Flags().StringVar(&args.Version, "version", "", "Version of the machine pool")
		// 	cmd.Flags().Set("version", version)
		// 	isVersionSet := cmd.Flags().Changed("version")
		// 	Expect(isVersionSet).To(BeTrue())
		// 	versionList := cmv1.NewVersionList().Items(cmv1.NewVersion().ID(version).ChannelGroup("stable").RawID(version).Default(true).Enabled(true).ROSAEnabled(true))
		// 	versionList.Build()
		// 	t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatResource(versionList)))
		// 	err = machinePool.CreateNodePools(t.RosaRuntime, cmd, clusterKey, cluster, &args)
		// 	Expect(err).To(HaveOccurred())

		// 	//Expect(err.Error()).To(Equal("Expected a valid name for the machine pool"))
		// })
		// It("", func() {
		// 	version := "4.15.0"
		// 	machinePool := &machinePool{}
		// 	t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatClusterList([]*cmv1.Cluster{cluster})))
		// 	clusterKey := "test-cluster-key"
		// 	awsBuilder := cmv1.NewAWS().STS(cmv1.NewSTS().Enabled(false))
		// 	versionBuilder := cmv1.NewVersion().ID(version).ChannelGroup("stable").RawID(version).Default(true).Enabled(true).ROSAEnabled(true).RawID(version)
		// 	clusterBuilder := cmv1.NewCluster().ID("test").State(cmv1.ClusterStateReady).Version(versionBuilder).AWS(awsBuilder)
		// 	cluster, err := clusterBuilder.Build()
		// 	Expect(err).ToNot(HaveOccurred())

		// 	cmd.Flags().StringVar(&args.Name, "name", "", "Name of the machine pool")
		// 	cmd.Flags().Set("name", "test")

		// 	cmd.Flags().StringVar(&args.Subnet, "subnet", "", "Subnet of the machine pool")
		// 	cmd.Flags().Set("subnet", "subnet-test")

		// 	cmd.Flags().Int32("min-replicas", 0, "Replicas of the machine pool")
		// 	cmd.Flags().Set("min-replicas", "1")
		// 	cmd.Flags().Int32("max-replicas", 0, "Replicas of the machine pool")
		// 	cmd.Flags().Set("max-replicas", "3")

			
		// 	err = machinePool.CreateNodePools(t.RosaRuntime, cmd, clusterKey, cluster, &args)
		// 	Expect(err).To(HaveOccurred())

		// 	//Expect(err.Error()).To(Equal("Expected a valid name for the machine pool"))
		// })
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

		BeforeEach(func() {
			validator = minReplicaValidator(true) // or false for non-multiAZ
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

		BeforeEach(func() {
			validator = maxReplicaValidator(1, true)
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

func NewTestRuntime() *TestingRuntime {
	t := &TestingRuntime{}
	t.InitRuntime()
	return t
}
// TestingRuntime is a wrapper for the structure used for testing
type TestingRuntime struct {
	SsoServer    *ghttp.Server
	ApiServer    *ghttp.Server
	RosaRuntime  *rosa.Runtime
	StdOutReader stdOutReader
}

func (t *TestingRuntime) InitRuntime() {
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
	Expect(err).To(BeNil())
	// Set up the connection with the fake config
	connection, err := sdk.NewConnectionBuilder().
		Logger(logger).
		Tokens(accessToken).
		URL(t.ApiServer.URL()).
		Build()
	// Initialize client object
	Expect(err).To(BeNil())
	ocmClient := ocm.NewClientWithConnection(connection)
	mockCtrl := gomock.NewController(GinkgoT())
	mockCtrl = gomock.NewController(GinkgoT())
	mockClient := mock.NewMockClient(mockCtrl)
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

func (t *TestingRuntime) Close() {
	ocm.SetClusterKey("")
}

func (t *TestingRuntime) SetCluster(clusterKey string, cluster *cmv1.Cluster) {
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

func FormatResource(resource interface{}) string {
	var outputJson bytes.Buffer
	var err error
	switch reflect.TypeOf(resource).String() {
	case "*v1.MachineTypeList":
		if res, ok := resource.([]*cmv1.MachineType); ok {
			err = cmv1.MarshalMachineTypeList(res, &outputJson)
		}
	case "*v1.VersionList":
		if res, ok := resource.([]*cmv1.Version); ok {
			err = cmv1.MarshalVersionList(res, &outputJson)
		}
	case "*v1.Account":
		if res, ok := resource.(*amsv1.Account); ok {
			err = amsv1.MarshalAccount(res, &outputJson)
		}
	case "*v1.QuotaCostList":
		if res, ok := resource.([]*amsv1.QuotaCost); ok {
			err = amsv1.MarshalQuotaCostList(res, &outputJson)
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
