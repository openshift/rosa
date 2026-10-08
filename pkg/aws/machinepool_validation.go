package aws

import (
	"context"
	"errors"
	"fmt"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"

	awsapi "github.com/openshift/rosa/pkg/aws/api_interface"
)

type ValidateMachinePoolAWSRequest struct {
	SubnetID     string
	VPCID        string
	InstanceType string
}

// ValidateMachinePoolAWS checks the worker subnet and instance availability before creating a pool.
func ValidateMachinePoolAWS(
	ctx context.Context, client awsapi.Ec2ApiClient, request ValidateMachinePoolAWSRequest,
) error {
	if request.SubnetID == "" || request.VPCID == "" || request.InstanceType == "" {
		return fmt.Errorf("subnet, cluster VPC and instance type are required for machine pool validation")
	}
	subnets, err := client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{SubnetIds: []string{request.SubnetID}})
	if err != nil {
		var apiError smithy.APIError
		if errors.As(err, &apiError) && (apiError.ErrorCode() == "InvalidSubnetID.NotFound" ||
			apiError.ErrorCode() == "InvalidSubnetID.Malformed") {
			return fmt.Errorf( //nolint:staticcheck // Preserve the existing OCM validation wording.
				"The subnet ID '%s' does not exist: %w", request.SubnetID, err)
		}
		return fmt.Errorf("failed to describe subnet '%s': %w", request.SubnetID, err)
	}
	if len(subnets.Subnets) != 1 {
		return fmt.Errorf( //nolint:staticcheck // Preserve the existing OCM validation wording.
			"The subnet ID '%s' does not exist", request.SubnetID)
	}
	subnet := subnets.Subnets[0]
	if awssdk.ToString(subnet.VpcId) != request.VPCID {
		return fmt.Errorf("subnet '%s' found but expected on VPC '%s'", request.SubnetID, request.VPCID)
	}
	availabilityZone := awssdk.ToString(subnet.AvailabilityZone)
	if availabilityZone == "" {
		return fmt.Errorf("subnet '%s' has no availability zone", request.SubnetID)
	}
	paginator := ec2.NewDescribeInstanceTypeOfferingsPaginator(client, &ec2.DescribeInstanceTypeOfferingsInput{
		LocationType: ec2types.LocationTypeAvailabilityZone,
		Filters: []ec2types.Filter{
			{Name: awssdk.String("location"), Values: []string{availabilityZone}},
			{Name: awssdk.String("instance-type"), Values: []string{request.InstanceType}},
		},
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to check instance type '%s' in availability zone '%s': %w",
				request.InstanceType, availabilityZone, err)
		}
		for _, offering := range page.InstanceTypeOfferings {
			if string(offering.InstanceType) == request.InstanceType && awssdk.ToString(offering.Location) == availabilityZone {
				return nil
			}
		}
	}
	return fmt.Errorf("instance type '%s' is not supported in availability zone '%s'",
		request.InstanceType, availabilityZone)
}
