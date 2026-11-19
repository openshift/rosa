package iam

import (
	"fmt"
)

// definitionVersion is the semver revision of the embedded definition set.
// Bump this (following semver) whenever a role is added, a service account
// changes, or a policy reference is updated. Overlays (D5) must carry a
// strictly greater version to be applied.
const definitionVersion = "0.1.0"

// embeddedDefinitionSet is the canonical, pinned source of truth for V2
// operator role definitions. Seeded from the rosa CLI's
// getHCPOperatorRoles() map (cmd/create/operatorroles/by_prefix_seaplane.go)
// and pkg/seaplane/roles.go; behavior must remain identical.
//
// Policy references are AWS-managed service-role policies, stored as paths
// relative to arn:<partition>:iam::aws:policy/ so definitions stay
// partition-agnostic.
var embeddedDefinitionSet = DefinitionSet{
	Version: definitionVersion,
	Roles: []OperatorRoleSpec{
		{
			Name: SuffixIngress,
			ServiceAccounts: []string{
				"system:serviceaccount:openshift-ingress-operator:ingress-operator",
			},
			ManagedPolicies: []string{"service-role/ROSAIngressOperatorPolicy"},
			Description:     "Manages AWS ELBs/NLBs for OpenShift routes",
		},
		{
			Name: SuffixCloudControllerManager,
			ServiceAccounts: []string{
				"system:serviceaccount:kube-system:kube-controller-manager",
			},
			ManagedPolicies: []string{"service-role/ROSAKubeControllerPolicy"},
			Description:     "Manages load balancers and node lifecycle",
		},
		{
			Name: SuffixEBSCSI,
			ServiceAccounts: []string{
				"system:serviceaccount:openshift-cluster-csi-drivers:aws-ebs-csi-driver-controller-sa",
			},
			ManagedPolicies: []string{"service-role/ROSAAmazonEBSCSIDriverOperatorPolicy"},
			Description:     "Creates and attaches EBS volumes",
		},
		{
			Name: SuffixImageRegistry,
			ServiceAccounts: []string{
				"system:serviceaccount:openshift-image-registry:cluster-image-registry-operator",
				"system:serviceaccount:openshift-image-registry:registry",
			},
			ManagedPolicies: []string{"service-role/ROSAImageRegistryOperatorPolicy"},
			Description:     "S3 access for the internal container image registry",
		},
		{
			Name: SuffixNetworkConfig,
			ServiceAccounts: []string{
				"system:serviceaccount:openshift-cloud-network-config-controller:cloud-credentials",
			},
			ManagedPolicies: []string{"service-role/ROSACloudNetworkConfigOperatorPolicy"},
			Description:     "Manages ENIs and cloud networking configuration",
		},
		{
			Name: SuffixControlPlaneOperator,
			ServiceAccounts: []string{
				"system:serviceaccount:kube-system:control-plane-operator",
			},
			ManagedPolicies: []string{"service-role/ROSAControlPlaneOperatorPolicy"},
			Description:     "Control plane operator managing hosted cluster lifecycle",
		},
		{
			Name: SuffixNodePoolManagement,
			ServiceAccounts: []string{
				"system:serviceaccount:kube-system:capa-controller-manager",
			},
			ManagedPolicies: []string{"service-role/ROSANodePoolManagementPolicy"},
			Description:     "Manages worker node pools",
		},
		{
			Name:            SuffixWorkerRole,
			ManagedPolicies: []string{"service-role/ROSAWorkerInstancePolicy", "AmazonSSMManagedInstanceCore"},
			Description:     "IAM role for hosted cluster worker node EC2 instances",
			IsWorkerRole:    true,
		},
	},
}

// DefaultDefinitionSet returns a copy of the embedded (pinned) definition
// set — the offline floor every client falls back to (R4).
func DefaultDefinitionSet() DefinitionSet {
	set := DefinitionSet{
		Version: embeddedDefinitionSet.Version,
		Roles:   make([]OperatorRoleSpec, len(embeddedDefinitionSet.Roles)),
	}
	copy(set.Roles, embeddedDefinitionSet.Roles)
	for i := range set.Roles {
		set.Roles[i].ServiceAccounts = append([]string(nil), set.Roles[i].ServiceAccounts...)
		set.Roles[i].ManagedPolicies = append([]string(nil), set.Roles[i].ManagedPolicies...)
	}
	return set
}

// DefinitionVersion returns the version of the embedded definition set.
func DefinitionVersion() string {
	return embeddedDefinitionSet.Version
}

// Validate checks the structural integrity of a definition set: unique
// role names, every non-worker role has at least one service account and
// policy, every worker-role rule has policies but no service accounts.
func (s DefinitionSet) Validate() error {
	if s.Version == "" {
		return fmt.Errorf("definition set has no version")
	}
	if len(s.Roles) == 0 {
		return fmt.Errorf("definition set has no roles")
	}
	seen := map[string]bool{}
	for _, role := range s.Roles {
		if role.Name == "" {
			return fmt.Errorf("definition set contains a role with no name")
		}
		if seen[role.Name] {
			return fmt.Errorf("duplicate role name %q", role.Name)
		}
		seen[role.Name] = true
		if role.IsWorkerRole {
			if len(role.ServiceAccounts) > 0 {
				return fmt.Errorf("worker role %q must not have service accounts (EC2 trust)", role.Name)
			}
		} else if len(role.ServiceAccounts) == 0 {
			return fmt.Errorf("role %q has no service accounts", role.Name)
		}
		if len(role.ManagedPolicies) == 0 {
			return fmt.Errorf("role %q has no managed policies", role.Name)
		}
	}
	return nil
}

// Role returns the spec for a role name suffix.
func (s DefinitionSet) Role(name string) (OperatorRoleSpec, bool) {
	for _, role := range s.Roles {
		if role.Name == name {
			return role, true
		}
	}
	return OperatorRoleSpec{}, false
}

// RoleNames returns role names for the given prefix, in definition order.
func (s DefinitionSet) RoleNames(prefix string) []string {
	names := make([]string, len(s.Roles))
	for i, role := range s.Roles {
		names[i] = RoleName(prefix, role.Name)
	}
	return names
}

// ValidateEmbedded runs Validate against the embedded set; called from
// tests and CI so a malformed floor can never ship.
func ValidateEmbedded() error {
	return embeddedDefinitionSet.Validate()
}
