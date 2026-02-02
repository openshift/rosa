package handler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/openshift-online/ocm-common/pkg/aws/aws_client"
	"github.com/openshift-online/ocm-common/pkg/test/kms_key"
	"github.com/openshift-online/ocm-common/pkg/test/vpc_client"

	"github.com/openshift/rosa/tests/utils/log"
)

func (rh *resourcesHandler) DeleteVPCChain(withSharedAccount bool) error {
	var err error
	var awsclient *aws_client.AWSClient
	awsSharedCredentialFile := rh.awsCredentialsFile
	if withSharedAccount {
		awsSharedCredentialFile = rh.awsSharedAccountCredentialsFile
	}
	if awsSharedCredentialFile == "" {
		awsclient, err = aws_client.CreateAWSClient("", rh.resources.Region)
	} else {
		awsclient, err = aws_client.CreateAWSClient("", rh.resources.Region, awsSharedCredentialFile)
	}
	if err != nil {
		return err
	}
	if rh.vpc == nil {
		rh.vpc = vpc_client.NewVPC()
		rh.vpc.VpcID = rh.resources.VpcID
	}
	rh.vpc.AWSClient = awsclient
	return rh.vpc.DeleteVPCChain(true)
}

func (rh *resourcesHandler) DeleteKMSKey(etcdKMS bool) (err error) {
	if etcdKMS {
		log.Logger.Infof("Delete kms key: %s", rh.resources.EtcdKMSKey)
		err = kms_key.ScheduleKeyDeletion(rh.resources.EtcdKMSKey, rh.resources.Region)
	} else {
		err = kms_key.ScheduleKeyDeletion(rh.resources.KMSKey, rh.resources.Region)
	}
	if err != nil && strings.Contains(err.Error(), "is pending deletion") {
		err = nil
	}
	return
}

func (rh *resourcesHandler) DeleteAuditLogRoleArn() error {
	roleName := strings.Split(rh.resources.AuditLogArn, "/")[1]
	awsClent, err := rh.GetAWSClient(false)
	if err != nil {
		return err
	}
	return awsClent.DeleteRoleAndPolicy(roleName, false)
}

func (rh *resourcesHandler) DeleteCWLogForwardRoleArn() error {
	roleName := strings.Split(rh.resources.LogForwardConigs.Cloudwatch.CloudwatchLogRoleArn, "/")[1]
	awsClent, err := rh.GetAWSClient(false)
	if err != nil {
		return err
	}
	return awsClent.DeleteRoleAndPolicy(roleName, false)
}

func (rh *resourcesHandler) DeleteHostedZone(hostedZoneID string) error {
	awsClient, err := rh.GetAWSClient(true)
	if err != nil {
		return err
	}
	return awsClient.DeleteHostedZone(hostedZoneID)
}

func (rh *resourcesHandler) DeleteDNSDomain() error {
	_, err := rh.rosaClient.OCMResource.DeleteDNSDomain(rh.resources.DNSDomain)
	if err != nil {
		return err
	}
	return nil
}

func (rh *resourcesHandler) DeleteSharedVPCRole(managedPolicy bool) error {
	awsClient, err := rh.GetAWSClient(true)
	if err != nil {
		return err
	}

	err = awsClient.DeleteRoleAndPolicy(rh.resources.SharedVPCRole, managedPolicy)
	return err
}

func (rh *resourcesHandler) DeleteHostedCPSharedVPCRoles(managedPolicy bool) error {
	awsClient, err := rh.GetAWSClient(true)
	if err != nil {
		return err
	}

	err = awsClient.DeleteRoleAndPolicy(rh.resources.HCPRoute53ShareRole, managedPolicy)
	if err != nil {
		return err
	}
	err = awsClient.DeleteRoleAndPolicy(rh.resources.HCPVPCEndpointShareRole, managedPolicy)
	return err
}

func (rh *resourcesHandler) DeleteAdditionalPrincipalsRole(managedPolicy bool) error {
	awsClient, err := rh.GetAWSClient(true)
	if err != nil {
		return err
	}
	roleName := strings.Split(rh.resources.AdditionalPrincipals,
		"/")[len(strings.Split(rh.resources.AdditionalPrincipals, "/"))-1]
	err = awsClient.DeleteRoleAndPolicy(roleName, managedPolicy)
	return err
}

func (rh *resourcesHandler) DeleteResourceShare() error {
	awsClient, err := rh.GetAWSClient(true)
	if err != nil {
		return err
	}

	return awsClient.DeleteResourceShare(rh.resources.ResourceShareArn)
}

func (rh *resourcesHandler) DeleteOperatorRoles() error {
	_, err := rh.rosaClient.OCMResource.DeleteOperatorRoles(
		"--prefix", rh.resources.OperatorRolesPrefix,
		"--mode", "auto",
		"-y")
	if err != nil {
		return err
	}
	return nil
}

func (rh *resourcesHandler) DeleteOIDCConfig() error {
	_, err := rh.rosaClient.OCMResource.DeleteOIDCConfig(
		"--oidc-config-id",
		rh.resources.OIDCConfigID,
		"--region",
		rh.resources.Region,
		"--mode",
		"auto",
		"-y")
	if err != nil {
		return err
	}
	return nil
}

func (rh *resourcesHandler) DeleteAccountRoles() error {
	_, err := rh.rosaClient.OCMResource.DeleteAccountRole(
		"--mode", "auto",
		"--prefix", rh.resources.AccountRolesPrefix,
		"-y")
	if err != nil {
		return err
	}
	return nil
}

func (rh *resourcesHandler) DeleteOCMRole() error {
	_, err := rh.rosaClient.OCMResource.DeleteOCMRole(
		"--mode", "auto",
		"--role-arn", rh.resources.OCMRoleArn,
		"-y")
	if err != nil {
		return err
	}
	return nil
}

func (rh *resourcesHandler) DeleteUserRole() error {
	_, err := rh.rosaClient.OCMResource.DeleteUserRole(
		"--mode", "auto",
		"--role-arn", rh.resources.UserRoleArn,
		"-y")
	if err != nil {
		return err
	}
	return nil
}

func (rh *resourcesHandler) GetEIPAssociationAndAllocationIDsByInstanceID(
	publicIP string, sharedVPC bool,
) (string, string, error) {
	var (
		err       error
		awsClient *aws_client.AWSClient
	)
	if sharedVPC {
		awsClient, err = rh.GetAWSClient(true)
	} else {
		awsClient, err = rh.GetAWSClient(false)
	}
	if err != nil {
		log.Logger.Errorf("Get AWS Client failed: %s", err)
		return "", "", err
	}
	input := &ec2.DescribeAddressesInput{
		PublicIps: []string{publicIP},
	}
	output, err := awsClient.Ec2Client.DescribeAddresses(context.TODO(), input)
	if err != nil {
		log.Logger.Errorf("Failed to describe addresses: %s", err)
		return "", "", err
	}

	if len(output.Addresses) == 0 {
		return "", "", fmt.Errorf("no EIP association found for instance ID %s", publicIP)
	}

	associationID := *output.Addresses[0].AssociationId
	allocationID := *output.Addresses[0].AllocationId
	return associationID, allocationID, nil
}

func (rh *resourcesHandler) CleanupProxyResources(instID string, sharedVPC bool) error {
	var (
		err       error
		awsClient *aws_client.AWSClient
	)

	if sharedVPC {
		awsClient, err = rh.GetAWSClient(true)
	} else {
		awsClient, err = rh.GetAWSClient(false)
	}
	if err != nil {
		log.Logger.Errorf("Get AWS Client failed: %s", err)
		return err
	}
	insOut, err := awsClient.Ec2Client.DescribeInstances(
		context.TODO(),
		&ec2.DescribeInstancesInput{
			InstanceIds: []string{instID},
		},
	)
	if err != nil {
		log.Logger.Errorf("Describe proxy instance failed: %s", err)
		return err
	}
	if len(insOut.Reservations) == 0 || len(insOut.Reservations[0].Instances) == 0 {
		err = fmt.Errorf("instance %s not found", rh.resources.ProxyInstanceID)
		return err
	}
	instance := insOut.Reservations[0].Instances[0]
	// release EIP
	associationID, allocationID, err := rh.GetEIPAssociationAndAllocationIDsByInstanceID(
		*instance.PublicIpAddress, sharedVPC,
	)
	keyName := *instance.KeyName
	SGID := *instance.SecurityGroups[0].GroupId
	if err != nil {
		log.Logger.Errorf("Get EIP Association ID failed: %s", err)
		return err
	}
	_, err = awsClient.DisassociateAddress(associationID)
	if err != nil {
		log.Logger.Errorf("Disassociate Addrress failed: %s", err)
		return err
	}
	err = awsClient.ReleaseAddressWithAllocationID(allocationID)
	if err != nil {
		log.Logger.Errorf("Release Address failed: %s", err)
		return err
	}
	log.Logger.Infof("Released EIP: %s", allocationID)

	// terminate proxy instance
	_, err = awsClient.Ec2Client.TerminateInstances(context.TODO(), &ec2.TerminateInstancesInput{
		InstanceIds: []string{instID},
	})
	if err != nil {
		log.Logger.Errorf("Terminate instance failed: %s", err)
		return err
	}
	log.Logger.Infof("Terminating instance: %s", instID)

	err = wait.PollUntilContextTimeout(
		context.TODO(),
		30*time.Second,
		10*time.Minute,
		false,
		func(ctx context.Context) (bool, error) {
			output, err := awsClient.Ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
				InstanceIds: []string{instID},
			})
			if err != nil {
				return false, err
			}
			if len(output.Reservations) == 0 || len(output.Reservations[0].Instances) == 0 {
				return false, nil
			}
			instance := output.Reservations[0].Instances[0]
			if instance.State != nil && instance.State.Name == types.InstanceStateNameTerminated {
				return true, nil
			}
			return false, nil
		},
	)
	if err != nil {
		log.Logger.Errorf("Instance termination confirmation failed: %s", err)
		return err
	}
	log.Logger.Infof("Instance %s is terminated", instID)

	// Delete secrity group
	_, err = awsClient.DeleteSecurityGroup(SGID)
	if err != nil {
		log.Logger.Errorf("Delete security group failed: %s", err)
		return err
	}
	log.Logger.Infof("Deleted security group: %s", SGID)

	// Delete key pair
	_, err = awsClient.DeleteKeyPair(keyName)
	if err != nil {
		log.Logger.Errorf("Delete key pair failed: %s", err)
		return err
	}
	log.Logger.Infof("Deleted key pair: %s", keyName)

	return nil
}
