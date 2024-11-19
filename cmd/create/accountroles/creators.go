package accountroles

import (
	"fmt"

	common "github.com/openshift-online/ocm-common/pkg/aws/validations"
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
	skipPermissionFiles() bool
	getAccountRolesMap() map[string]aws.AccountRole
}

func initCreator(r *rosa.Runtime, managedPolicies bool, classic bool, hostedCP bool, isClassicValueSet bool,
	isHostedCPValueSet bool) (creator, bool) {
	// Unmanaged should be used for fedramp
	if r.Creator.IsGovcloud {
		return &unmanagedPoliciesCreator{}, true
	}

	// Classic ROSA managed policies
	if managedPolicies && !hostedCP {
		return &managedPoliciesCreator{}, true
	}

	// If the user didn't select topologies (default flow creates both), or selected both topologies
	if !isClassicValueSet && !isHostedCPValueSet || hostedCP && classic {
		r.Reporter.Infof("By default, the create account-roles command creates two sets of account roles, " +
			"one for classic ROSA clusters, and one for Hosted Control Plane clusters." +
			"\nIn order to create a single set, please set one of the following flags: --classic or --hosted-cp")
		return &doubleRolesCreator{}, true
	}

	if hostedCP {
		return &hcpManagedPoliciesCreator{}, true
	}

	// Classic ROSA unmanaged policies
	if classic {
		return &unmanagedPoliciesCreator{}, true
	}

	return nil, false
}

type accountRolesCreationInput struct {
	prefix               string
	permissionsBoundary  string
	accountID            string
	env                  string
	policies             map[string]*cmv1.AWSSTSPolicy
	defaultPolicyVersion string
	path                 string
	isSharedVpc          bool
}

func buildRolesCreationInput(prefix, permissionsBoundary, accountID, env string,
	policies map[string]*cmv1.AWSSTSPolicy, defaultPolicyVersion string,
	path string, isSharedVpc bool) *accountRolesCreationInput {
	return &accountRolesCreationInput{
		prefix:               prefix,
		permissionsBoundary:  permissionsBoundary,
		accountID:            accountID,
		env:                  env,
		policies:             policies,
		defaultPolicyVersion: defaultPolicyVersion,
		path:                 path,
		isSharedVpc:          isSharedVpc,
	}
}

type managedPoliciesCreator struct{}

func (mp *managedPoliciesCreator) createRoles(r *rosa.Runtime, input *accountRolesCreationInput) error {
	r.Reporter.Infof("Creating classic account roles using '%s'", r.Creator.ARN)

	for file, role := range aws.AccountRoles {
		accRoleName := common.GetRoleName(input.prefix, role.Name)
		assumeRolePolicy := getAssumeRolePolicy(r.Creator.Partition, file, input)

		r.Reporter.Debugf("Creating role '%s'", accRoleName)
		tagsList := mp.getRoleTags(file, input)
		r.Reporter.Debugf("start to EnsureRole")
		roleARN, err := r.AWSClient.EnsureRole(r.Reporter, accRoleName, assumeRolePolicy, input.permissionsBoundary,
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
		err = r.AWSClient.AttachRolePolicy(r.Reporter, accRoleName, policyARN)
		if err != nil {
			return err
		}
	}

	return nil
}

func (mp *managedPoliciesCreator) printCommands(r *rosa.Runtime, input *accountRolesCreationInput) error {
	commands := []string{}
	for file, role := range aws.AccountRoles {
		accRoleName := common.GetRoleName(input.prefix, role.Name)
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
	tagsList[common.ManagedPolicies] = tags.True

	return tagsList
}

func (mp *managedPoliciesCreator) skipPermissionFiles() bool {
	return true
}

func (mp *managedPoliciesCreator) getAccountRolesMap() map[string]aws.AccountRole {
	return aws.AccountRoles
}

type unmanagedPoliciesCreator struct{}

func (up *unmanagedPoliciesCreator) createRoles(r *rosa.Runtime, input *accountRolesCreationInput) error {
	r.Reporter.Infof("Creating classic account roles using '%s'", r.Creator.ARN)

	for file, role := range aws.AccountRoles {
		accRoleName := common.GetRoleName(input.prefix, role.Name)
		assumeRolePolicy := getAssumeRolePolicy(r.Creator.Partition, file, input)
		tagsList := up.getRoleTags(file, input)
		filename := fmt.Sprintf("sts_%s_permission_policy", file)

		err := createRoleUnmanagedPolicy(r, input, accRoleName, assumeRolePolicy, tagsList, filename)
		if err != nil {
			return err
		}
	}

	return nil
}

func (up *unmanagedPoliciesCreator) printCommands(r *rosa.Runtime, input *accountRolesCreationInput) error {
	commands := []string{}
	for file, role := range aws.AccountRoles {
		accRoleName := common.GetRoleName(input.prefix, role.Name)
		iamTags := up.getRoleTags(file, input)

		createRole := buildCreateRoleCommand(accRoleName, file, iamTags, input)

		policyName := aws.GetPolicyName(accRoleName)
		policyDocument := fmt.Sprintf("file://sts_%s_permission_policy.json", file)

		createPolicy := buildCreatePolicyCommand(policyName, policyDocument, iamTags, input.path)

		policyARN := aws.GetPolicyARN(r.Creator.Partition, input.accountID, accRoleName, input.path)

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

func (up *unmanagedPoliciesCreator) skipPermissionFiles() bool {
	return false
}

func (up *unmanagedPoliciesCreator) getAccountRolesMap() map[string]aws.AccountRole {
	return aws.AccountRoles
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

func (db *doubleRolesCreator) skipPermissionFiles() bool {
	return false
}

func (db *doubleRolesCreator) getAccountRolesMap() map[string]aws.AccountRole {
	return aws.AccountRoles
}

func createRoleUnmanagedPolicy(r *rosa.Runtime, input *accountRolesCreationInput, accRoleName string,
	assumeRolePolicy string, tagsList map[string]string, filename string) error {
	r.Reporter.Debugf("Creating role '%s'", accRoleName)
	roleARN, err := r.AWSClient.EnsureRole(r.Reporter, accRoleName, assumeRolePolicy, input.permissionsBoundary,
		input.defaultPolicyVersion, tagsList, input.path, false)
	if err != nil {
		return err
	}
	r.Reporter.Infof("Created role '%s' with ARN '%s'", accRoleName, roleARN)

	policyPermissionDetail := aws.GetPolicyDetails(input.policies, filename)

	policyARN := aws.GetPolicyARN(r.Creator.Partition, r.Creator.AccountID, accRoleName, input.path)

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
	return r.AWSClient.AttachRolePolicy(r.Reporter, accRoleName, policyARN)
}

func getAssumeRolePolicy(partition string, file string, input *accountRolesCreationInput) string {
	filename := fmt.Sprintf("sts_%s_trust_policy", file)
	policyDetail := aws.GetPolicyDetails(input.policies, filename)
	return aws.InterpolatePolicyDocument(partition, policyDetail, map[string]string{
		"partition":      partition,
		"aws_account_id": aws.GetJumpAccount(input.env),
	})
}

type hcpManagedPoliciesCreator struct{}

func (hcp *hcpManagedPoliciesCreator) createRoles(r *rosa.Runtime, input *accountRolesCreationInput) error {
	r.Reporter.Infof("Creating hosted CP account roles using '%s'", r.Creator.ARN)

	for file, role := range aws.HCPAccountRoles {
		accRoleName := common.GetRoleName(input.prefix, role.Name)
		assumeRolePolicy := getAssumeRolePolicy(r.Creator.Partition, file, input)

		r.Reporter.Debugf("Creating role '%s'", accRoleName)
		tagsList := hcp.getRoleTags(file, input)
		roleARN, err := r.AWSClient.EnsureRole(r.Reporter, accRoleName, assumeRolePolicy, input.permissionsBoundary,
			input.defaultPolicyVersion, tagsList, input.path, true)
		if err != nil {
			return err
		}
		r.Reporter.Infof("Created role '%s' with ARN '%s'", accRoleName, roleARN)

		if role == aws.HCPAccountRoles[aws.HCPInstallerRole] {
			if input.isSharedVpc {
				err := attachHcpSharedVpcPolicy(r, args.route53RoleArn, accRoleName,
					input.path, input.defaultPolicyVersion)
				if err != nil {
					return err
				}
				err = attachHcpSharedVpcPolicy(r, args.vpcEndpointRoleArn, accRoleName,
					input.path, input.defaultPolicyVersion)
				if err != nil {
					return err
				}
			}
		}

		policyKeys := aws.GetHcpAccountRolePolicyKeys(file)
		for _, policyKey := range policyKeys {
			policyARN, err := aws.GetManagedPolicyARN(input.policies, policyKey)
			if err != nil {
				// EC2 policy is only available to orgs for zero-egress feature toggle enabled
				if policyKey == aws.WorkerEC2RegistryKey {
					r.Reporter.Infof("Skip attaching policy '%s' to role '%s' (zero egress feature toggle is not enabled)",
						policyKey, accRoleName)
					continue
				}
				return err
			}

			r.Reporter.Debugf("Attaching permission policy to role '%s'", policyKey)
			err = r.AWSClient.AttachRolePolicy(r.Reporter, accRoleName, policyARN)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (hcp *hcpManagedPoliciesCreator) printCommands(r *rosa.Runtime, input *accountRolesCreationInput) error {
	commands := []string{}
	for file, role := range aws.HCPAccountRoles {
		accRoleName := common.GetRoleName(input.prefix, role.Name)
		iamTags := hcp.getRoleTags(file, input)

		createRole := buildCreateRoleCommand(accRoleName, file, iamTags, input)

		commands = append(commands, createRole)

		policyKeys := aws.GetHcpAccountRolePolicyKeys(file)
		for _, policyKey := range policyKeys {
			policyARN, err := aws.GetManagedPolicyARN(input.policies, policyKey)
			if err != nil {
				// EC2 policy is only available to orgs for zero-egress feature toggle enabled
				if policyKey == aws.WorkerEC2RegistryKey {
					r.Reporter.Infof("Skip attaching policy '%s' to role '%s' (zero egress feature toggle is not enabled)",
						policyKey, accRoleName)
					continue
				}
				return err
			}

			attachRolePolicy := buildAttachRolePolicyCommand(accRoleName, policyARN)
			commands = append(commands, attachRolePolicy)
		}
	}

	r.Reporter.Infof("Run the following commands to create the hosted CP account roles and policies:\n")
	fmt.Println(awscb.JoinCommands(commands) + "\n")

	return nil
}

func (hcp *hcpManagedPoliciesCreator) getRoleTags(roleType string, input *accountRolesCreationInput) map[string]string {
	tagsList := getBaseRoleTags(roleType, input)
	tagsList[common.ManagedPolicies] = tags.True
	tagsList[tags.HypershiftPolicies] = tags.True

	return tagsList
}

func (hcp *hcpManagedPoliciesCreator) skipPermissionFiles() bool {
	return true
}

func (hcp *hcpManagedPoliciesCreator) getAccountRolesMap() map[string]aws.AccountRole {
	return aws.HCPAccountRoles
}

func getBaseRoleTags(roleType string, input *accountRolesCreationInput) map[string]string {
	return map[string]string{
		common.OpenShiftVersion: input.defaultPolicyVersion,
		tags.RolePrefix:         input.prefix,
		tags.RoleType:           roleType,
		tags.RedHatManaged:      tags.True,
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

func attachHcpSharedVpcPolicy(r *rosa.Runtime, sharedVpcRoleArn string, roleName string,
	path string, defaultPolicyVersion string) error {
	policyDetails := aws.InterpolatePolicyDocument(r.Creator.Partition, aws.SharedVpcDefaultPolicy, map[string]string{
		"shared_vpc_role_arn": sharedVpcRoleArn,
	})

	policyTags := map[string]string{
		tags.RedHatManaged: aws.TrueString,
	}

	userProvidedRoleName, err := aws.GetResourceIdFromARN(sharedVpcRoleArn)
	if err != nil {
		return err
	}
	policyName := fmt.Sprintf(aws.AssumeRolePolicyPrefix, userProvidedRoleName)
	policyArn := aws.GetPolicyARN(r.Creator.Partition, r.Creator.AccountID, roleName, path)
	policyArn, err = r.AWSClient.EnsurePolicyWithName(policyArn, policyDetails,
		defaultPolicyVersion, policyTags, path, policyName)
	if err != nil {
		return err
	}
	err = r.AWSClient.AttachRolePolicy(r.Reporter, roleName, policyArn)
	if err != nil {
		return err
	}
	return nil
}
