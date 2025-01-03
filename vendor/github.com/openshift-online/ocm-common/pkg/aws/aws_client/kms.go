package aws_client

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/openshift-online/ocm-common/pkg/log"
)

func (client *AWSClient) CreateKMSKeys(tagKey string, tagValue string, description string, policy string, multiRegion bool) (keyID string, keyArn string, err error) {
	//Create the key

	result, err := client.KmsClient.CreateKey(context.TODO(), &kms.CreateKeyInput{
		Tags: []types.Tag{
			{
				TagKey:   aws.String(tagKey),
				TagValue: aws.String(tagValue),
			},
		},
		Description: &description,
		Policy:      aws.String(policy),
		MultiRegion: &multiRegion,
	})

	if err != nil {
		log.LogError("Got error creating key: %s", err)

	}

	return *result.KeyMetadata.KeyId, *result.KeyMetadata.Arn, err
}

func (client *AWSClient) DescribeKMSKeys(keyID string) (kms.DescribeKeyOutput, error) {
	// Create the key
	result, err := client.KmsClient.DescribeKey(context.TODO(), &kms.DescribeKeyInput{
		KeyId: &keyID,
	})
	if err != nil {
		log.LogError("Got error describe key: %s", err)
	}
	return *result, err
}
func (client *AWSClient) ScheduleKeyDeletion(kmsKeyId string, pendingWindowInDays int32) (*kms.ScheduleKeyDeletionOutput, error) {
	result, err := client.KmsClient.ScheduleKeyDeletion(context.TODO(), &kms.ScheduleKeyDeletionInput{
		KeyId:               aws.String(kmsKeyId),
		PendingWindowInDays: &pendingWindowInDays,
	})

	if err != nil {
		log.LogError("Got error when ScheduleKeyDeletion: %s", err)
	}

	return result, err
}

func (client *AWSClient) GetKMSPolicy(keyID string, policyName string) (kms.GetKeyPolicyOutput, error) {

	if policyName == "" {
		policyName = "default"
	}
	result, err := client.KmsClient.GetKeyPolicy(context.TODO(), &kms.GetKeyPolicyInput{
		KeyId:      &keyID,
		PolicyName: &policyName,
	})
	if err != nil {
		log.LogError("Got error get KMS key policy: %s", err)
	}
	return *result, err
}

func (client *AWSClient) PutKMSPolicy(keyID string, policyName string, policy string) (kms.PutKeyPolicyOutput, error) {
	if policyName == "" {
		policyName = "default"
	}
	result, err := client.KmsClient.PutKeyPolicy(context.TODO(), &kms.PutKeyPolicyInput{
		KeyId:      &keyID,
		PolicyName: &policyName,
		Policy:     &policy,
	})
	if err != nil {
		log.LogError("Got error put KMS key policy: %s", err)
	}
	return *result, err
}

func (client *AWSClient) TagKeys(kmsKeyId string, tagKey string, tagValue string) (*kms.TagResourceOutput, error) {

	output, err := client.KmsClient.TagResource(context.TODO(), &kms.TagResourceInput{
		KeyId: &kmsKeyId,
		Tags: []types.Tag{
			{
				TagKey:   aws.String(tagKey),
				TagValue: aws.String(tagValue),
			},
		},
	})
	if err != nil {
		log.LogError("Got error add tag for KMS key: %s", err)
	}
	return output, err
}

func (client *AWSClient) ListKMSKeys() (*kms.ListKeysOutput, error) {

	result, err := client.KmsClient.ListKeys(context.TODO(), &kms.ListKeysInput{})
	if err != nil {
		log.LogError("Got error list key: %s", err)
	}
	return result, err
}
