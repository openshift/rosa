package profilehandler

import (
	"github.com/openshift-online/ocm-common/pkg/aws/aws_client"
	"github.com/openshift-online/ocm-common/pkg/test/kms_key"
	"github.com/openshift-online/ocm-common/pkg/test/vpc_client"
)

func DeleteVPCChain(vpcID string, region string) error {
	vpcClient, err := vpc_client.GenerateVPCByID(vpcID, region)
	if err != nil {
		return err
	}
	return vpcClient.DeleteVPCChain(true)
}

func ScheduleKMSDesiable(kmsKey string, region string) error {
	return kms_key.ScheduleKeyDeletion(kmsKey, region)
}

func DeleteAuditLogRoleArn(arn string, region string) error {
	awsClent, err := aws_client.CreateAWSClient("", region)
	if err != nil {
		return err
	}
	return awsClent.DeleteRoleAndPolicy(arn, false)
}
