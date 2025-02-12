package externalauthprovider

import (
	"fmt"
	"os"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	nameFlag                      = "name"
	issuerAudiencesFlag           = "issuer-audiences"
	issuerUrlFlag                 = "issuer-url"
	issuerCaFileFlag              = "issuer-ca-file"
	claimMappingGroupsClaimFlag   = "claim-mapping-groups-claim"
	claimMappingUsernameClaimFlag = "claim-mapping-username-claim"
	claimValidationRuleFlag       = "claim-validation-rule"
	consoleClientIdFlag           = "console-client-id"
	consoleClientSecretFlag       = "console-client-secret"
	defaultClaimMappingUsername   = "email"
	defaultClaimMappingGroups     = "groups"
)

type ExternalAuthServiceImpl struct {
	ocm *ocm.Client
}

type ExternalAuthService interface {
	IsExternalAuthProviderSupported(cluster *cmv1.Cluster) error
	CreateExternalAuthProvider(cluster *cmv1.Cluster, args ExternalAuthProvidersArgs) error
}

func NewExternalAuthService(ocm *ocm.Client) *ExternalAuthServiceImpl {
	return &ExternalAuthServiceImpl{
		ocm: ocm}
}

type ExternalAuthProvidersArgs struct {
	name                      string
	issuerAudiences           []string
	issuerUrl                 string
	issuerCaFile              string
	claimMappingGroupsClaim   string
	claimMappingUsernameClaim string
	claimValidationRule       []string
	consoleClientId           string
	consoleClientSecret       string
}

func (e *ExternalAuthServiceImpl) IsExternalAuthProviderSupported(cluster *cmv1.Cluster, clusterKey string) error {
	if cluster.State() != cmv1.ClusterStateReady {
		return fmt.Errorf("cluster '%s' is not yet ready", clusterKey)
	}

	err := ValidateHCPCluster(cluster)
	if err != nil {
		return err
	}

	if !cluster.ExternalAuthConfig().Enabled() {
		return fmt.Errorf("External authentication configuration is not enabled for cluster '%s'\n"+
			"Create a hosted control plane with '--external-auth-providers-enabled' parameter to enabled the configuration",
			clusterKey)
	}

	return nil
}

func (e *ExternalAuthServiceImpl) CreateExternalAuthProvider(cluster *cmv1.Cluster,
	clusterKey string,
	args *ExternalAuthProvidersArgs, r *rosa.Runtime) error {

	externalAuthConfig, err := CreateExternalAuthConfig(args)
	if err != nil {
		return fmt.Errorf("failed to create an external authentication provider for cluster '%s': %s",
			clusterKey, err)
	}

	_, err = r.OCMClient.CreateExternalAuth(cluster.ID(), externalAuthConfig)

	if err != nil {
		return fmt.Errorf("failed to create an external authentication provider for cluster '%s': %s",
			clusterKey, err)
	}
	return nil

}

func ValidateHCPCluster(cluster *cmv1.Cluster) error {
	if !cluster.Hypershift().Enabled() {
		return fmt.Errorf(
			"external authentication provider is only supported for Hosted Control Planes",
		)
	}
	return nil
}

func AddExternalAuthProvidersFlags(cmd *cobra.Command, prefix string) *ExternalAuthProvidersArgs {
	args := &ExternalAuthProvidersArgs{}

	cmd.Flags().StringVar(
		&args.name,
		nameFlag,
		"",
		"Name for the external authentication provider.",
	)

	cmd.Flags().StringSliceVar(
		&args.issuerAudiences,
		issuerAudiencesFlag,
		nil,
		"A comma-separated list of audiences that the token was issued for.",
	)

	cmd.Flags().StringVar(
		&args.issuerUrl,
		issuerUrlFlag,
		"",
		"The serving url of the token issuer.",
	)

	cmd.Flags().StringVar(
		&args.issuerCaFile,
		issuerCaFileFlag,
		"",
		"Path to certificate file to use when making requests to the issuer server.",
	)

	cmd.Flags().StringVar(
		&args.claimMappingGroupsClaim,
		claimMappingGroupsClaimFlag,
		"",
		"Describes rules on how to transform information from an ID token into a cluster identity.",
	)

	cmd.Flags().StringVar(
		&args.claimMappingUsernameClaim,
		claimMappingUsernameClaimFlag,
		"",
		"The name of the claim that should be used to construct usernames for the cluster identity.",
	)

	cmd.Flags().StringSliceVar(
		&args.claimValidationRule,
		claimValidationRuleFlag,
		nil,
		fmt.Sprintf("ClaimValidationRules are rules that are applied to validate token claims to authenticate users. "+
			"The input will be in a <claim>:<required_value> format. "+
			"To have multiple claim validation rules, you could separate the values by ','. "+
			"The input could be in a <claim>:<required_value>,<claim>:<required_value> format. "),
	)

	cmd.Flags().StringVar(
		&args.consoleClientId,
		consoleClientIdFlag,
		"",
		"The application or client id for your app registration that is used for console.",
	)

	cmd.Flags().StringVar(
		&args.consoleClientSecret,
		consoleClientSecretFlag,
		"",
		"The value of the client secret that is associated with your console app registration.",
	)

	return args
}

func GetExternalAuthOptions(
	cmd *pflag.FlagSet, prefix string, confirmBeforeAllArgs bool, externalAuthProvidersArgs *ExternalAuthProvidersArgs,
) (*ExternalAuthProvidersArgs, error) {

	var err error
	result := &ExternalAuthProvidersArgs{}

	result.name = externalAuthProvidersArgs.name
	result.issuerAudiences = externalAuthProvidersArgs.issuerAudiences
	result.issuerUrl = externalAuthProvidersArgs.issuerUrl
	result.issuerCaFile = externalAuthProvidersArgs.issuerCaFile
	result.claimMappingGroupsClaim = externalAuthProvidersArgs.claimMappingGroupsClaim
	result.claimMappingUsernameClaim = externalAuthProvidersArgs.claimMappingUsernameClaim
	result.claimValidationRule = externalAuthProvidersArgs.claimValidationRule
	result.consoleClientId = externalAuthProvidersArgs.consoleClientId
	result.consoleClientSecret = externalAuthProvidersArgs.consoleClientSecret

	issuerAudiencesSlice := externalAuthProvidersArgs.issuerAudiences
	claimValidationRuleSlice := externalAuthProvidersArgs.claimValidationRule

	if !IsExternalAuthProviderSetViaCLI(cmd, prefix) {
		if !interactive.Enabled() {
			return nil, nil
		}
	}

	if !cmd.Changed(nameFlag) {
		result.name, err = interactive.GetString(interactive.Input{
			Question: "Name",
			Default:  result.name,
			Help:     cmd.Lookup(nameFlag).Usage,
			Required: true,
		})
		if err != nil {
			return nil, err
		}

	}

	if !cmd.Changed(issuerAudiencesFlag) {
		issuerAudiencesInput, err := interactive.GetString(interactive.Input{
			Question: "Issuer audiences",
			Default:  strings.Join(issuerAudiencesSlice, ","),
			Help:     cmd.Lookup(issuerAudiencesFlag).Usage,
			Required: true,
		})
		if err != nil {
			return nil, err
		}

		issuerAudiencesSlice = helper.HandleEmptyStringOnSlice(strings.Split(issuerAudiencesInput, ","))
		result.issuerAudiences = issuerAudiencesSlice
	}

	if !cmd.Changed(issuerUrlFlag) {
		result.issuerUrl, err = interactive.GetString(interactive.Input{
			Question: "The serving url of the token issuer",
			Default:  result.issuerUrl,
			Validators: []interactive.Validator{
				interactive.IsURL,
			},
			Help:     cmd.Lookup(issuerUrlFlag).Usage,
			Required: true,
		})
		if err != nil {
			return nil, err
		}
	}

	if interactive.Enabled() && !cmd.Changed(issuerCaFileFlag) {
		result.issuerCaFile, err = interactive.GetString(interactive.Input{
			Question: "CA file path",
			Default:  result.issuerCaFile,
			Help:     cmd.Lookup(issuerCaFileFlag).Usage,
		})
		if err != nil {
			return nil, err
		}
	}

	if !cmd.Changed(claimMappingUsernameClaimFlag) {
		result.claimMappingUsernameClaim, err = interactive.GetString(interactive.Input{
			Question: "Claim mapping username",
			Default:  defaultClaimMappingUsername,
			Help:     cmd.Lookup(claimMappingUsernameClaimFlag).Usage,
			Required: true,
		})
		if err != nil {
			return nil, err
		}
	}

	if !cmd.Changed(claimMappingGroupsClaimFlag) {
		result.claimMappingGroupsClaim, err = interactive.GetString(interactive.Input{
			Question: "Claim mapping groups",
			Default:  defaultClaimMappingGroups,
			Help:     cmd.Lookup(claimMappingGroupsClaimFlag).Usage,
			Required: true,
		})
		if err != nil {
			return nil, err
		}
	}

	if interactive.Enabled() && !cmd.Changed(claimValidationRuleFlag) {
		claimValidationRuleInput, err := interactive.GetString(interactive.Input{
			Question: "Claim validation rule",
			Default:  strings.Join(claimValidationRuleSlice, ","),
			Help:     cmd.Lookup(claimValidationRuleFlag).Usage,
			Validators: []interactive.Validator{
				ocm.ValidateClaimValidationRules,
			},
		})
		if err != nil {
			return nil, err
		}
		claimValidationRuleSlice = helper.HandleEmptyStringOnSlice(strings.Split(claimValidationRuleInput, ","))
		result.claimValidationRule = claimValidationRuleSlice
	}

	if interactive.Enabled() && !cmd.Changed(consoleClientIdFlag) {
		result.consoleClientId, err = interactive.GetString(interactive.Input{
			Question: "Console client id",
			Default:  result.consoleClientId,
			Help:     cmd.Lookup(consoleClientIdFlag).Usage,
		})
		if err != nil {
			return nil, err
		}
	}

	if interactive.Enabled() && !cmd.Changed(consoleClientSecretFlag) {
		if result.consoleClientId != "" {
			// skips if no consoleClientId is provided
			result.consoleClientSecret, err = interactive.GetString(interactive.Input{
				Question: "Console client secret",
				Default:  result.consoleClientSecret,
				Help:     cmd.Lookup(consoleClientSecretFlag).Usage,
			})
			if err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

func CreateExternalAuthConfig(args *ExternalAuthProvidersArgs) (*cmv1.ExternalAuth, error) {
	externalAuthBuilder := cmv1.NewExternalAuth().ID(args.name)
	claimValidationRules := args.claimValidationRule

	// check parameters
	if args.name == "" || args.issuerUrl == "" || args.issuerAudiences == nil {
		return &cmv1.ExternalAuth{}, fmt.Errorf(
			"'--name', '--issuer-url' and '--issuer-audiences' parameters are mandatory " +
				"for creating an external authentication configuration")
	}

	tokenIssuerBuilder := cmv1.NewTokenIssuer().
		URL(args.issuerUrl).Audiences(args.issuerAudiences...)

	if args.issuerCaFile != "" {
		// Get certificate contents
		ca := ""
		if args.issuerCaFile != "" {
			cert, err := os.ReadFile(args.issuerCaFile)
			if err != nil {
				return &cmv1.ExternalAuth{}, fmt.Errorf("expected a valid certificate bundle: %s", err)
			}
			ca = string(cert)
		}
		// Set the CA file, if any
		if ca != "" {
			tokenIssuerBuilder.CA(ca)
		}
	}
	externalAuthBuilder.Issuer(tokenIssuerBuilder)

	if args.claimMappingGroupsClaim != "" || args.claimMappingUsernameClaim != "" || claimValidationRules != nil {
		groupClaimBuilder := cmv1.NewGroupsClaim().Claim(args.claimMappingGroupsClaim)
		usernameClaimBuilder := cmv1.NewUsernameClaim().Claim(args.claimMappingUsernameClaim)

		tokenClaimMappingsBuilder := cmv1.NewTokenClaimMappings().Groups(groupClaimBuilder).
			UserName(usernameClaimBuilder)
		claimBuilder := cmv1.NewExternalAuthClaim().Mappings(tokenClaimMappingsBuilder)

		if claimValidationRules != nil {
			var builders []*cmv1.TokenClaimValidationRuleBuilder
			for _, rule := range claimValidationRules {
				claimValidationRulesBuilder := cmv1.NewTokenClaimValidationRule()
				claimValidationRule := helper.HandleEmptyStringOnSlice(strings.Split(rule, ":"))
				if len(claimValidationRule) == 2 {
					claim := claimValidationRule[0]
					requiredValue := claimValidationRule[1]
					claimValidationRulesBuilder.Claim(claim).RequiredValue(requiredValue)
					builders = append(builders, claimValidationRulesBuilder)
				}
			}
			claimBuilder.ValidationRules(builders...)
		}
		externalAuthBuilder.Claim(claimBuilder)

	}

	if args.consoleClientId != "" || args.consoleClientSecret != "" {
		clientBuilder := cmv1.NewExternalAuthClientConfig().
			ID(args.consoleClientId).Secret(args.consoleClientSecret).Component(
			// Component will be "fixed" with a "constant" component for the openshift console
			cmv1.NewClientComponent().Name("console").Namespace("openshift-console"))
		externalAuthBuilder.Clients(clientBuilder)
	}

	externalAuthConfig, err := externalAuthBuilder.Build()

	if err != nil {
		return &cmv1.ExternalAuth{}, err
	}

	return externalAuthConfig, nil
}

func IsExternalAuthProviderSetViaCLI(cmd *pflag.FlagSet, prefix string) bool {
	for _, parameter := range []string{nameFlag, issuerAudiencesFlag, issuerUrlFlag,
		issuerCaFileFlag, claimMappingGroupsClaimFlag, claimMappingUsernameClaimFlag,
		claimValidationRuleFlag, consoleClientIdFlag, consoleClientSecretFlag} {

		if cmd.Changed(fmt.Sprintf("%s%s", prefix, parameter)) {
			return true
		}
	}

	return false
}
