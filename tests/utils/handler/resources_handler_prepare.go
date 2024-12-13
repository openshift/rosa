package handler

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/openshift-online/ocm-common/pkg/test/kms_key"
	"github.com/openshift-online/ocm-common/pkg/test/vpc_client"

	"github.com/openshift/rosa/pkg/ocm"
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
func (rh *resourcesHandler) PrepareVPC(vpcName string, cidrValue string, useExisting bool) (*vpc_client.VPC, error) {
	log.Logger.Info("Starting vpc preparation")
	vpc, err := vpc_client.PrepareVPC(vpcName, rh.resources.Region, cidrValue, useExisting, rh.awsSharedCredentialsFile)
	if err != nil {
		return nil, err
	}
	rh.vpc = vpc
	err = rh.registerVpcID(vpc.VpcID)
	log.Logger.Info("VPC preparation finished")
	if err != nil {
		return vpc, err
	}
	err = rh.registerVPC(vpc)
	return vpc, err
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

	if rh.vpc == nil {
		return nil, errors.New("VPC has not been initialized ...")
	}

	_, privateIP, caContent, err := rh.vpc.LaunchProxyInstance(zone, sshPemFileName, sshPemFileRecordDir)
	if err != nil {
		return nil, err
	}
	_, err = helper.CreateFileWithContent(caFile, caContent)
	if err != nil {
		return nil, err
	}
	return &ProxyDetail{
		HTTPsProxy:       fmt.Sprintf("https://%s:8080", privateIP),
		HTTPProxy:        fmt.Sprintf("http://%s:8080", privateIP),
		CABundleFilePath: caFile,
		NoProxy:          "quay.io",
	}, nil
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
		return nil, errors.New("VPC has not been initialized ...")
	}

	return rh.vpc.CreateAdditionalSecurityGroups(securityGroupCount, namePrefix, "")
}

// PrepareAccountRoles will prepare account roles according to the parameters
// openshiftVersion must follow 4.15.2-x format
func (rh *resourcesHandler) PrepareAccountRoles(
	namePrefix string,
	hcp bool,
	openshiftVersion string,
	channelGroup string,
	path string,
	permissionsBoundary string) (
	accRoles *rosacli.AccountRolesUnit, err error) {

	flags := rh.generateAccountRoleCreationFlag(
		namePrefix,
		hcp,
		openshiftVersion,
		channelGroup,
		path,
		permissionsBoundary,
	)

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
	sharedVPCRoleArn string,
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
	if sharedVPCRoleArn != "" {
		flags = append(flags, "--shared-vpc-role-arn", sharedVPCRoleArn)
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

func (rh *resourcesHandler) PrepareDNSDomain() (string, error) {
	var dnsDomain string
	var output bytes.Buffer
	var err error

	output, err = rh.rosaClient.OCMResource.CreateDNSDomain()
	if err != nil {
		return dnsDomain, err
	}
	parser := rosacli.NewParser()
	tip := parser.TextData.Input(output).Parse().Tip()
	dnsDomainStr := strings.Split(strings.Split(tip, "\n")[0], " ")[3]
	dnsDomain = strings.TrimSuffix(strings.TrimPrefix(dnsDomainStr, "‘"), "’")
	err = rh.registerDNSDomain(dnsDomain)
	if err != nil {
		log.Logger.Errorf("Error happened when record DNS Domain: %s", err.Error())
	}
	return dnsDomain, err
}

func (rh *resourcesHandler) PrepareHostedZone(clusterName string, dnsDomain string,
	vpcID string, private bool) (string, error) {

	awsClient, err := rh.GetAWSClient(true)
	if err != nil {
		return "", err
	}

	hostedZoneName := fmt.Sprintf("%s.%s", clusterName, dnsDomain)
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

	err = rh.registerHostedZoneID(hostedZoneID)
	if err != nil {
		log.Logger.Errorf("Error happened when record Hosted Zone ID: %s", err.Error())
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
	permissionsBoundary string) []string {
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
	return flags
}
