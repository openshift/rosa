package hyperfleet

import (
	"fmt"

	"github.com/hashicorp/go-version"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
)

// ResolveNodePoolReleaseImage maps the cluster's running version to its release image.
// The Platform API has no version catalog, so other versions require an explicit --image.
func ResolveNodePoolReleaseImage(cluster *v1alpha1.Cluster, requestedVersion, image string) (string, error) {
	if image != "" {
		return "", fmt.Errorf("--version and --image cannot be used together")
	}
	if _, err := version.NewVersion(requestedVersion); err != nil {
		return "", fmt.Errorf("expected a valid OpenShift version: %w", err)
	}
	if requestedVersion != cluster.Status.Version {
		return "", fmt.Errorf("expected a valid OpenShift version: version %q cannot be resolved by the Platform API; "+
			"use --image to specify its release image", requestedVersion)
	}
	if cluster.Spec.HostedCluster.Release.Image == "" {
		return "", fmt.Errorf("expected a valid OpenShift version: cluster version %q has no configured release image",
			requestedVersion)
	}
	return cluster.Spec.HostedCluster.Release.Image, nil
}
