package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/openshift-online/ocm-common/pkg/aws/aws_client"
	"github.com/openshift-online/ocm-common/pkg/test/kms_key"
	"github.com/openshift-online/ocm-common/pkg/test/vpc_client"

	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/constants"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/helper"
	"github.com/openshift/rosa/tests/utils/log"
)

func (rh *resourcesHandler) PrepareVersion(versionRequirement string,
	channelGroup string,
	hcp bool,
) (*rosacli.OpenShiftVersionTableOutput, error) {
	log.Logger.Infof("Got version requirement %s going to prepare accordingly", versionRequirement)
	log.Logger.Infof("Channel group = %s", channelGroup)
	versionList, err := rh.rosaClient.Version.ListAndReflectVersions(channelGroup, hcp)
	if err != nil {
		return nil, err
	}

	if constants.VersionLatestPattern.MatchString(versionRequirement) {
		return versionList.Latest()
	} else if constants.VersionMajorMinorPattern.MatchString(versionRequirement) {
		version, err := versionList.FindNearestBackwardMinorVersion(versionRequirement, 0, true)
		return version, err
	} else if constants.VersionRawPattern.MatchString(versionRequirement) {
		return &rosacli.OpenShiftVersionTableOutput{
			Version: versionRequirement,
		}, nil
	} else if constants.VersionFlexyPattern.MatchString(versionRequirement) {
		log.Logger.Debugf("Version requirement matched %s", constants.VersionFlexyPattern.String())
		latestVersion, err := versionList.Latest()
		if err != nil {
			return nil, err
		}
		log.Logger.Infof("Got the latest version id %s", latestVersion.Version)
		stream, step := strings.Split(versionRequirement, "-")[0], strings.Split(versionRequirement, "-")[1]
		versionStep, err := strconv.Atoi(step)
		if err != nil {
			return nil, err
		}
		log.Logger.Infof("Going to prepare version for %s stream %v versions lower", stream, versionStep)
		switch stream {
		case "y":
			var version *rosacli.OpenShiftVersionTableOutput
			version, err = versionList.FindYStreamUpgradableVersion(latestVersion.Version)
			return version, err
		case "z":
			var version *rosacli.OpenShiftVersionTableOutput
			version, err := versionList.FindZStreamUpgradableVersion(latestVersion.Version, versionStep)
			return version, err
		default:
			return nil, fmt.Errorf("not supported stream configuration %s", stream)
		}
	}
	return nil, fmt.Errorf("not supported version requirement: %s", versionRequirement)
}

// PrepareNames will generate the name for cluster creation
// if longname is set, it will generate the long name with constants.DefaultLongClusterNamelength
func (rh *resourcesHandler) PreparePrefix(profilePrefix string, nameLength int) string {
	if nameLength > ocm.MaxClusterNameLength {
		panic(fmt.Errorf("name length %d is longer than allowed max name length %d", nameLength, ocm.MaxClusterNameLength))
	}

	if len(profilePrefix) > nameLength {
		newProfilePrefix := helper.TrimNameByLength(profilePrefix, nameLength-4)
		log.Logger.Warnf("Profile name prefix %s is longer than "+
			"the nameLength for random generated. Trimed it to %s", profilePrefix, newProfilePrefix)
		profilePrefix = newProfilePrefix
	}
	return strings.TrimSuffix(helper.GenerateRandomName(profilePrefix, nameLength-len(profilePrefix)-1), "-")
}

// PrepareVPC will prepare a single vpc
func (rh *resourcesHandler) PrepareVPC(vpcName string, cidrValue string, useExisting bool, withSharedAccount bool) (
	*vpc_client.VPC, error) {
	log.Logger.Infof("Starting vpc preparation on region %s", rh.resources.Region)
	credentialFile := rh.awsCredentialsFile
	if withSharedAccount {
		credentialFile = rh.awsSharedAccountCredentialsFile
	}
	vpc, err := vpc_client.PrepareVPC(vpcName, rh.resources.Region, cidrValue, useExisting, credentialFile)
	if err != nil {
		return nil, err
	}
	rh.vpc = vpc
	err = rh.registerVpcID(vpc.VpcID, withSharedAccount)
	log.Logger.Info("VPC preparation finished")
	if err != nil {
		return vpc, err
	}
	err = rh.registerVPC(vpc)
	return vpc, err
}

// This AddTagsToSharedVPCBYOSubnets is to add tags 'kubernetes.io/role/internal-elb' and 'kubernetes.io/role/elb'
// on the shared subnets on cluster owner aws account
func (rh *resourcesHandler) AddTagsToSharedVPCBYOSubnets(subnets config.Subnets, region string) error {
	pubTags := map[string]string{
		"kubernetes.io/role/elb": "",
	}
	privateTags := map[string]string{
		"kubernetes.io/role/internal-elb": "",
	}
	awsclient, err := aws_client.CreateAWSClient("", region)
	if err != nil {
		return err
	}

	for _, pubSubnetID := range strings.Split(subnets.PublicSubnetIds, ",") {
		// Wait for the subnet id to be found by the aws client
		err = wait.PollUntilContextTimeout(
			context.Background(),
			20*time.Second,
			300*time.Second,
			false,
			func(context.Context) (bool, error) {
				_, err = awsclient.Ec2Client.DescribeSubnets(context.TODO(), &ec2.DescribeSubnetsInput{
					SubnetIds: []string{pubSubnetID},
				})
				if err != nil {
					if strings.Contains(err.Error(), "does not exist") {
						return false, nil
					}
					return false, err
				}
				return true, err
			})
		if err != nil {
			return fmt.Errorf("wait for subnet %s to be found failed: %s", pubSubnetID, err)
		}
		_, err = awsclient.TagResource(pubSubnetID, pubTags)
		if err != nil {
			return fmt.Errorf("tag subnet %s failed:%s", pubSubnetID, err)
		}
	}
	for _, priSubnetID := range strings.Split(subnets.PrivateSubnetIds, ",") {
		// Wait for the subnet id to be found by the aws client
		err = wait.PollUntilContextTimeout(
			context.Background(),
			20*time.Second,
			300*time.Second,
			false,
			func(context.Context) (bool, error) {
				_, err = awsclient.Ec2Client.DescribeSubnets(context.TODO(), &ec2.DescribeSubnetsInput{
					SubnetIds: []string{priSubnetID},
				})
				if err != nil {
					if strings.Contains(err.Error(), "does not exist") {
						return false, nil
					}
					return false, err
				}
				return true, err
			})
		if err != nil {
			return fmt.Errorf("wait for subnet %s to be found failed: %s", priSubnetID, err)
		}
		_, err = awsclient.TagResource(priSubnetID, privateTags)
		if err != nil {
			return fmt.Errorf("tag subnet %s failed:%s", priSubnetID, err)
		}
	}
	return nil
}

// This AddTagsToUnManagedBYOSubnets is to add tags 'kubernetes.io/cluster/unmanaged:true'
// This is a new change for 4.19+ version cluster
func (rh *resourcesHandler) AddTagsToUnManagedBYOSubnets(subnets []string, region string) error {
	tags := map[string]string{
		"kubernetes.io/cluster/unmanaged": "true",
	}
	awsclient, err := aws_client.CreateAWSClient("", region)
	if err != nil {
		return err
	}

	for _, subnetID := range subnets {
		// Wait for the subnet id to be found by the aws client
		err = wait.PollUntilContextTimeout(
			context.Background(),
			20*time.Second,
			300*time.Second,
			false,
			func(context.Context) (bool, error) {
				_, err = awsclient.Ec2Client.DescribeSubnets(context.TODO(), &ec2.DescribeSubnetsInput{
					SubnetIds: []string{subnetID},
				})
				if err != nil {
					if strings.Contains(err.Error(), "does not exist") {
						return false, nil
					}
					return false, err
				}
				return true, err
			})
		if err != nil {
			return fmt.Errorf("wait for subnet %s to be found failed: %s", subnetID, err)
		}
		_, err = awsclient.TagResource(subnetID, tags)
		if err != nil {
			return fmt.Errorf("tag subnet %s failed:%s", subnetID, err)
		}
	}
	return nil
}

// PrepareSubnets will prepare pair of subnets according to the vpcID and zones
// if zones are empty list it will list the zones and pick according to multi-zone parameter.
// when multi-zone=true, 3 zones will be pickup
func (rh *resourcesHandler) PrepareSubnets(zones []string, multiZone bool) (map[string][]string, error) {
	if rh.vpc == nil {
		return nil, errors.New("VPC has not been initialized")
	}
	resultMap := map[string][]string{}
	if len(zones) == 0 {
		log.Logger.Info("Got no zones indicated. List the zones and pick from the listed zones")
		resultZones, err := rh.vpc.AWSClient.ListAvaliableZonesForRegion(rh.resources.Region, "availability-zone")
		if err != nil {
			return resultMap, err
		}
		zones = resultZones[0:1]
		if multiZone {
			zones = resultZones[0:3]
		}
	}
	for _, zone := range zones {
		subnetMap, err := rh.vpc.PreparePairSubnetByZone(zone)
		if err != nil {
			return resultMap, err
		}
		for subnetType, subnet := range subnetMap {
			if _, ok := resultMap[subnetType]; !ok {
				resultMap[subnetType] = []string{}
			}
			resultMap[subnetType] = append(resultMap[subnetType], subnet.ID)
		}
	}
	err := rh.registerVPC(rh.vpc)
	return resultMap, err
}

func (rh *resourcesHandler) PrepareProxy(zone string, sshPemFileName string, sshPemFileRecordDir string,
	caFile string) (*ProxyDetail, error) {
	return rh.PrepareProxyWithAuth(zone, sshPemFileName, sshPemFileRecordDir, caFile, "", "")
}

func (rh *resourcesHandler) PrepareProxyWithAuth(zone string, sshPemFileName string, sshPemFileRecordDir string,
	caFile string, username string, password string) (*ProxyDetail, error) {

	if rh.vpc == nil {
		return nil, errors.New("VPC has not been initialized")
	}
	instance, privateIP, caContent, err := rh.vpc.LaunchProxyInstanceWithAuth(
		zone, sshPemFileName, sshPemFileRecordDir, username, password)
	if err != nil {
		return nil, err
	}
	_, err = helper.CreateFileWithContent(caFile, caContent)
	if err != nil {
		return nil, err
	}
	err = rh.registerProxyInstanceID(*instance.InstanceId)
	return &ProxyDetail{
		HTTPsProxy:       rh.vpc.GetHTTPSProxyURL(privateIP, username, password),
		HTTPProxy:        rh.vpc.GetProxyURL(privateIP, username, password),
		CABundleFilePath: caFile,
		NoProxy:          "quay.io",
		InstanceID:       *instance.InstanceId,
	}, err
}

func (rh *resourcesHandler) PrepareKMSKey(multiRegion bool, testClient string, hcp bool, etcdKMS bool) (string, error) {
	keyArn, err := kms_key.CreateOCMTestKMSKey(rh.resources.Region, multiRegion, testClient)
	if err != nil {
		return keyArn, err
	}
	if etcdKMS {
		err = rh.registerEtcdKMSKey(keyArn)
	} else {
		err = rh.registerKMSKey(keyArn)
	}
	if err != nil {
		return keyArn, err
	}
	if hcp {
		kms_key.AddTagToKMS(keyArn, rh.resources.Region, map[string]string{
			"red-hat": "true",
		})
	}
	return keyArn, err
}

func (rh *resourcesHandler) PrepareAdditionalSecurityGroups(
	securityGroupCount int,
	namePrefix string) ([]string, error) {
	if rh.vpc == nil {
		return nil, errors.New("VPC has not been initialized")
	}

	return rh.vpc.CreateAdditionalSecurityGroups(securityGroupCount, namePrefix, "")
}

func (rh *resourcesHandler) PrepareZeroEgressResources() error {
	if rh.vpc == nil {
		return errors.New("VPC has not been initialized")
	}

	//STEP1
	sgOutput, err := rh.vpc.AWSClient.CreateSecurityGroup(rh.vpc.VpcID,
		"allow-inbound-traffic", "allow inbound traffic")
	if err != nil {
		return err
	}
	//STEP2
	_, err = rh.vpc.AWSClient.AuthorizeSecurityGroupIngress(*sgOutput.GroupId, rh.vpc.CIDRValue, "-1", 0, 0)
	if err != nil {
		return err
	}
	//STEP3
	err = rh.vpc.AWSClient.CreateVPCEndpoint(rh.vpc.VpcID,
		fmt.Sprintf("com.amazonaws.%s.ecr.dkr", rh.vpc.Region), "Interface")
	if err != nil {
		return err
	}
	//STEP4
	err = rh.vpc.AWSClient.CreateVPCEndpoint(rh.vpc.VpcID,
		fmt.Sprintf("com.amazonaws.%s.s3", rh.vpc.Region), "Interface")
	if err != nil {
		return err
	}
	return err
}

// To prepare ocm role
func (rh *resourcesHandler) PrepareOCMRole(
	ocmRolePrefix string,
	admin bool,
	path string) (
	ocmRole *rosacli.OCMRole, err error) {
	// Assemble flags
	var flags []string
	if path != "" {
		flags = append(flags, "--path", path)
	}
	if admin {
		flags = append(flags, "--admin")
	}

	ocmResourceService := rh.rosaClient.OCMResource

	// Get account info
	rh.rosaClient.Runner.JsonFormat()
	whoamiOutput, err := ocmResourceService.Whoami()
	if err != nil {
		err = fmt.Errorf("error happens when get account information, %s", err.Error())
		return
	}
	rh.rosaClient.Runner.UnsetFormat()
	whoamiData := ocmResourceService.ReflectAccountsInfo(whoamiOutput)
	ocmOrganizationExternalID := whoamiData.OCMOrganizationExternalID

	// Check if there's already an OCM role created that is already linked
	ocmRoleList, output, err := ocmResourceService.ListOCMRole()
	if err != nil {
		err = fmt.Errorf("error happens when list ocm role before cluster preparation, %s", output.String())
		return
	}
	if ocmRoleList.OCMRole(ocmRolePrefix, ocmOrganizationExternalID).Linded == "Yes" {
		return
	}
	linkedRole := ocmRoleList.FindLinkedOCMRole()
	if (linkedRole != rosacli.OCMRole{}) {
		log.Logger.Infof("There's already an existing linked OCM role '%s'", linkedRole.RoleArn)
		return
	}

	// Check if there's any linked OCM roles via the API
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()
	roles, err := r.OCMClient.GetOrganizationLinkedOCMRoles(whoamiData.OCMOrganizationID)
	if err != nil {
		err = fmt.Errorf("error happens when checking for existing linked OCM roles: %s", err.Error())
		return
	}
	for _, role := range roles {
		if role != "" {
			output, err := ocmResourceService.UnlinkOCMRole("--role-arn", role, "-y")
			if err != nil {
				err = fmt.Errorf("error happens when unlinking existing OCM role: %s", output.String())
				return nil, err
			}
		}
	}

	// Create the actual OCM role
	flags = append(flags, "--prefix", ocmRolePrefix, "--mode", "auto", "-y")
	output, err = ocmResourceService.CreateOCMRole(
		flags...,
	)
	if err != nil {
		err = fmt.Errorf("error happens when create ocm role, %s", output.String())
		return
	}
	ocmRoleList, output, err = ocmResourceService.ListOCMRole()
	if err != nil {
		err = fmt.Errorf("error happens when list ocm role during cluster preparation, %s", output.String())
		return
	}
	ocmrole := ocmRoleList.OCMRole(ocmRolePrefix, ocmOrganizationExternalID)

	err = rh.registerOCMRoleArn(ocmrole.RoleArn)
	if err != nil {
		return
	}
	return &ocmrole, nil
}

// To prepare user role
func (rh *resourcesHandler) PrepareUserRole(
	userRolePrefix string,
	path string) (
	userole *rosacli.UserRole, err error) {
	// Assemble creation flags
	var flags []string
	if path != "" {
		flags = append(flags, "--path", path)
	}

	ocmResourceService := rh.rosaClient.OCMResource

	// Check for existing linked user role
	userRoleList, output, err := ocmResourceService.ListUserRole()
	if err != nil {
		err = fmt.Errorf("error happens when listing user roles, %s", output.String())
		return
	}
	linkedUserRole := userRoleList.FindLinkedUserRole()
	if (linkedUserRole != rosacli.UserRole{}) {
		log.Logger.Infof("There is already an existing linked user role '%s'", linkedUserRole.RoleArn)
		return
	}

	// Get account info
	rh.rosaClient.Runner.JsonFormat()
	whoamiOutput, err := ocmResourceService.Whoami()
	if err != nil {
		err = fmt.Errorf("error happens when get account information, %s", err.Error())
		return
	}
	rh.rosaClient.Runner.UnsetFormat()
	whoamiData := ocmResourceService.ReflectAccountsInfo(whoamiOutput)

	// Check if there's any linked user roles via the API
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()
	roles, err := r.OCMClient.GetAccountLinkedUserRoles(whoamiData.OCMAccountID)
	if err != nil {
		err = fmt.Errorf("error happens when checking for existing linked user roles: %s", err.Error())
		return
	}
	for _, role := range roles {
		if role != "" {
			output, err := ocmResourceService.UnlinkUserRole("--role-arn", role, "-y")
			if err != nil {
				err = fmt.Errorf("error happens when unlinking existing user role: %s", output.String())
				return nil, err
			}
		}
	}

	// Create the user role
	ocmAccountUsername := whoamiData.OCMAccountUsername
	flags = append(flags, "--prefix", userRolePrefix, "--mode", "auto", "-y")
	output, err = ocmResourceService.CreateUserRole(
		flags...,
	)
	if err != nil {
		err = fmt.Errorf("error happens when create user role, %s", output.String())
		return
	}
	userRoleList, output, err = ocmResourceService.ListUserRole()
	if err != nil {
		err = fmt.Errorf("error happens when list user role during cluster preparation, %s", output.String())
		return
	}
	userRole := userRoleList.UserRole(userRolePrefix, ocmAccountUsername)

	err = rh.registerUserRoleArn(userRole.RoleArn)
	if err != nil {
		return
	}
	return &userRole, nil
}

// PrepareAccountRoles will prepare account roles according to the parameters
// openshiftVersion must follow 4.15.2-x format
func (rh *resourcesHandler) PrepareAccountRoles(
	namePrefix string,
	hcp bool,
	openshiftVersion string,
	channelGroup string,
	path string,
	permissionsBoundary string,
	route53RoleARN string,
	vpcEndpointRoleArn string) (
	accRoles *rosacli.AccountRolesUnit, err error) {
	var flags []string
	if route53RoleARN != "" && vpcEndpointRoleArn != "" {
		flags = rh.generateAccountRoleCreationFlag(
			namePrefix,
			hcp,
			openshiftVersion,
			channelGroup,
			path,
			permissionsBoundary,
			route53RoleARN,
			vpcEndpointRoleArn,
		)
	} else {
		flags = rh.generateAccountRoleCreationFlag(
			namePrefix,
			hcp,
			openshiftVersion,
			channelGroup,
			path,
			permissionsBoundary,
			"",
			"",
		)
	}

	ocmResourceService := rh.rosaClient.OCMResource
	output, err := ocmResourceService.CreateAccountRole(
		flags...,
	)
	if err != nil {
		err = fmt.Errorf("error happens when create account-roles, %s", output.String())
		return
	}
	err = rh.registerAccountRolesPrefix(namePrefix)
	if err != nil {
		return
	}
	var accoutRoles *rosacli.AccountRolesUnit
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		accRoleList, output, err := ocmResourceService.ListAccountRole()
		if err != nil && strings.Contains(err.Error(), "cannot be found") {
			log.Logger.Infof("Some IAM role cannot be found, Retrying... (%d/%d)\n", i+1, maxRetries)
			continue
		} else if err != nil && !strings.Contains(err.Error(), "cannot be found") {
			err = fmt.Errorf("error happens when list account-roles, %s", output.String())
			return &rosacli.AccountRolesUnit{}, err
		} else {
			accoutRoles = accRoleList.DigAccountRoles(namePrefix, hcp)
			break
		}
	}
	return accoutRoles, nil

}

// PrepareOperatorRolesByOIDCConfig will prepare operator roles with OIDC config ID
// When sharedVPCRoleArn is not empty it will be set to the flag
func (rh *resourcesHandler) PrepareOperatorRolesByOIDCConfig(
	namePrefix string,
	oidcConfigID string,
	roleArn string,
	sharedRoute53RoleArn string,
	sharedVPCEndPointRoleArn string,
	hcp bool, channelGroup string) error {

	flags := []string{
		"-y",
		"--mode", "auto",
		"--prefix", namePrefix,
		"--role-arn", roleArn,
		"--oidc-config-id", oidcConfigID,
		"--channel-group", channelGroup,
	}
	if hcp {
		flags = append(flags, "--hosted-cp")
	}
	if sharedRoute53RoleArn != "" {
		flags = append(flags, "--route53-role-arn", sharedRoute53RoleArn)
	}
	if sharedVPCEndPointRoleArn != "" {
		flags = append(flags, "--vpc-endpoint-role-arn", sharedVPCEndPointRoleArn)
	}
	_, err := rh.rosaClient.OCMResource.CreateOperatorRoles(
		flags...,
	)
	if err != nil {
		return err
	}
	err = rh.registerOperatorRolesPrefix(namePrefix)
	return err
}

func (rh *resourcesHandler) PrepareAdminUser() (string, string) {
	userName := helper.GenerateRandomString(10)
	password := helper.GenerateRandomStringWithSymbols(14)
	return userName, password
}

func (rh *resourcesHandler) PrepareAuditlogRoleArnByOIDCConfig(
	auditLogRoleName string,
	oidcConfigID string) (string, error) {

	oidcConfig, err := rh.rosaClient.OCMResource.GetOIDCConfigFromList(oidcConfigID)
	if err != nil {
		return "", err
	}
	logRoleArn, err := rh.PrepareAuditlogRoleArnByIssuer(auditLogRoleName, oidcConfig.IssuerUrl)
	if err != nil {
		return logRoleArn, err
	}
	err = rh.registerAuditLogArn(logRoleArn)
	return logRoleArn, err

}

func (rh *resourcesHandler) PrepareAuditlogRoleArnByIssuer(auditLogRoleName string,
	oidcIssuerURL string) (string, error) {

	//nolint:staticcheck,all
	oidcIssuerURL = strings.TrimLeft(oidcIssuerURL, "https://")
	log.Logger.Infof("Preparing audit log role with name %s and oidcIssuerURL %s", auditLogRoleName, oidcIssuerURL)
	awsClient, err := rh.GetAWSClient(false)
	if err != nil {
		return "", err
	}
	policyName := fmt.Sprintf("%s-%s", "auditlogpolicy", auditLogRoleName)
	policyArn, err := awsClient.CreatePolicyForAuditLogForward(policyName)
	if err != nil {
		log.Logger.Errorf("Error happens when prepare audit log policy: %s", err.Error())
		return "", err
	}
	roleArn, err := awsClient.CreateRoleForAuditLogForward(auditLogRoleName, awsClient.AccountID, oidcIssuerURL, policyArn)
	auditLogRoleArn := aws.ToString(roleArn.Arn)
	if err != nil {
		log.Logger.Errorf("Error happens when prepare audit log role: %s", err.Error())
		return auditLogRoleArn, err
	}
	log.Logger.Infof("Create a new role for audit log forwarding: %s", auditLogRoleArn)
	err = rh.registerAuditLogArn(auditLogRoleArn)
	if err != nil {
		log.Logger.Errorf("Error happened when record audit log role: %s", err.Error())
		return auditLogRoleArn, err
	}
	err = awsClient.AttachIAMPolicy(auditLogRoleName, policyArn)
	if err != nil {
		log.Logger.Errorf("Error happens when attach audit log policy %s to role %s: %s",
			policyArn, auditLogRoleName, err.Error())
	}

	return auditLogRoleArn, err
}

func (rh *resourcesHandler) PrepareOperatorRolesByCluster(cluster string) error {
	flags := []string{
		"-y",
		"--mode", "auto",
		"--cluster", cluster,
	}
	_, err := rh.rosaClient.OCMResource.CreateOperatorRoles(
		flags...,
	)
	return err
}

// PrepareOIDCConfig will prepare the oidc config for the cluster,
// if the oidcConfigType="managed", roleArn and prefix won't be set
func (rh *resourcesHandler) PrepareOIDCConfig(
	oidcConfigType string,
	roleArn string,
	prefix string) (string, error) {

	var oidcConfigID string
	var output bytes.Buffer
	var err error
	switch oidcConfigType {
	case "managed":
		output, err = rh.rosaClient.OCMResource.CreateOIDCConfig(
			"-o", "json",
			"--mode", "auto",
			"--region", rh.resources.Region,
			"--managed",
			"-y",
		)
	case "unmanaged":
		output, err = rh.rosaClient.OCMResource.CreateOIDCConfig(
			"-o", "json",
			"--mode", "auto",
			"--prefix", prefix,
			"--region", rh.resources.Region,
			"--role-arn", roleArn,
			"--managed=false",
			"-y",
		)

	default:
		return "", fmt.Errorf("only 'managed' or 'unmanaged' oidc config is allowed")
	}
	if err != nil {
		return oidcConfigID, err
	}
	parser := rosacli.NewParser()
	oidcConfigID = parser.JsonData.Input(output).Parse().DigString("id")
	err = rh.registerOIDCConfigID(oidcConfigID)
	return oidcConfigID, err
}

func (rh *resourcesHandler) PrepareOIDCProvider(oidcConfigID string) error {
	_, err := rh.rosaClient.OCMResource.CreateOIDCProvider(
		"--mode", "auto",
		"-y",
		"--oidc-config-id", oidcConfigID,
	)
	return err
}
func (rh *resourcesHandler) PrepareOIDCProviderByCluster(cluster string) error {
	_, err := rh.rosaClient.OCMResource.CreateOIDCProvider(
		"--mode", "auto",
		"-y",
		"--cluster", cluster,
	)
	return err
}

func (rh *resourcesHandler) PrepareSharedVPCRole(sharedVPCRolePrefix string, installerRoleArn string,
	ingressOperatorRoleArn string) (string, string, error) {

	awsClient, err := rh.GetAWSClient(true)
	if err != nil {
		return "", "", err
	}

	policyName := fmt.Sprintf("%s-%s", sharedVPCRolePrefix, "shared-vpc-policy")
	policyArn, err := awsClient.CreatePolicyForSharedVPC(policyName)
	if err != nil {
		log.Logger.Errorf("Error happens when prepare shared vpc policy: %s", err.Error())
		return "", "", err
	}

	roleName := fmt.Sprintf("%s-%s", sharedVPCRolePrefix, "shared-vpc-role")
	if installerRoleArn == "" {
		log.Logger.Errorf("Can not create shared vpc role due to no installer role.")
		return "", "", err
	}
	log.Logger.Debugf("Got installer role arn: %s for shared vpc role preparation", installerRoleArn)
	log.Logger.Debugf("Got ingress role arn: %s for shared vpc role preparation", ingressOperatorRoleArn)

	roleArn, err := awsClient.CreateRoleForSharedVPC(roleName, installerRoleArn, ingressOperatorRoleArn)
	sharedVPCRoleArn := aws.ToString(roleArn.Arn)
	if err != nil {
		log.Logger.Errorf("Error happens when prepare shared vpc role: %s", err.Error())
		return roleName, sharedVPCRoleArn, err
	}
	log.Logger.Infof("Create a new role for shared VPC: %s", sharedVPCRoleArn)
	err = rh.registerSharedVPCRole(roleName)
	if err != nil {
		log.Logger.Errorf("Error happened when record shared VPC role: %s", err.Error())
		return roleName, sharedVPCRoleArn, err
	}
	err = awsClient.AttachIAMPolicy(roleName, policyArn)
	if err != nil {
		log.Logger.Errorf("Error happens when attach shared VPC policy %s to role %s: %s", policyArn,
			sharedVPCRoleArn, err.Error())
	}
	return roleName, sharedVPCRoleArn, err
}

func (rh *resourcesHandler) PrepareSharedRoute53RoleForHostedCP(route53RolePrefix string, installerRoleArn string,
	ingressOperatorRoleArn string, controlPlaneOperatorRoleArn string) (string, string, error) {

	awsClient, err := rh.GetAWSClient(true)
	if err != nil {
		return "", "", err
	}

	policyName := fmt.Sprintf("%s-%s", route53RolePrefix, "sharedvpc-r53-policy")
	sharedRoute53PolictStatement := map[string]interface{}{
		"Sid":    "Statement1",
		"Effect": "Allow",
		"Action": []string{
			"route53:ChangeResourceRecordSets",
			"route53:ListHostedZones",
			"route53:ListHostedZonesByName",
			"route53:ListResourceRecordSets",
			"route53:ChangeTagsForResource",
			"route53:GetAccountLimit",
			"route53:GetChange",
			"route53:GetHostedZone",
			"route53:ListTagsForResource",
			"route53:UpdateHostedZoneComment",
			"tag:GetResources",
			"tag:UntagResources",
		},
		"Resource": "*",
	}
	policyArn, err := awsClient.CreatePolicy(policyName, sharedRoute53PolictStatement)
	if err != nil {
		log.Logger.Errorf("Error happens when prepare shared route53 policy: %s", err.Error())
		return "", "", err
	}

	roleName := fmt.Sprintf("%s-%s", route53RolePrefix, "shared-route53-role")
	if installerRoleArn == "" || controlPlaneOperatorRoleArn == "" || ingressOperatorRoleArn == "" {
		log.Logger.Errorf(
			"Can not create shared vpc route53 role due to no installer role or ingress/controlplane operator role.",
		)
		return "", "", err
	}
	log.Logger.Debugf("Got installer role arn: %s for shared route53 role preparation", installerRoleArn)
	log.Logger.Debugf("Got ingress role arn: %s for shared route53 role preparation", ingressOperatorRoleArn)
	log.Logger.Debugf(
		"Got control plane operator role arn: %s for shared route53 role preparation",
		controlPlaneOperatorRoleArn,
	)

	assumeRolesArns := []string{installerRoleArn, ingressOperatorRoleArn, controlPlaneOperatorRoleArn}
	roleArn, err := awsClient.CreateRoleForSharedVPCHCP(roleName, assumeRolesArns)
	sharedVPCRoute53RoleArn := aws.ToString(roleArn.Arn)
	if err != nil {
		log.Logger.Errorf("Error happens when prepare shared route53 role: %s", err.Error())
		return roleName, sharedVPCRoute53RoleArn, err
	}
	log.Logger.Infof("Create a new route53 role for shared VPC: %s", sharedVPCRoute53RoleArn)
	err = rh.registerSharedRoute53Role(roleName)
	if err != nil {
		log.Logger.Errorf("Error happened when record shared VPC route53 role: %s", err.Error())
		return roleName, sharedVPCRoute53RoleArn, err
	}
	err = awsClient.AttachIAMPolicy(roleName, policyArn)
	if err != nil {
		log.Logger.Errorf("Error happens when attach shared route53 policy %s to role %s: %s", policyArn,
			sharedVPCRoute53RoleArn, err.Error())
	}
	return roleName, sharedVPCRoute53RoleArn, err
}

func (rh *resourcesHandler) PrepareSharedVPCEndPointRoleForHostedCP(
	vpcendpointRolePrefix string,
	installerRoleArn string,
	controlPlaneOperatorRoleArn string) (string, string, error) {

	awsClient, err := rh.GetAWSClient(true)
	if err != nil {
		return "", "", err
	}

	policyName := fmt.Sprintf("%s-%s", vpcendpointRolePrefix, "sharedvpc-vpc-endpoint-policy")
	sharedVpcEndPointPolictStatement := map[string]interface{}{
		"Sid":    "Statement1",
		"Effect": "Allow",
		"Action": []string{
			"ec2:CreateVpcEndpoint",
			"ec2:DescribeVpcEndpoints",
			"ec2:ModifyVpcEndpoint",
			"ec2:DeleteVpcEndpoints",
			"ec2:CreateTags",
			"ec2:CreateSecurityGroup",
			"ec2:AuthorizeSecurityGroupIngress",
			"ec2:AuthorizeSecurityGroupEgress",
			"ec2:DeleteSecurityGroup",
			"ec2:RevokeSecurityGroupIngress",
			"ec2:RevokeSecurityGroupEgress",
			"ec2:DescribeSecurityGroups",
			"ec2:DescribeVpcs",
			"route53:ListHostedZones",
			"route53:ChangeResourceRecordSets",
			"route53:ListResourceRecordSets",
		},
		"Resource": "*",
	}
	policyArn, err := awsClient.CreatePolicy(policyName, sharedVpcEndPointPolictStatement)
	if err != nil {
		log.Logger.Errorf("Error happens when prepare shared vpc-endpoint policy: %s", err.Error())
		return "", "", err
	}

	roleName := fmt.Sprintf("%s-%s", vpcendpointRolePrefix, "shared-vpcendpoint-role")
	if installerRoleArn == "" || controlPlaneOperatorRoleArn == "" {
		log.Logger.Errorf("Can not create shared vpc-endpoint role due to no installer role or controlplane operator role.")
		return "", "", err
	}
	log.Logger.Debugf("Got installer role arn: %s for shared vpc-endpoint role preparation", installerRoleArn)
	log.Logger.Debugf(
		"Got control plane operator role arn: %s for shared vpc-endpoint role preparation",
		controlPlaneOperatorRoleArn,
	)

	assumeRolesArns := []string{installerRoleArn, controlPlaneOperatorRoleArn}
	roleArn, err := awsClient.CreateRoleForSharedVPCHCP(roleName, assumeRolesArns)
	sharedVpcEndpointRoleArn := aws.ToString(roleArn.Arn)
	if err != nil {
		log.Logger.Errorf("Error happens when prepare shared vpc-endpoint role: %s", err.Error())
		return roleName, sharedVpcEndpointRoleArn, err
	}
	log.Logger.Infof("Create a new vpc-endpoint role for shared VPC: %s", sharedVpcEndpointRoleArn)
	err = rh.registerSharedVPCEndpointRole(roleName)
	if err != nil {
		log.Logger.Errorf("Error happened when record shared VPC vpc-endpoint role: %s", err.Error())
		return roleName, sharedVpcEndpointRoleArn, err
	}
	err = awsClient.AttachIAMPolicy(roleName, policyArn)
	if err != nil {
		log.Logger.Errorf("Error happens when attach shared vpc-endpoint policy %s to role %s: %s", policyArn,
			sharedVpcEndpointRoleArn, err.Error())
	}
	return roleName, sharedVpcEndpointRoleArn, err
}

func (rh *resourcesHandler) PrepareAdditionalPrincipalsRole(roleName string, installerRoleArn string) (string, error) {
	awsClient, err := rh.GetAWSClient(true)
	if err != nil {
		return "", err
	}
	policyArn := "arn:aws:iam::aws:policy/service-role/ROSAControlPlaneOperatorPolicy"
	if installerRoleArn == "" {
		log.Logger.Errorf("Can not create additional principal role due to no installer role.")
		return "", err
	}
	roleArn, err := awsClient.CreateRoleForAdditionalPrincipals(roleName, installerRoleArn)
	additionalPrincipalRoleArn := aws.ToString(roleArn.Arn)
	rh.registerAdditionalPrincipals(additionalPrincipalRoleArn)
	if err != nil {
		log.Logger.Errorf("Error happens when prepare additional principal role: %s", err.Error())
		return additionalPrincipalRoleArn, err
	}
	log.Logger.Infof("Create a new role for Additional Principal: %s", additionalPrincipalRoleArn)
	err = awsClient.AttachIAMPolicy(roleName, policyArn)
	if err != nil {
		log.Logger.Errorf(
			"Error happens when attach control plane operator policy %s to role %s: %s", policyArn,
			additionalPrincipalRoleArn, err.Error())
	}
	return additionalPrincipalRoleArn, err
}

func (rh *resourcesHandler) PrepareDNSDomain(hostedcp bool) (string, error) {
	var dnsDomain string
	var output bytes.Buffer
	var err error
	var dnsDomainStr string
	if hostedcp {
		output, err = rh.rosaClient.OCMResource.CreateDNSDomain("--hosted-cp")
	} else {
		output, err = rh.rosaClient.OCMResource.CreateDNSDomain()
	}
	if err != nil {
		return dnsDomain, err
	}
	parser := rosacli.NewParser()
	tip := parser.TextData.Input(output).Parse().Tip()
	for _, str := range strings.Split(tip, "\n") {
		if strings.Contains(str, "has been created") {
			dnsDomainStr = strings.Split(str, " ")[3]
			break
		}
	}
	if dnsDomainStr == "" {
		return dnsDomain, fmt.Errorf("failed to get dns domain from output: %s", tip)
	}
	dnsDomain = strings.TrimSuffix(strings.TrimPrefix(dnsDomainStr, "‘"), "’")
	err = rh.registerDNSDomain(dnsDomain)
	if err != nil {
		log.Logger.Errorf("Error happened when record DNS Domain: %s", err.Error())
	}
	return dnsDomain, err
}

func (rh *resourcesHandler) PrepareHostedZone(hostedZoneName string,
	vpcID string, private bool) (string, error) {

	awsClient, err := rh.GetAWSClient(true)
	if err != nil {
		return "", err
	}

	// hostedZoneName := fmt.Sprintf("%s.%s", clusterName, dnsDomain)
	callerReference := helper.GenerateRandomString(10)
	hostedZoneOutput, err := awsClient.CreateHostedZone(
		hostedZoneName,
		callerReference,
		vpcID,
		rh.resources.Region,
		private,
	)
	if err != nil {
		log.Logger.Errorf("Error happens when prepare hosted zone: %s", err.Error())
		return "", err
	}
	hostedZoneID := strings.Split(*hostedZoneOutput.HostedZone.Id, "/")[2]
	if strings.HasSuffix(hostedZoneName, "hypershift.local") {
		err = rh.registerIngressHostedZoneID(hostedZoneID)
		if err != nil {
			log.Logger.Errorf("Error happened when record Ingress Hosted Zone ID: %s", err.Error())
		}
	} else {
		err = rh.registerHostedCPInternalHostedZoneID(hostedZoneID)
		if err != nil {
			log.Logger.Errorf("Error happened when record Hosted Zone ID: %s", err.Error())
		}
	}

	return hostedZoneID, err
}

func (rh *resourcesHandler) PrepareSubnetArns(subnetIDs string) ([]string, error) {
	var subnetArns []string
	var resp []types.Subnet
	awsClient, err := rh.GetAWSClient(true)
	if err != nil {
		return nil, err
	}

	subnets := strings.Split(subnetIDs, ",")
	resp, err = awsClient.ListSubnetDetail(subnets...)
	if err != nil {
		return nil, err
	}

	for _, subnet := range resp {
		subnetArns = append(subnetArns, *subnet.SubnetArn)
	}
	return subnetArns, err
}
func (rh *resourcesHandler) PrepareSecurityGroupArns(sgIDs []string, useSharedVpcAcc bool) ([]string, error) {
	var (
		awsClient         *aws_client.AWSClient
		err               error
		awsAccountID      string
		securityGroupArns []string
	)
	if useSharedVpcAcc {
		awsClient, err = rh.GetAWSClient(true)
	} else {
		awsClient, err = rh.GetAWSClient(false)
	}
	if err != nil {
		log.Logger.Errorf("Error happens when prepareSecurityGroupArns: %s", err.Error())
		return securityGroupArns, err
	}
	awsAccountID = awsClient.AccountID
	for _, sgid := range sgIDs {
		securityGroupARN := fmt.Sprintf("arn:aws:ec2:%s:%s:security-group/%s",
			rh.vpc.Region,
			awsAccountID,
			sgid,
		)
		securityGroupArns = append(securityGroupArns, securityGroupARN)
	}
	return securityGroupArns, err
}

func (rh *resourcesHandler) PrepareResourceShare(resourceShareName string, resourceArns []string) (string, error) {
	var principles []string
	// Use directly aws client creation because we don't want the shared one here
	awsClient, err := rh.GetAWSClient(false)
	if err != nil {
		return "", err
	}
	principles = append(principles, awsClient.AccountID)

	sharedVPCAWSClient, err := rh.GetAWSClient(true)
	if err != nil {
		return "", err
	}

	sharedResourceOutput, err := sharedVPCAWSClient.CreateResourceShare(resourceShareName, resourceArns, principles)
	if err != nil {
		log.Logger.Errorf("Error happens when prepare resource share: %s", err.Error())
		return "", err
	}

	resourceShareArn := *sharedResourceOutput.ResourceShare.ResourceShareArn
	err = rh.registerResourceShareArn(resourceShareArn)
	if err != nil {
		log.Logger.Errorf("Error happened when record resource share: %s", err.Error())
	}
	return resourceShareArn, err
}

// generateAccountRoleCreationFlag will generate account role creation flags
func (rh *resourcesHandler) generateAccountRoleCreationFlag(
	namePrefix string,
	hcp bool,
	openshiftVersion string,
	channelGroup string,
	path string,
	permissionsBoundary string,
	route53RoleARN string,
	vpcEndpointRoleArn string) []string {
	flags := []string{
		"--prefix", namePrefix,
		"--mode", "auto",
		"-y",
	}
	if openshiftVersion != "" {
		majorVersion := helper.SplitMajorVersion(openshiftVersion)
		flags = append(flags, "--version", majorVersion)
	}
	if channelGroup != "" {
		flags = append(flags, "--channel-group", channelGroup)
	}
	if hcp {
		flags = append(flags, "--hosted-cp")
	} else {
		flags = append(flags, "--classic")
	}
	if path != "" {
		flags = append(flags, "--path", path)
	}
	if permissionsBoundary != "" {
		flags = append(flags, "--permissions-boundary", permissionsBoundary)
	}
	if route53RoleARN != "" {
		flags = append(flags, "--route53-role-arn", route53RoleARN)
	}
	if vpcEndpointRoleArn != "" {
		flags = append(flags, "--vpc-endpoint-role-arn", vpcEndpointRoleArn)
	}
	return flags
}

func (rh *resourcesHandler) PrepareS3ForLogForward(s3Name string, region string) (string, error) {
	awsClient, err := rh.GetAWSClient(false)
	if err != nil {
		return "", err
	}

	// Create an S3 client from the existing aws client config
	s3Client := s3.NewFromConfig(*awsClient.AWSConfig)

	_, err = s3Client.HeadBucket(context.TODO(), &s3.HeadBucketInput{
		Bucket: aws.String(s3Name),
	})
	if err == nil {
		return "", fmt.Errorf("bucket '%s' already exists", s3Name)
	}

	bucketInput := &s3.CreateBucketInput{
		Bucket: aws.String(s3Name),
	}
	if region != "" {
		bucketInput.CreateBucketConfiguration = &s3types.CreateBucketConfiguration{
			LocationConstraint: s3types.BucketLocationConstraint(region),
		}
	}

	_, err = s3Client.CreateBucket(context.TODO(), bucketInput)
	if err != nil {
		return "", err
	}
	err = rh.registerS3Bucket(s3Name)
	return s3Name, err
}
func (rh *resourcesHandler) PrepareCWLogGroup(groupName string, region string) (string, error) {
	awsClient, err := rh.GetAWSClient(false)
	if err != nil {
		return "", err
	}

	var cwClient *cloudwatchlogs.Client
	if awsClient.CloudWatchLogsClient != nil {
		cwClient = awsClient.CloudWatchLogsClient
	} else {
		cwClient = cloudwatchlogs.NewFromConfig(*awsClient.AWSConfig)
	}

	_, err = cwClient.CreateLogGroup(context.TODO(), &cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: aws.String(groupName),
	})
	if err != nil {
		var existsErr *cwtypes.ResourceAlreadyExistsException
		if !errors.As(err, &existsErr) {
			return "", err
		}
	}

	// Set retention to 1 day
	_, err = cwClient.PutRetentionPolicy(context.TODO(), &cloudwatchlogs.PutRetentionPolicyInput{
		LogGroupName:    aws.String(groupName),
		RetentionInDays: aws.Int32(1),
	})
	if err != nil {
		return "", err
	}
	err = rh.registerCWLogGroup(groupName)
	return groupName, err
}
func (rh *resourcesHandler) PrepareLogForwardRole(oidcProviderURL string, rolePrefix string) (string, error) {
	// Create AWS client (uses default region/profile)
	awsClient, err := rh.GetAWSClient(false)

	if err != nil {
		return "", err
	}

	// Generate role name with 4 random chars suffix
	roleName := fmt.Sprintf("%s-%s", rolePrefix, helper.GenerateRandomString(4))

	// Create policy for log forwarding (uses same statement as required)
	policyName := fmt.Sprintf("%s-policy", roleName)
	policyArn, err := awsClient.CreatePolicyForAuditLogForward(policyName)
	if err != nil {
		return "", err
	}

	// Create role with OIDC trust relationship and attach the policy
	role, err := awsClient.CreateRoleForAuditLogForward(roleName, awsClient.AccountID, oidcProviderURL, policyArn)
	if err != nil {
		return "", err
	}
	roleArn := aws.ToString(role.Arn)
	err = rh.registerLogForwardRole(roleArn)
	if err != nil {
		log.Logger.Errorf("Error happened when record resource share: %s", err.Error())
	}
	return roleArn, nil
}
