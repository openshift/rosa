package machinepool

import (
	"fmt"

	gomock "go.uber.org/mock/gomock"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	mock "github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/helper/features"
	mpOpts "github.com/openshift/rosa/pkg/options/machinepool"
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
				"Expected cluster's subnets to contain subnets IDs, but got an empty list"))
		})

		It("Should return an error is subnet is missing the VPC ID", func() {
			subnet := ec2types.Subnet{}

			_, err := getVpcIdFromSubnet(subnet)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(
				"Unexpected situation a VPC ID should have been selected based on chosen subnets"))
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
				300,
			)
			awsNodePool, err := awsNpBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			Expect(awsNodePool.AdditionalSecurityGroupIds()).To(Equal(securityGroupIds))
			Expect(awsNodePool.InstanceType()).To(Equal(instanceType))
			Expect(awsNodePool.Tags()).To(Equal(awsTags))
			Expect(awsNodePool.RootVolume().Size()).To(Equal(300))
		})
		It("Create AWS node pool with security group IDs when provided", func() {
			instanceType := "123"
			securityGroupIds := []string{"123"}

			awsNpBuilder := createAwsNodePoolBuilder(
				instanceType,
				securityGroupIds,
				"optional",
				map[string]string{},
				300,
			)
			awsNodePool, err := awsNpBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			Expect(awsNodePool.AdditionalSecurityGroupIds()).To(Equal(securityGroupIds))
			Expect(awsNodePool.InstanceType()).To(Equal(instanceType))
			Expect(awsNodePool.Tags()).To(HaveLen(0))
			Expect(awsNodePool.RootVolume().Size()).To(Equal(300))
		})
		It("Create AWS node pool without security group IDs if not provided", func() {
			instanceType := "123"

			awsNpBuilder := createAwsNodePoolBuilder(
				instanceType,
				[]string{},
				"optional",
				map[string]string{},
				300,
			)
			awsNodePool, err := awsNpBuilder.Build()
			Expect(err).ToNot(HaveOccurred())
			Expect(awsNodePool.AdditionalSecurityGroupIds()).To(HaveLen(0))
			Expect(awsNodePool.InstanceType()).To(Equal(instanceType))
			Expect(awsNodePool.Tags()).To(HaveLen(0))
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
			Expect(err.Error()).To(Equal("Expected cluster's subnets to contain subnets IDs, but got an empty list"))
		})
	})

	When("Retrieving subnets fails", func() {
		It("should return an error if unable to retrieve subnets", func() {
			mockAWS.EXPECT().GetVPCSubnets(gomock.Any()).Return(nil, fmt.Errorf("failed to retrieve subnets"))
			_, err := getSecurityGroupsOption(runtime, cmd, cluster)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Failed to retrieve available subnets: failed to retrieve subnets"))
		})

		It("should return an error VPC ID is empty", func() {
			mockAWS.EXPECT().GetVPCSubnets(gomock.Any()).Return([]ec2types.
				Subnet{{SubnetId: awssdk.String("subnet-123")}}, nil)
			_, err := getSecurityGroupsOption(runtime, cmd, cluster)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("Unexpected situation a VPC ID should have been selected based on chosen subnets"))
		})
	})
})

var _ = Describe("CreateAwsNodePoolBuilder", func() {
	It("correctly initializes AWSNodePoolBuilder with given parameters", func() {
		instanceType := "t2.micro"
		securityGroupIds := []string{"sg-12345"}
		awsTags := map[string]string{"env": "test"}
		httpTokens := "required"

		builder := createAwsNodePoolBuilder(instanceType, securityGroupIds, httpTokens, awsTags, 300)
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
		Expect(err.Error()).To(Equal("Unexpected situation a VPC ID should have been selected based on chosen subnets"))
	})
})
