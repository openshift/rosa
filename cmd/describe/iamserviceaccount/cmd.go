/*
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
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/iamserviceaccount"
	iamServiceAccountOpts "github.com/openshift/rosa/pkg/options/iamserviceaccount"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

func NewDescribeIamServiceAccountCommand() *cobra.Command {
	cmd, options := iamServiceAccountOpts.BuildIamServiceAccountDescribeCommandWithOptions()
	cmd.Run = rosa.DefaultRunner(rosa.RuntimeWithOCMAndAWS(), DescribeIamServiceAccountRunner(options))
	return cmd
}

var Cmd = NewDescribeIamServiceAccountCommand()

func DescribeIamServiceAccountRunner(userOptions *iamServiceAccountOpts.DescribeIamServiceAccountUserOptions) rosa.CommandRunner {
	return func(ctx context.Context, r *rosa.Runtime, cmd *cobra.Command, argv []string) error {
		var err error
		clusterKey := r.GetClusterKey()

		cluster := r.FetchCluster()
		if cluster.Name() != clusterKey && cluster.ID() != clusterKey {
			cluster, err = r.OCMClient.GetCluster(clusterKey, r.Creator)
			if err != nil {
				return fmt.Errorf("failed to get cluster '%s': %s", clusterKey, err)
			}
		}

		// Validate cluster supports IAM service accounts
		if err := userOptions.ValidateCluster(cluster); err != nil {
			return err
		}

		// Validate and prompt for user inputs
		if err := userOptions.ValidateAndPromptForInputs(r, cmd); err != nil {
			return err
		}

		// Get the final role name (either provided or generated)
		roleName := userOptions.RoleName

		// Get role details
		role, attachedPolicies, inlinePolicies, err := r.AWSClient.GetServiceAccountRoleDetails(roleName)
		if err != nil {
			return fmt.Errorf("failed to get role details: %s", err)
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
				return fmt.Errorf("failed to print output: %s", err)
			}
			return nil
		}

		// Text format
		printRoleDetails(roleOutput)
		return nil
	}
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
	roleOutput := ServiceAccountRoleDetails{
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
		roleOutput.Description = aws.ToString(role.Description)
	}

	if role.PermissionsBoundary != nil && role.PermissionsBoundary.PermissionsBoundaryArn != nil {
		roleOutput.PermissionsBoundary = aws.ToString(role.PermissionsBoundary.PermissionsBoundaryArn)
	}

	// Extract information from tags
	for _, tag := range role.Tags {
		key := aws.ToString(tag.Key)
		value := aws.ToString(tag.Value)
		roleOutput.Tags[key] = value

		switch key {
		case iamserviceaccount.ClusterTagKey:
			roleOutput.Cluster = value
		case iamserviceaccount.NamespaceTagKey:
			roleOutput.Namespace = value
		case iamserviceaccount.ServiceAccountTagKey:
			roleOutput.ServiceAccount = value
		}
	}

	// Convert attached policies
	for _, policy := range attachedPolicies {
		roleOutput.AttachedPolicies = append(roleOutput.AttachedPolicies, AttachedPolicyInfo{
			PolicyName: aws.ToString(policy.PolicyName),
			PolicyArn:  aws.ToString(policy.PolicyArn),
		})
	}

	// Add OIDC provider info if available
	if trustInfo != nil {
		roleOutput.OIDCProvider = trustInfo.OIDCProvider
	}

	return roleOutput
}

func parseTrustPolicy(trustPolicyDocument string) (*TrustPolicyInfo, error) {
	trustPolicyDecoded, err := url.QueryUnescape(trustPolicyDocument)
	if err != nil {
		return nil, fmt.Errorf("failed to decode trust policy: %w", err)
	}

	info := &TrustPolicyInfo{}

	// Extract OIDC provider from the trust policy
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
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Fprintf(writer, "Name:\t%s\n", role.RoleName)
	fmt.Fprintf(writer, "ARN:\t%s\n", role.ARN)
	if role.Cluster != "" {
		fmt.Fprintf(writer, "Cluster:\t%s\n", role.Cluster)
	}
	if role.Namespace != "" {
		fmt.Fprintf(writer, "Namespace:\t%s\n", role.Namespace)
	}
	if role.ServiceAccount != "" {
		fmt.Fprintf(writer, "Service Account:\t%s\n", role.ServiceAccount)
	}
	if role.CreatedDate != nil {
		fmt.Fprintf(writer, "Created:\t%s\n", role.CreatedDate.Format("2006-01-02 15:04:05 UTC"))
	}
	fmt.Fprintf(writer, "Path:\t%s\n", role.Path)
	if role.Description != "" {
		fmt.Fprintf(writer, "Description:\t%s\n", role.Description)
	}
	if role.PermissionsBoundary != "" {
		fmt.Fprintf(writer, "Permissions Boundary:\t%s\n", role.PermissionsBoundary)
	}
	fmt.Fprintf(writer, "Max Session Duration:\t%d seconds\n", role.MaxSessionDuration)
	if role.OIDCProvider != "" {
		fmt.Fprintf(writer, "OIDC Provider:\t%s\n", role.OIDCProvider)
	}

	// Attached policies
	if len(role.AttachedPolicies) > 0 {
		fmt.Fprintf(writer, "\nAttached Policies:\t\n")
		for _, policy := range role.AttachedPolicies {
			fmt.Fprintf(writer, "\t- %s (%s)\n", policy.PolicyName, policy.PolicyArn)
		}
	}

	// Inline policies
	if len(role.InlinePolicies) > 0 {
		fmt.Fprintf(writer, "\nInline Policies:\t\n")
		for _, policyName := range role.InlinePolicies {
			fmt.Fprintf(writer, "\t- %s\n", policyName)
		}
	}

	// Tags
	if len(role.Tags) > 0 {
		fmt.Fprintf(writer, "\nTags:\t\n")
		for key, value := range role.Tags {
			fmt.Fprintf(writer, "\t%s: %s\n", key, value)
		}
	}

	// Trust policy
	fmt.Fprintf(writer, "\nTrust Policy:\t\n")
	trustPolicy := role.TrustPolicy
	if decodedPolicy, err := url.QueryUnescape(trustPolicy); err == nil {
		trustPolicy = decodedPolicy
	}
	fmt.Fprintf(writer, "%s\n", trustPolicy)

	_ = writer.Flush()
}
