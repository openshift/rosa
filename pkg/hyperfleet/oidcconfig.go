package hyperfleet

import (
	"context"
	"fmt"

	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	hyperfleetclientset "github.com/openshift-online/rosa-hyperfleet-api/clientset"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
)

// ResolveOidcConfig finds an OIDC config by its metadata.name or server-assigned
// UID. The UID is used by Cluster references, while item routes use metadata.name.
func ResolveOidcConfig(
	ctx context.Context, client hyperfleetclientset.Interface, oidcKey string,
) (*v1alpha1.OidcConfig, error) {
	if oidcKey == "" {
		return nil, fmt.Errorf("OIDC config ID is required")
	}
	list, err := client.HyperfleetV1alpha1().OidcConfigs().List(ctx, platform.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list OIDC configs: %w", err)
	}
	for i := range list.Items {
		config := &list.Items[i]
		if config.Name == oidcKey || string(config.UID) == oidcKey {
			return config, nil
		}
	}
	return nil, fmt.Errorf("OIDC config '%s' not found", oidcKey)
}
