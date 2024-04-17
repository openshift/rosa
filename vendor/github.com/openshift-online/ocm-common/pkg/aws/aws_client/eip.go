package aws_client

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
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
		log.LogError("Create eip failed " + err.Error())
		return nil, err
	}
	log.LogInfo("Allocated EIP %s with ip %s", *respEIP.AllocationId, *respEIP.PublicIp)
	return respEIP, err
}

func (client *AWSClient) DisassociateAddress(associateID string) (*ec2.DisassociateAddressOutput, error) {
	inputDisassociate := &ec2.DisassociateAddressInput{
		AssociationId: aws.String(associateID),
		DryRun:        nil,
		PublicIp:      nil,
	}

	respDisassociate, err := client.Ec2Client.DisassociateAddress(context.TODO(), inputDisassociate)
	if err != nil {
		log.LogError("Disassociate eip failed " + err.Error())
		return nil, err
	}
	log.LogInfo("Disassociate eip success")
	return respDisassociate, err
}

func (client *AWSClient) AllocateEIPAndAssociateInstance(instanceID string) (string, error) {
	allocRes, err := client.AllocateEIPAddress()
	if err != nil {
		log.LogError("Failed allocated EIP: %s", err)
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
			_, err := client.ReleaseAddress(*allocRes.AllocationId)
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

func (client *AWSClient) ReleaseAddress(allocationID string) (*ec2.ReleaseAddressOutput, error) {
	inputRelease := &ec2.ReleaseAddressInput{
		AllocationId:       aws.String(allocationID),
		DryRun:             nil,
		NetworkBorderGroup: nil,
		PublicIp:           nil,
	}
	respRelease, err := client.Ec2Client.ReleaseAddress(context.TODO(), inputRelease)
	if err != nil {
		log.LogError("Release eip failed " + err.Error())
		return nil, err
	}
	log.LogInfo("Release eip success: " + allocationID)
	return respRelease, err
}
