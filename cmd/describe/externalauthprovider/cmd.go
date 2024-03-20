/*
Copyright (c) 2024 Red Hat, Inc.

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

package externalauthprovider

import (
	"fmt"
	"os"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/externalauthprovider"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "external-auth-provider",
	Aliases: []string{"externalauthproviders", "externalauthprovider", "external-auth-providers"},
	Short:   "Show details of an external authentication provider on a cluster",
	Long:    "Show details of an external authentication provider on a cluster.",
	Example: `  # Show details of an external authentication provider named "exauth" on a cluster named "mycluster"
  rosa describe external-auth-provider exauth --cluster=mycluster `,
	Run:    run,
	Hidden: true,
	Args:   cobra.MaximumNArgs(1),
}

var args struct {
	name string
}

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false
	ocm.AddClusterFlag(Cmd)
	output.AddFlag(Cmd)
	flags.StringVar(
		&args.name,
		"name",
		"",
		"Name for the external authentication provider of the cluster to target",
	)
}

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()
	err := runWithRuntime(r, cmd, argv)
	if err != nil {
		r.Reporter.Errorf(err.Error())
		os.Exit(1)
	}
}

func runWithRuntime(r *rosa.Runtime, cmd *cobra.Command, argv []string) error {
	externalAuthId := args.name
	// Allow the use also directly set the external authentication id as positional parameter
	if len(argv) == 1 && !cmd.Flag("name").Changed {
		externalAuthId = argv[0]
	}
	if externalAuthId == "" {
		return fmt.Errorf("you need to specify an external authentication provider name with '--name' parameter")
	}
	clusterKey := r.GetClusterKey()
	cluster := r.FetchCluster()

	externalAuthService := externalauthprovider.NewExternalAuthService(r.OCMClient)
	err := externalAuthService.IsExternalAuthProviderSupported(cluster, clusterKey)
	if err != nil {
		return err
	}

	r.Reporter.Debugf("Fetching the external authentication provider '%s' for cluster '%s'", externalAuthId, clusterKey)

	externalAuthConfig, exists, err := r.OCMClient.GetExternalAuth(cluster.ID(), externalAuthId)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("external authentication provider '%s' not found", externalAuthId)
	}

	externalAuthConfigList := describeExternalAuthProviders(r, cluster, clusterKey, externalAuthConfig)

	fmt.Print(externalAuthConfigList)
	return nil
}

func describeExternalAuthProviders(r *rosa.Runtime, cluster *cmv1.Cluster,
	clusterKey string, externalAuthConfig *cmv1.ExternalAuth) string {
	externalAuthOutput := fmt.Sprintf("\n"+
		"ID:                                    %s\n"+
		"Cluster ID:                            %s\n"+
		"Issuer audiences:                      %s\n"+
		"Issuer Url:                            %s\n"+
		"Claim mappings group:                  %s\n"+
		"Claim mappings username:               %s\n",
		externalAuthConfig.ID(),
		cluster.ID(),
		formatIssuerAudiences(externalAuthConfig.Issuer().Audiences()),
		externalAuthConfig.Issuer().URL(),
		externalAuthConfig.Claim().Mappings().Groups().Claim(),
		externalAuthConfig.Claim().Mappings().UserName().Claim(),
	)

	// validation rules
	validationRules := externalAuthConfig.Claim().ValidationRules()
	validationRulesOutput := formatValidationRules(validationRules)

	if validationRulesOutput != "" {
		externalAuthOutput = fmt.Sprintf("%s"+
			"Claim validation rules:"+
			"%s",
			externalAuthOutput,
			validationRulesOutput,
		)
	}

	if externalAuthConfig.Clients() != nil {
		externalAuthOutput = fmt.Sprintf("%s"+
			"Console client id:                     %s\n",
			externalAuthOutput,
			externalAuthConfig.Clients()[0].ID(),
		)
	}

	return externalAuthOutput
}

func formatValidationRules(validationRules []*cmv1.TokenClaimValidationRule) string {
	builder := make([]string, 0)
	for _, rule := range validationRules {
		builder = append(builder, fmt.Sprintf(`
%47s%1s
%47s%1s
`,
			"- Claim:", rule.Claim(),
			"- Value:", rule.RequiredValue(),
		))
	}
	return strings.Join(builder, "")
}

func formatIssuerAudiences(audiences []string) string {
	builder := make([]string, 0)
	for _, audience := range audiences {
		builder = append(builder, fmt.Sprintf(`
%41s%s`,
			"- ", audience,
		))
	}
	return strings.Join(builder, "")
}
