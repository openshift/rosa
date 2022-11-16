package roles

import (
	"fmt"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/aws"
	awscb "github.com/openshift/rosa/pkg/aws/commandbuilder"
	awscbRoles "github.com/openshift/rosa/pkg/aws/commandbuilder/helper/roles"
	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/zgalor/weberr"
)

const (
	RosaUpgradeAccRolesModeAuto = "ROSAUpgradeAccountRolesModeAuto"
)

func GetOperatorRoleName(cluster *cmv1.Cluster, missingOperator *cmv1.STSOperator) string {
	operatorIAMRoles := cluster.AWS().STS().OperatorIAMRoles()
	rolePrefix := ""
	if len(operatorIAMRoles) > 0 {
		roleARN := operatorIAMRoles[0].RoleARN()
		roleName, err := aws.GetResourceIdFromARN(roleARN)
		if err != nil {
			return ""
		}

		m := strings.LastIndex(roleName, "-openshift")
		if m != -1 {
			rolePrefix = roleName[0:m]
		}
	}
	role := fmt.Sprintf("%s-%s-%s", rolePrefix, missingOperator.Namespace(), missingOperator.Name())
	if len(role) > 64 {
		role = role[0:64]
	}
	return role
}

func GetOperatorPaths(awsClient aws.Client, operatorRoles []*cmv1.OperatorIAMRole) (
	string, string, error) {
	for _, operatorRole := range operatorRoles {
		roleName, err := aws.GetResourceIdFromARN(operatorRole.RoleARN())
		if err != nil {
			return "", "", err
		}
		rolePolicies, err := awsClient.GetAttachedPolicy(&roleName)
		if err != nil {
			return "", "", err
		}
		policySuffix := aws.GetOperatorPolicyName("", operatorRole.Namespace(), operatorRole.Name())
		for _, rolePolicy := range rolePolicies {
			index := strings.LastIndex(rolePolicy.PolicyName, "-openshift")
			policyNameSuffixOnly := rolePolicy.PolicyName[index:]
			if strings.Contains(policySuffix, policyNameSuffixOnly) {
				rolePath, err := aws.GetPathFromARN(operatorRole.RoleARN())
				if err != nil {
					return "", "", err
				}
				policyPath, err := aws.GetPathFromARN(rolePolicy.PolicyArn)
				if err != nil {
					return "", "", err
				}
				return rolePath, policyPath, nil
			}
		}
	}
	return "", "", weberr.Errorf("Can not detect operator policy path. " +
		"Existing operator roles do not have operator policies attached to them")
}

func BuildMissingOperatorRoleCommand(
	missingRoles map[string]*cmv1.STSOperator,
	cluster *cmv1.Cluster,
	accountID string,
	r *rosa.Runtime,
	policies map[string]string,
	rolePath string,
	policyPath string,
	operatorRolePolicyPrefix string,
) (string, error) {
	commands := []string{}
	for missingRole, operator := range missingRoles {
		roleName := GetOperatorRoleName(cluster, operator)
		policyARN := aws.GetOperatorPolicyARN(
			accountID,
			operatorRolePolicyPrefix,
			operator.Namespace(),
			operator.Name(),
			policyPath,
		)
		policyDetails := policies["operator_iam_role_policy"]
		policy, err := aws.GenerateOperatorRolePolicyDoc(cluster, accountID, operator, policyDetails)
		if err != nil {
			return "", err
		}
		filename := fmt.Sprintf("operator_%s_policy", missingRole)
		filename = aws.GetFormattedFileName(filename)
		r.Reporter.Debugf("Saving '%s' to the current directory", filename)
		err = helper.SaveDocument(policy, filename)
		if err != nil {
			return "", err
		}
		missingCommands := awscbRoles.ManualCommandsForMissingOperatorRole(
			awscbRoles.ManualCommandsForMissingOperatorRolesInput{
				ClusterID:                cluster.ID(),
				OperatorRolePolicyPrefix: operatorRolePolicyPrefix,
				Operator:                 operator,
				RoleName:                 roleName,
				Filename:                 filename,
				RolePath:                 rolePath,
				PolicyARN:                policyARN,
			},
		)
		commands = append(commands, missingCommands...)

	}
	return awscb.JoinCommands(commands), nil
}
