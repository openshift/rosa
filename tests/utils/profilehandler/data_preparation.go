package profilehandler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/openshift-online/ocm-common/pkg/aws/aws_client"
	"github.com/openshift-online/ocm-common/pkg/test/kms_key"
	"github.com/openshift-online/ocm-common/pkg/test/vpc_client"

	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/tests/ci/config"
	"github.com/openshift/rosa/tests/utils/common"
	con "github.com/openshift/rosa/tests/utils/common/constants"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/log"
)

func RecordUserDataInfo(filePath string, key string, value string) error {
	userData, _ := ParseUserData()

	if userData == nil {
		userData = &UserData{}
	}
	valueOfUserData := reflect.ValueOf(userData).Elem()
	valueOfUserData.FieldByName(key).SetString(value)
	_, err := common.CreateFileWithContent(filePath, userData)
	return err

}
func PrepareVersion(client *rosacli.Client, versionRequirement string, channelGroup string, hcp bool) (
	*rosacli.OpenShiftVersionTableOutput, error) {
	log.Logger.Infof("Got version requirement %s going to prepare accordingly", versionRequirement)
	log.Logger.Infof("Channel group = %s", channelGroup)
	versionList, err := client.Version.ListAndReflectVersions(channelGroup, hcp)
	if err != nil {
		return nil, err
	}

	if con.VersionLatestPattern.MatchString(versionRequirement) {
		return versionList.Latest()
	} else if con.VersionMajorMinorPattern.MatchString(versionRequirement) {
		version, err := versionList.FindNearestBackwardMinorVersion(versionRequirement, 0, true)
		return version, err
	} else if con.VersionRawPattern.MatchString(versionRequirement) {
		return &rosacli.OpenShiftVersionTableOutput{
			Version: versionRequirement,
		}, nil
	} else if con.VersionFlexyPattern.MatchString(versionRequirement) {
		log.Logger.Debugf("Version requirement matched %s", con.VersionFlexyPattern.String())
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
// if longname is set, it will generate the long name with con.DefaultLongClusterNamelength
func PreparePrefix(profilePrefix string, nameLength int) string {
	if nameLength > ocm.MaxClusterNameLength {
		panic(fmt.Errorf("name length %d is longer than allowed max name length %d", nameLength, ocm.MaxClusterNameLength))
	}

	if len(profilePrefix) > nameLength {
		newProfilePrefix := common.TrimNameByLength(profilePrefix, nameLength-4)
		log.Logger.Warnf("Profile name prefix %s is longer than "+
			"the nameLength for random generated. Trimed it to %s", profilePrefix, newProfilePrefix)
		profilePrefix = newProfilePrefix
	}
	return strings.TrimSuffix(common.GenerateRandomName(profilePrefix, nameLength-len(profilePrefix)-1), "-")
}

// PrepareVPC will prepare a single vpc
func PrepareVPC(region string, vpcName string, cidrValue string,
	awsSharedCredentialFile string) (*vpc_client.VPC, error) {
	log.Logger.Info("Starting vpc preparation")
	vpc, err := vpc_client.PrepareVPC(vpcName, region, cidrValue, false, awsSharedCredentialFile)
	if err != nil {
		return vpc, err
	}
	err = RecordUserDataInfo(config.Test.UserDataFile, "VpcID", vpc.VpcID)
	log.Logger.Info("VPC preparation finished")
	return vpc, err

}

// PrepareSubnets will prepare pair of subnets according to the vpcID and zones
// if zones are empty list it will list the zones and pick according to multi-zone parameter.
// when multi-zone=true, 3 zones will be pickup
func PrepareSubnets(vpcClient *vpc_client.VPC, region string,
	zones []string, multiZone bool) (map[string][]string, error) {
	resultMap := map[string][]string{}
	if len(zones) == 0 {
		log.Logger.Info("Got no zones indicated. List the zones and pick from the listed zones")
		resultZones, err := vpcClient.AWSClient.ListAvaliableZonesForRegion(region, "availability-zone")
		if err != nil {
			return resultMap, err
		}
		zones = resultZones[0:1]
		if multiZone {
			zones = resultZones[0:3]
		}
	}
	for _, zone := range zones {
		subnetMap, err := vpcClient.PreparePairSubnetByZone(zone)
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

	return resultMap, nil
}

func PrepareProxy(vpcClient *vpc_client.VPC,
	zone string,
	sshPemFileName string,
	sshPemFileRecordDir string,
	caFile string) (*ProxyDetail, error) {

	_, privateIP, caContent, err := vpcClient.LaunchProxyInstance(zone, sshPemFileName, sshPemFileRecordDir)
	if err != nil {
		return nil, err
	}
	_, err = common.CreateFileWithContent(caFile, caContent)
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

func PrepareKMSKey(region string, multiRegion bool, testClient string, hcp bool, etcdKMS bool) (string, error) {
	keyArn, err := kms_key.CreateOCMTestKMSKey(region, multiRegion, testClient)
	if err != nil {
		return keyArn, err
	}
	userDataKey := "KMSKey"
	if etcdKMS {
		userDataKey = "EtcdKMSKey"
	}
	err = RecordUserDataInfo(config.Test.UserDataFile, userDataKey, keyArn)
	if err != nil {
		return keyArn, err
	}
	if hcp {
		kms_key.AddTagToKMS(keyArn, region, map[string]string{
			"red-hat": "true",
		})
	}
	return keyArn, err
}

func ElaborateKMSKeyForSTSCluster(client *rosacli.Client, cluster string, etcdKMS bool) error {
	jsonData, err := client.Cluster.GetJSONClusterDescription(cluster)
	if err != nil {
		return err
	}
	accountRoles := []string{
		jsonData.DigString("aws", "sts", "role_arn"),
	}
	operaorRoleMap := map[string]string{}
	keyArn := jsonData.DigString("aws", "kms_key_arn")
	if etcdKMS {
		keyArn = jsonData.DigString("aws", "etcd_encryption", "kms_key_arn")
	}
	operatorRoles := jsonData.DigObject("aws", "sts", "operator_iam_roles").([]interface{})
	for _, operatorRole := range operatorRoles {
		role := operatorRole.(map[string]interface{})
		operaorRoleMap[role["name"].(string)] = role["role_arn"].(string)
	}
	region := jsonData.DigString("region", "id")
	isHCP := jsonData.DigBool("hypershift", "enabled")
	err = kms_key.ConfigKMSKeyPolicyForSTS(keyArn, region, isHCP, accountRoles, operaorRoleMap)
	if err != nil {
		log.Logger.Errorf("Elaborate the KMS key %s for cluster %s failed: %s", keyArn, cluster, err.Error())
	} else {
		log.Logger.Infof("Elaborate the KMS key %s for cluster %s successfully", keyArn, cluster)
	}

	return err
}

func PrepareAdditionalSecurityGroups(vpcClient *vpc_client.VPC,
	securityGroupCount int,
	namePrefix string) ([]string, error) {

	return vpcClient.CreateAdditionalSecurityGroups(securityGroupCount, namePrefix, "")
}

// PrepareAccountRoles will prepare account roles according to the parameters
// openshiftVersion must follow 4.15.2-x format
func PrepareAccountRoles(client *rosacli.Client,
	namePrefix string,
	hcp bool,
	openshiftVersion string,
	channelGroup string,
	path string,
	permissionsBoundary string) (
	accRoles *rosacli.AccountRolesUnit, err error) {

	flags := GenerateAccountRoleCreationFlag(client,
		namePrefix,
		hcp,
		openshiftVersion,
		channelGroup,
		path,
		permissionsBoundary,
	)

	output, err := client.OCMResource.CreateAccountRole(
		flags...,
	)
	if err != nil {
		err = fmt.Errorf("error happens when create account-roles, %s", output.String())
		return
	}
	err = RecordUserDataInfo(config.Test.UserDataFile, "AccountRolesPrefix", namePrefix)
	if err != nil {
		return
	}
	var accoutRoles *rosacli.AccountRolesUnit
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		accRoleList, output, err := client.OCMResource.ListAccountRole()
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
func PrepareOperatorRolesByOIDCConfig(client *rosacli.Client,
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
	_, err := client.OCMResource.CreateOperatorRoles(
		flags...,
	)
	if err != nil {
		return err
	}
	err = RecordUserDataInfo(config.Test.UserDataFile, "OperatorRolesPrefix", namePrefix)
	return err
}

func PrepareAdminUser() (string, string) {
	userName := common.GenerateRandomString(10)
	password := common.GenerateRandomStringWithSymbols(14)
	return userName, password
}

func PrepareAuditlogRoleArnByOIDCConfig(client *rosacli.Client,
	auditLogRoleName string,
	oidcConfigID string,
	region string) (string, error) {

	oidcConfig, err := client.OCMResource.GetOIDCConfigFromList(oidcConfigID)
	if err != nil {
		return "", err
	}
	logRoleArn, err := PrepareAuditlogRoleArnByIssuer(auditLogRoleName, oidcConfig.IssuerUrl, region)
	if err != nil {
		return logRoleArn, err
	}
	err = RecordUserDataInfo(config.Test.UserDataFile, "AuditLogArn", logRoleArn)
	return logRoleArn, err

}

func PrepareAuditlogRoleArnByIssuer(auditLogRoleName string, oidcIssuerURL string, region string) (string, error) {
	//nolint:staticcheck,all
	oidcIssuerURL = strings.TrimLeft(oidcIssuerURL, "https://")
	log.Logger.Infof("Preparing audit log role with name %s and oidcIssuerURL %s", auditLogRoleName, oidcIssuerURL)
	awsClient, err := aws_client.CreateAWSClient("", region)
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
	err = RecordUserDataInfo(config.Test.UserDataFile, "AuditLogArn", auditLogRoleArn)
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

func PrepareOperatorRolesByCluster(client *rosacli.Client, cluster string) error {
	flags := []string{
		"-y",
		"--mode", "auto",
		"--cluster", cluster,
	}
	_, err := client.OCMResource.CreateOperatorRoles(
		flags...,
	)
	return err
}

// PrepareOIDCConfig will prepare the oidc config for the cluster,
// if the oidcConfigType="managed", roleArn and prefix won't be set
func PrepareOIDCConfig(client *rosacli.Client,
	oidcConfigType string,
	region string,
	roleArn string,
	prefix string) (string, error) {

	var oidcConfigID string
	var output bytes.Buffer
	var err error
	switch oidcConfigType {
	case "managed":
		output, err = client.OCMResource.CreateOIDCConfig(
			"-o", "json",
			"--mode", "auto",
			"--region", region,
			"--managed",
			"-y",
		)
	case "unmanaged":
		output, err = client.OCMResource.CreateOIDCConfig(
			"-o", "json",
			"--mode", "auto",
			"--prefix", prefix,
			"--region", region,
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
	err = RecordUserDataInfo(config.Test.UserDataFile, "OIDCConfigID", oidcConfigID)
	return oidcConfigID, err
}

func PrepareOIDCProvider(client *rosacli.Client, oidcConfigID string) error {
	_, err := client.OCMResource.CreateOIDCProvider(
		"--mode", "auto",
		"-y",
		"--oidc-config-id", oidcConfigID,
	)
	return err
}
func PrepareOIDCProviderByCluster(client *rosacli.Client, cluster string) error {
	_, err := client.OCMResource.CreateOIDCProvider(
		"--mode", "auto",
		"-y",
		"--cluster", cluster,
	)
	return err
}

func PrepareTemporaryPolicyFor417(region string, capaControllerOperatorRoleName string) (string, error) {
	awsClient, err := aws_client.CreateAWSClient("", region)
	if err != nil {
		return "", err
	}
	policyName := common.GenerateRandomName("ROSANodePoolMissingPolicy", 2)
	policyDocument := map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{
				"Sid":    "CreateTags",
				"Effect": "Allow",
				"Action": "ec2:CreateTags",
				"Resource": []string{
					"arn:aws:ec2:*:*:instance/*",
					"arn:aws:ec2:*:*:volume/*",
					"arn:aws:ec2:*:*:network-interface/*"},
				"Condition": map[string]interface{}{
					"StringEquals": map[string]string{
						"ec2:CreateAction": "RunInstances",
					},
				},
			},
			{
				"Sid":    "CreateTagsCAPAControllerNetworkInterface",
				"Effect": "Allow",
				"Action": "ec2:CreateTags",
				"Resource": []string{
					"arn:aws:ec2:*:*:network-interface/*"},
				"Condition": map[string]interface{}{
					"StringEquals": map[string]string{
						"aws:RequestTag/red-hat-managed": "true",
					},
				},
			},
		},
	}
	policyBytes, _ := json.Marshal(policyDocument)
	policy, err := awsClient.CreateIAMPolicy(policyName, string(policyBytes),
		map[string]string{"openshift_version": "4.17"})
	if err != nil {
		return "", err
	}
	err = awsClient.AttachIAMPolicy(capaControllerOperatorRoleName, *policy.Arn)
	if err != nil {
		log.Logger.Errorf("Error happens when attach misssing policy %s to role %s: %s", *policy.Arn,
			capaControllerOperatorRoleName, err.Error())
	}
	log.Logger.Infof("Policy %s is attached to %s", *policy.Arn, capaControllerOperatorRoleName)
	return *policy.Arn, err
}

func PrepareSharedVPCRole(sharedVPCRolePrefix string, installerRoleArn string, ingressOperatorRoleArn string,
	region string, awsSharedCredentialFile string) (string, string, error) {
	awsClient, err := aws_client.CreateAWSClient("", region, awsSharedCredentialFile)
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
	err = RecordUserDataInfo(config.Test.UserDataFile, "SharedVPCRole", roleName)
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

func PrepareAdditionalPrincipalsRole(roleName string, installerRoleArn string,
	region string, awsSharedCredentialFile string) (string, error) {
	awsClient, err := aws_client.CreateAWSClient("", region, awsSharedCredentialFile)
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

func PrepareDNSDomain(client *rosacli.Client) (string, error) {
	var dnsDomain string
	var output bytes.Buffer
	var err error

	output, err = client.OCMResource.CreateDNSDomain()
	if err != nil {
		return dnsDomain, err
	}
	parser := rosacli.NewParser()
	tip := parser.TextData.Input(output).Parse().Tip()
	dnsDomainStr := strings.Split(strings.Split(tip, "\n")[0], " ")[3]
	dnsDomain = strings.TrimSuffix(strings.TrimPrefix(dnsDomainStr, "‘"), "’")
	err = RecordUserDataInfo(config.Test.UserDataFile, "DNSDomain", dnsDomain)
	if err != nil {
		log.Logger.Errorf("Error happened when record DNS Domain: %s", err.Error())
	}
	return dnsDomain, err
}

func PrepareHostedZone(clusterName string, dnsDomain string, vpcID string, region string, private bool,
	awsSharedCredentialFile string) (string, error) {
	awsClient, err := aws_client.CreateAWSClient("", region, awsSharedCredentialFile)
	if err != nil {
		return "", err
	}

	hostedZoneName := fmt.Sprintf("%s.%s", clusterName, dnsDomain)
	callerReference := common.GenerateRandomString(10)
	hostedZoneOutput, err := awsClient.CreateHostedZone(hostedZoneName, callerReference, vpcID, region, private)
	if err != nil {
		log.Logger.Errorf("Error happens when prepare hosted zone: %s", err.Error())
		return "", err
	}
	hostedZoneID := strings.Split(*hostedZoneOutput.HostedZone.Id, "/")[2]

	err = RecordUserDataInfo(config.Test.UserDataFile, "HostedZoneID", hostedZoneID)
	if err != nil {
		log.Logger.Errorf("Error happened when record Hosted Zone ID: %s", err.Error())
	}
	return hostedZoneID, err
}

func PrepareSubnetArns(subnetIDs string, region string, awsSharedCredentialFile string) ([]string, error) {
	var subnetArns []string
	var resp []types.Subnet
	awsClient, err := aws_client.CreateAWSClient("", region, awsSharedCredentialFile)
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

func PrepareResourceShare(resourceShareName string, resourceArns []string, region string,
	awsSharedCredentialFile string) (string, error) {
	var principles []string
	awsClient, err := aws_client.CreateAWSClient("", region)
	if err != nil {
		return "", err
	}
	principles = append(principles, awsClient.AccountID)

	sharedVPCAWSClient, err := aws_client.CreateAWSClient("", region, awsSharedCredentialFile)
	if err != nil {
		return "", err
	}

	sharedResourceOutput, err := sharedVPCAWSClient.CreateResourceShare(resourceShareName, resourceArns, principles)
	if err != nil {
		log.Logger.Errorf("Error happens when prepare resource share: %s", err.Error())
		return "", err
	}
	resourceShareArn := *sharedResourceOutput.ResourceShare.ResourceShareArn
	err = RecordUserDataInfo(config.Test.UserDataFile, "ResourceShareArn", resourceShareArn)
	if err != nil {
		log.Logger.Errorf("Error happened when record resource share: %s", err.Error())
	}
	return resourceShareArn, err
}
