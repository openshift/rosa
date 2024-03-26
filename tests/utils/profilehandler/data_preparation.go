package profilehandler

import (
	"bytes"
	"fmt"

	"github.com/openshift/rosa/tests/utils/common"
	CON "github.com/openshift/rosa/tests/utils/common/constants"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

func PrepareVersion(client *rosacli.Client, versionRequirement string, channelGroup string, hcp bool) (
	*rosacli.OpenShiftVersionTableOutput, error) {
	versionList, err := client.Version.ListAndReflectVersions(channelGroup, hcp)
	if err != nil {
		return nil, err
	}

	if CON.VersionLatestPattern.MatchString(versionRequirement) {
		return versionList.Latest()
	} else if CON.VersionMajorMinorPattern.MatchString(versionRequirement) {
		version, err := versionList.FindNearestBackwardMinorVersion(versionRequirement, 0, true)
		return version, err
	} else if CON.VersionRawPattern.MatchString(versionRequirement) {
		return &rosacli.OpenShiftVersionTableOutput{
			Version: versionRequirement,
		}, nil
	} else if CON.VersionFlexyPattern.MatchString(versionRequirement) {
		return nil, nil // Implement later
	}
	return nil, fmt.Errorf("not supported version requirement: %s", versionRequirement)
}

func PrepareAutoscalerDummy() string {
	return ""
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
	flags := []string{
		"--prefix", namePrefix,
		"--mode", "auto",
		"-y",
	}
	if openshiftVersion != "" {
		majorVersion := common.SplitMajorVersion(openshiftVersion)
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
	_, err = client.OCMResource.CreateAccountRole(
		flags...,
	)
	if err != nil {
		return
	}
	accRoleList, _, err := client.OCMResource.ListAccountRole()
	if err != nil {
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

func PrepareAdminUserDummy() (string, string) {
	return "", ""
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
