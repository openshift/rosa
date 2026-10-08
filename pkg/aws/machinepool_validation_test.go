package aws

import (
	"context"
	"fmt"

	"go.uber.org/mock/gomock"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/aws/mocks"
)

var _ = Describe("Machine pool AWS validation", func() {
	var client *mocks.MockEc2ApiClient
	var request ValidateMachinePoolAWSRequest
	BeforeEach(func() {
		client = mocks.NewMockEc2ApiClient(gomock.NewController(GinkgoT()))
		request = ValidateMachinePoolAWSRequest{SubnetID: "subnet-worker", VPCID: "vpc-cluster", InstanceType: "m5.xlarge"}
	})
	validSubnet := func() *ec2.DescribeSubnetsOutput {
		return &ec2.DescribeSubnetsOutput{Subnets: []ec2types.Subnet{{
			VpcId: awssdk.String("vpc-cluster"), AvailabilityZone: awssdk.String("us-east-1a"),
		}}}
	}
	It("checks offerings in the selected subnet AZ and follows pagination", func() {
		client.EXPECT().DescribeSubnets(gomock.Any(), &ec2.DescribeSubnetsInput{SubnetIds: []string{request.SubnetID}}).
			Return(validSubnet(), nil)
		calls := 0
		client.EXPECT().DescribeInstanceTypeOfferings(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, input *ec2.DescribeInstanceTypeOfferingsInput, _ ...func(*ec2.Options)) (*ec2.DescribeInstanceTypeOfferingsOutput, error) {
				Expect(input.LocationType).To(Equal(ec2types.LocationTypeAvailabilityZone))
				Expect(input.Filters).To(ConsistOf(
					ec2types.Filter{Name: awssdk.String("location"), Values: []string{"us-east-1a"}},
					ec2types.Filter{Name: awssdk.String("instance-type"), Values: []string{"m5.xlarge"}},
				))
				calls++
				if calls == 1 {
					Expect(input.NextToken).To(BeNil())
					return &ec2.DescribeInstanceTypeOfferingsOutput{NextToken: awssdk.String("next")}, nil
				}
				Expect(awssdk.ToString(input.NextToken)).To(Equal("next"))
				return &ec2.DescribeInstanceTypeOfferingsOutput{InstanceTypeOfferings: []ec2types.InstanceTypeOffering{{
					InstanceType: ec2types.InstanceTypeM5Xlarge, Location: awssdk.String("us-east-1a"),
				}}}, nil
			}).Times(2)
		Expect(ValidateMachinePoolAWS(context.Background(), client, request)).To(Succeed())
	})
	It("rejects an instance type without an offering", func() {
		request.InstanceType = "wrong"
		client.EXPECT().DescribeSubnets(gomock.Any(), gomock.Any()).Return(validSubnet(), nil)
		client.EXPECT().DescribeInstanceTypeOfferings(gomock.Any(), gomock.Any(), gomock.Any()).Return(&ec2.DescribeInstanceTypeOfferingsOutput{}, nil)
		Expect(ValidateMachinePoolAWS(context.Background(), client, request)).To(MatchError(
			"instance type 'wrong' is not supported in availability zone 'us-east-1a'"))
	})
	DescribeTable("rejects missing or malformed subnets before looking up offerings", func(code string) {
		request.SubnetID = "subnet-xxx"
		client.EXPECT().DescribeSubnets(gomock.Any(), gomock.Any()).Return(nil, &smithy.GenericAPIError{Code: code})
		Expect(ValidateMachinePoolAWS(context.Background(), client, request)).To(MatchError(ContainSubstring("The subnet ID 'subnet-xxx' does not exist")))
	}, Entry("missing", "InvalidSubnetID.NotFound"), Entry("malformed", "InvalidSubnetID.Malformed"))
	It("rejects an empty subnet response", func() {
		client.EXPECT().DescribeSubnets(gomock.Any(), gomock.Any()).Return(&ec2.DescribeSubnetsOutput{}, nil)
		Expect(ValidateMachinePoolAWS(context.Background(), client, request)).To(MatchError("The subnet ID 'subnet-worker' does not exist"))
	})
	It("rejects a subnet in a different VPC before looking up offerings", func() {
		subnets := validSubnet()
		subnets.Subnets[0].VpcId = awssdk.String("vpc-other")
		client.EXPECT().DescribeSubnets(gomock.Any(), gomock.Any()).Return(subnets, nil)
		Expect(ValidateMachinePoolAWS(context.Background(), client, request)).To(MatchError(
			"subnet 'subnet-worker' found but expected on VPC 'vpc-cluster'"))
	})
	It("preserves AWS permission errors", func() {
		permissionError := &smithy.GenericAPIError{Code: "UnauthorizedOperation"}
		client.EXPECT().DescribeSubnets(gomock.Any(), gomock.Any()).Return(nil, permissionError)
		Expect(ValidateMachinePoolAWS(context.Background(), client, request)).To(MatchError(ContainSubstring("UnauthorizedOperation")))
	})
	It("preserves instance offerings lookup errors", func() {
		client.EXPECT().DescribeSubnets(gomock.Any(), gomock.Any()).Return(validSubnet(), nil)
		client.EXPECT().DescribeInstanceTypeOfferings(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("offerings unavailable"))
		Expect(ValidateMachinePoolAWS(context.Background(), client, request)).To(MatchError(ContainSubstring("offerings unavailable")))
	})
	It("rejects missing request values without AWS calls", func() {
		request.VPCID = ""
		Expect(ValidateMachinePoolAWS(context.Background(), client, request)).To(MatchError(ContainSubstring("cluster VPC")))
	})
	It("rejects subnet records without an availability zone", func() {
		subnets := validSubnet()
		subnets.Subnets[0].AvailabilityZone = nil
		client.EXPECT().DescribeSubnets(gomock.Any(), gomock.Any()).Return(subnets, nil)
		Expect(ValidateMachinePoolAWS(context.Background(), client, request)).To(MatchError("subnet 'subnet-worker' has no availability zone"))
	})
})
