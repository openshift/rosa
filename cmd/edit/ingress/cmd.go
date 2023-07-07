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

	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

// Regular expression to used to make sure that the identifier given by the
// user is safe and that it there is no risk of SQL injection:
var ingressKeyRE = regexp.MustCompile(`^[a-z0-9]{3,5}$`)

var validLbTypes = []string{string(cmv1.LoadBalancerFlavorClassic), string(cmv1.LoadBalancerFlavorNlb)}
var validWildcardPolicies = []string{string(cmv1.WildcardPolicyWildcardsDisallowed),
	string(cmv1.WildcardPolicyWildcardsAllowed)}
var validNamespaceOwnershipPolicies = []string{string(cmv1.NamespaceOwnershipPolicyStrict),
	string(cmv1.NamespaceOwnershipPolicyInterNamespaceAllowed)}

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

const (
	privateFlag    = "private"
	labelMatchFlag = "label-match"
	lbTypeFlag     = "lb-type"

	routeSelectorFlag             = "route-selector"
	excludedNamespacesFlag        = "excluded-namespaces"
	wildcardPolicyFlag            = "wildcard-policy"
	namespaceOwnershipPolicyFlag  = "namespace-ownership-policy"
	clusterRoutesHostnameFlag     = "cluster-routes-hostname"
	clusterRoutesTlsSecretRefFlag = "cluster-routes-tls-secret-ref"
)

var args struct {
	private    bool
	labelMatch string
	lbType     string

	excludedNamespaces        string
	wildcardPolicy            string
	namespaceOwnershipPolicy  string
	clusterRoutesHostname     string
	clusterRoutesTlsSecretRef string
}

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
		&args.labelMatch,
		labelMatchFlag,
		"",
		"Route Selector for ingress. Format should be a comma-separated list of 'key=value'. "+
			"If no label is specified, all routes will be exposed on both routers.",
	)

	flags.StringVar(
		&args.labelMatch,
		routeSelectorFlag,
		"",
		fmt.Sprintf("Alias to '%s' flag.", labelMatchFlag),
	)

	flags.StringVar(
		&args.lbType,
		lbTypeFlag,
		"",
		fmt.Sprintf("Type of Load Balancer. Options are %s.", strings.Join(validLbTypes, ",")),
	)

	flags.StringVar(
		&args.excludedNamespaces,
		excludedNamespacesFlag,
		"",
		"Excluded namespaces for ingress. Format should be a comma-separated list 'value1, value2...'. "+
			"If no values are specified, all namespaces will be exposed.",
	)

	flags.StringVar(
		&args.wildcardPolicy,
		wildcardPolicyFlag,
		"",
		fmt.Sprintf("Wildcard Policy for ingress. Options are %s", strings.Join(validWildcardPolicies, ",")),
	)

	flags.StringVar(
		&args.namespaceOwnershipPolicy,
		namespaceOwnershipPolicyFlag,
		"",
		fmt.Sprintf("Namespace Ownership Policy for ingress. Options are %s",
			strings.Join(validNamespaceOwnershipPolicies, ",")),
	)

	flags.StringVar(
		&args.clusterRoutesHostname,
		clusterRoutesHostnameFlag,
		"",
		"Cluster Routes Hostname.",
	)

	flags.StringVar(
		&args.clusterRoutesTlsSecretRef,
		clusterRoutesTlsSecretRefFlag,
		"",
		"Cluster Routes TLS Secret Reference.",
	)

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
	return validNamespaceOwnershipPolicies, cobra.ShellCompDirectiveDefault
}

func wildcardPoliciesTypeCompletion(cmd *cobra.Command,
	args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return validWildcardPolicies, cobra.ShellCompDirectiveDefault
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

	if !interactive.Enabled() && shouldEnableInteractive(cmd.Flags(),
		[]string{labelMatchFlag, privateFlag, lbTypeFlag, routeSelectorFlag, excludedNamespacesFlag, wildcardPolicyFlag,
			namespaceOwnershipPolicyFlag, clusterRoutesHostnameFlag, clusterRoutesTlsSecretRefFlag}) {
		interactive.Enable()
	}

	cluster := r.FetchCluster()
	var labelMatch *string
	if cmd.Flags().Changed(labelMatchFlag) {
		if ocm.IsHyperShiftCluster(cluster) {
			r.Reporter.Errorf("Updating route selectors is not supported for Hosted Control Plane clusters")
			os.Exit(1)
		}
		labelMatch = &args.labelMatch
	} else if interactive.Enabled() && !ocm.IsHyperShiftCluster(cluster) {
		labelMatchArg, err := interactive.GetString(interactive.Input{
			Question: "Route Selector for ingress",
			Help:     cmd.Flags().Lookup(labelMatchFlag).Usage,
			Default:  args.labelMatch,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid comma-separated list of attributes: %s", err)
			os.Exit(1)
		}
		labelMatch = &labelMatchArg
	}

	var excludedNamespaces *string
	if cmd.Flags().Changed(excludedNamespacesFlag) {
		if ocm.IsHyperShiftCluster(cluster) {
			r.Reporter.Errorf("Updating excluded namespace is not supported for Hosted Control Plane clusters")
			os.Exit(1)
		}
		excludedNamespaces = &args.excludedNamespaces
	} else if interactive.Enabled() && !ocm.IsHyperShiftCluster(cluster) {
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

	var lbType *string
	if cmd.Flags().Changed(lbTypeFlag) {
		if ocm.IsHyperShiftCluster(cluster) {
			r.Reporter.Errorf("Updating Load Balancer Type is not supported for Hosted Control Plane clusters")
			os.Exit(1)
		}
		lbType = &args.lbType
	} else {
		if interactive.Enabled() {
			if !ocm.IsSts(cluster) {
				if lbType == nil {
					lbType = &validLbTypes[0]
				}
				lbTypeArg, err := interactive.GetOption(interactive.Input{
					Question: "Type of Load Balancer",
					Options:  validLbTypes,
					Required: true,
					Default:  lbType,
				})
				if err != nil {
					r.Reporter.Errorf("Expected a valid Load Balancer type: %s", err)
					os.Exit(1)
				}
				lbType = &lbTypeArg
			}
		}
	}

	var wildcardPolicy *string
	if cmd.Flags().Changed(wildcardPolicyFlag) {
		if ocm.IsHyperShiftCluster(cluster) {
			r.Reporter.Errorf("Updating Wildcard Policy is not supported for Hosted Control Plane clusters")
			os.Exit(1)
		}
		wildcardPolicy = &args.wildcardPolicy
	} else {
		if interactive.Enabled() && !ocm.IsHyperShiftCluster(cluster) {
			wildcardPolicyArg, err := interactive.GetOption(interactive.Input{
				Question: "Wildcard Policy",
				Options:  validWildcardPolicies,
				Default:  args.wildcardPolicy,
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid Wildcard Policy: %s", err)
				os.Exit(1)
			}
			wildcardPolicy = &wildcardPolicyArg
		}
	}

	var namespaceOwnershipPolicy *string
	if cmd.Flags().Changed(namespaceOwnershipPolicyFlag) {
		if ocm.IsHyperShiftCluster(cluster) {
			r.Reporter.Errorf("Updating Namespace Ownership Policy is not supported for Hosted Control Plane clusters")
			os.Exit(1)
		}
		namespaceOwnershipPolicy = &args.namespaceOwnershipPolicy
	} else {
		if interactive.Enabled() && !ocm.IsHyperShiftCluster(cluster) {
			namespaceOwnershipPolicyArg, err := interactive.GetOption(interactive.Input{
				Question: "Namespace Ownership Policy",
				Options:  validNamespaceOwnershipPolicies,
				Default:  args.namespaceOwnershipPolicy,
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid Namespace Ownership Policy: %s", err)
				os.Exit(1)
			}
			namespaceOwnershipPolicy = &namespaceOwnershipPolicyArg
		}
	}

	var clusterRoutesHostname *string
	if cmd.Flags().Changed(clusterRoutesHostnameFlag) {
		if ocm.IsHyperShiftCluster(cluster) {
			r.Reporter.Errorf("Updating Cluster Routes Hostname is not supported for Hosted Control Plane clusters")
			os.Exit(1)
		}
		clusterRoutesHostname = &args.clusterRoutesHostname
	} else if interactive.Enabled() && !ocm.IsHyperShiftCluster(cluster) {
		clusterRoutesHostnameArg, err := interactive.GetString(interactive.Input{
			Question: "Cluster Routes Hostname",
			Help:     cmd.Flags().Lookup(clusterRoutesHostnameFlag).Usage,
			Default:  args.clusterRoutesHostname,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid Cluster Routes Hostname: %s", err)
			os.Exit(1)
		}
		clusterRoutesHostname = &clusterRoutesHostnameArg
	}

	var clusterRoutesTlsSecretRef *string
	if cmd.Flags().Changed(clusterRoutesTlsSecretRefFlag) {
		if ocm.IsHyperShiftCluster(cluster) {
			r.Reporter.Errorf("Updating Cluster Routes Hostname is not supported for Hosted Control Plane clusters")
			os.Exit(1)
		}
		clusterRoutesTlsSecretRef = &args.clusterRoutesTlsSecretRef
	} else if interactive.Enabled() && !ocm.IsHyperShiftCluster(cluster) {
		clusterRoutesTlsSecretRefArg, err := interactive.GetString(interactive.Input{
			Question: "Cluster Routes TLS Secret Reference",
			Help:     cmd.Flags().Lookup(clusterRoutesTlsSecretRefFlag).Usage,
			Default:  args.clusterRoutesTlsSecretRef,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid Cluster Routes TLS Secret Reference: %s", err)
			os.Exit(1)
		}
		clusterRoutesTlsSecretRef = &clusterRoutesTlsSecretRefArg
	}

	if cluster.AWS().PrivateLink() && !ocm.IsHyperShiftCluster(cluster) {
		r.Reporter.Errorf("Cluster '%s' is PrivateLink and does not support updating ingresses", clusterKey)
		os.Exit(1)
	}

	// Edit API endpoint instead of ingresses
	if ingressID == "api" {
		clusterConfig := ocm.Spec{
			Private: private,
		}

		err := r.OCMClient.UpdateCluster(clusterKey, r.Creator, clusterConfig)
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

	curListening := ingress.Listening()
	curRouteSelectors := ingress.RouteSelectors()
	curLbType := ingress.LoadBalancerType()
	curWildcardPolicy := ingress.RouteWildcardPolicy()
	curNamespaceOwnershipPolicy := ingress.RouteNamespaceOwnershipPolicy()
	curExcludedNamespaces := ingress.ExcludedNamespaces()
	curClusterRoutesHostname := ingress.ClusterRoutesHostname()
	curClusterRoutesTlsSecretRef := ingress.ClusterRoutesTlsSecretRef()

	ingressBuilder := cmv1.NewIngress().ID(ingress.ID())

	// Toggle private mode
	if private != nil {
		if *private {
			ingressBuilder = ingressBuilder.Listening(cmv1.ListeningMethodInternal)
		} else {
			ingressBuilder = ingressBuilder.Listening(cmv1.ListeningMethodExternal)
		}
	}
	if labelMatch != nil {
		routeSelectors := map[string]string{}
		if *labelMatch != "" {
			routeSelectors, err = getRouteSelector(*labelMatch)
			if err != nil {
				r.Reporter.Errorf("%s", err)
				os.Exit(1)
			}
		}
		if len(routeSelectors) > 0 {
			ingressBuilder = ingressBuilder.RouteSelectors(routeSelectors)
		}
	}

	if lbType != nil {
		ingressBuilder = ingressBuilder.LoadBalancerType(cmv1.LoadBalancerFlavor(*lbType))
	}

	if excludedNamespaces != nil {
		_excludedNamespaces := []string{}
		if *excludedNamespaces != "" {
			_excludedNamespaces = strings.Split(*excludedNamespaces, ",")
			if err != nil {
				r.Reporter.Errorf("%s", err)
				os.Exit(1)
			}
		}
		if len(_excludedNamespaces) > 0 {
			ingressBuilder = ingressBuilder.ExcludedNamespaces(_excludedNamespaces...)
		}
	}

	if wildcardPolicy != nil {
		ingressBuilder = ingressBuilder.RouteWildcardPolicy(cmv1.WildcardPolicy(*wildcardPolicy))
	}

	if namespaceOwnershipPolicy != nil {
		ingressBuilder = ingressBuilder.RouteNamespaceOwnershipPolicy(
			cmv1.NamespaceOwnershipPolicy(*namespaceOwnershipPolicy))
	}

	ingress, err = ingressBuilder.Build()
	if err != nil {
		r.Reporter.Errorf("Failed to create ingress for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	sameRouteSelectors := labelMatch == nil || reflect.DeepEqual(curRouteSelectors, ingress.RouteSelectors())
	// If private arg is nil no change to listening method will be made anyway
	sameListeningMethod := private == nil || curListening == ingress.Listening()

	sameLbType := (lbType == nil) || (curLbType == ingress.LoadBalancerType())

	sameExcludedNamespaces := excludedNamespaces == nil ||
		reflect.DeepEqual(curExcludedNamespaces, ingress.ExcludedNamespaces())

	sameWildcardPolicy := (wildcardPolicy == nil) || (curWildcardPolicy == ingress.RouteWildcardPolicy())

	sameNamespaceOwnershipPolicy := (namespaceOwnershipPolicy == nil) ||
		(curNamespaceOwnershipPolicy == ingress.RouteNamespaceOwnershipPolicy())

	sameClusterRoutesHostname := (clusterRoutesHostname == nil) ||
		(curClusterRoutesHostname == ingress.ClusterRoutesHostname())

	sameClusterRoutesTlsSecretRef := (clusterRoutesTlsSecretRef == nil) ||
		(curClusterRoutesTlsSecretRef == ingress.ClusterRoutesTlsSecretRef())

	if sameListeningMethod && sameRouteSelectors && sameLbType &&
		sameExcludedNamespaces && sameWildcardPolicy && sameNamespaceOwnershipPolicy &&
		sameClusterRoutesHostname && sameClusterRoutesTlsSecretRef {
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
