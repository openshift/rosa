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
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws/rolebridge"
	rosaerrors "github.com/openshift/rosa/pkg/errors"
	"github.com/openshift/rosa/pkg/iamserviceaccount"
	iamServiceAccountOpts "github.com/openshift/rosa/pkg/options/iamserviceaccount"
	"github.com/openshift/rosa/pkg/rosa"
)

func NewCreateIamServiceAccountCommand() *cobra.Command {
	cmd, options := iamServiceAccountOpts.BuildIamServiceAccountCreateCommandWithOptions()
	cmd.Run = rosa.DefaultRunner(rosa.RuntimeWithOCMAndAWS(), CreateIamServiceAccountRunner(options))
	return cmd
}

var Cmd = NewCreateIamServiceAccountCommand()

func CreateIamServiceAccountRunner(
	userOptions *iamServiceAccountOpts.CreateIamServiceAccountUserOptions,
) rosa.CommandRunner {
	return func(ctx context.Context, r *rosa.Runtime, cmd *cobra.Command, argv []string) (err error) {
		defer func() {
			// A validation failure means the request itself was invalid and
			// no side effects occurred, so showing usage helps the caller
			// fix their command. An operational failure (e.g. the role was
			// created but attaching a policy failed) is not a usage
			// problem. This covers every return path below, not just the
			// one from service.CreateIAMServiceAccount, so a check that
			// fails fast before the workflow runs gets the same treatment.
			var validationErr *rosaerrors.ValidationError
			if errors.As(err, &validationErr) {
				_ = cmd.Usage()
			}
		}()

		cluster := r.FetchCluster()

		// Validate cluster has STS enabled
		if cluster.AWS().STS().RoleARN() == "" {
			return fmt.Errorf("cluster '%s' is not an STS cluster", cluster.Name())
		}

		// Get AWS creator information to determine partition for FedRAMP
		creator, err := r.AWSClient.GetCreator()
		if err != nil {
			return fmt.Errorf("failed to get AWS creator information: %w", err)
		}

		// Early feedback before the OIDC provider lookup (an AWS API call);
		// see "Intentional duplication" in guidelines/workflow-conventions.md.
		// CreateIAMServiceAccountRequest.Validate() checks all of this again:
		// pkg/iamserviceaccount may eventually be called by something other
		// than this CLI, so it must not assume these checks already ran.
		if len(userOptions.ServiceAccountNames) == 0 {
			return &rosaerrors.ValidationError{
				Field: "ServiceAccounts", Message: "at least one service account name is required",
			}
		}
		if len(userOptions.PolicyArns) == 0 && userOptions.InlinePolicy == "" {
			return &rosaerrors.ValidationError{Message: "at least one policy ARN or inline policy must be specified"}
		}
		for i, policyARN := range userOptions.PolicyArns {
			if policyARN == "" {
				return &rosaerrors.ValidationError{
					Field:   "PolicyARNs",
					Message: fmt.Sprintf("policy ARN at index %d is empty", i),
				}
			}
			if _, err := arn.Parse(policyARN); err != nil {
				return &rosaerrors.ValidationError{
					Field:   "PolicyARNs",
					Message: fmt.Sprintf("policy ARN at index %d is invalid", i),
					Err:     err,
				}
			}
		}
		if userOptions.RoleName == "" && len(userOptions.ServiceAccountNames) > 1 {
			return &rosaerrors.ValidationError{
				Field:   "RoleName",
				Message: "role name is required when specifying multiple service accounts",
			}
		}

		serviceAccounts := make([]iamserviceaccount.ServiceAccountIdentifier, len(userOptions.ServiceAccountNames))
		for i, name := range userOptions.ServiceAccountNames {
			serviceAccounts[i] = iamserviceaccount.ServiceAccountIdentifier{
				Name:      name,
				Namespace: userOptions.Namespace,
			}
		}

		oidcProviderARN, err := getOIDCProviderARN(r, cluster)
		if err != nil {
			return fmt.Errorf("failed to get OIDC provider ARN: %w", err)
		}

		// nil means "no inline policy requested," which must be decided by
		// whether the flag was provided, not by what it resolves to: a
		// file:// reference that resolves to empty content is a request for
		// an (invalid) empty inline policy, not the absence of one.
		var inlinePolicy *string
		if userOptions.InlinePolicy != "" {
			resolved, err := resolveInlinePolicy(userOptions.InlinePolicy)
			if err != nil {
				return err
			}
			inlinePolicy = &resolved
		}

		req := iamserviceaccount.CreateIAMServiceAccountRequest{
			ClusterName:         cluster.Name(),
			OIDCProviderARN:     oidcProviderARN,
			ServiceAccounts:     serviceAccounts,
			RoleName:            nilIfEmpty(userOptions.RoleName),
			PolicyARNs:          userOptions.PolicyArns,
			InlinePolicy:        inlinePolicy,
			PermissionsBoundary: nilIfEmpty(userOptions.PermissionsBoundary),
			Path:                nilIfEmpty(userOptions.Path),
			AccountID:           creator.AccountID,
			Partition:           creator.Partition,
			IsGovcloud:          creator.IsGovcloud,
		}

		adapter := rolebridge.New(r.AWSClient, r.Reporter)
		service := &iamserviceaccount.Service{}
		result, err := service.CreateIAMServiceAccount(ctx, adapter, req)
		if err != nil {
			return err
		}

		r.Reporter.Infof("Created IAM role '%s' with ARN '%s' using OIDC '%s'",
			result.RoleName, result.RoleARN, oidcProviderARN)
		if result.InlinePolicyName != "" {
			r.Reporter.Infof("Attached inline policy '%s' to role '%s'", result.InlinePolicyName, result.RoleName)
		}

		return nil
	}
}

// resolveInlinePolicy returns the inline policy document, reading it from
// disk when raw is a file:// reference. raw must be non-empty.
func resolveInlinePolicy(raw string) (string, error) {
	policyPath, ok := strings.CutPrefix(raw, "file://")
	if !ok {
		return raw, nil
	}
	policyBytes, err := os.ReadFile(policyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read policy file '%s': %w", policyPath, err)
	}
	return string(policyBytes), nil
}

// nilIfEmpty converts a CLI option's zero-value string into a nil pointer,
// matching CreateIAMServiceAccountRequest's "not provided" representation
// for optional fields.
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func getOIDCProviderARN(r *rosa.Runtime, cluster *cmv1.Cluster) (string, error) {
	oidcConfigEndpointUrl, ok := cluster.AWS().STS().GetOIDCEndpointURL()
	if oidcConfigEndpointUrl == "" || !ok {
		return "", fmt.Errorf("cluster with ID '%s' does not have an OIDC configuration", cluster.ID())
	}

	providerArn, err := r.AWSClient.GetOpenIDConnectProviderByOidcEndpointUrl(oidcConfigEndpointUrl)

	if err != nil || providerArn == "" {
		return "", fmt.Errorf("no OIDC provider found for cluster with ID '%s'", cluster.ID())
	}

	return providerArn, nil
}
