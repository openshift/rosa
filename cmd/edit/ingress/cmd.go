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
	"reflect"
	"regexp"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	utils "github.com/openshift/rosa/pkg/helper"
	helper "github.com/openshift/rosa/pkg/ingress"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/interactive/consts"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

// Regular expression to used to make sure that the identifier given by the
// user is safe and that it there is no risk of SQL injection:
var ingressKeyRE = regexp.MustCompile(`^[a-z0-9]{3,5}$`)

var validLbTypes = []string{string(cmv1.LoadBalancerFlavorClassic), string(cmv1.LoadBalancerFlavorNlb)}

var Cmd = &cobra.Command{
	Use:     "ingress ID",
	Aliases: []string{"route"},
	Short:   "Edit a cluster ingress (load balancer)",
	Long:    "Edit a cluster ingress for a cluster.",
	Example: `  # Make additional ingress with ID 'a1b2' private on a cluster named 'mycluster'
  rosa edit ingress --private --cluster=mycluster a1b2

  # Update the router selectors for the additional ingress with ID 'a1b2'
  rosa edit ingress --label-match=foo=bar --cluster=mycluster a1b2

  # Update the default ingress using the sub-domain identifier
  rosa edit ingress --private=false --cluster=mycluster apps

  # Update the load balancer type of the apps2 ingress 
  rosa edit ingress --lb-type=nlb --cluster=mycluster apps2`,
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

func shouldEnableInteractive(flagSet *pflag.FlagSet, params []string) bool {
	unchanged := true
	for _, s := range params {
		unchanged = unchanged && !flagSet.Changed(s)
	}
	return unchanged
}

var args struct {
	private       bool
	routeSelector string
	lbType        string

	excludedNamespaces        string
	wildcardPolicy            string
	namespaceOwnershipPolicy  string
	clusterRoutesHostname     string
	clusterRoutesTlsSecretRef string

	componentRoutes string
}

const (
	ingressV2DocLink = "https://access.redhat.com/articles/7028653"
)

func init() {
	flags := Cmd.Flags()

	ocm.AddClusterFlag(Cmd)

	flags.BoolVar(
		&args.private,
		privateFlag,
		false,
		"Restrict application route to direct, private connectivity.",
	)

	flags.StringVar(
		&args.routeSelector,
		labelMatchFlag,
		"",
		fmt.Sprintf("Alias to '%s' flag.", routeSelectorFlag),
	)

	flags.StringVar(
		&args.routeSelector,
		routeSelectorFlag,
		"",
		"Route Selector for ingress. Format should be a comma-separated list of 'key=value'. "+
			"If no label is specified, all routes will be exposed on both routers."+
			" For legacy ingress support these are inclusion labels, otherwise they are treated as exclusion label.",
	)

	flags.StringVar(
		&args.lbType,
		lbTypeFlag,
		"",
		fmt.Sprintf("Type of Load Balancer. Options are %s.", strings.Join(validLbTypes, ",")),
	)

	addIngressV2Flags(flags)

	Cmd.RegisterFlagCompletionFunc(lbTypeFlag, lbTypeCompletion)
	Cmd.RegisterFlagCompletionFunc(wildcardPolicyFlag, wildcardPoliciesTypeCompletion)
	Cmd.RegisterFlagCompletionFunc(namespaceOwnershipPolicyFlag, namespaceOwnershipPoliciesTypeCompletion)
}

// TODO: Generalize this functionality for type completion
func lbTypeCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return validLbTypes, cobra.ShellCompDirectiveDefault
}

func namespaceOwnershipPoliciesTypeCompletion(cmd *cobra.Command,
	args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return helper.ValidNamespaceOwnershipPolicies, cobra.ShellCompDirectiveDefault
}

func wildcardPoliciesTypeCompletion(cmd *cobra.Command,
	args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return helper.ValidWildcardPolicies, cobra.ShellCompDirectiveDefault
}

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	ingressKey := argv[0]
	if !ingressKeyRE.MatchString(ingressKey) {
		r.Reporter.Errorf(
			"Ingress identifier '%s' isn't valid: it must contain between three and five lowercase letters or digits",
			ingressKey,
		)
		os.Exit(1)
	}

	clusterKey := r.GetClusterKey()
	cluster := r.FetchCluster()

	if !interactive.Enabled() && shouldEnableInteractive(cmd.Flags(),
		[]string{labelMatchFlag, privateFlag, lbTypeFlag, routeSelectorFlag, excludedNamespacesFlag, wildcardPolicyFlag,
			namespaceOwnershipPolicyFlag, clusterRoutesHostnameFlag, clusterRoutesTlsSecretRefFlag, componentRoutesFlag}) {
		interactive.Enable()
	}

	hasLegacyIngressSupport := true
	isHypershift := ocm.IsHyperShiftCluster(cluster)
	if !isHypershift {
		var err error
		hasLegacyIngressSupport, err = r.OCMClient.HasLegacyIngressSupport(cluster)
		if err != nil {
			r.Reporter.Errorf("There was a problem checking version compatibility: %v", err)
			os.Exit(1)
		}
	}

	if IsIngressV2SetViaCLI(cmd.Flags()) {
		if isHypershift {
			r.Reporter.Errorf(
				"New ingress attributes %s can't be supplied for Hosted Control Plane clusters",
				utils.SliceToSortedString(exclusivelyIngressV2Flags),
			)
			os.Exit(1)
		} else if hasLegacyIngressSupport {
			r.Reporter.Errorf("New ingress attributes %s can't be supplied for legacy supported clusters."+
				" For more information on how to be supported please check: %s",
				utils.SliceToSortedString(exclusivelyIngressV2Flags), ingressV2DocLink)
			os.Exit(1)
		}
	}

	if cluster.AWS().PrivateLink() && !ocm.IsHyperShiftCluster(cluster) && hasLegacyIngressSupport {
		r.Reporter.Errorf(
			"Classic cluster '%s' is PrivateLink on legacy ingress support and does not allow updating ingresses",
			clusterKey)
		os.Exit(1)
	}

	var private *bool
	if cmd.Flags().Changed(privateFlag) {
		private = &args.private
	} else if interactive.Enabled() {
		privArg, err := interactive.GetBool(interactive.Input{
			Question: "Private ingress",
			Help:     cmd.Flags().Lookup(privateFlag).Usage,
			Default:  args.private,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid private value: %s", err)
			os.Exit(1)
		}
		private = &privArg
	}
	// Edit API endpoint instead of ingresses
	if ingressKey == "api" {
		clusterConfig := ocm.Spec{
			Private: private,
		}

		err := r.OCMClient.UpdateCluster(clusterKey, r.Creator, clusterConfig)
		if err != nil {
			r.Reporter.Errorf("Failed to update cluster API on cluster '%s': %v", clusterKey, err)
			os.Exit(1)
		}
		r.Reporter.Infof("Updated ingress '%s' on cluster '%s'", ingressKey, clusterKey)
		os.Exit(0)
	}

	ingress, err := r.OCMClient.GetIngress(cluster.ID(), ingressKey)
	if err != nil {
		r.Reporter.Errorf("Failed to fetch ingress: %v", err)
		os.Exit(1)
	}

	var routeSelector *string
	if cmd.Flags().Changed(routeSelectorFlag) || cmd.Flags().Changed(labelMatchFlag) {
		if ocm.IsHyperShiftCluster(cluster) {
			r.Reporter.Errorf("Updating route selectors is not supported for Hosted Control Plane clusters")
			os.Exit(1)
		}
		if ingress.Default() && hasLegacyIngressSupport {
			r.Reporter.Errorf("Updating route selectors for default ingress is not allowed for legacy ingress support")
			os.Exit(1)
		}
		routeSelector = &args.routeSelector
	} else if interactive.Enabled() && !ocm.IsHyperShiftCluster(cluster) &&
		(ingress.Default() && !hasLegacyIngressSupport || !ingress.Default()) {
		routeSelectorArg, err := interactive.GetString(interactive.Input{
			Question: "Route Selector for ingress",
			Help:     cmd.Flags().Lookup(routeSelectorFlag).Usage,
			Default:  args.routeSelector,
			Validators: []interactive.Validator{
				func(routeSelector interface{}) error {
					_, err := helper.GetRouteSelector(routeSelector.(string))
					return err
				},
			},
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid comma-separated list of attributes: %s", err)
			os.Exit(1)
		}
		routeSelector = &routeSelectorArg
	}

	var lbType *string
	if cmd.Flags().Changed(lbTypeFlag) {
		if ocm.IsHyperShiftCluster(cluster) {
			r.Reporter.Errorf("Updating Load Balancer Type is not supported for Hosted Control Plane clusters")
			os.Exit(1)
		}
		if ocm.IsSts(cluster) && hasLegacyIngressSupport {
			r.Reporter.Errorf("Updating Load Balancer Type is not supported for STS clusters on legacy ingress support")
			os.Exit(1)
		}
		lbType = &args.lbType
	} else if interactive.Enabled() && (!ocm.IsHyperShiftCluster(cluster) &&
		(!ocm.IsSts(cluster) || !hasLegacyIngressSupport)) {
		if lbType == nil {
			curLb := string(ingress.LoadBalancerType())
			lbType = &curLb
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

	var excludedNamespaces *string
	var wildcardPolicy *string
	var namespaceOwnershipPolicy *string
	var componentRoutes map[string]*cmv1.ComponentRouteBuilder
	if !hasLegacyIngressSupport {
		isInteractiveEnabledAndNotHcp := interactive.Enabled() && !ocm.IsHyperShiftCluster(cluster)
		if cmd.Flags().Changed(excludedNamespacesFlag) {
			if ocm.IsHyperShiftCluster(cluster) {
				r.Reporter.Errorf("Updating excluded namespace is not supported for Hosted Control Plane clusters")
				os.Exit(1)
			}
			excludedNamespaces = &args.excludedNamespaces
		} else if isInteractiveEnabledAndNotHcp {
			excludedNamespacesArg, err := interactive.GetString(interactive.Input{
				Question: "Excluded namespaces for ingress",
				Help:     cmd.Flags().Lookup(excludedNamespacesFlag).Usage,
				Default:  args.excludedNamespaces,
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid comma-separated list of attributes: %s", err)
				os.Exit(1)
			}
			excludedNamespaces = &excludedNamespacesArg
		}
		if cmd.Flags().Changed(wildcardPolicyFlag) {
			if ocm.IsHyperShiftCluster(cluster) {
				r.Reporter.Errorf("Updating Wildcard Policy is not supported for Hosted Control Plane clusters")
				os.Exit(1)
			}
			wildcardPolicy = &args.wildcardPolicy
		} else if isInteractiveEnabledAndNotHcp {
			wildcardPolicyArg, err := interactive.GetOption(interactive.Input{
				Question: "Wildcard Policy",
				Options:  helper.ValidWildcardPolicies,
				Help:     cmd.Flags().Lookup(wildcardPolicyFlag).Usage,
				Default:  args.wildcardPolicy,
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid Wildcard Policy: %s", err)
				os.Exit(1)
			}
			wildcardPolicy = &wildcardPolicyArg
		}
		if cmd.Flags().Changed(namespaceOwnershipPolicyFlag) {
			if ocm.IsHyperShiftCluster(cluster) {
				r.Reporter.Errorf(
					"Updating Namespace Ownership Policy is not supported for Hosted Control Plane clusters",
				)
				os.Exit(1)
			}
			namespaceOwnershipPolicy = &args.namespaceOwnershipPolicy
		} else if isInteractiveEnabledAndNotHcp {
			namespaceOwnershipPolicyArg, err := interactive.GetOption(interactive.Input{
				Question: "Namespace Ownership Policy",
				Options:  helper.ValidNamespaceOwnershipPolicies,
				Help:     cmd.Flags().Lookup(namespaceOwnershipPolicyFlag).Usage,
				Default:  args.namespaceOwnershipPolicy,
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid Namespace Ownership Policy: %s", err)
				os.Exit(1)
			}
			namespaceOwnershipPolicy = &namespaceOwnershipPolicyArg
		}

		if cmd.Flags().Changed(componentRoutesFlag) {
			if ocm.IsHyperShiftCluster(cluster) {
				r.Reporter.Errorf(
					"Updating Cluster Component Routes is not supported for Hosted Control Plane clusters",
				)
				os.Exit(1)
			}
			componentRoutes, err = parseComponentRoutes(args.componentRoutes)
			if err != nil {
				r.Reporter.Errorf("An error occurred whilst parsing the supplied component routes: %s", err)
				os.Exit(1)
			}
		} else if isInteractiveEnabledAndNotHcp {
			componentRoutes = map[string]*cmv1.ComponentRouteBuilder{}
			if confirm.Prompt(false, "Would you like to edit the component routes?") {
				for _, componentRoute := range expectedComponentRoutes {
					componentRouteBuilder := cmv1.NewComponentRoute()
					for _, parameterName := range expectedParameters {
						defaultValue := ""
						// TODO: use reflection, couldn't get it to work
						if parameterName == hostnameParameter {
							defaultValue = ingress.ComponentRoutes()[componentRoute].Hostname()
						} else if parameterName == tlsSecretRefParameter {
							defaultValue = ingress.ComponentRoutes()[componentRoute].TlsSecretRef()
						}
						parameterValue, err := interactive.GetString(interactive.Input{
							Question: fmt.Sprintf("%s route %s", componentRoute, parameterName),
							Default:  defaultValue,
						})
						if err != nil {
							r.Reporter.Errorf("Expected a valid component route '%s': %s", parameterName, err)
							os.Exit(1)
						}
						// TODO: use reflection, couldn't get it to work
						if parameterName == hostnameParameter {
							componentRouteBuilder.Hostname(parameterValue)
						} else if parameterName == tlsSecretRefParameter {
							componentRouteBuilder.TlsSecretRef(parameterValue)
						}
					}
					componentRoutes[componentRoute] = componentRouteBuilder
				}
			}
		}
	}

	curListening := ingress.Listening()
	curRouteSelectors := ingress.RouteSelectors()
	curLbType := ingress.LoadBalancerType()
	curWildcardPolicy := ingress.RouteWildcardPolicy()
	curNamespaceOwnershipPolicy := ingress.RouteNamespaceOwnershipPolicy()
	curExcludedNamespaces := ingress.ExcludedNamespaces()
	curComponentRoutes := ingress.ComponentRoutes()

	ingressBuilder := cmv1.NewIngress().ID(ingress.ID())

	// Toggle private mode
	if private != nil {
		if *private {
			ingressBuilder = ingressBuilder.Listening(cmv1.ListeningMethodInternal)
		} else {
			ingressBuilder = ingressBuilder.Listening(cmv1.ListeningMethodExternal)
		}
	}
	if routeSelector != nil {
		routeSelectors := map[string]string{}
		if *routeSelector != "" {
			routeSelectors, err = helper.GetRouteSelector(*routeSelector)
			if err != nil {
				r.Reporter.Errorf("%s", err)
				os.Exit(1)
			}
		}
		ingressBuilder = ingressBuilder.RouteSelectors(routeSelectors)
	}

	if lbType != nil {
		ingressBuilder = ingressBuilder.LoadBalancerType(cmv1.LoadBalancerFlavor(*lbType))
	}

	if excludedNamespaces != nil {
		_excludedNamespaces := helper.GetExcludedNamespaces(*excludedNamespaces)
		ingressBuilder = ingressBuilder.ExcludedNamespaces(_excludedNamespaces...)
	}

	if wildcardPolicy != nil &&
		!utils.Contains([]string{"", consts.SkipSelectionOption}, *wildcardPolicy) {
		ingressBuilder = ingressBuilder.RouteWildcardPolicy(cmv1.WildcardPolicy(*wildcardPolicy))
	}

	if namespaceOwnershipPolicy != nil &&
		!utils.Contains([]string{"", consts.SkipSelectionOption}, *namespaceOwnershipPolicy) {
		ingressBuilder = ingressBuilder.RouteNamespaceOwnershipPolicy(
			cmv1.NamespaceOwnershipPolicy(*namespaceOwnershipPolicy))
	}

	if len(componentRoutes) != 0 {
		ingressBuilder = ingressBuilder.ComponentRoutes(componentRoutes)
	}

	ingress, err = ingressBuilder.Build()
	if err != nil {
		r.Reporter.Errorf("Failed to create ingress for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	sameRouteSelectors := routeSelector == nil || reflect.DeepEqual(curRouteSelectors, ingress.RouteSelectors())
	// If private arg is nil no change to listening method will be made anyway
	sameListeningMethod := private == nil || curListening == ingress.Listening()

	sameLbType := (lbType == nil) || (curLbType == ingress.LoadBalancerType())

	sameExcludedNamespaces := excludedNamespaces == nil ||
		reflect.DeepEqual(curExcludedNamespaces, ingress.ExcludedNamespaces())

	sameWildcardPolicy := (wildcardPolicy == nil) || (curWildcardPolicy == ingress.RouteWildcardPolicy())

	sameNamespaceOwnershipPolicy := (namespaceOwnershipPolicy == nil) ||
		(curNamespaceOwnershipPolicy == ingress.RouteNamespaceOwnershipPolicy())

	sameComponentRoutes := (componentRoutes == nil) ||
		(reflect.DeepEqual(curComponentRoutes, ingress.ComponentRoutes()))

	if sameListeningMethod && sameRouteSelectors && sameLbType &&
		sameExcludedNamespaces && sameWildcardPolicy && sameNamespaceOwnershipPolicy &&
		sameComponentRoutes {
		r.Reporter.Warnf("No need to update ingress as there are no changes")
		os.Exit(0)
	}

	r.Reporter.Debugf("Updating ingress '%s' on cluster '%s'", ingress.ID(), clusterKey)
	_, err = r.OCMClient.UpdateIngress(cluster.ID(), ingress)
	if err != nil {
		r.Reporter.Errorf("Failed to update ingress '%s' on cluster '%s': %s",
			ingress.ID(), clusterKey, err)
		os.Exit(1)
	}
	r.Reporter.Infof("Updated ingress '%s' on cluster '%s'", ingress.ID(), clusterKey)
}
