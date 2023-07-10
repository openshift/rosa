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
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	helper "github.com/openshift/rosa/pkg/ingress"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	private    bool
	labelMatch string
	lbType     string
}

var validLbTypes = []string{"classic", "nlb"}

var Cmd = &cobra.Command{
	Use:     "ingress",
	Aliases: []string{"route", "routes", "ingresses"},
	Short:   "Add Ingress (load balancer) to the cluster",
	Long:    "Add an Ingress to determine application access to the cluster.",
	Example: `  # Add an internal ingress to a cluster named "mycluster"
  rosa create ingress --private --cluster=mycluster

  # Add a public ingress to a cluster
  rosa create ingress --cluster=mycluster

  # Add an ingress with route selector label match
  rosa create ingress -c mycluster --label-match="foo=bar,bar=baz"

  # Add an ingress of load balancer type nlb 
  rosa create ingress --lb-type=nlb -c mycluster`,
	Run: run,
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

	flags.StringVarP(
		&args.lbType,
		"lb-type",
		"",
		"",
		fmt.Sprintf("Type of Load Balancer. Options are %s.", strings.Join(validLbTypes, `,`)),
	)
	Cmd.RegisterFlagCompletionFunc("lb-type", typeCompletion)

	interactive.AddFlag(flags)
}

func typeCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return validLbTypes, cobra.ShellCompDirectiveDefault
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	clusterKey := r.GetClusterKey()
	var err error
	labelMatch := args.labelMatch
	if interactive.Enabled() {
		labelMatch, err = interactive.GetString(interactive.Input{
			Question: "Label match for ingress",
			Help:     cmd.Flags().Lookup("label-match").Usage,
			Default:  labelMatch,
			Validators: []interactive.Validator{
				labelValidator,
			},
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid comma-separated list of attributes: %s", err)
			os.Exit(1)
		}
	}
	routeSelectors, err := helper.GetRouteSelector(labelMatch)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	cluster := r.FetchCluster()
	if cluster.AWS().PrivateLink() {
		r.Reporter.Errorf("Cluster '%s' is PrivateLink and does not support creating new ingresses", clusterKey)
		os.Exit(1)
	}

	if cluster.State() != cmv1.ClusterStateReady {
		r.Reporter.Errorf("Cluster '%s' is not yet ready", clusterKey)
		os.Exit(1)
	}

	ingressBuilder := cmv1.NewIngress()

	if cmd.Flags().Changed("private") {
		if args.private {
			ingressBuilder = ingressBuilder.Listening(cmv1.ListeningMethodInternal)
		} else {
			ingressBuilder = ingressBuilder.Listening(cmv1.ListeningMethodExternal)
		}
	} else if interactive.Enabled() {
		private, err := interactive.GetBool(interactive.Input{
			Question: "Private ingress",
			Help:     cmd.Flags().Lookup("private").Usage,
			Default:  args.private,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid private value: %s", err)
			os.Exit(1)
		}
		if private {
			ingressBuilder = ingressBuilder.Listening(cmv1.ListeningMethodInternal)
		} else {
			ingressBuilder = ingressBuilder.Listening(cmv1.ListeningMethodExternal)
		}
	}

	if len(routeSelectors) > 0 {
		ingressBuilder = ingressBuilder.RouteSelectors(routeSelectors)
	}

	var lbType *string
	if cmd.Flags().Changed("lb-type") {
		lbType = &args.lbType
	} else {
		if interactive.Enabled() {
			if lbType == nil {
				lbType = &validLbTypes[0]
			}
			lbTypeArg, err := interactive.GetOption(interactive.Input{
				Question: "Type of Load Balancer",
				Options:  validLbTypes,
				Required: true,
				Default:  *lbType,
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid Load Balancer type: %s", err)
				os.Exit(1)
			}
			lbType = &lbTypeArg
		}
	}

	if lbType != nil {
		switch *lbType {
		case "nlb":
			ingressBuilder = ingressBuilder.LoadBalancerType(cmv1.LoadBalancerFlavorNlb)
		case "classic":
			ingressBuilder = ingressBuilder.LoadBalancerType(cmv1.LoadBalancerFlavorClassic)
		default:
			r.Reporter.Errorf("Expected a valid Load Balancer type. Options are: %s", strings.Join(validLbTypes, `,`))
			os.Exit(1)
		}
	}

	ingress, err := ingressBuilder.Build()
	if err != nil {
		r.Reporter.Errorf("Failed to create ingress for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	_, err = r.OCMClient.CreateIngress(cluster.ID(), ingress)
	if err != nil {
		r.Reporter.Errorf("Failed to add ingress to cluster '%s': %s", clusterKey, err)
		os.Exit(1)
	}

	r.Reporter.Infof("Ingress has been created on cluster '%s'.", clusterKey)
	r.Reporter.Infof("To view all ingresses, run 'rosa list ingresses -c %s'", clusterKey)
}

func labelValidator(val interface{}) error {
	if labelMatch, ok := val.(string); ok {
		_, err := helper.GetRouteSelector(labelMatch)
		if err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("can only validate strings, got %v", val)
}
