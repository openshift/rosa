package operatorroles

import (
	"context"
	"fmt"
	"os"

	common "github.com/openshift-online/ocm-common/pkg/aws/validations"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"

	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/rosa"
)

// hfEnabled, hfExitFn, and hfCreateOidcConfig are package-level
// vars so tests can stub the hyperfleet dispatch path.
var (
	hfEnabled             = hyperfleet.Enabled
	hfExitFn              = func(code int) { os.Exit(code) }
	hfCreateOperatorRoles = func() {
		r := rosa.NewRuntime().WithAWS().WithHyperFleet()
		defer r.Cleanup()
		runHyperfleetCreateOperatorRoles(r)
	}
)

// getOidcConfigIssuerUrl retrieves the OIDC config issuer URL from the Platform API.
// This is used in the hyperfleet v2 flow when creating operator roles with --oidc-config-id.
func getOidcConfigIssuerUrl(r *rosa.Runtime, oidcConfigId string) (string, error) {
	if !hyperfleet.Enabled() {
		// Fall back to OCM API for v1 flow
		oidcConfig, err := r.OCMClient.GetOidcConfig(oidcConfigId)
		if err != nil {
			return "", err
		}
		return oidcConfig.IssuerUrl(), nil
	}

	// Platform API v2 flow
	ctx := context.Background()
	r.Reporter.Debugf("Retrieving OIDC config '%s' from Platform API", oidcConfigId)

	oidcConfig, err := r.HyperFleetClient.HyperfleetV1alpha1().OidcConfigs().Get(
		ctx,
		oidcConfigId,
		platform.GetOptions{},
	)
	if err != nil {
		return "", err
	}

	if oidcConfig.Spec.IssuerUrl == "" {
		r.Reporter.Errorf("OIDC config '%s' does not have an issuer URL", oidcConfigId)
		os.Exit(1)
	}

	r.Reporter.Debugf("Found issuer URL: %s", oidcConfig.Spec.IssuerUrl)
	return oidcConfig.Spec.IssuerUrl, nil
}

// operatorRoleSpec defines an operator role to be created
type operatorRoleSpec struct {
	Name              string
	ServiceAccount    string // Empty for worker role (uses EC2 trust policy)
	ManagedPolicyArns []string
	Description       string
	IsWorkerRole      bool // True for worker node role
}

// getHCPOperatorRoles returns the list of operator roles needed for a hosted cluster
func getHCPOperatorRoles() []operatorRoleSpec {
	return []operatorRoleSpec{
		{
			Name:              "ingress",
			ServiceAccount:    "system:serviceaccount:openshift-ingress-operator:ingress-operator",
			ManagedPolicyArns: []string{"arn:aws:iam::aws:policy/service-role/ROSAIngressOperatorPolicy"},
			Description:       "Manages AWS ELBs/NLBs for OpenShift routes",
		},
		{
			Name:              "cloud-controller-manager",
			ServiceAccount:    "system:serviceaccount:kube-system:kube-controller-manager",
			ManagedPolicyArns: []string{"arn:aws:iam::aws:policy/service-role/ROSAKubeControllerPolicy"},
			Description:       "Manages load balancers and node lifecycle",
		},
		{
			Name:              "ebs-csi",
			ServiceAccount:    "system:serviceaccount:openshift-cluster-csi-drivers:aws-ebs-csi-driver-controller-sa",
			ManagedPolicyArns: []string{"arn:aws:iam::aws:policy/service-role/ROSAAmazonEBSCSIDriverOperatorPolicy"},
			Description:       "Creates and attaches EBS volumes",
		},
		{
			Name:              "image-registry",
			ServiceAccount:    "system:serviceaccount:openshift-image-registry:registry",
			ManagedPolicyArns: []string{"arn:aws:iam::aws:policy/service-role/ROSAImageRegistryOperatorPolicy"},
			Description:       "S3 access for the internal container image registry",
		},
		{
			Name:              "network-config",
			ServiceAccount:    "system:serviceaccount:openshift-cloud-network-config-controller:cloud-credentials",
			ManagedPolicyArns: []string{"arn:aws:iam::aws:policy/service-role/ROSACloudNetworkConfigOperatorPolicy"},
			Description:       "Manages ENIs and cloud networking configuration",
		},
		{
			Name:              "control-plane-operator",
			ServiceAccount:    "system:serviceaccount:kube-system:control-plane-operator",
			ManagedPolicyArns: []string{"arn:aws:iam::aws:policy/service-role/ROSAControlPlaneOperatorPolicy"},
			Description:       "Control plane operator managing hosted cluster lifecycle",
		},
		{
			Name:              "node-pool-management",
			ServiceAccount:    "system:serviceaccount:kube-system:capa-controller-manager",
			ManagedPolicyArns: []string{"arn:aws:iam::aws:policy/service-role/ROSANodePoolManagementPolicy"},
			Description:       "Manages worker node pools",
		},
		{
			Name:         "ROSA-Worker-Role",
			ServiceAccount: "", // EC2 service principal, not OIDC
			ManagedPolicyArns: []string{
				"arn:aws:iam::aws:policy/service-role/ROSAWorkerInstancePolicy",
				"arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore",
			},
			Description:  "IAM role for hosted cluster worker node EC2 instances",
			IsWorkerRole: true,
		},
	}
}

// runHyperfleetCreateOperatorRoles creates operator roles for hyperfleet v2 clusters.
// Unlike v1 which uses OCM for policies, v2 uses AWS managed policies for HCP operator roles.
func runHyperfleetCreateOperatorRoles(r *rosa.Runtime) {
	// Validate required flags
	if args.prefix == "" {
		r.Reporter.Errorf("--prefix is required for operator role creation")
		hfExitFn(1)
		return
	}
	if args.oidcConfigId == "" {
		r.Reporter.Errorf("--oidc-config-id is required for operator role creation")
		hfExitFn(1)
		return
	}

	// Get mode - default to auto if not specified
	mode, err := interactive.GetMode()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		hfExitFn(1)
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
		hfExitFn(1)
		return
	}

	r.Reporter.Infof("  OIDC Issuer URL: %s", oidcIssuerUrl)
	r.Reporter.Infof("  Mode: %s", mode)

	// Get OIDC issuer domain (without https://)
	oidcIssuerDomain := oidcIssuerUrl
	if len(oidcIssuerUrl) > 8 && oidcIssuerUrl[:8] == "https://" {
		oidcIssuerDomain = oidcIssuerUrl[8:]
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
				trustPolicy = fmt.Sprintf(`{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": {
      "Service": "ec2.amazonaws.com"
    },
    "Action": "sts:AssumeRole"
  }]
}`)
			} else {
				// OIDC trust policy for operator roles
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
        "%s:sub": "%s",
        "%s:aud": "openshift"
      }
    }
  }]
}`, r.Creator.Partition, r.Creator.AccountID, oidcIssuerDomain, oidcIssuerDomain, spec.ServiceAccount, oidcIssuerDomain)
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
				hfExitFn(1)
				return
			}
			r.Reporter.Infof("Created role '%s' with ARN '%s'", roleName, roleARN)

			// Attach AWS managed policies
			for _, policyArn := range spec.ManagedPolicyArns {
				r.Reporter.Debugf("Attaching managed policy '%s' to role '%s'", policyArn, roleName)
				err = r.AWSClient.AttachRolePolicy(r.Reporter, roleName, policyArn)
				if err != nil {
					r.Reporter.Errorf("Failed to attach policy to role '%s': %v", roleName, err)
					hfExitFn(1)
					return
				}
			}

			// Create instance profile for worker role
			if spec.IsWorkerRole {
				r.Reporter.Debugf("Creating instance profile for worker role '%s'", roleName)
				err = r.AWSClient.EnsureInstanceProfile(r.Reporter, roleName, roleName, tagsList)
				if err != nil {
					r.Reporter.Errorf("Failed to create instance profile for worker role '%s': %v", roleName, err)
					hfExitFn(1)
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

		for _, spec := range operatorRoles {
			roleName := fmt.Sprintf("%s-%s", args.prefix, spec.Name)

			fmt.Printf("\n# Create %s\n", spec.Description)
			fmt.Printf("aws iam create-role --role-name %s \\\n", roleName)
			fmt.Printf("  --description '%s' \\\n", spec.Description)

			if spec.IsWorkerRole {
				// EC2 trust policy
				fmt.Printf("  --assume-role-policy-document '{\n")
				fmt.Printf("    \"Version\": \"2012-10-17\",\n")
				fmt.Printf("    \"Statement\": [{\n")
				fmt.Printf("      \"Effect\": \"Allow\",\n")
				fmt.Printf("      \"Principal\": {\n")
				fmt.Printf("        \"Service\": \"ec2.amazonaws.com\"\n")
				fmt.Printf("      },\n")
				fmt.Printf("      \"Action\": \"sts:AssumeRole\"\n")
				fmt.Printf("    }]\n")
				fmt.Printf("  }' \\\n")
			} else {
				// OIDC trust policy
				fmt.Printf("  --assume-role-policy-document '{\n")
				fmt.Printf("    \"Version\": \"2012-10-17\",\n")
				fmt.Printf("    \"Statement\": [{\n")
				fmt.Printf("      \"Effect\": \"Allow\",\n")
				fmt.Printf("      \"Principal\": {\n")
				fmt.Printf("        \"Federated\": \"arn:%s:iam::%s:oidc-provider/%s\"\n", r.Creator.Partition, r.Creator.AccountID, oidcIssuerDomain)
				fmt.Printf("      },\n")
				fmt.Printf("      \"Action\": \"sts:AssumeRoleWithWebIdentity\",\n")
				fmt.Printf("      \"Condition\": {\n")
				fmt.Printf("        \"StringEquals\": {\n")
				fmt.Printf("          \"%s:sub\": \"%s\",\n", oidcIssuerDomain, spec.ServiceAccount)
				fmt.Printf("          \"%s:aud\": \"openshift\"\n", oidcIssuerDomain)
				fmt.Printf("        }\n")
				fmt.Printf("      }\n")
				fmt.Printf("    }]\n")
				fmt.Printf("  }' \\\n")
			}

			fmt.Printf("  --tags Key=red-hat-managed,Value=true Key=rosa.hypershift,Value=true Key=rosa_managed_policies,Value=true\n")

			// Attach managed policies
			for _, policyArn := range spec.ManagedPolicyArns {
				fmt.Printf("\naws iam attach-role-policy --role-name %s \\\n", roleName)
				fmt.Printf("  --policy-arn %s\n", policyArn)
			}

			// Create instance profile for worker role
			if spec.IsWorkerRole {
				fmt.Printf("\naws iam create-instance-profile --instance-profile-name %s\n", roleName)
				fmt.Printf("aws iam add-role-to-instance-profile --instance-profile-name %s --role-name %s\n", roleName, roleName)
			}
		}

		fmt.Println()

	default:
		r.Reporter.Errorf("Invalid mode: %s", mode)
		hfExitFn(1)
	}
}
