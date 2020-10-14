/*
Copyright (c) 2020 Red Hat, Inc.

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

package ingress

import (
	"os"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/moactl/pkg/aws"
	"github.com/openshift/moactl/pkg/logging"
	"github.com/openshift/moactl/pkg/ocm"
	rprtr "github.com/openshift/moactl/pkg/reporter"
)

var args struct {
	clusterKey string
	private    bool
	labelMatch string
}

var Cmd = &cobra.Command{
	Use:     "ingress",
	Aliases: []string{"route", "routes", "ingresses"},
	Short:   "Add Ingress to cluster",
	Long:    "Add an Ingress endpoint to determine API access to the cluster.",
	Example: `  # Add an internal ingress to a cluster named "mycluster"
  moactl create ingress --private --cluster=mycluster

  # Add a public ingress to a cluster
  moactl create ingress --cluster=mycluster

  # Add an ingress with route selector label match
  moactl create ingress -c mycluster --label-match="foo=bar,bar=baz"`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID of the cluster to add the ingress to (required).",
	)
	Cmd.MarkFlagRequired("cluster")

	flags.BoolVar(
		&args.private,
		"private",
		false,
		"Restrict application route to direct, private connectivity.",
	)

	flags.StringVar(
		&args.labelMatch,
		"label-match",
		"",
		"Label match for ingress. Format should be a comma-separated list of 'key=value'. "+
			"If no label is specified, all routes will be exposed on both routers.",
	)
}

func run(cmd *cobra.Command, _ []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	// Check that the cluster key (name, identifier or external identifier) given by the user
	// is reasonably safe so that there is no risk of SQL injection:
	clusterKey := args.clusterKey
	if !ocm.IsValidClusterKey(clusterKey) {
		reporter.Errorf(
			"Cluster name, identifier or external identifier '%s' isn't valid: it "+
				"must contain only letters, digits, dashes and underscores",
			clusterKey,
		)
		os.Exit(1)
	}

	routeSelectors := make(map[string]string)
	if args.labelMatch != "" {
		for _, labelMatch := range strings.Split(args.labelMatch, ",") {
			if !strings.Contains(labelMatch, "=") {
				reporter.Errorf("Expected key=value format for label-match")
				os.Exit(1)
			}
			tokens := strings.Split(labelMatch, "=")
			routeSelectors[strings.TrimSpace(tokens[0])] = strings.TrimSpace(tokens[1])
		}
	}

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
	ocmConnection, err := ocm.NewConnection().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create OCM connection: %v", err)
		os.Exit(1)
	}
	defer func() {
		err = ocmConnection.Close()
		if err != nil {
			reporter.Errorf("Failed to close OCM connection: %v", err)
		}
	}()

	// Get the client for the OCM collection of clusters:
	clustersCollection := ocmConnection.ClustersMgmt().V1().Clusters()

	// Try to find the cluster:
	reporter.Debugf("Loading cluster '%s'", clusterKey)
	cluster, err := ocm.GetCluster(clustersCollection, clusterKey, awsCreator.ARN)
	if err != nil {
		reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	if cluster.State() != cmv1.ClusterStateReady {
		reporter.Errorf("Cluster '%s' is not yet ready", clusterKey)
		os.Exit(1)
	}

	ingressBuilder := cmv1.NewIngress()
	if cmd.Flags().Changed("private") {
		if args.private {
			ingressBuilder = ingressBuilder.Listening(cmv1.ListeningMethodInternal)
		} else {
			ingressBuilder = ingressBuilder.Listening(cmv1.ListeningMethodExternal)
		}
	}
	if len(routeSelectors) > 0 {
		ingressBuilder = ingressBuilder.RouteSelectors(routeSelectors)
	}
	ingress, err := ingressBuilder.Build()
	if err != nil {
		reporter.Errorf("Failed to create ingress for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	res, err := clustersCollection.Cluster(cluster.ID()).
		Ingresses().
		Add().
		Body(ingress).
		Send()
	if err != nil {
		reporter.Debugf(err.Error())
		reporter.Errorf("Failed to add ingress to cluster '%s': %s", clusterKey, res.Error().Reason())
		os.Exit(1)
	}
}
