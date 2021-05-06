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
	"fmt"
	"os"
	"regexp"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	clusterprovider "github.com/openshift/rosa/pkg/cluster"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

// Regular expression to used to make sure that the identifier given by the
// user is safe and that it there is no risk of SQL injection:
var ingressKeyRE = regexp.MustCompile(`^[a-z0-9]{3,5}$`)

var args struct {
	clusterKey string
	private    bool
	labelMatch string
}

var Cmd = &cobra.Command{
	Use:     "ingress ID",
	Aliases: []string{"route"},
	Short:   "Edit the additional cluster ingress",
	Long:    "Edit the additional non-default application router for a cluster.",
	Example: `  # Make additional ingress with ID 'a1b2' private on a cluster named 'mycluster'
  rosa edit ingress --private --cluster=mycluster a1b2

  # Update the router selectors for the additional ingress with ID 'a1b2'
  rosa edit ingress --label-match=foo=bar --cluster=mycluster a1b2

  # Update the default ingress using the sub-domain identifier
  rosa edit ingress --private=false --cluster=mycluster apps`,
	Run: run,
	Args: func(_ *cobra.Command, argv []string) error {
		if len(argv) != 1 {
			return fmt.Errorf(
				"Expected exactly one command line parameter containing the id of the ingress",
			)
		}
		return nil
	},
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID of the cluster to edit the ingress of (required).",
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

func run(cmd *cobra.Command, argv []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	ingressID := argv[0]
	if !ingressKeyRE.MatchString(ingressID) {
		reporter.Errorf(
			"Ingress  identifier '%s' isn't valid: it must contain only letters or digits",
			ingressID,
		)
		os.Exit(1)
	}

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

	labelMatch := args.labelMatch
	routeSelectors := make(map[string]string)
	var err error
	if interactive.Enabled() {
		labelMatch, err = interactive.GetString(interactive.Input{
			Question: "Label match for ingress",
			Help:     cmd.Flags().Lookup("label-match").Usage,
			Default:  labelMatch,
		})
		if err != nil {
			reporter.Errorf("Expected a valid comma-separated list of attributes: %s", err)
			os.Exit(1)
		}
	}
	if labelMatch != "" {
		routeSelectors, err = getRouteSelector(labelMatch)
		if err != nil {
			reporter.Errorf("%s", err)
			os.Exit(1)
		}
	}

	var private *bool
	if cmd.Flags().Changed("private") {
		private = &args.private
	} else if interactive.Enabled() {
		privArg, err := interactive.GetBool(interactive.Input{
			Question: "Private ingress",
			Help:     cmd.Flags().Lookup("private").Usage,
			Default:  args.private,
		})
		if err != nil {
			reporter.Errorf("Expected a valid private value: %s", err)
			os.Exit(1)
		}
		private = &privArg
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

	if cluster.AWS().PrivateLink() {
		reporter.Errorf("Cluster '%s' is PrivateLink and does not support updating ingresses", clusterKey)
		os.Exit(1)
	}

	// Edit API endpoint instead of ingresses
	if ingressID == "api" {
		clusterConfig := clusterprovider.Spec{
			Private: private,
		}

		err = clusterprovider.UpdateCluster(clustersCollection, clusterKey, awsCreator.ARN, clusterConfig)
		if err != nil {
			reporter.Errorf("Failed to update cluster API on cluster '%s': %v", clusterKey, err)
			os.Exit(1)
		}

		os.Exit(0)
	}

	// Try to find the ingress:
	reporter.Debugf("Loading ingresses for cluster '%s'", clusterKey)
	ingresses, err := ocm.GetIngresses(clustersCollection, cluster.ID())
	if err != nil {
		reporter.Errorf("Failed to get ingresses for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	var ingress *cmv1.Ingress
	for _, item := range ingresses {
		if ingressID == "apps" && item.Default() {
			ingress = item
		}
		if ingressID == "apps2" && !item.Default() {
			ingress = item
		}
		if item.ID() == ingressID {
			ingress = item
		}
	}
	if ingress == nil {
		reporter.Errorf("Failed to get ingress '%s' for cluster '%s'", ingressID, clusterKey)
		os.Exit(1)
	}

	ingressBuilder := cmv1.NewIngress().ID(ingress.ID())

	// Toggle private mode
	if private != nil {
		if *private {
			ingressBuilder = ingressBuilder.Listening(cmv1.ListeningMethodInternal)
		} else {
			ingressBuilder = ingressBuilder.Listening(cmv1.ListeningMethodExternal)
		}
	}

	// Add route selectors
	if cmd.Flags().Changed("label-match") || len(routeSelectors) > 0 {
		ingressBuilder = ingressBuilder.RouteSelectors(routeSelectors)
	}

	ingress, err = ingressBuilder.Build()
	if err != nil {
		reporter.Errorf("Failed to create ingress for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	reporter.Debugf("Updating ingress '%s' on cluster '%s'", ingress.ID(), clusterKey)
	res, err := clustersCollection.
		Cluster(cluster.ID()).
		Ingresses().
		Ingress(ingress.ID()).
		Update().
		Body(ingress).
		Send()
	if err != nil {
		reporter.Debugf(err.Error())
		reporter.Errorf("Failed to update ingress '%s' on cluster '%s': %s",
			ingress.ID(), clusterKey, res.Error().Reason())
		os.Exit(1)
	}
	reporter.Infof("Updated ingress '%s' on cluster '%s'", ingress.ID(), clusterKey)
}

func getRouteSelector(labelMatches string) (map[string]string, error) {
	routeSelectors := make(map[string]string)

	for _, labelMatch := range strings.Split(labelMatches, ",") {
		if !strings.Contains(labelMatch, "=") {
			return nil, fmt.Errorf("Expected key=value format for label-match")
		}
		tokens := strings.Split(labelMatch, "=")
		routeSelectors[strings.TrimSpace(tokens[0])] = strings.TrimSpace(tokens[1])
	}

	return routeSelectors, nil
}
