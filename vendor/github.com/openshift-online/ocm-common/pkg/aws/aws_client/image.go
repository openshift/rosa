package aws_client

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/openshift-online/ocm-common/pkg/aws/consts"
	"github.com/openshift-online/ocm-common/pkg/log"
)

func (client *AWSClient) CopyImage(sourceImageID string, sourceRegion string, name string) (string, error) {
	copyImageInput := &ec2.CopyImageInput{
		Name:          &name,
		SourceImageId: &sourceImageID,
		SourceRegion:  &sourceRegion,
	}
	output, err := client.EC2().CopyImage(context.TODO(), copyImageInput)
	if err != nil {
		log.LogError("Error happens when copy image: %s", err)
		return "", err
	}
	return *output.ImageId, nil
}

func (client *AWSClient) DescribeImage(imageIDs []string, filters ...map[string][]string) (*ec2.DescribeImagesOutput, error) {
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

	describeImageInput := &ec2.DescribeImagesInput{
		Owners:  []string{consts.AmazonName},
		Filters: filterInput,
	}

	if len(imageIDs) != 0 {
		describeImageInput.ImageIds = imageIDs
	}
	output, err := client.EC2().DescribeImages(context.TODO(), describeImageInput)
	if err != nil {
		log.LogError("Describe image %s meet error: %s", imageIDs, err)
		return nil, err
	}

	return output, nil
}
