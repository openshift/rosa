package externalauthproviders

import (
	"fmt"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
)

type ExternalAuthProvidersArgs struct {
	name                             string
	issuerName                       string
	issuerAudiences                  []string
	issuerUrl                        string
	issuerCaFile                     string
	claimMappingGroupsClaim          string
	claimMappingUsernameClaim        string
	claimValidationRuleClaim         string
	claimValidationRuleRequiredValue string
}

func ValidateHCPCluster(cluster *cmv1.Cluster) error {
	if !cluster.Hypershift().Enabled() {
		return fmt.Errorf(
			"External authentication provider is only supported for Hosted Control Planes.",
		)

	}
	return nil
}

func AddExternalAuthProvidersFlags(cmd *cobra.Command, prefix string) *ExternalAuthProvidersArgs {
	args := &ExternalAuthProvidersArgs{}

	cmd.Flags().StringVar(
		&args.name,
		"name",
		"",
		"Name for the set of external authentication providers.",
	)

	cmd.Flags().StringVar(
		&args.issuerName,
		"issuer-name",
		"",
		"Name of the OIDC provider.",
	)

	cmd.Flags().StringArrayVar(
		&args.issuerAudiences,
		"issuer-audiences",
		nil,
		"An array of audiences that the token was issued for.",
	)

	cmd.Flags().StringVar(
		&args.issuerUrl,
		"issuer-url",
		"",
		"The serving url of the token issuer.",
	)

	// Is this a path? - yes it's a path, will need more testing
	cmd.Flags().StringVar(
		&args.issuerCaFile,
		"issuer-ca-file",
		"",
		"TBD.",
	)

	cmd.Flags().StringVar(
		&args.claimMappingGroupsClaim,
		"claim-mapping-groups-claim",
		"",
		"Describes rules on how to transform information from an ID token into a cluster identity.",
	)

	cmd.Flags().StringVar(
		&args.claimMappingUsernameClaim,
		"claim-mapping-username-claim",
		"",
		"The name of the claim that should be used to construct usernmaes for the cluster identity.",
	)

	cmd.Flags().StringVar(
		&args.claimValidationRuleClaim,
		"claim-validation-rule-claim",
		"",
		"Rules that are applied to validate token claims to authenticate users",
	)

	cmd.Flags().StringVar(
		&args.claimValidationRuleRequiredValue,
		"claim-validation-rule-required-value",
		"",
		"The token for the ClaimValidationRules.",
	)

	return args
}
