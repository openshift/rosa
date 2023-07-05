package accountroles

import (
	"fmt"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/aws"
	awscb "github.com/openshift/rosa/pkg/aws/commandbuilder"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/rosa"
)

type creator interface {
	createRoles(*rosa.Runtime, *accountRolesCreationInput) error
	getRoleTags(string, *accountRolesCreationInput) map[string]string
	printCommands(*rosa.Runtime, *accountRolesCreationInput) error
}

func initCreator(managedPolicies bool, classic bool, hostedCP bool) creator {
	// Classic ROSA managed policies
	if managedPolicies && !hostedCP {
		return &managedPoliciesCreator{}
	}

	if hostedCP && classic {
		return &doubleRolesCreator{}
	}

	if hostedCP {
		return &hcpManagedPoliciesCreator{}
	}

	// Default flow creates a set of roles with unmanaged policies
	return &unmanagedPoliciesCreator{}
}

type accountRolesCreationInput struct {
	prefix               string
	permissionsBoundary  string
	accountID            string
	env                  string
	policies             map[string]*cmv1.AWSSTSPolicy
	defaultPolicyVersion string
	path                 string
}

func buildRolesCreationInput(prefix, permissionsBoundary, accountID, env string,
	policies map[string]*cmv1.AWSSTSPolicy, defaultPolicyVersion string,
	path string) *accountRolesCreationInput {
	return &accountRolesCreationInput{
		prefix:               prefix,
		permissionsBoundary:  permissionsBoundary,
		accountID:            accountID,
		env:                  env,
		policies:             policies,
		defaultPolicyVersion: defaultPolicyVersion,
		path:                 path,
	}
}

type managedPoliciesCreator struct{}

func (mp *managedPoliciesCreator) createRoles(r *rosa.Runtime, input *accountRolesCreationInput) error {
	r.Reporter.Infof("Creating classic account roles using '%s'", r.Creator.ARN)

	for file, role := range aws.AccountRoles {
		accRoleName := aws.GetRoleName(input.prefix, role.Name)
		assumeRolePolicy := getAssumeRolePolicy(file, input)

		r.Reporter.Debugf("Creating role '%s'", accRoleName)
		tagsList := mp.getRoleTags(file, input)
		roleARN, err := r.AWSClient.EnsureRole(accRoleName, assumeRolePolicy, input.permissionsBoundary,
			input.defaultPolicyVersion, tagsList, input.path, true)
		if err != nil {
			return err
		}
		r.Reporter.Infof("Created role '%s' with ARN '%s'", accRoleName, roleARN)

		err = attachManagedPolicies(r, input, file, accRoleName)
		if err != nil {
			return err
		}
	}

	r.Reporter.Infof("To create a cluster with these roles, run the following command:\n" +
		"rosa create cluster --sts")

	return nil
}

func attachManagedPolicies(r *rosa.Runtime, input *accountRolesCreationInput, roleType string,
	accRoleName string) error {
	policyKeys := aws.GetAccountRolePolicyKeys(roleType)

	for _, policyKey := range policyKeys {
		policyARN, err := aws.GetManagedPolicyARN(input.policies, policyKey)
		if err != nil {
			return err
		}

		r.Reporter.Debugf("Attaching permission policy to role '%s'", policyKey)
		err = r.AWSClient.AttachRolePolicy(accRoleName, policyARN)
		if err != nil {
			return err
		}
	}

	return nil
}

func (mp *managedPoliciesCreator) printCommands(r *rosa.Runtime, input *accountRolesCreationInput) error {
	commands := []string{}
	for file, role := range aws.AccountRoles {
		accRoleName := aws.GetRoleName(input.prefix, role.Name)
		iamTags := mp.getRoleTags(file, input)

		createRole := buildCreateRoleCommand(accRoleName, file, iamTags, input)
		commands = append(commands, createRole)

		policyKeys := aws.GetAccountRolePolicyKeys(file)
		for _, policyKey := range policyKeys {
			policyARN, err := aws.GetManagedPolicyARN(input.policies, policyKey)
			if err != nil {
				return err
			}

			attachRolePolicy := buildAttachRolePolicyCommand(accRoleName, policyARN)
			commands = append(commands, attachRolePolicy)
		}
	}

	r.Reporter.Infof("Run the following commands to create the classic account roles and policies:\n")
	fmt.Println(awscb.JoinCommands(commands) + "\n")

	return nil
}

func (mp *managedPoliciesCreator) getRoleTags(roleType string, input *accountRolesCreationInput) map[string]string {
	tagsList := getBaseRoleTags(roleType, input)
	tagsList[tags.ManagedPolicies] = tags.True

	return tagsList
}

type unmanagedPoliciesCreator struct{}

func (up *unmanagedPoliciesCreator) createRoles(r *rosa.Runtime, input *accountRolesCreationInput) error {
	r.Reporter.Infof("Creating classic account roles using '%s'", r.Creator.ARN)

	for file, role := range aws.AccountRoles {
		accRoleName := aws.GetRoleName(input.prefix, role.Name)
		assumeRolePolicy := getAssumeRolePolicy(file, input)
		tagsList := up.getRoleTags(file, input)
		filename := fmt.Sprintf("sts_%s_permission_policy", file)

		err := createRoleUnmanagedPolicy(r, input, accRoleName, assumeRolePolicy, tagsList, filename)
		if err != nil {
			return err
		}
	}

	r.Reporter.Infof("To create a cluster with these roles, run the following command:\n" +
		"rosa create cluster --sts")

	return nil
}

func (up *unmanagedPoliciesCreator) printCommands(r *rosa.Runtime, input *accountRolesCreationInput) error {
	commands := []string{}
	for file, role := range aws.AccountRoles {
		accRoleName := aws.GetRoleName(input.prefix, role.Name)
		iamTags := up.getRoleTags(file, input)

		createRole := buildCreateRoleCommand(accRoleName, file, iamTags, input)

		policyName := aws.GetPolicyName(accRoleName)
		policyDocument := fmt.Sprintf("file://sts_%s_permission_policy.json", file)

		createPolicy := buildCreatePolicyCommand(policyName, policyDocument, iamTags, input.path)

		policyARN := aws.GetPolicyARN(input.accountID, accRoleName, input.path)

		attachRolePolicy := buildAttachRolePolicyCommand(accRoleName, policyARN)

		commands = append(commands, createRole, createPolicy, attachRolePolicy)
	}

	r.Reporter.Infof("Run the following commands to create the classic account roles and policies:\n")
	fmt.Println(awscb.JoinCommands(commands) + "\n")

	return nil
}

func (up *unmanagedPoliciesCreator) getRoleTags(roleType string, input *accountRolesCreationInput) map[string]string {
	return getBaseRoleTags(roleType, input)
}

type doubleRolesCreator struct{}

func (db *doubleRolesCreator) createRoles(r *rosa.Runtime, input *accountRolesCreationInput) error {
	// Create classic account roles
	unmanagedCreator := unmanagedPoliciesCreator{}
	err := unmanagedCreator.createRoles(r, input)
	if err != nil {
		return err
	}

	// Create Hypershift account roles
	hcpCreator := hcpManagedPoliciesCreator{}
	return hcpCreator.createRoles(r, input)
}

func (db *doubleRolesCreator) printCommands(r *rosa.Runtime, input *accountRolesCreationInput) error {
	// Build classic account roles command
	unmanagedCreator := unmanagedPoliciesCreator{}
	err := unmanagedCreator.printCommands(r, input)
	if err != nil {
		return err
	}

	// Build Hypershift account roles command
	hcpCreator := hcpManagedPoliciesCreator{}
	return hcpCreator.printCommands(r, input)
}

// getRoleTags is not needed, but here to satisfy the interface
func (db *doubleRolesCreator) getRoleTags(roleType string, input *accountRolesCreationInput) map[string]string {
	return nil
}

func createRoleUnmanagedPolicy(r *rosa.Runtime, input *accountRolesCreationInput, accRoleName string,
	assumeRolePolicy string, tagsList map[string]string, filename string) error {
	r.Reporter.Debugf("Creating role '%s'", accRoleName)
	roleARN, err := r.AWSClient.EnsureRole(accRoleName, assumeRolePolicy, input.permissionsBoundary,
		input.defaultPolicyVersion, tagsList, input.path, false)
	if err != nil {
		return err
	}
	r.Reporter.Infof("Created role '%s' with ARN '%s'", accRoleName, roleARN)

	policyPermissionDetail := aws.GetPolicyDetails(input.policies, filename)

	policyARN := aws.GetPolicyARN(r.Creator.AccountID, accRoleName, input.path)

	r.Reporter.Debugf("Creating permission policy '%s'", policyARN)
	if args.forcePolicyCreation {
		policyARN, err = r.AWSClient.ForceEnsurePolicy(policyARN, policyPermissionDetail,
			input.defaultPolicyVersion, tagsList, input.path)
	} else {
		policyARN, err = r.AWSClient.EnsurePolicy(policyARN, policyPermissionDetail,
			input.defaultPolicyVersion, tagsList, input.path)
	}
	if err != nil {
		return err
	}

	r.Reporter.Debugf("Attaching permission policy to role '%s'", filename)
	return r.AWSClient.AttachRolePolicy(accRoleName, policyARN)
}

func getAssumeRolePolicy(file string, input *accountRolesCreationInput) string {
	filename := fmt.Sprintf("sts_%s_trust_policy", file)
	policyDetail := aws.GetPolicyDetails(input.policies, filename)

	return aws.InterpolatePolicyDocument(policyDetail, map[string]string{
		"partition":      aws.GetPartition(),
		"aws_account_id": aws.GetJumpAccount(input.env),
	})
}

type hcpManagedPoliciesCreator struct{}

func (hcp *hcpManagedPoliciesCreator) createRoles(r *rosa.Runtime, input *accountRolesCreationInput) error {
	r.Reporter.Infof("Creating hosted CP account roles using '%s'", r.Creator.ARN)

	for file, role := range aws.HCPAccountRoles {
		accRoleName := aws.GetRoleName(input.prefix, role.Name)
		assumeRolePolicy := getAssumeRolePolicy(file, input)

		r.Reporter.Debugf("Creating role '%s'", accRoleName)
		tagsList := hcp.getRoleTags(file, input)
		roleARN, err := r.AWSClient.EnsureRole(accRoleName, assumeRolePolicy, input.permissionsBoundary,
			input.defaultPolicyVersion, tagsList, input.path, true)
		if err != nil {
			return err
		}
		r.Reporter.Infof("Created role '%s' with ARN '%s'", accRoleName, roleARN)

		policyKey := fmt.Sprintf("sts_hcp_%s_permission_policy", file)
		policyARN, err := aws.GetManagedPolicyARN(input.policies, policyKey)
		if err != nil {
			return err
		}

		r.Reporter.Debugf("Attaching permission policy to role '%s'", policyKey)
		err = r.AWSClient.AttachRolePolicy(accRoleName, policyARN)
		if err != nil {
			return err
		}
	}

	r.Reporter.Infof("To create a cluster with these roles, run the following command:\n" +
		"rosa create cluster --hosted-cp")

	return nil
}

func (hcp *hcpManagedPoliciesCreator) printCommands(r *rosa.Runtime, input *accountRolesCreationInput) error {
	commands := []string{}
	for file, role := range aws.HCPAccountRoles {
		accRoleName := aws.GetRoleName(input.prefix, role.Name)
		iamTags := hcp.getRoleTags(file, input)

		createRole := buildCreateRoleCommand(accRoleName, file, iamTags, input)

		policyKey := fmt.Sprintf("sts_hcp_%s_permission_policy", file)
		policyARN, err := aws.GetManagedPolicyARN(input.policies, policyKey)
		if err != nil {
			return err
		}

		attachRolePolicy := buildAttachRolePolicyCommand(accRoleName, policyARN)
		commands = append(commands, createRole, attachRolePolicy)
	}

	r.Reporter.Infof("Run the following commands to create the hosted CP account roles and policies:\n")
	fmt.Println(awscb.JoinCommands(commands) + "\n")

	return nil
}

func (hcp *hcpManagedPoliciesCreator) getRoleTags(roleType string, input *accountRolesCreationInput) map[string]string {
	tagsList := getBaseRoleTags(roleType, input)
	tagsList[tags.ManagedPolicies] = tags.True
	tagsList[tags.HypershiftPolicies] = tags.True

	return tagsList
}

func getBaseRoleTags(roleType string, input *accountRolesCreationInput) map[string]string {
	return map[string]string{
		tags.OpenShiftVersion: input.defaultPolicyVersion,
		tags.RolePrefix:       input.prefix,
		tags.RoleType:         roleType,
		tags.RedHatManaged:    tags.True,
	}
}

func buildCreateRoleCommand(accRoleName string, file string, iamTags map[string]string,
	input *accountRolesCreationInput) string {
	return awscb.NewIAMCommandBuilder().
		SetCommand(awscb.CreateRole).
		AddParam(awscb.RoleName, accRoleName).
		AddParam(awscb.AssumeRolePolicyDocument, fmt.Sprintf("file://sts_%s_trust_policy.json", file)).
		AddParam(awscb.PermissionsBoundary, input.permissionsBoundary).
		AddTags(iamTags).
		AddParam(awscb.Path, input.path).
		Build()
}

func buildCreatePolicyCommand(policyName string, policyDocument string, iamTags map[string]string, path string) string {
	return awscb.NewIAMCommandBuilder().
		SetCommand(awscb.CreatePolicy).
		AddParam(awscb.PolicyName, policyName).
		AddParam(awscb.PolicyDocument, policyDocument).
		AddTags(iamTags).
		AddParam(awscb.Path, path).
		Build()
}

func buildAttachRolePolicyCommand(accRoleName string, policyARN string) string {
	return awscb.NewIAMCommandBuilder().
		SetCommand(awscb.AttachRolePolicy).
		AddParam(awscb.RoleName, accRoleName).
		AddParam(awscb.PolicyArn, policyARN).
		Build()
}
