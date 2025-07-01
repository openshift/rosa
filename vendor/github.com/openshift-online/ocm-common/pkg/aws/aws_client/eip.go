package aws_client

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/openshift-online/ocm-common/pkg/log"
)

func (client *AWSClient) AllocateEIPAddress() (*ec2.AllocateAddressOutput, error) {
	inputs := &ec2.AllocateAddressInput{
		Address:               nil,
		CustomerOwnedIpv4Pool: nil,
		Domain:                "",
		DryRun:                nil,
		NetworkBorderGroup:    nil,
		PublicIpv4Pool:        nil,
		TagSpecifications:     nil,
	}

	respEIP, err := client.Ec2Client.AllocateAddress(context.TODO(), inputs)
	if err != nil {
		log.LogError("Create EIP failed %s", err.Error())
		return nil, err
	}
	log.LogInfo("Allocated EIP %s with ip %s", *respEIP.AllocationId, *respEIP.PublicIp)
	return respEIP, err
}

func (client *AWSClient) DisassociateAddress(associateID string) (*ec2.DisassociateAddressOutput, error) {
	inputDisassociate := &ec2.DisassociateAddressInput{
		AssociationId: aws.String(associateID),
	}

	respDisassociate, err := client.EC2().DisassociateAddress(context.TODO(), inputDisassociate)
	if err != nil {
		log.LogError("Disassociate EIP failed %s", err.Error())
		return nil, err
	}
	log.LogInfo("Disassociate EIP successfully")
	return respDisassociate, err
}

func (client *AWSClient) AllocateEIPAndAssociateInstance(instanceID string) (string, error) {
	allocRes, err := client.AllocateEIPAddress()
	if err != nil {
		log.LogError("Failed allocated EIP: %s", err)
		return "", err
	} else {
		log.LogInfo("Successfully allocated EIP: %s", *allocRes.PublicIp)
	}
	assocRes, err := client.EC2().AssociateAddress(context.TODO(),
		&ec2.AssociateAddressInput{
			AllocationId: allocRes.AllocationId,
			InstanceId:   aws.String(instanceID),
		})
	if err != nil {
		defer func() {
			err := client.ReleaseAddressWithAllocationID(*allocRes.AllocationId)
			log.LogError("Associate EIP allocation %s failed to instance ID %s", *allocRes.AllocationId, instanceID)
			if err != nil {
				log.LogError("Failed allocated EIP: %s", err)
			}
		}()
		return "", err

	}
	log.LogInfo("Successfully allocated %s with instance %s.\n\tallocation id: %s, association id: %s\n",
		*allocRes.PublicIp, instanceID, *allocRes.AllocationId, *assocRes.AssociationId)
	return *allocRes.PublicIp, nil
}

func (client *AWSClient) ReleaseAddressWithAllocationID(allocationID string) error {
	inputRelease := &ec2.ReleaseAddressInput{
		AllocationId:       aws.String(allocationID),
		DryRun:             nil,
		NetworkBorderGroup: nil,
		PublicIp:           nil,
	}
	_, err := client.Ec2Client.ReleaseAddress(context.TODO(), inputRelease)
	if err != nil {
		log.LogError("Release EIP %s failed: %s", allocationID, err.Error())
		return err
	}
	log.LogInfo("Release EIP %s successsully ", allocationID)
	return nil
}

func (client *AWSClient) DescribeAddresses(filters ...map[string][]string) (*ec2.DescribeAddressesOutput, error) {
	filterInput := []types.Filter{}
	for _, filter := range filters {
		for k, v := range filter {
			copyKey := k
			awsFilter := types.Filter{
				Name:   &copyKey,
				Values: v,
			}
			filterInput = append(filterInput, awsFilter)
		}
	}
	inputAdd := &ec2.DescribeAddressesInput{
		Filters: filterInput,
	}
	output, err := client.EC2().DescribeAddresses(context.TODO(), inputAdd)
	if err != nil {
		log.LogError("Describe EIP met error: %s", err.Error())
		return nil, err
	}
	return output, nil
}

func (client *AWSClient) ReleaseAddressWithFilter(filters ...map[string][]string) error {
	addOutput, err := client.DescribeAddresses(filters...)
	if err != nil {
		return err
	}
	if addOutput != nil {
		for _, add := range addOutput.Addresses {
			if add.AllocationId != nil {
				_, err = client.DisassociateAddress(*add.AssociationId)
				if err != nil {
					return err
				}
				err = client.ReleaseAddressWithAllocationID(*add.AllocationId)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
