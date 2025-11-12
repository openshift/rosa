package machinepool

import (
	"fmt"

	"go.uber.org/mock/gomock"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	mock "github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/helper/features"
	mpOpts "github.com/openshift/rosa/pkg/options/machinepool"
	"github.com/openshift/rosa/pkg/reporter"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/test"
	. "github.com/openshift/rosa/pkg/test"
)

var _ = Describe("Machine pool helper", func() {
	Context("Validates cluster's subnet list isn't empty", func() {
		var r *rosa.Runtime
		var cmd *cobra.Command

		aws := cmv1.NewAWS()
		cluster, err := cmv1.NewCluster().AWS(aws).Build()
		Expect(err).ToNot(HaveOccurred())

		It("should return an error if subnets list is empty", func() {
			_, err := getSecurityGroupsOption(r, cmd, cluster)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(
				"expected cluster's subnets to contain subnets IDs, but got an empty list"))
		})

		It("Should return an error is subnet is missing the VPC ID", func() {
			subnet := ec2types.Subnet{}

			_, err := getVpcIdFromSubnet(subnet)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(
				"unexpected situation a VPC ID should have been selected based on chosen subnets"))
		})

		It("Should return VPC ID from the subnet object", func() {
			vpcId := "123"
			subnet := ec2types.Subnet{
				VpcId: &vpcId,
			}

			vpcId, err := getVpcIdFromSubnet(subnet)
			Expect(err).ToNot(HaveOccurred())
			Expect(vpcId).To(Equal("123"))
		})
	})

	Context("It create an AWS node pool builder successfully", func() {
		It("Create AWS node pool with aws tags when provided", func() {
			instanceType := "123"
			securityGroupIds := []string{"123"}
			awsTags := map[string]string{"label": "value"}

			awsNpBuilder := createAwsNodePoolBuilder(
				instanceType,
				securityGroupIds,
				"optional",
				awsTags,
				nil,
			)
			awsNodePool, err := awsNpBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			Expect(awsNodePool.AdditionalSecurityGroupIds()).To(Equal(securityGroupIds))
			Expect(awsNodePool.InstanceType()).To(Equal(instanceType))
			Expect(awsNodePool.Tags()).To(Equal(awsTags))
		})
		It("Create AWS node pool with security group IDs when provided", func() {
			instanceType := "123"
			securityGroupIds := []string{"123"}

			awsNpBuilder := createAwsNodePoolBuilder(
				instanceType,
				securityGroupIds,
				"optional",
				map[string]string{},
				nil,
			)
			awsNodePool, err := awsNpBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			Expect(awsNodePool.AdditionalSecurityGroupIds()).To(Equal(securityGroupIds))
			Expect(awsNodePool.InstanceType()).To(Equal(instanceType))
			Expect(awsNodePool.Tags()).To(HaveLen(0))
		})
		It("Create AWS node pool without security group IDs if not provided", func() {
			instanceType := "123"

			awsNpBuilder := createAwsNodePoolBuilder(
				instanceType,
				[]string{},
				"optional",
				map[string]string{},
				nil,
			)
			awsNodePool, err := awsNpBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			Expect(awsNodePool.AdditionalSecurityGroupIds()).To(HaveLen(0))
			Expect(awsNodePool.InstanceType()).To(Equal(instanceType))
			Expect(awsNodePool.Tags()).To(HaveLen(0))
		})
		It("Create AWS node pool with aws tags when provided", func() {
			instanceType := "123"
			securityGroupIds := []string{"123"}
			npSize := 300

			awsNpBuilder := createAwsNodePoolBuilder(
				instanceType,
				securityGroupIds,
				"optional",
				map[string]string{},
				&npSize,
			)
			awsNodePool, err := awsNpBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			Expect(awsNodePool.AdditionalSecurityGroupIds()).To(Equal(securityGroupIds))
			Expect(awsNodePool.InstanceType()).To(Equal(instanceType))
			Expect(awsNodePool.RootVolume().Size()).To(Equal(300))
		})
	})

	Context("It validate version is compatible for security groups", func() {
		It("Skips validation if the version isn't provided", func() {
			version := ""
			isCompatible, err := features.IsFeatureSupported(features.AdditionalDay2SecurityGroupsHcpFeature, version)
			Expect(err).ToNot(HaveOccurred())
			Expect(isCompatible).To(BeTrue())
		})
		It("Returns false for 4.14.0", func() {
			version := "4.14.0"
			isCompatible, err := features.IsFeatureSupported(features.AdditionalDay2SecurityGroupsHcpFeature, version)
			Expect(err).ToNot(HaveOccurred())
			Expect(isCompatible).To(BeFalse())
		})
		It("Returns true for 4.15.0", func() {
			version := "4.15.0"
			isCompatible, err := features.IsFeatureSupported(features.AdditionalDay2SecurityGroupsHcpFeature, version)
			Expect(err).ToNot(HaveOccurred())
			Expect(isCompatible).To(BeTrue())
		})
	})

	Context("getSubnetFromUser", func() {
		r := &rosa.Runtime{}
		args := &mpOpts.CreateMachinepoolUserOptions{}
		cmd := &cobra.Command{}
		mockClusterReady := test.MockCluster(func(c *cmv1.ClusterBuilder) {
			c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
			c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
			c.State(cmv1.ClusterStateReady)
			c.Hypershift(cmv1.NewHypershift().Enabled(true))
			c.ExternalAuthConfig(cmv1.NewExternalAuthConfig().Enabled(true))
		})
		It("Should return the subnet if it's set", func() {
			cmd.Flags().StringVar(&args.Subnet, "subnet", "", "")
			cmd.Flags().Set("subnet", "test-subnet")
			output, err := getSubnetFromUser(cmd, r, true, mockClusterReady, args)
			Expect(err).ToNot(HaveOccurred())
			Expect(output).To(Equal("test-subnet"))
		})
	})
})

var _ = Describe("getMachinePoolAvailabilityZones Functionality", func() {
	var (
		cluster    *cmv1.Cluster
		r          *rosa.Runtime
		mockClient *mock.MockClient
		mockCtrl   *gomock.Controller
	)

	BeforeEach(func() {
		r = &rosa.Runtime{}
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = mock.NewMockClient(mockCtrl)
		r.AWSClient = mockClient

		var err error
		cluster = MockCluster(func(c *cmv1.ClusterBuilder) {
			c.State(cmv1.ClusterStateReady)
			b := cmv1.HypershiftBuilder{}
			b.Enabled(true)
			c.Hypershift(&b)
			c.MultiAZ(true).Nodes(cmv1.NewClusterNodes().AvailabilityZones("us-east-1a", "us-east-1b"))
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(cluster.MultiAZ()).To(Equal(true))
	})

	Context("When testing getMachinePoolAvailabilityZones", func() {
		It("Tests getMachinePoolAvailabilityZones with and without subnet input", func() {
			multiAZMachinePool := false
			availabilityZoneUserInput := "us-east-1a"
			subnetUserInput := ""

			azs, err := getMachinePoolAvailabilityZones(r, cluster,
				multiAZMachinePool, availabilityZoneUserInput, subnetUserInput)
			Expect(err).ToNot(HaveOccurred())
			Expect(azs).To(Equal([]string{"us-east-1a"}))

			multiAZMachinePool = true
			azs, err = getMachinePoolAvailabilityZones(r, cluster,
				multiAZMachinePool, availabilityZoneUserInput, subnetUserInput)
			Expect(err).ToNot(HaveOccurred())
			Expect(azs).To(Equal([]string{"us-east-1a", "us-east-1b"}))

			// Test with subnet input
			subnetUserInput = "subnet-123"
			mockClient.EXPECT().GetSubnetAvailabilityZone(subnetUserInput).
				Return("us-east-1a", nil)

			azs, err = getMachinePoolAvailabilityZones(r, cluster,
				multiAZMachinePool, availabilityZoneUserInput, subnetUserInput)
			Expect(err).ToNot(HaveOccurred())
			Expect(azs).To(Equal([]string{"us-east-1a"}))
		})
	})
})

var _ = Describe("getSubnetFromAvailabilityZone functionality", func() {
	var (
		r              *rosa.Runtime
		cmd            *cobra.Command
		args           *mpOpts.CreateMachinepoolUserOptions
		mockClient     *mock.MockClient
		az             string
		subnetId1      string
		subnetId2      string
		privateSubnets []ec2types.Subnet
		cluster        *cmv1.Cluster
	)

	BeforeEach(func() {
		mockClient = mock.NewMockClient(gomock.NewController(GinkgoT()))
		r = &rosa.Runtime{AWSClient: mockClient}
		cmd = &cobra.Command{}
		az = "us-east-1a"
		subnetId1 = "subnet-123"
		subnetId2 = "subnet-456"
		args = &mpOpts.CreateMachinepoolUserOptions{}
	})

	When("no availability zone is set", func() {
		BeforeEach(func() {
			privateSubnets = []ec2types.Subnet{{AvailabilityZone: &az, SubnetId: &subnetId1}}
			mockClient.EXPECT().GetVPCPrivateSubnets(gomock.Any()).Return(privateSubnets, nil)
			cluster = MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				b := cmv1.HypershiftBuilder{}
				b.Enabled(true)
				c.Hypershift(&b).
					Nodes(cmv1.NewClusterNodes().AvailabilityZones("us-east-1a")).AWS(cmv1.NewAWS().SubnetIDs(subnetId1, subnetId2))
			})
		})

		It("returns the correct subnet when one subnet is expected", func() {
			subnet, err := getSubnetFromAvailabilityZone(cmd, r, false, cluster, args)
			Expect(err).ToNot(HaveOccurred())
			Expect(subnet).To(Equal(subnetId1))
		})
	})

	When("an availability zone is set", func() {
		BeforeEach(func() {
			args.AvailabilityZone = "us-west-1a"
			privateSubnets = []ec2types.Subnet{{AvailabilityZone: &az, SubnetId: &subnetId1}}
			mockClient.EXPECT().GetVPCPrivateSubnets(gomock.Any()).Return(privateSubnets, nil)
			cluster = MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				b := cmv1.HypershiftBuilder{}
				b.Enabled(true)
				c.Hypershift(&b).
					Nodes(cmv1.NewClusterNodes().AvailabilityZones(az)).AWS(cmv1.NewAWS().SubnetIDs(subnetId1))
			})
		})

		It("handles errors correctly when the availability zone does not match", func() {
			subnet, err := getSubnetFromAvailabilityZone(cmd, r, true, cluster, args)
			Expect(err).To(HaveOccurred())
			Expect(subnet).To(Equal(""))
		})
	})
})

var _ = Describe("getSubnetOptions", func() {
	var (
		mockCtrl       *gomock.Controller
		mockAWS        *mock.MockClient
		runtime        *rosa.Runtime
		cluster        *cmv1.Cluster
		subnetIds      []string
		privateSubnets []ec2types.Subnet
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockAWS = mock.NewMockClient(mockCtrl)
		runtime = &rosa.Runtime{AWSClient: mockAWS}
		subnetIds = []string{"subnet-123", "subnet-456"}
		cluster = MockCluster(func(c *cmv1.ClusterBuilder) {
			c.State(cmv1.ClusterStateReady)
			b := cmv1.HypershiftBuilder{}
			b.Enabled(true)
			c.Hypershift(&b)
			c.MultiAZ(true).Nodes(cmv1.NewClusterNodes().AvailabilityZones("us-east-1a", "us-east-1b")).
				AWS(cmv1.NewAWS().SubnetIDs(subnetIds...))
		})
		privateSubnets = []ec2types.Subnet{
			{SubnetId: &subnetIds[0]},
			{SubnetId: &subnetIds[1]},
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	When("GetVPCPrivateSubnets returns subnets successfully", func() {
		BeforeEach(func() {
			mockAWS.EXPECT().GetVPCPrivateSubnets(subnetIds[0]).Return(privateSubnets, nil)
		})

		It("returns a slice of subnet options without error", func() {
			subnetOptions, err := getSubnetOptions(runtime, cluster)
			Expect(err).ToNot(HaveOccurred())
			Expect(subnetOptions).To(HaveLen(2))
		})
	})

	When("GetVPCPrivateSubnets returns an error", func() {
		BeforeEach(func() {
			mockAWS.EXPECT().GetVPCPrivateSubnets(subnetIds[0]).Return(nil, fmt.Errorf("error fetching subnets"))
		})

		It("returns an error and no subnet options", func() {
			subnetOptions, err := getSubnetOptions(runtime, cluster)
			Expect(err).To(HaveOccurred())
			Expect(subnetOptions).To(BeNil())
		})
	})
})

var _ = Describe("getSecurityGroupsOption", func() {
	var (
		mockCtrl  *gomock.Controller
		mockAWS   *mock.MockClient
		runtime   *rosa.Runtime
		cmd       *cobra.Command
		cluster   *cmv1.Cluster
		subnetIds []string
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockAWS = mock.NewMockClient(mockCtrl)
		runtime = &rosa.Runtime{AWSClient: mockAWS}
		cmd = &cobra.Command{}
		subnetIds = []string{"subnet-123", "subnet-456"}
		cluster = MockCluster(func(c *cmv1.ClusterBuilder) {
			c.State(cmv1.ClusterStateReady)
			b := cmv1.HypershiftBuilder{}
			b.Enabled(true)
			c.Hypershift(&b)
			c.MultiAZ(true).Nodes(cmv1.NewClusterNodes().AvailabilityZones("us-east-1a", "us-east-1b")).
				AWS(cmv1.NewAWS().SubnetIDs(subnetIds...))
		})
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	When("Validates cluster's subnet list isn't empty", func() {
		It("should return an error if subnets list is empty", func() {
			clusterWithNoSubnets := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				b := cmv1.HypershiftBuilder{}
				b.Enabled(true)
				c.Hypershift(&b)
				c.MultiAZ(true).Nodes(cmv1.NewClusterNodes().AvailabilityZones("us-east-1a", "us-east-1b")).
					AWS(cmv1.NewAWS())
			})
			_, err := getSecurityGroupsOption(runtime, cmd, clusterWithNoSubnets)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("expected cluster's subnets to contain subnets IDs, but got an empty list"))
		})
	})

	When("Retrieving subnets fails", func() {
		It("should return an error if unable to retrieve subnets", func() {
			mockAWS.EXPECT().GetVPCSubnets(gomock.Any()).Return(nil, fmt.Errorf("failed to retrieve subnets"))
			_, err := getSecurityGroupsOption(runtime, cmd, cluster)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to retrieve available subnets: failed to retrieve subnets"))
		})

		It("should return an error VPC ID is empty", func() {
			mockAWS.EXPECT().GetVPCSubnets(gomock.Any()).Return([]ec2types.
				Subnet{{SubnetId: awssdk.String("subnet-123")}}, nil)
			_, err := getSecurityGroupsOption(runtime, cmd, cluster)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("unexpected situation a VPC ID should have been selected based on chosen subnets"))
		})
	})
})

var _ = Describe("Machine pool min/max replicas validation", func() {
	DescribeTable("Machine pool min replicas validation",
		func(minReplicas int, autoscaling bool, multiAZ bool, hasError bool) {
			replicaSizeValidation := &ReplicaSizeValidation{
				ClusterVersion: "openshift-v4.14.14",
				MultiAz:        multiAZ,
				Autoscaling:    autoscaling,
				IsHostedCp:     false,
			}
			err := replicaSizeValidation.MinReplicaValidator()(minReplicas)
			if hasError {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).ToNot(HaveOccurred())
			}
		},
		Entry("Zero replicas - no autoscaling",
			0,
			false,
			false,
			false,
		),
		Entry("Negative replicas - no autoscaling",
			-1,
			false,
			false,
			true,
		),
		Entry("Zero replicas - autoscaling",
			0,
			true,
			false,
			false,
		),
		Entry("One replicas - autoscaling",
			1,
			true,
			false,
			false,
		),
		Entry("Multi-AZ - 3 replicas",
			3,
			true,
			true,
			false,
		),
		Entry("Multi-AZ - 2 replicas",
			2,
			true,
			true,
			true,
		),
	)
	DescribeTable("Machine pool max replicas validation",
		func(minReplicas int, maxReplicas int, multiAZ bool, hasError bool) {
			replicaSizeValidation := &ReplicaSizeValidation{
				MinReplicas:    minReplicas,
				ClusterVersion: "openshift-v4.14.14",
				MultiAz:        multiAZ,
				IsHostedCp:     false,
			}
			err := replicaSizeValidation.MaxReplicaValidator()(maxReplicas)
			if hasError {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).ToNot(HaveOccurred())
			}
		},
		Entry("Max > Min -> OK",
			1,
			2,
			false,
			false,
		),
		Entry("Min < Max -> OK",
			2,
			1,
			false,
			true,
		),
		Entry("Min = Max -> OK",
			2,
			2,
			false,
			false,
		),
		Entry("Not % 3 -> NOT OK",
			2,
			4,
			true,
			true,
		),
		Entry("Multi-AZ -> OK",
			3,
			6,
			true,
			false,
		),
	)
})

var _ = Describe("CreateAwsNodePoolBuilder", func() {
	It("correctly initializes AWSNodePoolBuilder with given parameters", func() {
		instanceType := "t2.micro"
		securityGroupIds := []string{"sg-12345"}
		awsTags := map[string]string{"env": "test"}
		httpTokens := "required"
		size := 300

		builder := createAwsNodePoolBuilder(instanceType, securityGroupIds, httpTokens, awsTags, &size)
		built, err := builder.Build()

		Expect(err).ToNot(HaveOccurred())
		Expect(built.InstanceType()).To(Equal(instanceType))
		Expect(built.AdditionalSecurityGroupIds()).To(ConsistOf(securityGroupIds))
		Expect(string(built.Ec2MetadataHttpTokens())).To(Equal(httpTokens))
		Expect(built.Tags()).To(Equal(awsTags))
		Expect(built.RootVolume().Size()).To(Equal(300))
	})
})

var _ = Describe("getVpcIdFromSubnet Function", func() {
	It("should return a VPC ID for a valid subnet", func() {
		vpcID := "vpc-12345"
		subnet := ec2types.Subnet{VpcId: &vpcID}

		gotVpcId, err := getVpcIdFromSubnet(subnet)
		Expect(err).ToNot(HaveOccurred())
		Expect(gotVpcId).To(Equal(vpcID))
	})

	It("should return an error if the VPC ID is empty", func() {
		subnet := ec2types.Subnet{VpcId: awssdk.String("")}

		_, err := getVpcIdFromSubnet(subnet)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("unexpected situation a VPC ID should have been selected based on chosen subnets"))
	})
})

var _ = Describe("ValidateClusterVersionWithMaxNodesLimit Function", func() {
	// Classic cluster validations for node count
	It("should return error if user creates mp with more than 180 nodes for classic cluster below v4.14.14", func() {
		clusterVersion := "v4.14.13"
		isHostedCp := false
		replicas := 181

		err := validateClusterVersionWithMaxNodesLimit(clusterVersion, replicas, isHostedCp)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal(fmt.Sprintf("should provide an integer number less than or equal to '%v'", 180)))
	})

	It("should return error if user creates mp with more than 249 nodes for classic cluster at or above v4.14.14", func() {
		clusterVersion := "v4.14.14"
		isHostedCp := false
		replicas := 250

		err := validateClusterVersionWithMaxNodesLimit(clusterVersion, replicas, isHostedCp)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal(fmt.Sprintf("should provide an integer number less than or equal to '%v'", 249)))
	})

	It("should accept if user creates mp with 180 nodes for classic cluster below v4.14.14", func() {
		clusterVersion := "v4.14.13"
		isHostedCp := false
		replicas := 180

		err := validateClusterVersionWithMaxNodesLimit(clusterVersion, replicas, isHostedCp)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should accept if user creates mp with 249 nodes for classic cluster at or above  v4.14.14", func() {
		clusterVersion := "v4.14.14"
		isHostedCp := false
		replicas := 249

		err := validateClusterVersionWithMaxNodesLimit(clusterVersion, replicas, isHostedCp)
		Expect(err).ToNot(HaveOccurred())
	})

	// Hosted CP cluster validations for node count
	It("should return error if user creates mp with more than 500 nodes for hcp cluster at or above v4.14.0", func() {
		clusterVersion := "v4.14.0"
		isHostedCp := true
		replicas := 501

		err := validateClusterVersionWithMaxNodesLimit(clusterVersion, replicas, isHostedCp)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal(fmt.Sprintf("should provide an integer number less than or equal to '%v'", 500)))
	})

	It("should accept if user creates mp with 500 nodes for hcp cluster at or above v4.14.0", func() {
		clusterVersion := "v4.14.0"
		isHostedCp := true
		replicas := 500

		err := validateClusterVersionWithMaxNodesLimit(clusterVersion, replicas, isHostedCp)
		Expect(err).ToNot(HaveOccurred())
	})
})

var _ = Describe("getZoneType Function", func() {
	It("should return 'N/A' when machine pool has no availability zones", func() {
		machinePool := &cmv1.MachinePool{}
		zoneType := getZoneType(machinePool)
		Expect(zoneType).To(Equal(zoneTypeNA))
	})

	It("should return 'Wavelength' when zone contains 'wl'", func() {
		machinePool, err := cmv1.NewMachinePool().
			AvailabilityZones("us-east-1-wl-1").
			Build()
		Expect(err).ToNot(HaveOccurred())

		zoneType := getZoneType(machinePool)
		Expect(zoneType).To(Equal(zoneTypeWavelength))
	})

	It("should return 'Wavelength' when zone contains 'WL' (case insensitive)", func() {
		machinePool, err := cmv1.NewMachinePool().
			AvailabilityZones("us-east-1-WL-1").
			Build()
		Expect(err).ToNot(HaveOccurred())

		zoneType := getZoneType(machinePool)
		Expect(zoneType).To(Equal(zoneTypeWavelength))
	})

	It("should return 'Outpost' when zone contains 'outpost'", func() {
		machinePool, err := cmv1.NewMachinePool().
			AvailabilityZones("us-east-1-outpost-1").
			Build()
		Expect(err).ToNot(HaveOccurred())

		zoneType := getZoneType(machinePool)
		Expect(zoneType).To(Equal(zoneTypeOutpost))
	})

	It("should return 'Outpost' when zone contains 'OUTPOST' (case insensitive)", func() {
		machinePool, err := cmv1.NewMachinePool().
			AvailabilityZones("us-east-1-OUTPOST-1").
			Build()
		Expect(err).ToNot(HaveOccurred())

		zoneType := getZoneType(machinePool)
		Expect(zoneType).To(Equal(zoneTypeOutpost))
	})

	It("should return 'LocalZone' when zone contains '-lz'", func() {
		machinePool, err := cmv1.NewMachinePool().
			AvailabilityZones("us-east-1-lz-1").
			Build()
		Expect(err).ToNot(HaveOccurred())

		zoneType := getZoneType(machinePool)
		Expect(zoneType).To(Equal(zoneTypeLocalZone))
	})

	It("should return 'LocalZone' when zone has more than 3 dashes", func() {
		machinePool, err := cmv1.NewMachinePool().
			AvailabilityZones("us-east-1-zone-extra-dash").
			Build()
		Expect(err).ToNot(HaveOccurred())

		zoneType := getZoneType(machinePool)
		Expect(zoneType).To(Equal(zoneTypeLocalZone))
	})

	It("should return 'Standard' for regular availability zones", func() {
		machinePool, err := cmv1.NewMachinePool().
			AvailabilityZones("us-east-1a").
			Build()
		Expect(err).ToNot(HaveOccurred())

		zoneType := getZoneType(machinePool)
		Expect(zoneType).To(Equal(zoneTypeStandard))
	})

	It("should return 'Standard' for multi-dash zones that don't match special patterns", func() {
		machinePool, err := cmv1.NewMachinePool().
			AvailabilityZones("us-east-1b").
			Build()
		Expect(err).ToNot(HaveOccurred())

		zoneType := getZoneType(machinePool)
		Expect(zoneType).To(Equal(zoneTypeStandard))
	})

	It("should check all zones and return the first special type found", func() {
		machinePool, err := cmv1.NewMachinePool().
			AvailabilityZones("us-east-1a", "us-east-1b", "us-east-1-wl-1").
			Build()
		Expect(err).ToNot(HaveOccurred())

		zoneType := getZoneType(machinePool)
		Expect(zoneType).To(Equal(zoneTypeWavelength))
	})
})

var _ = Describe("isWinLIEnabled Function", func() {
	It("should return 'Yes' when image_type label is 'windows'", func() {
		labels := map[string]string{
			labelImageType: imageTypeWindows,
		}
		result := isWinLIEnabled(labels)
		Expect(result).To(Equal(displayValueYes))
	})

	It("should return 'No' when image_type label is not 'windows'", func() {
		labels := map[string]string{
			labelImageType: "linux",
		}
		result := isWinLIEnabled(labels)
		Expect(result).To(Equal(displayValueNo))
	})

	It("should return 'No' when image_type label is empty", func() {
		labels := map[string]string{
			labelImageType: "",
		}
		result := isWinLIEnabled(labels)
		Expect(result).To(Equal(displayValueNo))
	})

	It("should return 'No' when image_type label is not present", func() {
		labels := map[string]string{
			"other_label": "some_value",
		}
		result := isWinLIEnabled(labels)
		Expect(result).To(Equal(displayValueNo))
	})

	It("should return 'No' when labels map is empty", func() {
		labels := map[string]string{}
		result := isWinLIEnabled(labels)
		Expect(result).To(Equal(displayValueNo))
	})

	It("should return 'No' when labels map is nil", func() {
		var labels map[string]string
		result := isWinLIEnabled(labels)
		Expect(result).To(Equal(displayValueNo))
	})

	It("should return 'No' when image_type contains 'windows' but is not exactly 'windows'", func() {
		labels := map[string]string{
			labelImageType: "windows-server",
		}
		result := isWinLIEnabled(labels)
		Expect(result).To(Equal(displayValueNo))
	})

	It("should work with other labels present", func() {
		labels := map[string]string{
			labelImageType: imageTypeWindows,
			"environment":  "production",
			"team":         "platform",
		}
		result := isWinLIEnabled(labels)
		Expect(result).To(Equal(displayValueYes))
	})
})

var _ = Describe("isDedicatedHost Function", func() {
	var (
		mockCtrl      *gomock.Controller
		mockAWSClient *mock.MockClient
		mockReporter  *reporter.Object
		runtime       *rosa.Runtime
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockAWSClient = mock.NewMockClient(mockCtrl)

		mockReporter = reporter.CreateReporter()

		runtime = &rosa.Runtime{
			AWSClient: mockAWSClient,
			Reporter:  mockReporter,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	It("should return 'No' when machine pool is nil", func() {
		result := isDedicatedHost(nil, runtime)
		Expect(result).To(Equal(displayValueNo))
	})

	It("should return 'No' when machine pool has no AWS configuration", func() {
		machinePool, err := cmv1.NewMachinePool().Build()
		Expect(err).ToNot(HaveOccurred())

		result := isDedicatedHost(machinePool, runtime)
		Expect(result).To(Equal(displayValueNo))
	})

	It("should return 'No' when machine pool AWS ID is empty", func() {
		machinePool, err := cmv1.NewMachinePool().
			AWS(cmv1.NewAWSMachinePool().ID("")).
			Build()
		Expect(err).ToNot(HaveOccurred())

		result := isDedicatedHost(machinePool, runtime)
		Expect(result).To(Equal(displayValueNo))
	})

	It("should return 'No' when runtime is nil", func() {
		machinePool, err := cmv1.NewMachinePool().
			AWS(cmv1.NewAWSMachinePool().ID("mp-12345")).
			Build()
		Expect(err).ToNot(HaveOccurred())

		result := isDedicatedHost(machinePool, nil)
		Expect(result).To(Equal(displayValueNo))
	})

	It("should return 'No' when runtime AWSClient is nil", func() {
		machinePool, err := cmv1.NewMachinePool().
			AWS(cmv1.NewAWSMachinePool().ID("mp-12345")).
			Build()
		Expect(err).ToNot(HaveOccurred())

		runtimeWithoutClient := &rosa.Runtime{AWSClient: nil, Reporter: mockReporter}
		result := isDedicatedHost(machinePool, runtimeWithoutClient)
		Expect(result).To(Equal(displayValueNo))
	})

	It("should return 'Yes' when machine pool has dedicated host", func() {
		const testMachinePoolID = "mp-12345"
		machinePool, err := cmv1.NewMachinePool().
			AWS(cmv1.NewAWSMachinePool().ID(testMachinePoolID)).
			Build()
		Expect(err).ToNot(HaveOccurred())

		mockAWSClient.EXPECT().
			CheckIfMachinePoolHasDedicatedHost([]string{testMachinePoolID}).
			Return(true, nil)

		result := isDedicatedHost(machinePool, runtime)
		Expect(result).To(Equal(displayValueYes))
	})

	It("should return 'No' when machine pool does not have dedicated host", func() {
		const testMachinePoolID = "mp-12345"
		machinePool, err := cmv1.NewMachinePool().
			AWS(cmv1.NewAWSMachinePool().ID(testMachinePoolID)).
			Build()
		Expect(err).ToNot(HaveOccurred())

		mockAWSClient.EXPECT().
			CheckIfMachinePoolHasDedicatedHost([]string{testMachinePoolID}).
			Return(false, nil)

		result := isDedicatedHost(machinePool, runtime)
		Expect(result).To(Equal(displayValueNo))
	})

	It("should return 'Unknown' when AWS client returns an error", func() {
		const testMachinePoolID = "mp-12345"
		machinePool, err := cmv1.NewMachinePool().
			AWS(cmv1.NewAWSMachinePool().ID(testMachinePoolID)).
			Build()
		Expect(err).ToNot(HaveOccurred())

		expectedError := fmt.Errorf("AWS API error")
		mockAWSClient.EXPECT().
			CheckIfMachinePoolHasDedicatedHost([]string{testMachinePoolID}).
			Return(false, expectedError)

		result := isDedicatedHost(machinePool, runtime)
		Expect(result).To(Equal(displayValueUnknown))
	})

	// Optional: Test specifically for the nil reporter case
	It("should return 'Unknown' when AWS client returns an error and reporter is nil", func() {
		const testMachinePoolID = "mp-12345"
		machinePool, err := cmv1.NewMachinePool().
			AWS(cmv1.NewAWSMachinePool().ID(testMachinePoolID)).
			Build()
		Expect(err).ToNot(HaveOccurred())

		expectedError := fmt.Errorf("AWS API error")
		mockAWSClient.EXPECT().
			CheckIfMachinePoolHasDedicatedHost([]string{testMachinePoolID}).
			Return(false, expectedError)

		result := isDedicatedHost(machinePool, runtime)
		Expect(result).To(Equal(displayValueUnknown))
	})
})
