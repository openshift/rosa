// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package operatorroles

import (
	"context"
	"fmt"
	"os"
	"strings"

	common "github.com/openshift-online/ocm-common/pkg/aws/validations"
	iam "github.com/openshift-online/rosa-seaplane-api/iam"

	awscb "github.com/openshift/rosa/pkg/aws/commandbuilder"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/helper"
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

// operatorRoleSpec is an alias of the canonical definition type owned by the
// iam module (ROSAENG-75447), the single source of truth for V2 operator
// role definitions.
type operatorRoleSpec = iam.OperatorRoleSpec

// getHCPOperatorRoles returns the canonical operator role definitions from
// the iam module. Retained as a thin wrapper for backwards compatibility of
// internal call sites and tests.
func getHCPOperatorRoles() []operatorRoleSpec {
	return seaplane.OperatorRoleDefinitions()
}

// oidcIssuerDomainFromUrl derives the OIDC issuer domain (host + path, without the
// scheme) from the issuer URL. This matches how the AWS OIDC provider is registered
// (see HasOpenIDConnectProvider and the v1 GenerateRolePolicyDoc), so the trust
// policy always references the provider that was actually created. A naive
// scheme-strip would diverge whenever the issuer carries a query string, fragment,
// or trailing slash.
func oidcIssuerDomainFromUrl(oidcIssuerUrl string) (string, error) {
	return iam.IssuerDomain(oidcIssuerUrl)
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

	// Get operator roles to create
	operatorRoles := getHCPOperatorRoles()

	switch mode {
	case interactive.ModeAuto:
		r.Reporter.Infof("Creating %d roles using '%s'", len(operatorRoles), r.Creator.ARN)

		for _, spec := range operatorRoles {
			roleName := iam.RoleName(args.prefix, spec.Name)

			var trustPolicy string
			if spec.IsWorkerRole {
				// EC2 trust policy for worker role
				trustPolicy = iam.BuildEC2TrustPolicy()
			} else {
				// OIDC trust policy for operator roles
				var err error
				trustPolicy, err = iam.BuildWebIdentityTrustPolicy(iam.TrustPolicyInput{
					Partition: r.Creator.Partition,
					AccountID: r.Creator.AccountID,
					IssuerURL: oidcIssuerUrl,
				}, spec)
				if err != nil {
					r.Reporter.Errorf("Failed to build trust policy for role '%s': %v", roleName, err)
					seaplaneExitFn(1)
					return
				}
			}

			r.Reporter.Debugf("Creating role '%s'", roleName)

			// Create the IAM role. Tags come from the iam module; the applied
			// definition version is recorded for future drift detection (D6).
			tagsList := iam.RoleTags(args.prefix)

			roleARN, err := r.AWSClient.EnsureRole(r.Reporter, roleName, trustPolicy, args.permissionsBoundary,
				"", tagsList, "", true)
			if err != nil {
				r.Reporter.Errorf("Failed to create role '%s': %v", roleName, err)
				seaplaneExitFn(1)
				return
			}
			r.Reporter.Infof("Created role '%s' with ARN '%s'", roleName, roleARN)

			// Attach AWS managed policies
			for _, policyArn := range iam.ManagedPolicyARNs(spec, r.Creator.Partition) {
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
			roleName := iam.RoleName(args.prefix, spec.Name)

			var trustPolicy string
			if spec.IsWorkerRole {
				// EC2 trust policy
				trustPolicy = "'" + iam.BuildEC2TrustPolicy() + "'"
			} else {
				// OIDC trust policy
				policy, err := iam.BuildWebIdentityTrustPolicy(iam.TrustPolicyInput{
					Partition: r.Creator.Partition,
					AccountID: r.Creator.AccountID,
					IssuerURL: oidcIssuerUrl,
				}, spec)
				if err != nil {
					r.Reporter.Errorf("Failed to build trust policy for role '%s': %v", roleName, err)
					seaplaneExitFn(1)
					return
				}
				trustPolicy = "'" + policy + "'"
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
			for _, policyArn := range iam.ManagedPolicyARNs(spec, r.Creator.Partition) {
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
