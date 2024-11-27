package roles

import (
	"fmt"

	"github.com/openshift/rosa/pkg/aws"
	awscb "github.com/openshift/rosa/pkg/aws/commandbuilder"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/rosa"
)

const policyDocumentBody = ` \
'{
  "Version": "2012-10-17",
  "Statement": {
    "Effect": "Allow",
    "Action": "sts:AssumeRole",
    "Resource": "%{shared_vpc_role_arn}"
  }
}'`

type ManualSharedVpcPolicyDetails struct {
	Command       string
	Name          string
	AlreadyExists bool
}

func GetHcpSharedVpcPolicyDetails(r *rosa.Runtime, roleArn string) (bool, string,
	string, error) {
	interpolatedPolicyDetails := aws.InterpolatePolicyDocument(r.Creator.Partition, policyDocumentBody,
		map[string]string{
			"shared_vpc_role_arn": roleArn,
		})

	roleName, err := aws.GetResourceIdFromARN(roleArn)
	if err != nil {
		return false, "", "", err
	}

	policyName := fmt.Sprintf(aws.AssumeRolePolicyPrefix, roleName)

	predictedPolicyArn := aws.GetPolicyArn(r.Creator.Partition, r.Creator.AccountID, policyName, "")

	existsQuery, _ := r.AWSClient.IsPolicyExists(predictedPolicyArn)

	var iamTags = map[string]string{
		tags.RedHatManaged: aws.TrueString,
		tags.HcpSharedVpc:  aws.TrueString,
	}

	createPolicy := awscb.NewIAMCommandBuilder().
		SetCommand(awscb.CreatePolicy).
		AddParam(awscb.PolicyName, policyName).
		AddParam(awscb.PolicyDocument, interpolatedPolicyDetails).
		AddTags(iamTags).
		AddParam(awscb.Path, "").
		Build()

	return existsQuery != nil, createPolicy, policyName, nil
}
