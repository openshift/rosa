// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

// Package tagging prepares day-2 ROSA HCP AWS tag edits for the OCM API.
package tagging

import (
	"fmt"
	"strings"

	commontags "github.com/openshift-online/ocm-common/pkg/aws/tags"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/helper/versions"
)

// MinimalVersion is the first control-plane version that reconciles AWS tag edits in place.
const MinimalVersion = "4.20.0-0.a"

// ValidateCluster rejects tag edits outside ROSA HCP 4.20 and later.
func ValidateCluster(cluster *cmv1.Cluster) error {
	if cluster == nil || cluster.Hypershift() == nil || !aws.IsHostedCP(cluster) {
		return fmt.Errorf("editing AWS tags is only supported for Hosted Control Plane clusters")
	}
	if cluster.Version() == nil || cluster.Version().RawID() == "" {
		return fmt.Errorf("cannot edit AWS tags without a control-plane version")
	}
	supported, err := SupportsControlPlaneVersion(cluster.Version().RawID())
	if err != nil {
		return fmt.Errorf("cannot check control-plane version for AWS tag editing: %w", err)
	}
	if !supported {
		return fmt.Errorf("editing AWS tags requires OpenShift version 4.20.0 or greater")
	}
	return nil
}

// SupportsControlPlaneVersion reports whether a raw OCM control-plane version supports day-2 AWS tags.
// It accepts both plain versions and the prefixed raw IDs returned by OCM.
func SupportsControlPlaneVersion(rawID string) (bool, error) {
	version := strings.TrimPrefix(rawID, "openshift-v")
	version = strings.TrimPrefix(version, "openshift-")
	return versions.IsGreaterThanOrEqual(version, MinimalVersion)
}

// ParseDesired parses the complete desired customer tag set. A single empty flag value clears it.
func ParseDesired(input []string) (map[string]string, error) {
	if len(input) == 1 && input[0] == "" {
		return map[string]string{}, nil
	}
	for _, item := range input {
		if item == "" {
			return nil, fmt.Errorf("an empty --tags value must be used alone to remove all tags")
		}
	}
	desired, err := aws.ParseUserTags(input, aws.UserTagOptions{})
	if err != nil {
		return nil, err
	}
	if err := commontags.Validate(desired); err != nil {
		return nil, err
	}
	for key := range desired {
		if strings.HasPrefix(key, "api.openshift.com/") {
			return nil, fmt.Errorf("AWS tag key %q is reserved", key)
		}
	}
	return desired, nil
}

// Diff returns the PATCH tag changes needed to replace current with desired.
// Nil return means that the sets are equal; nil values in the result delete keys.
func Diff(current, desired map[string]string) map[string]*string {
	if commontags.Equal(current, desired) {
		return nil
	}
	changes := commontags.ComputeDiff(current, desired)
	patch := make(map[string]*string, len(changes.Add)+len(changes.Update)+len(changes.Remove))
	for key, value := range changes.Add {
		value := value
		patch[key] = &value
	}
	for key, value := range changes.Update {
		value := value
		patch[key] = &value
	}
	for _, key := range changes.Remove {
		patch[key] = nil
	}
	return patch
}

// PoolOverrides removes inherited cluster tags from the current node-pool tag view.
// An explicit override equal to the inherited value is indistinguishable and redundant.
func PoolOverrides(clusterTags, poolTags map[string]string) map[string]string {
	overrides := make(map[string]string)
	for key, value := range poolTags {
		if inherited, ok := clusterTags[key]; !ok || inherited != value {
			overrides[key] = value
		}
	}
	return overrides
}
