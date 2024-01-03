package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

// Ec2ApiClient is an interface that defines the methods that we want to use
// from the Client type in the AWS SDK (github.com/aws/aws-sdk-go-v2/service/ec2)
// The AIM is to only contain methods that are defined in the AWS SDK's EC2
// Client.
// For the cases where logic is desired to be implemened combining EC2 calls and
// other logic use the pkg/aws.Client type.
// If you need to use a method provided by the AWS SDK's EC2 Client but it
// is not defined in this interface then it has to be added and all
// the types implementing this interface have to implement the new method.
// The reason this interface has been defined is so we can perform unit testing
// on methods that make use of the AWS EC2 service.
//

type Ec2ApiClient interface {
	AcceptAddressTransfer(ctx context.Context, params *ec2.AcceptAddressTransferInput, optFns ...func(*ec2.Options),
	) (*ec2.AcceptAddressTransferOutput, error)

	DescribeSecurityGroups(ctx context.Context, params *ec2.DescribeSecurityGroupsInput, optFns ...func(*ec2.Options),
	) (*ec2.DescribeSecurityGroupsOutput, error)
	DescribeNatGateways(ctx context.Context, params *ec2.DescribeNatGatewaysInput, optFns ...func(*ec2.Options),
	) (*ec2.DescribeNatGatewaysOutput, error)
	DescribeAddresses(ctx context.Context, params *ec2.DescribeAddressesInput, optFns ...func(*ec2.Options),
	) (*ec2.DescribeAddressesOutput, error)
	DescribeSecurityGroupRules(ctx context.Context,
		params *ec2.DescribeSecurityGroupRulesInput, optFns ...func(*ec2.Options),
	) (*ec2.DescribeSecurityGroupRulesOutput, error)
	DescribeVpcAttribute(ctx context.Context, params *ec2.DescribeVpcAttributeInput, optFns ...func(*ec2.Options),
	) (*ec2.DescribeVpcAttributeOutput, error)
	DescribeAvailabilityZones(ctx context.Context,
		params *ec2.DescribeAvailabilityZonesInput, optFns ...func(*ec2.Options),
	) (*ec2.DescribeAvailabilityZonesOutput, error)
	DescribeRegions(ctx context.Context, params *ec2.DescribeRegionsInput, optFns ...func(*ec2.Options),
	) (*ec2.DescribeRegionsOutput, error)
	DescribeReservedInstancesOfferings(ctx context.Context,
		params *ec2.DescribeReservedInstancesOfferingsInput, optFns ...func(*ec2.Options),
	) (*ec2.DescribeReservedInstancesOfferingsOutput, error)
	DescribeRouteTables(ctx context.Context, params *ec2.DescribeRouteTablesInput, optFns ...func(*ec2.Options),
	) (*ec2.DescribeRouteTablesOutput, error)
	DescribeSubnets(ctx context.Context, params *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options),
	) (*ec2.DescribeSubnetsOutput, error)
	DescribeVpcs(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options),
	) (*ec2.DescribeVpcsOutput, error)
	DescribeNetworkInterfaces(ctx context.Context,
		params *ec2.DescribeNetworkInterfacesInput, optFns ...func(*ec2.Options),
	) (*ec2.DescribeNetworkInterfacesOutput, error)
	DescribeInternetGateways(ctx context.Context, params *ec2.DescribeInternetGatewaysInput, optFns ...func(*ec2.Options),
	) (*ec2.DescribeInternetGatewaysOutput, error)
	DescribeInstanceTypeOfferings(ctx context.Context,
		params *ec2.DescribeInstanceTypeOfferingsInput, optFns ...func(*ec2.Options),
	) (*ec2.DescribeInstanceTypeOfferingsOutput, error)

	CreateSecurityGroup(ctx context.Context, params *ec2.CreateSecurityGroupInput, optFns ...func(*ec2.Options),
	) (*ec2.CreateSecurityGroupOutput, error)
	CreateSubnet(ctx context.Context, params *ec2.CreateSubnetInput, optFns ...func(*ec2.Options),
	) (*ec2.CreateSubnetOutput, error)
	CreateTags(ctx context.Context, params *ec2.CreateTagsInput, optFns ...func(*ec2.Options),
	) (*ec2.CreateTagsOutput, error)

	CreateVolume(ctx context.Context, params *ec2.CreateVolumeInput, optFns ...func(*ec2.Options),
	) (*ec2.CreateVolumeOutput, error)
	CreateVpc(ctx context.Context, params *ec2.CreateVpcInput, optFns ...func(*ec2.Options),
	) (*ec2.CreateVpcOutput, error)

	DeleteSecurityGroup(ctx context.Context, params *ec2.DeleteSecurityGroupInput, optFns ...func(*ec2.Options),
	) (*ec2.DeleteSecurityGroupOutput, error)
	DeleteSubnet(ctx context.Context, params *ec2.DeleteSubnetInput, optFns ...func(*ec2.Options),
	) (*ec2.DeleteSubnetOutput, error)
	DeleteTags(ctx context.Context, params *ec2.DeleteTagsInput, optFns ...func(*ec2.Options),
	) (*ec2.DeleteTagsOutput, error)

	DeleteVolume(ctx context.Context, params *ec2.DeleteVolumeInput, optFns ...func(*ec2.Options),
	) (*ec2.DeleteVolumeOutput, error)
	DeleteVpc(ctx context.Context, params *ec2.DeleteVpcInput, optFns ...func(*ec2.Options),
	) (*ec2.DeleteVpcOutput, error)

	RunInstances(ctx context.Context, params *ec2.RunInstancesInput, optFns ...func(*ec2.Options),
	) (*ec2.RunInstancesOutput, error)
}

// interface guard to ensure that all methods defined in the Ec2ApiClient
// interface are implemented by the real AWS EC2 client. This interface
// guard should always compile
var _ Ec2ApiClient = (*ec2.Client)(nil)
