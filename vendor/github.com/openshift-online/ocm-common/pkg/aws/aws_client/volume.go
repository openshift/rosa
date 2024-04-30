package aws_client

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/openshift-online/ocm-common/pkg/log"
)

func (client *AWSClient) DescribeVolumeByID(volumeID string) (*ec2.DescribeVolumesOutput, error) {

	output, err := client.Ec2Client.DescribeVolumes(context.TODO(), &ec2.DescribeVolumesInput{
		VolumeIds: []string{volumeID},
	})

	if err != nil {
		log.LogError("Got error describe volume: %s", err)
	}
	return output, err
}
