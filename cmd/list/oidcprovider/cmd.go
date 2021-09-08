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

package oidcprovider

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var args struct {
	clusterKey string
}

var Cmd = &cobra.Command{
	Use:     "oidc-providers",
	Aliases: []string{"oidcprovider", "oidc-provider", "oidcproviders"},
	Short:   "List OIDC provider(s)",
	Long:    "List Open ID Connect Provider(s) created for ROSA clusters",
	Example: `  # List all OIDC provider(s)
  rosa list oidc-provider`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID of the cluster to list the operator roles for.",
	)

	output.AddFlag(Cmd)
}

func run(_ *cobra.Command, _ []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	// Create the AWS client:
	awsClient, err := aws.NewClient().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create AWS client: %v", err)
		os.Exit(1)
	}

	awsCreator, err := awsClient.GetCreator()
	if err != nil {
		reporter.Errorf("Failed to get AWS creator: %v", err)
		os.Exit(1)
	}

	// Create the client for the OCM API:
	ocmClient, err := ocm.NewClient().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create OCM connection: %v", err)
		os.Exit(1)
	}
	defer func() {
		err = ocmClient.Close()
		if err != nil {
			reporter.Errorf("Failed to close OCM connection: %v", err)
		}
	}()

	// Check that the cluster key (name, identifier or external identifier) given by the user
	// is reasonably safe so that there is no risk of SQL injection:
	clusterKey := args.clusterKey
	if clusterKey != "" {
		// only check for valid key, no need to check if cluster exists, as oidc provider may exist for a
		// cluster that has been removed
		if !ocm.IsValidClusterKey(clusterKey) {
			reporter.Errorf(
				"Cluster name, identifier or external identifier '%s' isn't valid: it "+
					"must contain only letters, digits, dashes and underscores",
				clusterKey,
			)
			os.Exit(1)
		}
	}

	// get all clusters
	clusters, err := ocmClient.GetClusters(awsCreator, 1000)
	if err != nil {
		reporter.Errorf("Failed to get clusters : %v", err)
	}

	oidcProviders, err := awsClient.ListOpenIDConnectProviders(clusterKey, clusters)
	if err != nil {
		reporter.Errorf("Failed to get Open ID Connect Providers: %v", err)
		os.Exit(1)
	}

	if len(oidcProviders) == 0 {
		reporter.Infof("No Open ID Connect Providers are available")
		os.Exit(0)
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(writer, "OIDC Provider\tCluster\tIn Use\n")
	for _, oidc := range oidcProviders {
		fmt.Fprintf(
			writer,
			"%s\t%s\t%s\n",
			oidc.ARN,
			oidc.ClusterID,
			oidc.InUse,
		)
	}
	writer.Flush()
}
