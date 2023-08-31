package operatorroles

import (
	"fmt"
	"strings"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/rosa"
	errors "github.com/zgalor/weberr"
)

const assumePolicyAction = "AssumeRole"

func computePolicyARN(accountID string, prefix string, namespace string, name string, path string) string {
	if prefix == "" {
		prefix = aws.DefaultPrefix
	}
	policy := fmt.Sprintf("%s-%s-%s", prefix, namespace, name)
	if len(policy) > 64 {
		policy = policy[0:64]
	}
	if path != "" {
		return fmt.Sprintf("arn:%s:iam::%s:policy%s%s", aws.GetPartition(), accountID, path, policy)
	}
	return fmt.Sprintf("arn:%s:iam::%s:policy/%s", aws.GetPartition(), accountID, policy)
}

func validateIngressOperatorPolicyOverride(r *rosa.Runtime, policyArn string, sharedVpcRoleArn string,
	installerRolePrefix string) error {
	_, err := r.AWSClient.IsPolicyExists(policyArn)
	policyExists := err == nil
	if !policyExists {
		return nil
	}

	policyDocument, err := r.AWSClient.GetDefaultPolicyDocument(policyArn)
	if err != nil {
		return err
	}

	// The policy associated with the installer role. In the case it contains a different shared VPC role ARN,
	// don't override it.
	if strings.Contains(policyDocument, assumePolicyAction) && !strings.Contains(policyDocument, sharedVpcRoleArn) {
		return errors.UserErrorf("Policy with ARN '%s' contains 'sts:AssumeRole' action with different shared VPC role ARN "+
			"than '%s'."+
			"\nThe policy is associated with the installer role with the prefix '%s'."+
			"\nTo create operator roles with shared vpc role ARN '%s', "+
			"please provide a different value for '--installer-role-arn'.",
			policyArn, sharedVpcRoleArn, installerRolePrefix, sharedVpcRoleArn)
	}

	return nil
}
