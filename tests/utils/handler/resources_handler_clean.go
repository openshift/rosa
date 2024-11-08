package handler

import (
	"strings"

	"github.com/openshift-online/ocm-common/pkg/test/kms_key"

	"github.com/openshift/rosa/tests/utils/log"
)

func (rh *resourcesHandler) DeleteVPCChain() error {
	var err error
	if rh.vpc == nil {
		rh.vpc, err = rh.PrepareVPC(rh.resources.VpcID, "", true)
		if err != nil {
			return err
		}
	}
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

func (rh *resourcesHandler) DeleteHostedZone() error {
	awsClient, err := rh.GetAWSClient(true)
	if err != nil {
		return err
	}
	return awsClient.DeleteHostedZone(rh.resources.HostedZoneID)
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

func (rh *resourcesHandler) DeleteAdditionalPrincipalsRole(managedPolicy bool) error {
	awsClient, err := rh.GetAWSClient(true)
	if err != nil {
		return err
	}

	err = awsClient.DeleteRoleAndPolicy(rh.resources.AdditionalPrincipals, managedPolicy)
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
