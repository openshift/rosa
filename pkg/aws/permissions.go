package aws

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/openshift/moactl/assets"
)

// PermissionGroup is the group of permissions needed by cluster creation, operation, or teardown.
type PermissionGroup string

// PolicyStatement models an AWS policy statement entry.
type PolicyStatement struct {
	Sid string `json:"sid,omitempty"`
	// Effect indicates if this policy statement is to Allow or Deny.
	Effect string `json:"effect"`
	// Action describes the particular AWS service actions that should be allowed or denied.
	// (i.e. ec2:StartInstances, iam:ChangePassword)
	Action []string `json:"action"`
	// Resource specifies the object(s) this statement should apply to. (or "*" for all)
	Resource interface{} `json:"resource"`
}

// PolicyDocument models an AWS IAM policy document
type PolicyDocument struct {
	Version   string            `json:"version,omitempty"`
	ID        string            `json:"id,omitempty"`
	Statement []PolicyStatement `json:"statement"`
}

// SimulateParams captures any additional details that should be used
// when simulating permissions.
type SimulateParams struct {
	Region string
}

const (
	// PermissionCreateBase is a base set of permissions required in all installs where the installer creates resources.
	PermissionCreateBase PermissionGroup = "create-base"

	// PermissionDeleteBase is a base set of permissions required in all installs where the installer deletes resources.
	PermissionDeleteBase PermissionGroup = "delete-base"

	// PermissionCreateNetworking is an additional set of permissions required when the installer creates networking
	// resources.
	PermissionCreateNetworking PermissionGroup = "create-networking"

	// PermissionDeleteNetworking is a set of permissions required when the installer destroys networking resources.
	PermissionDeleteNetworking PermissionGroup = "delete-networking"

	// AWS Region
	region = "eu-central-1"
)

// Permissions list of permissions required to install a cluster
var permissions = map[PermissionGroup][]string{
	// Base set of permissions required for cluster creation
	PermissionCreateBase: {
		// EC2 related perms
		"organizations:ListPolicies",
		"ec2:AllocateAddress",
		"ec2:AssociateAddress",
		"ec2:AuthorizeSecurityGroupEgress",
		"ec2:AuthorizeSecurityGroupIngress",
		"ec2:CopyImage",
		"ec2:CreateNetworkInterface",
		"ec2:AttachNetworkInterface",
		"ec2:CreateSecurityGroup",
		"ec2:CreateTags",
		"ec2:CreateVolume",
		"ec2:DeleteSecurityGroup",
		"ec2:DeleteSnapshot",
		"ec2:DeregisterImage",
		"ec2:DescribeAccountAttributes",
		"ec2:DescribeAddresses",
		"ec2:DescribeAvailabilityZones",
		"ec2:DescribeDhcpOptions",
		"ec2:DescribeImages",
		"ec2:DescribeInstanceAttribute",
		"ec2:DescribeInstanceCreditSpecifications",
		"ec2:DescribeInstances",
		"ec2:DescribeInternetGateways",
		"ec2:DescribeKeyPairs",
		"ec2:DescribeNatGateways",
		"ec2:DescribeNetworkAcls",
		"ec2:DescribeNetworkInterfaces",
		"ec2:DescribePrefixLists",
		"ec2:DescribeRegions",
		"ec2:DescribeRouteTables",
		"ec2:DescribeSecurityGroups",
		"ec2:DescribeSubnets",
		"ec2:DescribeTags",
		"ec2:DescribeVolumes",
		"ec2:DescribeVpcAttribute",
		"ec2:DescribeVpcClassicLink",
		"ec2:DescribeVpcClassicLinkDnsSupport",
		"ec2:DescribeVpcEndpoints",
		"ec2:DescribeVpcs",
		"ec2:ModifyInstanceAttribute",
		"ec2:ModifyNetworkInterfaceAttribute",
		"ec2:ReleaseAddress",
		"ec2:RevokeSecurityGroupEgress",
		"ec2:RevokeSecurityGroupIngress",
		"ec2:RunInstances",
		"ec2:TerminateInstances",

		// ELB related perms
		"elasticloadbalancing:AddTags",
		"elasticloadbalancing:ApplySecurityGroupsToLoadBalancer",
		"elasticloadbalancing:AttachLoadBalancerToSubnets",
		"elasticloadbalancing:ConfigureHealthCheck",
		"elasticloadbalancing:CreateListener",
		"elasticloadbalancing:CreateLoadBalancer",
		"elasticloadbalancing:CreateLoadBalancerListeners",
		"elasticloadbalancing:CreateTargetGroup",
		"elasticloadbalancing:DeleteLoadBalancer",
		"elasticloadbalancing:DeregisterInstancesFromLoadBalancer",
		"elasticloadbalancing:DeregisterTargets",
		"elasticloadbalancing:DescribeInstanceHealth",
		"elasticloadbalancing:DescribeListeners",
		"elasticloadbalancing:DescribeLoadBalancerAttributes",
		"elasticloadbalancing:DescribeLoadBalancers",
		"elasticloadbalancing:DescribeTags",
		"elasticloadbalancing:DescribeTargetGroupAttributes",
		"elasticloadbalancing:DescribeTargetHealth",
		"elasticloadbalancing:ModifyLoadBalancerAttributes",
		"elasticloadbalancing:ModifyTargetGroup",
		"elasticloadbalancing:ModifyTargetGroupAttributes",
		"elasticloadbalancing:RegisterInstancesWithLoadBalancer",
		"elasticloadbalancing:RegisterTargets",
		"elasticloadbalancing:SetLoadBalancerPoliciesOfListener",

		// IAM related perms
		"iam:AddRoleToInstanceProfile",
		"iam:CreateInstanceProfile",
		"iam:CreateRole",
		"iam:DeleteInstanceProfile",
		"iam:DeleteRole",
		"iam:DeleteRolePolicy",
		"iam:GetInstanceProfile",
		"iam:GetRole",
		"iam:GetRolePolicy",
		"iam:GetUser",
		"iam:ListInstanceProfilesForRole",
		"iam:ListRoles",
		"iam:ListUsers",
		"iam:PassRole",
		"iam:PutRolePolicy",
		"iam:RemoveRoleFromInstanceProfile",
		"iam:SimulatePrincipalPolicy",
		"iam:TagRole",

		// Route53 related perms
		"route53:ChangeResourceRecordSets",
		"route53:ChangeTagsForResource",
		"route53:CreateHostedZone",
		"route53:DeleteHostedZone",
		"route53:GetChange",
		"route53:GetHostedZone",
		"route53:ListHostedZones",
		"route53:ListHostedZonesByName",
		"route53:ListResourceRecordSets",
		"route53:ListTagsForResource",
		"route53:UpdateHostedZoneComment",

		// S3 related perms
		"s3:CreateBucket",
		"s3:DeleteBucket",
		"s3:GetAccelerateConfiguration",
		"s3:GetBucketCors",
		"s3:GetBucketLocation",
		"s3:GetBucketLogging",
		"s3:GetBucketObjectLockConfiguration",
		"s3:GetBucketReplication",
		"s3:GetBucketRequestPayment",
		"s3:GetBucketTagging",
		"s3:GetBucketVersioning",
		"s3:GetBucketWebsite",
		"s3:GetEncryptionConfiguration",
		"s3:GetLifecycleConfiguration",
		"s3:GetReplicationConfiguration",
		"s3:ListBucket",
		"s3:PutBucketAcl",
		"s3:PutBucketTagging",
		"s3:PutEncryptionConfiguration",

		// More S3 (would be nice to limit 'Resource' to just the bucket we actually interact with...)
		"s3:DeleteObject",
		"s3:GetObject",
		"s3:GetObjectAcl",
		"s3:GetObjectTagging",
		"s3:GetObjectVersion",
		"s3:PutObject",
		"s3:PutObjectAcl",
		"s3:PutObjectTagging",
	},
	// Permissions required for deleting base cluster resources
	PermissionDeleteBase: {
		"autoscaling:DescribeAutoScalingGroups",
		"ec2:DeleteNetworkInterface",
		"ec2:DeleteVolume",
		"elasticloadbalancing:DeleteTargetGroup",
		"elasticloadbalancing:DescribeTargetGroups",
		"iam:DeleteAccessKey",
		"iam:DeleteUser",
		"iam:ListInstanceProfiles",
		"iam:ListRolePolicies",
		"iam:ListUserPolicies",
		"s3:DeleteObject",
		"tag:GetResources",
	},
	// Permissions required for creating network resources
	PermissionCreateNetworking: {
		"ec2:AssociateDhcpOptions",
		"ec2:AssociateRouteTable",
		"ec2:AttachInternetGateway",
		"ec2:CreateDhcpOptions",
		"ec2:CreateInternetGateway",
		"ec2:CreateNatGateway",
		"ec2:CreateRoute",
		"ec2:CreateRouteTable",
		"ec2:CreateSubnet",
		"ec2:CreateVpc",
		"ec2:CreateVpcEndpoint",
		"ec2:ModifySubnetAttribute",
		"ec2:ModifyVpcAttribute",
	},
	// Permissions required for deleting network resources
	PermissionDeleteNetworking: {
		"ec2:DeleteDhcpOptions",
		"ec2:DeleteInternetGateway",
		"ec2:DeleteNatGateway",
		"ec2:DeleteRoute",
		"ec2:DeleteRouteTable",
		"ec2:DeleteSubnet",
		"ec2:DeleteVpc",
		"ec2:DeleteVpcEndpoints",
		"ec2:DetachInternetGateway",
		"ec2:DisassociateRouteTable",
		"ec2:ReplaceRouteTableAssociation",
	},
}

// checkPermissionsUsingQueryClient will use queryClient to query whether the credentials in targetClient can perform
// the actions listed in the statementEntries. queryClient will need iam:GetUser and iam:SimulatePrincipalPolicy
func checkPermissionsUsingQueryClient(queryClient, targetClient *awsClient, policyDocument PolicyDocument,
	params *SimulateParams) (bool, error) {
	// Ignoring isRoot here since we only warn the user that its not best practice to use it.
	// TODO: Add a check for isRoot in the initializer
	targetUser, _, err := getClientDetails(targetClient)
	if err != nil {
		return false, fmt.Errorf("Error gathering AWS credentials details: %v", err)
	}

	allowList := []*string{}
	for _, statement := range policyDocument.Statement {
		for _, action := range statement.Action {
			allowList = append(allowList, aws.String(action))
		}
	}

	input := &iam.SimulatePrincipalPolicyInput{
		PolicySourceArn: targetUser.Arn,
		ActionNames:     allowList,
		ContextEntries:  []*iam.ContextEntry{},
	}

	if params != nil {
		if params.Region != "" {
			input.ContextEntries = append(input.ContextEntries, &iam.ContextEntry{
				ContextKeyName:   aws.String("aws:RequestedRegion"),
				ContextKeyType:   aws.String("stringList"),
				ContextKeyValues: []*string{aws.String(params.Region)},
			})
		}
	}

	// Either all actions are allowed and we'll return 'true', or it's a failure
	allClear := true
	// Collect all failed actions
	var failedActions []string

	err = queryClient.iamClient.SimulatePrincipalPolicyPages(input,
		func(response *iam.SimulatePolicyResponse, lastPage bool) bool {
			for _, result := range response.EvaluationResults {
				if *result.EvalDecision != "allowed" {
					// Don't bail out after the first failure, so we can log the full list
					// of failed/denied actions
					failedActions = append(failedActions, *result.EvalActionName)
					allClear = false
				}
			}
			return !lastPage
		})
	if err != nil {
		return false, fmt.Errorf("Error simulating policy: %v", err)
	}

	if !allClear {
		return false, fmt.Errorf("Actions not allowed with tested credentials: %v", failedActions)
	}

	return true, nil
}

func validatePolicyDocuments(queryClient, targetClient *awsClient, policyDocuments []PolicyDocument,
	sParams *SimulateParams) (bool, error) {
	permissionsOk := true

	for _, policyDocument := range policyDocuments {
		permissionsOk, err := checkPermissionsUsingQueryClient(queryClient, targetClient, policyDocument, sParams)
		if err != nil {
			return false, err
		}
		if !permissionsOk {
			return false, fmt.Errorf("Unable to validate permissions in %s", policyDocument.ID)
		}
	}

	return permissionsOk, nil
}

// generatePolicyDocument generates an IAM policy Document from a list of actions
func generatePolicyDocument(actions []string, id *string) PolicyDocument {
	policyDocument := PolicyDocument{}

	if id != nil {
		policyDocument.ID = aws.StringValue(id)
	}

	for _, action := range actions {
		policyStatement := PolicyStatement{
			Effect:   "Allow",
			Action:   []string{action},
			Resource: []string{"*"},
		}
		policyDocument.Statement = append(policyDocument.Statement, policyStatement)
	}

	return policyDocument
}

// readSCPPolicy reads a SCP policy from file into a Policy Document
// SCP policies are structured the same as a IAM Policy Document the contain
// IAM Policy Statements
func readSCPPolicy(policyDocumentPath string) PolicyDocument {
	policyDocumentFile, err := assets.Asset(policyDocumentPath)
	if err != nil {
		fmt.Println(fmt.Errorf("Unable to load file: %s", policyDocumentPath))
	}

	policyDocument := PolicyDocument{}

	err = json.Unmarshal(policyDocumentFile, &policyDocument)
	if err != nil {
		fmt.Println(fmt.Errorf("Error unmarshalling statement: %v", err))
	}

	return policyDocument
}
