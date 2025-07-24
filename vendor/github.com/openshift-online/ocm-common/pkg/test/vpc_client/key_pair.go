package vpc_client

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/openshift-online/ocm-common/pkg/log"
)

func (vpc *VPC) CreateKeyPair(keyName string) (*ec2.CreateKeyPairOutput, error) {
	output, err := vpc.AWSClient.CreateKeyPair(keyName)
	if err != nil {
		log.LogError("Create key pair meets error %s", err.Error())
		return nil, err
	}
	log.LogInfo("Create key pair %v successfully\n", *output.KeyPairId)

	return output, nil
}

func (vpc *VPC) getKeyPairNamesByVpcId() ([]string, error) {
	input := &ec2.DescribeKeyPairsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:VpcId"),
				Values: []string{vpc.VpcID},
			},
		},
	}

	result, err := vpc.AWSClient.Ec2Client.DescribeKeyPairs(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("describe key pairs failed: %w", err)
	}

	keyNames := make([]string, 0, len(result.KeyPairs))
	for _, kp := range result.KeyPairs {
		keyNames = append(keyNames, *kp.KeyName)
	}

	return keyNames, nil
}

func (vpc *VPC) DeleteKeyPair(keyNames []string) error {
	if len(keyNames) == 0 {
		vpcKeyNames, err := vpc.getKeyPairNamesByVpcId()
		if err != nil {
			return fmt.Errorf("get key pairs by vpc failed: %w", err)
		}
		keyNames = vpcKeyNames

		if len(keyNames) == 0 {
			log.LogInfo("No key pairs found for VPC %s", vpc.VpcID)
			return nil
		}
	}

	errorCount := 0
	for _, key := range keyNames {
		_, err := vpc.AWSClient.DeleteKeyPair(key)
		if err != nil {
			log.LogError("Delete key pair %s failed: %v (will continue)", key, err)
			errorCount++
		} else {
			log.LogInfo("Deleted key pair: %s", key)
		}
	}

	if errorCount > 0 {
		return fmt.Errorf("%d key pair deletions failed", errorCount)
	}

	log.LogInfo("Successfully deleted all key pairs")
	return nil
}
