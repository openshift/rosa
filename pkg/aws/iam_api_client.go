package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/iam"
)

// IamApiClient is an interface that defines the methods that we want to use
// from the Client type in the AWS SDK ("github.com/aws/aws-sdk-go-v2/service/iam")
// The aim is to only contain methods that are defined in the AWS SDK's IAM
// Client.
// For the cases where logic is desired to be implemened combining IAM calls
// and other logic use the pkg/aws.Client type.
// If you need to use a method provided by the AWS SDK's IAM Client but it
// is not defined in this interface then it has to be added and all
// the types implementing this interface have to implement the new method.
// The reason this interface has been defined is so we can perform unit testing
// on methods that make use of the AWS IAM service.
//

type IamApiClient interface {
	AddClientIDToOpenIDConnectProvider(ctx context.Context, params *iam.AddClientIDToOpenIDConnectProviderInput, optFns ...func(*iam.Options),
	) (*iam.AddClientIDToOpenIDConnectProviderOutput, error)
	AddRoleToInstanceProfile(ctx context.Context,
		params *iam.AddRoleToInstanceProfileInput, optFns ...func(*iam.Options),
	) (*iam.AddRoleToInstanceProfileOutput, error)

	AddUserToGroup(ctx context.Context, params *iam.AddUserToGroupInput, optFns ...func(*iam.Options),
	) (*iam.AddUserToGroupOutput, error)

	AttachGroupPolicy(ctx context.Context, params *iam.AttachGroupPolicyInput, optFns ...func(*iam.Options),
	) (*iam.AttachGroupPolicyOutput, error)

	AttachRolePolicy(ctx context.Context,
		params *iam.AttachRolePolicyInput, optFns ...func(*iam.Options),
	) (*iam.AttachRolePolicyOutput, error)

	AttachUserPolicy(ctx context.Context, params *iam.AttachUserPolicyInput, optFns ...func(*iam.Options),
	) (*iam.AttachUserPolicyOutput, error)

	ChangePassword(ctx context.Context, params *iam.ChangePasswordInput, optFns ...func(*iam.Options),
	) (*iam.ChangePasswordOutput, error)

	CreateAccessKey(ctx context.Context, params *iam.CreateAccessKeyInput, optFns ...func(*iam.Options),
	) (*iam.CreateAccessKeyOutput, error)

	CreateAccountAlias(ctx context.Context, params *iam.CreateAccountAliasInput, optFns ...func(*iam.Options),
	) (*iam.CreateAccountAliasOutput, error)

	CreateGroup(ctx context.Context, params *iam.CreateGroupInput, optFns ...func(*iam.Options),
	) (*iam.CreateGroupOutput, error)

	CreateInstanceProfile(ctx context.Context,
		params *iam.CreateInstanceProfileInput, optFns ...func(*iam.Options),
	) (*iam.CreateInstanceProfileOutput, error)

	CreateLoginProfile(ctx context.Context, params *iam.CreateLoginProfileInput, optFns ...func(*iam.Options),
	) (*iam.CreateLoginProfileOutput, error)

	CreateOpenIDConnectProvider(ctx context.Context,
		params *iam.CreateOpenIDConnectProviderInput, optFns ...func(*iam.Options),
	) (*iam.CreateOpenIDConnectProviderOutput, error)
	CreatePolicy(ctx context.Context,
		params *iam.CreatePolicyInput, optFns ...func(*iam.Options),
	) (*iam.CreatePolicyOutput, error)
	CreatePolicyVersion(ctx context.Context,
		params *iam.CreatePolicyVersionInput, optFns ...func(*iam.Options),
	) (*iam.CreatePolicyVersionOutput, error)
	CreateRole(ctx context.Context,
		params *iam.CreateRoleInput, optFns ...func(*iam.Options),
	) (*iam.CreateRoleOutput, error)

	CreateSAMLProvider(ctx context.Context, params *iam.CreateSAMLProviderInput, optFns ...func(*iam.Options),
	) (*iam.CreateSAMLProviderOutput, error)

	CreateServiceLinkedRole(ctx context.Context, params *iam.CreateServiceLinkedRoleInput, optFns ...func(*iam.Options),
	) (*iam.CreateServiceLinkedRoleOutput, error)

	CreateServiceSpecificCredential(ctx context.Context, params *iam.CreateServiceSpecificCredentialInput, optFns ...func(*iam.Options),
	) (*iam.CreateServiceSpecificCredentialOutput, error)

	CreateUser(ctx context.Context, params *iam.CreateUserInput, optFns ...func(*iam.Options),
	) (*iam.CreateUserOutput, error)

	CreateVirtualMFADevice(ctx context.Context, params *iam.CreateVirtualMFADeviceInput, optFns ...func(*iam.Options),
	) (*iam.CreateVirtualMFADeviceOutput, error)

	DeactivateMFADevice(ctx context.Context, params *iam.DeactivateMFADeviceInput, optFns ...func(*iam.Options),
	) (*iam.DeactivateMFADeviceOutput, error)

	DeleteAccessKey(ctx context.Context, params *iam.DeleteAccessKeyInput, optFns ...func(*iam.Options),
	) (*iam.DeleteAccessKeyOutput, error)

	DeleteAccountAlias(ctx context.Context, params *iam.DeleteAccountAliasInput, optFns ...func(*iam.Options),
	) (*iam.DeleteAccountAliasOutput, error)

	DeleteAccountPasswordPolicy(ctx context.Context, params *iam.DeleteAccountPasswordPolicyInput, optFns ...func(*iam.Options),
	) (*iam.DeleteAccountPasswordPolicyOutput, error)

	DeleteGroup(ctx context.Context, params *iam.DeleteGroupInput, optFns ...func(*iam.Options),
	) (*iam.DeleteGroupOutput, error)

	DeleteGroupPolicy(ctx context.Context, params *iam.DeleteGroupPolicyInput, optFns ...func(*iam.Options),
	) (*iam.DeleteGroupPolicyOutput, error)

	DeleteInstanceProfile(ctx context.Context,
		params *iam.DeleteInstanceProfileInput, optFns ...func(*iam.Options),
	) (*iam.DeleteInstanceProfileOutput, error)

	DeleteLoginProfile(ctx context.Context, params *iam.DeleteLoginProfileInput, optFns ...func(*iam.Options),
	) (*iam.DeleteLoginProfileOutput, error)

	DeleteOpenIDConnectProvider(ctx context.Context,
		params *iam.DeleteOpenIDConnectProviderInput, optFns ...func(*iam.Options),
	) (*iam.DeleteOpenIDConnectProviderOutput, error)

	DeletePolicy(ctx context.Context, params *iam.DeletePolicyInput, optFns ...func(*iam.Options),
	) (*iam.DeletePolicyOutput, error)

	DeleteRolePolicy(ctx context.Context, params *iam.DeleteRolePolicyInput, optFns ...func(*iam.Options),
	) (*iam.DeleteRolePolicyOutput, error)

	DeletePolicyVersion(ctx context.Context,
		params *iam.DeletePolicyVersionInput, optFns ...func(*iam.Options),
	) (*iam.DeletePolicyVersionOutput, error)
	DeleteRole(ctx context.Context,
		params *iam.DeleteRoleInput, optFns ...func(*iam.Options),
	) (*iam.DeleteRoleOutput, error)
	DeleteRolePermissionsBoundary(ctx context.Context,
		params *iam.DeleteRolePermissionsBoundaryInput, optFns ...func(*iam.Options),
	) (*iam.DeleteRolePermissionsBoundaryOutput, error)

	DetachRolePolicy(ctx context.Context,
		params *iam.DetachRolePolicyInput, optFns ...func(*iam.Options),
	) (*iam.DetachRolePolicyOutput, error)

	GetInstanceProfile(ctx context.Context,
		params *iam.GetInstanceProfileInput, optFns ...func(*iam.Options),
	) (*iam.GetInstanceProfileOutput, error)
	GetOpenIDConnectProvider(ctx context.Context,
		params *iam.GetOpenIDConnectProviderInput, optFns ...func(*iam.Options),
	) (*iam.GetOpenIDConnectProviderOutput, error)
	GetPolicy(ctx context.Context,
		params *iam.GetPolicyInput, optFns ...func(*iam.Options),
	) (*iam.GetPolicyOutput, error)
	GetRole(ctx context.Context,
		params *iam.GetRoleInput, optFns ...func(*iam.Options),
	) (*iam.GetRoleOutput, error)
	GetUser(ctx context.Context,
		params *iam.GetUserInput, optFns ...func(*iam.Options),
	) (*iam.GetUserOutput, error)

	GetPolicyVersion(ctx context.Context, params *iam.GetPolicyVersionInput, optFns ...func(*iam.Options),
	) (*iam.GetPolicyVersionOutput, error)

	GetRolePolicy(ctx context.Context, params *iam.GetRolePolicyInput, optFns ...func(*iam.Options),
	) (*iam.GetRolePolicyOutput, error)

	ListOpenIDConnectProviders(ctx context.Context, params *iam.ListOpenIDConnectProvidersInput, optFns ...func(*iam.Options),
	) (*iam.ListOpenIDConnectProvidersOutput, error)

	ListOpenIDConnectProviderTags(ctx context.Context, params *iam.ListOpenIDConnectProviderTagsInput, optFns ...func(*iam.Options),
	) (*iam.ListOpenIDConnectProviderTagsOutput, error)

	ListAttachedRolePolicies(ctx context.Context,
		params *iam.ListAttachedRolePoliciesInput, optFns ...func(*iam.Options),
	) (*iam.ListAttachedRolePoliciesOutput, error)
	ListPolicyTags(ctx context.Context,
		params *iam.ListPolicyTagsInput, optFns ...func(*iam.Options),
	) (*iam.ListPolicyTagsOutput, error)
	ListPolicyVersions(ctx context.Context,
		params *iam.ListPolicyVersionsInput, optFns ...func(*iam.Options),
	) (*iam.ListPolicyVersionsOutput, error)
	ListRoles(context.Context,
		*iam.ListRolesInput, ...func(*iam.Options),
	) (*iam.ListRolesOutput, error)

	ListPolicies(ctx context.Context,
		params *iam.ListPoliciesInput, optFns ...func(*iam.Options),
	) (*iam.ListPoliciesOutput, error)

	ListInstanceProfilesForRole(ctx context.Context, params *iam.ListInstanceProfilesForRoleInput, optFns ...func(*iam.Options),
	) (*iam.ListInstanceProfilesForRoleOutput, error)

	ListRolePolicies(ctx context.Context,
		params *iam.ListRolePoliciesInput, optFns ...func(*iam.Options),
	) (*iam.ListRolePoliciesOutput, error)
	ListRoleTags(ctx context.Context,
		params *iam.ListRoleTagsInput, optFns ...func(*iam.Options),
	) (*iam.ListRoleTagsOutput, error)

	ListUsers(ctx context.Context, params *iam.ListUsersInput, optFns ...func(*iam.Options),
	) (*iam.ListUsersOutput, error)

	ListAccessKeys(ctx context.Context, params *iam.ListAccessKeysInput, optFns ...func(*iam.Options),
	) (*iam.ListAccessKeysOutput, error)

	PutRolePermissionsBoundary(ctx context.Context,
		params *iam.PutRolePermissionsBoundaryInput, optFns ...func(*iam.Options),
	) (*iam.PutRolePermissionsBoundaryOutput, error)

	RemoveRoleFromInstanceProfile(ctx context.Context,
		params *iam.RemoveRoleFromInstanceProfileInput, optFns ...func(*iam.Options),
	) (*iam.RemoveRoleFromInstanceProfileOutput, error)

	PutRolePolicy(ctx context.Context, params *iam.PutRolePolicyInput, optFns ...func(*iam.Options),
	) (*iam.PutRolePolicyOutput, error)

	TagPolicy(ctx context.Context,
		params *iam.TagPolicyInput, optFns ...func(*iam.Options),
	) (*iam.TagPolicyOutput, error)

	TagUser(ctx context.Context, params *iam.TagUserInput, optFns ...func(*iam.Options),
	) (*iam.TagUserOutput, error)

	TagRole(ctx context.Context, params *iam.TagRoleInput, optFns ...func(*iam.Options),
	) (*iam.TagRoleOutput, error)

	UpdateAssumeRolePolicy(ctx context.Context,
		params *iam.UpdateAssumeRolePolicyInput, optFns ...func(*iam.Options),
	) (*iam.UpdateAssumeRolePolicyOutput, error)
}

// interface guard to ensure that all methods defined in the IamApiClient
// interface are implemented by the real AWS IAM client. This interface
// guard should always compile
var _ IamApiClient = (*iam.Client)(nil)