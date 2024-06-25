package profilehandler

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
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
			version, err := versionList.FindNearestBackwardMinorVersion(latestVersion.Version, int64(versionStep), true, true)
			return version, err
		case "z":
			version, err := versionList.FindNearestBackwardOptionalVersion(latestVersion.Version, versionStep, true)
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
	return common.GenerateRandomName(profilePrefix, nameLength-len(profilePrefix)-1)
}

// PrepareVPC will prepare a single vpc
func PrepareVPC(region string, vpcName string, cidrValue string) (*vpc_client.VPC, error) {
	log.Logger.Info("Starting vpc preparation")
	vpc, err := vpc_client.PrepareVPC(vpcName, region, cidrValue, false)
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
	accRoleList, output, err := client.OCMResource.ListAccountRole()
	if err != nil {
		err = fmt.Errorf("error happens when list account-roles, %s", output.String())
		return
	}
	roleDig := accRoleList.DigAccountRoles(namePrefix, hcp)

	return roleDig, nil

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
	roleArn, err := awsClient.CreateRoleForAuditLogForward(auditLogRoleName, awsClient.AccountID, oidcIssuerURL)
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
