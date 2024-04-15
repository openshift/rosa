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

package breakglasscredential

import (
	"fmt"
	"os"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/breakglasscredential"
	"github.com/openshift/rosa/pkg/externalauthprovider"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "break-glass-credential",
	Aliases: []string{"break-glass-credentials", "breakglasscredential", "breakglasscredentials"},
	Short:   "Show details of a break glass credential on a cluster",
	Long:    "Show details of a break glass credential on a cluster.",
	Example: `  # Show details of a break glass credential with ID "12345" on a cluster named "mycluster"
  rosa describe break-glass-credential 12345 --cluster=mycluster `,
	Run:    run,
	Hidden: true,
	Args:   cobra.MaximumNArgs(2),
}

var args struct {
	id         string
	kubeconfig bool
}

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false
	ocm.AddClusterFlag(Cmd)
	output.AddFlag(Cmd)
	flags.StringVar(
		&args.id,
		"id",
		"",
		"Id for the break glass credential of the cluster to target",
	)

	flags.BoolVar(
		&args.kubeconfig,
		"kubeconfig",
		false,
		"Retrieve the kubeconfig from the break glass credential",
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
	breakGlassCredentialId := args.id
	getKubeconfig := args.kubeconfig
	// Allow the use also directly set the break glass credential id as positional parameter
	if len(argv) == 1 && !cmd.Flag("id").Changed {
		breakGlassCredentialId = argv[0]
	}
	if breakGlassCredentialId == "" {
		return fmt.Errorf("you need to specify a break glass credential id with '--id' parameter")
	}
	clusterKey := r.GetClusterKey()
	cluster := r.FetchCluster()

	externalAuthService := externalauthprovider.NewExternalAuthService(r.OCMClient)
	err := externalAuthService.IsExternalAuthProviderSupported(cluster, clusterKey)
	if err != nil {
		return err
	}

	r.Reporter.Debugf("Fetching the break glass credential '%s' for cluster '%s'", breakGlassCredentialId, clusterKey)
	if !getKubeconfig {
		r.Reporter.Infof(
			"To retrieve only the kubeconfig for this credential "+
				"use: 'rosa describe break-glass-credential %s -c %s --kubeconfig'",
			breakGlassCredentialId, clusterKey)
	}

	breakGlassCredentialConfig, err := r.OCMClient.GetBreakGlassCredential(cluster.ID(), breakGlassCredentialId)
	if err != nil {
		return err
	}

	if breakGlassCredentialConfig.Status() == cmv1.BreakGlassCredentialStatusRevoked {
		return fmt.Errorf("Break glass credential '%s' for cluster '%s' has been revoked.",
			breakGlassCredentialId, clusterKey)
	}

	if output.HasFlag() {
		var formattedOutput map[string]interface{}
		formattedOutput, err = breakglasscredential.FormatBreakGlassCredentialOutput(breakGlassCredentialConfig)
		if err != nil {
			return err
		}
		return output.Print(formattedOutput)
	}

	if getKubeconfig {
		if breakGlassCredentialConfig.Kubeconfig() == "" {
			r.Reporter.Infof("The credential is not ready yet. Please wait a few minutes for it to be fully ready.")
			return nil
		}
		fmt.Print(breakGlassCredentialConfig.Kubeconfig())
		return nil
	}

	fmt.Print(describeBreakGlassCredential(r, cluster, clusterKey, breakGlassCredentialConfig))

	return nil
}

func describeBreakGlassCredential(r *rosa.Runtime, cluster *cmv1.Cluster,
	clusterKey string, config *cmv1.BreakGlassCredential) string {
	breakGlassCredentialOutput := fmt.Sprintf("\n"+
		"ID:                                    %s\n"+
		"Username:                              %s\n"+
		"Expire at:                             %s\n"+
		"Status:                                %s\n",
		config.ID(),
		config.Username(),
		config.ExpirationTimestamp().Format("Jan _2 2006 15:04:05 MST"),
		config.Status(),
	)

	revocationTimeStamp, ok := config.GetRevocationTimestamp()
	if ok {
		breakGlassCredentialOutput = fmt.Sprintf("%s"+
			"Revoked at:                            %s\n",
			breakGlassCredentialOutput,
			revocationTimeStamp.Format("Jan _2 2006 15:04:05 MST"),
		)
	}

	return breakGlassCredentialOutput
}
