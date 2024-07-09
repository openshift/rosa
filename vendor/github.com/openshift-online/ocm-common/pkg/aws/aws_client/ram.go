package aws_client

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/ram"
	"github.com/openshift-online/ocm-common/pkg/log"
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

func (awsClient AWSClient) DeleteResourceShare(resourceShareArn string) error {
	input := &ram.DeleteResourceShareInput{
		ResourceShareArn: &resourceShareArn,
	}

	_, err := awsClient.RamClient.DeleteResourceShare(context.TODO(), input)
	return err
}
