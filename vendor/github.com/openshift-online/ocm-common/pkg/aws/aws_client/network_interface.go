package aws_client

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/openshift-online/ocm-common/pkg/log"
)

func (client *AWSClient) DescribeNetWorkInterface(vpcID string) ([]types.NetworkInterface, error) {
	vpcFilter := "vpc-id"
	filter := []types.Filter{
		types.Filter{
			Name: &vpcFilter,
			Values: []string{
				vpcID,
			},
		},
	}
	input := &ec2.DescribeNetworkInterfacesInput{
		Filters: filter,
	}
	resp, err := client.Ec2Client.DescribeNetworkInterfaces(context.TODO(), input)
	if err != nil {
		return nil, err
	}
	return resp.NetworkInterfaces, err
}

func (client *AWSClient) GetNetworkInterfacesByInstanceID(instanceID string) ([]types.NetworkInterface, error) {
	input := &ec2.DescribeNetworkInterfacesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("attachment.instance-id"),
				Values: []string{instanceID},
			},
		},
	}

	output, err := client.Ec2Client.DescribeNetworkInterfaces(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe network interfaces: %w", err)
	}

	return output.NetworkInterfaces, nil
}

func (client *AWSClient) DeleteNetworkInterface(networkinterface types.NetworkInterface) error {
	association := networkinterface.Association
	if association != nil {
		if association.AllocationId != nil {
			err := client.ReleaseAddressWithAllocationID(*association.AllocationId)
			if err != nil {
				log.LogError("Release address failed for %s: %s", *networkinterface.NetworkInterfaceId, err)
				return err
			}

		}

	}
	deleteNIInput := &ec2.DeleteNetworkInterfaceInput{
		NetworkInterfaceId: networkinterface.NetworkInterfaceId,
	}
	_, err := client.Ec2Client.DeleteNetworkInterface(context.TODO(), deleteNIInput)
	if err != nil {
		log.LogError("Delete network interface %s failed： %s", *networkinterface.NetworkInterfaceId, err)
	} else {
		log.LogInfo("Deleted network interface %s", *networkinterface.NetworkInterfaceId)
	}
	return err
}
