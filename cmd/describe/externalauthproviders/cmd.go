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

package externalauthproviders

import (
	"fmt"
	"os"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/externalauthproviders"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "external-auth-providers",
	Aliases: []string{"externalauthproviders"},
	Short:   "Show details of external authentication providers on a cluster",
	Long:    "Show details of external authentication providers on a cluster.",
	Example: `  # Show details of an external authentication provider named "exauth" on a cluster named "mycluster"
  rosa describe external-auth-providers --cluster=mycluster exauth`,
	Run: run,
	Args: func(_ *cobra.Command, argv []string) error {
		if len(argv) != 1 {
			return fmt.Errorf(
				"Expected exactly one command line parameter containing the name of the external authentication configuration",
			)
		}
		return nil
	},
}

func init() {
	ocm.AddClusterFlag(Cmd)
	output.AddFlag(Cmd)
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
	clusterKey := r.GetClusterKey()
	cluster := r.FetchCluster()
	if cluster.State() != cmv1.ClusterStateReady {
		r.Reporter.Errorf("Cluster '%s' is not yet ready", clusterKey)
		os.Exit(1)
	}

	err := externalauthproviders.ValidateHCPCluster(cluster)

	if err != nil {
		r.Reporter.Errorf("%v", err)
		os.Exit(1)
	}

	externalAuthId := "abc"
	r.Reporter.Debugf("Fetching external authentication providers '%s' for cluster '%s'", externalAuthId, clusterKey)
	//will use externalAuthProviders to describe
	_, exists, err := r.OCMClient.GetExternalAuthProviders(cluster.ID(), externalAuthId)

	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("external authentication provider '%s' not found", externalAuthId)
	}

	fmt.Print(describeExternalAuthProviders(r, cluster, clusterKey))
	return nil

}

func describeExternalAuthProviders(r *rosa.Runtime, cluster *cmv1.Cluster, clusterKey string) string {

	return ""

	// Prepare string
	// ignore errors as the api is not in yet
	// return fmt.Sprintf("\n"+
	// 	"ID:                                  %s\n"+
	// 	"Cluster ID:                          %s\n"+
	// 	"Name:                                %s\n"+
	// 	"Issuer Name                          %s\n"+
	// 	"Issuer Audiences                     %s\n"+
	// 	"Issuer Url                           %s\n"+
	// 	"Issuer Ca File                       %s\n"+
	// 	"Claim Mapping Groups                 %s\n"+
	// 	"Claim Mapping Username               %s\n"+
	// 	"Claim Validation Rule                %s\n"+
	// 	"Claim Validation Rule Value          %s\n",

	// 	externalAuthProviders.ID(),
	// 	cluster.ID(),
	// 	externalAuthProviders.Name(),
	// 	externalAuthProviders.issuerName(),
	// 	externalAuthProviders.issuerAudiences(),
	// 	externalAuthProviders.issuerUrl(),
	// 	externalAuthProviders.issuerCaFile(),
	// 	externalAuthProviders.claimMappingGroupsClaim(),
	// 	externalAuthProviders.claimMappingUsernameClaim(),
	// 	externalAuthProviders.claimValidationRuleClaim(),
	// 	externalAuthProviders.claimValidationRuleRequiredValue(),
	// )
}
