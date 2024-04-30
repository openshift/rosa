package aws_client

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/openshift-online/ocm-common/pkg/log"
)

func (client *AWSClient) TagResource(resourceID string, tags map[string]string) (*ec2.CreateTagsOutput, error) {
	awsTags := []types.Tag{}
	for key, value := range tags {
		Key := key
		Value := value
		tag := types.Tag{
			Key:   &Key,
			Value: &Value,
		}
		awsTags = append(awsTags, tag)
	}
	updateBody := &ec2.CreateTagsInput{
		Resources: []string{resourceID},
		Tags:      awsTags,
	}

	output, err := client.Ec2Client.CreateTags(context.TODO(), updateBody)
	if err != nil {
		log.LogError("Tag resource %s failed: %s", resourceID, err.Error())
	} else {
		log.LogInfo("Tag resource %s successfully", resourceID)
	}
	return output, err
}

func (client *AWSClient) RemoveResourceTag(resourceID string, tagKey string, tagValue string) (*ec2.DeleteTagsOutput, error) {
	tags := []types.Tag{
		types.Tag{
			Key:   &tagKey,
			Value: &tagValue,
		},
	}
	updateBody := &ec2.DeleteTagsInput{
		Resources: []string{resourceID},
		Tags:      tags,
	}
	output, err := client.Ec2Client.DeleteTags(context.TODO(), updateBody)
	if err != nil {
		log.LogError("Remove resource tag %s:%s from %s failed", tagKey, tagValue, resourceID)
	} else {
		log.LogInfo("Remove resource tag %s:%s from %s successfully", tagKey, tagValue, resourceID)
	}
	return output, err
}
