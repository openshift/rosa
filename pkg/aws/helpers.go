package aws

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/aws/tags"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var RoleNameRE = regexp.MustCompile(`^[\w+=,.@-]+$`)

// UserTagKeyRE , UserTagValueRE - https://docs.aws.amazon.com/general/latest/gr/aws_tagging.html#tag-conventions
var UserTagKeyRE = regexp.MustCompile(`^[\pL\pZ\pN_.:/=+\-@]{1,128}$`)
var UserTagValueRE = regexp.MustCompile(`^[\pL\pZ\pN_.:/=+\-@]{0,256}$`)

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
	return fmt.Sprintf("arn:aws:iam::%s:policy/%s", accountID, name)
}

func GetRoleARN(accountID string, name string) string {
	return fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, name)
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
	rolePrefix := strings.TrimSuffix(roleName, fmt.Sprintf("-%s-Role", role.Name))
	return rolePrefix, nil
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
	generateOperatorRolePolicies bool) error {
	if generateAccountRolePolicies {
		for file := range AccountRoles {
			filename := fmt.Sprintf("sts_%s_trust_policy.json", file)
			path := fmt.Sprintf("templates/policies/%s", filename)
			policy, err := ReadPolicyDocument(path, map[string]string{
				"aws_account_id": JumpAccounts[env],
			})
			if err != nil {
				return err
			}
			reporter.Debugf("Saving '%s' to the current directory", filename)
			err = SaveDocument(policy, filename)
			if err != nil {
				return err
			}
			filename = fmt.Sprintf("sts_%s_permission_policy.json", file)
			path = fmt.Sprintf("templates/policies/%s", filename)
			policy, err = ReadPolicyDocument(path)
			if err != nil {
				return err
			}
			reporter.Debugf("Saving '%s' to the current directory", filename)
			err = SaveDocument(policy, filename)
			if err != nil {
				return err
			}
		}
	}
	if generateOperatorRolePolicies {
		for credrequest := range CredentialRequests {
			filename := fmt.Sprintf("openshift_%s_policy.json", credrequest)
			path := fmt.Sprintf("templates/policies/%s", filename)
			policy, err := ReadPolicyDocument(path)
			if err != nil {
				return err
			}
			reporter.Debugf("Saving '%s' to the current directory", filename)
			err = SaveDocument(policy, filename)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func SaveDocument(doc []byte, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(doc)
	if err != nil {
		return err
	}

	return nil
}

func GenerateOperatorPolicyFiles(reporter *rprtr.Object) error {
	for credrequest := range CredentialRequests {
		filename := fmt.Sprintf("openshift_%s_policy.json", credrequest)
		path := fmt.Sprintf("templates/policies/%s", filename)

		policy, err := ReadPolicyDocument(path)
		if err != nil {
			return err
		}

		reporter.Debugf("Saving '%s' to the current directory", filename)
		err = SaveDocument(policy, filename)
		if err != nil {
			return err
		}
	}

	return nil
}

func GenerateRolePolicyDoc(cluster *cmv1.Cluster, accountID string, operator Operator) (string, error) {
	oidcEndpointURL, err := url.ParseRequestURI(cluster.AWS().STS().OIDCEndpointURL())
	if err != nil {
		return "", err
	}
	issuerURL := fmt.Sprintf("%s%s", oidcEndpointURL.Host, oidcEndpointURL.Path)

	oidcProviderARN := fmt.Sprintf("arn:aws:iam::%s:oidc-provider/%s", accountID, issuerURL)

	serviceAccounts := []string{}
	for _, sa := range operator.ServiceAccountNames {
		serviceAccounts = append(serviceAccounts,
			fmt.Sprintf("system:serviceaccount:%s:%s", operator.Namespace, sa))
	}

	path := "templates/policies/operator_iam_role_policy.json"
	policy, err := ReadPolicyDocument(path, map[string]string{
		"oidc_provider_arn": oidcProviderARN,
		"issuer_url":        issuerURL,
		"service_accounts":  strings.Join(serviceAccounts, `" , "`),
	})
	if err != nil {
		return "", err
	}

	return string(policy), nil
}
