package profilehandler

import (
	"bytes"
	"fmt"

	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/tests/utils/common"
	con "github.com/openshift/rosa/tests/utils/common/constants"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

func PrepareVersion(client *rosacli.Client, versionRequirement string, channelGroup string, hcp bool) (
	*rosacli.OpenShiftVersionTableOutput, error) {
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
		return nil, nil // TODO
	}
	return nil, fmt.Errorf("not supported version requirement: %s", versionRequirement)
}

// PrepareNames will generate the name for cluster creation
// if longname is set, it will generate the long name with con.DefaultLongClusterNamelength
func PreparePrefix(profilePrefix string, nameLength int) string {
	if nameLength > ocm.MaxClusterNameLength {
		panic(fmt.Errorf("name length %d is longer than allowed max name length %d", nameLength, ocm.MaxClusterNameLength))
	}
	return common.GenerateRandomName(profilePrefix, nameLength-len(profilePrefix)-1)
}

func PrepareSubnetsDummy(vpcID string, region string, zones string) map[string][]string {
	return map[string][]string{}
}

func PrepareProxysDummy(vpcID string, region string, zones string) map[string]string {
	return map[string]string{
		"https_proxy":  "",
		"http_proxy":   "",
		"no_proxy":     "",
		"ca_file_path": "",
	}
}

func PrepareKMSKeyDummy(region string) string {
	return ""
}

func PrepareSecurityGroupsDummy(vpcID string, region string, securityGroupCount int) []string {
	return []string{}
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
	hcp bool) error {
	flags := []string{
		"-y",
		"--mode", "auto",
		"--prefix", namePrefix,
		"--role-arn", roleArn,
		"--oidc-config-id", oidcConfigID,
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
	return err
}

func PrepareAdminUser() (string, string) {
	userName := common.GenerateRandomString(10)
	password := common.GenerateRandomStringWithSymbols(14)
	return userName, password
}

func PrepareAuditlogDummy() string {
	return ""
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

// PrepareOIDCConfig will prepare the oidc config for the cluster, if the oidcConfigType="managed", roleArn and prefix won't be set
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
	return oidcConfigID, nil
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

func PrepareExternalAuthConfigDummy() {}
