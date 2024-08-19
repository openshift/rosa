package roles

import (
	"fmt"
	"os"
	"time"

	awsCommonUtils "github.com/openshift-online/ocm-common/pkg/aws/utils"
	awsCommonValidations "github.com/openshift-online/ocm-common/pkg/aws/validations"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/aws"
	awscb "github.com/openshift/rosa/pkg/aws/commandbuilder"
	awscbRoles "github.com/openshift/rosa/pkg/aws/commandbuilder/helper/roles"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	RosaUpgradeAccRolesModeAuto            = "ROSAUpgradeAccountRolesModeAuto"
	maxClusterNameLengthToUseForRolePrefix = 27
)

// GeOperatorRolePrefixFromClusterName returns a valid operator role prefix from the cluster name
// An operator role prefix is considered valid if it's length is less than or equal to 32 chars.
// A random 4 characters label is attached to the cluster name to reduce chances of collision.
// The cluster name and the random label are separate by '-'.
// If the cluster name is longer than 27 characters, only the first 27 characters will be used.
func GeOperatorRolePrefixFromClusterName(clusterName string) string {
	if len(clusterName) > maxClusterNameLengthToUseForRolePrefix {
		return fmt.Sprintf("%s-%s", clusterName[0:maxClusterNameLengthToUseForRolePrefix], helper.RandomLabel(4))
	}
	return fmt.Sprintf("%s-%s", clusterName, helper.RandomLabel(4))
}

func GetOperatorRoleName(cluster *cmv1.Cluster, missingOperator *cmv1.STSOperator) string {
	rolePrefix := cluster.AWS().STS().OperatorRolePrefix()
	role := fmt.Sprintf("%s-%s-%s", rolePrefix, missingOperator.Namespace(), missingOperator.Name())
	return awsCommonUtils.TruncateRoleName(role)
}

func BuildMissingOperatorRoleCommand(
	missingRoles map[string]*cmv1.STSOperator,
	cluster *cmv1.Cluster,
	accountID string,
	r *rosa.Runtime,
	policies map[string]*cmv1.AWSSTSPolicy,
	unifiedPath string,
	operatorRolePolicyPrefix string,
	managedPolicies bool,
) (string, error) {
	commands := []string{}
	for missingRole, operator := range missingRoles {
		roleName := GetOperatorRoleName(cluster, operator)

		var policyARN string
		var err error
		if managedPolicies {
			policyARN, err = aws.GetManagedPolicyARN(policies, fmt.Sprintf("openshift_%s_policy", missingRole))
			if err != nil {
				return "", err
			}
		} else {
			policyARN = aws.GetOperatorPolicyARN(
				r.Creator.Partition,
				accountID,
				operatorRolePolicyPrefix,
				operator.Namespace(),
				operator.Name(),
				unifiedPath,
			)
		}
		policyDetails := aws.GetPolicyDetails(policies, "operator_iam_role_policy")
		policy, err := aws.GenerateOperatorRolePolicyDoc(r.Creator.Partition, cluster, accountID, operator, policyDetails)
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
				ManagedPolicies:          managedPolicies,
			},
		)
		commands = append(commands, missingCommands...)

	}
	return awscb.JoinCommands(commands), nil
}

func ValidateAccountRolesManagedPolicies(r *rosa.Runtime, prefix string, hostedCPPolicies bool) error {
	policies, err := r.OCMClient.GetPolicies("")
	if err != nil {
		return fmt.Errorf("Failed to fetch policies: %v", err)
	}

	if hostedCPPolicies {
		return r.AWSClient.ValidateHCPAccountRolesManagedPolicies(prefix, policies)
	}

	return r.AWSClient.ValidateAccountRolesManagedPolicies(prefix, policies)
}

func ValidateUnmanagedAccountRoles(roleARNs []string, awsClient aws.Client, version string) error {
	// iterate and validate role arns against openshift version
	for _, ARN := range roleARNs {
		if ARN == "" {
			continue
		}
		// get role from arn
		role, err := awsClient.GetRoleByARN(ARN)
		if err != nil {
			return fmt.Errorf("Could not get Role '%s' : %v", ARN, err)
		}

		validVersion, err := awsCommonValidations.HasCompatibleVersionTags(
			role.Tags,
			ocm.GetVersionMinor(version),
		)
		if err != nil {
			return fmt.Errorf("Could not validate Role '%s' : %v", ARN, err)
		}
		if !validVersion {
			return fmt.Errorf("Account role '%s' is not compatible with version %s. "+
				"Run 'rosa create account-roles' to create compatible roles and try again.",
				ARN, version)
		}
	}

	return nil
}

func ValidateOperatorRolesManagedPolicies(r *rosa.Runtime, cluster *cmv1.Cluster,
	operatorRoles map[string]*cmv1.STSOperator, policies map[string]*cmv1.AWSSTSPolicy, mode string, prefix string,
	unifiedPath string, upgradeVersion string, hostedCPPolicies bool) error {
	if upgradeVersion != "" {
		missingRolesInCS, err := r.OCMClient.FindMissingOperatorRolesForUpgrade(cluster, upgradeVersion)
		if err != nil {
			return err
		}
		if len(missingRolesInCS) > 0 {
			r.Reporter.Infof("Starting to upgrade the operator IAM roles")
			err = CreateMissingRoles(r, missingRolesInCS, cluster, mode, prefix, policies, unifiedPath, true)
			if err != nil {
				return err
			}
		}
	}

	return r.AWSClient.ValidateOperatorRolesManagedPolicies(cluster, operatorRoles, policies, hostedCPPolicies)
}

func CreateMissingRoles(
	r *rosa.Runtime,
	missingRolesInCS map[string]*cmv1.STSOperator,
	cluster *cmv1.Cluster,
	mode string,
	prefix string,
	policies map[string]*cmv1.AWSSTSPolicy,
	unifiedPath string,
	managedPolicies bool,
) error {
	createdMissingRoles := 0
	for _, operator := range missingRolesInCS {
		roleName := GetOperatorRoleName(cluster, operator)
		exists, _, err := r.AWSClient.CheckRoleExists(roleName)
		if err != nil {
			return r.Reporter.Errorf("Error when detecting checking missing operator IAM roles %s", err)
		}
		if !exists {
			err = createOperatorRole(mode, r, cluster, prefix, missingRolesInCS, policies, unifiedPath, managedPolicies)
			if err != nil {
				r.Reporter.Errorf("%s", err)
				os.Exit(1)
			}
			createdMissingRoles++
		}
	}
	if createdMissingRoles == 0 {
		r.Reporter.Infof(
			"Missing roles/policies have already been created. Please continue with cluster upgrade process.",
		)
	}

	return nil
}

func createOperatorRole(
	mode string, r *rosa.Runtime, cluster *cmv1.Cluster, prefix string, missingRoles map[string]*cmv1.STSOperator,
	policies map[string]*cmv1.AWSSTSPolicy, unifiedPath string, managedPolicies bool) error {
	accountID := r.Creator.AccountID
	switch mode {
	case interactive.ModeAuto:
		err := upgradeMissingOperatorRole(missingRoles, cluster, accountID, prefix, r,
			policies, unifiedPath, managedPolicies)
		if err != nil {
			return err
		}
		helper.DisplaySpinnerWithDelay(r.Reporter, "Waiting for operator roles to reconcile", 5*time.Second)
	case interactive.ModeManual:
		commands, err := BuildMissingOperatorRoleCommand(
			missingRoles, cluster, accountID, r, policies, unifiedPath, prefix, managedPolicies)
		if err != nil {
			return err
		}
		if r.Reporter.IsTerminal() {
			r.Reporter.Infof("Run the following commands to create the operator roles:\n")
		}
		fmt.Println(commands)
	default:
		r.Reporter.Errorf("Invalid mode. Allowed values are %s", interactive.Modes)
		os.Exit(1)
	}
	return nil
}

func upgradeMissingOperatorRole(missingRoles map[string]*cmv1.STSOperator, cluster *cmv1.Cluster,
	accountID string, prefix string, r *rosa.Runtime, policies map[string]*cmv1.AWSSTSPolicy,
	unifiedPath string, managedPolicies bool) error {
	for key, operator := range missingRoles {
		roleName := GetOperatorRoleName(cluster, operator)
		if !confirm.Prompt(true, "Create the '%s' role?", roleName) {
			continue
		}
		policyDetails := aws.GetPolicyDetails(policies, "operator_iam_role_policy")

		var policyARN string
		var err error
		if managedPolicies {
			policyARN, err = aws.GetManagedPolicyARN(policies, fmt.Sprintf("openshift_%s_policy", key))
			if err != nil {
				return err
			}
		} else {
			policyARN = aws.GetOperatorPolicyARN(r.Creator.Partition,
				accountID, prefix, operator.Namespace(), operator.Name(), unifiedPath)
		}

		policy, err := aws.GenerateOperatorRolePolicyDoc(r.Creator.Partition, cluster, accountID, operator, policyDetails)
		if err != nil {
			return err
		}
		tagsList := map[string]string{
			tags.ClusterID:         cluster.ID(),
			tags.OperatorNamespace: operator.Namespace(),
			tags.OperatorName:      operator.Name(),
			tags.RedHatManaged:     "true",
		}
		if managedPolicies {
			tagsList[awsCommonValidations.ManagedPolicies] = "true"
		}
		r.Reporter.Debugf("Creating role '%s'", roleName)
		roleARN, err := r.AWSClient.EnsureRole(r.Reporter, roleName, policy, "", "",
			tagsList, unifiedPath, false)
		if err != nil {
			return err
		}
		r.Reporter.Infof("Created role '%s' with ARN '%s'", roleName, roleARN)
		r.Reporter.Debugf("Attaching permission policy '%s' to role '%s'", policyARN, roleName)
		err = r.AWSClient.AttachRolePolicy(r.Reporter, roleName, policyARN)
		if err != nil {
			return fmt.Errorf("Failed to attach role policy. Check your prefix or run "+
				"'rosa create account-roles' to create the necessary policies: %s", err)
		}
	}
	return nil
}

func ValidateAdditionalAllowedPrincipals(aapARNs []string) error {
	duplicate, found := aws.HasDuplicates(aapARNs)
	if found {
		return fmt.Errorf("Invalid additional allowed principals list, duplicate key '%s' found",
			duplicate)
	}
	for _, aap := range aapARNs {
		err := aws.ARNValidator(aap)
		if err != nil {
			return fmt.Errorf("Expected valid ARNs for additional allowed principals list: %s",
				err)
		}
	}
	return nil
}
