// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package ingress

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
	configv1 "github.com/openshift/api/config/v1"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/rosa"
)

const componentRouteNamespace = "openshift-console"

func editHyperfleetComponentRoutesRunner() rosa.CommandRunner {
	return func(ctx context.Context, r *rosa.Runtime, cmd *cobra.Command, argv []string) error {
		if !cmd.Flags().Changed(componentRoutesFlag) {
			return fmt.Errorf("HyperFleet ingress updates require --%s", componentRoutesFlag)
		}
		for _, name := range []string{
			privateFlag, labelMatchFlag, lbTypeFlag, routeSelectorFlag, excludedNamespacesFlag,
			wildcardPolicyFlag, namespaceOwnershipPolicyFlag, clusterRoutesHostnameFlag,
			clusterRoutesTlsSecretRefFlag,
		} {
			if cmd.Flags().Changed(name) {
				return fmt.Errorf("--%s is not supported for HyperFleet ingress updates", name)
			}
		}

		updates, err := parseComponentRoutesForAllowed(args.componentRoutes, expectedHcpComponentRoutes)
		if err != nil {
			return fmt.Errorf("failed to parse component routes: %w", err)
		}

		clusterKey := r.GetClusterKey()
		clusterUID, err := hyperfleet.ResolveClusterUID(ctx, r.HyperFleetClient, clusterKey)
		if err != nil {
			return err
		}
		clusters := r.HyperFleetClient.HyperfleetV1alpha1().Clusters()
		cluster, err := clusters.Get(ctx, clusterUID, platform.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get cluster %q: %w", clusterKey, err)
		}

		routes := map[string]v1alpha1.ComponentRouteConfiguration{}
		if cluster.Spec.HostedCluster.Configuration != nil &&
			cluster.Spec.HostedCluster.Configuration.Ingress != nil {
			for _, route := range cluster.Spec.HostedCluster.Configuration.Ingress.ComponentRoutes {
				routes[route.Namespace+"/"+route.Name] = route
			}
		}
		for name, builder := range updates {
			route, err := builder.Build()
			if err != nil {
				return fmt.Errorf("failed to build component route %q: %w", name, err)
			}
			key := componentRouteNamespace + "/" + name
			if route.Hostname() == "" && route.TlsSecretRef() == "" {
				delete(routes, key)
				continue
			}
			routes[key] = v1alpha1.ComponentRouteConfiguration{
				Namespace: componentRouteNamespace,
				Name:      name,
				Hostname:  configv1.Hostname(route.Hostname()),
				ServingCertKeyPairSecret: configv1.SecretNameReference{
					Name: route.TlsSecretRef(),
				},
			}
		}

		keys := make([]string, 0, len(routes))
		for key := range routes {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		componentRoutes := make([]v1alpha1.ComponentRouteConfiguration, 0, len(keys))
		for _, key := range keys {
			componentRoutes = append(componentRoutes, routes[key])
		}

		patch, err := json.Marshal(map[string]any{
			"spec": map[string]any{
				"hostedCluster": map[string]any{
					"configuration": map[string]any{
						"ingress": map[string]any{"componentRoutes": componentRoutes},
					},
				},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to build component-routes update: %w", err)
		}
		if _, err := clusters.Patch(ctx, clusterUID, types.MergePatchType, patch, platform.PatchOptions{}); err != nil {
			return fmt.Errorf("failed to update component routes for cluster %q: %w", clusterKey, err)
		}

		r.Reporter.Infof("Updated default ingress component routes on cluster '%s'", clusterKey)
		return nil
	}
}
