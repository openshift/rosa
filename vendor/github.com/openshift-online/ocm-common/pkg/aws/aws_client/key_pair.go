package aws_client

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/openshift-online/ocm-common/pkg/log"
)

func (client *AWSClient) CreateKeyPair(keyName string) (*ec2.CreateKeyPairOutput, error) {

	input := &ec2.CreateKeyPairInput{
		KeyName: &keyName,
	}

	output, err := client.Ec2Client.CreateKeyPair(context.TODO(), input)
	if err != nil {
		log.LogError("Create key pair error " + err.Error())
		return nil, err
	}
	log.LogInfo("Create key pair success: " + *output.KeyPairId)

	return output, err
}

func (client *AWSClient) DeleteKeyPair(keyName string) (*ec2.DeleteKeyPairOutput, error) {
	input := &ec2.DeleteKeyPairInput{
		KeyName: &keyName,
	}

	output, err := client.Ec2Client.DeleteKeyPair(context.TODO(), input)
	if err != nil {
		log.LogError("Delete key pair error " + err.Error())
		return nil, err
	}
	log.LogInfo("Delete key pair success")
	return output, err

}
