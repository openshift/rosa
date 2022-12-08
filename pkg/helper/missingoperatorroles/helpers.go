package missingoperatorroles

import (
	"fmt"
	"os"
	"time"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/aws"
	awscb "github.com/openshift/rosa/pkg/aws/commandbuilder"
	awscbRoles "github.com/openshift/rosa/pkg/aws/commandbuilder/helper/roles"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/zgalor/weberr"
)

func HandleMissingOperatorRoles(
	mode string, r *rosa.Runtime,
	cluster *v1.Cluster, missingRolesInCS map[string]*v1.STSOperator,
	policies map[string]string, unifiedPath string,
	operatorRolePolicyPrefix string, isInvokedFromClusterUpgrade bool) error {
	if len(missingRolesInCS) > 0 {
		createdMissingRoles := 0
		for _, operator := range missingRolesInCS {
			roleName := getMissingOperatorRoleName(cluster, operator)
			exists, _, err := r.AWSClient.CheckRoleExists(roleName)
			if err != nil {
				return weberr.Errorf("Error when detecting checking missing operator IAM roles %s", err)
			}
			if !exists {
				err = createMissingOperatorRoles(
					mode, r, cluster, missingRolesInCS, policies, unifiedPath, operatorRolePolicyPrefix, false)
				if err != nil {
					return weberr.Errorf("%s", err)
				}
				createdMissingRoles++
			}
		}
		if createdMissingRoles == 0 {
			r.Reporter.Infof(
				"Missing roles/policies have already been created. Please continue with cluster upgrade process.",
			)
		}
	}
	return nil
}

func getMissingOperatorRoleName(cluster *cmv1.Cluster, missingOperator *cmv1.STSOperator) string {
	rolePrefix := cluster.AWS().STS().OperatorRolePrefix()
	role := fmt.Sprintf("%s-%s-%s", rolePrefix, missingOperator.Namespace(), missingOperator.Name())
	if len(role) > 64 {
		role = role[0:64]
	}
	return role
}

func createMissingOperatorRoles(
	mode string, r *rosa.Runtime,
	cluster *v1.Cluster, missingRoles map[string]*v1.STSOperator,
	policies map[string]string, unifiedPath string,
	operatorRolePolicyPrefix string, isInvokedFromClusterUpgrade bool) error {
	accountID := r.Creator.AccountID
	switch mode {
	case aws.ModeAuto:
		err := upgradeMissingOperatorRole(
			missingRoles, cluster,
			accountID, r,
			policies, unifiedPath,
			operatorRolePolicyPrefix,
			isInvokedFromClusterUpgrade)
		if err != nil {
			return err
		}
		helper.DisplaySpinnerWithDelay(r.Reporter, "Waiting for operator roles to reconcile", 5*time.Second)
	case aws.ModeManual:
		commands, err := buildMissingOperatorRoleCommand(
			missingRoles, cluster,
			accountID, r,
			policies, unifiedPath,
			operatorRolePolicyPrefix)
		if err != nil {
			return err
		}
		if r.Reporter.IsTerminal() {
			r.Reporter.Infof("Run the following commands to create the operator roles:\n")
		}
		fmt.Println(commands)
		if isInvokedFromClusterUpgrade {
			r.Reporter.Infof("Run the following command to continue scheduling cluster upgrade"+
				" once account and operator roles have been upgraded : \n\n"+
				"\trosa upgrade cluster --cluster %s\n", cluster.ID())
			os.Exit(0)
		}
	default:
		r.Reporter.Errorf("Invalid mode. Allowed values are %s", aws.Modes)
		os.Exit(1)
	}
	return nil
}

func buildMissingOperatorRoleCommand(
	missingRoles map[string]*cmv1.STSOperator, cluster *cmv1.Cluster,
	accountID string, r *rosa.Runtime,
	policies map[string]string, unifiedPath string,
	operatorRolePolicyPrefix string) (string, error) {
	commands := []string{}
	for missingRole, operator := range missingRoles {
		roleName := getMissingOperatorRoleName(cluster, operator)
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

func upgradeMissingOperatorRole(
	missingRoles map[string]*v1.STSOperator, cluster *v1.Cluster,
	accountID string, r *rosa.Runtime,
	policies map[string]string, unifiedPath string,
	operatorRolePolicyPrefix string, isInvokedFromClusterUpgrade bool) error {
	for _, operator := range missingRoles {
		roleName := getMissingOperatorRoleName(cluster, operator)
		if !confirm.Prompt(true, "Create the '%s' role?", roleName) {
			if isInvokedFromClusterUpgrade {
				return weberr.Errorf("Operator roles need to be upgraded to proceed with cluster upgrade")
			}
			continue
		}
		policyDetails := policies["operator_iam_role_policy"]

		policyARN := aws.GetOperatorPolicyARN(
			accountID,
			operatorRolePolicyPrefix,
			operator.Namespace(),
			operator.Name(),
			unifiedPath,
		)
		policy, err := aws.GenerateOperatorRolePolicyDoc(cluster, accountID, operator, policyDetails)
		if err != nil {
			return err
		}
		r.Reporter.Debugf("Creating role '%s'", roleName)
		roleARN, err := r.AWSClient.EnsureRole(roleName, policy, "", "",
			map[string]string{
				tags.ClusterID:         cluster.ID(),
				tags.OperatorNamespace: operator.Namespace(),
				tags.OperatorName:      operator.Name(),
				tags.RedHatManaged:     "true",
			}, unifiedPath)
		if err != nil {
			return err
		}
		r.Reporter.Infof("Created role '%s' with ARN '%s'", roleName, roleARN)
		r.Reporter.Debugf("Attaching permission policy '%s' to role '%s'", policyARN, roleName)
		err = r.AWSClient.AttachRolePolicy(roleName, policyARN)
		if err != nil {
			return weberr.Errorf("Failed to attach role policy. Check your prefix or run "+
				"'rosa create operator-roles' to create the necessary policies: %s", err)
		}
	}
	return nil
}
