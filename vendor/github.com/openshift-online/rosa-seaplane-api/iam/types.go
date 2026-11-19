// Package iam is the single source of truth for ROSA Seaplane V2 (HCP)
// operator role definitions and trust-policy templates.
//
// Consumers (the rosa CLI and terraform-provider-rhcs) import this module
// instead of maintaining per-client copies of role suffixes, service
// accounts, permission policy references, and trust-policy JSON. The
// embedded definition set (see definitions.go) is the pinned "floor";
// a future remote overlay (D5) will be able to extend it at runtime.
//
// Design doc: docs/api/manage-operator-roles-trust-policy.md (ROSAENG-66827).
package iam

// Role name suffixes (after "<prefix>-") for seaplane V2 HCP operator roles.
// Naming convention established by the Platform API IAM CloudFormation template.
const (
	SuffixIngress                = "ingress"
	SuffixCloudControllerManager = "cloud-controller-manager"
	SuffixEBSCSI                 = "ebs-csi"
	SuffixImageRegistry          = "image-registry"
	SuffixNetworkConfig          = "network-config"
	SuffixControlPlaneOperator   = "control-plane-operator"
	SuffixNodePoolManagement     = "node-pool-management"
	SuffixWorkerRole             = "ROSA-Worker-Role"
)

// Discovery tag keys applied to every operator role created from these
// definitions. Values match the rosa CLI and the V1 tagging convention so
// existing discovery flows (e.g. rosa list operator-roles) keep working.
const (
	// TagRedHatManaged marks a resource as managed by Red Hat.
	TagRedHatManaged = "red-hat-managed"
	// TagHypershiftPolicies marks roles whose permissions are AWS-managed
	// policies for HCP (rather than OCM-served inline policies).
	TagHypershiftPolicies = "rosa_hcp_policies"
	// TagRolePrefix records the user-chosen operator roles prefix.
	TagRolePrefix = "rosa_role_prefix"
	// TagManagedPolicies allows rosa list operator-roles to find V2 roles.
	TagManagedPolicies = "rosa_managed_policies"
	// TagDefinitionVersion records the definition-set revision applied to a
	// role, enabling drift detection and reconciliation (D6).
	TagDefinitionVersion = "rosa_definition_version"
)

// OperatorRoleSpec defines a single operator role (or the worker instance
// role) that clients create for a seaplane V2 hosted cluster.
type OperatorRoleSpec struct {
	// Name is the role name suffix appended to the user-chosen prefix,
	// e.g. "<prefix>-ingress".
	Name string `json:"name"`

	// ServiceAccounts are the OIDC trust subjects
	// ("system:serviceaccount:<namespace>:<name>"). Empty for the worker
	// role, which uses an EC2 service trust policy instead.
	ServiceAccounts []string `json:"serviceAccounts,omitempty"`

	// ManagedPolicies lists AWS-managed policy paths (relative to
	// arn:<partition>:iam::aws:policy/), e.g. "service-role/ROSAIngressOperatorPolicy".
	// Storing paths instead of full ARNs keeps definitions partition-agnostic;
	// use ManagedPolicyARNs to render full ARNs for a partition.
	ManagedPolicies []string `json:"managedPolicies"`

	// Description is a human-readable summary of the role's purpose.
	Description string `json:"description"`

	// IsWorkerRole is true for the worker node instance role (EC2 trust,
	// instance profile created alongside).
	IsWorkerRole bool `json:"isWorkerRole,omitempty"`

	// MinOpenShiftVersion / MaxOpenShiftVersion optionally gate the role to a
	// range of OpenShift minor versions (empty = unbounded). Preserved from
	// V1 credential-request semantics so migrated clusters on older minors
	// resolve the correct operator set (design doc, migration analysis #4).
	MinOpenShiftVersion string `json:"minOpenShiftVersion,omitempty"`
	MaxOpenShiftVersion string `json:"maxOpenShiftVersion,omitempty"`
}

// DefinitionSet is a versioned collection of operator role definitions.
// The embedded set (definitions.go) is the pinned floor; a remote overlay
// (D5) may carry a newer revision using the same schema.
type DefinitionSet struct {
	// Version is the semver revision of this set. Overlays must carry a
	// strictly greater version to take effect.
	Version string `json:"version"`

	// Roles is the ordered list of operator role specs.
	Roles []OperatorRoleSpec `json:"roles"`
}

// TrustPolicyInput parameterizes the web-identity trust policy builder.
type TrustPolicyInput struct {
	// Partition is the AWS partition ("aws", "aws-cn", "aws-us-gov").
	Partition string `json:"partition"`

	// AccountID is the customer AWS account hosting the roles.
	AccountID string `json:"accountID"`

	// IssuerURL is the OIDC issuer URL of the (managed or unmanaged) OIDC
	// config, e.g. "https://oidc.s3.us-east-1.amazonaws.com/<id>".
	IssuerURL string `json:"issuerURL"`
}

// RolesRef is the set of operator role ARNs referenced by a hosted cluster.
// Field names and JSON tags mirror hypershift's AWSRolesRef so consumers
// can convert losslessly.
type RolesRef struct {
	IngressARN              string `json:"ingressArn,omitempty"`
	KubeCloudControllerARN  string `json:"kubeCloudControllerArn,omitempty"`
	StorageARN              string `json:"storageArn,omitempty"`
	ImageRegistryARN        string `json:"imageRegistryArn,omitempty"`
	NetworkARN              string `json:"networkArn,omitempty"`
	ControlPlaneOperatorARN string `json:"controlPlaneOperatorArn,omitempty"`
	NodePoolManagementARN   string `json:"nodePoolManagementArn,omitempty"`
}
