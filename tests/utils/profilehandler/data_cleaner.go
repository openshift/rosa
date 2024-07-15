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

func DeleteHostedZone(hostedZoneID string, region string, awsSharedCredentialFile string) error {
	awsClient, err := aws_client.CreateAWSClient("", region, awsSharedCredentialFile)
	if err != nil {
		return err
	}
	return awsClient.DeleteHostedZone(hostedZoneID)
}

func DeleteSharedVPCRole(sharedVPCRoleName string, managedPolicy bool, region string,
	awsSharedCredentialFile string) error {
	awsClient, err := aws_client.CreateAWSClient("", region, awsSharedCredentialFile)
	if err != nil {
		return err
	}

	err = awsClient.DeleteRoleAndPolicy(sharedVPCRoleName, managedPolicy)
	return err
}

func DeleteAdditionalPrincipalsRole(additionalPrincipalRoleName string,
	managedPolicy bool, region string,
	awsSharedCredentialFile string) error {
	awsClient, err := aws_client.CreateAWSClient("", region, awsSharedCredentialFile)
	if err != nil {
		return err
	}

	err = awsClient.DeleteRoleAndPolicy(additionalPrincipalRoleName, managedPolicy)
	return err
}

func DeleteSharedVPCChain(vpcID string, region string, awsSharedCredentialFile string) error {
	vpcClient, err := vpc_client.GenerateVPCByID(vpcID, region, awsSharedCredentialFile)
	if err != nil {
		return err
	}
	return vpcClient.DeleteVPCChain(true)
}

func DeleteResourceShare(resourceShareArn string, region string, awsSharedCredentialFile string) error {
	awsClient, err := aws_client.CreateAWSClient("", region, awsSharedCredentialFile)
	if err != nil {
		return err
	}

	return awsClient.DeleteResourceShare(resourceShareArn)
}
