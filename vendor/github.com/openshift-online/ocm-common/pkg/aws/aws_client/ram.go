package aws_client

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/ram"
	"github.com/aws/aws-sdk-go-v2/service/ram/types"
	"github.com/openshift-online/ocm-common/pkg/log"
	"time"
)

func (awsClient AWSClient) CreateResourceShare(resourceShareName string, resourceArns []string, principles []string) (*ram.CreateResourceShareOutput, error) {
	input := &ram.CreateResourceShareInput{
		Name:         &resourceShareName,
		ResourceArns: resourceArns,
		Principals:   principles,
	}

	resp, err := awsClient.RamClient.CreateResourceShare(context.TODO(), input)
	if err != nil {
		log.LogError("Create resource share failed with name %s: %s", resourceShareName, err.Error())
	} else {
		log.LogInfo("Create resource share succeed with name %s", resourceShareName)
	}
	return resp, err
}

func (awsClient AWSClient) GetResourceShareAssociations(resourceShareArn string,
	associationType types.ResourceShareAssociationType) (*ram.GetResourceShareAssociationsOutput, error) {
	input := &ram.GetResourceShareAssociationsInput{
		ResourceShareArns: []string{resourceShareArn},
		AssociationType:   associationType,
	}

	resp, err := awsClient.RamClient.GetResourceShareAssociations(context.TODO(), input)
	if err != nil {
		log.LogError("Get resource share association failed with name %s: %s", resourceShareArn, err.Error())
	} else {
		log.LogInfo("Get resource share association succeed with name %s", resourceShareArn)
	}
	return resp, err
}

func (awsClient AWSClient) DeleteResourceShare(resourceShareArn string) error {
	input := &ram.DeleteResourceShareInput{
		ResourceShareArn: &resourceShareArn,
	}

	_, err := awsClient.RamClient.DeleteResourceShare(context.TODO(), input)
	return err
}

func (awsClient AWSClient) PrepareResourceShare(resourceShareName string, resourceArns []string, accountID string) (string, error) {
	var principles []string
	principles = append(principles, accountID)

	sharedResourceOutput, err := awsClient.CreateResourceShare(resourceShareName, resourceArns, principles)
	if err != nil {
		return "", err
	}
	resourceShareArn := *sharedResourceOutput.ResourceShare.ResourceShareArn

	return resourceShareArn, err
}

func (awsClient AWSClient) CheckSubnetResourceShareAssociationsStatus(resourceShareArn string,
	subnetArns []string, timeout time.Duration) error {
	endTime := time.Now().Add(timeout)

	for time.Now().Before(endTime) {
		result, err := awsClient.GetResourceShareAssociations(resourceShareArn, types.ResourceShareAssociationTypeResource)
		if err != nil {
			return err
		}

		activeCount := 0
		for _, association := range result.ResourceShareAssociations {
			for _, subnetArn := range subnetArns {
				if *association.AssociatedEntity == subnetArn && association.Status == "ASSOCIATED" {
					activeCount++
					break
				}
			}
		}

		if activeCount == len(subnetArns) {
			log.LogInfo("All subnets are associated.")
			return nil
		}

		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf("Subnets resource shares did not become associated within %v", timeout)
}
