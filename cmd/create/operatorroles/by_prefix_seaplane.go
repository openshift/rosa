// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package operatorroles

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	common "github.com/openshift-online/ocm-common/pkg/aws/validations"

	awscb "github.com/openshift/rosa/pkg/aws/commandbuilder"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/helper"
	urlHelper "github.com/openshift/rosa/pkg/helper/url"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/seaplane"
)

// seaplaneEnabled, seaplaneExitFn, and seaplaneCreateOidcConfig are package-level
// vars so tests can stub the seaplane dispatch path.
var (
	seaplaneEnabled             = seaplane.Enabled
	seaplaneExitFn              = func(code int) { os.Exit(code) }
	seaplaneCreateOperatorRoles = func() {
		r := rosa.NewRuntime().WithSeaplane().WithAWSOnly()
		defer r.Cleanup()
		runSeaplaneCreateOperatorRoles(r)
	}
)

// getOidcConfigIssuerUrl retrieves the OIDC config issuer URL from the Platform API.
// This is used in the seaplane v2 flow when creating operator roles with --oidc-config-id.
// The config is resolved by its client-chosen metadata.name or its server-assigned UID,
// matching the Platform API resource identity model.
func getOidcConfigIssuerUrl(r *rosa.Runtime, oidcKey string) (string, error) {
	ctx := context.Background()
	r.Reporter.Debugf("Retrieving OIDC config '%s' from Platform API", oidcKey)

	oidcConfig, err := seaplane.ResolveOidcConfig(ctx, r.SeaplaneClient, oidcKey)
	if err != nil {
		return "", err
	}

	if oidcConfig.Spec.IssuerUrl == "" {
		r.Reporter.Errorf("OIDC config '%s' does not have an issuer URL", oidcKey)
		os.Exit(1)
	}

	r.Reporter.Debugf("Found issuer URL: %s", oidcConfig.Spec.IssuerUrl)
	return oidcConfig.Spec.IssuerUrl, nil
}

// operatorRoleSpec defines an operator role to be created
type operatorRoleSpec struct {
	Name              string
	ServiceAccounts   []string // Empty for worker role (uses EC2 trust policy)
	ManagedPolicyArns []string
	Description       string
	IsWorkerRole      bool // True for worker node role
}

func operatorRoleOIDCSubject(serviceAccounts []string) (string, error) {
	var subject any
	switch len(serviceAccounts) {
	case 0:
		return "", fmt.Errorf("no service accounts configured")
	case 1:
		subject = serviceAccounts[0]
	default:
		subject = serviceAccounts
	}

	encodedSubject, err := json.Marshal(subject)
	if err != nil {
		return "", err
	}
	return string(encodedSubject), nil
}

func getHCPOperatorRoles() []operatorRoleSpec {
	specs := map[string]operatorRoleSpec{
		seaplane.SuffixIngress: {
			ServiceAccounts:   []string{"system:serviceaccount:openshift-ingress-operator:ingress-operator"},
			ManagedPolicyArns: []string{"arn:aws:iam::aws:policy/service-role/ROSAIngressOperatorPolicy"},
			Description:       "Manages AWS ELBs/NLBs for OpenShift routes",
		},
		seaplane.SuffixCloudControllerManager: {
			ServiceAccounts:   []string{"system:serviceaccount:kube-system:kube-controller-manager"},
			ManagedPolicyArns: []string{"arn:aws:iam::aws:policy/service-role/ROSAKubeControllerPolicy"},
			Description:       "Manages load balancers and node lifecycle",
		},
		seaplane.SuffixEBSCSI: {
			ServiceAccounts:   []string{"system:serviceaccount:openshift-cluster-csi-drivers:aws-ebs-csi-driver-controller-sa"},
			ManagedPolicyArns: []string{"arn:aws:iam::aws:policy/service-role/ROSAAmazonEBSCSIDriverOperatorPolicy"},
			Description:       "Creates and attaches EBS volumes",
		},
		seaplane.SuffixImageRegistry: {
			ServiceAccounts: []string{
				"system:serviceaccount:openshift-image-registry:cluster-image-registry-operator",
				"system:serviceaccount:openshift-image-registry:registry",
			},
			ManagedPolicyArns: []string{"arn:aws:iam::aws:policy/service-role/ROSAImageRegistryOperatorPolicy"},
			Description:       "S3 access for the internal container image registry",
		},
		seaplane.SuffixNetworkConfig: {
			ServiceAccounts:   []string{"system:serviceaccount:openshift-cloud-network-config-controller:cloud-credentials"},
			ManagedPolicyArns: []string{"arn:aws:iam::aws:policy/service-role/ROSACloudNetworkConfigOperatorPolicy"},
			Description:       "Manages ENIs and cloud networking configuration",
		},
		seaplane.SuffixControlPlaneOperator: {
			ServiceAccounts:   []string{"system:serviceaccount:kube-system:control-plane-operator"},
			ManagedPolicyArns: []string{"arn:aws:iam::aws:policy/service-role/ROSAControlPlaneOperatorPolicy"},
			Description:       "Control plane operator managing hosted cluster lifecycle",
		},
		seaplane.SuffixNodePoolManagement: {
			ServiceAccounts:   []string{"system:serviceaccount:kube-system:capa-controller-manager"},
			ManagedPolicyArns: []string{"arn:aws:iam::aws:policy/service-role/ROSANodePoolManagementPolicy"},
			Description:       "Manages worker node pools",
		},
		seaplane.SuffixWorkerRole: {
			ManagedPolicyArns: []string{
				"arn:aws:iam::aws:policy/service-role/ROSAWorkerInstancePolicy",
				"arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore",
			},
			Description:  "IAM role for hosted cluster worker node EC2 instances",
			IsWorkerRole: true,
		},
	}

	roles := make([]operatorRoleSpec, len(seaplane.OperatorRoleSuffixes()))
	for i, suffix := range seaplane.OperatorRoleSuffixes() {
		spec, ok := specs[suffix]
		if !ok {
			panic("missing HCP operator role spec for suffix " + suffix)
		}
		spec.Name = suffix
		roles[i] = spec
	}
	return roles
}

// oidcIssuerDomainFromUrl derives the OIDC issuer domain (host + path, without the
// scheme) from the issuer URL. This matches how the AWS OIDC provider is registered
// (see HasOpenIDConnectProvider and the v1 GenerateRolePolicyDoc), so the trust
// policy always references the provider that was actually created. A naive
// scheme-strip would diverge whenever the issuer carries a query string, fragment,
// or trailing slash.
func oidcIssuerDomainFromUrl(oidcIssuerUrl string) (string, error) {
	parsedIssuerUrl, err := urlHelper.ParseRequestURI(oidcIssuerUrl)
	if err != nil {
		return "", err
	}
	if parsedIssuerUrl.Scheme != helper.ProtocolHttps {
		return "", fmt.Errorf("expected OIDC issuer URL '%s' to use an https:// scheme", oidcIssuerUrl)
	}
	return parsedIssuerUrl.Host + parsedIssuerUrl.Path, nil
}

// runSeaplaneCreateOperatorRoles creates operator roles for seaplane v2 clusters.
// Unlike v1 which uses OCM for policies, v2 uses AWS managed policies for HCP operator roles.
func runSeaplaneCreateOperatorRoles(r *rosa.Runtime) {
	// Validate required flags
	if args.prefix == "" {
		r.Reporter.Errorf("--prefix is required for operator role creation")
		seaplaneExitFn(1)
		return
	}
	if args.oidcConfigId == "" {
		r.Reporter.Errorf("--oidc-config-id is required for operator role creation")
		seaplaneExitFn(1)
		return
	}
	// A flag-like value means --oidc-config-id was passed without a value and the
	// flag parser consumed the next flag as its value. This typically happens in
	// automation when the OIDC config ID variable expands to nothing, and would
	// otherwise surface later as a confusing lookup error for an ID such as '--mode'.
	if strings.HasPrefix(args.oidcConfigId, "-") {
		r.Reporter.Errorf("Expected an OIDC config ID for --%s, got '%s'; "+
			"check that the --%s value was not omitted or empty",
			OidcConfigIdFlag, args.oidcConfigId, OidcConfigIdFlag)
		seaplaneExitFn(1)
		return
	}

	// Get mode - default to auto if not specified
	mode, err := interactive.GetMode()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		seaplaneExitFn(1)
		return
	}
	if mode == "" {
		mode = interactive.ModeAuto
	}

	r.Reporter.Infof("Creating operator roles for hosted cluster (V2) ...")
	r.Reporter.Infof("  Prefix: %s", args.prefix)
	r.Reporter.Infof("  OIDC Config ID: %s", args.oidcConfigId)

	// Get OIDC issuer URL from Platform API
	oidcIssuerUrl, err := getOidcConfigIssuerUrl(r, args.oidcConfigId)
	if err != nil {
		r.Reporter.Errorf("Failed to retrieve OIDC config: %v", err)
		r.Reporter.Infof("Run 'rosa list oidc-config' to see the available OIDC configurations")
		seaplaneExitFn(1)
		return
	}

	r.Reporter.Infof("  OIDC Issuer URL: %s", oidcIssuerUrl)
	r.Reporter.Infof("  Mode: %s", mode)

	// Get OIDC issuer domain (host + path, without the scheme)
	oidcIssuerDomain, err := oidcIssuerDomainFromUrl(oidcIssuerUrl)
	if err != nil {
		r.Reporter.Errorf("%v", err)
		seaplaneExitFn(1)
		return
	}

	// Get operator roles to create
	operatorRoles := getHCPOperatorRoles()

	switch mode {
	case interactive.ModeAuto:
		r.Reporter.Infof("Creating %d roles using '%s'", len(operatorRoles), r.Creator.ARN)

		for _, spec := range operatorRoles {
			roleName := fmt.Sprintf("%s-%s", args.prefix, spec.Name)

			var trustPolicy string
			if spec.IsWorkerRole {
				// EC2 trust policy for worker role
				trustPolicy = `{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": {
      "Service": ["ec2.amazonaws.com"]
    },
    "Action": "sts:AssumeRole"
  }]
}`
			} else {
				// OIDC trust policy for operator roles
				subject, err := operatorRoleOIDCSubject(spec.ServiceAccounts)
				if err != nil {
					r.Reporter.Errorf("Failed to build trust policy for role '%s': %v", roleName, err)
					seaplaneExitFn(1)
					return
				}
				trustPolicy = fmt.Sprintf(`{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": {
      "Federated": "arn:%s:iam::%s:oidc-provider/%s"
    },
    "Action": "sts:AssumeRoleWithWebIdentity",
    "Condition": {
      "StringEquals": {
        "%s:sub": %s,
        "%s:aud": "openshift"
      }
    }
  }]
}`,
					r.Creator.Partition,
					r.Creator.AccountID,
					oidcIssuerDomain,
					oidcIssuerDomain,
					subject,
					oidcIssuerDomain,
				)
			}

			r.Reporter.Debugf("Creating role '%s'", roleName)

			// Create the IAM role
			tagsList := map[string]string{
				tags.RedHatManaged:      helper.True,
				tags.HypershiftPolicies: helper.True,
				tags.RolePrefix:         args.prefix,
				common.ManagedPolicies:  helper.True, // Allows rosa list operator-roles to find v2 roles
			}

			roleARN, err := r.AWSClient.EnsureRole(r.Reporter, roleName, trustPolicy, args.permissionsBoundary,
				"", tagsList, "", true)
			if err != nil {
				r.Reporter.Errorf("Failed to create role '%s': %v", roleName, err)
				seaplaneExitFn(1)
				return
			}
			r.Reporter.Infof("Created role '%s' with ARN '%s'", roleName, roleARN)

			// Attach AWS managed policies
			for _, policyArn := range spec.ManagedPolicyArns {
				r.Reporter.Debugf("Attaching managed policy '%s' to role '%s'", policyArn, roleName)
				err = r.AWSClient.AttachRolePolicy(r.Reporter, roleName, policyArn)
				if err != nil {
					r.Reporter.Errorf("Failed to attach policy to role '%s': %v", roleName, err)
					seaplaneExitFn(1)
					return
				}
			}

			// Create instance profile for worker role
			if spec.IsWorkerRole {
				r.Reporter.Debugf("Creating instance profile for worker role '%s'", roleName)
				err = r.AWSClient.EnsureInstanceProfile(r.Reporter, roleName, roleName, tagsList)
				if err != nil {
					r.Reporter.Errorf("Failed to create instance profile for worker role '%s': %v", roleName, err)
					seaplaneExitFn(1)
					return
				}
			}
		}

		r.Reporter.Infof("Successfully created all operator roles and worker role")
		r.Reporter.Infof("To create a cluster with these roles, run:")
		r.Reporter.Infof("  rosa create cluster --hosted-cp --oidc-config-id %s --operator-roles-prefix %s",
			args.oidcConfigId, args.prefix)

	case interactive.ModeManual:
		r.Reporter.Infof("Run the following AWS CLI commands to create the operator roles:\n")

		commands := []string{}
		for _, spec := range operatorRoles {
			roleName := fmt.Sprintf("%s-%s", args.prefix, spec.Name)

			var trustPolicy string
			if spec.IsWorkerRole {
				// EC2 trust policy
				trustPolicy = `'{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": {
      "Service": ["ec2.amazonaws.com"]
    },
    "Action": "sts:AssumeRole"
  }]
}'`
			} else {
				// OIDC trust policy
				subject, err := operatorRoleOIDCSubject(spec.ServiceAccounts)
				if err != nil {
					r.Reporter.Errorf("Failed to build trust policy for role '%s': %v", roleName, err)
					seaplaneExitFn(1)
					return
				}
				trustPolicy = fmt.Sprintf(`'{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": {
      "Federated": "arn:%s:iam::%s:oidc-provider/%s"
    },
    "Action": "sts:AssumeRoleWithWebIdentity",
    "Condition": {
      "StringEquals": {
        "%s:sub": %s,
        "%s:aud": "openshift"
      }
    }
  }]
}'`,
					r.Creator.Partition,
					r.Creator.AccountID,
					oidcIssuerDomain,
					oidcIssuerDomain,
					subject,
					oidcIssuerDomain,
				)
			}

			// Build tags
			roleTags := map[string]string{
				tags.RedHatManaged:      helper.True,
				tags.HypershiftPolicies: helper.True,
				common.ManagedPolicies:  helper.True,
			}

			// Create role command
			commands = append(commands, fmt.Sprintf("# Create %s", spec.Description))
			createRoleCmd := awscb.NewIAMCommandBuilder().
				SetCommand(awscb.CreateRole).
				AddParam(awscb.RoleName, roleName).
				AddParam(awscb.AssumeRolePolicyDocument, trustPolicy).
				AddTags(roleTags).
				Build()
			commands = append(commands, createRoleCmd)

			// Attach managed policies
			for _, policyArn := range spec.ManagedPolicyArns {
				attachPolicyCmd := awscb.NewIAMCommandBuilder().
					SetCommand(awscb.AttachRolePolicy).
					AddParam(awscb.RoleName, roleName).
					AddParam(awscb.PolicyArn, policyArn).
					Build()
				commands = append(commands, attachPolicyCmd)
			}

			// Create instance profile for worker role
			if spec.IsWorkerRole {
				// Note: create-instance-profile and add-role-to-instance-profile are not
				// currently supported by commandbuilder, so we build them manually
				commands = append(commands,
					fmt.Sprintf("aws iam create-instance-profile --instance-profile-name %s", roleName),
					fmt.Sprintf("aws iam add-role-to-instance-profile --instance-profile-name %s --role-name %s", roleName, roleName),
				)
			}
		}

		fmt.Println(awscb.JoinCommands(commands))

	default:
		r.Reporter.Errorf("Invalid mode: %s", mode)
		seaplaneExitFn(1)
	}
}
