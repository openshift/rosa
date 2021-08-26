package aws

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/sirupsen/logrus"

	"github.com/openshift/rosa/pkg/arguments"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var RoleNameRE = regexp.MustCompile(`^[\w+=,.@-]+$`)

// UserTagKeyRE , UserTagValueRE - https://docs.aws.amazon.com/general/latest/gr/aws_tagging.html#tag-conventions
var UserTagKeyRE = regexp.MustCompile(`^[A-Za-zÀ-ȕ0-9_.:/=+\-@]{1,128}$`)
var UserTagValueRE = regexp.MustCompile(`^[A-Za-zÀ-ȕ0-9_.:/=+\-@]{0,256}$`)

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
