package config

import (
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	iamTypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/openshift-online/ocm-common/pkg/aws/aws_client"

	"github.com/openshift/rosa/tests/ci/config"
	"github.com/openshift/rosa/tests/utils/helper"
	"github.com/openshift/rosa/tests/utils/log"
)

// Takes a given prefix and generates the role and policy names
func generateNames(prefix string) (string, string) {
	roleName := prefix + "-autonode-operator-role"
	policyName := prefix + "-autonode-private-preview"
	return roleName, policyName
}

// Takes a given policy name and returns the expected ARN for that policy
func generatePolicyARN(awsclient *aws_client.AWSClient, policyName string) string {
	return fmt.Sprintf(
		"arn:%s:iam::%s:policy/%s",
		awsclient.GetAWSPartition(),
		awsclient.AccountID,
		policyName,
	)
}

func PrepareAutonodeRoleAndPolicy(prefix string, oidcProviderURL string, region string) (string, error) {
	var err error
	roleName, policyName := generateNames(prefix)
	var tags = map[string]string{
		"rosa_cli_testing": "true",
		"rosa_cli_prefix":  prefix,
	}
	var alreadyExists *iamTypes.EntityAlreadyExistsException
	var createdPolicy = false
	var createdRole = false

	awsclient, err := aws_client.CreateAWSClient("", region)
	if err != nil {
		return "", err
	}

	policyDocument, err := helper.ReadFileContent(path.Join(config.Test.ResourcesDir, "autonode_policy.json"))
	if err != nil {
		return "", err
	}
	log.Logger.Infof("Creating AutoNode IAM Policy '%s'", policyName)
	policy, err := awsclient.CreateIAMPolicy(policyName, policyDocument, tags)
	if errors.As(err, &alreadyExists) {
		log.Logger.Warnf("AutoNode IAM Policy '%s' already exists. Reusing existing IAM Policy", policyName)
		policy, err = awsclient.GetIAMPolicy(generatePolicyARN(awsclient, policyName))
		if err != nil {
			return "", err
		}
	} else if err != nil {
		log.Logger.Error("Failed to create AutoNode IAM Policy, cleaning up resources...")
		deleteError := DeleteAutonodeRoleAndPolicy(prefix, region)
		if deleteError != nil {
			return "", errors.Join(err, deleteError)
		}
		return "", err
	} else {
		createdPolicy = true
	}
	log.Logger.Infof("Waiting upto 15 seconds for IAM Policy '%s' to exist...", *policy.Arn)
	err = awsclient.WaitForResourceExisting("policy-"+*policy.Arn, 15)
	if err != nil {
		if createdPolicy {
			log.Logger.Error("Failed to wait for AutoNode IAM Policy to exist, cleaning up resources...")
			deleteError := DeleteAutonodeRoleAndPolicy(prefix, region)
			if deleteError != nil {
				return "", errors.Join(err, deleteError)
			}
		}
		return "", err
	}

	assumeRolePolicyDocument, err := helper.ReadFileContent(
		path.Join(config.Test.ResourcesDir, "autonode_trust_policy_template.json"),
	)
	if err != nil {
		if createdPolicy {
			log.Logger.Error("Failed to get AutoNode IAM Role trust policy, cleaning up resources...")
			deleteError := DeleteAutonodeRoleAndPolicy(prefix, region)
			if deleteError != nil {
				return "", errors.Join(err, deleteError)
			}
		}
		return "", err
	}
	oidcProviderURL, err = helper.ExtractOIDCProviderFromOidcUrl(oidcProviderURL)
	if err != nil {
		if createdPolicy {
			log.Logger.Error("Failed to get OIDC Provider for AutoNode IAM Role trust policy, cleaning up resources...")
			deleteError := DeleteAutonodeRoleAndPolicy(prefix, region)
			if deleteError != nil {
				return "", errors.Join(err, deleteError)
			}
		}
		return "", err
	}
	assumeRolePolicyDocument = strings.ReplaceAll(assumeRolePolicyDocument, "{AWS_ACCOUNT_ID}", awsclient.AccountID)
	assumeRolePolicyDocument = strings.ReplaceAll(assumeRolePolicyDocument, "{OIDC_PROVIDER_URL}", oidcProviderURL)
	log.Logger.Infof("Creating AutoNode IAM Role '%s'", roleName)
	role, err := awsclient.CreateRoleAndAttachPolicy(roleName, assumeRolePolicyDocument, "", tags, "", *policy.Arn)
	if errors.As(err, &alreadyExists) {
		log.Logger.Warnf("AutoNode IAM Role '%s' already exists. Reusing existing IAM Role", roleName)
		roleTemp, err := awsclient.GetRole(roleName)
		if err != nil {
			if createdPolicy {
				log.Logger.Error("Failed to get existing AutoNode IAM Role, cleaning up resources...")
				deleteError := DeleteAutonodeRoleAndPolicy(prefix, region)
				if deleteError != nil {
					return "", errors.Join(err, deleteError)
				}
			}
			return "", err
		}
		role = *roleTemp
	} else if err != nil {
		log.Logger.Error("Failed to create for AutoNode IAM Role, cleaning up resources...")
		deleteError := DeleteAutonodeRoleAndPolicy(prefix, region)
		if deleteError != nil {
			return "", errors.Join(err, deleteError)
		}
		return "", err
	} else {
		createdRole = true
	}
	log.Logger.Infof("Waiting upto 15 seconds for IAM Role '%s' to exist...", roleName)
	err = awsclient.WaitForResourceExisting("role-"+*role.RoleName, 15)
	if err != nil {
		if createdPolicy || createdRole {
			log.Logger.Error("Failed to wait for AutoNode IAM Role to exist, cleaning up resources...")
			deleteError := DeleteAutonodeRoleAndPolicy(prefix, region)
			if deleteError != nil {
				return "", errors.Join(err, deleteError)
			}
		}
		return "", err
	}

	log.Logger.Info("Waiting 3 seconds for AWS consistency...")
	time.Sleep(3 * time.Second)

	return *role.Arn, nil
}

func DeleteAutonodeRoleAndPolicy(prefix string, region string) error {
	var err error

	roleName, policyName := generateNames(prefix)

	awsclient, err := aws_client.CreateAWSClient("", region)
	if err != nil {
		return err
	}

	policyARN := generatePolicyARN(awsclient, policyName)
	var errs []error

	if detachErr := awsclient.DetachIAMPolicy(roleName, policyARN); detachErr != nil {
		var noSuchEntity *iamTypes.NoSuchEntityException
		if !errors.As(detachErr, &noSuchEntity) && !strings.Contains(detachErr.Error(), "is not attached") {
			errs = append(errs, fmt.Errorf("detach policy %s from role %s: %w", policyARN, roleName, detachErr))
		}
	}
	if deletePolErr := awsclient.DeleteIAMPolicy(policyARN); deletePolErr != nil {
		var noSuchEntity *iamTypes.NoSuchEntityException
		if !errors.As(deletePolErr, &noSuchEntity) {
			errs = append(errs, fmt.Errorf("delete policy %s: %w", policyARN, deletePolErr))
		}
	}
	if deleteRoleErr := awsclient.DeleteRole(roleName); deleteRoleErr != nil {
		var noSuchEntity *iamTypes.NoSuchEntityException
		if !errors.As(deleteRoleErr, &noSuchEntity) {
			errs = append(errs, fmt.Errorf("delete role %s: %w", roleName, deleteRoleErr))
		}
	}

	return errors.Join(errs...)
}
