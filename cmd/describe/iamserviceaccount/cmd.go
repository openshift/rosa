/*
Copyright (c) 2021 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package iamserviceaccount

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/iamserviceaccount"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	serviceAccountName string
	namespace          string
	roleName           string
}

var Cmd = &cobra.Command{
	Use:     "iamserviceaccount",
	Aliases: []string{"iam-service-account"},
	Short:   "Describe IAM role for Kubernetes service account",
	Long: "Show detailed information about an IAM role that was created for a " +
		"Kubernetes service account, including trust policy and attached permissions.",
	Example: `  # Describe by service account details
  rosa describe iamserviceaccount --cluster my-cluster \
    --name my-app \
    --namespace default

  # Describe by explicit role name
  rosa describe iamserviceaccount --cluster my-cluster \
    --role-name my-custom-role

  # Output as JSON
  rosa describe iamserviceaccount --cluster my-cluster \
    --name my-app \
    --namespace default \
    --output json`,
	Run:  run,
	Args: cobra.NoArgs,
}

func init() {
	flags := Cmd.Flags()

	ocm.AddClusterFlag(Cmd)

	flags.StringVar(
		&args.serviceAccountName,
		"name",
		"",
		"Name of the Kubernetes service account.",
	)

	flags.StringVar(
		&args.namespace,
		"namespace",
		"default",
		"Kubernetes namespace for the service account.",
	)

	flags.StringVar(
		&args.roleName,
		"role-name",
		"",
		"Name of the IAM role to describe (auto-detected if not specified).",
	)

	// Mark required flags
	_ = Cmd.MarkFlagRequired("cluster")

	interactive.AddFlag(flags)
	output.AddFlag(Cmd)
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	// Get cluster key using OCM standard method
	clusterKey, err := ocm.GetClusterKey()
	if err != nil {
		_ = r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	cluster := r.FetchCluster()
	if cluster.Name() != clusterKey && cluster.ID() != clusterKey {
		var err error
		cluster, err = r.OCMClient.GetCluster(clusterKey, r.Creator)
		if err != nil {
			_ = r.Reporter.Errorf("Failed to get cluster '%s': %s", clusterKey, err)
			os.Exit(1)
		}
	}

	// Validate cluster has STS enabled
	if cluster.AWS().STS().RoleARN() == "" {
		_ = r.Reporter.Errorf("Cluster '%s' is not an STS cluster", cluster.Name())
		os.Exit(1)
	}

	// Get role name - either explicit or derived from service account
	roleName := args.roleName
	serviceAccountName := args.serviceAccountName
	namespace := args.namespace

	if roleName == "" {
		// Need service account details to derive role name
		if interactive.Enabled() && serviceAccountName == "" {
			var err error
			serviceAccountName, err = interactive.GetString(interactive.Input{
				Question: "Service account name",
				Help:     cmd.Flags().Lookup("service-account-name").Usage,
				Required: true,
				Validators: []interactive.Validator{
					func(val interface{}) error {
						return iamserviceaccount.ValidateServiceAccountName(val.(string))
					},
				},
			})
			if err != nil {
				_ = r.Reporter.Errorf("Expected a valid service account name: %s", err)
				os.Exit(1)
			}
		}

		if serviceAccountName == "" {
			_ = r.Reporter.Errorf("Service account name is required when role name is not specified")
			os.Exit(1)
		}

		if err := iamserviceaccount.ValidateServiceAccountName(serviceAccountName); err != nil {
			_ = r.Reporter.Errorf("Invalid service account name: %s", err)
			os.Exit(1)
		}

		// Validate namespace
		if interactive.Enabled() {
			var err error
			namespace, err = interactive.GetString(interactive.Input{
				Question: "Namespace",
				Help:     cmd.Flags().Lookup("namespace").Usage,
				Default:  namespace,
				Required: true,
				Validators: []interactive.Validator{
					func(val interface{}) error {
						return iamserviceaccount.ValidateNamespaceName(val.(string))
					},
				},
			})
			if err != nil {
				_ = r.Reporter.Errorf("Expected a valid namespace: %s", err)
				os.Exit(1)
			}
		}

		if err := iamserviceaccount.ValidateNamespaceName(namespace); err != nil {
			_ = r.Reporter.Errorf("Invalid namespace: %s", err)
			os.Exit(1)
		}

		// Generate role name
		roleName = iamserviceaccount.GenerateRoleName(cluster.Name(), namespace, serviceAccountName)
	} else {
		// Using explicit role name
		if interactive.Enabled() {
			var err error
			roleName, err = interactive.GetString(interactive.Input{
				Question: "IAM role name",
				Help:     cmd.Flags().Lookup("role-name").Usage,
				Default:  roleName,
				Required: true,
			})
			if err != nil {
				_ = r.Reporter.Errorf("Expected a valid role name: %s", err)
				os.Exit(1)
			}
		}
	}

	// Get role details
	role, attachedPolicies, inlinePolicies, err := r.AWSClient.GetServiceAccountRoleDetails(roleName)
	if err != nil {
		_ = r.Reporter.Errorf("Failed to get role details: %s", err)
		os.Exit(1)
	}

	// Parse trust policy to extract OIDC information
	trustPolicyInfo, err := parseTrustPolicy(aws.ToString(role.AssumeRolePolicyDocument))
	if err != nil {
		r.Reporter.Debugf("Failed to parse trust policy: %s", err)
	}

	// Convert to output format
	roleOutput := convertRoleToOutput(role, attachedPolicies, inlinePolicies, trustPolicyInfo)

	// Output results
	if output.HasFlag() {
		err = output.Print(roleOutput)
		if err != nil {
			_ = r.Reporter.Errorf("Failed to print output: %s", err)
			os.Exit(1)
		}
		return
	}

	// Text format
	printRoleDetails(roleOutput)
}

type ServiceAccountRoleDetails struct {
	RoleName            string               `json:"roleName" yaml:"roleName"`
	ARN                 string               `json:"arn" yaml:"arn"`
	Cluster             string               `json:"cluster" yaml:"cluster"`
	Namespace           string               `json:"namespace" yaml:"namespace"`
	ServiceAccount      string               `json:"serviceAccount" yaml:"serviceAccount"`
	CreatedDate         *time.Time           `json:"createdDate,omitempty" yaml:"createdDate,omitempty"`
	Path                string               `json:"path" yaml:"path"`
	PermissionsBoundary string               `json:"permissionsBoundary,omitempty" yaml:"permissionsBoundary,omitempty"`
	Description         string               `json:"description,omitempty" yaml:"description,omitempty"`
	MaxSessionDuration  int32                `json:"maxSessionDuration" yaml:"maxSessionDuration"`
	AttachedPolicies    []AttachedPolicyInfo `json:"attachedPolicies" yaml:"attachedPolicies"`
	InlinePolicies      []string             `json:"inlinePolicies" yaml:"inlinePolicies"`
	TrustPolicy         string               `json:"trustPolicy" yaml:"trustPolicy"`
	OIDCProvider        string               `json:"oidcProvider,omitempty" yaml:"oidcProvider,omitempty"`
	Tags                map[string]string    `json:"tags" yaml:"tags"`
}

type AttachedPolicyInfo struct {
	PolicyName string `json:"policyName" yaml:"policyName"`
	PolicyArn  string `json:"policyArn" yaml:"policyArn"`
}

type TrustPolicyInfo struct {
	OIDCProvider string
}

func convertRoleToOutput(role *iamtypes.Role, attachedPolicies []iamtypes.AttachedPolicy, inlinePolicies []string, trustInfo *TrustPolicyInfo) ServiceAccountRoleDetails {
	output := ServiceAccountRoleDetails{
		RoleName:           aws.ToString(role.RoleName),
		ARN:                aws.ToString(role.Arn),
		CreatedDate:        role.CreateDate,
		Path:               aws.ToString(role.Path),
		MaxSessionDuration: aws.ToInt32(role.MaxSessionDuration),
		TrustPolicy:        aws.ToString(role.AssumeRolePolicyDocument),
		Tags:               make(map[string]string),
		AttachedPolicies:   make([]AttachedPolicyInfo, 0, len(attachedPolicies)),
		InlinePolicies:     inlinePolicies,
	}

	if role.Description != nil {
		output.Description = aws.ToString(role.Description)
	}

	if role.PermissionsBoundary != nil && role.PermissionsBoundary.PermissionsBoundaryArn != nil {
		output.PermissionsBoundary = aws.ToString(role.PermissionsBoundary.PermissionsBoundaryArn)
	}

	// Extract information from tags
	for _, tag := range role.Tags {
		key := aws.ToString(tag.Key)
		value := aws.ToString(tag.Value)
		output.Tags[key] = value

		switch key {
		case iamserviceaccount.ClusterTagKey:
			output.Cluster = value
		case iamserviceaccount.NamespaceTagKey:
			output.Namespace = value
		case iamserviceaccount.ServiceAccountTagKey:
			output.ServiceAccount = value
		}
	}

	// Convert attached policies
	for _, policy := range attachedPolicies {
		output.AttachedPolicies = append(output.AttachedPolicies, AttachedPolicyInfo{
			PolicyName: aws.ToString(policy.PolicyName),
			PolicyArn:  aws.ToString(policy.PolicyArn),
		})
	}

	// Add OIDC provider info if available
	if trustInfo != nil {
		output.OIDCProvider = trustInfo.OIDCProvider
	}

	return output
}

func parseTrustPolicy(trustPolicyDocument string) (*TrustPolicyInfo, error) {
	// This is a simple parsing approach - in production, you'd want more robust JSON parsing
	trustPolicyDecoded, err := url.QueryUnescape(trustPolicyDocument)
	if err != nil {
		return nil, fmt.Errorf("failed to decode trust policy: %w", err)
	}

	info := &TrustPolicyInfo{}

	// Extract OIDC provider from the trust policy
	// Look for patterns like "Federated": "arn:aws:iam::123456789012:oidc-provider/example.com"
	if strings.Contains(trustPolicyDecoded, "oidc-provider/") {
		parts := strings.Split(trustPolicyDecoded, "oidc-provider/")
		if len(parts) > 1 {
			providerPart := strings.Split(parts[1], "\"")[0]
			info.OIDCProvider = providerPart
		}
	}

	return info, nil
}

func printRoleDetails(role ServiceAccountRoleDetails) {
	fmt.Printf("Name:                    %s\n", role.RoleName)
	fmt.Printf("ARN:                     %s\n", role.ARN)
	if role.Cluster != "" {
		fmt.Printf("Cluster:                 %s\n", role.Cluster)
	}
	if role.Namespace != "" {
		fmt.Printf("Namespace:               %s\n", role.Namespace)
	}
	if role.ServiceAccount != "" {
		fmt.Printf("Service Account:         %s\n", role.ServiceAccount)
	}
	if role.CreatedDate != nil {
		fmt.Printf("Created:                 %s\n", role.CreatedDate.Format("2006-01-02 15:04:05 UTC"))
	}
	fmt.Printf("Path:                    %s\n", role.Path)
	if role.Description != "" {
		fmt.Printf("Description:             %s\n", role.Description)
	}
	if role.PermissionsBoundary != "" {
		fmt.Printf("Permissions Boundary:    %s\n", role.PermissionsBoundary)
	}
	fmt.Printf("Max Session Duration:    %d seconds\n", role.MaxSessionDuration)
	if role.OIDCProvider != "" {
		fmt.Printf("OIDC Provider:           %s\n", role.OIDCProvider)
	}

	fmt.Printf("\n")

	// Attached policies
	if len(role.AttachedPolicies) > 0 {
		fmt.Printf("Attached Policies:\n")
		for _, policy := range role.AttachedPolicies {
			fmt.Printf("  - %s (%s)\n", policy.PolicyName, policy.PolicyArn)
		}
		fmt.Printf("\n")
	}

	// Inline policies
	if len(role.InlinePolicies) > 0 {
		fmt.Printf("Inline Policies:\n")
		for _, policyName := range role.InlinePolicies {
			fmt.Printf("  - %s\n", policyName)
		}
		fmt.Printf("\n")
	}

	// Tags
	if len(role.Tags) > 0 {
		fmt.Printf("Tags:\n")
		for key, value := range role.Tags {
			fmt.Printf("  %s: %s\n", key, value)
		}
		fmt.Printf("\n")
	}

	// Trust policy
	fmt.Printf("Trust Policy:\n")
	// Pretty print the trust policy JSON (simplified approach)
	trustPolicy := role.TrustPolicy
	if decodedPolicy, err := url.QueryUnescape(trustPolicy); err == nil {
		trustPolicy = decodedPolicy
	}
	fmt.Printf("%s\n", trustPolicy)
}
