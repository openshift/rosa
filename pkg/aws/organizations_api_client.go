package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/organizations"
)

// OrganizationsApiClient is an interface that defines the methods that we want to use
// from the Client type in the AWS SDK ("github.com/aws/aws-sdk-go-v2/service/organizations")
// The AIM is to only contain methods that are defined in the AWS SDK's Organizations
// Client.
// For the cases where logic is desired to be implemented combining Organizations calls
// and other logic use the pkg/aws.Client type.
// If you need to use a method provided by the AWS SDK's Organizations Client but it
// is not defined in this interface then it has to be added and all
// the types implementing this interface have to implement the new method.
// The reason this interface has been defined is so we can perform unit testing
// on methods that make use of the AWS Organizations service.
//

type OrganizationsApiClient interface {
	AcceptHandshake(ctx context.Context, params *organizations.AcceptHandshakeInput, optFns ...func(*organizations.Options),
	) (*organizations.AcceptHandshakeOutput, error)

	AttachPolicy(ctx context.Context, params *organizations.AttachPolicyInput, optFns ...func(*organizations.Options),
	) (*organizations.AttachPolicyOutput, error)

	CancelHandshake(ctx context.Context,
		params *organizations.CancelHandshakeInput, optFns ...func(*organizations.Options),
	) (*organizations.CancelHandshakeOutput, error)

	CloseAccount(ctx context.Context, params *organizations.CloseAccountInput, optFns ...func(*organizations.Options),
	) (*organizations.CloseAccountOutput, error)

	CreateAccount(ctx context.Context, params *organizations.CreateAccountInput, optFns ...func(*organizations.Options),
	) (*organizations.CreateAccountOutput, error)

	CreateGovCloudAccount(ctx context.Context, params *organizations.CreateGovCloudAccountInput, optFns ...func(*organizations.Options),
	) (*organizations.CreateGovCloudAccountOutput, error)

	CreateOrganization(ctx context.Context, params *organizations.CreateOrganizationInput, optFns ...func(*organizations.Options),
	) (*organizations.CreateOrganizationOutput, error)

	CreateOrganizationalUnit(ctx context.Context, params *organizations.CreateOrganizationalUnitInput, optFns ...func(*organizations.Options),
	) (*organizations.CreateOrganizationalUnitOutput, error)

	CreatePolicy(ctx context.Context, params *organizations.CreatePolicyInput, optFns ...func(*organizations.Options),
	) (*organizations.CreatePolicyOutput, error)

	DeclineHandshake(ctx context.Context, params *organizations.DeclineHandshakeInput, optFns ...func(*organizations.Options),
	) (*organizations.DeclineHandshakeOutput, error)

	DeleteOrganization(ctx context.Context, params *organizations.DeleteOrganizationInput, optFns ...func(*organizations.Options),
	) (*organizations.DeleteOrganizationOutput, error)

	DeleteOrganizationalUnit(ctx context.Context, params *organizations.DeleteOrganizationalUnitInput, optFns ...func(*organizations.Options),
	) (*organizations.DeleteOrganizationalUnitOutput, error)

	DeletePolicy(ctx context.Context, params *organizations.DeletePolicyInput, optFns ...func(*organizations.Options),
	) (*organizations.DeletePolicyOutput, error)

	DeleteResourcePolicy(ctx context.Context, params *organizations.DeleteResourcePolicyInput, optFns ...func(*organizations.Options),
	) (*organizations.DeleteResourcePolicyOutput, error)

	DeregisterDelegatedAdministrator(ctx context.Context, params *organizations.DeregisterDelegatedAdministratorInput, optFns ...func(*organizations.Options),
	) (*organizations.DeregisterDelegatedAdministratorOutput, error)
	
	DescribeAccount(ctx context.Context, params *organizations.DescribeAccountInput, optFns ...func(*organizations.Options),
	) (*organizations.DescribeAccountOutput, error)
	
	DescribeCreateAccountStatus(ctx context.Context, params *organizations.DescribeCreateAccountStatusInput, optFns ...func(*organizations.Options),
	) (*organizations.DescribeCreateAccountStatusOutput, error)

	DescribeEffectivePolicy(ctx context.Context, params *organizations.DescribeEffectivePolicyInput, optFns ...func(*organizations.Options),
	) (*organizations.DescribeEffectivePolicyOutput, error)

	DescribeHandshake(ctx context.Context, params *organizations.DescribeHandshakeInput, optFns ...func(*organizations.Options),
	) (*organizations.DescribeHandshakeOutput, error)

	DescribeOrganization(ctx context.Context, params *organizations.DescribeOrganizationInput, optFns ...func(*organizations.Options),
	) (*organizations.DescribeOrganizationOutput, error)

	DescribeOrganizationalUnit(ctx context.Context, params *organizations.DescribeOrganizationalUnitInput, optFns ...func(*organizations.Options),
	) (*organizations.DescribeOrganizationalUnitOutput, error)

	DescribePolicy(ctx context.Context, params *organizations.DescribePolicyInput, optFns ...func(*organizations.Options),
	) (*organizations.DescribePolicyOutput, error)

	DescribeResourcePolicy(ctx context.Context, params *organizations.DescribeResourcePolicyInput, optFns ...func(*organizations.Options),
	) (*organizations.DescribeResourcePolicyOutput, error)

	DetachPolicy(ctx context.Context, params *organizations.DetachPolicyInput, optFns ...func(*organizations.Options),
	) (*organizations.DetachPolicyOutput, error)
	
	DisableAWSServiceAccess(ctx context.Context, params *organizations.DisableAWSServiceAccessInput, optFns ...func(*organizations.Options),
	) (*organizations.DisableAWSServiceAccessOutput, error)

	DisablePolicyType(ctx context.Context, params *organizations.DisablePolicyTypeInput, optFns ...func(*organizations.Options),
	) (*organizations.DisablePolicyTypeOutput, error)

	EnableAWSServiceAccess(ctx context.Context, params *organizations.EnableAWSServiceAccessInput, optFns ...func(*organizations.Options),
	) (*organizations.EnableAWSServiceAccessOutput, error)

	EnableAllFeatures(ctx context.Context, params *organizations.EnableAllFeaturesInput, optFns ...func(*organizations.Options),
	) (*organizations.EnableAllFeaturesOutput, error)

	EnablePolicyType(ctx context.Context, params *organizations.EnablePolicyTypeInput, optFns ...func(*organizations.Options),
	) (*organizations.EnablePolicyTypeOutput, error)

	InviteAccountToOrganization(ctx context.Context, params *organizations.InviteAccountToOrganizationInput, optFns ...func(*organizations.Options),
	) (*organizations.InviteAccountToOrganizationOutput, error)

	LeaveOrganization(ctx context.Context, params *organizations.LeaveOrganizationInput, optFns ...func(*organizations.Options),
	) (*organizations.LeaveOrganizationOutput, error)

	ListAccounts(ctx context.Context, params *organizations.ListAccountsInput, optFns ...func(*organizations.Options),
	) (*organizations.ListAccountsOutput, error)

	ListAWSServiceAccessForOrganization(ctx context.Context, params *organizations.ListAWSServiceAccessForOrganizationInput, optFns ...func(*organizations.Options),
	) (*organizations.ListAWSServiceAccessForOrganizationOutput, error)

	ListAccountsForParent(ctx context.Context, params *organizations.ListAccountsForParentInput, optFns ...func(*organizations.Options),
	) (*organizations.ListAccountsForParentOutput, error)

	ListChildren(ctx context.Context, params *organizations.ListChildrenInput, optFns ...func(*organizations.Options),
	) (*organizations.ListChildrenOutput, error)

	ListCreateAccountStatus(ctx context.Context, params *organizations.ListCreateAccountStatusInput, optFns ...func(*organizations.Options),
	) (*organizations.ListCreateAccountStatusOutput, error)

	ListDelegatedAdministrators(ctx context.Context, params *organizations.ListDelegatedAdministratorsInput, optFns ...func(*organizations.Options),
	) (*organizations.ListDelegatedAdministratorsOutput, error)

	ListDelegatedServicesForAccount(ctx context.Context, params *organizations.ListDelegatedServicesForAccountInput, optFns ...func(*organizations.Options),
	) (*organizations.ListDelegatedServicesForAccountOutput, error)

	ListHandshakesForAccount(ctx context.Context, params *organizations.ListHandshakesForAccountInput, optFns ...func(*organizations.Options),
	) (*organizations.ListHandshakesForAccountOutput, error)

	ListHandshakesForOrganization(ctx context.Context, params *organizations.ListHandshakesForOrganizationInput, optFns ...func(*organizations.Options),
	) (*organizations.ListHandshakesForOrganizationOutput, error)

	ListOrganizationalUnitsForParent(ctx context.Context, params *organizations.ListOrganizationalUnitsForParentInput, optFns ...func(*organizations.Options),
	) (*organizations.ListOrganizationalUnitsForParentOutput, error)

	ListParents(ctx context.Context, params *organizations.ListParentsInput, optFns ...func(*organizations.Options),
	) (*organizations.ListParentsOutput, error)

	ListPolicies(ctx context.Context, params *organizations.ListPoliciesInput, optFns ...func(*organizations.Options),
	) (*organizations.ListPoliciesOutput, error)

	ListPoliciesForTarget(ctx context.Context, params *organizations.ListPoliciesForTargetInput, optFns ...func(*organizations.Options),
	) (*organizations.ListPoliciesForTargetOutput, error)

	ListRoots(ctx context.Context, params *organizations.ListRootsInput, optFns ...func(*organizations.Options),
	) (*organizations.ListRootsOutput, error)

	ListTagsForResource(ctx context.Context, params *organizations.ListTagsForResourceInput, optFns ...func(*organizations.Options),
	) (*organizations.ListTagsForResourceOutput, error)

	ListTargetsForPolicy(ctx context.Context, params *organizations.ListTargetsForPolicyInput, optFns ...func(*organizations.Options),
	) (*organizations.ListTargetsForPolicyOutput, error)

	MoveAccount(ctx context.Context, params *organizations.MoveAccountInput, optFns ...func(*organizations.Options),
	) (*organizations.MoveAccountOutput, error)

	PutResourcePolicy(ctx context.Context, params *organizations.PutResourcePolicyInput, optFns ...func(*organizations.Options),
	) (*organizations.PutResourcePolicyOutput, error)

	RemoveAccountFromOrganization(ctx context.Context, params *organizations.RemoveAccountFromOrganizationInput, optFns ...func(*organizations.Options),
	) (*organizations.RemoveAccountFromOrganizationOutput, error)

	RegisterDelegatedAdministrator(ctx context.Context, params *organizations.RegisterDelegatedAdministratorInput, optFns ...func(*organizations.Options),
	) (*organizations.RegisterDelegatedAdministratorOutput, error)

	TagResource(ctx context.Context, params *organizations.TagResourceInput, optFns ...func(*organizations.Options),
	) (*organizations.TagResourceOutput, error)

	UntagResource(ctx context.Context, params *organizations.UntagResourceInput, optFns ...func(*organizations.Options),
	) (*organizations.UntagResourceOutput, error)

	UpdateOrganizationalUnit(ctx context.Context, params *organizations.UpdateOrganizationalUnitInput, optFns ...func(*organizations.Options),
	) (*organizations.UpdateOrganizationalUnitOutput, error)

	UpdatePolicy(ctx context.Context, params *organizations.UpdatePolicyInput, optFns ...func(*organizations.Options),
	) (*organizations.UpdatePolicyOutput, error)
}

// interface guard to ensure that all methods defined in the OrganizationsApiClient
// interface are implemented by the real AWS Organizations client. This interface
// guard should always compile
var _ OrganizationsApiClient = (*organizations.Client)(nil)

