package aws_client

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

func (client *AWSClient) CreateKeyPair(keyName string) (*ec2.CreateKeyPairOutput, error) {

	input := &ec2.CreateKeyPairInput{
		KeyName: &keyName,
	}

	output, err := client.Ec2Client.CreateKeyPair(context.TODO(), input)
	if err != nil {
		return nil, err
	}
	return output, err
}

func (client *AWSClient) DeleteKeyPair(keyName string) (*ec2.DeleteKeyPairOutput, error) {
	input := &ec2.DeleteKeyPairInput{
		KeyName: &keyName,
	}

	output, err := client.Ec2Client.DeleteKeyPair(context.TODO(), input)
	if err != nil {

		return nil, err
	}
	return output, err

}
