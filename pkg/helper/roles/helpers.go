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
	if rolePrefix == "" {

	}
	role := fmt.Sprintf("%s-%s-%s", rolePrefix, missingOperator.Namespace(), missingOperator.Name())
	if len(role) > 64 {
		role = role[0:64]
	}
	return role
}

func BuildMissingOperatorRoleCommand(
	missingRoles map[string]*cmv1.STSOperator,
	cluster *cmv1.Cluster,
	accountID string,
	r *rosa.Runtime,
	policies map[string]string,
	unifiedPath string,
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
			unifiedPath,
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
				RolePath:                 unifiedPath,
				PolicyARN:                policyARN,
			},
		)
		commands = append(commands, missingCommands...)

	}
	return awscb.JoinCommands(commands), nil
}
