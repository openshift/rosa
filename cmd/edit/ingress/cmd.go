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
	"github.com/spf13/pflag"

	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

// Regular expression to used to make sure that the identifier given by the
// user is safe and that it there is no risk of SQL injection:
var ingressKeyRE = regexp.MustCompile(`^[a-z0-9]{3,5}$`)

var args struct {
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

func areLocalFlagsUnchanged(cmd *cobra.Command) bool {
	parentFlagSet := cmd.Parent().Flags()
	localFlagsExcluded := []string{"cluster"}
	flagSet := cmd.Flags()
	areFlagsUnchanged := true
	flagSet.VisitAll(func(flag *pflag.Flag) {
		if parentFlagSet.Lookup(flag.Name) == nil &&
			!helper.Contains(localFlagsExcluded, flag.Name) && flagSet.Changed(flag.Name) {
			areFlagsUnchanged = false
		}
	})
	return areFlagsUnchanged
}

func init() {
	flags := Cmd.Flags()

	ocm.AddClusterFlag(Cmd)

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
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	ingressID := argv[0]
	if !ingressKeyRE.MatchString(ingressID) {
		r.Reporter.Errorf(
			"Ingress  identifier '%s' isn't valid: it must contain only letters or digits",
			ingressID,
		)
		os.Exit(1)
	}

	clusterKey := r.GetClusterKey()
	var err error
	labelMatch := args.labelMatch
	routeSelectors := make(map[string]string)

	if !interactive.Enabled() && areLocalFlagsUnchanged(cmd) {
		interactive.Enable()
	}

	if interactive.Enabled() {
		labelMatch, err = interactive.GetString(interactive.Input{
			Question: "Label match for ingress",
			Help:     cmd.Flags().Lookup("label-match").Usage,
			Default:  labelMatch,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid comma-separated list of attributes: %s", err)
			os.Exit(1)
		}
	}
	if labelMatch != "" {
		routeSelectors, err = getRouteSelector(labelMatch)
		if err != nil {
			r.Reporter.Errorf("%s", err)
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
			r.Reporter.Errorf("Expected a valid private value: %s", err)
			os.Exit(1)
		}
		private = &privArg
	}

	cluster := r.FetchCluster()
	if cluster.AWS().PrivateLink() {
		r.Reporter.Errorf("Cluster '%s' is PrivateLink and does not support updating ingresses", clusterKey)
		os.Exit(1)
	}

	// Edit API endpoint instead of ingresses
	if ingressID == "api" {
		clusterConfig := ocm.Spec{
			Private: private,
		}

		err = r.OCMClient.UpdateCluster(clusterKey, r.Creator, clusterConfig)
		if err != nil {
			r.Reporter.Errorf("Failed to update cluster API on cluster '%s': %v", clusterKey, err)
			os.Exit(1)
		}

		os.Exit(0)
	}

	// Try to find the ingress:
	r.Reporter.Debugf("Loading ingresses for cluster '%s'", clusterKey)
	ingresses, err := r.OCMClient.GetIngresses(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("Failed to get ingresses for cluster '%s': %v", clusterKey, err)
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
		r.Reporter.Errorf("Failed to get ingress '%s' for cluster '%s'", ingressID, clusterKey)
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
		r.Reporter.Errorf("Failed to create ingress for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	r.Reporter.Debugf("Updating ingress '%s' on cluster '%s'", ingress.ID(), clusterKey)
	if private == nil || len(routeSelectors) == 0 {
		r.Reporter.Warnf("No need to update ingress as there are no changes")
		os.Exit(0)
	}
	_, err = r.OCMClient.UpdateIngress(cluster.ID(), ingress)
	if err != nil {
		r.Reporter.Errorf("Failed to update ingress '%s' on cluster '%s': %s",
			ingress.ID(), clusterKey, err)
		os.Exit(1)
	}
	r.Reporter.Infof("Updated ingress '%s' on cluster '%s'", ingress.ID(), clusterKey)
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
