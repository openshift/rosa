package aws

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/fedramp"
	"github.com/openshift/rosa/pkg/helper"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var RoleNameRE = regexp.MustCompile(`^[\w+=,.@-]+$`)

// UserTagKeyRE , UserTagValueRE - https://docs.aws.amazon.com/general/latest/gr/aws_tagging.html#tag-conventions
var UserTagKeyRE = regexp.MustCompile(`^[\pL\pZ\pN_.:/=+\-@]{1,128}$`)
var UserTagValueRE = regexp.MustCompile(`^[\pL\pZ\pN_.:/=+\-@]{0,256}$`)

// the following regex defines five different patterns:
// first pattern is to validate IPv4 address
// second,is for IPv4 CIDR range validation
// third pattern is to validate domains
// and the fifth petterrn is to be able to remove the existing no-proxy value by typing empty string ("").
// nolint
var UserNoProxyRE = regexp.MustCompile(`^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$|^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])(\/(3[0-2]|[1-2][0-9]|[0-9]))$|^(.?[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9][a-z0-9-]{0,61}[a-z0-9]$|^""$`)

func GetJumpAccount(env string) string {
	jumpAccounts := JumpAccounts
	if fedramp.Enabled() {
		jumpAccounts = fedramp.JumpAccounts
	}
	return jumpAccounts[env]
}

// JumpAccounts are the various of AWS accounts used for the installer jump role in the various OCM environments
var JumpAccounts = map[string]string{
	"production":  "710019948333",
	"staging":     "644306948063",
	"integration": "896164604406",
}

func ARNValidator(input interface{}) error {
	if str, ok := input.(string); ok {
		if str == "" {
			return nil
		}
		_, err := arn.Parse(str)
		if err != nil {
			return fmt.Errorf("Invalid ARN: %s", err)
		}
		return nil
	}
	return fmt.Errorf("can only validate strings, got %v", input)
}

// GetRegion will return a region selected by the user or given as a default to the AWS client.
// If the region given is empty, it will first attempt to use the default, and, failing that, will
// prompt for user input.
func GetRegion(region string) (string, error) {
	if region == "" {
		defaultSession, err := session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		})

		if err != nil {
			return "", fmt.Errorf("Error creating default session for AWS client: %v", err)
		}

		region = *defaultSession.Config.Region
	}
	return region, nil
}

// getClientDetails will return the *iam.User associated with the provided client's credentials,
// a boolean indicating whether the user is the 'root' account, and any error encountered
// while trying to gather the info.
func getClientDetails(awsClient *awsClient) (*sts.GetCallerIdentityOutput, bool, error) {
	rootUser := false

	_, err := awsClient.ValidateCredentials()
	if err != nil {
		return nil, rootUser, err
	}

	user, err := awsClient.stsClient.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, rootUser, err
	}

	// Detect whether the AWS account's root user is being used
	parsed, err := arn.Parse(*user.Arn)
	if err != nil {
		return nil, rootUser, err
	}
	if parsed.AccountID == *user.UserId {
		rootUser = true
	}

	return user, rootUser, nil
}

/**
Currently user can rosa init using the region from their config or using --region
When checking for cloud formation we need to check in the region used by the user
*/
func GetAWSClientForUserRegion(reporter *rprtr.Object, logger *logrus.Logger) Client {
	// Get AWS region from env
	awsRegionInUserConfig, err := GetRegion(arguments.GetRegion())
	if err != nil {
		reporter.Errorf("Error getting region: %v", err)
		os.Exit(1)
	}
	if awsRegionInUserConfig == "" {
		reporter.Errorf("AWS Region not set")
		os.Exit(1)
	}

	// Create the AWS client:
	client, err := NewClient().
		Logger(logger).
		Region(awsRegionInUserConfig).
		Build()
	if err != nil {
		reporter.Errorf("Error creating aws client for stack validation: %v", err)
		os.Exit(1)
	}
	regionUsedForInit, err := client.GetClusterRegionTagForUser(AdminUserName)
	if err != nil || regionUsedForInit == "" {
		return client
	}

	if regionUsedForInit != awsRegionInUserConfig {
		// Create the AWS client with the region used in the init
		//So we can check for the stack in that region
		awsClient, err := NewClient().
			Logger(logger).
			Region(regionUsedForInit).
			Build()
		if err != nil {
			reporter.Errorf("Error creating aws client for stack validation: %v", err)
			os.Exit(1)
		}
		return awsClient
	}
	return client
}

func isSTS(ARN arn.ARN) bool {
	// If the client is using STS credentials we'll attempt to find the role
	// assumed by the user and validate that using PolicySimulator
	resource := strings.Split(ARN.Resource, "/")
	resourceType := 0
	// Example STS role ARN "arn:aws:sts::123456789123:assumed-role/OrganizationAccountAccessRole/UserAccess"
	// if the "service" is STS and the "resource-id" sectino of the ARN contains 3 sections delimited by
	// "/" we can validate its an assumed-role and assume the role name is the "parent-resource" and construct
	// a role ARN
	// https://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html
	if ARN.Service == "sts" &&
		resource[resourceType] == "assumed-role" {
		return true
	}
	return false
}

func resolveSTSRole(ARN arn.ARN) (*string, error) {
	// If the client is using STS credentials we'll attempt to find the role
	// assumed by the user and validate that using PolicySimulator
	resource := strings.Split(ARN.Resource, "/")
	parentResource := 1
	// Example STS role ARN "arn:aws:sts::123456789123:assumed-role/OrganizationAccountAccessRole/UserAccess"
	// if the "service" is STS and the "resource-id" sectino of the ARN contains 3 sections delimited by
	// "/" we can validate its an assumed-role and assume the role name is the "parent-resource" and construct
	// a role ARN
	// https://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html
	if isSTS(ARN) && len(resource) == 3 {
		// Construct IAM role ARN
		roleARNString := fmt.Sprintf(
			"arn:%s:iam::%s:role/%s", ARN.Partition, ARN.AccountID, resource[parentResource])
		// Parse it to validate its ok
		_, err := arn.Parse(roleARNString)
		if err != nil {
			return nil, fmt.Errorf("Unable to parse role ARN %s created from sts role: %v", roleARNString, err)
		}
		return &roleARNString, nil
	}

	return nil, fmt.Errorf("ARN %s doesn't appear to have a a resource-id that confirms to an STS user", ARN.String())
}

func UserTagValidator(input interface{}) error {
	if str, ok := input.(string); ok {
		if str == "" {
			return nil
		}
		tags := strings.Split(str, ",")
		for _, t := range tags {
			if !strings.Contains(t, ":") {
				return fmt.Errorf("invalid tag format, Tags are comma separated, for example: --tags=foo:bar,bar:baz")
			}
			tag := strings.Split(t, ":")
			if len(tag) != 2 {
				return fmt.Errorf("invalid tag format. Expected tag format: --tags=key:value")
			}
			if !UserTagKeyRE.MatchString(tag[0]) {
				return fmt.Errorf("expected a valid user tag key '%s' matching %s", tag[0], UserTagKeyRE.String())
			}
			if !UserTagValueRE.MatchString(tag[1]) {
				return fmt.Errorf("expected a valid user tag value '%s' matching %s", tag[1], UserTagValueRE.String())
			}
		}
		return nil
	}
	return fmt.Errorf("can only validate strings, got %v", input)
}

func UserTagDuplicateValidator(input interface{}) error {
	if str, ok := input.(string); ok {
		if str == "" {
			return nil
		}
		tags := strings.Split(str, ",")
		duplicate, found := HasDuplicateTagKey(tags)
		if found {
			return fmt.Errorf("user tag keys must be unique, duplicate key '%s' found", duplicate)
		}
		return nil
	}
	return fmt.Errorf("can only validate strings, got %v", input)
}

func HasDuplicateTagKey(tags []string) (string, bool) {
	visited := make(map[string]bool)
	for _, t := range tags {
		tag := strings.Split(t, ":")
		if visited[tag[0]] {
			return tag[0], true
		}
		visited[tag[0]] = true
	}
	return "", false
}

func UserNoProxyValidator(input interface{}) error {
	if str, ok := input.(string); ok {
		if str == "" {
			return nil
		}
		noProxyValues := strings.Split(str, ",")

		for _, v := range noProxyValues {
			if !UserNoProxyRE.MatchString(v) {
				return fmt.Errorf("expected a valid user no-proxy value: '%s' matching %s", v, UserNoProxyRE.String())
			}
		}
		return nil
	}
	return fmt.Errorf("can only validate strings, got %v", input)
}

func UserNoProxyDuplicateValidator(input interface{}) error {
	if str, ok := input.(string); ok {
		if str == "" {
			return nil
		}
		values := strings.Split(str, ",")
		duplicate, found := HasDuplicates(values)
		if found {
			return fmt.Errorf("no-proxy values must be unique, duplicate key '%s' found", duplicate)
		}
		return nil
	}
	return fmt.Errorf("can only validate strings, got %v", input)
}

func HasDuplicates(valSlice []string) (string, bool) {
	visited := make(map[string]bool)
	for _, v := range valSlice {
		if visited[v] {
			return v, true
		}
		visited[v] = true
	}
	return "", false
}

func GetTagValues(tagsValue []*iam.Tag) (roleType string, version string) {
	for _, tag := range tagsValue {
		switch aws.StringValue(tag.Key) {
		case tags.RoleType:
			roleType = aws.StringValue(tag.Value)
		case tags.OpenShiftVersion:
			version = aws.StringValue(tag.Value)
		}
	}
	return
}

func MarshalRoles(role []Role, b *bytes.Buffer) error {
	reqBodyBytes := new(bytes.Buffer)
	json.NewEncoder(reqBodyBytes).Encode(role)
	return prettyPrint(reqBodyBytes, b)
}

func prettyPrint(reqBodyBytes *bytes.Buffer, b *bytes.Buffer) error {
	err := json.Indent(b, reqBodyBytes.Bytes(), "", "  ")
	if err != nil {
		return err
	}
	return nil
}

func GetRoleName(prefix string, role string) string {
	name := fmt.Sprintf("%s-%s-Role", prefix, role)
	if len(name) > 64 {
		name = name[0:64]
	}
	return name
}

func GetOCMRoleName(prefix string, role string, postfix string) string {
	name := fmt.Sprintf("%s-%s-Role-%s", prefix, role, postfix)
	if len(name) > 64 {
		name = name[0:64]
	}
	return name
}

func GetUserRoleName(prefix string, role string, userName string) string {
	name := fmt.Sprintf("%s-%s-%s-Role", prefix, role, userName)
	if len(name) > 64 {
		name = name[0:64]
	}
	return name
}

func GetPolicyName(prefix string, namespace string, name string) string {
	policy := fmt.Sprintf("%s-%s-%s", prefix, namespace, name)
	if len(policy) > 64 {
		policy = policy[0:64]
	}
	return policy
}

func GetOperatorPolicyARN(accountID string, prefix string, namespace string, name string) string {
	return GetPolicyARN(accountID, GetPolicyName(prefix, namespace, name))
}

func GetPolicyARN(accountID string, name string) string {
	partition := GetPartition()
	return fmt.Sprintf("arn:%s:iam::%s:policy/%s", partition, accountID, name)
}

func GetRoleARN(accountID string, name string) string {
	partition := GetPartition()
	return fmt.Sprintf("arn:%s:iam::%s:role/%s", partition, accountID, name)
}

func GetOIDCProviderARN(accountID string, providerURL string) string {
	partition := GetPartition()
	return fmt.Sprintf("arn:%s:iam::%s:oidc-provider/%s", partition, accountID, providerURL)
}

func GetPartition() string {
	region, err := GetRegion(arguments.GetRegion())
	if err != nil || region == "" {
		return endpoints.AwsPartitionID
	}
	partition, ok := endpoints.PartitionForRegion(endpoints.DefaultPartitions(), region)
	if !ok || partition.ID() == "" {
		return endpoints.AwsPartitionID
	}
	return partition.ID()
}

func GetOperatorRoleName(cluster *cmv1.Cluster, operator Operator) string {
	for _, role := range cluster.AWS().STS().OperatorIAMRoles() {
		if role.Namespace() == operator.Namespace && role.Name() == operator.Name {
			return strings.SplitN(role.RoleARN(), "/", 2)[1]
		}
	}
	return ""
}

func GetPrefixFromAccountRole(cluster *cmv1.Cluster) (string, error) {
	role := AccountRoles[InstallerAccountRole]
	roleName, err := GetAccountRoleName(cluster)
	if err != nil {
		return "", err
	}
	rolePrefix := TrimRoleSuffix(roleName, fmt.Sprintf("-%s-Role", role.Name))
	return rolePrefix, nil
}

func GetPrefixFromOperatorRole(cluster *cmv1.Cluster) string {
	operator := cluster.AWS().STS().OperatorIAMRoles()[0]
	roleName := strings.SplitN(operator.RoleARN(), "/", 2)[1]
	rolePrefix := TrimRoleSuffix(roleName, fmt.Sprintf("-%s-%s", operator.Namespace(), operator.Name()))
	return rolePrefix
}

// Role names can be truncated if they are over 64 chars, so we need to make sure we aren't missing a truncated suffix
func TrimRoleSuffix(orig, sufix string) string {
	for i := len(sufix); i >= 0; i-- {
		if strings.HasSuffix(orig, sufix[:i]) {
			return orig[:len(orig)-i]
		}
	}
	return orig
}

func GetAccountRoleName(cluster *cmv1.Cluster) (string, error) {
	parsedARN, err := arn.Parse(cluster.AWS().STS().RoleARN())
	if err != nil {
		return "", err
	}
	roleName := strings.SplitN(parsedARN.Resource, "/", 2)[1]
	return roleName, nil
}

func GeneratePolicyFiles(reporter *rprtr.Object, env string, generateAccountRolePolicies bool,
	generateOperatorRolePolicies bool, policies map[string]string, credRequests map[string]*cmv1.STSOperator) error {
	if generateAccountRolePolicies {
		for file := range AccountRoles {
			//Get trust policy
			filename := fmt.Sprintf("sts_%s_trust_policy", file)
			policyDetail := policies[filename]
			policy := InterpolatePolicyDocument(policyDetail, map[string]string{
				"partition":      GetPartition(),
				"aws_account_id": GetJumpAccount(env),
			})

			filename = GetFormattedFileName(filename)
			reporter.Debugf("Saving '%s' to the current directory", filename)
			err := helper.SaveDocument(policy, filename)
			if err != nil {
				return err
			}
			//Get the permission policy
			filename = fmt.Sprintf("sts_%s_permission_policy", file)
			policyDetail = policies[filename]
			if policyDetail == "" {
				continue
			}
			//Check and save it as json file
			filename = GetFormattedFileName(filename)
			reporter.Debugf("Saving '%s' to the current directory", filename)
			err = helper.SaveDocument(policyDetail, filename)
			if err != nil {
				return err
			}
		}
	}
	if generateOperatorRolePolicies {
		for credrequest := range credRequests {
			filename := fmt.Sprintf("openshift_%s_policy", credrequest)
			policyDetail := policies[filename]
			//In case any missing policy we dont want to block the user.This might not happen
			if policyDetail == "" {
				continue
			}
			reporter.Debugf("Saving '%s' to the current directory", filename)
			filename = GetFormattedFileName(filename)
			err := helper.SaveDocument(policyDetail, filename)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func GetFormattedFileName(filename string) string {
	//Check and save it as json file
	ext := filepath.Ext(filename)
	if ext != ".json" {
		filename = fmt.Sprintf("%s.json", filename)
	}
	return filename
}

func BuildOperatorRolePolicies(prefix string, accountID string, awsClient Client, commands []string,
	defaultPolicyVersion string, credRequests map[string]*cmv1.STSOperator) []string {
	for credrequest, operator := range credRequests {
		policyARN := GetOperatorPolicyARN(accountID, prefix, operator.Namespace(), operator.Name())
		_, err := awsClient.IsPolicyExists(policyARN)
		if err != nil {
			name := GetPolicyName(prefix, operator.Namespace(), operator.Name())
			iamTags := fmt.Sprintf(
				"Key=%s,Value=%s Key=%s,Value=%s Key=%s,Value=%s Key=%s,Value=%s",
				tags.OpenShiftVersion, defaultPolicyVersion,
				tags.RolePrefix, prefix,
				"operator_namespace", operator.Namespace(),
				"operator_name", operator.Name(),
			)
			createPolicy := fmt.Sprintf("aws iam create-policy \\\n"+
				"\t--policy-name %s \\\n"+
				"\t--policy-document file://openshift_%s_policy.json \\\n"+
				"\t--tags %s",
				name, credrequest, iamTags)
			commands = append(commands, createPolicy)
		} else {
			policTags := fmt.Sprintf(
				"Key=%s,Value=%s",
				tags.OpenShiftVersion, defaultPolicyVersion,
			)
			createPolicy := fmt.Sprintf("aws iam create-policy-version \\\n"+
				"\t--policy-arn %s \\\n"+
				"\t--policy-document file://openshift_%s_policy.json \\\n"+
				"\t--set-as-default",
				policyARN, credrequest)
			tagPolicy := fmt.Sprintf("aws iam tag-policy \\\n"+
				"\t--tags %s \\\n"+
				"\t--policy-arn %s",
				policTags, policyARN)
			commands = append(commands, createPolicy, tagPolicy)
		}
	}
	return commands
}

func UpggradeOperatorRolePolicies(reporter *rprtr.Object, awsClient Client, accountID string,
	prefix string, policies map[string]string, defaultPolicyVersion string,
	credRequests map[string]*cmv1.STSOperator) error {
	for credrequest, operator := range credRequests {
		policyARN := GetOperatorPolicyARN(accountID, prefix, operator.Namespace(), operator.Name())
		filename := fmt.Sprintf("openshift_%s_policy", credrequest)
		policyDetails := policies[filename]
		policyARN, err := awsClient.EnsurePolicy(policyARN, policyDetails,
			defaultPolicyVersion, map[string]string{
				tags.OpenShiftVersion: defaultPolicyVersion,
				tags.RolePrefix:       prefix,
				"operator_namespace":  operator.Namespace(),
				"operator_name":       operator.Name(),
			})
		if err != nil {
			return err
		}
		reporter.Infof("Upgraded policy with ARN '%s' to version '%s'", policyARN, defaultPolicyVersion)
	}
	return nil
}

const subnetTemplate = "%s (%s)"

// SetSubnetOption Creates a subnet options using a predefined template.
func SetSubnetOption(subnet, zone string) string {
	return fmt.Sprintf(subnetTemplate, subnet, zone)
}

// ParseSubnet Parses the subnet from the option chosen by the user.
func ParseSubnet(subnetOption string) string {
	return strings.Split(subnetOption, " ")[0]
}

const (
	BYOVPCSingleAZSubnetsCount      = 2
	BYOVPCMultiAZSubnetsCount       = 6
	privateLinkSingleAZSubnetsCount = 1
	privateLinkMultiAZSubnetsCount  = 3
)

func ValidateSubnetsCount(multiAZ bool, privateLink bool, subnetsInputCount int) error {
	if privateLink {
		if multiAZ && subnetsInputCount != privateLinkMultiAZSubnetsCount {
			return fmt.Errorf("The number of subnets for a multi-AZ private link cluster should be %d, "+
				"instead received: %d", privateLinkMultiAZSubnetsCount, subnetsInputCount)
		}
		if !multiAZ && subnetsInputCount != privateLinkSingleAZSubnetsCount {
			return fmt.Errorf("The number of subnets for a single AZ private link cluster should be %d, "+
				"instead received: %d", privateLinkSingleAZSubnetsCount, subnetsInputCount)
		}
	} else {
		if multiAZ && subnetsInputCount != BYOVPCMultiAZSubnetsCount {
			return fmt.Errorf("The number of subnets for a multi-AZ cluster should be %d, "+
				"instead received: %d", BYOVPCMultiAZSubnetsCount, subnetsInputCount)
		}
		if !multiAZ && subnetsInputCount != BYOVPCSingleAZSubnetsCount {
			return fmt.Errorf("The number of subnets for a single AZ cluster should be %d, "+
				"instead received: %d", BYOVPCSingleAZSubnetsCount, subnetsInputCount)
		}
	}

	return nil
}
